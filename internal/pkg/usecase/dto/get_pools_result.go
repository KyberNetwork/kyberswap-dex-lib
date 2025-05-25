package dto

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

type (
	GetPoolsResult struct {
		Pools []*entity.Pool `json:"pools"`
	}
)

func NewGetPoolsResult(pools []*entity.Pool) *GetPoolsResult {
	return &GetPoolsResult{
		Pools: pools,
	}
}
