package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/mocks/usecase"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils/fixtures"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestGetTokens_Handle(t *testing.T) {
	t.Parallel()

	type TestCase struct {
		name    string
		prepare func(ctrl *gomock.Controller) *getTokensUseCase
		query   dto.GetTokensQuery
		result  *dto.GetTokensResult
		err     error
	}

	theErr := errors.New("some error")

	testCases := []TestCase{
		{
			name: "it should return correct result when there is no error",
			prepare: func(ctrl *gomock.Controller) *getTokensUseCase {
				mockTokenRepo := usecase.NewMockITokenRepository(ctrl)
				mockTokenRepo.
					EXPECT().
					FindByAddresses(gomock.Any(), []string{"tokenaddress1", "tokenaddress4"}).
					Return([]entity.Token{fixtures.Tokens[0], fixtures.Tokens[3]}, nil)

				mockTokenRepo.
					EXPECT().
					FindByAddresses(gomock.Any(), []string{"tokenaddress2"}).
					Return([]entity.Token{fixtures.Tokens[1]}, nil)

				mockPriceRepo := usecase.NewMockIPriceRepository(ctrl)
				mockPriceRepo.
					EXPECT().
					FindByAddresses(gomock.Any(), []string{"tokenaddress1", "tokenaddress4"}).
					Return([]entity.Price{fixtures.Prices[0], fixtures.Prices[3]}, nil)

				mockPoolRepo := usecase.NewMockIPoolRepository(ctrl)
				mockPoolRepo.
					EXPECT().
					FindByAddresses(gomock.Any(), []string{"pooladdress1"}).
					Return([]entity.Pool{fixtures.Pools[0]}, nil)

				return NewGetTokens(mockTokenRepo, mockPoolRepo, mockPriceRepo)
			},
			query: dto.GetTokensQuery{
				IDs:        []string{"tokenaddress1", "tokenaddress4"},
				Extra:      true,
				PoolTokens: true,
			},
			result: &dto.GetTokensResult{
				Tokens: []*dto.GetTokensResultToken{
					{
						Name:      "name1",
						Decimals:  6,
						Symbol:    "symbol1",
						Type:      "type1",
						Price:     100000,
						Liquidity: 10000,
						LPAddress: "lpaddress1",
					},
					{
						Name:      "name4",
						Decimals:  6,
						Symbol:    "symbol4",
						Type:      "type4",
						Price:     400000,
						Liquidity: 40000,
						LPAddress: "lpaddress4",
						Pool: &dto.GetTokensResultTokenPool{
							Address:     "pooladdress1",
							TotalSupply: 0.00000000000001,
							ReserveUSD:  10000,
							Tokens: []*dto.GetTokensResultTokenPoolToken{
								{
									Name:     "name1",
									Symbol:   "symbol1",
									Decimals: 6,
									Weight:   50,
									Type:     "type1",
								},
								{
									Name:     "name2",
									Symbol:   "symbol2",
									Decimals: 6,
									Weight:   50,
									Type:     "type2",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "it should return correct error when get token failed",
			prepare: func(ctrl *gomock.Controller) *getTokensUseCase {
				mockTokenRepo := usecase.NewMockITokenRepository(ctrl)
				mockTokenRepo.
					EXPECT().
					FindByAddresses(gomock.Any(), []string{"tokenaddress1", "tokenaddress4"}).
					Return(nil, theErr)

				return &getTokensUseCase{tokenRepo: mockTokenRepo}
			},
			query: dto.GetTokensQuery{
				IDs: []string{"tokenaddress1", "tokenaddress4"},
			},
			result: nil,
			err:    theErr,
		},
		{
			name: "it should return correct error when get price failed",
			prepare: func(ctrl *gomock.Controller) *getTokensUseCase {
				mockTokenRepo := usecase.NewMockITokenRepository(ctrl)
				mockTokenRepo.
					EXPECT().
					FindByAddresses(gomock.Any(), []string{"tokenaddress1", "tokenaddress4"}).
					Return([]entity.Token{fixtures.Tokens[0], fixtures.Tokens[3]}, nil)

				mockPriceRepo := usecase.NewMockIPriceRepository(ctrl)
				mockPriceRepo.
					EXPECT().
					FindByAddresses(gomock.Any(), []string{"tokenaddress1", "tokenaddress4"}).
					Return(nil, theErr)

				return &getTokensUseCase{tokenRepo: mockTokenRepo, priceRepo: mockPriceRepo}
			},
			query: dto.GetTokensQuery{
				IDs: []string{"tokenaddress1", "tokenaddress4"},
			},
			result: nil,
			err:    theErr,
		},
		{
			name: "it should return correct error when get pools failed",
			prepare: func(ctrl *gomock.Controller) *getTokensUseCase {
				mockTokenRepo := usecase.NewMockITokenRepository(ctrl)
				mockTokenRepo.
					EXPECT().
					FindByAddresses(gomock.Any(), []string{"tokenaddress1", "tokenaddress4"}).
					Return([]entity.Token{fixtures.Tokens[0], fixtures.Tokens[3]}, nil)

				mockPriceRepo := usecase.NewMockIPriceRepository(ctrl)
				mockPriceRepo.
					EXPECT().
					FindByAddresses(gomock.Any(), []string{"tokenaddress1", "tokenaddress4"}).
					Return([]entity.Price{fixtures.Prices[0], fixtures.Prices[3]}, nil)

				mockPoolRepo := usecase.NewMockIPoolRepository(ctrl)
				mockPoolRepo.
					EXPECT().
					FindByAddresses(gomock.Any(), []string{"pooladdress1"}).
					Return(nil, theErr)

				return NewGetTokens(mockTokenRepo, mockPoolRepo, mockPriceRepo)
			},
			query: dto.GetTokensQuery{
				IDs:        []string{"tokenaddress1", "tokenaddress4"},
				PoolTokens: true,
			},
			result: nil,
			err:    theErr,
		},
		{
			name: "it should return correct error when get pool tokens failed",
			prepare: func(ctrl *gomock.Controller) *getTokensUseCase {
				mockTokenRepo := usecase.NewMockITokenRepository(ctrl)
				mockTokenRepo.
					EXPECT().
					FindByAddresses(gomock.Any(), []string{"tokenaddress1", "tokenaddress4"}).
					Return([]entity.Token{fixtures.Tokens[0], fixtures.Tokens[3]}, nil)

				mockTokenRepo.
					EXPECT().
					FindByAddresses(gomock.Any(), []string{"tokenaddress2"}).
					Return(nil, theErr)

				mockPriceRepo := usecase.NewMockIPriceRepository(ctrl)
				mockPriceRepo.
					EXPECT().
					FindByAddresses(gomock.Any(), []string{"tokenaddress1", "tokenaddress4"}).
					Return([]entity.Price{fixtures.Prices[0], fixtures.Prices[3]}, nil)

				mockPoolRepo := usecase.NewMockIPoolRepository(ctrl)
				mockPoolRepo.
					EXPECT().
					FindByAddresses(gomock.Any(), []string{"pooladdress1"}).
					Return([]entity.Pool{fixtures.Pools[0]}, nil)

				return NewGetTokens(mockTokenRepo, mockPoolRepo, mockPriceRepo)
			},
			query: dto.GetTokensQuery{
				IDs:        []string{"tokenaddress1", "tokenaddress4"},
				PoolTokens: true,
			},
			result: nil,
			err:    theErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			getTokens := tc.prepare(ctrl)

			result, err := getTokens.Handle(context.Background(), tc.query)
			if result != nil || tc.result != nil {
				assert.ElementsMatch(t, tc.result.Tokens, result.Tokens)
			}
			assert.ErrorIs(t, tc.err, err)
		})
	}
}
