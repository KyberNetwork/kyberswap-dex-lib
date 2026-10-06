package kuruobv2

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func levels(prices []uint32, sizes ...int64) []Level {
	return toLevels(prices, lo.Map(sizes, func(x int64, _ int) *big.Int { return big.NewInt(x) }))
}

type testMarket struct {
	name        string
	staticExtra StaticExtra
	extra       Extra
	buys, sells [][2]int64 // {amountIn, estimateSwap amountOut}
}

// Books and expected outputs are OrderBook.getL2Book(10) / estimateSwap(isBuy, amountIn) on Monad
// testnet (10143) at block 68723542. Buy = quote (USDC) in, sell = base in.
var testMarkets = []testMarket{{
	name: "WETH/USDC 0x9d187971b64505ac81f12c5fd2ac9c5247ec62f3",
	staticExtra: StaticExtra{PricePrecision: 100, SizePrecision: uint256.NewInt(1e10),
		BaseSizeMultiplier: uint256.NewInt(1e8), QuoteDecimals: 6},
	extra: Extra{TakerFeePps: 7000,
		Bids: levels([]uint32{267820, 267766, 267713, 267659, 267606, 267552, 267498, 0, 0, 0},
			56007767, 56019062, 56030152, 56041456, 56052555, 56063868, 56075186, 0, 0, 0),
		Asks: levels([]uint32{268089, 268143, 268196, 268250, 268303, 268357, 268411, 0, 0, 0},
			55951569, 55940301, 55929246, 55917987, 55906942, 55895692, 55884446, 0, 0, 0)},
	buys:  [][2]int64{{1e6, 372749300000000}, {50e6, 18633005700000000}},
	sells: [][2]int64{{1e15, 2668131}, {2e16, 53502522}},
}, {
	name: "USDT/USDC 0x4a0888c502e64aeae11115508ec0955c70293dba",
	staticExtra: StaticExtra{PricePrecision: 1e6, SizePrecision: uint256.NewInt(1e6),
		BaseSizeMultiplier: uint256.NewInt(1), QuoteDecimals: 6},
	extra: Extra{TakerFeePps: 7000,
		Bids: levels([]uint32{998955, 998755, 998555, 998355, 998155, 0, 0, 0, 0, 0},
			8225374, 15018699, 15021707, 15024716, 15027727, 0, 0, 0, 0, 0),
		Asks: levels([]uint32{1000555, 1000755, 1000955, 1001155, 1001355, 1001555, 0, 0, 0, 0},
			7986357, 14988684, 14985689, 14982695, 14979703, 14976712, 0, 0, 0, 0)},
	buys:  [][2]int64{{1e6, 998745}, {30e6, 29956597}},
	sells: [][2]int64{{1e6, 998255}, {30e6, 29941967}},
}}

func newTestSim(t *testing.T, m testMarket) *PoolSimulator {
	sim, err := NewPoolSimulator(entity.Pool{
		Address: "market", Exchange: DexType, Type: DexType, Reserves: entity.PoolReserves{"1", "1"},
		Tokens: []*entity.PoolToken{{Address: "base"}, {Address: "quote"}},
		Extra:  string(lo.Must(json.Marshal(m.extra))), StaticExtra: string(lo.Must(json.Marshal(m.staticExtra))),
	})
	require.NoError(t, err)
	return sim
}

func calc(sim *PoolSimulator, tokenIn, tokenOut string, amountIn int64) (*pool.CalcAmountOutResult, error) {
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: big.NewInt(amountIn)}, TokenOut: tokenOut})
}

// Quotes must equal the on-chain estimateSwap to the wei, or minAmountOut at zero slippage reverts.
func TestCalcAmountOut_MatchesTestnetEstimateSwap(t *testing.T) {
	for _, m := range testMarkets {
		t.Run(m.name, func(t *testing.T) {
			sim := newTestSim(t, m)
			for dir, cases := range [][][2]int64{m.sells, m.buys} {
				tokenIn, tokenOut := lo.Ternary(dir == 0, "base", "quote"), lo.Ternary(dir == 0, "quote", "base")
				for _, c := range cases {
					res, err := calc(sim, tokenIn, tokenOut, c[0])
					require.NoError(t, err)
					require.Equal(t, c[1], res.TokenAmountOut.Amount.Int64(), "%s in=%d", tokenIn, c[0])
				}
			}
		})
	}
}

// Swapping past the whole book fills what exists and hands back the unspent input
// (estimateSwap(isBuy, 1e9) at the same block: sell (68318223, 68169026), buy (83048886, 82899840)).
func TestCalcAmountOut_BookExhausted(t *testing.T) {
	sim := newTestSim(t, testMarkets[1])
	res, err := calc(sim, "base", "quote", 1e9)
	require.NoError(t, err)
	require.EqualValues(t, 68169026, res.TokenAmountOut.Amount.Int64())
	require.EqualValues(t, 1e9-68318223, res.RemainingTokenAmountIn.Amount.Int64())
	require.Equal(t, gasBase+5*gasLevel, int(res.Gas))

	res, err = calc(sim, "quote", "base", 1e9)
	require.NoError(t, err)
	require.EqualValues(t, 82899840, res.TokenAmountOut.Amount.Int64())
	require.EqualValues(t, 1e9-83048886, res.RemainingTokenAmountIn.Amount.Int64())
}

// UpdateBalance consumes the walked levels without touching a clone taken earlier.
func TestUpdateBalance(t *testing.T) {
	sim := newTestSim(t, testMarkets[1])
	clone := sim.CloneState().(*PoolSimulator)
	res, err := calc(sim, "base", "quote", 10e6) // fills level 0 (8225374) and part of level 1
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "base", Amount: big.NewInt(10e6)},
		TokenAmountOut: *res.TokenAmountOut, SwapInfo: res.SwapInfo})

	require.Len(t, sim.Bids, 4)
	require.Equal(t, uint64(998755), sim.Bids[0].Price)
	require.EqualValues(t, 15018699-(10e6-8225374), sim.Bids[0].Size.Uint64())
	require.Len(t, clone.Bids, 5)
	require.EqualValues(t, 15018699, clone.Bids[1].Size.Uint64())
	require.Len(t, sim.Asks, 6)
}

func TestCalcAmountOut_Rejects(t *testing.T) {
	sim := newTestSim(t, testMarkets[0])
	_, err := calc(sim, "base", "quote", 1e7) // below one book quantity (1e8 atoms)
	require.ErrorIs(t, err, ErrZeroAmountOut)
	_, err = calc(sim, "base", "base", 1e18)
	require.ErrorIs(t, err, ErrInvalidToken)

	closed := testMarkets[0]
	closed.extra = Extra{}
	_, err = calc(newTestSim(t, closed), "quote", "base", 1e6)
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
}
