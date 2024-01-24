package usecase

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/spfav2"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/retryfinder"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func NewFinder(config valueobject.FinderOptions, getBestPaths func(sourceHash uint64, tokenIn, tokenOut string) []*entity.MinimalPath, whiteListTokenSet map[string]bool) findroute.IFinder {

	switch config.Type {
	case valueobject.FinderTypes.RetryDynamicPools:
		baseFinder := spfav2.NewSPFAv2Finder(
			config.MaxHops,
			whiteListTokenSet,
			config.DistributionPercent,
			config.MaxPathsInRoute,
			config.MaxPathsToGenerate,
			config.MaxPathsToReturn,
			config.MinPartUSD,
			config.MinThresholdAmountInUSD,
			config.MaxThresholdAmountInUSD,
			getBestPaths,
		)
		return retryfinder.NewRetryFinder(baseFinder)
	default:
		routeFinder := spfav2.NewSPFAv2Finder(
			config.MaxHops,
			whiteListTokenSet,
			config.DistributionPercent,
			config.MaxPathsInRoute,
			config.MaxPathsToGenerate,
			config.MaxPathsToReturn,
			config.MinPartUSD,
			config.MinThresholdAmountInUSD,
			config.MaxThresholdAmountInUSD,
			getBestPaths,
		)
		return routeFinder
	}
}
