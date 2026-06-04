package liquidcore

import (
	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolState struct {
	Token0    string
	Decimals0 uint8
	Decimals1 uint8
	Reserve0  *uint256.Int
	Reserve1  *uint256.Int

	SpotPrice *uint256.Int
}

type SwapResult struct {
	AmountOut   *uint256.Int
	FeeAmount   *uint256.Int
	NewReserve0 *uint256.Int
	NewReserve1 *uint256.Int
}

func NormalizeWithPrice(amount, spotPrice, scale *uint256.Int, decimals uint8) (*uint256.Int, error) {
	if spotPrice.IsZero() {
		return nil, ErrSpotPriceZero
	}

	scaled := new(uint256.Int).Mul(amount, scale)
	scaled.Mul(scaled, u1e18)
	scaled.Div(scaled, spotPrice)
	scaled.Div(scaled, u256.TenPow(decimals))

	return scaled, nil
}

func NormalizeNoPrice(amount *uint256.Int, decimals uint8) *uint256.Int {
	scaled, _ := new(uint256.Int).MulDivOverflow(amount, u1e18, u256.TenPow(decimals))

	return scaled
}

func Imbalance(weight *uint256.Int) *uint256.Int {
	if weight.Cmp(uBalanced) <= 0 {
		return new(uint256.Int).Sub(uBalanced, weight)
	}

	return new(uint256.Int).Sub(weight, uBalanced)
}

func DynamicFee(d *uint256.Int) *uint256.Int {
	if d.IsZero() {
		return new(uint256.Int).Set(uBaseFee)
	}

	fee, _ := new(uint256.Int).MulDivOverflow(d, d, uScale)
	fee.MulDivOverflow(fee, u6969, uScale)

	fee.Add(uBaseFee, fee)

	if fee.Lt(uMinFee) {
		fee.Set(uMinFee)
	}

	return fee
}

func CalcWeight(val0, totalVal *uint256.Int) *uint256.Int {
	if totalVal.IsZero() {
		return new(uint256.Int)
	}

	w, _ := new(uint256.Int).MulDivOverflow(val0, uScale, totalVal)

	return w
}

func BlendFee(wBefore, wAfter, imbBefore, imbAfter *uint256.Int) *uint256.Int {
	beforeAbove := wBefore.Gt(uBalanced)
	afterAbove := wAfter.Gt(uBalanced)

	crosses := (beforeAbove && !afterAbove) || (!beforeAbove && afterAbove)

	if crosses {
		totalImb := new(uint256.Int).Add(imbBefore, imbAfter)
		if totalImb.IsZero() {
			return new(uint256.Int).Set(uBaseFee)
		}

		dynFee := DynamicFee(imbAfter)
		part1 := new(uint256.Int).Mul(imbBefore, uBaseFee)
		part2 := new(uint256.Int).Mul(imbAfter, dynFee)
		part1.Add(part1, part2)

		return part1.Div(part1, totalImb)
	}

	if imbAfter.Cmp(imbBefore) <= 0 {
		return new(uint256.Int).Set(uBaseFee)
	}

	return DynamicFee(imbAfter)
}

func CalcSwap(pool *PoolState, fromToken string, amountIn *uint256.Int) (*SwapResult, error) {
	if amountIn.IsZero() {
		return nil, ErrZeroAmount
	} else if pool.SpotPrice.IsZero() {
		return nil, ErrSpotPriceZero
	}

	isFromToken0 := fromToken == pool.Token0
	dec0, dec1 := pool.Decimals0, pool.Decimals1
	var decInScale, decOutScale *uint256.Int
	if isFromToken0 {
		decInScale = u256.TenPow(dec0)
		decOutScale = u256.TenPow(dec1)
	} else {
		decInScale = u256.TenPow(dec1)
		decOutScale = u256.TenPow(dec0)
	}

	// scale = 1e10 for tokens with sum(decs) > 25, 1e6 otherwise
	scale := u1e6
	if dec0+dec1 > 25 {
		scale = u1e10
	}

	var rawOut, tmp uint256.Int
	if isFromToken0 { // amtIn * decOut/decIn * scale/price
		u256.MulDivDown(&rawOut, rawOut.Mul(amountIn, decOutScale), scale, tmp.Mul(pool.SpotPrice, decInScale))
	} else { // amtIn * decOut/decIn * price/scale
		u256.MulDivDown(&rawOut, rawOut.Mul(amountIn, decOutScale), pool.SpotPrice, tmp.Mul(scale, decInScale))
	}

	feeBps := tmp.Set(uBaseFee)
	normPrice := pool.SpotPrice

	var reserve0, reserve1, newReserve0, newReserve1 uint256.Int
	reserve0.Set(pool.Reserve0)
	reserve1.Set(pool.Reserve1)

	if isFromToken0 {
		newReserve0.Add(&reserve0, amountIn)
		if reserve1.Lt(&rawOut) {
			return nil, ErrInsufficientReserve
		}
		newReserve1.Sub(&reserve1, &rawOut)
	} else {
		if reserve0.Lt(&rawOut) {
			return nil, ErrInsufficientReserve
		}
		newReserve0.Sub(&reserve0, &rawOut)
		newReserve1.Add(&reserve1, amountIn)
	}

	for range 20 {
		val0Before, err := NormalizeWithPrice(&reserve0, normPrice, scale, dec0)
		if err != nil {
			return nil, err
		}
		val1Before := NormalizeNoPrice(&reserve1, dec1)
		totalBefore := val1Before.Add(val0Before, val1Before)

		val0After, err := NormalizeWithPrice(&newReserve0, normPrice, scale, dec0)
		if err != nil {
			return nil, err
		}
		val1After := NormalizeNoPrice(&newReserve1, dec1)
		totalAfter := val1After.Add(val0After, val1After)

		if totalBefore.IsZero() || totalAfter.IsZero() {
			feeBps.Set(uBaseFee)
			break
		}

		wBefore := CalcWeight(val0Before, totalBefore)
		wAfter := CalcWeight(val0After, totalAfter)
		imbBefore := Imbalance(wBefore)
		imbAfter := Imbalance(wAfter)

		newFee := BlendFee(wBefore, wAfter, imbBefore, imbAfter)
		if newFee.Lt(uMinFee) {
			newFee.Set(uMinFee)
		}

		if newFee.Eq(feeBps) {
			break
		}

		feeBps.Set(newFee)
	}

	feePortionOut := tmp.Mul(&rawOut, feeBps)
	feePortionOut.Div(feePortionOut, uScale)
	netOut := rawOut.Sub(&rawOut, feePortionOut)

	return &SwapResult{
		AmountOut:   netOut,
		FeeAmount:   feePortionOut,
		NewReserve0: &newReserve0,
		NewReserve1: &newReserve1,
	}, nil
}
