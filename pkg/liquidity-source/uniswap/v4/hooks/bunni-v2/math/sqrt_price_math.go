package math

import (
	"errors"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

func GetTickAtSqrtPrice(sqrtPriceX96 *uint256.Int) (int, error) {
	return v3Utils.GetTickAtSqrtRatioV2(sqrtPriceX96)
}

func GetSqrtPriceAtTick(tick int) (*uint256.Int, error) {
	var price uint256.Int
	err := v3Utils.GetSqrtRatioAtTickV2(tick, &price)
	return &price, err
}

func GetSqrtPriceTarget(zeroForOne bool, sqrtPriceNextX96, sqrtPriceLimitX96 *uint256.Int) *uint256.Int {
	var result uint256.Int

	if zeroForOne {
		result.Set(u256.Max(sqrtPriceNextX96, sqrtPriceLimitX96))
	} else {
		result.Set(u256.Min(sqrtPriceNextX96, sqrtPriceLimitX96))
	}

	return &result
}

func GetNextSqrtPriceFromInput(sqrtPX96, liquidity, amountIn *uint256.Int, zeroForOne bool) (*uint256.Int, error) {
	var price uint256.Int
	err := v3Utils.GetNextSqrtPriceFromInput(sqrtPX96, liquidity, amountIn, zeroForOne, &price)
	return &price, err
}

func GetNextSqrtPriceFromOutput(sqrtPX96, liquidity, amountOut *uint256.Int, zeroForOne bool) (*uint256.Int, error) {
	var price uint256.Int
	err := v3Utils.GetNextSqrtPriceFromOutput(sqrtPX96, liquidity, amountOut, zeroForOne, &price)
	return &price, err
}

func GetAmount0Delta(sqrtPriceAX96, sqrtPriceBX96, liquidity *uint256.Int, roundUp bool) (*uint256.Int, error) {
	var result uint256.Int
	err := v3Utils.GetAmount0DeltaV2(sqrtPriceAX96, sqrtPriceBX96, liquidity, roundUp, &result)
	return &result, err
}

func GetAmount1Delta(sqrtPriceAX96, sqrtPriceBX96, liquidity *uint256.Int, roundUp bool) (*uint256.Int, error) {
	var result uint256.Int
	err := v3Utils.GetAmount1DeltaV2(sqrtPriceAX96, sqrtPriceBX96, liquidity, roundUp, &result)
	return &result, err
}

func GetNextSqrtPriceFromAmount0RoundingUp(sqrtPX96, liquidity, amount *uint256.Int, add bool) (*uint256.Int, error) {
	if amount.IsZero() {
		return sqrtPX96.Clone(), nil
	}

	var numerator1 uint256.Int
	numerator1.Lsh(liquidity, 96)

	if add {
		var res uint256.Int
		res.Mul(amount, sqrtPX96)

		var temp uint256.Int
		temp.Div(&res, amount)
		if temp.Eq(sqrtPX96) {
			temp.Add(&numerator1, &res)

			if temp.Cmp(&numerator1) >= 0 {
				return FullMulDivUp(&numerator1, sqrtPX96, &temp)
			}
		}

		temp.Div(&numerator1, sqrtPX96)
		temp.Add(&temp, amount)

		res.Clear()
		v3Utils.DivRoundingUp(&res, &numerator1, &temp)
		return &res, nil

	} else {
		var res uint256.Int
		res.Mul(amount, sqrtPX96)

		var temp uint256.Int
		temp.Div(&res, amount)
		if !temp.Eq(sqrtPX96) || numerator1.Cmp(&res) <= 0 {
			return nil, ErrOverflow
		}

		temp.Sub(&numerator1, &res)

		return FullMulDivUp(&numerator1, sqrtPX96, &temp)
	}
}

func GetNextSqrtPriceFromAmount1RoundingDown(sqrtPX96, liquidity, amount *uint256.Int, add bool) (*uint256.Int, error) {
	var res uint256.Int

	if add {
		if amount.Cmp(v3Utils.MaxUint160) <= 0 {
			res.Lsh(amount, 96)
			res.Div(&res, liquidity)
		} else {
			result, err := FullMulDiv(amount, Q96, liquidity)
			if err != nil {
				return nil, err
			}
			res = *result
		}

		res.Add(sqrtPX96, &res)
		return &res, nil
	} else {
		if amount.Cmp(v3Utils.MaxUint160) <= 0 {
			var temp uint256.Int
			temp.Lsh(amount, 96)
			v3Utils.DivRoundingUp(&res, &temp, liquidity)
		} else {
			result, err := FullMulDivUp(amount, Q96, liquidity)
			if err != nil {
				return nil, err
			}
			res = *result
		}

		if sqrtPX96.Cmp(&res) <= 0 {
			return nil, errors.New("NotEnoughLiquidity")
		}

		res.Sub(sqrtPX96, &res)
		return &res, nil
	}
}
