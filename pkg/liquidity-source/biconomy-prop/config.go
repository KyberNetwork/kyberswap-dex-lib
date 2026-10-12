package biconomyprop

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	// Venue is the PropAMMVenue, 0x0000005880209CB9387C5904237ECeF97158DB69 on Base and BSC.
	Venue string `json:"venue"`
}
