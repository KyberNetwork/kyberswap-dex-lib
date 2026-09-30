package elfomofi

import _ "embed"

//go:embed abis/Factory.json
var factoryABIData []byte

//go:embed abis/Helper.json
var helperABIData []byte
