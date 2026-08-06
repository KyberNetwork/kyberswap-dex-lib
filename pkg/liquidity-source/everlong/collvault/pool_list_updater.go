package everlongcollvault

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

// GetNewPools lists the single configured CollVault venue, resolving the contract graph
// (collVault, settlement swapper, ALM adapter, asset decimals) on-chain from the
// rebalancer so only the rebalancer address and the two swap legs are configuration.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}

	curveParams, err := u.resolveCurveParams()
	if err != nil {
		return nil, nil, err
	}

	var (
		collVault, swapper gethcommon.Address
	)
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    rebalancerABI,
		Target: u.config.Rebalancer,
		Method: rebalancerMethodCollVault,
	}, []any{&collVault}).AddCall(&ethrpc.Call{
		ABI:    rebalancerABI,
		Target: u.config.Rebalancer,
		Method: rebalancerMethodSettlementSwapper,
	}, []any{&swapper})
	if _, err := req.Aggregate(); err != nil {
		return nil, nil, err
	}

	var (
		alm           gethcommon.Address
		assetDecimals uint8
		blockNumber   uint64
	)
	req = u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    swapperABI,
		Target: hexutil.Encode(swapper[:]),
		Method: swapperMethodAlm,
	}, []any{&alm}).AddCall(&ethrpc.Call{
		ABI:    collVaultABI,
		Target: hexutil.Encode(collVault[:]),
		Method: cvMethodAssetDecimals,
	}, []any{&assetDecimals})
	resp, err := req.Aggregate()
	if err != nil {
		return nil, nil, err
	}
	if resp.BlockNumber != nil {
		blockNumber = resp.BlockNumber.Uint64()
	}

	staticExtra, err := json.Marshal(StaticExtra{
		Rebalancer:       strings.ToLower(u.config.Rebalancer),
		Swapper:          hexutil.Encode(swapper[:]),
		CollVault:        hexutil.Encode(collVault[:]),
		ALM:              hexutil.Encode(alm[:]),
		CvDecimalsOffset: 18 - assetDecimals,
		CurveParams:      curveParams,
	})
	if err != nil {
		return nil, nil, err
	}

	u.hasInitialized = true

	return []entity.Pool{
		{
			Address:   hexutil.Encode(swapper[:]),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(u.config.Stable), Swappable: true},
				{Address: strings.ToLower(u.config.Volatile), Swappable: true},
			},
			Reserves:    entity.PoolReserves{"0", "0"},
			StaticExtra: string(staticExtra),
			Extra:       "{}",
			BlockNumber: blockNumber,
		},
	}, nil, nil
}

func (u *PoolsListUpdater) resolveCurveParams() (CurveParams, error) {
	if u.config.CurveParams != nil {
		cp := *u.config.CurveParams
		if cp.LeverageRatioWad == nil || cp.HZero == nil || cp.HJoin == nil || cp.HWall == nil ||
			cp.Width == nil || cp.DJoin == nil || cp.DWall == nil || cp.RescueSpreadPpm == nil ||
			cp.PhysicalCrFloorWad == nil {
			return CurveParams{}, ErrInvalidCurveParams
		}
		for _, v := range cp.BezierPhi {
			if v == nil {
				return CurveParams{}, ErrInvalidCurveParams
			}
		}
		for _, v := range cp.BezierIntegral {
			if v == nil {
				return CurveParams{}, ErrInvalidCurveParams
			}
		}
		return cp, nil
	}
	builtin, ok := curveParamsByChain[u.config.ChainID]
	if !ok {
		return CurveParams{}, ErrNoCurveParams
	}
	return builtin(), nil
}
