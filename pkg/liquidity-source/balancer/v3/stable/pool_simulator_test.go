package stable

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0xc4ce391d82d164c166df9c8336ddf84206b2f812","exchange":"balancer-v3-stable","type":"balancer-v3-stable","timestamp":1751293016,"reserves":["687804073931103275644","1783969556654743519024"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"},{"address":"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"}],"extra":"{\"hook\":{},\"fee\":\"20000000000000\",\"aggrFee\":\"500000000000000000\",\"balsE18\":[\"694069210892948295209\",\"2124492373418339554414\"],\"decs\":[\"1\",\"1\"],\"rates\":[\"1009108897721464489\",\"1190879275654308905\"],\"buffs\":[{\"dRate\":[\"976255\",\"976255817341\",\"976255817341645373\",\"976255817341645373456045\",\"976255817341645373456045753577\"],\"rRate\":[\"1024321\",\"1024321681096\",\"1024321681096877127\",\"1024321681096877127977750\",\"1024321681096877127977750950000\"]},{\"dRate\":[\"996629\",\"996629442697\",\"996629442697471179\",\"996629442697471179789157\",\"996629442697471179789157582365\"],\"rRate\":[\"1003381\",\"1003381956380\",\"1003381956380303285\",\"1003381956380303285385258\",\"1003381956380303285385258382000\"]}],\"surge\":{},\"ampParam\":\"5000000\"}","staticExtra":"{\"buffs\":[\"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9\",\"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29\"]}","blockNumber":22817774}`),
		&entityPool)
	poolSim = lo.Must(NewPoolSimulator(entityPool))
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		0: {
			0: {
				"1000000000": "invalid token",
			},
			1: { // underlying 0 -> underlying 1
				"799999":              "amount in is too small",
				"1000000000":          "830329790",
				"2000000000000000000": "1660660554496124258",
			},
			2: { // underlying 0 -> wrapped 0
				"799999":              "781003",
				"1000000000":          "976255000",
				"2000000000000000000": "1952511634683290746",
			},
			3: { // underlying 0 -> wrapped 1
				"799999":              "amount in is too small",
				"1000000000":          "827531905",
				"2000000000000000000": "1655063202937145788",
			},
		},
		1: {
			0: {
				"842557":              "amount out is too small",
				"1000000000":          "1204289875",
				"2000000000000000000": "2408577306041957034",
			},
			1: {
				"1000000000": "invalid token",
			},
			2: {
				"842557":              "amount out is too small",
				"1000000000":          "1175694997",
				"2000000000000000000": "2351387606540529096",
			},
			3: {
				"842557":              "839716",
				"1000000000":          "996629000",
				"2000000000000000000": "1993258885394942358",
			},
		},
		2: {
			0: {
				"842557":              "863048",
				"1000000000":          "1024321000",
				"2000000000000000000": "2048643362193754254",
			},
			1: {
				"842557":              "amount in is too small",
				"1000000000":          "850525518",
				"2000000000000000000": "1701050561113724421",
			},
			2: {
				"1000000000": "invalid token",
			},
			3: {
				"842557":              "amount in is too small",
				"1000000000":          "847659581",
				"2000000000000000000": "1695317072722991811",
			},
		},
		3: {
			0: {
				"842557":              "1018110",
				"1000000000":          "1208363268",
				"2000000000000000000": "2416722997438206073",
			},
			1: {
				"842557":              "845405",
				"1000000000":          "1003381000",
				"2000000000000000000": "2006763912760606570",
			},
			2: {
				"842557":              "993937",
				"1000000000":          "1179671670",
				"2000000000000000000": "2359339885152387011",
			},
			3: {
				"1000000000": "invalid token",
			},
		},
	})
}

func TestCalcAmountIn(t *testing.T) {
	testutil.TestCalcAmountIn(t, poolSim, 8)
}

func TestCanSwapFrom(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Underlying Swap",
			input:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			expected: []string{"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29", "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9"},
		},
		{
			name:     "Wrapped Swap",
			input:    "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			expected: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := poolSim.CanSwapFrom(tc.input)
			assert.ElementsMatch(t, tc.expected, result)
		})
	}
}
func TestCanSwapTo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Underlying Swap",
			input:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			expected: []string{"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"},
		},
		{
			name:     "Wrapped Swap",
			input:    "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			expected: []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := poolSim.CanSwapTo(tc.input)
			assert.ElementsMatch(t, tc.expected, result)
		})
	}
}

func TestUpdateBalance(t *testing.T) {
	t.Parallel()

	testPool := poolSim.CloneState().(*base.PoolSimulator)
	// Update reserves for test examples
	testPool.Info.Reserves[0] = big.NewInt(2e18)
	testPool.Info.Reserves[1] = big.NewInt(4e18)

	testcases := []struct {
		name             string
		tokenIn          string
		tokenInAmount    *big.Int
		tokenOut         string
		tokenOutAmount   *big.Int
		expectedReserves []*big.Int
		aggregateFee     *big.Int
	}{
		// buffer swap -> directly swaps/un/wraps on buffer/ERC4626 and does not affect pool reserves
		{
			name:             "Buffer Swap - Wrap",
			tokenIn:          "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", // WETH
			tokenInAmount:    big.NewInt(1e18),
			tokenOut:         "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9", // waEthLidoWETH
			tokenOutAmount:   big.NewInt(2e18),
			expectedReserves: testPool.GetReserves(), // Reserves remain unchanged
			aggregateFee:     big.NewInt(1e15),       // 0.001 (Not relevant for this test)
		},
		{
			name:             "Buffer Swap - Unwrap",
			tokenIn:          "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", // wstETH
			tokenInAmount:    big.NewInt(1e18),
			tokenOut:         "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29", // waEthLidowstETH
			tokenOutAmount:   big.NewInt(2e18),
			expectedReserves: testPool.GetReserves(), // Reserves remain unchanged
			aggregateFee:     big.NewInt(1e15),       // 0.001 (Not relevant for this test)
		},
		// swap -> amounts directly
		{
			name:           "Swap",
			tokenIn:        "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29", // waEthLidowstETH
			tokenInAmount:  big.NewInt(2e18),
			tokenOut:       "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9", // waEthLidoWETH
			tokenOutAmount: big.NewInt(1e18),
			expectedReserves: []*big.Int{
				big.NewInt(1e18), // waEthLidoWETH reserve - out
				big.NewInt(6e18), // waEthLidowstETH reserve + in
			},
			aggregateFee: big.NewInt(0), // Zero fee for test examples
		},
		// wrap>swap -> amount in will have rates
		{
			name:           "Wrap>Swap - underlyingIn[wrap]wrappedIn[swap]wrappedOut",
			tokenIn:        "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", // weth
			tokenInAmount:  big.NewInt(2e18),
			tokenOut:       "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29", // waEthLidowstETH
			tokenOutAmount: big.NewInt(1e18),
			expectedReserves: []*big.Int{
				big.NewInt(3952511634683290746), // waEthLidoWETH + in/ERC4626rate
				big.NewInt(3e18),                // waEthLidowstETH reserve - out
			},
			aggregateFee: big.NewInt(0), // Zero fee for test examples
		},
		// swap>unwrap -> amount out will have rates
		{
			name:           "Swap>Unwrap - wrappedIn[swap]wrappedOut[unwrap]underlyingOut",
			tokenIn:        "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9", // waEthLidoWETH
			tokenInAmount:  big.NewInt(2e18),
			tokenOut:       "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", // wstETH
			tokenOutAmount: big.NewInt(1e18),
			expectedReserves: []*big.Int{
				big.NewInt(4e18),                // waEthLidoWETH + in
				big.NewInt(3003370557302528821), // waEthLidowstETH reserve - out/ERC4626rate
			},
			aggregateFee: big.NewInt(0), // Zero fee for test examples
		},
		// Wrap>Swap>Unwrap -> amount in & out will have rates
		{
			name:           "Wrap>Swap>Unwrap - wrappedIn[swap]wrappedOut[unwrap]underlyingOut",
			tokenIn:        "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", // weth
			tokenInAmount:  big.NewInt(2e18),
			tokenOut:       "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", // wstETH
			tokenOutAmount: big.NewInt(1e18),
			expectedReserves: []*big.Int{
				big.NewInt(3952511634683290746), // waEthLidoWETH + in/ERC4626rate
				big.NewInt(3003370557302528821), // waEthLidowstETH reserve - out/ERC4626rate
			},
			aggregateFee: big.NewInt(0), // Zero fee for test examples
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// Clone the pool simulator for each test to avoid state pollution
			clonedPool := testPool.CloneState()

			// Create token amounts
			tokenAmountIn := poolpkg.TokenAmount{
				Token:  tc.tokenIn,
				Amount: tc.tokenInAmount,
			}

			tokenAmountOut := poolpkg.TokenAmount{
				Token:  tc.tokenOut,
				Amount: tc.tokenOutAmount,
			}

			// Create swap info
			swapInfo := shared.SwapInfo{
				Buffers:      []*shared.ExtraBuffer{},
				AggregateFee: tc.aggregateFee,
			}

			// Create empty fee as it doesn't matter
			emptyFee := poolpkg.TokenAmount{
				Token:  tc.tokenIn,
				Amount: big.NewInt(0),
			}

			// Create update balance params
			params := poolpkg.UpdateBalanceParams{
				TokenAmountIn:  tokenAmountIn,
				TokenAmountOut: tokenAmountOut,
				Fee:            emptyFee,
				SwapInfo:       swapInfo,
			}
			// Update balance
			clonedPool.UpdateBalance(params)

			// Get updated reserves
			updatedReserves := clonedPool.GetReserves()

			// Verify that all reserves match expected values
			assert.Equal(t, len(tc.expectedReserves), len(updatedReserves),
				"Expected reserves array length should match actual reserves")
			for i, expectedReserve := range tc.expectedReserves {
				assert.Equal(t, expectedReserve, updatedReserves[i], "Reserve %d should match expected value", i)
			}
		})
	}
}
