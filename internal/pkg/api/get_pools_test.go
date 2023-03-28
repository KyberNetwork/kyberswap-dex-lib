package api

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/mocks/api"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/test"
	"github.com/KyberNetwork/router-service/internal/pkg/validator"
)

func TestPoolController_GetPools(t *testing.T) {
	testCases := []struct {
		name    string
		prepare func(ctrl *gomock.Controller) test.HTTPTestCase
	}{
		{
			name: "it should return OK when passed correct pool addresses",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				getPoolsResult := &dto.GetPoolsResult{
					Pools: []*dto.GetPoolsResultPool{
						{
							ReserveUsd:   10000,
							AmplifiedTvl: 10,
							SwapFee:      100,
							Exchange:     "exchange1",
							Type:         "type1",
							Timestamp:    1658373335,
							Reserves:     []string{"10000", "20000"},
							Tokens: []*dto.GetPoolsResultPoolToken{
								{
									Address:   "tokenaddress1",
									Name:      "tokenName1",
									Symbol:    "tokenSymbol1",
									Decimals:  6,
									Weight:    50,
									Swappable: true,
								},
								{
									Address:   "tokenaddress2",
									Name:      "tokenName2",
									Symbol:    "tokenSymbol2",
									Decimals:  6,
									Weight:    50,
									Swappable: true,
								},
							},
							Extra:       "extra1",
							StaticExtra: "staticExtra1",
							TotalSupply: "10000",
						},
						{
							ReserveUsd:   20000,
							AmplifiedTvl: 20,
							SwapFee:      200,
							Exchange:     "exchange2",
							Type:         "type2",
							Timestamp:    1658373335,
							Reserves:     []string{"20000", "30000"},
							Tokens: []*dto.GetPoolsResultPoolToken{
								{
									Address:   "tokenaddress2",
									Name:      "tokenName2",
									Symbol:    "tokenSymbol2",
									Decimals:  6,
									Weight:    50,
									Swappable: true,
								},
								{
									Address:   "tokenaddress3",
									Name:      "tokenName3",
									Symbol:    "tokenSymbol3",
									Decimals:  6,
									Weight:    50,
									Swappable: true,
								},
							},
							Extra:       "extra2",
							StaticExtra: "staticExtra2",
							TotalSupply: "20000",
						},
					},
				}

				getPoolsParamsValidator := api.NewMockIGetPoolsParamsValidator(ctrl)
				getPoolsParamsValidator.EXPECT().
					Validate(params.GetPoolsParams{IDs: "poolAddress1,poolAddress2,poolAddress3"}).
					Return(nil)

				getPoolsUseCase := api.NewMockIGetPoolsUseCase(ctrl)
				getPoolsUseCase.EXPECT().
					Handle(gomock.Any(), dto.GetPoolsQuery{IDs: []string{"pooladdress1", "pooladdress2", "pooladdress3"}}).
					Return(getPoolsResult, nil)

				resp := SuccessResponse{
					Code:    0,
					Message: "successfully",
					Data:    getPoolsResult,
				}

				return test.HTTPTestCase{
					ReqMethod:      http.MethodGet,
					ReqURL:         "/api/v1/pools",
					ReqParams:      url.Values{"ids": {"poolAddress1,poolAddress2,poolAddress3"}},
					ReqBody:        nil,
					ReqHandler:     GetPools(getPoolsParamsValidator, getPoolsUseCase),
					RespHTTPStatus: http.StatusOK,
					RespBody:       resp,
				}
			},
		},
		{
			name: "it should return 400 when validate query failed",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				getPoolsParamsValidator := api.NewMockIGetPoolsParamsValidator(ctrl)
				getPoolsParamsValidator.EXPECT().
					Validate(params.GetPoolsParams{IDs: "poolAddress1,poolAddress2,poolAddress3"}).
					Return(validator.NewValidationError("ids", "required"))

				errorResponse := ErrorResponse{
					HTTPStatus: http.StatusBadRequest,
					Code:       4000,
					Message:    "bad request",
					Details: []interface{}{
						DetailsBadRequest{
							FieldViolations: []*DetailBadRequestFieldViolation{
								{
									Field:       "ids",
									Description: "required",
								},
							},
						},
					},
				}

				return test.HTTPTestCase{
					ReqMethod:      http.MethodGet,
					ReqURL:         "/api/v1/pools",
					ReqParams:      url.Values{"ids": {"poolAddress1,poolAddress2,poolAddress3"}},
					ReqBody:        nil,
					ReqHandler:     GetPools(getPoolsParamsValidator, nil),
					RespHTTPStatus: errorResponse.HTTPStatus,
					RespBody:       errorResponse,
				}
			},
		},
		{
			name: "it should return 500 when getPools.Handle failed",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				getPoolsParamsValidator := api.NewMockIGetPoolsParamsValidator(ctrl)
				getPoolsParamsValidator.EXPECT().
					Validate(params.GetPoolsParams{IDs: "poolAddress1"}).
					Return(nil)

				getPoolsUseCase := api.NewMockIGetPoolsUseCase(ctrl)
				getPoolsUseCase.EXPECT().
					Handle(gomock.Any(), dto.GetPoolsQuery{IDs: []string{"pooladdress1"}}).
					Return(nil, fmt.Errorf("some error"))

				return test.HTTPTestCase{
					ReqMethod:      http.MethodGet,
					ReqURL:         "/api/v1/pools",
					ReqParams:      url.Values{"ids": {"poolAddress1"}},
					ReqBody:        nil,
					ReqHandler:     GetPools(getPoolsParamsValidator, getPoolsUseCase),
					RespHTTPStatus: http.StatusInternalServerError,
					RespBody:       DefaultErrorResponse,
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

func Test_transformGetPoolsParams(t *testing.T) {
	t.Run("it should return correct query", func(t *testing.T) {
		reqParams := params.GetPoolsParams{
			IDs: " 0xae13d989dac2f0debff460ac112a837c89baa7cd , 0x32296969Ef14EB0c6d29669C550D4a0449130230",
		}

		query := transformGetPoolsParams(reqParams)

		expectedQuery := dto.GetPoolsQuery{
			IDs: []string{
				"0xae13d989dac2f0debff460ac112a837c89baa7cd",
				"0x32296969ef14eb0c6d29669c550d4a0449130230",
			},
		}

		assert.Equal(t, expectedQuery, query)
	})
}
