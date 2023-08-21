package traderjoev20

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// ReservesAndID https://github.com/traderjoe-xyz/joe-v2/blob/v2.0.0/src/LBPair.sol#L151
type ReservesAndID struct {
	ReserveX *big.Int
	ReserveY *big.Int
	//revive:disable:var-naming
	ActiveId *big.Int
}

// BinReserves https://github.com/traderjoe-xyz/joe-v2/blob/v2.0.0/src/LBPair.sol#L265
type BinReserves struct {
	ReserveX *big.Int
	ReserveY *big.Int
}

func (r ReservesAndID) GetPoolReserves() entity.PoolReserves {
	return entity.PoolReserves{
		r.ReserveX.String(),
		r.ReserveY.String(),
	}
}

type Extra struct {
	Liquidity *big.Int `json:"liquidity"`
	PriceX128 *big.Int `json:"priceX128"`
}
