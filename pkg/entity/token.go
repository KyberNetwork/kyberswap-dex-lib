package entity

type Token struct {
	Address     string `json:"address"`
	Symbol      string `json:"symbol"`
	Name        string `json:"name"`
	Decimals    uint8  `json:"decimals"`
	CgkID       string `json:"cgkId"` // = "API id" field in Coingecko token info
	Type        string `json:"type"`
	PoolAddress string `json:"poolAddress"`
}

type SimplifiedToken struct {
	Address  string `json:"address"`
	Decimals uint8  `json:"decimals"`
}
