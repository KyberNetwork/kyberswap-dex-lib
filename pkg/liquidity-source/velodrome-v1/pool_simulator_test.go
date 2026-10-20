package velodromev1

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestNewPoolSimulator(t *testing.T) {
	t.Parallel()
	t.Run("it should init pool simulator correctly", func(t *testing.T) {
		entityPool := entity.Pool{
			Address:   "0xaa7a44d696ca5033e6f7a2d3fbcf8d0913f018b7",
			Exchange:  "velodrome",
			Type:      "velodrome",
			Timestamp: 1699771973,
			Reserves:  []string{"3474496496", "1151246785735786"},
			Tokens: []*entity.PoolToken{
				{Address: "0x3e7ef8f50246f725885102e8238cbba33f276747", Swappable: true},
				{Address: "0xda10009cbd5d07dd0cecc66161fc93d7c9000da1", Swappable: true},
			},
			Extra:       "{\"isPaused\":true,\"fee\":5}",
			StaticExtra: "{\"feePrecision\":10000,\"decimal0\":\"0xde0b6b3a7640000\",\"decimal1\":\"0xde0b6b3a7640000\",\"stable\":false}",
		}

		poolSimulator, err := NewPoolSimulator(entityPool)

		assert.Nil(t, err)
		assert.True(t, poolSimulator.isPaused)
		assert.False(t, poolSimulator.stable)
		assert.EqualValues(t, uint64(5), poolSimulator.fee.Uint64())
		assert.EqualValues(t, defaultGas, poolSimulator.gas)

		// yaml `gas` (StaticExtra g) overrides the default, e.g. veplus-v2's measured hop cost
		entityPool.StaticExtra = `{"feePrecision":10000,"g":204078}`
		poolSimulator, err = NewPoolSimulator(entityPool)
		assert.Nil(t, err)
		assert.EqualValues(t, 204078, poolSimulator.gas)
	})
}

// TestPoolSimulator_getAmountOut
// [volatile][1to0]: https://optimistic.etherscan.io/tx/0x127f10c9a2562015e4881f45a2837a43100d11e33f4cc3198a15e57bf1f18869
// [volatile][0to1]: https://optimistic.etherscan.io/tx/0x1b9e4744d94390c48687c18ab6ddf951c8710580b6a9cb7c44545bae5a370705
// [stable][1to0]: https://optimistic.etherscan.io/tx/0x6d1ebd5f31077408f5b33718156403e0aef83d4a8614e7162a3d38bf9653cd7c
// [stable][0to1]: https://optimistic.etherscan.io/tx/0x5c92ec5d38d51e33952777e4468d8499cc1b9f0ce40e698caedde3f23b3d37e7
func TestPoolSimulator_getAmountOut(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		poolSimulator     PoolSimulator
		tokenAmountIn     poolpkg.TokenAmount
		tokenOut          string
		expectedAmountOut *big.Int
		expectedFee       *big.Int
	}{
		{
			name: "[volatile][1to0] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0x79c912fef520be002c2b6e57ec4324e260f38e50",
						Tokens: []string{"0x4200000000000000000000000000000000000006",
							"0x7f5c764cbc14f9669b88837ca1490cca17c31607"},
						Reserves: []*big.Int{bignumber.NewBig10("31229966656506421921"),
							bignumber.NewBig10("63506727363")},
					},
				},
				isPaused:     false,
				stable:       false,
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
				reserves: []*uint256.Int{uint256.MustFromDecimal("31229966656506421921"),
					uint256.MustFromDecimal("63506727363")},
				decimals: []*uint256.Int{uint256.MustFromDecimal("1000000000000000000"),
					uint256.MustFromDecimal("1000000")},
			},
			tokenAmountIn: poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
				Amount: bignumber.NewBig10("33762029")},
			tokenOut:          "0x4200000000000000000000000000000000000006",
			expectedAmountOut: bignumber.NewBig10("16585646993362100"),
			expectedFee:       bignumber.NewBig10("16881"),
		},
		{
			name: "[volatile][0to1] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0x79c912fef520be002c2b6e57ec4324e260f38e50",
						Tokens: []string{"0x4200000000000000000000000000000000000006",
							"0x7f5c764cbc14f9669b88837ca1490cca17c31607"},
						Reserves: []*big.Int{bignumber.NewBig10("31220354779450883153"),
							bignumber.NewBig10("63526279313")},
					},
				},
				isPaused:     false,
				stable:       false,
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
				reserves: []*uint256.Int{uint256.MustFromDecimal("31220354779450883153"),
					uint256.MustFromDecimal("63526279313")},
				decimals: []*uint256.Int{uint256.MustFromDecimal("1000000000000000000"),
					uint256.MustFromDecimal("1000000")},
			},
			tokenAmountIn: poolpkg.TokenAmount{Token: "0x4200000000000000000000000000000000000006",
				Amount: bignumber.NewBig10("3655170221820867")},
			tokenOut:          "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
			expectedAmountOut: bignumber.NewBig10("7432846"),
			expectedFee:       bignumber.NewBig10("1827585110910"),
		},
		{
			name: "[stable][1to0] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0xe08d427724d8a2673fe0be3a81b7db17be835b36",
						Tokens: []string{"0x7f5c764cbc14f9669b88837ca1490cca17c31607",
							"0x94b008aa00579c1307b0ef2c499ad98a8ce58e58"},
						Reserves: []*big.Int{bignumber.NewBig10("2052127179"), bignumber.NewBig10("1705017421")},
					},
				},
				isPaused:     false,
				stable:       true,
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
				reserves: []*uint256.Int{uint256.MustFromDecimal("2052127179"),
					uint256.MustFromDecimal("1705017421")},
				decimals: []*uint256.Int{uint256.MustFromDecimal("1000000"), uint256.MustFromDecimal("1000000")},
			},
			tokenAmountIn: poolpkg.TokenAmount{Token: "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58",
				Amount: bignumber.NewBig10("36283954")},
			tokenOut:          "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
			expectedAmountOut: bignumber.NewBig10("36307464"),
			expectedFee:       bignumber.NewBig10("18141"),
		},
		{
			name: "[stable][0to1] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0xe08d427724d8a2673fe0be3a81b7db17be835b36",
						Tokens: []string{"0x7f5c764cbc14f9669b88837ca1490cca17c31607",
							"0x94b008aa00579c1307b0ef2c499ad98a8ce58e58"},
						Reserves: []*big.Int{bignumber.NewBig10("6110873648"), bignumber.NewBig10("6651345170")},
					},
				},
				isPaused:     false,
				stable:       true,
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
				reserves: []*uint256.Int{uint256.MustFromDecimal("6110873648"),
					uint256.MustFromDecimal("6651345170")},
				decimals: []*uint256.Int{uint256.MustFromDecimal("1000000"), uint256.MustFromDecimal("1000000")},
			},
			tokenAmountIn: poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
				Amount: bignumber.NewBig10("302268108")},
			tokenOut:          "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58",
			expectedAmountOut: bignumber.NewBig10("302127234"),
			expectedFee:       bignumber.NewBig10("151134"),
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
				assert.Equalf(t, tc.expectedAmountOut, result.TokenAmountOut.Amount, "expected amount out: %s, got: %s",
					tc.expectedAmountOut.String(), result.TokenAmountOut.Amount.String())
				assert.Zero(t, tc.expectedFee.Cmp(result.Fee.Amount))
			}
		})
	}
}

func TestPoolSimulator_getAmountIn(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name             string
		poolSimulator    PoolSimulator
		tokenAmountOut   poolpkg.TokenAmount
		tokenIn          string
		expectedAmountIn *big.Int
		expectedFee      *big.Int
	}{
		{
			name: "[volatile][0to1] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0x79c912fef520be002c2b6e57ec4324e260f38e50",
						Tokens: []string{"0x4200000000000000000000000000000000000006",
							"0x7f5c764cbc14f9669b88837ca1490cca17c31607"},
						Reserves: []*big.Int{bignumber.NewBig10("31229966656506421921"),
							bignumber.NewBig10("63506727363")},
					},
				},
				isPaused:     false,
				stable:       false,
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
				reserves: []*uint256.Int{uint256.MustFromDecimal("31229966656506421921"),
					uint256.MustFromDecimal("63506727363")},
				decimals: []*uint256.Int{uint256.MustFromDecimal("1000000000000000000"),
					uint256.MustFromDecimal("1000000")},
			},
			tokenAmountOut: poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
				Amount: bignumber.NewBig10("33762029")},
			tokenIn:          "0x4200000000000000000000000000000000000006",
			expectedAmountIn: bignumber.NewBig10("16619902530526859"),
			expectedFee:      bignumber.NewBig10("0"),
		},
		{
			name: "[volatile][1to0] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0x79c912fef520be002c2b6e57ec4324e260f38e50",
						Tokens: []string{"0x4200000000000000000000000000000000000006",
							"0x7f5c764cbc14f9669b88837ca1490cca17c31607"},
						Reserves: []*big.Int{bignumber.NewBig10("31220354779450883153"),
							bignumber.NewBig10("63526279313")},
					},
				},
				isPaused:     false,
				stable:       false,
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
				reserves: []*uint256.Int{uint256.MustFromDecimal("31220354779450883153"),
					uint256.MustFromDecimal("63526279313")},
				decimals: []*uint256.Int{uint256.MustFromDecimal("1000000000000000000"),
					uint256.MustFromDecimal("1000000")},
			},
			tokenAmountOut: poolpkg.TokenAmount{Token: "0x4200000000000000000000000000000000000006",
				Amount: bignumber.NewBig10("3655170221820867")},
			tokenIn:          "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
			expectedAmountIn: bignumber.NewBig10("7442028"),
			expectedFee:      bignumber.NewBig10("0"),
		},
		{
			name: "[stable][1to0] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0xe08d427724d8a2673fe0be3a81b7db17be835b36",
						Tokens: []string{"0x7f5c764cbc14f9669b88837ca1490cca17c31607",
							"0x94b008aa00579c1307b0ef2c499ad98a8ce58e58"},
						Reserves: []*big.Int{bignumber.NewBig10("2052127179"), bignumber.NewBig10("1705017421")},
					},
				},
				isPaused:     false,
				stable:       true,
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
				reserves: []*uint256.Int{uint256.MustFromDecimal("2052127179"),
					uint256.MustFromDecimal("1705017421")},
				decimals: []*uint256.Int{uint256.MustFromDecimal("1000000"), uint256.MustFromDecimal("1000000")},
			},
			tokenAmountOut: poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
				Amount: bignumber.NewBig10("36307464")},
			tokenIn:          "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58",
			expectedAmountIn: bignumber.NewBig10("36283954"),
			expectedFee:      bignumber.NewBig10("0"),
		},
		{
			name: "[stable][0to1] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0xe08d427724d8a2673fe0be3a81b7db17be835b36",
						Tokens: []string{"0x7f5c764cbc14f9669b88837ca1490cca17c31607",
							"0x94b008aa00579c1307b0ef2c499ad98a8ce58e58"},
						Reserves: []*big.Int{bignumber.NewBig10("6110873648"), bignumber.NewBig10("6651345170")},
					},
				},
				isPaused:     false,
				stable:       true,
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
				reserves: []*uint256.Int{uint256.MustFromDecimal("6110873648"),
					uint256.MustFromDecimal("6651345170")},
				decimals: []*uint256.Int{uint256.MustFromDecimal("1000000"), uint256.MustFromDecimal("1000000")},
			},
			tokenAmountOut: poolpkg.TokenAmount{Token: "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58",
				Amount: bignumber.NewBig10("302127234")},
			tokenIn:          "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
			expectedAmountIn: bignumber.NewBig10("302268108"),
			expectedFee:      bignumber.NewBig10("0"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
				return tc.poolSimulator.CalcAmountIn(poolpkg.CalcAmountInParams{
					TokenAmountOut: tc.tokenAmountOut,
					TokenIn:        tc.tokenIn,
				})
			})

			if tc.expectedAmountIn != nil {
				assert.Nil(t, err)
				assert.Equalf(t, tc.expectedAmountIn, result.TokenAmountIn.Amount, "expected amount in: %s, got: %s",
					tc.expectedAmountIn.String(), result.TokenAmountIn.Amount.String())
				assert.Equalf(t, tc.expectedFee, result.Fee.Amount, "expected fee: %s, got: %s",
					tc.expectedFee.String(), result.Fee.Amount.String())
			}
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name             string
		poolSimulator    PoolSimulator
		params           poolpkg.UpdateBalanceParams
		expectedReserves []*uint256.Int
	}{
		{
			name: "[volatile][1to0] it should update reserve correctly",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0x79c912fef520be002c2b6e57ec4324e260f38e50",
						Tokens: []string{"0x4200000000000000000000000000000000000006",
							"0x7f5c764cbc14f9669b88837ca1490cca17c31607"},
						Reserves: []*big.Int{bignumber.NewBig10("31229966656506421921"),
							bignumber.NewBig10("63506727363")},
					},
				},
				isPaused:     false,
				stable:       false,
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
				reserves: []*uint256.Int{uint256.MustFromDecimal("31229966656506421921"),
					uint256.MustFromDecimal("63506727363")},
				decimals: []*uint256.Int{uint256.MustFromDecimal("1000000000000000000"),
					uint256.MustFromDecimal("1000000")},
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x4200000000000000000000000000000000000006",
					Amount: bignumber.NewBig10("16585646993362100")},
				TokenAmountIn: poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
					Amount: bignumber.NewBig10("33762029")},
				Fee: poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
					Amount: bignumber.NewBig10("16881")},
			},
			expectedReserves: []*uint256.Int{uint256.MustFromDecimal("31213381009513059821"),
				uint256.MustFromDecimal("63540472511")},
		},
		{
			name: "[volatile][0to1] it should update reserve correctly",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0x79c912fef520be002c2b6e57ec4324e260f38e50",
						Tokens: []string{"0x4200000000000000000000000000000000000006",
							"0x7f5c764cbc14f9669b88837ca1490cca17c31607"},
						Reserves: []*big.Int{bignumber.NewBig10("31220354779450883153"),
							bignumber.NewBig10("63526279313")},
					},
				},
				isPaused:     false,
				stable:       false,
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
				reserves: []*uint256.Int{uint256.MustFromDecimal("31220354779450883153"),
					uint256.MustFromDecimal("63526279313")},
				decimals: []*uint256.Int{uint256.MustFromDecimal("1000000000000000000"),
					uint256.MustFromDecimal("1000000")},
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
					Amount: bignumber.NewBig10("7432846")},
				TokenAmountIn: poolpkg.TokenAmount{Token: "0x4200000000000000000000000000000000000006",
					Amount: bignumber.NewBig10("3655170221820867")},
				Fee: poolpkg.TokenAmount{Token: "0x4200000000000000000000000000000000000006",
					Amount: bignumber.NewBig10("1827585110910")},
			},
			expectedReserves: []*uint256.Int{uint256.MustFromDecimal("31224008122087593110"),
				uint256.MustFromDecimal("63518846467")},
		},
		{
			name: "[stable][1to0] it should update reserve correctly",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0xe08d427724d8a2673fe0be3a81b7db17be835b36",
						Tokens: []string{"0x7f5c764cbc14f9669b88837ca1490cca17c31607",
							"0x94b008aa00579c1307b0ef2c499ad98a8ce58e58"},
						Reserves: []*big.Int{bignumber.NewBig10("2052127179"), bignumber.NewBig10("1705017421")},
					},
				},
				isPaused:     false,
				stable:       true,
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
				reserves: []*uint256.Int{uint256.MustFromDecimal("2052127179"),
					uint256.MustFromDecimal("1705017421")},
				decimals: []*uint256.Int{uint256.MustFromDecimal("1000000"), uint256.MustFromDecimal("1000000")},
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
					Amount: bignumber.NewBig10("36307464")},
				TokenAmountIn: poolpkg.TokenAmount{Token: "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58",
					Amount: bignumber.NewBig10("36283954")},
				Fee: poolpkg.TokenAmount{Token: "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58",
					Amount: bignumber.NewBig10("18141")},
			},
			expectedReserves: []*uint256.Int{uint256.MustFromDecimal("2015819715"),
				uint256.MustFromDecimal("1741283234")},
		},
		{
			name: "[stable][0to1] it should update reserve correctly",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0xe08d427724d8a2673fe0be3a81b7db17be835b36",
						Tokens: []string{"0x7f5c764cbc14f9669b88837ca1490cca17c31607",
							"0x94b008aa00579c1307b0ef2c499ad98a8ce58e58"},
						Reserves: []*big.Int{bignumber.NewBig10("6110873648"), bignumber.NewBig10("6651345170")},
					},
				},
				isPaused:     false,
				stable:       true,
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
				reserves: []*uint256.Int{uint256.MustFromDecimal("6110873648"),
					uint256.MustFromDecimal("6651345170")},
				decimals: []*uint256.Int{uint256.MustFromDecimal("1000000"), uint256.MustFromDecimal("1000000")},
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58",
					Amount: bignumber.NewBig10("302127234")},
				TokenAmountIn: poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
					Amount: bignumber.NewBig10("302268108")},
				Fee: poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
					Amount: bignumber.NewBig10("151134")},
			},
			expectedReserves: []*uint256.Int{uint256.MustFromDecimal("6412990622"),
				uint256.MustFromDecimal("6349217936")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.poolSimulator.UpdateBalance(tc.params)

			assert.Equal(t, tc.expectedReserves[0], tc.poolSimulator.reserves[0])
			assert.Equal(t, tc.expectedReserves[1], tc.poolSimulator.reserves[1])
		})
	}
}

func TestCalcAmountIn(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		entityPool entity.Pool
	}{
		{
			name: "[volatile] it should test calc amount in correctly",
			entityPool: entity.Pool{
				Address:   "0x79c912fef520be002c2b6e57ec4324e260f38e50",
				Exchange:  "velodrome",
				Type:      "velodrome",
				Timestamp: 1699771973,
				Reserves:  []string{"31229966656506421921", "63506727363"},
				Tokens: []*entity.PoolToken{
					{Address: "0x4200000000000000000000000000000000000006", Swappable: true, Decimals: 18},
					{Address: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Swappable: true, Decimals: 6},
				},
				Extra:       "{\"isPaused\":false,\"fee\":5}",
				StaticExtra: "{\"feePrecision\":10000,\"decimal0\":\"0xde0b6b3a7640000\",\"decimal1\":\"0xde0b6b3a7640000\",\"stable\":false}",
			},
		},
		{
			name: "[stable] it should test calc amount in correctly",
			entityPool: entity.Pool{
				Address:   "0xe08d427724d8a2673fe0be3a81b7db17be835b36",
				Exchange:  "velodrome",
				Type:      "velodrome",
				Timestamp: 1699771973,
				Reserves:  []string{"2052127179", "1705017421"},
				Tokens: []*entity.PoolToken{
					{Address: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Swappable: true, Decimals: 6},
					{Address: "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58", Swappable: true, Decimals: 6},
				},
				Extra:       "{\"isPaused\":false,\"fee\":5}",
				StaticExtra: "{\"feePrecision\":10000,\"decimal0\":\"0xde0b6b3a7640000\",\"decimal1\":\"0xde0b6b3a7640000\",\"stable\":true}",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			poolSimulator, err := NewPoolSimulator(tc.entityPool)
			assert.Nil(t, err)

			testutil.TestCalcAmountIn(t, poolSimulator, 16)
		})
	}
}

// TestPoolSimulator_AeroV2OnChain pins the sim to getAmountOut of the new-generation Aero v2 pools
// (factories 0xaEd253F1.../0x9bc8b3F6..., fee in bps from getBaseFee). Pools were created on an Optimism
// fork; expected values are the pool's on-chain getAmountOut at the same reserves. The mevTaxModule only
// raises the fee for txs paying a priority fee above its threshold, so the base fee is the quote fee.
func TestPoolSimulator_AeroV2OnChain(t *testing.T) {
	t.Parallel()
	tokens := func(a, b string, da, db uint8) []*entity.PoolToken {
		return []*entity.PoolToken{{Address: a, Decimals: da, Swappable: true}, {Address: b, Decimals: db, Swappable: true}}
	}
	const (
		usd6 = "0x1f1278221205d3085e12a356ee866cff98160bd0"
		tkA  = "0xbc738e692a807040ba51076253914c1755933964"
		tkB  = "0x433ffa94ed8f00ce4266adc5f73b8104a5a457f9"
	)
	stable := entity.Pool{Address: "0x9d5dde6eb1a6ecf5c9233e6c57fa5e89a8360fed", Exchange: "aero-v2-stable", Type: DexType,
		Reserves: []string{"5698650000000", "856066212694682583714240"}, Tokens: tokens(usd6, tkA, 6, 18),
		Extra: `{"fee":5}`, StaticExtra: `{"feePrecision":10000,"stable":true}`}
	volatile := entity.Pool{Address: "0x323fdb8e6e11c8125393cd15c924292d3a95e1bd", Exchange: "aero-v2-volatile", Type: DexType,
		Reserves: []string{"1700000000000", "500000000000000000000000"}, Tokens: tokens(usd6, tkB, 6, 18),
		Extra: `{"fee":30}`, StaticExtra: `{"feePrecision":10000}`}

	cases := []struct {
		pool      entity.Pool
		in, out   string
		amountIn  string
		expectOut string
	}{
		{stable, usd6, tkA, "1000000", "425054384880895219"},
		{stable, usd6, tkA, "25000000000", "10547467792929107738338"},
		{stable, usd6, tkA, "900000000000", "296150787558897018905890"},
		{stable, usd6, tkA, "3000000000000", "610008686639850301491959"},
		{stable, tkA, usd6, "1000000000000", "2"},
		{stable, tkA, usd6, "25000000000000000000000", "57750851792"},
		{stable, tkA, usd6, "900000000000000000000000", "1406950373987"},
		{volatile, usd6, tkB, "1000000", "293235122143872483"},
		{volatile, usd6, tkB, "50000000000", "14244078063834042917964"},
		{volatile, usd6, tkB, "1200000000000", "206532246927219997237950"},
		{volatile, tkB, usd6, "1000000000000", "3"},
		{volatile, tkB, usd6, "30000000000000000000000", "95954029929"},
		{volatile, tkB, usd6, "400000000000000000000000", "754294615042"},
	}
	for _, tc := range cases {
		sim, err := NewPoolSimulator(tc.pool)
		assert.NoError(t, err)
		res, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.amountIn)}, TokenOut: tc.out})
		if assert.NoError(t, err, tc.pool.Exchange+" "+tc.amountIn) {
			assert.Equal(t, tc.expectOut, res.TokenAmountOut.Amount.String(), tc.pool.Exchange+" "+tc.amountIn)
		}
	}
}
