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

	q24  = new(uint256.Int).Lsh(u256.U1, 24)
	q48  = new(uint256.Int).Lsh(u256.U1, 48)
	q96  = new(uint256.Int).Lsh(u256.U1, 96)
	q120 = new(uint256.Int).Lsh(u256.U1, 120)

	maxTargetMultiplierU = uint256.NewInt(MaxTargetMultiplier)
)

var ErrInvalidFeeConfig = errors.New("stable-stable: invalid fee config")

func CalculatePriceRatioX96(sqrtPrice1X96, sqrtPrice2X96 *uint256.Int) *uint256.Int {
	num, den := sqrtPrice1X96, sqrtPrice2X96
	if num.Cmp(den) > 0 {
		num, den = den, num
	}
	r := u256.MulDiv(num, q48, den)
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

// applyDecay computes target + factor*(previous-target) >> 24, the tail shared
// by both hooks' calculateDecayingFee once factorX24 has been derived.
func applyDecay(targetFeeE12, previousDecayingFeeE12, factorX24 *uint256.Int) *uint256.Int {
	delta := new(uint256.Int).Sub(previousDecayingFeeE12, targetFeeE12)
	delta.Mul(delta, factorX24)
	delta.Rsh(delta, 24)
	return delta.Add(delta, targetFeeE12)
}

// CalculateDecayingFee is the legacy hook's formula: it uses the on-chain logK
// directly (see decayFactorX24). Newer hooks dropped logK; use
// CalculateDecayingFeeV2 for those (see deriveLogK).
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

	return applyDecay(targetFeeE12, previousDecayingFeeE12, factorX24), nil
}

func decayFactorX24(k, logK, blocksPassed uint64) (*uint256.Int, error) {
	if blocksPassed <= 4 {
		return fastPowQ24(k, blocksPassed)
	}

	mag := uint256.NewInt(logK)
	mag.Mul(mag, uint256.NewInt(blocksPassed))
	mag.Lsh(mag, 40)

	return expDecayFactorX24(mag)
}

// CalculateDecayingFeeV2 mirrors StableFeeCalculation.calculateDecayingFee on
// hooks that dropped the on-chain logK column: logK is derived from k on the
// fly (deriveLogK) instead of being read from chain.
func CalculateDecayingFeeV2(
	targetFeeE12, previousDecayingFeeE12 *uint256.Int,
	k, blocksPassed uint64,
) (*uint256.Int, error) {
	if previousDecayingFeeE12.Lt(targetFeeE12) {
		return nil, ErrInvalidFeeConfig
	}

	factorX24, err := decayFactorX24V2(k, blocksPassed)
	if err != nil {
		return nil, err
	}

	return applyDecay(targetFeeE12, previousDecayingFeeE12, factorX24), nil
}

func decayFactorX24V2(k, blocksPassed uint64) (*uint256.Int, error) {
	if blocksPassed <= 4 {
		return fastPowQ24(k, blocksPassed)
	}

	logK, err := deriveLogK(k)
	if err != nil {
		return nil, err
	}

	mag := logK.Mul(logK, uint256.NewInt(blocksPassed))
	mag.Lsh(mag, 24)

	return expDecayFactorX24(mag)
}

// expDecayFactorX24 computes floor(exp(-magWad) * 2^24 / 1e18), the tail
// shared by decayFactorX24 and decayFactorX24V2 once each has assembled its
// (differently-scaled) wad exponent magnitude.
func expDecayFactorX24(magWad *uint256.Int) (*uint256.Int, error) {
	expI := i256.SafeToInt256(magWad)
	if expI == nil {
		return nil, ErrInvalidFeeConfig
	}
	expI = i256.Neg(expI)

	expWad, err := bunnimath.ExpWad(expI)
	if err != nil {
		return nil, err
	}

	return u256.MulDiv(expWad, q24, oneE18), nil
}

// deriveLogK mirrors StableFeeCalculation.deriveLogK: logK = ceil(-ln(k) / 2^24),
// where k is Q24 fixed point and the result is the wad-scale (1e18) decay rate
// decayFactorX24V2's slow path exponentiates with. This is specific to hooks
// that no longer store logK on-chain — it is NOT interchangeable with the
// legacy hook's on-chain logK, which uses a different (<<40, not <<24) scale.
func deriveLogK(k uint64) (*uint256.Int, error) {
	kQ96 := new(uint256.Int).Lsh(uint256.NewInt(k), 72)
	kQ96Int := i256.SafeToInt256(kQ96)
	if kQ96Int == nil {
		return nil, ErrInvalidFeeConfig
	}

	lnKQ96, err := bunnimath.LnQ96(kQ96Int)
	if err != nil {
		return nil, err
	}

	absLnKQ96 := i256.UnsafeToUInt256(i256.Abs(lnKQ96))

	return bunnimath.MulDivUp(absLnKQ96, oneE18, q120), nil
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
