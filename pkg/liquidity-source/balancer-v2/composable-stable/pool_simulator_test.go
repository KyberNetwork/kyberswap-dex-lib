package composablestable

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestRegularSwap(t *testing.T) {
	t.Run("1. Should return OK", func(t *testing.T) {
		// data
		reserve0, _ := new(big.Int).SetString("2596148429267407814265248164610048", 10)
		reserve1, _ := new(big.Int).SetString("6999791779383984752", 10)
		reserve2, _ := new(big.Int).SetString("3000000000000000000", 10)

		pool := poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Tokens: []string{
					"0x00C2A4be503869Fa751c2DbcB7156cc970b5a8dA",
					"0xD4e7C1F3DA1144c9E2CfD1b015eDA7652b4a4399",
					"0xF71d0774B214c4cf51E33Eb3d30Ef98132e4DBaA",
				},
				Reserves: []*big.Int{
					reserve0,
					reserve1,
					reserve2,
				},
			},
		}

		regularSimulator := &regularSimulator{
			Pool:              pool,
			swapFeePercentage: uint256.NewInt(100000000000000),
			scalingFactors: []*uint256.Int{
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(1000052057863883934),
				uint256.NewInt(1000000000000000000),
			},

			bptIndex: 0,
			amp:      uint256.NewInt(1500000),
		}

		poolSimulator := &PoolSimulator{
			Pool:             pool,
			regularSimulator: regularSimulator,
		}

		// input
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xD4e7C1F3DA1144c9E2CfD1b015eDA7652b4a4399",
			Amount: big.NewInt(999791779383984752),
		}
		tokenOut := "0xF71d0774B214c4cf51E33Eb3d30Ef98132e4DBaA"

		// expected
		expectedAmountOut := "998507669837625986"

		// calculation
		result, err := poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("2. Should return OK", func(t *testing.T) {
		// data
		reserve0, _ := new(big.Int).SetString("2596148429267407814265248164610048", 10)
		reserve1, _ := new(big.Int).SetString("6999791779383984752", 10)
		reserve2, _ := new(big.Int).SetString("3000000000000000000", 10)

		pool := poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Tokens: []string{
					"0x00C2A4be503869Fa751c2DbcB7156cc970b5a8dA",
					"0xD4e7C1F3DA1144c9E2CfD1b015eDA7652b4a4399",
					"0xF71d0774B214c4cf51E33Eb3d30Ef98132e4DBaA",
				},
				Reserves: []*big.Int{
					reserve0,
					reserve1,
					reserve2,
				},
			},
		}

		regularSimulator := &regularSimulator{
			Pool:              pool,
			swapFeePercentage: uint256.NewInt(100000000000000),
			scalingFactors: []*uint256.Int{
				uint256.NewInt(10000000000),
				uint256.NewInt(10000520578),
				uint256.NewInt(10000000000),
			},

			bptIndex: 0,
			amp:      uint256.NewInt(1500000),
		}

		poolSimulator := &PoolSimulator{
			Pool:             pool,
			regularSimulator: regularSimulator,
		}

		// input
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xF71d0774B214c4cf51E33Eb3d30Ef98132e4DBaA",
			Amount: big.NewInt(23142175917219494),
		}
		tokenOut := "0xD4e7C1F3DA1144c9E2CfD1b015eDA7652b4a4399"

		// expected
		expectedAmountOut := "23155810259460675"

		// calculation
		result, err := poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})
}

func TestBptSwap(t *testing.T) {
	t.Run("1. Join swap pool type ver 1 should return OK", func(t *testing.T) {
		// data
		reserve0, _ := new(big.Int).SetString("414101427485347", 10)
		reserve1, _ := new(big.Int).SetString("2596148429267348622595662702661260", 10)
		reserve2, _ := new(big.Int).SetString("1170046233780600", 10)

		bptTotalSupply := uint256.MustFromDecimal("2596148429267348624180999930418421")

		pool := poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Address: "0x2Ba7Aa2213Fa2C909Cd9E46FeD5A0059542b36B0",
				Tokens: []string{
					"0x0000000000085d4780B73119b644AE5ecd22b376",
					"0x2Ba7Aa2213Fa2C909Cd9E46FeD5A0059542b36B0",
					"0xA13a9247ea42D743238089903570127DdA72fE44",
				},
				Reserves: []*big.Int{
					reserve0,
					reserve1,
					reserve2,
				},
			},
		}

		bptSimulator := &bptSimulator{
			poolTypeVer:    poolTypeVer1,
			bptIndex:       1,
			bptTotalSupply: bptTotalSupply,
			amp:            uint256.NewInt(600000),
			scalingFactors: []*uint256.Int{
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(366332019912307),
			},
			lastJoinExit: LastJoinExitData{
				LastJoinExitAmplification: uint256.NewInt(600000),
				LastPostJoinExitInvariant: uint256.MustFromDecimal("114012967613307699384"),
			},
			rateProviders: []string{
				"0x0000000000000000000000000000000000000000",
				"0x0000000000000000000000000000000000000000",
				"0xA13a9247ea42D743238089903570127DdA72fE44",
			},
			tokenRateCaches: []TokenRateCache{
				{},
				{},
				{
					Rate:     uint256.MustFromDecimal("1003857034775170156"),
					OldRate:  uint256.MustFromDecimal("1000977462514719154"),
					Duration: uint256.NewInt(1000),
					Expires:  uint256.NewInt(1677904371),
				},
			},
			swapFeePercentage: uint256.NewInt(100000000000000),
			protocolFeePercentageCache: map[int]*uint256.Int{
				feeTypeSwap:  uint256.NewInt(0),
				feeTypeYield: uint256.NewInt(0),
			},
			tokenExemptFromYieldProtocolFee: []bool{
				false, false, true,
			},
			inRecoveryMode: true,
		}

		poolSimulator := &PoolSimulator{
			Pool:         pool,
			bptSimulator: bptSimulator,
		}

		// input
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xA13a9247ea42D743238089903570127DdA72fE44",
			Amount: big.NewInt(170046233780600),
		}
		tokenOut := "0x2Ba7Aa2213Fa2C909Cd9E46FeD5A0059542b36B0"

		// expected
		expectedAmountOut := "22005850083674"

		// calculation
		result, err := poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("2. Join swap pool type ver 1 should return OK", func(t *testing.T) {
		// data
		reserve0, _ := new(big.Int).SetString("1414101427485347", 10)
		reserve1, _ := new(big.Int).SetString("1596148429267348622595662702661260", 10)
		reserve2, _ := new(big.Int).SetString("2170046233780600", 10)

		bptTotalSupply := uint256.MustFromDecimal("2596148429267348624180999930418421")

		pool := poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Address: "0x2Ba7Aa2213Fa2C909Cd9E46FeD5A0059542b36B0",
				Tokens: []string{
					"0x0000000000085d4780B73119b644AE5ecd22b376",
					"0x2Ba7Aa2213Fa2C909Cd9E46FeD5A0059542b36B0",
					"0xA13a9247ea42D743238089903570127DdA72fE44",
				},
				Reserves: []*big.Int{
					reserve0,
					reserve1,
					reserve2,
				},
			},
		}

		bptSimulator := &bptSimulator{
			poolTypeVer:    poolTypeVer1,
			bptIndex:       1,
			bptTotalSupply: bptTotalSupply,
			amp:            uint256.NewInt(600000),
			scalingFactors: []*uint256.Int{
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(366332019912307),
			},
			lastJoinExit: LastJoinExitData{
				LastJoinExitAmplification: uint256.NewInt(600000),
				LastPostJoinExitInvariant: uint256.MustFromDecimal("114012967613307699384"),
			},
			rateProviders: []string{
				"0x0000000000000000000000000000000000000000",
				"0x0000000000000000000000000000000000000000",
				"0xA13a9247ea42D743238089903570127DdA72fE44",
			},
			tokenRateCaches: []TokenRateCache{
				{},
				{},
				{
					Rate:     uint256.MustFromDecimal("1003857034775170156"),
					OldRate:  uint256.MustFromDecimal("1000977462514719154"),
					Duration: uint256.NewInt(1000),
					Expires:  uint256.NewInt(1677904371),
				},
			},
			swapFeePercentage: uint256.NewInt(100000000000000),
			protocolFeePercentageCache: map[int]*uint256.Int{
				feeTypeSwap:  uint256.NewInt(0),
				feeTypeYield: uint256.NewInt(0),
			},
			tokenExemptFromYieldProtocolFee: []bool{
				false, false, true,
			},
			inRecoveryMode: true,
		}

		poolSimulator := &PoolSimulator{
			Pool:         pool,
			bptSimulator: bptSimulator,
		}

		// input
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x0000000000085d4780B73119b644AE5ecd22b376",
			Amount: big.NewInt(214101427485347),
		}
		tokenOut := "0x2Ba7Aa2213Fa2C909Cd9E46FeD5A0059542b36B0"

		// expected
		expectedAmountOut := "128189688116719916203223884786015"

		// calculation
		result, err := poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("3. Join swap pool type ver 5 should return OK", func(t *testing.T) {
		// data
		reserve0, _ := new(big.Int).SetString("2596148429267353763156769271943231", 10)
		reserve1, _ := new(big.Int).SetString("20405000000000000000000", 10)
		reserve2, _ := new(big.Int).SetString("10406089385", 10)
		reserve3, _ := new(big.Int).SetString("20404838434804858833196", 10)

		bptTotalSupply := uint256.MustFromDecimal("2596148429318671447367809085209495")

		pool := poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Address: "0x01536b22ea06e4a315e3daaf05a12683ed4dc14c",
				Tokens: []string{
					"0x01536b22ea06e4a315e3daaf05a12683ed4dc14c",
					"0x571f54D23cDf2211C83E9A0CbD92AcA36c48Fa02",
					"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
					"0xaF4ce7CD4F8891ecf1799878c3e9A35b8BE57E09",
				},
				Reserves: []*big.Int{
					reserve0,
					reserve1,
					reserve2,
					reserve3,
				},
			},
		}

		bptSimulator := &bptSimulator{
			poolTypeVer:    poolTypeVer5,
			bptIndex:       0,
			bptTotalSupply: bptTotalSupply,
			amp:            uint256.NewInt(200000),
			scalingFactors: []*uint256.Int{
				uint256.MustFromDecimal("1000000000000000000"),
				uint256.MustFromDecimal("1000000000000000000"),
				uint256.MustFromDecimal("1000000000000000000000000000000"),
				uint256.MustFromDecimal("1008208139884891050"),
			},
			lastJoinExit: LastJoinExitData{
				LastJoinExitAmplification: uint256.NewInt(200000),
				LastPostJoinExitInvariant: uint256.MustFromDecimal("51369044740270984486699"),
			},
			rateProviders: []string{
				"0x0000000000000000000000000000000000000000",
				"0x0000000000000000000000000000000000000000",
				"0x0000000000000000000000000000000000000000",
				"0xd8689E8740C23d73136744817347fd6aC464E842",
			},
			tokenRateCaches: []TokenRateCache{
				{},
				{},
				{},
				{
					Rate:     uint256.MustFromDecimal("1008130755672919714"),
					OldRate:  uint256.MustFromDecimal("1008130755672919714"),
					Duration: uint256.NewInt(10800),
					Expires:  uint256.NewInt(1700764235),
				},
			},
			swapFeePercentage: uint256.NewInt(500000000000000),
			protocolFeePercentageCache: map[int]*uint256.Int{
				feeTypeSwap:  uint256.NewInt(500000000000000000),
				feeTypeYield: uint256.NewInt(500000000000000000),
			},
			tokenExemptFromYieldProtocolFee: []bool{
				false, false, false, false,
			},
			exemptFromYieldProtocolFee: false,
			inRecoveryMode:             false,
		}

		poolSimulator := &PoolSimulator{
			Pool:         pool,
			bptSimulator: bptSimulator,
		}

		// input
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
			Amount: big.NewInt(2040500000000000),
		}
		tokenOut := "0x01536b22ea06e4a315e3daaf05a12683ed4dc14c"

		// expected
		expectedAmountOut := "72153658150470669505066070"

		// calculation
		result, err := poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("4. Join swap pool type ver 5 should return OK", func(t *testing.T) {
		// data
		reserve0, _ := new(big.Int).SetString("2596148429267353763156769271943231", 10)
		reserve1, _ := new(big.Int).SetString("20405000000000000000000", 10)
		reserve2, _ := new(big.Int).SetString("10406089385", 10)
		reserve3, _ := new(big.Int).SetString("20404838434804858833196", 10)

		bptTotalSupply := uint256.MustFromDecimal("2596148429318671447367809085209495")

		pool := poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Address: "0x01536b22ea06e4a315e3daaf05a12683ed4dc14c",
				Tokens: []string{
					"0x01536b22ea06e4a315e3daaf05a12683ed4dc14c",
					"0x571f54D23cDf2211C83E9A0CbD92AcA36c48Fa02",
					"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
					"0xaF4ce7CD4F8891ecf1799878c3e9A35b8BE57E09",
				},
				Reserves: []*big.Int{
					reserve0,
					reserve1,
					reserve2,
					reserve3,
				},
			},
		}

		bptSimulator := &bptSimulator{
			poolTypeVer:    poolTypeVer5,
			bptIndex:       0,
			bptTotalSupply: bptTotalSupply,
			amp:            uint256.NewInt(200000),
			scalingFactors: []*uint256.Int{
				uint256.MustFromDecimal("1000000000000000000"),
				uint256.MustFromDecimal("1000000000000000000"),
				uint256.MustFromDecimal("1000000000000000000000000000000"),
				uint256.MustFromDecimal("1008208139884891050"),
			},
			lastJoinExit: LastJoinExitData{
				LastJoinExitAmplification: uint256.NewInt(200000),
				LastPostJoinExitInvariant: uint256.MustFromDecimal("51369044740270984486699"),
			},
			rateProviders: []string{
				"0x0000000000000000000000000000000000000000",
				"0x0000000000000000000000000000000000000000",
				"0x0000000000000000000000000000000000000000",
				"0xd8689E8740C23d73136744817347fd6aC464E842",
			},
			tokenRateCaches: []TokenRateCache{
				{},
				{},
				{},
				{
					Rate:     uint256.MustFromDecimal("1008130755672919714"),
					OldRate:  uint256.MustFromDecimal("1008130755672919714"),
					Duration: uint256.NewInt(10800),
					Expires:  uint256.NewInt(1700764235),
				},
			},
			swapFeePercentage: uint256.NewInt(500000000000000),
			protocolFeePercentageCache: map[int]*uint256.Int{
				feeTypeSwap:  uint256.NewInt(500000000000000000),
				feeTypeYield: uint256.NewInt(500000000000000000),
			},
			tokenExemptFromYieldProtocolFee: []bool{
				false, false, false, false,
			},
			exemptFromYieldProtocolFee: false,
			inRecoveryMode:             false,
		}

		poolSimulator := &PoolSimulator{
			Pool:         pool,
			bptSimulator: bptSimulator,
		}

		// input
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xaF4ce7CD4F8891ecf1799878c3e9A35b8BE57E09",
			Amount: big.NewInt(4048384348048588331),
		}
		tokenOut := "0x01536b22ea06e4a315e3daaf05a12683ed4dc14c"

		// expected
		expectedAmountOut := "4071333855617864209"

		// calculation
		result, err := poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("1. Exit swap pool type ver 1 should return OK", func(t *testing.T) {
		// data
		reserve0, _ := new(big.Int).SetString("414101427485347", 10)
		reserve1, _ := new(big.Int).SetString("2596148429267348622595662702661260", 10)
		reserve2, _ := new(big.Int).SetString("1170046233780600", 10)

		bptTotalSupply := uint256.MustFromDecimal("2596148429267348624180999930418421")

		pool := poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Address: "0x2Ba7Aa2213Fa2C909Cd9E46FeD5A0059542b36B0",
				Tokens: []string{
					"0x0000000000085d4780B73119b644AE5ecd22b376",
					"0x2Ba7Aa2213Fa2C909Cd9E46FeD5A0059542b36B0",
					"0xA13a9247ea42D743238089903570127DdA72fE44",
				},
				Reserves: []*big.Int{
					reserve0,
					reserve1,
					reserve2,
				},
			},
		}

		bptSimulator := &bptSimulator{
			poolTypeVer:    poolTypeVer1,
			bptIndex:       1,
			bptTotalSupply: bptTotalSupply,
			amp:            uint256.NewInt(600000),
			scalingFactors: []*uint256.Int{
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(366332019912307),
			},
			lastJoinExit: LastJoinExitData{
				LastJoinExitAmplification: uint256.NewInt(600000),
				LastPostJoinExitInvariant: uint256.MustFromDecimal("114012967613307699384"),
			},
			rateProviders: []string{
				"0x0000000000000000000000000000000000000000",
				"0x0000000000000000000000000000000000000000",
				"0xA13a9247ea42D743238089903570127DdA72fE44",
			},
			tokenRateCaches: []TokenRateCache{
				{},
				{},
				{
					Rate:     uint256.MustFromDecimal("1003857034775170156"),
					OldRate:  uint256.MustFromDecimal("1000977462514719154"),
					Duration: uint256.NewInt(1000),
					Expires:  uint256.NewInt(1677904371),
				},
			},
			swapFeePercentage: uint256.NewInt(100000000000000),
			protocolFeePercentageCache: map[int]*uint256.Int{
				feeTypeSwap:  uint256.NewInt(0),
				feeTypeYield: uint256.NewInt(0),
			},
			tokenExemptFromYieldProtocolFee: []bool{
				false, false, true,
			},
			inRecoveryMode: true,
		}

		poolSimulator := &PoolSimulator{
			Pool:         pool,
			bptSimulator: bptSimulator,
		}

		// input
		amountIn, _ := new(big.Int).SetString("95662702661260", 10)
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x2Ba7Aa2213Fa2C909Cd9E46FeD5A0059542b36B0",
			Amount: amountIn,
		}
		tokenOut := "0xA13a9247ea42D743238089903570127DdA72fE44"

		// expected
		expectedAmountOut := "473156052715491"

		// calculation
		result, err := poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("2. Exit swap pool type ver 1 should return OK", func(t *testing.T) {
		// data
		reserve0, _ := new(big.Int).SetString("414101427485347", 10)
		reserve1, _ := new(big.Int).SetString("2596148429267348622595662702661260", 10)
		reserve2, _ := new(big.Int).SetString("1170046233780600", 10)
		// 414101427485347,2596148429267348622595662702661260,1170046233780600

		bptTotalSupply := uint256.MustFromDecimal("2596148429267348624180999930418421")

		pool := poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Address: "0x2Ba7Aa2213Fa2C909Cd9E46FeD5A0059542b36B0",
				Tokens: []string{
					"0x0000000000085d4780B73119b644AE5ecd22b376",
					"0x2Ba7Aa2213Fa2C909Cd9E46FeD5A0059542b36B0",
					"0xA13a9247ea42D743238089903570127DdA72fE44",
				},
				Reserves: []*big.Int{
					reserve0,
					reserve1,
					reserve2,
				},
			},
		}

		bptSimulator := &bptSimulator{
			poolTypeVer:    poolTypeVer1,
			bptIndex:       1,
			bptTotalSupply: bptTotalSupply,
			amp:            uint256.NewInt(600000),
			scalingFactors: []*uint256.Int{
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(366332019912307),
			},
			lastJoinExit: LastJoinExitData{
				LastJoinExitAmplification: uint256.NewInt(600000),
				LastPostJoinExitInvariant: uint256.MustFromDecimal("114012967613307699384"),
			},
			rateProviders: []string{
				"0x0000000000000000000000000000000000000000",
				"0x0000000000000000000000000000000000000000",
				"0xA13a9247ea42D743238089903570127DdA72fE44",
			},
			tokenRateCaches: []TokenRateCache{
				{},
				{},
				{
					Rate:     uint256.MustFromDecimal("1003857034775170156"),
					OldRate:  uint256.MustFromDecimal("1000977462514719154"),
					Duration: uint256.NewInt(1000),
					Expires:  uint256.NewInt(1677904371),
				},
			},
			swapFeePercentage: uint256.NewInt(100000000000000),
			protocolFeePercentageCache: map[int]*uint256.Int{
				feeTypeSwap:  uint256.NewInt(0),
				feeTypeYield: uint256.NewInt(0),
			},
			tokenExemptFromYieldProtocolFee: []bool{
				false, false, true,
			},
			inRecoveryMode: true,
		}

		poolSimulator := &PoolSimulator{
			Pool:         pool,
			bptSimulator: bptSimulator,
		}

		// input
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x2Ba7Aa2213Fa2C909Cd9E46FeD5A0059542b36B0",
			Amount: big.NewInt(59566270266126),
		}
		tokenOut := "0x0000000000085d4780B73119b644AE5ecd22b376"

		// expected
		expectedAmountOut := "17329834826337"

		// calculation
		result, err := poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("3. Exit swap pool type ver 5 should return OK", func(t *testing.T) {
		// data
		reserve0, _ := new(big.Int).SetString("2596148429267353763156769271943231", 10)
		reserve1, _ := new(big.Int).SetString("20405000000000000000000", 10)
		reserve2, _ := new(big.Int).SetString("10406089385", 10)
		reserve3, _ := new(big.Int).SetString("20404838434804858833196", 10)

		bptTotalSupply := uint256.MustFromDecimal("2596148429318671447367809085209495")

		pool := poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Address: "0x01536b22ea06e4a315e3daaf05a12683ed4dc14c",
				Tokens: []string{
					"0x01536b22ea06e4a315e3daaf05a12683ed4dc14c",
					"0x571f54D23cDf2211C83E9A0CbD92AcA36c48Fa02",
					"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
					"0xaF4ce7CD4F8891ecf1799878c3e9A35b8BE57E09",
				},
				Reserves: []*big.Int{
					reserve0,
					reserve1,
					reserve2,
					reserve3,
				},
			},
		}

		bptSimulator := &bptSimulator{
			poolTypeVer:    poolTypeVer5,
			bptIndex:       0,
			bptTotalSupply: bptTotalSupply,
			amp:            uint256.NewInt(200000),
			scalingFactors: []*uint256.Int{
				uint256.MustFromDecimal("1000000000000000000"),
				uint256.MustFromDecimal("1000000000000000000"),
				uint256.MustFromDecimal("1000000000000000000000000000000"),
				uint256.MustFromDecimal("1008208139884891050"),
			},
			lastJoinExit: LastJoinExitData{
				LastJoinExitAmplification: uint256.NewInt(200000),
				LastPostJoinExitInvariant: uint256.MustFromDecimal("51369044740270984486699"),
			},
			rateProviders: []string{
				"0x0000000000000000000000000000000000000000",
				"0x0000000000000000000000000000000000000000",
				"0x0000000000000000000000000000000000000000",
				"0xd8689E8740C23d73136744817347fd6aC464E842",
			},
			tokenRateCaches: []TokenRateCache{
				{},
				{},
				{},
				{
					Rate:     uint256.MustFromDecimal("1008130755672919714"),
					OldRate:  uint256.MustFromDecimal("1008130755672919714"),
					Duration: uint256.NewInt(10800),
				},
			},
			swapFeePercentage: uint256.NewInt(500000000000000),
			protocolFeePercentageCache: map[int]*uint256.Int{
				feeTypeSwap:  uint256.NewInt(500000000000000000),
				feeTypeYield: uint256.NewInt(500000000000000000),
			},
			tokenExemptFromYieldProtocolFee: []bool{
				false, false, false, false,
			},
			exemptFromYieldProtocolFee: false,
			inRecoveryMode:             false,
		}

		poolSimulator := &PoolSimulator{
			Pool:         pool,
			bptSimulator: bptSimulator,
		}

		// input
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x01536b22ea06e4a315e3daaf05a12683ed4dc14c",
			Amount: big.NewInt(2040500000000000),
		}
		tokenOut := "0xaF4ce7CD4F8891ecf1799878c3e9A35b8BE57E09"

		// expected
		expectedAmountOut := "2027780845478092"

		// calculation
		result, err := poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("4. Exit swap pool type ver 5 swap should return OK", func(t *testing.T) {
		// data
		reserve0, _ := new(big.Int).SetString("2596148429267353763156769271943231", 10)
		reserve1, _ := new(big.Int).SetString("20405000000000000000000", 10)
		reserve2, _ := new(big.Int).SetString("10406089385", 10)
		reserve3, _ := new(big.Int).SetString("20404838434804858833196", 10)

		bptTotalSupply := uint256.MustFromDecimal("2596148429318671447367809085209495")

		pool := poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Address: "0x01536b22ea06e4a315e3daaf05a12683ed4dc14c",
				Tokens: []string{
					"0x01536b22ea06e4a315e3daaf05a12683ed4dc14c",
					"0x571f54D23cDf2211C83E9A0CbD92AcA36c48Fa02",
					"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
					"0xaF4ce7CD4F8891ecf1799878c3e9A35b8BE57E09",
				},
				Reserves: []*big.Int{
					reserve0,
					reserve1,
					reserve2,
					reserve3,
				},
			},
		}

		bptSimulator := &bptSimulator{
			poolTypeVer:    poolTypeVer5,
			bptIndex:       0,
			bptTotalSupply: bptTotalSupply,
			amp:            uint256.NewInt(200000),
			scalingFactors: []*uint256.Int{
				uint256.MustFromDecimal("1000000000000000000"),
				uint256.MustFromDecimal("1000000000000000000"),
				uint256.MustFromDecimal("1000000000000000000000000000000"),
				uint256.MustFromDecimal("1008208139884891050"),
			},
			lastJoinExit: LastJoinExitData{
				LastJoinExitAmplification: uint256.NewInt(200000),
				LastPostJoinExitInvariant: uint256.MustFromDecimal("51369044740270984486699"),
			},
			rateProviders: []string{
				"0x0000000000000000000000000000000000000000",
				"0x0000000000000000000000000000000000000000",
				"0x0000000000000000000000000000000000000000",
				"0xd8689E8740C23d73136744817347fd6aC464E842",
			},
			tokenRateCaches: []TokenRateCache{
				{},
				{},
				{},
				{
					Rate:     uint256.MustFromDecimal("1008130755672919714"),
					OldRate:  uint256.MustFromDecimal("1008130755672919714"),
					Duration: uint256.NewInt(10800),
					Expires:  uint256.NewInt(1700764235),
				},
			},
			swapFeePercentage: uint256.NewInt(500000000000000),
			protocolFeePercentageCache: map[int]*uint256.Int{
				feeTypeSwap:  uint256.NewInt(500000000000000000),
				feeTypeYield: uint256.NewInt(500000000000000000),
			},
			tokenExemptFromYieldProtocolFee: []bool{
				false, false, false, false,
			},
			exemptFromYieldProtocolFee: false,
			inRecoveryMode:             false,
		}

		poolSimulator := &PoolSimulator{
			Pool:         pool,
			bptSimulator: bptSimulator,
		}

		// input
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x01536b22ea06e4a315e3daaf05a12683ed4dc14c",
			Amount: big.NewInt(4048384348048588331),
		}
		tokenOut := "0xaF4ce7CD4F8891ecf1799878c3e9A35b8BE57E09"

		// expected
		expectedAmountOut := "4023147984636196801"

		// calculation
		result, err := poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})
}
