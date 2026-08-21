package synthereum

import _ "embed"

//go:embed abis/SynthereumMultiLpLiquidityPool.json
var multiLpPoolABIData []byte

//go:embed abis/ERC4626Vault.json
var vaultABIData []byte

//go:embed abis/SynthereumFinder.json
var finderABIData []byte

//go:embed abis/SynthereumPriceFeed.json
var priceFeedABIData []byte

//go:embed abis/SynthereumFixedRateLendingWrapper.json
var wrapperABIData []byte

//go:embed abis/SynthereumRegistry.json
var registryABIData []byte
