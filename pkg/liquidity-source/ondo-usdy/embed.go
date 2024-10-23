package ondo_usdy

import _ "embed"

//go:embed abis/RWADynamicOracle.json
var rwaDynamicOracleABIJSON []byte

//go:embed abis/MUSD.json
var mUSDABIJSON []byte

//go:embed pools/mantle.json
var mantlePoolData []byte

var bytesByPath = map[string][]byte{
	"pools/mantle.json": mantlePoolData,
}
