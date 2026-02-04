package someswapv2

type Config struct {
	DexID               string `json:"dexID,omitempty"`
	Factory             string `json:"factory"`
	Router              string `json:"router"`
	Quoter              string `json:"quoter"`
	LPFeeManager        string `json:"lpFeeManager"`
	LiquidityLocker     string `json:"liquidityLocker"`
	CoreModule          string `json:"coreModule"`
	PermissionsRegistry string `json:"permissionsRegistry"`
	NewPoolLimit        int    `json:"newPoolLimit,omitempty"`
}
