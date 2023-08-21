package traderjoev21

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// Reserves https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/LBPair.sol#L160
type Reserves struct {
	ReserveX *big.Int
	ReserveY *big.Int
}

// BinReserves https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/LBPair.sol#L181
type BinReserves struct {
	BinReserveX *big.Int
	BinReserveY *big.Int
}

func (r Reserves) GetPoolReserves() entity.PoolReserves {
	return entity.PoolReserves{
		r.ReserveX.String(),
		r.ReserveY.String(),
	}
}

type Extra struct {
	Liquidity *big.Int `json:"liquidity"`
	PriceX128 *big.Int `json:"priceX128"`
}
