package mento

import _ "embed"

//go:embed abi/FPMMFactory.json
var factoryABIData []byte

//go:embed abi/FPMM.json
var poolABIData []byte

//go:embed abi/OracleAdapter.json
var oracleAdapterABIData []byte

//go:embed abi/SortedOracles.json
var sortedOraclesABIData []byte

//go:embed abi/BreakerBox.json
var breakerBoxABIData []byte

//go:embed abi/MarketHoursBreaker.json
var marketHoursBreakerABIData []byte
