package netstaking

import (
	"errors"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// refState is the staking and wrapper state the reference model reads.
type refState struct {
	index, netReserve, sNetStakingReserve, sNetWrapReserve *big.Int
}

type refQuote struct {
	out    *big.Int
	gas    int64
	action Action
	err    error
}

var wad = big.NewInt(1e18)

// refCalcAmountOut is the contracts' math, written in math/big apart from the simulator. wsNET is an Olympus wsOHM
// fork: wrap is sOHMTowOHM(a) = a * 1e18 / index, unwrap is wOHMTosOHM(a) = a * index / 1e18; Staking.stake and
// unstake move NET and sNET 1:1. Each leg pays out of a contract balance, which caps it: stake pays sNET from
// staking, unstake NET from staking, unwrap sNET from the wrapper; wrap mints. The composites chain the legs.
// The gas is the per-action estimate in constant.go, written out here.
func refCalcAmountOut(st refState, in, out string, amount *big.Int) refQuote {
	if amount.Cmp(new(big.Int).Lsh(big.NewInt(1), 256)) >= 0 {
		return refQuote{err: ErrInvalidTokenIn}
	}
	if amount.Sign() == 0 {
		return refQuote{err: ErrZeroAmount}
	}
	known := func(tok string) bool {
		return tok == testNETAddress || tok == testSNETAddress || tok == testWSNETAddress
	}
	if !known(in) {
		return refQuote{err: ErrInvalidTokenIn}
	}
	if in == out || !known(out) {
		return refQuote{err: ErrInvalidTokenOut}
	}
	over := func(v, reserve *big.Int) bool { return v.Cmp(reserve) > 0 }
	wrap := func(a *big.Int) *big.Int { return new(big.Int).Div(new(big.Int).Mul(a, wad), st.index) }
	unwrap := func(a *big.Int) *big.Int { return new(big.Int).Div(new(big.Int).Mul(a, st.index), wad) }
	ok := func(v *big.Int, gas int64, a Action) refQuote { return refQuote{out: v, gas: gas, action: a} }
	zeroIndex := st.index.Sign() == 0
	switch {
	case in == testNETAddress && out == testSNETAddress:
		if over(amount, st.sNetStakingReserve) {
			return refQuote{err: ErrInsufficientLiquidity}
		}
		return ok(amount, 42258, ActionStake)
	case in == testSNETAddress && out == testNETAddress:
		if over(amount, st.netReserve) {
			return refQuote{err: ErrInsufficientLiquidity}
		}
		return ok(amount, 40162, ActionUnstake)
	case in == testSNETAddress: // wrap: minted, no cap
		if zeroIndex {
			return refQuote{err: ErrIndexZero}
		}
		return ok(wrap(amount), 37900, ActionWrap)
	case in == testNETAddress: // stake (capped) then wrap
		if over(amount, st.sNetStakingReserve) {
			return refQuote{err: ErrInsufficientLiquidity}
		}
		if zeroIndex {
			return refQuote{err: ErrIndexZero}
		}
		return ok(wrap(amount), 42258+37900, ActionStakeAndWrap)
	}
	if zeroIndex {
		return refQuote{err: ErrIndexZero}
	}
	s := unwrap(amount)
	if over(s, st.sNetWrapReserve) {
		return refQuote{err: ErrInsufficientLiquidity}
	}
	if out == testSNETAddress {
		return ok(s, 37739, ActionUnwrap)
	}
	if over(s, st.netReserve) { // then unstake, 1:1
		return refQuote{err: ErrInsufficientLiquidity}
	}
	return ok(s, 37739+40162, ActionUnwrapAndUnstake)
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

func liveState(index *uint256.Int) refState {
	return refState{index.ToBig(), testNETReserve.ToBig(), testSNetStakingReserve.ToBig(), testSNetWrapReserve.ToBig()}
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

// TestCalcAmountOut_MatchesReferenceModel quotes every token pair and grid amount on the live, capped-reserve and
// zero-index states, and one where NET binds before the wrapper's sNET, against refCalcAmountOut.
func TestCalcAmountOut_MatchesReferenceModel(t *testing.T) {
	t.Parallel()
	capped := refState{indexLive.ToBig(), big.NewInt(5_000_000), big.NewInt(3_000_000), big.NewInt(2_000_000)}
	fixtures := []struct {
		name string
		sim  *PoolSimulator
		st   refState
	}{
		{"live", newSimulator(t, indexLive), liveState(indexLive)},
		{"capped", newSimulatorWithReserves(t, indexLive,
			uint256.NewInt(5_000_000), uint256.NewInt(3_000_000), uint256.NewInt(2_000_000)), capped},
		{"index0", newSimulator(t, uint256.NewInt(0)), liveState(uint256.NewInt(0))},
		// NET below the wrapper's sNET: the unstake leg of wsNET->NET binds, not the unwrap leg.
		{"netcap", newSimulatorWithReserves(t, indexLive,
			uint256.NewInt(1_000_000), uint256.NewInt(3_000_000), uint256.NewInt(2_000_000_000)),
			refState{indexLive.ToBig(), big.NewInt(1_000_000), big.NewInt(3_000_000), big.NewInt(2_000_000_000)}},
	}
	tokens := []string{testNETAddress, testSNETAddress, testWSNETAddress, "0x0000000000000000000000000000000000000001"}
	for _, fx := range fixtures {
		for _, in := range tokens {
			for _, out := range tokens {
				for _, amt := range gridAmounts() {
					checkAgainstRef(t, fx.sim, fx.st, in, out, amt)
				}
			}
		}
	}
}

// Spot values of the reference model at indexLive = 2718290645, by hand from the wsOHM formulas, so a wrong model
// would fail too: 1e9 * 1e18 / 2718290645 = 367878247986245047 and 1e17 * 2718290645 / 1e18 = 271829064.
func TestReferenceModel_SpotValues(t *testing.T) {
	t.Parallel()
	st := liveState(indexLive)
	for _, tc := range []struct {
		in, out, amount, want string
	}{
		{testSNETAddress, testWSNETAddress, "1000000000", "367878247986245047"},
		{testWSNETAddress, testSNETAddress, "100000000000000000", "271829064"},
		{testNETAddress, testSNETAddress, "1000", "1000"},
	} {
		amount, _ := new(big.Int).SetString(tc.amount, 10)
		q := refCalcAmountOut(st, tc.in, tc.out, amount)
		require.NoError(t, q.err)
		assert.Equal(t, tc.want, q.out.String())
	}
}

var invariantPairs = [][2]string{
	{testNETAddress, testSNETAddress}, {testSNETAddress, testNETAddress},
	{testSNETAddress, testWSNETAddress}, {testWSNETAddress, testSNETAddress},
	{testNETAddress, testWSNETAddress}, {testWSNETAddress, testNETAddress},
}

func quoteOut(s *PoolSimulator, in, out string, amt *big.Int) (*pool.CalcAmountOutResult, error) {
	return s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: amt}, TokenOut: out})
}

// TestInvariant_QuoteIsReadonly: quoting every pair leaves the simulator as it was.
func TestInvariant_QuoteIsReadonly(t *testing.T) {
	t.Parallel()
	s := newSimulator(t, indexLive)
	before := s.CloneState()
	for _, p := range invariantPairs {
		for _, amt := range gridAmounts() {
			_, _ = quoteOut(s, p[0], p[1], amt)
		}
	}
	assert.Equal(t, before, s)
}

// TestInvariant_CloneThenSwapLeavesOriginal: UpdateBalance on a clone moves the clone's
// reserves and not the original's.
func TestInvariant_CloneThenSwapLeavesOriginal(t *testing.T) {
	t.Parallel()
	for _, p := range invariantPairs {
		s := newSimulator(t, indexLive)
		before := s.CloneState()
		clone := s.CloneState()
		amt := big.NewInt(1_000_000_000_000) // the wsNET legs pay 0 sNET below about 3.7e8
		res, err := quoteOut(clone.(*PoolSimulator), p[0], p[1], amt)
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
// 1:1 legs exactly), out is non-decreasing in the input, and a NET or sNET output never exceeds the paying reserve.
func TestInvariant_SplitMonotonicBounded(t *testing.T) {
	t.Parallel()
	s := newSimulator(t, indexLive)
	amounts := []int64{1, 3, 999, 123_456_789, 1_000_000_007}
	for _, p := range invariantPairs {
		var prev *big.Int
		for _, a := range amounts {
			for _, b := range amounts {
				ra, errA := quoteOut(s, p[0], p[1], big.NewInt(a))
				rb, errB := quoteOut(s, p[0], p[1], big.NewInt(b))
				rw, errW := quoteOut(s, p[0], p[1], big.NewInt(a+b))
				require.NoError(t, errors.Join(errA, errB, errW))
				sum := new(big.Int).Add(ra.TokenAmountOut.Amount, rb.TokenAmountOut.Amount)
				whole := rw.TokenAmountOut.Amount
				assert.LessOrEqual(t, sum.Cmp(whole), 0, "split above whole %s->%s %d+%d", p[0][:6], p[1][:6], a, b)
				assert.LessOrEqual(t, whole.Cmp(new(big.Int).Add(sum, big.NewInt(1))), 0, "whole above split+1")
			}
			r, err := quoteOut(s, p[0], p[1], big.NewInt(a))
			require.NoError(t, err)
			if prev != nil {
				assert.GreaterOrEqual(t, r.TokenAmountOut.Amount.Cmp(prev), 0, "not monotonic %s->%s", p[0][:6], p[1][:6])
			}
			prev = r.TokenAmountOut.Amount
			switch p[1] {
			case testNETAddress:
				assert.LessOrEqual(t, r.TokenAmountOut.Amount.Cmp(testNETReserve.ToBig()), 0)
			case testSNETAddress:
				assert.LessOrEqual(t, r.TokenAmountOut.Amount.Cmp(testSNetStakingReserve.ToBig()), 0)
			}
		}
	}
}

// FuzzCalcAmountOut checks the simulator against the reference model on random amounts, pairs and reserves.
func FuzzCalcAmountOut(f *testing.F) {
	f.Add(uint64(1), uint64(0), uint8(0), uint8(1), uint64(62271915033974), uint64(13591391263407162106), uint64(3240061835399))
	f.Add(uint64(1e18), uint64(7), uint8(2), uint8(0), uint64(5_000_000), uint64(3_000_000), uint64(2_000_000))
	f.Add(uint64(0), uint64(1), uint8(1), uint8(2), uint64(0), uint64(0), uint64(0))
	f.Add(uint64(123456789012), uint64(0), uint8(2), uint8(1), uint64(1000), uint64(1), uint64(1))
	tokens := []string{testNETAddress, testSNETAddress, testWSNETAddress}
	f.Fuzz(func(t *testing.T, lo, hi uint64, i, j uint8, netR, stakeR, wrapR uint64) {
		amt := new(big.Int).Or(new(big.Int).Lsh(new(big.Int).SetUint64(hi), 64), new(big.Int).SetUint64(lo))
		s := newSimulatorWithReserves(t, indexLive, uint256.NewInt(netR), uint256.NewInt(stakeR), uint256.NewInt(wrapR))
		st := refState{indexLive.ToBig(), new(big.Int).SetUint64(netR), new(big.Int).SetUint64(stakeR),
			new(big.Int).SetUint64(wrapR)}
		checkAgainstRef(t, s, st, tokens[int(i)%3], tokens[int(j)%3], amt)
	})
}
