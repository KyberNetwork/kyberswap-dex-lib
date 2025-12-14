package gateway

import (
	_ "embed"
)

//go:embed abi/InfinifiGateway.json
var gatewayBytes []byte

//go:embed abi/ERC20.json
var erc20Bytes []byte

//go:embed abi/ERC4626.json
var erc4626Bytes []byte

//go:embed pools/ethereum.json
var ethereumPoolData []byte

var BytesByPath = map[string][]byte{
	"pools/ethereum.json": ethereumPoolData,
}

