package ilyrislb

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	tokX = "0x0bd7d308f8e1639fab988df18a8011f41eacad73" // WETH, 18dp
	tokY = "0x5fc5360d0400a0fd4f2af552add042d716f1d168" // USDG, 6dp
)

// newTestSim is a 5-bin book around active 7796: Y at and below, X at and above, 0.30% fee.
func newTestSim(t *testing.T) *PoolSimulator {
	e18, e6 := big256.TenPow(18).Dec(), "500000000"
	b := func(id int32, x, y string) Bin {
		return Bin{ID: id, ReserveX: *uint256.MustFromDecimal(x), ReserveY: *uint256.MustFromDecimal(y)}
	}
	extra, err := json.Marshal(Extra{
		ActiveID: 7796,
		Bins:     []Bin{b(7798, e18, "0"), b(7794, "0", e6), b(7795, "0", e6), b(7796, e18, e6), b(7797, e18, "0")},
		Fee:      &FeeParams{BaseFactor: 30_000, FilterPeriod: 30, DecayPeriod: 600, ReductionFactor: 5_000, MaxVolatilityAccumulator: 350_000, IDReference: 7796},
	})
	require.NoError(t, err)
	s, err := NewPoolSimulator(entity.Pool{
		Address: "0xpool", Exchange: DexType, Type: DexType,
		Tokens:      []*entity.PoolToken{{Address: tokX}, {Address: tokY}},
		StaticExtra: `{"binStepBps":10,"decimalsX":18,"decimalsY":6}`,
		Extra:       string(extra),
	})
	require.NoError(t, err)
	return s
}

func quote(s *PoolSimulator, in string, amount *big.Int) (*pool.CalcAmountOutResult, error) {
	out := tokX
	if in == tokX {
		out = tokY
	}
	return s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: amount}, TokenOut: out})
}

func swap(t *testing.T, s *PoolSimulator, in string, amount *big.Int) *pool.CalcAmountOutResult {
	t.Helper()
	res, err := quote(s, in, amount)
	require.NoError(t, err)
	s.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: amount},
		TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee, SwapInfo: res.SwapInfo})
	return res
}

func TestNewPoolSimulator(t *testing.T) {
	s := newTestSim(t)
	for i := 1; i < len(s.bins); i++ {
		require.Less(t, s.bins[i-1].ID, s.bins[i].ID, "traversal assumes ascending bins")
	}
	assert.Equal(t, "3000000000000000000", s.Info.Reserves[0].String())
	assert.EqualValues(t, 3_000_000, s.feeRate)

	ep := entity.Pool{Tokens: []*entity.PoolToken{{}, {}}, StaticExtra: `{"binStepBps":10}`,
		Extra: `{"activeId":1,"bins":[{"id":1,"x":"0","y":"0"}],"fee":{}}`}
	_, err := NewPoolSimulator(ep)
	assert.ErrorIs(t, err, ErrEmptyBook, "an empty book would quote zero, which routes as a real offer")
	ep.Extra = `{"activeId":1,"bins":[{"id":1,"x":"1","y":"0"}]}`
	_, err = NewPoolSimulator(ep)
	assert.ErrorIs(t, err, ErrMalformedExtra, "a missing fee state would quote at a zero fee")
}

// uint128 reserves exceed float64 precision; they must survive tracker -> extra -> simulator.
func TestMaxUint128ReserveSurvivesTheWire(t *testing.T) {
	maxU128, _ := new(big.Int).SetString("340282366920938463463374607431768211455", 10)
	c := newFakeChain()
	c.book[1].ReserveX = maxU128
	p, _, err := refresh(t, c, entity.Pool{Address: "0xpool"})
	require.NoError(t, err)
	s, err := NewPoolSimulator(p)
	require.NoError(t, err)
	assert.Equal(t, maxU128.String(), s.bins[1].ReserveX.Dec())
}

func TestCalcAmountOut(t *testing.T) {
	s := newTestSim(t)
	res, err := quote(s, tokY, big.NewInt(10_000_000))
	require.NoError(t, err)
	assert.Positive(t, res.TokenAmountOut.Amount.Sign())
	assert.Equal(t, tokY, res.Fee.Token, "BinPool takes the fee from the input")
	assert.EqualValues(t, 30_000, res.Fee.Amount.Int64())
	assert.EqualValues(t, baseSwapGas, res.Gas)

	// Gas grows per bin crossed: 3000 USDG buys past 7796 into 7797.
	res, err = quote(s, tokY, big.NewInt(3_000_000_000))
	require.NoError(t, err)
	assert.EqualValues(t, baseSwapGas+perExtraBinGas, res.Gas)

	// Quoting must not mutate: the router prices many amounts against one simulator.
	again, err := quote(s, tokY, big.NewInt(3_000_000_000))
	require.NoError(t, err)
	assert.Equal(t, res.TokenAmountOut.Amount, again.TokenAmountOut.Amount)

	// An unfillable size is an error, never a zero.
	_, err = quote(s, tokY, new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil))
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}

// swapExactIn calls the market guard; quoteExactIn does not. The simulator must reject what
// would revert. Any freeze ending after the snapshot blocks.
func TestGuardBlocksQuotes(t *testing.T) {
	s := newTestSim(t)
	s.swapsPaused = true
	_, err := quote(s, tokY, big.NewInt(1_000_000))
	assert.ErrorIs(t, err, ErrSwapsPaused)

	s = newTestSim(t)
	s.blockTimestamp, s.freezeEnd = 1000, 1001
	_, err = quote(s, tokY, big.NewInt(1_000_000))
	assert.ErrorIs(t, err, ErrCorporateActionFreeze)
	s.freezeEnd = 1000 // ended: frozen() is start <= now < end
	_, err = quote(s, tokY, big.NewInt(1_000_000))
	assert.NoError(t, err)
}

// UpdateBalance moves the crossed bins by the quoted fills, net of fee (BinPool credits the fee
// to accumulators, not reserves), so a second leg cannot re-spend them.
func TestUpdateBalanceSpendsTheBook(t *testing.T) {
	s := newTestSim(t)
	in := big.NewInt(5e17)
	res := swap(t, s, tokX, in)
	assert.EqualValues(t, 7794, s.activeID) // ~1210 USDG out: crosses 7796 and 7795

	var sumX, sumY uint256.Int
	for _, b := range s.bins {
		sumX.Add(&sumX, &b.ReserveX)
		sumY.Add(&sumY, &b.ReserveY)
	}
	assert.Equal(t, new(big.Int).Sub(big.NewInt(1_500_000_000), res.TokenAmountOut.Amount), sumY.ToBig())
	wantX := new(big.Int).Mul(big.NewInt(3), big256.TenPow(18).ToBig())
	wantX.Sub(wantX.Add(wantX, in), res.Fee.Amount)
	assert.Equal(t, wantX, sumX.ToBig())
	assert.Equal(t, sumY.ToBig(), s.Info.Reserves[1])
	assert.Equal(t, sumX.ToBig(), s.Info.Reserves[0])

	s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: "foreign"}) // must be a no-op, not a panic
}

// A swap applied to a clone must leave the original untouched; a nil or shallow clone breaks
// split routing.
func TestCloneStateIsolatesUpdateBalance(t *testing.T) {
	s, fresh := newTestSim(t), newTestSim(t)
	s.fee.VariableFeeControl, fresh.fee.VariableFeeControl = 122_448, 122_448
	clone := s.CloneState().(*PoolSimulator)
	swap(t, clone, tokY, big.NewInt(3_000_000_000))

	require.NotEqual(t, s.activeID, clone.activeID)
	require.NotEqual(t, s.fee, clone.fee)
	assert.Equal(t, fresh.activeID, s.activeID)
	assert.Equal(t, fresh.fee, s.fee)
	assert.Equal(t, fresh.feeRate, s.feeRate)
	assert.Equal(t, fresh.Info.Reserves, s.Info.Reserves)
	assert.Equal(t, fresh.bins, s.bins)
}

// BinPool charges amountIn - amountIn*(1e9-rate)/1e9 at the full 1e9-precision rate.
func TestSubBpsFeeRateIsCharged(t *testing.T) {
	s := newTestSim(t)
	s.feeRate = 3_123_456
	res, err := quote(s, tokY, big.NewInt(10_000_000))
	require.NoError(t, err)
	assert.EqualValues(t, 10_000_000-10_000_000*(1_000_000_000-3_123_456)/1_000_000_000, res.Fee.Amount.Int64())
}

// The rate is fixed per swap; _commitVolatility then folds the bins moved into the accumulator,
// so the next swap in a route pays more. Rates: baseFactor*binStep*10 + ceil(c*(acc*step)^2/1e11).
func TestSecondSwapPaysPostSwapVolatilityFee(t *testing.T) {
	for _, c := range []struct {
		name         string
		now, tlu     uint64
		va           uint32
		rate1, rate2 uint64
	}{
		{"fresh window, acc 0 -> 1 bin", 0, 0, 0, 3_000_000, 3_012_245},
		// 100s after the last swap: past filterPeriod, inside decayPeriod, so the reference
		// decays to va/2 = 100000 and re-anchors at the pre-swap bin.
		{"decayed reference", 1000, 900, 200_000, 4_224_480, 4_481_621},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := newTestSim(t)
			s.fee.VariableFeeControl, s.fee.TimeLastUpdate, s.fee.VolatilityAccumulator = 122_448, c.tlu, c.va
			s.blockTimestamp = c.now
			s.feeRate = s.fee.totalFeeRate(s.binStepBps, s.activeID, s.blockTimestamp)
			require.Equal(t, c.rate1, s.feeRate)

			swap(t, s, tokY, big.NewInt(3_000_000_000)) // 7796 -> 7797
			res, err := quote(s, tokY, big.NewInt(10_000_000))
			require.NoError(t, err)
			assert.EqualValues(t, 10_000_000-10_000_000*(1_000_000_000-c.rate2)/1_000_000_000, res.Fee.Amount.Int64())
		})
	}
}

// Four swaps on an anvil fork of Robinhood from the state in testdata (three in one block, the
// fourth 28s later). Expected values are the on-chain Swap events: amountOut, fee, finalId and
// volatilityAccumulatorAfter. The fourth crosses 45 bins, past BinPool's 16-hop fast path.
func TestSequentialSwapsMatchFork(t *testing.T) {
	s := liveSim(t)
	for _, c := range []struct {
		in       string
		amount   int64
		out, fee string
		finalID  int32
		va       uint32
	}{
		{tokX, 800_000_000_000_000, "2085394", "80000000000", 7863, 140_000},
		{tokX, 200_000_000_000_000, "515260", "499996200000", 7859, 180_000},
		{tokY, 1_000_000, "384904743862370", "4068", 7866, 110_000},
		{tokX, 3_000_000_000_000_000, "7556670", "4744863000000", 7822, 350_000},
	} {
		res := swap(t, s, c.in, big.NewInt(c.amount))
		assert.Equal(t, c.out, res.TokenAmountOut.Amount.String())
		assert.Equal(t, c.fee, res.Fee.Amount.String())
		assert.Equal(t, c.finalID, s.activeID)
		assert.Equal(t, c.va, s.fee.VolatilityAccumulator)
	}
	assert.EqualValues(t, 15_099_880, s.feeRate, "getTotalFeeRate() on the fork after the 4th swap")
}

// pool-service ships simulators to router-service via msgpack with unexported fields; every
// pricing field (bins, decimal factor, fee state, guard) must survive or quotes drift.
func TestMsgpackRoundTrip(t *testing.T) {
	s := liveSim(t)
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.IncludeUnexported(true)
	enc.SetForceAsArray(true)
	require.NoError(t, enc.Encode(s))
	dec := msgpack.NewDecoder(&buf)
	dec.IncludeUnexported(true)
	var decoded PoolSimulator
	require.NoError(t, dec.Decode(&decoded))

	want := swap(t, s, tokX, big.NewInt(3e15))
	got := swap(t, &decoded, tokX, big.NewInt(3e15))
	assert.Equal(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount)
	assert.Equal(t, s.fee, decoded.fee)
	assert.Equal(t, s.feeRate, decoded.feeRate)
}
