package msgpack

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2"
	tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax/virtual"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestUniswapV2PoolSimulatorRoundTrip(t *testing.T) {
	t.Parallel()

	const (
		token0 = "0x0b3e328455c4059eeb9e3f84b5543f74e24e7e1b"
		token1 = "0xff8104251e7761163fac3211ef5583fb3f8583d6"
	)

	extra, err := json.Marshal(uniswapv2.Extra{
		Fee:          3,
		FeePrecision: 1000,
		TaxInfo: &tokentax.TaxInfo{
			Protocol:   virtual.Protocol,
			Token:      token1,
			BuyTaxBps:  uint256.NewInt(100),
			SellTaxBps: uint256.NewInt(100),
			Checked:    true,
		},
	})
	require.NoError(t, err)

	entityPool := entity.Pool{
		Address: "0x0000000000000000000000000000000000000001",
		Type:    uniswapv2.DexType,
		Tokens: []*entity.PoolToken{
			{Address: token0},
			{Address: token1},
		},
		Reserves: entity.PoolReserves{
			"64759685176877841920",
			"2092201468546951388637",
		},
		Extra: string(extra),
	}
	simulator, err := uniswapv2.NewPoolSimulator(entityPool)
	require.NoError(t, err)

	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  token0,
			Amount: big.NewInt(1_000_000_000_000_000_000),
		},
		TokenOut: token1,
	}
	before, err := simulator.CalcAmountOut(params)
	require.NoError(t, err)

	encoded, err := EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{
		entityPool.Address: simulator,
	})
	require.NoError(t, err)
	decoded, err := DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)

	after, err := decoded[entityPool.Address].CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, before.TokenAmountOut.Amount, after.TokenAmountOut.Amount)
	require.Equal(t, before.SwapInfo, after.SwapInfo)
}
