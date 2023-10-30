package levelfinance

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"

const (
	DexTypeLevelFinance = "level-finance"

	defaultWeight = 1
	zeroString    = "0"

	liquidityPoolMethodAllAssets            = "allAssets"
	liquidityPoolMethodOracle               = "oracle"
	liquidityPoolMethodTotalWeight          = "totalWeight"
	liquidityPoolMethodVirtualPoolValue     = "virtualPoolValue"
	liquidityPoolMethodFee                  = "fee"
	liquidityPoolMethodIsStableCoin         = "isStableCoin"
	liquidityPoolMethodTargetWeights        = "targetWeights"
	liquidityPoolMethodTrancheAssets        = "trancheAssets"
	liquidityPoolMethodRiskFactor           = "riskFactor"
	liquidityPoolMethodTotalRiskFactor      = "totalRiskFactor"
	liquidityPoolMethodGetAllTranchesLength = "getAllTranchesLength"
	liquidityPoolMethodAllTranches          = "allTranches"
	liquidityPoolMethodGetAllTranches       = "getAllTranches"

	oracleMethodGetPrice = "getPrice"
)

var (
	DefaultGas = Gas{Swap: 125000}

	precision = bignumber.TenPowInt(10)
)
