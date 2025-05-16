package syncswapclassic

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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
				Amount: utils.NewBig("100000000000000000000"),
			},
			tokenOut: "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
			expectedAmountOut: &poolPkg.TokenAmount{
				Token:  "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
				Amount: utils.NewBig("118248315577"),
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
				Amount: utils.NewBig("10000000000000"),
			},
			tokenOut: "0x2aa69e007c32cf6637511353b89dce0b473851a9",
			expectedAmountOut: &poolPkg.TokenAmount{
				Token:  "0x2aa69e007c32cf6637511353b89dce0b473851a9",
				Amount: utils.NewBig("6906646455383488382692"),
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
				Amount: utils.NewBig("100000000000000000000"),
			},
			tokenIn: "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
			expectedAmountIn: &poolPkg.TokenAmount{
				Token:  "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
				Amount: utils.NewBig("119335360391"),
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
				Amount: utils.NewBig("10000000000000"),
			},
			tokenIn: "0x2aa69e007c32cf6637511353b89dce0b473851a9",
			expectedAmountIn: &poolPkg.TokenAmount{
				Token:  "0x2aa69e007c32cf6637511353b89dce0b473851a9",
				Amount: utils.NewBig("10770787930182619874183"),
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

func TestGetAmountOutWithUpdateBalance(t *testing.T) {
	t.Parallel()
	entityPool := entity.Pool{
		Address:  "0x624202a3913fc479bd29f0e5165164575b74a8e6",
		Exchange: "syncswap",
		Type:     "syncswap-classic",
		Reserves: []string{"11064950780", "20363616"},
		Tokens: []*entity.PoolToken{
			{Address: "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4", Swappable: true},
			{Address: "0x3c1bca5a656e69edcd0d4e36bebb3fcdaca60cf1", Swappable: true},
		},
		Extra: "{\"swapFee0To1\":300,\"swapFee1To0\":300}",
	}

	pool, err := NewPoolSimulator(entityPool)
	assert.NoError(t, err)
	res, err := pool.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{
			Token:  "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4",
			Amount: big.NewInt(52436037),
		},
		TokenOut: "0x3c1bca5a656e69edcd0d4e36bebb3fcdaca60cf1",
	})
	assert.NoError(t, err)
	assert.Equal(t, res.TokenAmountOut.Amount.String(), "95759")

	pool.UpdateBalance(poolPkg.UpdateBalanceParams{
		TokenAmountIn: poolPkg.TokenAmount{
			Token:  "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4",
			Amount: big.NewInt(52436037),
		},
		TokenAmountOut: poolPkg.TokenAmount{
			Token:  "0x3c1bca5a656e69edcd0d4e36bebb3fcdaca60cf1",
			Amount: res.TokenAmountOut.Amount,
		},
		Fee: poolPkg.TokenAmount{
			Token:  res.Fee.Token,
			Amount: res.Fee.Amount,
		},
		SwapInfo: res.SwapInfo,
	})

	res, err = pool.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{
			Token:  "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4",
			Amount: big.NewInt(78292),
		},
		TokenOut: "0x3c1bca5a656e69edcd0d4e36bebb3fcdaca60cf1",
	})
	assert.NoError(t, err)
	assert.Equal(t, res.TokenAmountOut.Amount.String(), "142")
}
