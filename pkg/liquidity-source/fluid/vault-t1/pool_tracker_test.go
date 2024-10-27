package vaultT1

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPoolTracker(t *testing.T) {
	// @dev test is guaranteed to work on block (because liquidation is available)
	// const testBlockNumber = uint64(20812089)
	t.Skip()

	_ = logger.SetLogLevel("debug")

	var (
		config = Config{
			VaultLiquidationResolver: "0x6Cd1E75b524D3CCa4c3320436d6F09e24Dadd613",
		}
	)

	logger.Debugf("Starting TestPoolTracker with config: %+v", config)

	client := ethrpc.New("https://ethereum.kyberengineering.io")
	client.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	poolTracker := NewPoolTracker(&config, client)
	require.NotNil(t, poolTracker)
	logger.Debugf("PoolTracker initialized: %+v", poolTracker)

	t.Run("wstETH_weETH_Pool", func(t *testing.T) {
		poolAddr := "0x40D9b8417E6E1DcD358f04E3328bCEd061018A82"
		testPool := entity.Pool{
			Address:  poolAddr,
			Exchange: string(valueobject.ExchangeFluidVaultT1),
			Type:     DexType,
			Reserves: entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{
					Address:   "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
					Name:      "wstETH",
					Symbol:    "wstETH",
					Decimals:  18,
					Weight:    1,
					Swappable: true,
				},
				{
					Address:   "0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee",
					Name:      "weETH",
					Symbol:    "weETH",
					Decimals:  18,
					Weight:    1,
					Swappable: true,
				},
			},
		}

		logger.Debugf("Testing wstETH_weETH_Pool with address: %s", poolAddr)

		newPool, err := poolTracker.GetNewPoolState(context.Background(), testPool, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		logger.Debugf("GetNewPoolState completed for wstETH_weETH_Pool, new pool: %+v", newPool)

		require.Equal(t, poolAddr, newPool.Address)

		require.Len(t, newPool.Reserves, 2)
		for i, reserve := range newPool.Reserves {
			require.NotEqual(t, "0", reserve, "Reserve should not be zero")
			logger.Debugf("wstETH_weETH_Pool Reserve[%d]: %s", i, reserve)
		}

		var extra struct {
			WithAbsorb bool     `json:"withAbsorb"`
			Ratio      *big.Int `json:"ratio"`
		}
		err = json.Unmarshal([]byte(newPool.Extra), &extra)
		require.NoError(t, err)
		logger.Debugf("Unmarshaled extra data for wstETH_weETH_Pool: %+v", extra)

		require.NotNil(t, extra.WithAbsorb)
		require.NotNil(t, extra.Ratio)
		require.IsType(t, &big.Int{}, extra.Ratio)

		jsonEncoded, _ := json.MarshalIndent(newPool, "", "  ")
		t.Logf("Updated wstETH-weETH Pool: %s\n", string(jsonEncoded))
	})

	t.Run("USDC_ETH_Pool", func(t *testing.T) {
		poolAddr := "0x0C8C77B7FF4c2aF7F6CEBbe67350A490E3DD6cB3"
		testPool := entity.Pool{
			Address: poolAddr,
			Type:    DexType,
			Tokens: []*entity.PoolToken{
				{Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"}, // USDC
				{Address: "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"}, // ETH
			},
		}
		logger.Debugf("Testing USDC_ETH_Pool with address: %s", poolAddr)

		newPool, err := poolTracker.GetNewPoolState(context.Background(), testPool, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		logger.Debugf("GetNewPoolState completed for USDC_ETH_Pool, new pool: %+v", newPool)

		require.Equal(t, poolAddr, newPool.Address)

		require.Len(t, newPool.Reserves, 2)
		for i, reserve := range newPool.Reserves {
			require.Equal(t, "0", reserve, "Reserve should be zero")
			logger.Debugf("USDC_ETH_Pool Reserve[%d]: %s", i, reserve)
		}

		var extra struct {
			WithAbsorb bool     `json:"withAbsorb"`
			Ratio      *big.Int `json:"ratio"`
		}
		err = json.Unmarshal([]byte(newPool.Extra), &extra)
		require.NoError(t, err)
		logger.Debugf("Unmarshaled extra data for USDC_ETH_Pool: %+v", extra)

		require.NotNil(t, extra.WithAbsorb)
		require.NotNil(t, extra.Ratio)
		require.IsType(t, &big.Int{}, extra.Ratio)

		jsonEncoded, _ := json.MarshalIndent(newPool, "", "  ")
		t.Logf("Updated USDC_ETH Pool: %s\n", string(jsonEncoded))
	})

}
