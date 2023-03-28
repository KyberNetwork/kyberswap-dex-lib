package core

import (
	"github.com/oleiade/lane"

	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

// minHopsToToken performs BFS and returns minimum swap required (min hops)
// to swap token with other tokens which appear in both poolByAddress and tokenByAddress
func minHopsToToken(
	poolByAddress map[string]poolPkg.IPool,
	tokenByAddress map[string]entity.Token,
	token string,
) map[string]uint32 {
	var (
		minHopsByToken = make(map[string]uint32)
		tokenQueue     = lane.NewQueue()
	)

	if _, ok := tokenByAddress[token]; !ok {
		return minHopsByToken
	}

	minHopsByToken[token] = 0
	tokenQueue.Enqueue(token)

	for !tokenQueue.Empty() {
		dequeuedToken := tokenQueue.Dequeue().(string)

		for _, pool := range poolByAddress {
			/*
				TODO: we should refactor this code later
				everytime we call GetTokenIndex,it loops through all pool tokens
				we should have a map poolAddressesByTokenAddress map[string][]string instead.
			*/
			if pool.GetTokenIndex(dequeuedToken) < 0 {
				continue
			}

			swappableTokens := pool.CanSwapTo(dequeuedToken)

			for _, swappableToken := range swappableTokens {
				if _, ok := tokenByAddress[swappableToken]; !ok {
					continue
				}

				if _, ok := minHopsByToken[swappableToken]; ok {
					continue
				}

				minHopsByToken[swappableToken] = minHopsByToken[dequeuedToken] + 1
				tokenQueue.Enqueue(swappableToken)
			}
		}
	}

	return minHopsByToken
}
