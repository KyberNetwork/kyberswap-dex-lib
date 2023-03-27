package service

import (
	"context"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
)

type CatalogToken struct {
	Source  string `json:"source"`
	Address string `json:"address"`
	ChainID string `json:"chainId"`
}

type ITokenCatalogRepository interface {
	Upsert(ctx context.Context, token CatalogToken) error
}

type ListOrdersFilter struct {
	ChainID             valueobject.ChainID
	MakerAsset          string
	TakerAsset          string
	ExcludeExpiredOrder bool
}

type ILimitOrderRepository interface {
	ListAllPairs(ctx context.Context, chainID valueobject.ChainID) ([]*valueobject.LimitOrderPair, error)
	ListOrders(ctx context.Context, filter ListOrdersFilter) ([]*valueobject.Order, error)
}
