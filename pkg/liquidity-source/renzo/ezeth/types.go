package ezeth

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolExtra struct {
	Paused                bool `json:"paused"`
	StrategyManagerPaused bool `json:"strategyManagerPaused"`

	CollateralTokenIndex map[string]int `json:"collateralTokenIndex"`

	// RestakeManager.calculateTVLs
	OperatorDelegatorTokenTVLs [][]*big.Int `json:"operatorDelegatorTokenTvls"`
	OperatorDelegatorTVLs      []*big.Int   `json:"operatorDelegatorTvls"`
	TotalTVL                   *big.Int     `json:"totalTvl"`

	// RestakeManager.chooseOperatorDelegatorForDeposit
	OperatorDelegatorAllocations []*big.Int `json:"operatorDelegatorAllocations"`

	// OperatorDelegator.tokenStrategyMapping
	TokenStrategyMapping []map[string]bool `json:"tokenStrategyMapping"`

	// ezETH.totalSupply
	TotalSupply *big.Int `json:"totalSupply"`

	// RestakeManager.maxDepositTVL
	MaxDepositTVL *big.Int `json:"maxDepositTvl"`

	// renzoOracle.tokenOracleLookup
	TokenOracleLookup map[string]Oracle `json:"tokenOracleLookup"`

	CollateralTokenTvlLimits map[string]*big.Int `json:"collateralTokenTvlLimits"`

	collaterals []*entity.PoolToken
}
