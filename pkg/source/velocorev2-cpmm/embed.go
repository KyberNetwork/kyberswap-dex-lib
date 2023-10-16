package velocorev2cpmm

import _ "embed"

//go:embed abis/ConstantProductPoolFactory.json
var factoryABIJson []byte

//go:embed abis/ConstantProductPool.json
var poolABIJson []byte
