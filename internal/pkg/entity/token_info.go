package entity

type TokenInfo struct {
	Address    string `mapstructure:"address"`
	IsFOT      bool   `mapstructure:"isFOT"`
	IsHoneypot bool   `mapstructure:"isHoneypot"`
}
