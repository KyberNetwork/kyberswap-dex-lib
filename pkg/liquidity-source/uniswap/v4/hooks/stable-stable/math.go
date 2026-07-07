package stablestable

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/i256"
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"

	bunnimath "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/math"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	MaxOptimalFeeE6     uint64 = 1e4
	MaxTargetMultiplier uint64 = 100
)

var (
	oneE6  = u256.TenPow(6)
	oneE12 = u256.TenPow(12)
	oneE18 = u256.TenPow(18)

	undefinedDecayingFeeE12 = new(uint256.Int).AddUint64(oneE12, 1)

	q48 = new(uint256.Int).Lsh(u256.U1, 48)
	q96 = new(uint256.Int).Lsh(u256.U1, 96)

	maxTargetMultiplierU = uint256.NewInt(MaxTargetMultiplier)
)

var ErrInvalidFeeConfig = errors.New("stable-stable: invalid fee config")

func CalculatePriceRatioX96(sqrtPrice1X96, sqrtPrice2X96 *uint256.Int) *uint256.Int {
	num, den := sqrtPrice1X96, sqrtPrice2X96
	if num.Cmp(den) > 0 {
		num, den = den, num
	}
	r := new(uint256.Int).Mul(num, q48)
	r.Div(r, den)
	return r.Mul(r, r)
}

func CalculateCloseBoundaryFee(priceRatioX96 *uint256.Int, optimalFeeE6 uint64) (magnitudeE12 *uint256.Int, isOutside bool) {
	sub := new(uint256.Int).Mul(oneE12, priceRatioX96)
	sub.MulDivOverflow(sub, oneE6, uint256.NewInt(1e6-optimalFeeE6))
	sub.Rsh(sub, 96)

	if sub.Cmp(oneE12) >= 0 {
		return new(uint256.Int).Sub(sub, oneE12), false
	}
	return new(uint256.Int).Sub(oneE12, sub), true
}

func CalculateInsideOptimalRangeFee(
	priceRatioX96 *uint256.Int, optimalFeeE6 uint64,
	ammPriceBelowRP, userSellsZeroForOne bool,
) (*uint256.Int, error) {
	oneMinusOptE6 := uint256.NewInt(1e6 - optimalFeeE6)

	subE12 := new(uint256.Int).Mul(oneE12, oneMinusOptE6)
	if ammPriceBelowRP == userSellsZeroForOne {
		subE12.MulDivOverflow(subE12, q96, priceRatioX96)
		subE12.Div(subE12, oneE6)
	} else {
		subE12.Mul(subE12, priceRatioX96)
		subE12.Rsh(subE12, 96)
		subE12.Div(subE12, oneE6)
	}

	if subE12.Gt(oneE12) {
		return nil, ErrInvalidFeeConfig
	}

	return new(uint256.Int).Sub(oneE12, subE12), nil
}

func CalculateFarBoundaryFee(priceRatioX96 *uint256.Int, optimalFeeE6 uint64) *uint256.Int {
	num := new(uint256.Int).Mul(oneE12, uint256.NewInt(1e6-optimalFeeE6))
	num.Mul(num, priceRatioX96)
	num.Rsh(num, 96)
	num.Div(num, oneE6)

	return new(uint256.Int).Sub(oneE12, num)
}

func AdjustPreviousFeeForPriceMovement(priceRatioX96, previousDecayingFeeE12 *uint256.Int) *uint256.Int {
	oneMinusPrev := new(uint256.Int).Sub(oneE12, previousDecayingFeeE12)
	num := new(uint256.Int).Mul(priceRatioX96, oneMinusPrev)
	num.Rsh(num, 96)
	return new(uint256.Int).Sub(oneE12, num)
}

func CalculateDecayingFee(
	targetFeeE12, previousDecayingFeeE12 *uint256.Int,
	k, logK, blocksPassed uint64,
) (*uint256.Int, error) {
	if previousDecayingFeeE12.Lt(targetFeeE12) {
		return nil, ErrInvalidFeeConfig
	}

	factorX24, err := decayFactorX24(k, logK, blocksPassed)
	if err != nil {
		return nil, err
	}

	delta := new(uint256.Int).Sub(previousDecayingFeeE12, targetFeeE12)
	delta.Mul(delta, factorX24)
	delta.Rsh(delta, 24)

	return new(uint256.Int).Add(targetFeeE12, delta), nil
}

func decayFactorX24(k, logK, blocksPassed uint64) (*uint256.Int, error) {
	if blocksPassed <= 4 {
		return fastPowQ24(k, blocksPassed)
	}

	mag := uint256.NewInt(logK)
	mag.Mul(mag, uint256.NewInt(blocksPassed))
	mag.Lsh(mag, 40)

	expI := i256.SafeToInt256(mag)
	if expI == nil {
		return nil, ErrInvalidFeeConfig
	}
	expI = i256.Neg(expI)

	expWad, err := bunnimath.ExpWad(expI)
	if err != nil {
		return nil, err
	}

	out := new(uint256.Int).Lsh(expWad, 24)
	out.Div(out, oneE18)

	return out, nil
}

func fastPowQ24(k, n uint64) (*uint256.Int, error) {
	kU := uint256.NewInt(k)
	switch n {
	case 0:
		return new(uint256.Int).Lsh(u256.U1, 24), nil
	case 1:
		return kU, nil
	case 2:
		r := new(uint256.Int).Mul(kU, kU)
		return r.Rsh(r, 24), nil
	case 3:
		zz := new(uint256.Int).Mul(kU, kU)
		r := new(uint256.Int).Mul(kU, zz)
		return r.Rsh(r, 48), nil
	case 4:
		zz := new(uint256.Int).Mul(kU, kU)
		r := new(uint256.Int).Mul(zz, zz)
		return r.Rsh(r, 72), nil
	default:
		return nil, ErrInvalidFeeConfig
	}
}

var _ = (*int256.Int)(nil)
