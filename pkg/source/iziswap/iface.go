package iziswap

import "context"

type IClient interface {
	ListPools(ctx context.Context, params ListPoolsParams) ([]PoolInfo, error)
}
