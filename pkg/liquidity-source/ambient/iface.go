package ambient

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type IPoolDatastore interface {
	Get(ctx context.Context, address string) (entity.Pool, error)
}
