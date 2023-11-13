package velodromev1

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestNewPoolSimulator(t *testing.T) {
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
		assert.Zero(t, number.NewUint256("1000000000000000000").Cmp(poolSimulator.decimals0))
		assert.Zero(t, number.NewUint256("1000000000000000000").Cmp(poolSimulator.decimals1))

	})
}

// TestPoolSimulator_getAmountOut
// [volatile][1to0]: https://optimistic.etherscan.io/tx/0x127f10c9a2562015e4881f45a2837a43100d11e33f4cc3198a15e57bf1f18869
// [volatile][0to1]: https://optimistic.etherscan.io/tx/0x1b9e4744d94390c48687c18ab6ddf951c8710580b6a9cb7c44545bae5a370705
// [stable][1to0]: https://optimistic.etherscan.io/tx/0x6d1ebd5f31077408f5b33718156403e0aef83d4a8614e7162a3d38bf9653cd7c
// [stable][0to1]: https://optimistic.etherscan.io/tx/0x5c92ec5d38d51e33952777e4468d8499cc1b9f0ce40e698caedde3f23b3d37e7
func TestPoolSimulator_getAmountOut(t *testing.T) {
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
						Address:  "0x79c912fef520be002c2b6e57ec4324e260f38e50",
						Tokens:   []string{"0x4200000000000000000000000000000000000006", "0x7f5c764cbc14f9669b88837ca1490cca17c31607"},
						Reserves: []*big.Int{utils.NewBig10("31229966656506421921"), utils.NewBig10("63506727363")},
					},
				},
				isPaused:     false,
				stable:       false,
				decimals0:    number.NewUint256("1000000000000000000"),
				decimals1:    number.NewUint256("1000000"),
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountIn:     poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Amount: utils.NewBig10("33762029")},
			tokenOut:          "0x4200000000000000000000000000000000000006",
			expectedAmountOut: utils.NewBig10("16585646993362100"),
			expectedFee:       utils.NewBig10("16881"),
		},
		{
			name: "[volatile][0to1] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x79c912fef520be002c2b6e57ec4324e260f38e50",
						Tokens:   []string{"0x4200000000000000000000000000000000000006", "0x7f5c764cbc14f9669b88837ca1490cca17c31607"},
						Reserves: []*big.Int{utils.NewBig10("31220354779450883153"), utils.NewBig10("63526279313")},
					},
				},
				isPaused:     false,
				stable:       false,
				decimals0:    number.NewUint256("1000000000000000000"),
				decimals1:    number.NewUint256("1000000"),
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountIn:     poolpkg.TokenAmount{Token: "0x4200000000000000000000000000000000000006", Amount: utils.NewBig10("3655170221820867")},
			tokenOut:          "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
			expectedAmountOut: utils.NewBig10("7432846"),
			expectedFee:       utils.NewBig10("1827585110910"),
		},
		{
			name: "[stable][1to0] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0xe08d427724d8a2673fe0be3a81b7db17be835b36",
						Tokens:   []string{"0x7f5c764cbc14f9669b88837ca1490cca17c31607", "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58"},
						Reserves: []*big.Int{utils.NewBig10("2052127179"), utils.NewBig10("1705017421")},
					},
				},
				isPaused:     false,
				stable:       true,
				decimals0:    number.NewUint256("1000000"),
				decimals1:    number.NewUint256("1000000"),
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountIn:     poolpkg.TokenAmount{Token: "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58", Amount: utils.NewBig10("36283954")},
			tokenOut:          "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
			expectedAmountOut: utils.NewBig10("36307464"),
			expectedFee:       utils.NewBig10("18141"),
		},
		{
			name: "[stable][0to1] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0xe08d427724d8a2673fe0be3a81b7db17be835b36",
						Tokens:   []string{"0x7f5c764cbc14f9669b88837ca1490cca17c31607", "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58"},
						Reserves: []*big.Int{utils.NewBig10("6110873648"), utils.NewBig10("6651345170")},
					},
				},
				isPaused:     false,
				stable:       true,
				decimals0:    number.NewUint256("1000000"),
				decimals1:    number.NewUint256("1000000"),
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountIn:     poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Amount: utils.NewBig10("302268108")},
			tokenOut:          "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58",
			expectedAmountOut: utils.NewBig10("302127234"),
			expectedFee:       utils.NewBig10("151134"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountOut(tc.tokenAmountIn, tc.tokenOut)

			if tc.expectedAmountOut != nil {
				assert.Nil(t, err)
				assert.Zero(t, tc.expectedAmountOut.Cmp(result.TokenAmountOut.Amount))
				assert.Zero(t, tc.expectedFee.Cmp(result.Fee.Amount))
			}
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	testCases := []struct {
		name             string
		poolSimulator    PoolSimulator
		params           poolpkg.UpdateBalanceParams
		expectedReserves []*big.Int
	}{
		{
			name: "[volatile][1to0] it should update reserve correctly",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x79c912fef520be002c2b6e57ec4324e260f38e50",
						Tokens:   []string{"0x4200000000000000000000000000000000000006", "0x7f5c764cbc14f9669b88837ca1490cca17c31607"},
						Reserves: []*big.Int{utils.NewBig10("31229966656506421921"), utils.NewBig10("63506727363")},
					},
				},
				isPaused:     false,
				stable:       false,
				decimals0:    number.NewUint256("1000000000000000000"),
				decimals1:    number.NewUint256("1000000"),
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x4200000000000000000000000000000000000006", Amount: utils.NewBig10("16585646993362100")},
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Amount: utils.NewBig10("33762029")},
				Fee:            poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Amount: utils.NewBig10("16881")},
			},
			expectedReserves: []*big.Int{utils.NewBig10("31213381009513059821"), utils.NewBig10("63540472511")},
		},
		{
			name: "[volatile][0to1] it should update reserve correctly",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x79c912fef520be002c2b6e57ec4324e260f38e50",
						Tokens:   []string{"0x4200000000000000000000000000000000000006", "0x7f5c764cbc14f9669b88837ca1490cca17c31607"},
						Reserves: []*big.Int{utils.NewBig10("31220354779450883153"), utils.NewBig10("63526279313")},
					},
				},
				isPaused:     false,
				stable:       false,
				decimals0:    number.NewUint256("1000000000000000000"),
				decimals1:    number.NewUint256("1000000"),
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Amount: utils.NewBig10("7432846")},
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0x4200000000000000000000000000000000000006", Amount: utils.NewBig10("3655170221820867")},
				Fee:            poolpkg.TokenAmount{Token: "0x4200000000000000000000000000000000000006", Amount: utils.NewBig10("1827585110910")},
			},
			expectedReserves: []*big.Int{utils.NewBig10("31224008122087593110"), utils.NewBig10("63518846467")},
		},
		{
			name: "[stable][1to0] it should update reserve correctly",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0xe08d427724d8a2673fe0be3a81b7db17be835b36",
						Tokens:   []string{"0x7f5c764cbc14f9669b88837ca1490cca17c31607", "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58"},
						Reserves: []*big.Int{utils.NewBig10("2052127179"), utils.NewBig10("1705017421")},
					},
				},
				isPaused:     false,
				stable:       true,
				decimals0:    number.NewUint256("1000000"),
				decimals1:    number.NewUint256("1000000"),
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Amount: utils.NewBig10("36307464")},
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58", Amount: utils.NewBig10("36283954")},
				Fee:            poolpkg.TokenAmount{Token: "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58", Amount: utils.NewBig10("18141")},
			},
			expectedReserves: []*big.Int{utils.NewBig10("2015819715"), utils.NewBig10("1741283234")},
		},
		{
			name: "[stable][0to1] it should update reserve correctly",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0xe08d427724d8a2673fe0be3a81b7db17be835b36",
						Tokens:   []string{"0x7f5c764cbc14f9669b88837ca1490cca17c31607", "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58"},
						Reserves: []*big.Int{utils.NewBig10("6110873648"), utils.NewBig10("6651345170")},
					},
				},
				isPaused:     false,
				stable:       true,
				decimals0:    number.NewUint256("1000000"),
				decimals1:    number.NewUint256("1000000"),
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58", Amount: utils.NewBig10("302127234")},
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Amount: utils.NewBig10("302268108")},
				Fee:            poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Amount: utils.NewBig10("151134")},
			},
			expectedReserves: []*big.Int{utils.NewBig10("6412990622"), utils.NewBig10("6349217936")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.poolSimulator.UpdateBalance(tc.params)

			assert.Zero(t, tc.expectedReserves[0].Cmp(tc.poolSimulator.Info.Reserves[0]))
			assert.Zero(t, tc.expectedReserves[1].Cmp(tc.poolSimulator.Info.Reserves[1]))
		})
	}
}
