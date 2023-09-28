package usecase

import (
	"context"
	"math/big"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	"github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func TestBuildRouteUseCase_Handle(t *testing.T) {
	t.Parallel()

	theErr := errors.New("some error")

	testCases := []struct {
		name    string
		prepare func(ctrl *gomock.Controller) *buildRouteUseCase
		command dto.BuildRouteCommand
		result  *dto.BuildRouteResult
		err     error
	}{
		{
			name: "it should return correct error when encoder return error",
			prepare: func(ctrl *gomock.Controller) *buildRouteUseCase {
				clientDataEncoder := usecase.NewMockIClientDataEncoder(ctrl)
				clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil)

				encoder := usecase.NewMockIEncoder(ctrl)
				encoder.EXPECT().
					Encode(gomock.Any()).
					Return("", theErr).AnyTimes()
				encoder.EXPECT().
					GetExecutorAddress().
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

				return &buildRouteUseCase{
					tokenRepository:   tokenRepository,
					priceRepository:   priceRepository,
					l1Encoder:         encoder,
					l2Encoder:         encoder,
					clientDataEncoder: clientDataEncoder,
					config:            BuildRouteConfig{ChainID: valueobject.ChainIDEthereum},
				}
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
			},
			result: nil,
			err:    theErr,
		},
		{
			name: "it should return correct result when there is no error",
			prepare: func(ctrl *gomock.Controller) *buildRouteUseCase {
				clientDataEncoder := usecase.NewMockIClientDataEncoder(ctrl)

				clientDataEncoder.EXPECT().Encode(gomock.Any(), gomock.Any()).Return([]byte{}, nil)

				encoder := usecase.NewMockIEncoder(ctrl)

				encoder.EXPECT().
					Encode(gomock.Any()).
					Return("abc", nil)
				encoder.EXPECT().
					GetExecutorAddress().
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

				return &buildRouteUseCase{
					tokenRepository:   tokenRepository,
					priceRepository:   priceRepository,
					l1Encoder:         encoder,
					l2Encoder:         encoder,
					clientDataEncoder: clientDataEncoder,
					config:            BuildRouteConfig{ChainID: valueobject.ChainIDEthereum},
				}
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
				SlippageTolerance: 5,
				Recipient:         "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			},
			result: &dto.BuildRouteResult{
				AmountIn:      "20000",
				AmountInUSD:   "0.02",
				AmountOut:     "10000",
				AmountOutUSD:  "0.01",
				Gas:           "0",
				GasUSD:        "0",
				OutputChange:  OutputChangeNoChange,
				Data:          "abc",
				RouterAddress: "0x01",
			},
			err: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := tc.prepare(ctrl)

			result, err := uc.Handle(context.Background(), tc.command)

			assert.Equal(t, tc.result, result)
			assert.ErrorIs(t, err, tc.err)
		})
	}
}
