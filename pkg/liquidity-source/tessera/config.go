package tessera

type Config struct {
	DexId             string `json:"dexId"`
	TesseraEngine     string `json:"tesseraEngine"`
	TesseraIndexer    string `json:"tesseraIndexer"`
	TesseraSwap       string `json:"tesseraSwap"`
	MaxPrefetchPoints int    `json:"maxPrefetchPoints"`
}
