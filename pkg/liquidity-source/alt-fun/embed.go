package altfun

import _ "embed"

//go:embed abis/Pair.json
var pairJSON []byte

//go:embed abis/Bonding.json
var bondingJSON []byte

//go:embed abis/Factory.json
var factoryJSON []byte

//go:embed abis/LeveragedToken.json
var leveragedTokenJSON []byte

//go:embed abis/GlobalStorage.json
var globalStorageJSON []byte

//go:embed abis/Zap.json
var zapJSON []byte
