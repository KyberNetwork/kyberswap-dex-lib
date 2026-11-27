package syntropiaprop

import _ "embed"

//go:embed abis/Pool.json
var poolABIData []byte

//go:embed abis/Lens.json
var lensABIData []byte
