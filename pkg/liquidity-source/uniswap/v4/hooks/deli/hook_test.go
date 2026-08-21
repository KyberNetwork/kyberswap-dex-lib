package deli

import (
	"context"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0x386b66a4321dc97d30edc35539bbdd21d0f93901ec236b48a763fc835f7e5c4c","swapFee":3000,"exchange":"uniswap-v4","type":"uniswap-v4","timestamp":1761402967,"reserves":["1314503995115196217","1488509"],"tokens":[{"address":"0x4e74d4db6c0726ccded4656d0bce448876bb4c7a","symbol":"wBLT","decimals":18,"swappable":true},{"address":"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"liquidity\":1398803570097,\"sqrtPriceX96\":84309090720919547100448,\"tickSpacing\":60,\"tick\":-275081,\"ticks\":[{\"index\":-887220,\"liquidityGross\":1398803570097,\"liquidityNet\":1398803570097},{\"index\":887220,\"liquidityGross\":1398803570097,\"liquidityNet\":-1398803570097}]}","staticExtra":"{\"0x0\":[false,false],\"fee\":8388608,\"tS\":60,\"hooks\":\"0xc384b99a6e5cd1a800b2d83ab71bab7bd712b0cc\",\"uR\":\"0x6ff5693b99212da76ad316178a184ab56d299b43\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":37306810}`),
		&entityPool)
	poolSim = lo.Must(uniswapv4.NewPoolSimulator(entityPool, 8453))

	ctx = context.Background()
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		0: {
			1: {
				"12936925243928912": "14420",
				"13145039951151962": "14650",
			},
		},
		1: {
			0: {
				"14650": "12734915516792294",
				"14885": "12936925243928912",
			},
		},
	})
}

func Test_CloneState_UpdateBalance(t *testing.T) {
	cloned := poolSim.CloneState()
	tokenAmountIn := pool.TokenAmount{
		Token:  "0x4e74d4db6c0726ccded4656d0bce448876bb4c7a",
		Amount: bignumber.NewBig10("12936925243928912"),
	}
	tokenOut := "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmountIn,
		TokenOut:      tokenOut,
	})
	require.NoError(t, err)
	assert.Equal(t, bignumber.NewBig10("14420"), result.TokenAmountOut.Amount)
	cloned.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmountIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})

	result, err = poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmountIn,
		TokenOut:      tokenOut,
	})
	require.NoError(t, err)
	assert.Equal(t, bignumber.NewBig10("14420"), result.TokenAmountOut.Amount)

	result, err = cloned.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmountIn,
		TokenOut:      tokenOut,
	})
	require.NoError(t, err)
	assert.Equal(t, bignumber.NewBig10("14143"), result.TokenAmountOut.Amount)
}
