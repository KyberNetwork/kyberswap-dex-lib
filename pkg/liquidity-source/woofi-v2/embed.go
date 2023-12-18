package woofiv2

import _ "embed"

//go:embed abi/WooPPV2.json
var WooPPV2ABIBytes []byte

//go:embed abi/IntegrationHelper.json
var IntegrationHelperABIBytes []byte

//go:embed abi/WooracleV2.json
var WooracleV2ABIBytes []byte

//go:embed abi/ERC20.json
var Erc20ABIBytes []byte
