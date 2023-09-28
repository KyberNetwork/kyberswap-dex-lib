package params

type (
	DecodeSwapDataParams struct {
		EncodedData string `json:"data"`
		DecoderType string `json:"decoderType"`
	}
)
