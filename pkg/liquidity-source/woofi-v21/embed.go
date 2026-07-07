package woofiv21

import _ "embed"

//go:embed abi/WooPPV2_1.json
var WooPPV2ABIBytes []byte

//go:embed abi/IntegrationHelper.json
var IntegrationHelperABIBytes []byte

//go:embed abi/WooracleV2_2_1.json
var WooracleV2ABIBytes []byte

//go:embed abi/Cloracle.json
var CloracleABIBytes []byte

//go:embed abi/ERC20.json
var Erc20ABIBytes []byte
