package common

import (
	"context"
	"math/big"
	"sort"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type nodeInfo struct {
	tokenAmount         valueobject.TokenAmount
	poolAddressesOnPath []string
	tokensOnPath        []*entity.Token
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
// The paths returns here are taken from a memPool. Be sure to call valueobject.ReturnPaths once its scope is finished
func GenKthBestPaths(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn valueobject.TokenAmount,
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
			tokensOnPath:   []*entity.Token{data.TokenByAddress[input.TokenInAddress]},
		},
	}
	for currentHop := uint32(0); currentHop < maxHops; currentHop++ {

		nextLayer, err := genNextLayerOfPaths(ctx, input, data, data.TokenToPoolAddress, hopsToTokenOut, maxHops, currentHop, prevLayer)
		if err != nil {
			return nil, err
		}
		// fmt.Printf("This layer has %v paths\n", len(nextLayer[input.TokenOutAddress]))
		paths = append(paths, getKthPathAtTokenOut(ctx, input, data, tokenAmountIn, nextLayer[input.TokenOutAddress], maxPathsToReturn)...)

		nextLayer[input.TokenOutAddress] = nil

		nextLayer = getKthBestPathsForEachToken(nextLayer, maxPathsToGenerate, input.GasInclude)

		prevLayer = nextLayer

	}
	// fmt.Println()
	// fmt.Println(len(paths))
	return paths, nil
}

func genNextLayerOfPaths(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenToPoolAddresses map[string]*types.AddressList,
	hopsToTokenOut map[string]uint32,
	maxHops uint32,
	currentHop uint32,
	currentLayer map[string][]*nodeInfo,
) (map[string][]*nodeInfo, error) {
	var (
		intermediateResults sync.Map // map[int][]*nodeInfo
		wg                  errgroup.Group
		numItr              int
	)

	for fromToken, pathsToToken := range currentLayer {
		for _, fromNodeInfo := range pathsToToken {
			itr, _fromToken, _fromNodeInfo := numItr, fromToken, fromNodeInfo

			wg.Go(func() error {
				// get possible path of length currentHop + 1 by traveling one edge/ appending a pool
				nextNodeInfo, err := getNextLayerFromToken(ctx, input, data, tokenToPoolAddresses, hopsToTokenOut, maxHops, currentHop, _fromToken, _fromNodeInfo)
				if err != nil {
					return err
				}
				intermediateResults.Store(itr, nextNodeInfo)
				return nil
			})

			numItr++
		}
	}

	if err := wg.Wait(); err != nil {
		return nil, err
	}

	nextLayer := make(map[string][]*nodeInfo)
	for itr := 0; itr < numItr; itr++ {
		_nextNodeInfo, _ := intermediateResults.Load(itr)
		nextNodeInfo := _nextNodeInfo.([]*nodeInfo)
		for _, info := range nextNodeInfo {
			nextLayer[info.tokenAmount.Token] = append(nextLayer[info.tokenAmount.Token], info)
		}
	}
	return nextLayer, nil
}

func getNextLayerFromToken(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenToPoolAddresses map[string]*types.AddressList,
	hopsToTokenOut map[string]uint32,
	maxHops uint32,
	currentHop uint32,
	fromTokenAddress string,
	fromNodeInfo *nodeInfo,
) ([]*nodeInfo, error) {
	type IntermediateParam struct {
		pool           poolpkg.IPoolSimulator
		toTokenAddress string
		toTokenInfo    *entity.Token
	}
	type IntermediateResult struct {
		toTokenAmount    *valueobject.TokenAmount
		toTotalGasAmount int64
	}
	var (
		intermediateParams  []IntermediateParam
		intermediateResults sync.Map // map[int]IntermediateResult
		wg                  errgroup.Group
		numItr              int
	)

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
		toTokenInfo   *entity.Token
		pool          poolpkg.IPoolSimulator

		remainingHopToTokenOut uint32
		ok                     bool
	)
	// if there is no adjacent from this token
	if tokenToPoolAddresses[fromTokenAddress] == nil {
		return nil, nil
	}
	for i := 0; i < tokenToPoolAddresses[fromTokenAddress].TrueLen; i++ {
		poolAddress := tokenToPoolAddresses[fromTokenAddress].Arr[i]
		// If next pool addr == current pool addr -> skip because we have not update reserve balance on GenKBestPaths,
		// so the way which go two same pools on a path will give wrong result.
		if usedPools.Has(poolAddress) {
			continue
		}

		pool, ok = data.PoolBucket.GetPool(poolAddress)
		if !ok {
			// the adjacent list might include pool which is not in this particular bucket
			continue
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

			itr, _pool, _toTokenInfo := numItr, pool, toTokenInfo
			wg.Go(func() error {
				// it is ok for prices[tokenTo] to default to zero
				toTokenAmount, toTotalGasAmount, err := CalcNewTokenAmountAndGas(_pool, fromNodeInfo.tokenAmount, fromNodeInfo.totalGasAmount, _toTokenInfo, data, input)
				if err != nil || toTokenAmount == nil || toTokenAmount.Amount.Sign() == 0 {
					logger.Debugf(ctx, "cannot calculate amountOut, error:%v", err)
					return nil
				}

				intermediateResults.Store(itr, IntermediateResult{toTokenAmount, toTotalGasAmount})
				return nil
			})

			intermediateParams = append(intermediateParams, IntermediateParam{pool, toTokenAddress, toTokenInfo})

			numItr++
		}
	}

	wg.Wait()

	for itr, param := range intermediateParams {
		_result, ok := intermediateResults.Load(itr)
		if !ok {
			continue
		}
		result := _result.(IntermediateResult)

		var (
			pool             = param.pool
			toTokenInfo      = param.toTokenInfo
			toTokenAmount    = result.toTokenAmount
			toTotalGasAmount = result.toTotalGasAmount
		)

		// append pool and tokens to path
		nextNodeInfos = append(nextNodeInfos, &nodeInfo{
			tokenAmount:         *toTokenAmount,
			totalGasAmount:      toTotalGasAmount,
			poolAddressesOnPath: append(append([]string{}, fromNodeInfo.poolAddressesOnPath...), pool.GetAddress()),
			tokensOnPath:        append(append([]*entity.Token{}, fromNodeInfo.tokensOnPath...), toTokenInfo),
		})
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
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn valueobject.TokenAmount,
	nodeInfoAtTokenOut []*nodeInfo,
	maxPathsToReturn uint32,
) (paths []*valueobject.Path) {

	sort.Slice(nodeInfoAtTokenOut, func(i, j int) bool {
		return betterAmountOut(nodeInfoAtTokenOut[i], nodeInfoAtTokenOut[j], input.GasInclude)
	})
	if uint32(len(nodeInfoAtTokenOut)) > maxPathsToReturn {
		nodeInfoAtTokenOut = nodeInfoAtTokenOut[:maxPathsToReturn]
	}

	var (
		intermediateResults = make([]*valueobject.Path, len(nodeInfoAtTokenOut))
		wg                  errgroup.Group
	)

	for kthPath, pathInfo := range nodeInfoAtTokenOut {
		_kthPath, _pathInfo := kthPath, pathInfo
		wg.Go(func() error {
			tokenOut := _pathInfo.tokensOnPath[len(_pathInfo.tokensOnPath)-1].Address
			path, err := valueobject.NewPath(data.PoolBucket, _pathInfo.poolAddressesOnPath, _pathInfo.tokensOnPath, tokenAmountIn, tokenOut,
				data.PriceUSDByAddress[input.TokenOutAddress], data.TokenNativeBuyPrice(input.TokenOutAddress), data.TokenByAddress[tokenOut].Decimals,
				valueobject.GasOption{GasFeeInclude: input.GasInclude, Price: input.GasPrice, TokenPrice: input.GasTokenPriceUSD}, data.SwapLimits,
			)
			if err != nil {
				logger.WithFields(ctx, logger.Fields{"error": err}).
					Errorf("cannot generate %v_th path (hop = %v) from token %v to token %v %v", _kthPath, len(_pathInfo.poolAddressesOnPath), input.TokenInAddress, tokenOut, input.AmountIn)
			} else {
				intermediateResults[_kthPath] = path
			}
			return nil
		})
	}

	wg.Wait()

	for kthPath := range nodeInfoAtTokenOut {
		if path := intermediateResults[kthPath]; path != nil {
			paths = append(paths, path)
		}
	}
	return paths
}

func betterAmountOut(nodeA, nodeB *nodeInfo, gasFeeInclude bool) bool {
	// compare amountUsd and amount first
	if cmp := nodeA.tokenAmount.Compare(&nodeB.tokenAmount, gasFeeInclude); cmp != 0 {
		return cmp > 0
	}

	// if that amount is equal, we compare nodeId in alphabetical order
	return utils.CompareStringSlices(nodeA.poolAddressesOnPath, nodeB.poolAddressesOnPath) == -1
}

// return newTokenAmount, newTotalGasAmount, error
func calcNewTokenAmountAndGasInUSD(
	pool poolpkg.IPoolSimulator,
	fromAmountIn valueobject.TokenAmount, fromTotalGasAmount int64,
	tokenOut string, tokenOutPrice float64, tokenOutDecimal uint8,
	gasPrice *big.Float, gasTokenPrice float64,
	swapLimit poolpkg.SwapLimit,
) (*valueobject.TokenAmount, int64, error) {
	calcAmountOutResult, err := poolpkg.CalcAmountOut(pool, poolpkg.TokenAmount{
		Token:  fromAmountIn.Token,
		Amount: fromAmountIn.Amount,
	}, tokenOut, swapLimit)
	if err != nil {
		return nil, 0, err
	}
	newTotalGasAmount := calcAmountOutResult.Gas + fromTotalGasAmount

	calcAmountOutResult.TokenAmountOut.AmountUsd =
		utils.CalcTokenAmountUsd(calcAmountOutResult.TokenAmountOut.Amount, tokenOutDecimal, tokenOutPrice) -
			utils.CalcGasUsd(gasPrice, newTotalGasAmount, gasTokenPrice)
	return valueobject.FromDexLibAmount(calcAmountOutResult.TokenAmountOut), newTotalGasAmount, nil
}

func calcNewTokenAmountAndGasInNative(
	pool poolpkg.IPoolSimulator,
	fromAmountIn valueobject.TokenAmount, fromTotalGasAmount int64,
	tokenOut string, tokenOutPriceNative *big.Float,
	gasPrice *big.Float,
	swapLimit poolpkg.SwapLimit,
) (*valueobject.TokenAmount, int64, error) {
	calcAmountOutResult, err := poolpkg.CalcAmountOut(pool, poolpkg.TokenAmount{
		Token:  fromAmountIn.Token,
		Amount: fromAmountIn.Amount,
	}, tokenOut, swapLimit)
	if err != nil {
		return nil, 0, err
	}
	newTotalGasAmount := calcAmountOutResult.Gas + fromTotalGasAmount

	tokenAmountOut := &valueobject.TokenAmount{
		Token:          tokenOut,
		Amount:         calcAmountOutResult.TokenAmountOut.Amount,
		AmountAfterGas: big.NewInt(0),
	}

	// if we don't have price for tokenOut, then let's just keep AmountAfterGas to zero, `betterAmountOut` will fallback to comparing `Amount`
	if tokenOutPriceNative == nil {
		return tokenAmountOut, newTotalGasAmount, nil
	}

	// gas amount doesn't have decimal, so just multiply with gasPrice directly
	gasAmountInNative, _ := new(big.Float).Mul(gasPrice, new(big.Float).SetInt64(newTotalGasAmount)).Int(&big.Int{})
	// tokenOutPriceNative should have been divided by token decimal already, so here just multiply
	amountOutInNative, _ := new(big.Float).Mul(tokenOutPriceNative, new(big.Float).SetInt(calcAmountOutResult.TokenAmountOut.Amount)).Int(&big.Int{})
	tokenAmountOut.AmountAfterGas.Sub(amountOutInNative, gasAmountInNative)

	return tokenAmountOut, newTotalGasAmount, nil
}

func CalcNewTokenAmountAndGas(
	pool poolpkg.IPoolSimulator,
	fromAmountIn valueobject.TokenAmount, fromTotalGasAmount int64,
	tokenOut *entity.Token,
	data findroute.FinderData,
	input findroute.Input,
) (*valueobject.TokenAmount, int64, error) {
	if data.PriceNativeByAddress != nil {
		// this will be called for tokenOut and intermediate tokens, so should use buy price
		return calcNewTokenAmountAndGasInNative(
			pool, fromAmountIn, fromTotalGasAmount,
			tokenOut.Address, data.TokenNativeBuyPrice(tokenOut.Address),
			input.GasPrice, data.SwapLimits[pool.GetType()])
	}
	return calcNewTokenAmountAndGasInUSD(
		pool, fromAmountIn, fromTotalGasAmount,
		tokenOut.Address, data.PriceUSDByAddress[tokenOut.Address],
		tokenOut.Decimals, input.GasPrice, input.GasTokenPriceUSD, data.SwapLimits[pool.GetType()])
}
