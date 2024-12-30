package usecase

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/repository/poolrank"
)

type RemovePoolIndexUseCase struct {
	poolRankRepo IPoolRankRepository
}

func NewRemovePoolIndexUseCase(repo IPoolRankRepository) *RemovePoolIndexUseCase {
	return &RemovePoolIndexUseCase{
		poolRankRepo: repo,
	}
}

func (u *RemovePoolIndexUseCase) RemovePoolAddressFromIndexes(ctx context.Context, addresses []string) error {
	if len(addresses) == 0 {
		return nil
	}

	// we don't have enough information to check if the pool belong to any indexes set, so we will send commands to all nativeTvl and amplifiedNativeTvl set
	err := u.poolRankRepo.RemoveAddressesFromWhitelistIndex(ctx, poolrank.SortByTVLNative, addresses, true)
	if err != nil {
		return err
	}

	err = u.poolRankRepo.RemoveAddressesFromWhitelistIndex(ctx, poolrank.SortByAmplifiedTVLNative, addresses, false)
	if err != nil {
		return err
	}

	return nil

}

func (u *RemovePoolIndexUseCase) RemovePoolAddressFromLiqScoreIndexes(ctx context.Context, addresses ...string) error {
	if len(addresses) == 0 {
		return nil
	}

	// we don't have enough information to check if the pool belong to any indexes set, so we will send commands to all nativeTvl and amplifiedNativeTvl set
	err := u.poolRankRepo.RemoveAddressesFromWhitelistIndex(ctx, poolrank.SortByLiquidityScoreTvl, addresses, true)
	if err != nil {
		return err
	}

	err = u.poolRankRepo.RemoveAddressesFromWhitelistIndex(ctx, poolrank.SortByLiquidityScore, addresses, false)
	if err != nil {
		return err
	}

	return nil

}
