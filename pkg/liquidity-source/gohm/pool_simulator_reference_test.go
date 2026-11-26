package gohm

import (
	"errors"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// refState is the Olympus staking state the reference model reads.
type refState struct {
	index, ohmReserve, sohmReserve *big.Int
	warmup                         bool
}

type refQuote struct {
	out    *big.Int
	gas    int64
	action Action
	err    error
}

var wad = big.NewInt(1e18)

// refCalcAmountOut is the Olympus V2 contracts' math, written in math/big apart from the simulator:
//   - gOHM.balanceTo(a) = a * 1e18 / index and gOHM.balanceFrom(a) = a * index / 1e18 (gOHM.sol);
//   - OlympusStaking.stake/unstake with _rebasing = true move OHM and sOHM 1:1; with _rebasing = false they go through
//     balanceTo / balanceFrom; wrap is balanceTo and unwrap is balanceFrom;
//   - a non-zero warmupPeriod sends gOHM stakes to the warmup queue (claim later), so no ratio leg is atomic;
//   - the contract pays OHM (unstake) or sOHM (stake, unwrap) from its balance, so the output is capped by it.
//
// The gas is the per-action estimate in constant.go, written out here.
func refCalcAmountOut(st refState, in, out string, amount *big.Int) refQuote {
	if amount.Cmp(new(big.Int).Lsh(big.NewInt(1), 256)) >= 0 {
		return refQuote{err: ErrInvalidTokenIn}
	}
	if amount.Sign() == 0 {
		return refQuote{err: ErrZeroAmount}
	}
	known := func(tok string) bool {
		return tok == testOHMAddress || tok == testSOHMAddress || tok == testGOHMAddress
	}
	if !known(in) {
		return refQuote{err: ErrInvalidTokenIn}
	}
	if in == out || !known(out) {
		return refQuote{err: ErrInvalidTokenOut}
	}
	capped := func(v, reserve *big.Int, gas int64, a Action) refQuote {
		if v.Cmp(reserve) > 0 {
			return refQuote{err: ErrInsufficientLiquidity}
		}
		return refQuote{out: v, gas: gas, action: a}
	}
	balanceTo := func() *big.Int { return new(big.Int).Div(new(big.Int).Mul(amount, wad), st.index) }
	balanceFrom := func() *big.Int { return new(big.Int).Div(new(big.Int).Mul(amount, st.index), wad) }
	switch {
	case in == testOHMAddress && out == testSOHMAddress: // stake, rebasing: 1:1, sOHM from the contract
		return capped(amount, st.sohmReserve, 42258, ActionStakeToSOHM)
	case in == testSOHMAddress && out == testOHMAddress: // unstake, rebasing: 1:1, OHM from the contract
		return capped(amount, st.ohmReserve, 40162, ActionUnstakeSOHM)
	case st.warmup:
		return refQuote{err: ErrWarmupActive}
	case st.index.Sign() == 0:
		return refQuote{err: ErrIndexZero}
	case in == testOHMAddress: // stake to gOHM: minted, no cap
		return refQuote{out: balanceTo(), gas: 42258, action: ActionStakeToGOHM}
	case in == testSOHMAddress: // wrap: minted, no cap
		return refQuote{out: balanceTo(), gas: 37900, action: ActionWrap}
	case out == testOHMAddress: // unstake from gOHM
		return capped(balanceFrom(), st.ohmReserve, 40162, ActionUnstakeGOHM)
	default: // unwrap
		return capped(balanceFrom(), st.sohmReserve, 37739, ActionUnwrap)
	}
}

// gridAmounts covers zero, dust, typical sizes, cap-hitting sizes and a value above 2^256-1.
func gridAmounts() []*big.Int {
	out := []*big.Int{big.NewInt(0), big.NewInt(1), big.NewInt(7)}
	for e := 3; e <= 33; e += 3 {
		v := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(e)), nil)
		out = append(out, v, new(big.Int).Add(v, big.NewInt(int64(e*131))))
	}
	return append(out, new(big.Int).Lsh(big.NewInt(1), 256))
}

type refFixture struct {
	name string
	sim  func(t *testing.T) *PoolSimulator
	st   refState
}

func refFixtures() []refFixture {
	live := refState{index: indexAt25147546.ToBig(), ohmReserve: testOHMReserve.ToBig(), sohmReserve: testSOHMReserve.ToBig()}
	capped := live
	capped.ohmReserve, capped.sohmReserve = big.NewInt(5_000_000), big.NewInt(3_000_000)
	index0 := live
	index0.index = big.NewInt(0)
	warmup := live
	warmup.warmup = true
	return []refFixture{
		{"live", func(t *testing.T) *PoolSimulator { return newSimulator(t, indexAt25147546, 0) }, live},
		{"capped", func(t *testing.T) *PoolSimulator {
			return newSimulatorWithReserves(t, indexAt25147546, 0, uint256.NewInt(5_000_000), uint256.NewInt(3_000_000))
		}, capped},
		{"index0", func(t *testing.T) *PoolSimulator { return newSimulator(t, uint256.NewInt(0), 0) }, index0},
		{"warmup", func(t *testing.T) *PoolSimulator { return newSimulator(t, indexAt25147546, 3) }, warmup},
	}
}

func checkAgainstRef(t *testing.T, sim *PoolSimulator, st refState, in, out string, amt *big.Int) {
	t.Helper()
	want := refCalcAmountOut(st, in, out, amt)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: in, Amount: amt},
		TokenOut:      out,
	})
	if want.err != nil {
		assert.Truef(t, errors.Is(err, want.err), "%s->%s %s: err %v, want %v", in[:6], out[:6], amt, err, want.err)
		return
	}
	if !assert.NoErrorf(t, err, "%s->%s %s", in[:6], out[:6], amt) {
		return
	}
	assert.Equalf(t, want.out.String(), res.TokenAmountOut.Amount.String(), "%s->%s %s", in[:6], out[:6], amt)
	assert.Equalf(t, want.gas, res.Gas, "gas %s->%s", in[:6], out[:6])
	assert.Equalf(t, SwapInfo{Action: want.action}, res.SwapInfo, "action %s->%s", in[:6], out[:6])
}

// TestCalcAmountOut_MatchesReferenceModel quotes every token pair and grid amount on the live, capped-reserve,
// zero-index and warmup states against refCalcAmountOut.
func TestCalcAmountOut_MatchesReferenceModel(t *testing.T) {
	t.Parallel()
	tokens := []string{testOHMAddress, testSOHMAddress, testGOHMAddress, "0x0000000000000000000000000000000000000001"}
	for _, fx := range refFixtures() {
		sim := fx.sim(t)
		for _, in := range tokens {
			for _, out := range tokens {
				for _, amt := range gridAmounts() {
					checkAgainstRef(t, sim, fx.st, in, out, amt)
				}
			}
		}
	}
}

// Spot values of the reference model at block 25147546's index, by hand from gOHM.balanceTo / balanceFrom, so a
// wrong model would fail too: 1e12 * 1e18 / 269238508004 = 3714178953870682139 and 1e20 * 269238508004 / 1e18 =
// 26923850800400.
func TestReferenceModel_SpotValues(t *testing.T) {
	t.Parallel()
	st := refFixtures()[0].st
	for _, tc := range []struct {
		in, out, amount, want string
	}{
		{testOHMAddress, testGOHMAddress, "1000000000000", "3714178953870682139"},
		{testGOHMAddress, testSOHMAddress, "100000000000000000000", "26923850800400"},
		{testOHMAddress, testSOHMAddress, "1000", "1000"},
	} {
		amount, _ := new(big.Int).SetString(tc.amount, 10)
		q := refCalcAmountOut(st, tc.in, tc.out, amount)
		require.NoError(t, q.err)
		assert.Equal(t, tc.want, q.out.String())
	}
}

var invariantPairs = [][2]string{
	{testOHMAddress, testSOHMAddress}, {testSOHMAddress, testOHMAddress},
	{testOHMAddress, testGOHMAddress}, {testSOHMAddress, testGOHMAddress},
	{testGOHMAddress, testOHMAddress}, {testGOHMAddress, testSOHMAddress},
}

func quoteOut(t *testing.T, s *PoolSimulator, in, out string, amt *big.Int) (*pool.CalcAmountOutResult, error) {
	t.Helper()
	return s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: amt}, TokenOut: out})
}

// TestInvariant_QuoteIsReadonly: quoting every pair leaves the simulator as it was.
func TestInvariant_QuoteIsReadonly(t *testing.T) {
	t.Parallel()
	s := newSimulator(t, indexAt25147546, 0)
	before := s.CloneState()
	for _, p := range invariantPairs {
		for _, amt := range gridAmounts() {
			_, _ = quoteOut(t, s, p[0], p[1], amt)
		}
	}
	assert.Equal(t, before, s)
}

// TestInvariant_CloneThenSwapLeavesOriginal: UpdateBalance on a clone moves the clone's
// reserves and not the original's.
func TestInvariant_CloneThenSwapLeavesOriginal(t *testing.T) {
	t.Parallel()
	for _, p := range invariantPairs {
		s := newSimulator(t, indexAt25147546, 0)
		before := s.CloneState()
		clone := s.CloneState()
		amt := big.NewInt(1_000_000_000)
		res, err := quoteOut(t, clone.(*PoolSimulator), p[0], p[1], amt)
		require.NoError(t, err)
		clone.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: p[0], Amount: amt},
			TokenAmountOut: *res.TokenAmountOut,
			SwapInfo:       res.SwapInfo,
		})
		assert.Equal(t, before, s, "%s->%s", p[0][:6], p[1][:6])
		assert.NotEqual(t, before, clone, "UpdateBalance did not move the clone %s->%s", p[0][:6], p[1][:6])
	}
}

// TestInvariant_SplitMonotonicBounded: for every pair, out(a)+out(b) <= out(a+b) <= out(a)+out(b)+1 (floor division,
// 1:1 legs exactly), out is non-decreasing in the input, and a capped output never exceeds its reserve.
func TestInvariant_SplitMonotonicBounded(t *testing.T) {
	t.Parallel()
	s := newSimulator(t, indexAt25147546, 0)
	amounts := []int64{1, 3, 999, 1_000_000_007, 123_456_789_012}
	for _, p := range invariantPairs {
		var prev *big.Int
		for _, a := range amounts {
			for _, b := range amounts {
				ra, errA := quoteOut(t, s, p[0], p[1], big.NewInt(a))
				rb, errB := quoteOut(t, s, p[0], p[1], big.NewInt(b))
				rw, errW := quoteOut(t, s, p[0], p[1], big.NewInt(a+b))
				require.NoError(t, errors.Join(errA, errB, errW))
				sum := new(big.Int).Add(ra.TokenAmountOut.Amount, rb.TokenAmountOut.Amount)
				whole := rw.TokenAmountOut.Amount
				assert.LessOrEqual(t, sum.Cmp(whole), 0, "split above whole %s->%s %d+%d", p[0][:6], p[1][:6], a, b)
				assert.LessOrEqual(t, whole.Cmp(new(big.Int).Add(sum, big.NewInt(1))), 0, "whole above split+1")
			}
			r, err := quoteOut(t, s, p[0], p[1], big.NewInt(a))
			require.NoError(t, err)
			if prev != nil {
				assert.GreaterOrEqual(t, r.TokenAmountOut.Amount.Cmp(prev), 0, "not monotonic %s->%s", p[0][:6], p[1][:6])
			}
			prev = r.TokenAmountOut.Amount
			switch p[1] {
			case testOHMAddress:
				assert.LessOrEqual(t, r.TokenAmountOut.Amount.Cmp(testOHMReserve.ToBig()), 0)
			case testSOHMAddress:
				assert.LessOrEqual(t, r.TokenAmountOut.Amount.Cmp(testSOHMReserve.ToBig()), 0)
			}
		}
	}
}

// FuzzCalcAmountOut checks the simulator against the reference model on random amounts, pairs and reserves. The
// amount stays below 2^128: the simulator's a*1e18 and a*index are 256-bit products.
func FuzzCalcAmountOut(f *testing.F) {
	f.Add(uint64(1), uint64(0), uint8(0), uint8(1), uint64(13862025744940277), uint64(28443408722380364))
	f.Add(uint64(1e18), uint64(7), uint8(2), uint8(0), uint64(5_000_000), uint64(3_000_000))
	f.Add(uint64(0), uint64(1), uint8(1), uint8(2), uint64(0), uint64(0))
	f.Add(uint64(123456789012), uint64(0), uint8(2), uint8(1), uint64(1000), uint64(1))
	tokens := []string{testOHMAddress, testSOHMAddress, testGOHMAddress}
	f.Fuzz(func(t *testing.T, lo, hi uint64, i, j uint8, ohm, sohm uint64) {
		amt := new(big.Int).Or(new(big.Int).Lsh(new(big.Int).SetUint64(hi), 64), new(big.Int).SetUint64(lo))
		in, out := tokens[int(i)%3], tokens[int(j)%3]
		s := newSimulatorWithReserves(t, indexAt25147546, 0, uint256.NewInt(ohm), uint256.NewInt(sohm))
		st := refState{index: indexAt25147546.ToBig(), ohmReserve: new(big.Int).SetUint64(ohm),
			sohmReserve: new(big.Int).SetUint64(sohm)}
		checkAgainstRef(t, s, st, in, out, amt)
	})
}
