package altfun

const defaultAPIURL = "https://api.alt.fun/api/v1/tokens"

type Config struct {
	DexID                string `json:"dexID"`
	ZapAddress           string `json:"zapAddress"`
	BondingAddress       string `json:"bondingAddress"`
	FactoryAddress       string `json:"factoryAddress"`
	GlobalStorageAddress string `json:"globalStorageAddress"`
	APIURL               string `json:"apiURL"`
	NewPoolLimit         int    `json:"newPoolLimit"`
}
