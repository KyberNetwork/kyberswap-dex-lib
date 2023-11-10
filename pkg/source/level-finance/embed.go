package levelfinance

import _ "embed"

//go:embed abi/LiquidityPool.json
var LiquidityPoolABIBytes []byte

//go:embed abi/LevelOracle.json
var LevelOracleABIBytes []byte

//go:embed abi/LiquidityCalculator.json
var LiquidityCalculatorABIBytes []byte
