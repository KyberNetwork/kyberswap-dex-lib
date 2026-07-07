package alphix

import (
	"github.com/ethereum/go-ethereum/common"
)

const (
	gasJitBeforeSwap = 248527
	gasJitAfterSwap  = 489358
)

// HookAddresses contains all known Alphix JIT hook addresses across chains.
var HookAddresses = []common.Address{
	common.HexToAddress("0x0e4b892Df7C5Bcf5010FAF4AA106074e555660C0"), // base USDS/USDC
	common.HexToAddress("0x5e645C3D580976Ca9e3fe77525D954E73a0Ce0C0"), // arbitrum USDC/USDT
}

// LvrFeeHookAddresses contains all known AlphixLVRFee hook addresses across chains.
var LvrFeeHookAddresses = []common.Address{
	common.HexToAddress("0x7cBbfF9C4fcd74B221C535F4fB4B1Db04F1B9044"), // base
}

// ProHookAddresses contains all known AlphixPro hook addresses across chains.
// Each hook is multi-pool: one address can serve many pools, each with its own
// per-pool config tracked independently via Track().
var ProHookAddresses = []common.Address{
	common.HexToAddress("0x2f9Cf87A6CbFA53C3F1B184900de17298e3F9080"), // base
}
