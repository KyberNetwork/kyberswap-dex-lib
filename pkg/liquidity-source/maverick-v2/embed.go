package maverickv2

import _ "embed"

//go:embed abis/MaverickV2Factory.json
var maverickV2FactoryABIJson []byte

//go:embed abis/MaverickV2Pool.json
var maverickV2PoolABIJson []byte
