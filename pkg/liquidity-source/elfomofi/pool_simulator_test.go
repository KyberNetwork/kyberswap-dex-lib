package elfomofi

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte("{\"address\":\"elfomofi_0x4200000000000000000000000000000000000006_0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\",\"exchange\":\"elfomofi\",\"type\":\"elfomofi\",\"timestamp\":1768382669,\"reserves\":[\"6950940416823151180\",\"133555180156\"],\"tokens\":[{\"address\":\"0x4200000000000000000000000000000000000006\",\"symbol\":\"WETH\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"l\\\":[[[0,0],[1e-07,3310.0000000000005],[9e-07,3330],[9e-06,3328.8888888888887],[9e-05,3328.8999999999996],[0.0009,3328.9133333333334],[0.009000000000000001,3328.9141111111107],[0.09000000000000001,3328.9139],[0.9,3328.91391],[9,391.44381699999997],[90,1407.8141321666665]],[[0,0],[1e-06,0.000300043335],[9e-06,0.00030004333700000006],[9e-05,0.0003000433371222222],[0.0009,0.0003000433371122222],[0.009000000000000001,0.0003000433371111111],[0.09000000000000001,0.00030004333711128886],[0.9,0.00030004333711128225],[9,0.00030004333711128247],[90,0.0003000433371112823],[900,0.0003000433371112824],[9000,0.00030000999562856614],[90000,4.389785687838637e-05]]]}\",\"staticExtra\":\"{\\\"factoryAddress\\\":\\\"0x0000000000000000000000000000000000000001\\\"}\"}"),
		&entityPool)
	poolSim = lo.Must(NewPoolSimulator(pool.FactoryParams{EntityPool: entityPool}))
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		1: {
			0: {
				"3005810": "901873263122461",
			},
		},
		0: {
			1: {
				"1000000000000000": "3328910",
			},
		},
	})
}

// TestPoolSimulator_GreedyMatchesBatch guards against the bug where UpdateBalance only tracked
// reserves as a liquidity cap, so repeated small quotes always priced off the cheapest bracket
// instead of depleting it. With order-book levels, N small quotes must add up to ~the same total
// as one quote for the full amount.
func TestPoolSimulator_GreedyMatchesBatch(t *testing.T) {
	t.Parallel()

	greedy := lo.Must(NewPoolSimulator(pool.FactoryParams{EntityPool: entityPool}))
	chunk, _ := new(big.Int).SetString("100000000000000000", 10) // 0.1 WETH
	weth, usdc := entityPool.Tokens[0].Address, entityPool.Tokens[1].Address

	total := new(big.Int)
	for range 10 {
		res, err := greedy.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: weth, Amount: chunk},
			TokenOut:      usdc,
		})
		if err != nil {
			t.Fatal(err)
		}
		greedy.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: weth, Amount: chunk},
			TokenAmountOut: *res.TokenAmountOut,
		})
		total.Add(total, res.TokenAmountOut.Amount)
	}

	batch := lo.Must(NewPoolSimulator(pool.FactoryParams{EntityPool: entityPool}))
	full, _ := new(big.Int).SetString("1000000000000000000", 10) // 1 WETH
	res, err := batch.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: full},
		TokenOut:      usdc,
	})
	if err != nil {
		t.Fatal(err)
	}

	diff := new(big.Int).Sub(res.TokenAmountOut.Amount, total)
	if diff.CmpAbs(big.NewInt(2)) > 0 {
		t.Fatalf("greedy total %s should be within rounding of batch %s", total, res.TokenAmountOut.Amount)
	}
}
