package everlongcvamm

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

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

// getNewPoolState refreshes the whole venue state in ONE multicall (one block): the
// funded band, the authoritative inventory coordinate xWad (never the derived price),
// the anchor, kappa, the accounted reserves (solvency), both directional fees (the fee
// law's realized-variance input is unobservable off-chain, so the fee is read, never
// computed) and the pause flag.
func (t *PoolTracker) getNewPoolState(ctx context.Context, p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount) (entity.Pool, error) {
	var (
		sup             supportRaw
		xWad            = new(big.Int)
		anchor          = new(big.Int)
		kappa           = new(big.Int)
		reserveStable   = new(big.Int)
		reserveVolatile = new(big.Int)
		feeStableIn     = new(big.Int)
		feeVolatileIn   = new(big.Int)
		paused          bool
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}
	req.AddCall(&ethrpc.Call{
		ABI: almABI, Target: p.Address, Method: almMethodGetSupport,
	}, []any{&sup})
	req.AddCall(&ethrpc.Call{
		ABI: almABI, Target: p.Address, Method: almMethodXWad,
	}, []any{&xWad})
	req.AddCall(&ethrpc.Call{
		ABI: almABI, Target: p.Address, Method: almMethodAnchorSqrtCurveX96,
	}, []any{&anchor})
	req.AddCall(&ethrpc.Call{
		ABI: almABI, Target: p.Address, Method: almMethodKappa,
	}, []any{&kappa})
	req.AddCall(&ethrpc.Call{
		ABI: almABI, Target: p.Address, Method: almMethodReserveStable,
	}, []any{&reserveStable})
	req.AddCall(&ethrpc.Call{
		ABI: almABI, Target: p.Address, Method: almMethodReserveVolatile,
	}, []any{&reserveVolatile})
	req.AddCall(&ethrpc.Call{
		ABI: almABI, Target: p.Address, Method: almMethodPoolFeeDirectional, Params: []any{true},
	}, []any{&feeStableIn})
	req.AddCall(&ethrpc.Call{
		ABI: almABI, Target: p.Address, Method: almMethodPoolFeeDirectional, Params: []any{false},
	}, []any{&feeVolatileIn})
	req.AddCall(&ethrpc.Call{
		ABI: almABI, Target: p.Address, Method: almMethodPaused,
	}, []any{&paused})

	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}

	support, err := supportFromRaw(&sup)
	if err != nil {
		return p, err
	}
	xWadU, err := u256(xWad)
	if err != nil {
		return p, err
	}
	anchorU, err := u256(anchor)
	if err != nil {
		return p, err
	}
	kappaU, err := u256(kappa)
	if err != nil {
		return p, err
	}
	feeStableInU, err := u256(feeStableIn)
	if err != nil {
		return p, err
	}
	feeVolatileInU, err := u256(feeVolatileIn)
	if err != nil {
		return p, err
	}
	if reserveStable.Sign() < 0 || reserveVolatile.Sign() < 0 {
		return p, ErrOverflow
	}

	extraBytes, err := json.Marshal(Extra{
		Support:          support,
		XWad:             xWadU,
		AnchorSqrtX96:    anchorU,
		Kappa:            kappaU,
		FeeStableInWad:   feeStableInU,
		FeeVolatileInWad: feeVolatileInU,
		Paused:           paused,
	})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{reserveStable.String(), reserveVolatile.String()}
	p.Timestamp = time.Now().Unix()
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}
	return p, nil
}

func supportFromRaw(raw *supportRaw) (Support, error) {
	aWad, err := u256(raw.AWad)
	if err != nil {
		return Support{}, err
	}
	xLo, err := u256(raw.XLo)
	if err != nil {
		return Support{}, err
	}
	xHi, err := u256(raw.XHi)
	if err != nil {
		return Support{}, err
	}
	yHi, err := u256(raw.YHi)
	if err != nil {
		return Support{}, err
	}
	return Support{AWad: aWad, XLo: xLo, XHi: xHi, YHi: yHi}, nil
}

func u256(v *big.Int) (*uint256.Int, error) {
	if v == nil {
		return nil, ErrOverflow
	}
	res, overflow := uint256.FromBig(v)
	if overflow {
		return nil, ErrOverflow
	}
	return res, nil
}
