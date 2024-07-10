package bancorv21

type Config struct {
	DexID string `json:"dexID"`

	ConverterRegistry    string `json:"converterRegistry"`
	BancorNetworkAddress string `json:"bancorNetworkAddress"`
}
