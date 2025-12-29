package tessera

type Config struct {
	DexId             string `json:"dexId"`
	TesseraTreasury   string `json:"tesseraTreasury"`
	TesseraEngine     string `json:"tesseraEngine"`
	TesseraIndexer    string `json:"tesseraIndexer"`
	TesseraSwap       string `json:"tesseraSwap"`
	MaxPrefetchPoints int    `json:"maxPrefetchPoints"`
}
