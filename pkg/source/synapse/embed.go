package synapse

import _ "embed"

//go:embed abi/SwapFlashLoan.json
var swapFlashLoanData []byte

//go:embed abi/ERC20.json
var erc20Data []byte

//go:embed pools/arbitrum.json
var arbitrumPoolData []byte

//go:embed pools/avalanche.json
var avalanchePoolData []byte

//go:embed pools/bsc.json
var bscPoolData []byte

//go:embed pools/ethereum.json
var ethereumPoolData []byte

//go:embed pools/fantom.json
var fantomPoolData []byte

//go:embed pools/optimism.json
var optimismPoolData []byte

//go:embed pools/polygon.json
var polygonPoolData []byte

var bytesByPath = map[string][]byte{
	"pools/arbitrum.json":  arbitrumPoolData,
	"pools/avalanche.json": avalanchePoolData,
	"pools/bsc.json":       bscPoolData,
	"pools/ethereum.json":  ethereumPoolData,
	"pools/fantom.json":    fantomPoolData,
	"pools/optimism.json":  optimismPoolData,
	"pools/polygon.json":   polygonPoolData,
}
