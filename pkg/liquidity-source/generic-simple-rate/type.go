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
	Rate            *uint256.Int `json:"rate"`
	RateUnit        *uint256.Int `json:"rateUnit"`
	Paused          bool         `json:"paused"`
	IsBidirectional bool         `json:"isBidirectional"`
	IsRateInversed  bool         `json:"isRateInversed"`
	DefaultGas      int64        `json:"defaultGas"`
}
