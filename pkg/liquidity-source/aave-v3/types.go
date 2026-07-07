package aavev3

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Extra struct {
	IsActive bool `json:"isActive,omitempty"`
	IsFrozen bool `json:"isFrozen,omitempty"`
	IsPaused bool `json:"isPaused,omitempty"`
}

type StaticExtra struct {
	AavePoolAddress string `json:"aavePoolAddress"`
}

type RPCReserveData struct {
	Data struct {
		Configuration               ReserveConfigurationMap
		LiquidityIndex              *big.Int
		CurrentLiquidityRate        *big.Int
		VariableBorrowIndex         *big.Int
		CurrentVariableBorrowRate   *big.Int
		CurrentStableBorrowRate     *big.Int
		LastUpdateTimestamp         *big.Int
		ID                          uint16
		ATokenAddress               common.Address
		StableDebtTokenAddress      common.Address
		VariableDebtTokenAddress    common.Address
		InterestRateStrategyAddress common.Address
		AccruedToTreasury           *big.Int
		Unbacked                    *big.Int
		IsolationModeTotalDebt      *big.Int
	}
}

type RPCConfiguration struct {
	Configuration struct {
		Data ReserveConfigurationMap
	}
	BlockNumber uint64
}

type ReserveConfigurationMap struct {
	Data *big.Int
}

type SwapInfo struct {
	IsSupply        bool   `json:"isSupply"`
	AavePoolAddress string `json:"aavePoolAddress"`
}

type PoolMeta struct {
	BlockNumber uint64
}
