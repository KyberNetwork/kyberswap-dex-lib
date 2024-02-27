package bancor_v21

type Config struct {
	DexID        string `json:"dexID"`
	NewPoolLimit int    `json:"newPoolLimit"`

	ConverterRegistry    string `json:"converterRegistry"`
	BancorNetworkAddress string `json:"bancorNetworkAddress"`
}
