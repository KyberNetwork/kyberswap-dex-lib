package common

import (
	"testing"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswap"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func TestMinHopToTokenOut(t *testing.T) {
	t.Run("test correctness of minHopToTokenOut", func(t *testing.T) {
		tokenByAddress := map[string]entity.Token{
			"a": {Address: "a"},
			"b": {Address: "b"},
			"c": {Address: "c"},
			"d": {Address: "d"},
			"e": {Address: "e"},
			"f": {Address: "f"},
		}
		poolEntityList := []entity.Pool{
			{
				Address:  "pool1",
				Tokens:   entity.PoolTokens{&entity.PoolToken{Address: "a"}, &entity.PoolToken{Address: "b"}},
				Reserves: entity.PoolReserves{"1", "1"},
			},
			{
				Address:  "pool2",
				Tokens:   entity.PoolTokens{&entity.PoolToken{Address: "a"}, &entity.PoolToken{Address: "c"}},
				Reserves: entity.PoolReserves{"1", "1"},
			},
			{
				Address:  "pool3",
				Tokens:   entity.PoolTokens{&entity.PoolToken{Address: "b"}, &entity.PoolToken{Address: "d"}},
				Reserves: entity.PoolReserves{"1", "1"},
			},
			{
				Address:  "pool4",
				Tokens:   entity.PoolTokens{&entity.PoolToken{Address: "c"}, &entity.PoolToken{Address: "e"}},
				Reserves: entity.PoolReserves{"1", "1"},
			},
			{
				Address:  "pool5",
				Tokens:   entity.PoolTokens{&entity.PoolToken{Address: "a"}, &entity.PoolToken{Address: "d"}},
				Reserves: entity.PoolReserves{"1", "1"},
			},
		}
		poolByAddress := make(map[string]poolpkg.IPoolSimulator)
		for _, poolEntity := range poolEntityList {
			pool, err := uniswap.NewPoolSimulator(poolEntity)
			assert.Nil(t, err)
			poolByAddress[pool.GetAddress()] = pool
		}
		tokenToPoolAddress := make(map[string][]string)
		for poolAddress, pool := range poolByAddress {
			for _, tokenAddress := range pool.GetTokens() {
				tokenToPoolAddress[tokenAddress] = append(tokenToPoolAddress[tokenAddress], poolAddress)
			}
		}
		tokenOut := "a"
		minHopsToTokenOut, err := MinHopsToTokenOut(poolByAddress, tokenByAddress, tokenToPoolAddress, tokenOut)
		assert.Nil(t, err)
		assert.EqualValues(t, map[string]uint32{"a": 0, "b": 1, "c": 1, "d": 1, "e": 2}, minHopsToTokenOut)
	})

	t.Run("stress test minHopToTokenOut", func(t *testing.T) {
		nTokens := 100_000
		nPools := 200_000
		tokenByAddress := GenerateRandomTokenByAddress(nTokens)
		var tokenAddressList []string
		for tokenAddress := range tokenByAddress {
			tokenAddressList = append(tokenAddressList, tokenAddress)
		}
		poolByAddress, err := GenerateRandomPoolByAddress(nPools, tokenAddressList)
		assert.Nil(t, err)
		tokenToPoolAddress := make(map[string][]string)
		for poolAddress, pool := range poolByAddress {
			for _, tokenAddress := range pool.GetTokens() {
				tokenToPoolAddress[tokenAddress] = append(tokenToPoolAddress[tokenAddress], poolAddress)
			}
		}
		tokenOut := tokenAddressList[RandInt(0, nTokens)]
		_, err = MinHopsToTokenOut(poolByAddress, tokenByAddress, tokenToPoolAddress, tokenOut)
		assert.Nil(t, err)
		// spew.Dump(minHopToTokenOut)
	})
}
