package composablestable

import (
	"math/big"
	"strconv"
	"testing"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	pools := []*PoolSimulator{
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{
						"0x00C2A4be503869Fa751c2DbcB7156cc970b5a8dA",
						"0xD4e7C1F3DA1144c9E2CfD1b015eDA7652b4a4399",
						"0xF71d0774B214c4cf51E33Eb3d30Ef98132e4DBaA",
					},
					Reserves: []*big.Int{
						bignumber.NewBig("2596148429267407814265248164610048"),
						bignumber.NewBig("6999791779383984752"),
						bignumber.NewBig("3000000000000000000"),
					},
				},
			},
			regularSimulator: &regularSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{
							"0x00C2A4be503869Fa751c2DbcB7156cc970b5a8dA",
							"0xD4e7C1F3DA1144c9E2CfD1b015eDA7652b4a4399",
							"0xF71d0774B214c4cf51E33Eb3d30Ef98132e4DBaA",
						},
						Reserves: []*big.Int{
							bignumber.NewBig("2596148429267407814265248164610048"),
							bignumber.NewBig("6999791779383984752"),
							bignumber.NewBig("3000000000000000000"),
						},
					},
				},
				swapFeePercentage: uint256.NewInt(100000000000000),
				scalingFactors: []*uint256.Int{
					uint256.NewInt(10000000000),
					uint256.NewInt(10000520578),
					uint256.NewInt(10000000000),
				},
				bptIndex: 0,
				amp:      uint256.NewInt(1500000),
			},
		},
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Address: "0x2Ba7Aa2213Fa2C909Cd9E46FeD5A0059542b36B0",
					Tokens: []string{
						"0x0000000000085d4780B73119b644AE5ecd22b376",
						"0x2Ba7Aa2213Fa2C909Cd9E46FeD5A0059542b36B0",
						"0xA13a9247ea42D743238089903570127DdA72fE44",
					},
					Reserves: []*big.Int{
						bignumber.NewBig("414101427485347"),
						bignumber.NewBig("2596148429267348622595662702661260"),
						bignumber.NewBig("1170046233780600"),
					},
				},
			},
			bptSimulator: &bptSimulator{
				poolTypeVer:    poolTypeVer1,
				bptIndex:       1,
				bptTotalSupply: uint256.MustFromDecimal("2596148429267348624180999930418421"),
				amp:            uint256.NewInt(600000),
				scalingFactors: []*uint256.Int{
					uint256.NewInt(1000000000000000000),
					uint256.NewInt(1000000000000000000),
					uint256.NewInt(366332019912307),
				},
				lastJoinExit: LastJoinExitData{
					LastJoinExitAmplification: uint256.NewInt(600000),
					LastPostJoinExitInvariant: uint256.MustFromDecimal("114012967613307699384"),
				},
				rateProviders: []string{
					"0x0000000000000000000000000000000000000000",
					"0x0000000000000000000000000000000000000000",
					"0xA13a9247ea42D743238089903570127DdA72fE44",
				},
				tokenRateCaches: []TokenRateCache{
					{},
					{},
					{
						Rate:     uint256.MustFromDecimal("1003857034775170156"),
						OldRate:  uint256.MustFromDecimal("1000977462514719154"),
						Duration: uint256.NewInt(1000),
						Expires:  uint256.NewInt(1677904371),
					},
				},
				swapFeePercentage: uint256.NewInt(100000000000000),
				protocolFeePercentageCache: map[string]*uint256.Int{
					strconv.FormatInt(int64(feeTypeSwap), 10):  uint256.NewInt(0),
					strconv.FormatInt(int64(feeTypeYield), 10): uint256.NewInt(0),
				},
				tokenExemptFromYieldProtocolFee: []bool{
					false, false, true,
				},
				inRecoveryMode: true,
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
