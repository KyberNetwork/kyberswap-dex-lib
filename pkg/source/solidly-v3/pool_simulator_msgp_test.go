package solidlyv3

import (
	"encoding/json"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	rawPools := []string{
		`{
			"address": "0x6146be494fee4c73540cb1c5f87536abf1452500",
			"swapFee": 100,
			"type": "solidly-v3",
			"timestamp": 1705358961,
			"reserves": [
				"137746578201",
				"1484208757880"
			],
			"tokens": [
				{
					"address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					"name": "USD Coin",
					"symbol": "USDC",
					"decimals": 6,
					"weight": 50,
					"swappable": true
				},
				{
					"address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
					"name": "Tether USD",
					"symbol": "USDT",
					"decimals": 6,
					"weight": 50,
					"swappable": true
				}
			],
			"extra": "{\"liquidity\":2187336922123374,\"sqrtPriceX96\":79257523413855207489606556516,\"tickSpacing\":1,\"tick\":7,\"ticks\":[{\"index\":-51,\"liquidityGross\":4023735441,\"liquidityNet\":4023735441},{\"index\":-20,\"liquidityGross\":5377610146132,\"liquidityNet\":5377610146132},{\"index\":-15,\"liquidityGross\":210582075003910,\"liquidityNet\":210582075003910},{\"index\":-12,\"liquidityGross\":22472177151754,\"liquidityNet\":22472177151754},{\"index\":-10,\"liquidityGross\":74074635783340,\"liquidityNet\":74074635783340},{\"index\":-8,\"liquidityGross\":433140695370165,\"liquidityNet\":433140695370165},{\"index\":-7,\"liquidityGross\":39366634237420,\"liquidityNet\":39366634237420},{\"index\":-6,\"liquidityGross\":500,\"liquidityNet\":500},{\"index\":-5,\"liquidityGross\":426569798759873,\"liquidityNet\":372153403538201},{\"index\":-3,\"liquidityGross\":1219258653378841,\"liquidityNet\":1219258653378841},{\"index\":-2,\"liquidityGross\":22472177151754,\"liquidityNet\":-22472177151754},{\"index\":-1,\"liquidityGross\":895763774632318,\"liquidityNet\":740462860612512},{\"index\":0,\"liquidityGross\":1110632836915778,\"liquidityNet\":-1110632836915778},{\"index\":1,\"liquidityGross\":87397254001105,\"liquidityNet\":19516406989539},{\"index\":3,\"liquidityGross\":191372132878096,\"liquidityNet\":177871973029250},{\"index\":5,\"liquidityGross\":1059772830149054,\"liquidityNet\":406901047912696},{\"index\":6,\"liquidityGross\":997092488198795,\"liquidityNet\":-997092488198795},{\"index\":7,\"liquidityGross\":596352227500000,\"liquidityNet\":596352227500000},{\"index\":8,\"liquidityGross\":1139269436790704,\"liquidityNet\":-1139269436790704},{\"index\":9,\"liquidityGross\":649809057995322,\"liquidityNet\":-649809057995322},{\"index\":10,\"liquidityGross\":398254403601907,\"liquidityNet\":-398254403601907},{\"index\":50,\"liquidityGross\":4023735441,\"liquidityNet\":-4023735441}]}"
		}`,
	}
	var err error
	poolEntities := make([]*entity.Pool, len(rawPools))
	for i, rawPool := range rawPools {
		poolEntities[i] = new(entity.Pool)
		err = json.Unmarshal([]byte(rawPool), poolEntities[i])
		require.NoError(t, err)
	}
	pools := make([]*PoolSimulator, len(poolEntities))
	for i, poolEntity := range poolEntities {
		pools[i], err = NewPoolSimulator(*poolEntity, valueobject.ChainIDEthereum)
		require.NoError(t, err)
	}
	for _, pool := range pools {
		b, err := pool.MarshalMsg(nil)
		require.NoError(t, err)
		actual := new(PoolSimulator)
		_, err = actual.UnmarshalMsg(b)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(pool, actual, testutil.CmpOpts(PoolSimulator{})...))
	}
}
