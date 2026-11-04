package alphix

import (
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"testing"

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

const identityGolden = "testdata/before_swap_identity.golden"

// TestBeforeSwap_Identity pins BeforeSwap results (a fresh hook per quote) on
// the JIT fixture at three prices, both directions, exact in and out.
// Regenerate with UPDATE_GOLDEN=1.
func TestBeforeSwap_Identity(t *testing.T) {
	prices := map[string]*uint256.Int{
		"tick0":   q96(),
		"low":     new(uint256.Int).Div(new(uint256.Int).Mul(q96(), uint256.NewInt(9997)), uint256.NewInt(10000)),
		"high":    new(uint256.Int).Div(new(uint256.Int).Mul(q96(), uint256.NewInt(10003)), uint256.NewInt(10000)),
		"outside": new(uint256.Int).Mul(q96(), uint256.NewInt(2)),
	}
	names := []string{"tick0", "low", "high", "outside"}
	amounts := []*big.Int{big.NewInt(1)}
	for e := 3; e <= 15; e += 2 {
		amounts = append(amounts, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(e)), nil))
	}
	var b strings.Builder
	for _, n := range names {
		for _, zfo := range []bool{true, false} {
			for _, calcOut := range []bool{true, false} {
				for _, amt := range amounts {
					res, err := beforeSwap(jitHook(prices[n]), zfo, calcOut, amt)
					fmt.Fprintf(&b, "%s zfo=%v out=%v %s => ", n, zfo, calcOut, amt)
					if err != nil {
						fmt.Fprintf(&b, "err %v\n", err)
						continue
					}
					fmt.Fprintf(&b, "fee %d ds %s du %s gas %d next %v\n", res.SwapFee, res.DeltaSpecified, res.DeltaUnspecified, res.Gas, res.SwapInfo)
				}
			}
		}
	}
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(identityGolden, []byte(b.String()), 0o644))
	}
	want, err := os.ReadFile(identityGolden)
	require.NoError(t, err)
	require.Equal(t, string(want), b.String())
}
