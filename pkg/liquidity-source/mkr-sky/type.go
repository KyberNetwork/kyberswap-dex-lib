package mkr_sky

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolItem struct {
	ID      string             `json:"id"`
	Type    string             `json:"type"`
	LpToken string             `json:"lpToken"`
	Tokens  []entity.PoolToken `json:"tokens"`
}

type StaticExtra struct {
	Rate *big.Int `json:"rate"`
}
