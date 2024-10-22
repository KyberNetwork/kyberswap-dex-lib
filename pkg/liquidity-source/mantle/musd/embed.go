package musd

import _ "embed"

//go:embed abis/RWADynamicOracle.json
var rwaDynamicOracleABIJSON []byte

//go:embed abis/MUSD.json
var mUSDABIJSON []byte
