package ambient

import "github.com/ethereum/go-ethereum/common"

const (
	DexTypeAmbient = "ambient"

	fetchLimit = 1000
)

var (
	// NativeTokenPlaceholderAddress is the address that Ambient uses to represent native token in pools.
	NativeTokenPlaceholderAddress = common.HexToAddress("0x0")
)
