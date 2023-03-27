package core

import "context"

type IScanDex interface {
	InitPool(ctx context.Context) error
	UpdateNewPools(ctx context.Context)
	UpdateReserves(ctx context.Context)
	UpdateTotalSupply(ctx context.Context)
}
