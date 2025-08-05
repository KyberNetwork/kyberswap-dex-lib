package bunniv2

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/oracle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

func decodeHookParams(data []byte) HookParams {
	var result HookParams

	result.FeeMin = uint256.NewInt(0)
	result.FeeMax = uint256.NewInt(0)
	result.FeeQuadraticMultiplier = uint256.NewInt(0)
	// result.MaxAmAmmFee = uint256.NewInt(0)
	result.SurgeFeeHalfLife = uint256.NewInt(0)
	result.VaultSurgeThreshold0 = uint256.NewInt(0)
	result.VaultSurgeThreshold1 = uint256.NewInt(0)
	// result.MinRentMultiplier = uint256.NewInt(0)

	var firstWord, secondWord, temp uint256.Int
	firstWord.SetBytes(data[:32])
	secondWord.SetBytes(data[32:64])

	mask24 := uint256.NewInt(0xFFFFFF)   // 2^24 - 1
	mask16 := uint256.NewInt(0xFFFF)     // 2^16 - 1
	mask32 := uint256.NewInt(0xFFFFFFFF) // 2^32 - 1
	// mask48 := uint256.NewInt(0xFFFFFFFFFFFF) // 2^48 - 1

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
	// temp.Rsh(&firstWord, 96)
	// result.MaxAmAmmFee.And(&temp, mask24)

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

	// //  uint16((firstWord >> 184) & mask16)
	// temp.Rsh(&firstWord, 184)
	// temp.And(&temp, mask16)
	// result.RebalanceThreshold = uint16(temp.Uint64())

	// // uint16((firstWord >> 200) & mask16)
	// temp.Rsh(&firstWord, 200)
	// temp.And(&temp, mask16)
	// result.RebalanceMaxSlippage = uint16(temp.Uint64())

	// //  uint16((firstWord >> 216) & mask16)
	// temp.Rsh(&firstWord, 216)
	// temp.And(&temp, mask16)
	// result.RebalanceTwapSecondsAgo = uint16(temp.Uint64())

	// //  uint16((firstWord >> 232) & mask16)
	// temp.Rsh(&firstWord, 232)
	// temp.And(&temp, mask16)
	// result.RebalanceOrderTTL = uint16(temp.Uint64())

	//  (firstWord >> 248) != 0
	temp.Rsh(&firstWord, 248)
	result.AmAmmEnabled = temp.Uint64() != 0

	//  uint32(secondWord & mask32)
	temp.And(&secondWord, mask32)
	result.OracleMinInterval = uint32(temp.Uint64())

	// //  uint48((secondWord >> 32) & mask48)
	// temp.Rsh(&secondWord, 32)
	// result.MinRentMultiplier.And(&temp, mask48)

	return result
}

func decodeObservations(data []common.Hash) []*oracle.Observation {
	var observations = make([]*oracle.Observation, len(data))

	// Define masks
	mask8 := big.NewInt(0xff)                                   // 8 bits
	mask32 := big.NewInt(0xffffffff)                            // 32 bits
	mask24 := big.NewInt(0xffffff)                              // 24 bits
	mask56, _ := new(big.Int).SetString("00ffffffffffffff", 16) // 56 bits

	// Thresholds for signed conversion
	max24 := big.NewInt(0x7fffff)  // 2^23 - 1
	mod24 := big.NewInt(0x1000000) // 2^24

	max56, _ := new(big.Int).SetString("7fffffffffffff", 16)  // 2^55 - 1
	mod56, _ := new(big.Int).SetString("100000000000000", 16) // 2^56

	for i, raw := range data {
		val := raw.Big()

		// Extract fields
		blockTimestamp := new(big.Int).And(val, mask32)

		prevTick := new(big.Int).And(new(big.Int).Rsh(val, 32), mask24)
		tickCumulative := new(big.Int).And(new(big.Int).Rsh(val, 56), mask56)
		initialized := new(big.Int).And(new(big.Int).Rsh(val, 112), mask8).Cmp(big.NewInt(1)) == 0

		// Convert signed prevTick
		prevTickSigned := new(big.Int).Set(prevTick)
		if prevTick.Cmp(max24) > 0 {
			prevTickSigned.Sub(prevTick, mod24)
		}

		// Convert signed tickCumulative
		tickCumulativeSigned := new(big.Int).Set(tickCumulative)
		if tickCumulative.Cmp(max56) > 0 {
			tickCumulativeSigned.Sub(tickCumulative, mod56)
		}

		observations[i] = &oracle.Observation{
			BlockTimestamp: uint32(blockTimestamp.Uint64()),
			PrevTick:       int(prevTickSigned.Int64()),
			TickCumulative: tickCumulativeSigned.Int64(),
			Initialized:    initialized,
		}
	}

	return observations
}

func decodeAmmPayload(manager common.Address, data [6]byte) AmAmm {
	return AmAmm{
		AmAmmManager: manager,
		SwapFee0For1: new(uint256.Int).SetBytes(data[:3]),
		SwapFee1For0: new(uint256.Int).SetBytes(data[3:]),
	}
}

func decodeVaultSharePrices(data common.Hash) VaultSharePrices {
	return VaultSharePrices{}
}

func decodeHookFee(data common.Hash) *uint256.Int {
	return new(uint256.Int).SetBytes(data.Bytes())
}

func decodeCuratorFees(data common.Hash) CuratorFees {
	// Similar to TypeScript: const feeRate = bigIntify(decodedCuratorFees.and(mask16))
	mask16 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 16), big.NewInt(1))
	feeRate := new(big.Int).And(data.Big(), mask16)

	return CuratorFees{
		FeeRate: new(uint256.Int).SetBytes(feeRate.Bytes()),
	}
}

func decodeObservationState(data []common.Hash) ObservationState {
	// First hash - decode the observation state (similar to TypeScript: decodedState = BigNumber.from(decoded[0][0]))
	decodedState := data[0].Big()

	// Extract index, cardinality, cardinalityNext using bitwise operations
	// Similar to TypeScript: decodedState.and(mask32)
	mask32 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 32), big.NewInt(1))

	index := new(big.Int).And(decodedState, mask32)
	cardinality := new(big.Int).And(new(big.Int).Rsh(decodedState, 32), mask32)
	cardinalityNext := new(big.Int).And(new(big.Int).Rsh(decodedState, 64), mask32)

	// Second hash - decode the intermediate observation (similar to TypeScript: decodedIntermediateObservation = BigNumber.from(decoded[0][1]))
	decodedIntermediateObservation := data[1].Big()

	// Extract blockTimestamp, prevTick, tickCumulative, initialized
	// Similar to TypeScript: decodedIntermediateObservation.and(mask32)
	blockTimestamp := new(big.Int).And(decodedIntermediateObservation, mask32)

	// Similar to TypeScript: decodedIntermediateObservation.shr(32).and(mask24)
	mask24 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 24), big.NewInt(1))
	prevTick := new(big.Int).And(new(big.Int).Rsh(decodedIntermediateObservation, 32), mask24)

	// Similar to TypeScript: decodedIntermediateObservation.shr(56).and(mask56)
	mask56 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 56), big.NewInt(1))
	tickCumulative := new(big.Int).And(new(big.Int).Rsh(decodedIntermediateObservation, 56), mask56)

	// Similar to TypeScript: decodedIntermediateObservation.shr(112).and(mask8).eq(1)
	mask8 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 8), big.NewInt(1))
	initialized := new(big.Int).And(new(big.Int).Rsh(decodedIntermediateObservation, 112), mask8).Cmp(big.NewInt(1)) == 0

	// Handle signed values for prevTick (similar to TypeScript logic)
	// Similar to TypeScript: prevTick.gt(BigNumber.from('0x7fffff')) ? prevTick.sub(BigNumber.from('0x1000000')) : prevTick
	prevTickSigned := prevTick
	if prevTick.Cmp(new(big.Int).Lsh(big.NewInt(1), 23)) > 0 {
		prevTickSigned = new(big.Int).Sub(prevTick, new(big.Int).Lsh(big.NewInt(1), 24))
	}

	// Handle signed values for tickCumulative (similar to TypeScript logic)
	// Similar to TypeScript: tickCumulative.gt(BigNumber.from('0x7fffffffffffff')) ? tickCumulative.sub(BigNumber.from('0x100000000000000')) : tickCumulative
	tickCumulativeSigned := tickCumulative
	if tickCumulative.Cmp(new(big.Int).Lsh(big.NewInt(1), 55)) > 0 {
		tickCumulativeSigned = new(big.Int).Sub(tickCumulative, new(big.Int).Lsh(big.NewInt(1), 56))
	}

	return ObservationState{
		Index:           uint32(index.Uint64()),
		Cardinality:     uint32(cardinality.Uint64()),
		CardinalityNext: uint32(cardinalityNext.Uint64()),
		IntermediateObservation: &oracle.Observation{
			BlockTimestamp: uint32(blockTimestamp.Uint64()),
			PrevTick:       int(prevTickSigned.Int64()),
			TickCumulative: tickCumulativeSigned.Int64(),
			Initialized:    initialized,
		},
	}
}
