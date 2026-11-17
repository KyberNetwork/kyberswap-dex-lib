package oneinchaqua

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// ---- oracle and checker ------------------------------------------------------------------------------

var bigMax256 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))

// refMinAmountIn is the independent oracle of CalcAmountIn: the smallest amountIn at which the math/big model
// of the program reaches target or fails (its arithmetic overflows), found by growing the amountIn 16 times
// per step and bisecting. The model never falls as amountIn grows, and its failures start at one amountIn
// and cover everything above, so the predicate "reaches or fails" is monotone.
func refMinAmountIn(prog []Instruction, balIn, balOut, target *big.Int, inLt bool) (a *big.Int, viaError, found bool) {
	hit := func(x *big.Int) (reached, failed bool) {
		_, out, _, ok := refRun(prog, balIn, balOut, x, inLt)
		if !ok {
			return true, true
		}
		return out.Cmp(target) >= 0, false
	}
	hi := big.NewInt(1)
	for {
		if reached, _ := hit(hi); reached {
			break
		}
		if hi.Cmp(bigMax256) == 0 {
			return nil, false, false
		}
		hi = new(big.Int).Lsh(hi, 4)
		if hi.Cmp(bigMax256) > 0 {
			hi = new(big.Int).Set(bigMax256)
		}
	}
	lo := new(big.Int).Rsh(hi, 4) // a failed probe, or 0
	if hi.Cmp(bigMax256) == 0 {
		lo = new(big.Int) // the cap is not 16 times a probe
	}
	for new(big.Int).Sub(hi, lo).Cmp(big.NewInt(1)) > 0 {
		mid := new(big.Int).Rsh(new(big.Int).Add(lo, hi), 1)
		if reached, _ := hit(mid); reached {
			hi = mid
		} else {
			lo = mid
		}
	}
	_, failed := hit(hi)
	return hi, failed, true
}

func inParams(s *PoolSimulator, in int, target *uint256.Int, limit pool.SwapLimit) pool.CalcAmountInParams {
	return pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: s.Info.Tokens[1-in], Amount: target.ToBig()},
		TokenIn:        s.Info.Tokens[in], Limit: limit,
	}
}

// verifyAmountIn runs CalcAmountIn and checks everything the contract says, against CalcAmountOut and against
// the independent oracle: the result is the smallest amountIn, its quote is the CalcAmountOut quote of it,
// nothing is mutated, and an error is only returned where the exact-in direction has no answer either.
func verifyAmountIn(t testing.TB, s *PoolSimulator, in int, target *uint256.Int, limit pool.SwapLimit) (*pool.CalcAmountInResult, error) {
	t.Helper()
	before := snapOf(s)
	res, err := s.CalcAmountIn(inParams(s, in, target, limit))
	require.Equal(t, before, snapOf(s), "CalcAmountIn changed the pool")
	if err != nil {
		require.NotErrorIs(t, err, ErrNoConvergence)
	}

	inLt := (in == 0) == s.token0Lt
	want, viaError, found := refMinAmountIn(s.program, s.State.Raw[in].ToBig(), s.State.Raw[1-in].ToBig(), target.ToBig(), inLt)
	var wantQuote *pool.CalcAmountOutResult
	var wantErr error
	if found && !viaError {
		var w uint256.Int
		require.False(t, w.SetFromBig(want))
		wantQuote, wantErr = quoteOut(s, in, &w, limit)
	}
	if !found || viaError || wantErr != nil {
		require.Error(t, err, "target %s: no amountIn quotes it (oracle %v found %v viaError %v quoteErr %v) but CalcAmountIn gave %v",
			target.Dec(), want, found, viaError, wantErr, res)
		require.Nil(t, res)
		return nil, err
	}
	require.NoError(t, err, "target %s: amountIn %s quotes it", target.Dec(), want)

	a := res.TokenAmountIn
	require.Equal(t, s.Info.Tokens[in], a.Token)
	require.Equal(t, want.String(), a.Amount.String(), "target %s: not the smallest amountIn", target.Dec())
	require.Equal(t, s.Info.Tokens[1-in], res.RemainingTokenAmountOut.Token)
	require.Zero(t, res.RemainingTokenAmountOut.Amount.Sign())

	// It is the CalcAmountOut quote of that amountIn: same fee, gas, SwapInfo, and enough out.
	require.Equal(t, wantQuote.Fee, res.Fee)
	require.Equal(t, wantQuote.Gas, res.Gas)
	require.Equal(t, wantQuote.SwapInfo, res.SwapInfo)
	require.GreaterOrEqual(t, wantQuote.TokenAmountOut.Amount.Cmp(target.ToBig()), 0)

	// One wei less is not enough: it quotes less than the target, or nothing.
	if a.Amount.Cmp(big.NewInt(1)) > 0 {
		var less uint256.Int
		less.SetFromBig(a.Amount)
		less.SubUint64(&less, 1)
		r, err := quoteOut(s, in, &less, limit)
		if err == nil {
			require.Negative(t, r.TokenAmountOut.Amount.Cmp(target.ToBig()), "amountIn-1 quotes the target too")
		} else {
			require.ErrorIs(t, err, ErrZeroAmountOut)
		}
	}
	// Mutating a result changes nothing in the pool.
	other, err := s.CalcAmountIn(inParams(s, in, target, limit))
	require.NoError(t, err)
	other.TokenAmountIn.Amount.SetInt64(0)
	other.Fee.Amount.SetInt64(-1)
	require.Equal(t, before, snapOf(s))
	return res, nil
}

// ---- generators --------------------------------------------------------------------------------------

// randUpTo returns a value in [1, max] whose bit length is uniform: dust to the cap.
func randUpTo(rng *rand.Rand, max *uint256.Int) *uint256.Int {
	if max.IsZero() {
		return uint256.NewInt(1)
	}
	bits := 1 + rng.IntN(max.BitLen())
	var v uint256.Int
	for i := range v {
		v[i] = rng.Uint64()
	}
	v.Rsh(&v, uint(256-bits))
	if v.IsZero() {
		v.SetOne()
	}
	if v.Gt(max) {
		v.Set(max)
	}
	return &v
}

type rawProfileCase struct {
	name string
	raw  [2]*uint256.Int
}

var propProfiles = []rawProfileCase{
	{"A", [2]*uint256.Int{e18(1000), e18(2000)}},
	{"tiny", [2]*uint256.Int{uint256.NewInt(1), uint256.NewInt(3)}},
	{"steep", [2]*uint256.Int{uint256.NewInt(7), u("1000000000000000000000000000000")}},
	{"flat", [2]*uint256.Int{u("1000000000000000000000000000000"), uint256.NewInt(7)}},
	{"empty-in", [2]*uint256.Int{new(uint256.Int), e18(5000)}},
	{"huge", [2]*uint256.Int{new(uint256.Int).Lsh(uint256.NewInt(1), 200), new(uint256.Int).Lsh(uint256.NewInt(1), 210)}},
	{"cap248", [2]*uint256.Int{max248, max248}},
}

// walletModes are the maker wallet limits of the property sims: not binding, binding at half the strategy's
// out side, binding at a few wei.
var openWallet = [2]*uint256.Int{unlimited, unlimited}

var walletModes = []string{"open", "half", "dust"}

func propSim(t testing.TB, program []byte, launch bool, raw [2]*uint256.Int, mode string) *PoolSimulator {
	o := poolOpts{program: program, raw: raw}
	if launch {
		o.app = testLaunch
	}
	switch mode {
	case "open":
		o.wallet = [2]*uint256.Int{unlimited, unlimited}
	case "half":
		half := func(v *uint256.Int) *uint256.Int { return new(uint256.Int).AddUint64(new(uint256.Int).Rsh(v, 1), 1) }
		o.wallet = [2]*uint256.Int{half(raw[0]), half(raw[1])}
		o.allowance = [2]*uint256.Int{unlimited, unlimited}
	case "dust":
		o.wallet = [2]*uint256.Int{uint256.NewInt(5), uint256.NewInt(9)}
		o.allowance = [2]*uint256.Int{unlimited, unlimited}
	}
	return newSim(t, o)
}

// targetsFor lists the targets of a direction: the edges, a few sizes from dust to the cap, one above.
func targetsFor(rng *rand.Rand, rawOut *uint256.Int, random int) []*uint256.Int {
	ts := []*uint256.Int{uint256.NewInt(1), uint256.NewInt(2), uint256.NewInt(3)}
	if !rawOut.IsZero() {
		ts = append(ts, rawOut.Clone(), new(uint256.Int).AddUint64(rawOut, 1))
		if rawOut.Gt(uint256.NewInt(1)) {
			ts = append(ts, new(uint256.Int).SubUint64(rawOut, 1))
		}
	}
	for range random {
		ts = append(ts, randUpTo(rng, rawOut))
	}
	return ts
}

// ---- tests -------------------------------------------------------------------------------------------

// The shared exact-out test of the repository, on every live shape and both routers.
func TestCalcAmountInTestutil(t *testing.T) {
	t.Parallel()
	for _, ls := range liveShapes {
		for _, launch := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/launch=%v", ls.name, launch), func(t *testing.T) {
				t.Parallel()
				o := poolOpts{program: buildProgram(ls.wrappers, ls.salt), wallet: [2]*uint256.Int{unlimited, unlimited}}
				if launch {
					o.app = testLaunch
				}
				testutil.TestCalcAmountIn(t, newSim(t, o))
			})
		}
	}
}

// Property: every live shape, both routers, both directions, every profile and wallet mode, targets from
// dust to the strategy balance and one above.
func TestCalcAmountInPropertyLiveShapes(t *testing.T) {
	t.Parallel()
	for _, ls := range liveShapes {
		t.Run(ls.name, func(t *testing.T) {
			t.Parallel()
			var ok, refused int
			rng := rand.New(rand.NewPCG(1, uint64(len(ls.name))))
			program := buildProgram(ls.wrappers, ls.salt)
			for _, launch := range []bool{false, true} {
				for _, p := range propProfiles {
					for _, mode := range walletModes {
						s := propSim(t, program, launch, p.raw, mode)
						for in := range 2 {
							for _, target := range targetsFor(rng, &s.State.Raw[1-in], 3) {
								if _, err := verifyAmountIn(t, s, in, target, nil); err == nil {
									ok++
								} else {
									refused++
								}
							}
						}
					}
				}
			}
			// both outcomes are exercised: quotes and refusals
			require.Greater(t, ok, 100, "quoted cases")
			require.Greater(t, refused, 100, "refused cases")
		})
	}
}

// Property: every ordering of up to three wrappers and both salt variants.
func TestCalcAmountInPropertyEveryOrdering(t *testing.T) {
	t.Parallel()
	for _, seq := range wrapperSequences(3) {
		for _, salt := range []string{"none", "both"} {
			t.Run(orDash(seq)+"/"+salt, func(t *testing.T) {
				t.Parallel()
				rng := rand.New(rand.NewPCG(2, uint64(len(seq))*7+uint64(len(salt))))
				program := buildProgram(seq, salt)
				for _, p := range propProfiles[:6] {
					s := propSim(t, program, false, p.raw, "open")
					for in := range 2 {
						for _, target := range targetsFor(rng, &s.State.Raw[1-in], 2) {
							_, _ = verifyAmountIn(t, s, in, target, nil)
						}
					}
				}
			})
		}
	}
}

// Sweep: consecutive targets near the exact-in boundary, where the rounding of every wrapper decides.
func TestCalcAmountInSweepSmallTargets(t *testing.T) {
	t.Parallel()
	for _, ls := range liveShapes {
		t.Run(ls.name, func(t *testing.T) {
			t.Parallel()
			s := propSim(t, buildProgram(ls.wrappers, ls.salt), false,
				[2]*uint256.Int{uint256.NewInt(1000), uint256.NewInt(997)}, "open")
			for in := range 2 {
				for target := uint64(1); target <= 80; target++ {
					_, _ = verifyAmountIn(t, s, in, uint256.NewInt(target), nil)
				}
			}
		})
	}
}

// The exact-in rounding rules decide, not the router's exact-out branches: with a floor protocol fee of 50%
// on a 1:1 strategy, a net of 2 (the least that pays out 1 wei) needs 3 in (3 - floor(3/2) = 2, and 2 - 1 = 1
// is too little), where inverting by ceil(net*1e9/(1e9-bps)) would ask for 4.
func TestCalcAmountInUsesExactInRounding(t *testing.T) {
	t.Parallel()
	protocol := newSim(t, poolOpts{program: prog(iProtocol(500_000_000, testRecipient), iSwap()), raw: [2]*uint256.Int{e18(1000), e18(1000)}})
	res, err := verifyAmountIn(t, protocol, 0, uint256.NewInt(1), nil)
	require.NoError(t, err)
	require.Equal(t, "3", res.TokenAmountIn.Amount.String())
	flatOne := newSim(t, poolOpts{program: prog(iFlat(500_000_000), iSwap()), raw: [2]*uint256.Int{e18(1000), e18(1000)}})
	res, err = verifyAmountIn(t, flatOne, 0, uint256.NewInt(1), nil)
	require.NoError(t, err)
	require.Equal(t, "4", res.TokenAmountIn.Amount.String(), "flat 50%: x - ceil(x/2) = floor(x/2) >= 2 needs 4")

	// Golden values for a plain 1:1 swap of raw 1000e18 each with a net of 1: out(1) = floor(1e21/(1e21+1)) = 0.
	// Use a steep curve where one wei of net pays out more than one wei.
	steep := newSim(t, poolOpts{program: prog(iProtocol(500_000_000, testRecipient), iSwap()), raw: [2]*uint256.Int{uint256.NewInt(1), e18(1000)}})
	res, err = verifyAmountIn(t, steep, 0, uint256.NewInt(1), nil)
	require.NoError(t, err)
	require.Equal(t, "1", res.TokenAmountIn.Amount.String(), "floor(1*5e8/1e9) = 0 is taken, so net = 1")
	flat := newSim(t, poolOpts{program: prog(iFlat(500_000_000), iSwap()), raw: [2]*uint256.Int{uint256.NewInt(1), e18(1000)}})
	res, err = verifyAmountIn(t, flat, 0, uint256.NewInt(1), nil)
	require.NoError(t, err)
	require.Equal(t, "2", res.TokenAmountIn.Amount.String(), "ceil(1*5e8/1e9) = 1 is taken, so 2 for a net of 1")
}

// Golden values from the closed form of single-wrapper programs, with big.Int arithmetic.
func TestCalcAmountInGolden(t *testing.T) {
	t.Parallel()
	bIn, bOut := e18(1000), e18(2000)
	target := e18(100)
	// xyc only: net = ceil(out*bIn/(bOut-out)).
	net := new(big.Int).Mul(target.ToBig(), bIn.ToBig())
	d := new(big.Int).Sub(bOut.ToBig(), target.ToBig())
	netCeil := new(big.Int).Div(new(big.Int).Add(net, new(big.Int).Sub(d, big.NewInt(1))), d)
	s := newSim(t, poolOpts{program: prog(iSwap()), raw: [2]*uint256.Int{bIn, bOut}})
	res, err := verifyAmountIn(t, s, 0, target, nil)
	require.NoError(t, err)
	require.Equal(t, netCeil.String(), res.TokenAmountIn.Amount.String())
	require.Equal(t, "52631578947368421053", netCeil.String())

	// flat fee 1%: a = ceil(net*1e9/(1e9-1e7)).
	flatSim := newSim(t, poolOpts{program: prog(iFlat(10_000_000), iSwap()), raw: [2]*uint256.Int{bIn, bOut}})
	res, err = verifyAmountIn(t, flatSim, 0, target, nil)
	require.NoError(t, err)
	want := new(big.Int).Mul(netCeil, big.NewInt(1_000_000_000))
	want.Add(want, big.NewInt(989_999_999)).Div(want, big.NewInt(990_000_000))
	require.Equal(t, want.String(), res.TokenAmountIn.Amount.String())

	// protocol fee 1%: a = floor((net-1)*1e9/(1e9-1e7)) + 1, and the fee is collected.
	protocolSim := newSim(t, poolOpts{program: prog(iProtocol(10_000_000, testRecipient), iSwap()), raw: [2]*uint256.Int{bIn, bOut}})
	res, err = verifyAmountIn(t, protocolSim, 0, target, nil)
	require.NoError(t, err)
	want = new(big.Int).Mul(new(big.Int).Sub(netCeil, big.NewInt(1)), big.NewInt(1_000_000_000))
	want.Div(want, big.NewInt(990_000_000)).Add(want, big.NewInt(1))
	require.Equal(t, want.String(), res.TokenAmountIn.Amount.String())
	require.NotZero(t, res.Fee.Amount.Sign())

	// the protocol fee before the flat fee: the flat fee is inverted first (it is the innermost).
	both := newSim(t, poolOpts{program: prog(iProtocol(20_000_000, testRecipient), iFlat(30_000_000), iSwap()), raw: [2]*uint256.Int{bIn, bOut}})
	res, err = verifyAmountIn(t, both, 0, target, nil)
	require.NoError(t, err)
	inner := new(big.Int).Mul(netCeil, big.NewInt(1_000_000_000))
	inner.Add(inner, big.NewInt(969_999_999)).Div(inner, big.NewInt(970_000_000)) // flat 3%
	outer := new(big.Int).Mul(new(big.Int).Sub(inner, big.NewInt(1)), big.NewInt(1_000_000_000))
	outer.Div(outer, big.NewInt(980_000_000)).Add(outer, big.NewInt(1)) // protocol 2%
	require.Equal(t, outer.String(), res.TokenAmountIn.Amount.String())
}

// Invalid inputs: every one is an error and leaves the pool alone.
func TestCalcAmountInInvalidInput(t *testing.T) {
	t.Parallel()
	s := newSim(t, poolOpts{})
	t0, t1 := s.Info.Tokens[0], s.Info.Tokens[1]
	huge := new(big.Int).Lsh(big.NewInt(1), 256)
	for name, tc := range map[string]struct {
		p    pool.CalcAmountInParams
		want error
	}{
		"unknown token in":     {pool.CalcAmountInParams{TokenIn: "0xdead", TokenAmountOut: pool.TokenAmount{Token: t1, Amount: big.NewInt(1)}}, ErrInvalidToken},
		"unknown token out":    {pool.CalcAmountInParams{TokenIn: t0, TokenAmountOut: pool.TokenAmount{Token: "0xdead", Amount: big.NewInt(1)}}, ErrInvalidToken},
		"empty token in":       {pool.CalcAmountInParams{TokenAmountOut: pool.TokenAmount{Token: t1, Amount: big.NewInt(1)}}, ErrInvalidToken},
		"same token":           {pool.CalcAmountInParams{TokenIn: t0, TokenAmountOut: pool.TokenAmount{Token: t0, Amount: big.NewInt(1)}}, ErrInvalidToken},
		"nil amount":           {pool.CalcAmountInParams{TokenIn: t0, TokenAmountOut: pool.TokenAmount{Token: t1}}, ErrInvalidAmountOut},
		"zero amount":          {pool.CalcAmountInParams{TokenIn: t0, TokenAmountOut: pool.TokenAmount{Token: t1, Amount: big.NewInt(0)}}, ErrInvalidAmountOut},
		"negative amount":      {pool.CalcAmountInParams{TokenIn: t0, TokenAmountOut: pool.TokenAmount{Token: t1, Amount: big.NewInt(-5)}}, ErrInvalidAmountOut},
		"amount above 2^256-1": {pool.CalcAmountInParams{TokenIn: t0, TokenAmountOut: pool.TokenAmount{Token: t1, Amount: huge}}, ErrInvalidAmountOut},
	} {
		before := snapOf(s)
		res, err := s.CalcAmountIn(tc.p)
		require.ErrorIs(t, err, tc.want, name)
		require.Nil(t, res, name)
		require.Equal(t, before, snapOf(s), name)
	}
	// 2^256-1 is a valid uint256 and simply exceeds the strategy balance.
	_, err := s.CalcAmountIn(pool.CalcAmountInParams{TokenIn: t0, TokenAmountOut: pool.TokenAmount{Token: t1, Amount: new(big.Int).Sub(huge, big.NewInt(1))}})
	require.ErrorIs(t, err, ErrInsufficientRaw)
}

// The state guards of CalcAmountOut apply to CalcAmountIn, in the same order.
func TestCalcAmountInStateGuards(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		o    poolOpts
		want error
	}{
		"docked":          {poolOpts{docked: true}, ErrStrategyTerminal},
		"inactive":        {poolOpts{inactive: true}, ErrStrategyTerminal},
		"not tracked yet": {poolOpts{untracked: true}, ErrNoWalletSnapshot},
	} {
		s := newSim(t, tc.o)
		for in := range 2 {
			_, err := s.CalcAmountIn(inParams(s, in, uint256.NewInt(1), nil))
			require.ErrorIs(t, err, tc.want, name)
			_, outErr := quoteOut(s, in, uint256.NewInt(1), nil)
			require.ErrorIs(t, outErr, tc.want, name+" (exact-in)")
		}
	}
}

// Zero balances: an empty in side is refused unless a concentrate gives it liquidity; an empty out side
// holds nothing to take.
func TestCalcAmountInZeroBalances(t *testing.T) {
	t.Parallel()
	plain := newSim(t, poolOpts{program: prog(iSwap()), raw: [2]*uint256.Int{new(uint256.Int), e18(10)}})
	_, err := plain.CalcAmountIn(inParams(plain, 0, uint256.NewInt(1), nil))
	require.ErrorIs(t, err, ErrZeroBalance)
	// the other direction has an empty out side
	_, err = plain.CalcAmountIn(inParams(plain, 1, uint256.NewInt(1), nil))
	require.ErrorIs(t, err, ErrInsufficientRaw)

	conc := newSim(t, poolOpts{program: buildProgram("C", "none"), raw: [2]*uint256.Int{new(uint256.Int), e18(10)}})
	for in := range 2 {
		for _, target := range []uint64{1, 5, 1_000_000} {
			_, _ = verifyAmountIn(t, conc, in, uint256.NewInt(target), nil)
		}
	}
	// no balance at all
	empty := newSim(t, poolOpts{program: prog(iSwap()), raw: [2]*uint256.Int{new(uint256.Int), new(uint256.Int)}})
	_, err = empty.CalcAmountIn(inParams(empty, 0, uint256.NewInt(1), nil))
	require.ErrorIs(t, err, ErrInsufficientRaw)
}

// A fee of 100% leaves nothing for the swap; 100% minus one part in 1e9 needs a huge but finite amountIn.
func TestCalcAmountInFullFee(t *testing.T) {
	t.Parallel()
	for _, mk := range []func(uint32) []byte{iFlat, func(b uint32) []byte { return iProtocol(b, testRecipient) }} {
		s := newSim(t, poolOpts{program: prog(mk(1_000_000_000), iSwap())})
		_, err := s.CalcAmountIn(inParams(s, 0, uint256.NewInt(1), nil))
		require.ErrorIs(t, err, ErrNoAmountIn)
		_, err = quoteOut(s, 0, e18(1), nil)
		require.ErrorIs(t, err, ErrZeroAmountOut, "the exact-in direction agrees: nothing comes out")

		near := newSim(t, poolOpts{program: prog(mk(999_999_999), iSwap()), raw: [2]*uint256.Int{e18(1000), e18(2000)}})
		for in := range 2 {
			_, _ = verifyAmountIn(t, near, in, uint256.NewInt(1), nil)
			_, _ = verifyAmountIn(t, near, in, e18(1), nil)
		}
	}
	// the fee applies at the inner level too
	inner := newSim(t, poolOpts{program: prog(iFlat(10), iProtocol(1_000_000_000, testRecipient), iSwap())})
	_, err := inner.CalcAmountIn(inParams(inner, 0, uint256.NewInt(1), nil))
	require.ErrorIs(t, err, ErrNoAmountIn)
}

// Caps: the strategy balance, the maker wallet (own snapshot and shared limit), each at, one below and
// one above the boundary.
func TestCalcAmountInCaps(t *testing.T) {
	t.Parallel()
	program := buildProgram("CF", "none")
	raw := [2]*uint256.Int{e18(1000), e18(2000)}

	// Strategy balance: with no concentrate the whole balance can never be paid; one above is refused as well.
	plain := newSim(t, poolOpts{program: prog(iSwap()), raw: raw})
	for _, target := range []*uint256.Int{raw[1], new(uint256.Int).AddUint64(raw[1], 1)} {
		_, err := plain.CalcAmountIn(inParams(plain, 0, target, nil))
		require.ErrorIs(t, err, ErrInsufficientRaw)
		require.ErrorIs(t, err, pool.ErrNotEnoughInventory)
	}
	_, err := verifyAmountIn(t, plain, 0, new(uint256.Int).SubUint64(raw[1], 1), nil)
	require.NoError(t, err, "one wei below the balance has an amountIn (and a huge one)")
	// With a concentrate the virtual balance is larger: the whole strategy balance can be paid.
	conc := newSim(t, poolOpts{program: program, raw: raw})
	res, err := verifyAmountIn(t, conc, 0, raw[1], nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	_, err = conc.CalcAmountIn(inParams(conc, 0, new(uint256.Int).AddUint64(raw[1], 1), nil))
	require.ErrorIs(t, err, ErrInsufficientRaw)

	// Wallet, own snapshot: the limit is min(balance, allowance) and must cover what the smallest amountIn
	// pays, X, which can exceed the target by a rounding step.
	x := e18(10)
	ref, err := verifyAmountIn(t, newSim(t, poolOpts{program: program, raw: raw}), 0, x, nil)
	require.NoError(t, err)
	paid := ref.SwapInfo.(*SwapInfo).AmountOut
	paidM1, paidP1 := new(uint256.Int).SubUint64(&paid, 1), new(uint256.Int).AddUint64(&paid, 1)
	require.False(t, paid.Lt(x))
	for name, tc := range map[string]struct {
		wallet, allowance *uint256.Int
		want              error
	}{
		"balance equals paid":      {&paid, unlimited, nil},
		"balance one below paid":   {paidM1, unlimited, ErrInsufficientWallet},
		"balance one above paid":   {paidP1, unlimited, nil},
		"balance below target":     {new(uint256.Int).SubUint64(x, 1), unlimited, ErrInsufficientWallet},
		"allowance equals paid":    {unlimited, &paid, nil},
		"allowance one below paid": {unlimited, paidM1, ErrInsufficientWallet},
		"allowance zero":           {unlimited, new(uint256.Int), ErrInsufficientWallet},
		"balance zero":             {new(uint256.Int), unlimited, ErrInsufficientWallet},
	} {
		s := newSim(t, poolOpts{program: program, raw: raw, wallet: [2]*uint256.Int{nil, tc.wallet}, allowance: [2]*uint256.Int{nil, tc.allowance}})
		_, err := s.CalcAmountIn(inParams(s, 0, x, nil))
		if tc.want != nil {
			require.ErrorIs(t, err, tc.want, name)
			require.ErrorIs(t, err, pool.ErrNotEnoughInventory, name)
		} else {
			require.NoError(t, err, name)
		}
		_, _ = verifyAmountIn(t, s, 0, x, nil)
	}

	// Wallet, shared limit: the key replaces the snapshot; a missing key and a negative one hold nothing.
	s := newSim(t, poolOpts{program: program, raw: raw})
	key := s.limitKeys[1]
	inv := func(v *big.Int) pool.SwapLimit {
		m := map[string]*big.Int{}
		if v != nil {
			m[key] = v
		}
		return swaplimit.NewInventory(DexType, m)
	}
	huge := new(big.Int).Lsh(big.NewInt(1), 300)
	for name, tc := range map[string]struct {
		limit *big.Int
		want  error
	}{
		"limit equals paid":    {paid.ToBig(), nil},
		"limit one below paid": {paidM1.ToBig(), ErrInsufficientWallet},
		"limit one above paid": {paidP1.ToBig(), nil},
		"limit zero":           {new(big.Int), ErrInsufficientWallet},
		"limit negative":       {big.NewInt(-1), ErrInsufficientWallet},
		"limit missing":        {nil, ErrInsufficientWallet},
		"limit above 2^256":    {huge, nil},
		"limit far above":      {e18(1_000_000).ToBig(), nil},
		"limit one wei":        {big.NewInt(1), ErrInsufficientWallet},
	} {
		l := inv(tc.limit)
		_, err := s.CalcAmountIn(inParams(s, 0, x, l))
		if tc.want != nil {
			require.ErrorIs(t, err, tc.want, name)
		} else {
			require.NoError(t, err, name)
		}
		_, _ = verifyAmountIn(t, s, 0, x, l)
	}
	// The shared limit wins over a smaller own snapshot, and a larger one does not help a small snapshot.
	own := newSim(t, poolOpts{program: program, raw: raw, wallet: [2]*uint256.Int{nil, new(uint256.Int).SubUint64(x, 1)}})
	_, err = own.CalcAmountIn(inParams(own, 0, x, nil))
	require.ErrorIs(t, err, ErrInsufficientWallet)
	_, err = own.CalcAmountIn(inParams(own, 0, x, swaplimit.NewInventory(DexType, map[string]*big.Int{own.limitKeys[1]: e18(1000).ToBig()})))
	require.NoError(t, err)

	// The smallest amountIn can pay out more than the target: the cap binds on what it pays, not on the
	// target. 1 wei in pays 500e18 on a (1, 1000e18) strategy, so a 400e18 target under a 450e18 wallet has
	// no amountIn at all.
	steep := newSim(t, poolOpts{program: prog(iSwap()), raw: [2]*uint256.Int{uint256.NewInt(1), e18(1000)},
		wallet: [2]*uint256.Int{nil, e18(450)}})
	res, err = steep.CalcAmountIn(inParams(steep, 0, e18(400), nil))
	require.ErrorIs(t, err, ErrInsufficientWallet)
	require.Nil(t, res)
	_, err = quoteOut(steep, 0, uint256.NewInt(1), nil)
	require.ErrorIs(t, err, ErrInsufficientWallet, "the exact-in direction agrees")
	steepOK := newSim(t, poolOpts{program: prog(iSwap()), raw: [2]*uint256.Int{uint256.NewInt(1), e18(1000)}})
	res, err = verifyAmountIn(t, steepOK, 0, e18(400), nil)
	require.NoError(t, err)
	require.Equal(t, "1", res.TokenAmountIn.Amount.String())
	// same with the strategy balance: 1 wei pays 500e18 but the strategy holds 100e18 of out
	tight := newSim(t, poolOpts{program: prog(iSwap()), raw: [2]*uint256.Int{uint256.NewInt(1), uint256.NewInt(100)}})
	res, err = verifyAmountIn(t, tight, 0, uint256.NewInt(40), nil)
	require.NoError(t, err)
	require.Equal(t, "1", res.TokenAmountIn.Amount.String())
}

// Protocol fee: collected, skipped on the current router (wallet or strategy short), refused on the launch
// router; the quote of CalcAmountIn carries the same fee and gas as CalcAmountOut's.
func TestCalcAmountInProtocolFee(t *testing.T) {
	t.Parallel()
	feeProgram := prog(iProtocol(10_000_000, testRecipient), iSwap(), iSalt(1)) // 1%
	target := e18(10)

	collected := newSim(t, poolOpts{program: feeProgram})
	res, err := verifyAmountIn(t, collected, 0, target, nil)
	require.NoError(t, err)
	require.NotZero(t, res.Fee.Amount.Sign(), "fee collected")
	require.Equal(t, int64(defaultGasFeeCollected), res.Gas)
	info := res.SwapInfo.(*SwapInfo)
	require.Equal(t, info.FeeIn, info.FeeCollected)
	require.False(t, info.FeeIn.IsZero())

	// Fee skipped on the current router: the strategy's in side is below the fee (1000 wei held vs a 1% fee of about 1990).
	thin := poolOpts{program: feeProgram, raw: [2]*uint256.Int{uint256.NewInt(1000), e18(2000)}}
	skipped := newSim(t, thin)
	thinTarget := e18(1990) // needs about 199000 in: a fee of about 1990 against 1000 held
	res, err = verifyAmountIn(t, skipped, 0, thinTarget, nil)
	require.NoError(t, err)
	require.Zero(t, res.Fee.Amount.Sign(), "fee skipped")
	require.Equal(t, int64(defaultGas), res.Gas)
	info = res.SwapInfo.(*SwapInfo)
	require.False(t, info.FeeIn.IsZero())
	require.True(t, info.FeeCollected.IsZero())
	// ... and refused on the launch router, in both directions.
	launch := newSim(t, poolOpts{program: feeProgram, app: testLaunch, raw: thin.raw})
	_, err = launch.CalcAmountIn(inParams(launch, 0, thinTarget, nil))
	require.ErrorIs(t, err, ErrFeeShortfall)
	_, _ = verifyAmountIn(t, launch, 0, thinTarget, nil)

	// Wallet short of the fee through the shared limit: the tokenIn key decides the fee, the tokenOut key the cap.
	s := newSim(t, poolOpts{program: feeProgram})
	inv := func(in, out *big.Int) pool.SwapLimit {
		return swaplimit.NewInventory(DexType, map[string]*big.Int{s.limitKeys[0]: in, s.limitKeys[1]: out})
	}
	res, err = verifyAmountIn(t, s, 0, target, inv(big.NewInt(0), e18(1000).ToBig()))
	require.NoError(t, err)
	require.Zero(t, res.Fee.Amount.Sign(), "the maker wallet holds no tokenIn for the fee")
	sl := newSim(t, poolOpts{program: feeProgram, app: testLaunch})
	_, err = sl.CalcAmountIn(inParams(sl, 0, target, inv(big.NewInt(0), e18(1000).ToBig())))
	require.ErrorIs(t, err, ErrFeeShortfall)
	res, err = verifyAmountIn(t, s, 0, target, inv(e18(1_000).ToBig(), e18(1000).ToBig()))
	require.NoError(t, err)
	require.NotZero(t, res.Fee.Amount.Sign())

	// The fee collection boundary: the largest amountIn whose fee the strategy still holds. A target whose
	// amountIn lands exactly on and just above it must give collected and then skipped.
	for _, rawIn := range []uint64{1, 2, 99, 100, 101, 5_000} {
		b := newSim(t, poolOpts{program: prog(iProtocol(10_000_000, testRecipient), iSwap()),
			raw: [2]*uint256.Int{uint256.NewInt(rawIn), e18(2000)}})
		for target := uint64(1); target <= 40; target++ {
			_, _ = verifyAmountIn(t, b, 0, uint256.NewInt(target*1_000_000_000_000_000), nil)
		}
	}
}

// The push credits amountIn on top of the strategy balance, which must stay within uint248.
func TestCalcAmountInUint248Bound(t *testing.T) {
	t.Parallel()
	rawOut := new(uint256.Int).Lsh(uint256.NewInt(1), 180)
	// Exact amountIn for rawOut/2 on an (rawIn, rawOut) xyc strategy is rawIn: headroom decides.
	for _, headroom := range []uint64{0, 1, 2, 1_000} {
		rawIn := new(uint256.Int).Sub(max248, uint256.NewInt(headroom))
		s := newSim(t, poolOpts{program: prog(iSwap()), raw: [2]*uint256.Int{rawIn, rawOut}, wallet: openWallet, allowance: openWallet})
		// target half of the out side needs about rawIn in: past the push bound unless the headroom covers it.
		target := new(uint256.Int).Rsh(rawOut, 1)
		res, err := verifyAmountIn(t, s, 0, target, nil)
		if err == nil {
			require.False(t, new(uint256.Int).Add(rawIn, uint256.MustFromBig(res.TokenAmountIn.Amount)).Gt(max248))
		} else {
			require.ErrorIs(t, err, ErrOverflow)
		}
	}
	// A small target near the bound: an amountIn that fits, and the largest target whose amountIn fits.
	// Headroom 2^100 above rawIn: a target of t needs about t*2^68 in, so 1 and 2^20 fit and 2^40 and up do not.
	rawIn := new(uint256.Int).Sub(max248, new(uint256.Int).Lsh(uint256.NewInt(1), 100))
	s := newSim(t, poolOpts{program: prog(iSwap()), raw: [2]*uint256.Int{rawIn, rawOut}, wallet: openWallet, allowance: openWallet})
	var ok, refused int
	for _, target := range []*uint256.Int{
		uint256.NewInt(1), uint256.NewInt(1 << 20), uint256.NewInt(1 << 30), new(uint256.Int).Lsh(uint256.NewInt(1), 40),
		new(uint256.Int).Lsh(uint256.NewInt(1), 100), new(uint256.Int).Lsh(uint256.NewInt(1), 150), new(uint256.Int).Lsh(uint256.NewInt(1), 179),
	} {
		if _, err := verifyAmountIn(t, s, 0, target, nil); err == nil {
			ok++
		} else {
			require.ErrorIs(t, err, ErrOverflow)
			refused++
		}
	}
	require.Positive(t, ok)
	require.Positive(t, refused)

	// With a fee collected the push is measured after the pull: the fee frees room.
	withFee := newSim(t, poolOpts{program: prog(iProtocol(10_000_000, testRecipient), iSwap()),
		raw: [2]*uint256.Int{new(uint256.Int).Sub(max248, uint256.NewInt(990_000)), rawOut}, wallet: openWallet, allowance: openWallet})
	for _, target := range []uint64{1, 1000, 1_000_000, 1_000_000_000} {
		_, _ = verifyAmountIn(t, withFee, 0, uint256.NewInt(target), nil)
	}
}

// Concentrate arithmetic that overflows is an error, not a wrap; the balances may be so large that the
// inversion itself overflows.
func TestCalcAmountInOverflow(t *testing.T) {
	t.Parallel()
	// Concentrate with an unreachable ratio on a huge balance: effectiveBalances fails.
	prog1 := prog(iConcentrate(uint256.NewInt(1), new(uint256.Int).Lsh(uint256.NewInt(1), 255)), iSwap())
	for _, raw := range [][2]*uint256.Int{
		{max248, max248}, {new(uint256.Int).Lsh(uint256.NewInt(1), 240), uint256.NewInt(1)}, {uint256.NewInt(1), max248},
	} {
		s := newSim(t, poolOpts{program: prog1, raw: raw, wallet: openWallet, allowance: openWallet})
		for in := range 2 {
			for _, target := range []uint64{1, 7, 1 << 40} {
				_, _ = verifyAmountIn(t, s, in, uint256.NewInt(target), nil)
			}
		}
	}
	// net = target*balIn/(balOut-target) beyond 2^256: balIn near 2^248, balOut tiny, a target one below
	// balOut gives net = (balOut-1)*balIn.
	huge := newSim(t, poolOpts{program: prog(iSwap()), raw: [2]*uint256.Int{max248, new(uint256.Int).Lsh(uint256.NewInt(1), 100)}, wallet: openWallet, allowance: openWallet})
	target := new(uint256.Int).Sub(new(uint256.Int).Lsh(uint256.NewInt(1), 100), uint256.NewInt(1))
	_, err := verifyAmountIn(t, huge, 0, target, nil)
	require.ErrorIs(t, err, ErrOverflow)
	// fee inversion beyond 2^256: net just below 2^256/1e9 times the denominator, with a tiny fee.
	feeBig := newSim(t, poolOpts{program: prog(iFlat(1), iSwap()), raw: [2]*uint256.Int{max248, new(uint256.Int).Lsh(uint256.NewInt(1), 230)}, wallet: openWallet, allowance: openWallet})
	for _, tgt := range []*uint256.Int{
		new(uint256.Int).Lsh(uint256.NewInt(1), 229), new(uint256.Int).Sub(new(uint256.Int).Lsh(uint256.NewInt(1), 230), uint256.NewInt(1)),
		new(uint256.Int).Lsh(uint256.NewInt(1), 100),
	} {
		_, _ = verifyAmountIn(t, feeBig, 0, tgt, nil)
	}
	feeBigP := newSim(t, poolOpts{program: prog(iProtocol(1, testRecipient), iSwap()), raw: [2]*uint256.Int{max248, new(uint256.Int).Lsh(uint256.NewInt(1), 230)}, wallet: openWallet, allowance: openWallet})
	for _, tgt := range []*uint256.Int{
		new(uint256.Int).Lsh(uint256.NewInt(1), 229), new(uint256.Int).Sub(new(uint256.Int).Lsh(uint256.NewInt(1), 230), uint256.NewInt(1)),
	} {
		_, _ = verifyAmountIn(t, feeBigP, 0, tgt, nil)
	}
}

// invertFees against an independent bisection of the forward fee map written with big.Int.
func TestInvertFees(t *testing.T) {
	t.Parallel()
	forward := func(program []Instruction, a *big.Int) *big.Int {
		x := new(big.Int).Set(a)
		for _, ins := range program {
			var fee *big.Int
			prod := new(big.Int).Mul(x, big.NewInt(int64(ins.Bps)))
			switch ins.Op {
			case opFlatFeeIn:
				fee = new(big.Int).Div(new(big.Int).Add(prod, big.NewInt(feeBpsDenominator-1)), big.NewInt(feeBpsDenominator))
			case opAquaProtocolFee:
				fee = new(big.Int).Div(prod, big.NewInt(feeBpsDenominator))
			default:
				continue
			}
			x.Sub(x, fee)
		}
		return x
	}
	bpsList := []uint32{0, 1, 2, 3, 333_333_333, 500_000_000, 500_000_001, 999_999_998, 999_999_999}
	rng := rand.New(rand.NewPCG(3, 3))
	for _, kinds := range []string{"F", "P", "FP", "PF", "FF", "PP", "FPF", "PFP", "PPF", "FFP"} {
		for range 120 {
			var program []Instruction
			for _, k := range kinds {
				ins := Instruction{Bps: bpsList[rng.IntN(len(bpsList))]}
				if k == 'F' {
					ins.Op = opFlatFeeIn
				} else {
					ins.Op = opAquaProtocolFee
				}
				program = append(program, ins)
			}
			program = append(program, Instruction{Op: opXYCSwap})
			net := randUpTo(rng, uint256.NewInt(1<<62))
			var x uint256.Int
			require.NoError(t, invertFees(&x, program, net))
			// x is the smallest amount that leaves net: forward(x) >= net > forward(x-1)
			require.GreaterOrEqual(t, forward(program, x.ToBig()).Cmp(net.ToBig()), 0, "%s net %s: x %s too small", kinds, net.Dec(), x.Dec())
			require.Negative(t, forward(program, new(big.Int).Sub(x.ToBig(), big.NewInt(1))).Cmp(net.ToBig()), "%s net %s: x %s not minimal", kinds, net.Dec(), x.Dec())
		}
	}
	// no wrapper: x is net
	var x uint256.Int
	require.NoError(t, invertFees(&x, []Instruction{{Op: opSalt}, {Op: opXYCSwap}}, uint256.NewInt(77)))
	require.Equal(t, "77", x.Dec())
	// 100% fee: unreachable, at any level
	for _, op := range []uint8{opFlatFeeIn, opAquaProtocolFee} {
		require.ErrorIs(t, invertFees(&x, []Instruction{{Op: op, Bps: feeBpsDenominator}, {Op: opXYCSwap}}, uint256.NewInt(1)), ErrNoAmountIn)
		require.ErrorIs(t, invertFees(&x, []Instruction{{Op: op, Bps: feeBpsDenominator - 1}, {Op: op, Bps: feeBpsDenominator}, {Op: opXYCSwap}}, uint256.NewInt(1)), ErrNoAmountIn)
	}
	// overflow of the result: net near 2^256 with a fee
	nearMax := new(uint256.Int).Sub(new(uint256.Int).Lsh(uint256.NewInt(1), 255), uint256.NewInt(1))
	for _, op := range []uint8{opFlatFeeIn, opAquaProtocolFee} {
		require.ErrorIs(t, invertFees(&x, []Instruction{{Op: op, Bps: 500_000_000}, {Op: opXYCSwap}}, nearMax.Lsh(nearMax, 1)), ErrOverflow)
		// x*1e9/(1e9-1) fits in 256 bits but the +1 of the protocol fee or the ceiling does not
	}
	maxU := new(uint256.Int).SetAllOne()
	require.ErrorIs(t, invertFees(&x, []Instruction{{Op: opFlatFeeIn, Bps: 1}, {Op: opXYCSwap}}, maxU), ErrOverflow)
	require.ErrorIs(t, invertFees(&x, []Instruction{{Op: opAquaProtocolFee, Bps: 1}, {Op: opXYCSwap}}, maxU), ErrOverflow)
	// protocol fee of 0: x - 0 = net exactly; adding one to a net of 2^256-1 overflows? (net-1)*1e9/1e9 + 1 = net
	require.NoError(t, invertFees(&x, []Instruction{{Op: opAquaProtocolFee, Bps: 0}, {Op: opXYCSwap}}, maxU))
	require.True(t, x.Eq(maxU))
}

func TestMulDivCeil(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(4, 4))
	var z uint256.Int
	require.False(t, mulDivCeil(&z, uint256.NewInt(1), uint256.NewInt(1), new(uint256.Int)), "division by zero")
	for range 3000 {
		x, y, d := randUpTo(rng, new(uint256.Int).SetAllOne()), randUpTo(rng, new(uint256.Int).SetAllOne()), randUpTo(rng, new(uint256.Int).SetAllOne())
		if rng.IntN(3) == 0 { // small operands: the product fits
			x, y = randUpTo(rng, uint256.NewInt(1<<40)), randUpTo(rng, uint256.NewInt(1<<40))
		}
		want := new(big.Int).Mul(x.ToBig(), y.ToBig())
		qr := new(big.Int)
		want.QuoRem(want, d.ToBig(), qr)
		if qr.Sign() != 0 {
			want.Add(want, big.NewInt(1))
		}
		ok := mulDivCeil(&z, x, y, d)
		require.Equal(t, want.BitLen() <= 256, ok)
		if ok {
			require.Equal(t, want.String(), z.Dec())
		}
		// aliasing the destination with an operand
		xc := x.Clone()
		ok2 := mulDivCeil(xc, xc, y, d)
		require.Equal(t, ok, ok2)
		if ok {
			require.Equal(t, want.String(), xc.Dec())
		}
	}
	// exact divisions are not rounded up, a result of exactly 2^256-1 is fine, one more overflows
	require.True(t, mulDivCeil(&z, uint256.NewInt(6), uint256.NewInt(7), uint256.NewInt(21)))
	require.Equal(t, "2", z.Dec())
	maxU := new(uint256.Int).SetAllOne()
	require.True(t, mulDivCeil(&z, maxU, uint256.NewInt(5), uint256.NewInt(5)))
	require.True(t, z.Eq(maxU))
	require.False(t, mulDivCeil(&z, maxU, uint256.NewInt(5), uint256.NewInt(4)))
	// the floor is 2^256-1 and a remainder lifts it over the top: (2^256-1)*(2^256-1) / (2^256-2) = 2^256-1 + 1/(2^256-2)
	require.False(t, mulDivCeil(&z, maxU, maxU, new(uint256.Int).Sub(maxU, uint256.NewInt(1))))
}

// effectiveBalances equals the balances runProgram sees at the swap.
func TestEffectiveBalances(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(5, 5))
	for _, seq := range wrapperSequences(3) {
		program := mustParse(t, buildProgram(seq, "both"), false)
		for _, p := range propProfiles[:6] {
			for _, lt := range []bool{false, true} {
				bIn, bOut := p.raw[0].Clone(), p.raw[1].Clone()
				effErr := effectiveBalances(program, bIn, bOut, lt)
				amount := randUpTo(rng, uint256.NewInt(1<<60))
				// the forward run on effective balances equals the full run
				full, fullErr := runQuote(program, p.raw[0], p.raw[1], amount, lt)
				if effErr != nil {
					require.Error(t, fullErr, "%s %s", seq, p.name)
					continue
				}
				var res quoteResult
				err := runProgramFrom(&res, program, bIn, bOut, amount, lt, true)
				if fullErr != nil {
					require.Error(t, err, "%s %s", seq, p.name)
					continue
				}
				require.NoError(t, err)
				require.True(t, res.AmountOut.Eq(&full.AmountOut), "%s %s", seq, p.name)
				require.True(t, res.AmountIn.Eq(&full.AmountIn))
				require.Equal(t, full.FeeIns, res.FeeIns)
			}
		}
	}
	// it stops at the swap
	program := []Instruction{{Op: opXYCSwap}, {Op: opXYCConcentrate, SqrtPMin: *uint256.NewInt(1), SqrtPMax: *uint256.NewInt(2)}}
	a, b := uint256.NewInt(5), uint256.NewInt(7)
	require.NoError(t, effectiveBalances(program, a, b, true))
	require.Equal(t, [2]string{"5", "7"}, [2]string{a.Dec(), b.Dec()})
}

// refineAmountIn walks to the smallest amountIn from a guess that is off by a few wei and gives up after
// maxRefineSteps; the guess is never trusted.
func TestRefineAmountIn(t *testing.T) {
	t.Parallel()
	program := mustParse(t, buildProgram("F", "none"), false)
	effIn, effOut := e18(1000), e18(2000)
	target := e18(100)
	for _, lt := range []bool{false, true} {
		var exact uint256.Int
		var net, den uint256.Int
		den.Sub(effOut, target)
		require.True(t, mulDivCeil(&net, target, effIn, &den))
		require.NoError(t, invertFees(&exact, program, &net))
		for off := -(maxRefineSteps + 2); off <= maxRefineSteps+2; off++ {
			guess := exact.Clone()
			if off >= 0 {
				guess.Add(guess, uint256.NewInt(uint64(off)))
			} else {
				guess.Sub(guess, uint256.NewInt(uint64(-off)))
			}
			err := refineAmountIn(new(quoteAlloc), program, guess, target, effIn, effOut, lt)
			if off > maxRefineSteps-1 || -off > maxRefineSteps-1 {
				require.ErrorIs(t, err, ErrNoConvergence, "offset %d", off)
				continue
			}
			require.NoError(t, err, "offset %d", off)
			require.True(t, guess.Eq(&exact), "offset %d: %s != %s", off, guess.Dec(), exact.Dec())
		}
	}
	// the smallest possible amountIn: 1 wei, where there is no amountIn-1 to try
	steepIn, steepOut := uint256.NewInt(1), e18(1000)
	one := uint256.NewInt(1)
	require.NoError(t, refineAmountIn(new(quoteAlloc), []Instruction{{Op: opXYCSwap}}, one, uint256.NewInt(5), steepIn, steepOut, true))
	require.Equal(t, "1", one.Dec())
	// a guess that is too small with a floor of 1 and an error from the forward run propagates
	over := new(uint256.Int).SetAllOne()
	require.ErrorIs(t, refineAmountIn(new(quoteAlloc), []Instruction{{Op: opXYCSwap}}, over, uint256.NewInt(1), effIn, effOut, true), ErrOverflow)
	// the forward run of amountIn-1 failing propagates: amountIn-1 fits in the product but amountIn does not
	// cannot happen (overflow is monotone), so a program the interpreter refuses stands for it
	require.ErrorIs(t, refineAmountIn(new(quoteAlloc), []Instruction{{Op: opXYCSwap}, {Op: opFlatFeeIn}}, uint256.NewInt(3), uint256.NewInt(1), effIn, effOut, true), ErrBadProgram)
	// walking up past 2^256-1 is an overflow, not a wrap: a 1-wei-in balance, a 1e-9 fee and a maximal amount
	feeProg := []Instruction{{Op: opFlatFeeIn, Bps: 1}, {Op: opXYCSwap}}
	maxIn := new(uint256.Int).SetAllOne()
	require.ErrorIs(t, refineAmountIn(new(quoteAlloc), feeProg, maxIn, uint256.NewInt(1), uint256.NewInt(1), uint256.NewInt(1), true), ErrOverflow)
}

// Results of CalcAmountIn feed UpdateBalance exactly like those of CalcAmountOut, so a route that quotes
// exact-out hop by hop keeps the pool and the shared limit consistent.
func TestCalcAmountInUpdateBalance(t *testing.T) {
	t.Parallel()
	for _, ls := range liveShapes {
		program := buildProgram(ls.wrappers, ls.salt)
		for _, launch := range []bool{false, true} {
			o := poolOpts{program: program, wallet: [2]*uint256.Int{e18(900), e18(900)}, allowance: [2]*uint256.Int{e18(5000), unlimited}}
			if launch {
				o.app = testLaunch
			}
			byIn, byOut := newSim(t, o), newSim(t, o)
			invA := swaplimit.NewInventory(DexType, map[string]*big.Int{byIn.limitKeys[0]: e18(900).ToBig(), byIn.limitKeys[1]: e18(900).ToBig()})
			invB := invA.Clone()
			for hop, step := range []struct {
				in     int
				target *uint256.Int
			}{{0, e18(10)}, {1, e18(7)}, {0, e18(3)}, {1, e18(40)}, {0, e18(2)}} {
				resIn, err := byIn.CalcAmountIn(inParams(byIn, step.in, step.target, invA))
				require.NoError(t, err, "%s hop %d", ls.name, hop)
				amountIn := uint256.MustFromBig(resIn.TokenAmountIn.Amount)
				// the same hop through exact-in on the twin pool
				resOut, err := quoteOut(byOut, step.in, amountIn, invB)
				require.NoError(t, err)
				require.Equal(t, resOut.SwapInfo, resIn.SwapInfo)
				byIn.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: *resIn.TokenAmountIn,
					TokenAmountOut: pool.TokenAmount{Token: byIn.Info.Tokens[1-step.in], Amount: resOut.TokenAmountOut.Amount},
					Fee:            *resIn.Fee, SwapInfo: resIn.SwapInfo, SwapLimit: invA})
				byOut.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: byOut.Info.Tokens[step.in], Amount: amountIn.ToBig()},
					TokenAmountOut: *resOut.TokenAmountOut, Fee: *resOut.Fee, SwapInfo: resOut.SwapInfo, SwapLimit: invB})
				require.Equal(t, snapOf(byOut), snapOf(byIn), "%s hop %d", ls.name, hop)
				for _, k := range byIn.limitKeys {
					require.Equal(t, invB.GetLimit(k), invA.GetLimit(k), "%s hop %d limit", ls.name, hop)
				}
			}
		}
	}
}

// A result owns its integers: nothing it holds is shared with the pool, with a clone or with another result.
func TestCalcAmountInResultOwnership(t *testing.T) {
	t.Parallel()
	s := newSim(t, poolOpts{program: buildProgram("PCF", "trail")})
	a, err := s.CalcAmountIn(inParams(s, 0, e18(10), nil))
	require.NoError(t, err)
	b, err := s.CalcAmountIn(inParams(s, 0, e18(10), nil))
	require.NoError(t, err)
	require.NotSame(t, a, b)
	require.NotSame(t, a.TokenAmountIn.Amount, b.TokenAmountIn.Amount)
	require.NotSame(t, a.Fee.Amount, b.Fee.Amount)
	require.NotSame(t, a.SwapInfo, b.SwapInfo)
	require.NotSame(t, a.RemainingTokenAmountOut.Amount, b.RemainingTokenAmountOut.Amount)
	a.TokenAmountIn.Amount.SetInt64(1)
	a.Fee.Amount.SetInt64(-1)
	a.RemainingTokenAmountOut.Amount.SetInt64(9)
	a.SwapInfo.(*SwapInfo).AmountIn.SetAllOne()
	c, err := s.CalcAmountIn(inParams(s, 0, e18(10), nil))
	require.NoError(t, err)
	require.Equal(t, b.TokenAmountIn.Amount, c.TokenAmountIn.Amount)
	require.Equal(t, b.SwapInfo, c.SwapInfo)
	clone := s.CloneState().(*PoolSimulator)
	d, err := clone.CalcAmountIn(inParams(clone, 0, e18(10), nil))
	require.NoError(t, err)
	require.Equal(t, b.TokenAmountIn.Amount, d.TokenAmountIn.Amount)
}

// The pool registers as an exact-out source.
func TestCalcAmountInRegistered(t *testing.T) {
	t.Parallel()
	_, ok := pool.CanCalcAmountIn[DexType]
	require.True(t, ok)
	var _ pool.IPoolExactOutSimulator = (*PoolSimulator)(nil)
}

// ---- fuzz --------------------------------------------------------------------------------------------

// shapeCatalog is every program the fuzzer picks from: each ordering of up to three wrappers with both
// salt variants.
var shapeCatalog = func() []struct{ seq, salt string } {
	var out []struct{ seq, salt string }
	for _, seq := range wrapperSequences(3) {
		for _, salt := range []string{"none", "lead", "trail", "both"} {
			out = append(out, struct{ seq, salt string }{seq, salt})
		}
	}
	return out
}()

func clamp248(b []byte) *uint256.Int {
	var v uint256.Int
	v.SetBytes(b)
	v.Rsh(&v, 8) // at most 248 bits
	return &v
}

// FuzzCalcAmountIn: any strategy balances, program shape, wallet, router and target. The checker compares
// with the independent oracle, so a wrong answer fails and not only a panic.
func FuzzCalcAmountIn(f *testing.F) {
	rng := rand.New(rand.NewPCG(6, 6))
	for i := range 40 {
		var rawIn, rawOut, target [32]byte
		for j := range rawIn {
			rawIn[j], rawOut[j], target[j] = byte(rng.Uint32()), byte(rng.Uint32()), byte(rng.Uint32())
		}
		f.Add(uint16(i*13), uint8(i), uint8(i/2), rawIn[:], rawOut[:], target[:], uint8(i))
	}
	f.Add(uint16(0), uint8(0), uint8(0), []byte{3}, []byte{9}, []byte{1}, uint8(0))
	f.Fuzz(func(t *testing.T, shape uint16, flags, shift uint8, rawInB, rawOutB, targetB []byte, walletMode uint8) {
		if len(rawInB) > 32 || len(rawOutB) > 32 || len(targetB) > 32 {
			t.Skip()
		}
		sc := shapeCatalog[int(shape)%len(shapeCatalog)]
		rawIn, rawOut := clamp248(rawInB), clamp248(rawOutB)
		// shift the balances down so that small and large strategies are both reached
		rawIn.Rsh(rawIn, uint(shift%200))
		rawOut.Rsh(rawOut, uint(shift/2%200))
		s := propSim(t, buildProgram(sc.seq, sc.salt), flags&1 == 1, [2]*uint256.Int{rawIn, rawOut}, walletModes[int(walletMode)%len(walletModes)])
		in := int(flags >> 1 & 1)
		var target uint256.Int
		target.SetBytes(targetB)
		// the target is a share of the out side (so that it lands inside the strategy) or the raw value
		out := &s.State.Raw[1-in]
		if flags&4 == 0 && !out.IsZero() {
			target.Mod(&target, out)
			target.AddUint64(&target, 1)
		}
		if target.IsZero() {
			target.SetOne()
		}
		_, _ = verifyAmountIn(t, s, in, &target, nil)
	})
}

// A short deterministic run of the fuzz body over many random inputs, so plain `go test` covers it.
func TestCalcAmountInRandomized(t *testing.T) {
	t.Parallel()
	for w := range 4 {
		t.Run(fmt.Sprint(w), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewPCG(7, uint64(w)))
			var ok, refused int
			for range 1500 {
				sc := shapeCatalog[rng.IntN(len(shapeCatalog))]
				rawIn, rawOut := randUpTo(rng, max248), randUpTo(rng, max248)
				if rng.IntN(4) == 0 {
					rawIn.Rsh(rawIn, uint(rng.IntN(200)))
				}
				if rng.IntN(4) == 0 {
					rawOut.Rsh(rawOut, uint(rng.IntN(200)))
				}
				s := propSim(t, buildProgram(sc.seq, sc.salt), rng.IntN(2) == 0, [2]*uint256.Int{rawIn, rawOut}, walletModes[rng.IntN(3)])
				in := rng.IntN(2)
				if _, err := verifyAmountIn(t, s, in, randUpTo(rng, &s.State.Raw[1-in]), nil); err == nil {
					ok++
				} else {
					refused++
				}
			}
			require.Greater(t, ok, 300)
			require.Greater(t, refused, 30)
		})
	}
}

// the checker's own failure modes: it must notice a wrong answer.
func TestRefMinAmountInOracle(t *testing.T) {
	t.Parallel()
	program := mustParse(t, buildProgram("F", "none"), false)
	a, viaErr, found := refMinAmountIn(program, e18(1000).ToBig(), e18(2000).ToBig(), e18(100).ToBig(), true)
	require.True(t, found)
	require.False(t, viaErr)
	_, out, _, ok := refRun(program, e18(1000).ToBig(), e18(2000).ToBig(), a, true)
	require.True(t, ok)
	require.GreaterOrEqual(t, out.Cmp(e18(100).ToBig()), 0)
	_, outLess, _, _ := refRun(program, e18(1000).ToBig(), e18(2000).ToBig(), new(big.Int).Sub(a, big.NewInt(1)), true)
	require.Negative(t, outLess.Cmp(e18(100).ToBig()))
	// a target above what the strategy can pay is never reached (out < balanceOut), so the oracle finds none
	_, viaErr, found = refMinAmountIn(program, e18(1000).ToBig(), e18(2000).ToBig(), e18(2001).ToBig(), true)
	require.True(t, found)
	require.True(t, viaErr, "only the model's overflow ends the search")
}
