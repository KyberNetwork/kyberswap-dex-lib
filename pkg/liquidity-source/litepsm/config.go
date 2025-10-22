package litepsm

type Config struct {
	DexID string               `json:"dexId"`
	PSMs  map[string]PSMConfig `json:"psms"`
}
