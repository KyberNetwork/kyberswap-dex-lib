package liquiditybookv21

import (
	"math"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

func (p *parameters) getTotalFee(binStep uint16) *big.Int {
	var baseFee, variableFee big.Int
	baseFee = *p.getBaseFee(binStep, &baseFee)
	variableFee = *p.getVariableFee(binStep, &variableFee)
	return new(big.Int).Add(&baseFee, &variableFee)
}

func (p *parameters) getBaseFee(binStep uint16, baseFee *big.Int) *big.Int {
	baseFactor := p.StaticFeeParams.BaseFactor
	baseFee.Mul(
		new(big.Int).Mul(big.NewInt(int64(baseFactor)), big.NewInt(int64(binStep))),
		bignumber.TenPowInt(10), // 1e10
	)
	return baseFee
}

func (p *parameters) getVariableFee(binStep uint16, variableFee *big.Int) *big.Int {
	variableFeeControl := p.StaticFeeParams.VariableFeeControl
	if variableFeeControl == 0 {
		return bignumber.ZeroBI
	}

	volAcc := p.VariableFeeParams.VolatilityAccumulator
	prod := new(big.Int).Mul(big.NewInt(int64(volAcc)), big.NewInt(int64(binStep)))
	variableFee.Div(
		new(big.Int).Add(
			new(big.Int).Mul(
				new(big.Int).Mul(prod, prod),
				big.NewInt(int64(variableFeeControl)),
			),
			big.NewInt(99),
		),
		bignumber.TenPowInt(2), // 100
	)
	return variableFee
}
