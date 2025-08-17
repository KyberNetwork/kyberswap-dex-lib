package bunniv2

import (
	"encoding/binary"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/oracle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

func decodeHookParams(data []byte) HookParams {
	var result HookParams

	// first 32 bytes (firstWord)
	// | feeMin 3 | feeMax 3 | feeQuadraticMultiplier 3 | feeTwapSecondsAgo 3 |
	// | maxAmAmmFee 3 | surgeFeeHalfLife 2 | surgeFeeAutostartThreshold 2 |
	// | vaultSurgeThreshold0 2 | vaultSurgeThreshold1 2 | rebalanceThreshold 2 |
	// | rebalanceMaxSlippage 2 | rebalanceTwapSecondsAgo 2 | rebalanceOrderTTL 2 | amAmmEnabled 1 |

	var feeMin, feeMax, feeQuad uint256.Int
	feeMin.SetBytes(data[0:3])
	feeMax.SetBytes(data[3:6])
	feeQuad.SetBytes(data[6:9])
	result.FeeMin = &feeMin
	result.FeeMax = &feeMax
	result.FeeQuadraticMultiplier = &feeQuad

	result.FeeTwapSecondsAgo = uint32(data[9])<<16 | uint32(data[10])<<8 | uint32(data[11])

	var surgeHalf uint256.Int
	surgeHalf.SetBytes(data[15:17])
	result.SurgeFeeHalfLife = &surgeHalf

	result.SurgeFeeAutostartThreshold = binary.BigEndian.Uint16(data[17:19])

	var v0, v1 uint256.Int
	v0.SetBytes(data[19:21])
	v1.SetBytes(data[21:23])
	result.VaultSurgeThreshold0 = &v0
	result.VaultSurgeThreshold1 = &v1

	result.RebalanceThreshold = binary.BigEndian.Uint16(data[23:25])

	result.AmAmmEnabled = data[31] != 0

	result.OracleMinInterval = binary.BigEndian.Uint32(data[32:36])

	return result
}

func decodeObservations(data []common.Hash) []*oracle.Observation {
	observations := make([]*oracle.Observation, len(data))

	for i, raw := range data {
		bt := binary.BigEndian.Uint32(raw[28:32])

		ptU := (uint32(raw[25]) << 16) | (uint32(raw[26]) << 8) | uint32(raw[27])
		var prevTick int32
		if ptU > 0x7FFFFF {
			prevTick = int32(ptU - 0x1000000)
		} else {
			prevTick = int32(ptU)
		}

		tcU := (uint64(raw[18]) << 48) |
			(uint64(raw[19]) << 40) |
			(uint64(raw[20]) << 32) |
			(uint64(raw[21]) << 24) |
			(uint64(raw[22]) << 16) |
			(uint64(raw[23]) << 8) |
			uint64(raw[24])

		var tickCumulative int64
		if tcU > 0x7FFFFFFFFFFFFF {
			tickCumulative = int64(tcU - 0x100000000000000)
		} else {
			tickCumulative = int64(tcU)
		}

		initialized := raw[17] == 1

		observations[i] = &oracle.Observation{
			BlockTimestamp: bt,
			PrevTick:       int(prevTick),
			TickCumulative: tickCumulative,
			Initialized:    initialized,
		}
	}

	return observations
}

func decodeAmmPayload(manager common.Address, data [6]byte) AmAmm {
	var swapFee0For1, swapFee1For0 uint256.Int
	swapFee0For1.SetBytes(data[:3])
	swapFee1For0.SetBytes(data[3:])

	return AmAmm{
		AmAmmManager: manager,
		SwapFee0For1: &swapFee0For1,
		SwapFee1For0: &swapFee1For0,
	}
}

func decodeVaultSharePrices(data common.Hash) VaultSharePrices {
	initialized := data[31] == 1
	var sp0, sp1 uint256.Int
	sp0.SetBytes(data[16:31])
	sp1.SetBytes(data[1:16])

	return VaultSharePrices{
		Initialized:  initialized,
		SharedPrice0: &sp0,
		SharedPrice1: &sp1,
	}
}

func decodeHookFee(data common.Hash) *uint256.Int {
	var fee uint256.Int
	fee.SetBytes(data[8:12])
	return &fee
}

func decodeCuratorFees(data common.Hash) CuratorFees {
	var feeRate uint256.Int
	feeRate.SetBytes(data[30:32])

	return CuratorFees{
		FeeRate: &feeRate,
	}
}

func decodeRebalanceOrderDeadline(data common.Hash) uint32 {
	return binary.BigEndian.Uint32(data[:])
}

func decodeObservationState(data []common.Hash) ObservationState {
	rawState := data[0]
	rawInter := data[1]

	index := binary.BigEndian.Uint32(rawState[28:32])
	cardinality := binary.BigEndian.Uint32(rawState[24:28])
	cardinalityNext := binary.BigEndian.Uint32(rawState[20:24])
	blockTimestamp := binary.BigEndian.Uint32(rawInter[28:32])

	prevTickRaw := (uint32(rawInter[25]) << 16) | (uint32(rawInter[26]) << 8) | uint32(rawInter[27])
	var prevTick int32
	if prevTickRaw&0x800000 != 0 {
		prevTick = int32(prevTickRaw | 0xFF000000)
	} else {
		prevTick = int32(prevTickRaw)
	}

	tickCumulativeRaw := (uint64(rawInter[18]) << 48) |
		(uint64(rawInter[19]) << 40) |
		(uint64(rawInter[20]) << 32) |
		(uint64(rawInter[21]) << 24) |
		(uint64(rawInter[22]) << 16) |
		(uint64(rawInter[23]) << 8) |
		uint64(rawInter[24])

	var tickCumulative int64
	if tickCumulativeRaw&0x80000000000000 != 0 {
		tickCumulative = int64(tickCumulativeRaw | 0xFF00000000000000)
	} else {
		tickCumulative = int64(tickCumulativeRaw)
	}

	initialized := rawInter[17] == 1

	return ObservationState{
		Index:           index,
		Cardinality:     cardinality,
		CardinalityNext: cardinalityNext,
		IntermediateObservation: &oracle.Observation{
			BlockTimestamp: uint32(blockTimestamp),
			PrevTick:       int(prevTick),
			TickCumulative: tickCumulative,
			Initialized:    initialized,
		},
	}
}
