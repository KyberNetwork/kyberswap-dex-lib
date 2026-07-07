package solidlyv2

import _ "embed"

//go:embed abis/Pool.json
var poolABIJson []byte

//go:embed abis/Factory.json
var factoryABIJson []byte

//go:embed abis/Memecore.json
var memecoreABIJson []byte

//go:embed abis/ShadowLegacy.json
var shadowLegacyABIJson []byte
