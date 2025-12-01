package dexv2

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestCalcTripleSlot(t *testing.T) {
	poolAddress := "0x8d1b5f8da63fa29b191672231d3845740a11fcbef6c76e077cfffe56cc27c707_d3"
	dexID, dexType := parseFluidDexV2PoolAddress(poolAddress)

	tickIdx := 100

	dexIDHash := common.HexToHash(dexID)

	liquidityGross := calculateTripleMappingStorageSlot(
		DEX_V2_TICK_LIQUIDITY_GROSS_MAPPING_SLOT,
		dexType,
		dexIDHash,
		tickIdx,
	)
	assert.Equal(t, "0x36f12c7f41a54c979ef350e9e5b3c23f231dfeb1d8344b0f8111fc9e79f81e69", liquidityGross.Hex())

	liquidityNet := calculateTripleMappingStorageSlot(
		DEX_V2_TICK_DATA_MAPPING_SLOT,
		dexType,
		dexIDHash,
		tickIdx,
	)
	assert.Equal(t, "0xafdfcb0b7e403175ea35e5828411d6511c59463259919355a913e68d0d27db28", liquidityNet.Hex())
}
