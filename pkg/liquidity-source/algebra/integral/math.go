package integral

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func getInputTokenDelta01(to, from, liquidity *big.Int) (*big.Int, error) {
	return getToken0Delta(to, from, liquidity, true)
}

func getInputTokenDelta10(to, from, liquidity *big.Int) (*big.Int, error) {
	return getToken1Delta(from, to, liquidity, true)
}

func getOutputTokenDelta01(to, from, liquidity *big.Int) (*big.Int, error) {
	return getToken1Delta(to, from, liquidity, false)
}

func getOutputTokenDelta10(to, from, liquidity *big.Int) (*big.Int, error) {
	return getToken0Delta(from, to, liquidity, false)
}

func getToken0Delta(priceLower, priceUpper, liquidity *big.Int, roundUp bool) (*big.Int, error) {
	priceDelta := new(big.Int).Sub(priceUpper, priceLower)
	if priceDelta.Cmp(priceUpper) >= 0 {
		return nil, errors.New("price delta must be greater than price upper")
	}

	liquidityShifted := new(big.Int).Lsh(liquidity, RESOLUTION)

	if roundUp {
		mulDivResult, err := mulDivRoundingUp(priceDelta, liquidityShifted, priceUpper)
		if err != nil {
			return nil, err
		}

		token0Delta, err := unsafeDivRoundingUp(mulDivResult, priceLower)
		if err != nil {
			return nil, err
		}

		return token0Delta, nil
	} else {
		mulDivResult, err := mulDiv(priceDelta, liquidityShifted, priceUpper)
		if err != nil {
			return nil, err
		}

		token0Delta := new(big.Int).Div(mulDivResult, priceLower)

		return token0Delta, nil
	}
}

func getToken1Delta(priceLower, priceUpper, liquidity *big.Int, roundUp bool) (*big.Int, error) {
	if priceUpper.Cmp(priceLower) < 0 {
		return nil, errors.New("price upper must be greater than price lower")
	}

	priceDelta := new(big.Int).Sub(priceUpper, priceLower)

	if roundUp {
		token1Delta, err := mulDivRoundingUp(priceDelta, liquidity, Q96)
		if err != nil {
			return nil, err
		}

		return token1Delta, nil
	} else {
		token1Delta, err := mulDiv(priceDelta, liquidity, Q96)
		if err != nil {
			return nil, err
		}

		return token1Delta, nil
	}
}

func getNewPriceAfterInput(price, liquidity, input *big.Int, zeroToOne bool) (*big.Int, error) {
	return getNewPrice(price, liquidity, input, zeroToOne, true)
}

func getNewPriceAfterOutput(price, liquidity, output *big.Int, zeroToOne bool) (*big.Int, error) {
	return getNewPrice(price, liquidity, output, zeroToOne, false)
}

func getNewPrice(
	price, liquidity *big.Int,
	amount *big.Int,
	zeroToOne, fromInput bool,
) (*big.Int, error) {
	if price.Sign() == 0 {
		return nil, fmt.Errorf("price cannot be zero")
	}
	if liquidity.Sign() == 0 {
		return nil, fmt.Errorf("liquidity cannot be zero")
	}
	if amount.Sign() == 0 {

		return new(big.Int).Set(price), nil
	}

	liquidityShifted := new(big.Int).Lsh(liquidity, RESOLUTION)

	if zeroToOne == fromInput {
		if fromInput {
			product := new(big.Int).Mul(amount, price)
			if new(big.Int).Div(product, amount).Cmp(price) != 0 {
				return nil, fmt.Errorf("product overflow")
			}

			denominator := new(big.Int).Add(liquidityShifted, product)
			if denominator.Cmp(liquidityShifted) < 0 {
				return nil, fmt.Errorf("denominator underflow")
			}

			resultPrice, err := mulDivRoundingUp(liquidityShifted, price, denominator)
			if err != nil {
				return nil, err
			}
			if resultPrice.BitLen() > 160 {
				return nil, fmt.Errorf("resulting price exceeds 160 bits")
			}
			return resultPrice, nil
		} else {
			product := new(big.Int).Mul(amount, price)
			if new(big.Int).Div(product, amount).Cmp(price) != 0 {
				return nil, fmt.Errorf("product overflow")
			}
			if liquidityShifted.Cmp(product) <= 0 {
				return nil, fmt.Errorf("denominator underflow")
			}

			denominator := new(big.Int).Sub(liquidityShifted, product)
			resultPrice, err := mulDivRoundingUp(liquidityShifted, price, denominator)
			if err != nil {
				return nil, err
			}
			if resultPrice.BitLen() > 160 {
				return nil, fmt.Errorf("resulting price exceeds 160 bits")
			}
			return resultPrice, nil
		}
	} else {
		if fromInput {
			shiftedAmount := new(big.Int)
			var err error
			if amount.Cmp(new(big.Int).Sub(new(big.Int).Lsh(bignumber.One, 160), bignumber.One)) <= 0 {
				shiftedAmount = new(big.Int).Lsh(amount, RESOLUTION)
				shiftedAmount.Div(shiftedAmount, liquidity)
			} else {
				shiftedAmount, err = mulDiv(amount, new(big.Int).Lsh(bignumber.One, RESOLUTION), liquidity)
				if err != nil {
					return nil, err
				}
			}

			resultPrice := new(big.Int).Add(price, shiftedAmount)
			if resultPrice.BitLen() > 160 {
				return nil, fmt.Errorf("resulting price exceeds 160 bits")
			}
			return resultPrice, nil
		} else {
			var quotient *big.Int
			var err error
			if amount.Cmp(new(big.Int).Sub(new(big.Int).Lsh(bignumber.One, 160), bignumber.One)) <= 0 {
				shiftedAmount := new(big.Int).Lsh(amount, RESOLUTION)
				quotient, err = unsafeDivRoundingUp(shiftedAmount, liquidity)
				if err != nil {
					return nil, err
				}
			} else {
				quotient, err = mulDivRoundingUp(amount, new(big.Int).Lsh(bignumber.One, RESOLUTION), liquidity)
				if err != nil {
					return nil, err
				}
			}

			if price.Cmp(quotient) <= 0 {
				return nil, fmt.Errorf("price must be greater than quotient")
			}

			resultPrice := new(big.Int).Sub(price, quotient)
			if resultPrice.BitLen() > 160 {
				return nil, fmt.Errorf("resulting price exceeds 160 bits")
			}
			return resultPrice, nil
		}
	}
}

func unsafeDivRoundingUp(x, y *big.Int) (*big.Int, error) {
	if y.Sign() == 0 {
		return nil, errors.New("division by zero")
	}

	quotient := new(big.Int).Div(x, y)

	remainder := new(big.Int).Mod(x, y)
	if remainder.Sign() > 0 {
		quotient.Add(quotient, bignumber.One)
	}

	return quotient, nil
}

func mulDivRoundingUp(a, b, denominator *big.Int) (*big.Int, error) {
	if denominator.Sign() == 0 {
		return nil, errors.New("division by zero")
	}

	product := new(big.Int).Mul(a, b)
	if a.Sign() == 0 || new(big.Int).Div(product, a).Cmp(b) == 0 {
		quotient := new(big.Int).Div(product, denominator)
		remainder := new(big.Int).Mod(product, denominator)
		if remainder.Sign() > 0 {
			quotient.Add(quotient, bignumber.One)
		}
		return quotient, nil
	}

	mulMod := new(big.Int).Mod(product, denominator)
	quotient := new(big.Int).Div(product, denominator)
	if mulMod.Sign() > 0 {
		if quotient.Cmp(MAX_UINT256) >= 0 {
			return nil, errors.New("overflow")
		}
		quotient.Add(quotient, bignumber.One)
	}

	return quotient, nil
}

func mulDiv(a, b, denominator *big.Int) (*big.Int, error) {
	if denominator.Sign() == 0 {
		return nil, errors.New("denominator must be greater than zero")
	}

	prod0 := new(big.Int).Mul(a, b)
	prod1 := new(big.Int)

	mulMod := new(big.Int).Mod(new(big.Int).Mul(a, b), new(big.Int).SetUint64(^uint64(0)))
	prod1 = new(big.Int).Sub(new(big.Int).Sub(mulMod, prod0), bignumber.ZeroBI)
	if mulMod.Cmp(prod0) < 0 {
		prod1.Sub(prod1, bignumber.One)
	}

	if denominator.Cmp(prod1) <= 0 {
		return nil, errors.New("denominator must be greater than prod1")
	}

	if prod1.Sign() == 0 {
		return new(big.Int).Div(prod0, denominator), nil
	}

	remainder := new(big.Int).Mod(new(big.Int).Mul(a, b), denominator)
	prod1.Sub(prod1, bignumber.ZeroBI)
	if remainder.Cmp(prod0) > 0 {
		prod1.Sub(prod1, bignumber.One)
	}
	prod0.Sub(prod0, remainder)

	twos := new(big.Int).And(new(big.Int).Neg(denominator), denominator)
	denominator.Div(denominator, twos)

	prod0.Div(prod0, twos)

	twosComplement := new(big.Int).Add(new(big.Int).Div(new(big.Int).Sub(bignumber.ZeroBI, twos), twos), bignumber.One)
	prod0.Or(prod0, new(big.Int).Mul(prod1, twosComplement))

	inv := new(big.Int).Mul(denominator, bignumber.Three)
	inv.Xor(inv, bignumber.Two)
	for i := 0; i < 6; i++ {
		inv.Mul(inv, new(big.Int).Sub(bignumber.Two, new(big.Int).Mul(denominator, inv)))
	}

	result := new(big.Int).Mul(prod0, inv)
	result.Mod(result, new(big.Int).Lsh(bignumber.One, 256))

	return result, nil
}

func addDelta(x *big.Int, y *big.Int) (*big.Int, error) {
	if y.Sign() < 0 {
		absY := new(big.Int).Abs(y)
		result := new(big.Int).Sub(x, absY)

		if result.Cmp(x) >= 0 {
			return nil, fmt.Errorf("liquiditySub: underflow error")
		}

		return result, nil
	} else {
		result := new(big.Int).Add(x, y)

		if result.Cmp(x) < 0 {
			return nil, fmt.Errorf("liquidityAdd: overflow error")
		}

		return result, nil
	}
}
