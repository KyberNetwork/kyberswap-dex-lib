package metricpropamm

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func depthSim(diff int, bids, asks []PairBin) *PoolSimulator {
	huge := bignumber.NewBig("1" + "000000000000000000000000000000000000000")
	return &PoolSimulator{
		Info: pool.PoolInfo{Tokens: []string{"a", "b"}, Reserves: []*big.Int{huge, new(big.Int).Set(huge)}},
		extra: Extra{QuoteAvailable: true, InitBid: big.NewInt(1), InitAsk: big.NewInt(1), MaxAge: 1 << 40,
			Bids: ConvertBins(bids), Asks: ConvertBins(asks)},
		poolTimestamp: time.Now().Unix(),
		decimalsDiff:  diff,
	}
}

func quote(t *testing.T, s *PoolSimulator, in, out, amt string) string {
	t.Helper()
	r, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: in, Amount: bignumber.NewBig(amt)}, TokenOut: out})
	require.NoError(t, err)
	return r.TokenAmountOut.Amount.String()
}

// Metric docs example (WETH/USDC bids, sell 5 WETH): the proportional share gives 13535579525,
// the exact in-bin fill 13535754492. Legacy price averaging cannot produce the latter.
func TestDepthDocExampleBids(t *testing.T) {
	bids := []PairBin{
		{BinIdx: -1, Price: "49937913286041055521250", CummlativeVolume: "7430706558", CumulativeInput: "2744786401520641763"},
		{BinIdx: -2, Price: "49932918995283375647711", CummlativeVolume: "21731301391", CumulativeInput: "8027598362225637937"},
	}
	s := depthSim(12, bids, nil)
	require.Equal(t, "13535754492", quote(t, s, "a", "b", "5000000000000000000"))
	// an order ending exactly on a bin edge takes the whole bin
	require.Equal(t, "7430706558", quote(t, s, "a", "b", "2744786401520641763"))
}

// Live Ethereum TEST/WETH asks (6.25% bins), snapshot-matched with MetricOmmSwapQuoter.quoteLiveExactInSingle.
// The estimate must stay at or just under the on-chain amount; legacy averaging overshot it by ~4e-4.
func TestDepthAsksVsOnChain(t *testing.T) {
	asks := []PairBin{
		{BinIdx: 0, Price: "39591324469", CummlativeVolume: "25491833097124322491019563", CumulativeInput: "53352173083312259"},
		{BinIdx: 1, Price: "41920225908", CummlativeVolume: "53224589131911939843496922", CumulativeInput: "114624230948645291"},
		{BinIdx: 2, Price: "44249127347", CummlativeVolume: "78742612724814343675399011", CumulativeInput: "174224764803291247"},
	}
	s := depthSim(0, nil, asks)
	for _, c := range []struct{ in, sim, chain string }{
		{"17784057694437419", "8642861302178277680024930", "8642861302565720663737683"},
		{"83988202015978775", "39556261818046938629457961", "39556261818628092766197632"},
		{"134491075566860609", "61885226129908243235374919", "61885226130283288765005063"},
	} {
		got := quote(t, s, "b", "a", c.in)
		require.Equal(t, c.sim, got)
		g, ch := bignumber.NewBig(got), bignumber.NewBig(c.chain)
		require.True(t, g.Cmp(ch) <= 0, "never above the pool")
		require.True(t, new(big.Int).Mul(new(big.Int).Sub(ch, g), big.NewInt(1e9)).Cmp(ch) < 0, "within 1e-9 of the pool")
	}
}

// A flat bin (E=0) falls back to the proportional share.
func TestDepthFlatBinProportional(t *testing.T) {
	// asks, no decimals gap: closeIn = O*p/2^64 = 2e6 = binIn
	asks := []PairBin{{BinIdx: 0, Price: "36893488147419103232", CummlativeVolume: "1000000", CumulativeInput: "2000000"}}
	require.Equal(t, "250000", quote(t, depthSim(0, nil, asks), "b", "a", "500000"))
}

func TestDepthInsufficientAndUpdateBalance(t *testing.T) {
	bids := []PairBin{
		{BinIdx: 0, Price: "36893488147419103232", CummlativeVolume: "1000", CumulativeInput: "500"},
		{BinIdx: 1, Price: "36893488147419103232", CummlativeVolume: "3000", CumulativeInput: "1500"},
	}
	s := depthSim(0, bids, nil)
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "a", Amount: big.NewInt(1501)}, TokenOut: "b"})
	require.ErrorIs(t, err, ErrInsufficientLiquidity)

	// swap 500 in / 1000 out consumes bin 0; bin 1 restarts from zero
	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "a", Amount: big.NewInt(500)},
		TokenAmountOut: pool.TokenAmount{Token: "b", Amount: big.NewInt(1000)},
	})
	require.Len(t, s.extra.Bids, 1)
	require.Equal(t, "2000", s.extra.Bids[0].CumulativeVolume.String())
	require.Equal(t, "1000", s.extra.Bids[0].CumulativeInput.String())
}
