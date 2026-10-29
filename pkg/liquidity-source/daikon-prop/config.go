package daikonprop

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	// Router is the PAMMRouter: the only address that moves tokens. Manager and vault are read from it.
	Router string `json:"router"`
	// PartnerID attributes swaps to us in swapData.
	PartnerID uint64 `json:"partnerId"`
	// FeePpm is the swapData fee in millionths of the quote-token amount, capped at MaxFeePpm.
	FeePpm uint64 `json:"feePpm"`
}
