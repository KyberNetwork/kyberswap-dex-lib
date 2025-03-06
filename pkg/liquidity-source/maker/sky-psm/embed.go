package skypsm

import _ "embed"

//go:embed abis/SSROracle.json
var ssrOracleABIData []byte

//go:embed abis/PSM3.json
var psm3ABIData []byte
