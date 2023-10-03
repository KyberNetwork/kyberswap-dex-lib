package generatepath

import (
	"context"
	"fmt"
	"math/big"
	"runtime"
	"strings"
	"sync"
	"time"

	aevmcommon "github.com/KyberNetwork/aevm/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	gEthCommon "github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/common"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type useCase struct {
	poolManager        IPoolManager
	tokenRepository    ITokenRepository
	priceRepository    IPriceRepository
	poolRankRepository getroute.IPoolRankRepository
	gasRepository      IGasRepository
	bestPathRepository IBestPathRepository

	config     Config
	sourceHash uint64
	mu         sync.RWMutex
}

func NewUseCase(
	poolManager IPoolManager,
	tokenRepository ITokenRepository,
	priceRepository IPriceRepository,
	poolRankRepository getroute.IPoolRankRepository,
	gasRepository IGasRepository,
	bestPathRepository IBestPathRepository,
	config Config,
) *useCase {
	return &useCase{
		poolManager:        poolManager,
		tokenRepository:    tokenRepository,
		priceRepository:    priceRepository,
		poolRankRepository: poolRankRepository,
		gasRepository:      gasRepository,
		bestPathRepository: bestPathRepository,
		config:             config,
		sourceHash:         valueobject.HashSources(config.AvailableSources),
	}
}

func (uc *useCase) ApplyConfig(config Config) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	differentSources := false
	if len(config.AvailableSources) != len(uc.config.AvailableSources) {
		differentSources = true
	}
	for i := range config.AvailableSources {
		if config.AvailableSources[i] != uc.config.AvailableSources[i] {
			differentSources = true
		}
	}
	// if we have a new sources, rehash it.
	if differentSources {
		uc.sourceHash = valueobject.HashSources(config.AvailableSources)
	}

	uc.config = config
}

func (uc *useCase) Handle(ctx context.Context) {
	if err := uc.poolManager.Reload(); err != nil {
		logger.Errorf("could not reload Pools", err)
		return
	}

	var numTasks int

	// get token and amounts generated from k-mean cluster algo
	tokensToMaintain, timestamp, err := uc.bestPathRepository.GetPregenTokenAmounts(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"err": err,
		}).Error("failed to get pregen kmeans data")
		return
	} else {
		logger.Infof("Using kmeans-data, sourceHash %v", uc.sourceHash)
		// use 7d duration log data after processing instead of default pregen config data.
	}

	deadline := time.Unix(timestamp, 0).Add(uc.config.ConfigGeneratorDataTtl)
	// if kMeansData is stale, skip
	if deadline.Before(time.Now()) {
		logger.WithFields(logger.Fields{}).Error("kMeans data is too stale")
		return
	}

	for _, amounts := range tokensToMaintain {
		numTasks += len(amounts.Amounts)
	}

	var (
		results            = make(chan *genBestPathsResult, numTasks)
		wg                 sync.WaitGroup
		ctxTimeout, cancel = context.WithTimeout(ctx, uc.config.PathGeneratorDataTtl)
	)
	defer cancel()

	// Create a buffered channel for task distribution
	tasks := make(chan genBestPathsTask, numTasks*2)
	numWorkers := runtime.GOMAXPROCS(0)
	logger.Infof("numworkers: %d Total task: %d", numWorkers, numTasks)
	// Create worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			for task := range tasks {
				results <- uc.worker(ctxTimeout, task)
			}
			wg.Done()
		}()
	}

	for _, t := range tokensToMaintain {
		tokenIn := strings.ToLower(t.TokenAddress)
		var tokenOuts []string
		for _, t := range tokensToMaintain {
			tokenOut := strings.ToLower(t.TokenAddress)
			if tokenOut != tokenIn {
				tokenOuts = append(tokenOuts, tokenOut)
			}
		}
		for _, amountInStr := range t.Amounts {
			amountIn, _ := new(big.Int).SetString(amountInStr, 10)
			task := genBestPathsTask{tokenIn, tokenOuts, amountIn}
			tasks <- task // Enqueue the task
		}
	}

	close(tasks) // Close the task channel to signal no more tasks will be added
	wg.Wait()
	close(results)

	// Deduplicate paths and store into repository
	distinctPaths := make(map[string][]*entity.MinimalPath) // distinctPaths maps between tokenIn-tokenOut to best paths
	for r := range results {
		if r == nil {
			continue
		}
		tokenIn := strings.ToLower(r.tokenIn)
		for tokenOut, paths := range r.bestPathsByTokenOut {
			tokenOut = strings.ToLower(tokenOut)
			tokenPair := fmt.Sprintf("%s-%s", tokenIn, tokenOut)
			if _, ok := distinctPaths[tokenPair]; !ok {
				distinctPaths[tokenPair] = make([]*entity.MinimalPath, 0)
			}
			distinctPaths[tokenPair] = append(distinctPaths[tokenPair], paths...)
		}
	}

	pathSuccess := 0
	pathFail := 0
	logger.Infof("distinctPaths: %d \n", len(distinctPaths))
	for tokenPair, paths := range distinctPaths {
		distinctPaths[tokenPair] = dedupBestPaths(paths)
		tokenPairSplit := strings.Split(tokenPair, "-")
		tokenIn, tokenOut := tokenPairSplit[0], tokenPairSplit[1]
		err := uc.bestPathRepository.SetBestPaths(uc.sourceHash, tokenIn, tokenOut, distinctPaths[tokenPair], uc.config.PathGeneratorDataTtl)
		if err == nil {
			pathSuccess = pathSuccess + 1
		} else {
			logger.Errorf("Error while saving path %s", err)
			pathFail = pathFail + 1
		}
	}
	logger.Infof("successfully generate and save paths-> Detail: sourceHash %v success %d paths, fail %d paths", uc.sourceHash, pathSuccess, pathFail)

}

func (uc *useCase) worker(ctx context.Context, task genBestPathsTask) *genBestPathsResult {
	paths, err := uc.generateBestPaths(ctx, task.tokenIn, task.tokenOuts, task.amountIn)
	log := logger.WithFields(logger.Fields{
		"tokenIn":   task.tokenIn,
		"tokenOuts": task.tokenOuts,
		"amountIn":  task.amountIn.String(),
	})
	if err != nil {
		log.WithFields(logger.Fields{"error": err}).Error("cannot generate best paths")
		return nil
	}

	return &genBestPathsResult{task.tokenIn, paths}
}

func (uc *useCase) generateBestPaths(
	ctx context.Context,
	tokenIn string,
	tokenOuts []string,
	amountIn *big.Int,
) (map[string][]*entity.MinimalPath, error) {
	sources := uc.config.AvailableSources

	poolAddresses, err := uc.listPoolMultiTokenOuts(ctx, tokenIn, tokenOuts)
	if err != nil {
		return nil, err
	}

	// Step 1: get pool set
	var (
		stateRoot aevmcommon.Hash
	)
	if aevmClient := uc.poolManager.GetAEVMClient(); aevmClient != nil {
		stateRoot, err = aevmClient.LatestStateRoot()
		if err != nil {
			return nil, err
		}
	}

	poolByAddress, err := uc.poolManager.GetPoolByAddress(ctx, poolAddresses.List(), sources, gEthCommon.Hash(stateRoot))
	if err != nil {
		return nil, err
	}

	allTokens := append([]string{tokenIn, uc.config.GasTokenAddress}, tokenOuts...)
	tokenByAddress, err := uc.getTokenByAddress(ctx, allTokens)
	if err != nil {
		return nil, err
	}

	tokenPriceByAddress, err := uc.getPriceByAddress(ctx, allTokens)
	if err != nil {
		return nil, err
	}

	tokenPriceUSDByAddress := make(map[string]float64, len(tokenPriceByAddress))
	for address, price := range tokenPriceByAddress {
		preferredPrice, _ := price.GetPreferredPrice()
		tokenPriceUSDByAddress[address] = preferredPrice
	}

	tokenAmountIn := poolpkg.TokenAmount{
		Token:  tokenIn,
		Amount: new(big.Int).Set(amountIn),
		AmountUsd: utils.CalcTokenAmountUsd(
			amountIn,
			tokenByAddress[tokenIn].Decimals,
			tokenPriceUSDByAddress[tokenIn],
		),
	}

	gasPrice, err := uc.getGasPrice(ctx, nil)
	if err != nil {
		return nil, err
	}
	gasTokenAddress := strings.ToLower(uc.config.GasTokenAddress)
	gasTokenPrice := tokenPriceByAddress[gasTokenAddress]
	gasTokenPriceUSD, _ := gasTokenPrice.GetPreferredPrice()

	pathsByTokenOutAddress, err := common.GenKthBestPathsV2(
		ctx,
		findroute.Input{
			TokenInAddress:   tokenIn,
			AmountIn:         new(big.Int).Set(amountIn),
			GasPrice:         gasPrice,
			GasTokenPriceUSD: gasTokenPriceUSD,
			GasInclude:       true,
		},
		findroute.FinderData{
			PoolBucket:        valueobject.NewPoolBucket(poolByAddress),
			TokenByAddress:    tokenByAddress,
			PriceUSDByAddress: tokenPriceUSDByAddress,
		},
		tokenAmountIn,
		uc.config.SPFAFinderOptions.MaxHops,
		uc.config.SPFAFinderOptions.MaxPathsToGenerate,
		uc.config.SPFAFinderOptions.MaxPathsToReturn,
	)
	if err != nil {
		return nil, err
	}

	bestPathsByTokenOutAddress := make(map[string][]*entity.MinimalPath)
	for tokenOutAddress, paths := range pathsByTokenOutAddress {
		bestPathsByTokenOutAddress[tokenOutAddress] = pathsToBestPaths(paths)
	}

	return bestPathsByTokenOutAddress, nil
}

func (uc *useCase) getTokenByAddress(
	ctx context.Context,
	tokenAddresses []string,
) (map[string]entity.Token, error) {
	tokens, err := uc.tokenRepository.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	tokenByAddress := make(map[string]entity.Token, len(tokens))
	for _, token := range tokens {
		tokenByAddress[token.Address] = *token
	}

	return tokenByAddress, nil
}

func (uc *useCase) getPriceByAddress(
	ctx context.Context,
	tokenAddresses []string,
) (map[string]*entity.Price, error) {
	prices, err := uc.priceRepository.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	priceByAddress := make(map[string]*entity.Price, len(prices))
	for _, price := range prices {
		priceByAddress[price.Address] = price
	}

	return priceByAddress, nil
}

func (uc *useCase) getGasPrice(
	ctx context.Context,
	queryGasPrice *big.Float,
) (*big.Float, error) {
	if queryGasPrice != nil {
		return queryGasPrice, nil
	}

	suggestedGasPrice, err := uc.gasRepository.GetSuggestedGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	return new(big.Float).SetInt(suggestedGasPrice), nil
}

func pathsToBestPaths(paths []*valueobject.Path) []*entity.MinimalPath {
	bestPaths := make([]*entity.MinimalPath, len(paths))
	for i, p := range paths {
		tokens := make([]string, len(p.Tokens))
		for j, t := range p.Tokens {
			tokens[j] = t.Address
		}
		bestPaths[i] = &entity.MinimalPath{
			Pools:  p.PoolAddresses,
			Tokens: tokens,
		}
	}
	return bestPaths
}
