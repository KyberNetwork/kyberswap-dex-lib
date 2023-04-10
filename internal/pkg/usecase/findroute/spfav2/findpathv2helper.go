package spfav2

import (
	"container/heap"
	"context"
	"math/big"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/core"
	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
)

type findPathV2Helper struct {
	tokenAmountIn               poolPkg.TokenAmount
	poolAddressToLastUsedSplit  map[string]int
	pathIdToLastCalculatedSplit []int
	addedPathIds                sets.Int
	pq                          *priorityQueue

	maxPathsInRoute int
	currentSplit    int
}

func NewFindPathV2Helper(numberOfPaths, maxPathsInRoute int, tokenAmountIn poolPkg.TokenAmount, cmpFunc func(a, b int) bool) *findPathV2Helper {
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
	paths []*core.Path,
	newAmountIn poolPkg.TokenAmount,
) *core.Path {
	span, _ := tracer.StartSpanFromContext(ctx, "findPathV2Helper.bestPathExactInV2")
	defer span.Finish()

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
			for _, pool := range paths[pathId].Pools {
				h.poolAddressToLastUsedSplit[pool.GetAddress()] = h.currentSplit
			}
			h.addedPathIds.Insert(pathId)
			h.currentSplit += 1

			var bestPath = paths[pathId]
			// if amount used to generate is different from splitAmountIn, this is possible when amountInUsd is small
			if h.tokenAmountIn.CompareTo(&newAmountIn) != 0 {
				bestPath = newPath(input, data, bestPath.Pools, bestPath.Tokens, newAmountIn, h.addedPathIds.Has(pathId))
			}

			return bestPath
		}

		// we recalculate the path here
		paths[pathId] = newPath(input, data, paths[pathId].Pools, paths[pathId].Tokens, h.tokenAmountIn, h.addedPathIds.Has(pathId))
		h.pathIdToLastCalculatedSplit[pathId] = h.currentSplit
		heap.Pop(h.pq)
		if paths[pathId] != nil {
			heap.Push(h.pq, pathId)
		}
	}
	return nil
}

func (h *findPathV2Helper) needToRecalculatePath(pathId int, path *core.Path) bool {
	for _, pool := range path.Pools {
		if lastUsed, ok := h.poolAddressToLastUsedSplit[pool.GetAddress()]; ok && lastUsed >= h.pathIdToLastCalculatedSplit[pathId] {
			return true
		}
	}
	return false
}

func newPath(
	input findroute.Input,
	data findroute.FinderData,
	pools []poolPkg.IPool,
	tokens []entity.Token,
	tokenAmountIn poolPkg.TokenAmount,
	disregardGasFee bool,
) *core.Path {
	// if the path is added, we set disregardGasFee = true
	var gasOption core.GasOption
	if disregardGasFee {
		gasOption = core.GasOption{GasFeeInclude: false, Price: big.NewFloat(0), TokenPrice: 0}
	} else {
		gasOption = core.GasOption{GasFeeInclude: input.GasInclude, Price: input.GasPrice, TokenPrice: input.GasTokenPriceUSD}
	}
	path, err := core.NewPath(pools, tokens, tokenAmountIn, input.TokenOutAddress,
		data.PriceUSDByAddress[input.TokenOutAddress], data.TokenByAddress[input.TokenOutAddress].Decimals, gasOption,
	)
	if err != nil {
		return nil
	}
	return path
}
