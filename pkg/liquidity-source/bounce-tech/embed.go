package bouncetech

import _ "embed"

//go:embed abis/LeveragedToken.json
var leveragedTokenJSON []byte

//go:embed abis/Factory.json
var factoryJSON []byte

//go:embed abis/GlobalStorage.json
var globalStorageJSON []byte
