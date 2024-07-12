package ambient

import "github.com/ethereum/go-ethereum/common"

const (
	DexTypeAmbient = "ambient"

	fetchLimit = 1000
)

var (
	// Address that Ambient uses to represent native token in pools.
	nativeTokenPlaceholderAddress = common.HexToAddress("0x0")
)
