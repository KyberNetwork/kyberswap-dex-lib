package alphix

import (
	"math"
	"math/big"
	"sync"
	"testing"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// jitHook is TestBeforeSwap_JitSimulation's stablecoin fixture at the given price.
func jitHook(sqrtPriceX96 *uint256.Int) *Hook {
	return &Hook{
		Hook:             &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alphix},
		SqrtPriceX96:     sqrtPriceX96,
		SwapFee:          3000,
		TickLower:        -8,
		TickUpper:        8,
		Amount0Available: uint256.NewInt(1_000_000_000_000),
		Amount1Available: uint256.NewInt(1_000_000_000_000),
		PoolManagerBalances: [2]*uint256.Int{
			uint256.NewInt(1_000_000_000),
			uint256.NewInt(1_000_000_000),
		},
	}
}

func q96() *uint256.Int { return new(uint256.Int).Lsh(uint256.NewInt(1), 96) }

// factoryHook builds the fixture through the registered factory, as the pool
// simulator does.
func factoryHook(t *testing.T) *Hook {
	t.Helper()
	extra, err := json.Marshal(jitHook(nil))
	require.NoError(t, err)
	tickExtra, err := json.Marshal(map[string]any{"sqrtPriceX96": q96().Dec(), "liquidity": "0", "tick": 0})
	require.NoError(t, err)
	h, ok := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{
		HookExtra: extra,
		Pool:      &entity.Pool{Extra: string(tickExtra)},
	})
	require.True(t, ok)
	hk := h.(*Hook)
	require.NotNil(t, hk.SqrtPriceX96, "fixture: factory must read sqrtPriceX96")
	return hk
}

func beforeSwap(h *Hook, zeroForOne, calcOut bool, amt *big.Int) (*uniswapv4.BeforeSwapResult, error) {
	return h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: calcOut, ZeroForOne: zeroForOne, AmountSpecified: amt})
}

// TestBeforeSwap_LeavesHookUnchanged: a quote must not write the shared hook.
func TestBeforeSwap_LeavesHookUnchanged(t *testing.T) {
	h := jitHook(q96())
	before := *h
	_, err := beforeSwap(h, true, true, big.NewInt(1_000_000_000))
	require.NoError(t, err)
	assert.Equal(t, before, *h, "BeforeSwap wrote the hook")
}

// TestBeforeSwap_ConcurrentSharedHook quotes one hook from many goroutines; run with -race.
func TestBeforeSwap_ConcurrentSharedHook(t *testing.T) {
	want, err := beforeSwap(jitHook(q96()), true, true, big.NewInt(1_000_000_000))
	require.NoError(t, err)
	shared := jitHook(q96())
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 50 {
				res, err := beforeSwap(shared, true, true, big.NewInt(1_000_000_000))
				if !assert.NoError(t, err) || !assert.Equal(t, want.DeltaSpecified.String(), res.DeltaSpecified.String()) {
					return
				}
			}
		})
	}
	wg.Wait()
}

// TestFactory_BuildsJitLiquidity: the simulator's hook carries its JIT
// liquidity from construction, so quotes only read it.
func TestFactory_BuildsJitLiquidity(t *testing.T) {
	h := factoryHook(t)
	require.NotNil(t, h.jitLiquidity)
	want, err := beforeSwap(jitHook(q96()), true, true, big.NewInt(1_000_000_000))
	require.NoError(t, err)
	got, err := beforeSwap(h, true, true, big.NewInt(1_000_000_000))
	require.NoError(t, err)
	assert.Equal(t, want.DeltaSpecified.String(), got.DeltaSpecified.String())
	assert.Equal(t, want.DeltaUnspecified.String(), got.DeltaUnspecified.String())
}

// The JIT liquidity the factory builds, against Uniswap's LiquidityAmounts.getLiquidityForAmounts written here in
// math/big apart from the hook: inside the range it is min(mulDiv(amount0, mulDiv(sqrtP, sqrtU, Q96), sqrtU - sqrtP),
// mulDiv(amount1, Q96, sqrtP - sqrtL)). The range's sqrt prices are TickMath's (checked against 1.0001^(tick/2)).
// This does not come from the hook.
func TestFactory_JitLiquidityMatchesLiquidityAmounts(t *testing.T) {
	hk := factoryHook(t)
	require.NotNil(t, hk.jitLiquidity)
	var lower, upper uint256.Int
	require.NoError(t, v3Utils.GetSqrtRatioAtTickV2(hk.TickLower, &lower))
	require.NoError(t, v3Utils.GetSqrtRatioAtTickV2(hk.TickUpper, &upper))
	q := q96().ToBig()
	for tick, s := range map[int]*uint256.Int{hk.TickLower: &lower, hk.TickUpper: &upper} {
		got, _ := new(big.Float).Quo(new(big.Float).SetInt(s.ToBig()), new(big.Float).SetInt(q)).Float64()
		assert.InDelta(t, math.Pow(1.0001, float64(tick)/2), got, 1e-12, "sqrt price at tick %d", tick)
	}
	p, l, u := hk.SqrtPriceX96.ToBig(), lower.ToBig(), upper.ToBig()
	mulDiv := func(a, b, c *big.Int) *big.Int { return new(big.Int).Div(new(big.Int).Mul(a, b), c) }
	liq0 := mulDiv(hk.Amount0Available.ToBig(), mulDiv(p, u, q), new(big.Int).Sub(u, p))
	liq1 := mulDiv(hk.Amount1Available.ToBig(), q, new(big.Int).Sub(p, l))
	want := liq0
	if liq1.Cmp(liq0) < 0 {
		want = liq1
	}
	assert.Equal(t, want.String(), hk.jitLiquidity.Dec())
}
