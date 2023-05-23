package saddle

import _ "embed"

//go:embed abi/SwapFlashLoan.json
var swapFlashLoanData []byte

//go:embed abi/ERC20.json
var erc20Data []byte

//go:embed pools/arbitrum.json
var arbitrumPoolData []byte

//go:embed pools/ethereum.json
var ethereumPoolData []byte

//go:embed pools/fantom.json
var fantomPoolData []byte

var bytesByPath = map[string][]byte{
	"pools/arbitrum.json": arbitrumPoolData,
	"pools/ethereum.json": ethereumPoolData,
	"pools/fantom.json":   fantomPoolData,
}
