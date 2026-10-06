package ilyris

import "github.com/holiman/uint256"

// FeeParams is BinPool.feeConfig() plus feeState(): everything getTotalFeeRate() reads
// besides binStepBps, activeId and block.timestamp.
type FeeParams struct {
	BaseFactor               uint32 `json:"baseFactor"`
	FilterPeriod             uint16 `json:"filterPeriod"`
	DecayPeriod              uint16 `json:"decayPeriod"`
	ReductionFactor          uint16 `json:"reductionFactor"`
	VariableFeeControl       uint32 `json:"variableFeeControl"`
	MaxVolatilityAccumulator uint32 `json:"maxVolatilityAccumulator"`

	VolatilityAccumulator uint32 `json:"volatilityAccumulator"`
	VolatilityReference   uint32 `json:"volatilityReference"`
	IDReference           int32  `json:"idReference"`
	TimeLastUpdate        uint64 `json:"timeLastUpdate"`
}

const volatilityUnit = 10_000 // VOLATILITY_UNIT: one bin of movement

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
			volRef = uint32(uint64(f.VolatilityAccumulator)*uint64(f.ReductionFactor)/bps) & 0xFFFFFF // uint24()
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
	acc := uint64(volRef) + uint64(delta)*volatilityUnit
	if acc > uint64(f.MaxVolatilityAccumulator) {
		return f.MaxVolatilityAccumulator
	}
	return uint32(acc)
}

// totalFeeRate mirrors BinPool.getTotalFeeRate at 1e9 precision.
func (f *FeeParams) totalFeeRate(binStepBps uint32, activeID int32, now uint64) uint64 {
	acc, _, _ := f.projected(activeID, now)
	rate := uint64(f.BaseFactor)*uint64(binStepBps)*10 + variableFeeRate(f.VariableFeeControl, binStepBps, acc)
	return min(rate, feePrecision/10) // MAX_FEE_RATE = 1e8
}

// commit mirrors BinPool._commitVolatility, which runs while activeId is still the pre-swap bin.
func (f *FeeParams) commit(preSwapActiveID, finalID int32, now uint64) {
	_, volRef, idRef := f.projected(preSwapActiveID, now)
	f.VolatilityAccumulator = f.accumulatorFor(volRef, idRef, finalID)
	f.VolatilityReference, f.IDReference, f.TimeLastUpdate = volRef, idRef, now
}

// variableFeeRate is ceil(control * (acc*binStep)^2 / 1e11). The product reaches ~2^92, so it
// is computed in uint256; the quotient fits uint64.
func variableFeeRate(control, binStepBps, acc uint32) uint64 {
	if control == 0 {
		return 0
	}
	var term, num uint256.Int
	term.SetUint64(uint64(acc) * uint64(binStepBps))
	num.SetUint64(uint64(control))
	num.Mul(&num, &term)
	num.Mul(&num, &term)
	num.Add(&num, variableFeeScale)
	num.SubUint64(&num, 1)
	return num.Div(&num, variableFeeScale).Uint64()
}
