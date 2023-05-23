package polydex

import _ "embed"

//go:embed abis/Pair.json
var pairJson []byte

//go:embed abis/PolydexFactory.json
var polydexFactoryJson []byte
