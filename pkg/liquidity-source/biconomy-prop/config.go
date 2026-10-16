package biconomyprop

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	// Venue is the PropAMMVenue, 0x0000009E5F98cDB4dd1E1d2E15B220217c88B3b8 on Base.
	Venue string `json:"venue"`
}
