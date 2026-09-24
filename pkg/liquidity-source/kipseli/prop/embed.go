package prop

import _ "embed"

//go:embed abis/Lens.json
var lensABIData []byte

//go:embed abis/Swap.json
var swapABIData []byte

//go:embed abis/LensPamm.json
var lensPammABIData []byte

//go:embed abis/RouterPamm.json
var routerPammABIData []byte

//go:embed abis/PositionCap.json
var positionCapABIData []byte
