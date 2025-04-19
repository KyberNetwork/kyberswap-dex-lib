package erc4626

type Config struct {
	DexId      string   `json:"dexId"`
	Vault      string   `json:"vault"`
	ShareToken string   `json:"shareToken"`
	AssetToken string   `json:"assetToken"`
	Gas        Gas      `json:"gas"`
	SwapTypes  SwapType `json:"swapTypes"`
}
