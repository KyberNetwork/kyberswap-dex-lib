package ekubov3

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
	ve33DataFetcherMethod               = "getVe33QuoteData"
	mevCaptureV21GetConfigMethod        = "getConfig"
	mevCaptureV21GetAnchorMethod        = "getAnchor"
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

	mevCaptureV21ConfigData struct {
		HalfLife               uint32 `json:"halfLife"`
		SlopeK                 uint8  `json:"slopeK"`
		SegmentExp             uint8  `json:"segmentExp"`
		SegmentWidthMultiplier uint32 `json:"segmentWidthMultiplier"`
		JLin                   uint8  `json:"jLin"`
		MaxFee                 uint64 `json:"maxFee"`
		ClampTicks             uint32 `json:"clampTicks"`
		MGate                  uint8  `json:"mGate"`
	}

	mevCaptureV21AnchorData struct {
		LastUpdateTime uint32 `json:"lastUpdateTime"`
		LRefTime       uint32 `json:"lRefTime"`
		LastPosTime    uint32 `json:"lastPosTime"`
		LRefBits       uint8  `json:"lRefBits"`
		SnapBits       uint8  `json:"snapBits"`
		LRefRaiseTime  uint32 `json:"lRefRaiseTime"`
		AnchorX16      int64  `json:"anchorX16"`
	}

	ve33QuoteData struct {
		QuoteData quoteData `json:"quoteData"`
		SwapFee   uint64    `json:"swapFee"`
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

	twammPoolKeys, boostedFeeConcentratedPoolKeys, ve33PoolKeys, basicPoolKeys := make([]pools.AnyPoolKey, 0), make([]pools.AnyPoolKey, 0), make([]pools.AnyPoolKey, 0), make([]poolKeyWithExtType, 0)
	mevCaptureV21PoolKeys := make([]pools.AnyPoolKey, 0)

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
		case ExtensionTypeVe33:
			ve33PoolKeys = append(ve33PoolKeys, key)
		case ExtensionTypeMevCaptureV21:
			mevCaptureV21PoolKeys = append(mevCaptureV21PoolKeys, key)
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
					pool = pools.NewConcentratedPool(poolKey.ToConcentrated(config), newConcentratedPoolState(data))
				default:
					logger.Errorf("unexpected pool type config %T", config)
					continue
				}
			case ExtensionTypeOracle:
				if fullRangeConfig, ok := config.(pools.FullRangePoolTypeConfig); ok {
					pool = pools.NewOraclePool(poolKey.ToFullRange(fullRangeConfig), newOraclePoolState(data))
				} else {
					logger.Errorf("expected full range pool type config for Oracle pool, received %T", config)
					continue
				}
			case ExtensionTypeMevCapture:
				if concentratedConfig, ok := config.(pools.ConcentratedPoolTypeConfig); ok {
					pool = pools.NewMevCapturePool(poolKey.ToConcentrated(concentratedConfig), newConcentratedPoolState(data))
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

	for _, poolKeyBatch := range lo.Chunk(ve33PoolKeys, maxBatchSize) {
		if f.config.Ve33DataFetcher == "" {
			return nil, fmt.Errorf("missing Ve33DataFetcher for Ve33 pools")
		}

		req := f.ethrpcClient.R().SetContext(ctx)
		if overrides != nil {
			req.SetOverrides(overrides)
		}

		batchQuoteData := make([]ve33QuoteData, len(poolKeyBatch))
		resp, err := req.AddCall(&ethrpc.Call{
			ABI:    abis.Ve33DataFetcherABI,
			Target: f.config.Ve33DataFetcher,
			Method: ve33DataFetcherMethod,
			Params: []any{
				lo.Map(poolKeyBatch, func(poolKey pools.AnyPoolKey, _ int) pools.AbiPoolKey {
					return poolKey.ToAbi()
				}),
				minTickSpacingsPerPool,
			},
		}, []any{&batchQuoteData}).Aggregate()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve quote data for Ve33 pools: %w", err)
		}

		var blockNumber uint64
		if resp.BlockNumber != nil {
			blockNumber = resp.BlockNumber.Uint64()
		}

		for i, poolKey := range poolKeyBatch {
			data := &batchQuoteData[i]
			var underlying pools.Pool
			switch config := poolKey.Config.TypeConfig.(type) {
			case pools.FullRangePoolTypeConfig:
				underlying = pools.NewFullRangePool(poolKey.ToFullRange(config), newFullRangePoolState(&data.QuoteData))
			case pools.StableswapPoolTypeConfig:
				underlying = pools.NewStableswapPool(poolKey.ToStableswap(config), newStableswapPoolState(&data.QuoteData))
			case pools.ConcentratedPoolTypeConfig:
				underlying = pools.NewConcentratedPool(poolKey.ToConcentrated(config), newConcentratedPoolState(&data.QuoteData))
			default:
				logger.Errorf("unexpected pool type config %T for Ve33 pool", config)
				continue
			}

			fetchedPools = append(fetchedPools, fetchedPool{
				Pool:        pools.NewVe33Pool(underlying, data.SwapFee),
				blockNumber: blockNumber,
				key:         poolKey,
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
			dataFetcher := f.config.TwammDataFetcher(poolKey.Extension())
			if dataFetcher == "" {
				return nil, fmt.Errorf("missing TWAMMDataFetcher for extension %s", poolKey.Extension())
			}

			req.AddCall(&ethrpc.Call{
				ABI:    abis.TwammDataFetcherABI,
				Target: dataFetcher,
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
			if fullRangeConfig, ok := config.(pools.FullRangePoolTypeConfig); ok {
				fetchedPools = append(fetchedPools, fetchedPool{
					PoolWithBlockNumber{
						Pool: pools.NewTwammPool(
							poolKey.ToFullRange(fullRangeConfig),
							newTwammPoolState(&tuple.A.twammQuoteData),
						),
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
		batchQuoteData := make([]quoteData, batchSize)
		batchBoostedFeesData := make([]struct{ boostedFeesQuoteData }, batchSize)

		req.AddCall(&ethrpc.Call{
			ABI:    abis.QuoteDataFetcherABI,
			Target: f.config.QuoteDataFetcher,
			Method: quoteDataFetcherMethod,
			Params: []any{
				lo.Map(poolKeyBatch, func(poolKey pools.AnyPoolKey, _ int) pools.AbiPoolKey {
					return poolKey.ToAbi()
				}),
				minTickSpacingsPerPool,
			},
		}, []any{&batchQuoteData})
		for i, poolKey := range poolKeyBatch {
			req.AddCall(&ethrpc.Call{
				ABI:    abis.BoostedFeesDataFetcherABI,
				Target: f.config.BoostedFeesDataFetcher,
				Method: boostedFeesDataFetcherMethod,
				Params: []any{poolKey.ToAbi()},
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

			if concentratedConfig, ok := config.(pools.ConcentratedPoolTypeConfig); ok {
				fetchedPools = append(fetchedPools, fetchedPool{
					PoolWithBlockNumber{
						Pool:        pools.NewBoostedFeesPool(poolKey.ToConcentrated(concentratedConfig), newBoostedFeesPoolState(&tuple.B, &tuple.C.boostedFeesQuoteData)),
						blockNumber: blockNumber,
					},
					poolKey,
				})
			} else {
				logger.Errorf("expected concentrated pool type config for BoostedFees pool, received %T", config)
			}
		}
	}

	for _, poolKeyBatch := range lo.Chunk(mevCaptureV21PoolKeys, maxBatchSize) {
		batch, err := f.fetchMevCaptureV21Pools(ctx, poolKeyBatch, overrides)
		if err != nil {
			return nil, err
		}
		fetchedPools = append(fetchedPools, batch...)
	}

	return fetchedPools, nil
}

// fetchMevCaptureV21Pools fetches the Core state and initialized ticks, the extension's configuration and each pool's
// anchor state in one multicall, so that all of them are read at the same block.
func (f *dataFetchers) fetchMevCaptureV21Pools(
	ctx context.Context,
	poolKeys []pools.AnyPoolKey,
	overrides map[common.Address]gethclient.OverrideAccount,
) ([]fetchedPool, error) {
	req := f.ethrpcClient.R().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	batchQuoteData := make([]quoteData, len(poolKeys))
	req.AddCall(&ethrpc.Call{
		ABI:    abis.QuoteDataFetcherABI,
		Target: f.config.QuoteDataFetcher,
		Method: quoteDataFetcherMethod,
		Params: []any{
			lo.Map(poolKeys, func(poolKey pools.AnyPoolKey, _ int) pools.AbiPoolKey {
				return poolKey.ToAbi()
			}),
			minTickSpacingsPerPool,
		},
	}, []any{&batchQuoteData})

	extensions := lo.Uniq(lo.Map(poolKeys, func(poolKey pools.AnyPoolKey, _ int) common.Address {
		return poolKey.Extension()
	}))
	configs := make([]struct{ mevCaptureV21ConfigData }, len(extensions))
	for i, extension := range extensions {
		req.AddCall(&ethrpc.Call{
			ABI:    abis.MevCaptureV21ABI,
			Target: hexutil.Encode(extension[:]),
			Method: mevCaptureV21GetConfigMethod,
		}, []any{&configs[i]})
	}

	anchors := make([]mevCaptureV21AnchorData, len(poolKeys))
	for i, poolKey := range poolKeys {
		poolId, err := poolKey.NumId()
		if err != nil {
			return nil, err
		}
		req.AddCall(&ethrpc.Call{
			ABI:    abis.MevCaptureV21ABI,
			Target: hexutil.Encode(poolKey.Config.Extension[:]),
			Method: mevCaptureV21GetAnchorMethod,
			Params: []any{common.BytesToHash(poolId)},
		}, []any{&anchors[i]})
	}

	resp, err := req.Aggregate()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve quote data for MEVCapture v2.1 pools: %w", err)
	}

	var blockNumber uint64
	if resp.BlockNumber != nil {
		blockNumber = resp.BlockNumber.Uint64()
	}

	fetchedPools := make([]fetchedPool, 0, len(poolKeys))
	for i, poolKey := range poolKeys {
		concentratedConfig, ok := poolKey.Config.TypeConfig.(pools.ConcentratedPoolTypeConfig)
		if !ok {
			logger.Errorf("expected concentrated pool type config for MEVCapture v2.1 pool, received %T",
				poolKey.Config.TypeConfig)
			continue
		}

		config := configs[lo.IndexOf(extensions, poolKey.Extension())].mevCaptureV21ConfigData
		pool, err := pools.NewMevCaptureV21Pool(
			poolKey.ToConcentrated(concentratedConfig),
			newMevCaptureV21PoolState(&batchQuoteData[i], &config, &anchors[i]),
		)
		if err != nil {
			logger.Errorf("invalid MEVCapture v2.1 pool state for %v: %v", poolKey.ToAbi(), err)
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

	return fetchedPools, nil
}

func NewDataFetchers(ethrpcClient *ethrpc.Client, config *Config) *dataFetchers {
	return &dataFetchers{
		ethrpcClient: ethrpcClient,
		config:       config,
	}
}

func newConcentratedPoolState(data *quoteData) *pools.ConcentratedPoolState {
	ticks := lo.Map(data.Ticks, func(tick pools.TickRPC, _ int) pools.Tick {
		return pools.Tick{
			Number:         tick.Number,
			LiquidityDelta: big256.SFromBig(tick.LiquidityDelta),
		}
	})
	state := pools.NewConcentratedPoolState(
		pools.NewConcentratedPoolSwapState(
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

func newMevCaptureV21PoolState(
	data *quoteData,
	config *mevCaptureV21ConfigData,
	anchor *mevCaptureV21AnchorData,
) *pools.MevCaptureV21PoolState {
	return pools.NewMevCaptureV21PoolState(
		&pools.MevCaptureV21PoolSwapState{
			SqrtRatio: math.FloatSqrtRatioToFixed(big256.FromBig(data.SqrtRatioFloat)),
			Liquidity: big256.FromBig(data.Liquidity),
			Tick:      data.Tick,
			Anchor: pools.MevCaptureV21Anchor{
				LastUpdateTime: anchor.LastUpdateTime,
				LRefTime:       anchor.LRefTime,
				LastPosTime:    anchor.LastPosTime,
				LRefBits:       anchor.LRefBits,
				SnapBits:       anchor.SnapBits,
				LRefRaiseTime:  anchor.LRefRaiseTime,
				AnchorX16:      anchor.AnchorX16,
			},
		},
		pools.MevCaptureV21Config{
			HalfLife:   config.HalfLife,
			SlopeK:     config.SlopeK,
			SegmentExp: config.SegmentExp,
			JLin:       config.JLin,
			MaxFee:     config.MaxFee,
			ClampTicks: config.ClampTicks,
			MGate:      config.MGate,
		},
		lo.Map(data.Ticks, func(tick pools.TickRPC, _ int) pools.Tick {
			return pools.Tick{
				Number:         tick.Number,
				LiquidityDelta: big256.SFromBig(tick.LiquidityDelta),
			}
		}),
		[2]int32{data.MinTick, data.MaxTick},
	)
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

	return pools.NewBoostedFeesPoolState(newConcentratedPoolState(quoteData), timedState)
}
