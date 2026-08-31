package stonkbrokersfunv2

import _ "embed"

//go:embed abi/Pad.json
var padBytes []byte

//go:embed abi/Lens.json
var lensBytes []byte

//go:embed abi/AggregatorV3.json
var aggregatorV3Bytes []byte

//go:embed abi/TwapPool.json
var twapPoolBytes []byte
