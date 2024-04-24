package mantisswap

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
			"address": "0x62ba5e1ab1fa304687f132f67e35bfc5247166ad",
			"type": "mantisswap",
			"timestamp": 1705354354,
			"reserves": [
				"3206954397",
				"4036310239",
				"1749719254748797676026"
			],
			"tokens": [
				{
					"address": "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
					"weight": 1,
					"swappable": true
				},
				{
					"address": "0xc2132d05d31c914a87c6611c10748aeb04b58e8f",
					"weight": 1,
					"swappable": true
				},
				{
					"address": "0x8f3cf7ad23cd3cadbd9735aff958023239c6a063",
					"weight": 1,
					"swappable": true
				}
			],
			"extra": "{\"Paused\":false,\"SwapAllowed\":true,\"BaseFee\":100,\"LpRatio\":50,\"SlippageA\":8,\"SlippageN\":16,\"SlippageK\":1000000000000000000,\"LPs\":{\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":{\"address\":\"0xe03aec0d08b3158350a9ab99f6cea7ba9513b889\",\"decimals\":6,\"asset\":3206954397,\"liability\":3082104986,\"liabilityLimit\":2000000000000},\"0x8f3cf7ad23cd3cadbd9735aff958023239c6a063\":{\"address\":\"0x4b3bfcaa4f8bd4a276b81c110640da634723e64b\",\"decimals\":18,\"asset\":1749719254748797676026,\"liability\":2538765916906832854207,\"liabilityLimit\":2000000000000000000000000},\"0xc2132d05d31c914a87c6611c10748aeb04b58e8f\":{\"address\":\"0xe8a1ead2f4c454e319b76fa3325b754c47ce1820\",\"decimals\":6,\"asset\":4036310239,\"liability\":2921143438,\"liabilityLimit\":2000000000000}}}"
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
