package hyeth

import (
	"errors"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func bigS(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic(s)
	}
	return v
}

// refState is getPoolJSON's extra, the chain's state at block 22317092.
type refState struct {
	compSup, compAss, compHyb, hySup, dpru, maxDeposit, maxRedeem, feeI, feeR *big.Int
	epru                                                                      []*big.Int
}

func baseRefState(epru ...string) refState {
	st := refState{
		compSup: bigS("4946361947932843870115"), compAss: bigS("5005345678839792956730"),
		compHyb: bigS("1015907674038080762600"), hySup: bigS("809233550815085194542"),
		dpru: bigS("1255394901774434537"), maxDeposit: bigS("1000000024671486719480691603261"),
		maxRedeem: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)),
		feeI:      big.NewInt(0), feeR: big.NewInt(0),
	}
	for _, e := range epru {
		st.epru = append(st.epru, bigS(e))
	}
	return st
}

type refQuote struct {
	out, ts, ta, hyBal, hySup, maxDep *big.Int
	gas                               int64
	err                               error
}

var wad = big.NewInt(1e18)

func mulDiv(a, b, c *big.Int) *big.Int { return new(big.Int).Div(new(big.Int).Mul(a, b), c) }

// refCalcAmountOut is the contracts' math in math/big, apart from the simulator:
//   - DebtIssuanceModule (0x04b5…f0f3) _calculateTotalFees: fee = q * feeRate / 1e18, issue mints q + fee, redeem
//     burns q and pays for q - fee; _getTotalIssuanceUnits = defaultPositionRealUnit + the positive external units;
//     redemption units = totalQuantity * units / 1e18 (floor), issuance units round up;
//   - the component vault (ERC-4626): previewRedeem(s) = s * (totalAssets + 1) / (totalSupply + 1), and
//     ExecutorHelper9's pre-issue estimate of the deposit, shares = assets * totalSupply / totalAssets, with the
//     issued quantity = shares * 1e18 / (hyETH's share balance * 1e18 / hyETH supply + external units) * 1e18 /
//     (1e18 + issueFee); maxDeposit and maxRedeem bound the vault leg;
//   - FlashMintHyETHV3 pays WETH for the redeemed assets 1:1. Any token pair that is not hyETH in quotes as an issue.
//
// The gas (redeem 700000, issue 1000000) is the package default.
func refCalcAmountOut(st refState, in, out string, amount *big.Int) refQuote {
	known := func(tok string) bool { return tok == testHyeth || tok == testWeth }
	switch {
	case !known(in) || !known(out):
		return refQuote{err: ErrInvalidToken}
	case amount.Cmp(new(big.Int).Lsh(big.NewInt(1), 256)) >= 0:
		return refQuote{err: ErrInvalidAmountIn}
	case amount.Sign() <= 0:
		return refQuote{err: ErrInsufficientInputAmount}
	}
	ext := new(big.Int)
	for _, e := range st.epru {
		if e.Sign() > 0 {
			ext.Add(ext, e)
		}
	}
	units := new(big.Int).Add(st.dpru, ext)
	if in == testHyeth {
		burnt := new(big.Int).Sub(amount, mulDiv(amount, st.feeR, wad))
		position := mulDiv(burnt, units, wad)
		if position.Cmp(st.maxRedeem) > 0 {
			return refQuote{err: ErrERC4626RedeemMoreThanMax}
		}
		// redeem burns amount from the caller (at most the supply) and redeems position out of hyETH's shares
		if amount.Cmp(st.hySup) > 0 || position.Cmp(st.compHyb) > 0 {
			return refQuote{err: ErrRedeemMoreThanSupply}
		}
		assets := mulDiv(position, new(big.Int).Add(st.compAss, big.NewInt(1)), new(big.Int).Add(st.compSup, big.NewInt(1)))
		maxDep := new(big.Int).Add(st.maxDeposit, assets)
		if maxDep.BitLen() > 256 {
			maxDep = st.maxRedeem // saturates at 2^256-1
		}
		return refQuote{
			out: assets, gas: 700000, maxDep: maxDep,
			ts: new(big.Int).Sub(st.compSup, position), ta: new(big.Int).Sub(st.compAss, assets),
			hyBal: new(big.Int).Sub(st.compHyb, position), hySup: new(big.Int).Sub(st.hySup, burnt),
		}
	}
	if amount.Cmp(st.maxDeposit) > 0 {
		return refQuote{err: ErrERC4626DepositMoreThanMax}
	}
	shares := mulDiv(amount, st.compSup, st.compAss)
	equityUnit := new(big.Int).Add(mulDiv(st.compHyb, wad, st.hySup), ext)
	qty := mulDiv(mulDiv(shares, wad, equityUnit), wad, new(big.Int).Add(wad, st.feeI))
	minted := new(big.Int).Add(qty, mulDiv(qty, st.feeI, wad))
	used := new(big.Int).Mul(minted, units)
	used.Add(used, new(big.Int).Sub(wad, big.NewInt(1))).Div(used, wad) // round up
	return refQuote{
		out: qty, gas: 1000000, maxDep: new(big.Int).Sub(st.maxDeposit, amount),
		ts: new(big.Int).Add(st.compSup, shares), ta: new(big.Int).Add(st.compAss, amount),
		hyBal: new(big.Int).Add(st.compHyb, used), hySup: new(big.Int).Add(st.hySup, minted),
	}
}

// poolFrom builds getPoolJSON with the extra's fields replaced (key -> raw JSON value).
func poolFrom(t *testing.T, repl map[string]string) *PoolSimulator {
	t.Helper()
	var poolE entity.Pool
	require.NoError(t, json.Unmarshal([]byte(getPoolJSON), &poolE))
	var extra map[string]any
	require.NoError(t, json.Unmarshal([]byte(poolE.Extra), &extra))
	for k, v := range repl {
		var val any
		require.NoError(t, json.Unmarshal([]byte(v), &val))
		extra[k] = val
	}
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	poolE.Extra = string(b)
	p, err := NewPoolSimulator(poolE)
	require.NoError(t, err)
	return p
}

type refFixture struct {
	name string
	repl map[string]string
	st   refState
}

func refFixtures() []refFixture {
	fees := baseRefState("1000000000000000")
	fees.feeI, fees.feeR = big.NewInt(1e15), big.NewInt(3e15)
	capped := baseRefState()
	capped.maxDeposit, capped.maxRedeem = big.NewInt(1e18), big.NewInt(2e18)
	return []refFixture{
		{"epru-none", nil, baseRefState()},
		{"epru-one", map[string]string{"epru": `["1000000000000000"]`}, baseRefState("1000000000000000")},
		{"epru-mix", map[string]string{"epru": `["1000000000000000","0","2500000000000000"]`},
			baseRefState("1000000000000000", "0", "2500000000000000")},
		{"fees", map[string]string{"epru": `["1000000000000000"]`, "feeI": `"1000000000000000"`,
			"feeR": `"3000000000000000"`}, fees},
		{"capped", map[string]string{"maxDeposit": `"1000000000000000000"`, "maxRedeem": `"2000000000000000000"`}, capped},
	}
}

func refAmounts() []*big.Int {
	amounts := []*big.Int{big.NewInt(0), big.NewInt(1), big.NewInt(999)}
	for e := 6; e <= 30; e += 3 {
		v := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(e)), nil)
		amounts = append(amounts, v, new(big.Int).Add(v, big.NewInt(int64(e*7))))
	}
	return append(amounts, new(big.Int).Lsh(big.NewInt(1), 256))
}

// TestCalcAmountOut_MatchesReferenceModel quotes both directions (and WETH->WETH, which quotes as an issue) on the
// chain fixture with and without external units, with fees and with vault caps, against refCalcAmountOut: the
// output, the gas and every post-swap field of SwapInfo.
func TestCalcAmountOut_MatchesReferenceModel(t *testing.T) {
	t.Parallel()
	pairs := [][2]string{{testHyeth, testWeth}, {testWeth, testHyeth}, {testWeth, testWeth}, {testHyeth, "0x0000000000000000000000000000000000000001"}}
	for _, fx := range refFixtures() {
		for _, d := range pairs {
			for _, amt := range refAmounts() {
				want := refCalcAmountOut(fx.st, d[0], d[1], amt)
				res, err := quote(poolFrom(t, fx.repl), d[0], d[1], amt)
				msg := []any{"%s %s>%s %s", fx.name, d[0][:6], d[1][:6], amt}
				if want.err != nil {
					assert.Truef(t, errors.Is(err, want.err), "%v: err %v, want %v", msg, err, want.err)
					continue
				}
				if !assert.NoError(t, err, msg...) {
					continue
				}
				si := res.SwapInfo.(SwapInfo)
				assert.Equal(t, want.out.String(), res.TokenAmountOut.Amount.String(), msg...)
				assert.Equal(t, want.gas, res.Gas, msg...)
				fee := fx.st.feeI
				if d[0] == testHyeth {
					fee = fx.st.feeR
				}
				for _, c := range [][2]any{{fee, si.Fee}, {want.ts, si.TotalSupply}, {want.ta, si.TotalAssets},
					{want.hyBal, si.HyethBalance}, {want.hySup, si.HyethSupply}, {want.maxDep, si.MaxDeposit}} {
					assert.Equal(t, c[0].(*big.Int).String(), c[1].(*uint256.Int).Dec(), msg...)
				}
			}
		}
	}
}

// The model against the module's eth_call at block 22317092: getRequiredComponentRedemptionUnits(hyETH, 1e18) =
// 1255394901774434537 (DebtIssuanceModule, as in TestRedeemAndDepositMatchChainAt22317092), and the redeem output
// is the vault leg on it: 1255394901774434537 * 5005345678839792956731 / 4946361947932843870116.
func TestReferenceModel_RedeemUnitsMatchChain(t *testing.T) {
	t.Parallel()
	q := refCalcAmountOut(baseRefState(), testHyeth, testWeth, big.NewInt(1e18))
	require.NoError(t, q.err)
	assert.Equal(t, "1255394901774434537", new(big.Int).Sub(bigS("4946361947932843870115"), q.ts).String())
	assert.Equal(t, mulDiv(bigS("1255394901774434537"), bigS("5005345678839792956731"),
		bigS("4946361947932843870116")).String(), q.out.String())
}

// TestInvariant_HyethQuoteReadonlyAndClone: a quote leaves the simulator unchanged, and UpdateBalance on a clone
// leaves the original unchanged.
func TestInvariant_HyethQuoteReadonlyAndClone(t *testing.T) {
	t.Parallel()
	p := newPoolWithEpru(t, "1000000000000000")
	before := p.CloneState()
	for _, d := range [][2]string{{testHyeth, testWeth}, {testWeth, testHyeth}} {
		for _, amt := range refAmounts() {
			_, _ = quote(p, d[0], d[1], amt)
		}
	}
	assert.Equal(t, before, p)

	clone := p.CloneState()
	res, err := quote(clone.(*PoolSimulator), testWeth, testHyeth, big.NewInt(1e18))
	require.NoError(t, err)
	clone.UpdateBalance(updateParams(testWeth, big.NewInt(1e18), res))
	assert.Equal(t, before, p)
}

// TestInvariant_HyethSplitMonotonicBounded: in both directions, splitting the input never gives more than the
// whole, the output is non-decreasing in the input, and a redeem pays at most the vault's assets.
func TestInvariant_HyethSplitMonotonicBounded(t *testing.T) {
	t.Parallel()
	p := newPoolWithEpru(t, "1000000000000000")
	amounts := []int64{1, 999, 1e9, 3e15, 1e18}
	for _, d := range [][2]string{{testHyeth, testWeth}, {testWeth, testHyeth}} {
		var prev *big.Int
		for _, a := range amounts {
			ra, err := quote(p, d[0], d[1], big.NewInt(a))
			require.NoError(t, err)
			for _, b := range amounts {
				rb, errB := quote(p, d[0], d[1], big.NewInt(b))
				rw, errW := quote(p, d[0], d[1], big.NewInt(a+b))
				require.NoError(t, errors.Join(errB, errW))
				sum := new(big.Int).Add(ra.TokenAmountOut.Amount, rb.TokenAmountOut.Amount)
				assert.LessOrEqual(t, sum.Cmp(rw.TokenAmountOut.Amount), 0, "split above whole %v %d+%d", d, a, b)
			}
			if prev != nil {
				assert.GreaterOrEqual(t, ra.TokenAmountOut.Amount.Cmp(prev), 0, "not monotonic %v", d)
			}
			prev = ra.TokenAmountOut.Amount
			if d[0] == testHyeth {
				assert.LessOrEqual(t, ra.TokenAmountOut.Amount.Cmp(bigS("5005345678839792956730")), 0)
			}
		}
	}
}

// FuzzCalcAmountOut checks the simulator against the reference model on random amounts, both directions, and
// external units.
func FuzzCalcAmountOut(f *testing.F) {
	f.Add(uint64(1e18), uint64(0), true, uint64(0))
	f.Add(uint64(1), uint64(0), false, uint64(1e15))
	f.Add(uint64(999), uint64(12345), true, uint64(25e14))
	f.Add(uint64(0), uint64(1), false, uint64(0))
	f.Fuzz(func(t *testing.T, lo, hi uint64, redeem bool, epru uint64) {
		amt := new(big.Int).Or(new(big.Int).Lsh(new(big.Int).SetUint64(hi), 64), new(big.Int).SetUint64(lo))
		in, out := testWeth, testHyeth
		if redeem {
			in, out = testHyeth, testWeth
		}
		e := new(big.Int).SetUint64(epru).String()
		want := refCalcAmountOut(baseRefState(e), in, out, amt)
		res, err := quote(newPoolWithEpru(t, e), in, out, amt)
		if want.err != nil {
			assert.Truef(t, errors.Is(err, want.err), "err %v, want %v", err, want.err)
			return
		}
		require.NoError(t, err)
		assert.Equal(t, want.out.String(), res.TokenAmountOut.Amount.String())
	})
}

func updateParams(in string, amt *big.Int, res *pool.CalcAmountOutResult) pool.UpdateBalanceParams {
	return pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: in, Amount: amt},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	}
}

// TestCalcAmountOut_RedeemBounds pins each redeem bound on its own. With the external units of epru-mix a hyETH
// takes 1.2589e18 shares, so 808e18 hyETH (below the 809.23e18 supply) needs 1017.2e18 shares, above hyETH's
// 1015.9e18; with no external units 809e18 fits both and 810e18 is above the supply.
func TestCalcAmountOut_RedeemBounds(t *testing.T) {
	t.Parallel()
	e18 := func(n int64) *big.Int { return new(big.Int).Mul(big.NewInt(n), big.NewInt(1e18)) }
	for _, tc := range []struct {
		epru []string
		amt  *big.Int
		err  error
	}{
		{nil, e18(809), nil},
		{nil, e18(810), ErrRedeemMoreThanSupply},
		{[]string{"1000000000000000", "0", "2500000000000000"}, e18(806), nil},
		{[]string{"1000000000000000", "0", "2500000000000000"}, e18(808), ErrRedeemMoreThanSupply},
	} {
		_, err := quote(newPoolWithEpru(t, tc.epru...), testHyeth, testWeth, tc.amt)
		if tc.err == nil {
			assert.NoError(t, err, "%v %s", tc.epru, tc.amt)
		} else {
			assert.ErrorIs(t, err, tc.err, "%v %s", tc.epru, tc.amt)
		}
		want := refCalcAmountOut(baseRefState(tc.epru...), testHyeth, testWeth, tc.amt)
		assert.Equal(t, tc.err, want.err, "reference model %v %s", tc.epru, tc.amt)
	}
}
