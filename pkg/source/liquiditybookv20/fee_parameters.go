package liquiditybookv20

import "math"

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
