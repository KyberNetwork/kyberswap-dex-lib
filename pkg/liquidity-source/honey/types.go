package honey

import "github.com/holiman/uint256"

type Extra struct {
	RegisteredAssets       []string       `json:"registeredAssets"`
	IsBasketEnabledMint    bool           `json:"isBasketEnabledMint"`
	IsBasketEnabledRedeem  bool           `json:"isBasketEnabledRedeem"`
	ForceBasketMode        bool           `json:"forceBasketMode"`
	IsPegged               []bool         `json:"isPegged"`
	IsBadCollateral        []bool         `json:"isBadCollateral"`
	MintRates              []*uint256.Int `json:"mintRates"`
	RedeemRates            []*uint256.Int `json:"redeemRates"`
	VaultsDecimals         []uint8        `json:"vaultsDecimals"`
	AssetsDecimals         []uint8        `json:"assetsDecimals"`
	PolFeeCollectorFeeRate *uint256.Int   `json:"polFeeCollectorFeeRate"`
}
