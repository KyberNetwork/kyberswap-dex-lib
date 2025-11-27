package dexv2

import (
	"fmt"
	"strconv"
	"strings"
)

func encodeFluidDexV2PoolAddress(dexId string, dexType int) string {
	return fmt.Sprintf("%s_d%d", dexId, dexType)
}

func parseFluidDexV2PoolAddress(address string) (string, int) {
	parts := strings.Split(address, "_d")
	dexType, _ := strconv.Atoi(parts[1])

	return parts[0], dexType
}
