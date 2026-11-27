package kalos_test

import (
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/kalos"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// testdata/pools.json is both live Kalos V4 pools as the v4 factory lists them and the v4 tracker stores them (at one
// block, Timestamp that block's), with what Elysium's V4Quoter quoted for each at that block; TestLiveQuotesMatchV4Quoter
// writes it (KALOS_WRITE_FIXTURE=1).
type recordedPool struct {
	Name   string          `json:"name"`
	Pool   entity.Pool     `json:"pool"`
	Quotes []recordedQuote `json:"quotes"`
}

type recordedQuote struct {
	ZeroForOne bool   `json:"zeroForOne"`
	AmountIn   string `json:"amountIn"`
	AmountOut  string `json:"amountOut"`
}

func loadFixture(t *testing.T) []recordedPool {
	t.Helper()
	raw, err := os.ReadFile("testdata/pools.json")
	require.NoError(t, err)
	var pools []recordedPool
	require.NoError(t, json.Unmarshal(raw, &pools))
	require.Len(t, pools, 2)
	return pools
}

func amountOut(t *testing.T, sim pool.IPoolSimulator, p entity.Pool, q recordedQuote) (*pool.CalcAmountOutResult, error) {
	t.Helper()
	amountIn, ok := new(big.Int).SetString(q.AmountIn, 10)
	require.True(t, ok)
	tokenIn, tokenOut := p.Tokens[1].Address, p.Tokens[0].Address
	if q.ZeroForOne {
		tokenIn, tokenOut = tokenOut, tokenIn
	}
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn}, TokenOut: tokenOut})
}

// The simulator quotes every recorded V4Quoter answer to the wei, through the msgpack cache too.
func TestFixture_MatchesV4Quoter(t *testing.T) {
	t.Parallel()
	for _, fp := range loadFixture(t) {
		sim, err := uniswapv4.NewPoolSimulator(fp.Pool, valueobject.ChainIDElysium)
		require.NoError(t, err)
		require.Equal(t, string(kalos.DexType), sim.GetExchange())
		sims := map[string]pool.IPoolSimulator{fp.Pool.Address: sim}
		for range 2 {
			encoded, err := msgpack.EncodePoolSimulatorsMap(sims)
			require.NoError(t, err)
			sims, err = msgpack.DecodePoolSimulatorsMap(encoded)
			require.NoError(t, err)
		}
		require.NotEmpty(t, fp.Quotes)
		for _, q := range fp.Quotes {
			// The decoded simulator has lost the indexing clock (an unexported field), so it is not compared at the
			// fixture's block time; the original one is.
			res, err := amountOut(t, sim, fp.Pool, q)
			require.NoError(t, err)
			require.Equal(t, q.AmountOut, res.TokenAmountOut.Amount.String(), "%s zfo=%v in=%s", fp.Name,
				q.ZeroForOne, q.AmountIn)
			require.Positive(t, res.Gas)
		}
		decoded := sims[fp.Pool.Address]
		require.Equal(t, string(kalos.DexType), decoded.GetExchange())
		_, err = amountOut(t, decoded, fp.Pool, fp.Quotes[1])
		require.NoError(t, err, "the cached simulator keeps the hook's state")
	}
}

// CalcAmountIn reverses the exact-in quote (the hook's fee inverted, never its exact-out branch).
func TestFixture_RoundTrip(t *testing.T) {
	t.Parallel()
	for _, fp := range loadFixture(t) {
		sim, err := uniswapv4.NewPoolSimulator(fp.Pool, valueobject.ChainIDElysium)
		require.NoError(t, err)
		for _, q := range fp.Quotes[1:] {
			tokenIn, tokenOut := fp.Pool.Tokens[1].Address, fp.Pool.Tokens[0].Address
			if q.ZeroForOne {
				tokenIn, tokenOut = tokenOut, tokenIn
			}
			out, _ := new(big.Int).SetString(q.AmountOut, 10)
			resIn, err := sim.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{Token: tokenOut, Amount: out}, TokenIn: tokenIn})
			require.NoError(t, err)
			in, _ := new(big.Int).SetString(q.AmountIn, 10)
			// The least input that delivers the output: at most the recorded input, and it delivers at least out.
			require.LessOrEqual(t, resIn.TokenAmountIn.Amount.Cmp(in), 0, "%s zfo=%v", fp.Name, q.ZeroForOne)
			back, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: *resIn.TokenAmountIn, TokenOut: tokenOut})
			require.NoError(t, err)
			require.GreaterOrEqual(t, back.TokenAmountOut.Amount.Cmp(out), 0, "%s zfo=%v", fp.Name, q.ZeroForOne)
		}
	}
}

// Quoting never moves the simulator; UpdateBalance moves only the clone it is given, and a later leg on that clone
// sees the volatility state the first swap left.
func TestFixture_CloneAndUpdateBalance(t *testing.T) {
	t.Parallel()
	for _, fp := range loadFixture(t) {
		sim, err := uniswapv4.NewPoolSimulator(fp.Pool, valueobject.ChainIDElysium)
		require.NoError(t, err)
		q := fp.Quotes[len(fp.Quotes)-1] // a tenth of the input-side reserve: the price moves
		first, err := amountOut(t, sim, fp.Pool, q)
		require.NoError(t, err)
		again, err := amountOut(t, sim, fp.Pool, q)
		require.NoError(t, err)
		require.Equal(t, first.TokenAmountOut.Amount, again.TokenAmountOut.Amount, "%s: quoting moved the pool", fp.Name)

		cloned := sim.CloneState()
		amountIn, _ := new(big.Int).SetString(q.AmountIn, 10)
		tokenIn := fp.Pool.Tokens[1].Address
		if q.ZeroForOne {
			tokenIn = fp.Pool.Tokens[0].Address
		}
		cloned.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
			TokenAmountOut: *first.TokenAmountOut, Fee: *first.Fee, SwapInfo: first.SwapInfo})
		info := first.SwapInfo.(uniswapv4.SwapInfo).HookSwapInfo.(*kalos.SwapInfo)
		require.Equal(t, uint64(fp.Pool.Timestamp), info.State.LastSwap, "%s", fp.Name)

		after, err := amountOut(t, sim, fp.Pool, q)
		require.NoError(t, err)
		require.Equal(t, first.TokenAmountOut.Amount, after.TokenAmountOut.Amount, "%s: the original moved", fp.Name)
		second, err := amountOut(t, cloned, fp.Pool, q)
		require.NoError(t, err)
		require.Negative(t, second.TokenAmountOut.Amount.Cmp(first.TokenAmountOut.Amount),
			"%s: the same swap again pays a worse price", fp.Name)
	}
}
