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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var MainnetConfig = Config{
	ChainId: valueobject.ChainIDEthereum,
	HTTPConfig: HTTPConfig{
		BaseURL:    "https://eth-mainnet-api.ekubo.org",
		Timeout:    durationjson.Duration{Duration: 10 * time.Second},
		RetryCount: 1,
	},
	Core:        "0xe0e0e08A6A4b9Dc7bD67BCB7aadE5cF48157d444",
	DataFetcher: "0x91cB8a896cAF5e60b1F7C4818730543f849B408c",
	Router:      "0x9995855C00494d039aB6792f18e368e530DFf931",
	Extensions: map[string]pool.ExtensionType{
		"0x51d02a5948496a67827242eabc5725531342527c": pool.Oracle,
	},
}

func TestPoolListUpdater(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	plUpdater := NewPoolListUpdater(&MainnetConfig, ethrpc.New("https://eth.drpc.org"))

	newPools, _, err := plUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(newPools), 0)

	testPk := quoting.NewPoolKey(
		valueobject.ZeroAddress,
		"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		quoting.Config{
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

	newPools, _, err = plUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, len(newPools), 0)
}
