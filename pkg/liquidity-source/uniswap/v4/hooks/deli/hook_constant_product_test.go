package deli

import (
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
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
	cpEntityPool entity.Pool
	_            = json.Unmarshal([]byte(`{"address":"0xb10b10215cb7e7ddf94ad812a427c6c99929b3f73b278604280c1a00b1f1bb9a","swapFee":1000,"exchange":"uniswap-v4","type":"uniswap-v4","timestamp":1761323571,"reserves":["380000000000000000","450000"],"tokens":[{"address":"0x4e74d4db6c0726ccded4656d0bce448876bb4c7a","symbol":"wBLT","decimals":18,"swappable":true},{"address":"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"liquidity\":0,\"sqrtPriceX96\":79228162514264337593543950336,\"tickSpacing\":1,\"tick\":0,\"ticks\":[]}","staticExtra":"{\"0x0\":[false,false],\"fee\":1000,\"tS\":1,\"hooks\":\"0x00c9da9abc5303219ead3cf0307b5a8a7644bac8\",\"uR\":\"0x6ff5693b99212da76ad316178a184ab56d299b43\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":37267112}`),
		&cpEntityPool)
	cpPoolSim = lo.Must(uniswapv4.NewPoolSimulator(cpEntityPool, 8453))
)

func TestCP_Track(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	var extra uniswapv4.Extra
	var staticExtra uniswapv4.StaticExtra
	require.NoError(t, json.Unmarshal([]byte(cpEntityPool.Extra), &extra))
	require.NoError(t, json.Unmarshal([]byte(cpEntityPool.StaticExtra), &staticExtra))
	hookParam := &uniswapv4.HookParam{
		RpcClient:   ethrpc.New("https://base.drpc.org"),
		Pool:        &cpEntityPool,
		HookAddress: staticExtra.HooksAddress,
	}
	hook, ok := uniswapv4.GetHook(staticExtra.HooksAddress, hookParam)
	require.True(t, ok)

	reserves, err := hook.GetReserves(ctx, hookParam)
	require.NoError(t, err)
	t.Log(reserves)
}

func TestCPCalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, cpPoolSim, map[int]map[int]map[string]string{
		0: {
			1: {
				"3758651075753225": "4403",
				"3800000000000000": "4451",
			},
		},
		1: {
			0: {
				"4451": "3718123998780767",
				"4500": "3758651075753225",
			},
		},
	})
}

func Test_CP_CloneState_UpdateBalance(t *testing.T) {
	cloned := cpPoolSim.CloneState()
	tokenAmountIn := pool.TokenAmount{
		Token:  "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		Amount: bignumber.NewBig10("4451"),
	}
	tokenOut := "0x4e74d4db6c0726ccded4656d0bce448876bb4c7a"
	result, err := cpPoolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmountIn,
		TokenOut:      tokenOut,
	})
	require.NoError(t, err)
	assert.Equal(t, bignumber.NewBig10("3718123998780767"), result.TokenAmountOut.Amount)
	cloned.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmountIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})

	result, err = cpPoolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmountIn,
		TokenOut:      tokenOut,
	})
	require.NoError(t, err)
	assert.Equal(t, bignumber.NewBig10("3718123998780767"), result.TokenAmountOut.Amount)

	result, err = cloned.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmountIn,
		TokenOut:      tokenOut,
	})
	require.NoError(t, err)
	assert.Equal(t, bignumber.NewBig10("3646033418782424"), result.TokenAmountOut.Amount)
}
