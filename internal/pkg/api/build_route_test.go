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
						Data             string `json:"data"`
						RouterAddress    string `json:"routerAddress"`
						TransactionValue string `json:"transactionValue"`
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
						Data             string `json:"data"`
						RouterAddress    string `json:"routerAddress"`
						TransactionValue string `json:"transactionValue"`
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
						Data:             "data",
						RouterAddress:    "addr",
						TransactionValue: "",
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

func TestBuildRoute_transactionValue(t *testing.T) {
	testCases := []struct {
		name                     string
		requestBody              string
		expectedTransactionValue string
	}{
		{
			name: "tx.value should match amountIn when tokenIn is the native token",
			requestBody: `{
				"routeSummary": {
						"tokenIn": "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
						"amountIn": "1234567980000",
						"amountInUsd": "0",
						"tokenInMarketPriceAvailable": false,
						"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
						"amountOut": "8879",
						"amountOutUsd": "0",
						"tokenOutMarketPriceAvailable": false,
						"gas": "410000",
						"gasPrice": "5773716359",
						"gasUsd": "0",
						"extraFee": {
							"feeAmount": "0",
							"chargeFeeBy": "",
							"isInBps": false,
							"feeReceiver": ""
						},
						"route": [
							[
								{
									"pool": "0xa53620f536e2c06d18f02791f1c1178c1d51f955",
									"tokenIn": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
									"tokenOut": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
									"limitReturnAmount": "0",
									"swapAmount": "1234567980000",
									"amountOut": "8",
									"exchange": "maverick-v1",
									"poolLength": 2,
									"poolType": "maverick-v1",
									"poolExtra": null,
									"extra": {}
								},
								{
									"pool": "0xbc03ce3f4236c82a3a3270af02c15a6a42857e90",
									"tokenIn": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
									"tokenOut": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
									"limitReturnAmount": "0",
									"swapAmount": "8",
									"amountOut": "7513",
									"exchange": "pancake",
									"poolLength": 2,
									"poolType": "uniswap-v2",
									"poolExtra": {
										"fee": 25,
										"feePrecision": 10000,
										"blockNumber": 20863902
									},
									"extra": null
								},
								{
									"pool": "0xc3141fc45791cca3f21f2a926fd8598c39a4c6d2",
									"tokenIn": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
									"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
									"limitReturnAmount": "0",
									"swapAmount": "7513",
									"amountOut": "8879",
									"exchange": "balancer-v2-weighted",
									"poolLength": 7,
									"poolType": "balancer-v2-weighted",
									"poolExtra": {
										"vault": "0xba12222222228d8ba445958a75a0704d566bf2c8",
										"poolId": "0xc3141fc45791cca3f21f2a926fd8598c39a4c6d20001000000000000000003b0",
										"tokenOutIndex": 6,
										"blockNumber": 20875534
									},
									"extra": null
								}
							]
						]
					},
				"sender": "0x42d0ed91b55065fabcfb9ab3516437d01430c0e6",
				"recipient": "0x42d0ed91b55065fabcfb9ab3516437d01430c0e6",
				"slippageTolerance": 500
			}`,
			expectedTransactionValue: "1234567980000",
		},
		{
			name: "tx.value should be 0 when tokenIn is not the native token",
			requestBody: `{
				"routeSummary": {
						"tokenIn": "0xb50721bcf8d664c30412cfbc6cf7a15145234ad1",
						"amountIn": "14548465465768",
						"amountInUsd": "0",
						"tokenInMarketPriceAvailable": false,
						"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
						"amountOut": "29",
						"amountOutUsd": "0",
						"tokenOutMarketPriceAvailable": false,
						"gas": "350000",
						"gasPrice": "5641539780",
						"gasUsd": "0",
						"extraFee": {
							"feeAmount": "0",
							"chargeFeeBy": "",
							"isInBps": false,
							"feeReceiver": ""
						},
						"route": [
							[
								{
									"pool": "0x1af399b58330501594ab8c015be0ad953c55f09a",
									"tokenIn": "0xb50721bcf8d664c30412cfbc6cf7a15145234ad1",
									"tokenOut": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
									"limitReturnAmount": "0",
									"swapAmount": "14548465465768",
									"amountOut": "5102248160",
									"exchange": "sushiswap",
									"poolLength": 2,
									"poolType": "uniswap-v2",
									"poolExtra": {
										"fee": 3,
										"feePrecision": 1000,
										"blockNumber": 20864060
									},
									"extra": null
								},
								{
									"pool": "0xd8dec118e1215f02e10db846dcbbfe27d477ac19",
									"tokenIn": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
									"tokenOut": "0x6b175474e89094c44da98b954eedeac495271d0f",
									"limitReturnAmount": "0",
									"swapAmount": "5102248160",
									"amountOut": "23533010883099",
									"exchange": "uniswapv3",
									"poolLength": 2,
									"poolType": "uniswapv3",
									"poolExtra": {
										"blockNumber": 0
									},
									"extra": {}
								},
								{
									"pool": "0xc3141fc45791cca3f21f2a926fd8598c39a4c6d2",
									"tokenIn": "0x6b175474e89094c44da98b954eedeac495271d0f",
									"tokenOut": "0xdac17f958d2ee523a2206206994597c13d831ec7",
									"limitReturnAmount": "0",
									"swapAmount": "23533010883099",
									"amountOut": "29",
									"exchange": "balancer-v2-weighted",
									"poolLength": 7,
									"poolType": "balancer-v2-weighted",
									"poolExtra": {
										"vault": "0xba12222222228d8ba445958a75a0704d566bf2c8",
										"poolId": "0xc3141fc45791cca3f21f2a926fd8598c39a4c6d20001000000000000000003b0",
										"tokenOutIndex": 6,
										"blockNumber": 20875534
									},
									"extra": null
								}
							]
						]
					},
				"sender": "0x42d0ed91b55065fabcfb9ab3516437d01430c0e6",
				"recipient": "0x42d0ed91b55065fabcfb9ab3516437d01430c0e6",
				"slippageTolerance": 1
			}`,
			expectedTransactionValue: "0",
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
						Data             string `json:"data"`
						RouterAddress    string `json:"routerAddress"`
						TransactionValue string `json:"transactionValue"`
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
						Data:             "data",
						RouterAddress:    "addr",
						TransactionValue: "",
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
				if strings.EqualFold(argCommand.RouteSummary.TokenIn, "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee") ||
					strings.EqualFold(argCommand.RouteSummary.TokenIn, "0xeeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE") {
					assert.Equal(t, argCommand.RouteSummary.AmountIn.String(), tc.expectedTransactionValue)
				} else {
					assert.Equal(t, "0", tc.expectedTransactionValue)
				}
			})
	}
}
