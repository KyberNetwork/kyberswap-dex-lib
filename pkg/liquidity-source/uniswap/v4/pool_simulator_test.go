package uniswapv4

import (
	_ "embed"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	//go:embed sample_pool.json
	poolData string
)

func TestPoolSimulator(t *testing.T) {
	t.Parallel()
	var (
		chainID = 1
		poolEnt entity.Pool
	)
	assert.NoError(t, json.Unmarshal([]byte(poolData), &poolEnt))

	pSim, err := NewPoolSimulator(poolEnt, valueobject.ChainID(chainID))
	assert.NoError(t, err)

	tokenAmountIn := pool.TokenAmount{
		Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		Amount: bignumber.NewBig10("15497045801"),
	}
	got, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmountIn,
		TokenOut:      "0x3b50805453023a91a8bf641e279401a0b23fa6f9",
	})
	assert.NoError(t, err)
	assert.Equal(t, bignumber.NewBig10("1070524829112273927801315"), got.TokenAmountOut.Amount)

	pSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmountIn,
		TokenAmountOut: *got.TokenAmountOut,
		Fee:            *got.Fee,
		SwapInfo:       got.SwapInfo,
	})

	got, err = pSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			Amount: bignumber.NewBig10("15500255685"),
		},
		TokenOut: "0x3b50805453023a91a8bf641e279401a0b23fa6f9",
	})
	assert.NoError(t, err)
	assert.Equal(t, bignumber.NewBig10("871374850342807317560423"), got.TokenAmountOut.Amount)
}
