package syncswapv2classic

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestGetAmountOut(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		entityPool        entity.Pool
		tokenAmountIn     poolPkg.TokenAmount
		tokenOut          string
		swapFee           *big.Int
		expectedAmountOut *poolPkg.TokenAmount
		expectedErr       error
	}{
		{
			name: "test normal case",
			entityPool: entity.Pool{
				Address:  "0x1788f8dec1c2054d653f8330eedcdf3dfbeb42ac",
				Exchange: "syncswap",
				Type:     "syncswap-classic",
				Reserves: []string{
					"38819698878426432914729",
					"46113879614283",
				},
				Tokens: []*entity.PoolToken{
					{
						Address:   "0x2aa69e007c32cf6637511353b89dce0b473851a9",
						Swappable: true,
					},
					{
						Address:   "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
						Swappable: true,
					},
				},
				Extra: "{\"swapFee0To1\":200,\"swapFee1To0\":200}",
			},
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x2aa69e007c32cf6637511353b89dce0b473851a9",
				Amount: bignumber.NewBig("100000000000000000000"),
			},
			tokenOut: "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
			expectedAmountOut: &poolPkg.TokenAmount{
				Token:  "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
				Amount: bignumber.NewBig("118248315577"),
			},
			expectedErr: nil,
		},
		{
			name: "test token1 as tokenIn",
			entityPool: entity.Pool{
				Address:  "0x1788f8dec1c2054d653f8330eedcdf3dfbeb42ac",
				Exchange: "syncswap",
				Type:     "syncswap-classic",
				Reserves: []string{
					"38819698878426432914729",
					"46113879614283",
				},
				Tokens: []*entity.PoolToken{
					{
						Address:   "0x2aa69e007c32cf6637511353b89dce0b473851a9",
						Swappable: true,
					},
					{
						Address:   "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
						Swappable: true,
					},
				},
				Extra: "{\"swapFee0To1\":200,\"swapFee1To0\":200}",
			},
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
				Amount: bignumber.NewBig("10000000000000"),
			},
			tokenOut: "0x2aa69e007c32cf6637511353b89dce0b473851a9",
			expectedAmountOut: &poolPkg.TokenAmount{
				Token:  "0x2aa69e007c32cf6637511353b89dce0b473851a9",
				Amount: bignumber.NewBig("6906646455383488382692"),
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := NewPoolSimulator(tc.entityPool)
			assert.Nil(t, err)
			calcAmountOutResult, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
				return pool.CalcAmountOut(poolPkg.CalcAmountOutParams{
					TokenAmountIn: tc.tokenAmountIn,
					TokenOut:      tc.tokenOut,
					Limit:         nil,
				})
			})

			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedAmountOut, calcAmountOutResult.TokenAmountOut)
		})
	}
}

func TestGetAmountIn(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name             string
		entityPool       entity.Pool
		tokenAmountOut   poolPkg.TokenAmount
		tokenIn          string
		swapFee          *big.Int
		expectedAmountIn *poolPkg.TokenAmount
		expectedErr      error
	}{
		{
			name: "test normal case",
			entityPool: entity.Pool{
				Address:  "0x1788f8dec1c2054d653f8330eedcdf3dfbeb42ac",
				Exchange: "syncswap",
				Type:     "syncswap-classic",
				Reserves: []string{
					"38819698878426432914729",
					"46113879614283",
				},
				Tokens: []*entity.PoolToken{
					{
						Address:   "0x2aa69e007c32cf6637511353b89dce0b473851a9",
						Swappable: true,
					},
					{
						Address:   "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
						Swappable: true,
					},
				},
				Extra: "{\"swapFee0To1\":200,\"swapFee1To0\":200}",
			},
			tokenAmountOut: poolPkg.TokenAmount{
				Token:  "0x2aa69e007c32cf6637511353b89dce0b473851a9",
				Amount: bignumber.NewBig("100000000000000000000"),
			},
			tokenIn: "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
			expectedAmountIn: &poolPkg.TokenAmount{
				Token:  "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
				Amount: bignumber.NewBig("119335360391"),
			},
			expectedErr: nil,
		},
		{
			name: "test token1 as tokenIn",
			entityPool: entity.Pool{
				Address:  "0x1788f8dec1c2054d653f8330eedcdf3dfbeb42ac",
				Exchange: "syncswap",
				Type:     "syncswap-classic",
				Reserves: []string{
					"38819698878426432914729",
					"46113879614283",
				},
				Tokens: []*entity.PoolToken{
					{
						Address:   "0x2aa69e007c32cf6637511353b89dce0b473851a9",
						Swappable: true,
					},
					{
						Address:   "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
						Swappable: true,
					},
				},
				Extra: "{\"swapFee0To1\":200,\"swapFee1To0\":200}",
			},
			tokenAmountOut: poolPkg.TokenAmount{
				Token:  "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
				Amount: bignumber.NewBig("10000000000000"),
			},
			tokenIn: "0x2aa69e007c32cf6637511353b89dce0b473851a9",
			expectedAmountIn: &poolPkg.TokenAmount{
				Token:  "0x2aa69e007c32cf6637511353b89dce0b473851a9",
				Amount: bignumber.NewBig("10770787930182619874183"),
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := NewPoolSimulator(tc.entityPool)
			assert.Nil(t, err)
			calcAmountInResult, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountInResult, error) {
				return pool.CalcAmountIn(poolPkg.CalcAmountInParams{
					TokenAmountOut: tc.tokenAmountOut,
					TokenIn:        tc.tokenIn,
					Limit:          nil,
				})
			})

			assert.Equal(t, tc.expectedErr, err)
			assert.Equalf(t, tc.expectedAmountIn.Amount, calcAmountInResult.TokenAmountIn.Amount, "expected amount in %s, got %s", tc.expectedAmountIn.Amount.String(), calcAmountInResult.TokenAmountIn.Amount.String())
		})
	}
}

// Regression: Linea 0x6554..9403 keeps the full input in reserves.
// Reserves after block 32244297, fee 290/100000. Expected outputs and reserves
// were read from Tenderly VNet fork at 0x1ec024a (state after 32244297):
// getReserves, getAmountOut and two sequential swap(transfer+swap) executions.
func TestLineaSequentialSwapsKeepFullInput(t *testing.T) {
	linea := "0x1789e0043623282d5dcc7f213d703c6d8bafbb04"
	weth := "0xe5d7c2a44ffddf6b295a15c148167daaaf5cf34f"
	entityPool := entity.Pool{
		Address:  "0x65543c91d48e31a4f5c3b187e88e4904d4b79403",
		Exchange: "syncswapv2-classic",
		Type:     "syncswapv2-classic",
		Reserves: []string{
			"2979193780085638750074035",
			"3146801319602445574",
		},
		Tokens: []*entity.PoolToken{
			{Address: linea, Swappable: true},
			{Address: weth, Swappable: true},
		},
		Extra: `{"swapFee0To1":290,"swapFee1To0":290,"vaultAddress":"0x7160570bb153edd0ea1775ec2b2ac9b65f1ab61b"}`,
	}

	pool, err := NewPoolSimulator(entityPool)
	assert.Nil(t, err)

	// Swap 1: LINEA -> WETH, independent quote on untouched reserves.
	swap1In := bignumber.NewBig("9059104159612778614458057")
	out1, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
		return pool.CalcAmountOut(poolPkg.CalcAmountOutParams{
			TokenAmountIn: poolPkg.TokenAmount{Token: linea, Amount: swap1In},
			TokenOut:      weth,
		})
	})
	assert.Nil(t, err)
	assert.Equal(t, "2366339264415356624", out1.TokenAmountOut.Amount.String())
	// CalcAmountOut must be pure.
	assert.Equal(t, "2979193780085638750074035", pool.GetReserves()[0].String())
	assert.Equal(t, "3146801319602445574", pool.GetReserves()[1].String())

	// Untouched quote for swap 2 input must be the independent quote, not the sequential one.
	untouchedOut2, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
		return pool.CalcAmountOut(poolPkg.CalcAmountOutParams{
			TokenAmountIn: poolPkg.TokenAmount{Token: linea, Amount: bignumber.NewBig("1960676120623460553723918")},
			TokenOut:      weth,
		})
	})
	assert.Nil(t, err)
	assert.Equal(t, "1246805082615335503", untouchedOut2.TokenAmountOut.Amount.String())

	// Apply swap 1 with full input. A second instance from the same entity must stay untouched.
	original, err := NewPoolSimulator(entityPool)
	assert.Nil(t, err)
	pool.UpdateBalance(poolPkg.UpdateBalanceParams{
		TokenAmountIn:  poolPkg.TokenAmount{Token: linea, Amount: swap1In},
		TokenAmountOut: poolPkg.TokenAmount{Token: weth, Amount: out1.TokenAmountOut.Amount},
	})
	assert.Equal(t, "12038297939698417364532092", pool.GetReserves()[0].String())
	assert.Equal(t, "780462055187088950", pool.GetReserves()[1].String())
	assert.Equal(t, "2979193780085638750074035", original.GetReserves()[0].String())
	assert.Equal(t, "3146801319602445574", original.GetReserves()[1].String())

	// Swap 2 right after swap 1 on the same state.
	swap2In := bignumber.NewBig("1960676120623460553723918")
	out2, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
		return pool.CalcAmountOut(poolPkg.CalcAmountOutParams{
			TokenAmountIn: poolPkg.TokenAmount{Token: linea, Amount: swap2In},
			TokenOut:      weth,
		})
	})
	assert.Nil(t, err)
	assert.Equal(t, "109037677720091173", out2.TokenAmountOut.Amount.String())

	pool.UpdateBalance(poolPkg.UpdateBalanceParams{
		TokenAmountIn:  poolPkg.TokenAmount{Token: linea, Amount: swap2In},
		TokenAmountOut: poolPkg.TokenAmount{Token: weth, Amount: out2.TokenAmountOut.Amount},
	})
	assert.Equal(t, "13998974060321877918256010", pool.GetReserves()[0].String())
	assert.Equal(t, "671424377466997777", pool.GetReserves()[1].String())

	// Reverse direction on the resulting state: 1 WETH -> LINEA.
	revOut, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
		return pool.CalcAmountOut(poolPkg.CalcAmountOutParams{
			TokenAmountIn: poolPkg.TokenAmount{Token: weth, Amount: bignumber.NewBig("1000000000000000000")},
			TokenOut:      linea,
		})
	})
	assert.Nil(t, err)
	assert.Equal(t, "8365701588811836956022232", revOut.TokenAmountOut.Amount.String())
}

func TestLineaSmallInputsRounding(t *testing.T) {
	linea := "0x1789e0043623282d5dcc7f213d703c6d8bafbb04"
	weth := "0xe5d7c2a44ffddf6b295a15c148167daaaf5cf34f"
	entityPool := entity.Pool{
		Address:  "0x65543c91d48e31a4f5c3b187e88e4904d4b79403",
		Exchange: "syncswapv2-classic",
		Type:     "syncswapv2-classic",
		Reserves: []string{
			"2979193780085638750074035",
			"3146801319602445574",
		},
		Tokens: []*entity.PoolToken{
			{Address: linea, Swappable: true},
			{Address: weth, Swappable: true},
		},
		Extra: `{"swapFee0To1":290,"swapFee1To0":290,"vaultAddress":"0x7160570bb153edd0ea1775ec2b2ac9b65f1ab61b"}`,
	}

	pool, err := NewPoolSimulator(entityPool)
	assert.Nil(t, err)

	// Dust LINEA rounds to zero output and must error instead of returning zero.
	for _, amt := range []string{"1", "2", "1000"} {
		_, err := pool.CalcAmountOut(poolPkg.CalcAmountOutParams{
			TokenAmountIn: poolPkg.TokenAmount{Token: linea, Amount: bignumber.NewBig(amt)},
			TokenOut:      weth,
		})
		assert.NotNil(t, err, "amountIn %s should not quote", amt)
	}

	// 1e6 LINEA still quotes 1 wei WETH.
	tinyOut, err := pool.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: linea, Amount: bignumber.NewBig("1000000")},
		TokenOut:      weth,
	})
	assert.Nil(t, err)
	assert.Equal(t, "1", tinyOut.TokenAmountOut.Amount.String())

	// CalcAmountIn round-trip: asking for swap-2 output needs no more than swap-2 input.
	calcIn, err := pool.CalcAmountIn(poolPkg.CalcAmountInParams{
		TokenAmountOut: poolPkg.TokenAmount{Token: weth, Amount: bignumber.NewBig("1000")},
		TokenIn:        linea,
	})
	assert.Nil(t, err)
	assert.Equal(t, "949490695", calcIn.TokenAmountIn.Amount.String())
}
