package buildroute_test

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/kyber-pmm"
	kyberpmmClient "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/kyber-pmm/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	hashflowv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hashflow-v3"
	nativev1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native/v1"
	nativev1Client "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native/v1/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	routerEntities "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/buildroute"
	mockEncode "github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/encode"
	"github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/encode/clientdata"
	. "github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/crypto"
)

type dummyL1FeeCalculator struct{}

var randomSalt = "randomSalt"

func (d *dummyL1FeeCalculator) CalculateL1Fee(_ context.Context, _ *valueobject.RouteSummary, _ string) (*big.Int, error) {
	return nil, nil
}

func TestBuildRouteUseCase_Handle(t *testing.T) {
	t.Parallel()

	recipient := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	sender := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c759bc2"
	amountIn := big.NewInt(20000)

	dummyL1FeeCalculator := &dummyL1FeeCalculator{}

	testCases := []struct {
		name    string
		prepare func(ctrl *gomock.Controller, config Config, wg *sync.WaitGroup) *BuildRouteUseCase
		command func() dto.BuildRouteCommand
		config  Config
		result  *dto.BuildRouteResult
		err     error
	}{
		{
			name: "it should return correct error when return error",
			prepare: func(ctrl *gomock.Controller, config Config, wg *sync.WaitGroup) *BuildRouteUseCase {
				clientDataEncoder := clientdata.NewMockIClientDataEncoder(ctrl)
				clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil).AnyTimes()

				encoder := mockEncode.NewMockIEncoder(ctrl)
				encoder.EXPECT().
					GetExecutorAddress(gomock.Any()).
					Return("0x00").AnyTimes()
				encoder.EXPECT().
					GetRouterAddress().
					Return("0x01").AnyTimes()

				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.SimplifiedToken{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						},
						nil,
					).AnyTimes()
				onchainPriceRepo := buildroute.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						map[string]*routerEntities.OnchainPrice{
							"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
								USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
							},
							"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": {
								USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
							}}, nil,
					).AnyTimes()

				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)

				executorBalanceRepository := buildroute.NewMockIExecutorBalanceRepository(ctrl)
				executorBalanceRepository.EXPECT().HasToken(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()
				executorBalanceRepository.EXPECT().HasPoolApproval(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()

				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Times(0)

				alphaFeeRepository := buildroute.NewMockIAlphaFeeRepository(ctrl)
				alphaFeeRepository.EXPECT().GetByRouteId(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

				publisherRepository := buildroute.NewMockIPublisherRepository(ctrl)
				publisherRepository.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				return NewBuildRouteUseCase(
					config,
					tokenRepository,
					poolRepository,
					executorBalanceRepository,
					onchainPriceRepo,
					alphaFeeRepository,
					nil,
					publisherRepository,
					gasEstimator,
					dummyL1FeeCalculator,
					nil,
					clientDataEncoder,
					encoder,
				)
			},
			command: func() dto.BuildRouteCommand {
				return dto.BuildRouteCommand{
					RouteSummary: &valueobject.RouteSummary{
						TokenIn:          "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						AmountIn:         big.NewInt(20000),
						AmountInUSD:      0,
						TokenOut:         "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
						AmountOut:        big.NewInt(0),
						AmountOutUSD:     0,
						Gas:              0,
						GasPrice:         big.NewFloat(100.2),
						GasUSD:           0,
						ExtraFee:         valueobject.ExtraFee{},
						Route:            [][]valueobject.Swap{},
						Timestamp:        time.Now().Unix(),
						OriginalChecksum: 12499967010441707798,
					},
					SlippageTolerance: 5,
					Recipient:         "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					Sender:            sender,
				}
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: false},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true},
				},
				PublisherConfig: PublisherConfig{
					AggregatorTransactionTopic: "aggregator-transaction",
				},
			},
			result: nil,
			err:    ErrCannotKeepDustTokenOut,
		},
		{
			name: "it should return correct amountOut when executor has tokenOut",
			prepare: func(ctrl *gomock.Controller, config Config, wg *sync.WaitGroup) *BuildRouteUseCase {
				clientDataEncoder := clientdata.NewMockIClientDataEncoder(ctrl)

				clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil).Times(2)

				encoder := mockEncode.NewMockIEncoder(ctrl)
				encodedData := "mockEncodedData"

				encoder.EXPECT().
					Encode(gomock.Any()).
					Return(encodedData, nil).Times(2)
				encoder.EXPECT().
					GetExecutorAddress(gomock.Any()).
					Return("0x00").AnyTimes()
				encoder.EXPECT().
					GetRouterAddress().
					Return("0x01").AnyTimes()

				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.SimplifiedToken{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						},
						nil,
					)
				onchainPriceRepo := buildroute.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						map[string]*routerEntities.OnchainPrice{
							"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
								USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
							},
							"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": {
								USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
							}}, nil,
					).AnyTimes()

				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)

				executorBalanceRepository := buildroute.NewMockIExecutorBalanceRepository(ctrl)
				executorBalanceRepository.EXPECT().HasToken(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{true}, nil).AnyTimes()
				executorBalanceRepository.EXPECT().HasPoolApproval(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()

				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				tx := NewUnsignedTransaction(
					sender,
					recipient,
					encodedData,
					constant.Zero,
					nil,
				)
				gasEstimator.EXPECT().EstimateGasAndPriceUSD(gomock.Any(), gomock.Eq(tx)).Times(1).Return(uint64(10),
					1.5, big.NewInt(1000), nil)

				alphaFeeRepository := buildroute.NewMockIAlphaFeeRepository(ctrl)
				alphaFeeRepository.EXPECT().GetByRouteId(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

				publisherRepository := buildroute.NewMockIPublisherRepository(ctrl)
				publisherRepository.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				return NewBuildRouteUseCase(
					config,
					tokenRepository,
					poolRepository,
					executorBalanceRepository,
					onchainPriceRepo,
					alphaFeeRepository,
					nil,
					publisherRepository,
					gasEstimator,
					dummyL1FeeCalculator,
					nil,
					clientDataEncoder,
					encoder,
				)
			},
			command: func() dto.BuildRouteCommand {
				return dto.BuildRouteCommand{
					RouteSummary: &valueobject.RouteSummary{
						TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						AmountIn:     big.NewInt(20000),
						AmountInUSD:  0,
						TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
						AmountOut:    big.NewInt(1000),
						AmountOutUSD: 0,
						Gas:          0,
						GasPrice:     big.NewFloat(100.2),
						GasUSD:       0,
						ExtraFee:     valueobject.ExtraFee{},
						Route: [][]valueobject.Swap{
							{
								{
									Pool:       "0xabc",
									SwapAmount: big.NewInt(20000),
									AmountOut:  big.NewInt(1000),
								},
							},
						},
					},
					SlippageTolerance:   5,
					Recipient:           recipient,
					Sender:              sender,
					EnableGasEstimation: true,
				}
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: false},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true},
				}},
			result: &dto.BuildRouteResult{
				AmountIn:         "20000",
				AmountInUSD:      "0.02",
				AmountOut:        "1000",
				AmountOutUSD:     "0.001",
				Gas:              "10",
				GasUSD:           "1.5",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "0",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			err: nil,
		},
		{
			name: "it should return correct amountOut when executor does not have tokenOut",
			prepare: func(ctrl *gomock.Controller, config Config, wg *sync.WaitGroup) *BuildRouteUseCase {
				clientDataEncoder := clientdata.NewMockIClientDataEncoder(ctrl)

				clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil).Times(2)

				encoder := mockEncode.NewMockIEncoder(ctrl)
				encodedData := "mockEncodedData"

				encoder.EXPECT().
					Encode(gomock.Any()).
					Return(encodedData, nil).Times(2)
				encoder.EXPECT().
					GetExecutorAddress(gomock.Any()).
					Return("0x00").AnyTimes()
				encoder.EXPECT().
					GetRouterAddress().
					Return("0x01").AnyTimes()

				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.SimplifiedToken{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						},
						nil,
					)
				onchainPriceRepo := buildroute.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						map[string]*routerEntities.OnchainPrice{
							"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
								USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
							},
							"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": {
								USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
							}}, nil,
					).AnyTimes()

				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)

				executorBalanceRepository := buildroute.NewMockIExecutorBalanceRepository(ctrl)
				executorBalanceRepository.EXPECT().HasToken(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()
				executorBalanceRepository.EXPECT().HasPoolApproval(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()

				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				tx := NewUnsignedTransaction(
					sender,
					recipient,
					encodedData,
					constant.Zero,
					nil,
				)
				gasEstimator.EXPECT().EstimateGasAndPriceUSD(gomock.Any(), gomock.Eq(tx)).Times(1).Return(uint64(10),
					1.5, big.NewInt(996), nil)

				alphaFeeRepository := buildroute.NewMockIAlphaFeeRepository(ctrl)
				alphaFeeRepository.EXPECT().GetByRouteId(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

				publisherRepository := buildroute.NewMockIPublisherRepository(ctrl)
				publisherRepository.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				return NewBuildRouteUseCase(
					config,
					tokenRepository,
					poolRepository,
					executorBalanceRepository,
					onchainPriceRepo,
					alphaFeeRepository,
					nil,
					publisherRepository,
					gasEstimator,
					dummyL1FeeCalculator,
					nil,
					clientDataEncoder,
					encoder,
				)
			},
			command: func() dto.BuildRouteCommand {
				return dto.BuildRouteCommand{
					RouteSummary: &valueobject.RouteSummary{
						TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						AmountIn:     big.NewInt(20000),
						AmountInUSD:  0,
						TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
						AmountOut:    big.NewInt(997),
						AmountOutUSD: 0,
						Gas:          0,
						GasPrice:     big.NewFloat(100.2),
						GasUSD:       0,
						ExtraFee:     valueobject.ExtraFee{},
						Route: [][]valueobject.Swap{
							{
								{
									Pool:       "0xabc",
									SwapAmount: big.NewInt(20000),
									AmountOut:  big.NewInt(1000),
								},
							},
						},
					},
					SlippageTolerance:   5,
					Recipient:           recipient,
					Sender:              sender,
					EnableGasEstimation: true,
				}
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: false},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true},
				}},
			result: &dto.BuildRouteResult{
				AmountIn:         "20000",
				AmountInUSD:      "0.02",
				AmountOut:        "996",
				AmountOutUSD:     "0.000996",
				Gas:              "10",
				GasUSD:           "1.5",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "0",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			err: nil,
		},
		{
			name: "it should return correct result and run estimate Gas when there is no error and Feature flag is on",
			prepare: func(ctrl *gomock.Controller, config Config, wg *sync.WaitGroup) *BuildRouteUseCase {
				clientDataEncoder := clientdata.NewMockIClientDataEncoder(ctrl)

				clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil).Times(2)

				encoder := mockEncode.NewMockIEncoder(ctrl)
				encodedData := "mockEncodedData"

				encoder.EXPECT().
					Encode(gomock.Any()).
					Return(encodedData, nil).Times(2)
				encoder.EXPECT().
					GetExecutorAddress(gomock.Any()).
					Return("0x00").AnyTimes()
				encoder.EXPECT().
					GetRouterAddress().
					Return("0x01").AnyTimes()

				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.SimplifiedToken{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						},
						nil,
					)

				onchainPriceRepo := buildroute.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						map[string]*routerEntities.OnchainPrice{
							"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
								USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
							},
							"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": {
								USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
							}}, nil,
					).AnyTimes()
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)

				executorBalanceRepository := buildroute.NewMockIExecutorBalanceRepository(ctrl)
				executorBalanceRepository.EXPECT().HasToken(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()
				executorBalanceRepository.EXPECT().HasPoolApproval(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()

				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				tx := NewUnsignedTransaction(
					sender,
					recipient,
					encodedData,
					constant.Zero,
					nil,
				)
				gasEstimator.EXPECT().EstimateGasAndPriceUSD(gomock.Any(), gomock.Eq(tx)).Times(1).Return(uint64(10),
					1.5, big.NewInt(9999), nil)

				alphaFeeRepository := buildroute.NewMockIAlphaFeeRepository(ctrl)
				alphaFeeRepository.EXPECT().GetByRouteId(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

				publisherRepository := buildroute.NewMockIPublisherRepository(ctrl)
				publisherRepository.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				return NewBuildRouteUseCase(
					config,
					tokenRepository,
					poolRepository,
					executorBalanceRepository,
					onchainPriceRepo,
					alphaFeeRepository,
					nil,
					publisherRepository,
					gasEstimator,
					dummyL1FeeCalculator,
					nil,
					clientDataEncoder,
					encoder,
				)
			},
			command: func() dto.BuildRouteCommand {
				return dto.BuildRouteCommand{
					RouteSummary: &valueobject.RouteSummary{
						TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						AmountIn:     big.NewInt(20000),
						AmountInUSD:  0,
						TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
						AmountOut:    big.NewInt(9999),
						AmountOutUSD: 0,
						Gas:          0,
						GasPrice:     big.NewFloat(100.2),
						GasUSD:       0,
						ExtraFee:     valueobject.ExtraFee{},
						Route: [][]valueobject.Swap{
							{
								{
									Pool:       "0xabc",
									SwapAmount: big.NewInt(20000),
									AmountOut:  big.NewInt(9999),
								},
							},
						},
					},
					SlippageTolerance:   5,
					Recipient:           recipient,
					Sender:              sender,
					EnableGasEstimation: true,
				}
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: false},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true},
				}},
			result: &dto.BuildRouteResult{
				AmountIn:         "20000",
				AmountInUSD:      "0.02",
				AmountOut:        "9998",
				AmountOutUSD:     "0.009998",
				Gas:              "10",
				GasUSD:           "1.5",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "0",

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

				clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil).Times(2)

				encoder := mockEncode.NewMockIEncoder(ctrl)
				encodedData := "mockEncodedData"

				encoder.EXPECT().
					Encode(gomock.Any()).
					Return(encodedData, nil).Times(2)
				encoder.EXPECT().
					GetExecutorAddress(gomock.Any()).
					Return("0x00").AnyTimes()
				encoder.EXPECT().
					GetRouterAddress().
					Return("0x01").AnyTimes()

				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.SimplifiedToken{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						},
						nil,
					)
				tokenRepository.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).
					Return(
						[]*routerEntities.TokenInfo{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", IsFOT: false, IsHoneypot: false},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", IsFOT: false, IsHoneypot: false},
						},
						nil,
					)

				onchainPriceRepo := buildroute.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						map[string]*routerEntities.OnchainPrice{
							"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
								USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
							},
							"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": {
								USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
							}}, nil,
					).AnyTimes()
				wg.Add(1)
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return([]string{}, nil).Times(1)

				executorBalanceRepository := buildroute.NewMockIExecutorBalanceRepository(ctrl)
				executorBalanceRepository.EXPECT().HasToken(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()
				executorBalanceRepository.EXPECT().HasPoolApproval(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()

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
				}).Return(uint64(10), big.NewInt(9999), nil)

				alphaFeeRepository := buildroute.NewMockIAlphaFeeRepository(ctrl)
				alphaFeeRepository.EXPECT().GetByRouteId(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

				publisherRepository := buildroute.NewMockIPublisherRepository(ctrl)
				publisherRepository.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				return NewBuildRouteUseCase(
					config,
					tokenRepository,
					poolRepository,
					executorBalanceRepository,
					onchainPriceRepo,
					alphaFeeRepository,
					nil,
					publisherRepository,
					gasEstimator,
					dummyL1FeeCalculator,
					nil,
					clientDataEncoder,
					encoder,
				)
			},
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(20000),
					AmountInUSD:  0,
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(10000),
					AmountOutUSD: 0,
					Gas:          15,
					GasPrice:     big.NewFloat(100.2),
					GasUSD:       100,
					ExtraFee:     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(10000),
								SwapAmount: big.NewInt(1000000),
							},
						},
					},
					Timestamp: time.Now().Unix(),
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:      route,
					SlippageTolerance: 5,
					Recipient:         recipient,
					Sender:            sender,
				}
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true},
				},
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup:      make(map[string]bool),
					CorrelatedGroup1: make(map[string]bool),
					CorrelatedGroup2: make(map[string]bool),
					CorrelatedGroup3: make(map[string]bool),
				},
				Salt: randomSalt,
			},
			result: &dto.BuildRouteResult{
				AmountIn:         "20000",
				AmountInUSD:      "0.02",
				AmountOut:        "9999",
				AmountOutUSD:     "0.009999",
				Gas:              "15",
				GasUSD:           "100",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "0",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			err: nil,
		},
		{
			name: "it should return correct result and run estimate Gas when there is no error and Feature flag is on with token in is Ether",
			prepare: func(ctrl *gomock.Controller, config Config, wg *sync.WaitGroup) *BuildRouteUseCase {
				clientDataEncoder := clientdata.NewMockIClientDataEncoder(ctrl)

				clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil).Times(2)

				encoder := mockEncode.NewMockIEncoder(ctrl)
				encodedData := "mockEncodedData"

				encoder.EXPECT().
					Encode(gomock.Any()).
					Return(encodedData, nil).Times(2)
				encoder.EXPECT().
					GetExecutorAddress(gomock.Any()).
					Return("0x00").AnyTimes()
				encoder.EXPECT().
					GetRouterAddress().
					Return("0x01").AnyTimes()

				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.SimplifiedToken{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						},
						nil,
					)
				tokenRepository.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).
					Return(
						[]*routerEntities.TokenInfo{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", IsFOT: false, IsHoneypot: false},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", IsFOT: false, IsHoneypot: false},
						},
						nil,
					)

				onchainPriceRepo := buildroute.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						map[string]*routerEntities.OnchainPrice{
							"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
								USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
							},
							"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": {
								USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
							}}, nil,
					).AnyTimes()
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				wg.Add(1)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return([]string{}, nil).Times(1)

				executorBalanceRepository := buildroute.NewMockIExecutorBalanceRepository(ctrl)
				executorBalanceRepository.EXPECT().HasToken(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()
				executorBalanceRepository.EXPECT().HasPoolApproval(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()

				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				tx := NewUnsignedTransaction(
					sender,
					recipient,
					encodedData,
					amountIn,
					nil,
				)
				gasEstimator.EXPECT().EstimateGasAndPriceUSD(gomock.Any(), gomock.Eq(tx)).Times(1).Return(uint64(10),
					1.5, big.NewInt(9999), nil)

				alphaFeeRepository := buildroute.NewMockIAlphaFeeRepository(ctrl)
				alphaFeeRepository.EXPECT().GetByRouteId(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

				publisherRepository := buildroute.NewMockIPublisherRepository(ctrl)
				publisherRepository.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				return NewBuildRouteUseCase(
					config,
					tokenRepository,
					poolRepository,
					executorBalanceRepository,
					onchainPriceRepo,
					alphaFeeRepository,
					nil,
					publisherRepository,
					gasEstimator,
					dummyL1FeeCalculator,
					nil,
					clientDataEncoder,
					encoder,
				)
			},
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      valueobject.EtherAddress,
					AmountIn:     amountIn,
					AmountInUSD:  0,
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(10000),
					AmountOutUSD: 0,
					Gas:          0,
					GasPrice:     big.NewFloat(100.2),
					GasUSD:       0,
					ExtraFee:     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(10000),
								SwapAmount: big.NewInt(100000),
							},
						},
					},
					Timestamp: time.Now().Unix(),
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					SlippageTolerance:   5,
					Recipient:           recipient,
					Sender:              sender,
					EnableGasEstimation: true,
				}
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true},
				},
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup:      make(map[string]bool),
					CorrelatedGroup1: make(map[string]bool),
					CorrelatedGroup2: make(map[string]bool),
					CorrelatedGroup3: make(map[string]bool),
				},
				Salt: randomSalt,
			},
			result: &dto.BuildRouteResult{
				AmountIn:         "20000",
				AmountInUSD:      "0.02",
				AmountOut:        "9999",
				AmountOutUSD:     "0.009999",
				Gas:              "10",
				GasUSD:           "1.5",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "20000",

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
			result, err := uc.Handle(context.Background(), tc.command())
			wg.Wait()

			assert.Equal(t, tc.result, result)
			if tc.err != nil {
				assert.Equal(t, tc.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}

}

func TestBuildRouteUseCase_HandleWithGasEstimation(t *testing.T) {
	t.Parallel()

	recipient := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	sender := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756bc2"

	testCases := []struct {
		name            string
		command         func() dto.BuildRouteCommand
		estimateGas     func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator
		poolRepository  func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository
		tokenRepository func(ctrl *gomock.Controller) *buildroute.MockITokenRepository
		result          *dto.BuildRouteResult
		config          Config
		err             error
	}{
		{
			name: "it should return correct result and run estimate Gas when there is no error, feature flag is on",
			command: func() dto.BuildRouteCommand {
				return dto.BuildRouteCommand{
					RouteSummary: &valueobject.RouteSummary{
						TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						AmountIn:     big.NewInt(20000),
						AmountInUSD:  0,
						TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
						AmountOut:    big.NewInt(10000),
						AmountOutUSD: 0,
						Gas:          12,
						GasPrice:     big.NewFloat(100.2),
						GasUSD:       1.5,
						ExtraFee:     valueobject.ExtraFee{},
						Route: [][]valueobject.Swap{
							{
								{
									Pool:       "0xabc",
									AmountOut:  big.NewInt(10000),
									SwapAmount: big.NewInt(10000),
									TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
									TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
								},
							},
						},
					},
					SlippageTolerance:   5,
					Recipient:           recipient,
					EnableGasEstimation: true,
					Sender:              sender,
				}
			},
			result: &dto.BuildRouteResult{
				AmountIn:         "20000",
				AmountInUSD:      "0.02",
				AmountOut:        "9999",
				AmountOutUSD:     "0.009999",
				Gas:              "1234",
				GasUSD:           "1.5",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "0",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			estimateGas: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().EstimateGasAndPriceUSD(gomock.Any(), gomock.Any()).Return(uint64(1234), 1.5, big.NewInt(9999), nil).Times(1)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)

				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller) *buildroute.MockITokenRepository {
				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.SimplifiedToken{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
							{Address: "0x6b175474e89094c44da98b954eedeac495271d0f", Decimals: 18},
						},
						nil,
					)
				tokenRepository.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Times(0)
				return tokenRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true},
				}},
			err: nil,
		},
		{
			name: "it should return correct result and disable run estimate Gas when there is no error, feature flag is on",
			command: func() dto.BuildRouteCommand {
				return dto.BuildRouteCommand{
					RouteSummary: &valueobject.RouteSummary{
						TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						AmountIn:     big.NewInt(20000),
						AmountInUSD:  0,
						TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
						AmountOut:    big.NewInt(10000),
						AmountOutUSD: 0,
						Gas:          12,
						GasPrice:     big.NewFloat(100.2),
						GasUSD:       0,
						ExtraFee:     valueobject.ExtraFee{},
						Route: [][]valueobject.Swap{
							{
								{
									Pool:       "0xabc",
									SwapAmount: big.NewInt(20000),
									AmountOut:  big.NewInt(10000),
									TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
									TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
								},
							},
						},
					},
					SlippageTolerance:   5,
					Recipient:           recipient,
					EnableGasEstimation: false,
				}
			},
			result: &dto.BuildRouteResult{
				AmountIn:         "20000",
				AmountInUSD:      "0.02",
				AmountOut:        "9999",
				AmountOutUSD:     "0.009999",
				Gas:              "12",
				GasUSD:           "0",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "0",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			estimateGas: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().EstimateGasAndPriceUSD(gomock.Any(), gomock.Any()).Times(0)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)

				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller) *buildroute.MockITokenRepository {
				tokenRepo := buildroute.NewMockITokenRepository(ctrl)
				tokenRepo.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.SimplifiedToken{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						},
						nil,
					).AnyTimes()
				tokenRepo.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Times(0)

				return tokenRepo
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true},
				}},
			err: nil,
		},
		{
			name: "it should return correct result and run estimate Gas in goroutine when there is no error because feature flag is on but disable estimateGas",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(20000),
					AmountInUSD:  0,
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(10000),
					AmountOutUSD: 0,
					Gas:          7,
					GasPrice:     big.NewFloat(100.2),
					GasUSD:       0,
					ExtraFee:     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(10000),
								SwapAmount: big.NewInt(1000),
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
							},
						},
					},
					Timestamp: time.Now().Unix(),
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					SlippageTolerance:   5,
					Recipient:           recipient,
					Sender:              "0xabc",
					EnableGasEstimation: false,
				}
			},
			result: &dto.BuildRouteResult{
				AmountIn:              "20000",
				AmountInUSD:           "0.02",
				AmountOut:             "9999",
				AmountOutUSD:          "0.009999",
				Gas:                   "7",
				GasUSD:                "0",
				OutputChange:          OutputChangeNoChange,
				Data:                  "mockEncodedData",
				RouterAddress:         "0x01",
				TransactionValue:      "0",
				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			estimateGas: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				wg.Add(1)
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Times(1).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(uint64(1), big.NewInt(10000), nil)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				counters := []routerEntities.FaultyPoolTracker{
					{Address: "0xabc", TotalCount: 1, FailedCount: 0, Tokens: []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab"}},
				}
				wg.Add(1)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Eq(counters)).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return([]string{"0xabc"}, nil).Times(1)

				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller) *buildroute.MockITokenRepository {
				tokenRepo := buildroute.NewMockITokenRepository(ctrl)
				tokenRepo.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.SimplifiedToken{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						},
						nil,
					).AnyTimes()
				tokenRepo.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Times(0)

				return tokenRepo
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true},
				},
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup:      make(map[string]bool),
					CorrelatedGroup1: make(map[string]bool),
					CorrelatedGroup2: make(map[string]bool),
					CorrelatedGroup3: make(map[string]bool),
				},
				Salt: randomSalt,
			},
			err: nil,
		},
		{
			name: "it should return error when EnableGasEstimation is true and sender is empty, feature flag is on",
			command: func() dto.BuildRouteCommand {
				return dto.BuildRouteCommand{
					RouteSummary: &valueobject.RouteSummary{
						TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						AmountIn:     big.NewInt(20000),
						AmountInUSD:  0,
						TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
						AmountOut:    big.NewInt(10000),
						AmountOutUSD: 0,
						Gas:          12,
						GasPrice:     big.NewFloat(100.2),
						GasUSD:       0,
						ExtraFee:     valueobject.ExtraFee{},
						Route: [][]valueobject.Swap{
							{
								{
									Pool:       "0xabc",
									AmountOut:  big.NewInt(10000),
									SwapAmount: big.NewInt(20000),
									TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
									TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
								},
							},
						},
					},
					SlippageTolerance:   5,
					Recipient:           recipient,
					EnableGasEstimation: true,
				}
			},
			result: nil,
			estimateGas: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().EstimateGasAndPriceUSD(gomock.Any(), gomock.Any()).Times(0)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)

				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller) *buildroute.MockITokenRepository {
				tokenRepo := buildroute.NewMockITokenRepository(ctrl)
				tokenRepo.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.SimplifiedToken{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
							{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						},
						nil,
					).AnyTimes()
				tokenRepo.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Times(0)

				return tokenRepo
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true},
				},
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup:      make(map[string]bool),
					CorrelatedGroup1: make(map[string]bool),
					CorrelatedGroup2: make(map[string]bool),
					CorrelatedGroup3: make(map[string]bool),
				},
				Salt: randomSalt,
			},
			err: ErrSenderEmptyWhenEnableEstimateGas,
		},
		{
			name: "it should count faulty pools when estimate gas error is return amount not enough, feature flag is on",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
					AmountIn:     big.NewInt(500000),
					AmountInUSD:  0.00000000192722,
					TokenOut:     "0x6b175474e89094c44da98b954eedeac495271d0f",
					AmountOut:    big.NewInt(1626105316),
					AmountOutUSD: 0.000000001626105316,
					Gas:          185000,
					GasPrice:     big.NewFloat(9511845152),
					GasUSD:       6.782624739119853,
					ExtraFee:     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xa478c2975ab1ea89e8196811f51a7b7ade33eb11",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "0x6b175474e89094c44da98b954eedeac495271d0f",
								AmountOut:  big.NewInt(1626105316),
								SwapAmount: big.NewInt(1626105316),
							},
						},
					},
					Timestamp: time.Now().Unix(),
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					Sender:              sender,
					SlippageTolerance:   1,
					Recipient:           recipient,
					EnableGasEstimation: true,
				}
			},
			estimateGas: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().EstimateGasAndPriceUSD(gomock.Any(), gomock.Any()).Times(1).Return(uint64(185000),
					0.07912413535198341, big.NewInt(1526105316), nil)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)

				counters := []routerEntities.FaultyPoolTracker{
					{Address: "0xa478c2975ab1ea89e8196811f51a7b7ade33eb11", TotalCount: 1, FailedCount: 1, Tokens: []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x6b175474e89094c44da98b954eedeac495271d0f"}},
				}
				wg.Add(1)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Eq(counters)).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return([]string{"0xa478c2975ab1ea89e8196811f51a7b7ade33eb11"}, nil).Times(1)

				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller) *buildroute.MockITokenRepository {
				tokenRepo := buildroute.NewMockITokenRepository(ctrl)
				tokenRepo.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.SimplifiedToken{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
							{Address: "0x6b175474e89094c44da98b954eedeac495271d0f", Decimals: 18},
						},
						nil,
					).AnyTimes()
				tokenRepo.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Times(0)

				return tokenRepo
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				Salt:         randomSalt,
				TokenGroups: &valueobject.TokenGroupConfig{
					CorrelatedGroup1: map[string]bool{
						"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true,
						"0x6b175474e89094c44da98b954eedeac495271d0f": true,
					},
				},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0x6b175474e89094c44da98b954eedeac495271d0f": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 500,
						},
					},
				},
			},
			result: &dto.BuildRouteResult{
				SuggestedSlippage: 615 + 5, // 0.05% buffer for correlated
			},
			err: ErrEstimateGasFailed(ErrReturnAmountIsNotEnough),
		},
		{
			name: "it should not count faulty pools when estimate gas error is some error, feature flag is on",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
					AmountIn:     big.NewInt(500000),
					AmountInUSD:  0.00000000192722,
					TokenOut:     "0x6b175474e89094c44da98b954eedeac495271d0f",
					AmountOut:    big.NewInt(1626105316),
					AmountOutUSD: 0.000000001626105316,
					Gas:          185000,
					GasPrice:     big.NewFloat(9511845152),
					GasUSD:       6.782624739119853,
					ExtraFee:     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xa478c2975ab1ea89e8196811f51a7b7ade33eb11",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "0x6b175474e89094c44da98b954eedeac495271d0f",
								AmountOut:  big.NewInt(1626105316),
								SwapAmount: big.NewInt(1626105316),
							},
						},
					},
					Timestamp: time.Now().Unix(),
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:      route,
					Sender:            sender,
					SlippageTolerance: 5,
					Recipient:         recipient,
				}
			},
			result: &dto.BuildRouteResult{
				AmountIn:         "500000",
				AmountInUSD:      "0.5",
				AmountOut:        "1626105315",
				AmountOutUSD:     "0.000000001626105315",
				Gas:              "185000",
				GasUSD:           "6.782624739119853",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "500000",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			estimateGas: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				wg.Add(1)
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Do(func(arg0, arg2 interface{}) {
					defer wg.Done()
				}).Times(1).Return(uint64(0), big.NewInt(0), errors.New("test error"))
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller) *buildroute.MockITokenRepository {
				tokenRepo := buildroute.NewMockITokenRepository(ctrl)
				tokenRepo.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.SimplifiedToken{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
							{Address: "0x6b175474e89094c44da98b954eedeac495271d0f", Decimals: 18},
						},
						nil,
					).AnyTimes()
				tokenRepo.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Times(0)

				return tokenRepo
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: false, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0x6b175474e89094c44da98b954eedeac495271d0f": true, "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee": true},
				},
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup:      make(map[string]bool),
					CorrelatedGroup1: make(map[string]bool),
					CorrelatedGroup2: make(map[string]bool),
					CorrelatedGroup3: make(map[string]bool),
				},
				Salt: randomSalt,
			},
			err: nil,
		},
		{
			name: "it should not count faulty pools and still call estimate gas, feature flag is off",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
					AmountIn:     big.NewInt(500000),
					AmountInUSD:  0.5,
					TokenOut:     "0x6b175474e89094c44da98b954eedeac495271d0f",
					AmountOut:    big.NewInt(1626105316),
					AmountOutUSD: 0.000000001626105316,
					Gas:          185000,
					GasPrice:     big.NewFloat(9511845152),
					GasUSD:       6.782624739119853,
					ExtraFee:     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xa478c2975ab1ea89e8196811f51a7b7ade33eb11",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "0x6b175474e89094c44da98b954eedeac495271d0f",
								AmountOut:  big.NewInt(1626105316),
								SwapAmount: big.NewInt(1111111),
							},
						},
					},
					Timestamp: time.Now().Unix(),
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					Sender:              sender,
					SlippageTolerance:   5,
					Recipient:           recipient,
					EnableGasEstimation: true,
				}
			},
			result: &dto.BuildRouteResult{
				AmountIn:         "500000",
				AmountInUSD:      "0.5",
				AmountOut:        "1626105315",
				AmountOutUSD:     "0.000000001626105315",
				Gas:              "185000",
				GasUSD:           "6.782624739119853",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "500000",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			estimateGas: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().EstimateGasAndPriceUSD(gomock.Any(), gomock.Any()).Times(0)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)

				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller) *buildroute.MockITokenRepository {
				tokenRepo := buildroute.NewMockITokenRepository(ctrl)
				tokenRepo.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Return(
						[]*entity.SimplifiedToken{
							{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
							{Address: "0x6b175474e89094c44da98b954eedeac495271d0f", Decimals: 18},
						},
						nil,
					).AnyTimes()
				tokenRepo.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Times(0)

				return tokenRepo
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: false, IsFaultyPoolDetectorEnable: false},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true},
				},
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup:      make(map[string]bool),
					CorrelatedGroup1: make(map[string]bool),
					CorrelatedGroup2: make(map[string]bool),
					CorrelatedGroup3: make(map[string]bool),
				},
				Salt: randomSalt,
			},
			err: nil,
		},
	}

	wg := sync.WaitGroup{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			clientDataEncoder := clientdata.NewMockIClientDataEncoder(ctrl)
			clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil).AnyTimes()

			encoder := mockEncode.NewMockIEncoder(ctrl)
			encodedData := "mockEncodedData"

			encoder.EXPECT().
				Encode(gomock.Any()).
				Return(encodedData, nil).AnyTimes()
			encoder.EXPECT().
				GetExecutorAddress(gomock.Any()).
				Return("0x00").AnyTimes()
			encoder.EXPECT().
				GetRouterAddress().
				Return("0x01").AnyTimes()

			tokenRepository := tc.tokenRepository(ctrl)

			onchainpriceRepo := buildroute.NewMockIOnchainPriceRepository(ctrl)
			onchainpriceRepo.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).
				Return(
					map[string]*routerEntities.OnchainPrice{
						"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
							USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
						},
						"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": {
							USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
						},
						"0x6b175474e89094c44da98b954eedeac495271d0f": {
							USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
						}}, nil,
				).AnyTimes()

			executorBalanceRepository := buildroute.NewMockIExecutorBalanceRepository(ctrl)
			executorBalanceRepository.EXPECT().HasToken(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()
			executorBalanceRepository.EXPECT().HasPoolApproval(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()

			gasEstimator := tc.estimateGas(ctrl, &wg)
			poolRepository := tc.poolRepository(ctrl, &wg)

			alphaFeeRepository := buildroute.NewMockIAlphaFeeRepository(ctrl)
			alphaFeeRepository.EXPECT().GetByRouteId(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

			publisherRepository := buildroute.NewMockIPublisherRepository(ctrl)
			publisherRepository.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			usecase := NewBuildRouteUseCase(
				tc.config,
				tokenRepository,
				poolRepository,
				executorBalanceRepository,
				onchainpriceRepo,
				alphaFeeRepository,
				nil,
				publisherRepository,
				gasEstimator,
				&dummyL1FeeCalculator{},
				nil,
				clientDataEncoder,
				encoder,
			)

			result, err := usecase.Handle(context.Background(), tc.command())
			wg.Wait()

			assert.Equal(t, tc.result, result)
			if tc.err != nil {
				assert.Equal(t, tc.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildRouteUseCase_HandleWithTrackingFaultyPools(t *testing.T) {
	t.Parallel()

	recipient := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	sender := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756bc2"

	testCases := []struct {
		name            string
		command         func() dto.BuildRouteCommand
		gasEstimator    func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator
		poolRepository  func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository
		tokenRepository func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockITokenRepository
		result          *dto.BuildRouteResult
		config          Config
		err             error
	}{
		{
			name: "it should return correct result and increase total count (failed count is 0) on Redis, check FOT on whitelist tokens",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(2000000000000000000),
					AmountInUSD:  float64(2000000000000000000),
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(4488767370609711072),
					AmountOutUSD: float64(4488767370609711072),
					Gas:          345000,
					GasPrice:     big.NewFloat(100000000),
					GasUSD:       0.07912413535198341,
					ExtraFee:     valueobject.ExtraFee{},
					Timestamp:    time.Now().Unix(),
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(996023110963288),
								SwapAmount: big.NewInt(2000000000000000000),
								Exchange:   "pancake",
								PoolType:   "uniswap-v2",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "wlToken1",
							},
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(4488767370609711072),
								SwapAmount: big.NewInt(996023110963288),
								Exchange:   "smardex",
								PoolType:   "smardex",
								TokenIn:    "wlToken1",
								TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
							},
						},
					},
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					SlippageTolerance:   5,
					Recipient:           recipient,
					EnableGasEstimation: true,
					Sender:              sender,
				}
			},
			result: &dto.BuildRouteResult{
				AmountIn:         "2000000000000000000",
				AmountInUSD:      "2000000000000",
				AmountOut:        "4488767370609711071",
				AmountOutUSD:     "4488767370609.711",
				Gas:              "345000",
				GasUSD:           "0.07912413535198341",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "0",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			gasEstimator: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().EstimateGasAndPriceUSD(gomock.Any(), gomock.Any()).Return(uint64(345000),
					0.07912413535198341, big.NewInt(4488767370609711072), nil).Times(1)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				wg.Add(1)
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				counters := []routerEntities.FaultyPoolTracker{
					{
						Address:     "0xabc",
						TotalCount:  1,
						FailedCount: 0,
						Tokens:      []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "wlToken1"},
					},
					{
						Address:     "0xabc",
						TotalCount:  1,
						FailedCount: 0,
						Tokens:      []string{"wlToken1", "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab"},
					},
				}
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Eq(counters)).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return([]string{"0xabc"}, nil).Times(1)
				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockITokenRepository {
				wg.Add(2)
				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(
					[]*entity.SimplifiedToken{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
					},
					nil,
				)
				tokenRepository.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(
					[]*routerEntities.TokenInfo{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", IsFOT: false, IsHoneypot: false},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", IsFOT: false, IsHoneypot: false},
					},
					nil,
				)
				return tokenRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": false, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": false, "wlToken1": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 500,
						},
					},
				},
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup:      make(map[string]bool),
					CorrelatedGroup1: make(map[string]bool),
					CorrelatedGroup2: make(map[string]bool),
					CorrelatedGroup3: make(map[string]bool),
				},
				Salt: randomSalt,
			},
			err: nil,
		},
		{
			name: "it should return correct result although increase total count on Redis failed, route contains all whitelist tokens, no need to check fot tokens",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(2000000000000000000),
					AmountInUSD:  float64(2000000000000000000),
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(4488767370609711072),
					AmountOutUSD: float64(4488767370609711072),
					Gas:          345000,
					GasPrice:     big.NewFloat(100000000),
					GasUSD:       0.07912413535198341,
					ExtraFee:     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(996023110963288),
								SwapAmount: big.NewInt(2000000000000000000),
								Exchange:   "pancake",
								PoolType:   "uniswap-v2",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2",
							},
							{
								Pool:       "0xabcd",
								AmountOut:  big.NewInt(4488767370609711072),
								SwapAmount: big.NewInt(996023110963288),
								Exchange:   "smardex",
								PoolType:   "smardex",
								TokenIn:    "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2",
								TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
							},
						},
					},
					Timestamp: time.Now().Unix(),
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					SlippageTolerance:   5,
					Recipient:           recipient,
					EnableGasEstimation: false,
					Sender:              sender,
				}
			},
			result: &dto.BuildRouteResult{
				AmountIn:         "2000000000000000000",
				AmountInUSD:      "2000000000000",
				AmountOut:        "4488767370609711071",
				AmountOutUSD:     "4488767370609.711",
				Gas:              "345000",
				GasUSD:           "0.07912413535198341",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "0",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			gasEstimator: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				wg.Add(1)
				// EnableGasEstimation = false so estimate gas in background
				gasEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(uint64(0), big.NewInt(0), ErrReturnAmountIsNotEnough)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockITokenRepository {
				wg.Add(1)
				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(
					[]*entity.SimplifiedToken{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
						{Address: "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2", Decimals: 6},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
					},
					nil,
				)
				tokenRepository.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Times(0)
				return tokenRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 500,
						},
					},
				},
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup:      make(map[string]bool),
					CorrelatedGroup1: make(map[string]bool),
					CorrelatedGroup2: make(map[string]bool),
					CorrelatedGroup3: make(map[string]bool),
				},
				Salt: randomSalt,
			},
			err: nil,
		},
		{
			name: "it should return correct result and not increase total count on Redis when feature flag is off",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(2000000000000000000),
					AmountInUSD:  float64(2000000000000000000),
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(4488767370609711072),
					AmountOutUSD: float64(4488767370609711072),
					Gas:          345000,
					GasPrice:     big.NewFloat(100000000),
					GasUSD:       0.07912413535198341,
					ExtraFee:     valueobject.ExtraFee{},
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
					Timestamp: time.Now().Unix(),
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					SlippageTolerance:   5,
					Recipient:           recipient,
					EnableGasEstimation: true,
					Sender:              sender,
				}
			},
			result: &dto.BuildRouteResult{
				AmountIn:         "2000000000000000000",
				AmountInUSD:      "2000000000000",
				AmountOut:        "4488767370609711071",
				AmountOutUSD:     "4488767370609.711",
				Gas:              "345000",
				GasUSD:           "0.07912413535198341",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "0",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			gasEstimator: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().EstimateGasAndPriceUSD(gomock.Any(), gomock.Any()).Times(0)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)
				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockITokenRepository {
				wg.Add(1)
				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(
					[]*entity.SimplifiedToken{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
					},
					nil,
				)
				tokenRepository.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Times(0)
				return tokenRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: false, IsFaultyPoolDetectorEnable: false},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 500,
						},
					},
				},
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup:      make(map[string]bool),
					CorrelatedGroup1: make(map[string]bool),
					CorrelatedGroup2: make(map[string]bool),
					CorrelatedGroup3: make(map[string]bool),
				},
				Salt: randomSalt,
			},
			err: nil,
		},
		{
			name: "it should return correct result, but increase total count only (failed count = 0) because slippage below min threshold",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(2000000000000000000),
					AmountInUSD:  float64(2000000000000000000),
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(4488767370609711072),
					AmountOutUSD: float64(4488767370609711072),
					Gas:          345000,
					GasPrice:     big.NewFloat(100000000),
					GasUSD:       0.07912413535198341,
					ExtraFee:     valueobject.ExtraFee{},
					Timestamp:    time.Now().Unix(),
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(996023110963288),
								SwapAmount: big.NewInt(2000000000000000000),
								Exchange:   "pancake",
								PoolType:   "uniswap-v2",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "wlToken1",
							},
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(4488767370609711072),
								SwapAmount: big.NewInt(996023110963288),
								Exchange:   "smardex",
								PoolType:   "smardex",
								TokenIn:    "wlToken1",
								TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
							},
						},
					},
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					SlippageTolerance:   30,
					Recipient:           recipient,
					EnableGasEstimation: true,
					Sender:              sender,
				}
			},
			result: &dto.BuildRouteResult{
				AmountIn:         "2000000000000000000",
				AmountInUSD:      "2000000000000",
				AmountOut:        "4488767370609711071",
				AmountOutUSD:     "4488767370609.711",
				Gas:              "345000",
				GasUSD:           "0.07912413535198341",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "0",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			gasEstimator: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				wg.Add(1)
				// config.IsGasEstimatorEnabled = false so estimate gas in background
				gasEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(uint64(345000), big.NewInt(4388767370609711071), nil)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				counters := []routerEntities.FaultyPoolTracker{
					{
						Address:     "0xabc",
						TotalCount:  1,
						FailedCount: 0,
						Tokens:      []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "wlToken1"},
					},
					{
						Address:     "0xabc",
						TotalCount:  1,
						FailedCount: 0,
						Tokens:      []string{"wlToken1", "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab"},
					},
				}
				wg.Add(1)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Eq(counters)).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return([]string{"0xabc"}, nil).Times(1)
				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockITokenRepository {
				wg.Add(1)
				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(
					[]*entity.SimplifiedToken{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						{Address: "wlToken1", Decimals: 6},
					},
					nil,
				)
				tokenRepository.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Times(0)
				return tokenRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: false, IsFaultyPoolDetectorEnable: true},
				Salt:         randomSalt,
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup:      make(map[string]bool),
					CorrelatedGroup1: make(map[string]bool),
					CorrelatedGroup2: make(map[string]bool),
					CorrelatedGroup3: make(map[string]bool),
				},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true, "wlToken1": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 274, // suggested slippage = 273 so failed_count = 0
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "it should return correct result and increase total count on Redis because slippage above min threshold",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(2000000000000000000),
					AmountInUSD:  float64(2000000000000000000),
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(4488767370609711072),
					AmountOutUSD: float64(4488767370609711072),
					Gas:          345000,
					GasPrice:     big.NewFloat(100000000),
					GasUSD:       0.07912413535198341,
					ExtraFee:     valueobject.ExtraFee{},
					Timestamp:    time.Now().Unix(),
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(996023110963288),
								SwapAmount: big.NewInt(2000000000000000000),
								Exchange:   "pancake",
								PoolType:   "uniswap-v2",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "wlToken1",
							},
							{
								Pool:       "0xabcd",
								AmountOut:  big.NewInt(4488767370609711072),
								SwapAmount: big.NewInt(996023110963288),
								Exchange:   "smardex",
								PoolType:   "smardex",
								TokenIn:    "wlToken1",
								TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
							},
						},
					}}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					SlippageTolerance:   50,
					Recipient:           recipient,
					EnableGasEstimation: true,
					Sender:              sender,
				}
			},
			result: &dto.BuildRouteResult{
				AmountIn:         "2000000000000000000",
				AmountInUSD:      "2000000000000",
				AmountOut:        "4488767370609711071",
				AmountOutUSD:     "4488767370609.711",
				Gas:              "345000",
				GasUSD:           "0.07912413535198341",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "0",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			gasEstimator: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				wg.Add(1)
				// config.IsGasEstimatorEnabled = false so estimate gas in background
				gasEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(uint64(345000), big.NewInt(60), nil)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				counters := []routerEntities.FaultyPoolTracker{
					{
						Address:     "0xabc",
						TotalCount:  1,
						FailedCount: 1,
						Tokens:      []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "wlToken1"},
					},
					{
						Address:     "0xabcd",
						TotalCount:  1,
						FailedCount: 1,
						Tokens:      []string{"wlToken1", "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab"},
					},
				}
				wg.Add(1)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Eq(counters)).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Times(1).Return([]string{"0xabc", "0xabcd"}, nil)
				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockITokenRepository {
				wg.Add(1)
				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(
					[]*entity.SimplifiedToken{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						{Address: "wlToken1", Decimals: 6},
					},
					nil,
				)
				tokenRepository.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Times(0)
				return tokenRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: false, IsFaultyPoolDetectorEnable: true},
				Salt:         randomSalt,
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup: map[string]bool{
						"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true,
						"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true,
					},
				},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true, "wlToken1": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 40,
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "it should return correct result and not increase total count on Redis because token out is FOT token",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(2000000000000000000),
					AmountInUSD:  float64(2000000000000000000),
					TokenOut:     "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2",
					AmountOut:    big.NewInt(4488767370609711072),
					AmountOutUSD: float64(4488767370609711072),
					Gas:          345000,
					GasPrice:     big.NewFloat(100000000),
					GasUSD:       0.07912413535198341,
					ExtraFee:     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(996023110963288),
								SwapAmount: big.NewInt(2000000000000000000),
								Exchange:   "pancake",
								PoolType:   "uniswap-v2",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "wlToken1",
							},
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(4488767370609711072),
								SwapAmount: big.NewInt(996023110963288),
								Exchange:   "smardex",
								PoolType:   "smardex",
								TokenIn:    "wlToken1",
								TokenOut:   "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2",
							},
						},
					},
					Timestamp: time.Now().Unix(),
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					SlippageTolerance:   5,
					Recipient:           recipient,
					EnableGasEstimation: true,
					Sender:              sender,
				}
			},
			result: &dto.BuildRouteResult{
				AmountIn:         "2000000000000000000",
				AmountInUSD:      "2000000000000",
				AmountOut:        "4488767370609711071",
				AmountOutUSD:     "4488767370609.711",
				Gas:              "345000",
				GasUSD:           "0.07912413535198341",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "0",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			gasEstimator: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				wg.Add(1)
				gasEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(uint64(345000), big.NewInt(4388767370609711071), nil)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)
				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockITokenRepository {
				wg.Add(2)
				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						defer wg.Done()
					}).Return(
					[]*entity.SimplifiedToken{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
						{Address: "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2", Decimals: 6},
						{Address: "wlToken1", Decimals: 6},
					},
					nil,
				)
				tokenRepository.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(
					[]*routerEntities.TokenInfo{
						{Address: "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2", IsFOT: true, IsHoneypot: false},
					},
					nil,
				)
				return tokenRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: false, IsFaultyPoolDetectorEnable: true},
				Salt:         randomSalt,
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup: map[string]bool{
						"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true,
						"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true,
					},
				},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true, "wlToken1": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 0,
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "it should return correct result and only increase total count in AMM dexes",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(1002),
					AmountInUSD:  float64(1000),
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(1000),
					AmountOutUSD: float64(1000),
					Gas:          50,
					GasPrice:     big.NewFloat(1),
					GasUSD:       0.07912413535198341,
					ExtraFee:     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(1000),
								SwapAmount: big.NewInt(1002),
								Exchange:   "pancake",
								PoolType:   "uniswap-v2",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "correlatedToken1",
							},
							{
								Pool:       "0xabcd",
								AmountOut:  big.NewInt(1000),
								SwapAmount: big.NewInt(1000),
								Exchange:   "kyber-pmm",
								PoolType:   "kyber-pmm",
								TokenIn:    "correlatedToken1",
								TokenOut:   "wlToken1",
							},
							{
								Pool:       "0xabcde",
								AmountOut:  big.NewInt(1000),
								SwapAmount: big.NewInt(1000),
								Exchange:   "swaap-v2",
								PoolType:   "swaap-v2",
								TokenIn:    "wlToken1",
								TokenOut:   "wlToken2",
							},
							{
								Pool:       "0xabcdef",
								AmountOut:  big.NewInt(1000),
								SwapAmount: big.NewInt(1000),
								Exchange:   "hashflow-v3",
								PoolType:   "hashflow-v3",
								TokenIn:    "wlToken2",
								TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
							},
						},
					},
					Timestamp: time.Now().Unix(),
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					SlippageTolerance:   50,
					Recipient:           recipient,
					EnableGasEstimation: true,
					Sender:              sender,
				}
			},
			result: &dto.BuildRouteResult{
				AmountIn:         "1002",
				AmountInUSD:      "0.001002",
				AmountOut:        "999",
				AmountOutUSD:     "0.000999",
				Gas:              "50",
				GasUSD:           "0.07912413535198341",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "0",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			gasEstimator: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				wg.Add(1)
				// config.IsGasEstimatorEnabled = false so estimate gas in background
				gasEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(uint64(50), big.NewInt(900), nil)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				counters := []routerEntities.FaultyPoolTracker{
					{
						Address:     "0xabc",
						TotalCount:  1,
						FailedCount: 1,
						Tokens:      []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "correlatedToken1"},
					},
				}
				wg.Add(1)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Eq(counters)).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Times(1).Return([]string{"0xabc"}, nil)
				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockITokenRepository {
				wg.Add(2)
				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(
					[]*entity.SimplifiedToken{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
						{Address: "correlatedToken1", Decimals: 6},
						{Address: "wlToken1", Decimals: 6},
						{Address: "wlToken2", Decimals: 6},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
					},
					nil,
				)
				tokenRepository.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(
					[]*routerEntities.TokenInfo{
						{Address: "correlatedToken1", IsFOT: false, IsHoneypot: false},
					},
					nil,
				)
				return tokenRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: false, IsFaultyPoolDetectorEnable: true},
				Salt:         randomSalt,
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup: map[string]bool{
						"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true,
						"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true,
					},
				},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true, "wlToken1": true, "wlToken2": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 40,
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "it should return correct result and not increase total count because some tokens is honeypot tokens",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(1002),
					AmountInUSD:  float64(1000),
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(1000),
					AmountOutUSD: float64(1000),
					Gas:          50,
					GasPrice:     big.NewFloat(1),
					GasUSD:       0.07912413535198341,
					ExtraFee:     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(1000),
								SwapAmount: big.NewInt(1002),
								Exchange:   "pancake",
								PoolType:   "uniswap-v2",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "correlatedTokenHoneypot1",
							},
							{
								Pool:       "0xabcd",
								AmountOut:  big.NewInt(1000),
								SwapAmount: big.NewInt(1000),
								Exchange:   "uniswap",
								PoolType:   "uniswap",
								TokenIn:    "correlatedTokenHoneypot1",
								TokenOut:   "wlToken1",
							},
							{
								Pool:       "0xabcde",
								AmountOut:  big.NewInt(1000),
								SwapAmount: big.NewInt(1000),
								Exchange:   "uniswap",
								PoolType:   "uniswap",
								TokenIn:    "wlToken1",
								TokenOut:   "wlToken2",
							},
							{
								Pool:       "0xabcdef",
								AmountOut:  big.NewInt(1000),
								SwapAmount: big.NewInt(1000),
								Exchange:   "uniswap",
								PoolType:   "uniswap",
								TokenIn:    "wlToken2",
								TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
							},
						},
					},
					Timestamp: time.Now().Unix(),
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					SlippageTolerance:   50,
					Recipient:           recipient,
					EnableGasEstimation: true,
					Sender:              sender,
				}
			},
			result: &dto.BuildRouteResult{
				AmountIn:         "1002",
				AmountInUSD:      "0.001002",
				AmountOut:        "999",
				AmountOutUSD:     "0.000999",
				Gas:              "50",
				GasUSD:           "0.07912413535198341",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "0",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			gasEstimator: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				wg.Add(1)
				// config.IsGasEstimatorEnabled = false so estimate gas in background
				gasEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(uint64(0), big.NewInt(100), nil)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)
				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockITokenRepository {
				wg.Add(2)
				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(
					[]*entity.SimplifiedToken{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
						{Address: "correlatedTokenHoneypot1", Decimals: 6},
						{Address: "wlToken1", Decimals: 6},
						{Address: "wlToken2", Decimals: 6},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
					},
					nil,
				)
				tokenRepository.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(
					[]*routerEntities.TokenInfo{
						{Address: "correlatedTokenHoneypot1", IsFOT: false, IsHoneypot: true},
					},
					nil,
				)
				return tokenRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: false, IsFaultyPoolDetectorEnable: true},
				Salt:         randomSalt,
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true, "wlToken1": true, "wlToken2": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 40,
						},
					},
				},
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup:      make(map[string]bool),
					CorrelatedGroup1: make(map[string]bool),
					CorrelatedGroup2: make(map[string]bool),
					CorrelatedGroup3: make(map[string]bool),
				},
			},
			err: nil,
		},
		{
			name: "it should return not count faulty pools on Redis because checksum is not correct",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(2000000000000000000),
					AmountInUSD:  float64(2000000000000000000),
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(4488767370609711072),
					AmountOutUSD: float64(4488767370609711072),
					Gas:          345000,
					GasPrice:     big.NewFloat(100000000),
					GasUSD:       0.07912413535198341,
					ExtraFee:     valueobject.ExtraFee{},
					Timestamp:    time.Now().Unix(),
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(996023110963288),
								SwapAmount: big.NewInt(2000000000000000000),
								Exchange:   "pancake",
								PoolType:   "uniswap-v2",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "wlToken1",
							},
							{
								Pool:       "0xabcd",
								AmountOut:  big.NewInt(4488767370609711072),
								SwapAmount: big.NewInt(996023110963288),
								Exchange:   "smardex",
								PoolType:   "smardex",
								TokenIn:    "wlToken1",
								TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
							},
						},
					},
					OriginalChecksum: 12345678,
				}
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					SlippageTolerance:   50,
					Recipient:           recipient,
					EnableGasEstimation: true,
					Sender:              sender,
				}
			},
			result: &dto.BuildRouteResult{
				AmountIn:         "2000000000000000000",
				AmountInUSD:      "2000000000000",
				AmountOut:        "4488767370609711071",
				AmountOutUSD:     "4488767370609.711",
				Gas:              "345000",
				GasUSD:           "0.07912413535198341",
				OutputChange:     OutputChangeNoChange,
				Data:             "mockEncodedData",
				RouterAddress:    "0x01",
				TransactionValue: "0",

				AdditionalCostUsd:     "0",
				AdditionalCostMessage: "",
			},
			gasEstimator: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				wg.Add(1)
				// config.IsGasEstimatorEnabled = false so estimate gas in background
				gasEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(uint64(0), big.NewInt(0), ErrReturnAmountIsNotEnough)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Any()).Times(0)
				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockITokenRepository {
				wg.Add(1)
				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(
					[]*entity.SimplifiedToken{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
						{Address: "wlToken1", Decimals: 6},
					},
					nil,
				)
				tokenRepository.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Times(0)
				return tokenRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: false, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true, "wlToken1": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 40,
						},
					},
				},
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup:      make(map[string]bool),
					CorrelatedGroup1: make(map[string]bool),
					CorrelatedGroup2: make(map[string]bool),
					CorrelatedGroup3: make(map[string]bool),
				},
				Salt: randomSalt,
			},
			err: nil,
		},
		{
			name: "swapSinglePool failed at sequence: 0 hop: 1 then call AddFaultyPools to pool-service",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(1000000),
					AmountInUSD:  float64(1000000),
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(1000000),
					AmountOutUSD: float64(1000000),
					Gas:          345000,
					GasPrice:     big.NewFloat(100000000),
					GasUSD:       0.07912413535198341,
					ExtraFee:     valueobject.ExtraFee{},
					Timestamp:    time.Now().Unix(),
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(1000000),
								SwapAmount: big.NewInt(2000000),
								Exchange:   "pancake",
								PoolType:   "uniswap-v2",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "wlToken1",
							},
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(2000000),
								SwapAmount: big.NewInt(1000000),
								Exchange:   "uniswap-v4",
								PoolType:   "uniswap-v4",
								TokenIn:    "wlToken1",
								TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
							},
						},
					},
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					SlippageTolerance:   10,
					Recipient:           recipient,
					EnableGasEstimation: true,
					Sender:              sender,
				}
			},
			result: nil,
			gasEstimator: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				wg.Add(1)
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().EstimateGasAndPriceUSD(gomock.Any(), gomock.Any()).
					Do(func(arg0, arg1 interface{}) {
						defer wg.Done()
					}).
					Return(uint64(0), 0.0, nil, errors.New("swapSinglePool failed at sequence: 0 hop: 1: some error")).
					Times(1)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				wg.Add(1)
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				poolRepository.EXPECT().AddFaultyPools(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(nil).Times(1)
				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockITokenRepository {
				wg.Add(1)
				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(
					[]*entity.SimplifiedToken{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
					},
					nil,
				).Times(1)
				return tokenRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": false, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": false, "wlToken1": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 500,
						},
					},
				},
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup:      make(map[string]bool),
					CorrelatedGroup1: make(map[string]bool),
					CorrelatedGroup2: make(map[string]bool),
					CorrelatedGroup3: make(map[string]bool),
				},
				Salt: randomSalt,
			},
			err: ErrEstimateGasFailed(errors.New("swapSinglePool failed at sequence: 0 hop: 1: some error")),
		},
		{
			name: "return amount is not enough due to low slippage tolerance",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(1000000),
					AmountInUSD:  float64(1000000),
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(1000000),
					AmountOutUSD: float64(1000000),
					Gas:          345000,
					GasPrice:     big.NewFloat(100000000),
					GasUSD:       0.07912413535198341,
					ExtraFee:     valueobject.ExtraFee{},
					Timestamp:    time.Now().Unix(),
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(1000000),
								SwapAmount: big.NewInt(2000000),
								Exchange:   "pancake",
								PoolType:   "uniswap-v2",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "wlToken1",
							},
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(2000000),
								SwapAmount: big.NewInt(1000000),
								Exchange:   "uniswap-v4",
								PoolType:   "uniswap-v4",
								TokenIn:    "wlToken1",
								TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
							},
						},
					},
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					SlippageTolerance:   10,
					Recipient:           recipient,
					EnableGasEstimation: true,
					Sender:              sender,
				}
			},
			gasEstimator: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIGasEstimator {
				gasEstimator := buildroute.NewMockIGasEstimator(ctrl)
				gasEstimator.EXPECT().EstimateGasAndPriceUSD(gomock.Any(), gomock.Any()).Return(uint64(345000),
					0.07912413535198341, big.NewInt(900000), nil).Times(1)
				return gasEstimator
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				wg.Add(1)
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				counters := []routerEntities.FaultyPoolTracker{
					{
						Address:     "0xabc",
						TotalCount:  1,
						FailedCount: 1,
						Tokens:      []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "wlToken1"},
					},
					{
						Address:     "0xabc",
						TotalCount:  1,
						FailedCount: 1,
						Tokens:      []string{"wlToken1", "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab"},
					},
				}
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Eq(counters)).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return([]string{"0xabc"}, nil).Times(1)
				return poolRepository
			},
			tokenRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockITokenRepository {
				wg.Add(2)
				tokenRepository := buildroute.NewMockITokenRepository(ctrl)
				tokenRepository.EXPECT().
					FindByAddresses(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(
					[]*entity.SimplifiedToken{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
					},
					nil,
				)
				tokenRepository.EXPECT().
					FindTokenInfoByAddress(gomock.Any(), gomock.Any()).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return(
					[]*routerEntities.TokenInfo{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", IsFOT: false, IsHoneypot: false},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", IsFOT: false, IsHoneypot: false},
					},
					nil,
				)
				return tokenRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				Salt:         randomSalt,
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup: map[string]bool{
						"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true,
						"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true,
					},
				},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": false, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": false, "wlToken1": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 500,
						},
					},
				},
			},
			result: &dto.BuildRouteResult{
				SuggestedSlippage: 1000 + 1, // 0.01% buffer for stable
			},
			err: ErrEstimateGasFailed(ErrReturnAmountIsNotEnough),
		},
	}

	wg := sync.WaitGroup{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			clientDataEncoder := clientdata.NewMockIClientDataEncoder(ctrl)
			clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil).AnyTimes()

			encoder := mockEncode.NewMockIEncoder(ctrl)
			encodedData := "mockEncodedData"

			encoder.EXPECT().
				Encode(gomock.Any()).
				Return(encodedData, nil).AnyTimes()
			encoder.EXPECT().
				GetExecutorAddress(gomock.Any()).
				Return("0x00").AnyTimes()
			encoder.EXPECT().
				GetRouterAddress().
				Return("0x01").AnyTimes()

			tokenRepository := tc.tokenRepository(ctrl, &wg)

			onchainpriceRepo := buildroute.NewMockIOnchainPriceRepository(ctrl)
			onchainpriceRepo.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).
				Return(
					map[string]*routerEntities.OnchainPrice{
						"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
							USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
						},
						"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": {
							USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
						},
						"0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2": {
							USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
						}}, nil,
				).AnyTimes()

			executorBalanceRepository := buildroute.NewMockIExecutorBalanceRepository(ctrl)
			executorBalanceRepository.EXPECT().HasToken(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()
			executorBalanceRepository.EXPECT().HasPoolApproval(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()

			poolRepository := tc.poolRepository(ctrl, &wg)

			alphaFeeRepository := buildroute.NewMockIAlphaFeeRepository(ctrl)
			alphaFeeRepository.EXPECT().GetByRouteId(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

			publisherRepository := buildroute.NewMockIPublisherRepository(ctrl)
			publisherRepository.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			usecase := NewBuildRouteUseCase(
				tc.config,
				tokenRepository,
				poolRepository,
				executorBalanceRepository,
				onchainpriceRepo,
				alphaFeeRepository,
				nil,
				publisherRepository,
				tc.gasEstimator(ctrl, &wg),
				&dummyL1FeeCalculator{},
				nil,
				clientDataEncoder,
				encoder,
			)

			result, err := usecase.Handle(context.Background(), tc.command())
			wg.Wait()

			assert.Equal(t, tc.result, result)
			if tc.err != nil {
				assert.Equal(t, tc.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildRouteUseCase_HandleWithTrackingFaultyPoolsRFQ(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                 string
		command              func() dto.BuildRouteCommand
		rfqHandlerByExchange func(ctrl *gomock.Controller) map[valueobject.Exchange]pool.IPoolRFQ
		poolRepository       func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository
		config               Config
		err                  error
	}{
		{
			name: "it should return correct result and increase total count (failed count is 1) on Redis when rfq with kyber-pmm failed",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(2000000000000000000),
					AmountInUSD:  float64(2000000000000000000),
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(4488767370609711072),
					AmountOutUSD: float64(4488767370609711072),
					Gas:          345000,
					GasPrice:     big.NewFloat(100000000),
					GasUSD:       0.07912413535198341,
					ExtraFee:     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(996023110963288),
								SwapAmount: big.NewInt(2000000000000000000),
								Exchange:   "kyber-pmm",
								PoolType:   "kyber-pmm",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "wlToken1",
							},
							{
								Pool:       "0xxyz",
								AmountOut:  big.NewInt(4488767370609711072),
								SwapAmount: big.NewInt(996023110963288),
								Exchange:   "smardex",
								PoolType:   "smardex",
								TokenIn:    "wlToken1",
								TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
							},
						},
					},
					Timestamp: time.Now().Unix(),
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					SlippageTolerance:   5,
					EnableGasEstimation: true,
				}
			},
			rfqHandlerByExchange: func(ctrl *gomock.Controller) map[valueobject.Exchange]pool.IPoolRFQ {
				rfqHandlerByExchange := map[valueobject.Exchange]pool.IPoolRFQ{}
				rfqHandler := buildroute.NewMockIPoolRFQ(ctrl)
				rfqHandlerByExchange[kyberpmm.DexTypeKyberPMM] = rfqHandler
				rfqHandler.EXPECT().RFQ(gomock.Any(), gomock.Any()).Times(1).Return(nil, kyberpmmClient.ErrFirmQuoteFailed)
				rfqHandler.EXPECT().SupportBatch().Return(false).AnyTimes()

				return rfqHandlerByExchange
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				wg.Add(1)
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				counters := []routerEntities.FaultyPoolTracker{
					{
						Address:     "0xabc",
						TotalCount:  1,
						FailedCount: 1,
						Tokens:      []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "wlToken1"},
					},
				}
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Eq(counters)).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return([]string{"0xabc"}, nil).Times(1)
				return poolRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true, "wlToken1": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 500,
						},
					},
				},
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup:      make(map[string]bool),
					CorrelatedGroup1: make(map[string]bool),
					CorrelatedGroup2: make(map[string]bool),
					CorrelatedGroup3: make(map[string]bool),
				},
				Salt: randomSalt,
			},
			err: errors.WithMessagef(kyberpmmClient.ErrFirmQuoteFailed, "rfq failed: swaps data: %v", []valueobject.Swap{
				{
					Pool:       "0xabc",
					AmountOut:  big.NewInt(996023110963288),
					SwapAmount: big.NewInt(2000000000000000000),
					Exchange:   "kyber-pmm",
					PoolType:   "kyber-pmm",
					TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					TokenOut:   "wlToken1",
				},
			}),
		},
		{
			name: "it should return correct result and increase total count (failed count is 1) on Redis when rfq with native-v1 failed",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(2000000000000000000),
					AmountInUSD:  float64(2000000000000000000),
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(4488767370609711072),
					AmountOutUSD: float64(4488767370609711072),
					Gas:          345000,
					GasPrice:     big.NewFloat(100000000),
					GasUSD:       0.07912413535198341,
					ExtraFee:     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(996023110963288),
								SwapAmount: big.NewInt(2000000000000000000),
								Exchange:   "kyber-pmm",
								PoolType:   "kyber-pmm",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "wlToken1",
							},
							{
								Pool:       "0xxyz",
								AmountOut:  big.NewInt(4488767370609711072),
								SwapAmount: big.NewInt(996023110963288),
								Exchange:   "native-v1",
								PoolType:   "native-v1",
								TokenIn:    "wlToken1",
								TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
							},
						},
					},
					Timestamp: time.Now().Unix(),
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					EnableGasEstimation: true,
				}
			},
			rfqHandlerByExchange: func(ctrl *gomock.Controller) map[valueobject.Exchange]pool.IPoolRFQ {
				rfqHandlerByExchange := map[valueobject.Exchange]pool.IPoolRFQ{}
				pmmRfqHandler := buildroute.NewMockIPoolRFQ(ctrl)
				pmmRfqHandler.EXPECT().RFQ(gomock.Any(), gomock.Any()).Times(1).Return(&pool.RFQResult{
					NewAmountOut: big.NewInt(996023110963288),
				}, nil)
				pmmRfqHandler.EXPECT().SupportBatch().Return(false).AnyTimes()
				rfqHandlerByExchange[kyberpmm.DexTypeKyberPMM] = pmmRfqHandler

				nativev1RfqHandler := buildroute.NewMockIPoolRFQ(ctrl)
				rfqHandlerByExchange[nativev1.DexType] = nativev1RfqHandler
				nativev1RfqHandler.EXPECT().RFQ(gomock.Any(), gomock.Any()).Times(1).Return(nil, nativev1Client.ErrRFQAllPricerFailed)
				nativev1RfqHandler.EXPECT().SupportBatch().Return(false).AnyTimes()

				return rfqHandlerByExchange
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				wg.Add(2)
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				pmmCounter := []routerEntities.FaultyPoolTracker{
					{
						Address:     "0xabc",
						TotalCount:  1,
						FailedCount: 0,
						Tokens:      []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "wlToken1"},
					},
				}
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Eq(pmmCounter)).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return([]string{"0xabc"}, nil).Times(1)
				nativev1Counter := []routerEntities.FaultyPoolTracker{
					{
						Address:     "0xxyz",
						TotalCount:  1,
						FailedCount: 1,
						Tokens:      []string{"wlToken1", "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab"},
					},
				}
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Eq(nativev1Counter)).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return([]string{"0xabc"}, nil).Times(1)
				return poolRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true, "wlToken1": true, "wlToken2": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 500,
						},
					},
				},
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup:      make(map[string]bool),
					CorrelatedGroup1: make(map[string]bool),
					CorrelatedGroup2: make(map[string]bool),
					CorrelatedGroup3: make(map[string]bool),
				},
				Salt: randomSalt,
			},
			err: errors.WithMessagef(nativev1Client.ErrRFQAllPricerFailed, "rfq failed: swaps data: %v", []valueobject.Swap{
				{
					Pool:       "0xxyz",
					AmountOut:  big.NewInt(4488767370609711072),
					SwapAmount: big.NewInt(996023110963288),
					Exchange:   "native-v1",
					PoolType:   "native-v1",
					TokenIn:    "wlToken1",
					TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
				},
			}),
		},
		{
			name: "it should return correct result and increase total count (failed count is 0) on Redis when rfq firm has no error",
			command: func() dto.BuildRouteCommand {
				route := &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(2000000000000000000),
					AmountInUSD:  float64(2000000000000000000),
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(4488767370609711072),
					AmountOutUSD: float64(4488767370609711072),
					Gas:          345000,
					GasPrice:     big.NewFloat(100000000),
					GasUSD:       0.07912413535198341,
					ExtraFee:     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								AmountOut:  big.NewInt(996023110963288),
								SwapAmount: big.NewInt(2000000000000000000),
								Exchange:   "hashflow-v3",
								PoolType:   "hashflow-v3",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "wlToken1",
							},
							{
								Pool:       "0xxyz",
								AmountOut:  big.NewInt(4488767370609711072),
								SwapAmount: big.NewInt(996023110963288),
								Exchange:   "native-v1",
								PoolType:   "native-v1",
								TokenIn:    "wlToken1",
								TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
							},
						},
					},
					Timestamp: time.Now().Unix(),
				}
				route.OriginalChecksum = crypto.NewChecksum(route, randomSalt).Hash()
				return dto.BuildRouteCommand{
					RouteSummary:        route,
					EnableGasEstimation: true,
				}
			},
			rfqHandlerByExchange: func(ctrl *gomock.Controller) map[valueobject.Exchange]pool.IPoolRFQ {
				rfqHandlerByExchange := map[valueobject.Exchange]pool.IPoolRFQ{}
				hashflowHandler := buildroute.NewMockIPoolRFQ(ctrl)
				hashflowHandler.EXPECT().SupportBatch().Return(true).AnyTimes()
				hashflowHandler.EXPECT().BatchRFQ(gomock.Any(), gomock.Any()).AnyTimes().Return([]*pool.RFQResult{
					{
						NewAmountOut: big.NewInt(996023110963288),
					},
				}, nil)
				hashflowHandler.EXPECT().RFQ(gomock.Any(), gomock.Any()).AnyTimes().Return(&pool.RFQResult{
					NewAmountOut: big.NewInt(996023110963288),
				}, nil)
				rfqHandlerByExchange[hashflowv3.DexType] = hashflowHandler

				nativev1RfqHandler := buildroute.NewMockIPoolRFQ(ctrl)
				rfqHandlerByExchange[nativev1.DexType] = nativev1RfqHandler
				nativev1RfqHandler.EXPECT().RFQ(gomock.Any(), gomock.Any()).Times(1).Return(&pool.RFQResult{
					NewAmountOut: big.NewInt(996023110963288),
				}, nil)
				nativev1RfqHandler.EXPECT().SupportBatch().Return(true).AnyTimes()

				return rfqHandlerByExchange
			},
			poolRepository: func(ctrl *gomock.Controller, wg *sync.WaitGroup) *buildroute.MockIPoolRepository {
				wg.Add(2)
				poolRepository := buildroute.NewMockIPoolRepository(ctrl)
				pmmCounter := []routerEntities.FaultyPoolTracker{
					{
						Address:     "0xabc",
						TotalCount:  1,
						FailedCount: 0,
						Tokens:      []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "wlToken1"},
					},
				}
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Eq(pmmCounter)).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return([]string{"0xabc"}, nil).Times(1)
				nativev1Counter := []routerEntities.FaultyPoolTracker{
					{
						Address:     "0xxyz",
						TotalCount:  1,
						FailedCount: 0,
						Tokens:      []string{"wlToken1", "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab"},
					},
				}
				poolRepository.EXPECT().TrackFaultyPools(gomock.Any(), gomock.Eq(nativev1Counter)).Do(func(arg0, arg1 interface{}) {
					defer wg.Done()
				}).Return([]string{"0xabc"}, nil).Times(1)
				return poolRepository
			},
			config: Config{
				ChainID:      valueobject.ChainIDEthereum,
				FeatureFlags: valueobject.FeatureFlags{IsGasEstimatorEnabled: true, IsFaultyPoolDetectorEnable: true},
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true, "wlToken1": true, "wlToken2": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 500,
						},
					},
				},
				TokenGroups: &valueobject.TokenGroupConfig{
					StableGroup:      make(map[string]bool),
					CorrelatedGroup1: make(map[string]bool),
					CorrelatedGroup2: make(map[string]bool),
					CorrelatedGroup3: make(map[string]bool),
				},
				Salt: randomSalt,
			},
			err: nil,
		},
	}

	wg := sync.WaitGroup{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			command := tc.command()

			encoder := mockEncode.NewMockIEncoder(ctrl)
			encoder.EXPECT().
				GetExecutorAddress(gomock.Any()).
				Return("0x00").AnyTimes()
			encoder.EXPECT().
				GetRouterAddress().
				Return("0x01").AnyTimes()

			executorBalanceRepository := buildroute.NewMockIExecutorBalanceRepository(ctrl)
			executorBalanceRepository.EXPECT().
				HasToken(gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]bool{true}, nil).AnyTimes()

			clientDataEncoder := clientdata.NewMockIClientDataEncoder(ctrl)
			tokenRepository := buildroute.NewMockITokenRepository(ctrl)
			tokenRepository.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).Return([]*entity.SimplifiedToken{
				{
					Address:  command.RouteSummary.TokenIn,
					Decimals: 1,
				},
				{
					Address:  command.RouteSummary.TokenOut,
					Decimals: 1,
				},
			}, nil)
			onchainPriceRepo := buildroute.NewMockIOnchainPriceRepository(ctrl)
			onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).
				Return(
					map[string]*routerEntities.OnchainPrice{
						command.RouteSummary.TokenIn: {
							USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
						},
						command.RouteSummary.TokenOut: {
							USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
						}}, nil,
				).AnyTimes()
			poolRepository := tc.poolRepository(ctrl, &wg)

			alphaFeeRepository := buildroute.NewMockIAlphaFeeRepository(ctrl)
			alphaFeeRepository.EXPECT().GetByRouteId(gomock.Any(), gomock.Any()).Times(0)

			publisherRepository := buildroute.NewMockIPublisherRepository(ctrl)
			publisherRepository.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			usecase := NewBuildRouteUseCase(
				tc.config,
				tokenRepository,
				poolRepository,
				executorBalanceRepository,
				onchainPriceRepo,
				alphaFeeRepository,
				nil,
				publisherRepository,
				nil,
				&dummyL1FeeCalculator{},
				tc.rfqHandlerByExchange(ctrl),
				clientDataEncoder,
				encoder,
			)

			_, err := usecase.Handle(context.Background(), command)
			wg.Wait()

			if tc.err != nil {
				assert.Equal(t, tc.err.Error(), err.Error())
			}
		})
	}
}

func TestBuildRouteUseCase_RFQAcceptableSlippage(t *testing.T) {
	testCases := []struct {
		name                 string
		command              dto.BuildRouteCommand
		rfqHandlerByExchange func(ctrl *gomock.Controller) map[valueobject.Exchange]pool.IPoolRFQ
		config               Config
		err                  error
	}{
		{
			name: "it should not return error when rfq slippage is acceptable",
			command: dto.BuildRouteCommand{
				RouteSummary: &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(2000000000000000000),
					AmountInUSD:  float64(2000000000000000000),
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(4488767370609711072),
					AmountOutUSD: float64(4488767370609711072),
					Gas:          345000,
					GasPrice:     big.NewFloat(100000000),
					GasUSD:       0.07912413535198341,
					ExtraFee:     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								SwapAmount: big.NewInt(2000000000000000000),
								AmountOut:  big.NewInt(4488767370609711072),
								Exchange:   "hashflow-v3",
								PoolType:   "hashflow-v3",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
							},
						},
					},
				},
				SlippageTolerance: 2000,
			},
			rfqHandlerByExchange: func(ctrl *gomock.Controller) map[valueobject.Exchange]pool.IPoolRFQ {
				rfqHandlerByExchange := map[valueobject.Exchange]pool.IPoolRFQ{}
				hashflowHandler := buildroute.NewMockIPoolRFQ(ctrl)
				hashflowHandler.EXPECT().SupportBatch().Return(true).AnyTimes()
				hashflowHandler.EXPECT().BatchRFQ(gomock.Any(), gomock.Any()).AnyTimes().Return([]*pool.RFQResult{
					{
						NewAmountOut: big.NewInt(4488767370609711071),
					},
				}, nil)
				hashflowHandler.EXPECT().RFQ(gomock.Any(), gomock.Any()).Times(1).Return(&pool.RFQResult{
					NewAmountOut: big.NewInt(4488767370609711071),
				}, nil)
				rfqHandlerByExchange[hashflowv3.DexType] = hashflowHandler

				return rfqHandlerByExchange
			},
			config: Config{
				ChainID:                       valueobject.ChainIDEthereum,
				RFQAcceptableSlippageFraction: 1000,
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 500,
						},
					},
				},
				FeatureFlags: valueobject.FeatureFlags{
					IsFaultyPoolDetectorEnable: false,
				},
			},
			err: nil,
		},
		{
			name: "it should return error when rfq slippage is not acceptable",
			command: dto.BuildRouteCommand{
				RouteSummary: &valueobject.RouteSummary{
					TokenIn:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					AmountIn:     big.NewInt(2000000000000000000),
					AmountInUSD:  float64(2000000000000000000),
					TokenOut:     "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
					AmountOut:    big.NewInt(4488767370609711072),
					AmountOutUSD: float64(4488767370609711072),
					Gas:          345000,
					GasPrice:     big.NewFloat(100000000),
					GasUSD:       0.07912413535198341,
					ExtraFee:     valueobject.ExtraFee{},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:       "0xabc",
								SwapAmount: big.NewInt(2000000000000000000),
								AmountOut:  big.NewInt(4488767370609711072),
								Exchange:   "hashflow-v3",
								PoolType:   "hashflow-v3",
								TokenIn:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
								TokenOut:   "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab",
							},
						},
					},
				},
				SlippageTolerance: 2000,
			},
			rfqHandlerByExchange: func(ctrl *gomock.Controller) map[valueobject.Exchange]pool.IPoolRFQ {
				rfqHandlerByExchange := map[valueobject.Exchange]pool.IPoolRFQ{}
				hashflowHandler := buildroute.NewMockIPoolRFQ(ctrl)
				hashflowHandler.EXPECT().SupportBatch().Return(true).AnyTimes()
				hashflowHandler.EXPECT().BatchRFQ(gomock.Any(), gomock.Any()).AnyTimes().Return([]*pool.RFQResult{
					{
						// Smaller than expected return amount 15% (greater than slippage, but smaller than acceptable RFQ slippage).
						NewAmountOut: big.NewInt(3815452265018254411),
					},
				}, nil)
				hashflowHandler.EXPECT().RFQ(gomock.Any(), gomock.Any()).AnyTimes().Return(&pool.RFQResult{
					NewAmountOut: big.NewInt(3815452265018254411),
				}, nil)
				rfqHandlerByExchange[hashflowv3.DexType] = hashflowHandler

				return rfqHandlerByExchange
			},
			config: Config{
				ChainID:                       valueobject.ChainIDEthereum,
				RFQAcceptableSlippageFraction: 1000,
				FaultyPoolsConfig: FaultyPoolsConfig{
					WhitelistedTokenSet: map[string]bool{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": true, "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": true},
					SlippageConfigByGroup: map[string]SlippageGroupConfig{
						"stable": {
							Buffer:       1,
							MinThreshold: 50,
						},
						"correlated": {
							Buffer:       5,
							MinThreshold: 100,
						},
						"default": {
							Buffer:       50,
							MinThreshold: 500,
						},
					},
				},
			},
			err: ErrQuotedAmountSmallerThanEstimated,
		},
	}

	wg := sync.WaitGroup{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			clientDataEncoder := clientdata.NewMockIClientDataEncoder(ctrl)
			clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil).AnyTimes()

			encoder := mockEncode.NewMockIEncoder(ctrl)
			encodedData := "mockEncodedData"

			encoder.EXPECT().
				Encode(gomock.Any()).
				Return(encodedData, nil).AnyTimes()
			encoder.EXPECT().
				GetExecutorAddress(gomock.Any()).
				Return("0x00").AnyTimes()
			encoder.EXPECT().
				GetRouterAddress().
				Return("0x01").AnyTimes()

			tokenRepository := buildroute.NewMockITokenRepository(ctrl)
			tokenRepository.EXPECT().
				FindByAddresses(gomock.Any(), gomock.Any()).
				Return(
					[]*entity.SimplifiedToken{
						{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 6},
						{Address: "0xc3d088842dcf02c13699f936bb83dfbbc6f721ab", Decimals: 6},
					},
					nil,
				).AnyTimes()
			onchainpriceRepo := buildroute.NewMockIOnchainPriceRepository(ctrl)
			onchainpriceRepo.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).
				Return(
					map[string]*routerEntities.OnchainPrice{
						"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
							USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
						},
						"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": {
							USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
						}}, nil,
				).AnyTimes()

			executorBalanceRepository := buildroute.NewMockIExecutorBalanceRepository(ctrl)
			executorBalanceRepository.EXPECT().HasToken(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()
			executorBalanceRepository.EXPECT().HasPoolApproval(gomock.Any(), gomock.Any(), gomock.Any()).Return([]bool{false}, nil).AnyTimes()

			alphaFeeRepository := buildroute.NewMockIAlphaFeeRepository(ctrl)
			alphaFeeRepository.EXPECT().GetByRouteId(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

			publisherRepository := buildroute.NewMockIPublisherRepository(ctrl)
			publisherRepository.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			usecase := NewBuildRouteUseCase(
				tc.config,
				tokenRepository,
				nil,
				executorBalanceRepository,
				onchainpriceRepo,
				alphaFeeRepository,
				nil,
				publisherRepository,
				nil,
				&dummyL1FeeCalculator{},
				tc.rfqHandlerByExchange(ctrl),
				clientDataEncoder,
				encoder,
			)

			_, err := usecase.Handle(context.Background(), tc.command)
			wg.Wait()

			if tc.err != nil {
				assert.Equal(t, tc.err.Error(), err.Error())
			}
		})
	}
}

func TestBuildRouteUseCase_ValidateReturnAmount(t *testing.T) {
	t.Parallel()

	uc := NewBuildRouteUseCase(
		Config{
			ChainID: valueobject.ChainIDEthereum,
			TokenGroups: &valueobject.TokenGroupConfig{
				StableGroup: map[string]bool{
					"stable-1": true,
					"stable-2": true,
				},
				CorrelatedGroup1: map[string]bool{
					"correlated-1": true,
					"correlated-2": true,
				},
			},
			FaultyPoolsConfig: FaultyPoolsConfig{
				SlippageConfigByGroup: map[string]SlippageGroupConfig{
					"stable": {
						Buffer:       1,
						MinThreshold: 50,
					},
					"correlated": {
						Buffer:       5,
						MinThreshold: 100,
					},
					"default": {
						Buffer:       50,
						MinThreshold: 500,
					},
				},
			},
		},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	testCases := []struct {
		name             string
		tokenIn          string
		tokenOut         string
		returnAmount     *big.Int
		routeAmountOut   *big.Int
		slippage         float64
		expectedError    string
		expectedSlippage float64
	}{
		{
			name:             "nil returnAmount",
			tokenIn:          "token-1",
			tokenOut:         "token-2",
			returnAmount:     nil,
			routeAmountOut:   big.NewInt(1000),
			slippage:         100, // 1%
			expectedError:    "invalid returnAmount",
			expectedSlippage: 0,
		},
		{
			name:             "nil routeAmountOut",
			tokenIn:          "token-1",
			tokenOut:         "token-2",
			returnAmount:     big.NewInt(1000),
			routeAmountOut:   nil,
			slippage:         100, // 1%
			expectedError:    "invalid routeAmountOut",
			expectedSlippage: 0,
		},
		{
			name:             "returnAmount equals routeAmountOut",
			tokenIn:          "token-1",
			tokenOut:         "token-2",
			returnAmount:     big.NewInt(1000),
			routeAmountOut:   big.NewInt(1000),
			slippage:         100, // 1%
			expectedError:    "",
			expectedSlippage: 0,
		},
		{
			name:             "returnAmount within slippage tolerance",
			tokenIn:          "token-1",
			tokenOut:         "token-2",
			returnAmount:     big.NewInt(990), // 1% slippage
			routeAmountOut:   big.NewInt(1000),
			slippage:         100, // 1% tolerance
			expectedError:    "",
			expectedSlippage: 0,
		},
		{
			name:             "returnAmount exceeds slippage tolerance",
			tokenIn:          "token-1",
			tokenOut:         "token-2",
			returnAmount:     big.NewInt(950), // 5% slippage
			routeAmountOut:   big.NewInt(1000),
			slippage:         100, // 1% tolerance
			expectedError:    "return amount is not enough",
			expectedSlippage: 500 + 50, // 0.5% buffer
		},
		{
			name:             "zero slippage tolerance",
			tokenIn:          "token-1",
			tokenOut:         "token-2",
			returnAmount:     big.NewInt(999),
			routeAmountOut:   big.NewInt(1000),
			slippage:         0,
			expectedError:    "return amount is not enough",
			expectedSlippage: 10 + 50, // 0.5% buffer
		},
		{
			name:             "random amount out 1",
			tokenIn:          "token-1",
			tokenOut:         "token-2",
			returnAmount:     func() *big.Int { n, _ := new(big.Int).SetString("823428482371234567891", 10); return n }(),
			routeAmountOut:   func() *big.Int { n, _ := new(big.Int).SetString("833428482371234567891", 10); return n }(),
			slippage:         100, // 1% tolerance
			expectedError:    "return amount is not enough",
			expectedSlippage: 120 + 50, // 0.5% buffer
		},
		{
			name:             "random amount out 2",
			tokenIn:          "token-1",
			tokenOut:         "token-2",
			returnAmount:     big.NewInt(900000),
			routeAmountOut:   big.NewInt(1000000),
			slippage:         5, // 0.05% tolerance
			expectedError:    "return amount is not enough",
			expectedSlippage: 1000 + 50, // 0.5% buffer
		},
		{
			name:             "stable token pair",
			tokenIn:          "stable-1",
			tokenOut:         "stable-2",
			returnAmount:     big.NewInt(950), // 5% slippage
			routeAmountOut:   big.NewInt(1000),
			slippage:         100, // 1% tolerance
			expectedError:    "return amount is not enough",
			expectedSlippage: 500 + 1, // 0.01% buffer for stable
		},
		{
			name:             "correlated token pair",
			tokenIn:          "correlated-1",
			tokenOut:         "correlated-2",
			returnAmount:     big.NewInt(950), // 5% slippage
			routeAmountOut:   big.NewInt(1000),
			slippage:         100, // 1% tolerance
			expectedError:    "return amount is not enough",
			expectedSlippage: 500 + 5, // 0.05% buffer for correlated
		},
		{
			name:             "mixed token pair",
			tokenIn:          "stable-1",
			tokenOut:         "correlated-1",
			returnAmount:     big.NewInt(950), // 5% slippage
			routeAmountOut:   big.NewInt(1000),
			slippage:         100, // 1% tolerance
			expectedError:    "return amount is not enough",
			expectedSlippage: 500 + 50, // 0.5% buffer
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			suggestedSlippage, err := uc.ValidateReturnAmount(
				context.Background(),
				tc.tokenIn,
				tc.tokenOut,
				tc.returnAmount,
				tc.routeAmountOut,
				tc.slippage,
			)
			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
				assert.Equal(t, tc.expectedSlippage, suggestedSlippage)
			}
		})
	}
}

func TestExtractPoolIndexFromError(t *testing.T) {
	tests := []struct {
		name        string
		errMsg      string
		expectedSeq int
		expectedHop int
		expectedOk  bool
	}{
		{
			name:        "basic case",
			errMsg:      "execution reverted: swapSingleSequence failed: Error(swapSinglePool failed at sequence: 0 hop: 1: Error(...))",
			expectedSeq: 0,
			expectedHop: 1,
			expectedOk:  true,
		},
		{
			name:        "double digit values",
			errMsg:      "execution reverted: swapSingleSequence failed: Error(swapSinglePool failed at sequence: 12 hop: 34: Error(...))",
			expectedSeq: 12,
			expectedHop: 34,
			expectedOk:  true,
		},
		{
			name:        "large numbers",
			errMsg:      "execution reverted: swapSingleSequence failed: Error(swapSinglePool failed at sequence: 1234 hop: 5678: Error(...))",
			expectedSeq: 1234,
			expectedHop: 5678,
			expectedOk:  true,
		},
		{
			name:        "zero values",
			errMsg:      "swapSinglePool failed at sequence: 0 hop: 0: some error",
			expectedSeq: 0,
			expectedHop: 0,
			expectedOk:  true,
		},
		{
			name:        "pattern at beginning",
			errMsg:      "swapSinglePool failed at sequence: 5 hop: 10: other error details",
			expectedSeq: 5,
			expectedHop: 10,
			expectedOk:  true,
		},
		{
			name:        "pattern in middle",
			errMsg:      "some prefix swapSinglePool failed at sequence: 99 hop: 88: some suffix",
			expectedSeq: 99,
			expectedHop: 88,
			expectedOk:  true,
		},
		{
			name:        "missing sequence",
			errMsg:      "execution reverted: swapSingleSequence failed: Error(swapSinglePool failed at hop: 5: Error(...))",
			expectedSeq: 0,
			expectedHop: 0,
			expectedOk:  false,
		},
		{
			name:        "missing hop",
			errMsg:      "execution reverted: swapSingleSequence failed: Error(swapSinglePool failed at sequence: 2: Error(...))",
			expectedSeq: 0,
			expectedHop: 0,
			expectedOk:  false,
		},
		{
			name:        "both missing",
			errMsg:      "execution reverted: unrelated error message",
			expectedSeq: 0,
			expectedHop: 0,
			expectedOk:  false,
		},
		{
			name:        "invalid sequence number",
			errMsg:      "swapSinglePool failed at sequence: abc hop: 1: error",
			expectedSeq: 0,
			expectedHop: 0,
			expectedOk:  false,
		},
		{
			name:        "invalid hop number",
			errMsg:      "swapSinglePool failed at sequence: 1 hop: xyz: error",
			expectedSeq: 0,
			expectedHop: 0,
			expectedOk:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := errors.New(tc.errMsg)
			seq, hop, ok := ExtractPoolIndexFromError(err)
			assert.Equal(t, tc.expectedSeq, seq)
			assert.Equal(t, tc.expectedHop, hop)
			assert.Equal(t, tc.expectedOk, ok)
		})
	}
}
