package ekubov3

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestConfigSupportsTwammV1AndV2(t *testing.T) {
	t.Parallel()

	v1Twamm := common.HexToAddress("0xd4F1060cB9c1A13e1d2d20379b8aa2cF7541eD9b")
	v2Twamm := common.HexToAddress("0xd47f1B1eDCfEaBb08F6eBd8FC337c27E636C75BA")
	cfg := &Config{
		Twamm: TwammConfig{
			V1: TwammDeployment{
				Address:     v1Twamm,
				DataFetcher: "v1-data-fetcher",
			},
			V2: TwammDeployment{
				Address:     v2Twamm,
				DataFetcher: "v2-data-fetcher",
			},
		},
	}

	require.Equal(t, ExtensionTypeTwamm, cfg.ExtensionType(v1Twamm))
	require.Equal(t, ExtensionTypeTwamm, cfg.ExtensionType(v2Twamm))
	require.Equal(t, "v1-data-fetcher", cfg.TwammDataFetcher(v1Twamm))
	require.Equal(t, "v2-data-fetcher", cfg.TwammDataFetcher(v2Twamm))
	require.Empty(t, cfg.TwammDataFetcher(common.HexToAddress("0x1111111111111111111111111111111111111111")))
}

func TestConfigSupportsVe33(t *testing.T) {
	t.Parallel()

	ve33 := common.HexToAddress("0xd100000000000000000000000000000000000000")
	cfg := &Config{Ve33: ve33}

	require.Equal(t, ExtensionTypeVe33, cfg.ExtensionType(ve33))
	require.Equal(t, ExtensionTypeNoSwapCallPoints, cfg.ExtensionType(common.HexToAddress("0x1")))
}
