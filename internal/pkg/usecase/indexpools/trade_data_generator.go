package indexpools

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"math/big"
	"os"
	"strings"
	"sync"

	aevmclient "github.com/KyberNetwork/aevm/client"
	aevmcommon "github.com/KyberNetwork/aevm/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	dexlibValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/iter"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	routerpoolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/poolrank"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getpools"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	ctxutils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

var ErrTokensHaveWrongPrice = errors.New("tokens have wrong prices")
var ErrNotEnoughSuccessTradeData = errors.New("can not generate enough success trade data")
var ErrNotFindStartAmount = errors.New("can not find start amount")

type TradeDataGenerator struct {
	poolRepo           IPoolRepository
	onchainPriceRepo   IOnchainPriceRepository
	tokenRepo          ITokenRepository
	getPoolsUseCase    IGetPoolsIncludingBasePools
	poolFactory        IPoolFactory
	aevmClient         aevmclient.Client
	minDataPointNumber int
	maxDataPointNumber int
	keyGenerator       poolrank.KeyGenerator
	poolRankRepo       IPoolRankRepository

	config *TradeDataGeneratorConfig
	mu     sync.RWMutex
}

func NewTradeDataGenerator(poolRepo IPoolRepository,
	onchainPriceRepo IOnchainPriceRepository,
	tokenRepo ITokenRepository,
	poolRankRepo IPoolRankRepository,
	getPoolsUseCase IGetPoolsIncludingBasePools,
	client aevmclient.Client,
	factory IPoolFactory,
	config *TradeDataGeneratorConfig) *TradeDataGenerator {
	minDataPointNumber := config.MinDataPointNumber
	if minDataPointNumber == 0 {
		minDataPointNumber = MIN_DATA_POINT_NUMBER_DEFAULT
	}
	maxDataPointNumber := config.MaxDataPointNumber
	if maxDataPointNumber == 0 {
		minDataPointNumber = MAX_DATA_POINT_NUMBER_DEFAULT
	}
	if config.InvalidPriceImpactThreshold == 0.0 {
		config.InvalidPriceImpactThreshold = INVALID_PRICE_IMPACT_THRESHOLD
	}
	return &TradeDataGenerator{
		poolRepo:           poolRepo,
		onchainPriceRepo:   onchainPriceRepo,
		tokenRepo:          tokenRepo,
		poolRankRepo:       poolRankRepo,
		getPoolsUseCase:    getPoolsUseCase,
		aevmClient:         client,
		poolFactory:        factory,
		config:             config,
		minDataPointNumber: minDataPointNumber,
		maxDataPointNumber: maxDataPointNumber,
		keyGenerator:       *poolrank.NewKeyGenerator(config.ChainName),
	}
}

func (gen *TradeDataGenerator) ApplyConfig(config TradeDataGeneratorConfig) {
	gen.mu.Lock()
	gen.config.AvailableSources = config.AvailableSources
	gen.config.DexUseAEVM = config.DexUseAEVM
	gen.mu.Unlock()
}

func (gen *TradeDataGenerator) findPricesByAddresses(ctx context.Context, addresses []string) (map[string]*routerEntity.OnchainPrice, error) {
	result := make(map[string]*routerEntity.OnchainPrice, len(addresses))
	prices, err := gen.onchainPriceRepo.FindByAddresses(ctx, addresses)
	if err != nil {
		return nil, err
	}

	for token, p := range prices {
		if p == nil || p.USDPrice.Buy == nil && p.USDPrice.Sell == nil {
			log.Ctx(ctx).Error().
				Str("struct", "TradeDataGenerator").
				Str("method", "getPrices").
				Str("token", token).
				Msg("getPrices prices of token is nil")
			continue
		}
		result[token] = p
	}

	return result, nil
}

func (gen *TradeDataGenerator) getPrices(ctx context.Context, tokens mapset.Set[string]) map[string]*routerEntity.OnchainPrice {
	prices := make(map[string]*routerEntity.OnchainPrice, tokens.Cardinality())
	chunks := lo.Chunk(tokens.ToSlice(), PRICE_CHUNK_SIZE)
	mapper := iter.Mapper[[]string, map[string]*routerEntity.OnchainPrice]{MaxGoroutines: gen.config.MaxGoroutines}
	chunkResults := mapper.Map(chunks, func(chunk *[]string) map[string]*routerEntity.OnchainPrice {
		prices, err := gen.findPricesByAddresses(ctx, *chunk)
		if err != nil {
			log.Ctx(ctx).Err(err).
				Str("struct", "TradeDataGenerator").
				Str("method", "getPrices").
				Msg("getPrices failed")
		}

		return prices
	})
	for _, res := range chunkResults {
		if res == nil {
			continue
		}
		for token, price := range res {
			prices[token] = price
		}
	}

	return prices
}

func (gen *TradeDataGenerator) getTokens(ctx context.Context, tokens mapset.Set[string]) map[string]*entity.SimplifiedToken {
	result := make(map[string]*entity.SimplifiedToken, tokens.Cardinality())
	chunks := lo.Chunk(tokens.ToSlice(), PRICE_CHUNK_SIZE)
	mapper := iter.Mapper[[]string, []*entity.SimplifiedToken]{MaxGoroutines: gen.config.MaxGoroutines}
	chunkResults := mapper.Map(chunks, func(chunk *[]string) []*entity.SimplifiedToken {
		tokens, err := gen.tokenRepo.FindByAddresses(ctx, *chunk)
		if err != nil {
			log.Ctx(ctx).Err(err).
				Str("struct", "TradeDataGenerator").
				Str("method", "getTokens").
				Msg("getTokens failed")
		}

		return tokens
	})
	for _, res := range chunkResults {
		if res == nil {
			continue
		}
		for _, token := range res {
			result[token.Address] = token
		}
	}

	return result
}

func (gen *TradeDataGenerator) getBlacklistPools(ctx context.Context) (mapset.Set[string], error) {
	blacklist, err := gen.poolRepo.GetPoolsInBlacklist(ctx)
	if err != nil {
		return mapset.NewThreadUnsafeSet[string](), err
	}

	return mapset.NewThreadUnsafeSet(blacklist...), nil

}

func (gen *TradeDataGenerator) Handle(ctx context.Context,
	indexBlacklistWlPools mapset.Set[string],
	input mapset.Set[TradesGenerationInput]) TradeDataGenerationResult {
	if input.IsEmpty() {
		addresses, err := gen.poolRepo.FindAllAddresses(ctx)
		if err != nil {
			log.Ctx(ctx).Err(err).
				Str("struct", "TradeDataGenerator").
				Str("method", "Handle").
				Msg("FindAllAddresses failed")
			return TradeDataGenerationResult{}
		}
		return gen.handleAllPools(ctx, indexBlacklistWlPools, constant.DexUseSwapLimit, addresses)
	}

	// pre-process input to separate rfq dexes and others
	dexUseSwapLimit := mapset.NewThreadUnsafeSet[string]()
	addresses := make([]string, 0, input.Cardinality())
	dexUseSwapLimitMap := mapset.NewThreadUnsafeSet(constant.DexUseSwapLimit...)
	input.Each(func(tgi TradesGenerationInput) bool {
		if dexUseSwapLimitMap.ContainsOne(tgi.Exchange) {
			dexUseSwapLimit.Add(tgi.Exchange)
		}
		addresses = append(addresses, tgi.Pool)
		return false
	})

	return gen.handleAllPools(ctx, indexBlacklistWlPools, dexUseSwapLimit.ToSlice(), addresses)

}

func (gen *TradeDataGenerator) handleAllPools(ctx context.Context,
	indexBlacklistWlPools mapset.Set[string],
	dexUseSwapLimit []string,
	addresses []string) TradeDataGenerationResult {
	// 1. Prepare data for handle chunk by chunk
	availableSourceSet := mapset.NewThreadUnsafeSet(gen.config.AvailableSources...)
	dynamicBlacklistPools, err := gen.getBlacklistPools(ctx)
	if err != nil {
		// ignore blacklist pools if we can not get it because blacklist pools filter is enable in pool manager
		log.Ctx(ctx).Debug().Msg("[TradeDataGenerator] blacklist pools get failed")
	}

	poolFilter := func(pool *entity.Pool) bool {
		return availableSourceSet.ContainsOne(pool.Exchange)
	}

	// 2. Proceed separately for RFQ dexes
	poolAddressFilter := func(addr string, _ int) bool {
		lowerAddr := strings.ToLower(addr)
		return !gen.config.BlacklistedPoolSet[lowerAddr] &&
			!dynamicBlacklistPools.ContainsOne(lowerAddr) &&
			!indexBlacklistWlPools.ContainsOne(lowerAddr)
	}
	alreadyProceedPools, rfqResult := gen.handleRFQDexes(
		ctx,
		dexUseSwapLimit,
		poolFilter,
		poolAddressFilter)

	// 3. Proceed remain dexes
	nonRFQPoolAddressFilter := func(addr string, _ int) bool {
		lowerAddr := strings.ToLower(addr)
		return !alreadyProceedPools.ContainsOne(lowerAddr) &&
			!gen.config.BlacklistedPoolSet[lowerAddr] &&
			!dynamicBlacklistPools.ContainsOne(lowerAddr) &&
			!indexBlacklistWlPools.ContainsOne(lowerAddr)
	}
	allPoolResult := gen.handlePools(
		ctx,
		poolFilter,
		nonRFQPoolAddressFilter,
		addresses,
	)

	return TradeDataGenerationResult{
		Blacklist:       rfqResult.Blacklist.Union(allPoolResult.Blacklist),
		OutputFileNames: rfqResult.OutputFileNames.Union(allPoolResult.OutputFileNames),
	}
}

func (gen *TradeDataGenerator) handleRFQDexes(ctx context.Context,
	dexUseSwapLimit []string,
	poolFilter getpools.PoolFilter,
	poolAddressFilter getpools.PoolAddressFilter) (mapset.Set[string], TradeDataGenerationResult) {
	alreadyProceed := mapset.NewThreadUnsafeSet[string]()
	blacklistPools := mapset.NewThreadUnsafeSet[string]()
	var zeroPoolScores []routerEntity.PoolScore
	jobID := ctxutils.GetJobID(ctx)
	fileNameResults := mapset.NewThreadUnsafeSet[string]()

	for _, dex := range dexUseSwapLimit {
		addresses, err := gen.poolRepo.FindAddressesByDex(ctx, string(dex))
		if err != nil {
			log.Ctx(ctx).Err(err).
				Str("struct", "TradeDataGenerator").
				Str("method", "handleRFQDexes").
				Msg("FindAddressesByDex failed")
			continue
		}
		log.Ctx(ctx).Error().
			Str("struct", "TradeDataGenerator").
			Str("method", "handleRFQDexes").
			Str("dex", dex).
			Int("chunkLen", len(addresses)).
			Msg("Start indexing rfq dex")
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
		result, err := gen.proceedChunk(ctx, addresses, poolFilter, calcSwapLimit, mapset.NewThreadUnsafeSet[string]())

		if err != nil {
			log.Ctx(ctx).Err(err).
				Str("struct", "TradeDataGenerator").
				Str("method", "rfqBatch").
				Msg("processChunk failed")
		} else {
			filename := fmt.Sprintf("Chunk_%s.txt_%s", dex, jobID)
			fileNames, err := gen.writeTradeData(ctx, result, filename)
			log.Ctx(ctx).Info().
				Str("struct", "TradeDataGenerator").
				Str("method", "handleRFQDexes").
				Str("dex", dex).
				Int("blackedList", result.Blacklist.Cardinality()).
				Int("zeroPoolScore", len(result.ZeroScorePools)).
				Int("success", len(result.Successed)).
				Stringer("fileNames", fileNames).
				Msg("Start write trade data into file")
			blacklistPools = blacklistPools.Union(result.Blacklist)
			zeroPoolScores = append(zeroPoolScores, result.ZeroScorePools...)
			fileNameResults = fileNameResults.Union(fileNames)
			if err != nil {
				log.Ctx(ctx).Err(err).
					Str("struct", "TradeDataGenerator").
					Str("method", "rfqBatch").
					Msg("writeTradeData failed")
			}
		}
	}

	gen.saveZeroScore(ctx, zeroPoolScores)

	return alreadyProceed, TradeDataGenerationResult{
		Blacklist:       blacklistPools,
		OutputFileNames: fileNameResults,
	}

}

func (gen *TradeDataGenerator) handlePools(ctx context.Context,
	poolFilter getpools.PoolFilter,
	poolAddressFilter getpools.PoolAddressFilter,
	addresses []string) TradeDataGenerationResult {

	addresses = lo.Filter(addresses, poolAddressFilter)
	fileNames := mapset.NewThreadUnsafeSet[string]()
	blacklistPools := mapset.NewThreadUnsafeSet[string]()
	seenBasePools := mapset.NewThreadUnsafeSet[string]()

	// no need to calculate swap limit for AMM dexes
	noSwapLimit := func(poolSimulator []poolpkg.IPoolSimulator) map[string]map[string]*big.Int {
		return nil
	}
	chunks := lo.Chunk(addresses, gen.config.ChunkSize)
	jobID := ctxutils.GetJobID(ctx)
	for startId, chunk := range chunks {
		fileName := fmt.Sprintf("Chunk_%d.txt_%s", startId+1, jobID)
		// 0 is always RFQ chunk, so fileId of AMM always starts from 1
		result, err := gen.proceedChunk(ctx, chunk, poolFilter, noSwapLimit, seenBasePools)
		if err != nil {
			log.Ctx(ctx).Err(err).
				Str("struct", "TradeDataGenerator").
				Str("method", "Handle").
				Msg("processChunk failed")
		} else {
			files, err := gen.writeTradeData(ctx, result, fileName)
			if err != nil {
				log.Ctx(ctx).Err(err).
					Str("struct", "TradeDataGenerator").
					Str("method", "Handle").
					Msg("writeTradeData failed")
			} else {
				fileNames = fileNames.Union(files)
			}
			blacklistPools = blacklistPools.Union(result.Blacklist)
			gen.saveZeroScore(ctx, result.ZeroScorePools)
		}
	}

	return TradeDataGenerationResult{
		Blacklist:       blacklistPools,
		OutputFileNames: fileNames,
	}

}

func (gen *TradeDataGenerator) writeTradeData(ctx context.Context, output TradesGenerationOutput, filename string) (mapset.Set[string], error) {
	var whitelistBuffer *bufio.Writer
	var successedBuffer *bufio.Writer
	names := mapset.NewThreadUnsafeSet[string]()

	var failedBuffer *bufio.Writer
	if gen.config.ExportFailedTrade {
		failedFile, err := os.OpenFile(strings.Join([]string{gen.config.FilePath, filename}, ""), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Ctx(ctx).Err(err).
				Str("struct", "TradeDataGenerator").
				Str("method", "writeTradeData").
				Msg("init failed buffer failed")
		}
		defer failedFile.Close()
		failedBuffer = bufio.NewWriter(failedFile)
	}

	for tradeDataId, input := range output.Successed {
		jsonStr, err := json.Marshal(input)
		if err != nil {
			continue
		}
		log.Ctx(ctx).Debug().Msgf("Generate trade data success data %s\n",
			fmt.Sprintf("%s:%s:%s\n", tradeDataId.Pool, tradeDataId.Type, jsonStr))
		if tradeDataId.Type == valueobject.WHITELIST_WHITELIST {
			if whitelistBuffer == nil {
				whitelistFileName := strings.Join([]string{gen.config.FilePath, WHITELIST_FILENAME}, "")
				// open single file for whitelist-whitelist set
				whitelistFile, err := os.OpenFile(whitelistFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return names, err
				}
				defer whitelistFile.Close()
				names.Add(whitelistFileName)
				whitelistBuffer = bufio.NewWriter(whitelistFile)
			}
			whitelistBuffer.WriteString(fmt.Sprintf("%s,%s,%s\n", tradeDataId.Type, tradeDataId.Pool, jsonStr))
			continue
		}

		if successedBuffer == nil {
			name := strings.Join([]string{gen.config.FilePath, filename}, "")
			file, err := os.Create(name)
			if err != nil {
				return names, err
			}
			defer file.Close()
			names.Add(name)
			successedBuffer = bufio.NewWriter(file)
		}
		successedBuffer.WriteString(fmt.Sprintf("%s,%s,%s\n", tradeDataId.Type, tradeDataId.Pool, jsonStr))
	}

	for p, errTrades := range output.Failed {
		jsonErr, err := json.Marshal(errTrades)
		if err != nil {
			continue
		}
		// push logs to grafana
		if gen.config.ExportFailedTrade {
			failedBuffer.Write([]byte(fmt.Sprintf("%s:%s\n", p, jsonErr)))
		} else if gen.config.LogError {
			log.Ctx(ctx).Error().Msgf("Generate trade data failed %s:%s", p, jsonErr)
		}
	}

	if successedBuffer != nil {
		successedBuffer.Flush()
	}

	if whitelistBuffer != nil {
		whitelistBuffer.Flush()
	}

	return names, nil
}

func (gen *TradeDataGenerator) proceedChunk(ctx context.Context,
	chunk []string,
	poolFiler getpools.PoolFilter,
	calculateSwapLimit CalculateSwapLimit,
	seenBasePools mapset.Set[string],
) (TradesGenerationOutput, error) {
	tradeChan := make(chan []TradeData, len(chunk))
	var wg sync.WaitGroup
	var tradeDataInChunk []TradeData

	pools, err := gen.getPoolsUseCase.Handle(ctx, chunk, poolFiler)
	if err != nil {
		return TradesGenerationOutput{}, err
	}

	if len(pools) == 0 {
		return TradesGenerationOutput{}, errors.New("can not get pool entities or all pools are filtered out")
	}

	totalPools, indexBlacklistTrack := gen.removeZeroReservesPools(pools)

	// collect all tokens
	tokenAddresses := mapset.NewThreadUnsafeSet[string]()
	for _, p := range pools {
		if !p.HasReserves() {
			continue
		}
		for _, t := range p.Tokens {
			tokenAddresses.Add(t.Address)
		}
	}
	if tokenAddresses.IsEmpty() {
		return TradesGenerationOutput{}, errors.New("tokens is empty")
	}

	// Get prices and tokens data
	// Get tokens before prices to reuse local cache
	tokens := gen.getTokens(ctx, tokenAddresses)
	if len(tokens) == 0 {
		return TradesGenerationOutput{}, fmt.Errorf("get tokens from Redis failed %v", tokenAddresses)
	}

	prices := gen.getPrices(ctx, tokenAddresses)
	if len(prices) == 0 {
		return TradesGenerationOutput{}, errors.New("get prices from onchain prices failed")
	}

	nativePriceUsdBF, err := gen.onchainPriceRepo.GetNativePriceInUsd(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err).
			Str("struct", "TradeDataGenerator").
			Str("method", "processChunk").
			Msg("could not get native price in usd")
	}
	nativePriceUsd, _ := nativePriceUsdBF.Float64()

	var stateRoot aevmcommon.Hash
	if gen.config.UseAEVM {
		stateRoot, err = gen.aevmClient.LatestStateRoot(ctx)
		if err != nil {
			// ignore pools if we can't get last state from aevm
			// should not depend on aevm to terminate job
			log.Ctx(ctx).Err(err).
				Str("struct", "TradeDataGenerator").
				Str("method", "processChunk").
				Msg("could not get latest state root for AEVM pools")
		}
	}

	// init pool simulators and swap limits
	// when we proceed rfq dexes, we must calculate swap limits
	// However swap limit can only be calculated by fetch all pools belong to a single source set
	// So this is a little tricky, when we proceed rfq pools, we must pack all pools belong to 1 source in a single chunk, otherwise swap limit is calculated not correctly
	poolInterfaces := gen.poolFactory.NewPoolsIgnoreAEVM(ctx, totalPools, common.Hash(stateRoot), gen.config.UseAEVM)
	swapLimits := gen.poolFactory.NewSwapLimit(calculateSwapLimit(poolInterfaces), types.PoolManagerExtraData{})

	// record pools that has swap errors, format: <pool:[]LiquidityScoreCalcInput>
	hasError := map[TradeDataId]*LiquidityScoreCalcInput{}
	result := map[TradeDataId]*LiquidityScoreCalcInput{}
	tvl := map[string]float64{}
	totalPoolAddresses := mapset.NewThreadUnsafeSet[string]()
	// when a pair of token have no price, we don't need to generate trade data, just init PoolScore as zero and save them to Redis
	zeroPoolScores := []routerEntity.PoolScore{}
	// keep entity.Pools in order to calculate tvl
	poolMap := map[string]*entity.Pool{}
	for _, p := range totalPools {
		poolMap[p.Address] = p
		totalPoolAddresses.Add(p.Address)
	}
	if totalPoolAddresses.Cardinality() != len(chunk) {
		log.Ctx(ctx).Info().Msgf("total PoolAddresses is differ from chunk size diff pool %v",
			mapset.NewThreadUnsafeSet(chunk...).Difference(totalPoolAddresses))
	}
	tradaDataCount := 0
	setsNeededTobeIndexed := mapset.NewThreadUnsafeSet[valueobject.TradeDataType]()
	for key := range gen.config.SetsNeededTobeIndexed {
		setsNeededTobeIndexed.Add(valueobject.TradeDataType(key))
	}

	for _, pool := range poolInterfaces {
		if seenBasePools.ContainsOne(pool.GetAddress()) {
			continue
		}
		// ignore aevm pools if we can't get latest state from aevm
		if gen.config.UseAEVM && gen.config.DexUseAEVM[pool.GetExchange()] && stateRoot == (aevmcommon.Hash{}) {
			continue
		}
		poolTokens := pool.GetTokens()
		tvlNative, err := business.CalculatePoolTVL(ctx, poolMap[pool.GetAddress()], prices, true)
		if err != nil {
			log.Ctx(ctx).Err(err).
				Str("struct", "TradeDataGenerator").
				Str("method", "processChunk").
				Str("pool", pool.GetAddress()).
				Msg("tvlNative could not be calculated")
			tvlNative = 0.0
		}
		tvl[pool.GetAddress()] = tvlNative * nativePriceUsd

		for i := range poolTokens {
			tokenI := poolTokens[i]

			// all logic from curve family is already covered in CanSwapFrom
			targets := pool.CanSwapFrom(tokenI)
			for j := range targets {
				tokenJ := targets[j]
				tradeDataTypes := gen.getPairType(tokenI, tokenJ)
				// filter out tokens that are not satisfied the conditions
				if tokenJ == tokenI || !setsNeededTobeIndexed.ContainsAny(tradeDataTypes...) {
					continue
				}
				tradaDataCount++

				if !gen.hasReserve(pool, poolMap[pool.GetAddress()], tokenI) && !gen.hasReserve(pool,
					poolMap[pool.GetAddress()], tokenJ) {
					if gen.config.LogError {
						log.Ctx(ctx).Info().Msgf("token has no reserve both direction - direct set tokenI %s tokenJ %s pool %s i %d j %d\n",
							tokenI, tokenJ, pool.GetAddress(), i, j)
					}
					zeroPoolScores = append(zeroPoolScores, routerEntity.PoolScore{
						Key: gen.keyGenerator.DirectPairKeyWithoutSort(poolrank.SortByLiquidityScoreTvl, tokenI,
							tokenJ),
						Pool:     pool.GetAddress(),
						TvlInUsd: tvl[pool.GetAddress()],
					})
					continue
				}

				// handle some special cases where pool.GetTokens() differ from canSwapFrom/canSwapTo tokens
				if tokens[tokenI] == nil || tokens[tokenJ] == nil {
					missedTokens := gen.getTokens(ctx, mapset.NewThreadUnsafeSet(tokenI, tokenJ))
					missedPrices := gen.getPrices(ctx, mapset.NewThreadUnsafeSet(tokenI, tokenJ))
					for t, v := range missedTokens {
						tokens[t] = v
						prices[t] = missedPrices[t]
					}
				}

				if (prices[tokenI] == nil || prices[tokenI].GetSellPriceIfAny() == 0) &&
					(prices[tokenJ] == nil || prices[tokenJ].GetBuyPriceIfAny() == 0) {
					if gen.config.LogError {
						log.Ctx(ctx).Info().Msgf("debug prices is nil - direct set tokenI %s tokenJ %s pool %s\n",
							tokenI, tokenJ, pool.GetAddress())
					}
					zeroPoolScores = append(zeroPoolScores, routerEntity.PoolScore{
						Key: gen.keyGenerator.DirectPairKeyWithoutSort(poolrank.SortByLiquidityScoreTvl, tokenI,
							tokenJ),
						Pool:     pool.GetAddress(),
						TvlInUsd: tvl[pool.GetAddress()],
					})
					continue
				}
				// If pool contains number of tokens greater than max allowance, we will not calculate liquidity score for this pool
				if len(poolTokens) > gen.config.MaxTokensLen {
					tvlOfPair, err := business.CalculatePoolTVLForTokenPair(ctx, poolMap[pool.GetAddress()], prices,
						[]int{i, j})
					if err != nil {
						log.Ctx(ctx).Err(err).
							Str("struct", "TradeDataGenerator").
							Str("method", "generateTradeData").
							Str("pool", pool.GetAddress()).
							Ints("tokenId", []int{i, j}).
							Msg("calculate pool tvl for token pair failed")
					}
					keys := gen.generateTradeDataKey(tokenI, tokenJ)
					for _, key := range keys {
						zeroPoolScores = append(zeroPoolScores, routerEntity.PoolScore{
							Key:            key,
							Pool:           pool.GetAddress(),
							TvlInUsd:       tvlOfPair,
							LiquidityScore: gen.config.PoolHasManyTokensDefaultScore,
						})
					}

					continue
				}

				if gen.config.UseAEVM && gen.config.DexUseAEVM[pool.GetExchange()] {
					if tvlNative >= MIN_TVL_USD_AEVM_POOL_THRESHOLD {
						keys := gen.generateTradeDataKey(tokenI, tokenJ)
						for _, k := range keys {
							zeroPoolScores = append(zeroPoolScores, routerEntity.PoolScore{
								Key:            k,
								Pool:           pool.GetAddress(),
								TvlInUsd:       tvlNative,
								LiquidityScore: DEFAULT_AEVM_POOL_SCORE,
							})
						}
					} else {
						wg.Add(1)
						go func(ctx context.Context,
							tokenIn, tokenOut string,
							tokens map[string]*entity.SimplifiedToken,
							prices map[string]*routerEntity.OnchainPrice,
							pool poolpkg.IPoolSimulator) {
							defer wg.Done()
							trade := gen.generateTradeData(ctx, tokenI, tokenJ, tokens, prices, pool,
								swapLimits[pool.GetType()], tradeDataTypes[0])
							tradeChan <- trade
						}(ctx, tokenI, tokenJ, tokens, prices, pool)
					}
					continue
				}

				// for every pair of tokens, we need at least 6 data points with amount range from 10^0...10^6
				// for some pools might serve more amount in larger than 10^6, we can have maximum 12 data points
				tradeData := gen.generateTradeData(ctx, tokenI, tokenJ, tokens, prices, pool,
					swapLimits[pool.GetType()], tradeDataTypes[0])
				tradeDataInChunk = append(tradeDataInChunk, tradeData...)
			}
		}
		if pool.GetType() == pooltypes.PoolTypes.CurveBase ||
			pool.GetType() == pooltypes.PoolTypes.CurveStablePlain ||
			pool.GetType() == pooltypes.PoolTypes.CurvePlainOracle ||
			pool.GetType() == pooltypes.PoolTypes.CurveAave ||
			pool.GetType() == pooltypes.PoolTypes.CurveMeta ||
			pool.GetType() == pooltypes.PoolTypes.CurveStableMetaNg {
			seenBasePools.Add(pool.GetAddress())
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

	countSuccess := 0
	countError := 0
	for _, tr := range tradeDataInChunk {
		tradeDataId := TradeDataId{Type: tr.Type, Pool: tr.Pool}
		if tr.Err != nil {
			if _, ok := hasError[tradeDataId]; !ok {
				hasError[tradeDataId] = &LiquidityScoreCalcInput{
					TradeData: []TradeData{},
					Liquidity: tvl[tr.Pool],
				}
			}
			countError++
			tr.ErrMessage = tr.getError()
			hasError[tradeDataId].AddTradeData(tr)

			// handle special errors can not generate enough trade data and tokens have wrong prices
			if errors.Is(tr.Err, ErrNotEnoughSuccessTradeData) ||
				errors.Is(tr.Err, ErrTokensHaveWrongPrice) ||
				errors.Is(tr.Err, ErrNotFindStartAmount) {
				log.Ctx(ctx).Info().Err(tr.Err).
					Str("struct", "TradeDataGenerator").
					Str("method", "proceedChunk").
					Str("pool", tr.Pool).
					Str("tokenIn", tr.TokenIn).
					Str("tokenOut", tr.TokenOut).
					Msg("add to ZeroScore")
				keys := gen.generateTradeDataKey(tr.TokenIn, tr.TokenOut)
				score := 0.0
				if errors.Is(tr.Err, ErrNotEnoughSuccessTradeData) ||
					errors.Is(tr.Err, ErrNotFindStartAmount) &&
						gen.poolHasLimitCheck(tr.Exchange) {
					score = DEFAULT_RFQ_SCORE
				}
				for _, k := range keys {
					zeroPoolScores = append(zeroPoolScores, routerEntity.PoolScore{
						Key:            k,
						Pool:           tr.Pool,
						TvlInUsd:       tvl[tr.Pool],
						LiquidityScore: score,
					})
				}
			}
			continue
		}
		if _, ok := result[tradeDataId]; !ok {
			result[tradeDataId] = &LiquidityScoreCalcInput{
				TradeData: []TradeData{},
				Liquidity: tvl[tr.Pool],
			}
		}
		result[tradeDataId].AddTradeData(tr)
		countSuccess++
	}

	// Add zero reserves pools and pools which doesn't yield any successful swaps to blacklist
	indexBlacklistTrack = indexBlacklistTrack.Union(gen.filterFailedPools(result, hasError))

	log.Ctx(ctx).Info().
		Str("struct", "TradeDataGenerator").
		Str("method", "proceedChunk").
		Int("chunk", len(chunk)).
		Int("successResult", countSuccess).
		Int("failedResult", countError).
		Int("zeroPoolScores", len(zeroPoolScores)).
		Int("tradeDataCount", tradaDataCount).
		Msg("proceedChunk done")

	return TradesGenerationOutput{
		Successed:      result,
		Failed:         hasError,
		Blacklist:      indexBlacklistTrack,
		ZeroScorePools: zeroPoolScores,
	}, nil
}

// return values: trade data that are generated from calcAmountOut, error and list of error pool address
// TODO: need to take into account below case:
// at 10$ amountOut = X, at 100$ error, but at 1000$ we still have valid amountOut, what should we set data point for 100$ amount in?
func (gen *TradeDataGenerator) generateTradeData(ctx context.Context,
	tokenIn, tokenOut string,
	tokens map[string]*entity.SimplifiedToken,
	prices map[string]*routerEntity.OnchainPrice,
	pool poolpkg.IPoolSimulator,
	limit poolpkg.SwapLimit,
	tradeDataType valueobject.TradeDataType) []TradeData {
	calcAmountOutInstance := routerpoolpkg.NewCustomFuncs(gen.config.DexUseAEVM)
	var result []TradeData
	lastTradeData := TradeData{AmountOutUsd: float64(-1)}
	amountInUsd := 1.0
	var amountIn *big.Int
	var err error
	key := gen.generateHighestTradeDataKey(tokenIn, tokenOut)
	// used for tracking number of success trade data, in case rfq we might not generage enough data points to calculate liquidity score
	successCount := 0

	if prices[tokenIn] == nil || prices[tokenIn].GetSellPriceIfAny() == 0 {
		log.Ctx(ctx).Info().
			Str("struct", "TradeDataGenerator").
			Str("method", "generateTradeData").
			Str("tokenIn", tokenIn).
			Msg("price of token in is nil")
		amountIn, err = gen.findStartAmount(ctx, tokenIn, tokenOut, tokens, prices, pool, limit,
			calcAmountOutInstance.CalcAmountOut)
		if err != nil || amountIn == nil {
			log.Ctx(ctx).Err(err).
				Str("struct", "TradeDataGenerator").
				Str("method", "generateTradeData").
				Str("pool", pool.GetAddress()).
				Str("tokenIn", tokenIn).
				Str("tokenOut", tokenOut).
				Msg("findStartAmount failed")
			return []TradeData{
				{
					Key:      key,
					Type:     tradeDataType,
					TokenIn:  tokenIn,
					TokenOut: tokenOut,
					Err:      err,
					Pool:     pool.GetAddress(),
					Exchange: pool.GetExchange(),
				},
			}
		}
	} else {
		amountIn = business.CalcAmountFromUSD(amountInUsd, tokens[tokenIn].Decimals, prices[tokenIn].GetSellPriceIfAny())
	}

	if gen.poolHasLimitCheck(pool.GetType()) {
		startAmountIn, err := gen.findStartAmountForPoolHasLimitCheck(ctx, tokenIn, tokenOut, amountIn, 0, tokens, prices, pool, limit, calcAmountOutInstance.CalcAmountOut)
		if err != nil || startAmountIn == nil {
			log.Ctx(ctx).Err(err).
				Str("struct", "TradeDataGenerator").
				Str("method", "generateTradeData").
				Msgf("can not find start amount in for pool %v tokenIn %v amountIn %v", pool.GetAddress(), tokenIn,
					amountIn.Text(10))
			return []TradeData{
				{
					Key:      key,
					Type:     tradeDataType,
					Err:      ErrNotFindStartAmount,
					TokenIn:  tokenIn,
					TokenOut: tokenOut,
					Pool:     pool.GetAddress(),
					Exchange: pool.GetExchange(),
				},
			}
		}
		amountIn = startAmountIn
		if prices[tokenIn] != nil && prices[tokenIn].GetSellPriceIfAny() != 0 {
			amountInUsd, _ = business.CalcAmountUSD(amountIn, tokens[tokenIn].Decimals, prices[tokenIn].GetSellPriceIfAny()).Float64()
		}
	}

	for i := 0; i <= gen.minDataPointNumber || (!lastTradeData.hasError() && i <= gen.maxDataPointNumber); i++ {
		// use sell price for token in
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
			amountOutUsdResult = 0.0
			priceDifferencePercentage := 0.0
			if prices[tokenOut] != nil && prices[tokenOut].GetBuyPriceIfAny() != 0 {
				amountOutUsdResult, _ = business.CalcAmountUSD(amountOut.TokenAmountOut.Amount, tokens[tokenOut].Decimals, prices[tokenOut].GetBuyPriceIfAny()).Float64()

				if amountOutUsdResult > MAX_AMOUNT_OUT_USD {
					amountOutUsdResult = 0.0
				} else {
					priceDifferencePercentage = math.Abs(amountInUsd-amountOutUsdResult) / amountInUsd
					if priceDifferencePercentage >= gen.config.InvalidPriceImpactThreshold {
						log.Ctx(ctx).Error().
							Str("struct", "TradeDataGenerator").
							Str("method", "generateTradeData").
							Str("pool", pool.GetAddress()).
							Str("tokenIn", tokenIn).
							Str("tokenOut", tokenOut).
							Stringer("AmountIn", amountIn).
							Float64("AmountInUsd", amountInUsd).
							Any("AmountOut", amountOut).
							Float64("AmountOutUsd", amountOutUsdResult).
							Msg("tokens have incorrect prices")
						return []TradeData{
							{
								Key:      key,
								Type:     tradeDataType,
								Err:      ErrTokensHaveWrongPrice,
								TokenIn:  tokenIn,
								TokenOut: tokenOut,
								Pool:     pool.GetAddress(),
								Exchange: pool.GetExchange(),
							},
						}
					}
					// Handle case where 2 consecutive points have a big diffrentiate prices impact, we have to generate 2 extra points between them
					// example: if At data point 10^1, P1 = 40%; At data point 10^2 P2 = 90%, then delta price impact = 50%, then we have to generate more extra point 20 and 50
					if i > 0 && i < MAX_EXPONENT_GENERATE_EXTRA_POINT && math.Abs(priceDifferencePercentage-lastTradeData.PriceImpact) > PRICE_IMPACT_THRESHOLD {
						extraPoints := gen.generateExtraDataPointsTradeData(ctx, tokenOut, tokens, prices, pool, limit, calcAmountOutInstance.CalcAmountOut, poolpkg.TokenAmount{
							Token:     tokenIn,
							Amount:    amountIn,
							AmountUsd: amountInUsd,
						}, tradeDataType)
						result = append(result, extraPoints...)
					}
				}
			}

			lastTradeData = TradeData{
				Key:          key,
				Type:         tradeDataType,
				TokenIn:      tokenIn,
				TokenOut:     tokenOut,
				PriceImpact:  priceDifferencePercentage,
				AmountInUsd:  amountInUsd,
				AmountOutUsd: amountOutUsdResult,
				Pool:         pool.GetAddress(),
				AmountIn:     amountIn.Text(10),
				Exchange:     pool.GetExchange(),
			}
		}

		if swapErrResult == nil {
			successCount++
		}

		result = append(result, TradeData{
			Key:          key,
			Type:         tradeDataType,
			TokenIn:      tokenIn,
			TokenOut:     tokenOut,
			AmountInUsd:  amountInUsd,
			AmountOutUsd: amountOutUsdResult,
			Pool:         pool.GetAddress(),
			Err:          swapErrResult,
			AmountIn:     amountIn.Text(10),
			Exchange:     pool.GetExchange(),
		})
		amountIn = amountIn.Mul(amountIn, utils.TenPowDecimals[1])
		amountInUsd *= 10
	}

	if successCount < MIN_DATA_POINT_NUMBER_DEFAULT {
		log.Ctx(ctx).Error().
			Str("struct", "TradeDataGenerator").
			Str("method", "generateTradeData").
			Str("pool", pool.GetAddress()).
			Str("tokenIn", tokenIn).
			Str("tokenOut", tokenOut).
			Any("tradeData", result).
			Msg("not generage enough trade data")
		return []TradeData{
			{
				Key:      key,
				Type:     tradeDataType,
				Err:      ErrNotEnoughSuccessTradeData,
				TokenIn:  tokenIn,
				TokenOut: tokenOut,
				Pool:     pool.GetAddress(),
				Exchange: pool.GetExchange(),
			},
		}
	}

	return result

}

func (gen *TradeDataGenerator) findStartAmount(
	ctx context.Context,
	nonPriceToken, knownPriceToken string,
	tokens map[string]*entity.SimplifiedToken,
	prices map[string]*routerEntity.OnchainPrice,
	pool poolpkg.IPoolSimulator,
	limit poolpkg.SwapLimit,
	calcAmountFunc func(ctx context.Context, pool poolpkg.IPoolSimulator, tokenAmountIn poolpkg.TokenAmount, tokenOut string, limit poolpkg.SwapLimit) (*poolpkg.CalcAmountOutResult, error)) (*big.Int, error) {

	// start from 1$, if we can not swap 1$ from wl to token at this pool, considered the pool is invalid and ignore
	amountInUsd := 1.0
	knownPriceAmountIn := business.CalcAmountFromUSD(amountInUsd, tokens[knownPriceToken].Decimals, prices[knownPriceToken].GetSellPriceIfAny())
	amountOut, err := calcAmountFunc(ctx, pool, poolpkg.TokenAmount{
		Token:     knownPriceToken,
		Amount:    knownPriceAmountIn,
		AmountUsd: amountInUsd,
	}, nonPriceToken, limit)
	if err != nil {
		return nil, err
	}
	if amountOut == nil || !amountOut.IsValid() {
		return nil, errors.New("can not find start amount amountOut is invalid")
	}

	return amountOut.TokenAmountOut.Amount, nil
}

func (gen *TradeDataGenerator) findStartAmountForPoolHasLimitCheck(
	ctx context.Context,
	tokenIn, tokenOut string,
	amountIn *big.Int,
	count int,
	tokens map[string]*entity.SimplifiedToken,
	prices map[string]*routerEntity.OnchainPrice,
	pool poolpkg.IPoolSimulator,
	limit poolpkg.SwapLimit,
	calcAmountFunc func(ctx context.Context, pool poolpkg.IPoolSimulator, tokenAmountIn poolpkg.TokenAmount, tokenOut string, limit poolpkg.SwapLimit) (*poolpkg.CalcAmountOutResult, error)) (*big.Int, error) {
	if count > gen.minDataPointNumber {
		return nil, errors.Errorf("can not find start amount in for rfq dex %v count %d amountIn %s", pool.GetAddress(), count, amountIn.Text(10))
	}

	tokenAmountIn := new(big.Int).Mul(amountIn, utils.TenPowDecimals[count])
	// tokenAmountInUsd := business.CalcAmountUSD(tokenAmountIn, tokens[tokenIn].Decimals, prices[tokenIn].GetSellPriceIfAny())
	amountOut, err := calcAmountFunc(ctx, pool, poolpkg.TokenAmount{
		Token:  tokenIn,
		Amount: tokenAmountIn,
	}, tokenOut, limit)
	if gen.errAmountInLessThanMinAllowed(dexlibValueObject.Exchange(pool.GetExchange()), err) {
		return gen.findStartAmountForPoolHasLimitCheck(ctx, tokenIn, tokenOut, amountIn, count+1, tokens, prices, pool, limit, calcAmountFunc)
	}

	if amountOut != nil && amountOut.IsValid() {
		return tokenAmountIn, nil
	}

	return nil, err
}

func (gen *TradeDataGenerator) generateTradeDataKey(tokenIn, tokenOut string) []string {
	isTokenInWhitelist := gen.config.WhitelistedTokenSet[strings.ToLower(tokenIn)]
	isTokenOutWhitelist := gen.config.WhitelistedTokenSet[strings.ToLower(tokenOut)]
	result := make([]string, 0, 4)

	if isTokenInWhitelist && isTokenOutWhitelist {
		result = append(result, gen.keyGenerator.WhitelistToWhitelistPairKey(poolrank.SortByLiquidityScoreTvl))
	}

	if isTokenInWhitelist {
		result = append(result, gen.keyGenerator.WhitelistToTokenPairKey(poolrank.SortByLiquidityScoreTvl, tokenOut))
	}

	if isTokenOutWhitelist {
		result = append(result, gen.keyGenerator.TokenToWhitelistPairKey(poolrank.SortByLiquidityScoreTvl, tokenIn))
	}

	result = append(result, gen.keyGenerator.DirectPairKeyWithoutSort(poolrank.SortByLiquidityScoreTvl, tokenIn, tokenOut))

	return result

}

func (gen *TradeDataGenerator) generateHighestTradeDataKey(tokenIn, tokenOut string) string {
	keys := gen.generateTradeDataKey(tokenIn, tokenOut)

	return keys[0]
}

func (gen *TradeDataGenerator) generateExtraDataPointsTradeData(ctx context.Context,
	tokenOut string,
	tokens map[string]*entity.SimplifiedToken,
	prices map[string]*routerEntity.OnchainPrice,
	pool poolpkg.IPoolSimulator,
	limit poolpkg.SwapLimit,
	calcAmountFunc func(ctx context.Context, pool poolpkg.IPoolSimulator, tokenAmountIn poolpkg.TokenAmount, tokenOut string, limit poolpkg.SwapLimit) (*poolpkg.CalcAmountOutResult, error),
	tokenIn poolpkg.TokenAmount,
	tradeDataType valueobject.TradeDataType) []TradeData {
	amountInList := make([]*poolpkg.TokenAmount, 0, 2)
	amountInList = append(amountInList, &poolpkg.TokenAmount{
		Amount:    new(big.Int).Div(tokenIn.Amount, big.NewInt(5)),
		AmountUsd: tokenIn.AmountUsd / 5,
		Token:     tokenIn.Token,
	})
	amountInList = append(amountInList, &poolpkg.TokenAmount{
		Amount:    new(big.Int).Div(tokenIn.Amount, big.NewInt(2)),
		AmountUsd: tokenIn.AmountUsd / 2,
		Token:     tokenIn.Token,
	})
	result := make([]TradeData, 0, 2)

	for _, amountIn := range amountInList {
		amountOut, err := calcAmountFunc(ctx, pool, poolpkg.TokenAmount{
			Token:     tokenIn.Token,
			Amount:    amountIn.Amount,
			AmountUsd: amountIn.AmountUsd,
		}, tokenOut, limit)

		if err != nil || amountOut == nil || !amountOut.IsValid() {
			if gen.config.LogError {
				log.Ctx(ctx).Err(err).
					Str("struct", "TradeDataGenerator").
					Str("method", "generateExtraDataPointsTradeData").
					Str("poolAddress", pool.GetAddress()).
					Stringer("amount", amountIn.Amount).
					Float64("amountUsd", amountIn.AmountUsd).
					Msg("calculate amount out failed")
			}
			continue
		}

		amountOutUsdResult, _ := business.CalcAmountUSD(amountOut.TokenAmountOut.Amount, tokens[tokenOut].Decimals, prices[tokenOut].GetBuyPriceIfAny()).Float64()

		result = append(result, TradeData{
			Key:          gen.generateHighestTradeDataKey(amountIn.Token, tokenOut),
			Type:         tradeDataType,
			TokenIn:      amountIn.Token,
			TokenOut:     tokenOut,
			AmountInUsd:  amountIn.AmountUsd,
			AmountOutUsd: amountOutUsdResult,
			Pool:         pool.GetAddress(),
			AmountIn:     amountIn.Amount.Text(10),
			Exchange:     pool.GetExchange(),
		})
	}

	return result

}

func (gen *TradeDataGenerator) saveZeroScore(ctx context.Context, zeroScorePools []routerEntity.PoolScore) {
	// update zero liquidity score
	if len(zeroScorePools) != 0 {
		err := gen.poolRankRepo.AddScoreToSortedSets(ctx, zeroScorePools)
		if err != nil {
			log.Ctx(ctx).Err(err).
				Str("struct", "TradeDataGenerator").
				Str("method", "saveZeroScore").
				Msg("update zero pool score failed")
		}
		// for debug supporting only
		if gen.config.ExportZeroScores {
			zeroScoresFile, err := os.OpenFile("zero_scores.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Ctx(ctx).Err(err).
					Str("struct", "TradeDataGenerator").
					Str("method", "writeTradeData").
					Msg("init failed buffer failed")
			} else {
				defer func() { _ = zeroScoresFile.Close() }()
				zeroScoresBuffer := bufio.NewWriter(zeroScoresFile)
				for _, score := range zeroScorePools {
					jsonScore, _ := json.Marshal(score)
					_, _ = zeroScoresBuffer.WriteString(string(jsonScore))
					_, _ = zeroScoresBuffer.WriteString("\n")
				}
				_ = zeroScoresBuffer.Flush()
			}

		}
	}
}
