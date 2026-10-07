package ilyris

import (
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// Read-only constants; never use them as a receiver.
var (
	scale            = big256.TenPow(18) // BinMath.SCALE
	binStepScale     = big256.TenPow(14)
	feePrecisionU    = big256.TenPow(9)
	variableFeeScale = big256.TenPow(11) // VARIABLE_FEE_SCALE
	maxUint128       = new(uint256.Int).SubUint64(new(uint256.Int).Lsh(uint256.NewInt(1), 128), 1)
)

// mulDiv mirrors BinMath._mulDiv: floor(x*y/d), reverting when the quotient exceeds uint256.
func mulDiv(z, x, y, d *uint256.Int) error {
	if _, overflow := z.MulDivOverflow(x, y, d); overflow {
		return ErrOverflow
	}
	return nil
}

// mulDivUp mirrors BinMath._mulDivUp, including the checked +1.
func mulDivUp(z, x, y, d *uint256.Int) error {
	if v3Utils.MulDivRoundingUpV2(x, y, d, z) != nil {
		return ErrOverflow
	}
	return nil
}

// priceFromID mirrors BinMath.priceFromId: (1 + binStep/1e4)^|id| at 1e18, flooring at every
// squaring/multiply step, inverted with one more floor for negative ids.
func priceFromID(z *uint256.Int, binStepBps uint32, id int32) error {
	var base uint256.Int
	base.Add(scale, base.Mul(base.SetUint64(uint64(binStepBps)), binStepScale))
	n := uint32(id)
	if id < 0 {
		n = uint32(-id)
	}
	z.Set(scale)
	for ; n != 0; n >>= 1 {
		if n&1 != 0 {
			if err := mulDiv(z, z, &base, scale); err != nil {
				return err
			}
		}
		if n > 1 {
			if err := mulDiv(&base, &base, &base, scale); err != nil {
				return err
			}
		}
	}
	if id < 0 {
		if err := mulDiv(z, scale, scale, z); err != nil {
			return err
		}
	}
	if z.IsZero() {
		return ErrOverflow // BinOutOfRange
	}
	return nil
}

// decimalFactor is the X<->Y conversion term BinMath builds per call: SCALE*10^(dx-dy) when
// dx >= dy, else SCALE/10^(dy-dx). quoteFromX divides by it, xFromQuote multiplies by it.
func decimalFactor(z *uint256.Int, decimalsX, decimalsY uint8) *uint256.Int {
	if decimalsX >= decimalsY {
		return z.Mul(scale, big256.TenPow(decimalsX-decimalsY))
	}
	return z.Div(scale, big256.TenPow(decimalsY-decimalsX))
}

// projected mirrors BinPool._projectedVolatility at block time now.
func (f *FeeParams) projected(activeID int32, now uint64) (acc, volRef uint32, idRef int32) {
	volRef, idRef = f.VolatilityReference, f.IDReference
	var elapsed uint64
	if now > f.TimeLastUpdate {
		elapsed = now - f.TimeLastUpdate
	}
	if elapsed >= uint64(f.FilterPeriod) {
		idRef, volRef = activeID, 0
		if elapsed < uint64(f.DecayPeriod) {
			volRef = uint32(uint64(f.VolatilityAccumulator)*uint64(f.ReductionFactor)/bps) & 0xFFFFFF
		}
	}
	return f.accumulatorFor(volRef, idRef, activeID), volRef, idRef
}

// accumulatorFor mirrors BinPool._accumulatorFor.
func (f *FeeParams) accumulatorFor(volRef uint32, idRef, id int32) uint32 {
	delta := int64(id) - int64(idRef)
	if delta < 0 {
		delta = -delta
	}
	return uint32(min(uint64(volRef)+uint64(delta)*volatilityUnit, uint64(f.MaxVolatilityAccumulator)))
}

// totalFeeRate mirrors BinPool.getTotalFeeRate at 1e9 precision.
func (f *FeeParams) totalFeeRate(binStepBps uint32, activeID int32, now uint64) uint64 {
	acc, _, _ := f.projected(activeID, now)
	base := uint64(f.BaseFactor) * uint64(binStepBps) * 10
	return min(base+variableFeeRate(f.VariableFeeControl, binStepBps, acc), maxFeeRate)
}

// commit mirrors BinPool._commitVolatility, which runs while activeId is still pre-swap.
func (f *FeeParams) commit(preSwapActiveID, finalID int32, now uint64) {
	_, volRef, idRef := f.projected(preSwapActiveID, now)
	f.VolatilityAccumulator = f.accumulatorFor(volRef, idRef, finalID)
	f.VolatilityReference, f.IDReference, f.TimeLastUpdate = volRef, idRef, now
}

// variableFeeRate is ceil(control*(acc*binStep)^2/1e11). The product reaches ~2^92.
func variableFeeRate(control, binStepBps, acc uint32) uint64 {
	if control == 0 {
		return 0
	}
	var term, num uint256.Int
	term.SetUint64(uint64(acc) * uint64(binStepBps))
	num.Mul(num.Mul(num.SetUint64(uint64(control)), &term), &term)
	return big256.DivUp(&num, variableFeeScale).Uint64()
}
