package nerve

import (
	_ "embed"
)

//go:embed pools/bsc.json
var nervePoolsBytes []byte

var BytesByPath = map[string][]byte{
	"pools/bsc.json": nervePoolsBytes,
}

//go:embed abis/Swap.json
var nerveSwapJson []byte

//go:embed abis/ERC20.json
var erc20Json []byte
