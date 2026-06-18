package uniswapv2

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	poolEncoded = `{"address":"0x9eb0bc7a207f77811ee365729d00152622a745b7","exchange":"pancake","type":"uniswap-v2","timestamp":1739501947,"reserves":["5789592094546501478373016","793623036600773033475"],"tokens":[{"address":"0x6d5ad1592ed9d6d1df9b93c793ab759573ed6714","name":"","symbol":"","decimals":0,"weight":0,"swappable":true},{"address":"0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c","name":"","symbol":"","decimals":0,"weight":0,"swappable":true}],"extra":"{\"fee\":25,\"feePrecision\":10000}"}`
	poolEntity  entity.Pool
	_           = lo.Must(0, json.Unmarshal([]byte(poolEncoded), &poolEntity))
	poolSim     = lo.Must(NewPoolSimulator(poolEntity))
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		poolSimulator     PoolSimulator
		tokenAmountIn     pool.TokenAmount
		tokenOut          string
		expectedAmountOut *big.Int
		expectedError     error
	}{
		{
			name: "[swap0to1] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
						Tokens: []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							"0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{bignumber.NewBig("10089138480746"), bignumber.NewBig("10066716097576")},
					},
				},
				reserves: []*uint256.Int{uint256.MustFromDecimal("10089138480746"),
					uint256.MustFromDecimal("10066716097576")},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
				taxHandler:   noopTokenTaxHandler{},
			},
			tokenAmountIn: pool.TokenAmount{
				Amount: bignumber.NewBig("125224746"),
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			tokenOut:          "0xdac17f958d2ee523a2206206994597c13d831ec7",
			expectedAmountOut: bignumber.NewBig("124570062"),
			expectedError:     nil,
		},
		{
			name: "[swap1to0] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x576cea6d4461fcb3a9d43e922c9b54c0f791599a",
						Tokens: []string{"0x32a7c02e79c4ea1008dd6564b35f131428673c41",
							"0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{bignumber.NewBig("70361282326226590645832"),
							bignumber.NewBig("54150601005")},
					},
				},
				reserves: []*uint256.Int{uint256.MustFromDecimal("70361282326226590645832"),
					uint256.MustFromDecimal("54150601005")},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
				taxHandler:   noopTokenTaxHandler{},
			},
			tokenAmountIn: pool.TokenAmount{
				Amount: bignumber.NewBig("124570062"),
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			},
			tokenOut:          "0x32a7c02e79c4ea1008dd6564b35f131428673c41",
			expectedAmountOut: bignumber.NewBig("161006857684289764421"),
			expectedError:     nil,
		},
		{
			// VIRTUAL => REPPO with 1% buy tax on REPPO
			// Verified against on-chain tx output: 31404648971357222354
			name: "[swap0to1] token-tax buy: VIRTUAL=>REPPO exact match",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x70000c1cb3ee34a7323211607ac3162665b49549",
						Tokens: []string{
							"0x0b3e328455c4059eeb9e3f84b5543f74e24e7e1b",
							"0xff8104251e7761163fac3211ef5583fb3f8583d6",
						},
						Reserves: []*big.Int{
							bignumber.NewBig("64759685176877841920"),
							bignumber.NewBig("2092201468546951388637"),
						},
					},
				},
				reserves: []*uint256.Int{
					uint256.MustFromDecimal("64759685176877841920"),
					uint256.MustFromDecimal("2092201468546951388637"),
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
				taxHandler: NewTaxHandler(
					"0xff8104251e7761163fac3211ef5583fb3f8583d6",
					uint256.NewInt(100),
					uint256.NewInt(100),
				),
			},
			tokenAmountIn: pool.TokenAmount{
				Amount: bignumber.NewBig("1000000000000000000"),
				Token:  "0x0b3e328455c4059eeb9e3f84b5543f74e24e7e1b",
			},
			tokenOut:          "0xff8104251e7761163fac3211ef5583fb3f8583d6",
			expectedAmountOut: bignumber.NewBig("31404648971357222354"),
			expectedError:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return tc.poolSimulator.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: tc.tokenAmountIn,
					TokenOut:      tc.tokenOut,
					Limit:         nil,
				})
			})

			if tc.expectedError != nil {
				assert.ErrorIs(t, tc.expectedError, err)
			} else {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
			}
		})
	}
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name             string
		poolSimulator    PoolSimulator
		tokenAmountOut   pool.TokenAmount
		tokenIn          string
		expectedAmountIn *big.Int
		expectedError    error
	}{
		{
			name: "[swap0to1] it should return correct amountIn and fee",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
						Tokens: []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							"0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{bignumber.NewBig("100000000"), bignumber.NewBig("100000000")},
					},
				},
				reserves: []*uint256.Int{uint256.MustFromDecimal("100000000"),
					uint256.MustFromDecimal("100000000")},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			tokenAmountOut: pool.TokenAmount{
				Amount: bignumber.NewBig("20000000"),
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			tokenIn:          "0xdac17f958d2ee523a2206206994597c13d831ec7",
			expectedAmountIn: bignumber.NewBig("25075226"),
			expectedError:    nil,
		},
		{
			name: "[swap1to0] it should return correct amountIn and fee",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x576cea6d4461fcb3a9d43e922c9b54c0f791599a",
						Tokens: []string{"0x32a7c02e79c4ea1008dd6564b35f131428673c41",
							"0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{bignumber.NewBig("100000000000000000000"), bignumber.NewBig("100000000")},
					},
				},
				reserves: []*uint256.Int{uint256.MustFromDecimal("100000000000000000000"),
					uint256.MustFromDecimal("100000000")},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			tokenAmountOut: pool.TokenAmount{
				Amount: bignumber.NewBig("20000000"),
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			},
			tokenIn:          "0x32a7c02e79c4ea1008dd6564b35f131428673c41",
			expectedAmountIn: bignumber.NewBig("25075225677031093280"),
			expectedError:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: tc.tokenAmountOut,
				TokenIn:        tc.tokenIn,
				Limit:          nil,
			})

			if tc.expectedError != nil {
				assert.ErrorIs(t, tc.expectedError, err)
			} else {
				assert.Equal(t, tc.expectedAmountIn, result.TokenAmountIn.Amount)
			}
		})
	}

	testutil.TestCalcAmountIn(t, poolSim)
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name             string
		poolSimulator    PoolSimulator
		params           pool.UpdateBalanceParams
		expectedReserves []*uint256.Int
	}{
		{
			name: "[swap0to1] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
						Tokens: []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							"0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{bignumber.NewBig("10089138480746"), bignumber.NewBig("10066716097576")},
					},
				},
				reserves: []*uint256.Int{uint256.MustFromDecimal("10089138480746"),
					uint256.MustFromDecimal("10066716097576")},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			params: pool.UpdateBalanceParams{
				TokenAmountIn: pool.TokenAmount{Token: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					Amount: bignumber.NewBig("125224746")},
				TokenAmountOut: pool.TokenAmount{Token: "0xdac17f958d2ee523a2206206994597c13d831ec7",
					Amount: bignumber.NewBig("124570062")},
				Fee: pool.TokenAmount{Token: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					Amount: bignumber.NewBig("375674")},
			},
			expectedReserves: []*uint256.Int{uint256.MustFromDecimal("10089263705492"),
				uint256.MustFromDecimal("10066591527514")},
		},
		{
			name: "[swap1to0] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x576cea6d4461fcb3a9d43e922c9b54c0f791599a",
						Tokens: []string{"0x32a7c02e79c4ea1008dd6564b35f131428673c41",
							"0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{bignumber.NewBig("70361282326226590645832"),
							bignumber.NewBig("54150601005")},
					},
				},
				reserves: []*uint256.Int{uint256.MustFromDecimal("70361282326226590645832"),
					uint256.MustFromDecimal("54150601005")},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			params: pool.UpdateBalanceParams{
				TokenAmountIn: pool.TokenAmount{Token: "0xdac17f958d2ee523a2206206994597c13d831ec7",
					Amount: bignumber.NewBig("124570062")},
				TokenAmountOut: pool.TokenAmount{Token: "0x32a7c02e79c4ea1008dd6564b35f131428673c41",
					Amount: bignumber.NewBig("161006857684289764421")},
			},
			expectedReserves: []*uint256.Int{uint256.MustFromDecimal("70200275468542300881411"),
				uint256.MustFromDecimal("54275171067")},
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

func TestPercentToBps(t *testing.T) {
	t.Parallel()
	// four.meme rates are in percent; the tracker normalizes to bps (rate * 100).
	assert.Equal(t, uint256.NewInt(100), percentToBps(true, big.NewInt(1)))   // 1% -> 100bp
	assert.Equal(t, uint256.NewInt(1000), percentToBps(true, big.NewInt(10))) // 10% -> 1000bp
	assert.Nil(t, percentToBps(true, nil))
	assert.Nil(t, percentToBps(false, big.NewInt(5)))
}

// newTaxPoolSim builds the VIRTUAL/agent tax pool used to verify tax-aware reserve updates.
// token0 = VIRTUAL (no tax), token1 = agent token (0xff81..., buy=sell=100bp).
func newTaxPoolSim() *PoolSimulator {
	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address: "0x70000c1cb3ee34a7323211607ac3162665b49549",
			Tokens: []string{
				"0x0b3e328455c4059eeb9e3f84b5543f74e24e7e1b",
				"0xff8104251e7761163fac3211ef5583fb3f8583d6",
			},
		}},
		reserves: []*uint256.Int{
			uint256.MustFromDecimal("64759685176877841920"),
			uint256.MustFromDecimal("2092201468546951388637"),
		},
		fee:          number.NewUint256("3"),
		feePrecision: number.NewUint256("1000"),
		taxHandler: NewTaxHandler(
			"0xff8104251e7761163fac3211ef5583fb3f8583d6",
			uint256.NewInt(100),
			uint256.NewInt(100),
		),
	}
}

// TestPoolSimulator_UpdateBalance_Tax verifies that reserves move by the pair-side amounts:
// the pair receives effectiveAmountIn (after sell tax) and sends grossAmountOut (before buy tax).
func TestPoolSimulator_UpdateBalance_Tax(t *testing.T) {
	t.Parallel()

	const (
		virtual = "0x0b3e328455c4059eeb9e3f84b5543f74e24e7e1b"
		agent   = "0xff8104251e7761163fac3211ef5583fb3f8583d6"
	)

	t.Run("sell agent: reserve grows by effectiveAmountIn, not full amountIn", func(t *testing.T) {
		s := newTaxPoolSim()
		reserveAgent0 := new(uint256.Int).Set(s.reserves[1])
		reserveVirtual0 := new(uint256.Int).Set(s.reserves[0])

		amountIn := bignumber.NewBig("1000000000000000000") // 1e18 agent
		res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: agent, Amount: amountIn},
			TokenOut:      virtual,
		})
		assert.NoError(t, err)

		swapInfo := res.SwapInfo.(SwapInfo)
		// sell tax 100bp: effective = 1e18 - floor(1e18*100/10000) = 0.99e18
		assert.Equal(t, bignumber.NewBig("990000000000000000"), swapInfo.EffectiveAmountIn)

		s.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: agent, Amount: amountIn},
			TokenAmountOut: pool.TokenAmount{Token: virtual, Amount: res.TokenAmountOut.Amount},
			SwapInfo:       swapInfo,
		})

		// agent reserve += effectiveAmountIn (0.99e18), NOT full 1e18
		wantAgent := new(uint256.Int).Add(reserveAgent0, uint256.MustFromDecimal("990000000000000000"))
		assert.Equal(t, wantAgent, s.reserves[1])
		// virtual reserve -= grossAmountOut (virtual is untaxed, gross == net)
		wantVirtual := new(uint256.Int).Sub(reserveVirtual0, uint256.MustFromBig(swapInfo.GrossAmountOut))
		assert.Equal(t, wantVirtual, s.reserves[0])
	})

	t.Run("buy agent: reserve shrinks by grossAmountOut, not user net out", func(t *testing.T) {
		s := newTaxPoolSim()
		reserveAgent0 := new(uint256.Int).Set(s.reserves[1])
		reserveVirtual0 := new(uint256.Int).Set(s.reserves[0])

		amountIn := bignumber.NewBig("1000000000000000000") // 1e18 virtual
		res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: virtual, Amount: amountIn},
			TokenOut:      agent,
		})
		assert.NoError(t, err)

		swapInfo := res.SwapInfo.(SwapInfo)
		// virtual has no sell tax: effective == full amountIn
		assert.Equal(t, amountIn, swapInfo.EffectiveAmountIn)
		// buy tax 100bp makes user net < pair gross
		assert.True(t, swapInfo.GrossAmountOut.Cmp(res.TokenAmountOut.Amount) > 0)

		s.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: virtual, Amount: amountIn},
			TokenAmountOut: pool.TokenAmount{Token: agent, Amount: res.TokenAmountOut.Amount},
			SwapInfo:       swapInfo,
		})

		// virtual reserve += full amountIn
		wantVirtual := new(uint256.Int).Add(reserveVirtual0, uint256.MustFromBig(amountIn))
		assert.Equal(t, wantVirtual, s.reserves[0])
		// agent reserve -= grossAmountOut (pair sends gross, not the user's net amount)
		wantAgent := new(uint256.Int).Sub(reserveAgent0, uint256.MustFromBig(swapInfo.GrossAmountOut))
		assert.Equal(t, wantAgent, s.reserves[1])
	})
}

func TestPoolSimulator_getAmountOut(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		poolSimulator     PoolSimulator
		reserveIn         *uint256.Int
		reserveOut        *uint256.Int
		amountIn          *uint256.Int
		expectedAmountOut *uint256.Int
	}{
		{
			name:              "it should return correct amountOut",
			poolSimulator:     PoolSimulator{fee: uint256.NewInt(3), feePrecision: uint256.NewInt(1000)},
			reserveIn:         number.NewUint256("10089138480746"),
			reserveOut:        number.NewUint256("10066716097576"),
			amountIn:          number.NewUint256("125224746"),
			expectedAmountOut: number.NewUint256("124570062"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			amountOut := tc.poolSimulator.getAmountOut(tc.amountIn, tc.reserveIn, tc.reserveOut)

			assert.Equal(t, 0, tc.expectedAmountOut.Cmp(amountOut))
		})
	}
}

func TestPoolSimulator_getAmountIn(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name             string
		poolSimulator    PoolSimulator
		reserveIn        *uint256.Int
		reserveOut       *uint256.Int
		amountOut        *uint256.Int
		expectedAmountIn *uint256.Int
		expectedErr      error
	}{
		{
			name:             "it should return correct amountIn",
			poolSimulator:    PoolSimulator{fee: uint256.NewInt(3), feePrecision: uint256.NewInt(1000)},
			reserveIn:        number.NewUint256("100000000"),
			reserveOut:       number.NewUint256("100000000000000000000"),
			amountOut:        number.NewUint256("20000000000000000000"),
			expectedAmountIn: number.NewUint256("25075226"),
			expectedErr:      nil,
		},
		{
			name:             "it should return correct ErrDSMathSubUnderflow error",
			poolSimulator:    PoolSimulator{fee: uint256.NewInt(3), feePrecision: uint256.NewInt(1000)},
			reserveIn:        number.NewUint256("1160689189059097452"),
			reserveOut:       number.NewUint256("1161607"),
			amountOut:        number.NewUint256("500000000"),
			expectedAmountIn: nil,
			expectedErr:      ErrDSMathSubUnderflow,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			amountIn, err := tc.poolSimulator.getAmountIn(tc.amountOut, tc.reserveIn, tc.reserveOut)
			assert.ErrorIs(t, err, tc.expectedErr)

			if err == nil {
				fmt.Printf("amountIn: %s\n", amountIn.String())
				assert.Equal(t, 0, tc.expectedAmountIn.Cmp(amountIn))
			}
		})
	}
}

func BenchmarkPoolSimulatorCalcAmountOut(b *testing.B) {
	testCases := []struct {
		name              string
		poolSimulator     PoolSimulator
		tokenAmountIn     pool.TokenAmount
		tokenOut          string
		expectedAmountOut *big.Int
		expectedError     error
	}{
		{
			name: "[swap0to1] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
						Tokens: []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							"0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{bignumber.NewBig("10089138480746"), bignumber.NewBig("10066716097576")},
					},
				},
				reserves: []*uint256.Int{uint256.MustFromDecimal("10089138480746"),
					uint256.MustFromDecimal("10066716097576")},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			tokenAmountIn: pool.TokenAmount{
				Amount: bignumber.NewBig("125224746"),
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			tokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
		},
		{
			name: "[swap1to0] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x576cea6d4461fcb3a9d43e922c9b54c0f791599a",
						Tokens: []string{"0x32a7c02e79c4ea1008dd6564b35f131428673c41",
							"0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{bignumber.NewBig("70361282326226590645832"),
							bignumber.NewBig("54150601005")},
					},
				},
				reserves: []*uint256.Int{uint256.MustFromDecimal("70361282326226590645832"),
					uint256.MustFromDecimal("54150601005")},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			tokenAmountIn: pool.TokenAmount{
				Amount: bignumber.NewBig("124570062"),
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			},
			tokenOut: "0x32a7c02e79c4ea1008dd6564b35f131428673c41",
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = tc.poolSimulator.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: tc.tokenAmountIn,
					TokenOut:      tc.tokenOut,
					Limit:         nil,
				})
			}
		})
	}
}

func BenchmarkPoolSimulatorCalcAmountIn(b *testing.B) {
	testCases := []struct {
		name             string
		poolSimulator    PoolSimulator
		tokenAmountOut   pool.TokenAmount
		tokenIn          string
		expectedAmountIn *big.Int
		expectedError    error
	}{
		{
			name: "[swap0to1] it should return correct amountIn and fee",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
						Tokens: []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							"0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{bignumber.NewBig("100000000"), bignumber.NewBig("100000000")},
					},
				},
				reserves: []*uint256.Int{uint256.MustFromDecimal("100000000"),
					uint256.MustFromDecimal("100000000")},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			tokenAmountOut: pool.TokenAmount{
				Amount: bignumber.NewBig("20000000"),
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			tokenIn:          "0xdac17f958d2ee523a2206206994597c13d831ec7",
			expectedAmountIn: bignumber.NewBig("25075226"),
			expectedError:    nil,
		},
		{
			name: "[swap1to0] it should return correct amountIn and fee",
			poolSimulator: PoolSimulator{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address: "0x576cea6d4461fcb3a9d43e922c9b54c0f791599a",
						Tokens: []string{"0x32a7c02e79c4ea1008dd6564b35f131428673c41",
							"0xdac17f958d2ee523a2206206994597c13d831ec7"},
						Reserves: []*big.Int{bignumber.NewBig("100000000000000000000"), bignumber.NewBig("100000000")},
					},
				},
				reserves: []*uint256.Int{uint256.MustFromDecimal("100000000000000000000"),
					uint256.MustFromDecimal("100000000")},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			tokenAmountOut: pool.TokenAmount{
				Amount: bignumber.NewBig("20000000"),
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			},
			tokenIn:          "0x32a7c02e79c4ea1008dd6564b35f131428673c41",
			expectedAmountIn: bignumber.NewBig("25075225677031093280"),
			expectedError:    nil,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = tc.poolSimulator.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: tc.tokenAmountOut,
					TokenIn:        tc.tokenIn,
					Limit:          nil,
				})
			}
		})
	}
}
