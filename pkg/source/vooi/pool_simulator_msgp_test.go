package vooi

import (
	"testing"

	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	pools := []*PoolSimulator{
		{
			a:      utils.NewBig("1000000000000000"),
			lpFee:  utils.NewBig("100000000000000"),
			paused: false,

			assetByToken: map[string]Asset{
				"0x176211869ca2b568f2a7d4ee941e073a821ee1ff": {
					Cash:        utils.NewBig("109755508503386757517651"),
					Liability:   utils.NewBig("111705981295320099096585"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("111566749817365287931821"),
					Decimals:    6,
					Token:       common.HexToAddress("0x176211869ca2b568f2a7d4ee941e073a821ee1ff"),
					Active:      true,
				},
				"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5": {
					Cash:        utils.NewBig("31982154523056912809541"),
					Liability:   utils.NewBig("32315214600033595639643"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("32294182514254242328808"),
					Decimals:    18,
					Token:       common.HexToAddress("0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5"),
					Active:      true,
				},
				"0xa219439258ca9da29e9cc4ce5596924745e12b93": {
					Cash:        utils.NewBig("108295197536453495651912"),
					Liability:   utils.NewBig("106011578468650285163146"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("105904081724488185350374"),
					Decimals:    6,
					Token:       common.HexToAddress("0xa219439258ca9da29e9cc4ce5596924745e12b93"),
					Active:      true,
				},
			},
			indexByToken: map[string]int{
				"0x176211869ca2b568f2a7d4ee941e073a821ee1ff": 0,
				"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5": 2,
				"0xa219439258ca9da29e9cc4ce5596924745e12b93": 1,
			},

			gas: defaultGas,
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
