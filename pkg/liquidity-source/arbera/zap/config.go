package arberazap

type Config struct {
	DexID string             `json:"dexID"`
	Pools map[string]PoolCfg `json:"pools"`
}

type PoolCfg struct {
	LeftToken  string `json:"leftToken"`
	VaultToken string `json:"vaultToken"`
	Den1Token  string `json:"den1Token"`
	Den2Token  string `json:"den2Token"`
	RightToken string `json:"rightToken"`
	LstToken   string `json:"lstToken"`
	DenAmmPool string `json:"denAmmPool"`
}
