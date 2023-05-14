package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
)

func mockPoolTokensTestIndexPools() []*entity.PoolToken {
	token1 := &entity.PoolToken{
		Address:   "tokenaddress1",
		Name:      "tokenName1",
		Symbol:    "tokenSymbol1",
		Decimals:  6,
		Weight:    50,
		Swappable: true,
	}
	token2 := &entity.PoolToken{
		Address:   "tokenaddress2",
		Name:      "tokenName2",
		Symbol:    "tokenSymbol2",
		Decimals:  6,
		Weight:    50,
		Swappable: true,
	}
	token3 := &entity.PoolToken{
		Address:   "tokenaddress3",
		Name:      "tokenName3",
		Symbol:    "tokenSymbol3",
		Decimals:  6,
		Weight:    50,
		Swappable: false,
	}
	token4 := &entity.PoolToken{
		Address:   "tokenaddress4",
		Name:      "tokenName4",
		Symbol:    "tokenSymbol4",
		Decimals:  6,
		Weight:    50,
		Swappable: true,
	}
	token5 := &entity.PoolToken{
		Address:   "tokenaddress5",
		Name:      "tokenName5",
		Symbol:    "tokenSymbol5",
		Decimals:  6,
		Weight:    50,
		Swappable: true,
	}
	return []*entity.PoolToken{token1, token2, token3, token4, token5}
}

func mockPoolsTestIndexPools() []entity.Pool {
	poolTokens := mockPoolTokensTestIndexPools()
	return []entity.Pool{
		{
			Address:      "pooladdress1",
			ReserveUsd:   10000,
			AmplifiedTvl: 10,
			SwapFee:      100,
			Exchange:     "exchange1",
			Type:         "type1",
			Timestamp:    1658373335,
			Reserves:     []string{"10000", "20000"},
			Tokens:       []*entity.PoolToken{poolTokens[0], poolTokens[1], poolTokens[2]},
			Extra:        "extra1",
			StaticExtra:  "staticExtra1",
			TotalSupply:  "10000",
		},
		{
			Address:      "pooladdress2",
			ReserveUsd:   20000,
			AmplifiedTvl: 0,
			SwapFee:      200,
			Exchange:     "exchange2",
			Type:         constant.PoolTypes.CurveMeta,
			Timestamp:    1658373335,
			Reserves:     []string{"20000", "30000"},
			Tokens:       []*entity.PoolToken{poolTokens[1], poolTokens[0]},
			Extra:        "extra2",
			StaticExtra:  fmt.Sprintf(`{"underlyingTokens": ["%s", "%s"]}`, poolTokens[3].Address, poolTokens[4].Address),
			TotalSupply:  "20000",
		},
		{
			Address:      "pooladdress3",
			ReserveUsd:   0,
			AmplifiedTvl: 30,
			SwapFee:      300,
			Exchange:     "exchange3",
			Type:         constant.PoolTypes.CurveAave,
			Timestamp:    1658373335,
			Reserves:     []string{},
			Tokens:       []*entity.PoolToken{poolTokens[0], poolTokens[1]},
			Extra:        "extra2",
			StaticExtra:  fmt.Sprintf(`{"underlyingTokens": ["%s", "%s"]}`, poolTokens[3].Address, poolTokens[4].Address),
			TotalSupply:  "30000",
		},
		{
			Address:      "pooladdress4",
			ReserveUsd:   0,
			AmplifiedTvl: 0,
			SwapFee:      300,
			Exchange:     "exchange3",
			Type:         constant.PoolTypes.CurveAave,
			Timestamp:    1658373335,
			Reserves:     []string{},
			Tokens:       []*entity.PoolToken{poolTokens[0], poolTokens[1]},
			Extra:        "extra2",
			StaticExtra:  fmt.Sprintf(`{"underlyingTokens": ["%s", "%s"]}`, poolTokens[3].Address, poolTokens[4].Address),
			TotalSupply:  "30000",
		},
	}
}

func TestIndexPools_Handle(t *testing.T) {
	t.Parallel()

	type TestCase struct {
		name    string
		prepare func(ctrl *gomock.Controller) *IndexPoolsUseCase
		command dto.IndexPoolsCommand
		result  *dto.IndexPoolsResult
	}

	theError := errors.New("some error")

	mockTokens := mockPoolTokensTestIndexPools()
	mockPools := mockPoolsTestIndexPools()

	testCases := []TestCase{
		{
			name: "it should return nil result when no pool failed to index",
			prepare: func(ctrl *gomock.Controller) *IndexPoolsUseCase {
				mockConfig := IndexPoolsConfig{
					WhitelistedTokenSet: map[string]bool{
						mockTokens[0].Address: true,
						mockTokens[2].Address: true,
						mockTokens[3].Address: true,
					},
					ChunkSize: 100,
				}

				mockPoolRepo := usecase.NewMockIPoolRepository(ctrl)
				mockPoolRepo.EXPECT().
					FindByAddresses(
						gomock.Any(),
						[]string{
							mockPools[0].Address,
							mockPools[1].Address,
							mockPools[2].Address,
							mockPools[3].Address,
							"pooladdress5",
						},
					).Return(mockPools, nil)

				mockPoolRankRepo := usecase.NewMockIPoolRankRepository(ctrl)
				mockPoolRankRepo.EXPECT().AddToSortedSetScoreByTvl(
					gomock.Any(),
					mockPools[0],
					mockTokens[0].Address,
					mockTokens[1].Address,
					true,
					false,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSetScoreByAmplifiedTvl(
					gomock.Any(),
					mockPools[0],
					mockTokens[0].Address,
					mockTokens[1].Address,
					true,
					false,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSetScoreByTvl(
					gomock.Any(),
					mockPools[1],
					mockTokens[1].Address,
					mockTokens[0].Address,
					false,
					true,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSetScoreByTvl(
					gomock.Any(),
					mockPools[1],
					mockTokens[3].Address,
					mockTokens[4].Address,
					true,
					false,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSetScoreByAmplifiedTvl(
					gomock.Any(),
					mockPools[2],
					mockTokens[0].Address,
					mockTokens[1].Address,
					true,
					false,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSetScoreByAmplifiedTvl(
					gomock.Any(),
					mockPools[2],
					mockTokens[3].Address,
					mockTokens[4].Address,
					true,
					false,
				).Return(nil)

				return NewIndexPoolsUseCase(mockPoolRepo, mockPoolRankRepo, mockConfig)
			},
			command: dto.IndexPoolsCommand{PoolAddresses: []string{
				mockPools[0].Address,
				mockPools[1].Address,
				mockPools[2].Address,
				mockPools[3].Address,
				"pooladdress5",
			}},
			result: nil,
		},
		{
			name: "it should return correct failed pool addresses when some pools were failed to index",
			prepare: func(ctrl *gomock.Controller) *IndexPoolsUseCase {
				mockConfig := IndexPoolsConfig{
					WhitelistedTokenSet: map[string]bool{
						mockTokens[0].Address: true,
						mockTokens[2].Address: true,
						mockTokens[3].Address: true,
					},
					ChunkSize: 100,
				}

				mockPoolRepo := usecase.NewMockIPoolRepository(ctrl)
				mockPoolRepo.EXPECT().
					FindByAddresses(
						gomock.Any(),
						[]string{
							mockPools[0].Address,
							mockPools[1].Address,
							mockPools[2].Address,
							mockPools[3].Address,
							"pooladdress5",
						},
					).Return(mockPools, nil)

				mockPoolRankRepo := usecase.NewMockIPoolRankRepository(ctrl)
				mockPoolRankRepo.EXPECT().AddToSortedSetScoreByTvl(
					gomock.Any(),
					mockPools[0],
					mockTokens[0].Address,
					mockTokens[1].Address,
					true,
					false,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSetScoreByAmplifiedTvl(
					gomock.Any(),
					mockPools[0],
					mockTokens[0].Address,
					mockTokens[1].Address,
					true,
					false,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSetScoreByTvl(
					gomock.Any(),
					mockPools[1],
					mockTokens[1].Address,
					mockTokens[0].Address,
					false,
					true,
				).Return(theError)
				mockPoolRankRepo.EXPECT().AddToSortedSetScoreByTvl(
					gomock.Any(),
					mockPools[1],
					mockTokens[3].Address,
					mockTokens[4].Address,
					true,
					false,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSetScoreByAmplifiedTvl(
					gomock.Any(),
					mockPools[2],
					mockTokens[0].Address,
					mockTokens[1].Address,
					true,
					false,
				).Return(nil)
				mockPoolRankRepo.EXPECT().AddToSortedSetScoreByAmplifiedTvl(
					gomock.Any(),
					mockPools[2],
					mockTokens[3].Address,
					mockTokens[4].Address,
					true,
					false,
				).Return(nil)

				return NewIndexPoolsUseCase(mockPoolRepo, mockPoolRankRepo, mockConfig)
			},
			command: dto.IndexPoolsCommand{PoolAddresses: []string{
				mockPools[0].Address,
				mockPools[1].Address,
				mockPools[2].Address,
				mockPools[3].Address,
				"pooladdress5",
			}},
			result: dto.NewIndexPoolsResult([]string{"pooladdress2"}),
		},
		{
			name: "it should return correct failed pool addresses when repository returns error",
			prepare: func(ctrl *gomock.Controller) *IndexPoolsUseCase {
				mockConfig := IndexPoolsConfig{
					ChunkSize: 100,
				}

				mockPoolRepo := usecase.NewMockIPoolRepository(ctrl)
				mockPoolRepo.EXPECT().
					FindByAddresses(gomock.Any(), []string{"pooladdress1", "pooladdress2", "pooladdress3"}).
					Return(nil, theError)
				mockPoolRankRepo := usecase.NewMockIPoolRankRepository(ctrl)
				return NewIndexPoolsUseCase(mockPoolRepo, mockPoolRankRepo, mockConfig)
			},
			command: dto.IndexPoolsCommand{PoolAddresses: []string{"pooladdress1", "pooladdress2", "pooladdress3"}},
			result:  dto.NewIndexPoolsResult([]string{"pooladdress1", "pooladdress2", "pooladdress3"}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			indexPools := tc.prepare(ctrl)

			result := indexPools.Handle(context.Background(), tc.command)

			assert.Equal(t, tc.result, result)
		})
	}
}
