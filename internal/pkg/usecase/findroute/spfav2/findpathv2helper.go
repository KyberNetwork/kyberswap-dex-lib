package spfav2

import (
	"container/heap"
	"context"
	"math/big"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type findPathV2Helper struct {
	tokenAmountIn               poolpkg.TokenAmount
	poolAddressToLastUsedSplit  map[string]int
	pathIdToLastCalculatedSplit []int
	addedPathIds                sets.Int
	pq                          *priorityQueue

	maxPathsInRoute int
	currentSplit    int
}

func NewFindPathV2Helper(numberOfPaths, maxPathsInRoute int, tokenAmountIn poolpkg.TokenAmount, cmpFunc func(a, b int) bool) *findPathV2Helper {
	return &findPathV2Helper{
		tokenAmountIn,
		make(map[string]int),
		make([]int, numberOfPaths),
		sets.NewInt(),
		NewPriorityQueue(numberOfPaths, cmpFunc),
		maxPathsInRoute,
		0,
	}
}

func (h *findPathV2Helper) bestPathExactInV2(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	paths []*valueobject.Path,
	newAmountIn poolpkg.TokenAmount,
) *valueobject.Path {
	span, _ := tracer.StartSpanFromContext(ctx, "spfav2Finder.bestPathExactInV2")
	defer span.End()

	for h.pq.Len() > 0 {
		pathId := h.pq.Top().(int)
		// if we only want to use added path, and this path was not added. skip this path
		if h.addedPathIds.Len() == h.maxPathsInRoute && !h.addedPathIds.Has(pathId) {
			paths[pathId] = nil
			heap.Pop(h.pq)
			continue
		}

		// if no pool of this path is updated after the last time we calculate this path, we can reuse this path
		// otherwise, we recalculate the path
		if !h.needToRecalculatePath(pathId, paths[pathId]) {
			for _, poolAddress := range paths[pathId].PoolAddresses {
				h.poolAddressToLastUsedSplit[poolAddress] = h.currentSplit
			}
			h.addedPathIds.Insert(pathId)
			h.currentSplit += 1

			var bestPath = paths[pathId]
			// if amount used to generate is different from splitAmountIn, this is possible when amountInUsd is small
			if h.tokenAmountIn.CompareTo(&newAmountIn) != 0 {
				bestPath = newPath(input, data, bestPath.PoolAddresses, bestPath.Tokens, newAmountIn, h.addedPathIds.Has(pathId))
			}

			return bestPath
		}

		// we recalculate the path here
		paths[pathId] = newPath(input, data, paths[pathId].PoolAddresses, paths[pathId].Tokens, h.tokenAmountIn, h.addedPathIds.Has(pathId))
		h.pathIdToLastCalculatedSplit[pathId] = h.currentSplit
		heap.Pop(h.pq)
		if paths[pathId] != nil {
			heap.Push(h.pq, pathId)
		}
	}
	return nil
}

func (h *findPathV2Helper) needToRecalculatePath(pathId int, path *valueobject.Path) bool {
	for _, poolAddress := range path.PoolAddresses {
		if lastUsed, ok := h.poolAddressToLastUsedSplit[poolAddress]; ok && lastUsed >= h.pathIdToLastCalculatedSplit[pathId] {
			return true
		}
	}
	return false
}

func newPath(
	input findroute.Input,
	data findroute.FinderData,
	poolAddresses []string,
	tokens []entity.Token,
	tokenAmountIn poolpkg.TokenAmount,
	disregardGasFee bool,
) *valueobject.Path {
	// if the path is added, we set disregardGasFee = true
	var gasOption valueobject.GasOption
	if disregardGasFee {
		gasOption = valueobject.GasOption{GasFeeInclude: false, Price: big.NewFloat(0), TokenPrice: 0}
	} else {
		gasOption = valueobject.GasOption{GasFeeInclude: input.GasInclude, Price: input.GasPrice, TokenPrice: input.GasTokenPriceUSD}
	}
	path, err := valueobject.NewPath(data.PoolBucket, poolAddresses, tokens, tokenAmountIn, input.TokenOutAddress,
		data.PriceUSDByAddress[input.TokenOutAddress], data.TokenByAddress[input.TokenOutAddress].Decimals, gasOption, data.PMMInventory,
	)
	if err != nil {
		return nil
	}
	return path
}
