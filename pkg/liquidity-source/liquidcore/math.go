package liquidcore

import (
	"errors"

	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	uScale     = uint256.NewInt(100_000)
	uBalanced  = uint256.NewInt(50_000)
	uBaseFee   = uint256.NewInt(25)
	uMinFee    = uint256.NewInt(1)
	uMaxDevBps = uint256.NewInt(2500)
	uDevDenom  = uint256.NewInt(10_000)
	u6969      = uint256.NewInt(6969)

	ErrZeroAmount             = errors.New("zero amount")
	ErrPriceDeviationTooLarge = errors.New("price deviation too large")
	ErrSpotPriceZero          = errors.New("spot price is zero")
)

type PoolState struct {
	Token0    string
	Token1    string
	Decimals0 uint8
	Decimals1 uint8
	Reserve0  *uint256.Int
	Reserve1  *uint256.Int

	SpotPrice   *uint256.Int
	OraclePrice *uint256.Int
}

type SwapResult struct {
	AmountOut   *uint256.Int
	FeeAmount   *uint256.Int
	FeeBps      *uint256.Int
	NewReserve0 *uint256.Int
	NewReserve1 *uint256.Int
}

func InvertPrice(spotPrice *uint256.Int) (*uint256.Int, error) {
	if spotPrice.IsZero() {
		return nil, ErrSpotPriceZero
	}

	return new(uint256.Int).Div(u256.TenPow(12), spotPrice), nil
}

func CheckPriceDeviation(spotPrice, oraclePrice *uint256.Int) error {
	var diff uint256.Int
	if spotPrice.Cmp(oraclePrice) >= 0 {
		diff.Sub(spotPrice, oraclePrice)
	} else {
		diff.Sub(oraclePrice, spotPrice)
	}

	maxAllowed, _ := new(uint256.Int).MulDivOverflow(oraclePrice, uMaxDevBps, uDevDenom)
	if diff.Gt(maxAllowed) {
		return ErrPriceDeviationTooLarge
	}

	return nil
}

func NormalizeWithPrice(amount, spotPrice *uint256.Int, decimals uint8) (*uint256.Int, error) {
	if spotPrice.IsZero() {
		return nil, ErrSpotPriceZero
	}

	scaled := new(uint256.Int).Mul(amount, u256.TenPow(6))
	scaled.Mul(scaled, u256.TenPow(18))
	scaled.Div(scaled, spotPrice)
	scaled.Div(scaled, u256.TenPow(decimals))

	return scaled, nil
}

func NormalizeNoPrice(amount *uint256.Int, decimals uint8) *uint256.Int {
	scaled, _ := new(uint256.Int).MulDivOverflow(amount, u256.TenPow(18), u256.TenPow(decimals))

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
		sum := new(uint256.Int).Add(part1, part2)

		return sum.Div(sum, totalImb)
	}

	if imbAfter.Cmp(imbBefore) <= 0 {
		return new(uint256.Int).Set(uBaseFee)
	}

	return DynamicFee(imbAfter)
}

func CalcSwap(pool *PoolState, fromToken, toToken string, amountIn *uint256.Int) (*SwapResult, error) {
	if amountIn.IsZero() {
		return nil, ErrZeroAmount
	}

	if pool.SpotPrice.IsZero() {
		return nil, ErrSpotPriceZero
	}

	if err := CheckPriceDeviation(
		pool.SpotPrice,
		new(uint256.Int).Mul(pool.OraclePrice, u256.U100)); err != nil {
		return nil, err
	}

	isFromToken0 := fromToken == pool.Token0

	var price *uint256.Int
	if isFromToken0 {
		price = new(uint256.Int).Set(pool.SpotPrice)
	} else {
		var err error
		price, err = InvertPrice(pool.SpotPrice)
		if err != nil {
			return nil, err
		}
	}

	dec0 := pool.Decimals0
	dec1 := pool.Decimals1

	var decInScale, decOutScale *uint256.Int
	if isFromToken0 {
		decInScale = u256.TenPow(dec0)
		decOutScale = u256.TenPow(dec1)
	} else {
		decInScale = u256.TenPow(dec1)
		decOutScale = u256.TenPow(dec0)
	}

	calcRawOut := func() *uint256.Int {
		raw := new(uint256.Int).Mul(amountIn, u256.TenPow(6))
		raw.Mul(raw, decOutScale)
		denom := new(uint256.Int).Mul(price, decInScale)
		raw.Div(raw, denom)
		return raw
	}

	feeBps := new(uint256.Int).Set(uBaseFee)
	normPrice := pool.SpotPrice

	for iteration := 0; iteration < 20; iteration++ {
		rawOut := calcRawOut()

		reserve0 := new(uint256.Int).Set(pool.Reserve0)
		reserve1 := new(uint256.Int).Set(pool.Reserve1)

		var newReserve0, newReserve1 *uint256.Int
		if isFromToken0 {
			newReserve0 = new(uint256.Int).Add(reserve0, amountIn)
			if reserve1.Lt(rawOut) {
				return nil, ErrInsufficientReserve
			}
			newReserve1 = new(uint256.Int).Sub(reserve1, rawOut)
		} else {
			if reserve0.Lt(rawOut) {
				return nil, ErrInsufficientReserve
			}
			newReserve0 = new(uint256.Int).Sub(reserve0, rawOut)
			newReserve1 = new(uint256.Int).Add(reserve1, amountIn)
		}

		val0Before, err := NormalizeWithPrice(reserve0, normPrice, dec0)
		if err != nil {
			return nil, err
		}
		val1Before := NormalizeNoPrice(reserve1, dec1)
		totalBefore := new(uint256.Int).Add(val0Before, val1Before)

		val0After, err := NormalizeWithPrice(newReserve0, normPrice, dec0)
		if err != nil {
			return nil, err
		}
		val1After := NormalizeNoPrice(newReserve1, dec1)
		totalAfter := new(uint256.Int).Add(val0After, val1After)

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

	rawOut := calcRawOut()
	feePortionOut := new(uint256.Int).Mul(rawOut, feeBps)
	feePortionOut.Div(feePortionOut, uScale)
	netOut := new(uint256.Int).Sub(rawOut, feePortionOut)

	var newR0, newR1 *uint256.Int
	if isFromToken0 {
		newR0 = new(uint256.Int).Add(pool.Reserve0, amountIn)
		newR1 = new(uint256.Int).Sub(pool.Reserve1, rawOut)
	} else {
		newR0 = new(uint256.Int).Sub(pool.Reserve0, rawOut)
		newR1 = new(uint256.Int).Add(pool.Reserve1, amountIn)
	}

	return &SwapResult{
		AmountOut:   netOut,
		FeeAmount:   feePortionOut,
		FeeBps:      new(uint256.Int).Set(feeBps),
		NewReserve0: newR0,
		NewReserve1: newR1,
	}, nil
}
