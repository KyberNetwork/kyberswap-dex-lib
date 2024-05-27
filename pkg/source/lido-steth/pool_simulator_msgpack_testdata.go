package lido_steth

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/samber/lo"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	var pools []*PoolSimulator
	{
		tokens := []string{"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", "0xae7ab96520de3a18e5e111b5eaab095312d7fe84"}
		p, err := NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: []string{"1", "1"},
			Tokens:   lo.Map(tokens, func(adr string, _ int) *entity.PoolToken { return &entity.PoolToken{Address: adr} }),
		}, valueobject.ChainIDEthereum)
		if err != nil {
			panic(err)
		}
		pools = append(pools, p)
	}
	return pools
}
