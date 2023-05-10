package core

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	poolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

var (
	ErrInvalidPool = errors.New("invalid pool")
	ErrInvalidSwap = errors.New("invalid swap")
)

type Route struct {
	Input         poolpkg.TokenAmount
	Output        poolpkg.TokenAmount
	Paths         []Path
	Pools         []poolpkg.IPool
	OriginalPools []poolpkg.IPool
	TotalGas      int64

	MidPrice    *big.Int
	PriceImpact *big.Int
}

func NewRoute(
	pools []poolpkg.IPool,
	originalPools []poolpkg.IPool,
	input poolpkg.TokenAmount,
	outputToken string,
	paths []Path,
) *Route {
	var route = Route{
		Input: input,
		Output: poolpkg.TokenAmount{
			Token:  outputToken,
			Amount: big.NewInt(0),
		},
		Paths:         paths,
		Pools:         pools,
		OriginalPools: originalPools,
	}

	return &route
}

func NewEmptyRouteFromPoolData(
	tokenInAddress string,
	tokenOutAddress string,
	poolByAddress map[string]poolpkg.IPool,
) *Route {
	poolLists := make([]poolpkg.IPool, 0, len(poolByAddress))
	for _, pool := range poolByAddress {
		poolLists = append(poolLists, pool)
	}
	// TODO reimplement core.Path in findroute package
	// disregard original pool here (setting nil) since Finalize Route is not used
	return &Route{
		Pools: poolLists,
		Input: poolpkg.TokenAmount{
			Token:     tokenInAddress,
			Amount:    big.NewInt(0),
			AmountUsd: 0,
		},
		Output: poolpkg.TokenAmount{
			Token:     tokenOutAddress,
			Amount:    constant.Zero,
			AmountUsd: 0,
		},
		Paths:         nil,
		OriginalPools: nil,
	}
}

func NewRouteFromPaths(
	tokenInAddress string,
	tokenOutAddress string,
	poolByAddress map[string]poolpkg.IPool,
	paths []*Path,
) *Route {
	pathsDeref := make([]Path, len(paths))
	for i, path := range paths {
		pathsDeref[i] = *path
	}
	poolLists := make([]poolpkg.IPool, 0, len(poolByAddress))
	for _, pool := range poolByAddress {
		poolLists = append(poolLists, pool)
	}
	var route = Route{
		Input: poolpkg.TokenAmount{
			Token:     tokenInAddress,
			Amount:    big.NewInt(0),
			AmountUsd: 0,
		},
		Output: poolpkg.TokenAmount{
			Token:     tokenOutAddress,
			Amount:    big.NewInt(0),
			AmountUsd: 0,
		},
		Paths:         pathsDeref,
		Pools:         poolLists,
		OriginalPools: nil,
	}
	for _, path := range paths {
		route.Input.Amount = new(big.Int).Add(route.Input.Amount, path.Input.Amount)
		route.Input.AmountUsd += path.Input.AmountUsd
		route.Output.Amount = new(big.Int).Add(route.Output.Amount, path.Output.Amount)
		route.Output.AmountUsd += path.Output.AmountUsd
		route.TotalGas += path.TotalGas
	}
	return &route
}

func (r *Route) AddPath(path *Path) bool {
	if r.Input.Token != path.Input.Token || r.Output.Token != path.Output.Token {
		return false
	}
	var tokenAmountIn = path.Input
	var isOk = true
	for i := range path.Pools {
		var poolIndex = -1
		for id, pool := range r.Pools {
			if pool.Equals(path.Pools[i]) {
				poolIndex = id
				break
			}
		}
		if poolIndex < 0 {
			isOk = false
			break
		}
		var pool = r.Pools[poolIndex]
		calcAmountOutResult, err := pool.CalcAmountOut(tokenAmountIn, path.Tokens[i+1].Address)
		if err != nil {
			fmt.Printf(
				"PoolAddress: %v, PoolExchange: %v, CalcAmountOut[ tokenAmountIn: %v, tokenOut: %s, error: %v]\n",
				pool.GetAddress(), pool.GetExchange(), tokenAmountIn, path.Tokens[i+1].Address, err,
			)
			isOk = false
			break
		}
		if calcAmountOutResult.TokenAmountOut == nil || calcAmountOutResult.TokenAmountOut.Amount.Cmp(constant.Zero) <= 0 {
			fmt.Printf(
				"PoolAddress: %v, PoolExchange: %v, CalcAmountOut[ tokenAmountIn: %v, tokenOut: %s, error: tokenAmountOut %v is not correct]\n",
				pool.GetAddress(), pool.GetExchange(), tokenAmountIn, path.Tokens[i+1].Address, calcAmountOutResult.TokenAmountOut,
			)
			isOk = false
			break
		}

		tokenAmountOut, fee := calcAmountOutResult.TokenAmountOut, calcAmountOutResult.Fee

		updateBalanceParams := poolpkg.UpdateBalanceParams{
			TokenAmountIn:  tokenAmountIn,
			TokenAmountOut: *tokenAmountOut,
			Fee:            *fee,
			SwapInfo:       calcAmountOutResult.SwapInfo,
		}
		r.Pools[poolIndex].UpdateBalance(updateBalanceParams)
		tokenAmountIn = *tokenAmountOut
	}
	if !isOk {
		return false
	}
	var merged = false
	for i := range r.Paths {
		if r.Paths[i].Merge(path) {
			merged = true
			break
		}
	}
	if !merged {
		r.Paths = append(r.Paths, *path)
	}
	r.Input.Amount = new(big.Int).Add(r.Input.Amount, path.Input.Amount)
	r.Input.AmountUsd += path.Input.AmountUsd
	r.Output.Amount = new(big.Int).Add(r.Output.Amount, path.Output.Amount)
	r.Output.AmountUsd += path.Output.AmountUsd

	return true
}

func (r *Route) Summarize(pools []poolpkg.IPool) (valueobject.Route, error) {
	poolMap := make(map[string]poolpkg.IPool, len(r.OriginalPools))
	for _, pool := range pools {
		poolMap[pool.GetAddress()] = pool
	}

	routeAmountIn := big.NewInt(0)
	routeAmountOut := big.NewInt(0)
	var routeGas int64

	resultRoute := make([][]valueobject.Swap, 0, len(r.Paths))

	for _, path := range r.Paths {
		routeAmountIn = new(big.Int).Add(routeAmountIn, path.Input.Amount)
		routeGas += path.TotalGas

		resultPath := make([]valueobject.Swap, 0, len(path.Pools))
		tokenAmountIn := path.Input
		for idx, pathPool := range path.Pools {
			pool, ok := poolMap[pathPool.GetAddress()]
			if !ok {
				return valueobject.Route{}, ErrInvalidPool
			}

			calcAmountOutResult, err := pool.CalcAmountOut(tokenAmountIn, path.Tokens[idx+1].Address)
			if err != nil {
				fmt.Println(err)
				return valueobject.Route{}, ErrInvalidSwap
			}
			tokenAmountOut, fee := calcAmountOutResult.TokenAmountOut, calcAmountOutResult.Fee
			if tokenAmountOut == nil || tokenAmountOut.Amount == nil || tokenAmountOut.Amount.Cmp(constant.Zero) <= 0 {
				return valueobject.Route{}, ErrInvalidSwap
			}

			resultPath = append(
				resultPath,
				valueobject.Swap{
					Pool:              pool.GetAddress(),
					TokenIn:           tokenAmountIn.Token,
					TokenOut:          tokenAmountOut.Token,
					SwapAmount:        tokenAmountIn.Amount,
					LimitReturnAmount: constant.Zero,
					AmountOut:         tokenAmountOut.Amount,
					Exchange:          valueobject.Exchange(pool.GetExchange()),
					PoolLength:        len(pool.GetTokens()),
					PoolType:          pool.GetType(),
					PoolExtra:         pool.GetMetaInfo(tokenAmountIn.Token, tokenAmountOut.Token),
					Extra:             calcAmountOutResult.SwapInfo,
				},
			)

			updateBalanceParams := poolpkg.UpdateBalanceParams{
				TokenAmountIn:  tokenAmountIn,
				TokenAmountOut: *tokenAmountOut,
				Fee:            *fee,
			}
			pool.UpdateBalance(updateBalanceParams)
			tokenAmountIn = *tokenAmountOut
		}
		routeAmountOut = new(big.Int).Add(routeAmountOut, tokenAmountIn.Amount)
		resultRoute = append(resultRoute, resultPath)
	}

	r.Input.Amount = routeAmountIn
	r.Output.Amount = routeAmountOut
	r.TotalGas = routeGas

	return valueobject.Route{
		InputAmount:  routeAmountIn,
		OutputAmount: routeAmountOut,
		TotalGas:     routeGas,
		Route:        resultRoute,
	}, nil
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

// ToCachedRoute transform Route to CachedRoute
func (r *Route) ToCachedRoute() (CachedRoute, error) {
	paths := make([]CachedPath, 0, len(r.Paths))
	for _, path := range r.Paths {
		poolIDs := make([]string, 0, len(path.Pools))
		for _, pool := range path.Pools {
			poolIDs = append(poolIDs, pool.GetAddress())
		}

		paths = append(paths, CachedPath{
			Input:       path.Input,
			Output:      path.Output,
			TotalGas:    path.TotalGas,
			PoolIDs:     poolIDs,
			Tokens:      path.Tokens,
			PriceImpact: path.PriceImpact,
		})
	}

	return CachedRoute{
		Input: r.Input,
		Paths: paths,
	}, nil
}

func (r *Route) ExtractPoolAddresses() []string {
	poolAddressSet := make(map[string]struct{})

	for _, path := range r.Paths {
		for _, pool := range path.Pools {
			poolAddressSet[pool.GetAddress()] = struct{}{}
		}
	}

	poolAddresses := make([]string, 0, len(poolAddressSet))
	for poolAddress := range poolAddressSet {
		poolAddresses = append(poolAddresses, poolAddress)
	}

	return poolAddresses
}
