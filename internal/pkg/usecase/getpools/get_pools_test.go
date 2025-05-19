package getpools

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mockusecase "github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/fixtures"
)

func TestGetPools_Handle(t *testing.T) {
	t.Parallel()

	type TestCase struct {
		name    string
		prepare func(ctrl *gomock.Controller) *GetPoolsUseCase
		query   dto.GetPoolsQuery
		result  *dto.GetPoolsResult
		err     error
	}

	theError := errors.New("some error")

	testCases := []TestCase{
		{
			name: "it should return correct result when repository returns no error",
			prepare: func(ctrl *gomock.Controller) *GetPoolsUseCase {
				mockPoolRepo := mockusecase.NewMockIPoolRepository(ctrl)
				mockPoolRepo.EXPECT().
					FindByAddresses(gomock.Any(), []string{"pooladdress1", "pooladdress2", "pooladdress3"}).
					Return(fixtures.Pools, nil)

				return NewGetPoolsUseCase(mockPoolRepo)
			},
			query: dto.GetPoolsQuery{IDs: []string{"pooladdress1", "pooladdress2", "pooladdress3"}},
			result: &dto.GetPoolsResult{
				Pools: []*dto.GetPoolsResultPool{
					{
						Address:      "pooladdress1",
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
						Address:      "poolAddress2",
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
			},
			err: nil,
		},
		{
			name: "it should return correct error when repository returns error",
			prepare: func(ctrl *gomock.Controller) *GetPoolsUseCase {

				mockPoolRepo := mockusecase.NewMockIPoolRepository(ctrl)
				mockPoolRepo.EXPECT().
					FindByAddresses(gomock.Any(), []string{"pooladdress1", "pooladdress2", "pooladdress3"}).
					Return(nil, theError)

				return NewGetPoolsUseCase(mockPoolRepo)
			},
			query:  dto.GetPoolsQuery{IDs: []string{"pooladdress1", "pooladdress2", "pooladdress3"}},
			result: nil,
			err:    theError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			getPools := tc.prepare(ctrl)

			result, err := getPools.Handle(context.Background(), tc.query)

			assert.Equal(t, tc.result, result)
			assert.ErrorIs(t, tc.err, err)
		})
	}
}
