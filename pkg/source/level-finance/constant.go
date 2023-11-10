package levelfinance

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"math/big"
)

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
	liquidityPoolMethodLiquidityCalculator  = "liquidityCalculator"
	liquidityPoolMethodDaoFee               = "daoFee"
	liquidityPoolMethodLiquidationFee       = "liquidationFee"
	liquidityPoolMethodPositionFee          = "positionFee"

	oracleMethodGetPrice = "getPrice"

	liquidityCalculatorMethodBaseSwapFee             = "baseSwapFee"
	liquidityCalculatorMethodStableCoinBaseSwapFee   = "stableCoinBaseSwapFee"
	liquidityCalculatorMethodStableCoinTaxBasisPoint = "stableCoinTaxBasisPoint"
	liquidityCalculatorMethodTaxBasicPoint           = "taxBasisPoint"
)

var (
	DefaultGas = Gas{Swap: 125000}

	precision  = bignumber.TenPowInt(10)
	minSwapFee = big.NewInt(10000000)
)
