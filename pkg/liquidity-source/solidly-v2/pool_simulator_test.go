package solidlyv2

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	poolEncoded = `{"address":"0x9e4cb8b916289864321661ce02cf66aa5ba63c94","amplifiedTvl":1996183.055839599,"exchange":"solidly-v2","type":"solidly-v2","timestamp":1738874095,"reserves":["166579067762010917945","30215077318001718108921"],"tokens":[{"address":"0x4200000000000000000000000000000000000006","name":"","symbol":"","decimals":0,"weight":0,"swappable":true},{"address":"0xde5ed76e7c05ec5e4572cfc88d1acea165109e44","name":"","symbol":"","decimals":0,"weight":0,"swappable":true}],"extra":"{\"isPaused\":false,\"fee\":100}","staticExtra":"{\"feePrecision\":10000,\"decimal0\":\"1000000000000000000\",\"decimal1\":\"1000000000000000000\",\"stable\":false,\"decBig\":null}"}`
	poolEntity  entity.Pool
	_           = lo.Must(0, json.Unmarshal([]byte(poolEncoded), &poolEntity))
	poolSim     = lo.Must(NewPoolSimulator(poolEntity))

	stablePoolEncoded = `{"address":"0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0","exchange":"solidly-v2","type":"solidly-v2","timestamp":1738874095,"reserves":["165363502891169888414","70707320014274856246"],"tokens":[{"address":"0x3e29d3a9316dab217754d13b28646b76607c5f04","name":"","symbol":"","decimals":0,"weight":0,"swappable":true},{"address":"0x6806411765af15bddd26f8f544a34cc40cb9838b","name":"","symbol":"","decimals":0,"weight":0,"swappable":true}],"extra":"{\"isPaused\":false,\"fee\":5}","staticExtra":"{\"feePrecision\":10000,\"decimal0\":\"1000000000000000000\",\"decimal1\":\"1000000000000000000\",\"stable\":true,\"decBig\":null}"}`
	stablePoolEntity  entity.Pool
	_                 = lo.Must(0, json.Unmarshal([]byte(stablePoolEncoded), &stablePoolEntity))
	stablePoolSim     = lo.Must(NewPoolSimulator(stablePoolEntity))
)

func TestNewPoolSimulator(t *testing.T) {
	t.Parallel()
	t.Run("it should init pool simulator correctly", func(t *testing.T) {
		entityPool := entity.Pool{
			Address:   "0x8134a2fdc127549480865fb8e5a9e8a8a95a54c5",
			Exchange:  "solidly-v2",
			Type:      "solidly-v2",
			Timestamp: 1700031705,
			Reserves:  []string{"2455334631692", "48474602535901272544258453"},
			Tokens: []*entity.PoolToken{
				{Address: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Swappable: true},
				{Address: "0x9560e827af36c94d2ac33a39bce1fe78631088db", Swappable: true},
			},
			Extra:       "{\"isPaused\":true,\"fee\":5}",
			StaticExtra: "{\"feePrecision\":10000,\"decimal0\":\"0xf4240\",\"decimal1\":\"0xde0b6b3a7640000\",\"stable\":false}",
		}

		poolSimulator, err := NewPoolSimulator(entityPool)

		assert.Nil(t, err)
		assert.True(t, poolSimulator.isPaused)
		assert.False(t, poolSimulator.stable)
		assert.EqualValues(t, uint64(5), poolSimulator.fee.Uint64())
		assert.Zero(t, number.NewUint256("1000000").Cmp(poolSimulator.decimals0))
		assert.Zero(t, number.NewUint256("1000000000000000000").Cmp(poolSimulator.decimals1))

	})
}

// TestPoolSimulator_getAmountOut
// [volatile][1to0]: https://optimistic.etherscan.io/address/0x8134a2fdc127549480865fb8e5a9e8a8a95a54c5
// [volatile][0to1]: https://optimistic.etherscan.io/address/0x8134a2fdc127549480865fb8e5a9e8a8a95a54c5
// [stable][1to0]: https://optimistic.etherscan.io/address/0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0
// [stable][0to1]: https://optimistic.etherscan.io/address/0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0
func TestPoolSimulator_getAmountOut(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		poolSimulator     PoolSimulator
		tokenAmountIn     poolpkg.TokenAmount
		tokenOut          string
		expectedAmountOut *big.Int
		expectedFee       *big.Int
		calcInThreshold   int64
	}{
		{
			name: "[volatile][0to1] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x8134a2fdc127549480865fb8e5a9e8a8a95a54c5",
						Tokens:   []string{"0x7F5c764cBc14f9669B88837ca1490cCa17c31607", "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db"},
						Reserves: []*big.Int{bignumber.NewBig10("2458244583526"), bignumber.NewBig10("48437610421475879640774762")},
					},
				},
				isPaused:     false,
				stable:       false,
				decimals0:    number.NewUint256("1000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(100),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountIn:     poolpkg.TokenAmount{Token: "0x7F5c764cBc14f9669B88837ca1490cCa17c31607", Amount: bignumber.NewBig10("33762029")},
			tokenOut:          "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db",
			expectedAmountOut: bignumber.NewBig10("658590483453928603087"),
			expectedFee:       bignumber.NewBig10("337620"),
			calcInThreshold:   10,
		},
		{
			name: "[volatile][1to0] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x8134a2fdc127549480865fb8e5a9e8a8a95a54c5",
						Tokens:   []string{"0x7F5c764cBc14f9669B88837ca1490cCa17c31607", "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db"},
						Reserves: []*big.Int{bignumber.NewBig10("2458244583526"), bignumber.NewBig10("48437697487082485250805965")},
					},
				},
				isPaused:     false,
				stable:       false,
				decimals0:    number.NewUint256("1000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(100),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountIn:     poolpkg.TokenAmount{Token: "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db", Amount: bignumber.NewBig10("4843761042147587964077")},
			tokenOut:          "0x7F5c764cBc14f9669B88837ca1490cCa17c31607",
			expectedAmountOut: bignumber.NewBig10("243341685"),
			expectedFee:       bignumber.NewBig10("48437610421475879640"),
			calcInThreshold:   1,
		},
		{
			name: "[stable][1to0] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0",
						Tokens:   []string{"0x3e29d3a9316dab217754d13b28646b76607c5f04", "0x6806411765af15bddd26f8f544a34cc40cb9838b"},
						Reserves: []*big.Int{bignumber.NewBig10("165363502891169888414"), bignumber.NewBig10("70707320014274856246")},
					},
				},
				isPaused:     false,
				stable:       true,
				decimals0:    number.NewUint256("1000000000000000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountIn:     poolpkg.TokenAmount{Token: "0x6806411765af15bddd26f8f544a34cc40cb9838b", Amount: bignumber.NewBig10("7070085324939016")},
			tokenOut:          "0x3e29d3a9316dab217754d13b28646b76607c5f04",
			expectedAmountOut: bignumber.NewBig10("8040168956751976"),
			expectedFee:       bignumber.NewBig10("3535042662469"),
			calcInThreshold:   10,
		},
		{
			name: "[stable][0to1] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0",
						Tokens:   []string{"0x3e29d3a9316dab217754d13b28646b76607c5f04", "0x6806411765af15bddd26f8f544a34cc40cb9838b"},
						Reserves: []*big.Int{bignumber.NewBig10("165363502891169888414"), bignumber.NewBig10("70707320014274856246")},
					},
				},
				isPaused:     false,
				stable:       true,
				decimals0:    number.NewUint256("1000000000000000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountIn:     poolpkg.TokenAmount{Token: "0x3e29d3a9316dab217754d13b28646b76607c5f04", Amount: bignumber.NewBig10("7070085324939016")},
			tokenOut:          "0x6806411765af15bddd26f8f544a34cc40cb9838b",
			expectedAmountOut: bignumber.NewBig10("6210478971090850"),
			expectedFee:       bignumber.NewBig10("3535042662469"),
			calcInThreshold:   10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
				return tc.poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{TokenAmountIn: tc.tokenAmountIn, TokenOut: tc.tokenOut})
			})

			if tc.expectedAmountOut != nil {
				assert.Nil(t, err)
				assert.Zero(t, tc.expectedAmountOut.Cmp(result.TokenAmountOut.Amount))
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
			name: "[volatile][1to0] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x8134a2fdc127549480865fb8e5a9e8a8a95a54c5",
						Tokens:   []string{"0x7F5c764cBc14f9669B88837ca1490cCa17c31607", "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db"},
						Reserves: []*big.Int{bignumber.NewBig10("2458244583526"), bignumber.NewBig10("48437610421475879640774762")},
					},
				},
				isPaused:     false,
				stable:       false,
				decimals0:    number.NewUint256("1000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(100),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountOut:   poolpkg.TokenAmount{Token: "0x7F5c764cBc14f9669B88837ca1490cCa17c31607", Amount: bignumber.NewBig10("33762029")},
			tokenIn:          "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db",
			expectedAmountIn: bignumber.NewBig10("671980897831826369735"),
			expectedFee:      bignumber.NewBig10("0"),
		},
		{
			name: "[volatile][0to1] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x8134a2fdc127549480865fb8e5a9e8a8a95a54c5",
						Tokens:   []string{"0x7F5c764cBc14f9669B88837ca1490cCa17c31607", "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db"},
						Reserves: []*big.Int{bignumber.NewBig10("2458244583526"), bignumber.NewBig10("48437697487082485250805965")},
					},
				},
				isPaused:     false,
				stable:       false,
				decimals0:    number.NewUint256("1000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(100),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountOut:   poolpkg.TokenAmount{Token: "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db", Amount: bignumber.NewBig10("4843761042147587964077")},
			tokenIn:          "0x7F5c764cBc14f9669B88837ca1490cCa17c31607",
			expectedAmountIn: bignumber.NewBig10("248331921"),
			expectedFee:      bignumber.NewBig10("0"),
		},
		{
			name: "[stable][1to0] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0",
						Tokens:   []string{"0x3e29d3a9316dab217754d13b28646b76607c5f04", "0x6806411765af15bddd26f8f544a34cc40cb9838b"},
						Reserves: []*big.Int{bignumber.NewBig10("165363502891169888414"), bignumber.NewBig10("70707320014274856246")},
					},
				},
				isPaused:     false,
				stable:       true,
				decimals0:    number.NewUint256("1000000000000000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountOut:   poolpkg.TokenAmount{Token: "0x3e29d3a9316dab217754d13b28646b76607c5f04", Amount: bignumber.NewBig10("8040168956751976")},
			tokenIn:          "0x6806411765af15bddd26f8f544a34cc40cb9838b",
			expectedAmountIn: bignumber.NewBig10("7070085324939017"),
			expectedFee:      bignumber.NewBig10("0"),
		},
		{
			name: "[stable][0to1] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0",
						Tokens:   []string{"0x3e29d3a9316dab217754d13b28646b76607c5f04", "0x6806411765af15bddd26f8f544a34cc40cb9838b"},
						Reserves: []*big.Int{bignumber.NewBig10("165363502891169888414"), bignumber.NewBig10("70707320014274856246")},
					},
				},
				isPaused:     false,
				stable:       true,
				decimals0:    number.NewUint256("1000000000000000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountOut:   poolpkg.TokenAmount{Token: "0x6806411765af15bddd26f8f544a34cc40cb9838b", Amount: bignumber.NewBig10("6210478971090850")},
			tokenIn:          "0x3e29d3a9316dab217754d13b28646b76607c5f04",
			expectedAmountIn: bignumber.NewBig10("7070085324939017"),
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
				assert.Equalf(t, tc.expectedAmountIn, result.TokenAmountIn.Amount, "expected amount in: %s, got: %s", tc.expectedAmountIn.String(), result.TokenAmountIn.Amount.String())
				assert.Equalf(t, tc.expectedFee, result.Fee.Amount, "expected fee: %s, got: %s", tc.expectedFee.String(), result.Fee.Amount.String())
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
		expectedReserves []*big.Int
	}{
		{
			name: "[volatile][1to0] it should update reserve correctly",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x8134a2fdc127549480865fb8e5a9e8a8a95a54c5",
						Tokens:   []string{"0x7F5c764cBc14f9669B88837ca1490cCa17c31607", "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db"},
						Reserves: []*big.Int{bignumber.NewBig10("2458244583526"), bignumber.NewBig10("48437610421475879640774762")},
					},
				},
				isPaused:     false,
				stable:       false,
				decimals0:    number.NewUint256("1000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(100),
				feePrecision: uint256.NewInt(10000),
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x7F5c764cBc14f9669B88837ca1490cCa17c31607", Amount: bignumber.NewBig10("243341685")},
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db", Amount: bignumber.NewBig10("4843761042147587964077")},
				Fee:            poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Amount: bignumber.NewBig10("337620")},
			},
			expectedReserves: []*big.Int{bignumber.NewBig10("2458001241841"), bignumber.NewBig10("48442454182518027228401219")},
		},
		{
			name: "[volatile][0to1] it should update reserve correctly",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x8134a2fdc127549480865fb8e5a9e8a8a95a54c5",
						Tokens:   []string{"0x7F5c764cBc14f9669B88837ca1490cCa17c31607", "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db"},
						Reserves: []*big.Int{bignumber.NewBig10("2458244583526"), bignumber.NewBig10("48437610421475879640774762")},
					},
				},
				isPaused:     false,
				stable:       false,
				decimals0:    number.NewUint256("1000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(100),
				feePrecision: uint256.NewInt(10000),
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db", Amount: bignumber.NewBig10("658590483453928603087")},
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0x7F5c764cBc14f9669B88837ca1490cCa17c31607", Amount: bignumber.NewBig10("33762029")},
				Fee:            poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Amount: bignumber.NewBig10("337620")},
			},
			expectedReserves: []*big.Int{bignumber.NewBig10("2458278007935"), bignumber.NewBig10("48436951830992425712171675")},
		},
		{
			name: "[stable][1to0] it should update reserve correctly",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0",
						Tokens:   []string{"0x3e29d3a9316dab217754d13b28646b76607c5f04", "0x6806411765af15bddd26f8f544a34cc40cb9838b"},
						Reserves: []*big.Int{bignumber.NewBig10("165363502891169888414"), bignumber.NewBig10("70707320014274856246")},
					},
				},
				isPaused:     false,
				stable:       false,
				decimals0:    number.NewUint256("1000000000000000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x3e29d3a9316dab217754d13b28646b76607c5f04", Amount: bignumber.NewBig10("8040168956751976")},
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0x6806411765af15bddd26f8f544a34cc40cb9838b", Amount: bignumber.NewBig10("7070085324939016")},
				Fee:            poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Amount: bignumber.NewBig10("3535042662469")},
			},
			expectedReserves: []*big.Int{bignumber.NewBig10("165355462722213136438"), bignumber.NewBig10("70714386564557132793")},
		},
		{
			name: "[stable][0to1] it should update reserve correctly",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0",
						Tokens:   []string{"0x3e29d3a9316dab217754d13b28646b76607c5f04", "0x6806411765af15bddd26f8f544a34cc40cb9838b"},
						Reserves: []*big.Int{bignumber.NewBig10("165363502891169888414"), bignumber.NewBig10("70707320014274856246")},
					},
				},
				isPaused:     false,
				stable:       false,
				decimals0:    number.NewUint256("1000000000000000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x6806411765af15bddd26f8f544a34cc40cb9838b", Amount: bignumber.NewBig10("6210478971090850")},
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0x3e29d3a9316dab217754d13b28646b76607c5f04", Amount: bignumber.NewBig10("7070085324939016")},
				Fee:            poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Amount: bignumber.NewBig10("3535042662469")},
			},
			expectedReserves: []*big.Int{bignumber.NewBig10("165370569441452164961"), bignumber.NewBig10("70701109535303765396")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.poolSimulator.UpdateBalance(tc.params)
			fmt.Println("reserves, reserves1", tc.poolSimulator.Info.Reserves[0].String(), tc.poolSimulator.Info.Reserves[1])

			assert.Zero(t, tc.expectedReserves[0].Cmp(tc.poolSimulator.Info.Reserves[0]))
			assert.Zero(t, tc.expectedReserves[1].Cmp(tc.poolSimulator.Info.Reserves[1]))
		})
	}
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountIn(t, poolSim)
	testutil.TestCalcAmountIn(t, stablePoolSim)
}

