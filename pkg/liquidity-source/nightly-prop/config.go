package nightlyprop

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	// RouterAddress is the NightlyRouter proxy. It lists the pools, holds
	// the swap role on each pool and is the address that takers approve.
	RouterAddress string `json:"routerAddress"`
	// PartnerAddress is the contract that calls the router's swapWithFees.
	// It needs the router's PARTNER_ROLE and a fee recipient. The tracker
	// reads each pool's state for it.
	PartnerAddress string `json:"partnerAddress"`
}
