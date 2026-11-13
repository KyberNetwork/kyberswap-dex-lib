package doppleramm

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

const maxLaunchLevels = 64

type launchMarketState struct {
	SwapFeePips  *big.Int `abi:"swapFeePips"`
	SqrtPriceX96 *big.Int `abi:"sqrtPriceX96"`
	Level        *big.Int `abi:"level"`
	Unused3      *big.Int `abi:"unused3"`
	Unused4      *big.Int `abi:"unused4"`
	Liquidity    *big.Int `abi:"liquidity"`
}

type launchLevelInfo struct {
	Initialized  bool     `abi:"initialized"`
	LiquidityNet *big.Int `abi:"liquidityNet"`
}

type launchLevelBounds struct {
	Spacing  *big.Int `abi:"spacing"`
	Count    *big.Int `abi:"count"`
	MinLevel *big.Int `abi:"minLevel"`
	MaxLevel *big.Int `abi:"maxLevel"`
}

// fetchLevels lists the initialized levels of a launch market, ascending. They
// are fixed at creation, so this runs once per market.
func (t *PoolTracker) fetchLevels(ctx context.Context, module string, marketID *big.Int) ([]int32, error) {
	var bounds launchLevelBounds
	if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: launchABI, Target: module, Method: "levelBounds", Params: []any{marketID}}, []any{&bounds}).
		Call(); err != nil {
		return nil, err
	}
	if bounds.Count == nil || !bounds.Count.IsInt64() || bounds.Count.Int64() < 2 || bounds.Count.Int64() > maxLaunchLevels ||
		bounds.MinLevel == nil || !bounds.MinLevel.IsInt64() {
		return nil, ErrUnsupportedMarket
	}
	count, level := int(bounds.Count.Int64()), bounds.MinLevel.Int64()
	levels := make([]int32, 0, count)
	for len(levels) < count {
		var next *big.Int
		if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).
			AddCall(&ethrpc.Call{ABI: launchABI, Target: module, Method: "nextInitializedLevel",
				Params: []any{marketID, int32(level)}}, []any{&next}).
			Call(); err != nil {
			return nil, err
		}
		if next == nil || !next.IsInt64() || next.Int64() < level || next.Int64() > v3.MaxTick {
			return nil, ErrUnsupportedMarket
		}
		levels = append(levels, int32(next.Int64()))
		level = next.Int64() + 1
	}
	return levels, nil
}

func (t *PoolTracker) getLaunchState(
	ctx context.Context, p entity.Pool, static StaticExtra, marketID *big.Int,
) (entity.Pool, error) {
	if len(static.Levels) == 0 {
		levels, err := t.fetchLevels(ctx, static.Module, marketID)
		if err != nil {
			return p, err
		}
		static.Levels = levels
		staticBytes, err := json.Marshal(static)
		if err != nil {
			return p, err
		}
		p.StaticExtra = string(staticBytes)
	}

	var reserves marketReserves
	var fee protocolFee
	var state launchMarketState
	infos := make([]launchLevelInfo, len(static.Levels))
	req := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: ammABI, Target: static.AMM, Method: "reserves", Params: []any{marketID}}, []any{&reserves}).
		AddCall(&ethrpc.Call{ABI: ammABI, Target: static.AMM, Method: "protocolFeeForMarket", Params: []any{marketID}}, []any{&fee}).
		AddCall(&ethrpc.Call{ABI: launchABI, Target: static.Module, Method: "marketStates", Params: []any{marketID}}, []any{&state})
	for i, level := range static.Levels {
		req.AddCall(&ethrpc.Call{ABI: launchABI, Target: static.Module, Method: "levelInfo",
			Params: []any{marketID, level}}, []any{&infos[i]})
	}
	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	for _, ok := range resp.Result {
		if !ok {
			return p, ErrCallReverted
		}
	}
	if reserves.ReserveX == nil || reserves.ReserveY == nil || state.SqrtPriceX96 == nil || state.Liquidity == nil ||
		state.SwapFeePips == nil || !state.SwapFeePips.IsUint64() || state.SwapFeePips.Uint64() >= feePipsDenominator {
		return p, ErrCallReverted
	}
	if reserves.ReserveX.Sign() <= 0 && reserves.ReserveY.Sign() <= 0 {
		return p, ErrInsufficientLiquidity
	}

	extra := Extra{
		TokenX:      static.TokenX,
		BaseFeeBps:  fee.SwapBaseFeeBps,
		SwapFeePips: uint32(state.SwapFeePips.Uint64()),
	}
	sqrtP, overflowP := uint256.FromBig(state.SqrtPriceX96)
	liquidity, overflowL := uint256.FromBig(state.Liquidity)
	if overflowP || overflowL {
		return p, ErrOverflow
	}
	extra.SqrtPriceX96, extra.Liquidity = sqrtP, liquidity
	extra.Nets = make([]*int256.Int, len(infos))
	for i, info := range infos {
		if info.LiquidityNet == nil {
			return p, ErrCallReverted
		}
		net, err := int256.FromBig(info.LiquidityNet)
		if err != nil {
			return p, err
		}
		extra.Nets[i] = net
	}
	if err := checkLaunchLiquidity(static.Levels, &extra); err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{reserves.ReserveX.String(), reserves.ReserveY.String()}
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}
	p.Timestamp = time.Now().Unix()
	return p, nil
}

// checkLaunchLiquidity rejects state this source cannot model: the level nets
// must sum to zero, and the liquidity at the current price must equal the sum of
// the nets at or below it. A position the level list misses (for example fee
// compounding into full-range liquidity) breaks one of these.
func checkLaunchLiquidity(levels []int32, extra *Extra) error {
	var sum, atPrice, bound uint256.Int
	for i, level := range levels {
		if err := v3.GetSqrtRatioAtTick(int(level), &bound); err != nil {
			return err
		}
		sum.Add(&sum, (*uint256.Int)(extra.Nets[i]))
		if !bound.Gt(extra.SqrtPriceX96) {
			atPrice.Set(&sum)
		}
	}
	if !sum.IsZero() || !atPrice.Eq(extra.Liquidity) {
		return ErrLaunchState
	}
	return nil
}
