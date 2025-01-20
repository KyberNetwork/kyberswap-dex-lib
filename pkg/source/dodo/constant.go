package dodo

import (
	"math/big"
)

const (
	DexTypeDodo = "dodo"

	PoolTypeDodoClassical      = "dodo-classical"
	PoolTypeDodoVendingMachine = "dodo-dvm"
	PoolTypeDodoStable         = "dodo-dsp"
	PoolTypeDodoPrivate        = "dodo-dpp"

	// SubgraphPoolType DodoV1
	subgraphPoolTypeDodoClassical = "CLASSICAL"
	// SubgraphPoolType DodoV2
	subgraphPoolTypeDodoVendingMachine = "DVM"
	subgraphPoolTypeDodoStable         = "DSP"
	subgraphPoolTypeDodoPrivate        = "DPP"

	// Contract methods
	poolMethodGetExpectedTarget = "getExpectedTarget"
	poolMethodK                 = "_K_"
	poolMethodRStatus           = "_R_STATUS_"
	poolMethodGetOraclePrice    = "getOraclePrice"
	poolMethodLpFeeRate         = "_LP_FEE_RATE_"
	poolMethodMtFeeRate         = "_MT_FEE_RATE_"
	poolMethodBaseBalance       = "_BASE_BALANCE_"
	poolMethodQuoteBalance      = "_QUOTE_BALANCE_"
	poolMethodTradeAllowed      = "_TRADE_ALLOWED_"

	poolMethodGetPMMStateForCall = "getPMMStateForCall"
	poolMethodGetUserFeeRate     = "getUserFeeRate"

	defaultTokenWeight   = 50
	defaultTokenDecimals = 18

	zeroString = "0"

	TypeV1Pool = "CLASSICAL"

	rStatusOne      = 0
	rStatusAboveOne = 1
	rStatusBelowOne = 2
)

var (
	oneBF, _ = new(big.Float).SetString("1000000000000000000")

	zeroBI = big.NewInt(0)

	subgraphTypeToPoolTypeMap = map[string]string{
		subgraphPoolTypeDodoClassical:      PoolTypeDodoClassical,
		subgraphPoolTypeDodoStable:         PoolTypeDodoStable,
		subgraphPoolTypeDodoVendingMachine: PoolTypeDodoVendingMachine,
		subgraphPoolTypeDodoPrivate:        PoolTypeDodoPrivate,
	}

	DefaultGas = Gas{
		SellBaseV1: 170000,
		BuyBaseV1:  224000,
		SellBaseV2: 128000,
		BuyBaseV2:  116000,
	}
)
