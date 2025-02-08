package ezeth

import (
	"errors"
	"math/big"
)

const (
	DexType = "renzo-ezeth"

	RestakeManager  = "0x74a09653a083691711cf8215a6ab074bb4e99ef5"
	EzEthToken      = "0xbf5495efe5db9ce00f80364c8b423567e58d2110"
	WETH            = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	StrategyManager = "0x858646372cc42e1a627fce94aa7a7033e7cf075a"
)

const (
	// unlimited reserve
	defaultReserves = "10000000000000000000"
)

const (
	EzEthTokenMethodTotalSupply = "totalSupply"

	RestakeManagerMethodCalculateTVLs             = "calculateTVLs"
	RestakeManagerMethodCollateralTokenTvlLimits  = "collateralTokenTvlLimits"
	RestakeManagerMethodCollateralTokens          = "collateralTokens"
	RestakeManagerMethodGetCollateralTokensLength = "getCollateralTokensLength"
	RestakeManagerMethodMaxDepositTVL             = "maxDepositTVL"
	RestakeManagerMethodPaused                    = "paused"
	RestakeManagerMethodRenzoOracle               = "renzoOracle"

	RenzoOracleMethodTokenOracleLookUp = "tokenOracleLookup"
	TokenOracleMethodLatestRoundData   = "latestRoundData"

	RestakeManagerMethodGetOperatorDelegatorsLength  = "getOperatorDelegatorsLength"
	RestakeManagerMethodOperatorDelegators           = "operatorDelegators"
	RestakeManagerMethodOperatorDelegatorAllocations = "operatorDelegatorAllocations"

	OperatorDelegatorMethodTokenStrategyMapping = "tokenStrategyMapping"

	StrategyManagerMethodPaused = "paused"
)

var (
	ErrInvalidCollateral      = errors.New("invalid collateral")
	ErrInvalidTokenOut        = errors.New("invalid tokenOut")
	ErrMaxTVLReached          = errors.New("max tvl reached")
	ErrMaxTokenTVLReached     = errors.New("max token tvl reached")
	ErrInvalidTokenAmount     = errors.New("invalid tokenAmount")
	ErrOracleNotFound         = errors.New("oracle not found")
	ErrOracleExpired          = errors.New("oracle expired")
	ErrInvalidOraclePrice     = errors.New("invalid oracle price")
	ErrPoolPaused             = errors.New("pool paused")
	ErrStrategyManagerPaused  = errors.New("strategy manager paused")
	ErrRevertNotFound         = errors.New("revert not found")
	ErrRevertInvalidZeroInput = errors.New("revert invalid zero input")
)

var (
	// Scale factor for all values of prices
	SCALE_FACTOR = big.NewInt(1e18)
	// The maxmimum staleness allowed for a price feed from chainlink
	MAX_TIME_WINDOW int64 = 86400 + 60 // 24 hours + 60 seconds
)
