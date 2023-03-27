package entity

import (
	"strconv"
	"strings"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
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
	return utils.Join(t.Symbol, t.Name, t.Decimals, t.CgkID, t.Type, t.PoolAddress)
}

func DecodeToken(key, member string) Token {
	var t Token
	split := strings.Split(member, ":")
	t.Address = key
	t.Symbol = split[0]
	t.Name = split[1]
	decimals, _ := strconv.ParseUint(split[2], 10, 64)
	t.Decimals = uint8(decimals)
	t.CgkID = split[3]
	t.Type = split[4]

	if len(split) >= 6 {
		t.PoolAddress = split[5]
	}

	return t
}
