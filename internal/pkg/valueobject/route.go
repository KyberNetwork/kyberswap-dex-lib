package valueobject

import (
	"math/big"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

var (
	ErrPathMismatchedToken = errors.New("path does not have the same input and output token as route")
)

type Route struct {
	Input    poolpkg.TokenAmount `json:"input"`
	Output   poolpkg.TokenAmount `json:"output"`
	Paths    []*Path             `json:"paths"`
	TotalGas int64               `json:"totalGas"`
}

func NewRoute(
	tokenInAddress string,
	tokenOutAddress string,
) *Route {
	return &Route{
		Input: poolpkg.TokenAmount{
			Token:     tokenInAddress,
			Amount:    constant.Zero,
			AmountUsd: 0,
		},
		Output: poolpkg.TokenAmount{
			Token:     tokenOutAddress,
			Amount:    constant.Zero,
			AmountUsd: 0,
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
		route.Output.Amount = new(big.Int).Add(route.Output.Amount, path.Output.Amount)
		route.Output.AmountUsd += path.Output.AmountUsd
		route.TotalGas += path.TotalGas
	}
	return route
}

// AddPath will add the path into Route.
// it will also modify request's copy of IPool( poolByAddress). Once the Path is added,
// the poolByAddress of the modified pool will be assigned to a different pointer to avoid changing data of other's request
func (r *Route) AddPath(poolBucket *PoolBucket, p *Path, ivt *poolpkg.Inventory) error {
	if r.Input.Token != p.Input.Token || r.Output.Token != p.Output.Token {
		return errors.Wrapf(
			ErrPathMismatchedToken,
			"[Route.AddPath] Expecting tokenInAddress: [%s] , tokenOutAddress: [%s] | Received tokenInAddress: [%s] , tokenOutAddress: [%s] ",
			r.Input.Token, r.Output.Token, p.Input.Token, p.Output.Token,
		)
	}

	var (
		currentAmount = p.Input
		pool          poolpkg.IPoolSimulator
		ok            bool
	)

	for i, poolAddress := range p.PoolAddresses {
		if pool, ok = poolBucket.GetPool(poolAddress); !ok {
			return errors.Wrapf(
				ErrNoIPool,
				"[Route.AddPath] poolAddress: [%s]",
				poolAddress,
			)
		}
		calcAmountOutResult, err := poolpkg.CalcAmountOut(pool, currentAmount, p.Tokens[i+1].Address)
		if pool.GetType() == constant.PoolTypes.KyberPMM {

			if ivt.GetBalance(p.Tokens[i+1].Address).Cmp(calcAmountOutResult.TokenAmountOut.Amount) < 0 {
				return errors.Wrapf(ErrInvalidSwap,
					"[Route.AddPath] CalcAmountOut returns error | poolAddress: [%s], exchange: [%s], tokenIn: [%s], amountIn: [%s], tokenOut: [%s], err: [%v]",
					poolAddress, pool.GetExchange(), currentAmount.Token, currentAmount.Amount, p.Tokens[i+1].Address, errors.New("not enough inventory"))
			}
		}
		if err != nil {
			return errors.Wrapf(
				ErrInvalidSwap,
				"[Route.AddPath] CalcAmountOut returns error | poolAddress: [%s], exchange: [%s], tokenIn: [%s], amountIn: [%s], tokenOut: [%s], err: [%v]",
				poolAddress, pool.GetExchange(), currentAmount.Token, currentAmount.Amount, p.Tokens[i+1].Address, err,
			)
		}
		if calcAmountOutResult.TokenAmountOut == nil || calcAmountOutResult.TokenAmountOut.Amount.Cmp(constant.Zero) <= 0 {
			return errors.Wrapf(
				ErrInvalidSwap,
				"[Route.AddPath] CalcAmountOut returns nil or invalid amountOut | poolAddress: [%s], exchange: [%s], tokenIn: [%s], amountIn: [%s], tokenOut: [%s], tokenAmountOut: [%v]", pool.GetAddress(), pool.GetExchange(), currentAmount.Token, currentAmount.Amount, p.Tokens[i+1].Address, calcAmountOutResult.TokenAmountOut,
			)
		}

		tokenAmountOut, fee := calcAmountOutResult.TokenAmountOut, calcAmountOutResult.Fee

		updateBalanceParams := poolpkg.UpdateBalanceParams{
			TokenAmountIn:  currentAmount,
			TokenAmountOut: *tokenAmountOut,
			Fee:            *fee,
			SwapInfo:       calcAmountOutResult.SwapInfo,
			Inventory:      ivt,
		}
		// clone the pool before updating it, so it doesn't modify the original data copied from pool manager
		pool = poolBucket.ClonePool(poolAddress)

		// modify our copy
		pool.UpdateBalance(updateBalanceParams)
		currentAmount = *tokenAmountOut
	}

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
	r.Output.Amount = new(big.Int).Add(r.Output.Amount, p.Output.Amount)
	r.Output.AmountUsd += p.Output.AmountUsd

	return nil
}

func (r *Route) CompareTo(other *Route, gasInclude bool) int {
	if gasInclude {
		if r.Output.Amount.Cmp(constant.Zero) > 0 && r.Output.AmountUsd > other.Output.AmountUsd {
			return 1
		}
		if other.Output.Amount.Cmp(constant.Zero) > 0 && r.Output.AmountUsd < other.Output.AmountUsd {
			return -1
		}
	}
	return r.Output.Amount.Cmp(other.Output.Amount)
}

func (r *Route) ExtractPoolAddresses() []string {
	poolAddressSet := make(map[string]struct{})

	for _, path := range r.Paths {
		for _, poolAddress := range path.PoolAddresses {
			poolAddressSet[poolAddress] = struct{}{}
		}
	}

	poolAddresses := make([]string, 0, len(poolAddressSet))
	for poolAddress := range poolAddressSet {
		poolAddresses = append(poolAddresses, poolAddress)
	}

	return poolAddresses
}
