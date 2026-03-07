package obric

type Config struct {
	DexId        string `json:"dexId"`
	Factory      string `json:"factory"`
	NewPoolLimit int    `json:"newPoolLimit"`
}
