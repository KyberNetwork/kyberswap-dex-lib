package warpx

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/stretchr/testify/assert"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	// 1. Setup
	// Create a pool with 1000 token0 and 1000 token1
	// Fee 0.3%
	p, err := NewPoolSimulator(entity.Pool{
		Address:  "0xPool",
		Exchange: DexType,
		Type:     DexType,
		Reserves: []string{"1000000000000000000000", "1000000000000000000000"}, // 1000 * 1e18
		Tokens: []*entity.PoolToken{
			{Address: "0xToken0", Decimals: 18, Swappable: true},
			{Address: "0xToken1", Decimals: 18, Swappable: true},
		},
		Extra: `{"fee": 30, "feePrecision": 10000, "routerAddress": "0xRouter"}`,
	})
	assert.NoError(t, err)

	// 2. Test Swap
	// Swap 10 token0 -> token1
	// Expected: (10 * 997 * 1000) / (1000 * 1000 + 10 * 997) ~= 9.87...
	
	amountIn, _ := new(big.Int).SetString("10000000000000000000", 10) // 10 * 1e18
	tokenIn := "0xToken0"
	tokenOut := "0xToken1"

	res, err := p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
		TokenOut:      tokenOut,
		Limit:         nil,
	})

	assert.NoError(t, err)
	assert.NotNil(t, res)
	
	// Verified calculation: 
	// amountInWithFee = 10 * 0.997 = 9.97
	// numerator = 9.97 * 1000 = 9970
	// denominator = 1000 + 9.97 = 1009.97
	// output = 9970 / 1009.97 = 9.8715...
	
	expected := new(big.Int)
	expected.SetString("9871580343970612988", 10) // ~9.8715

	assert.Equal(t, expected.String(), res.TokenAmountOut.Amount.String())
	assert.Equal(t, int64(defaultGas), res.Gas)
}
