package metricpropamm

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var _ pool.IAmountInRange = (*PoolSimulator)(nil)

const (
	rangeToken0 = "0x4200000000000000000000000000000000000006" // 18 decimals
	rangeToken1 = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913" // 6 decimals
)

// price returns a Q64 price of `x` token1-units per token0-unit, the encoding Bin.Price uses.
func price(x float64) *big.Int {
	f := new(big.Float).Mul(big.NewFloat(x), new(big.Float).SetInt(Q64BI))
	r, _ := f.Int(nil)
	return r
}

// newRangePool builds a simulator over an explicit bin book. Reserves are set huge so the output
// reserve never binds and the book's depth is the only limit.
func newRangePool(t *testing.T, asks, bids []Bin, opts ...func(*PoolSimulator)) *PoolSimulator {
	t.Helper()
	s := &PoolSimulator{
		Info: pool.PoolInfo{
			Address:  "0xpool",
			Exchange: DexType,
			Type:     DexType,
			Tokens:   []string{rangeToken0, rangeToken1},
			Reserves: []*big.Int{bignumber.NewBig("1000000000000000000000000"), bignumber.NewBig("1000000000000000")},
		},
		poolTimestamp: time.Now().Unix(),
		extra: Extra{
			InitBid: price(2000), InitAsk: price(2000),
			QuoteAvailable: true, MaxAge: 60,
			Asks: asks, Bids: bids,
		},
		decimalsDiff: 12,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// A two-bin book on the token0->token1 side. getRate for that direction walks Bids (zeroToOne),
// whose CumulativeVolume is token1 (amountOut) and converts back to token0 at each bin's price.
func rangeBids() []Bin {
	return []Bin{
		{BinIdx: 0, Price: price(2000), CumulativeVolume: bignumber.NewBig("2000000000")},  // 2,000 USDC
		{BinIdx: 1, Price: price(1990), CumulativeVolume: bignumber.NewBig("20000000000")}, // 20,000 USDC
	}
}

func TestPoolSimulator_AmountInRange(t *testing.T) {
	t.Parallel()
	s := newRangePool(t, nil, rangeBids())

	minIn, maxIn, err := s.AmountInRange(rangeToken0, rangeToken1, nil)
	require.NoError(t, err)
	assert.Nil(t, minIn, "any positive amount inside the first bin quotes, so there is no minimum")
	require.NotNil(t, maxIn)
	assert.Positive(t, maxIn.Sign())
	t.Logf("book depth 20,000 USDC -> maxIn %s wei of token0", maxIn)
}

// THE PIN. The bound is only worth anything if it is where the refusal actually is, so this walks
// the boundary with CalcAmountOut rather than trusting the arithmetic: at maxIn it must quote, and
// just above it must refuse with ErrInsufficientLiquidity.
func TestPoolSimulator_AmountInRangeMatchesWhereCalcAmountOutRefuses(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		asks, bids []Bin
		in, out    string
	}{
		{"token0 -> token1, two bins", nil, rangeBids(), rangeToken0, rangeToken1},
		{"token0 -> token1, one bin", nil, rangeBids()[:1], rangeToken0, rangeToken1},
		{
			"token1 -> token0 walks Asks",
			[]Bin{
				{BinIdx: 0, Price: price(2000), CumulativeVolume: bignumber.NewBig("1000000000000000000")},
				{BinIdx: 1, Price: price(2010), CumulativeVolume: bignumber.NewBig("5000000000000000000")},
			},
			nil, rangeToken1, rangeToken0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newRangePool(t, tc.asks, tc.bids)

			_, maxIn, err := s.AmountInRange(tc.in, tc.out, nil)
			require.NoError(t, err)
			require.Positive(t, maxIn.Sign())

			_, err = s.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: maxIn},
				TokenOut:      tc.out,
			})
			assert.NoError(t, err, "maxIn itself must quote")

			// One part in 10,000 above: the per-bin conversion truncates, so the exact boundary can
			// sit a few wei either side of the sum. A refusal has to appear just above it, not at
			// some much larger size.
			above := new(big.Int).Add(maxIn, new(big.Int).Div(maxIn, big.NewInt(10_000)))
			_, err = s.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: above},
				TokenOut:      tc.out,
			})
			assert.ErrorIs(t, err, ErrInsufficientLiquidity, "above maxIn must be refused")
		})
	}
}

// The book's depth is in amountOut, so the conversion is the whole substance of this method. A
// maxIn computed without it would be wrong by roughly the price -- here a factor of ~2000 -- which
// this pins by requiring the bound to agree with the real boundary to within a part in 10,000.
func TestPoolSimulator_AmountInRangeConvertsOutputVolumeToInputUnits(t *testing.T) {
	t.Parallel()
	s := newRangePool(t, nil, rangeBids())

	_, maxIn, err := s.AmountInRange(rangeToken0, rangeToken1, nil)
	require.NoError(t, err)

	// Find the true boundary by bisection on CalcAmountOut, independent of the implementation.
	lo, hi := big.NewInt(1), new(big.Int).Mul(maxIn, big.NewInt(4))
	for range 80 {
		mid := new(big.Int).Add(lo, new(big.Int).Rsh(new(big.Int).Sub(hi, lo), 1))
		if mid.Cmp(lo) <= 0 || mid.Cmp(hi) >= 0 {
			break
		}
		_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: rangeToken0, Amount: mid},
			TokenOut:      rangeToken1,
		})
		if err == nil {
			lo = mid
		} else {
			hi = mid
		}
	}
	t.Logf("stated maxIn=%s  bisected boundary=%s", maxIn, lo)

	diff := new(big.Int).Sub(lo, maxIn)
	diff.Abs(diff)
	assert.Positive(t, new(big.Int).Div(lo, big.NewInt(10_000)).Cmp(diff),
		"stated %s should match the real boundary %s within 1bp", maxIn, lo)
}

// Both gates mean the pool quotes nothing at any size. A zero maximum states that exactly, which
// is cheaper for the caller than a probe that cannot succeed.
func TestPoolSimulator_AmountInRangeZeroWhenThePoolQuotesNothing(t *testing.T) {
	t.Parallel()

	for name, opt := range map[string]func(*PoolSimulator){
		"quotes unavailable": func(s *PoolSimulator) { s.extra.QuoteAvailable = false },
		"state is stale":     func(s *PoolSimulator) { s.poolTimestamp = time.Now().Unix() - 3600 },
		"empty book":         func(s *PoolSimulator) { s.extra.Bids = nil },
	} {
		t.Run(name, func(t *testing.T) {
			s := newRangePool(t, nil, rangeBids(), opt)

			minIn, maxIn, err := s.AmountInRange(rangeToken0, rangeToken1, nil)
			require.NoError(t, err)
			assert.Nil(t, minIn)
			require.NotNil(t, maxIn)
			assert.Equal(t, 0, maxIn.Sign())

			// And the simulator really does refuse, which is what makes zero honest.
			_, err = s.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: rangeToken0, Amount: big.NewInt(1e9)},
				TokenOut:      rangeToken1,
			})
			assert.Error(t, err)
		})
	}
}

// noStaleCheck is how the factory disables the freshness gate during indexing. The range has to
// honour it, or it would report zero for a pool the simulator is perfectly willing to quote --
// understating, the one direction that fails silently.
func TestPoolSimulator_AmountInRangeHonoursNoStaleCheck(t *testing.T) {
	t.Parallel()
	s := newRangePool(t, nil, rangeBids(), func(s *PoolSimulator) {
		s.poolTimestamp = time.Now().Unix() - 3600
		s.noStaleCheck = true
	})

	_, maxIn, err := s.AmountInRange(rangeToken0, rangeToken1, nil)
	require.NoError(t, err)
	assert.Positive(t, maxIn.Sign(), "stale but exempt: the book is still quotable")

	_, err = s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: rangeToken0, Amount: maxIn},
		TokenOut:      rangeToken1,
	})
	assert.NoError(t, err)
}

// Malformed book data must not become a confident bound, and must not panic: calculateAmountFromPrice
// divides by the price.
func TestPoolSimulator_AmountInRangeUnsupportedOnMalformedBins(t *testing.T) {
	t.Parallel()

	for name, bids := range map[string][]Bin{
		"zero price": {{Price: new(big.Int), CumulativeVolume: bignumber.NewBig("2000000000")}},
		"nil price":  {{Price: nil, CumulativeVolume: bignumber.NewBig("2000000000")}},
		"nil volume": {{Price: price(2000), CumulativeVolume: nil}},
		"cumulative volume goes backwards": {
			{Price: price(2000), CumulativeVolume: bignumber.NewBig("20000000000")},
			{Price: price(1990), CumulativeVolume: bignumber.NewBig("2000000000")},
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := newRangePool(t, nil, bids)
			assert.NotPanics(t, func() {
				_, _, err := s.AmountInRange(rangeToken0, rangeToken1, nil)
				assert.ErrorIs(t, err, pool.ErrUnsupported)
			})
		})
	}
}

func TestPoolSimulator_AmountInRangeRejectsBadDirections(t *testing.T) {
	t.Parallel()
	s := newRangePool(t, nil, rangeBids())

	for _, tc := range [][2]string{
		{"0xnope", rangeToken1},
		{rangeToken0, "0xnope"},
		{rangeToken0, rangeToken0},
	} {
		_, _, err := s.AmountInRange(tc[0], tc[1], nil)
		assert.ErrorIs(t, err, ErrInvalidToken, "%v", tc)
	}
}
