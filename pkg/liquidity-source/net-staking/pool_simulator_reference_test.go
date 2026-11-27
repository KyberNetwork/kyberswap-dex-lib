package netstaking

import (
	"errors"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
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

// Spot values of the reference model at indexLive = 2718290645, by hand from the wsOHM formulas, so a wrong model
// would fail too: 1e9 * 1e18 / 2718290645 = 367878247986245047 and 1e17 * 2718290645 / 1e18 = 271829064. The
// simulator gives them too.
func TestReferenceModel_SpotValues(t *testing.T) {
	t.Parallel()
	st := liveState(indexLive)
	sim := newSimulator(t, indexLive)
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
		res, err := quoteOut(sim, tc.in, tc.out, amount)
		require.NoError(t, err)
		assert.Equal(t, tc.want, res.TokenAmountOut.Amount.String())
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

// A clone's swap leaves the original's quote, and concurrent quotes agree, for every pair.
func TestCloneStateAndConcurrentQuotes(t *testing.T) {
	t.Parallel()
	s := newSimulator(t, indexLive)
	for _, p := range invariantPairs {
		params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p[0], Amount: big.NewInt(1e9)},
			TokenOut: p[1]}
		testutil.TestCloneState(t, s, params, nil)
		_, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) { return s.CalcAmountOut(params) })
		require.NoError(t, err)
	}
}

// FuzzCalcAmountOut checks the simulator against the reference model on random amounts, pairs and reserves.
func FuzzCalcAmountOut(f *testing.F) {
	f.Add(uint64(1), uint64(0), uint8(0), uint8(1), uint64(62271915033974), uint64(13591391263407162106), uint64(3240061835399))
	f.Add(uint64(1e18), uint64(7), uint8(2), uint8(0), uint64(5_000_000), uint64(3_000_000), uint64(2_000_000))
	f.Add(uint64(0), uint64(1), uint8(1), uint8(2), uint64(0), uint64(0), uint64(0))
	f.Add(uint64(123456789012), uint64(0), uint8(2), uint8(1), uint64(1000), uint64(1), uint64(1))
	// NET below the wrapper's sNET: the unstake leg of wsNET->NET binds, not the unwrap leg.
	f.Add(uint64(1e18), uint64(0), uint8(2), uint8(0), uint64(1_000_000), uint64(3_000_000), uint64(2_000_000_000))
	tokens := []string{testNETAddress, testSNETAddress, testWSNETAddress}
	f.Fuzz(func(t *testing.T, lo, hi uint64, i, j uint8, netR, stakeR, wrapR uint64) {
		amt := new(big.Int).Or(new(big.Int).Lsh(new(big.Int).SetUint64(hi), 64), new(big.Int).SetUint64(lo))
		s := newSimulatorWithReserves(t, indexLive, uint256.NewInt(netR), uint256.NewInt(stakeR), uint256.NewInt(wrapR))
		st := refState{indexLive.ToBig(), new(big.Int).SetUint64(netR), new(big.Int).SetUint64(stakeR),
			new(big.Int).SetUint64(wrapR)}
		checkAgainstRef(t, s, st, tokens[int(i)%3], tokens[int(j)%3], amt)
	})
}
