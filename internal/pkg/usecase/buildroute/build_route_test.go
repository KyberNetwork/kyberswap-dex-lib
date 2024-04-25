package buildroute_test

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase"
	"github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/buildroute"
	mockEncode "github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/encode"
	"github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/encode/clientdata"
	. "github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type dummyL1FeeCalculator struct{}

func (d *dummyL1FeeCalculator) CalculateL1Fee(ctx context.Context, chainId valueobject.ChainID, encodedSwapData string) (*big.Int, error) {
	return nil, nil
}

func TestBuildRouteUseCase_Handle(t *testing.T) {
	t.Parallel()

	theErr := errors.New("some error")
	recipient := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	sender := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c759bc2"
	amountIn := big.NewInt(20000)

	dummyL1FeeCalculator := &dummyL1FeeCalculator{}

	testCases := []struct {
		name    string
		prepare func(ctrl *gomock.Controller, config Config, wg *sync.WaitGroup) *BuildRouteUseCase
		command dto.BuildRouteCommand
		config  Config
		result  *dto.BuildRouteResult
		err     error
	}{
		{
			name: "it should return correct error when encoder return error",
			prepare: func(ctrl *gomock.Controller, config Config, wg *sync.WaitGroup) *BuildRouteUseCase {
				clientDataEncoder := clientdata.NewMockIClientDataEncoder(ctrl)
				clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil)

				encodeBuilder := usecase.NewMockIEncodeBuilder(ctrl)
				encoder := mockEncode.NewMockIEncoder(ctrl)
				encodeBuilder.EXPECT().GetEncoder(gomock.Any()).Return(encoder).AnyTimes()
				encoder.EXPECT().
					Encode(gomock.Any()).
					Return("", theErr).AnyTimes()
				encoder.EXPECT().
					GetExecutorAddress(gomock.Any()).
					Return("0x00").AnyTimes()
				encoder.EXPECT().
					GetRouterAddress().
					Return("0x01").AnyTimes()

				tokenRepository := usecase.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.Token{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						},
						nil,
					)

				priceRepository := usecase.NewMockIPriceRepository(ctrl)
				priceRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.Price{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", MarketPrice: 1, PreferPriceSource: entity.PriceSourceCoingecko},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", MarketPrice: 1, PreferPriceSource: entity.PriceSourceCoingecko},
						},
						nil,
					)

				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().IncreasePoolsTotalCount(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

				executorBalanceRepository := buildroute.NewMockIExecutorBalanceRepository(ctrl)
				executorBalanceRepository.EXPECT().HasToken(gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()
				executorBalanceRepository.EXPECT().HasPoolApproval(gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()

				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Times(0)

				return NewBuildRouteUseCase(
					tokenRepository,
					priceRepository,
					poolRepository,
					executorBalanceRepository,
					gasEstimator,
					dummyL1FeeCalculator,
					nil,
					clientDataEncoder,
					encodeBuilder,
					nil,
					config,
				)
			},
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:                      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:                     big.NewInt(20000),
					AmountInUSD:                  0,
					TokenInMarketPriceAvailable:  false,
					TokenOut:                     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:                    big.NewInt(0),
					AmountOutUSD:                 0,
					TokenOutMarketPriceAvailable: false,
					Gas:                          0,
					GasPrice:                     big.NewFloat(100.2),
					GasUSD:                       0,
					ExtraFee:                     valueobject.ExtraFee{},
					Route:                        [][]valueobject.Swap{},
				},
				SlippageTolerance: 5,
				Recipient:         "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Sender:            sender,
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: false},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WindowSize:        time.Minute * 15,
					FaultyExpiredTime: time.Minute * 3,
				}},
			result: nil,
			err:    theErr,
		},
		{
			name: "it should return correct result and run estimate Gas when there is no error and Feature flag is on",
			prepare: func(ctrl *gomock.Controller, config Config, wg *sync.WaitGroup) *BuildRouteUseCase {
				clientDataEncoder := clientdata.NewMockIClientDataEncoder(ctrl)

				clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil)

				encoder := mockEncode.NewMockIEncoder(ctrl)
				encodeBuilder := usecase.NewMockIEncodeBuilder(ctrl)
				encodeBuilder.EXPECT().GetEncoder(gomock.Any()).AnyTimes().Return(encoder)
				encodedData := "mockEncodedData"

				encoder.EXPECT().
					Encode(gomock.Any()).
					Return(encodedData, nil)
				encoder.EXPECT().
					GetExecutorAddress(gomock.Any()).
					Return("0x00").AnyTimes()
				encoder.EXPECT().
					GetRouterAddress().
					Return("0x01").AnyTimes()

				tokenRepository := usecase.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.Token{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						},
						nil,
					)

				priceRepository := usecase.NewMockIPriceRepository(ctrl)
				priceRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.Price{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", MarketPrice: 1, PreferPriceSource: entity.PriceSourceCoingecko},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", MarketPrice: 1, PreferPriceSource: entity.PriceSourceCoingecko},
						},
						nil,
					)
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().IncreasePoolsTotalCount(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

				executorBalanceRepository := buildroute.NewMockIExecutorBalanceRepository(ctrl)
				executorBalanceRepository.EXPECT().HasToken(gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()
				executorBalanceRepository.EXPECT().HasPoolApproval(gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()

				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				tx := NewUnsignedTransaction(
					sender,
					recipient,
					encodedData,
					constant.Zero,
					nil,
				)
				gasEstimator.EXPECT().Execute(gomock.Any(), gomock.Eq(tx)).Times(1).Return(uint64(10), float64(1.5), nil)

				return NewBuildRouteUseCase(
					tokenRepository,
					priceRepository,
					poolRepository,
					executorBalanceRepository,
					gasEstimator,
					dummyL1FeeCalculator,
					nil,
					clientDataEncoder,
					encodeBuilder,
					nil,
					config,
				)
			},
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:                      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:                     big.NewInt(20000),
					AmountInUSD:                  0,
					TokenInMarketPriceAvailable:  false,
					TokenOut:                     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:                    big.NewInt(10000),
					AmountOutUSD:                 0,
					TokenOutMarketPriceAvailable: false,
					Gas:                          0,
					GasPrice:                     big.NewFloat(100.2),
					GasUSD:                       0,
					ExtraFee:                     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:      "0xabc",
								AmountOut: big.NewInt(10000),
							},
						},
					},
				},
				SlippageTolerance:   5,
				Recipient:           recipient,
				Sender:              sender,
				EnableGasEstimation: true,
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: false},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WindowSize:        time.Minute * 15,
					FaultyExpiredTime: time.Minute * 3,
				}},
			result: &dto.BuildRouteResult{
				AmountIn:      "20000",
				AmountInUSD:   "0.02",
				AmountOut:     "10000",
				AmountOutUSD:  "0.01",
				Gas:           "10",
				GasUSD:        "1.5",
				OutputChange:  OutputChangeNoChange,
				Data:          "mockEncodedData",
				RouterAddress: "0x01",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			err: nil,
		},
		{
			name: "it should return correct result and run estimate Gas async when there is no error and Feature flag is on",
			prepare: func(ctrl *gomock.Controller, config Config, wg *sync.WaitGroup) *BuildRouteUseCase {
				wg.Add(1)
				clientDataEncoder := clientdata.NewMockIClientDataEncoder(ctrl)

				clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil)

				encoder := mockEncode.NewMockIEncoder(ctrl)
				encodeBuilder := usecase.NewMockIEncodeBuilder(ctrl)
				encodeBuilder.EXPECT().GetEncoder(gomock.Any()).AnyTimes().Return(encoder)
				encodedData := "mockEncodedData"

				encoder.EXPECT().
					Encode(gomock.Any()).
					Return(encodedData, nil)
				encoder.EXPECT().
					GetExecutorAddress(gomock.Any()).
					Return("0x00").AnyTimes()
				encoder.EXPECT().
					GetRouterAddress().
					Return("0x01").AnyTimes()

				tokenRepository := usecase.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.Token{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						},
						nil,
					)

				priceRepository := usecase.NewMockIPriceRepository(ctrl)
				priceRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.Price{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", MarketPrice: 1, PreferPriceSource: entity.PriceSourceCoingecko},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", MarketPrice: 1, PreferPriceSource: entity.PriceSourceCoingecko},
						},
						nil,
					)
				wg.Add(1)
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().IncreasePoolsTotalCount(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(arg0, arg1, arg2 interface{}) {
					defer wg.Done()
				}).Return(map[string]int64{}, []error{}).Times(1)

				executorBalanceRepository := buildroute.NewMockIExecutorBalanceRepository(ctrl)
				executorBalanceRepository.EXPECT().HasToken(gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()
				executorBalanceRepository.EXPECT().HasPoolApproval(gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()

				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				tx := NewUnsignedTransaction(
					sender,
					recipient,
					encodedData,
					constant.Zero,
					nil,
				)
				gasEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Eq(tx)).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(uint64(10), nil)

				return NewBuildRouteUseCase(
					tokenRepository,
					priceRepository,
					poolRepository,
					executorBalanceRepository,
					gasEstimator,
					dummyL1FeeCalculator,
					nil,
					clientDataEncoder,
					encodeBuilder,
					nil,
					config,
				)
			},
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:                      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:                     big.NewInt(20000),
					AmountInUSD:                  0,
					TokenInMarketPriceAvailable:  false,
					TokenOut:                     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:                    big.NewInt(10000),
					AmountOutUSD:                 0,
					TokenOutMarketPriceAvailable: false,
					Gas:                          15,
					GasPrice:                     big.NewFloat(100.2),
					GasUSD:                       100,
					ExtraFee:                     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:      "0xabc",
								AmountOut: big.NewInt(10000),
							},
						},
					},
				},
				SlippageTolerance: 5,
				Recipient:         recipient,
				Sender:            sender,
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WindowSize:        time.Minute * 15,
					FaultyExpiredTime: time.Minute * 3,
				}},
			result: &dto.BuildRouteResult{
				AmountIn:      "20000",
				AmountInUSD:   "0.02",
				AmountOut:     "10000",
				AmountOutUSD:  "0.01",
				Gas:           "15",
				GasUSD:        "100",
				OutputChange:  OutputChangeNoChange,
				Data:          "mockEncodedData",
				RouterAddress: "0x01",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			err: nil,
		},
		{
			name: "it should return correct result and run estimate Gas when there is no error and Feature flag is on with token in is Ether",
			prepare: func(ctrl *gomock.Controller, config Config, wg *sync.WaitGroup) *BuildRouteUseCase {
				clientDataEncoder := clientdata.NewMockIClientDataEncoder(ctrl)

				clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil)

				encoder := mockEncode.NewMockIEncoder(ctrl)
				encodeBuilder := usecase.NewMockIEncodeBuilder(ctrl)
				encodeBuilder.EXPECT().GetEncoder(gomock.Any()).AnyTimes().Return(encoder)
				encodedData := "mockEncodedData"

				encoder.EXPECT().
					Encode(gomock.Any()).
					Return(encodedData, nil)
				encoder.EXPECT().
					GetExecutorAddress(gomock.Any()).
					Return("0x00").AnyTimes()
				encoder.EXPECT().
					GetRouterAddress().
					Return("0x01").AnyTimes()

				tokenRepository := usecase.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.Token{
							{Address: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", Decimals: 6},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						},
						nil,
					)

				priceRepository := usecase.NewMockIPriceRepository(ctrl)
				priceRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.Price{
							{Address: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", MarketPrice: 1, PreferPriceSource: entity.PriceSourceCoingecko},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", MarketPrice: 1, PreferPriceSource: entity.PriceSourceCoingecko},
						},
						nil,
					)
				wg.Add(1)
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().IncreasePoolsTotalCount(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(arg0, arg1, arg2 interface{}) {
					defer wg.Done()
				}).Return(map[string]int64{}, []error{}).Times(1)

				executorBalanceRepository := buildroute.NewMockIExecutorBalanceRepository(ctrl)
				executorBalanceRepository.EXPECT().HasToken(gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()
				executorBalanceRepository.EXPECT().HasPoolApproval(gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()

				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				tx := NewUnsignedTransaction(
					sender,
					recipient,
					encodedData,
					amountIn,
					nil,
				)
				gasEstimator.EXPECT().Execute(gomock.Any(), gomock.Eq(tx)).Times(1).Return(uint64(10), float64(1.5), nil)

				return NewBuildRouteUseCase(
					tokenRepository,
					priceRepository,
					poolRepository,
					executorBalanceRepository,
					gasEstimator,
					dummyL1FeeCalculator,
					nil,
					clientDataEncoder,
					encodeBuilder,
					nil,
					config,
				)
			},
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:                      valueobject.EtherAddress,
					AmountIn:                     amountIn,
					AmountInUSD:                  0,
					TokenInMarketPriceAvailable:  false,
					TokenOut:                     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:                    big.NewInt(10000),
					AmountOutUSD:                 0,
					TokenOutMarketPriceAvailable: false,
					Gas:                          0,
					GasPrice:                     big.NewFloat(100.2),
					GasUSD:                       0,
					ExtraFee:                     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:      "0xabc",
								AmountOut: big.NewInt(10000),
							},
						},
					},
				},
				SlippageTolerance:   5,
				Recipient:           recipient,
				Sender:              sender,
				EnableGasEstimation: true,
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WindowSize:        time.Minute * 15,
					FaultyExpiredTime: time.Minute * 3,
				}},
			result: &dto.BuildRouteResult{
				AmountIn:      "20000",
				AmountInUSD:   "0.02",
				AmountOut:     "10000",
				AmountOutUSD:  "0.01",
				Gas:           "10",
				GasUSD:        "1.5",
				OutputChange:  OutputChangeNoChange,
				Data:          "mockEncodedData",
				RouterAddress: "0x01",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			err: nil,
		},
	}

	wg := sync.WaitGroup{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := tc.prepare(ctrl, tc.config, &wg)
			result, err := uc.Handle(context.Background(), tc.command)
			wg.Wait()

			assert.Equal(t, tc.result, result)
			assert.ErrorIs(t, err, tc.err)
		})
	}

}

func TestBuildRouteUseCase_HandleWithGasEstimation(t *testing.T) {
	t.Parallel()

	recipient := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	sender := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756bc2"
	returnAmountNotEnoughError := errors.New("execution reverted: Return amount is not enough")

	testCases := []struct {
		name           string
		command        dto.BuildRouteCommand
		estimateGas    func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator
		poolRepository func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository
		result         *dto.BuildRouteResult
		config         Config
		err            error
	}{
		{
			name: "it should return correct result and run estimate Gas when there is no error, feature flag is on",
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:                      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:                     big.NewInt(20000),
					AmountInUSD:                  0,
					TokenInMarketPriceAvailable:  false,
					TokenOut:                     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:                    big.NewInt(10000),
					AmountOutUSD:                 0,
					TokenOutMarketPriceAvailable: false,
					Gas:                          12,
					GasPrice:                     big.NewFloat(100.2),
					GasUSD:                       1.5,
					ExtraFee:                     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:      "0xabc",
								AmountOut: big.NewInt(10000),
							},
						},
					},
				},
				SlippageTolerance:   5,
				Recipient:           recipient,
				EnableGasEstimation: true,
				Sender:              sender,
			},
			result: &dto.BuildRouteResult{
				AmountIn:      "20000",
				AmountInUSD:   "0.02",
				AmountOut:     "10000",
				AmountOutUSD:  "0.01",
				Gas:           "1234",
				GasUSD:        "1.5",
				OutputChange:  OutputChangeNoChange,
				Data:          "mockEncodedData",
				RouterAddress: "0x01",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			estimateGas: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(uint64(1234), float64(1.5), nil).Times(1)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().IncreasePoolsTotalCount(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)

				return poolRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WindowSize:        time.Minute * 15,
					FaultyExpiredTime: time.Minute * 3,
				}},
			err: nil,
		},
		{
			name: "it should return correct result and disable run estimate Gas when there is no error, feature flag is on",
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:                      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:                     big.NewInt(20000),
					AmountInUSD:                  0,
					TokenInMarketPriceAvailable:  false,
					TokenOut:                     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:                    big.NewInt(10000),
					AmountOutUSD:                 0,
					TokenOutMarketPriceAvailable: false,
					Gas:                          12,
					GasPrice:                     big.NewFloat(100.2),
					GasUSD:                       0,
					ExtraFee:                     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:      "0xabc",
								AmountOut: big.NewInt(10000),
							},
						},
					},
				},
				SlippageTolerance:   5,
				Recipient:           recipient,
				EnableGasEstimation: false,
			},
			result: &dto.BuildRouteResult{
				AmountIn:      "20000",
				AmountInUSD:   "0.02",
				AmountOut:     "10000",
				AmountOutUSD:  "0.01",
				Gas:           "12",
				GasUSD:        "0",
				OutputChange:  OutputChangeNoChange,
				Data:          "mockEncodedData",
				RouterAddress: "0x01",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			estimateGas: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(0)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().IncreasePoolsTotalCount(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)

				return poolRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WindowSize:        time.Minute * 15,
					FaultyExpiredTime: time.Minute * 3,
				}},
			err: nil,
		},
		{
			name: "it should return correct result and run estimate Gas in goroutine when there is no error because feature flag is on but disable estimateGas",
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:                      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:                     big.NewInt(20000),
					AmountInUSD:                  0,
					TokenInMarketPriceAvailable:  false,
					TokenOut:                     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:                    big.NewInt(10000),
					AmountOutUSD:                 0,
					TokenOutMarketPriceAvailable: false,
					Gas:                          7,
					GasPrice:                     big.NewFloat(100.2),
					GasUSD:                       0,
					ExtraFee:                     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:      "0xabc",
								AmountOut: big.NewInt(10000),
							},
						},
					},
				},
				SlippageTolerance:   5,
				Recipient:           recipient,
				Sender:              "0xabc",
				EnableGasEstimation: false,
			},
			result: &dto.BuildRouteResult{
				AmountIn:      "20000",
				AmountInUSD:   "0.02",
				AmountOut:     "10000",
				AmountOutUSD:  "0.01",
				Gas:           "7",
				GasUSD:        "0",
				OutputChange:  OutputChangeNoChange,
				Data:          "mockEncodedData",
				RouterAddress: "0x01",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			estimateGas: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				wg.Add(1)
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Times(1).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(uint64(1), nil)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				wg.Add(1)
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().IncreasePoolsTotalCount(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(arg0, arg1, arg2 interface{}) {
					defer wg.Done()
				}).Return(map[string]int64{"0xabc:13:11:60": 1}, []error{}).Times(1)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)

				return poolRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WindowSize:        time.Minute * 15,
					FaultyExpiredTime: time.Minute * 3,
				}},
			err: nil,
		},
		{
			name: "it should return error when EnableGasEstimation is true and sender is empty, feature flag is on",
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:                      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:                     big.NewInt(20000),
					AmountInUSD:                  0,
					TokenInMarketPriceAvailable:  false,
					TokenOut:                     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:                    big.NewInt(10000),
					AmountOutUSD:                 0,
					TokenOutMarketPriceAvailable: false,
					Gas:                          12,
					GasPrice:                     big.NewFloat(100.2),
					GasUSD:                       0,
					ExtraFee:                     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:      "0xabc",
								AmountOut: big.NewInt(10000),
							},
						},
					},
				},
				SlippageTolerance:   5,
				Recipient:           recipient,
				EnableGasEstimation: true,
			},
			result: nil,
			estimateGas: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(0)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().IncreasePoolsTotalCount(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)

				return poolRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WindowSize:        time.Minute * 15,
					FaultyExpiredTime: time.Minute * 3,
				}},
			err: ErrSenderEmptyWhenEnableEstimateGas,
		},
		{
			name: "it should count faulty pools when estimate gas error is return amount not enough, feature flag is on",
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:                      "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
					AmountIn:                     big.NewInt(500000),
					AmountInUSD:                  0.00000000192722,
					TokenInMarketPriceAvailable:  false,
					TokenOut:                     "0x6b175474e89094c44da98b954eedeac495271d0f",
					AmountOut:                    big.NewInt(1626105316),
					AmountOutUSD:                 0.000000001626105316,
					TokenOutMarketPriceAvailable: false,
					Gas:                          185000,
					GasPrice:                     big.NewFloat(9511845152),
					GasUSD:                       6.782624739119853,
					ExtraFee:                     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:      "0xa478c2975ab1ea89e8196811f51a7b7ade33eb11",
								TokenIn:   "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:  "0x6b175474e89094c44da98b954eedeac495271d0f",
								AmountOut: big.NewInt(1626105316),
							},
						},
					},
				},
				Sender:              sender,
				SlippageTolerance:   5,
				Recipient:           recipient,
				EnableGasEstimation: true,
			},
			result: nil,
			estimateGas: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(1).Return(uint64(0), float64(0.0), returnAmountNotEnoughError)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				wg.Add(2)
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().IncreasePoolsTotalCount(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(arg0, arg1, arg2 interface{}) {
					defer wg.Done()
				}).Return(map[string]int64{"0xa478c2975ab1ea89e8196811f51a7b7ade33eb11:13:11:60": 1}, []error{}).Times(1)
				addr := []string{"0xa478c2975ab1ea89e8196811f51a7b7ade33eb11"}
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Eq(addr)).Times(1).Do(func(arg0, arg2 interface{}) {
					defer wg.Done()
				}).Return(addr, nil)

				return poolRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WindowSize:        time.Minute * 15,
					FaultyExpiredTime: time.Minute * 3,
				}},
			err: errors.WithMessagef(ErrEstimateGasFailed, "estimate gas failed due to %s", returnAmountNotEnoughError.Error()),
		},
		{
			name: "it should not count faulty pools when estimate gas error is some error, feature flag is on",
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:                      "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
					AmountIn:                     big.NewInt(500000),
					AmountInUSD:                  0.00000000192722,
					TokenInMarketPriceAvailable:  false,
					TokenOut:                     "0x6b175474e89094c44da98b954eedeac495271d0f",
					AmountOut:                    big.NewInt(1626105316),
					AmountOutUSD:                 0.000000001626105316,
					TokenOutMarketPriceAvailable: false,
					Gas:                          185000,
					GasPrice:                     big.NewFloat(9511845152),
					GasUSD:                       6.782624739119853,
					ExtraFee:                     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:      "0xa478c2975ab1ea89e8196811f51a7b7ade33eb11",
								TokenIn:   "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:  "0x6b175474e89094c44da98b954eedeac495271d0f",
								AmountOut: big.NewInt(1626105316),
							},
						},
					},
				},
				Sender:            sender,
				SlippageTolerance: 5,
				Recipient:         recipient,
			},
			result: &dto.BuildRouteResult{
				AmountIn:      "500000",
				AmountInUSD:   "0.5",
				AmountOut:     "1626105316",
				AmountOutUSD:  "0.000000001626105316",
				Gas:           "185000",
				GasUSD:        "6.782624739119853",
				OutputChange:  OutputChangeNoChange,
				Data:          "mockEncodedData",
				RouterAddress: "0x01",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			estimateGas: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Times(1).Return(uint64(0), errors.New("test error"))
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				wg.Add(1)
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().IncreasePoolsTotalCount(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(arg0, arg1, arg2 interface{}) {
					defer wg.Done()
				}).Return(map[string]int64{"0xa478c2975ab1ea89e8196811f51a7b7ade33eb11:13:11:60": 1}, []error{}).Times(1)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)

				return poolRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: false, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WindowSize:        time.Minute * 15,
					FaultyExpiredTime: time.Minute * 3,
				}},
			err: nil,
		},
		{
			name: "it should not count faulty pools and still call estimate gas, feature flag is off",
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:                      "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
					AmountIn:                     big.NewInt(500000),
					AmountInUSD:                  0.5,
					TokenInMarketPriceAvailable:  false,
					TokenOut:                     "0x6b175474e89094c44da98b954eedeac495271d0f",
					AmountOut:                    big.NewInt(1626105316),
					AmountOutUSD:                 0.000000001626105316,
					TokenOutMarketPriceAvailable: false,
					Gas:                          185000,
					GasPrice:                     big.NewFloat(9511845152),
					GasUSD:                       6.782624739119853,
					ExtraFee:                     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:      "0xa478c2975ab1ea89e8196811f51a7b7ade33eb11",
								TokenIn:   "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:  "0x6b175474e89094c44da98b954eedeac495271d0f",
								AmountOut: big.NewInt(1626105316),
							},
						},
					},
				},
				Sender:              sender,
				SlippageTolerance:   5,
				Recipient:           recipient,
				EnableGasEstimation: true,
			},
			result: &dto.BuildRouteResult{
				AmountIn:      "500000",
				AmountInUSD:   "0.5",
				AmountOut:     "1626105316",
				AmountOutUSD:  "0.000000001626105316",
				Gas:           "185000",
				GasUSD:        "6.782624739119853",
				OutputChange:  OutputChangeNoChange,
				Data:          "mockEncodedData",
				RouterAddress: "0x01",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			estimateGas: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(0)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().IncreasePoolsTotalCount(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)

				return poolRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: false, IsFaultyPoolDetectorEnable: false},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WindowSize:        time.Minute * 15,
					FaultyExpiredTime: time.Minute * 3,
				}},
			err: nil,
		},
	}

	wg := sync.WaitGroup{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			clientDataEncoder := clientdata.NewMockIClientDataEncoder(ctrl)
			clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil)

			encoder := mockEncode.NewMockIEncoder(ctrl)
			encodeBuilder := usecase.NewMockIEncodeBuilder(ctrl)
			encodeBuilder.EXPECT().GetEncoder(gomock.Any()).AnyTimes().Return(encoder)
			encodedData := "mockEncodedData"

			encoder.EXPECT().
				Encode(gomock.Any()).
				Return(encodedData, nil)
			encoder.EXPECT().
				GetExecutorAddress(gomock.Any()).
				Return("0x00").AnyTimes()
			encoder.EXPECT().
				GetRouterAddress().
				Return("0x01").AnyTimes()

			tokenRepository := usecase.NewMockITokenRepository(ctrl)
			tokenRepository.EXPECT().
				FindByAddresses(gomock.Any(), gomock.Any()).
				Return(
					[]*entity.Token{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						{Address: "0x6b175474e89094c44da98b954eedeac495271d0f", Decimals: 18},
					},
					nil,
				)

			priceRepository := usecase.NewMockIPriceRepository(ctrl)
			priceRepository.EXPECT().
				FindByAddresses(gomock.Any(), gomock.Any()).
				Return(
					[]*entity.Price{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", MarketPrice: 1, PreferPriceSource: entity.PriceSourceCoingecko},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", MarketPrice: 1, PreferPriceSource: entity.PriceSourceCoingecko},
						{Address: "0x6b175474e89094c44da98b954eedeac495271d0f", MarketPrice: 1, PreferPriceSource: entity.PriceSourceCoingecko},
					},
					nil,
				)

			executorBalanceRepository := buildroute.NewMockIExecutorBalanceRepository(ctrl)
			executorBalanceRepository.EXPECT().HasToken(gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()
			executorBalanceRepository.EXPECT().HasPoolApproval(gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()

			gasEstimator := tc.estimateGas(ctrl, &wg)
			poolRepository := tc.poolRepository(ctrl, &wg)
			usecase := NewBuildRouteUseCase(
				tokenRepository,
				priceRepository,
				poolRepository,
				executorBalanceRepository,
				gasEstimator,
				&dummyL1FeeCalculator{},
				nil,
				clientDataEncoder,
				encodeBuilder,
				nil,
				tc.config,
			)

			result, err := usecase.Handle(context.Background(), tc.command)
			wg.Wait()

			assert.Equal(t, tc.result, result)
			if tc.err != nil {
				assert.Equal(t, tc.err.Error(), err.Error())
			}
		})
	}
}

func TestBuildRouteUseCase_HandleWithTrackingKeyTotalCountFaultyPools(t *testing.T) {
	t.Parallel()

	recipient := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	sender := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756bc2"

	testError := errors.New("test error")

	testCases := []struct {
		name            string
		command         dto.BuildRouteCommand
		countTotalPools func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository
		result          *dto.BuildRouteResult
		config          Config
		err             error
		nowFunc         func() time.Time
	}{
		{
			name: "it should return correct result and increase total count on Redis",
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:                      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:                     big.NewInt(2000000000000000000),
					AmountInUSD:                  float64(2000000000000000000),
					TokenInMarketPriceAvailable:  false,
					TokenOut:                     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:                    big.NewInt(4488767370609711072),
					AmountOutUSD:                 float64(4488767370609711072),
					TokenOutMarketPriceAvailable: false,
					Gas:                          345000,
					GasPrice:                     big.NewFloat(100000000),
					GasUSD:                       float64(0.07912413535198341),
					ExtraFee:                     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(996023110963288),
								SwapAmount: big.NewInt(2000000000000000000),
								Exchange:   "pancake",
								PoolType:   "uniswap-v2",
							},
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(4488767370609711072),
								SwapAmount: big.NewInt(996023110963288),
								Exchange:   "smardex",
								PoolType:   "smardex",
							},
						},
					},
				},
				SlippageTolerance:   5,
				Recipient:           recipient,
				EnableGasEstimation: true,
				Sender:              sender,
			},
			result: &dto.BuildRouteResult{
				AmountIn:      "2000000000000000000",
				AmountInUSD:   "2000000000000",
				AmountOut:     "4488767370609711072",
				AmountOutUSD:  "4488767370609.711",
				Gas:           "345000",
				GasUSD:        "0.07912413535198341",
				OutputChange:  OutputChangeNoChange,
				Data:          "mockEncodedData",
				RouterAddress: "0x01",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			countTotalPools: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				wg.Add(1)
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				counterMap := map[string]int64{"0xabc:13:11:60": 2}
				poolRepository.EXPECT().IncreasePoolsTotalCount(gomock.Any(), gomock.Eq(counterMap), gomock.Any()).Do(func(arg0, arg1, arg2 interface{}) {
					defer wg.Done()
				}).Return(map[string]int64{"0xabc:13:11:60": 2}, []error{}).Times(1)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)
				return poolRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WindowSize:        time.Minute * 15,
					FaultyExpiredTime: time.Minute * 3,
				}},
			err: nil,
			nowFunc: func() time.Time {
				time, _ := time.Parse(time.RFC3339, "2023-12-13T11:45:26.371Z")
				return time
			},
		},
		{
			name: "it should return correct result although increase total count on Redis failed",
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:                      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:                     big.NewInt(2000000000000000000),
					AmountInUSD:                  float64(2000000000000000000),
					TokenInMarketPriceAvailable:  false,
					TokenOut:                     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:                    big.NewInt(4488767370609711072),
					AmountOutUSD:                 float64(4488767370609711072),
					TokenOutMarketPriceAvailable: false,
					Gas:                          345000,
					GasPrice:                     big.NewFloat(100000000),
					GasUSD:                       float64(0.07912413535198341),
					ExtraFee:                     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(996023110963288),
								SwapAmount: big.NewInt(2000000000000000000),
								Exchange:   "pancake",
								PoolType:   "uniswap-v2",
							},
							{
								Pool:       "0xabcd",
								AmountOut:  big.NewInt(4488767370609711072),
								SwapAmount: big.NewInt(996023110963288),
								Exchange:   "smardex",
								PoolType:   "smardex",
							},
						},
					},
				},
				SlippageTolerance:   5,
				Recipient:           recipient,
				EnableGasEstimation: true,
				Sender:              sender,
			},
			result: &dto.BuildRouteResult{
				AmountIn:      "2000000000000000000",
				AmountInUSD:   "2000000000000",
				AmountOut:     "4488767370609711072",
				AmountOutUSD:  "4488767370609.711",
				Gas:           "345000",
				GasUSD:        "0.07912413535198341",
				OutputChange:  OutputChangeNoChange,
				Data:          "mockEncodedData",
				RouterAddress: "0x01",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			countTotalPools: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				wg.Add(1)
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				counterMap := map[string]int64{"0xabc:13:11:15": 1, "0xabcd:13:11:15": 1}
				poolRepository.EXPECT().IncreasePoolsTotalCount(gomock.Any(), gomock.Eq(counterMap), gomock.Any()).Do(func(arg0, arg1, arg2 interface{}) {
					defer wg.Done()
				}).Return(map[string]int64{}, []error{testError}).Times(1)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)
				return poolRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WindowSize:        time.Minute * 15,
					FaultyExpiredTime: time.Minute * 3,
				}},
			err: nil,
			nowFunc: func() time.Time {
				time, _ := time.Parse(time.RFC3339, "2023-12-13T11:00:26.371Z")
				return time
			},
		},
		{
			name: "it should return correct result and not increase total count on Redis when feature flag is off",
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:                      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:                     big.NewInt(2000000000000000000),
					AmountInUSD:                  float64(2000000000000000000),
					TokenInMarketPriceAvailable:  false,
					TokenOut:                     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:                    big.NewInt(4488767370609711072),
					AmountOutUSD:                 float64(4488767370609711072),
					TokenOutMarketPriceAvailable: false,
					Gas:                          345000,
					GasPrice:                     big.NewFloat(100000000),
					GasUSD:                       float64(0.07912413535198341),
					ExtraFee:                     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(996023110963288),
								SwapAmount: big.NewInt(2000000000000000000),
								Exchange:   "pancake",
								PoolType:   "uniswap-v2",
							},
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(4488767370609711072),
								SwapAmount: big.NewInt(996023110963288),
								Exchange:   "smardex",
								PoolType:   "smardex",
							},
						},
					},
				},
				SlippageTolerance:   5,
				Recipient:           recipient,
				EnableGasEstimation: true,
				Sender:              sender,
			},
			result: &dto.BuildRouteResult{
				AmountIn:      "2000000000000000000",
				AmountInUSD:   "2000000000000",
				AmountOut:     "4488767370609711072",
				AmountOutUSD:  "4488767370609.711",
				Gas:           "345000",
				GasUSD:        "0.07912413535198341",
				OutputChange:  OutputChangeNoChange,
				Data:          "mockEncodedData",
				RouterAddress: "0x01",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			countTotalPools: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().IncreasePoolsTotalCount(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)
				return poolRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: false, IsFaultyPoolDetectorEnable: false},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WindowSize:        time.Minute * 15,
					FaultyExpiredTime: time.Minute * 3,
				}},
			err: nil,
			nowFunc: func() time.Time {
				time, _ := time.Parse(time.RFC3339, "2023-12-13T11:45:26.371Z")
				return time
			},
		},
	}

	wg := sync.WaitGroup{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			clientDataEncoder := clientdata.NewMockIClientDataEncoder(ctrl)
			clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil)

			encoder := mockEncode.NewMockIEncoder(ctrl)
			encodeBuilder := usecase.NewMockIEncodeBuilder(ctrl)
			encodeBuilder.EXPECT().GetEncoder(gomock.Any()).AnyTimes().Return(encoder)
			encodedData := "mockEncodedData"

			encoder.EXPECT().
				Encode(gomock.Any()).
				Return(encodedData, nil)
			encoder.EXPECT().
				GetExecutorAddress(gomock.Any()).
				Return("0x00").AnyTimes()
			encoder.EXPECT().
				GetRouterAddress().
				Return("0x01").AnyTimes()

			tokenRepository := usecase.NewMockITokenRepository(ctrl)
			tokenRepository.EXPECT().
				FindByAddresses(gomock.Any(), gomock.Any()).
				Return(
					[]*entity.Token{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
					},
					nil,
				)

			priceRepository := usecase.NewMockIPriceRepository(ctrl)
			priceRepository.EXPECT().
				FindByAddresses(gomock.Any(), gomock.Any()).
				Return(
					[]*entity.Price{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", MarketPrice: 1, PreferPriceSource: entity.PriceSourceCoingecko},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", MarketPrice: 1, PreferPriceSource: entity.PriceSourceCoingecko},
					},
					nil,
				)

			executorBalanceRepository := buildroute.NewMockIExecutorBalanceRepository(ctrl)
			executorBalanceRepository.EXPECT().HasToken(gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()
			executorBalanceRepository.EXPECT().HasPoolApproval(gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()
			gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
			if tc.config.FeatureFlags.IsGasEstimatorEnabled {
				gasEstimator.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(uint64(345000), float64(0.07912413535198341), nil).Times(1)
			} else {
				gasEstimator.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(0)
			}

			poolRepository := tc.countTotalPools(ctrl, &wg)

			usecase := NewBuildRouteUseCase(
				tokenRepository,
				priceRepository,
				poolRepository,
				executorBalanceRepository,
				gasEstimator,
				&dummyL1FeeCalculator{},
				nil,
				clientDataEncoder,
				encodeBuilder,
				tc.nowFunc,
				tc.config,
			)

			result, err := usecase.Handle(context.Background(), tc.command)
			wg.Wait()

			assert.Equal(t, tc.result, result)
			assert.ErrorIs(t, tc.err, err)
		})
	}
}
