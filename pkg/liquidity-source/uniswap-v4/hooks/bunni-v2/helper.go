package bunniv2

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/oracle"
	"github.com/holiman/uint256"
)

func DecodeHookParams(hookParams []byte) DecodedHookParams {
	var result DecodedHookParams

	result.FeeMin = uint256.NewInt(0)
	result.FeeMax = uint256.NewInt(0)
	result.FeeQuadraticMultiplier = uint256.NewInt(0)
	result.MaxAmAmmFee = uint256.NewInt(0)
	result.SurgeFeeHalfLife = uint256.NewInt(0)
	result.VaultSurgeThreshold0 = uint256.NewInt(0)
	result.VaultSurgeThreshold1 = uint256.NewInt(0)
	result.MinRentMultiplier = uint256.NewInt(0)

	var firstWord, secondWord, temp uint256.Int
	firstWord.SetBytes(hookParams[:32])
	secondWord.SetBytes(hookParams[32:64])

	mask24 := uint256.NewInt(0xFFFFFF)       // 2^24 - 1
	mask16 := uint256.NewInt(0xFFFF)         // 2^16 - 1
	mask32 := uint256.NewInt(0xFFFFFFFF)     // 2^32 - 1
	mask48 := uint256.NewInt(0xFFFFFFFFFFFF) // 2^48 - 1

	//  uint24(firstWord & mask24)
	result.FeeMin.And(&firstWord, mask24)

	//  uint24((firstWord >> 24) & mask24)
	temp.Rsh(&firstWord, 24)
	result.FeeMax.And(&temp, mask24)

	//  uint24((firstWord >> 48) & mask24)
	temp.Rsh(&firstWord, 48)
	result.FeeQuadraticMultiplier.And(&temp, mask24)

	//  uint24((firstWord >> 72) & mask24)
	temp.Rsh(&firstWord, 72)
	temp.And(&temp, mask24)
	result.FeeTwapSecondsAgo = uint32(temp.Uint64())

	//  uint24((firstWord >> 96) & mask24)
	temp.Rsh(&firstWord, 96)
	result.MaxAmAmmFee.And(&temp, mask24)

	//  uint16((firstWord >> 120) & mask16)
	temp.Rsh(&firstWord, 120)
	temp.And(&temp, mask16)
	result.SurgeFeeHalfLife.SetUint64(temp.Uint64())

	//  uint16((firstWord >> 136) & mask16)
	temp.Rsh(&firstWord, 136)
	temp.And(&temp, mask16)
	result.SurgeFeeAutostartThreshold = uint16(temp.Uint64())

	//  uint16((firstWord >> 152) & mask16)
	temp.Rsh(&firstWord, 152)
	temp.And(&temp, mask16)
	result.VaultSurgeThreshold0.SetUint64(temp.Uint64())

	//  uint16((firstWord >> 168) & mask16)
	temp.Rsh(&firstWord, 168)
	temp.And(&temp, mask16)
	result.VaultSurgeThreshold1.SetUint64(temp.Uint64())

	//  uint16((firstWord >> 184) & mask16)
	temp.Rsh(&firstWord, 184)
	temp.And(&temp, mask16)
	result.RebalanceThreshold = uint16(temp.Uint64())

	// uint16((firstWord >> 200) & mask16)
	temp.Rsh(&firstWord, 200)
	temp.And(&temp, mask16)
	result.RebalanceMaxSlippage = uint16(temp.Uint64())

	//  uint16((firstWord >> 216) & mask16)
	temp.Rsh(&firstWord, 216)
	temp.And(&temp, mask16)
	result.RebalanceTwapSecondsAgo = uint16(temp.Uint64())

	//  uint16((firstWord >> 232) & mask16)
	temp.Rsh(&firstWord, 232)
	temp.And(&temp, mask16)
	result.RebalanceOrderTTL = uint16(temp.Uint64())

	//  (firstWord >> 248) != 0
	temp.Rsh(&firstWord, 248)
	result.AmAmmEnabled = temp.Uint64() != 0

	//  uint32(secondWord & mask32)
	temp.And(&secondWord, mask32)
	result.OracleMinInterval = uint32(temp.Uint64())

	//  uint48((secondWord >> 32) & mask48)
	temp.Rsh(&secondWord, 32)
	result.MinRentMultiplier.And(&temp, mask48)

	return result
}

type ObservationState struct {
	Index                   uint32
	Cardinality             uint32
	CardinalityNext         uint32
	IntermediateObservation oracle.Observation
}

type LdfState struct {
	Initialized bool
	LastMinTick int
}

func DecodeLdfState(ldfState [32]byte) LdfState {
	u24 := int32(ldfState[1])<<16 |
		int32(ldfState[2])<<8 |
		int32(ldfState[3])

	signed := (u24 << 8) >> 8

	return LdfState{
		Initialized: ldfState[0] == 1,
		LastMinTick: int(signed),
	}
}
