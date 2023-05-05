package token

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

type IFallbackRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Token, error)
}
