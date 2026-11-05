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
	// SupplyOnlySettlement selects the on-chain executor flow where D4 settles
	// S=B after each hop, then supply-only input/output endpoints.
	// It identifies executor behavior, not a setting on the Fluid DEX contract.
	SupplyOnlySettlement bool `json:"supplyOnlySettlement"`
}
