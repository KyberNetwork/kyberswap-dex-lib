package generic_simple_rate

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/holiman/uint256"
)

type PoolItem struct {
	ID      string             `json:"id"`
	Type    string             `json:"type"`
	LpToken string             `json:"lpToken"`
	Tokens  []entity.PoolToken `json:"tokens"`
}

type PoolExtra struct {
	Paused          bool         `json:"paused"`
	Rate            *uint256.Int `json:"rate"`
	RateUnit        *uint256.Int `json:"rateUnit"`
	IsRateInversed  bool         `json:"isRateInversed"`
	IsBidirectional bool         `json:"isBidirectional"`
	DefaultGas      int64        `json:"defaultGas"`
}
