package msgpack

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/st0x"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestST0xSimulatorSerialization(t *testing.T) {
	const token0 = "0x0000000000000000000000000000000000000001"
	const token1 = "0x0000000000000000000000000000000000000002"
	p := entity.Pool{
		Address:     "0xe26507ae709887aa01dab0e959ee4b3a8a1c025e5c29e1b2644b62ea950a4253",
		Reserves:    entity.PoolReserves{"1000", "2000"},
		Tokens:      []*entity.PoolToken{{Address: token0, Decimals: 18}, {Address: token1, Decimals: 18}},
		Extra:       `{"liquidity":0,"sqrtPriceX96":79228162514264337593543950336,"tick":0,"tickSpacing":1,"ticks":[],"hX":{"p":1000000000000000000}}`,
		StaticExtra: `{"hooks":"0x9af1a97021b92c47219d2fb24deba51c248a2a88","fee":0,"tS":1}`,
	}
	sim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainID(8453))
	require.NoError(t, err)
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: token0, Amount: big.NewInt(10)}, TokenOut: token1}
	result, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: params.TokenAmountIn, TokenAmountOut: *result.TokenAmountOut, Fee: *result.Fee, SwapInfo: result.SwapInfo,
	})
	encoded, err := EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{p.Address: sim})
	require.NoError(t, err)
	decoded, err := DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	restored := decoded[p.Address]
	actual, err := restored.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(10), actual.TokenAmountOut.Amount)
	// The input reserve grew to 1010 and must also survive serialization.
	reverse, err := restored.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: token1, Amount: big.NewInt(1010)}, TokenOut: token0,
	})
	require.NoError(t, err)
	require.Equal(t, big.NewInt(1010), reverse.TokenAmountOut.Amount)
	// The hook's updated output reserve is 1990, including after serialization.
	params.TokenAmountIn.Amount = big.NewInt(1991)
	_, err = restored.CalcAmountOut(params)
	require.ErrorIs(t, err, st0x.ErrInsufficientRsv)
}
