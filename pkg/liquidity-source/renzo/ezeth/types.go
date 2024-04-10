package ezeth

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolExtra struct {
	Paused                     bool         `json:"paused"`
	OperatorDelegatorTokenTVLs [][]*big.Int `json:"operatorDelegatorTokenTvls"`
	OperatorDelegatorTVLs      []*big.Int   `json:"operatorDelegatorTvls"`
	TotalTVL                   *big.Int     `json:"totalTvl"`
	MaxDepositTVL              *big.Int     `json:"maxDepositTvl"`

	CollateralTokenIndex map[string]int `json:"collateralTokenIndex"`

	collaterals []*entity.PoolToken
}
