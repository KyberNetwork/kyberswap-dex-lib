package mkr_sky

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

type StaticExtra struct {
	Rate *uint256.Int `json:"rate"`
}
