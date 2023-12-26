package common

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/oleiade/lane"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
)

// MinHopsToTokenOut perform BFS from tokenOut
func MinHopsToTokenOut(
	poolByAddress map[string]poolpkg.IPoolSimulator,
	tokenByAddress map[string]entity.Token,
	tokenToPoolAddresses map[string][]string,
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
		for _, poolAddress := range tokenToPoolAddresses[token] {
			if pool, ok = poolByAddress[poolAddress]; !ok {
				return nil, findroute.ErrNoIPool
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
	tokenByAddress map[string]entity.Token,
	tokenToPoolAddresses map[string][]string,
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
		for _, poolAddress := range tokenToPoolAddresses[token] {
			if pool, ok = poolByAddress[poolAddress]; !ok {
				return nil, findroute.ErrNoIPool
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
