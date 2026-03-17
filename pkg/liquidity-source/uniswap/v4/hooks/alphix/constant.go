package alphix

import (
	"github.com/ethereum/go-ethereum/common"
)

const (
	gasJitBeforeSwap = 248527
	gasJitAfterSwap  = 489358
)

// HookAddresses contains all known Alphix hook addresses across chains.
var HookAddresses = []common.Address{
	common.HexToAddress("0x831CfDf7c0E194f5369f204b3DD2481B843d60c0"), // base ETH/USDC
	common.HexToAddress("0x0e4b892Df7C5Bcf5010FAF4AA106074e555660C0"), // base USDS/USDC
	common.HexToAddress("0x5e645C3D580976Ca9e3fe77525D954E73a0Ce0C0"), // arbitrum USDC/USDT
}
