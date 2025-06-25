package aavev3

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/source/liquiditysource"
	"github.com/ethereum/go-ethereum/common"
)

type Pool struct {
	ID                        string
	Address                   common.Address
	Token0                    common.Address
	Token1                    common.Address
	Reserve0                  *big.Int
	Reserve1                  *big.Int
	AToken0                   common.Address
	AToken1                   common.Address
	VariableDebtToken0        common.Address
	VariableDebtToken1        common.Address
	LiquidityIndex            *big.Int
	VariableBorrowIndex       *big.Int
	CurrentLiquidityRate      *big.Int
	CurrentVariableBorrowRate *big.Int
	LastUpdateTimestamp       uint32
	Fee                       uint32
	Extra                     string
}

type PoolExtra struct {
	Reserve0                  *big.Int `json:"reserve0"`
	Reserve1                  *big.Int `json:"reserve1"`
	AToken0                   string   `json:"aToken0"`
	AToken1                   string   `json:"aToken1"`
	VariableDebtToken0        string   `json:"variableDebtToken0"`
	VariableDebtToken1        string   `json:"variableDebtToken1"`
	LiquidityIndex            *big.Int `json:"liquidityIndex"`
	VariableBorrowIndex       *big.Int `json:"variableBorrowIndex"`
	CurrentLiquidityRate      *big.Int `json:"currentLiquidityRate"`
	CurrentVariableBorrowRate *big.Int `json:"currentVariableBorrowRate"`
	LastUpdateTimestamp       uint32   `json:"lastUpdateTimestamp"`
}

type ReserveData struct {
	CurrentLiquidityRate      *big.Int
	CurrentVariableBorrowRate *big.Int
	CurrentStableBorrowRate   *big.Int
	LiquidityIndex            *big.Int
	VariableBorrowIndex       *big.Int
	LastUpdateTimestamp       uint32
}

type SimulateSwapParams struct {
	TokenIn   common.Address
	TokenOut  common.Address
	AmountIn  *big.Int
	AmountOut *big.Int
	IsExactIn bool
	Pool      *Pool
}

type SimulateSwapResult struct {
	AmountIn  *big.Int
	AmountOut *big.Int
	Fee       *big.Int
	Gas       uint64
}

type PoolSimulator struct {
	config *Config
}

type PoolListUpdater struct {
	config *Config
}

type PoolTracker struct {
	config *Config
}

type LogDecoder struct {
	config *Config
}

var _ liquiditysource.PoolSimulator = (*PoolSimulator)(nil)
var _ liquiditysource.PoolListUpdater = (*PoolListUpdater)(nil)
var _ liquiditysource.PoolTracker = (*PoolTracker)(nil)
var _ liquiditysource.LogDecoder = (*LogDecoder)(nil)
