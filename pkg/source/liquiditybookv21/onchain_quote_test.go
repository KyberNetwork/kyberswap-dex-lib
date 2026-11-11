package liquiditybookv21

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// onchainQuote is what a pair's getSwapOut or getSwapIn returned at one block.
type onchainQuote struct {
	Kind     string `json:"kind"` // "out": getSwapOut of an amount in; "in": getSwapIn of an amount out
	Amount   string `json:"amount"`
	SwapForY bool   `json:"swapForY"`
	Result   string `json:"result"`
}

func (q onchainQuote) String() string {
	if q.Kind == "in" {
		return fmt.Sprintf("%s out, swapForY %v", q.Amount, q.SwapForY)
	}
	return fmt.Sprintf("%s in, swapForY %v", q.Amount, q.SwapForY)
}

type onchainFixture struct {
	Pool   entity.Pool    `json:"pool"`
	Quotes []onchainQuote `json:"quotes"`
}

// loadFixtures reads each JSON file that pattern matches, keyed by its name.
func loadFixtures[T any](tb testing.TB, pattern string) map[string]T {
	files, err := filepath.Glob(pattern)
	require.NoError(tb, err)
	require.NotEmpty(tb, files)
	fixtures := make(map[string]T, len(files))
	for _, file := range files {
		b, err := os.ReadFile(file)
		require.NoError(tb, err)
		var fixture T
		require.NoError(tb, json.Unmarshal(b, &fixture), file)
		fixtures[filepath.Base(file)] = fixture
	}
	return fixtures
}

// assertQuotes checks that sim quotes each amount as the chain did, to the wei.
func assertQuotes(t *testing.T, sim *PoolSimulator, quotes []onchainQuote) {
	t.Helper()
	tokenX, tokenY := sim.GetTokens()[0], sim.GetTokens()[1]
	for _, q := range quotes {
		in, out := tokenX, tokenY
		if !q.SwapForY {
			in, out = tokenY, tokenX
		}
		amount := bignumber.NewBig10(q.Amount)
		var got *big.Int
		var err error
		switch q.Kind {
		case "out":
			var res *pool.CalcAmountOutResult
			if res, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: in, Amount: amount}, TokenOut: out}); err == nil {
				got = res.TokenAmountOut.Amount
			}
		case "in":
			var res *pool.CalcAmountInResult
			if res, err = sim.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{Token: out, Amount: amount}, TokenIn: in}); err == nil {
				got = res.TokenAmountIn.Amount
			}
		default:
			t.Fatalf("quote kind %q", q.Kind)
		}
		if assert.NoError(t, err, q) {
			assert.Equal(t, q.Result, got.String(), q)
		}
	}
}

// TestCalcAmountMatchesChain: each fixture holds a pair's state at one block, read through its
// getters, and what the pair's own getSwapOut and getSwapIn returned at that block, for amounts from
// one wei to the pair's depth, both ways. The pairs cover every v2.1 and v2.2 fork we track
// (TraderJoe, Pharaoh, Metropolis, SectorOne, Swapline, BeanExchange, HyperBrick, E3),
// and some have an empty active bin, kept in their set. The simulator must quote each to the wei.
func TestCalcAmountMatchesChain(t *testing.T) {
	for name, fixture := range loadFixtures[onchainFixture](t, "testdata/onchain/*.json") {
		t.Run(name, func(t *testing.T) {
			sim, err := NewPoolSimulator(fixture.Pool)
			require.NoError(t, err)

			assertQuotes(t, sim, fixture.Quotes)
		})
	}
}

// BenchmarkCalcAmountOnchain quotes a Metropolis pair of 125 bins, with an empty active bin, across
// many bins one way and within one bin the other, by amount in and by amount out.
func BenchmarkCalcAmountOnchain(b *testing.B) {
	fixtures := loadFixtures[onchainFixture](b, "testdata/onchain/*.json")
	fixture := fixtures["sonic-0xb7c1b259af90accd4230d062458d747ac4774563.json"]
	sim, err := NewPoolSimulator(fixture.Pool)
	require.NoError(b, err)
	tokenX, tokenY := fixture.Pool.Tokens[0].Address, fixture.Pool.Tokens[1].Address

	var deepest, small *big.Int
	for _, q := range fixture.Quotes {
		if q.Kind != "out" || q.Result == "0" {
			continue
		}
		a := bignumber.NewBig10(q.Amount)
		if deepest == nil || a.Cmp(deepest) > 0 {
			deepest = a
		}
		if small == nil || a.Cmp(small) < 0 {
			small = a
		}
	}
	require.NotNil(b, deepest)
	out, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: deepest}, TokenOut: tokenY})
	if err != nil {
		out, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenY, Amount: deepest}, TokenOut: tokenX})
		tokenX, tokenY = tokenY, tokenX
	}
	require.NoError(b, err)

	b.Run("CalcAmountOut", func(b *testing.B) {
		b.ReportAllocs()
		params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: deepest},
			TokenOut: tokenY}
		for b.Loop() {
			if _, err := sim.CalcAmountOut(params); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("CalcAmountIn", func(b *testing.B) {
		b.ReportAllocs()
		params := pool.CalcAmountInParams{TokenAmountOut: *out.TokenAmountOut, TokenIn: tokenX}
		for b.Loop() {
			if _, err := sim.CalcAmountIn(params); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("CalcAmountOutOneBin", func(b *testing.B) {
		b.ReportAllocs()
		params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: small},
			TokenOut: tokenY}
		for b.Loop() {
			_, _ = sim.CalcAmountOut(params)
		}
	})
}
