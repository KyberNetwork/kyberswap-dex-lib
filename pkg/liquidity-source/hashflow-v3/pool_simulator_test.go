package hashflowv3

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/stretchr/testify/assert"
)

var entityPool = entity.Pool{
	Address:  "hashflow_v3_mm22_0xd26114cd6ee289accf82350c8d8487fedb8a0c07_0xdac17f958d2ee523a2206206994597c13d831ec7",
	Exchange: "hashflow-v3",
	Type:     "hashflow-v3",
	Reserves: []string{"64160215600609997156352", "152481964"},
	Tokens: []*entity.PoolToken{
		{Address: "0xd26114cd6ee289accf82350c8d8487fedb8a0c07", Decimals: 18, Swappable: true},
		{Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", Decimals: 6, Swappable: true},
	},
	Extra:       "{\"zeroToOnePriceLevels\":[{\"q\":\"21.491858434308554\",\"p\":\"0.6924563136573486\"},{\"q\":\"2127.693984996547\",\"p\":\"0.6924563136573486\"},{\"q\":\"6450.785753788268\",\"p\":\"0.695858410957807\"},{\"q\":\"7095.864329167098\",\"p\":\"0.6955119978476955\"},{\"q\":\"7805.450762083805\",\"p\":\"0.6951337575443223\"},{\"q\":\"8588.352341200025\",\"p\":\"0.6945303753566658\"},{\"q\":\"9458.145774233493\",\"p\":\"0.6932765640141211\"},{\"q\":\"10403.960351656831\",\"p\":\"0.6927876203981647\"},{\"q\":\"11466.95813207097\",\"p\":\"0.6908910457830065\"},{\"q\":\"741.5123129786516\",\"p\":\"0.6865341216331126\"}],\"oneToZeroPriceLevels\":[{\"q\":\"1.52481964177280676980070027723634\",\"p\":\"1.414875966391418599745334487109784\"},{\"q\":\"150.957144535507867877909404465650\",\"p\":\"1.414875966391418599745334487109784\"}]}",
	StaticExtra: "{\"marketMaker\":\"mm22\"}",
}

func TestPoolSimulator_NewPool(t *testing.T) {
	poolSimulator, err := NewPoolSimulator(entityPool)
	assert.NoError(t, err)
	assert.Equal(t, "0xd26114cd6ee289accf82350c8d8487fedb8a0c07", poolSimulator.Token0.Address)
	assert.Equal(t, "0xdac17f958d2ee523a2206206994597c13d831ec7", poolSimulator.Token1.Address)
	assert.Equal(t, "mm22", poolSimulator.MarketMaker)
	assert.NotNil(t, poolSimulator.ZeroToOnePriceLevels)
	assert.NotNil(t, poolSimulator.OneToZeroPriceLevels)
	assert.Equal(t, []string{"0xdac17f958d2ee523a2206206994597c13d831ec7"}, poolSimulator.CanSwapTo("0xd26114cd6ee289accf82350c8d8487fedb8a0c07"))
	assert.Equal(t, []string{"0xdac17f958d2ee523a2206206994597c13d831ec7"}, poolSimulator.CanSwapFrom("0xd26114cd6ee289accf82350c8d8487fedb8a0c07"))
	assert.Equal(t, []string{"0xd26114cd6ee289accf82350c8d8487fedb8a0c07"}, poolSimulator.CanSwapTo("0xdac17f958d2ee523a2206206994597c13d831ec7"))
	assert.Equal(t, []string{"0xd26114cd6ee289accf82350c8d8487fedb8a0c07"}, poolSimulator.CanSwapFrom("0xdac17f958d2ee523a2206206994597c13d831ec7"))
}

func TestPoolSimulator_GetAmountOut(t *testing.T) {
	poolSimulator, err := NewPoolSimulator(entityPool)
	assert.NoError(t, err)

	tests := []struct {
		name              string
		amountIn          *big.Int
		expectedAmountOut *big.Int
		expectedErr       error
	}{
		{
			name:        "it should return error when swap lower than min level", // Lowest level ~1.5 USDC
			amountIn:    big.NewInt(1_000_000),
			expectedErr: ErrAmountInIsLessThanLowestPriceLevel,
		},
		{
			name:        "it should return error when swap higher than total level", // Total level ~151.5 USDC
			amountIn:    big.NewInt(200_000_000),
			expectedErr: ErrAmountInIsGreaterThanHighestPriceLevel,
		},
		{
			name:              "it should return correct amountOut when swap in levels",
			amountIn:          big.NewInt(3_000_000),
			expectedAmountOut: bigIntFromString("4244627899174255799"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Swap one to zero
			params := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
					Amount: tc.amountIn,
				},
				TokenOut: "0xd26114cd6ee289accf82350c8d8487fedb8a0c07",
			}

			result, err := poolSimulator.CalcAmountOut(params)
			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Equal(t, 0, result.TokenAmountOut.Amount.Cmp(tc.expectedAmountOut))
			}
		})
	}
}

func bigIntFromString(s string) *big.Int {
	value, _ := new(big.Int).SetString(s, 10)
	return value
}
