package liquiditybookv21

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type bin struct {
	ID       *big.Int `json:"id"`
	ReserveX *big.Int `json:"reserveX"`
	ReserveY *big.Int `json:"reserveY"`
}

func (b *bin) isEmpty(swapForX bool) bool {
	if swapForX {
		return b.ReserveX.Cmp(bignumber.ZeroBI) == 0
	}
	return b.ReserveY.Cmp(bignumber.ZeroBI) == 0
}

func (b *bin) getAmounts(
	parameters *parameters,
	binStep uint16,
	swapForY bool,
	activeID uint32,
	amountsInLeft *big.Int,
) (*big.Int, *big.Int, *big.Int) {
	// price :=

	return nil, nil, nil
}
