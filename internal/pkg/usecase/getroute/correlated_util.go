package getroute

import (
	"strings"

	aevmclient "github.com/KyberNetwork/aevm/client"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
)

const tokenPairSeparator = "-"

func splitTokenPair(tokenPair string) []string {
	return strings.Split(tokenPair, tokenPairSeparator)
}

func convertCorrelatedPairsMap(correlatedPairs map[string]string) map[string]map[string]string {
	correlatedPairsMap := make(map[string]map[string]string)

	for pairStr, poolAddress := range correlatedPairs {
		tokens := splitTokenPair(pairStr)
		if len(tokens) != 2 {
			continue
		}

		token0 := tokens[0]
		token1 := tokens[1]

		if _, ok := correlatedPairsMap[token0]; !ok {
			correlatedPairsMap[token0] = make(map[string]string)
		}

		if _, ok := correlatedPairsMap[token1]; !ok {
			correlatedPairsMap[token1] = make(map[string]string)
		}

		correlatedPairsMap[token0][token1] = poolAddress
		correlatedPairsMap[token1][token0] = poolAddress
	}

	return correlatedPairsMap
}

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
