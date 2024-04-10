package ezeth

import _ "embed"

//go:embed abis/RestakeManager.json
var restakeManagerABIJson []byte

//go:embed abis/RenzoOracle.json
var renzoOracleABIJson []byte

//go:embed abis/PriceFeed.json
var priceFeedABIJson []byte
