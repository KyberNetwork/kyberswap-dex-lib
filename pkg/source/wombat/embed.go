package wombat

import _ "embed"

//go:embed abi/PoolV2.json
var PoolV2ABIData []byte

//go:embed abi/Asset.json
var AssetABIData []byte

//go:embed abi/DynamicAsset.json
var DynamicAssetABIData []byte

//go:embed abi/CrossChainPool.json
var CrossChainPoolABIData []byte
