package usecase

import (
	aevmclient "github.com/KyberNetwork/aevm/client"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	aevmfinder "github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/aevm"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/hillclimb"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/retryfinder"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/spfav2"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolmanager"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func NewPathFinder(
	aevmClient aevmclient.Client,
	poolsPublisher poolmanager.IPoolsPublisher,
	config getroute.AggregatorConfig,
) findroute.IFinder {
	finderOptions := config.FinderOptions
	var baseFinder findroute.IFinder

	baseFinder = spfav2.NewSPFAv2Finder(
		finderOptions.MaxHops,
		config.WhitelistedTokenSet,
		finderOptions.DistributionPercent,
		finderOptions.MaxPathsInRoute,
		finderOptions.MaxPathsToGenerate,
		finderOptions.MaxPathsToReturn,
		finderOptions.MinPartUSD,
		finderOptions.MinThresholdAmountInUSD,
		finderOptions.MaxThresholdAmountInUSD,
		config.DexUseAEVM,
	)

	if finderOptions.Type == valueobject.FinderTypes.RetryDynamicPools {
		baseFinder = retryfinder.NewRetryFinder(baseFinder)
	}

	if config.FeatureFlags.IsHillClimbEnabled {
		baseFinder = hillclimb.NewHillClimbingFinder(
			finderOptions.HillClimbDistributionPercent,
			finderOptions.HillClimbIteration,
			finderOptions.HillClimbMinPartUSD,
			baseFinder,
		)
	}

	baseFinder = aevmfinder.NewAEVMFinder(baseFinder, aevmClient, poolsPublisher, finderOptions)

	return baseFinder
}
