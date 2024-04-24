package dodo

import (
	"fmt"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	var pools []*PoolSimulator
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			SwapFee:  0.001 + 0.002,
			Tokens:   []*entity.PoolToken{{Address: "BASE", Decimals: 18}, {Address: "QUOTE", Decimals: 18}},
			Extra: fmt.Sprintf("{\"reserves\": [%v, %v], \"targetReserves\": [%v, %v],\"i\": %v,\"k\": %v,\"rStatus\": %v,\"mtFeeRate\": \"%v\",\"lpFeeRate\": \"%v\" }",
				decStr(10), decStr(1000),
				decStr(10), decStr(1000),
				decStr(100),          // i=100
				"100000000000000000", // k=0.1
				0,
				"0.001",
				"0.002",
			),
			StaticExtra: fmt.Sprintf("{\"tokens\": [\"%v\",\"%v\"], \"type\": \"%v\", \"dodoV1SellHelper\": \"%v\"}",
				"BASE", "QUOTE", "DPP", ""),
		})
		require.NoError(t, err)
		pools = append(pools, p)
	}
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			SwapFee:  0.001 + 0.002,
			Tokens:   []*entity.PoolToken{{Address: "BASE", Decimals: 18}, {Address: "QUOTE", Decimals: 18}},
			Extra: fmt.Sprintf("{\"reserves\": [%v, %v], \"targetReserves\": [%v, %v],\"i\": %v,\"k\": %v,\"rStatus\": %v,\"mtFeeRate\": \"%v\",\"lpFeeRate\": \"%v\" }",
				decStr(10), decStr(1000),
				decStr(10), decStr(1000),
				decStr(100),          // i=100
				"100000000000000000", // k=0.1
				0,
				"0.001",
				"0.002",
			),
			StaticExtra: fmt.Sprintf("{\"tokens\": [\"%v\",\"%v\"], \"type\": \"%v\", \"dodoV1SellHelper\": \"%v\"}",
				"BASE", "QUOTE", "DPP", ""),
		})
		require.NoError(t, err)
		pools = append(pools, p)
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
