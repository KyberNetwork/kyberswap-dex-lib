package kipseliprop

import _ "embed"

//go:embed abis/Lens.json
var lensABIData []byte

//go:embed abis/Swap.json
var swapABIData []byte
