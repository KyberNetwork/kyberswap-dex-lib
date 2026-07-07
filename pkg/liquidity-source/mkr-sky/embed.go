package mkr_sky

import _ "embed"

//go:embed pools/ethereum.json
var ethereumPoolData []byte

//go:embed abis/mkrSky.json
var mkrSkyABIData []byte

var bytesByPath = map[string][]byte{
	"pools/ethereum.json": ethereumPoolData,
}
