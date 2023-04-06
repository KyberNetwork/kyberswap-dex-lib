package entity

import (
	"encoding/json"
)

const TokenEntity = "token"
const TokenKey = "tokens"

type Token struct {
	Address     string `json:"address"`
	Symbol      string `json:"symbol"`
	Name        string `json:"name"`
	Decimals    uint8  `json:"decimals"`
	CgkID       string `json:"cgkId"` // = "API id" field in Coingecko token info
	Type        string `json:"type"`
	PoolAddress string `json:"poolAddress"`
}

func (t Token) Encode() string {
	bytes, _ := json.Marshal(t)

	return string(bytes)
}

func DecodeToken(key, member string) Token {
	var t Token
	err := json.Unmarshal([]byte(member), &t)

	if err != nil {
		return Token{}
	}

	t.Address = key

	return t
}
