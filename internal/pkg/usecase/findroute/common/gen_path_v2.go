package common

import (
	"context"
	"sort"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"k8s.io/apimachinery/pkg/util/sets"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// GenKthBestPathsV2 Run similar algorithm as GenKthBestPaths
// But return best paths for all tokens available in data.TokenByAddress instead of just tokenOut
// TODO: Note that input.TokenOutAddress does not matter here because we generate for all tokens out
func GenKthBestPathsV2(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolpkg.TokenAmount,
	maxHops, maxPathsToGenerate, maxPathsToReturn uint32,
) (map[string][]*valueobject.Path, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "GenKthBestPathsV2")
	defer span.Finish()

	// Must be able to get info about tokenIn
	if _, ok := data.TokenByAddress[input.TokenInAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenIn
	}

	// Optimize graph traversal by using adjacent list
	tokenToPoolAddress := make(map[string][]string)
	for poolAddress, pool := range data.PoolBucket.PerRequestPoolsByAddress {
		for _, fromToken := range pool.GetTokens() {
			if _, ok := data.TokenByAddress[fromToken]; ok {
				tokenToPoolAddress[fromToken] = append(tokenToPoolAddress[fromToken], poolAddress)
			}
		}
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
			tokensOnPath:   []entity.Token{data.TokenByAddress[input.TokenInAddress]},
		},
	}
	for currentHop := uint32(0); currentHop < maxHops; currentHop++ {

		nextLayer, err := genNextLayerOfPathsV2(input, data, tokenToPoolAddress, prevLayer)
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
	input findroute.Input,
	data findroute.FinderData,
	tokenToPoolAddresses map[string][]string,
	currentLayer map[string][]*nodeInfo,
) (map[string][]*nodeInfo, error) {
	nextLayer := make(map[string][]*nodeInfo)
	for fromToken, pathsToToken := range currentLayer {
		for _, fromNodeInfo := range pathsToToken {
			// get possible path of length currentHop + 1 by traveling one edge/ appending a pool
			nextNodeInfo, err := getNextLayerFromTokenV2(input, data, tokenToPoolAddresses, fromToken, fromNodeInfo)
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

func getNextLayerFromTokenV2(
	input findroute.Input,
	data findroute.FinderData,
	tokenToPoolAddresses map[string][]string,
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

		ok bool
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
			// it is ok for prices[tokenTo] to default to zero
			toTokenAmount, toTotalGasAmount, err := calcNewTokenAmountAndGas(pool, fromNodeInfo.tokenAmount, fromNodeInfo.totalGasAmount, toTokenAddress, data.PriceUSDByAddress[toTokenAddress], toTokenInfo.Decimals, input.GasPrice, input.GasTokenPriceUSD)
			if err != nil || toTokenAmount == nil || toTokenAmount.Amount.Int64() == 0 {
				continue
			}
			// append pool and tokens to path
			nextNodeInfos = append(nextNodeInfos, &nodeInfo{
				tokenAmount:         *toTokenAmount,
				totalGasAmount:      toTotalGasAmount,
				poolAddressesOnPath: append(append([]string{}, fromNodeInfo.poolAddressesOnPath...), poolAddress),
				tokensOnPath:        append(append([]entity.Token{}, fromNodeInfo.tokensOnPath...), toTokenInfo),
			})
		}
	}
	return nextNodeInfos, nil
}

func getKthPathAtTokenOutV2(
	input findroute.Input,
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
