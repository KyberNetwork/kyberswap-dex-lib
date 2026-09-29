package lunya

import _ "embed"

//go:embed abis/LunyaPool.json
var poolABIBytes []byte

//go:embed abis/LunyaPlugin.json
var pluginABIBytes []byte

//go:embed abis/LunyaPoolFactory.json
var factoryABIBytes []byte
