package nightlyprop

import _ "embed"

//go:embed abis/NightlyRouter.json
var routerABIData []byte

//go:embed abis/NightlyPool.json
var poolABIData []byte
