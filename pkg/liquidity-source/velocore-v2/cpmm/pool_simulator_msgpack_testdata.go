package cpmm

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/goccy/go-json"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	rawPools := []string{
		`{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","exchange":"velocorev2-cpmm","type":"velocorev2-cpmm","timestamp":1697617544,"reserves":["5192296858534827628430578339644164","7774767","1310912514297980345043"],"tokens":[{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","swappable":true},{"address":"0xa219439258ca9da29e9cc4ce5596924745e12b93","swappable":true},{"address":"0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1","swappable":true}],"extra":"{\"fee1e9\":10000000,\"feeMultiplier\":0}","staticExtra":"{\"poolTokenNumber\":3,\"weights\":[2,1,1]}"}`,
	}
	var err error
	poolEntities := make([]*entity.Pool, len(rawPools))
	for i, rawPool := range rawPools {
		poolEntities[i] = new(entity.Pool)
		err = json.Unmarshal([]byte(rawPool), poolEntities[i])
		if err != nil {
			panic(err)
		}
	}
	pools := make([]*PoolSimulator, len(poolEntities))
	for i, poolEntity := range poolEntities {
		pools[i], err = NewPoolSimulator(*poolEntity)
		if err != nil {
			panic(err)
		}
	}
	return pools
}
