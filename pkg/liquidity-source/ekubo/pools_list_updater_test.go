package ekubo

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/pools"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var MainnetConfig = Config{
	DexId:   DexType,
	ChainId: valueobject.ChainIDEthereum,
	HTTPConfig: HTTPConfig{
		BaseURL:    "https://eth-mainnet-api.ekubo.org",
		Timeout:    durationjson.Duration{Duration: 10 * time.Second},
		RetryCount: 1,
	},
	Core:             common.HexToAddress("0xe0e0e08A6A4b9Dc7bD67BCB7aadE5cF48157d444"),
	Oracle:           common.HexToAddress("0x51d02a5948496a67827242eabc5725531342527c"),
	Twamm:            common.HexToAddress("0xd4279c050da1f5c5b2830558c7a08e57e12b54ec"),
	BasicDataFetcher: "0x91cB8a896cAF5e60b1F7C4818730543f849B408c",
	TwammDataFetcher: "0x8C4C1F26A9F26372b88f418A939044773eE5dC01",
	Router:           "0x9995855C00494d039aB6792f18e368e530DFf931",
}

func TestPoolListUpdater(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	plUpdater := NewPoolListUpdater(&MainnetConfig, ethrpc.New("https://ethereum.kyberengineering.io").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")))

	newPools, _, err := plUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(newPools), 0)

	testPk := pools.NewPoolKey(
		common.HexToAddress(valueobject.ZeroAddress),
		common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
		pools.PoolConfig{
			Fee:         0,
			TickSpacing: 0,
			Extension:   common.HexToAddress("0x51d02a5948496a67827242eabc5725531342527c"),
		},
	)

	require.True(t, slices.ContainsFunc(newPools, func(el entity.Pool) bool {
		var staticExtra StaticExtra
		err := json.Unmarshal([]byte(el.StaticExtra), &staticExtra)
		require.NoError(t, err)

		pk := staticExtra.PoolKey

		return pk.Token0.Cmp(testPk.Token0) == 0 && pk.Token1.Cmp(testPk.Token1) == 0 && slices.Equal(pk.Config.Compressed(), testPk.Config.Compressed())
	}))
}
