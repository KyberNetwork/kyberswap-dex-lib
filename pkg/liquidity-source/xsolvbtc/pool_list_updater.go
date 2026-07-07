package xsolvbtc

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolListUpdater)

func NewPoolListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	return &PoolListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}
	u.hasInitialized = true
	extra := &PoolExtra{}

	blockNumber, err := updateExtra(ctx, extra, u.cfg, u.ethrpcClient)
	if err != nil {
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, nil, err
	}

	return []entity.Pool{
		{
			Address:   strings.ToLower(u.cfg.Pool),
			Exchange:  string(valueobject.ExchangeXSolvBTC),
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{reserves, reserves},
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(u.cfg.SolvBTC),
					Swappable: true,
				},
				{
					Address:   strings.ToLower(u.cfg.XsolvBTC),
					Swappable: true,
				},
			},
			BlockNumber: blockNumber,
			Extra:       string(extraBytes),
		},
	}, nil, nil
}

func updateExtra(ctx context.Context, extra *PoolExtra, cfg *Config, ethrpcClient *ethrpc.Client) (uint64, error) {
	req := ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    PoolABI,
		Target: cfg.Pool,
		Method: "depositAllowed",
	}, []any{&(extra.DepositAllowed)})
	req.AddCall(&ethrpc.Call{
		ABI:    PoolABI,
		Target: cfg.Pool,
		Method: "withdrawFeeRate",
	}, []any{&(extra.WithdrawFeeRate)})
	req.AddCall(&ethrpc.Call{
		ABI:    PoolABI,
		Target: cfg.Pool,
		Method: "maxMultiplier",
	}, []any{&(extra.MaxMultiplier)})
	req.AddCall(&ethrpc.Call{
		ABI:    xsolvBTCABI,
		Target: cfg.XsolvBTC,
		Method: "getOracle",
	}, []any{&(extra.Oracle)})

	if extra.Oracle != valueobject.AddrZero {
		req.AddCall(&ethrpc.Call{
			ABI:    OracleABI,
			Target: extra.Oracle.Hex(),
			Method: "getNav",
			Params: []any{common.HexToAddress(cfg.XsolvBTC)},
		}, []any{&(extra.Nav)})
	}
	resp, err := req.Aggregate()
	if err != nil {
		return 0, err
	}
	return resp.BlockNumber.Uint64(), nil
}
