package fraxswap

import (
	"fmt"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/samber/lo"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	var pools []*PoolSimulator
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: []string{"20", "20"},
			Tokens:   lo.Map([]string{"a", "b"}, func(adr string, _ int) *entity.PoolToken { return &entity.PoolToken{Address: adr} }),
			Extra:    fmt.Sprintf("{\"reserve0\": %v, \"reserve1\": %v, \"fee\": %v}", "20", "20", 9997),
		})
		if err != nil {
			panic(err)
		}
		pools = append(pools, p)
	}
	return pools
}
