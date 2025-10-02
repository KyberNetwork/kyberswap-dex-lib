package mkr_sky

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var testPool = entity.Pool{
	Address:  "0xbdcfca946b6cdd965f99a839e4435bcdc1bc470b",
	Exchange: "mkr-sky",
	Type:     "mkr-sky",
	SwapFee:  0.01,
	Reserves: []string{
		defaultReserves,
		defaultReserves,
	},
	Tokens: []*entity.PoolToken{
		{
			Address:   "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2",
			Swappable: true,
		},
		{
			Address:   "0x56072c95faa701256059aa122697b133aded9279",
			Swappable: false,
		},
	},
	StaticExtra: "{\"rate\":24000}",
}

func TestGetAmountOut(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		entityPool        entity.Pool
		tokenAmountIn     poolPkg.TokenAmount
		tokenOut          string
		expectedAmountOut *poolPkg.TokenAmount
		expectedErr       error
	}{
		{
			name:       "swap MKR to SKY",
			entityPool: testPool,
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2",
				Amount: bignumber.NewBig("1000000000000000000"),
			},
			tokenOut: "0x56072c95faa701256059aa122697b133aded9279",
			expectedAmountOut: &poolPkg.TokenAmount{
				Token:  "0x56072c95faa701256059aa122697b133aded9279",
				Amount: bignumber.NewBig("23760000000000000000000"),
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := NewPoolSimulator(tc.entityPool)
			assert.Nil(t, err)

			assert.Contains(t, pool.CanSwapTo(tc.tokenOut), tc.tokenAmountIn.Token)

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
