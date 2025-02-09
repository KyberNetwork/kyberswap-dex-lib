package skypsm

import _ "embed"

//go:embed abis/SSROracle.json
var ssrOracleABIData []byte

//go:embed pools/base.json
var basePoolData []byte

var bytesByPath = map[string][]byte{
	"pools/base.json": basePoolData,
}
