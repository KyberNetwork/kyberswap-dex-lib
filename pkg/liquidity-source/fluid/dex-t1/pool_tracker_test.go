package dexT1

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestPoolTracker(t *testing.T) {
	_ = logger.SetLogLevel("debug")

	var (
		config = Config{
			ChainID: valueobject.ChainIDEthereum,
		}
	)

	logger.Debugf("Starting TestPoolTracker with config: %+v", config)

	client := ethrpc.New("https://ethereum.kyberengineering.io")
	client.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	poolTracker := NewPoolTracker(&config, client)
	require.NotNil(t, poolTracker)
	logger.Debugf("PoolTracker initialized: %+v", poolTracker)

	t.Run("wstETH_ETH_Pool", func(t *testing.T) {
		poolAddr := "0x6d83f60eEac0e50A1250760151E81Db2a278e03a"
		testPool := entity.Pool{
			Address:  poolAddr,
			Exchange: string(valueobject.ExchangeFluidDexT1),
			Type:     DexType,
			Reserves: entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{
					Address:   "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
					Weight:    1,
					Swappable: true,
				},
				{
					Address:   "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
					Weight:    1,
					Swappable: false,
				},
			},
		}

		logger.Debugf("Testing wstETH_weETH_Pool with address: %s", poolAddr)

		newPool, err := poolTracker.GetNewPoolState(context.Background(), testPool, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		logger.Debugf("GetNewPoolState completed for wstETH_ETH_Pool, new pool: %+v", newPool)

		require.Equal(t, poolAddr, newPool.Address)

		require.NotEqual(t, "0", newPool.Reserves[0], "Reserve should not be zero")
		require.NotEqual(t, "0", newPool.Reserves[1], "Reserve should not be zero")

		var extra PoolExtra
		err = json.Unmarshal([]byte(newPool.Extra), &extra)
		require.NoError(t, err)
		require.True(t, extra.CollateralReserves.Token0RealReserves.Cmp(big.NewInt(0)) > 0)
		require.True(t, extra.CollateralReserves.Token1RealReserves.Cmp(big.NewInt(0)) > 0)
		require.True(t, extra.CollateralReserves.Token0ImaginaryReserves.Cmp(big.NewInt(0)) > 0)
		require.True(t, extra.CollateralReserves.Token1ImaginaryReserves.Cmp(big.NewInt(0)) > 0)
		require.True(t, extra.DebtReserves.Token0Debt.Cmp(big.NewInt(0)) > 0)
		require.True(t, extra.DebtReserves.Token1Debt.Cmp(big.NewInt(0)) > 0)
		require.True(t, extra.DebtReserves.Token0RealReserves.Cmp(big.NewInt(0)) > 0)
		require.True(t, extra.DebtReserves.Token1RealReserves.Cmp(big.NewInt(0)) > 0)
		require.True(t, extra.DebtReserves.Token0ImaginaryReserves.Cmp(big.NewInt(0)) > 0)
		require.True(t, extra.DebtReserves.Token1ImaginaryReserves.Cmp(big.NewInt(0)) > 0)

		jsonEncoded, _ := json.MarshalIndent(newPool, "", "  ")
		t.Logf("Updated wstETH-weETH Pool: %s\n", string(jsonEncoded))
	})

}
