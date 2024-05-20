package common

import (
	"context"
	"sort"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/sets"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// GenKthBestPathsV2 Run similar algorithm as GenKthBestPaths
// But return best paths for all tokens available in data.TokenByAddress instead of just tokenOut
// TODO: Note that input.TokenOutAddress does not matter here because we generate for all tokens out
func GenKthBestPathsV2(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn valueobject.TokenAmount,
	maxHops, maxPathsToGenerate, maxPathsToReturn uint32,
) (map[string][]*valueobject.Path, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "GenKthBestPathsV2")
	defer span.End()

	// Must be able to get info about tokenIn
	if _, ok := data.TokenByAddress[input.TokenInAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenIn
	}

	// Perform BFS with layers, layer ith contains paths of length i.
	// For each token in each layer, we store at most kth paths
	var (
		prevLayer              = make(map[string][]*nodeInfo)
		pathsByTokenOutAddress = make(map[string][]*valueobject.Path)
	)

	prevLayer[input.TokenInAddress] = []*nodeInfo{
		{
			tokenAmount:    tokenAmountIn,
			totalGasAmount: 0,
			tokensOnPath:   []*entity.Token{data.TokenByAddress[input.TokenInAddress]},
		},
	}
	for currentHop := uint32(0); currentHop < maxHops; currentHop++ {

		nextLayer, err := genNextLayerOfPathsV2(ctx, input, data, prevLayer)
		if err != nil {
			return nil, err
		}

		for tokenOutAddress := range nextLayer {
			pathsByTokenOutAddress[tokenOutAddress] = append(
				pathsByTokenOutAddress[tokenOutAddress],
				getKthPathAtTokenOutV2(input, tokenAmountIn, nextLayer[tokenOutAddress], maxPathsToReturn)...,
			)
		}

		// keep 'k' best paths for each token only to reduce computation
		nextLayer = getKthBestPathsForEachToken(nextLayer, maxPathsToGenerate, input.GasInclude)

		prevLayer = nextLayer

	}
	return pathsByTokenOutAddress, nil
}

func genNextLayerOfPathsV2(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
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
				nextNodeInfo, err := getNextLayerFromTokenV2(ctx, input, data, _fromToken, _fromNodeInfo)
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

func getNextLayerFromTokenV2(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	fromTokenAddress string,
	fromNodeInfo *nodeInfo,
) ([]*nodeInfo, error) {
	type IntermediateParam struct {
		poolAddress    string
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

		ok bool
	)

	if data.TokenToPoolAddress[fromTokenAddress] == nil {
		return nil, findroute.ErrNoPoolsFromToken
	}
	for i := 0; i < data.TokenToPoolAddress[fromTokenAddress].TrueLen; i++ {
		poolAddress := data.TokenToPoolAddress[fromTokenAddress].Arr[i]
		// If next pool addr == current pool addr -> skip because we have not update reserve balance on GenKBestPaths,
		// so the way which go two same pools on a path will give wrong result.
		if usedPools.Has(poolAddress) {
			continue
		}

		pool, ok = data.PoolBucket.GetPool(poolAddress)
		if !ok {
			return nil, findroute.ErrNoIPool
		}
		for _, toTokenAddress := range pool.CanSwapTo(fromTokenAddress) {
			// must-have info for fromToken on path
			toTokenInfo, ok = data.TokenByAddress[toTokenAddress]
			if !ok {
				continue
			}
			// completely avoid cycles
			if usedTokens.Has(toTokenAddress) {
				continue
			}

			itr, _pool, _toTokenAddress, _toTokenInfo := numItr, pool, toTokenAddress, toTokenInfo
			wg.Go(func() error {
				// it is ok for prices[tokenTo] to default to zero
				toTokenAmount, toTotalGasAmount, err := calcNewTokenAmountAndGasInUSD(ctx, _pool, fromNodeInfo.tokenAmount, fromNodeInfo.totalGasAmount, _toTokenAddress, data.PriceUSDByAddress[_toTokenAddress], _toTokenInfo.Decimals, input.GasPrice, input.GasTokenPriceUSD, data.SwapLimits[_pool.GetType()])
				if err != nil || toTokenAmount == nil || toTokenAmount.Amount.Sign() == 0 {
					return nil
				}

				intermediateResults.Store(itr, IntermediateResult{toTokenAmount, toTotalGasAmount})
				return nil
			})

			intermediateParams = append(intermediateParams, IntermediateParam{poolAddress, toTokenAddress, toTokenInfo})

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
			poolAddress      = param.poolAddress
			toTokenInfo      = param.toTokenInfo
			toTokenAmount    = result.toTokenAmount
			toTotalGasAmount = result.toTotalGasAmount
		)

		// append pool and tokens to path
		nextNodeInfos = append(nextNodeInfos, &nodeInfo{
			tokenAmount:         *toTokenAmount,
			totalGasAmount:      toTotalGasAmount,
			poolAddressesOnPath: append(append([]string{}, fromNodeInfo.poolAddressesOnPath...), poolAddress),
			tokensOnPath:        append(append([]*entity.Token{}, fromNodeInfo.tokensOnPath...), toTokenInfo),
		})
	}

	return nextNodeInfos, nil
}

func getKthPathAtTokenOutV2(
	input findroute.Input,
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

	for _, pathInfo := range nodeInfoAtTokenOut {
		paths = append(paths, &valueobject.Path{
			Input:         tokenAmountIn,
			Output:        pathInfo.tokenAmount,
			TotalGas:      pathInfo.totalGasAmount,
			PoolAddresses: pathInfo.poolAddressesOnPath,
			Tokens:        pathInfo.tokensOnPath,
		})
	}
	return paths
}
