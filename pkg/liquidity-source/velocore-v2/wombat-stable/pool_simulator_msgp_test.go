package wombatstable

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/goccy/go-json"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	rawPools := []string{
		`{
			"address": "0x61cb3a0c59825464474ebb287a3e7d2b9b59d093",
			"type": "velocore-v2-wombat-stable",
			"timestamp": 1705576647,
			"reserves": [
				"11195773019488324321309",
				"9192257736"
			],
			"tokens": [
				{
					"address": "0x7d43aabc515c356145049227cee54b608342c0ad",
					"weight": 1,
					"swappable": true
				},
				{
					"address": "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
					"weight": 1,
					"swappable": true
				}
			],
			"extra": "{\"amp\":250000000000000,\"fee1e18\":100000000000000,\"lpTokenBalances\":{\"0x176211869ca2b568f2a7d4ee941e073a821ee1ff\":340282366920938463463374607416268936368,\"0x7d43aabc515c356145049227cee54b608342c0ad\":340282366920938458576602139746458171455},\"tokenInfo\":{\"0x176211869ca2b568f2a7d4ee941e073a821ee1ff\":{\"indexPlus1\":2,\"scale\":12},\"0x7d43aabc515c356145049227cee54b608342c0ad\":{\"indexPlus1\":1,\"scale\":0}}}",
			"staticExtra": "{\"vault\":\"0x1d0188c4B276A09366D05d6Be06aF61a73bC7535\",\"wrappers\":{\"0x1e1f509963a6d33e169d9497b11c7dbfe73b7f13\":\"0xb30e7a2e6f7389ca5ddc714da4c991b7a1dcc88e\",\"0xb79dd08ea68a908a97220c76d19a6aa9cbde4376\":\"0x3f006b0493ff32b33be2809367f5f6722cb84a7b\"}}",
			"blockNumber": 1711060
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
		pools[i], err = NewPoolSimulator(*poolEntity)
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
