package everlongclamm

import (
	"context"
	"math/big"
	"sort"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
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

func (t *PoolTracker) getNewPoolState(ctx context.Context, p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var (
		slot0        slot0Raw
		rungs        rungsRaw
		liquidity    = new(big.Int)
		poolFee0For1 = new(big.Int)
		poolFee1For0 = new(big.Int)
		poolID       = common.HexToHash(p.Address)
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}
	req.AddCall(&ethrpc.Call{
		ABI:    poolManagerABI,
		Target: staticExtra.PoolManager,
		Method: poolManagerMethodGetSlot0,
		Params: []any{poolID},
	}, []any{&slot0})
	req.AddCall(&ethrpc.Call{
		ABI:    poolManagerABI,
		Target: staticExtra.PoolManager,
		Method: poolManagerMethodGetLiquidity,
		Params: []any{poolID},
	}, []any{&liquidity})
	req.AddCall(&ethrpc.Call{
		ABI:    almABI,
		Target: staticExtra.ALM,
		Method: almMethodGetRungs,
	}, []any{&rungs})
	req.AddCall(&ethrpc.Call{
		ABI:    almABI,
		Target: staticExtra.ALM,
		Method: almMethodPoolFee,
		Params: []any{true},
	}, []any{&poolFee0For1})
	req.AddCall(&ethrpc.Call{
		ABI:    almABI,
		Target: staticExtra.ALM,
		Method: almMethodPoolFee,
		Params: []any{false},
	}, []any{&poolFee1For0})

	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}

	if len(rungs.Lowers) != len(rungs.Uppers) || len(rungs.Lowers) != len(rungs.Liquidities) {
		return p, ErrInvalidRungs
	}

	ticks, err := rungsToTicks(rungs.Lowers, rungs.Uppers, rungs.Liquidities)
	if err != nil {
		return p, err
	}

	if slot0.SqrtPriceX96 == nil || slot0.Tick == nil {
		return p, ErrPoolNotInitialized
	}
	sqrtPriceU256, overflow := uint256.FromBig(slot0.SqrtPriceX96)
	if overflow {
		return p, uniswapv3.ErrOverflow
	}
	liquidityU256, overflow := uint256.FromBig(liquidity)
	if overflow {
		return p, uniswapv3.ErrOverflow
	}
	poolFee0For1U256, overflow := uint256.FromBig(poolFee0For1)
	if overflow {
		return p, uniswapv3.ErrOverflow
	}
	poolFee1For0U256, overflow := uint256.FromBig(poolFee1For0)
	if overflow {
		return p, uniswapv3.ErrOverflow
	}

	tickInt := int(slot0.Tick.Int64())
	reserve0, reserve1, err := ladderReserves(sqrtPriceU256, tickInt, ticks)
	if err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(Extra{
		ExtraTickU256: uniswapv3.ExtraTickU256{
			Liquidity:    liquidityU256,
			SqrtPriceX96: sqrtPriceU256,
			TickSpacing:  uint64(staticExtra.TickSpacing),
			Tick:         &tickInt,
			Ticks:        ticks,
		},
		PoolFee0For1Wad: poolFee0For1U256,
		PoolFee1For0Wad: poolFee1For0U256,
	})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{reserve0.String(), reserve1.String()}
	p.Timestamp = time.Now().Unix()
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}
	return p, nil
}

// rungsToTicks flattens the ALM rung ladder ([lower,upper] ranges with liquidity L) into
// the standard V3 tick list: +L net at each lower edge, -L at each upper edge; gross
// accumulates L at both edges. The result is sorted by tick index and nets to zero.
func rungsToTicks(lowers, uppers, liquidities []*big.Int) ([]uniswapv3.TickU256, error) {
	net := make(map[int]*int256.Int, 2*len(lowers))
	gross := make(map[int]*uint256.Int, 2*len(lowers))
	addEdge := func(tick int, liq *uint256.Int, sign int) {
		if net[tick] == nil {
			net[tick] = new(int256.Int)
			gross[tick] = new(uint256.Int)
		}
		liqSigned, _ := int256.FromBig(liq.ToBig()) // liq is uint128 — always fits int256

		if sign < 0 {
			liqSigned.Neg(liqSigned)
		}
		net[tick].Add(net[tick], liqSigned)
		gross[tick].Add(gross[tick], liq)
	}
	for i := range lowers {
		liq, overflow := uint256.FromBig(liquidities[i])
		if overflow {
			return nil, uniswapv3.ErrOverflow
		}
		lower, upper := int(lowers[i].Int64()), int(uppers[i].Int64())
		if lower >= upper {
			return nil, ErrInvalidRungs
		}
		addEdge(lower, liq, 1)
		addEdge(upper, liq, -1)
	}
	indexes := make([]int, 0, len(net))
	for idx := range net {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	ticks := make([]uniswapv3.TickU256, 0, len(indexes))
	for _, idx := range indexes {
		ticks = append(ticks, uniswapv3.TickU256{
			Index:          idx,
			LiquidityGross: gross[idx],
			LiquidityNet:   net[idx],
		})
	}
	return ticks, nil
}

// ladderReserves computes the token amounts currently backing the tick ladder at the
// given price — the pool's effective reserves (the ALM is the sole LP, so the ladder IS
// the pool's entire liquidity).
func ladderReserves(sqrtPriceX96 *uint256.Int, tickCurrent int,
	ticks []uniswapv3.TickU256) (*big.Int, *big.Int, error) {
	var (
		reserve0, reserve1     uint256.Int
		liquidity              int256.Int
		sqrtLower, sqrtUpper   uint256.Int
		amount0, amount1, liqU uint256.Int
	)
	for i := 0; i+1 < len(ticks); i++ {
		liquidity.Add(&liquidity, ticks[i].LiquidityNet)
		if liquidity.Sign() <= 0 {
			continue
		}
		liqU.Set((*uint256.Int)(&liquidity))
		lower, upper := ticks[i].Index, ticks[i+1].Index
		if err := uniswapv3.GetSqrtRatioAtTick(lower, &sqrtLower); err != nil {
			return nil, nil, err
		}
		if err := uniswapv3.GetSqrtRatioAtTick(upper, &sqrtUpper); err != nil {
			return nil, nil, err
		}
		switch {
		case tickCurrent < lower:
			if err := uniswapv3.GetAmount0DeltaV2(&sqrtLower, &sqrtUpper, &liqU, false, &amount0); err != nil {
				return nil, nil, err
			}
			reserve0.Add(&reserve0, &amount0)
		case tickCurrent >= upper:
			if err := uniswapv3.GetAmount1DeltaV2(&sqrtLower, &sqrtUpper, &liqU, false, &amount1); err != nil {
				return nil, nil, err
			}
			reserve1.Add(&reserve1, &amount1)
		default:
			if err := uniswapv3.GetAmount0DeltaV2(sqrtPriceX96, &sqrtUpper, &liqU, false, &amount0); err != nil {
				return nil, nil, err
			}
			if err := uniswapv3.GetAmount1DeltaV2(&sqrtLower, sqrtPriceX96, &liqU, false, &amount1); err != nil {
				return nil, nil, err
			}
			reserve0.Add(&reserve0, &amount0)
			reserve1.Add(&reserve1, &amount1)
		}
	}
	return reserve0.ToBig(), reserve1.ToBig(), nil
}
