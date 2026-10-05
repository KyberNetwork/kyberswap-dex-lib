package biconomyprop

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	// Venue is the PropAMMVenue, 0x0000008792fE035f85b03593e10cF8ee59e69Fa2 on Base and BSC.
	Venue string `json:"venue"`
}
