package platypus

import _ "embed"

//go:embed abi/Pool.json
var poolABIData []byte

//go:embed abi/Asset.json
var assetABIData []byte

//go:embed abi/StakedAVAX.json
var stakedAvaxABIData []byte

//go:embed abi/Oracle.json
var oracleABIData []byte

//go:embed abi/Chainlink.json
var chainlinkABIData []byte
