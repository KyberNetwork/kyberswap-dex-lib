package pamm

import _ "embed"

//go:embed abi/Lens.json
var lensABIData []byte

//go:embed abi/Router.json
var routerABIData []byte

//go:embed abi/PositionCap.json
var positionCapABIData []byte
