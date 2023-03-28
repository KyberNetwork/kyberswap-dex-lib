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

func TestTokenController_GetTokens(t *testing.T) {
	testCases := []struct {
		name    string
		prepare func(ctrl *gomock.Controller) test.HTTPTestCase
	}{
		{
			name: "it should return OK when passed correct token addresses",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				getTokensResult := &dto.GetTokensResult{
					Tokens: []*dto.GetTokensResultToken{
						{
							Name:     "token1",
							Decimals: 6,
							Symbol:   "T1",
							Type:     "type1",
							Price:    100000,
						},
						{
							Name:     "token2",
							Decimals: 6,
							Symbol:   "T2",
							Type:     "type2",
							Price:    100000,
						},
					},
				}

				mockGetTokensParamsValidator := api.NewMockIGetTokensParamsValidator(ctrl)
				mockGetTokensParamsValidator.EXPECT().
					Validate(params.GetTokensParams{IDs: "tokenAddress1,tokenAddress2"}).
					Return(nil)

				mockGetTokens := api.NewMockIGetTokensUseCase(ctrl)
				mockGetTokens.EXPECT().
					Handle(gomock.Any(), dto.GetTokensQuery{IDs: []string{"tokenaddress1", "tokenaddress2"}}).
					Return(getTokensResult, nil)

				resp := SuccessResponse{
					Code:    0,
					Message: "successfully",
					Data:    getTokensResult,
				}

				return test.HTTPTestCase{
					ReqMethod:      http.MethodGet,
					ReqURL:         "/api/v1/tokens",
					ReqParams:      url.Values{"ids": {"tokenAddress1,tokenAddress2"}},
					ReqHandler:     GetTokens(mockGetTokensParamsValidator, mockGetTokens),
					RespHTTPStatus: http.StatusOK,
					RespBody:       resp,
				}
			},
		},
		{
			name: "it should return 400 when validate query failed",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				mockGetTokensParamsValidator := api.NewMockIGetTokensParamsValidator(ctrl)
				mockGetTokensParamsValidator.EXPECT().
					Validate(params.GetTokensParams{}).
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
					ReqURL:         "/api/v1/tokens",
					ReqHandler:     GetTokens(mockGetTokensParamsValidator, nil),
					RespHTTPStatus: errorResponse.HTTPStatus,
					RespBody:       errorResponse,
				}
			},
		},
		{
			name: "it should return 500 when getTokens.Handle failed",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				mockGetTokensParamsValidator := api.NewMockIGetTokensParamsValidator(ctrl)
				mockGetTokensParamsValidator.EXPECT().
					Validate(params.GetTokensParams{IDs: "tokenAddress1"}).
					Return(nil)

				mockGetTokens := api.NewMockIGetTokensUseCase(ctrl)
				mockGetTokens.EXPECT().
					Handle(gomock.Any(), dto.GetTokensQuery{IDs: []string{"tokenaddress1"}}).
					Return(nil, fmt.Errorf("some error"))

				return test.HTTPTestCase{
					ReqMethod:      http.MethodGet,
					ReqURL:         "/api/v1/tokens",
					ReqParams:      url.Values{"ids": {"tokenAddress1"}},
					ReqHandler:     GetTokens(mockGetTokensParamsValidator, mockGetTokens),
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

func Test_transformGetTokensParams(t *testing.T) {
	t.Run("it should return correct ids", func(t *testing.T) {
		p := params.GetTokensParams{
			IDs:        " 0xae13d989dac2f0debff460ac112a837c89baa7cd , 0x32296969Ef14EB0c6d29669C550D4a0449130230",
			PoolTokens: true,
			Extra:      true,
		}

		query := transformGetTokensParams(p)

		expectedQuery := dto.GetTokensQuery{
			IDs: []string{
				"0xae13d989dac2f0debff460ac112a837c89baa7cd",
				"0x32296969ef14eb0c6d29669c550d4a0449130230",
			},
			PoolTokens: true,
			Extra:      true,
		}

		assert.Equal(t, expectedQuery, query)
	})
}
