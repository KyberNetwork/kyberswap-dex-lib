package hashflowv3

import (
	"math"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

var (
	tokenOMG  = &entity.PoolToken{Address: "0xd26114cd6ee289accf82350c8d8487fedb8a0c07", Decimals: 18, Swappable: true}
	tokenUSDT = &entity.PoolToken{Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", Decimals: 6, Swappable: true}
)

var entityPool = entity.Pool{
	Address:     "hashflow_v3_mm22_0xd26114cd6ee289accf82350c8d8487fedb8a0c07_0xdac17f958d2ee523a2206206994597c13d831ec7",
	Exchange:    "hashflow-v3",
	Type:        "hashflow-v3",
	Reserves:    []string{"64160215600609997156352", "152481964"},
	Tokens:      []*entity.PoolToken{tokenOMG, tokenUSDT},
	Extra:       "{\"zeroToOnePriceLevels\":[{\"q\":\"21.491858434308554\",\"p\":\"0.6924563136573486\"},{\"q\":\"2127.693984996547\",\"p\":\"0.6924563136573486\"},{\"q\":\"6450.785753788268\",\"p\":\"0.695858410957807\"},{\"q\":\"7095.864329167098\",\"p\":\"0.6955119978476955\"},{\"q\":\"7805.450762083805\",\"p\":\"0.6951337575443223\"},{\"q\":\"8588.352341200025\",\"p\":\"0.6945303753566658\"},{\"q\":\"9458.145774233493\",\"p\":\"0.6932765640141211\"},{\"q\":\"10403.960351656831\",\"p\":\"0.6927876203981647\"},{\"q\":\"11466.95813207097\",\"p\":\"0.6908910457830065\"},{\"q\":\"741.5123129786516\",\"p\":\"0.6865341216331126\"}],\"oneToZeroPriceLevels\":[{\"q\":\"1.52481964177280676980070027723634\",\"p\":\"1.414875966391418599745334487109784\"},{\"q\":\"150.957144535507867877909404465650\",\"p\":\"1.414875966391418599745334487109784\"}]}",
	StaticExtra: "{\"marketMaker\":\"mm22\"}",
}

func TestPoolSimulator_NewPool(t *testing.T) {
	poolSimulator, err := NewPoolSimulator(entityPool)
	assert.NoError(t, err)
	assert.Equal(t, tokenOMG.Address, poolSimulator.Token0.Address)
	assert.Equal(t, tokenUSDT.Address, poolSimulator.Token1.Address)
	assert.Equal(t, "mm22", poolSimulator.MarketMaker)
	assert.NotNil(t, poolSimulator.ZeroToOnePriceLevels)
	assert.NotNil(t, poolSimulator.OneToZeroPriceLevels)
	assert.Equal(t, []string{tokenUSDT.Address}, poolSimulator.CanSwapTo(tokenOMG.Address))
	assert.Equal(t, []string{tokenUSDT.Address}, poolSimulator.CanSwapFrom(tokenOMG.Address))
	assert.Equal(t, []string{tokenOMG.Address}, poolSimulator.CanSwapTo(tokenUSDT.Address))
	assert.Equal(t, []string{tokenOMG.Address}, poolSimulator.CanSwapFrom(tokenUSDT.Address))
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
			name:        "it should return error when swap lower than min level", // Lowest level ~1.5 USDT
			amountIn:    floatToWei(t, 1.0, tokenUSDT.Decimals),
			expectedErr: ErrAmountInIsLessThanLowestPriceLevel,
		},
		{
			name:        "it should return error when swap higher than total level", // Total level ~151.5 USDT
			amountIn:    floatToWei(t, 200.0, tokenUSDT.Decimals),
			expectedErr: ErrAmountInIsGreaterThanHighestPriceLevel,
		},
		{
			name:              "it should return correct amountOut when swap in levels",
			amountIn:          floatToWei(t, 3.0, tokenUSDT.Decimals),
			expectedAmountOut: bigIntFromString("4244627899174255799"),
		},
		{
			name:              "it should return correct amountOut when swap in all levels",
			amountIn:          floatToWei(t, 152.0, tokenUSDT.Decimals),
			expectedAmountOut: bigIntFromString("215061146891495627168"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Swap one to zero
			params := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  tokenUSDT.Address,
					Amount: tc.amountIn,
				},
				TokenOut: tokenOMG.Address,
			}

			result, err := poolSimulator.CalcAmountOut(params)
			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Equal(t, 0, result.TokenAmountOut.Amount.Cmp(tc.expectedAmountOut))
			}
		})
	}
}

func TestPoolSimulator_GetAmountIn(t *testing.T) {
	poolSimulator, err := NewPoolSimulator(entityPool)
	assert.NoError(t, err)

	tests := []struct {
		name             string
		amountOut        *big.Int
		expectedAmountIn *big.Int
		expectedErr      error
	}{
		{
			name:        "it should return error when swap lower than min level", // Lowest level ~2.1 OMG
			amountOut:   floatToWei(t, 2.0, tokenOMG.Decimals),
			expectedErr: ErrAmountOutIsLessThanLowestPriceLevel,
		},
		{
			name:        "it should return error when swap higher than total level", // Total level ~214.8 OMG
			amountOut:   floatToWei(t, 220.0, tokenOMG.Decimals),
			expectedErr: ErrAmountOutIsGreaterThanHighestPriceLevel,
		},
		{
			name:             "it should return correct amountIn when swap in levels",
			amountOut:        bigIntFromString("4244627899174255799"),
			expectedAmountIn: bigIntFromString("2999999"),
		},
		{
			name:             "it should return correct amountIn when swap in all levels",
			amountOut:        bigIntFromString("215061146891495627168"),
			expectedAmountIn: bigIntFromString("152000000"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Swap one to zero
			params := pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{
					Token:  tokenOMG.Address,
					Amount: tc.amountOut,
				},
				TokenIn: tokenUSDT.Address,
			}

			result, err := poolSimulator.CalcAmountIn(params)
			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Equal(t, tc.expectedAmountIn, result.TokenAmountIn.Amount)
			}
		})
	}
}

func bigIntFromString(s string) *big.Int {
	value, _ := new(big.Int).SetString(s, 10)
	return value
}

func floatToWei(t *testing.T, amount float64, decimals uint8) *big.Int {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		t.Fatalf("invalid number: %f", amount)
	}

	d := decimal.NewFromFloat(amount)
	expo := decimal.New(1, int32(decimals))

	return d.Mul(expo).BigInt()
}
