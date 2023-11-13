package common

import (
	"context"
	"math/big"
	"sort"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type nodeInfo struct {
	tokenAmount         poolpkg.TokenAmount
	poolAddressesOnPath []string
	tokensOnPath        []entity.Token
	totalGasAmount      int64
}

// GenKthBestPaths Find several best paths from tokenIn to tokenOut
// we represent graph node as pair (token, hops) because we want to handle negative cycles
// edges are now from (X, hop) to (Y, hop + 1) => make the graph a DAG => no cycle
//
// Perform BFS from (tokenIn,0) to find the best path to token out
// Because we are performing BFS and that only edges between (X, hop) -> (Y, hop+1) exist
// => The order of traversal looks like: (, 0) ... (, 0) (, 1) ... (, 1) ... (, hop-1), ... (,hop-1), (,hop)... (, hop)
//
// For each pair (token, hop), we maintain maxPathsToGenerate best paths
// => return the best maxPathsToGenerate paths of each length (from 1 -> maxHops) from tokenIn to tokenOut
//
// The maximum number of paths returned is maxPathsToGenerate * maxHops
func GenKthBestPaths(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolpkg.TokenAmount,
	tokenToPoolAddress map[string][]string,
	hopsToTokenOut map[string]uint32,
	maxHops, maxPathsToGenerate, maxPathsToReturn uint32,
) ([]*valueobject.Path, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "GenKthBestPaths")
	defer span.End()

	// Must be able to get info about tokenIn
	if _, ok := data.TokenByAddress[input.TokenInAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenIn
	}
	// Must be able to get info about tokenOut
	if _, ok := data.TokenByAddress[input.TokenOutAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenOut
	}

	if minHopFromTokenIn, ok := hopsToTokenOut[input.TokenInAddress]; !ok || minHopFromTokenIn > maxHops {
		return nil, nil
	}

	// Perform BFS with layers, layer ith contains paths of length i.
	// For each token in each layer, we store at most kth paths
	var (
		prevLayer = make(map[string][]*nodeInfo)
		paths     []*valueobject.Path
	)

	prevLayer[input.TokenInAddress] = []*nodeInfo{
		{
			tokenAmount:    tokenAmountIn,
			totalGasAmount: 0,
			tokensOnPath:   []entity.Token{data.TokenByAddress[input.TokenInAddress]},
		},
	}
	for currentHop := uint32(0); currentHop < maxHops; currentHop++ {

		nextLayer, err := genNextLayerOfPaths(input, data, tokenToPoolAddress, hopsToTokenOut, maxHops, currentHop, prevLayer)
		if err != nil {
			return nil, err
		}
		// fmt.Printf("This layer has %v paths\n", len(nextLayer[input.TokenOutAddress]))
		paths = append(paths, getKthPathAtTokenOut(input, data, tokenAmountIn, nextLayer[input.TokenOutAddress], maxPathsToReturn)...)

		nextLayer[input.TokenOutAddress] = nil

		nextLayer = getKthBestPathsForEachToken(nextLayer, maxPathsToGenerate, input.GasInclude)

		prevLayer = nextLayer

	}
	// fmt.Println()
	// fmt.Println(len(paths))
	return paths, nil
}

func genNextLayerOfPaths(
	input findroute.Input,
	data findroute.FinderData,
	tokenToPoolAddresses map[string][]string,
	hopsToTokenOut map[string]uint32,
	maxHops uint32,
	currentHop uint32,
	currentLayer map[string][]*nodeInfo,
) (map[string][]*nodeInfo, error) {
	nextLayer := make(map[string][]*nodeInfo)
	for fromToken, pathsToToken := range currentLayer {
		for _, fromNodeInfo := range pathsToToken {
			// get possible path of length currentHop + 1 by traveling one edge/ appending a pool
			nextNodeInfo, err := getNextLayerFromToken(input, data, tokenToPoolAddresses, hopsToTokenOut, maxHops, currentHop, fromToken, fromNodeInfo)
			if err != nil {
				return nil, err
			}
			for _, info := range nextNodeInfo {
				nextLayer[info.tokenAmount.Token] = append(nextLayer[info.tokenAmount.Token], info)
			}
		}
	}
	return nextLayer, nil
}

func getNextLayerFromToken(
	input findroute.Input,
	data findroute.FinderData,
	tokenToPoolAddresses map[string][]string,
	hopsToTokenOut map[string]uint32,
	maxHops uint32,
	currentHop uint32,
	fromTokenAddress string,
	fromNodeInfo *nodeInfo,
) ([]*nodeInfo, error) {
	usedTokens := sets.NewString()
	usedPools := sets.NewString()
	for _, tokenOnPath := range fromNodeInfo.tokensOnPath {
		usedTokens.Insert(tokenOnPath.Address)
	}
	for _, poolOnPath := range fromNodeInfo.poolAddressesOnPath {
		usedPools.Insert(poolOnPath)
	}

	var (
		nextNodeInfos []*nodeInfo
		toTokenInfo   entity.Token
		pool          poolpkg.IPoolSimulator

		remainingHopToTokenOut uint32
		ok                     bool
	)
	for _, poolAddress := range tokenToPoolAddresses[fromTokenAddress] {
		// If next pool addr == current pool addr -> skip because we have not update reserve balance on GenKBestPaths,
		// so the way which go two same pools on a path will give wrong result.
		if usedPools.Has(poolAddress) {
			continue
		}

		pool, ok = data.PoolBucket.GetPool(poolAddress)
		if !ok {
			return nil, findroute.ErrNoIPool
		}
		for _, toTokenAddress := range pool.CanSwapFrom(fromTokenAddress) {
			// must-have info for fromToken on path
			toTokenInfo, ok = data.TokenByAddress[toTokenAddress]
			if !ok {
				continue
			}
			// completely avoid cycles
			if usedTokens.Has(toTokenAddress) {
				continue
			}
			if remainingHopToTokenOut, ok = hopsToTokenOut[toTokenAddress]; !ok || currentHop+1+remainingHopToTokenOut > maxHops {
				continue
			}
			// it is ok for prices[tokenTo] to default to zero
			toTokenAmount, toTotalGasAmount, err := calcNewTokenAmountAndGas(pool, fromNodeInfo.tokenAmount, fromNodeInfo.totalGasAmount, toTokenAddress, data.PriceUSDByAddress[toTokenAddress], toTokenInfo.Decimals, input.GasPrice, input.GasTokenPriceUSD)
			if err != nil || toTokenAmount == nil || toTokenAmount.Amount.Int64() == 0 {
				logger.Errorf("cannot calculate amountOut, error:%v", err)
				continue
			}

			if pool.GetType() == constant.PoolTypes.KyberPMM {

				if data.PMMInventory.GetBalance(toTokenInfo.Address).Cmp(toTokenAmount.Amount) < 0 {
					continue
				}
			}

			// append pool and tokens to path
			nextNodeInfos = append(nextNodeInfos, &nodeInfo{
				tokenAmount:         *toTokenAmount,
				totalGasAmount:      toTotalGasAmount,
				poolAddressesOnPath: append(append([]string{}, fromNodeInfo.poolAddressesOnPath...), pool.GetAddress()),
				tokensOnPath:        append(append([]entity.Token{}, fromNodeInfo.tokensOnPath...), toTokenInfo),
			})
		}
	}
	return nextNodeInfos, nil
}

func getKthBestPathsForEachToken(
	nextLayer map[string][]*nodeInfo,
	maxPathsToGenerate uint32,
	gasInclude bool,
) map[string][]*nodeInfo {
	sortedNextLayer := make(map[string][]*nodeInfo)
	for token, pathsToToken := range nextLayer {
		sort.Slice(pathsToToken, func(i, j int) bool {
			return betterAmountOut(pathsToToken[i], pathsToToken[j], gasInclude)
		})
		// only keep k best paths
		if uint32(len(pathsToToken)) > maxPathsToGenerate {
			pathsToToken = pathsToToken[:maxPathsToGenerate]
		}
		sortedNextLayer[token] = pathsToToken
	}
	return sortedNextLayer
}

func getKthPathAtTokenOut(
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolpkg.TokenAmount,
	nodeInfoAtTokenOut []*nodeInfo,
	maxPathsToReturn uint32,
) (paths []*valueobject.Path) {

	sort.Slice(nodeInfoAtTokenOut, func(i, j int) bool {
		return betterAmountOut(nodeInfoAtTokenOut[i], nodeInfoAtTokenOut[j], input.GasInclude)
	})
	if uint32(len(nodeInfoAtTokenOut)) > maxPathsToReturn {
		nodeInfoAtTokenOut = nodeInfoAtTokenOut[:maxPathsToReturn]
	}

	for kthPath, pathInfo := range nodeInfoAtTokenOut {
		tokenOut := pathInfo.tokensOnPath[len(pathInfo.tokensOnPath)-1].Address
		path, err := valueobject.NewPath(data.PoolBucket, pathInfo.poolAddressesOnPath, pathInfo.tokensOnPath, tokenAmountIn, tokenOut,
			data.PriceUSDByAddress[input.TokenOutAddress], data.TokenByAddress[tokenOut].Decimals,
			valueobject.GasOption{GasFeeInclude: input.GasInclude, Price: input.GasPrice, TokenPrice: input.GasTokenPriceUSD}, data.PMMInventory,
		)
		if err != nil {
			logger.WithFields(logger.Fields{"error": err}).
				Errorf("cannot generate %v_th path (hop = %v) from token %v to token %v %v", kthPath, len(pathInfo.poolAddressesOnPath), input.TokenInAddress, tokenOut, input.AmountIn)
		} else {
			paths = append(paths, path)
		}
	}
	return paths
}

func betterAmountOut(nodeA, nodeB *nodeInfo, gasFeeInclude bool) bool {
	// If we consider gas fee, prioritize node with more AmountUsd
	// If amountUsd is the same, compare amountOut regardless of gasFeeInclude
	if gasFeeInclude && !utils.Float64AlmostEqual(nodeA.tokenAmount.AmountUsd, nodeB.tokenAmount.AmountUsd) {
		return nodeA.tokenAmount.AmountUsd > nodeB.tokenAmount.AmountUsd
	}
	// Otherwise, prioritize node with more token Amount
	cmp := nodeA.tokenAmount.Amount.Cmp(nodeB.tokenAmount.Amount)
	if cmp != 0 {
		return cmp > 0
	}
	// if that amount is equal, we compare nodeId in alphabetical order
	return utils.CompareStringSlices(nodeA.poolAddressesOnPath, nodeB.poolAddressesOnPath) == -1
}

// return newTokenAmount, newTotalGasAmount, error
func calcNewTokenAmountAndGas(
	pool poolpkg.IPoolSimulator,
	fromAmountIn poolpkg.TokenAmount, fromTotalGasAmount int64,
	tokenOut string, tokenOutPrice float64, tokenOutDecimal uint8,
	gasPrice *big.Float, gasTokenPrice float64,
) (*poolpkg.TokenAmount, int64, error) {
	calcAmountOutResult, err := poolpkg.CalcAmountOut(pool, fromAmountIn, tokenOut)
	if err != nil {
		return nil, 0, err
	}
	newTotalGasAmount := calcAmountOutResult.Gas + fromTotalGasAmount
	calcAmountOutResult.TokenAmountOut.AmountUsd =
		utils.CalcTokenAmountUsd(calcAmountOutResult.TokenAmountOut.Amount, tokenOutDecimal, tokenOutPrice) -
			utils.CalcGasUsd(gasPrice, newTotalGasAmount, gasTokenPrice)
	return calcAmountOutResult.TokenAmountOut, newTotalGasAmount, nil
}
