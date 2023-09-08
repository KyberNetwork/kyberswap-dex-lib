package limitorder

type Config struct {
	DexID             string `json:"dexID"`
	LimitOrderHTTPUrl string `json:"limitOrderHTTPUrl"`
	ChainID           uint   `json:"chainID"`
	SupportMultiSCs   bool   `json:"supportMultiSCs"`

	ContractAddresses []string `json:"contractAddresses"`
}
