package ondo_usdy

import _ "embed"

//go:embed abis/RWADynamicOracle.json
var rwaDynamicOracleABIJSON []byte

// joined ABI for rUSDY on ethereum and rUSDYW on mantle
//
//go:embed abis/rUSDY.json
var rUSDYABIJSON []byte

//go:embed pools/mantle.json
var mantlePoolData []byte

//go:embed pools/ethereum.json
var ethereumPoolData []byte

var bytesByPath = map[string][]byte{
	"pools/mantle.json":   mantlePoolData,
	"pools/ethereum.json": ethereumPoolData,
}
