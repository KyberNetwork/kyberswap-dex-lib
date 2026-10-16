package saturnusdat

import _ "embed"

//go:embed abi/SwapFacility.json
var swapFacilityABIJson []byte

//go:embed abi/USDat.json
var usdatABIJson []byte
