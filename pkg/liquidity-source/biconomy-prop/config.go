package biconomyprop

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	// Venue is the PropAMMVenue, 0x000000Da21a0f02b2626874870b6447Db220C1EF on Base and BSC.
	Venue string `json:"venue"`
}
