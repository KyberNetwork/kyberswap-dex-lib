package getroute

import (
	aevmclient "github.com/KyberNetwork/aevm/client"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
)

func initAdditionHopFinderEngines(config Config, aevmClient aevmclient.Client) (finderEngine.IPathFinderEngine, finderEngine.IPathFinderEngine) {
	oneAdditionHopConfig := config
	oneAdditionHopConfig.Aggregator.FinderOptions.MaxHops = config.Aggregator.FinderOptions.MaxHops + 1

	// We skip checking error here because we are sure that the config is valid
	oneHopFinder, oneHopFinalizer, _ := InitializeFinderEngine(oneAdditionHopConfig, aevmClient)
	oneHopFinderEngine := finderEngine.NewPathFinderEngine(oneHopFinder, oneHopFinalizer)

	twoAdditionHopsConfig := config
	twoAdditionHopsConfig.Aggregator.FinderOptions.MaxHops = config.Aggregator.FinderOptions.MaxHops + 2

	twoHopFinder, twoHopFinalizer, _ := InitializeFinderEngine(twoAdditionHopsConfig, aevmClient)
	twoHopsFinderEngine := finderEngine.NewPathFinderEngine(twoHopFinder, twoHopFinalizer)

	return oneHopFinderEngine, twoHopsFinderEngine
}
