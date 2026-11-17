package oneinchaqua

import (
	"errors"
	"math/big"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// ref* is an independent big.Int model of the program semantics used as the oracle for the uint256
// code: every intermediate must fit in 256 bits, as the router's checked arithmetic requires.

var (
	refOne = big.NewInt(1e18)
	refBps = big.NewInt(feeBpsDenominator)
	ref256 = new(big.Int).Lsh(big.NewInt(1), 256)
)

func fits(x *big.Int) bool { return x.Sign() >= 0 && x.Cmp(ref256) < 0 }

func refMulDiv(x, y, d *big.Int) (*big.Int, bool) {
	r := new(big.Int).Mul(x, y)
	r.Quo(r, d)
	return r, fits(r)
}

func refCeilDiv(x, d *big.Int) *big.Int {
	if x.Sign() == 0 {
		return new(big.Int)
	}
	r := new(big.Int).Sub(x, big.NewInt(1))
	r.Quo(r, d)
	return r.Add(r, big.NewInt(1))
}

func refComputeL(bLt, bGt, pmin, pmax *big.Int) (*big.Int, bool) {
	ratio, ok := refMulDiv(pmin, refOne, pmax)
	if !ok {
		return nil, false
	}
	alpha := new(big.Int).Sub(refOne, ratio)
	b1, ok1 := refMulDiv(bLt, pmin, refOne)
	b2, ok2 := refMulDiv(bGt, refOne, pmax)
	if !ok1 || !ok2 {
		return nil, false
	}
	beta := new(big.Int).Add(b1, b2)
	if !fits(beta) {
		return nil, false
	}
	four, ok := refMulDiv(new(big.Int).Lsh(alpha, 2), bLt, refOne)
	if !ok {
		return nil, false
	}
	four.Mul(four, bGt)
	if !fits(four) {
		return nil, false
	}
	disc := new(big.Int).Mul(beta, beta)
	if !fits(disc) {
		return nil, false
	}
	disc.Add(disc, four)
	if !fits(disc) {
		return nil, false
	}
	num := new(big.Int).Add(beta, new(big.Int).Sqrt(disc))
	if !fits(num) {
		return nil, false
	}
	return refMulDiv(num, refOne, new(big.Int).Lsh(alpha, 1))
}

type refFrame struct {
	taker    *big.Int
	protocol bool
	fee      *big.Int
}

func refRun(program []Instruction, balIn, balOut, amtIn *big.Int, inLt bool) (in, out *big.Int, fees []*big.Int, ok bool) {
	balIn, balOut, amtIn = new(big.Int).Set(balIn), new(big.Int).Set(balOut), new(big.Int).Set(amtIn)
	out = new(big.Int)
	var stack []refFrame
	for _, p := range program {
		switch p.Op {
		case opSalt:
		case opXYCConcentrate:
			pmin, pmax := p.SqrtPMin.ToBig(), p.SqrtPMax.ToBig()
			bLt, bGt := balOut, balIn
			if inLt {
				bLt, bGt = balIn, balOut
			}
			l, good := refComputeL(bLt, bGt, pmin, pmax)
			if !good {
				return nil, nil, nil, false
			}
			var addIn, addOut *big.Int
			if inLt {
				prod := new(big.Int).Mul(l, refOne)
				if !fits(prod) {
					return nil, nil, nil, false
				}
				addIn = refCeilDiv(prod, pmax)
				addOut, good = refMulDiv(l, pmin, refOne)
			} else {
				prod := new(big.Int).Mul(l, pmin)
				if !fits(prod) {
					return nil, nil, nil, false
				}
				addIn = refCeilDiv(prod, refOne)
				addOut, good = refMulDiv(l, refOne, pmax)
			}
			if !good {
				return nil, nil, nil, false
			}
			balIn.Add(balIn, addIn)
			balOut.Add(balOut, addOut)
			if !fits(balIn) || !fits(balOut) {
				return nil, nil, nil, false
			}
		case opFlatFeeIn:
			prod := new(big.Int).Mul(amtIn, big.NewInt(int64(p.Bps)))
			if !fits(prod) {
				return nil, nil, nil, false
			}
			stack = append(stack, refFrame{taker: new(big.Int).Set(amtIn)})
			amtIn.Sub(amtIn, refCeilDiv(prod, refBps))
		case opAquaProtocolFee:
			prod := new(big.Int).Mul(amtIn, big.NewInt(int64(p.Bps)))
			if !fits(prod) {
				return nil, nil, nil, false
			}
			fee := prod.Quo(prod, refBps)
			stack = append(stack, refFrame{taker: new(big.Int).Set(amtIn), protocol: true, fee: fee})
			amtIn.Sub(amtIn, fee)
		case opXYCSwap:
			if balIn.Sign() == 0 || balOut.Sign() == 0 {
				return nil, nil, nil, false
			}
			num := new(big.Int).Mul(amtIn, balOut)
			den := new(big.Int).Add(balIn, amtIn)
			if !fits(num) || !fits(den) {
				return nil, nil, nil, false
			}
			out = num.Quo(num, den)
		}
	}
	for _, s := range slices.Backward(stack) {
		amtIn = s.taker
		if s.protocol {
			fees = append(fees, s.fee)
		}
	}
	return amtIn, out, fees, true
}

func requireSameAsRef(t testing.TB, program []Instruction, balIn, balOut, amtIn *uint256.Int, inLt bool) {
	t.Helper()
	wantIn, wantOut, wantFees, ok := refRun(program, balIn.ToBig(), balOut.ToBig(), amtIn.ToBig(), inLt)
	got, err := runQuote(program, balIn, balOut, amtIn, inLt)
	if !ok {
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrOverflow) || errors.Is(err, ErrZeroBalance), "got %v", err)
		return
	}
	require.NoError(t, err)
	require.Equal(t, wantIn.String(), got.AmountIn.Dec())
	require.Equal(t, wantOut.String(), got.AmountOut.Dec())
	require.Len(t, got.FeeIns, len(wantFees))
	for i := range wantFees {
		require.Equal(t, wantFees[i].String(), got.FeeIns[i].Dec())
	}
}

// Ported from the SwapVM XYCSwap tests (basic swap, no fee and 0.3% fee on 1000e18 / 1000e18).
func TestRunProgramXYCSwapPorted(t *testing.T) {
	t.Parallel()
	poolA, poolB, amountIn := e18(1000), e18(1000), e18(10)

	noFee := mustParse(t, prog(iSwap()), false)
	res, err := runQuote(noFee, poolA, poolB, amountIn, true)
	require.NoError(t, err)
	want := new(uint256.Int).Div(new(uint256.Int).Mul(amountIn, poolB), new(uint256.Int).Add(poolA, amountIn))
	require.Equal(t, want, &res.AmountOut, "output matches x*y=k")
	require.Equal(t, amountIn, &res.AmountIn)
	require.Empty(t, res.FeeIns)

	withFee := mustParse(t, prog(iFlat(3_000_000), iSwap()), false) // 0.3%
	res, err = runQuote(withFee, poolA, poolB, amountIn, true)
	require.NoError(t, err)
	afterFee := new(uint256.Int).Div(new(uint256.Int).Mul(amountIn, uint256.NewInt(1e9-3_000_000)), uBps)
	want = new(uint256.Int).Div(new(uint256.Int).Mul(afterFee, poolB), new(uint256.Int).Add(poolA, afterFee))
	require.Equal(t, want, &res.AmountOut, "output accounts for the fee")
	require.Equal(t, amountIn, &res.AmountIn, "taker amountIn is restored after the fee wrapper")

	// A second swap of the same size on the updated balances gets a worse rate.
	poolA2, poolB2 := new(uint256.Int).Add(poolA, amountIn), new(uint256.Int).Sub(poolB, &res.AmountOut)
	res2, err := runQuote(noFee, poolA2, poolB2, amountIn, true)
	require.NoError(t, err)
	require.True(t, res2.AmountOut.Lt(&res.AmountOut))
}

// Ported from the SwapVM Aqua flat-fee test: 10% fee in, output from the post-fee amountIn.
func TestRunProgramFlatFeePorted(t *testing.T) {
	t.Parallel()
	balA, balB, amountIn := e18(1000), e18(2000), e18(100)
	res, err := runQuote(mustParse(t, prog(iFlat(100_000_000), iSwap()), false), balA, balB, amountIn, true)
	require.NoError(t, err)
	fee := new(uint256.Int).Div(new(uint256.Int).Mul(amountIn, uint256.NewInt(100_000_000)), uBps)
	net := new(uint256.Int).Sub(amountIn, fee)
	want := new(uint256.Int).Div(new(uint256.Int).Mul(net, balB), new(uint256.Int).Add(balA, net))
	require.Equal(t, want, &res.AmountOut)
	require.Equal(t, amountIn, &res.AmountIn)

	// A 100% fee leaves nothing to swap: the quote has zero output.
	res, err = runQuote(mustParse(t, prog(iFlat(feeBpsDenominator), iSwap()), false), balA, balB, amountIn, true)
	require.NoError(t, err)
	require.True(t, res.AmountOut.IsZero())
}

// The flat fee rounds up and the protocol fee rounds down, both in taker favor of the opposite rule.
func TestRunProgramFeeRounding(t *testing.T) {
	t.Parallel()
	bal := e18(1000)
	one := uint256.NewInt(1)
	// 1 wei at 0.0000001%: ceil(1*1/1e9) = 1, so the flat fee swallows the whole input.
	res, err := runQuote(mustParse(t, prog(iFlat(1), iSwap()), false), bal, bal, one, true)
	require.NoError(t, err)
	require.True(t, res.AmountOut.IsZero())
	// The protocol fee floors to zero on the same input and the swap sees 1 wei.
	res, err = runQuote(mustParse(t, prog(iProtocol(1, testRecipient), iSwap()), false), bal, bal, one, true)
	require.NoError(t, err)
	require.Len(t, res.FeeIns, 1)
	require.True(t, res.FeeIns[0].IsZero())
	require.Equal(t, uint64(0), res.AmountOut.Uint64()) // 1*1000e18/(1000e18+1) = 0
	// Larger input: protocol fee is floor(amount*bps/1e9).
	res, err = runQuote(mustParse(t, prog(iProtocol(10_000_000, testRecipient), iSwap()), false), bal, bal, uint256.NewInt(999), true)
	require.NoError(t, err)
	require.Equal(t, uint64(9), res.FeeIns[0].Uint64())
}

// Two protocol fees: the wrapper that is innermost in the program is pulled first, and each fee is taken
// from the amount left by the outer ones.
func TestRunProgramFeeOrder(t *testing.T) {
	t.Parallel()
	program := mustParse(t, prog(iSalt(1), iProtocol(100_000_000, testRecipient), iProtocol(200_000_000, testRecipient), iFlat(50_000_000), iSwap()), false)
	amountIn := uint256.NewInt(1_000_000)
	res, err := runQuote(program, e18(10), e18(10), amountIn, true)
	require.NoError(t, err)
	require.Len(t, res.FeeIns, 2)
	require.Equal(t, uint64(180_000), res.FeeIns[0].Uint64(), "inner fee: 20% of 900000")
	require.Equal(t, uint64(100_000), res.FeeIns[1].Uint64(), "outer fee: 10% of 1000000")
	require.Equal(t, amountIn, &res.AmountIn)
	requireSameAsRef(t, program, e18(10), e18(10), amountIn, true)
}

// The concentrate step rounds the in-side up (taker-unfavorable); regression from the SwapVM tests with
// balances 1 / 1000 and a price range of 1e18 -/+ 1.
func TestConcentrateRoundsInSideUp(t *testing.T) {
	t.Parallel()
	pmin, pmax := uint256.NewInt(1e18-1), uint256.NewInt(1e18+1)
	balLt, balGt := uint256.NewInt(1), uint256.NewInt(1000)
	l, err := liquidityOf(balLt, balGt, pmin, pmax)
	require.NoError(t, err)
	var floor, tmp uint256.Int
	require.True(t, mulDiv(&floor, l, uOne, pmax))
	var ceil uint256.Int
	tmp.Mul(l, uOne)
	ceilDiv(&ceil, &tmp, pmax)
	require.True(t, ceil.Gt(&floor), "the case is one where floor and ceil differ")

	in, out := *balLt, *balGt
	require.NoError(t, concentrate(&in, &out, pmin, pmax, true))
	require.Equal(t, new(uint256.Int).Add(balLt, &ceil), &in)
	var outAdd uint256.Int
	require.True(t, mulDiv(&outAdd, l, pmin, uOne))
	require.Equal(t, new(uint256.Int).Add(balGt, &outAdd), &out)
}

// Ported from the SwapVM concentrate zero-balance tests: with the spot at a range bound one real balance
// is zero, and the other direction is the valid one.
func TestConcentrateZeroBalanceBoundaries(t *testing.T) {
	t.Parallel()
	pminSqrt := new(uint256.Int).Sqrt(new(uint256.Int).Mul(u("10000000000000000"), uOne)) // sqrt(0.01e18*1e18) = 0.1e18
	pmaxSqrt := new(uint256.Int).Sqrt(new(uint256.Int).Mul(e18(25), uOne))                // 5e18
	targetL := e18(100_000)
	program := mustParse(t, prog(iConcentrate(pminSqrt, pmaxSqrt), iSwap()), false)
	swapAmount := e18(10)

	// Spot above the upper bound: bLt = 0, bGt = L * (spot - pmin) / 1e18; selling Lt for Gt works.
	spot := new(uint256.Int).AddUint64(pmaxSqrt, 100)
	var bGt uint256.Int
	require.True(t, mulDiv(&bGt, targetL, new(uint256.Int).Sub(spot, pminSqrt), uOne))
	res, err := runQuote(program, new(uint256.Int), &bGt, swapAmount, true)
	require.NoError(t, err)
	require.False(t, res.AmountOut.IsZero())
	require.Equal(t, swapAmount, &res.AmountIn)
	// The empty side cannot be bought: no Gt balance in, zero Lt out for Gt in.
	res, err = runQuote(program, &bGt, new(uint256.Int), swapAmount, false)
	require.NoError(t, err)
	require.False(t, res.AmountOut.IsZero()) // virtual Lt liquidity exists after the concentrate step
	requireSameAsRef(t, program, &bGt, new(uint256.Int), swapAmount, false)

	// Spot below the lower bound: bGt = 0, bLt = L * (1/spot - 1/pmax); selling Gt for Lt works.
	spot = new(uint256.Int).SubUint64(pminSqrt, 100)
	var invSpot, invMax, bLt uint256.Int
	require.True(t, mulDiv(&invSpot, uOne, uOne, spot))
	require.True(t, mulDiv(&invMax, uOne, uOne, pmaxSqrt))
	require.True(t, mulDiv(&bLt, targetL, new(uint256.Int).Sub(&invSpot, &invMax), uOne))
	res, err = runQuote(program, new(uint256.Int), &bLt, swapAmount, false)
	require.NoError(t, err)
	require.False(t, res.AmountOut.IsZero())
	require.Equal(t, swapAmount, &res.AmountIn)
	requireSameAsRef(t, program, new(uint256.Int), &bLt, swapAmount, false)
}

func TestRunProgramErrors(t *testing.T) {
	t.Parallel()
	bal, amt := e18(10), uint256.NewInt(1000)
	// Both balances must be non-zero when the swap runs.
	_, err := runQuote(mustParse(t, prog(iSwap()), false), new(uint256.Int), bal, amt, true)
	require.ErrorIs(t, err, ErrZeroBalance)
	_, err = runQuote(mustParse(t, prog(iSwap()), false), bal, new(uint256.Int), amt, true)
	require.ErrorIs(t, err, ErrZeroBalance)

	// Hand-built programs the parser refuses.
	for name, p := range map[string][]Instruction{
		"no swap":           {{Op: opSalt}},
		"unknown opcode":    {{Op: 0x30}, {Op: opXYCSwap}},
		"wrapper after":     {{Op: opXYCSwap}, {Op: opFlatFeeIn}},
		"flat bps":          {{Op: opFlatFeeIn, Bps: feeBpsDenominator + 1}, {Op: opXYCSwap}},
		"protocol bps":      {{Op: opAquaProtocolFee, Bps: feeBpsDenominator + 1}, {Op: opXYCSwap}},
		"unknown after xyc": {{Op: opXYCSwap}, {Op: 0x30}},
	} {
		_, err := runQuote(p, bal, bal, amt, true)
		require.ErrorIs(t, err, ErrBadProgram, name)
	}
}

// Every checked-arithmetic site of the program math returns ErrOverflow where the router reverts.
func TestRunProgramOverflowSites(t *testing.T) {
	t.Parallel()
	max := new(uint256.Int).Not(new(uint256.Int))
	half := new(uint256.Int).Rsh(max, 1)
	pmin, pmax := sqrtPrice(e18(1)), sqrtPrice(e18(4))
	swap := mustParse(t, prog(iSwap()), false)
	flat := mustParse(t, prog(iFlat(1_000_000_000), iSwap()), false)
	protocol := mustParse(t, prog(iProtocol(1_000_000_000, testRecipient), iSwap()), false)
	conc := mustParse(t, prog(iConcentrate(pmin, pmax), iSwap()), false)

	for name, tc := range map[string]struct {
		program []Instruction
		in, out *uint256.Int
		amount  *uint256.Int
		inLt    bool
	}{
		"swap num":                       {swap, e18(1), max, e18(10), true},
		"swap den":                       {swap, max, e18(1), uint256.NewInt(2), true},
		"flat fee mul":                   {flat, e18(1), e18(1), max, true},
		"protocol fee mul":               {protocol, e18(1), e18(1), max, true},
		"concentrate beta mulDiv in lt":  {conc, max, e18(1), e18(1), true},
		"concentrate beta mulDiv gt":     {conc, e18(1), max, e18(1), true},
		"concentrate fourAC":             {conc, half, half, e18(1), true},
		"concentrate beta sum":           {conc, max, max, e18(1), true},
		"concentrate disc":               {conc, new(uint256.Int).Rsh(max, 20), new(uint256.Int).Rsh(max, 20), e18(1), false},
		"concentrate balance add in":     {conc, new(uint256.Int).Rsh(max, 2), new(uint256.Int).Rsh(max, 70), e18(1), true},
		"concentrate balance add out":    {conc, new(uint256.Int).Rsh(max, 70), new(uint256.Int).Rsh(max, 2), e18(1), true},
		"concentrate L * price overflow": {conc, new(uint256.Int).Rsh(max, 10), new(uint256.Int).Rsh(max, 10), e18(1), false},
	} {
		requireSameAsRef(t, tc.program, tc.in, tc.out, tc.amount, tc.inLt)
		_ = name
	}

	// Direct hits on each concentrate overflow branch, checked against the oracle over a sweep of
	// large magnitudes and both directions.
	rng := rand.New(rand.NewPCG(1, 2))
	hits := 0
	for range 4000 {
		a := new(uint256.Int).Rsh(max, uint(rng.IntN(250)))
		b := new(uint256.Int).Rsh(max, uint(rng.IntN(250)))
		for _, lt := range []bool{true, false} {
			requireSameAsRef(t, conc, a, b, uint256.NewInt(1+rng.Uint64()), lt)
			if _, err := runQuote(conc, a, b, uint256.NewInt(7), lt); err != nil {
				hits++
			}
		}
	}
	require.Positive(t, hits, "the sweep reaches the overflow branches")
}

func TestComputeLOverflowBranches(t *testing.T) {
	t.Parallel()
	max := new(uint256.Int).Not(new(uint256.Int))
	pmin, pmax := sqrtPrice(e18(1)), sqrtPrice(e18(4))
	small := uint256.NewInt(1000)
	for name, tc := range map[string]struct{ bLt, bGt *uint256.Int }{
		"beta first term":  {max, small},
		"beta second term": {small, max},
		"beta sum":         {new(uint256.Int).Rsh(max, 1), new(uint256.Int).Rsh(max, 1)},
		"fourAC mulDiv":    {max, uint256.NewInt(0)},
		"fourAC mul":       {new(uint256.Int).Rsh(max, 60), new(uint256.Int).Rsh(max, 60)},
		"disc add":         {new(uint256.Int).Rsh(max, 70), new(uint256.Int).Rsh(max, 20)},
		"result":           {new(uint256.Int).Rsh(max, 64), new(uint256.Int).Rsh(max, 64)},
	} {
		got, err := liquidityOf(tc.bLt, tc.bGt, pmin, pmax)
		want, ok := refComputeL(tc.bLt.ToBig(), tc.bGt.ToBig(), pmin.ToBig(), pmax.ToBig())
		if ok {
			require.NoError(t, err, name)
			require.Equal(t, want.String(), got.Dec(), name)
		} else {
			require.ErrorIs(t, err, ErrOverflow, name)
		}
	}
}

// Inputs found by random search that reach the overflow sites of concentrate which the grid skips:
// the L*sqrtPmin product, and the balance additions.
func TestConcentrateOverflowVectors(t *testing.T) {
	t.Parallel()
	pmin := u("41434005084054764129358893811819112594628804517")
	pmax := u("106825183143674097389025401469111952625743676683074512356464394900593464351")
	program := []Instruction{{Op: opXYCConcentrate, SqrtPMin: *pmin, SqrtPMax: *pmax}, {Op: opXYCSwap}}
	balLt, balGt := u("6463300147"), u("6178255904588932738105680602416049")
	// tokenIn is the greater token: balanceIn = bGt, and L*sqrtPmin overflows.
	requireSameAsRef(t, program, balGt, balLt, uint256.NewInt(1000), false)
	_, err := runQuote(program, balGt, balLt, uint256.NewInt(1000), false)
	require.ErrorIs(t, err, ErrOverflow)

	pmin2 := u("36025724224109817158817165803800369838823823059501792634591229093175507")
	pmax2 := u("108459402709664198692753221497931381808464941912905117355489654747559659")
	program2 := []Instruction{{Op: opXYCConcentrate, SqrtPMin: *pmin2, SqrtPMax: *pmax2}, {Op: opXYCSwap}}
	bGt := u("86971514567132891689963162313431466463774079859419833848274168878514612862852")
	// tokenIn is the lower token: balanceOut = bGt and the out-side addition overflows.
	_, err = runQuote(program2, new(uint256.Int), bGt, uint256.NewInt(1000), true)
	require.ErrorIs(t, err, ErrOverflow)
	requireSameAsRef(t, program2, new(uint256.Int), bGt, uint256.NewInt(1000), true)
	// tokenIn is the greater token: balanceIn = bGt and the in-side addition overflows.
	_, err = runQuote(program2, bGt, new(uint256.Int), uint256.NewInt(1000), false)
	require.ErrorIs(t, err, ErrOverflow)
	requireSameAsRef(t, program2, bGt, new(uint256.Int), uint256.NewInt(1000), false)
	// computeL rejects a zero upper bound instead of dividing by it.
	_, err = liquidityOf(uOne, uOne, new(uint256.Int), new(uint256.Int))
	require.ErrorIs(t, err, ErrOverflow)
}

func TestMulDivCeilDiv(t *testing.T) {
	t.Parallel()
	var z uint256.Int
	require.False(t, mulDiv(&z, uOne, uOne, new(uint256.Int)), "zero divisor")
	max := new(uint256.Int).Not(new(uint256.Int))
	require.False(t, mulDiv(&z, max, max, uint256.NewInt(2)), "result above 2^256")
	require.True(t, mulDiv(&z, max, max, max))
	require.Equal(t, max, &z)

	ceilDiv(&z, new(uint256.Int), uint256.NewInt(7))
	require.True(t, z.IsZero())
	ceilDiv(&z, uint256.NewInt(7), uint256.NewInt(7))
	require.Equal(t, uint64(1), z.Uint64())
	ceilDiv(&z, uint256.NewInt(8), uint256.NewInt(7))
	require.Equal(t, uint64(2), z.Uint64())
	ceilDiv(&z, max, uint256.NewInt(1))
	require.Equal(t, max, &z, "no overflow at the top")
}

// Live router quotes (Ethereum, BSC, Robinhood) at a pinned block: strategy bytes, raw balances and the
// amounts the router returned.
func TestGoldenRouterQuotes(t *testing.T) {
	t.Parallel()
	vectors := loadGolden(t)
	require.Len(t, vectors, 48)
	for _, v := range vectors {
		o, err := DecodeOrder(hexutil.MustDecode(v.Strategy))
		require.NoError(t, err)
		launch := strings.EqualFold(v.App, testLaunchHex)
		program, err := ParseOrder(o, launch)
		require.NoError(t, err)

		in, out := v.In, 1-v.In
		inLt := common.HexToAddress(v.Tokens[in]).Cmp(common.HexToAddress(v.Tokens[out])) < 0
		res, err := runQuote(program, u(v.Raw[in]), u(v.Raw[out]), u(v.AmountIn), inLt)
		require.NoError(t, err)
		require.Equal(t, v.AmountOut, res.AmountOut.Dec(), "chain %d amountIn %s", v.ChainId, v.AmountIn)
		require.Equal(t, v.AmountIn, res.AmountIn.Dec())
	}
}

// Randomized comparison with the oracle over every shape, magnitudes from dust to near 2^256.
func TestRunProgramRandomVsOracle(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(7, 11))
	pminA, pmaxA := sqrtPrice(e18(1)), sqrtPrice(e18(4))
	pminB, pmaxB := uint256.NewInt(1e18-1), uint256.NewInt(1e18+1)
	shapes := [][]byte{
		prog(iSwap()),
		prog(iFlat(10_000_000), iSwap()),
		prog(iConcentrate(pminA, pmaxA), iFlat(10_000_000), iSwap(), iSalt(3)),
		prog(iFlat(3), iConcentrate(pminB, pmaxB), iSwap()),
		prog(iProtocol(10_000_000, testRecipient), iConcentrate(pminA, pmaxA), iFlat(10_000_000), iSwap(), iSalt(1)),
		prog(iSalt(1), iProtocol(100_000_000, testRecipient), iProtocol(200_000_000, testRecipient), iFlat(50_000_000), iSwap()),
	}
	randWord := func() *uint256.Int {
		z := new(uint256.Int)
		z.SetUint64(rng.Uint64())
		for range rng.IntN(4) {
			z.Lsh(z, 64).AddUint64(z, rng.Uint64())
		}
		return z.Rsh(z, uint(rng.IntN(200)))
	}
	for range 3000 {
		for _, shape := range shapes {
			program := mustParse(t, shape, false)
			requireSameAsRef(t, program, randWord(), randWord(), randWord(), rng.IntN(2) == 0)
		}
	}
}

// Properties the pricing must keep: the output never exceeds the out balance plus the virtual liquidity,
// is monotone in the input, and a bigger flat fee never raises it.
func TestRunProgramProperties(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(3, 5))
	pmin, pmax := sqrtPrice(e18(1)), sqrtPrice(e18(4))
	conc := mustParse(t, prog(iConcentrate(pmin, pmax), iFlat(10_000_000), iSwap()), false)
	plainFee := func(bps uint32) []Instruction { return mustParse(t, prog(iFlat(bps), iSwap()), false) }
	for range 2000 {
		balIn := new(uint256.Int).Rsh(uint256.NewInt(rng.Uint64()), uint(rng.IntN(60)))
		balOut := new(uint256.Int).Add(new(uint256.Int).Rsh(uint256.NewInt(rng.Uint64()), uint(rng.IntN(60))), uint256.NewInt(1))
		a1 := new(uint256.Int).Rsh(uint256.NewInt(rng.Uint64()), uint(rng.IntN(60)))
		a2 := new(uint256.Int).Add(a1, uint256.NewInt(rng.Uint64()%1000))
		lt := rng.IntN(2) == 0

		r1, err1 := runQuote(conc, balIn, balOut, a1, lt)
		r2, err2 := runQuote(conc, balIn, balOut, a2, lt)
		if err1 == nil && err2 == nil {
			require.False(t, r2.AmountOut.Lt(&r1.AmountOut), "monotone in amountIn")
		}

		if balIn.IsZero() {
			continue
		}
		lowFee, errLow := runQuote(plainFee(1_000_000), balIn, balOut, a2, lt)
		highFee, errHigh := runQuote(plainFee(50_000_000), balIn, balOut, a2, lt)
		require.NoError(t, errLow)
		require.NoError(t, errHigh)
		require.False(t, highFee.AmountOut.Gt(&lowFee.AmountOut), "a bigger fee never raises the output")
		require.True(t, lowFee.AmountOut.Lt(balOut) || balOut.IsZero() || lowFee.AmountOut.IsZero())
	}
}

func FuzzRunProgram(f *testing.F) {
	pmin, pmax := sqrtPrice(e18(1)), sqrtPrice(e18(4))
	shapes := [][]byte{
		prog(iSwap()),
		prog(iConcentrate(pmin, pmax), iFlat(10_000_000), iSwap()),
		prog(iProtocol(10_000_000, testRecipient), iConcentrate(pmin, pmax), iFlat(10_000_000), iSwap(), iSalt(1)),
		prog(iSalt(1), iProtocol(100_000_000, testRecipient), iProtocol(200_000_000, testRecipient), iFlat(50_000_000), iSwap()),
	}
	f.Add(uint64(1), uint64(2), uint64(3), uint64(4), uint64(5), uint64(6), uint8(0), true)
	f.Add(^uint64(0), ^uint64(0), uint64(0), uint64(0), uint64(1e18), uint64(1e18), uint8(1), false)
	f.Add(uint64(1e18), uint64(0), uint64(0), uint64(0), uint64(5e18), uint64(0), uint8(2), true)
	f.Fuzz(func(t *testing.T, a, b, c, d, e, g uint64, shape uint8, inLt bool) {
		program := mustParse(t, shapes[int(shape)%len(shapes)], false)
		balIn := &uint256.Int{a, b, 0, 0}
		balOut := &uint256.Int{c, d, 0, 0}
		amount := &uint256.Int{e, g, 0, 0}
		requireSameAsRef(t, program, balIn, balOut, amount, inLt)
		// Wide values too.
		requireSameAsRef(t, program, &uint256.Int{a, b, c, d}, &uint256.Int{c, d, e, g}, &uint256.Int{e, g, a, b}, inLt)
	})
}

// magnitudeGrid covers dust to the top of uint256 so that every checked-arithmetic site of the
// concentrate math is reached, and each result is compared with the oracle.
func magnitudeGrid() []*uint256.Int {
	max := new(uint256.Int).Not(new(uint256.Int))
	grid := []*uint256.Int{new(uint256.Int), uint256.NewInt(1), uOne, max}
	for k := uint(8); k < 256; k += 30 {
		v := new(uint256.Int).Lsh(uint256.NewInt(1), k)
		grid = append(grid, v, new(uint256.Int).SubUint64(v, 1))
	}
	return grid
}

func TestConcentrateMagnitudeGrid(t *testing.T) {
	t.Parallel()
	grid := magnitudeGrid()
	var rawHits, errHits int
	for _, bLt := range grid {
		for _, bGt := range grid {
			for _, pmin := range grid {
				for _, pmax := range grid {
					if !pmin.Lt(pmax) {
						continue
					}
					got, err := liquidityOf(bLt, bGt, pmin, pmax)
					want, ok := refComputeL(bLt.ToBig(), bGt.ToBig(), pmin.ToBig(), pmax.ToBig())
					if pmax.IsZero() {
						continue
					}
					if ok {
						require.NoError(t, err)
						require.Equal(t, want.String(), got.Dec())
						rawHits++
					} else {
						require.ErrorIs(t, err, ErrOverflow)
						errHits++
					}
					if !ok {
						continue
					}
					for _, lt := range []bool{true, false} {
						program := []Instruction{{Op: opXYCConcentrate, SqrtPMin: *pmin, SqrtPMax: *pmax}, {Op: opXYCSwap}}
						requireSameAsRef(t, program, bLt, bGt, uint256.NewInt(1000), lt)
					}
				}
			}
		}
	}
	require.Positive(t, rawHits)
	require.Positive(t, errHits)
}
