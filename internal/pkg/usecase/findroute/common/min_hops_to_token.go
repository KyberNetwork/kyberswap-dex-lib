package common

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/oleiade/lane"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
)

// MinHopsToTokenOut perform BFS from tokenOut
func MinHopsToTokenOut(
	poolByAddress map[string]poolpkg.IPoolSimulator,
	tokenByAddress map[string]*entity.Token,
	tokenToPoolAddresses map[string]*types.AddressList,
	tokenOut string,
) (map[string]uint32, error) {
	var (
		minHop = make(map[string]uint32)
		queue  = lane.NewQueue()
		pool   poolpkg.IPoolSimulator
		ok     bool
	)

	minHop[tokenOut] = 0
	queue.Enqueue(tokenOut)

	for !queue.Empty() {
		var token = queue.Dequeue().(string)
		//no pool from this token
		if tokenToPoolAddresses[token] == nil {
			continue
		}
		for i := 0; i < tokenToPoolAddresses[token].TrueLen; i++ {
			poolAddress := tokenToPoolAddresses[token].Arr[i]
			//the adjacent map might include pools not in this particular bucket
			if pool, ok = poolByAddress[poolAddress]; !ok {
				continue
			}
			for _, tokenTo := range pool.CanSwapTo(token) {
				// must-have info for token on path
				if _, ok = tokenByAddress[tokenTo]; !ok {
					continue
				}
				if _, alreadyFound := minHop[tokenTo]; alreadyFound {
					continue
				}
				minHop[tokenTo] = minHop[token] + 1
				queue.Enqueue(tokenTo)
			}
		}
	}
	return minHop, nil
}

// MinHopsToTokenOutWithWhitelist performs BFS from `tokenOut`
// only considering `tokenIn` and "hop tokens" that are in the whitelist.
func MinHopsToTokenOutWithWhitelist(
	poolByAddress map[string]poolpkg.IPoolSimulator,
	tokenByAddress map[string]*entity.Token,
	tokenToPoolAddresses map[string]*types.AddressList,
	whitelistedHopTokens map[string]bool,
	tokenIn string,
	tokenOut string,
) (map[string]uint32, error) {
	var (
		minHop = make(map[string]uint32)
		queue  = lane.NewQueue()
		pool   poolpkg.IPoolSimulator
		ok     bool
	)

	minHop[tokenOut] = 0
	queue.Enqueue(tokenOut)

	for !queue.Empty() {
		var token = queue.Dequeue().(string)
		if tokenToPoolAddresses[token] == nil {
			//there is no adjacent from this token
			continue
		}
		for i := 0; i < tokenToPoolAddresses[token].TrueLen; i++ {
			poolAddress := tokenToPoolAddresses[token].Arr[i]
			if pool, ok = poolByAddress[poolAddress]; !ok {
				//this pool might not be available in current bucket
				continue
			}
			for _, tokenTo := range pool.CanSwapTo(token) {
				// must-have info for token on path
				if _, ok = tokenByAddress[tokenTo]; !ok {
					continue
				}

				if _, alreadyFound := minHop[tokenTo]; alreadyFound {
					continue
				}

				isHopToken := tokenTo != tokenIn
				if isHopToken && !whitelistedHopTokens[tokenTo] {
					continue
				}

				minHop[tokenTo] = minHop[token] + 1
				queue.Enqueue(tokenTo)
			}
		}
	}

	return minHop, nil
}
