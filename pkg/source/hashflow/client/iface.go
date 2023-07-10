package client

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/hashflow"
)

type IClient interface {
	ListMarketMakers(ctx context.Context) ([]string, error)
	ListPriceLevels(ctx context.Context, marketMakers []string) ([]hashflow.Pair, error)
}
