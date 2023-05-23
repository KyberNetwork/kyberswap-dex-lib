package platypus

import _ "embed"

//go:embed abi/Pool.json
var poolABIData []byte

//go:embed abi/Asset.json
var assetABIData []byte

//go:embed abi/StakedAVAX.json
var stakedAvaxABIData []byte
