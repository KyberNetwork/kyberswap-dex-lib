package saturn

import _ "embed"

//go:embed abis/SwapFacility.json
var swapFacilityABIJson []byte

//go:embed abis/USDat.json
var usdatABIJson []byte
