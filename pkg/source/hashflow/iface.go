package hashflow

import (
	"context"
)

type IClient interface {
	ListMarketMakers(ctx context.Context) ([]string, error)
	ListPriceLevels(ctx context.Context, marketMakers []string) ([]Pair, error)
}
