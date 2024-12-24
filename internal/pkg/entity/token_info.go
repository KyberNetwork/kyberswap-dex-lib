package entity

type TokenInfo struct {
	Address    string `json:"address"`
	IsFOT      bool   `json:"isFOT"`
	IsHoneypot bool   `json:"isHoneypot"`
}
