package gohm

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

// Spot values at block 25147546's index, by hand from gOHM.balanceTo / balanceFrom: 1e12 * 1e18 / 269238508004 =
// 3714178953870682139 and 1e20 * 269238508004 / 1e18 = 26923850800400. The simulator and the reference model both
// give them.
func TestReferenceModel_SpotValues(t *testing.T) {
	t.Parallel()
	st := refState{index: indexAt25147546.ToBig(), ohmReserve: testOHMReserve.ToBig(), sohmReserve: testSOHMReserve.ToBig()}
	sim := newSimulator(t, indexAt25147546, 0)
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
		res, err := quoteOut(t, sim, tc.in, tc.out, amount)
		require.NoError(t, err)
		assert.Equal(t, tc.want, res.TokenAmountOut.Amount.String())
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

// A clone's swap leaves the original's quote, and concurrent quotes agree, for every pair.
func TestCloneStateAndConcurrentQuotes(t *testing.T) {
	t.Parallel()
	s := newSimulator(t, indexAt25147546, 0)
	for _, p := range invariantPairs {
		params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p[0], Amount: big.NewInt(1e9)},
			TokenOut: p[1]}
		testutil.TestCloneState(t, s, params, nil)
		_, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) { return s.CalcAmountOut(params) })
		require.NoError(t, err)
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
