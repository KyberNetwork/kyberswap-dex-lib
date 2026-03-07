package obric

import _ "embed"

//go:embed abi/Registry.json
var registryABIJson []byte

//go:embed abi/Pool.json
var poolABIJson []byte
