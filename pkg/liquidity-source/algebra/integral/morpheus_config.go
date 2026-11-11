package integral

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

// Morpheus is the Algebra Integral 1.2.3 deployment on Robinhood Chain (chain id 4663).
// Pool-service should enable an algebra-integral source with these values:
//
//	exchange / DexID: valueobject.ExchangeMorpheus ("morpheus")
//	subgraphAPI:      MorpheusAnalyticsSubgraph
//	tickLensAddress:  MorpheusTickLens
//	useBasePluginV2:  true  (pools use Algebra BasePluginV2 dynamic fees)
const (
	MorpheusFactory = "0x10253594A832f967994b44f33411940533302ACb"
	MorpheusPoolDeployer = "0xd7cB0E0692f2D55A17bA81c1fE5501D66774fC4A"
	MorpheusTickLens = "0x0BFaCE9a5c9F884a4f09fadB83b69e81EA41424B"
	// Public Algebra analytics subgraph used by the Morpheus DEX frontend.
	MorpheusAnalyticsSubgraph =
		"https://api.goldsky.com/api/public/project_cm8pwdzcow9bu01xm6gdhatu4/subgraphs/analytics/v1.0.0/gn"
)

// MorpheusConfig returns the recommended algebra-integral Config for Robinhood / Morpheus.
func MorpheusConfig() *Config {
	return &Config{
		DexID:              valueobject.ExchangeMorpheus,
		SubgraphAPI:        MorpheusAnalyticsSubgraph,
		AllowSubgraphError: false,
		AlwaysUseTickLens:  false,
		TickLensAddress:    MorpheusTickLens,
		UseBasePluginV2:    true,
	}
}
