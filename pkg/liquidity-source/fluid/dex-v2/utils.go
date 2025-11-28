package dexv2

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func encodeFluidDexV2PoolAddress(dexId string, dexType int) string {
	return fmt.Sprintf("%s_d%d", dexId, dexType)
}

func parseFluidDexV2PoolAddress(address string) (string, int) {
	parts := strings.Split(address, "_d")
	dexType, _ := strconv.Atoi(parts[1])

	return parts[0], dexType
}

func calculateMappingStorageSlot(slot uint64, key common.Address) common.Hash {
	paddedKey := common.LeftPadBytes(key.Bytes(), 32)

	slotBig := new(big.Int).SetUint64(slot)
	paddedSlot := common.LeftPadBytes(slotBig.Bytes(), 32)

	input := append(paddedKey, paddedSlot...)

	return crypto.Keccak256Hash(input)
}
