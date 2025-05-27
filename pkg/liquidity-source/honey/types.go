package honey

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type Extra struct {
	RegisteredAssets       []string       `json:"registeredAssets"`
	IsBasketEnabledMint    bool           `json:"isBasketEnabledMint"`
	IsBasketEnabledRedeem  bool           `json:"isBasketEnabledRedeem"`
	ForceBasketMode        bool           `json:"forceBasketMode"`
	IsPegged               []bool         `json:"isPegged"`
	IsBadCollateral        []bool         `json:"isBadCollateral"`
	MintRates              []*uint256.Int `json:"mintRates"`
	RedeemRates            []*uint256.Int `json:"redeemRates"`
	Vaults                 []string       `json:"vaults"`
	VaultsDecimals         []uint8        `json:"vaultsDecimals"`
	VaultsMaxRedeems       []*uint256.Int `json:"vaultsMaxRedeems"`
	AssetsDecimals         []uint8        `json:"assetsDecimals"`
	PolFeeCollectorFeeRate *uint256.Int   `json:"polFeeCollectorFeeRate"`
}

type PoolItem struct {
	ID     string             `json:"id"`
	Type   string             `json:"type"`
	Tokens []entity.PoolToken `json:"tokens"`
}

type SwapInfo struct {
	deltaShares *uint256.Int
	assetIndex  int
}
