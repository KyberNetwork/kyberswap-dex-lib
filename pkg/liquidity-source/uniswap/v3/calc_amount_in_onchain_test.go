package uniswapv3

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// onchainFixture is one pool at one block: its state read from the chain, the chain's QuoterV2
// answers for quotes by output, and the pool's own exact-in swap of each quoted input.
type onchainFixture struct {
	Chain  string         `json:"chain"`
	Block  uint64         `json:"block"`
	Quoter string         `json:"quoter"`
	Pool   entity.Pool    `json:"pool"`
	Quotes []onchainQuote `json:"quotes"`
}

type onchainQuote struct {
	ZeroForOne bool   `json:"zeroForOne"`
	AmountOut  string `json:"amountOut"`
	// ExactOut is quoteExactOutputSingle with no price limit, absent when it reverted, as it does
	// when the swap stops at the limit with output still owed: ExactOutRevert holds what it
	// reverted with, and ExactOutAtLimit the same quote with that limit given explicitly.
	ExactOut        *onchainExactOut `json:"exactOut"`
	ExactOutRevert  string           `json:"exactOutRevert"`
	ExactOutAtLimit *onchainExactOut `json:"exactOutAtLimit"`
	// ExactIn is quoteExactInputSingle of the input the exact-out quote asked. TickAfter and
	// LiquidityAfter come from the pool's swap of that input, run in an eth_call and stopped in its
	// callback, which paid the same output and left the same price.
	ExactIn struct {
		AmountIn          string `json:"amountIn"`
		AmountOut         string `json:"amountOut"`
		SqrtPriceX96After string `json:"sqrtPriceX96After"`
		TickAfter         int    `json:"tickAfter"`
		LiquidityAfter    string `json:"liquidityAfter"`
	} `json:"exactIn"`
}

func (q onchainQuote) String() string {
	return fmt.Sprintf("%s out, zeroForOne %v", q.AmountOut, q.ZeroForOne)
}

// tokens returns the quote's input and output tokens.
func (q onchainQuote) tokens(sim *PoolSimulator) (in, out string) {
	in, out = sim.Info.Tokens[0], sim.Info.Tokens[1]
	if !q.ZeroForOne {
		in, out = out, in
	}
	return in, out
}

type onchainExactOut struct {
	SqrtPriceLimitX96 string `json:"sqrtPriceLimitX96"`
	AmountIn          string `json:"amountIn"`
	SqrtPriceX96After string `json:"sqrtPriceX96After"`
}

func loadOnchainFixtures(t *testing.T) map[string]onchainFixture {
	t.Helper()
	files, err := filepath.Glob("testdata/onchain/*.json")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	fixtures := make(map[string]onchainFixture, len(files))
	for _, file := range files {
		b, err := os.ReadFile(file)
		require.NoError(t, err)
		var fixture onchainFixture
		require.NoError(t, json.Unmarshal(b, &fixture))
		fixtures[filepath.Base(file)] = fixture
	}
	return fixtures
}

func bigInt(t *testing.T, s string) *big.Int {
	t.Helper()
	v, ok := new(big.Int).SetString(s, 10)
	require.True(t, ok, s)
	return v
}

// quoteByOutput runs CalcAmountIn for the quote's output. ok is false for an output the ticks do
// not pay when the walk runs past the pool's outermost tick before the price limit, as on a pool
// whose outermost ticks are not at MinTick and MaxTick: the simulator refuses that quote.
func quoteByOutput(t *testing.T, sim *PoolSimulator, q onchainQuote) (res *pool.CalcAmountInResult, ok bool) {
	t.Helper()
	in, out := q.tokens(sim)
	res, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: out, Amount: bigInt(t, q.AmountOut)}, TokenIn: in})
	if err != nil && q.ExactOut == nil && (errors.Is(err, ErrBelowSmallest) || errors.Is(err, ErrAtOrAboveLargest)) {
		return nil, false
	}
	require.NoError(t, err, "%s", q)
	return res, true
}

// floatWalkTolerance bounds how far a quote can stray from the chain's when its walk prices a long
// run of empty words in float64 (walkWordsFloat), as every walk to the price limit does over a
// full-range position: 1e-12 of the amount, and only against the swapper. The largest seen in these
// fixtures is 2.4e-13, most are near floatBias.
var floatWalkTolerance = big.NewInt(1e12)

// assertAgainstSwapper checks that got, the quote's amount named by what, errs from the chain's want
// by at most floatWalkTolerance, and only against the swapper: more when got is an input, less when
// it is an output.
func assertAgainstSwapper(t *testing.T, q onchainQuote, what string, want, got *big.Int, input bool) {
	t.Helper()
	diff := new(big.Int).Sub(got, want)
	if !input {
		diff.Neg(diff)
	}
	if diff.Sign() < 0 || new(big.Int).Mul(diff, floatWalkTolerance).Cmp(want) > 0 {
		assert.Fail(t, "off the chain's amount, or for the swapper",
			"%s: %s: chain %s, simulator %s", q, what, want, got)
	}
}

// TestCalcAmountInMatchesQuoterV2: each fixture holds a pool of a type this package simulates, with
// its ticks, price and balances at one block, and what the chain's QuoterV2 answered at that block
// for outputs from one wei to the pool's balance of the token, both ways. CalcAmountIn asks the input
// the quoter asks, and pays an output the quoter pays whole. Past what the ticks pay, though the pool
// still holds the tokens, the quoter reverts, as the swap reaches the price limit with output still
// owed: CalcAmountIn reports that part as unpaid, and the part it pays is what the chain's exact-in
// swap of the input to the limit pays. Where a walk prices empty words in float64 the quote errs
// against the swapper within floatWalkTolerance, so an output at the edge of what the ticks pay can
// show a few wei unpaid.
func TestCalcAmountInMatchesQuoterV2(t *testing.T) {
	var beyond int
	for name, fixture := range loadOnchainFixtures(t) {
		t.Run(name, func(t *testing.T) {
			sim, err := NewPoolSimulator(fixture.Pool, 0)
			require.NoError(t, err)
			var exact int
			for _, q := range fixture.Quotes {
				res, ok := quoteByOutput(t, sim, q)
				if !ok {
					continue
				}
				amountOut, in, unpaid := bigInt(t, q.AmountOut), res.TokenAmountIn.Amount, new(big.Int)
				if res.RemainingTokenAmountOut != nil {
					unpaid = res.RemainingTokenAmountOut.Amount
				}
				paid := new(big.Int).Sub(amountOut, unpaid)

				if q.ExactOut != nil {
					chainIn := bigInt(t, q.ExactOut.AmountIn)
					switch {
					case in.Cmp(chainIn) == 0:
						exact++
						assert.Zero(t, unpaid.Sign(), "%s: the quoter pays it whole, unpaid %s", q, unpaid)
					case unpaid.Sign() == 0:
						assertAgainstSwapper(t, q, "input", chainIn, in, true)
					default:
						assertAgainstSwapper(t, q, "paid, where the quoter pays it whole", amountOut, paid, false)
					}
					continue
				}

				beyond++
				require.NotNil(t, q.ExactOutAtLimit, "%s: the fixture has no quote at the limit", q)
				require.Positive(t, unpaid.Sign(), "%s: the quoter reverts, nothing is unpaid", q)
				assertAgainstSwapper(t, q, "input to the price limit", bigInt(t, q.ExactOutAtLimit.AmountIn), in, true)
				assertAgainstSwapper(t, q, "paid, by the chain's exact in of its input to the limit",
					bigInt(t, q.ExactIn.AmountOut), paid, false)
			}
			assert.Positive(t, exact, "no quote matches the quoter to the wei")
		})
	}
	assert.Positive(t, beyond, "no fixture has an output past what its ticks pay")
}

// TestCalcAmountInNextStateMatchesChain: a swap is always executed exact in, so after a quote by
// output the pool is where its own exact-in swap of the quoted input leaves it on the chain, and
// that price, tick and liquidity are what UpdateBalance sets. Each quote whose input matches the
// quoter's to the wei is checked, which leaves out those a float64 walk prices.
func TestCalcAmountInNextStateMatchesChain(t *testing.T) {
	for name, fixture := range loadOnchainFixtures(t) {
		t.Run(name, func(t *testing.T) {
			sim, err := NewPoolSimulator(fixture.Pool, 0)
			require.NoError(t, err)
			var checked int
			for _, q := range fixture.Quotes {
				res, ok := quoteByOutput(t, sim, q)
				if !ok || res.TokenAmountIn.Amount.String() != q.ExactIn.AmountIn {
					continue
				}
				checked++

				_, out := q.tokens(sim)
				next := sim.CloneState().(*PoolSimulator)
				next.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  *res.TokenAmountIn,
					TokenAmountOut: pool.TokenAmount{Token: out, Amount: bigInt(t, q.ExactIn.AmountOut)},
					SwapInfo:       res.SwapInfo,
				})
				assert.Equal(t, q.ExactIn.SqrtPriceX96After, next.V3Pool.SqrtRatioX96.Dec(), "%s: price", q)
				assert.Equal(t, q.ExactIn.TickAfter, next.V3Pool.TickCurrent, "%s: tick", q)
				assert.Equal(t, q.ExactIn.LiquidityAfter, next.V3Pool.Liquidity.Dec(), "%s: liquidity", q)
			}
			assert.Greater(t, checked, len(fixture.Quotes)/2, "quotes checked against the chain")
		})
	}
}

// TestExactInEndMatchesWholeSwap: GetInputAmountExactInEnd redoes only the last step of the
// exact-out walk exact in. For every output of the fixtures, and 20 fractions of each, its end must
// be the whole exact-in swap's, unless a run of words was priced in float, where the two walks carry
// different float errors.
func TestExactInEndMatchesWholeSwap(t *testing.T) {
	for name, fixture := range loadOnchainFixtures(t) {
		t.Run(name, func(t *testing.T) {
			sim, err := NewPoolSimulator(fixture.Pool, 0)
			require.NoError(t, err)
			checked := 0
			for _, q := range fixture.Quotes {
				full := bigInt(t, q.AmountOut)
				for i := int64(1); i <= 21; i++ {
					var out uint256.Int
					if out.SetFromBig(new(big.Int).Div(new(big.Int).Mul(full, big.NewInt(i)), big.NewInt(21))); out.IsZero() {
						continue
					}
					res, end, err := sim.V3Pool.GetInputAmountExactInEnd(q.ZeroForOne, out, uint256.Int{})
					if err != nil || res.CrossEmptyWordLoops >= floatWalkAfterWords {
						continue
					}
					whole, err := sim.V3Pool.Swap(q.ZeroForOne, res.AmountCalculated, uint256.Int{})
					require.NoError(t, err)
					assert.Equal(t, whole.SqrtRatioX96.Dec(), end.SqrtRatioX96.Dec(), "%s out, price", out.Dec())
					assert.Equal(t, whole.Liquidity.Dec(), end.Liquidity.Dec(), "%s out, liquidity", out.Dec())
					assert.Equal(t, whole.CurrentTick, end.CurrentTick, "%s out, tick", out.Dec())
					checked++
				}
			}
			require.Positive(t, checked)
		})
	}
}
