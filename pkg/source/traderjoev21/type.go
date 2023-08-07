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

func (r Reserves) GetPoolReserves() entity.PoolReserves {
	return entity.PoolReserves{
		r.ReserveX.String(),
		r.ReserveY.String(),
	}
}
