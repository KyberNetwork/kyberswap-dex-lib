package syncswapv2classic

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap/syncswapclassic"
)

func NewPoolSimulator(entityPool entity.Pool) (*syncswapclassic.PoolSimulator, error) {
	return syncswapclassic.NewPoolSimulator(entityPool)
}
