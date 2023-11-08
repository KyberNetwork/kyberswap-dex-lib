package fxdx

import _ "embed"

//go:embed abis/Vault.json
var vaultJson []byte

//go:embed abis/ERC20.json
var erc20Json []byte

//go:embed abis/VaultPriceFeed.json
var vaultPriceFeedJson []byte

//go:embed abis/ChainlinkFlags.json
var chainlinkFlagsJson []byte

//go:embed abis/PancakePair.json
var pancakePairJson []byte

//go:embed abis/FastPriceFeed.json
var fastPriceFeedJson []byte

//go:embed abis/PriceFeed.json
var priceFeedJson []byte

//go:embed abis/FeeUtilsV2.json
var feeUtilsV2Json []byte
