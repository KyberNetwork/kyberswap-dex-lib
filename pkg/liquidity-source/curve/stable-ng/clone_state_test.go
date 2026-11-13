package stableng

import (
	"math/big"
	"sync"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// cloneTestPool is a base pool of curve-stable-meta-ng pools; a meta <-> base swap applied to a clone of a
// meta pool reaches the clone of its base pool through ApplyAddLiquidity / ApplyRemoveLiquidityOneCoinU256.
const cloneTestPool = `{"address":"0x383e6b4437b59fff47b619cba855ca29342a8559","exchange":"curve-stable-ng","type":"curve-stable-ng","timestamp":1710382680,"reserves":["21024903652839","16240730126117","37260809758180318203561662"],"tokens":[{"address":"0x6c3ea9036406852006290770bedfcaba0e23a0e8","symbol":"PYUSD","decimals":6,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"InitialA\":\"15000\",\"FutureA\":\"15000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"1000000\",\"AdminFee\":\"5000000000\",\"OffpegFeeMultiplier\":\"50000000000\",\"RateMultipliers\":[\"1000000000000000000000000000000\",\"1000000000000000000000000000000\"]}","staticExtra":"{\"APrecision\":\"100\",\"IsNativeCoins\":[false,false]}","blockNumber":19430235}`

func newClonePool(t *testing.T) *PoolSimulator {
	t.Helper()
	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(cloneTestPool), &ep))
	p, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	return p
}

func cloneReserveStrings(rs []*big.Int) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.String()
	}
	return out
}

// cloneBaseOps are the base-pool operations that write a pool's reserves, each applied to p.
var cloneBaseOps = map[string]func(t *testing.T, p *PoolSimulator){
	"ApplyRemoveLiquidityOneCoinU256": func(t *testing.T, p *PoolSimulator) {
		tokenAmount := uint256.MustFromDecimal("1000000000000000000000")
		var dy, dyFee uint256.Int
		if err := p.CalculateWithdrawOneCoinU256(tokenAmount, 0, &dy, &dyFee); err != nil {
			t.Error(err)
			return
		}
		if err := p.ApplyRemoveLiquidityOneCoinU256(0, tokenAmount, &dy, &dyFee); err != nil {
			t.Error(err)
		}
	},
	"ApplyAddLiquidity": func(t *testing.T, p *PoolSimulator) {
		n := len(p.GetTokens())
		amounts, fees := make([]uint256.Int, n), make([]uint256.Int, n)
		amounts[0].Set(uint256.MustFromDecimal("1000000"))
		var mint uint256.Int
		if err := p.CalculateTokenAmountU256(amounts, true, &mint, fees); err != nil {
			t.Error(err)
			return
		}
		if err := p.ApplyAddLiquidity(amounts, fees, &mint); err != nil {
			t.Error(err)
		}
	},
}

// An operation applied to a clone moves the clone's reserves (GetReserves) and leaves the original's as they were.
func TestCloneStateLeavesTheOriginalReserves(t *testing.T) {
	for name, op := range cloneBaseOps {
		t.Run(name, func(t *testing.T) {
			orig := newClonePool(t)
			before := cloneReserveStrings(orig.GetReserves())

			clone := orig.CloneState().(*PoolSimulator)
			op(t, clone)

			require.NotEqual(t, before, cloneReserveStrings(clone.GetReserves()), "the clone's reserves did not move")
			require.Equal(t, before, cloneReserveStrings(orig.GetReserves()), "an operation on a clone moved the original's reserves")
		})
	}
}

// Operations applied to clones of one pool on two goroutines, as two requests of a router do, share no writes:
// under -race this test reports none.
func TestCloneStateOperationsOnClonesConcurrently(t *testing.T) {
	for name, op := range cloneBaseOps {
		t.Run(name, func(t *testing.T) {
			orig := newClonePool(t)
			var wg sync.WaitGroup
			for range 2 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for range 50 {
						op(t, orig.CloneState().(*PoolSimulator))
					}
				}()
			}
			wg.Wait()
		})
	}
}

// BenchmarkCloneState measures CloneState alone, as a route finder calls it before each UpdateBalance.
func BenchmarkCloneState(b *testing.B) {
	p := newClonePool(&testing.T{})
	b.ReportAllocs()
	for b.Loop() {
		_ = p.CloneState()
	}
}

// BenchmarkCloneSwapUpdate is a clone, a quote and an UpdateBalance on the clone.
func BenchmarkCloneSwapUpdate(b *testing.B) {
	p := newClonePool(&testing.T{})
	tokens := p.GetTokens()
	in := pool.TokenAmount{Token: tokens[0], Amount: new(big.Int).Div(p.GetReserves()[0], big.NewInt(1000))}
	b.ReportAllocs()
	for b.Loop() {
		c := p.CloneState()
		res, err := c.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: in, TokenOut: tokens[1]})
		if err != nil {
			b.Fatal(err)
		}
		c.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: in, TokenAmountOut: *res.TokenAmountOut,
			Fee: *res.Fee, SwapInfo: res.SwapInfo})
	}
}
