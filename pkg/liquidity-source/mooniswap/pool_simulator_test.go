package mooniswap

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	// ETH-USDT pool at block 24740993
	token0 := "0x0000000000000000000000000000000000000000"
	token1 := "0xdac17f958d2ee523a2206206994597c13d831ec7"

	p := entity.Pool{
		Address:  "0xbba17b81ab4193455be10741512d0e71520f43cb",
		Exchange: "mooniswap",
		Type:     "mooniswap",
		Tokens: []*entity.PoolToken{
			{Address: token0, Swappable: true},
			{Address: token1, Swappable: true},
		},
		Reserves:    []string{"6208659185333448735", "12972544827"},
		Extra:       `{"fee":"2650302140801805","slpFee":"835653904615203690","bA0":"6208659185333448735","bA1":"12972544827","bR0":"6208659185333448735","bR1":"12972544827"}`,
		StaticExtra: `{}`,
	}

	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	t.Run("ETH to USDT", func(t *testing.T) {
		result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  token0,
				Amount: big.NewInt(1000000000000000000), // 1 ETH
			},
			TokenOut: token1,
		})
		require.NoError(t, err)
		require.Equal(t, "1587806769", result.TokenAmountOut.Amount.String())
	})

	t.Run("USDT to ETH", func(t *testing.T) {
		result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  token1,
				Amount: big.NewInt(1000000000), // 1000 USDT
			},
			TokenOut: token0,
		})
		require.NoError(t, err)
		require.Equal(t, "416809127717440146", result.TokenAmountOut.Amount.String())
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0x1234567890123456789012345678901234567890",
				Amount: big.NewInt(1000),
			},
			TokenOut: token1,
		})
		require.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("zero amount", func(t *testing.T) {
		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  token0,
				Amount: big.NewInt(0),
			},
			TokenOut: token1,
		})
		require.Error(t, err)
	})
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	token0 := "0x0000000000000000000000000000000000000000"
	token1 := "0xdac17f958d2ee523a2206206994597c13d831ec7"

	p := entity.Pool{
		Address:  "0xbba17b81ab4193455be10741512d0e71520f43cb",
		Exchange: "mooniswap",
		Type:     "mooniswap",
		Tokens: []*entity.PoolToken{
			{Address: token0, Swappable: true},
			{Address: token1, Swappable: true},
		},
		Reserves:    []string{"6208659185333448735", "12972544827"},
		Extra:       `{"fee":"2650302140801805","slpFee":"835653904615203690","bA0":"6208659185333448735","bA1":"12972544827","bR0":"6208659185333448735","bR1":"12972544827"}`,
		StaticExtra: `{}`,
	}

	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	// Do a swap and update
	result1, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: token0, Amount: big.NewInt(1e18)},
		TokenOut:      token1,
	})
	require.NoError(t, err)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: token0, Amount: big.NewInt(1e18)},
		TokenAmountOut: *result1.TokenAmountOut,
	})

	// Second swap should give less output (price impact)
	result2, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: token0, Amount: big.NewInt(1e18)},
		TokenOut:      token1,
	})
	require.NoError(t, err)
	require.True(t, result2.TokenAmountOut.Amount.Cmp(result1.TokenAmountOut.Amount) < 0,
		"second swap should give less: first=%s second=%s",
		result1.TokenAmountOut.Amount, result2.TokenAmountOut.Amount)
}

func TestPoolSimulator_CloneState(t *testing.T) {
	token0 := "0x0000000000000000000000000000000000000000"
	token1 := "0xdac17f958d2ee523a2206206994597c13d831ec7"

	p := entity.Pool{
		Address:  "0xbba17b81ab4193455be10741512d0e71520f43cb",
		Exchange: "mooniswap",
		Type:     "mooniswap",
		Tokens: []*entity.PoolToken{
			{Address: token0, Swappable: true},
			{Address: token1, Swappable: true},
		},
		Reserves:    []string{"6208659185333448735", "12972544827"},
		Extra:       `{"fee":"2650302140801805","slpFee":"835653904615203690","bA0":"6208659185333448735","bA1":"12972544827","bR0":"6208659185333448735","bR1":"12972544827"}`,
		StaticExtra: `{}`,
	}

	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	cloned := sim.CloneState().(*PoolSimulator)

	// Mutate original
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: token0, Amount: big.NewInt(1e18)},
		TokenAmountOut: pool.TokenAmount{Token: token1, Amount: big.NewInt(1000000)},
	})

	// Clone should be unaffected
	result, err := cloned.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: token0, Amount: big.NewInt(1e18)},
		TokenOut:      token1,
	})
	require.NoError(t, err)
	require.Equal(t, "1587806769", result.TokenAmountOut.Amount.String())
}
