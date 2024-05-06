package balancerv1

import (
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	pools := []*PoolSimulator{
		{
			records: map[string]Record{
				"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
					Bound:   true,
					Balance: number.NewUint256("181453339134494385762"),
					Denorm:  number.NewUint256("25000000000000000000"),
				},
				"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": {
					Bound:   true,
					Balance: number.NewUint256("982184296"),
					Denorm:  number.NewUint256("25000000000000000000"),
				},
			},
			publicSwap: true,
			swapFee:    number.NewUint256("4000000000000000"),
			totalAmountsIn: map[string]*uint256.Int{
				"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": uint256.NewInt(0),
				"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": uint256.NewInt(0),
			},
			maxTotalAmountsIn: map[string]*uint256.Int{
				"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
				"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
			},
		},
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
