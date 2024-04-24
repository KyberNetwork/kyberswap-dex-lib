package wombatmain

import (
	"encoding/json"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	rawPools := []string{
		`{
			"address": "0xa45c0abeef67c363364e0e73832df9986aba3800",
			"type": "wombat-main",
			"timestamp": 1705358001,
			"reserves": [
				"27437517755",
				"104442256607"
			],
			"tokens": [
				{
					"address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					"decimals": 6,
					"weight": 50,
					"swappable": true
				},
				{
					"address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
					"decimals": 6,
					"weight": 50,
					"swappable": true
				}
			],
			"extra": "{\"paused\":false,\"haircutRate\":20000000000000,\"ampFactor\":250000000000000,\"startCovRatio\":1500000000000000000,\"endCovRatio\":1800000000000000000,\"assetMap\":{\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\":{\"isPause\":false,\"address\":\"0x6966553568634F4225330D559a8783DE7649C7D3\",\"cash\":27437517755009846067811,\"liability\":46154718915224891070477,\"underlyingTokenDecimals\":6,\"relativePrice\":null},\"0xdac17f958d2ee523a2206206994597c13d831ec7\":{\"isPause\":false,\"address\":\"0x752945079a0446AA7efB6e9E1789751cDD601c95\",\"cash\":104442256607693995284288,\"liability\":69617497322874416078864,\"underlyingTokenDecimals\":6,\"relativePrice\":null}}}"
		}`,
	}
	poolEntites := make([]entity.Pool, len(rawPools))
	for i, rawPool := range rawPools {
		err := json.Unmarshal([]byte(rawPool), &poolEntites[i])
		require.NoError(t, err)
	}
	var err error
	pools := make([]*PoolSimulator, len(poolEntites))
	for i, poolEntity := range poolEntites {
		pools[i], err = NewPoolSimulator(poolEntity)
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
