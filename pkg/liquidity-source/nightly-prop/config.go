package nightlyprop

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	// RouterAddress is the NightlyRouter proxy. It lists the pools, holds
	// the swap role on each pool and is the address that takers approve.
	RouterAddress string `json:"routerAddress"`
}
