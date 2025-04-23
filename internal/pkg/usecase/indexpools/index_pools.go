package indexpools

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/iter"
	"golang.org/x/exp/maps"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/poolrank"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/mempool"
)

type IndexPoolsUseCase struct {
	poolRepo     IPoolRepository
	poolRankRepo IPoolRankRepository

	onchainPriceRepo IOnchainPriceRepository

	config IndexPoolsConfig

	mu sync.RWMutex
}

type IndexResult int

const (
	INDEX_RESULT_SUCCESS IndexResult = iota
	INDEX_RESULT_FAIL
	INDEX_RESULT_SKIP_OLD
)

var (
	ErrIndexResultFailed = errors.New("index result failed")
)

type IndexProcessingHandler func(ctx context.Context, poolIndex *PoolIndex) error

type PoolIndex struct {
	Pool                *entity.Pool
	Token0              string
	Token1              string
	IsToken0Whitelisted bool
	IsToken1Whitelisted bool
	TvlNative           float64
	AmplifiedTvlNative  float64
}

func NewPoolIndex(pool *entity.Pool, tokenI string, tokenJ string, whitelist map[string]bool, tvlNative float64, amplifiedTvlNative float64) *PoolIndex {
	return &PoolIndex{
		Pool:                pool,
		Token0:              tokenI,
		Token1:              tokenJ,
		IsToken0Whitelisted: whitelist[tokenI],
		IsToken1Whitelisted: whitelist[tokenJ],
		TvlNative:           tvlNative,
		AmplifiedTvlNative:  amplifiedTvlNative,
	}
}

func NewIndexPoolsUseCase(
	poolRepo IPoolRepository,
	poolRankRepo IPoolRankRepository,
	onchainPriceRepo IOnchainPriceRepository,
	config IndexPoolsConfig,
) *IndexPoolsUseCase {
	return &IndexPoolsUseCase{
		poolRepo:         poolRepo,
		poolRankRepo:     poolRankRepo,
		onchainPriceRepo: onchainPriceRepo,
		config:           config,
	}
}

func (u *IndexPoolsUseCase) ApplyConfig(config IndexPoolsConfig) {
	u.mu.Lock()
	u.config = config
	u.mu.Unlock()
}

func (u *IndexPoolsUseCase) Handle(ctx context.Context, command dto.IndexPoolsCommand) *dto.IndexPoolsResult {
	var (
		totalCount, oldPoolCount int
		totalFailedPoolAddresses []string
	)

	if command.UsePoolAddresses {
		totalCount = len(command.PoolAddresses)
	} else {
		totalCount = int(u.poolRepo.Count(ctx))
	}

	// process chunk by chunk
	getChunkPoolCommand := dto.NewGetChunkPoolCommand(
		u.config.ChunkSize, command.UsePoolAddresses, command.PoolAddresses,
	)

	for {
		if getChunkPoolCommand.IsLastCommand {
			break
		}

		chunkPool, failedPoolAddresses, err := u.getChunkPool(ctx, &getChunkPoolCommand)
		if err != nil {
			logger.Errorf(ctx, "error get chunk pool: %v", err)
			totalFailedPoolAddresses = append(totalFailedPoolAddresses, failedPoolAddresses...)
			continue
		}

		// filter out pools that haven't been updated recently
		pools := lo.Filter(chunkPool, func(pool *entity.Pool, _ int) bool {
			return command.IgnorePoolsBeforeTimestamp <= 0 || (*pool).Timestamp >= command.IgnorePoolsBeforeTimestamp
		})
		oldPoolCount += len(chunkPool) - len(pools)

		var nativePriceByToken map[string]*routerEntity.OnchainPrice
		// collect prices for all pools' tokens first
		nativePriceByToken, err = u.getPricesForAllTokens(ctx, pools)
		if err != nil {
			logger.Errorf(ctx, "error fetching pool tokens prices %v", err)
			totalFailedPoolAddresses = append(totalFailedPoolAddresses,
				lo.Map(pools, func(pool *entity.Pool, _ int) string { return pool.Address })...)
			continue
		}

		// if `u.config.NumParallel==0` (default) then will use GOMAXPROCS
		// should be set to higher since indexing wait for IO a lot
		mapper := iter.Mapper[*entity.Pool, IndexResult]{MaxGoroutines: u.config.MaxGoroutines}

		indexPoolsResults := mapper.Map(pools, func(pool **entity.Pool) IndexResult {
			err := u.processIndex(ctx, *pool, nativePriceByToken, u.savePoolIndex)
			if err != nil && strings.Contains(err.Error(), ErrIndexResultFailed.Error()) {
				return INDEX_RESULT_FAIL
			}

			return INDEX_RESULT_SUCCESS
		})

		for i, p := range pools {
			if indexPoolsResults[i] == INDEX_RESULT_FAIL {
				totalFailedPoolAddresses = append(totalFailedPoolAddresses, p.Address)
			}
		}
		mempool.ReserveMany(chunkPool...)
	}

	return dto.NewIndexPoolsResult(totalCount, totalFailedPoolAddresses, oldPoolCount)
}

func (u *IndexPoolsUseCase) getChunkPool(
	ctx context.Context,
	command *dto.GetChunkPoolCommand,
) ([]*entity.Pool, []string, error) {
	if command.IsLastCommand {
		return nil, nil, nil
	}

	var chunkPool []*entity.Pool
	var failedPoolAddresses []string
	var err error

	if command.UsePoolAddresses {
		startIndex := command.AddressChunkIndex * command.ChunkSize
		lastIndex := (command.AddressChunkIndex + 1) * command.ChunkSize
		if lastIndex > len(command.PoolAddresses) {
			lastIndex = len(command.PoolAddresses)
		}
		chunkPool, err = u.poolRepo.FindByAddresses(ctx, command.PoolAddresses[startIndex:lastIndex])
		if err != nil {
			logger.Errorf(ctx, "error get pools by addresses: %v", err)
			failedPoolAddresses = command.PoolAddresses[startIndex:lastIndex]
		}

		command.AddressChunkIndex += 1
		if lastIndex >= len(command.PoolAddresses) {
			command.IsLastCommand = true
		}
	} else {
		var newCursor uint64
		chunkPool, failedPoolAddresses, newCursor, err = u.poolRepo.ScanPools(ctx, command.Cursor, command.ChunkSize)
		if err != nil {
			logger.Errorf(ctx, "error get all pools: %v", err)
		}

		command.Cursor = newCursor
		if command.Cursor == 0 {
			command.IsLastCommand = true
		}
	}

	return chunkPool, failedPoolAddresses, err
}

func (u *IndexPoolsUseCase) processIndex(ctx context.Context, pool *entity.Pool, nativePriceByToken map[string]*routerEntity.OnchainPrice, handler IndexProcessingHandler) error {
	if !pool.HasReserves() && !pool.HasAmplifiedTvl() {
		return nil
	}

	var (
		tvlNative, amplifiedTvlNative float64
	)

	if nativePriceByToken != nil {
		var err error
		tvlNative, err = business.CalculatePoolTVL(ctx, pool, nativePriceByToken)
		if err != nil {
			// just reset score here without returning error
			tvlNative = 0
		}

		var useTvl bool
		amplifiedTvlNative, useTvl, err = business.CalculatePoolAmplifiedTVL(ctx, pool, nativePriceByToken)
		if err != nil {
			// just reset score here without returning error
			amplifiedTvlNative = 0
		} else if useTvl {
			amplifiedTvlNative = tvlNative
		}
	}

	var result error

	// Index tokens in the main pool
	if err := u.processMainPoolIndexes(ctx, pool, tvlNative, amplifiedTvlNative, handler); err != nil {
		result = err
	}

	// Index tokens in nested/base pools depending on the pool type
	switch pool.Type {
	case pooltypes.PoolTypes.CurveAave:
		if err := u.processCurveAave(ctx, pool, tvlNative, amplifiedTvlNative, handler); err != nil && result == nil {
			result = err
		}
	case pooltypes.PoolTypes.CurveMeta, pooltypes.PoolTypes.CurveStableMetaNg:
		if err := u.processCurveMeta(ctx, pool, tvlNative, amplifiedTvlNative, handler); err != nil && result == nil {
			result = err
		}
	case pooltypes.PoolTypes.BalancerV2Stable, pooltypes.PoolTypes.BalancerV2Weighted:
		if err := u.processBalancerV2(ctx, pool, tvlNative, amplifiedTvlNative, handler); err != nil && result == nil {
			result = err
		}
	}

	return result
}

func (u *IndexPoolsUseCase) processMainPoolIndexes(ctx context.Context, pool *entity.Pool, tvl, amplifiedTvl float64, handler IndexProcessingHandler) error {
	var result error
	for i, tokenI := range pool.Tokens {
		if !tokenI.Swappable || len(pool.Reserves)-1 < i {
			continue
		}
		for j := i + 1; j < len(pool.Tokens); j++ {
			tokenJ := pool.Tokens[j]
			if !tokenJ.Swappable || len(pool.Reserves)-1 < j {
				continue
			}

			if pool.HasReserve(pool.Reserves[i]) || pool.HasReserve(pool.Reserves[j]) {
				if err := handler(ctx, NewPoolIndex(pool, tokenI.Address, tokenJ.Address, u.config.WhitelistedTokenSet, tvl, amplifiedTvl)); err != nil {
					result = err
				}
			}
		}
	}

	return result
}

func (u *IndexPoolsUseCase) processCurveAave(ctx context.Context, pool *entity.Pool, tvl, amplifiedTvl float64, handler IndexProcessingHandler) error {
	var extra struct {
		UnderlyingTokens []string `json:"underlyingTokens"`
	}

	if err := json.Unmarshal([]byte(pool.StaticExtra), &extra); err != nil {
		return nil
	}

	var result error
	for i := 0; i < len(extra.UnderlyingTokens); i++ {
		for j := i + 1; j < len(extra.UnderlyingTokens); j++ {
			if len(pool.Reserves)-1 < j {
				continue
			}
			tokenI := extra.UnderlyingTokens[i]
			tokenJ := extra.UnderlyingTokens[j]
			if pool.HasReserve(pool.Reserves[i]) || pool.HasReserve(pool.Reserves[j]) {
				if err := handler(ctx, NewPoolIndex(pool, tokenI, tokenJ, u.config.WhitelistedTokenSet, tvl, amplifiedTvl)); err != nil {
					result = err
				}
			}
		}
	}

	return result
}

func (u *IndexPoolsUseCase) processCurveMeta(ctx context.Context, pool *entity.Pool, tvl, amplifiedTvl float64, handler IndexProcessingHandler) error {
	// `underlyingTokens` here are metaCoin[0:numMetaCoin-1] ++ baseCoin[0:numBaseCoin]
	// we'll index for each pair of metaCoin and baseCoin
	// note that technically we can use this pool to swap between base coins, but we shouldn't, because:
	// - it cost less gas to swap base coins directly at base pool instead
	// - router might return incorrect output, because:
	//		- router find 2 paths, one through base pool and one through meta pool
	//		- router consume the 1st path and update balance for base pool accordingly
	//		- but that won't affect meta pool, because we're storing them separatedly in pool bucket
	//		- so router will incorrectly think that the 2nd path is still usable, while it's not
	// 	so rejecting base coin swap here will help us avoid that (note that we might still get another edge case:
	//		path1: basecoin1 -> basepool -> basecoin2
	// 		path2: basecoin1 -> metapool -> metacoin1 -> anotherpool -> basecoin2
	//		after consuming path1, router still assuming that path2 is usable while it's not
	//		to fix this we'll need to change the way we update balance for base & meta pool, which is much more complicated
	// 	)

	var extra struct {
		UnderlyingTokens []string `json:"underlyingTokens"`
	}

	if err := json.Unmarshal([]byte(pool.StaticExtra), &extra); err != nil {
		return nil
	}

	var result error

	numUsableMetaCoin := len(pool.Tokens) - 1
	numUnderlyingCoins := len(extra.UnderlyingTokens)
	if numUnderlyingCoins <= numUsableMetaCoin {
		return nil
	}

	for i := 0; i < numUsableMetaCoin; i++ {
		if !pool.HasReserve(pool.Reserves[i]) {
			continue
		}
		tokenI := pool.Tokens[i].Address
		for j := numUsableMetaCoin; j < numUnderlyingCoins; j++ {
			tokenJ := extra.UnderlyingTokens[j]
			if err := handler(ctx, NewPoolIndex(pool, tokenI, tokenJ, u.config.WhitelistedTokenSet, tvl, amplifiedTvl)); err != nil {
				result = err
			}
		}
	}

	return result
}

func (u *IndexPoolsUseCase) processBalancerV2(ctx context.Context, pool *entity.Pool, tvl, amplifiedTvl float64, handler IndexProcessingHandler) error {
	var extra struct {
		BasePools map[string][]string `json:"basePools"`
	}

	if err := json.Unmarshal([]byte(pool.StaticExtra), &extra); err != nil {
		return nil
	}

	var result error

	basePoolTokens := map[string]struct{}{}

	// step 1: Collect all tokens from base pools, excluding the base pool's own address
	for basePoolAddr, tokens := range extra.BasePools {
		for _, token := range tokens {
			if token != basePoolAddr {
				basePoolTokens[token] = struct{}{}
			}
		}
	}

	// step 2: Create indexes from the main pool tokens to all base pool tokens
	for i := range pool.Tokens {
		if !pool.HasReserve(pool.Reserves[i]) {
			continue
		}
		tokenI := pool.Tokens[i].Address
		for tokenJ := range basePoolTokens {
			if err := handler(ctx, NewPoolIndex(pool, tokenI, tokenJ, u.config.WhitelistedTokenSet, tvl, amplifiedTvl)); err != nil {
				result = err
			}
		}
	}

	// step 3: Create indexes between base pool tokens
	basePoolAddresses := lo.Keys(extra.BasePools)
	for i := 0; i < len(basePoolAddresses); i++ {
		for j := i + 1; j < len(basePoolAddresses); j++ {
			basePoolA := basePoolAddresses[i]
			basePoolB := basePoolAddresses[j]
			for _, tokenA := range extra.BasePools[basePoolA] {
				if tokenA == basePoolA {
					continue
				}
				for _, tokenB := range extra.BasePools[basePoolB] {
					// ensure we don't create an index directly between base pool addresses
					if tokenB == basePoolB || tokenA == basePoolB || tokenB == basePoolA {
						continue
					}

					if err := handler(ctx, NewPoolIndex(pool, tokenA, tokenB, u.config.WhitelistedTokenSet, tvl, amplifiedTvl)); err != nil {
						result = err
					}
				}
			}
		}
	}

	return result
}

type priceAndError struct {
	prices map[string]*routerEntity.OnchainPrice
	err    error
}

func (u *IndexPoolsUseCase) getPricesForAllTokens(ctx context.Context, pools []*entity.Pool) (map[string]*routerEntity.OnchainPrice, error) {
	// collect all tokens
	tokens := make(map[string]struct{}, len(pools)*3)
	for _, p := range pools {
		if !p.HasReserves() {
			continue
		}
		for _, t := range p.Tokens {
			tokens[t.Address] = struct{}{}
		}
	}

	// then get price for each chunks in parallel
	prices := make(map[string]*routerEntity.OnchainPrice, len(tokens))
	chunks := lo.Chunk(maps.Keys(tokens), 100)

	mapper := iter.Mapper[[]string, priceAndError]{MaxGoroutines: u.config.MaxGoroutines}
	chunkResults := mapper.Map(chunks, func(chunk *[]string) priceAndError {
		chunkPrices, err := u.onchainPriceRepo.FindByAddresses(ctx, *chunk)
		if err != nil {
			return priceAndError{nil, err}
		}
		return priceAndError{chunkPrices, nil}
	})

	for _, res := range chunkResults {
		if res.err != nil {
			return nil, res.err
		}
		for token, price := range res.prices {
			prices[token] = price
		}
	}

	return prices, nil
}

func (u *IndexPoolsUseCase) savePoolIndex(ctx context.Context, poolIndex *PoolIndex) error {
	var shouldAddToTvlNativeIndex bool
	if poolIndex.TvlNative > 0 {
		shouldAddToTvlNativeIndex = true
	} else {
		directIndexLength, err := u.poolRankRepo.GetDirectIndexLength(ctx, poolrank.SortByTVLNative, poolIndex.Token0, poolIndex.Token1)
		if err != nil {
			logger.Warnf(ctx, "failed to get direct index length %v", err)
		} else {
			shouldAddToTvlNativeIndex = directIndexLength < int64(u.config.MaxDirectIndexLenForZeroTvl)
		}
	}

	if shouldAddToTvlNativeIndex {
		if err := u.poolRankRepo.AddToSortedSet(ctx, poolIndex.Token0, poolIndex.Token1, poolIndex.IsToken0Whitelisted, poolIndex.IsToken1Whitelisted,
			poolrank.SortByTVLNative, poolIndex.Pool.Address, poolIndex.TvlNative, true); err != nil {
			logger.Errorf(ctx, "failed to add to sorted set %v", err)
			return ErrIndexResultFailed
		}
	}

	if poolIndex.AmplifiedTvlNative > 0 {
		if err := u.poolRankRepo.AddToSortedSet(ctx, poolIndex.Token0, poolIndex.Token1, poolIndex.IsToken0Whitelisted, poolIndex.IsToken1Whitelisted,
			poolrank.SortByAmplifiedTVLNative, poolIndex.Pool.Address, poolIndex.AmplifiedTvlNative, false); err != nil {
			logger.Debugf(ctx, "failed to add to sorted set %v", err)
			return ErrIndexResultFailed
		}
	}

	return nil
}

func (u *IndexPoolsUseCase) removePoolIndex(ctx context.Context, poolIndex *PoolIndex) error {
	var result error

	if err := u.poolRankRepo.RemoveFromSortedSet(ctx, poolIndex.Token0, poolIndex.Token1, poolIndex.IsToken0Whitelisted, poolIndex.IsToken1Whitelisted,
		poolrank.SortByTVLNative, poolIndex.Pool.Address, true); err != nil {
		logger.Errorf(ctx, "removePoolIndex SortByTVLNative %v", err)
		result = err
	}

	if err := u.poolRankRepo.RemoveFromSortedSet(ctx, poolIndex.Token0, poolIndex.Token1, poolIndex.IsToken0Whitelisted, poolIndex.IsToken1Whitelisted,
		poolrank.SortByAmplifiedTVLNative, poolIndex.Pool.Address, false); err != nil {
		logger.Errorf(ctx, "removePoolIndex SortByAmplifiedTVLNative %v", err)
		result = err
	}

	return result
}

func (u *IndexPoolsUseCase) RemovePoolFromIndexes(ctx context.Context, pool *entity.Pool) error {
	return u.processIndex(ctx, pool, nil, u.removePoolIndex)

}
