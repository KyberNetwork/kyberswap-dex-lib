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

	collaterals []*entity.PoolToken
}
