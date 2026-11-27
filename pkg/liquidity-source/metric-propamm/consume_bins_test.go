package metricpropamm

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// testBins are two bins at prices 110 and 120 holding 100 and 100 more of the maker token, starting at price 100.
func testBins(withInput bool) []Bin {
	bins := []Bin{
		{BinIdx: 1, Price: big.NewInt(110), CumulativeVolume: big.NewInt(100)},
		{BinIdx: 2, Price: big.NewInt(120), CumulativeVolume: big.NewInt(200)},
	}
	if withInput {
		bins[0].CumulativeInput = big.NewInt(1000)
		bins[1].CumulativeInput = big.NewInt(2100)
	}
	return bins
}

func binStrings(bins []Bin) [][3]string {
	out := make([][3]string, len(bins))
	for i, b := range bins {
		in := "<nil>"
		if b.CumulativeInput != nil {
			in = b.CumulativeInput.String()
		}
		out[i] = [3]string{b.Price.String(), b.CumulativeVolume.String(), in}
	}
	return out
}

func TestConsumeBins(t *testing.T) {
	for _, tc := range []struct {
		name              string
		amountIn, out     int64
		wantStart         int64
		wantBins          [][3]string
		withInput         bool
		wantInputAfterBin [][3]string
	}{
		// Inside the first bin: the start price moves a quarter of the way from 100 to 110.
		{name: "inside the first bin", amountIn: 250, out: 25, wantStart: 102,
			wantBins: [][3]string{{"110", "75", "<nil>"}, {"120", "175", "<nil>"}}},
		// Exactly the first bin: it is dropped and the start price is its price.
		{name: "at a bin boundary", amountIn: 1000, out: 100, wantStart: 110,
			wantBins: [][3]string{{"120", "100", "<nil>"}}},
		// Inside the second bin: halfway from 110 to 120.
		{name: "inside the second bin", amountIn: 1550, out: 150, wantStart: 115,
			wantBins: [][3]string{{"120", "50", "<nil>"}}},
		// Every bin: nothing left, the start price is the last bin's.
		{name: "through every bin", amountIn: 2100, out: 200, wantStart: 120, wantBins: [][3]string{}},
		// Nothing out: same bins, same start.
		{name: "zero out", amountIn: 0, out: 0, wantStart: 100,
			wantBins: [][3]string{{"110", "100", "<nil>"}, {"120", "200", "<nil>"}}},
		// Cumulative input moves with the input.
		{name: "cumulative input", amountIn: 250, out: 25, wantStart: 102, withInput: true,
			wantBins: [][3]string{{"110", "75", "750"}, {"120", "175", "1850"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bins, start := testBins(tc.withInput), big.NewInt(100)
			before := binStrings(bins)

			got, gotStart := consumeBins(bins, start, big.NewInt(tc.amountIn), big.NewInt(tc.out))

			require.Equal(t, tc.wantBins, binStrings(got))
			require.Equal(t, big.NewInt(tc.wantStart).String(), gotStart.String())
			require.Equal(t, before, binStrings(bins), "consumeBins changed its bins argument")
			require.Equal(t, "100", start.String(), "consumeBins changed its start argument")
		})
	}
}

func TestConsumeBinsEmpty(t *testing.T) {
	got, start := consumeBins(nil, big.NewInt(100), big.NewInt(1), big.NewInt(1))
	require.Empty(t, got)
	require.Equal(t, "100", start.String())
}

// Pools stored before cumulative input was tracked quote with getRateLegacy, which starts its walk at InitBid or
// InitAsk. After a swap that ends inside a bin, the swapped side's start price is where the fill ended, and the
// other side's is unchanged. A clone's swap leaves the original's start prices and bins.
func TestUpdateBalanceMovesTheLegacyStartPrice(t *testing.T) {
	s := newTestSimulator(t)
	for i := range s.extra.Bids {
		s.extra.Bids[i].CumulativeInput = nil
	}
	for i := range s.extra.Asks {
		s.extra.Asks[i].CumulativeInput = nil
	}
	initBid, initAsk := new(big.Int).Set(s.extra.InitBid), new(big.Int).Set(s.extra.InitAsk)
	bids := binStrings(s.extra.Bids)

	c := s.CloneState().(*PoolSimulator)
	first := c.extra.Bids[0]
	out := new(big.Int).Div(first.CumulativeVolume, big.NewInt(2))
	c.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testWETH, Amount: big.NewInt(1)},
		TokenAmountOut: pool.TokenAmount{Token: testUSDC, Amount: out},
	})

	want := new(big.Int).Sub(first.Price, initBid)
	want.Mul(want, out).Div(want, first.CumulativeVolume).Add(want, initBid) // floored, as getRateLegacy
	require.Equal(t, want.String(), c.extra.InitBid.String(), "the bid side starts where the fill ended")
	require.NotEqual(t, initBid.String(), c.extra.InitBid.String())
	require.Equal(t, initAsk.String(), c.extra.InitAsk.String(), "the other side is unchanged")

	require.Equal(t, initBid.String(), s.extra.InitBid.String(), "the clone's swap moved the original's start")
	require.Equal(t, bids, binStrings(s.extra.Bids), "the clone's swap changed the original's bins")
}

// The same on the ask side: a swap of USDC for WETH moves InitAsk and leaves InitBid.
func TestUpdateBalanceMovesTheLegacyAskStartPrice(t *testing.T) {
	s := newTestSimulator(t)
	for i := range s.extra.Asks {
		s.extra.Asks[i].CumulativeInput = nil
	}
	initBid, initAsk := new(big.Int).Set(s.extra.InitBid), new(big.Int).Set(s.extra.InitAsk)

	first := s.extra.Asks[0]
	out := new(big.Int).Div(first.CumulativeVolume, big.NewInt(2))
	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testUSDC, Amount: big.NewInt(1)},
		TokenAmountOut: pool.TokenAmount{Token: testWETH, Amount: out},
	})

	want := new(big.Int).Sub(first.Price, initAsk)
	want.Mul(want, out).Quo(want, first.CumulativeVolume).Add(want, initAsk)
	require.Equal(t, want.String(), s.extra.InitAsk.String(), "the ask side starts where the fill ended")
	require.NotEqual(t, initAsk.String(), s.extra.InitAsk.String())
	require.Equal(t, initBid.String(), s.extra.InitBid.String(), "the other side is unchanged")
}

// On the bid side prices fall, so the interpolation divides a negative difference. It floors, as getRateLegacy
// does: 25 of 100 from 100 toward 90 is 97.5, so 97 (truncation would give 98, 1 wei above the walk).
func TestConsumeBinsFloorsOnTheBidSide(t *testing.T) {
	bins := []Bin{{BinIdx: 1, Price: big.NewInt(90), CumulativeVolume: big.NewInt(100)}}
	_, start := consumeBins(bins, big.NewInt(100), big.NewInt(250), big.NewInt(25))
	require.Equal(t, "97", start.String())
}
