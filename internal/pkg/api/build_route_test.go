package api

import (
	"context"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/mocks/api"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/test"
	timeutil "github.com/KyberNetwork/router-service/internal/pkg/utils/time"
	"github.com/KyberNetwork/router-service/internal/pkg/validator"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func TestBuildRoute(t *testing.T) {
	testError := errors.New("some error")
	testCases := []struct {
		name    string
		prepare func(ctrl *gomock.Controller) test.HTTPTestCase
	}{
		{
			name: "it should return 400 when bind failed",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				return test.HTTPTestCase{
					ReqMethod:      http.MethodPost,
					ReqURL:         "/api/v1/route/build",
					ReqHandler:     BuildRoute(nil, nil, timeutil.NowFunc),
					RespHTTPStatus: http.StatusBadRequest,
					RespBody: ErrorResponse{
						HTTPStatus: http.StatusBadRequest,

						Code:    4002,
						Message: "unable to bind request body",
					},
				}
			},
		},
		{
			name: "it should return 400 when validate failed",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				mockBuildRouteParamValidator := api.NewMockIBuildRouteParamsValidator(ctrl)
				mockBuildRouteParamValidator.EXPECT().
					Validate(gomock.Any(), gomock.Any()).
					Return(validator.NewValidationError("amountIn", "invalid"))

				errResponse := ErrorResponse{
					HTTPStatus: http.StatusBadRequest,
					Code:       4000,
					Message:    "bad request",
					Details: []interface{}{
						DetailsBadRequest{
							FieldViolations: []*DetailBadRequestFieldViolation{
								{
									Field:       "amountIn",
									Description: "invalid",
								},
							},
						},
					},
				}

				return test.HTTPTestCase{
					ReqMethod:      http.MethodPost,
					ReqURL:         "/api/v1/route/build",
					ReqHandler:     BuildRoute(mockBuildRouteParamValidator, nil, timeutil.NowFunc),
					ReqBody:        strings.NewReader("{}"),
					RespHTTPStatus: http.StatusBadRequest,
					RespBody:       errResponse,
				}
			},
		},
		{
			name: "it should return 400 when enable gas estimation but sender address is empty",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				mockBuildRouteParamValidator := api.NewMockIBuildRouteParamsValidator(ctrl)
				mockBuildRouteParamValidator.EXPECT().
					Validate(gomock.Any(), gomock.Any()).
					Return(nil)

				mockBuildRouteUseCase := api.NewMockIBuildRouteUseCase(ctrl)
				mockBuildRouteUseCase.EXPECT().
					Handle(gomock.Any(), gomock.Any()).
					Return(&dto.BuildRouteResult{}, buildroute.ErrSenderEmptyWhenEnableEstimateGas)

				errResponse := ErrorResponse{
					HTTPStatus: http.StatusUnprocessableEntity,
					Code:       40010,
					Message:    "sender address can not be empty when enable gas estimation",
				}

				return test.HTTPTestCase{
					ReqMethod:  http.MethodPost,
					ReqURL:     "/api/v1/route/build",
					ReqHandler: BuildRoute(mockBuildRouteParamValidator, mockBuildRouteUseCase, timeutil.NowFunc),
					ReqBody: strings.NewReader(`{
						"routeSummary": {
							"tokenIn": "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
							"amountIn": "1000000000000000000",
							"amountInUsd": "1829.51",
							"tokenInMarketPriceAvailable": false,
							"tokenOut": "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
							"amountOut": "1816792704",
							"amountOutUsd": "1825.8766675199997",
							"tokenOutMarketPriceAvailable": false,
							"gas": "250000",
							"gasPrice": "1169251241",
							"gasUsd": "0.5347892094804775",
							"extraFee": {
								"feeAmount": "0",
								"chargeFeeBy": "",
								"isInBps": false,
								"feeReceiver": ""
							},
							"route": [
								[
									{
										"pool": "0xf5d215d9c84778f85746d15762daf39b9e83a2d6",
										"tokenIn": "0xe5d7c2a44ffddf6b295a15c148167daaaf5cf34f",
										"tokenOut": "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
										"limitReturnAmount": "0",
										"swapAmount": "1000000000000000000",
										"amountOut": "1816792704",
										"exchange": "woofi-v2",
										"poolLength": 2,
										"poolType": "woofi-v2",
										"poolExtra": null,
										"extra": {}
									}
								]
							]
						},
					"enableGasEstimation": true,
					"slippageTolerance": 500,
					"recipient": "0x0193a8a52D77E27bDd4f12E0cDd52d8Ff1d97d68",
					"source": "kyberswap",
					"skipSimulateTx": false
				}`),
					RespHTTPStatus: http.StatusBadRequest,
					RespBody:       errResponse,
				}
			},
		},
		{
			name: "it should return 400 when build command failed",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				mockBuildRouteParamValidator := api.NewMockIBuildRouteParamsValidator(ctrl)
				mockBuildRouteParamValidator.EXPECT().
					Validate(gomock.Any(), gomock.Any()).
					Return(nil)

				errResponse := ErrorResponse{
					HTTPStatus: http.StatusBadRequest,
					Code:       4003,
					Message:    "invalid route",
				}

				return test.HTTPTestCase{
					ReqMethod:      http.MethodPost,
					ReqURL:         "/api/v1/route/build",
					ReqHandler:     BuildRoute(mockBuildRouteParamValidator, nil, timeutil.NowFunc),
					ReqBody:        strings.NewReader(`{"route":{}}`),
					RespHTTPStatus: http.StatusBadRequest,
					RespBody:       errResponse,
				}
			},
		},
		{
			name: "it should return 500 when encode failed",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				mockBuildRouteParamValidator := api.NewMockIBuildRouteParamsValidator(ctrl)
				mockBuildRouteParamValidator.EXPECT().
					Validate(gomock.Any(), gomock.Any()).
					Return(nil)

				mockBuildRouteUseCase := api.NewMockIBuildRouteUseCase(ctrl)
				mockBuildRouteUseCase.EXPECT().
					Handle(gomock.Any(), gomock.Any()).
					Return(&dto.BuildRouteResult{}, errors.New("some error"))

				errResponse := ErrorResponse{
					HTTPStatus: http.StatusInternalServerError,
					Code:       500,
					Message:    "internal server error",
				}

				return test.HTTPTestCase{
					ReqMethod:      http.MethodPost,
					ReqURL:         "/api/v1/route/build",
					ReqHandler:     BuildRoute(mockBuildRouteParamValidator, mockBuildRouteUseCase, timeutil.NowFunc),
					ReqBody:        strings.NewReader(`{"routeSummary":{"amountIn":"10000","amountInUsd":"10000","amountOut":"9999","amountOutUsd":"9999","gas":"20","gasUsd":"20","extraFee":{"feeAmount":"0"}}}`),
					RespHTTPStatus: http.StatusInternalServerError,
					RespBody:       errResponse,
				}
			},
		},
		{
			name: "it should return 422 when estimate gas failed",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				mockBuildRouteParamValidator := api.NewMockIBuildRouteParamsValidator(ctrl)
				mockBuildRouteParamValidator.EXPECT().
					Validate(gomock.Any(), gomock.Any()).
					Return(nil)

				mockBuildRouteUseCase := api.NewMockIBuildRouteUseCase(ctrl)
				estimateGasFailedErr := buildroute.ErrEstimateGasFailed(testError)
				mockBuildRouteUseCase.EXPECT().
					Handle(gomock.Any(), gomock.Any()).
					Return(&dto.BuildRouteResult{}, estimateGasFailedErr)

				errResponse := ErrorResponse{
					HTTPStatus: http.StatusUnprocessableEntity,
					Code:       estimateGasFailedErr.Code(),
					Message:    estimateGasFailedErr.Error(),
					Details:    []interface{}{estimateGasFailedErr.Error()},
				}

				return test.HTTPTestCase{
					ReqMethod:  http.MethodPost,
					ReqURL:     "/api/v1/route/build",
					ReqHandler: BuildRoute(mockBuildRouteParamValidator, mockBuildRouteUseCase, timeutil.NowFunc),
					ReqBody: strings.NewReader(`{
						"routeSummary": {
							"tokenIn": "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
							"amountIn": "1000000000000000000",
							"amountInUsd": "1829.51",
							"tokenInMarketPriceAvailable": false,
							"tokenOut": "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
							"amountOut": "1816792704",
							"amountOutUsd": "1825.8766675199997",
							"tokenOutMarketPriceAvailable": false,
							"gas": "250000",
							"gasPrice": "1169251241",
							"gasUsd": "0.5347892094804775",
							"extraFee": {
								"feeAmount": "0",
								"chargeFeeBy": "",
								"isInBps": false,
								"feeReceiver": ""
							},
							"route": [
								[
									{
										"pool": "0xf5d215d9c84778f85746d15762daf39b9e83a2d6",
										"tokenIn": "0xe5d7c2a44ffddf6b295a15c148167daaaf5cf34f",
										"tokenOut": "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
										"limitReturnAmount": "0",
										"swapAmount": "1000000000000000000",
										"amountOut": "1816792704",
										"exchange": "woofi-v2",
										"poolLength": 2,
										"poolType": "woofi-v2",
										"poolExtra": null,
										"extra": {}
									}
								]
							]
						},
					"enableGasEstimation": true,
					"slippageTolerance": 500,
					"recipient": "0x0193a8a52D77E27bDd4f12E0cDd52d8Ff1d97d68",
					"sender": "0x0193a8a52D77E27bDd4f12E0cDd52d8Ff1d97c67",
					"source": "kyberswap",
					"skipSimulateTx": false
				}`),
					RespHTTPStatus: http.StatusUnprocessableEntity,
					RespBody:       errResponse,
				}
			},
		},
		{
			name: "it should return 200 when there is no error",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				mockBuildRouteParamValidator := api.NewMockIBuildRouteParamsValidator(ctrl)
				mockBuildRouteParamValidator.EXPECT().
					Validate(gomock.Any(), gomock.Any()).
					Return(nil)

				mockBuildRouteUseCase := api.NewMockIBuildRouteUseCase(ctrl)
				mockBuildRouteUseCase.EXPECT().
					Handle(gomock.Any(), gomock.Any()).
					Return(&dto.BuildRouteResult{Data: "data", RouterAddress: "addr"}, nil)

				resp := SuccessResponse{
					Code:    0,
					Message: "successfully",
					Data: struct {
						AmountIn     string `json:"amountIn"`
						AmountInUSD  string `json:"amountInUsd"`
						AmountOut    string `json:"amountOut"`
						AmountOutUSD string `json:"amountOutUsd"`
						Gas          string `json:"gas"`
						GasUSD       string `json:"gasUsd"`

						AdditionalCostUsd     string `json:"additionalCostUsd"`
						AdditionalCostMessage string `json:"additionalCostMessage"`

						OutputChange struct {
							Amount  string  `json:"amount"`
							Percent float64 `json:"percent"`
							Level   int     `json:"level"`
						} `json:"outputChange"`
						Data          string `json:"data"`
						RouterAddress string `json:"routerAddress"`
					}{
						AmountIn:     "",
						AmountInUSD:  "",
						AmountOut:    "",
						AmountOutUSD: "",
						Gas:          "",
						GasUSD:       "",

						AdditionalCostUsd:     "",
						AdditionalCostMessage: "",

						OutputChange: struct {
							Amount  string  `json:"amount"`
							Percent float64 `json:"percent"`
							Level   int     `json:"level"`
						}{
							Amount:  "",
							Percent: 0,
							Level:   0,
						},
						Data:          "data",
						RouterAddress: "addr",
					},
				}

				return test.HTTPTestCase{
					ReqMethod:      http.MethodPost,
					ReqURL:         "/api/v1/route/build",
					ReqHandler:     BuildRoute(mockBuildRouteParamValidator, mockBuildRouteUseCase, timeutil.NowFunc),
					ReqBody:        strings.NewReader(`{"routeSummary":{"amountIn":"10000","amountInUsd":"10000","amountOut":"9999","amountOutUsd":"9999","gas":"20","gasUsd":"20","extraFee":{"feeAmount":"0"}}}`),
					RespHTTPStatus: http.StatusOK,
					RespBody:       resp,
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			check := tc.prepare(ctrl)

			check.Run(t)
		})
	}
}

func TestBuildRoute_EnableGasEstimation(t *testing.T) {
	testCases := []struct {
		name                    string
		requestBody             string
		expectedEstimationParam bool
	}{
		{
			name:                    "it should return 200 with estimation param is true when there is no error",
			requestBody:             `{"routeSummary":{"amountIn":"10000","amountInUsd":"10000","amountOut":"9999","amountOutUsd":"9999","gas":"20","gasUsd":"20","extraFee":{"feeAmount":"0"}},"recipient":"mockRecipient","enableGasEstimation":true,"deadline":1697469122}`,
			expectedEstimationParam: true,
		},
		{
			name:                    "it should return 200 with estimation param is false and estimate gas when there is no error",
			requestBody:             `{"routeSummary":{"amountIn":"10000","amountInUsd":"10000","amountOut":"9999","amountOutUsd":"9999","gas":"20","gasUsd":"20","extraFee":{"feeAmount":"0"}},"recipient":"mockRecipient","enableGasEstimation":false,"deadline":1697469122}`,
			expectedEstimationParam: false,
		},
		{
			name:                    "it should return 200 with estimation param is false by default and estimate gas when there is no error",
			requestBody:             `{"routeSummary":{"amountIn":"10000","amountInUsd":"10000","amountOut":"9999","amountOutUsd":"9999","gas":"20","gasUsd":"20","extraFee":{"feeAmount":"0"}},"recipient":"mockRecipient","deadline":1697469122}`,
			expectedEstimationParam: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name,
			func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockBuildRouteParamValidator := api.NewMockIBuildRouteParamsValidator(ctrl)
				mockBuildRouteParamValidator.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)

				var argCommand dto.BuildRouteCommand
				mockBuildRouteUseCase := api.NewMockIBuildRouteUseCase(ctrl)
				mockBuildRouteUseCase.EXPECT().
					Handle(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, command dto.BuildRouteCommand) (*dto.BuildRouteResult, error) {
						argCommand = command
						return &dto.BuildRouteResult{Data: "data", RouterAddress: "addr"}, nil
					})

				resp := SuccessResponse{
					Code:    0,
					Message: "successfully",
					Data: struct {
						AmountIn     string `json:"amountIn"`
						AmountInUSD  string `json:"amountInUsd"`
						AmountOut    string `json:"amountOut"`
						AmountOutUSD string `json:"amountOutUsd"`
						Gas          string `json:"gas"`
						GasUSD       string `json:"gasUsd"`

						AdditionalCostUsd     string `json:"additionalCostUsd"`
						AdditionalCostMessage string `json:"additionalCostMessage"`

						OutputChange struct {
							Amount  string  `json:"amount"`
							Percent float64 `json:"percent"`
							Level   int     `json:"level"`
						} `json:"outputChange"`
						Data          string `json:"data"`
						RouterAddress string `json:"routerAddress"`
					}{
						AmountIn:     "",
						AmountInUSD:  "",
						AmountOut:    "",
						AmountOutUSD: "",
						Gas:          "",
						GasUSD:       "",

						AdditionalCostUsd:     "",
						AdditionalCostMessage: "",

						OutputChange: struct {
							Amount  string  `json:"amount"`
							Percent float64 `json:"percent"`
							Level   int     `json:"level"`
						}{
							Amount:  "",
							Percent: 0,
							Level:   0,
						},
						Data:          "data",
						RouterAddress: "addr",
					}}
				check := test.HTTPTestCase{
					ReqMethod:      http.MethodPost,
					ReqURL:         "/api/v1/route/build",
					ReqHandler:     BuildRoute(mockBuildRouteParamValidator, mockBuildRouteUseCase, timeutil.NowFunc),
					ReqBody:        strings.NewReader(tc.requestBody),
					RespHTTPStatus: http.StatusOK,
					RespBody:       resp,
				}

				check.Run(t)
				assert.Equal(t, argCommand.EnableGasEstimation, tc.expectedEstimationParam)
			})
	}
}

func Test_transformBuildRouteParams(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		params  params.BuildRouteParams
		command dto.BuildRouteCommand
		nowFunc func() time.Time
		err     error
	}{
		{
			name:    "it should return correct error when amountIn is invalid",
			params:  params.BuildRouteParams{RouteSummary: params.RouteSummary{AmountIn: ""}},
			command: dto.BuildRouteCommand{},
			nowFunc: timeutil.NowFunc,
			err:     ErrInvalidRoute,
		},
		{
			name: "it should return correct error when amountOut is invalid",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					AmountIn:  "10000",
					AmountOut: "a",
				},
			},
			command: dto.BuildRouteCommand{},
			nowFunc: timeutil.NowFunc,
			err:     ErrInvalidRoute,
		},
		{
			name: "it should return correct error when gasPrice is invalid",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					AmountIn:  "1000000",
					AmountOut: "999999",
					GasPrice:  "a",
				},
			},
			command: dto.BuildRouteCommand{},
			nowFunc: timeutil.NowFunc,
			err:     ErrInvalidRoute,
		},
		{
			name: "it should return correct error when feeAmount is invalid",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					AmountIn:  "1000000",
					AmountOut: "999999",
					ExtraFee: params.ExtraFee{
						FeeAmount: "a",
					},
				},
			},
			command: dto.BuildRouteCommand{},
			nowFunc: timeutil.NowFunc,
			err:     ErrInvalidRoute,
		},
		{
			name: "it should return correct error when swap.LimitReturnAmount is invalid",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					AmountIn:  "1000000",
					AmountOut: "999999",
					Route: [][]params.Swap{
						{
							{
								LimitReturnAmount: "a",
							},
						},
					},
				},
			},
			command: dto.BuildRouteCommand{},
			nowFunc: timeutil.NowFunc,
			err:     ErrInvalidRoute,
		},
		{
			name: "it should return correct error when swap.SwapAmount is invalid",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					AmountIn:  "1000000",
					AmountOut: "999999",
					Route: [][]params.Swap{
						{
							{
								LimitReturnAmount: "0",
								SwapAmount:        "a",
							},
						},
					},
				},
			},
			command: dto.BuildRouteCommand{},
			nowFunc: timeutil.NowFunc,
			err:     ErrInvalidRoute,
		},
		{
			name: "it should return correct error when swap.AmountOut is invalid",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					AmountIn:  "1000000",
					AmountOut: "999999",
					Route: [][]params.Swap{
						{
							{
								LimitReturnAmount: "0",
								SwapAmount:        "10000",
								AmountOut:         "a",
							},
						},
					},
				},
			},
			command: dto.BuildRouteCommand{},
			nowFunc: timeutil.NowFunc,
			err:     ErrInvalidRoute,
		},
		{
			name: "it should return correct command",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					AmountIn:     "10000",
					AmountInUSD:  "10000.1",
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
					AmountOut:    "9999",
					AmountOutUSD: "9998.9",
					Gas:          "2000",
					GasUSD:       "2000",
					GasPrice:     "20",
					ExtraFee: params.ExtraFee{
						FeeAmount:   "1",
						ChargeFeeBy: "currency_in",
						IsInBps:     true,
						FeeReceiver: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
					},
					Route: [][]params.Swap{
						{
							{
								Pool:              "Pool",
								TokenIn:           "TokenIn",
								TokenOut:          "TokenOut",
								LimitReturnAmount: "0",
								SwapAmount:        "10000",
								AmountOut:         "9999",
								Exchange:          "Exchange",
								PoolLength:        2,
								PoolType:          "PoolType",
								Extra:             "",
							},
						},
					},
				},
				SlippageTolerance: 850,
				Recipient:         "0xeeeee79b0fead91f3e65f86e8915cb59c1a4c664",
				Referral:          "referral",
				Source:            "source",
				Permit:            "0x1111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111",
			},
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					AmountIn:     big.NewInt(10000),
					AmountInUSD:  10000.1,
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
					AmountOut:    big.NewInt(9999),
					AmountOutUSD: 9998.9,
					Gas:          2000,
					GasUSD:       2000,
					GasPrice:     func() *big.Float { gasPrice, _ := new(big.Float).SetString("20"); return gasPrice }(),
					ExtraFee: valueobject.ExtraFee{
						FeeAmount:   big.NewInt(1),
						ChargeFeeBy: valueobject.ChargeFeeByCurrencyIn,
						IsInBps:     true,
						FeeReceiver: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
					},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:              "Pool",
								TokenIn:           "TokenIn",
								TokenOut:          "TokenOut",
								LimitReturnAmount: big.NewInt(0),
								SwapAmount:        big.NewInt(10000),
								AmountOut:         big.NewInt(9999),
								Exchange:          "Exchange",
								PoolLength:        2,
								PoolType:          "PoolType",
								Extra:             "",
							},
						},
					},
				},
				Deadline:          1665561367,
				SlippageTolerance: 850,
				Recipient:         "0xeeeee79b0fead91f3e65f86e8915cb59c1a4c664",
				Referral:          "referral",
				Source:            "source",
				Permit:            common.FromHex("0x1111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111"),
			},
			nowFunc: func() time.Time {
				return time.Unix(1665560167, 0)
			},
			err: nil,
		},
		{
			name: "it should return correct command when there is a permit",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					AmountIn:     "10000",
					AmountInUSD:  "10000.1",
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
					AmountOut:    "9999",
					AmountOutUSD: "9998.9",
					Gas:          "2000",
					GasUSD:       "2000",
					GasPrice:     "20",
					ExtraFee: params.ExtraFee{
						FeeAmount:   "1",
						ChargeFeeBy: "currency_in",
						IsInBps:     true,
						FeeReceiver: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
					},
					Route: [][]params.Swap{
						{
							{
								Pool:              "Pool",
								TokenIn:           "TokenIn",
								TokenOut:          "TokenOut",
								LimitReturnAmount: "0",
								SwapAmount:        "10000",
								AmountOut:         "9999",
								Exchange:          "Exchange",
								PoolLength:        2,
								PoolType:          "PoolType",
								Extra:             "",
							},
						},
					},
				},
				SlippageTolerance: 850,
				Recipient:         "0xeeeee79b0fead91f3e65f86e8915cb59c1a4c664",
				Referral:          "referral",
				Source:            "source",
				Sender:            "sender",
				Permit:            "0x1111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111",
			},
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					AmountIn:     big.NewInt(10000),
					AmountInUSD:  10000.1,
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
					AmountOut:    big.NewInt(9999),
					AmountOutUSD: 9998.9,
					Gas:          2000,
					GasUSD:       2000,
					GasPrice:     func() *big.Float { gasPrice, _ := new(big.Float).SetString("20"); return gasPrice }(),
					ExtraFee: valueobject.ExtraFee{
						FeeAmount:   big.NewInt(1),
						ChargeFeeBy: valueobject.ChargeFeeByCurrencyIn,
						IsInBps:     true,
						FeeReceiver: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
					},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:              "Pool",
								TokenIn:           "TokenIn",
								TokenOut:          "TokenOut",
								LimitReturnAmount: big.NewInt(0),
								SwapAmount:        big.NewInt(10000),
								AmountOut:         big.NewInt(9999),
								Exchange:          "Exchange",
								PoolLength:        2,
								PoolType:          "PoolType",
								Extra:             "",
							},
						},
					},
				},
				Deadline:          1665561367,
				SlippageTolerance: 850,
				Recipient:         "0xeeeee79b0fead91f3e65f86e8915cb59c1a4c664",
				Referral:          "referral",
				Source:            "source",
				Sender:            "sender",
				Permit:            common.FromHex("0x1111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111"),
			},
			nowFunc: func() time.Time {
				return time.Unix(1665560167, 0)
			},
			err: nil,
		},
		{
			name: "it should return correct command when there is no permit",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					AmountIn:     "10000",
					AmountInUSD:  "10000.1",
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
					AmountOut:    "9999",
					AmountOutUSD: "9998.9",
					Gas:          "2000",
					GasUSD:       "2000",
					GasPrice:     "20",
					ExtraFee: params.ExtraFee{
						FeeAmount:   "1",
						ChargeFeeBy: "currency_in",
						IsInBps:     true,
						FeeReceiver: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
					},
					Route: [][]params.Swap{
						{
							{
								Pool:              "Pool",
								TokenIn:           "TokenIn",
								TokenOut:          "TokenOut",
								LimitReturnAmount: "0",
								SwapAmount:        "10000",
								AmountOut:         "9999",
								Exchange:          "Exchange",
								PoolLength:        2,
								PoolType:          "PoolType",
								Extra:             "",
							},
						},
					},
				},
				SlippageTolerance: 850,
				Recipient:         "0xeeeee79b0fead91f3e65f86e8915cb59c1a4c664",
				Referral:          "referral",
				Source:            "source",
				Sender:            "sender",
			},
			command: dto.BuildRouteCommand{
				RouteSummary: valueobject.RouteSummary{
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					AmountIn:     big.NewInt(10000),
					AmountInUSD:  10000.1,
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
					AmountOut:    big.NewInt(9999),
					AmountOutUSD: 9998.9,
					Gas:          2000,
					GasUSD:       2000,
					GasPrice:     func() *big.Float { gasPrice, _ := new(big.Float).SetString("20"); return gasPrice }(),
					ExtraFee: valueobject.ExtraFee{
						FeeAmount:   big.NewInt(1),
						ChargeFeeBy: valueobject.ChargeFeeByCurrencyIn,
						IsInBps:     true,
						FeeReceiver: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
					},
					Route: [][]valueobject.Swap{
						{
							{
								Pool:              "Pool",
								TokenIn:           "TokenIn",
								TokenOut:          "TokenOut",
								LimitReturnAmount: big.NewInt(0),
								SwapAmount:        big.NewInt(10000),
								AmountOut:         big.NewInt(9999),
								Exchange:          "Exchange",
								PoolLength:        2,
								PoolType:          "PoolType",
								Extra:             "",
							},
						},
					},
				},
				Deadline:          1665561367,
				SlippageTolerance: 850,
				Recipient:         "0xeeeee79b0fead91f3e65f86e8915cb59c1a4c664",
				Referral:          "referral",
				Source:            "source",
				Permit:            []byte(""),
				Sender:            "sender",
			},
			nowFunc: func() time.Time {
				return time.Unix(1665560167, 0)
			},
			err: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			command, err := transformBuildRouteParams(tc.params, tc.nowFunc)

			assert.Equal(t, tc.command, command)
			assert.ErrorIs(t, err, tc.err)
		})
	}
}
