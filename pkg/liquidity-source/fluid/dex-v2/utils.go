package dexv2

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func encodeFluidDexV2PoolAddress(dexId string, dexType uint32) string {
	return fmt.Sprintf("%s_%d", dexId, dexType)
}

func parseFluidDexV2PoolAddress(address string) (string, uint32) {
	parts := strings.Split(address, "_")
	dexType, _ := strconv.ParseUint(parts[1], 10, 32)

	return parts[0], uint32(dexType)
}

func calculateMappingStorageSlot(slot uint64, key common.Address) common.Hash {
	paddedKey := common.LeftPadBytes(key.Bytes(), 32)

	slotBig := new(big.Int).SetUint64(slot)
	paddedSlot := common.LeftPadBytes(slotBig.Bytes(), 32)

	input := append(paddedKey, paddedSlot...)

	return crypto.Keccak256Hash(input)
}

func calculateDoubleMappingStorageSlot(slot uint64, dexType uint32, dexId common.Hash) common.Hash {
	slotBig := new(big.Int).SetUint64(slot)
	paddedSlot := common.LeftPadBytes(slotBig.Bytes(), 32)

	dexTypeBI := new(big.Int).SetInt64(int64(dexType))
	key1 := common.LeftPadBytes(dexTypeBI.Bytes(), 32)
	key2 := common.LeftPadBytes(dexId.Bytes(), 32)

	intermediateSlot := crypto.Keccak256(append(key1, paddedSlot...))
	return crypto.Keccak256Hash(append(key2, intermediateSlot...))
}

func calculateTripleMappingStorageSlot(
	slot uint64, dexType uint32, dexId common.Hash, tickIdx int,
) common.Hash {
	slotBig := new(big.Int).SetUint64(slot)
	paddedSlot := common.LeftPadBytes(slotBig.Bytes(), 32)

	dexTypeBI := new(big.Int).SetInt64(int64(dexType))
	key1 := common.LeftPadBytes(dexTypeBI.Bytes(), 32)

	key2 := common.LeftPadBytes(dexId.Bytes(), 32)

	tickIdxBI := new(big.Int).SetInt64(int64(tickIdx))
	if tickIdxBI.Sign() < 0 {
		tickIdxBI.Add(tickIdxBI, two256)
	}
	key3 := common.LeftPadBytes(tickIdxBI.Bytes(), 32)

	intermediateSlot1 := crypto.Keccak256(append(key1, paddedSlot...))
	intermediateSlot2 := crypto.Keccak256(append(key2, intermediateSlot1...))
	return crypto.Keccak256Hash(append(key3, intermediateSlot2...))
}

func extractTokenReserves(tokenReserves *big.Int) (*big.Int, *big.Int) {
	var token0Reserves, token1Reserves big.Int
	token0Reserves.Set(tokenReserves).
		Rsh(&token0Reserves, BITS_DEX_V2_TOKEN_RESERVES_TOKEN_0_RESERVES).
		And(&token0Reserves, X128)

	token1Reserves.Set(tokenReserves).
		Rsh(&token1Reserves, BITS_DEX_V2_TOKEN_RESERVES_TOKEN_1_RESERVES).
		And(&token1Reserves, X128)

	return &token0Reserves, &token1Reserves
}
