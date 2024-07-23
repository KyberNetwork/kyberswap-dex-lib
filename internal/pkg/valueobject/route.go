package valueobject

import (
	"context"
	"math/big"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/huandu/go-clone"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	routerpoolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

var (
	ErrPathMismatchedToken = errors.New("path does not have the same input and output token as route")
)

type ChunkInfo struct {
	AmountIn     *big.Int `json:"amountIn"`
	AmountOut    *big.Int `json:"amountOut"`
	AmountInUsd  float64  `json:"amountInUsd"`
	AmountOutUsd float64  `json:"amountOutUsd"`
}

type Route struct {
	Input    TokenAmount `json:"input"`
	Output   TokenAmount `json:"output"`
	Paths    []*Path     `json:"paths"`
	TotalGas int64       `json:"totalGas"`
}

func (r *Route) HasOnlyOneSwap() bool {
	if r.Paths == nil || len(r.Paths) != 1 {
		return false
	}

	if r.Paths[0] == nil || len(r.Paths[0].PoolAddresses) != 1 {
		return false
	}

	return true

}

func NewRoute(
	tokenInAddress string,
	tokenOutAddress string,
) *Route {
	return &Route{
		Input: TokenAmount{
			Token:          tokenInAddress,
			Amount:         big.NewInt(0),
			AmountAfterGas: big.NewInt(0),
			AmountUsd:      0,
		},
		Output: TokenAmount{
			Token:          tokenOutAddress,
			Amount:         big.NewInt(0),
			AmountAfterGas: big.NewInt(0),
			AmountUsd:      0,
		},
		Paths: nil,
	}
}

func NewRouteFromPaths(
	tokenInAddress string,
	tokenOutAddress string,
	paths []*Path,
) *Route {
	var route = NewRoute(tokenInAddress, tokenOutAddress)
	route.Paths = paths

	for _, path := range paths {
		route.Input.Amount = new(big.Int).Add(route.Input.Amount, path.Input.Amount)
		route.Input.AmountUsd += path.Input.AmountUsd
		route.Output.Amount.Add(route.Output.Amount, path.Output.Amount)
		if path.Output.AmountAfterGas != nil {
			route.Output.AmountAfterGas.Add(route.Output.AmountAfterGas, path.Output.AmountAfterGas)
		}
		route.Output.AmountUsd += path.Output.AmountUsd
		route.TotalGas += path.TotalGas
	}
	return route
}

// AddPath will add the path into Route.
// it will also modify request's copy of IPool( poolByAddress). Once the Path is added,
// the poolByAddress of the modified pool will be assigned to a different pointer to avoid changing data of other's request
func (r *Route) AddPath(ctx context.Context, poolBucket *PoolBucket, p *Path, swapLimits map[string]poolpkg.SwapLimit) (fErr error) {
	if r.Input.Token != p.Input.Token || r.Output.Token != p.Output.Token {
		return errors.WithMessagef(
			ErrPathMismatchedToken,
			"[Route.AddPath] Expecting tokenInAddress: [%s] , tokenOutAddress: [%s] | Received tokenInAddress: [%s] , tokenOutAddress: [%s] ",
			r.Input.Token, r.Output.Token, p.Input.Token, p.Output.Token,
		)
	}

	var (
		currentAmount = *p.Input.ToDexLibAmount()
		pool          poolpkg.IPoolSimulator
		ok            bool
		backUpPools   = make([]poolpkg.IPoolSimulator, len(p.PoolAddresses))
	)
	defer func() {
		if fErr != nil {
			poolBucket.RollBackPools(backUpPools)
		}
	}()
	for i, poolAddress := range p.PoolAddresses {
		if pool, ok = poolBucket.GetPool(poolAddress); !ok {
			fErr = errors.WithMessagef(
				ErrNoIPool,
				"[Route.AddPath] poolAddress: [%s]",
				poolAddress,
			)
			return fErr
		}
		swapLimit := swapLimits[pool.GetType()]

		calcAmountOutResult, err := routerpoolpkg.CalcAmountOut(ctx, pool, currentAmount, p.Tokens[i+1].Address, swapLimit)
		if err != nil {
			fErr = errors.WithMessagef(
				ErrInvalidSwap,
				"[Route.AddPath] CalcAmountOut returns error | poolAddress: [%s], exchange: [%s], tokenIn: [%s], amountIn: [%s], tokenOut: [%s], err: [%v]",
				poolAddress, pool.GetExchange(), currentAmount.Token, currentAmount.Amount, p.Tokens[i+1].Address, err,
			)
			return fErr
		}
		if calcAmountOutResult.TokenAmountOut == nil || calcAmountOutResult.TokenAmountOut.Amount.Cmp(constant.Zero) <= 0 {
			fErr = errors.WithMessagef(
				ErrInvalidSwap,
				"[Route.AddPath] CalcAmountOut returns nil or invalid amountOut | poolAddress: [%s], exchange: [%s], tokenIn: [%s], amountIn: [%s], tokenOut: [%s], tokenAmountOut: [%v]", pool.GetAddress(), pool.GetExchange(), currentAmount.Token, currentAmount.Amount, p.Tokens[i+1].Address, calcAmountOutResult.TokenAmountOut,
			)
			return fErr
		}

		tokenAmountOut, fee := calcAmountOutResult.TokenAmountOut, calcAmountOutResult.Fee

		updateBalanceParams := poolpkg.UpdateBalanceParams{
			TokenAmountIn:  currentAmount,
			TokenAmountOut: *tokenAmountOut,
			Fee:            *fee,
			SwapInfo:       calcAmountOutResult.SwapInfo,
			SwapLimit:      swapLimit,
		}

		//backing up the pool if there were error and we need to roll back
		backUpPools[i] = clone.Slowly(pool).(poolpkg.IPoolSimulator)
		// clone the pool before updating it, so it doesn't modify the original data copied from pool manager
		pool = poolBucket.ClonePool(poolAddress)

		// modify our copy
		pool.UpdateBalance(updateBalanceParams)
		currentAmount = *tokenAmountOut
	}

	//no more error from here
	var merged = false
	for i := range r.Paths {
		if r.Paths[i].Merge(p) {
			merged = true
			break
		}
	}
	if !merged {
		r.Paths = append(r.Paths, p)
	}

	r.Input.Amount = new(big.Int).Add(r.Input.Amount, p.Input.Amount)
	r.Input.AmountUsd += p.Input.AmountUsd
	r.Output.Amount.Add(r.Output.Amount, p.Output.Amount)
	if p.Output.AmountAfterGas != nil {
		r.Output.AmountAfterGas.Add(r.Output.AmountAfterGas, p.Output.AmountAfterGas)
	}
	r.Output.AmountUsd += p.Output.AmountUsd

	return nil
}

func (r *Route) CompareTo(other *Route, gasInclude bool) int {
	if gasInclude {
		// compare amount in native unit if available
		if r.Output.AmountAfterGas != nil && other.Output.AmountAfterGas != nil &&
			r.Output.AmountAfterGas.Sign() != 0 && other.Output.AmountAfterGas.Sign() != 0 {
			return r.Output.AmountAfterGas.Cmp(other.Output.AmountAfterGas)
		}

		// otherwise use usd amount
		if !utils.Float64AlmostEqual(r.Output.AmountUsd, other.Output.AmountUsd) {
			if r.Output.Amount.Sign() != 0 && r.Output.AmountUsd > other.Output.AmountUsd {
				return 1
			}
			if other.Output.Amount.Sign() != 0 && r.Output.AmountUsd < other.Output.AmountUsd {
				return -1
			}
		}
	}
	return r.Output.Amount.Cmp(other.Output.Amount)
}

func (r *Route) ExtractPoolAddresses() sets.String {
	poolAddressSet := sets.NewString()

	for _, path := range r.Paths {
		for _, poolAddress := range path.PoolAddresses {
			poolAddressSet.Insert(poolAddress)
		}
	}

	return poolAddressSet
}
