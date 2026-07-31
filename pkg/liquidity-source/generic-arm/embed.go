package genericarm

import _ "embed"

//go:embed abis/lidoarm.json
var lidoArmABIData []byte

//go:embed abis/base_asset_configs_v2.json
var baseAssetConfigsV2ABIData []byte
