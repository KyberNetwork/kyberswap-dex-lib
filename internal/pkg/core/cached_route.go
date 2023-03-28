package core

import (
	"math/big"

	"github.com/pkg/errors"

	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

var (
	ErrZeroInputAmount = errors.New("cached route with input amount = 0")
	ErrPoolNotFound    = errors.New("pool not found")
)

type CachedRoute struct {
	Input poolPkg.TokenAmount `json:"input"`
	Paths []CachedPath        `json:"paths"`
}

type CachedPath struct {
	Input       poolPkg.TokenAmount `json:"input"`
	Output      poolPkg.TokenAmount `json:"output"`
	TotalGas    int64               `json:"totalGas"`
	PoolIDs     []string            `json:"poolIds"`
	Tokens      []entity.Token      `json:"tokens"`
	PriceImpact *big.Int            `json:"priceImpact"`
}

// ToRoute transforms CachedRoute to Route
func (r *CachedRoute) ToRoute(pools []poolPkg.IPool, originalPools []poolPkg.IPool) (*Route, error) {
	poolByAddress := make(map[string]poolPkg.IPool, len(pools))
	for _, pool := range pools {
		poolByAddress[pool.GetAddress()] = pool
	}

	paths := make([]Path, 0, len(r.Paths))
	for _, cachedRoutePath := range r.Paths {
		pathPools := make([]poolPkg.IPool, 0, len(cachedRoutePath.PoolIDs))
		for _, poolID := range cachedRoutePath.PoolIDs {
			pool, ok := poolByAddress[poolID]
			if !ok {
				return nil, errors.Wrapf(ErrPoolNotFound, "poolId: [%s]", poolID)
			}

			pathPools = append(pathPools, pool)
		}

		paths = append(paths, Path{
			Input:       cachedRoutePath.Input,
			Output:      cachedRoutePath.Output,
			Pools:       pathPools,
			TotalGas:    cachedRoutePath.TotalGas,
			Tokens:      cachedRoutePath.Tokens,
			PriceImpact: cachedRoutePath.PriceImpact,
		})
	}

	return &Route{
		Input:         r.Input,
		OriginalPools: originalPools,
		Paths:         paths,
	}, nil
}

// RedistributeInputAmount redistribute input amount of each path
func (r *CachedRoute) RedistributeInputAmount(amountIn *big.Int, tokenDecimals uint8, tokenPrice float64) error {
	if len(r.Input.Amount.Bits()) == 0 {
		return ErrZeroInputAmount
	}

	restInputAmount := amountIn

	for idx := range r.Paths {
		if idx == len(r.Paths)-1 {
			r.Paths[idx].Input.Amount = restInputAmount
			r.Paths[idx].Input.AmountUsd = utils.CalcTokenAmountUsd(restInputAmount, tokenDecimals, tokenPrice)
			continue
		}

		distributionRate := new(big.Float).Quo(
			new(big.Float).SetInt(r.Paths[idx].Input.Amount),
			new(big.Float).SetInt(r.Input.Amount),
		)

		pathInputAmount, _ := new(big.Float).Mul(
			new(big.Float).SetInt(amountIn),
			distributionRate,
		).Int(nil)

		r.Paths[idx].Input.Amount = pathInputAmount
		r.Paths[idx].Input.AmountUsd = utils.CalcTokenAmountUsd(pathInputAmount, tokenDecimals, tokenPrice)

		restInputAmount = new(big.Int).Sub(restInputAmount, pathInputAmount)
	}

	return nil
}
