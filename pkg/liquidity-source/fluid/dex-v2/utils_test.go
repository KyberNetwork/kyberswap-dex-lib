package dexv2

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestCalcTripleSlot(t *testing.T) {
	poolAddress := "0x8d1b5f8da63fa29b191672231d3845740a11fcbef6c76e077cfffe56cc27c707_d3"
	dexID, dexType := parseFluidDexV2PoolAddress(poolAddress)

	tickIdx := 100

	dexIDHash := common.HexToHash(dexID)
	fmt.Println("dexType:", dexType)
	fmt.Println("dexIDHash:", dexIDHash.Hex())
	fmt.Println("tickIdx:", tickIdx)

	liquidityGross := calculateTripleMappingStorageSlot(
		DEX_V2_TICK_LIQUIDITY_GROSS_MAPPING_SLOT,
		dexType,
		dexIDHash,
		tickIdx,
	)
	fmt.Println("LiquidityGross slot:", liquidityGross.Hex())

	liquidityNet := calculateTripleMappingStorageSlot(
		DEX_V2_TICK_DATA_MAPPING_SLOT,
		dexType,
		dexIDHash,
		tickIdx,
	)
	fmt.Println("LiquidityNet slot:", liquidityNet.Hex())
}
