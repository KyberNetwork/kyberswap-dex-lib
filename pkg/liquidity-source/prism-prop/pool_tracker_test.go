package prismprop

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// realOrderBookHex is a real getOrderBook(WETH, USDC) response captured from
// Base mainnet at block 0x30221fe (50,470,910), pinned alongside a same-block
// getAmountOut(WETH, USDC, 0.5e18) call that returned 1232.452951 USDC -- see
// TestToLevels_MatchesGetAmountOut.
const realOrderBookHex = "00000000000000000000000000000000000000000000000000000000000000200000000000000000000000004200000000000000000000000000000000000006000000000000000000000000833589fcd6edb6e08f4c7c32d4f71b54bda0291300000000000000000000000000000000000000000000000000000000030221fe00000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000046000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000007a1dc3a700000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000119799812dea11197f10000000000000000000000000000000000000000000000000000000003022200000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000001f87d53c4d19e000000000000000000000000000000000000000000000000000000000014dd043900000000000000000000000000000000000000000000000003a8a4a5e8ebb8f90000000000000000000000000000000000000000000000000000000026bbf7650000000000000000000000000000000000000000000000000000441a32f12b07000000000000000000000000000000000000000000000000000000000002d0fa0000000000000000000000000000000000000000000000000b42cc26d2f1dc00000000000000000000000000000000000000000000000000000000007735ee4000000000000000000000000000000000000000000000000005a09b559e7c5e29000000000000000000000000000000000000000000000000000000003b9233460000000000000000000000000000000000000000000000000e1449f011b3cdd700000000000000000000000000000000000000000000000000000000950a6c8b00000000000000000000000000000000000000000000000013b3645a2812953000000000000000000000000000000000000000000000000000000000d08b78d9000000000000000000000000000000000000000000000000000180eb881d96d000000000000000000000000000000000000000000000000000000000000fea9b0000000000000000000000000000000000000000000000001c26fe62396298000000000000000000000000000000000000000000000000000000000129fe216000000000000000000000000000000000000000000000000010e2cfaed78ca64f00000000000000000000000000000000000000000000000000000000b2b9d0cd0000000000000000000000000000000000000000000000000b442eb361d5f1b100000000000000000000000000000000000000000000000000000000773ed27f0000000000000000000000000000000000000000000000001a85ebe6022aac000000000000000000000000000000000000000000000000000000000118b4fe8100000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000c6a3b9b2580f78ee0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000fffffffffffffffffffffffffffe0000000000000000000000000000000000000000000000000000000003022200000000000000000000000000000000000000000000000000000000000000000d00000000000000000000000000000000000000000000000001f87072eec8a6af0000000000000000000000000000000000000000000000000000000014dd586900000000000000000000000000000000000000000000000003a86aca8e200fb20000000000000000000000000000000000000000000000000000000026bbab230000000000000000000000000000000000000000000000000000660b12bd135700000000000000000000000000000000000000000000000000000000000438880000000000000000000000000000000000000000000000000b4282906c860a4a00000000000000000000000000000000000000000000000000000000773c2308000000000000000000000000000000000000000000000000059f636bf665285a000000000000000000000000000000000000000000000000000000003b8aaec40000000000000000000000000000000000000000000000000e150110f5c7f4dd0000000000000000000000000000000000000000000000000000000095204fcf00000000000000000000000000000000000000000000000013af3fff3cdb015f00000000000000000000000000000000000000000000000000000000d075c53d0000000000000000000000000000000000000000000000000005247d9940403900000000000000000000000000000000000000000000000000000000003675b50000000000000000000000000000000000000000000000001c2646694dadb9f0000000000000000000000000000000000000000000000000000000012a1f0a0400000000000000000000000000000000000000000000000010d9305ebbe3183900000000000000000000000000000000000000000000000000000000b2729cbb0000000000000000000000000000000000000000000000000b4d160a6f7c8cd80000000000000000000000000000000000000000000000000000000077b1eb8d00000000000000000000000000000000000000000000000021b65aaa4b44eddd00000000000000000000000000000000000000000000000000000001651630e700000000000000000000000000000000000000000000000008830ee7e8b78251000000000000000000000000000000000000000000000000000000005a2912e2"

func TestToLevels_MatchesGetAmountOut(t *testing.T) {
	data, err := hex.DecodeString(realOrderBookHex)
	require.NoError(t, err)

	var res getOrderBookResult
	require.NoError(t, routerABI.UnpackIntoInterface(&res, methodGetOrderBook, data))

	require.Len(t, res.Book.Side0.Orders, 12)
	require.Len(t, res.Book.Side1.Orders, 13)

	levels := toLevels(res.Book.Side0, res.Book.Side1, 18, 6) // WETH -> USDC
	require.Len(t, levels, 26)                                // 25 real orders + zero-size minTrade sentinel

	assert.Zerof(t, levels[0].Size(), "levels[0] must be a zero-size sentinel -- order-book takes minTrade from it")

	for i := 2; i < len(levels); i++ {
		assert.GreaterOrEqualf(t, levels[i-1].Price(), levels[i].Price(),
			"levels[1:] must be sorted by descending price so order-book's greedy walk consumes best price first")
	}

	// Walk the ladder for 0.5 WETH the same way order-book.getAmountOut does,
	// and compare against a same-block getAmountOut(WETH, USDC, 0.5e18) call
	// that returned 1232.452951 USDC. The two aren't expected to match
	// exactly -- getOrderBook exposes raw maker prices while getAmountOut
	// applies the router's own taker spread on top (no on-chain fee getter
	// was found among prism-prop's resolved selectors) -- but they must be
	// close, or the parsing/merge/sort above is wrong.
	amountIn := 0.5
	var amountOut float64
	for _, lvl := range levels {
		take := min(lvl.Size(), amountIn)
		amountOut += take * lvl.Price()
		if amountIn -= take; amountIn <= 0 {
			break
		}
	}

	const wantAmountOut = 1232.452951
	assert.InEpsilonf(t, wantAmountOut, amountOut, 0.001, // 10bps tolerance for the router's taker spread
		"parsed ladder should reproduce getAmountOut's quote to within the router's fee/spread")
}

// TestToLevels_SmallTradeNotRejected pins the actual bug: without the
// zero-size sentinel at levels[0], order-book.NewPoolSimulatorWith reads
// minTrade from the best-priced real order's own size, wrongly rejecting
// smaller trades that later (worse-priced) orders could still fill.
func TestToLevels_SmallTradeNotRejected(t *testing.T) {
	data, err := hex.DecodeString(realOrderBookHex)
	require.NoError(t, err)

	var res getOrderBookResult
	require.NoError(t, routerABI.UnpackIntoInterface(&res, methodGetOrderBook, data))

	levels := toLevels(res.Book.Side0, res.Book.Side1, 18, 6)

	smallest := levels[1]
	for _, lvl := range levels[1:] {
		if lvl.Size() < smallest.Size() {
			smallest = lvl
		}
	}
	require.Greater(t, smallest.Size(), 0.0)
	require.Greaterf(t, levels[1].Size(), smallest.Size(),
		"fixture must have a smaller order than the best-priced one for this test to mean anything")

	extraBytes, err := json.Marshal(orderbook.Extra{LevelsFrom: [2][]orderbook.Level{levels, nil}})
	require.NoError(t, err)

	sim, err := orderbook.NewPoolSimulatorWith(entity.Pool{
		Exchange:  string(DexType),
		Type:      orderbook.DexType,
		Timestamp: time.Now().Unix(),
		Tokens: []*entity.PoolToken{
			{Address: "0x4200000000000000000000000000000000000006", Decimals: 18},
			{Address: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", Decimals: 6},
		},
		Reserves: entity.PoolReserves{"0", "0"},
		Extra:    string(extraBytes),
	}, time.Hour)
	require.NoError(t, err)

	amountIn, _ := big.NewFloat(smallest.Size() * 1e18).Int(nil)
	_, err = sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{Token: "0x4200000000000000000000000000000000000006", Amount: amountIn},
		TokenOut:      "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
	})
	assert.NoErrorf(t, err, "a trade sized to the smallest real order must not be rejected as below minTrade")
}

// TestCalibrateFee_Math pins the fee math against ratios observed in a real
// on-chain swap (see pool_tracker.go's calibrateFee doc): the router's
// actual output was consistently ~7.2-7.3bps below what raw maker prices
// implied, in both directions.
func TestCalibrateFee_Math(t *testing.T) {
	rawOut := big.NewInt(482_894_669)
	realOut := big.NewInt(482_543_390) // observed total/quote ratio 0.9992725556471198

	fee := feeFrom(realOut, rawOut)
	assert.InDeltaf(t, 0.0007274, fee, 1e-6, "fee should match the observed ~7.27bps shortfall")
	assert.Equal(t, fee, clampFee(fee), "a realistic fee must not get clamped")

	assert.Zero(t, clampFee(-0.001), "a negative fee (real quote came back higher than raw) must clamp to 0")
	assert.Equal(t, maxCalibratedFee, clampFee(1.0),
		"a nonsense fee from a bad reference quote must clamp to the safety ceiling")
}

func TestBestOrderAmountIn(t *testing.T) {
	levels := []orderbook.Level{
		{0, 0},        // sentinel
		{2.5, 2463.0}, // best real order: 2.5 WETH @ 2463 USDC/WETH
	}

	amountIn, rawOut, ok := bestOrderAmountIn(levels, 18, 6)
	require.True(t, ok)
	assert.Equal(t, big.NewInt(2_500_000_000_000_000_000), amountIn)
	assert.Equal(t, big.NewInt(6_157_500_000), rawOut)

	_, _, ok = bestOrderAmountIn(levels[:1], 18, 6)
	assert.False(t, ok, "no real orders (only the sentinel) must report not-ok, not divide by zero downstream")
}
