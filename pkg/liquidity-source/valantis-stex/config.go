package valantisstex

type Config struct {
	DexId string                `json:"dexId"`
	Stex  map[string]StexConfig `json:"stex"`
}

type StexConfig struct {
	Gas [2]uint64 `json:"gas"`
}
