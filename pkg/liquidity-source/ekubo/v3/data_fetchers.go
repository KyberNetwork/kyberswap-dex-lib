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
	maxBatchSize                  = 100
	minTickSpacingsPerPool uint32 = 2
	basicDataFetcherMethod        = "getQuoteData"
	twammDataFetcherMethod        = "getPoolState"
)

type (
	QuoteData struct {
		Tick           int32           `json:"tick"`
		SqrtRatioFloat *big.Int        `json:"sqrtRatio"`
		Liquidity      *big.Int        `json:"liquidity"`
		MinTick        int32           `json:"minTick"`
		MaxTick        int32           `json:"maxTick"`
		Ticks          []pools.TickRPC `json:"ticks"`
	}

	TimeSaleRateInfo struct {
		Time           uint64   `json:"time"`
		SaleRateDelta0 *big.Int `json:"saleRateDelta0"`
		SaleRateDelta1 *big.Int `json:"saleRateDelta1"`
	}

	TwammQuoteData struct {
		SqrtRatio                     *big.Int           `json:"sqrtRatio"`
		Tick                          int32              `json:"tick"`
		Liquidity                     *big.Int           `json:"liquidity"`
		LastVirtualOrderExecutionTime uint64             `json:"lastVirtualOrderExecutionTime"`
		SaleRateToken0                *big.Int           `json:"saleRateToken0"`
		SaleRateToken1                *big.Int           `json:"saleRateToken1"`
		SaleRateDeltas                []TimeSaleRateInfo `json:"saleRateDeltas"`
	}

	dataFetchers struct {
		ethrpcClient *ethrpc.Client
		config       *Config
	}
)

func (f *dataFetchers) fetchPools(
	ctx context.Context,
	poolKeys []*pools.PoolKey[pools.PoolTypeConfig],
	overrides map[common.Address]gethclient.OverrideAccount,
) ([]*PoolWithBlockNumber, error) {
	if len(poolKeys) == 0 {
		return nil, nil
	}

	twammPoolKeys, basicPoolKeys := lo.FilterReject(poolKeys, func(key *pools.PoolKey[pools.PoolTypeConfig], _ int) bool {
		extensionType := f.config.SupportedExtensions()[key.Extension()]
		return extensionType == ExtensionTypeTwamm
	})

	poolStates := make([]*PoolWithBlockNumber, 0, len(poolKeys))

	for startIdx := 0; startIdx < len(basicPoolKeys); startIdx += maxBatchSize {
		endIdx := min(startIdx+maxBatchSize, len(basicPoolKeys))

		req := f.ethrpcClient.R().SetContext(ctx)
		if overrides != nil {
			req.SetOverrides(overrides)
		}

		batchQuoteData := make([]QuoteData, endIdx-startIdx)
		resp, err := req.AddCall(&ethrpc.Call{
			ABI:    abis.QuoteDataFetcherABI,
			Target: f.config.QuoteDataFetcher,
			Method: basicDataFetcherMethod,
			Params: []any{
				lo.Map(basicPoolKeys[startIdx:endIdx], func(poolKey *pools.PoolKey[pools.PoolTypeConfig], _ int) pools.AbiPoolKey {
					return poolKey.ToAbi()
				}),
				minTickSpacingsPerPool,
			},
		}, []any{&batchQuoteData}).Aggregate()
		if err != nil {
			logger.Errorf("failed to retrieve quote data from basic data fetcher: %v", err)
			return nil, err
		}

		var blockNumber uint64
		if resp.BlockNumber != nil {
			blockNumber = resp.BlockNumber.Uint64()
		}

		for i, data := range batchQuoteData {
			poolKey := basicPoolKeys[startIdx+i]
			extension := poolKey.Extension()

			extensionType, ok := f.config.SupportedExtensions()[extension]
			if !ok {
				return nil, fmt.Errorf("requested pool data for unknown extension %v", extension)
			}

			var pool Pool
			switch extensionType {
			case ExtensionTypeBase:
				switch poolTypeConfig := poolKey.Config.TypeConfig.(type) {
				case pools.FullRangePoolTypeConfig:
					pool = pools.NewFullRangePool(poolKey.ToFullRange(), NewFullRangePoolState(&data))
				case pools.StableswapPoolTypeConfig:
					pool = pools.NewStableswapPool(poolKey.ToStableswap(poolTypeConfig), NewStableswapPoolState(&data))
				case pools.ConcentratedPoolTypeConfig:
					pool = pools.NewBasePool(poolKey.ToConcentrated(poolTypeConfig), NewBasePoolState(&data))
				default:
					return nil, fmt.Errorf("unexpected pool key %v", poolKey)
				}
			case ExtensionTypeOracle:
				_, ok = poolKey.Config.TypeConfig.(pools.FullRangePoolTypeConfig)
				if !ok {
					return nil, fmt.Errorf("oracle pool should have full range pool type config, got %T", poolKey.Config.TypeConfig)
				}
				pool = pools.NewOraclePool(poolKey.ToFullRange(), NewOraclePoolState(&data))
			case ExtensionTypeMevCapture:
				poolTypeConfig, ok := poolKey.Config.TypeConfig.(pools.ConcentratedPoolTypeConfig)
				if !ok {
					return nil, fmt.Errorf("MEV-capture pool should have concentrated pool type config, got %T", poolKey.Config.TypeConfig)
				}
				pool = pools.NewMevCapturePool(poolKey.ToConcentrated(poolTypeConfig), NewBasePoolState(&data))
			default:
				return nil, fmt.Errorf("unexpected extension type %v", extensionType)
			}

			poolStates = append(poolStates, &PoolWithBlockNumber{
				pool,
				blockNumber,
			})
		}
	}

	for startIdx := 0; startIdx < len(twammPoolKeys); startIdx += maxBatchSize {
		endIdx := min(startIdx+maxBatchSize, len(twammPoolKeys))

		req := f.ethrpcClient.R().SetContext(ctx)
		if overrides != nil {
			req.SetOverrides(overrides)
		}

		batchQuoteData := make([]struct{ TwammQuoteData }, endIdx-startIdx)
		for i, poolKey := range twammPoolKeys[startIdx:endIdx] {
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
			logger.Errorf("failed to retrieve quote data from TWAMM data fetcher: %v", err)
			return nil, err
		}

		var blockNumber uint64
		if resp.BlockNumber != nil {
			blockNumber = resp.BlockNumber.Uint64()
		}

		for i, data := range batchQuoteData {
			poolStates = append(poolStates, &PoolWithBlockNumber{
				pools.NewTwammPool(twammPoolKeys[startIdx+i].ToFullRange(), NewTwammPoolState(&data.TwammQuoteData)),
				blockNumber,
			})
		}
	}

	return poolStates, nil
}

func NewDataFetchers(ethrpcClient *ethrpc.Client, config *Config) *dataFetchers {
	return &dataFetchers{
		ethrpcClient: ethrpcClient,
		config:       config,
	}
}

func NewBasePoolState(data *QuoteData) *pools.BasePoolState {
	ticks := lo.Map(data.Ticks, func(tick pools.TickRPC, _ int) pools.Tick {
		return pools.Tick{
			Number:         tick.Number,
			LiquidityDelta: big256.SFromBig(tick.LiquidityDelta),
		}
	})
	state := pools.BasePoolState{
		BasePoolSwapState: &pools.BasePoolSwapState{
			SqrtRatio:       math.FloatSqrtRatioToFixed(big256.FromBig(data.SqrtRatioFloat)),
			Liquidity:       big256.FromBig(data.Liquidity),
			ActiveTickIndex: -1,
		},
		ActiveTick:  data.Tick,
		SortedTicks: ticks,
		TickBounds:  [2]int32{data.MinTick, data.MaxTick},
	}
	state.AddLiquidityCutoffs()
	state.ActiveTickIndex = pools.NearestInitializedTickIndex(state.SortedTicks, state.ActiveTick)

	return &state
}

func NewFullRangePoolState(data *QuoteData) *pools.FullRangePoolState {
	return &pools.FullRangePoolState{
		FullRangePoolSwapState: &pools.FullRangePoolSwapState{
			SqrtRatio: math.FloatSqrtRatioToFixed(big256.FromBig(data.SqrtRatioFloat)),
		},
		Liquidity: big256.FromBig(data.Liquidity),
	}
}

func NewStableswapPoolState(data *QuoteData) *pools.StableswapPoolState {
	return &pools.StableswapPoolState{
		StableswapPoolSwapState: &pools.StableswapPoolSwapState{
			SqrtRatio: math.FloatSqrtRatioToFixed(big256.FromBig(data.SqrtRatioFloat)),
		},
		Liquidity: big256.FromBig(data.Liquidity),
	}
}

func NewOraclePoolState(data *QuoteData) *pools.OraclePoolState {
	return &pools.OraclePoolState{
		FullRangePoolSwapState: &pools.FullRangePoolSwapState{
			SqrtRatio: math.FloatSqrtRatioToFixed(big256.FromBig(data.SqrtRatioFloat)),
		},
		Liquidity: big256.FromBig(data.Liquidity),
	}
}

func NewTwammPoolState(data *TwammQuoteData) *pools.TwammPoolState {
	return &pools.TwammPoolState{
		FullRangePoolState: &pools.FullRangePoolState{
			FullRangePoolSwapState: &pools.FullRangePoolSwapState{
				SqrtRatio: math.FloatSqrtRatioToFixed(big256.FromBig(data.SqrtRatio)),
			},
			Liquidity: big256.FromBig(data.Liquidity),
		},
		Token0SaleRate:    big256.FromBig(data.SaleRateToken0),
		Token1SaleRate:    big256.FromBig(data.SaleRateToken1),
		LastExecutionTime: data.LastVirtualOrderExecutionTime,
		VirtualOrderDeltas: lo.Map(data.SaleRateDeltas, func(srd TimeSaleRateInfo, _ int) pools.TwammSaleRateDelta {
			return pools.TwammSaleRateDelta{
				Time:           srd.Time,
				SaleRateDelta0: big256.SFromBig(srd.SaleRateDelta0),
				SaleRateDelta1: big256.SFromBig(srd.SaleRateDelta1),
			}
		}),
	}
}
