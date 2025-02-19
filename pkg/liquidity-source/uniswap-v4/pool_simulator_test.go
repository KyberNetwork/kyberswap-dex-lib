package uniswapv4_test

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/stretchr/testify/assert"
)

var (
	//go:embed pool_simulator_test_pool_data_20240213.json
	poolData string
)

func TestPoolSimulator(t *testing.T) {
	var (
		chainID = 1
		poolEnt entity.Pool
	)
	assert.NoError(t, json.Unmarshal([]byte(poolData), &poolEnt))

	t.Log(poolEnt)
	pSim, err := uniswapv4.NewPoolSimulator(poolEnt, valueobject.ChainID(chainID))
	assert.NoError(t, err)

	out, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: uniswapv4.NewBig10("1000000000000000000"),
		},
		TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
	})
	assert.NoError(t, err)

	t.Logf("TokenAmountOut: %s", out.TokenAmountOut.Amount.String())
	t.Logf("Fee: %s", out.Fee.Amount.String())
	t.Logf("RemainingTokenAmountIn: %s", out.RemainingTokenAmountIn.Amount.String())
	t.Logf("Gas: %+v", out.Gas)
	t.Logf("SwapInfo: %+v", out.SwapInfo)
}
