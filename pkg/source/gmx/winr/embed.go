package winr

import _ "embed"

//go:embed abis/Vault.json
var vaultJson []byte

//go:embed abis/PriceOracleRouter.json
var priceOracleRouterJson []byte
