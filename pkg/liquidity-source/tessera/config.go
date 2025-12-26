package tessera

type Config struct {
	DexId             string `json:"dexId"`
	EngineAddr        string `json:"engineAddr"`
	IndexerAddr       string `json:"indexerAddr"`
	RouterAddr        string `json:"routerAddr"`
	MaxPrefetchPoints int    `json:"maxPrefetchPoints"`
}
