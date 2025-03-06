package skypsm

type Config struct {
	DexID      string   `json:"dexID"`
	PsmAddress string   `json:"psmAddress"`
	Tokens     []string `json:"tokens"`
}
