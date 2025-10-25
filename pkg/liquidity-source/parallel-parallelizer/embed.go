package parallelparallelizer

import (
	_ "embed"
)

//go:embed abis/parallelizer.json
var ParallelizerJson []byte

//go:embed abis/chainlink.json
var ChainlinkJson []byte

//go:embed abis/morpho.json
var MorphoJson []byte
