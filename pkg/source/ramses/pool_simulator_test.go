package ramses

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	velodromev1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velodrome-v1"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestNewPoolSimulator(t *testing.T) {
	t.Parallel()
	t.Run("it should init pool simulator correctly", func(t *testing.T) {
		var pool *entity.Pool
		jsonPool := `{"address":"0xf9c642d206e7974d7d01758568d3e30019c7f022","timestamp":1756356175,"reserves":["228695083172926728","155298192376701482337"],"extra":"{\"isPaused\":false,\"fee\":100}","staticExtra":"{\"stable\":false}"}`
		err := json.Unmarshal([]byte(jsonPool), &pool)
		assert.Nil(t, err)

		poolSimulator, err := velodromev1.NewPoolSimulator(*pool)

		assert.Nil(t, err)
		assert.False(t, poolSimulator.IsPaused)
		assert.False(t, poolSimulator.Stable)
		assert.EqualValues(t, uint64(100), poolSimulator.Fee.Uint64())
		assert.EqualValues(t, uint64(228695083172926728), poolSimulator.Info.Reserves[0].Uint64())

	})
}

func TestPoolSimulator_getAmountOut(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		poolSimulator     velodromev1.PoolSimulator
		tokenAmountIn     poolpkg.TokenAmount
		tokenOut          string
		expectedAmountOut *big.Int
		expectedFee       *big.Int
	}{
		{
			name: "it should return correct amount",
			poolSimulator: velodromev1.PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0xf9c642d206e7974d7d01758568d3e30019c7f022",
						Tokens:   []string{"0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "0x99c409e5f62e4bd2ac142f17cafb6810b8f0baae"},
						Reserves: []*big.Int{bignumber.NewBig10("228695083172926728"), bignumber.NewBig10("155298192376701482337")},
					},
				},
				IsPaused:     false,
				Stable:       false,
				Decimals0:    number.NewUint256("1000000000000000000"),
				Decimals1:    number.NewUint256("1000000"),
				Fee:          uint256.NewInt(100),
				FeePrecision: uint256.NewInt(10000),
			},
			tokenAmountIn:     poolpkg.TokenAmount{Token: "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", Amount: bignumber.NewBig10("33762029")},
			tokenOut:          "0x99c409e5f62e4bd2ac142f17cafb6810b8f0baae",
			expectedAmountOut: bignumber.NewBig10("22697253592"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
				return tc.poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: tc.tokenAmountIn,
					TokenOut:      tc.tokenOut,
				})
			})

			if tc.expectedAmountOut != nil {
				assert.Nil(t, err)
				assert.Equalf(t, tc.expectedAmountOut, result.TokenAmountOut.Amount, "expected amount out: %s, got: %s", tc.expectedAmountOut.String(), result.TokenAmountOut.Amount.String())
			}
		})
	}
}
