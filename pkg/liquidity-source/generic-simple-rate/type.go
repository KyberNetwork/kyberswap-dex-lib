package generic_simple_rate

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/holiman/uint256"
)

type PoolItem struct {
	Address  string             `json:"address"`
	Type     string             `json:"type"`
	Tokens   []entity.PoolToken `json:"tokens"`
	Exchange string             `json:"exchange"`
}

type PoolExtra struct {
	Rate            *uint256.Int `json:"rate"`
	RateUnit        *uint256.Int `json:"rateUnit"`
	Paused          bool         `json:"paused"`
	IsBidirectional bool         `json:"isBidirectional"`
	DefaultGas      int64        `json:"defaultGas"`
}
