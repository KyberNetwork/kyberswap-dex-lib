package saddle

import _ "embed"

//go:embed abi/SwapFlashLoan.json
var swapFlashLoanData []byte

//go:embed abi/ERC20.json
var erc20Data []byte

// Saddle pool data

//go:embed pools/saddle/arbitrum.json
var saddleArbitrumPoolData []byte

//go:embed pools/saddle/ethereum.json
var saddleEthereumPoolData []byte

//go:embed pools/saddle/fantom.json
var saddleFantomPoolData []byte

// Synapse pool data

//go:embed pools/synapse/arbitrum.json
var synapseArbitrumPoolData []byte

//go:embed pools/synapse/avalanche.json
var synapseAvalanchePoolData []byte

//go:embed pools/synapse/bsc.json
var synapseBSCPoolData []byte

//go:embed pools/synapse/ethereum.json
var synapseEthereumPoolData []byte

//go:embed pools/synapse/fantom.json
var synapseFantomPoolData []byte

//go:embed pools/synapse/optimism.json
var synapseOptimismPoolData []byte

//go:embed pools/synapse/polygon.json
var synapsePolygonPoolData []byte

//go:embed pools/synapse/base.json
var synapseBasePoolData []byte

// Axial pool data

//go:embed pools/axial/avalanche.json
var axialAvalanchePoolData []byte

// Alien Base StableSwap data

//go:embed pools/alien-base-stableswap/base.json
var alienBaseStableSwapPoolData []byte

var bytesByPath = map[string][]byte{
	"pools/saddle/arbitrum.json": saddleArbitrumPoolData,
	"pools/saddle/ethereum.json": saddleEthereumPoolData,
	"pools/saddle/fantom.json":   saddleFantomPoolData,

	"pools/synapse/arbitrum.json":  synapseArbitrumPoolData,
	"pools/synapse/avalanche.json": synapseAvalanchePoolData,
	"pools/synapse/bsc.json":       synapseBSCPoolData,
	"pools/synapse/ethereum.json":  synapseEthereumPoolData,
	"pools/synapse/fantom.json":    synapseFantomPoolData,
	"pools/synapse/optimism.json":  synapseOptimismPoolData,
	"pools/synapse/polygon.json":   synapsePolygonPoolData,
	"pools/synapse/base.json":      synapseBasePoolData,

	"pools/axial/avalanche.json": axialAvalanchePoolData,

	"pools/alien-base-stableswap/base.json": alienBaseStableSwapPoolData,
}
