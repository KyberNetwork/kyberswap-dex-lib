package indexpools

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"

	aevmclient "github.com/KyberNetwork/aevm/client"
	aevmcommon "github.com/KyberNetwork/aevm/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	routerpoolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getpools"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/pkg/logger"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
	"golang.org/x/exp/maps"
)

type TradeDataGenerator struct {
	poolRepo           IPoolRepository
	onchainPriceRepo   IOnchainPriceRepository
	tokenRepo          ITokenRepository
	getPoolsUseCase    IGetPoolsIncludingBasePools
	poolFactory        IPoolFactory
	aevmClient         aevmclient.Client
	minDataPointNumber int
	maxDataPointNumber int

	config TradeDataGeneratorConfig
	mu     sync.RWMutex
}

func NewTradeDataGenerator(poolRepo IPoolRepository,
	onchainPriceRepo IOnchainPriceRepository,
	tokenRepo ITokenRepository,
	getPoolsUseCase IGetPoolsIncludingBasePools,
	client aevmclient.Client,
	factory IPoolFactory,
	config TradeDataGeneratorConfig) *TradeDataGenerator {
	minDataPointNumber := config.MinDataPointNumber
	if minDataPointNumber == 0 {
		minDataPointNumber = MIN_DATA_POINT_NUMBER_DEFAULT
	}
	maxDataPointNumber := config.MaxDataPointNumber
	if maxDataPointNumber == 0 {
		minDataPointNumber = MAX_DATA_POINT_NUMBER_DEFAULT
	}
	return &TradeDataGenerator{
		poolRepo:           poolRepo,
		onchainPriceRepo:   onchainPriceRepo,
		tokenRepo:          tokenRepo,
		getPoolsUseCase:    getPoolsUseCase,
		aevmClient:         client,
		poolFactory:        factory,
		config:             config,
		minDataPointNumber: minDataPointNumber,
		maxDataPointNumber: maxDataPointNumber,
	}
}

func (u *TradeDataGenerator) ApplyConfig(config TradeDataGeneratorConfig) {
	u.mu.Lock()
	u.config = config
	u.mu.Unlock()
}

func (gen *TradeDataGenerator) getPrices(ctx context.Context, addresses []string) (map[string]*price, error) {
	result := make(map[string]*price, len(addresses))
	prices, err := gen.onchainPriceRepo.FindByAddresses(ctx, addresses)
	if err != nil {
		return nil, err
	}

	for token, p := range prices {
		if p == nil || p.USDPrice.Buy == nil && p.USDPrice.Sell == nil {
			logger.WithFields(ctx,
				logger.Fields{
					"struct": "TradeDataGenerator",
					"method": "getPrices",
				}).Errorf("getPrices prices of token %s is nil", token)
			continue
		}
		var tokenPrice price
		if p.USDPrice.Buy != nil {
			tokenPrice.buyPrice, _ = p.USDPrice.Buy.Float64()
		}
		if p.USDPrice.Sell != nil {
			tokenPrice.sellPrice, _ = p.USDPrice.Sell.Float64()
		}
		result[token] = &tokenPrice
	}

	return result, nil
}

func (gen *TradeDataGenerator) getTokens(ctx context.Context, addresses []string) (map[string]*entity.Token, error) {
	result := make(map[string]*entity.Token, len(addresses))
	tokens, err := gen.tokenRepo.FindByAddresses(ctx, addresses)
	if err != nil {
		return result, err
	}
	for _, token := range tokens {
		result[token.Address] = token
	}

	return result, nil

}

func (gen *TradeDataGenerator) getBlacklistPools(ctx context.Context) (mapset.Set[string], error) {
	blacklist, err := gen.poolRepo.GetPoolsInBlacklist(ctx)
	if err != nil {
		return mapset.NewThreadUnsafeSet[string](), err
	}

	return mapset.NewThreadUnsafeSet(blacklist...), nil

}

func (gen *TradeDataGenerator) Handle(ctx context.Context,
	output chan<- TradesGenerationOutput,
	indexBlacklistWlPools mapset.Set[string],
	input mapset.Set[TradesGenerationInput]) {
	if input.IsEmpty() {
		addresses, err := gen.poolRepo.FindAllAddresses(ctx)
		if err != nil {
			logger.WithFields(ctx,
				logger.Fields{
					"struct": "TradeDataGenerator",
					"method": "Handle",
					"error":  err,
				}).Errorf("FindAllAddresses failed")
			return
		}
		gen.handleAllPools(ctx, output, indexBlacklistWlPools, constant.DexUseSwapLimit, addresses)
		return
	}

	// pre-process input to seperate rfq dexes and others
	rfqDexes := mapset.NewThreadUnsafeSet[string]()
	addresses := make([]string, 0, input.Cardinality())
	dexUseSwapLimit := mapset.NewThreadUnsafeSet(constant.DexUseSwapLimit...)
	input.Each(func(tgi TradesGenerationInput) bool {
		if dexUseSwapLimit.ContainsOne(tgi.Exchange) {
			rfqDexes.Add(tgi.Exchange)
		}
		addresses = append(addresses, tgi.Pool)
		return false
	})

	gen.handleAllPools(ctx, output, indexBlacklistWlPools, rfqDexes.ToSlice(), addresses)

}

func (gen *TradeDataGenerator) handleAllPools(ctx context.Context,
	output chan<- TradesGenerationOutput,
	indexBlacklistWlPools mapset.Set[string],
	rfqDexes []string,
	addresses []string) {
	defer close(output)
	// 1. Prepare data for handle chunk by chunk
	availableSourceSet := mapset.NewThreadUnsafeSet(gen.config.AvailableSources...)
	prices, err := gen.getPrices(ctx, maps.Keys(gen.config.WhitelistedTokenSet))
	if err != nil {
		logger.WithFields(ctx,
			logger.Fields{
				"struct": "TradeDataGenerator",
				"method": "Handle",
				"error":  err,
			}).Errorf("getPrices failed")
		return
	}
	tokens, err := gen.getTokens(ctx, maps.Keys(gen.config.WhitelistedTokenSet))
	if err != nil {
		logger.WithFields(ctx,
			logger.Fields{
				"struct": "TradeDataGenerator",
				"method": "Handle",
				"error":  err,
			}).Errorf("getTokens failed")
		return
	}
	dynamicBlacklistPools, err := gen.getBlacklistPools(ctx)
	if err != nil {
		// ignore blacklist pools if we can not get it because blacklist pools filter is enable in pool manager
		logger.Debugf(ctx, "[TradeDataGenerator] blacklist pools get failed")
	}

	poolFilter := func(pool *entity.Pool) bool {
		if !availableSourceSet.ContainsOne(pool.Exchange) {
			return false
		}
		whitelistTokens := lo.Filter(pool.Tokens, func(token *entity.PoolToken, _ int) bool {
			return gen.config.WhitelistedTokenSet[strings.ToLower(token.Address)]
		})

		return len(whitelistTokens) >= 2
	}

	// 2. Proceed separately for RFQ dexes
	poolAddressFilter := func(addr string, _ int) bool {
		lowerAddr := strings.ToLower(addr)
		return !gen.config.BlacklistedPoolSet[lowerAddr] &&
			!dynamicBlacklistPools.ContainsOne(lowerAddr) &&
			!indexBlacklistWlPools.ContainsOne(lowerAddr)
	}
	alreadyProceed := gen.handleRFQDexes(
		ctx,
		rfqDexes,
		output,
		prices,
		tokens,
		poolFilter,
		poolAddressFilter)

	// 3. Proceed remain dexes
	nonRFQPoolAddressFilter := func(addr string, _ int) bool {
		lowerAddr := strings.ToLower(addr)
		return !alreadyProceed.ContainsOne(lowerAddr) &&
			!gen.config.BlacklistedPoolSet[lowerAddr] &&
			!dynamicBlacklistPools.ContainsOne(lowerAddr) &&
			!indexBlacklistWlPools.ContainsOne(lowerAddr)
	}
	gen.handlePools(
		ctx,
		output,
		prices,
		tokens,
		poolFilter,
		nonRFQPoolAddressFilter,
		addresses,
	)
}

func (u *TradeDataGenerator) handleRFQDexes(ctx context.Context,
	rfqDexes []string,
	output chan<- TradesGenerationOutput,
	prices map[string]*price,
	tokens map[string]*entity.Token,
	poolFilter getpools.PoolFilter,
	poolAddressFilter getpools.PoolAddressFilter) mapset.Set[string] {
	alreadyProceed := mapset.NewSet[string]()

	for _, dex := range rfqDexes {
		addresses, err := u.poolRepo.FindAddressesByDex(ctx, dex)
		if err != nil {
			logger.WithFields(ctx,
				logger.Fields{
					"struct": "TradeDataGenerator",
					"method": "rfqBatch",
					"error":  err,
				}).Errorf("FindAddressesByDex failed")
			continue
		}
		alreadyProceed.Append(addresses...)
		addresses = lo.Filter(addresses, poolAddressFilter)

		calcSwapLimit := func(poolSimulators []poolpkg.IPoolSimulator) map[string]map[string]*big.Int {
			if len(poolSimulators) == 0 {
				return nil
			}
			dexLimit := map[string]*big.Int{}

			for _, pool := range poolSimulators {
				limitMap := pool.CalculateLimit()
				for k, v := range limitMap {
					if old, exist := dexLimit[k]; !exist || old.Cmp(v) < 0 {
						dexLimit[k] = v
					}
				}
			}
			return map[string]map[string]*big.Int{poolSimulators[0].GetType(): dexLimit}

		}
		result, err := u.proceedChunk(ctx, addresses, prices, tokens, poolFilter, calcSwapLimit)
		if err != nil {
			logger.WithFields(ctx,
				logger.Fields{
					"struct": "TradeDataGenerator",
					"method": "rfqBatch",
					"error":  err,
				}).Errorf("processChunk failed")
		} else {
			output <- result
		}
	}

	return alreadyProceed

}

func (u *TradeDataGenerator) handlePools(ctx context.Context,
	output chan<- TradesGenerationOutput,
	prices map[string]*price,
	tokens map[string]*entity.Token,
	poolFilter getpools.PoolFilter,
	poolAddressFilter getpools.PoolAddressFilter,
	addresses []string) mapset.Set[string] {
	indexBlacklistTrack := mapset.NewThreadUnsafeSet[string]()

	addresses = lo.Filter(addresses, poolAddressFilter)

	// no need to calculate swap limit for AMM dexes
	noSwapLimit := func(poolSimulator []poolpkg.IPoolSimulator) map[string]map[string]*big.Int {
		return nil
	}
	chunks := lo.Chunk(addresses, u.config.ChunkSize)
	for _, chunk := range chunks {
		result, err := u.proceedChunk(ctx, chunk, prices, tokens, poolFilter, noSwapLimit)
		if err != nil {
			logger.WithFields(ctx,
				logger.Fields{
					"struct": "TradeDataGenerator",
					"method": "Handle",
					"error":  err,
				}).Errorf("processChunk failed")
		} else {
			output <- result
		}
	}

	return indexBlacklistTrack

}

func (gen *TradeDataGenerator) proceedChunk(ctx context.Context,
	chunk []string,
	prices map[string]*price,
	tokens map[string]*entity.Token,
	poolFiler getpools.PoolFilter,
	calculateSwapLimit CalculateSwapLimit,
) (TradesGenerationOutput, error) {
	tradeChan := make(chan []TradeData, len(chunk))
	var wg sync.WaitGroup
	tradeDataInChunk := []TradeData{}

	whitelistPools, err := gen.getPoolsUseCase.Handle(ctx, chunk, poolFiler)
	if err != nil {
		return TradesGenerationOutput{}, err
	}

	whitelistPools, indexBlacklistTrack := gen.removeZeroReservesPools(whitelistPools)

	var stateRoot aevmcommon.Hash
	if gen.config.UseAEVM {
		stateRoot, err = gen.aevmClient.LatestStateRoot(ctx)
		if err != nil {
			// ignore pools if we can't get last state from aevm
			// should not depend on aevm to terminate job
			logger.WithFields(ctx,
				logger.Fields{
					"struct": "TradeDataGenerator",
					"method": "processChunk",
					"error":  err,
				}).Errorf("could not get latest state root for AEVM pools")
		}
	}

	// init pool simulators and swap limits
	// when we proceed rfq dexes, we must calculate swap limits
	// However swap limit can only be calculated by fetch all pools belong to a single source set
	// So this is a little tricky, when we proceed rfq pools, we must pack all pools belong to 1 source in a single chunk, otherwise swap limit is calculated not correctly
	poolInterfaces := gen.poolFactory.NewPools(ctx, whitelistPools, common.Hash(stateRoot))
	swapLimits := gen.poolFactory.NewSwapLimit(calculateSwapLimit(poolInterfaces), types.PoolManagerExtraData{})

	// record pools that has swap errors, format: <pool:tokenA-tokenB:[]TradeData>
	hasError := map[string]map[TradePair][]TradeData{}
	// <pool:tokenA-tokenB:[]TradeData>
	result := map[string]map[TradePair][]TradeData{}

	for _, pool := range poolInterfaces {
		// ignore aevm pools if we can't get latest state from aevm
		if gen.config.UseAEVM && gen.config.DexUseAEVM[pool.GetExchange()] && len(stateRoot) == 0 {
			continue
		}
		poolTokens := pool.GetTokens()

		for i := 0; i < len(poolTokens); i++ {
			tokenI := poolTokens[i]
			if !gen.config.WhitelistedTokenSet[tokenI] || !gen.hasReserve(tokenI) {
				continue
			}
			targets := pool.CanSwapFrom(tokenI)
			for j := 0; j < len(targets); j++ {
				tokenJ := targets[j]
				if !gen.config.WhitelistedTokenSet[tokenJ] || tokenJ == tokenI || !gen.hasReserve(tokenJ) {
					continue
				}

				if prices[tokenI] == nil || prices[tokenJ] == nil {
					logger.WithFields(ctx,
						logger.Fields{
							"struct": "TradeDataGenerator",
							"method": "processChunk",
						}).Errorf("prices of token %s or %s is nil", tokenI, tokenJ)
					continue
				}

				if gen.config.UseAEVM && gen.config.DexUseAEVM[pool.GetExchange()] {
					wg.Add(1)
					go func(ctx context.Context,
						tokenIn, tokenOut string,
						tokens map[string]*entity.Token,
						prices map[string]*price,
						pool poolpkg.IPoolSimulator) {
						defer wg.Done()
						trade := gen.generateTradeData(ctx, tokenI, tokenJ, tokens, prices, pool, swapLimits[pool.GetType()])
						tradeChan <- trade
					}(ctx, tokenI, tokenJ, tokens, prices, pool)
					continue
				}

				// for every pair of tokens, we need at least 6 data points with amount range from 10^0...10^6
				// for some pools might serve more amount in larger than 10^6, we can have maximum 12 data points
				tradeData := gen.generateTradeData(ctx, tokenI, tokenJ, tokens, prices, pool, swapLimits[pool.GetType()])
				tradeDataInChunk = append(tradeDataInChunk, tradeData...)
			}
		}
	}

	// close trade chan, must be in another goroutine to avoid locking
	go func() {
		wg.Wait()
		close(tradeChan)
	}()

	for tr := range tradeChan {
		tradeDataInChunk = append(tradeDataInChunk, tr...)
	}

	for _, tr := range tradeDataInChunk {
		pair := TradePair{tokenIn: tr.TokenIn, tokenOut: tr.TokenOut}
		if tr.hasError() {
			if _, ok := hasError[tr.Pool]; !ok {
				hasError[tr.Pool] = map[TradePair][]TradeData{}
			}
			tr.ErrMessage = tr.getError()
			hasError[tr.Pool][pair] = append(hasError[tr.Pool][pair], tr)
			continue
		}
		if _, ok := result[tr.Pool]; !ok {
			result[tr.Pool] = map[TradePair][]TradeData{}
		}
		result[tr.Pool][pair] = append(result[tr.Pool][pair], tr)
	}

	// only allow pools that has at least 2 tokens that can swap successfully from each other
	// for pools with only 2 tokens, but they allow only one direction swap, these pools are still valid to be indexed
	for p, trades := range result {
		_, ok := hasError[p]
		if ok && len(trades) < 1 {
			delete(result, p)
		}
	}
	// Add zero reserves pools and pools which doesn't yeild any successful swaps to blacklist
	indexBlacklistTrack = indexBlacklistTrack.Union(gen.getExhaustedReservesWhitelistPools(result, hasError))

	return TradesGenerationOutput{
		Successed: result,
		Failed:    hasError,
		Blacklist: indexBlacklistTrack,
	}, nil
}

// return values: trade data that are generated from calcAmountOut, error and list of error pool address
// TODO: need to take into account below case:
// at 10$ amountOut = X, at 100$ error, but at 1000$ we still have valid amountOut, what should we set data point for 100$ amount in?
func (gen *TradeDataGenerator) generateTradeData(ctx context.Context,
	tokenIn, tokenOut string,
	tokens map[string]*entity.Token,
	prices map[string]*price,
	pool poolpkg.IPoolSimulator,
	limit poolpkg.SwapLimit) []TradeData {
	calcAmountOutInstance := routerpoolpkg.NewCustomFuncs(gen.config.DexUseAEVM)
	result := []TradeData{}
	lastTradeData := TradeData{AmountOutUsd: float64(-1)}

	for i := 0; i <= gen.minDataPointNumber || (!lastTradeData.hasError() && i <= gen.maxDataPointNumber); i++ {
		amountInUsd := math.Pow10(i)
		// use sell price for token in
		amountIn := business.CalcAmountFromUSD(amountInUsd, tokens[tokenIn].Decimals, prices[tokenIn].getSellPrice())
		amountOut, err := calcAmountOutInstance.CalcAmountOut(ctx, pool, poolpkg.TokenAmount{
			Token:     tokenIn,
			Amount:    amountIn,
			AmountUsd: amountInUsd,
		}, tokenOut, limit)

		var swapErrResult error
		var amountOutUsdResult float64
		if err != nil {
			// If we haven't executed any swap successfully yet, trade data results error
			if lastTradeData.AmountOutUsd == float64(-1) {
				swapErrResult = err
			} else {
				// If we have executed at least one swap successfully,
				// current trade data output will be equal to last successful swap output,
				// trade data is considered to be successful
				amountOutUsdResult = lastTradeData.AmountOutUsd
			}
			lastTradeData.Err = err
		} else if amountOut == nil || !amountOut.IsValid() {
			// Some dex ex: limit order, calcAmountOut doesn't return error, instead it returns 0 amount out
			// We need to take into account these dexes and consider calcAmountOut error if it returns 0
			lastTradeData.Err = fmt.Errorf("calcAmountOut error %v amountOut %v", ErrAmountOutNotValid, amountOut)
			swapErrResult = fmt.Errorf("calcAmountOut error %v amountOut %v", ErrAmountOutNotValid, amountOut)
		} else {
			// use buy price for token out
			amountOutUsdResult, _ = business.CalcAmountUSD(amountOut.TokenAmountOut.Amount, tokens[tokenOut].Decimals, prices[tokenOut].getBuyPrice()).Float64()
			// Handle case where 2 consecutive points have a big diffrentiate prices impact, we have to generate 2 extra points between them
			// example: if At data point 10^1, P1 = 40%; At data point 10^2 P2 = 90%, then delta price impact = 50%, then we have to generate more extra point 20 and 50
			priceImpact := (amountInUsd - amountOutUsdResult) / amountInUsd
			if i > 0 && i < MAX_EXPONENT_GENERATE_EXTRA_POINT && math.Abs(priceImpact-lastTradeData.PriceImpact) > PRICE_IMPACT_THRESHOLD {
				extraPoints := gen.generateExtraDataPointsTradeData(ctx, tokenIn, tokenOut, tokens, prices, pool, limit, calcAmountOutInstance.CalcAmountOut, i-1)
				result = append(result, extraPoints...)
			}
			lastTradeData = TradeData{
				TokenIn:      tokenIn,
				TokenOut:     tokenOut,
				PriceImpact:  priceImpact,
				AmountInUsd:  amountInUsd,
				AmountOutUsd: amountOutUsdResult,
				Pool:         pool.GetAddress(),
				AmountIn:     amountIn.Text(10),
				Dex:          pool.GetExchange(),
			}

			if amountOutUsdResult == float64(0) {
				logger.WithFields(ctx,
					logger.Fields{
						"struct": "TradeDataGenerator",
						"method": "generateTradeData",
					}).Errorf("amountOutUsd is zero in trade data %v", lastTradeData)
			}
		}

		result = append(result, TradeData{
			TokenIn:      tokenIn,
			TokenOut:     tokenOut,
			AmountInUsd:  amountInUsd,
			AmountOutUsd: amountOutUsdResult,
			Pool:         pool.GetAddress(),
			Err:          swapErrResult,
			AmountIn:     amountIn.Text(10),
			Dex:          pool.GetExchange(),
		})
	}

	return result

}

func (gen *TradeDataGenerator) generateExtraDataPointsTradeData(ctx context.Context,
	tokenIn, tokenOut string,
	tokens map[string]*entity.Token,
	prices map[string]*price,
	pool poolpkg.IPoolSimulator,
	limit poolpkg.SwapLimit,
	calcAmountFunc func(ctx context.Context, pool poolpkg.IPoolSimulator, tokenAmountIn poolpkg.TokenAmount, tokenOut string, limit poolpkg.SwapLimit) (*poolpkg.CalcAmountOutResult, error),
	exponent int) []TradeData {
	amountInList := make([]float64, 0, 2)
	amountInList = append(amountInList, 2*math.Pow10(exponent))
	amountInList = append(amountInList, 5*math.Pow10(exponent))
	result := make([]TradeData, 0, 2)

	for _, amountInUsd := range amountInList {
		// use sell price for token in
		amountIn := business.CalcAmountFromUSD(amountInUsd, tokens[tokenIn].Decimals, prices[tokenIn].getSellPrice())
		amountOut, err := calcAmountFunc(ctx, pool, poolpkg.TokenAmount{
			Token:     tokenIn,
			Amount:    amountIn,
			AmountUsd: amountInUsd,
		}, tokenOut, limit)

		if err != nil {
			logger.WithFields(ctx,
				logger.Fields{
					"struct": "TradeDataGenerator",
					"method": "generateExtraDataPointsTradeData",
				}).Errorf("error when calculate amount out %v poolAddress %v amountInUsd %f", err, pool.GetAddress(), amountInUsd)
			continue
		}

		amountOutUsdResult, _ := business.CalcAmountUSD(amountOut.TokenAmountOut.Amount, tokens[tokenOut].Decimals, prices[tokenOut].getBuyPrice()).Float64()
		result = append(result, TradeData{
			TokenIn:      tokenIn,
			TokenOut:     tokenOut,
			AmountInUsd:  amountInUsd,
			AmountOutUsd: amountOutUsdResult,
			Pool:         pool.GetAddress(),
			AmountIn:     amountIn.Text(10),
			Dex:          pool.GetExchange(),
		})
	}

	return result

}
