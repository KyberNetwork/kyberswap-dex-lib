package token

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type IFallbackRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Token, error)
}
