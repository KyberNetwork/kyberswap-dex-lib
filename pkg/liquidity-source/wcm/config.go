package wcm

type Config struct {
	DexID           string `json:"dexID"`
	ExchangeAddress string `json:"exchangeAddress"`
	RouterAddress   string `json:"routerAddress"`
	MaxOrderLevels  int    `json:"maxOrderLevels"`
}
