package stabull

import _ "embed"

//go:embed abis/StabullPool.json
var stabullPoolABIData []byte

//go:embed abis/Assimilator.json
var assimilatorABIData []byte
