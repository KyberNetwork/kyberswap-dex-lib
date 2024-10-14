package getcustomroute

import (
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
)

// ReplaceAggregatorConfig replaces the aggregator config with the custom route config.
func ReplaceAggregatorConfig(
	config getroute.Config,
	customrouteConfig Config,
) getroute.Config {
	config.Aggregator.FinderOptions.MaxHops = customrouteConfig.Aggregator.FinderOptions.MaxHops

	return config
}
