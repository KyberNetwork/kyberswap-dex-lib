package ekubo

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/pools"
)

type BasicQuoteData struct {
	Tick           int32        `json:"tick"`
	SqrtRatioFloat *big.Int     `json:"sqrtRatio"`
	Liquidity      *big.Int     `json:"liquidity"`
	MinTick        int32        `json:"minTick"`
	MaxTick        int32        `json:"maxTick"`
	Ticks          []pools.Tick `json:"ticks"`
}

type TwammSaleRateDelta struct {
	Time           *big.Int `json:"time"`
	SaleRateDelta0 *big.Int `json:"saleRateDelta0"`
	SaleRateDelta1 *big.Int `json:"saleRateDelta1"`
}

type TwammQuoteData struct {
	SqrtRatio                     *big.Int             `json:"sqrtRatio"`
	Tick                          int32                `json:"tick"`
	Liquidity                     *big.Int             `json:"liquidity"`
	LastVirtualOrderExecutionTime *big.Int             `json:"lastVirtualOrderExecutionTime"`
	SaleRateToken0                *big.Int             `json:"saleRateToken0"`
	SaleRateToken1                *big.Int             `json:"saleRateToken1"`
	SaleRateDeltas                []TwammSaleRateDelta `json:"saleRateDeltas"`
}

type dataFetcherAddresses struct {
	basic string
	twamm string
}

type dataFetchers struct {
	ethrpcClient        *ethrpc.Client
	fetcherAddresses    dataFetcherAddresses
	supportedExtensions map[common.Address]ExtensionType
}

func NewDataFetchers(ethrpcClient *ethrpc.Client, cfg *Config) *dataFetchers {
	return &dataFetchers{
		ethrpcClient: ethrpcClient,
		fetcherAddresses: dataFetcherAddresses{
			basic: cfg.BasicDataFetcher,
			twamm: cfg.TwammDataFetcher,
		},
		supportedExtensions: cfg.SupportedExtensions(),
	}
}

const (
	maxBatchSize                  = 100
	minTickSpacingsPerPool uint32 = 2
	basicDataFetcherMethod        = "getQuoteData"
	twammDataFetcherMethod        = "getPoolState"
)

func (f *dataFetchers) fetchPools(
	ctx context.Context,
	poolKeys []*pools.PoolKey,
) ([]*PoolWithBlockNumber, error) {
	if len(poolKeys) == 0 {
		return nil, nil
	}

	twammPoolKeys, basicPoolKeys := lo.FilterReject(poolKeys, func(key *pools.PoolKey, _ int) bool {
		extensionType := f.supportedExtensions[key.Config.Extension]
		return extensionType == ExtensionTypeTwamm
	})

	poolStates := make([]*PoolWithBlockNumber, 0, len(poolKeys))

	for startIdx := 0; startIdx < len(basicPoolKeys); startIdx += maxBatchSize {
		endIdx := min(startIdx+maxBatchSize, len(basicPoolKeys))

		req := f.ethrpcClient.R().SetContext(ctx)
		batchQuoteData := make([]BasicQuoteData, endIdx-startIdx)
		resp, err := req.AddCall(&ethrpc.Call{
			ABI:    abis.BasicDataFetcherABI,
			Target: f.fetcherAddresses.basic,
			Method: basicDataFetcherMethod,
			Params: []any{
				lo.Map(basicPoolKeys[startIdx:endIdx], func(poolKey *pools.PoolKey, _ int) pools.AbiPoolKey {
					return poolKey.ToAbi()
				}),
				minTickSpacingsPerPool,
			},
		}, []any{&batchQuoteData}).Call()
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
			extension := poolKey.Config.Extension

			extensionType, ok := f.supportedExtensions[poolKey.Config.Extension]
			if !ok {
				return nil, fmt.Errorf("requested pool data for unknown extension %v", extension)
			}

			var pool Pool
			switch extensionType {
			case ExtensionTypeBase:
				if poolKey.Config.TickSpacing == 0 {
					pool = pools.NewFullRangePool(poolKey, NewFullRangePoolState(&data))
				} else {
					pool = pools.NewBasePool(poolKey, NewBasePoolState(&data))
				}
			case ExtensionTypeOracle:
				pool = pools.NewOraclePool(poolKey, NewOraclePoolState(&data))
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
		batchQuoteData := make([]TwammQuoteData, endIdx-startIdx)
		for i, poolKey := range twammPoolKeys[startIdx:endIdx] {
			req.AddCall(&ethrpc.Call{
				ABI:    abis.TwammDataFetcherABI,
				Target: f.fetcherAddresses.twamm,
				Method: twammDataFetcherMethod,
				Params: []any{
					poolKey.ToAbi(),
				},
			}, []any{&batchQuoteData[i]}) // FIXME Somehow tries to unmarshal into a *big.Int
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
				pools.NewTwammPool(twammPoolKeys[startIdx+i], NewTwammPoolState(&data)),
				blockNumber,
			})
		}
	}

	return poolStates, nil
}

func NewBasePoolState(data *BasicQuoteData) *pools.BasePoolState {
	state := pools.BasePoolState{
		BasePoolSwapState: &pools.BasePoolSwapState{
			SqrtRatio:       math.FloatSqrtRatioToFixed(data.SqrtRatioFloat),
			Liquidity:       data.Liquidity,
			ActiveTickIndex: pools.NearestInitializedTickIndex(data.Ticks, data.Tick),
		},
		ActiveTick:  data.Tick,
		SortedTicks: data.Ticks,
		TickBounds:  [2]int32{data.MinTick, data.MaxTick},
	}
	state.AddLiquidityCutoffs()

	return &state
}

func NewFullRangePoolState(data *BasicQuoteData) *pools.FullRangePoolState {
	return &pools.FullRangePoolState{
		FullRangePoolSwapState: &pools.FullRangePoolSwapState{
			SqrtRatio: math.FloatSqrtRatioToFixed(data.SqrtRatioFloat),
		},
		Liquidity: data.Liquidity,
	}
}

func NewOraclePoolState(data *BasicQuoteData) *pools.OraclePoolState {
	return &pools.OraclePoolState{
		FullRangePoolSwapState: &pools.FullRangePoolSwapState{
			SqrtRatio: math.FloatSqrtRatioToFixed(data.SqrtRatioFloat),
		},
		Liquidity: data.Liquidity,
	}
}

func NewTwammPoolState(data *TwammQuoteData) *pools.TwammPoolState {
	return &pools.TwammPoolState{
		FullRangePoolState: &pools.FullRangePoolState{
			FullRangePoolSwapState: &pools.FullRangePoolSwapState{
				SqrtRatio: math.FloatSqrtRatioToFixed(data.SqrtRatio),
			},
			Liquidity: data.Liquidity,
		},
		Token0SaleRate:    data.SaleRateToken0,
		Token1SaleRate:    data.SaleRateToken1,
		LastExecutionTime: data.LastVirtualOrderExecutionTime.Uint64(),
		VirtualOrderDeltas: lo.Map(data.SaleRateDeltas, func(srd TwammSaleRateDelta, _ int) pools.TwammSaleRateDelta {
			return pools.TwammSaleRateDelta{
				Time:           srd.Time.Uint64(),
				SaleRateDelta0: srd.SaleRateDelta0,
				SaleRateDelta1: srd.SaleRateDelta1,
			}
		}),
	}
}
