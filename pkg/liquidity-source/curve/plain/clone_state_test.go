package plain

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// cloneTestPool is a base pool of curve-stable-meta-ng pools; a meta <-> base swap applied to a clone of a
// meta pool reaches the clone of its base pool through ApplyAddLiquidity / ApplyRemoveLiquidityOneCoinU256.
const cloneTestPool = `{"address":"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7","exchange":"curve-stable-plain","type":"curve-stable-plain","timestamp":1710405854,"reserves":["74882317978601283428112533","76066551886323","32115318520985","177637651221630809031052488"],"tokens":[{"address":"0x6b175474e89094c44da98b954eedeac495271d0f","symbol":"DAI","decimals":18,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"InitialA\":\"5000\",\"FutureA\":\"2000\",\"InitialATime\":1653559305,\"FutureATime\":1654158027,\"SwapFee\":\"1000000\",\"AdminFee\":\"5000000000\"}","staticExtra":"{\"APrecision\":\"1\",\"LpToken\":\"0x6c3F90f043a72FA612cbac8115EE7e52BDe6E490\",\"IsNativeCoin\":[false,false,false]}","blockNumber":19432140}`

func newClonePool(t *testing.T) *PoolSimulator {
	t.Helper()
	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(cloneTestPool), &ep))
	p, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	return p
}

// cloneBaseOps are the base-pool operations that write a pool's reserves, each applied to p.
var cloneBaseOps = map[string]func(t testing.TB, p *PoolSimulator){
	"ApplyRemoveLiquidityOneCoinU256": func(t testing.TB, p *PoolSimulator) {
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
	"ApplyAddLiquidity": func(t testing.TB, p *PoolSimulator) {
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
	"AddLiquidityU256": func(t testing.TB, p *PoolSimulator) {
		amounts := make([]uint256.Int, len(p.GetTokens()))
		amounts[0].Set(uint256.MustFromDecimal("1000000000000000000000"))
		if _, err := p.AddLiquidityU256(amounts); err != nil {
			t.Error(err)
		}
	},
	"RemoveLiquidityOneCoinU256": func(t testing.TB, p *PoolSimulator) {
		if _, err := p.RemoveLiquidityOneCoinU256(uint256.MustFromDecimal("1000000000000000000000"), 0); err != nil {
			t.Error(err)
		}
	},
}

// An operation on a clone changes only the clone, also while clones change on several goroutines and the original
// is read.
func TestCloneStateIsolatesBaseOps(t *testing.T) {
	ops := make(map[string]testutil.CloneOp, len(cloneBaseOps))
	for name, op := range cloneBaseOps {
		ops[name] = func(t testing.TB, sim pool.IPoolSimulator) { op(t, sim.(*PoolSimulator)) }
	}
	testutil.TestCloneIsolation(t, newClonePool(t), ops)
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
