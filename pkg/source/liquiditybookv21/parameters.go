package liquiditybookv21

import (
	"math"

	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/LBPair.sol#L60
type parameters struct {
	StaticFeeParams   staticFeeParams   `json:"staticFeeParams"`
	VariableFeeParams variableFeeParams `json:"variableFeeParams"`
	ActiveBinID       uint32            `json:"-"`
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

	volRef := uint32(uint64(volAcc) * uint64(reductionFactor) / basisPointMax)
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
	deltaID := uint64(math.Abs(float64(activeID) - float64(idReference)))
	volAcc := uint64(p.VariableFeeParams.VolatilityReference) + deltaID*basisPointMax

	maxVolAcc := uint64(p.StaticFeeParams.MaxVolatilityAccumulator)
	if volAcc > maxVolAcc {
		volAcc = maxVolAcc
	}

	p.VariableFeeParams.VolatilityAccumulator = uint32(volAcc)

	return p
}

func (p *parameters) getTotalFee(binStep uint16) *uint256.Int {
	var baseFee, variableFee uint256.Int
	p.getBaseFee(binStep, &baseFee)
	p.getVariableFee(binStep, &variableFee)
	return baseFee.Add(&baseFee, &variableFee)
}

func (p *parameters) getBaseFee(binStep uint16, baseFee *uint256.Int) *uint256.Int {
	baseFactor := uint256.NewInt(uint64(p.StaticFeeParams.BaseFactor))
	baseFee.Mul(
		baseFee.Mul(baseFactor, baseFee.SetUint64(uint64(binStep))),
		big256.TenPowInt(10), // 1e10
	)
	return baseFee
}

func (p *parameters) getVariableFee(binStep uint16, variableFee *uint256.Int) *uint256.Int {
	variableFeeControl := p.StaticFeeParams.VariableFeeControl
	if variableFeeControl == 0 {
		return big256.ZeroBI
	}

	volAcc := uint256.NewInt(uint64(p.VariableFeeParams.VolatilityAccumulator))
	variableFee.Mul(volAcc, variableFee.SetUint64(uint64(binStep)))
	variableFee.Div(
		variableFee.Add(
			variableFee.Mul(
				variableFee.Mul(variableFee, variableFee),
				volAcc.SetUint64(uint64(variableFeeControl)),
			),
			volAcc.SetUint64(99),
		),
		big256.TenPowInt(2), // 100
	)
	return variableFee
}
