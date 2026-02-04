package ekubov3

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	maxBatchSize                        = 100
	minTickSpacingsPerPool       uint32 = 2
	quoteDataFetcherMethod              = "getQuoteData"
	twammDataFetcherMethod              = "getPoolState"
	boostedFeesDataFetcherMethod        = "getPoolState"
)

type (
	quoteData struct {
		Tick           int32           `json:"tick"`
		SqrtRatioFloat *big.Int        `json:"sqrtRatio"`
		Liquidity      *big.Int        `json:"liquidity"`
		MinTick        int32           `json:"minTick"`
		MaxTick        int32           `json:"maxTick"`
		Ticks          []pools.TickRPC `json:"ticks"`
	}

	timeSaleRateInfo struct {
		Time           uint64   `json:"time"`
		SaleRateDelta0 *big.Int `json:"saleRateDelta0"`
		SaleRateDelta1 *big.Int `json:"saleRateDelta1"`
	}

	boostedTimeDonateRateInfo struct {
		Time             uint64   `json:"time"`
		DonateRateDelta0 *big.Int `json:"donateRateDelta0"`
		DonateRateDelta1 *big.Int `json:"donateRateDelta1"`
	}

	twammQuoteData struct {
		SqrtRatio                     *big.Int           `json:"sqrtRatio"`
		Tick                          int32              `json:"tick"`
		Liquidity                     *big.Int           `json:"liquidity"`
		LastVirtualOrderExecutionTime uint64             `json:"lastVirtualOrderExecutionTime"`
		SaleRateToken0                *big.Int           `json:"saleRateToken0"`
		SaleRateToken1                *big.Int           `json:"saleRateToken1"`
		SaleRateDeltas                []timeSaleRateInfo `json:"saleRateDeltas"`
	}

	boostedFeesQuoteData struct {
		SqrtRatio        *big.Int                    `json:"sqrtRatio"`
		Tick             int32                       `json:"tick"`
		Liquidity        *big.Int                    `json:"liquidity"`
		LastDonateTime   uint64                      `json:"lastDonateTime"`
		DonateRateToken0 *big.Int                    `json:"donateRateToken0"`
		DonateRateToken1 *big.Int                    `json:"donateRateToken1"`
		DonateRateDeltas []boostedTimeDonateRateInfo `json:"donateRateDeltas"`
	}

	dataFetchers struct {
		ethrpcClient *ethrpc.Client
		config       *Config
	}

	fetchedPool struct {
		PoolWithBlockNumber
		key pools.AnyPoolKey
	}
)

func (f *dataFetchers) fetchPools(
	ctx context.Context,
	poolKeys []pools.AnyPoolKey,
	overrides map[common.Address]gethclient.OverrideAccount,
) ([]fetchedPool, error) {
	if len(poolKeys) == 0 {
		return nil, nil
	}

	type poolKeyWithExtType struct {
		key     pools.AnyPoolKey
		extType ExtensionType
	}

	twammPoolKeys, boostedFeeConcentratedPoolKeys, basicPoolKeys := make([]pools.AnyPoolKey, 0), make([]pools.AnyPoolKey, 0), make([]poolKeyWithExtType, 0)

	for _, key := range poolKeys {
		extType := f.config.ExtensionType(key.Extension())

		switch extType {
		case ExtensionTypeNoSwapCallPoints, ExtensionTypeMevCapture, ExtensionTypeOracle:
			basicPoolKeys = append(basicPoolKeys, poolKeyWithExtType{
				key,
				extType,
			})
		case ExtensionTypeTwamm:
			twammPoolKeys = append(twammPoolKeys, key)
		case ExtensionTypeBoostedFeesConcentrated:
			boostedFeeConcentratedPoolKeys = append(boostedFeeConcentratedPoolKeys, key)
		default:
			logger.Errorf("unknown extension %v", key.Extension())
		}
	}

	fetchedPools := make([]fetchedPool, 0, len(poolKeys))

	for _, poolKeyBatch := range lo.Chunk(basicPoolKeys, maxBatchSize) {
		req := f.ethrpcClient.R().SetContext(ctx)
		if overrides != nil {
			req.SetOverrides(overrides)
		}

		batchQuoteData := make([]quoteData, len(poolKeyBatch))
		resp, err := req.AddCall(&ethrpc.Call{
			ABI:    abis.QuoteDataFetcherABI,
			Target: f.config.QuoteDataFetcher,
			Method: quoteDataFetcherMethod,
			Params: []any{
				lo.Map(poolKeyBatch, func(keyWithExtType poolKeyWithExtType, _ int) pools.AbiPoolKey {
					return keyWithExtType.key.ToAbi()
				}),
				minTickSpacingsPerPool,
			},
		}, []any{&batchQuoteData}).Aggregate()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve quote data from QuoteDataFetcher: %w", err)
		}

		var blockNumber uint64
		if resp.BlockNumber != nil {
			blockNumber = resp.BlockNumber.Uint64()
		}

		for _, tuple := range lo.Zip2(batchQuoteData, poolKeyBatch) {
			data := &tuple.A
			poolKeyWithExtType := tuple.B
			poolKey, extType := poolKeyWithExtType.key, poolKeyWithExtType.extType
			config := poolKey.Config.TypeConfig

			var pool pools.Pool
			switch extType {
			case ExtensionTypeNoSwapCallPoints:
				switch config := config.(type) {
				case pools.FullRangePoolTypeConfig:
					pool = pools.NewFullRangePool(poolKey.ToFullRange(config), newFullRangePoolState(data))
				case pools.StableswapPoolTypeConfig:
					pool = pools.NewStableswapPool(poolKey.ToStableswap(config), newStableswapPoolState(data))
				case pools.ConcentratedPoolTypeConfig:
					pool = pools.NewBasePool(poolKey.ToConcentrated(config), newBasePoolState(data))
				default:
					logger.Errorf("unexpected pool type config %T", config)
					continue
				}
			case ExtensionTypeOracle:
				if config, ok := config.(pools.FullRangePoolTypeConfig); ok {
					pool = pools.NewOraclePool(poolKey.ToFullRange(config), newOraclePoolState(data))
				} else {
					logger.Errorf("expected full range pool type config for Oracle pool, received %T", config)
					continue
				}
			case ExtensionTypeMevCapture:
				if config, ok := config.(pools.ConcentratedPoolTypeConfig); ok {
					pool = pools.NewMevCapturePool(poolKey.ToConcentrated(config), newBasePoolState(data))
				} else {
					logger.Errorf("expected concentrated pool type config for MEVCapture pool, received %T", config)
					continue
				}
			default:
				logger.Errorf("received unknown extension type %v for basic pool key", extType)
				continue
			}

			fetchedPools = append(fetchedPools, fetchedPool{
				PoolWithBlockNumber{
					Pool:        pool,
					blockNumber: blockNumber,
				},
				poolKey,
			})
		}
	}

	for _, poolKeyBatch := range lo.Chunk(twammPoolKeys, maxBatchSize) {
		req := f.ethrpcClient.R().SetContext(ctx)
		if overrides != nil {
			req.SetOverrides(overrides)
		}

		batchQuoteData := make([]struct{ twammQuoteData }, len(poolKeyBatch))
		for i, poolKey := range poolKeyBatch {
			req.AddCall(&ethrpc.Call{
				ABI:    abis.TwammDataFetcherABI,
				Target: f.config.TwammDataFetcher,
				Method: twammDataFetcherMethod,
				Params: []any{
					poolKey.ToAbi(),
				},
			}, []any{&batchQuoteData[i]})
		}
		resp, err := req.Aggregate()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve quote data from TWAMMDataFetcher: %w", err)
		}

		var blockNumber uint64
		if resp.BlockNumber != nil {
			blockNumber = resp.BlockNumber.Uint64()
		}

		for _, tuple := range lo.Zip2(batchQuoteData, poolKeyBatch) {
			poolKey := tuple.B
			config := poolKey.Config.TypeConfig
			if config, ok := config.(pools.FullRangePoolTypeConfig); ok {
				fetchedPools = append(fetchedPools, fetchedPool{
					PoolWithBlockNumber{
						Pool:        pools.NewTwammPool(poolKey.ToFullRange(config), newTwammPoolState(&tuple.A.twammQuoteData)),
						blockNumber: blockNumber,
					},
					poolKey,
				})
			} else {
				logger.Errorf("expected full range pool type config for TWAMM pool, received %T", config)
			}
		}
	}

	for _, poolKeyBatch := range lo.Chunk(boostedFeeConcentratedPoolKeys, maxBatchSize) {
		req := f.ethrpcClient.R().SetContext(ctx)
		if overrides != nil {
			req.SetOverrides(overrides)
		}

		batchSize := len(poolKeyBatch)
		batchQuoteData := make([]struct{ quoteData }, batchSize)
		batchBoostedFeesData := make([]struct{ boostedFeesQuoteData }, batchSize)

		for i, poolKey := range poolKeyBatch {
			abiPoolKey := poolKey.ToAbi()

			req.AddCall(&ethrpc.Call{
				ABI:    abis.QuoteDataFetcherABI,
				Target: f.config.QuoteDataFetcher,
				Method: quoteDataFetcherMethod,
				Params: []any{
					abiPoolKey,
				},
			}, []any{&batchQuoteData[i]})

			req.AddCall(&ethrpc.Call{
				ABI:    abis.BoostedFeesDataFetcherABI,
				Target: f.config.BoostedFeesDataFetcher,
				Method: boostedFeesDataFetcherMethod,
				Params: []any{
					abiPoolKey,
				},
			}, []any{&batchBoostedFeesData[i]})
		}
		resp, err := req.Aggregate()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve quote data for concentrated BoostedFees pools: %w", err)
		}

		var blockNumber uint64
		if resp.BlockNumber != nil {
			blockNumber = resp.BlockNumber.Uint64()
		}

		for _, tuple := range lo.Zip3(poolKeyBatch, batchQuoteData, batchBoostedFeesData) {
			poolKey := tuple.A
			config := poolKey.Config.TypeConfig

			if config, ok := config.(pools.ConcentratedPoolTypeConfig); ok {
				fetchedPools = append(fetchedPools, fetchedPool{
					PoolWithBlockNumber{
						Pool:        pools.NewBoostedFeesPool(poolKey.ToConcentrated(config), newBoostedFeesPoolState(&tuple.B.quoteData, &tuple.C.boostedFeesQuoteData)),
						blockNumber: blockNumber,
					},
					poolKey,
				})
			} else {
				logger.Errorf("expected concentrated pool type config for BoostedFees pool, received %T", config)
			}
		}
	}

	return fetchedPools, nil
}

func NewDataFetchers(ethrpcClient *ethrpc.Client, config *Config) *dataFetchers {
	return &dataFetchers{
		ethrpcClient: ethrpcClient,
		config:       config,
	}
}

func newBasePoolState(data *quoteData) *pools.BasePoolState {
	ticks := lo.Map(data.Ticks, func(tick pools.TickRPC, _ int) pools.Tick {
		return pools.Tick{
			Number:         tick.Number,
			LiquidityDelta: big256.SFromBig(tick.LiquidityDelta),
		}
	})
	state := pools.NewBasePoolState(
		pools.NewBasePoolSwapState(
			math.FloatSqrtRatioToFixed(big256.FromBig(data.SqrtRatioFloat)),
			big256.FromBig(data.Liquidity),
			-1,
		),
		ticks,
		[2]int32{data.MinTick, data.MaxTick},
		data.Tick,
	)
	state.AddLiquidityCutoffs()
	state.ActiveTickIndex = pools.NearestInitializedTickIndex(state.SortedTicks, state.ActiveTick)

	return state
}

func newFullRangePoolState(data *quoteData) *pools.FullRangePoolState {
	return pools.NewFullRangePoolState(
		pools.NewFullRangePoolSwapState(math.FloatSqrtRatioToFixed(big256.FromBig(data.SqrtRatioFloat))),
		big256.FromBig(data.Liquidity),
	)
}

func newStableswapPoolState(data *quoteData) *pools.StableswapPoolState {
	return pools.NewStableswapPoolState(
		pools.NewStableswapPoolSwapState(math.FloatSqrtRatioToFixed(big256.FromBig(data.SqrtRatioFloat))),
		big256.FromBig(data.Liquidity),
	)
}

func newOraclePoolState(data *quoteData) *pools.OraclePoolState {
	return pools.NewFullRangePoolState(
		pools.NewFullRangePoolSwapState(math.FloatSqrtRatioToFixed(big256.FromBig(data.SqrtRatioFloat))),
		big256.FromBig(data.Liquidity),
	)
}

func newTwammPoolState(data *twammQuoteData) *pools.TwammPoolState {
	return pools.NewTwammPoolState(
		pools.NewFullRangePoolState(
			pools.NewFullRangePoolSwapState(math.FloatSqrtRatioToFixed(big256.FromBig(data.SqrtRatio))),
			big256.FromBig(data.Liquidity),
		),
		pools.NewTimedPoolState(pools.NewTimedPoolSwapState(big256.FromBig(data.SaleRateToken0), big256.FromBig(data.SaleRateToken1), data.LastVirtualOrderExecutionTime), lo.Map(data.SaleRateDeltas, func(srd timeSaleRateInfo, _ int) pools.TimeRateDelta {
			return pools.NewTimeRateDelta(
				srd.Time,
				big256.SFromBig(srd.SaleRateDelta0),
				big256.SFromBig(srd.SaleRateDelta1),
			)
		})),
	)
}

func newBoostedFeesPoolState(quoteData *quoteData, boostedFeesData *boostedFeesQuoteData) *pools.BoostedFeesPoolState {
	timedState := pools.NewTimedPoolState(
		pools.NewTimedPoolSwapState(
			big256.FromBig(boostedFeesData.DonateRateToken0),
			big256.FromBig(boostedFeesData.DonateRateToken1),
			boostedFeesData.LastDonateTime,
		),
		lo.Map(boostedFeesData.DonateRateDeltas, func(srd boostedTimeDonateRateInfo, _ int) pools.TimeRateDelta {
			return pools.NewTimeRateDelta(
				srd.Time,
				big256.SFromBig(srd.DonateRateDelta0),
				big256.SFromBig(srd.DonateRateDelta1),
			)
		}),
	)

	return pools.NewBoostedFeesPoolState(newBasePoolState(quoteData), timedState)
}
