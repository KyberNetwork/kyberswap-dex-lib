package ezeth

import _ "embed"

//go:embed abis/EzETHToken.json
var ezETHTokenABIJson []byte

//go:embed abis/RestakeManager.json
var restakeManagerABIJson []byte

//go:embed abis/RenzoOracle.json
var renzoOracleABIJson []byte

//go:embed abis/PriceFeed.json
var priceFeedABIJson []byte

//go:embed abis/StrategyManager.json
var strategyManagerABIJson []byte

//go:embed abis/OperatorDelegator.json
var operatorDelegatorABIJson []byte

//go:embed abis/TokenOracle.json
var tokenOracleABIJson []byte
