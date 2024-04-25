//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple Token

package entity

type Token struct {
	Address     string `json:"address"`
	Symbol      string `json:"symbol"`
	Name        string `json:"name"`
	Decimals    uint8  `json:"decimals"`
	CgkID       string `json:"cgkId"` // = "API id" field in Coingecko token info
	Type        string `json:"type"`
	PoolAddress string `json:"poolAddress"`
}
