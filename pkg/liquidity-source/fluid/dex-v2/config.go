package dexv2

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID                string
	ChainID              valueobject.ChainID `json:"chainID"`
	Dex                  string              `json:"dex"`
	Resolver             string              `json:"resolver"`
	Liquidity            string              `json:"liquidity"`
	SubgraphAPI          string              `json:"subgraphAPI"`
	TrackLiquidityLimits bool                `json:"trackLiquidityLimits"`
	// SupplyOnlySettlement selects the executor flow from SC PRs #781 and #987:
	// D4 settles S=B after each hop, then supply-only input/output endpoints.
	// It identifies executor behavior, not a setting on the Fluid DEX contract.
	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/pull/781
	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/pull/987
	SupplyOnlySettlement bool `json:"supplyOnlySettlement"`
}
