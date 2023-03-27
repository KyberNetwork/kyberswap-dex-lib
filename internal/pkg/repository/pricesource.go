package repository

import (
	"context"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/config"
)

type IPriceSourceRepository interface {
	FetchPrice(ctx context.Context, cfg *config.Common, tokens []string) (map[string]float64, error)
}
