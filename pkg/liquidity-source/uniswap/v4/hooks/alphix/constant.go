package alphix

import (
	"github.com/ethereum/go-ethereum/common"
)

// jitBeforeSwapGas is the estimated additional gas for JIT liquidity simulation in BeforeSwap.
const jitBeforeSwapGas = 30000

// HookAddresses contains all known Alphix hook addresses across chains.
// Base: ETH/USDC, USDS/USDC | Arbitrum: USDC/USDT
var HookAddresses = []common.Address{
	common.HexToAddress("0x831CfDf7c0E194f5369f204b3DD2481B843d60c0"),
	common.HexToAddress("0x0e4b892Df7C5Bcf5010FAF4AA106074e555660C0"),
	common.HexToAddress("0x5e645C3D580976Ca9e3fe77525D954E73a0Ce0C0"),
}
