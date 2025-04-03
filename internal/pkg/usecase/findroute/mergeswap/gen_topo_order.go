package mergeswap

import (
	"context"
	"errors"

	"github.com/KyberNetwork/pathfinder-lib/pkg/entity"
	"github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/common"
	mapset "github.com/deckarep/golang-set/v2"
)

var ErrGraphHasCycle = errors.New("graph has cycle")

func generateTokenTopoOrder(ctx context.Context, params entity.FinderParams, route *common.ConstructRoute) ([]string, error) {
	// Kahn's algorithm: https://en.wikipedia.org/wiki/Topological_sorting
	tokenNoIncomingDegree := mapset.NewThreadUnsafeSet(params.TokenIn)
	inDegree := make(map[string]int)

	for _, path := range route.Paths {
		for _, token := range path.TokensOrder[1:] {
			inDegree[token]++
		}
	}

	topoOrder := []string{}

	for {
		if tokenNoIncomingDegree.Cardinality() == 0 {
			break
		}

		targetToken, _ := tokenNoIncomingDegree.Pop()

		topoOrder = append(topoOrder, targetToken)

		for _, path := range route.Paths {
			for idx, token := range path.TokensOrder[:len(path.TokensOrder)-1] {
				nextToken := path.TokensOrder[idx+1]
				if token == targetToken {
					inDegree[nextToken]--

					if inDegree[nextToken] == 0 {
						tokenNoIncomingDegree.Add(nextToken)
					}
				}
			}
		}
	}

	if len(topoOrder) != len(inDegree)+1 {
		return nil, ErrGraphHasCycle
	}

	return topoOrder, nil
}
