package liquiditybookv21

import "math"

// https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/LBPair.sol#L60
type parameters struct {
	StaticFeeParams   staticFeeParams
	VariableFeeParams variableFeeParams
	ActiveBinID       uint32
}

func (p *parameters) updateReferences(blockTimestamp uint64) *parameters {
	dt := blockTimestamp - p.VariableFeeParams.TimeOfLastUpdate

	if dt >= uint64(p.StaticFeeParams.FilterPeriod) {
		p = p.updateIdReference()

		if dt < uint64(p.StaticFeeParams.DecayPeriod) {
			p = p.updateVolatilityReference()
		} else {
			p = p.setVolatilityReference(0)
		}
	}

	return p.updateTimeOfLastUpdate(blockTimestamp)
}

func (p *parameters) updateIdReference() *parameters {
	p.VariableFeeParams.IdReference = p.ActiveBinID
	return p
}

func (p *parameters) updateVolatilityReference() *parameters {
	volAcc := p.VariableFeeParams.VolatilityAccumulator
	reductionFactor := uint32(p.StaticFeeParams.ReductionFactor)

	volRef := volAcc * reductionFactor / basisPointMax
	return p.setVolatilityReference(volRef)
}

func (p *parameters) setVolatilityReference(volRef uint32) *parameters {
	p.VariableFeeParams.VolatilityReference = volRef
	return p
}

func (p *parameters) updateTimeOfLastUpdate(blockTimestamp uint64) *parameters {
	p.VariableFeeParams.TimeOfLastUpdate = blockTimestamp
	return p
}

func (p *parameters) updateVolatilityAccumulator(activeID uint32) *parameters {
	idReference := uint64(p.VariableFeeParams.IdReference)
	deltaID := uint64(math.Abs(float64(uint64(activeID) - idReference)))
	volAcc := uint64(p.VariableFeeParams.VolatilityReference) + deltaID*basisPointMax

	maxVolAcc := uint64(p.StaticFeeParams.MaxVolatilityAccumulator)
	if volAcc > maxVolAcc {
		volAcc = maxVolAcc
	}

	p.VariableFeeParams.VolatilityAccumulator = uint32(volAcc)

	return p
}
