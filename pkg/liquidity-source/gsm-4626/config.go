package gsm4626

type Config struct {
	DexId      string   `json:"dexId"`
	AavePoolV3 string   `json:"aavePoolV3"`
	GSMs       []string `json:"gsms"`
}
