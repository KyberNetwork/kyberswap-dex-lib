package liquiditybookv20

import (
	"math"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
)

type feeParameters struct {
	BinStep                  uint16 `json:"binStep"`
	BaseFactor               uint16 `json:"baseFactor"`
	FilterPeriod             uint16 `json:"filterPeriod"`
	DecayPeriod              uint16 `json:"decayPeriod"`
	ReductionFactor          uint16 `json:"reductionFactor"`
	VariableFeeControl       uint32 `json:"variableFeeControl"`
	ProtocolShare            uint16 `json:"protocolShare"`
	MaxVolatilityAccumulated uint32 `json:"maxVolatilityAccumulated"`
	VolatilityAccumulated    uint32 `json:"volatilityAccumulated"`
	VolatilityReference      uint32 `json:"volatilityReference"`
	IndexRef                 uint32 `json:"indexRef"`
	Time                     uint64 `json:"time"`
}

func (fp *feeParameters) updateVariableFeeParameters(blockTimestamp uint64, activeID uint32) {
	delta := blockTimestamp - fp.Time

	if delta >= uint64(fp.FilterPeriod) || fp.Time == 0 {
		fp.IndexRef = activeID
		if delta < uint64(fp.DecayPeriod) {
			fp.VolatilityReference = uint32(uint64(fp.ReductionFactor) * uint64(fp.VolatilityAccumulated) / basisPointMax)
		} else {
			fp.VolatilityReference = 0
		}
	}

	fp.Time = blockTimestamp

	fp.updateVolatilityAccumulated(activeID)
}

func (fp *feeParameters) updateVolatilityAccumulated(activeID uint32) {
	absSub := math.Abs(float64(activeID) - float64(fp.IndexRef))
	volatilityAccumulated := uint64(absSub)*basisPointMax + uint64(fp.VolatilityReference)

	if volatilityAccumulated > uint64(fp.MaxVolatilityAccumulated) {
		volatilityAccumulated = uint64(fp.MaxVolatilityAccumulated)
	}

	fp.VolatilityAccumulated = uint32(volatilityAccumulated)
}

func (fp *feeParameters) getFeeAmount(amount *big.Int) *big.Int {
	fee := fp.getTotalFee()
	denominator := new(big.Int).Sub(precison, fee)
	result := new(big.Int).Div(
		new(big.Int).Sub(
			new(big.Int).Add(new(big.Int).Mul(amount, fee), denominator),
			integer.One(),
		),
		denominator,
	)
	return result
}

func (fp *feeParameters) getTotalFee() *big.Int {
	baseFee := fp.getBaseFee()
	variableFee := fp.getVariableFee()
	return new(big.Int).Add(baseFee, variableFee)
}

func (fp *feeParameters) getBaseFee() *big.Int {
	baseFactor := fp.BaseFactor
	result := new(big.Int).Mul(
		new(big.Int).Mul(big.NewInt(int64(baseFactor)), big.NewInt(int64(fp.BinStep))),
		big.NewInt(1e10),
	)
	return result
}

func (fp *feeParameters) getVariableFee() *big.Int {
	if fp.VariableFeeControl == 0 {
		return integer.Zero()
	}

	prod := new(big.Int).Mul(
		big.NewInt(int64(fp.VolatilityAccumulated)),
		big.NewInt(int64(fp.BinStep)),
	)
	variableFee := new(big.Int).Div(
		new(big.Int).Add(
			new(big.Int).Mul(
				new(big.Int).Mul(prod, prod),
				big.NewInt(int64(fp.VariableFeeControl)),
			),
			big.NewInt(99),
		),
		big.NewInt(100),
	)
	return variableFee
}

func (fp *feeParameters) getFeeAmountDistribution(fees *big.Int) (*big.Int, *big.Int) {
	total := fees
	protocol := new(big.Int).Div(
		new(big.Int).Mul(total, big.NewInt(int64(fp.ProtocolShare))),
		big.NewInt(basisPointMax),
	)
	return total, protocol
}
