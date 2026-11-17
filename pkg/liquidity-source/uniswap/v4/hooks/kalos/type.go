package kalos

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// poolConfigRaw decodes poolConfig(id), field order matching the ABI's outputs.
type poolConfigRaw struct {
	Registered    bool
	Kind          uint8
	FeeSide       uint8
	Preset        uint8
	KalosShareBps uint16
	LaunchToken   common.Address
	BookFrozen    bool
}

// feeParamsResult and feeStateResult are decode targets: go-ethereum unpacks a single tuple output into the first
// field of a struct.
type feeParamsResult struct{ Params feeParamsRaw }

type feeParamsRaw struct {
	MinFeeZeroForOne         *big.Int
	MaxFeeZeroForOne         *big.Int
	MinFeeOneForZero         *big.Int
	MaxFeeOneForZero         *big.Int
	Dynamic                  bool
	Directional              bool
	FilterPeriod             uint16
	DecayPeriod              uint16
	ReductionFactor          uint16
	VariableFeeControl       *big.Int
	MaxVolatilityAccumulator *big.Int
	ResetPeriod              uint16
	ResetTickFilter          *big.Int
	SameMomentSeconds        uint8
}

type feeStateResult struct{ State feeStateRaw }

type feeStateRaw struct {
	ReferenceTick       *big.Int
	LastTick            *big.Int
	Peak                *big.Int
	VolatilityReference *big.Int
	LastSwap            *big.Int
	ResetTime           *big.Int
	MoveTime            *big.Int
	PeakTime            *big.Int
}

func (r *feeParamsRaw) params() FeeParams {
	return FeeParams{
		MinFeeZeroForOne:         uint32(r.MinFeeZeroForOne.Uint64()),
		MaxFeeZeroForOne:         uint32(r.MaxFeeZeroForOne.Uint64()),
		MinFeeOneForZero:         uint32(r.MinFeeOneForZero.Uint64()),
		MaxFeeOneForZero:         uint32(r.MaxFeeOneForZero.Uint64()),
		Dynamic:                  r.Dynamic,
		Directional:              r.Directional,
		FilterPeriod:             uint64(r.FilterPeriod),
		DecayPeriod:              uint64(r.DecayPeriod),
		ReductionFactor:          uint64(r.ReductionFactor),
		VariableFeeControl:       r.VariableFeeControl.Uint64(),
		MaxVolatilityAccumulator: r.MaxVolatilityAccumulator.Uint64(),
		ResetPeriod:              uint64(r.ResetPeriod),
		ResetTickFilter:          r.ResetTickFilter.Uint64(),
		SameMomentSeconds:        uint64(r.SameMomentSeconds),
	}
}

func (r *feeStateRaw) state() FeeState {
	return FeeState{
		ReferenceTick:       int32(r.ReferenceTick.Int64()),
		LastTick:            int32(r.LastTick.Int64()),
		Peak:                int32(r.Peak.Int64()),
		VolatilityReference: r.VolatilityReference.Uint64(),
		LastSwap:            r.LastSwap.Uint64(),
		ResetTime:           r.ResetTime.Uint64(),
		MoveTime:            r.MoveTime.Uint64(),
		PeakTime:            r.PeakTime.Uint64(),
	}
}
