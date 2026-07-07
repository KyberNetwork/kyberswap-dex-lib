package hiddenocean

import _ "embed"

//go:embed abi/HiddenOceanPool.json
var poolABIBytes []byte

//go:embed abi/HiddenOceanRegistry.json
var registryABIBytes []byte

//go:embed abi/ERC20.json
var erc20ABIBytes []byte
