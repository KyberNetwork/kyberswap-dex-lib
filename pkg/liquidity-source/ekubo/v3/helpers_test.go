package ekubov3

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const chainIDSepolia = valueobject.ChainID(11155111)

var (
	MainnetConfig = NewConfig(
		DexType,
		valueobject.ChainIDEthereum,
		"https://api.studio.thegraph.com/query/1718652/ekubo-v-3/version/latest",
		common.HexToAddress("0x00000000000014aA86C5d3c41765bb24e11bd701"),
		common.HexToAddress("0x517E506700271AEa091b02f42756F5E174Af5230"),
		common.HexToAddress("0xd4F1060cB9c1A13e1d2d20379b8aa2cF7541eD9b"),
		common.HexToAddress("0x5555fF9Ff2757500BF4EE020DcfD0210CFfa41Be"),
		common.HexToAddress("0xd4B54d0ca6979Da05F25895E6e269E678ba00f9e"),
		common.HexToAddress("0xd26f20001a72a18C002b00e6710000d68700ce00"),
		"0x5a3f0f1da4ac0c4b937d5685f330704c8e8303f1",
		"0xc07e5b80750247c8b5d7234a9c79dfc58785392b",
		"0x7A2fF5819Dc71Bb99133a97c38dA512E60c30475",
	)
	SepoliaConfig = NewConfig(
		DexType,
		chainIDSepolia,
		"",
		common.HexToAddress("0x00000000000014aA86C5d3c41765bb24e11bd701"),
		common.HexToAddress("0x517E506700271AEa091b02f42756F5E174Af5230"),
		common.HexToAddress("0xd4F1060cB9c1A13e1d2d20379b8aa2cF7541eD9b"),
		common.HexToAddress("0x5555fF9Ff2757500BF4EE020DcfD0210CFfa41Be"),
		common.HexToAddress("0xd4B54d0ca6979Da05F25895E6e269E678ba00f9e"),
		common.HexToAddress("0xd26f20001a72a18C002b00e6710000d68700ce00"),
		"0x5a3f0f1da4ac0c4b937d5685f330704c8e8303f1",
		"0xc07e5b80750247c8b5d7234a9c79dfc58785392b",
		"0x7A2fF5819Dc71Bb99133a97c38dA512E60c30475",
	)
)

func anyPoolKey(
	token0 string,
	token1 string,
	extension string,
	fee uint64,
	poolTypeConfig pools.PoolTypeConfig,
) pools.AnyPoolKey {
	return pools.AnyPoolKey{PoolKey: pools.NewPoolKey(
		common.HexToAddress(token0),
		common.HexToAddress(token1),
		pools.NewPoolConfig(common.HexToAddress(extension), fee, poolTypeConfig),
	)}
}
