package velocorev2stable

import _ "embed"

//go:embed abis/WombatPool.json
var poolABIJson []byte

//go:embed abis/WombatRegistry.json
var registryABIJson []byte
