package velodromev2

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
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	poolEncoded = `{"address":"0x9e4cb8b916289864321661ce02cf66aa5ba63c94","amplifiedTvl":1996183.055839599,"exchange":"aerodrome","type":"velodrome-v2","timestamp":1738874095,"reserves":["166579067762010917945","30215077318001718108921"],"tokens":[{"address":"0x4200000000000000000000000000000000000006","name":"","symbol":"","decimals":0,"weight":0,"swappable":true},{"address":"0xde5ed76e7c05ec5e4572cfc88d1acea165109e44","name":"","symbol":"","decimals":0,"weight":0,"swappable":true}],"extra":"{\"isPaused\":false,\"fee\":100}","staticExtra":"{\"feePrecision\":10000,\"decimal0\":\"1000000000000000000\",\"decimal1\":\"1000000000000000000\",\"stable\":false,\"decBig\":null}"}`
	poolEntity  entity.Pool
	_           = lo.Must(0, json.Unmarshal([]byte(poolEncoded), &poolEntity))
	poolSim     = lo.Must(NewPoolSimulator(poolEntity))
)

// TestPoolSimulator_getAmountOut
// [volatile][1to0]: https://optimistic.etherscan.io/address/0x8134a2fdc127549480865fb8e5a9e8a8a95a54c5
// [volatile][0to1]: https://optimistic.etherscan.io/address/0x8134a2fdc127549480865fb8e5a9e8a8a95a54c5
// [stable][1to0]: https://optimistic.etherscan.io/address/0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0
// [stable][0to1]: https://optimistic.etherscan.io/address/0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0
func TestPoolSimulator_getAmountOut(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		poolSimulator     poolpkg.IPoolSimulator
		tokenAmountIn     poolpkg.TokenAmount
		tokenOut          string
		expectedAmountOut *big.Int
		expectedFee       *big.Int
	}{
		{
			name: "[volatile][0to1] it should return correct amount",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x8134a2fdc127549480865fb8e5a9e8a8a95a54c5",
						Tokens:   []string{"0x7F5c764cBc14f9669B88837ca1490cCa17c31607", "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db"},
						Reserves: []*big.Int{utils.NewBig10("2458244583526"), utils.NewBig10("48437610421475879640774762")},
					},
				},
				isPaused:     false,
				stable:       false,
				decimals0:    number.NewUint256("1000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(100),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountIn:     poolpkg.TokenAmount{Token: "0x7F5c764cBc14f9669B88837ca1490cCa17c31607", Amount: utils.NewBig10("33762029")},
			tokenOut:          "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db",
			expectedAmountOut: utils.NewBig10("658590483453928603087"),
			expectedFee:       utils.NewBig10("337620"),
		},
		{
			name: "[volatile][1to0] it should return correct amount",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x8134a2fdc127549480865fb8e5a9e8a8a95a54c5",
						Tokens:   []string{"0x7F5c764cBc14f9669B88837ca1490cCa17c31607", "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db"},
						Reserves: []*big.Int{utils.NewBig10("2458244583526"), utils.NewBig10("48437697487082485250805965")},
					},
				},
				isPaused:     false,
				stable:       false,
				decimals0:    number.NewUint256("1000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(100),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountIn:     poolpkg.TokenAmount{Token: "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db", Amount: utils.NewBig10("4843761042147587964077")},
			tokenOut:          "0x7F5c764cBc14f9669B88837ca1490cCa17c31607",
			expectedAmountOut: utils.NewBig10("243341685"),
			expectedFee:       utils.NewBig10("48437610421475879640"),
		},
		{
			name: "[stable][1to0] it should return correct amount",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0",
						Tokens:   []string{"0x3e29d3a9316dab217754d13b28646b76607c5f04", "0x6806411765af15bddd26f8f544a34cc40cb9838b"},
						Reserves: []*big.Int{utils.NewBig10("165363502891169888414"), utils.NewBig10("70707320014274856246")},
					},
				},
				isPaused:     false,
				stable:       true,
				decimals0:    number.NewUint256("1000000000000000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountIn:     poolpkg.TokenAmount{Token: "0x6806411765af15bddd26f8f544a34cc40cb9838b", Amount: utils.NewBig10("7070085324939016")},
			tokenOut:          "0x3e29d3a9316dab217754d13b28646b76607c5f04",
			expectedAmountOut: utils.NewBig10("8040168956751976"),
			expectedFee:       utils.NewBig10("3535042662469"),
		},
		{
			name: "[stable][0to1] it should return correct amount",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0",
						Tokens:   []string{"0x3e29d3a9316dab217754d13b28646b76607c5f04", "0x6806411765af15bddd26f8f544a34cc40cb9838b"},
						Reserves: []*big.Int{utils.NewBig10("165363502891169888414"), utils.NewBig10("70707320014274856246")},
					},
				},
				isPaused:     false,
				stable:       true,
				decimals0:    number.NewUint256("1000000000000000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(5),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountIn:     poolpkg.TokenAmount{Token: "0x3e29d3a9316dab217754d13b28646b76607c5f04", Amount: utils.NewBig10("7070085324939016")},
			tokenOut:          "0x6806411765af15bddd26f8f544a34cc40cb9838b",
			expectedAmountOut: utils.NewBig10("6210478971090850"),
			expectedFee:       utils.NewBig10("3535042662469"),
		},
		{
			name:              "[volatile][0to1] aerodrome should return correct amount",
			poolSimulator:     poolSim,
			tokenAmountIn:     poolpkg.TokenAmount{Token: "0x4200000000000000000000000000000000000006", Amount: utils.NewBig10("10000000000000000")},
			tokenOut:          "0xde5ed76e7c05ec5e4572cfc88d1acea165109e44",
			expectedAmountOut: utils.NewBig10("1795612695527072515"),
			expectedFee:       utils.NewBig10("100000000000000"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
				return tc.poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{TokenAmountIn: tc.tokenAmountIn, TokenOut: tc.tokenOut})
			})

			if tc.expectedAmountOut != nil {
				assert.Nil(t, err)
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
				assert.Equal(t, tc.expectedFee, result.Fee.Amount)
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
						Reserves: []*big.Int{utils.NewBig10("2458244583526"), utils.NewBig10("48437610421475879640774762")},
					},
				},
				isPaused:     false,
				stable:       false,
				decimals0:    number.NewUint256("1000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(100),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountOut:   poolpkg.TokenAmount{Token: "0x7F5c764cBc14f9669B88837ca1490cCa17c31607", Amount: utils.NewBig10("33762029")},
			tokenIn:          "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db",
			expectedAmountIn: utils.NewBig10("671980897831826369735"),
			expectedFee:      utils.NewBig10("0"),
		},
		{
			name: "[volatile][0to1] it should return correct amount",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x8134a2fdc127549480865fb8e5a9e8a8a95a54c5",
						Tokens:   []string{"0x7F5c764cBc14f9669B88837ca1490cCa17c31607", "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db"},
						Reserves: []*big.Int{utils.NewBig10("2458244583526"), utils.NewBig10("48437697487082485250805965")},
					},
				},
				isPaused:     false,
				stable:       false,
				decimals0:    number.NewUint256("1000000"),
				decimals1:    number.NewUint256("1000000000000000000"),
				fee:          uint256.NewInt(100),
				feePrecision: uint256.NewInt(10000),
			},
			tokenAmountOut:   poolpkg.TokenAmount{Token: "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db", Amount: utils.NewBig10("4843761042147587964077")},
			tokenIn:          "0x7F5c764cBc14f9669B88837ca1490cCa17c31607",
			expectedAmountIn: utils.NewBig10("248331921"),
			expectedFee:      utils.NewBig10("0"),
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
						Reserves: []*big.Int{utils.NewBig10("2458244583526"), utils.NewBig10("48437610421475879640774762")},
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
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x7F5c764cBc14f9669B88837ca1490cCa17c31607", Amount: utils.NewBig10("243341685")},
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db", Amount: utils.NewBig10("4843761042147587964077")},
				Fee:            poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Amount: utils.NewBig10("337620")},
			},
			expectedReserves: []*big.Int{utils.NewBig10("2458001241841"), utils.NewBig10("48442454182518027228401219")},
		},
		{
			name: "[volatile][0to1] it should update reserve correctly",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x8134a2fdc127549480865fb8e5a9e8a8a95a54c5",
						Tokens:   []string{"0x7F5c764cBc14f9669B88837ca1490cCa17c31607", "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db"},
						Reserves: []*big.Int{utils.NewBig10("2458244583526"), utils.NewBig10("48437610421475879640774762")},
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
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x9560e827aF36c94D2Ac33a39bCE1Fe78631088Db", Amount: utils.NewBig10("658590483453928603087")},
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0x7F5c764cBc14f9669B88837ca1490cCa17c31607", Amount: utils.NewBig10("33762029")},
				Fee:            poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Amount: utils.NewBig10("337620")},
			},
			expectedReserves: []*big.Int{utils.NewBig10("2458278007935"), utils.NewBig10("48436951830992425712171675")},
		},
		{
			name: "[stable][1to0] it should update reserve correctly",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0",
						Tokens:   []string{"0x3e29d3a9316dab217754d13b28646b76607c5f04", "0x6806411765af15bddd26f8f544a34cc40cb9838b"},
						Reserves: []*big.Int{utils.NewBig10("165363502891169888414"), utils.NewBig10("70707320014274856246")},
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
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x3e29d3a9316dab217754d13b28646b76607c5f04", Amount: utils.NewBig10("8040168956751976")},
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0x6806411765af15bddd26f8f544a34cc40cb9838b", Amount: utils.NewBig10("7070085324939016")},
				Fee:            poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Amount: utils.NewBig10("3535042662469")},
			},
			expectedReserves: []*big.Int{utils.NewBig10("165355462722213136438"), utils.NewBig10("70714386564557132793")},
		},
		{
			name: "[stable][0to1] it should update reserve correctly",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0",
						Tokens:   []string{"0x3e29d3a9316dab217754d13b28646b76607c5f04", "0x6806411765af15bddd26f8f544a34cc40cb9838b"},
						Reserves: []*big.Int{utils.NewBig10("165363502891169888414"), utils.NewBig10("70707320014274856246")},
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
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x6806411765af15bddd26f8f544a34cc40cb9838b", Amount: utils.NewBig10("6210478971090850")},
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0x3e29d3a9316dab217754d13b28646b76607c5f04", Amount: utils.NewBig10("7070085324939016")},
				Fee:            poolpkg.TokenAmount{Token: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Amount: utils.NewBig10("3535042662469")},
			},
			expectedReserves: []*big.Int{utils.NewBig10("165370569441452164961"), utils.NewBig10("70701109535303765396")},
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
}
