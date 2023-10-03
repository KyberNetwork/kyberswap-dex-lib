package generatepath

import (
	"context"

	"k8s.io/apimachinery/pkg/util/sets"
)

func (uc *useCase) listPoolMultiTokenOuts(
	ctx context.Context,
	tokenInAddress string,
	tokenAddresses []string,
) (sets.String, error) {
	allPoolIDs := sets.NewString()

	for _, tokenOutAddress := range tokenAddresses {
		bestPools, err := uc.poolRankRepository.FindBestPoolIDs(ctx, tokenInAddress, tokenOutAddress, uc.config.GetBestPoolsOptions)
		if err != nil {
			return nil, err
		}

		allPoolIDs.Insert(bestPools...)
	}

	return allPoolIDs, nil
}
