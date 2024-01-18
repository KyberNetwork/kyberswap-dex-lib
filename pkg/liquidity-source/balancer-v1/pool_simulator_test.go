package balancerv1

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	testCases := []struct {
		name              string
		poolSimulator     PoolSimulator
		tokenAmountIn     poolpkg.TokenAmount
		tokenOut          string
		expectedAmountOut *big.Int
		expectedError     error
	}{
		{
			name: "it should return correct amountOut",
			poolSimulator: PoolSimulator{
				records: map[string]Record{
					"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
						Bound:   true,
						Balance: number.NewUint256("181453339134494385762"),
						Denorm:  number.NewUint256("25000000000000000000"),
					},
					"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": {
						Bound:   true,
						Balance: number.NewUint256("982184296"),
						Denorm:  number.NewUint256("25000000000000000000"),
					},
				},
				publicSwap: true,
				swapFee:    number.NewUint256("4000000000000000"),
				totalAmountsIn: map[string]*uint256.Int{
					"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": uint256.NewInt(0),
					"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": uint256.NewInt(0),
				},
				maxTotalAmountsIn: map[string]*uint256.Int{
					"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
					"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
				},
			},
			tokenAmountIn:     poolpkg.TokenAmount{Token: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Amount: utils.NewBig("81275824825923290")},
			tokenOut:          "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
			expectedAmountOut: utils.NewBig("437981"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return tc.poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{TokenAmountIn: tc.tokenAmountIn, TokenOut: tc.tokenOut})
			})

			assert.ErrorIs(t, err, tc.expectedError)
			if tc.expectedAmountOut != nil {
				assert.Equal(t, 0, tc.expectedAmountOut.Cmp(result.TokenAmountOut.Amount))
			}
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	testCases := []struct {
		name               string
		poolSimulator      PoolSimulator
		params             poolpkg.UpdateBalanceParams
		expectedBalanceIn  *uint256.Int
		expectedBalanceOut *uint256.Int
	}{
		{
			name: "it should return correct amountOut",
			poolSimulator: PoolSimulator{
				records: map[string]Record{
					"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
						Bound:   true,
						Balance: number.NewUint256("181453339134494385762"),
						Denorm:  number.NewUint256("25000000000000000000"),
					},
					"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": {
						Bound:   true,
						Balance: number.NewUint256("982184296"),
						Denorm:  number.NewUint256("25000000000000000000"),
					},
				},
				publicSwap: true,
				swapFee:    number.NewUint256("4000000000000000"),
				totalAmountsIn: map[string]*uint256.Int{
					"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": uint256.NewInt(0),
					"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": uint256.NewInt(0),
				},
				maxTotalAmountsIn: map[string]*uint256.Int{
					"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
					"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
				},
			},
			params: poolpkg.UpdateBalanceParams{
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Amount: utils.NewBig("81275824825923290")},
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", Amount: utils.NewBig("437981")},
			},
			expectedBalanceIn:  number.NewUint256("181534614959320309052"),
			expectedBalanceOut: number.NewUint256("981746315"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.poolSimulator.UpdateBalance(tc.params)

			assert.Equal(t, 0, tc.expectedBalanceIn.Cmp(tc.poolSimulator.records[tc.params.TokenAmountIn.Token].Balance))
			assert.Equal(t, 0, tc.expectedBalanceOut.Cmp(tc.poolSimulator.records[tc.params.TokenAmountOut.Token].Balance))
		})
	}
}
