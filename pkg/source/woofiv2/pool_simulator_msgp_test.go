package woofiv2

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
			"address": "0x3b3e4b4741e91af52d0e9ad8660573e951c88524",
			"type": "woofi-v2",
			"timestamp": 1705358688,
			"reserves": [
				"31282179327010309344344",
				"639257231329655513",
				"3188807719",
				"431310663958",
				"1056570480673"
			],
			"tokens": [
				{
					"address": "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
					"weight": 1,
					"swappable": true
				},
				{
					"address": "0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab",
					"weight": 1,
					"swappable": true
				},
				{
					"address": "0x152b9d0fdc40c096757f570a51e494bd4b943e50",
					"weight": 1,
					"swappable": true
				},
				{
					"address": "0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7",
					"weight": 1,
					"swappable": true
				},
				{
					"address": "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e",
					"weight": 1,
					"swappable": true
				}
			],
			"extra": "{\"quoteToken\":\"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e\",\"unclaimedFee\":1416472393,\"wooracle\":\"0xc13843aE0D2C5ca9E0EfB93a78828446D8173d19\",\"tokenInfos\":{\"0x152b9d0fdc40c096757f570a51e494bd4b943e50\":{\"reserve\":3188807719,\"feeRate\":25,\"decimals\":8,\"state\":{\"price\":4270690000000,\"spread\":1509220000000000,\"coeff\":1677660000,\"woFeasible\":true,\"decimals\":8,\"cloPrice\":4268113575240,\"cloPreferred\":false}},\"0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab\":{\"reserve\":639257231329655513,\"feeRate\":25,\"decimals\":18,\"state\":{\"price\":251868000000,\"spread\":1237180000000000,\"coeff\":1684050000,\"woFeasible\":true,\"decimals\":8,\"cloPrice\":251369732133,\"cloPreferred\":false}},\"0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7\":{\"reserve\":431310663958,\"feeRate\":5,\"decimals\":6,\"state\":{\"price\":99927658,\"spread\":120028000000000,\"coeff\":1677720000,\"woFeasible\":true,\"decimals\":8,\"cloPrice\":99936938,\"cloPreferred\":false}},\"0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7\":{\"reserve\":31282179327010309344344,\"feeRate\":25,\"decimals\":18,\"state\":{\"price\":3574010066,\"spread\":1241160000000000,\"coeff\":2480100000,\"woFeasible\":true,\"decimals\":8,\"cloPrice\":3570116355,\"cloPreferred\":false}},\"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e\":{\"reserve\":1056570480673,\"feeRate\":0,\"decimals\":6,\"state\":{\"price\":100000000,\"spread\":0,\"coeff\":0,\"woFeasible\":true,\"decimals\":8,\"cloPrice\":100000000,\"cloPreferred\":false}}},\"timestamp\":1705358667,\"staleDuration\":300,\"bound\":10000000000000000}"
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
