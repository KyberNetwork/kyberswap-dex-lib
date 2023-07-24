package syncswapstable

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"

	"github.com/stretchr/testify/assert"
)

func TestGetAmountOut(t *testing.T) {
	testCases := []struct {
		name              string
		entityPool        entity.Pool
		tokenAmountIn     poolPkg.TokenAmount
		tokenOut          string
		expectedAmountOut *poolPkg.TokenAmount
		expectedErr       error
	}{
		{
			name: "test token0 as tokenIn",
			entityPool: entity.Pool{
				Address:  "0x92eae0b3a75f3ef6c50369ce8ca96b285d2139b8",
				Exchange: "syncswap",
				Type:     "syncswap-stable",
				Reserves: []string{
					"276926762767",
					"284081796016",
				},
				Tokens: []*entity.PoolToken{
					{
						Address:   "0x3355df6d4c9c3035724fd0e3914de96a5a83aaf4",
						Swappable: true,
					},
					{
						Address:   "0xfc7e56298657b002b3e656400e746b7212912757",
						Swappable: true,
					},
				},
				Extra: "{\"swapFee0To1\":40,\"swapFee1To0\":40,\"token0PrecisionMultiplier\":1000000000000,\"token1PrecisionMultiplier\":1000000000000}",
			},
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x3355df6d4c9c3035724fd0e3914de96a5a83aaf4",
				Amount: bignumber.NewBig("100000000000"),
			},
			tokenOut: "0xfc7e56298657b002b3e656400e746b7212912757",
			expectedAmountOut: &poolPkg.TokenAmount{
				Token:  "0xfc7e56298657b002b3e656400e746b7212912757",
				Amount: bignumber.NewBig("99922559468"),
			},
			expectedErr: nil,
		},
		{
			name: "test token1 as tokenIn",
			entityPool: entity.Pool{
				Address:  "0x92eae0b3a75f3ef6c50369ce8ca96b285d2139b8",
				Exchange: "syncswap",
				Type:     "syncswap-stable",
				Reserves: []string{
					"276838614939",
					"284170002373",
				},
				Tokens: []*entity.PoolToken{
					{
						Address:   "0x3355df6d4c9c3035724fd0e3914de96a5a83aaf4",
						Swappable: true,
					},
					{
						Address:   "0xfc7e56298657b002b3e656400e746b7212912757",
						Swappable: true,
					},
				},
				Extra: "{\"swapFee0To1\":40,\"swapFee1To0\":40,\"token0PrecisionMultiplier\":1000000000000,\"token1PrecisionMultiplier\":1000000000000}",
			},
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0xfc7e56298657b002b3e656400e746b7212912757",
				Amount: bignumber.NewBig("100000000000"),
			},
			tokenOut: "0x3355df6d4c9c3035724fd0e3914de96a5a83aaf4",
			expectedAmountOut: &poolPkg.TokenAmount{
				Token:  "0x3355df6d4c9c3035724fd0e3914de96a5a83aaf4",
				Amount: bignumber.NewBig("99915796719"),
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := NewPoolSimulator(tc.entityPool)
			assert.Nil(t, err)
			calcAmountOutResult, err := pool.CalcAmountOut(tc.tokenAmountIn, tc.tokenOut)

			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedAmountOut, calcAmountOutResult.TokenAmountOut)
		})
	}
}
