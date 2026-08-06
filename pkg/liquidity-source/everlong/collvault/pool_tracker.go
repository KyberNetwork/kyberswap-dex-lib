package everlongcollvault

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ pool.GetNewPoolStateParams) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(ctx context.Context, p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params.Overrides)
}

// getNewPoolState refreshes the full vault snapshot in ONE multicall (all reads pinned
// to the same block): exchangeState + the ALM/CollVault words the CR math and the
// token-leg preview need, incl. the reference reserves bounding the max leverage lot.
func (t *PoolTracker) getNewPoolState(ctx context.Context, p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var (
		collateral, debt, priceWad, spreadPpm   = new(big.Int), new(big.Int), new(big.Int), new(big.Int)
		almStable, almVolatile                  = new(big.Int), new(big.Int)
		almSupply, cvTotalAssets, cvTotalSupply = new(big.Int), new(big.Int), new(big.Int)
		withdrawFeeBp                           = new(big.Int)
		refStable, refAsset, refRaw             = new(big.Int), new(big.Int), new(big.Int)
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}
	req.AddCall(&ethrpc.Call{
		ABI:    rebalancerABI,
		Target: staticExtra.Rebalancer,
		Method: rebalancerMethodExchangeState,
	}, []any{&collateral, &debt, &priceWad, &spreadPpm})
	req.AddCall(&ethrpc.Call{
		ABI:    almABI,
		Target: staticExtra.ALM,
		Method: almMethodGetTotalAmounts,
	}, []any{&almStable, &almVolatile})
	req.AddCall(&ethrpc.Call{
		ABI:    almABI,
		Target: staticExtra.ALM,
		Method: erc20MethodTotalSupply,
	}, []any{&almSupply})
	req.AddCall(&ethrpc.Call{
		ABI:    collVaultABI,
		Target: staticExtra.CollVault,
		Method: cvMethodTotalAssets,
	}, []any{&cvTotalAssets})
	req.AddCall(&ethrpc.Call{
		ABI:    collVaultABI,
		Target: staticExtra.CollVault,
		Method: erc20MethodTotalSupply,
	}, []any{&cvTotalSupply})
	req.AddCall(&ethrpc.Call{
		ABI:    collVaultABI,
		Target: staticExtra.CollVault,
		Method: cvMethodGetWithdrawFee,
	}, []any{&withdrawFeeBp})
	req.AddCall(&ethrpc.Call{
		ABI:    almABI,
		Target: staticExtra.ALM,
		Method: almMethodGetReservesAtReference,
	}, []any{&refStable, &refAsset, &refRaw})

	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}
	for _, v := range []*big.Int{collateral, debt, priceWad, spreadPpm, almStable, almVolatile,
		almSupply, cvTotalAssets, cvTotalSupply, withdrawFeeBp, refStable, refAsset, refRaw} {
		if v == nil || v.Sign() < 0 {
			return p, ErrInvalidSnapshotWord
		}
	}

	extraBytes, err := json.Marshal(Extra{
		Collateral: collateral, Debt: debt, PriceWad: priceWad, SpreadPpm: spreadPpm,
		AlmStableReserve: almStable, AlmVolatileReserve: almVolatile, AlmSupply: almSupply,
		CvTotalAssets: cvTotalAssets, CvTotalSupply: cvTotalSupply,
		CvDecimalsOffset: staticExtra.CvDecimalsOffset, WithdrawFeeBp: withdrawFeeBp,
		RefStableReserve: refStable, RefAssetReserve: refAsset, RefRawReferenceWad: refRaw,
	})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	// The ALM reserves are the physical inventory both swap directions settle against —
	// the router-visible liquidity proxy.
	p.Reserves = entity.PoolReserves{almStable.String(), almVolatile.String()}
	p.Timestamp = time.Now().Unix()
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}
	return p, nil
}
