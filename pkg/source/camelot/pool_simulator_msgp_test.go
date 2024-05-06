package camelot

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	pools := []*PoolSimulator{
		{
			Pool: pool.Pool{
				Info: pool.PoolInfo{
					Tokens: []string{
						"0x5979d7b546e38e414f7e9822514be443a4800529",
						"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
					},
					Reserves: []*big.Int{
						bignumber.NewBig("1481252219344464578434"),
						bignumber.NewBig("3236537897421945761324"),
					},
				},
			},
			Token0FeePercent:     big.NewInt(40),
			Token1FeePercent:     big.NewInt(40),
			PrecisionMultiplier0: big.NewInt(1000000000000000000),
			PrecisionMultiplier1: big.NewInt(1000000000000000000),
			StableSwap:           true,
			FeeDenominator:       big.NewInt(100000),
		},
		{
			Pool: pool.Pool{
				Info: pool.PoolInfo{
					Tokens: []string{
						"0x5979d7b546e38e414f7e9822514be443a4800529",
						"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
					},
					Reserves: []*big.Int{
						bignumber.NewBig("1481252219344464578434"),
						bignumber.NewBig("3236537897421945761324"),
					},
				},
			},
			Token0FeePercent:     big.NewInt(40),
			Token1FeePercent:     big.NewInt(40),
			PrecisionMultiplier0: big.NewInt(1000000000000000000),
			PrecisionMultiplier1: big.NewInt(1000000000000000000),
			StableSwap:           true,
			FeeDenominator:       big.NewInt(100000),
		},
		{
			Pool: pool.Pool{
				Info: pool.PoolInfo{
					Tokens: []string{
						"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
						"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
					},
					Reserves: []*big.Int{
						bignumber.NewBig("4446200649353806287147"),
						bignumber.NewBig("7387929715114"),
					},
				},
			},
			Token0FeePercent:     big.NewInt(300),
			Token1FeePercent:     big.NewInt(300),
			PrecisionMultiplier0: big.NewInt(1000000000000000000),
			PrecisionMultiplier1: big.NewInt(1000000),
			StableSwap:           false,
			FeeDenominator:       big.NewInt(100000),
		},
		{
			Pool: pool.Pool{
				Info: pool.PoolInfo{
					Tokens: []string{
						"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
						"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
					},
					Reserves: []*big.Int{
						bignumber.NewBig("4446169910492564197660"),
						bignumber.NewBig("7387985285550"),
					},
				},
			},
			Token0FeePercent:     big.NewInt(300),
			Token1FeePercent:     big.NewInt(300),
			PrecisionMultiplier0: big.NewInt(1000000000000000000),
			PrecisionMultiplier1: big.NewInt(1000000),
			StableSwap:           false,
			FeeDenominator:       big.NewInt(100000),
		},
		{
			Pool: pool.Pool{
				Info: pool.PoolInfo{
					Tokens: []string{
						"0x5979d7b546e38e414f7e9822514be443a4800529",
						"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
					},
					Reserves: []*big.Int{
						bignumber.NewBig("1118199781455197144456"),
						bignumber.NewBig("2468857930499458348529"),
					},
				},
			},
			Token0FeePercent:     big.NewInt(40),
			Token1FeePercent:     big.NewInt(40),
			PrecisionMultiplier0: big.NewInt(1000000000000000000),
			PrecisionMultiplier1: big.NewInt(1000000000000000000),
			StableSwap:           true,
			Factory: &Factory{
				FeeTo:         [20]byte{0x01, 0x02},
				OwnerFeeShare: big.NewInt(40000),
			},
			FeeDenominator: big.NewInt(100000),
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
