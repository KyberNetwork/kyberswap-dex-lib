package overnightusdp

import _ "embed"

//go:embed abis/Exchange.json
var exchangeABIJson []byte

//go:embed abis/ERC20.json
var erc20ABIJson []byte
