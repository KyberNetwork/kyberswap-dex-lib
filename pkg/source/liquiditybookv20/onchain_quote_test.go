package liquiditybookv20

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// onchainQuote is what LBRouter.getSwapOut returned for a pair at one block.
type onchainQuote struct {
	AmountIn  string `json:"amountIn"`
	SwapForY  bool   `json:"swapForY"`
	AmountOut string `json:"amountOut"`
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

// assertQuotes checks that sim quotes each amount in as the chain did, to the wei.
func assertQuotes(t *testing.T, sim *PoolSimulator, quotes []onchainQuote) {
	t.Helper()
	tokenX, tokenY := sim.GetTokens()[0], sim.GetTokens()[1]
	for _, q := range quotes {
		in, out := tokenX, tokenY
		if !q.SwapForY {
			in, out = tokenY, tokenX
		}
		amountIn, _ := new(big.Int).SetString(q.AmountIn, 10)
		got, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: in, Amount: amountIn},
			TokenOut:      out,
		})
		if assert.NoError(t, err, "%s in, swapForY %v", q.AmountIn, q.SwapForY) {
			assert.Equal(t, q.AmountOut, got.TokenAmountOut.Amount.String(), "%s in, swapForY %v",
				q.AmountIn, q.SwapForY)
		}
	}
}

// TestCalcAmountOutMatchesChain: each fixture holds a pair's state at one block, read through its
// getters, and what TraderJoe's LBRouter.getSwapOut quoted for it at that block, for amounts from a
// few wei to the pair's depth, both ways. The pairs are TraderJoe's on Arbitrum and Avalanche and
// SectorOne's on Base, MegaETH and Robinhood, which run the same pair code; five of the nine have an
// empty active bin, kept in their set. The simulator must quote each amount to the wei.
func TestCalcAmountOutMatchesChain(t *testing.T) {
	type onchainFixture struct {
		Pool   entity.Pool    `json:"pool"`
		Quotes []onchainQuote `json:"quotes"`
	}
	for name, fixture := range loadFixtures[onchainFixture](t, "testdata/onchain/*.json") {
		t.Run(name, func(t *testing.T) {
			sim, err := NewPoolSimulator(fixture.Pool)
			require.NoError(t, err)
			assertQuotes(t, sim, fixture.Quotes)
		})
	}
}
