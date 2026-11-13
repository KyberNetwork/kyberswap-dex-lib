package uniswapv3

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// priceLimitFixture is one pool at one block and the chain's QuoterV2 quoteExactInputSingle answers
// for exact-in swaps with an explicit sqrtPriceLimitX96 and with 0.
type priceLimitFixture struct {
	Chain  string      `json:"chain"`
	Block  uint64      `json:"block"`
	Quoter string      `json:"quoter"`
	Pool   entity.Pool `json:"pool"`
	Quotes []struct {
		ZeroForOne        bool   `json:"zeroForOne"`
		AmountIn          string `json:"amountIn"`
		SqrtPriceLimitX96 string `json:"sqrtPriceLimitX96"`
		AmountOut         string `json:"amountOut"`
	} `json:"quotes"`
}

// TestEdgePriceLimitsQuoteLikeTheChain quotes the Ethereum pool 0xa3c40b2c (fee 100, spacing 1,
// only the full-range ticks) at block 26373319 with the limits routers send for no limit:
// MIN_SQRT_RATIO + 1 and MAX_SQRT_RATIO - 1, the extremes the pool accepts. The swaps walk enough
// empty words that the float word walk runs, and it converted the limit to a tick with
// GetTickAtSqrtRatio, which refused MAX_SQRT_RATIO - 1: TickMath takes any price below
// MAX_SQRT_RATIO. The quote failed with "invalid sqrt ratio" while a limit of 0, which the swap
// turns into the same value after that check, quoted. Each quote must give what the chain's
// QuoterV2 gave, less at most the float walk's bias against the swapper.
func TestEdgePriceLimitsQuoteLikeTheChain(t *testing.T) {
	b, err := os.ReadFile("testdata/pricelimit/ethereum-0xa3c40b2c9489da8d38d8a0bf12d52894f0359bce.json")
	require.NoError(t, err)
	var fixture priceLimitFixture
	require.NoError(t, json.Unmarshal(b, &fixture))
	sim, err := NewPoolSimulator(fixture.Pool, valueobject.ChainIDEthereum)
	require.NoError(t, err)

	for _, q := range fixture.Quotes {
		in, out := sim.Info.Tokens[0], sim.Info.Tokens[1]
		if !q.ZeroForOne {
			in, out = out, in
		}
		amountIn, ok := new(big.Int).SetString(q.AmountIn, 10)
		require.True(t, ok)
		res, err := sim.CalcAmountOutWithPriceLimit(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: in, Amount: amountIn}, TokenOut: out},
			*uint256.MustFromDecimal(q.SqrtPriceLimitX96))
		require.NoError(t, err, "zeroForOne %v, limit %s", q.ZeroForOne, q.SqrtPriceLimitX96)

		chain, ok := new(big.Int).SetString(q.AmountOut, 10)
		require.True(t, ok)
		got := res.TokenAmountOut.Amount
		require.LessOrEqual(t, got.Cmp(chain), 0, "quote %s above the chain's %s", got, chain)
		short := new(big.Float).Quo(new(big.Float).SetInt(new(big.Int).Sub(chain, got)), new(big.Float).SetInt(chain))
		shortF, _ := short.Float64()
		require.Less(t, shortF, 1e-12, "quote %s, chain %s", got, chain)
	}
}
