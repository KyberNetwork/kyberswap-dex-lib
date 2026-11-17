package nightlyprop

import "github.com/ethereum/go-ethereum/common"

type StaticExtra struct {
	RouterAddress string `json:"routerAddress"`
}

// poolsListUpdaterMetadata is the cursor of GetNewPools: the pools that the router listed on the
// last call. It is a set, because getRoutes returns the pools in no fixed order.
type poolsListUpdaterMetadata struct {
	KnownPools []string `json:"knownPools"`
}

type poolTokens struct {
	Base  common.Address
	Quote common.Address
}
