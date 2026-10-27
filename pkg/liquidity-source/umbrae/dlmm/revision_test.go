package umbraedlmm

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// SwapRevisionV2Oct2026 (Robinhood Chain): the three changes over the Base V2 build, each against a
// hand-computed case, with the Base revision shown unchanged on the same input.

func TestGetCarriedVolatility(t *testing.T) {
	fp := FeeParameters{FilterPeriod: 30, DecayPeriod: 600, ReductionFactor: 5000}
	// Inside the filter window: the stored accumulator, untouched.
	require.Equal(t, uint64(1000), getCarriedVolatility(1000, 1000, fp, 1010))
	// A new window: decayed 1000*(600-60)/600 = 900, half of it carries.
	require.Equal(t, uint64(450), getCarriedVolatility(1000, 1000, fp, 1060))
	require.Equal(t, uint64(900), getDecayedVolatility(1000, 1000, 30, 600, 1060), "Base keeps the full decayed value")
	// Past the decay period nothing is left to carry.
	require.Equal(t, uint64(0), getCarriedVolatility(1000, 1000, fp, 1700))
	// filterPeriod 0: every swap opens a new window, even in the same second.
	require.Equal(t, uint64(500), getCarriedVolatility(1000, 1000, FeeParameters{DecayPeriod: 600, ReductionFactor: 5000}, 1000))
	// reductionFactor 10000 carries everything: the Base behaviour.
	fp.ReductionFactor = basisPoints
	require.Equal(t, uint64(900), getCarriedVolatility(1000, 1000, fp, 1060))
}

// One bin at the 1:1 price, 18/18 decimals, bin step 25: 100 out-units in the bin.
func dustCase(t *testing.T, amountIn uint64, swapForY, absorb bool) (out, consumed *uint256.Int) {
	t.Helper()
	one := uint256.NewInt(1)
	out, consumed, err := calculateSwapWithinBin(
		uint256.NewInt(100), uint256.NewInt(amountIn), getNormalizedPriceFromId(activeBinID, 25),
		new(uint256.Int).Set(e18), one, one, swapForY, activeBinID, 25, absorb)
	require.NoError(t, err)
	return out, consumed
}

func TestCalculateSwapWithinBin_Dust(t *testing.T) {
	for _, swapForY := range []bool{true, false} {
		// 101 in: the bin empties for 100, and the 1 left buys 1 here but 0 at the adjacent bin
		// (0.9975 selling X, 1/1.0025 selling Y): revision 1 charges it to this bin.
		out, consumed := dustCase(t, 101, swapForY, true)
		require.Equal(t, "100", out.Dec())
		require.Equal(t, "101", consumed.Dec(), "swapForY=%v: the dust is taken here", swapForY)
		_, consumed = dustCase(t, 101, swapForY, false)
		require.Equal(t, "100", consumed.Dec(), "swapForY=%v: Base leaves it to walk on", swapForY)
		// 102 in: the 2 left buy at least 1 at the adjacent bin, so they walk on in both builds.
		_, consumed = dustCase(t, 102, swapForY, true)
		require.Equal(t, "100", consumed.Dec())
	}
}

// Two bins, no fee: 101 X against 100 Y in the active bin and 100 Y one bin lower. The 1 X left
// after the first bin buys nothing at the second, so the Base build walks on, finds no bin that
// fills it and reverts; the Robinhood build charges it to the first bin and fills.
func TestCalcAmountOut_DustFillsInsteadOfReverting(t *testing.T) {
	extra := Extra{
		ActiveID: activeBinID,
		Bins: []Bin{
			{ID: activeBinID - 1, ReserveX: u("0"), ReserveY: u("100")},
			{ID: activeBinID, ReserveX: u("0"), ReserveY: u("100")},
		},
		FeeParameters: FeeParameters{MaxVolatilityAccumulator: 35000},
	}
	in := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: big.NewInt(101)}, TokenOut: tokenY}

	rh := newSim(t, StaticExtra{BinStep: 25, DecimalsX: 18, DecimalsY: 18, SwapRevision: SwapRevisionV2Oct2026}, extra)
	res, err := rh.CalcAmountOut(in)
	require.NoError(t, err)
	require.Equal(t, "100", res.TokenAmountOut.Amount.String())
	require.Equal(t, uint32(activeBinID), res.SwapInfo.(SwapInfo).newActiveID, "stays in the bin it emptied")

	base := newSim(t, StaticExtra{BinStep: 25, DecimalsX: 18, DecimalsY: 18}, extra)
	_, err = base.CalcAmountOut(in)
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
}

// 102 bins below the price, 1e18 Y each, no fee, bin step 25 (101 moves stay inside the 10000-bps
// movement bound). Filling 101 bins is 100 moves; reaching the 102nd is the 101st move, which the
// Robinhood build reverts.
func TestCalcAmountOut_TraversalLimit(t *testing.T) {
	bins := make([]Bin, 0, 102)
	for i := uint32(0); i < 102; i++ {
		bins = append(bins, Bin{ID: activeBinID - i, ReserveX: u("0"), ReserveY: u("1000000000000000000")})
	}
	extra := Extra{ActiveID: activeBinID, Bins: bins, FeeParameters: FeeParameters{MaxVolatilityAccumulator: 35000}}
	rhStatic := StaticExtra{BinStep: 25, DecimalsX: 18, DecimalsY: 18, SwapRevision: SwapRevisionV2Oct2026}

	// X needed to empty the first n bins: the ceil-inverse at each bin's price.
	drain := func(n uint32) *big.Int {
		total := new(uint256.Int)
		for i := uint32(0); i < n; i++ {
			c, err := mulDivCeil(uint256.NewInt(1e18), e18, getNormalizedPriceFromId(activeBinID-i, 25))
			require.NoError(t, err)
			total.Add(total, c)
		}
		return total.ToBig()
	}
	swap := func(static StaticExtra, amount *big.Int) (*pool.CalcAmountOutResult, error) {
		return newSim(t, static, extra).CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: amount}, TokenOut: tokenY,
		})
	}

	// Ends inside bin 101: 100 moves, allowed.
	within := new(big.Int).Sub(drain(101), big.NewInt(1e17))
	res, err := swap(rhStatic, within)
	require.NoError(t, err)
	require.Equal(t, uint32(activeBinID-100), res.SwapInfo.(SwapInfo).newActiveID)

	// Needs bin 102: the 101st move.
	past := new(big.Int).Add(drain(101), big.NewInt(1e17))
	_, err = swap(rhStatic, past)
	require.ErrorIs(t, err, ErrSwapTraversalLimit)
	res, err = swap(StaticExtra{BinStep: 25, DecimalsX: 18, DecimalsY: 18}, past)
	require.NoError(t, err, "Base has no traversal limit")
	require.Equal(t, uint32(activeBinID-101), res.SwapInfo.(SwapInfo).newActiveID)
}

// The variable fee a swap starts with: an accumulator of 100 tracked 60s after its last update
// (filter 30s, decay 600s) decays to 90 on both builds; Robinhood carries half of it.
func TestCalcAmountOut_CarriedVolatilityFee(t *testing.T) {
	extra := Extra{
		ActiveID:              activeBinID,
		Bins:                  []Bin{{ID: activeBinID, ReserveX: u("0"), ReserveY: u("1000000000000000000000")}},
		FeeParameters:         FeeParameters{BaseFactor: 30, FilterPeriod: 30, DecayPeriod: 600, ReductionFactor: 5000, VariableFeeControl: 4000, MaxVolatilityAccumulator: 35000},
		VariableFeeCap:        500,
		VolatilityAccumulator: 100,
		VolatilityReference:   activeBinID,
		LastVolatilityUpdate:  1000,
		Timestamp:             1060,
	}
	in := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: big.NewInt(1e18)}, TokenOut: tokenY}
	feeOf := func(rev uint8) string {
		res, err := newSim(t, StaticExtra{BinStep: 25, DecimalsX: 18, DecimalsY: 18, SwapRevision: rev}, extra).CalcAmountOut(in)
		require.NoError(t, err)
		return res.Fee.Amount.String()
	}
	// Rate in bps: 30 + (vol*25)^2*4000/1e10 -> vol 90: 30+2 = 32 (Base); vol 45: 30+0 = 30 (Robinhood).
	require.Equal(t, calculateDynamicFee(30, 4000, 90, 25, 500).Uint64(), uint64(32))
	require.Equal(t, calculateDynamicFee(30, 4000, 45, 25, 500).Uint64(), uint64(30))
	fee32, _ := getFeeAmountFrom(uint256.NewInt(1e18), uint256.NewInt(32))
	fee30, _ := getFeeAmountFrom(uint256.NewInt(1e18), uint256.NewInt(30))
	require.Equal(t, fee32.Dec(), feeOf(SwapRevisionV2))
	require.Equal(t, fee30.Dec(), feeOf(SwapRevisionV2Oct2026))
}

// Pools listed before SwapRevision existed carry no field and keep the Base behaviour.
func TestStaticExtra_SwapRevisionDefault(t *testing.T) {
	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(`{"binStep":25,"decimalsX":18,"decimalsY":6,"router":"0x1"}`), &se))
	require.Equal(t, SwapRevisionV2, se.SwapRevision)
	b, _ := json.Marshal(StaticExtra{BinStep: 25, SwapRevision: SwapRevisionV2Oct2026})
	require.Contains(t, string(b), `"swapRevision":1`)
}
