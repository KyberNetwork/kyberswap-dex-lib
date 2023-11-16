package balancerweighted

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwap_2token(t *testing.T) {
	// get test data from https://balancer.tools/priceImpact (SWAP tab)
	var poolInfo = entity.Pool{
		Address:  "adr",
		SwapFee:  0.0025,
		Reserves: []string{"5000000", "7000"},
		Tokens: entity.PoolTokens{
			&entity.PoolToken{Address: "BAL", Weight: 80},
			&entity.PoolToken{Address: "WETH", Weight: 20},
		},
		StaticExtra: "{\"vaultAddress\":\"v1\",\"poolId\":\"p1\",\"tokenDecimals\":[1,19]}",
	}
	var p, err = NewPoolSimulator(poolInfo)
	require.Nil(t, err)

	result, err := p.CalcAmountOut(
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: "BAL", Amount: big.NewInt(1000)},
			TokenOut:      "WETH",
			Limit:         nil,
		})
	require.Nil(t, err)
	assert.Equal(t, big.NewInt(5), result.TokenAmountOut.Amount)
	assert.Equal(t, big.NewInt(3), result.Fee.Amount)
}

func TestSwap_3token(t *testing.T) {
	// get test data from https://balancer.tools/priceImpact (SWAP tab)
	var poolInfo = entity.Pool{
		Address:  "adr",
		SwapFee:  0.0025,
		Reserves: []string{"5000000", "7000", "300000"},
		Tokens: entity.PoolTokens{
			&entity.PoolToken{Address: "BAL", Weight: 40},
			&entity.PoolToken{Address: "WETH", Weight: 10},
			&entity.PoolToken{Address: "DAI", Weight: 50},
		},
		StaticExtra: "{\"vaultAddress\":\"v1\",\"poolId\":\"p1\",\"tokenDecimals\":[1,19,1]}",
	}
	var p, err = NewPoolSimulator(poolInfo)
	require.Nil(t, err)
	assert.Equal(t, []string{"WETH", "DAI"}, p.CanSwapTo("BAL"))
	assert.Equal(t, 0, len(p.CanSwapTo("BALxx")))

	// weight(BAL)/weight(WETH) is still the same as above, so amount out should be the same
	result, err := p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "BAL", Amount: big.NewInt(1000)},
		TokenOut:      "WETH",
		Limit:         nil,
	})
	require.Nil(t, err)
	assert.Equal(t, big.NewInt(5), result.TokenAmountOut.Amount)
	assert.Equal(t, big.NewInt(3), result.Fee.Amount)

	// BAL -> DAI
	result, err = p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "BAL", Amount: big.NewInt(1000)},
		TokenOut:      "DAI",
		Limit:         nil,
	})
	require.Nil(t, err)
	assert.Equal(t, big.NewInt(47), result.TokenAmountOut.Amount)
	assert.Equal(t, big.NewInt(3), result.Fee.Amount)
}
