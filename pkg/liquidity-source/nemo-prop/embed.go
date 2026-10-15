package nemoprop

import _ "embed"

//go:embed abis/NemoSwap.json
var nemoSwapABIData []byte

//go:embed abis/NemoPricing.json
var nemoPricingABIData []byte
