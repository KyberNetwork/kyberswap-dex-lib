package uniswapv4

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// avaxDustPool is the router-api payload for the Avalanche BTC.b/WETH.e 0.01% V4 pool, verbatim.
//
// It holds a single full-range position of 4787383 liquidity, which at the current price works out
// to reserves of 2 units of BTC.b and 1.05e13 of WETH.e — roughly five cents of inventory. That is
// what makes it the sharpest case for word-bounded stepping: one bitmap word of price movement
// costs a fraction of one unit of token0, so the on-chain per-word round-up of amountIn dominates
// everything else in the swap.
const avaxDustPool = `{
  "address": "0x19644199c9a151af99960a39a43686bac3522857b16a8f537a162e791dab0871",
  "swapFee": 100,
  "exchange": "uniswap-v4",
  "type": "uniswap-v4",
  "timestamp": 1772817016,
  "reserves": ["2", "10466736059725"],
  "tokens": [
    {"address":"0x152b9d0fdc40c096757f570a51e494bd4b943e50","symbol":"BTC.b","decimals":8,"swappable":true},
    {"address":"0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab","symbol":"WETH.e","decimals":18,"swappable":true}
  ],
  "extra": "{\"liquidity\":4787383,\"sqrtPriceX96\":173217865696950102619138552042419888,\"tickSpacing\":1,\"tick\":291969,\"ticks\":[{\"index\":-887272,\"liquidityGross\":4787383,\"liquidityNet\":4787383},{\"index\":887272,\"liquidityGross\":4787383,\"liquidityNet\":-4787383}]}",
  "staticExtra": "{\"0x0\":[false,false],\"fee\":100,\"tS\":1,\"hooks\":\"0x0000000000000000000000000000000000000000\",\"uR\":\"0x94b75331ae8d42c1b61065089b7d48fe14aa73b7\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}",
  "blockNumber": 79724099
}`

const (
	avaxBTCb  = "0x152b9d0fdc40c096757f570a51e494bd4b943e50"
	avaxWETHe = "0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab"
)

// TestDustPoolWordBoundedSteps pins the simulator to a PoolManager.swap that actually ran.
//
// Before word-bounded stepping the simulator jumped straight from tick 291969 to the only
// initialized tick below it (-887272), spent all 18 post-fee units of input on moving the price,
// and quoted 9331551613090 — 89% of the pool's entire token1 balance. The chain instead walked 9
// bitmap words, each costing 1 unit of input rounded up plus 1 unit of fee, exhausted the 19 units
// after 2178 ticks, and paid out 1079518536551. Routing on the old quote is what made the real
// transaction revert with "Return amount is not enough".
//
// Ground truth is Tenderly simulation 9a8a0d70-28e2-4222-b54f-0d04392f167a, whose Swap event reads
// amount0 -19, amount1 1079518536551, sqrtPriceX96 155352516287591836139164025014435689,
// tick 289791, liquidity 4787383.
func TestDustPoolWordBoundedSteps(t *testing.T) {
	t.Parallel()

	var poolEnt entity.Pool
	require.NoError(t, json.Unmarshal([]byte(avaxDustPool), &poolEnt))
	sim, err := NewPoolSimulator(poolEnt, valueobject.ChainIDAvalancheCChain)
	require.NoError(t, err)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: avaxBTCb, Amount: bignumber.NewBig10("19")},
		TokenOut:      avaxWETHe,
	})
	require.NoError(t, err)
	require.Equal(t, "1079518536551", res.TokenAmountOut.Amount.String())

	// Gas has to see the same work. This swap crosses no initialized tick at all, so under the
	// old model — which priced only initialized crossings — it came out as bare base gas, making
	// the pool look as cheap as it looked deep. The nine bitmap words it really walks are what
	// PoolManager charged 92773 gas for.
	require.Equal(t, defaultGas.BaseGas+9*uniswapv3.CrossEmptyWordGas, res.Gas,
		"nine empty-word crossings, no initialized ticks")
}

// TestDustPoolOutputIsBoundedByWordCost checks the property the on-chain loop enforces and the
// single-jump simulator broke: a bitmap word costs at least one unit of input plus its fee, so the
// price can only travel as far as the input can pay for word by word.
//
// Without word-bounded stepping all of these collapse toward the pool's whole token1 balance —
// 10 units alone quoted 8418510341968, 80% of it — which is the shape of the bug rather than of
// the pool.
//
// The expectations come from an independent transcription of V4's swap loop, checked against the
// Tenderly trace pinned in TestDustPoolWordBoundedSteps at 19 units in.
func TestDustPoolOutputIsBoundedByWordCost(t *testing.T) {
	t.Parallel()

	var poolEnt entity.Pool
	require.NoError(t, json.Unmarshal([]byte(avaxDustPool), &poolEnt))

	for _, tc := range []struct {
		amountIn string
		want     string
	}{
		{"2", "67380838795"},       // 1 word crossed, ends at tick 291840
		{"5", "199637723176"},      // 2 words, tick 291584
		{"10", "586401573504"},     // 5 words, tick 290816
		{"19", "1079518536551"},    // 9 words, tick 289792
		{"100", "4912420720935"},   // 50 words, tick 279296
		{"1000", "10399346476185"}, // 394 words, tick 191232
	} {
		t.Run(tc.amountIn, func(t *testing.T) {
			t.Parallel()

			sim, err := NewPoolSimulator(poolEnt, valueobject.ChainIDAvalancheCChain)
			require.NoError(t, err)

			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: avaxBTCb, Amount: bignumber.NewBig10(tc.amountIn)},
				TokenOut:      avaxWETHe,
			})
			require.NoError(t, err)
			require.Equal(t, tc.want, res.TokenAmountOut.Amount.String())
			require.Less(t, res.TokenAmountOut.Amount.Uint64(), uint64(10466736059725),
				"output must stay inside the pool's token1 balance")
		})
	}
}
