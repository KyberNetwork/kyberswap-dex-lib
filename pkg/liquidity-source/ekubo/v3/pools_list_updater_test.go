package ekubov3

import (
	"context"
	"slices"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var MainnetConfig = &Config{
	DexId:            DexType,
	ChainId:          valueobject.ChainIDEthereum,
	SubgraphAPI:      "https://api.studio.thegraph.com/query/1718652/ekubo-v-3/version/latest",
	Core:             common.HexToAddress("0x00000000000014aA86C5d3c41765bb24e11bd701"),
	Oracle:           common.HexToAddress("0x517E506700271AEa091b02f42756F5E174Af5230"),
	Twamm:            common.HexToAddress("0xd4F1060cB9c1A13e1d2d20379b8aa2cF7541eD9b"),
	MevCapture:       common.HexToAddress("0x5555fF9Ff2757500BF4EE020DcfD0210CFfa41Be"),
	QuoteDataFetcher: "0x5a3f0f1da4ac0c4b937d5685f330704c8e8303f1",
	TwammDataFetcher: "0xc07e5b80750247c8b5d7234a9c79dfc58785392b",
}

func TestPoolListUpdater(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	plUpdater := NewPoolListUpdater(MainnetConfig, ethrpc.New("https://ethereum.drpc.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
		graphql.NewClient(MainnetConfig.SubgraphAPI))

	newPools, _, err := plUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(newPools), 0)

	testPk := pools.NewPoolKey(
		common.HexToAddress(valueobject.ZeroAddress),
		common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
		pools.NewPoolConfig(common.Address{}, 9223372036854775, pools.NewConcentratedPoolTypeConfig(1000)),
	)

	require.True(t, slices.ContainsFunc(newPools, func(el entity.Pool) bool {
		var staticExtra StaticExtra
		err := json.Unmarshal([]byte(el.StaticExtra), &staticExtra)
		require.NoError(t, err)

		pk := staticExtra.PoolKey

		return pk.Token0.Cmp(testPk.Token0) == 0 && pk.Token1.Cmp(testPk.Token1) == 0 &&
			pk.Config.Compressed() == testPk.Config.Compressed()
	}))
}
