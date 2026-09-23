package thogprop

import (
	"math/big"
)

// maxU256 is (1<<256)-1, the ceiling every amount/word is checked against.
var maxU256 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))

// maxRiskNotionalUsdWad is exactQuote's MAX_RISK_NOTIONAL_USD_WAD constant.
var maxRiskNotionalUsdWad, _ = new(big.Int).SetString("664619068552888026590626425199190042", 10)

// State is exactQuote's decoded snapshot image -- the ten packed words plus
// balances/readiness flags, all as arbitrary-precision integers.
//
// This math is ported using math/big rather than uint256.Int deliberately:
// exactQuote's own pseudocode requires "arbitrary-precision intermediates"
// for maxFlooredInput (it multiplies (maximum+1) which can be 2**256, one
// past uint256's range) and mixes signed 16/56-bit fields that are simplest
// to keep correct as big.Int during this initial port. The pipeline's
// deferred simulator-optimize pass (run after dex-verify is green) is the
// sanctioned place to convert this to zero-alloc uint256 without changing
// behavior -- see dex-implement-math/references/simulator-optimize.md.
type State struct {
	V1, V2, V3     *big.Int
	R1, R2, R3     *big.Int
	I1, I2         *big.Int
	P1, P2         *big.Int
	GloballyPaused bool
	RiskV3Ready    bool
	PairRiskReady  bool
	Balances       []*big.Int // indexed by tokenMeta.Index
}

func bits(word *big.Int, shift, width uint) *big.Int {
	mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), width), big.NewInt(1))
	return new(big.Int).And(new(big.Int).Rsh(word, shift), mask)
}

// signedN reinterprets a width-bit unsigned field as two's-complement signed.
func signedN(raw *big.Int, width uint) *big.Int {
	signBit := new(big.Int).Lsh(big.NewInt(1), width-1)
	if raw.Cmp(signBit) < 0 {
		return raw
	}
	return new(big.Int).Sub(raw, new(big.Int).Lsh(big.NewInt(1), width))
}

func ceilDiv(numerator, denominator *big.Int) *big.Int {
	sum := new(big.Int).Sub(new(big.Int).Add(numerator, denominator), big.NewInt(1))
	return new(big.Int).Div(sum, denominator)
}

// trunc0 divides truncating toward zero, matching Solidity/EVM signed
// division (and exactQuote's explicit trunc0 pseudocode) -- unlike Go's
// big.Int.Div, which floors.
func trunc0(numerator, denominator *big.Int) *big.Int {
	q := new(big.Int)
	q.Quo(numerator, denominator)
	return q
}

// maxFlooredInput returns the largest x for which floor(x*multiplier/divisor)
// <= maximum, clamped to maxU256. Ported with arbitrary precision because
// (maximum+1) can equal 2**256 when maximum == maxU256.
func maxFlooredInput(maximum, divisor, multiplier *big.Int) *big.Int {
	num := new(big.Int).Mul(new(big.Int).Add(maximum, big.NewInt(1)), divisor)
	result := new(big.Int).Sub(ceilDiv(num, multiplier), big.NewInt(1))
	if result.Cmp(maxU256) > 0 {
		return new(big.Int).Set(maxU256)
	}
	return result
}

func wordFor(s *State, w priceWord) *big.Int {
	switch w {
	case priceWordV1:
		return s.V1
	case priceWordV2:
		return s.V2
	default:
		return s.V3
	}
}

// priceWad decodes a token's price as USD WAD per whole token, the "lower
// endpoint" used for priceIn and inventory/fee accounting.
func priceWad(s *State, t tokenMeta) (*big.Int, error) {
	encoded := bits(wordFor(s, t.Word), t.Shift, 48)
	mantissa := new(big.Int).And(encoded, big.NewInt((1<<40)-1))
	exponent := new(big.Int).Rsh(encoded, 40)
	if mantissa.Sign() <= 0 || exponent.Cmp(big.NewInt(18)) > 0 {
		return nil, ErrStalePrice
	}
	return new(big.Int).Mul(mantissa, pow10(18-exponent.Uint64())), nil
}

// priceOutWad advances the mantissa by one so packed precision can't
// increase taker output; used only for the output-side fair-value leg.
func priceOutWad(s *State, t tokenMeta) (*big.Int, error) {
	encoded := bits(wordFor(s, t.Word), t.Shift, 48)
	mantissa := new(big.Int).And(encoded, big.NewInt((1<<40)-1))
	exponent := new(big.Int).Rsh(encoded, 40)
	if mantissa.Cmp(big.NewInt(200000)) < 0 || exponent.Cmp(big.NewInt(18)) > 0 {
		return nil, ErrStalePrice
	}
	mantissa.Add(mantissa, big.NewInt(1))
	return new(big.Int).Mul(mantissa, pow10(18-exponent.Uint64())), nil
}

func pow10(n uint64) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), new(big.Int).SetUint64(n), nil)
}

// covarianceFieldByCategory maps an (left,right) category pair (left<=right)
// to its packed field index within r2 (fields 0-5) or r3 (fields 6-9).
var covarianceFieldByCategory = map[[2]int]int{
	{1, 1}: 0, {1, 2}: 1, {1, 3}: 2,
	{2, 2}: 3, {2, 3}: 4, {3, 3}: 5,
	{1, 4}: 6, {2, 4}: 7, {3, 4}: 8, {4, 4}: 9,
}

func covariance(s *State, left, right int) *big.Int {
	if left == 0 || right == 0 {
		return big.NewInt(0)
	}
	if left > right {
		left, right = right, left
	}
	field := covarianceFieldByCategory[[2]int{left, right}]
	if field < 6 {
		return signedN(bits(s.R2, uint(field)*16, 16), 16)
	}
	return signedN(bits(s.R3, uint(field-6)*16, 16), 16)
}

func exposure(s *State, category int) *big.Int {
	if category == 4 {
		return signedN(bits(s.I2, 0, 56), 56)
	}
	return signedN(bits(s.I1, uint(category)*56, 56), 56)
}

// anchorTokenIndexByCategory: WMON, WETH, WBTC, XAUt0 are categories 1-4's
// own anchor price source.
var anchorTokenIndexByCategory = map[int]int{1: 3, 2: 4, 3: 5, 4: 7}

func exposureGradient(s *State, category int) (*big.Int, error) {
	if category == 0 {
		return big.NewInt(0), nil
	}
	gradient := big.NewInt(0)
	for other := 1; other <= 4; other++ {
		anchorPrice, err := priceWad(s, tokenTable[anchorTokenIndexByCategory[other]])
		if err != nil {
			return nil, err
		}
		exposureUsdWad := trunc0(new(big.Int).Mul(exposure(s, other), anchorPrice), big.NewInt(exposureScale))
		gradient.Add(gradient, trunc0(new(big.Int).Mul(exposureUsdWad, covariance(s, category, other)), big.NewInt(covarianceScale)))
	}
	return gradient, nil
}

func penaltyUsdWad(s *State, tokenIn, tokenOut tokenMeta, notionalUsdWad *big.Int) (*big.Int, error) {
	lambdaQ := bits(s.R1, 110, 16)
	if lambdaQ.Sign() == 0 || tokenIn.Category == tokenOut.Category {
		return big.NewInt(0), nil
	}

	varianceRate := new(big.Int).Add(covariance(s, tokenIn.Category, tokenIn.Category), covariance(s, tokenOut.Category, tokenOut.Category))
	varianceRate.Sub(varianceRate, new(big.Int).Mul(big.NewInt(2), covariance(s, tokenIn.Category, tokenOut.Category)))

	delta := trunc0(new(big.Int).Mul(new(big.Int).Mul(notionalUsdWad, notionalUsdWad), varianceRate), pow10(27))

	gradIn, err := exposureGradient(s, tokenIn.Category)
	if err != nil {
		return nil, err
	}
	gradOut, err := exposureGradient(s, tokenOut.Category)
	if err != nil {
		return nil, err
	}
	term2 := trunc0(new(big.Int).Mul(big.NewInt(2), new(big.Int).Mul(notionalUsdWad, new(big.Int).Sub(gradIn, gradOut))), pow10(18))
	delta.Add(delta, term2)

	if delta.Sign() <= 0 {
		return big.NewInt(0), nil
	}
	return trunc0(new(big.Int).Mul(delta, lambdaQ), big.NewInt(lambdaScale)), nil
}

// pairIndex returns exactQuote's symmetric pair field index for two token
// indices, left<right required by the caller.
func pairIndex(left, right int) int {
	return left*(2*8-left-1)/2 + (right - left - 1)
}

// ExactQuote ports exactQuote() from THOGAMM_AGGREGATOR_INTEGRATION.md's
// in-memory mirror spec verbatim, operation order and all. Verified
// exact-to-the-wei against four live makerQuoteExactInput() fixtures at
// block 107066625 (see math_test.go) covering cross-category-with-penalty,
// same-category, and both XAUt0/index-7 directions.
//
// Every `require` in the pseudocode becomes a typed sentinel error here,
// not a hard failure: the pathfinder calls this at many sizes and treats
// "no quote at this size" as routine.
func ExactQuote(s *State, tokenIn, tokenOut tokenMeta, amountIn *big.Int, executionBlock uint64, fastLaneHot bool) (*big.Int, uint64, error) {
	if s.GloballyPaused {
		return nil, 0, ErrGloballyPaused
	}
	if bits(s.V1, 0, 1).Sign() != 0 {
		return nil, 0, ErrPriceSurfacePaused
	}
	if !s.PairRiskReady {
		return nil, 0, ErrPairRiskNotReady
	}
	if tokenIn.Index == tokenIdxXAUt || tokenOut.Index == tokenIdxXAUt {
		if !s.RiskV3Ready {
			return nil, 0, ErrRiskV3NotReady
		}
	}
	if amountIn.Sign() <= 0 || amountIn.Cmp(maxU256) > 0 {
		return nil, 0, ErrZeroNotional
	}
	if tokenIn.Index == tokenOut.Index {
		return nil, 0, ErrSameToken
	}

	sides := new(big.Int).Or(bits(s.V1, 1, 14), new(big.Int).Lsh(bits(s.V3, 48, 2), 14))
	sellBit := new(big.Int).Lsh(big.NewInt(1), uint(tokenIn.Index)*2)
	buyBit := new(big.Int).Lsh(big.NewInt(1), uint(tokenOut.Index)*2+1)
	if new(big.Int).And(sides, sellBit).Sign() == 0 || new(big.Int).And(sides, buyBit).Sign() == 0 {
		return nil, 0, ErrSideDisabled
	}

	postedBlock := bits(s.V1, 31, 32).Uint64()
	if postedBlock == 0 || executionBlock < postedBlock {
		return nil, 0, ErrStalePrice
	}
	age := executionBlock - postedBlock
	maxAge := bits(s.R1, 198, 8).Uint64()
	if maxAge == 0 {
		return nil, 0, ErrZeroMaxAge
	}
	if age > maxAge {
		return nil, 0, ErrStalePrice
	}

	inputPriceWad, err := priceWad(s, tokenIn)
	if err != nil {
		return nil, 0, err
	}
	outputPriceWad, err := priceOutWad(s, tokenOut)
	if err != nil {
		return nil, 0, err
	}
	inputScale := pow10(uint64(tokenIn.Decimals))
	outputScale := pow10(uint64(tokenOut.Decimals))

	maxNotional := maxFlooredInput(maxU256, outputPriceWad, outputScale)
	lambdaQ := bits(s.R1, 110, 16)
	if tokenIn.Category != tokenOut.Category && lambdaQ.Sign() != 0 {
		if maxRiskNotionalUsdWad.Cmp(maxNotional) < 0 {
			maxNotional = maxRiskNotionalUsdWad
		}
	}

	inputLimit := maxFlooredInput(maxNotional, inputScale, inputPriceWad)
	if amountIn.Cmp(inputLimit) > 0 {
		return nil, 0, ErrAmountTooLarge
	}

	notionalUsdWad := new(big.Int).Div(new(big.Int).Mul(amountIn, inputPriceWad), inputScale)
	if notionalUsdWad.Sign() <= 0 {
		return nil, 0, ErrZeroNotional
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
	var pairRaw *big.Int
	if pi < 25 {
		pairRaw = bits(s.P1, uint(pi)*10, 10)
	} else {
		pairRaw = bits(s.P2, uint(pi-25)*10, 10)
	}

	sum := new(big.Int).Add(bits(s.R1, categoryShiftIn, 10), bits(s.R1, categoryShiftOut, 10))
	sum.Add(sum, bits(s.R1, tokenShiftIn, 10))
	sum.Add(sum, bits(s.R1, tokenShiftOut, 10))
	sum.Add(sum, pairRaw)
	perBlockWidening := bits(s.R1, 190, 8)
	spreadUnits := new(big.Int).Mul(big.NewInt(4), sum)
	spreadUnits.Add(spreadUnits, new(big.Int).Mul(new(big.Int).SetUint64(age), perBlockWidening))
	if spreadUnits.Cmp(big.NewInt(spreadDenominator)) >= 0 {
		return nil, 0, ErrSpreadTooWide
	}

	fairOut := new(big.Int).Div(new(big.Int).Mul(notionalUsdWad, outputScale), outputPriceWad)
	amountOut := new(big.Int).Div(new(big.Int).Mul(fairOut, new(big.Int).Sub(big.NewInt(spreadDenominator), spreadUnits)), big.NewInt(spreadDenominator))

	penalty, err := penaltyUsdWad(s, tokenIn, tokenOut, notionalUsdWad)
	if err != nil {
		return nil, 0, err
	}
	if penalty.Sign() > 0 {
		outTokenPriceWad, err := priceWad(s, tokenOut)
		if err != nil {
			return nil, 0, err
		}
		penaltyOut := ceilDiv(new(big.Int).Mul(penalty, outputScale), outTokenPriceWad)
		if amountOut.Cmp(penaltyOut) <= 0 {
			return nil, 0, ErrPenaltyExceedsOut
		}
		amountOut.Sub(amountOut, penaltyOut)
	}

	if fastLaneHot {
		amountOut.Mul(amountOut, big.NewInt(10000-fastLaneFrictionBps))
		amountOut.Div(amountOut, big.NewInt(10000))
	}

	if amountOut.Sign() <= 0 {
		return nil, 0, ErrZeroAmountOut
	}
	if tokenOut.Index >= len(s.Balances) || amountOut.Cmp(s.Balances[tokenOut.Index]) > 0 {
		return nil, 0, ErrInsufficientBal
	}

	return amountOut, postedBlock, nil
}
