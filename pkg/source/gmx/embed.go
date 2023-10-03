package gmx

import _ "embed"

//go:embed abis/ChainlinkFlags.json
var chainlinkFlagsJson []byte

//go:embed abis/FastPriceFeedV1.json
var fastPriceFeedV1Json []byte

//go:embed abis/FastPriceFeedV2.json
var fastPriceFeedV2Json []byte

//go:embed abis/PancakePair.json
var pancakePairJson []byte

//go:embed abis/PriceFeed.json
var priceFeedJson []byte

//go:embed abis/Vault.json
var vaultJson []byte

//go:embed abis/VaultPriceFeed.json
var vaultPriceFeedJson []byte

//go:embed abis/ERC20.json
var erc20Json []byte
