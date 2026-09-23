package thogprop

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// maxU256 is (1<<256)-1, the ceiling every amount/word is checked against.
var maxU256 = new(uint256.Int).Not(new(uint256.Int))

// maxRiskNotionalUsdWad is exactQuote's MAX_RISK_NOTIONAL_USD_WAD constant.
var maxRiskNotionalUsdWad = uint256.MustFromDecimal("664619068552888026590626425199190042")

// pow10Table[n] = 10^n for n in [0,18], the only exponents priceWad/priceOutWad ever need
// (exactQuote requires exponent<=18).
var pow10Table = func() [19]uint256.Int {
	var t [19]uint256.Int
	t[0].SetOne()
	for i := 1; i < len(t); i++ {
		t[i].Mul(&t[i-1], uint256.NewInt(10))
	}
	return t
}()

// State is exactQuote's decoded snapshot image -- the ten packed words plus
// balances/readiness flags.
type State struct {
	V1, V2, V3     uint256.Int
	R1, R2, R3     uint256.Int
	I1, I2         uint256.Int
	P1, P2         uint256.Int
	GloballyPaused bool
	RiskV3Ready    bool
	PairRiskReady  bool
	Balances       []uint256.Int // indexed by tokenMeta.Index
}

func bits(word *uint256.Int, shift, width uint) uint256.Int {
	var z, mask uint256.Int
	mask.SetOne()
	mask.Lsh(&mask, width)
	mask.SubUint64(&mask, 1)
	z.Rsh(word, shift)
	z.And(&z, &mask)
	return z
}

// signedN reinterprets a width-bit unsigned field as two's-complement signed by sign-extending
// it into the full 256-bit width: shift the field into the top bits, then arithmetic-shift back.
func signedN(raw *uint256.Int, width uint) uint256.Int {
	var z uint256.Int
	shift := 256 - width
	z.Lsh(raw, shift)
	z.SRsh(&z, shift)
	return z
}

// maxFlooredInput returns the largest x for which floor(x*multiplier/divisor) <= maximum, clamped
// to maxU256. exactQuote's pseudocode is `ceilDiv((maximum+1)*divisor, multiplier) - 1`, but
// maximum+1 overflows uint256 when maximum==maxU256 (a real, expected input here -- maxNotional's
// first call always passes maxU256). Rewritten algebraically to avoid ever forming (maximum+1):
//
//	ceilDiv((maximum+1)*divisor, multiplier) - 1
//	  == floor(((maximum+1)*divisor - 1) / multiplier)                (ceilDiv(N,M)-1 == floor((N-1)/M) for N>0)
//	  == floor((maximum*divisor + divisor - 1) / multiplier)
//	  == q + floor((rem + divisor - 1) / multiplier)                  where maximum*divisor = q*multiplier + rem
//
// maximum*divisor is the genuinely wide (up to 512-bit) term; MulDivOverflow/MulMod compute it
// with full internal precision, so q and rem are each exact without ever materializing the
// intermediate. The correction term is a small, ordinary uint256 division.
func maxFlooredInput(maximum, divisor, multiplier *uint256.Int) uint256.Int {
	var q uint256.Int
	_, overflow := q.MulDivOverflow(maximum, divisor, multiplier)
	if overflow {
		return *maxU256
	}

	var rem uint256.Int
	rem.MulMod(maximum, divisor, multiplier)

	var corrNum uint256.Int
	corrNum.Add(&rem, divisor)
	corrNum.SubUint64(&corrNum, 1)
	var corr uint256.Int
	corr.Div(&corrNum, multiplier)

	result, carry := q.AddOverflow(&q, &corr)
	if carry || result.Cmp(maxU256) > 0 {
		return *maxU256
	}
	return *result
}

func wordFor(s *State, w priceWord) *uint256.Int {
	switch w {
	case priceWordV1:
		return &s.V1
	case priceWordV2:
		return &s.V2
	default:
		return &s.V3
	}
}

// priceWad decodes a token's price as USD WAD per whole token, the "lower endpoint" used for
// priceIn and inventory/fee accounting.
func priceWad(s *State, t tokenMeta) (uint256.Int, error) {
	encoded := bits(wordFor(s, t.Word), t.Shift, 48)
	var mantissa uint256.Int
	mantissa.And(&encoded, uint256.NewInt((1<<40)-1))
	exponent := new(uint256.Int).Rsh(&encoded, 40)
	if mantissa.IsZero() || exponent.CmpUint64(18) > 0 {
		return uint256.Int{}, ErrStalePrice
	}
	mantissa.Mul(&mantissa, &pow10Table[18-exponent.Uint64()])
	return mantissa, nil
}

// priceOutWad advances the mantissa by one so packed precision can't increase taker output; used
// only for the output-side fair-value leg.
func priceOutWad(s *State, t tokenMeta) (uint256.Int, error) {
	encoded := bits(wordFor(s, t.Word), t.Shift, 48)
	var mantissa uint256.Int
	mantissa.And(&encoded, uint256.NewInt((1<<40)-1))
	exponent := new(uint256.Int).Rsh(&encoded, 40)
	if mantissa.CmpUint64(200000) < 0 || exponent.CmpUint64(18) > 0 {
		return uint256.Int{}, ErrStalePrice
	}
	mantissa.AddUint64(&mantissa, 1)
	mantissa.Mul(&mantissa, &pow10Table[18-exponent.Uint64()])
	return mantissa, nil
}

// covarianceFieldByCategory maps an (left,right) category pair (left<=right) to its packed field
// index within r2 (fields 0-5) or r3 (fields 6-9).
var covarianceFieldByCategory = map[[2]int]int{
	{1, 1}: 0, {1, 2}: 1, {1, 3}: 2,
	{2, 2}: 3, {2, 3}: 4, {3, 3}: 5,
	{1, 4}: 6, {2, 4}: 7, {3, 4}: 8, {4, 4}: 9,
}

func covariance(s *State, left, right int) uint256.Int {
	if left == 0 || right == 0 {
		return uint256.Int{}
	}
	if left > right {
		left, right = right, left
	}
	field := covarianceFieldByCategory[[2]int{left, right}]
	if field < 6 {
		b := bits(&s.R2, uint(field)*16, 16)
		return signedN(&b, 16)
	}
	b := bits(&s.R3, uint(field-6)*16, 16)
	return signedN(&b, 16)
}

func exposure(s *State, category int) uint256.Int {
	if category == 4 {
		b := bits(&s.I2, 0, 56)
		return signedN(&b, 56)
	}
	b := bits(&s.I1, uint(category)*56, 56)
	return signedN(&b, 56)
}

// anchorTokenIndexByCategory: WMON, WETH, WBTC, XAUt0 are categories 1-4's own anchor price source.
var anchorTokenIndexByCategory = map[int]int{1: 3, 2: 4, 3: 5, 4: 7}

// exposureGradient magnitude stays comfortably under 2^255 across realistic exposure/covariance
// ranges (56-bit exposure x ~100-bit anchor price / 1e6, x 16-bit covariance / 1e9, summed over 4
// terms) -- safe for plain signed Mul+SDiv, verified against the golden fixtures in math_test.go.
func exposureGradient(s *State, category int) (uint256.Int, error) {
	var gradient uint256.Int
	if category == 0 {
		return gradient, nil
	}
	for other := 1; other <= 4; other++ {
		anchorPrice, err := priceWad(s, tokenTable[anchorTokenIndexByCategory[other]])
		if err != nil {
			return uint256.Int{}, err
		}
		exp := exposure(s, other)
		var exposureUsdWad uint256.Int
		exposureUsdWad.Mul(&exp, &anchorPrice)
		exposureUsdWad.SDiv(&exposureUsdWad, uint256.NewInt(exposureScale))

		cov := covariance(s, category, other)
		var term uint256.Int
		term.Mul(&exposureUsdWad, &cov)
		term.SDiv(&term, uint256.NewInt(covarianceScale))
		gradient.Add(&gradient, &term)
	}
	return gradient, nil
}

// penaltyUsdWad's variance term (notionalUsdWad^2 * varianceRate) is the one signed product whose
// magnitude approaches the 255-bit signed boundary: notionalUsdWad is bounded by
// maxRiskNotionalUsdWad (~120 bits) whenever this branch runs (maxNotional is risk-capped to the
// same constant under the same tokenIn.Category!=tokenOut.Category && lambdaQ!=0 condition that
// gates this function), so notionalUsdWad^2*varianceRate stays under 2^255 -- verified numerically
// (255-bit magnitude at the documented worst case) before choosing plain Mul+SDiv over a wider type.
func penaltyUsdWad(s *State, tokenIn, tokenOut tokenMeta, notionalUsdWad *uint256.Int) (uint256.Int, error) {
	lambdaQ := bits(&s.R1, 110, 16)
	if lambdaQ.IsZero() || tokenIn.Category == tokenOut.Category {
		return uint256.Int{}, nil
	}

	var varianceRate uint256.Int
	covII := covariance(s, tokenIn.Category, tokenIn.Category)
	covOO := covariance(s, tokenOut.Category, tokenOut.Category)
	covIO := covariance(s, tokenIn.Category, tokenOut.Category)
	varianceRate.Add(&covII, &covOO)
	var twiceCovIO uint256.Int
	twiceCovIO.Mul(uint256.NewInt(2), &covIO)
	varianceRate.Sub(&varianceRate, &twiceCovIO)

	// delta term1: notionalUsdWad^2 * varianceRate / 1e27. notionalUsdWad^2 alone can exceed
	// 256 bits in general, but per this function's guard above, notionalUsdWad is bounded by
	// maxRiskNotionalUsdWad here (see comment above the func), keeping the full product under
	// 2^255 (verified numerically) -- safe for plain Mul.
	var delta, tmp, e27 uint256.Int
	tmp.Mul(notionalUsdWad, notionalUsdWad)
	delta.Mul(&tmp, &varianceRate)
	e27.Mul(&pow10Table[18], uint256.NewInt(1_000_000_000))
	delta.SDiv(&delta, &e27)

	gradIn, err := exposureGradient(s, tokenIn.Category)
	if err != nil {
		return uint256.Int{}, err
	}
	gradOut, err := exposureGradient(s, tokenOut.Category)
	if err != nil {
		return uint256.Int{}, err
	}
	var gradDiff, term2 uint256.Int
	gradDiff.Sub(&gradIn, &gradOut)
	term2.Mul(notionalUsdWad, &gradDiff)
	term2.Mul(&term2, uint256.NewInt(2))
	term2.SDiv(&term2, &pow10Table[18])
	delta.Add(&delta, &term2)

	if delta.Sign() <= 0 {
		return uint256.Int{}, nil
	}

	var penalty uint256.Int
	penalty.Mul(&delta, &lambdaQ)
	penalty.SDiv(&penalty, uint256.NewInt(lambdaScale))
	return penalty, nil
}

// pairIndex returns exactQuote's symmetric pair field index for two token indices, left<right
// required by the caller.
func pairIndex(left, right int) int {
	return left*(2*8-left-1)/2 + (right - left - 1)
}

// ExactQuote ports exactQuote() from THOGAMM_AGGREGATOR_INTEGRATION.md's in-memory mirror spec
// verbatim, operation order and all. Verified exact-to-the-wei against four live
// makerQuoteExactInput() fixtures at block 107066625 (see math_test.go) covering
// cross-category-with-penalty, same-category, and both XAUt0/index-7 directions.
//
// Every `require` in the pseudocode becomes a typed sentinel error here, not a hard failure: the
// pathfinder calls this at many sizes and treats "no quote at this size" as routine.
func ExactQuote(s *State, tokenIn, tokenOut tokenMeta, amountIn *uint256.Int, executionBlock uint64, fastLaneHot bool) (uint256.Int, uint64, error) {
	if s.GloballyPaused {
		return uint256.Int{}, 0, ErrGloballyPaused
	}
	priceSurfacePauseBit := bits(&s.V1, 0, 1)
	if !priceSurfacePauseBit.IsZero() {
		return uint256.Int{}, 0, ErrPriceSurfacePaused
	}
	if !s.PairRiskReady {
		return uint256.Int{}, 0, ErrPairRiskNotReady
	}
	if tokenIn.Index == tokenIdxXAUt || tokenOut.Index == tokenIdxXAUt {
		if !s.RiskV3Ready {
			return uint256.Int{}, 0, ErrRiskV3NotReady
		}
	}
	if amountIn.IsZero() {
		return uint256.Int{}, 0, ErrZeroNotional
	}
	if tokenIn.Index == tokenOut.Index {
		return uint256.Int{}, 0, ErrSameToken
	}

	v3High2 := bits(&s.V3, 48, 2)
	var sides uint256.Int
	sides.Lsh(&v3High2, 14)
	v1Mid14 := bits(&s.V1, 1, 14)
	sides.Or(&sides, &v1Mid14)
	var sellBit, buyBit uint256.Int
	sellBit.SetOne()
	sellBit.Lsh(&sellBit, uint(tokenIn.Index)*2)
	buyBit.SetOne()
	buyBit.Lsh(&buyBit, uint(tokenOut.Index)*2+1)
	var sellCheck, buyCheck uint256.Int
	sellCheck.And(&sides, &sellBit)
	buyCheck.And(&sides, &buyBit)
	if sellCheck.IsZero() || buyCheck.IsZero() {
		return uint256.Int{}, 0, ErrSideDisabled
	}

	postedBlockBig := bits(&s.V1, 31, 32)
	postedBlock := postedBlockBig.Uint64()
	if postedBlock == 0 || executionBlock < postedBlock {
		return uint256.Int{}, 0, ErrStalePrice
	}
	age := executionBlock - postedBlock
	maxAgeBits := bits(&s.R1, 198, 8)
	maxAge := maxAgeBits.Uint64()
	if maxAge == 0 {
		return uint256.Int{}, 0, ErrZeroMaxAge
	}
	if age > maxAge {
		return uint256.Int{}, 0, ErrStalePrice
	}

	inputPriceWad, err := priceWad(s, tokenIn)
	if err != nil {
		return uint256.Int{}, 0, err
	}
	outputPriceWad, err := priceOutWad(s, tokenOut)
	if err != nil {
		return uint256.Int{}, 0, err
	}
	inputScale := pow10Table[tokenIn.Decimals]
	outputScale := pow10Table[tokenOut.Decimals]

	maxNotional := maxFlooredInput(maxU256, &outputPriceWad, &outputScale)
	lambdaQ := bits(&s.R1, 110, 16)
	if tokenIn.Category != tokenOut.Category && !lambdaQ.IsZero() {
		if maxRiskNotionalUsdWad.Cmp(&maxNotional) < 0 {
			maxNotional = *maxRiskNotionalUsdWad
		}
	}

	inputLimit := maxFlooredInput(&maxNotional, &inputScale, &inputPriceWad)
	if amountIn.Cmp(&inputLimit) > 0 {
		return uint256.Int{}, 0, ErrAmountTooLarge
	}

	var notionalUsdWad uint256.Int
	if _, overflow := notionalUsdWad.MulDivOverflow(amountIn, &inputPriceWad, &inputScale); overflow {
		return uint256.Int{}, 0, ErrOverflow
	}
	if notionalUsdWad.IsZero() {
		return uint256.Int{}, 0, ErrZeroNotional
	}

	categoryShiftIn := categoryShift(tokenIn.Category)
	categoryShiftOut := categoryShift(tokenOut.Category)
	tokenShiftIn := tokenShift(tokenIn.Index)
	tokenShiftOut := tokenShift(tokenOut.Index)

	left, right := tokenIn.Index, tokenOut.Index
	if left > right {
		left, right = right, left
	}
	pi := pairIndex(left, right)
	var pairRaw uint256.Int
	if pi < 25 {
		pairRaw = bits(&s.P1, uint(pi)*10, 10)
	} else {
		pairRaw = bits(&s.P2, uint(pi-25)*10, 10)
	}

	bCatIn := bits(&s.R1, categoryShiftIn, 10)
	bCatOut := bits(&s.R1, categoryShiftOut, 10)
	bTokIn := bits(&s.R1, tokenShiftIn, 10)
	bTokOut := bits(&s.R1, tokenShiftOut, 10)
	var sum uint256.Int
	sum.Add(&bCatIn, &bCatOut)
	sum.Add(&sum, &bTokIn)
	sum.Add(&sum, &bTokOut)
	sum.Add(&sum, &pairRaw)

	perBlockWidening := bits(&s.R1, 190, 8)
	var spreadUnits, ageTerm uint256.Int
	spreadUnits.Mul(uint256.NewInt(4), &sum)
	ageTerm.Mul(uint256.NewInt(age), &perBlockWidening)
	spreadUnits.Add(&spreadUnits, &ageTerm)
	if spreadUnits.CmpUint64(spreadDenominator) >= 0 {
		return uint256.Int{}, 0, ErrSpreadTooWide
	}

	var fairOut uint256.Int
	if _, overflow := fairOut.MulDivOverflow(&notionalUsdWad, &outputScale, &outputPriceWad); overflow {
		return uint256.Int{}, 0, ErrOverflow
	}

	var spreadFactor uint256.Int
	spreadFactor.Sub(uint256.NewInt(spreadDenominator), &spreadUnits)
	var amountOut uint256.Int
	if _, overflow := amountOut.MulDivOverflow(&fairOut, &spreadFactor, uint256.NewInt(spreadDenominator)); overflow {
		return uint256.Int{}, 0, ErrOverflow
	}

	penalty, err := penaltyUsdWad(s, tokenIn, tokenOut, &notionalUsdWad)
	if err != nil {
		return uint256.Int{}, 0, err
	}
	if !penalty.IsZero() {
		outTokenPriceWad, err := priceWad(s, tokenOut)
		if err != nil {
			return uint256.Int{}, 0, err
		}
		penaltyOut := ceilDivMinus1WithMul(&penalty, &outputScale, &outTokenPriceWad)
		if amountOut.Cmp(&penaltyOut) <= 0 {
			return uint256.Int{}, 0, ErrPenaltyExceedsOut
		}
		amountOut.Sub(&amountOut, &penaltyOut)
	}

	if fastLaneHot {
		var haircutFactor uint256.Int
		haircutFactor.SetUint64(10000 - fastLaneFrictionBps)
		if _, overflow := amountOut.MulDivOverflow(&amountOut, &haircutFactor, uint256.NewInt(10000)); overflow {
			return uint256.Int{}, 0, ErrOverflow
		}
	}

	if amountOut.IsZero() {
		return uint256.Int{}, 0, ErrZeroAmountOut
	}
	if tokenOut.Index >= len(s.Balances) || amountOut.Cmp(&s.Balances[tokenOut.Index]) > 0 {
		return uint256.Int{}, 0, ErrInsufficientBal
	}

	return amountOut, postedBlock, nil
}

// ceilDivMinus1WithMul computes ceilDiv(a*b, c) with full precision -- a is a WAD-scale penalty
// (far smaller than notionalUsdWad's own bound), b/c are token scale/price, so a*b is safely within
// 256 bits; kept as a full-precision MulDivUp regardless since penalty magnitude isn't otherwise
// bounded by a documented protocol constant the way notionalUsdWad is.
func ceilDivMinus1WithMul(a, b, c *uint256.Int) uint256.Int {
	var res uint256.Int
	big256.MulDivUp(&res, a, b, c)
	return res
}
