package equalizer

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
			"address": "0xf3f1f5760a614b8146eec5d1c94658720c2425b9",
			"swapFee": 0.002666666666666667,
			"type": "equalizer",
			"timestamp": 1705345162,
			"reserves": [
				"173810100394741222630",
				"441959784673"
			],
			"tokens": [
				{
					"address": "0x4200000000000000000000000000000000000006",
					"decimals": 18,
					"weight": 50,
					"swappable": true
				},
				{
					"address": "0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca",
					"decimals": 6,
					"weight": 50,
					"swappable": true
				}
			],
			"staticExtra": "{\"stable\":false}"
		}`,
	}
	poolEntites := make([]entity.Pool, len(rawPools))
	for i, rawPool := range rawPools {
		require.NoError(t, json.Unmarshal([]byte(rawPool), &poolEntites[i]))
	}
	var err error
	pools := make([]*PoolSimulator, len(rawPools))
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
