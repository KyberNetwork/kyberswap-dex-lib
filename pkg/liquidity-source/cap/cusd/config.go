package cusd

type Config struct {
	DexId    string `json:"dexId"`
	Vault    string `json:"vault"`
	Oracle   string `json:"oracle"`
	Executor string `json:"executor"`
}
