package kyberpmm

import "context"

type IClient interface {
	ListTokens(ctx context.Context) (map[string]TokenItem, error)
	ListPairs(ctx context.Context) (map[string]PairItem, error)
	ListPriceLevels(ctx context.Context) (map[string]PriceItem, error)
	Firm(ctx context.Context, params FirmRequestParams) (FirmResult, error)
}
