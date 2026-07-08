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

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
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

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
	var (
		chainID = 42161
		poolEnt entity.Pool
	)
	assert.NoError(t, json.Unmarshal([]byte(`{"address":"0x9969d64e96abcfec89bb3816ddd6fbad39e5f510e35c49ca04cbb9ba9b6cab65","swapFee":990000,"exchange":"uniswap-v4","type":"uniswap-v4","timestamp":1764066577,"reserves":["30346","39055449087309741"],"tokens":[{"address":"0xaf88d065e77c8cc2239327c5edb3a432268e5831","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xb688ba096b7bb75d7841e47163cd12d18b36a5bf","symbol":"mPendle","decimals":18,"swappable":true}],"extra":"{\"liquidity\":34426962013,\"sqrtPriceX96\":89879887344939541440282565011026845,\"tickSpacing\":19800,\"tick\":278847,\"ticks\":[{\"index\":-871200,\"liquidityGross\":34426962013,\"liquidityNet\":34426962013},{\"index\":871200,\"liquidityGross\":34426962013,\"liquidityNet\":-34426962013}]}","staticExtra":"{\"0x0\":[false,false],\"fee\":990000,\"tS\":19800,\"hooks\":\"0x0000000000000000000000000000000000000000\",\"uR\":\"0xa51afafe0263b40edaef0df8781ea9aa03e381a3\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":403938805}`), &poolEnt))

	pSim, err := NewPoolSimulator(poolEnt, valueobject.ChainID(chainID))
	assert.NoError(t, err)

	tokenAmountOut := pool.TokenAmount{
		Token:  "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
		Amount: bignumber.NewBig10("15497045801"),
	}
	got, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmountOut,
		TokenOut:      "0xb688ba096b7bb75d7841e47163cd12d18b36a5bf",
	})
	assert.NoError(t, err)
	assert.Equal(t, bignumber.NewBig10("39047802574032313"), got.TokenAmountOut.Amount)

	pSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmountOut,
		TokenAmountOut: *got.TokenAmountOut,
		Fee:            *got.Fee,
		SwapInfo:       got.SwapInfo,
	})

	got, err = pSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
			Amount: bignumber.NewBig10("15500255685"),
		},
		TokenOut: "0xb688ba096b7bb75d7841e47163cd12d18b36a5bf",
	})
	assert.NoError(t, err)
	assert.Equal(t, bignumber.NewBig10("3823278233713"), got.TokenAmountOut.Amount)
}
