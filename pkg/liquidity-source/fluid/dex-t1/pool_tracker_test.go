package dexT1

import (
	"context"
	"math/big"
	"os"
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
	_ = logger.SetLogLevel("debug")

	if os.Getenv("CI") != "" {
		t.Skip()
	}

	var (
		config = Config{
			DexReservesResolver: "0xF38082d58bF0f1e07C04684FF718d69a70f21e62",
		}
	)

	logger.Debugf("Starting TestPoolTracker with config: %+v", config)

	client := ethrpc.New("https://ethereum.kyberengineering.io")
	client.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	poolTracker := NewPoolTracker(&config, client)
	require.NotNil(t, poolTracker)
	logger.Debugf("PoolTracker initialized: %+v", poolTracker)

	t.Run("wstETH_ETH_Pool", func(t *testing.T) {
		poolAddr := "0x0B1a513ee24972DAEf112bC777a5610d4325C9e7"

		staticExtraBytes, _ := json.Marshal(&StaticExtra{
			DexReservesResolver: config.DexReservesResolver,
		})

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
					Decimals:  18,
				},
				{
					Address:   "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
					Weight:    1,
					Swappable: true,
					Decimals:  18,
				},
			},
			StaticExtra: string(staticExtraBytes),
		}

		logger.Debugf("Testing wstETH_ETH_Pool with address: %s", poolAddr)

		newPool, err := poolTracker.GetNewPoolState(context.Background(), testPool, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		logger.Debugf("GetNewPoolState completed for wstETH_ETH_Pool, new pool: %+v", newPool)

		var extra PoolExtra
		err = json.Unmarshal([]byte(newPool.Extra), &extra)
		require.NoError(t, err)
		require.Equal(t, poolAddr, newPool.Address)

		require.Equal(t, true, newPool.Tokens[0].Swappable)
		require.Equal(t, true, newPool.Tokens[1].Swappable)
		require.Equal(t, 0.01, newPool.SwapFee)

		reserve0, _ := new(big.Int).SetString(newPool.Reserves[0], 10)
		reserve1, _ := new(big.Int).SetString(newPool.Reserves[1], 10)

		require.True(t, reserve0.Sign() > 0)
		require.True(t, reserve1.Sign() > 0)

		require.True(t, extra.CollateralReserves.Token0RealReserves.Sign() > 0)
		require.True(t, extra.CollateralReserves.Token1RealReserves.Sign() > 0)
		require.True(t, extra.CollateralReserves.Token0ImaginaryReserves.Sign() > 0)
		require.True(t, extra.CollateralReserves.Token1ImaginaryReserves.Sign() > 0)
		require.True(t, extra.DebtReserves.Token0Debt.Sign() > 0)
		require.True(t, extra.DebtReserves.Token1Debt.Sign() > 0)
		require.True(t, extra.DebtReserves.Token0RealReserves.Sign() > 0)
		require.True(t, extra.DebtReserves.Token1RealReserves.Sign() > 0)
		require.True(t, extra.DebtReserves.Token0ImaginaryReserves.Sign() > 0)
		require.True(t, extra.CenterPrice.Sign() > 0)

		logger.Debugf("Reserve0: %s", newPool.Reserves[0])
		logger.Debugf("Reserve1: %s", newPool.Reserves[1])

		logger.Debugf("CenterPrice: %s", extra.CenterPrice.String())

		logger.Debugf("Debt Reserves: Token0Debt: %s", extra.DebtReserves.Token0Debt.String())
		logger.Debugf("Debt Reserves: Token1Debt: %s", extra.DebtReserves.Token1Debt.String())
		logger.Debugf("Debt Reserves: Token0RealReserves: %s", extra.DebtReserves.Token0RealReserves.String())
		logger.Debugf("Debt Reserves: Token1RealReserves: %s", extra.DebtReserves.Token1RealReserves.String())
		logger.Debugf("Debt Reserves: Token0ImaginaryReserves: %s", extra.DebtReserves.Token0ImaginaryReserves.String())
		logger.Debugf("Debt Reserves: Token1ImaginaryReserves: %s", extra.DebtReserves.Token1ImaginaryReserves.String())

		logger.Debugf("Collateral Reserves: Token0RealReserves: %s", extra.CollateralReserves.Token0RealReserves.String())
		logger.Debugf("Collateral Reserves: Token1RealReserves: %s", extra.CollateralReserves.Token1RealReserves.String())
		logger.Debugf("Collateral Reserves: Token0ImaginaryReserves: %s", extra.CollateralReserves.Token0ImaginaryReserves.String())
		logger.Debugf("Collateral Reserves: Token1ImaginaryReserves: %s", extra.CollateralReserves.Token1ImaginaryReserves.String())

		jsonEncoded, _ := json.MarshalIndent(newPool, "", "  ")
		t.Logf("Updated wstETH-ETH Pool: %s\n", string(jsonEncoded))
	})

}
