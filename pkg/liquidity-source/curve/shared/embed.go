package shared

import _ "embed"

//go:embed abi/ERC20.json
var erc20ABIBytes []byte

//go:embed abi/cERC20.json
var cerc20ABIBytes []byte

//go:embed abi/Oracle.json
var oracleABIBytes []byte
