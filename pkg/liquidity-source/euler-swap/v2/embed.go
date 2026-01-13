package v2

import "embed"

//go:embed abis/*.json
var abiData embed.FS
