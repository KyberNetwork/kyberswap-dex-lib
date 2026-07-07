package gyro2clp

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	t.Run("1. should return correct result", func(t *testing.T) {
		// input
		poolStr := `{"address":"0xdac42eeb17758daa38caf9a3540c808247527ae3","exchange":"gyroscope-2clp","type":"gyroscope-2clp","timestamp":1702978154,"reserves":["41488841728","42841512988282624073636"],"tokens":[{"address":"0x2791bca1f2de4661ed88a30c99a7a9449aa84174","weight":1,"swappable":true},{"address":"0x8f3cf7ad23cd3cadbd9735aff958023239c6a063","weight":1,"swappable":true}],"extra":"{\"swapFeePercentage\":\"0xb5e620f48000\",\"paused\":false}","staticExtra":"{\"poolId\":\"0xdac42eeb17758daa38caf9a3540c808247527ae3000200000000000000000a2b\",\"poolType\":\"Gyro2\",\"poolTypeVersion\":0,\"scalingFactors\":[\"0xc9f2c9cd04674edea40000000\",\"0xde0b6b3a7640000\"],\"sqrtParameters\":[\"0xdd7d21d9fd0cd67\",\"0xde9959a7b067d3c\"],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}","blockNumber":51305088}`
		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		amountIn, _ := new(big.Int).SetString("1488841728", 10)
		params := poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
				Amount: amountIn,
			},
			TokenOut: "0x8f3cf7ad23cd3cadbd9735aff958023239c6a063",
		}

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "1488513423284045013413"

		// actual
		actual, err := s.CalcAmountOut(params)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expectedAmountOut, actual.TokenAmountOut.Amount.String())
	})

	t.Run("2. should return correct result", func(t *testing.T) {
		// input
		poolStr := `{"address":"0xdac42eeb17758daa38caf9a3540c808247527ae3","exchange":"gyroscope-2clp","type":"gyroscope-2clp","timestamp":1702978154,"reserves":["41488841728","42841512988282624073636"],"tokens":[{"address":"0x2791bca1f2de4661ed88a30c99a7a9449aa84174","weight":1,"swappable":true},{"address":"0x8f3cf7ad23cd3cadbd9735aff958023239c6a063","weight":1,"swappable":true}],"extra":"{\"swapFeePercentage\":\"0xb5e620f48000\",\"paused\":false}","staticExtra":"{\"poolId\":\"0xdac42eeb17758daa38caf9a3540c808247527ae3000200000000000000000a2b\",\"poolType\":\"Gyro2\",\"poolTypeVersion\":0,\"scalingFactors\":[\"0xc9f2c9cd04674edea40000000\",\"0xde0b6b3a7640000\"],\"sqrtParameters\":[\"0xdd7d21d9fd0cd67\",\"0xde9959a7b067d3c\"],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}","blockNumber":51305088}`
		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		amountIn, _ := new(big.Int).SetString("21841512988282624073636", 10)
		params := poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0x8f3cf7ad23cd3cadbd9735aff958023239c6a063",
				Amount: amountIn,
			},
			TokenOut: "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
		}

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "21807468855"

		// actual
		actual, err := s.CalcAmountOut(params)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expectedAmountOut, actual.TokenAmountOut.Amount.String())
	})

	t.Run("3. should return correct result", func(t *testing.T) {
		// input
		poolStr := `{"address":"0x918390ee7d83e79e3020a7f72df3f181cc9c029d","exchange":"gyroscope-2clp","type":"gyroscope-2clp","timestamp":1702978154,"reserves":["5001","4996253122268084"],"tokens":[{"address":"0x2791bca1f2de4661ed88a30c99a7a9449aa84174","weight":1,"swappable":true},{"address":"0x37b8e1152fb90a867f3dcca6e8d537681b04705e","weight":1,"swappable":true}],"extra":"{\"swapFeePercentage\":\"0xb5e620f48000\",\"paused\":false}","staticExtra":"{\"poolId\":\"0x918390ee7d83e79e3020a7f72df3f181cc9c029d000200000000000000000c0c\",\"poolType\":\"Gyro2\",\"poolTypeVersion\":0,\"scalingFactors\":[\"0xc9f2c9cd04674edea40000000\",\"0xde0b6b3a7640000\"],\"sqrtParameters\":[\"0xddef04b92227207\",\"0xde27dca5c29b233\"],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}","blockNumber":51305088}`
		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		amountIn, _ := new(big.Int).SetString("2001", 10)
		params := poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
				Amount: amountIn,
			},
			TokenOut: "0x37b8e1152fb90a867f3dcca6e8d537681b04705e",
		}

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "1999600120046494"

		// actual
		actual, err := s.CalcAmountOut(params)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expectedAmountOut, actual.TokenAmountOut.Amount.String())
	})

	t.Run("4. should return correct result", func(t *testing.T) {
		// input
		poolStr := `{"address":"0x918390ee7d83e79e3020a7f72df3f181cc9c029d","exchange":"gyroscope-2clp","type":"gyroscope-2clp","timestamp":1702978154,"reserves":["5001","4996253122268084"],"tokens":[{"address":"0x2791bca1f2de4661ed88a30c99a7a9449aa84174","weight":1,"swappable":true},{"address":"0x37b8e1152fb90a867f3dcca6e8d537681b04705e","weight":1,"swappable":true}],"extra":"{\"swapFeePercentage\":\"0xb5e620f48000\",\"paused\":false}","staticExtra":"{\"poolId\":\"0x918390ee7d83e79e3020a7f72df3f181cc9c029d000200000000000000000c0c\",\"poolType\":\"Gyro2\",\"poolTypeVersion\":0,\"scalingFactors\":[\"0xc9f2c9cd04674edea40000000\",\"0xde0b6b3a7640000\"],\"sqrtParameters\":[\"0xddef04b92227207\",\"0xde27dca5c29b233\"],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}","blockNumber":51305088}`
		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		amountIn, _ := new(big.Int).SetString("3996253122268084", 10)
		params := poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0x37b8e1152fb90a867f3dcca6e8d537681b04705e",
				Amount: amountIn,
			},
			TokenOut: "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
		}

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "3993"

		// actual
		actual, err := s.CalcAmountOut(params)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expectedAmountOut, actual.TokenAmountOut.Amount.String())
	})

	t.Run("5. should return correct result", func(t *testing.T) {
		// input
		poolStr := `{"address":"0x918390ee7d83e79e3020a7f72df3f181cc9c029d","exchange":"gyroscope-2clp","type":"gyroscope-2clp","timestamp":1702978154,"reserves":["5001","4996253122268084"],"tokens":[{"address":"0x2791bca1f2de4661ed88a30c99a7a9449aa84174","weight":1,"swappable":true},{"address":"0x37b8e1152fb90a867f3dcca6e8d537681b04705e","weight":1,"swappable":true}],"extra":"{\"swapFeePercentage\":\"0xb5e620f48000\",\"paused\":false}","staticExtra":"{\"poolId\":\"0x918390ee7d83e79e3020a7f72df3f181cc9c029d000200000000000000000c0c\",\"poolType\":\"Gyro2\",\"poolTypeVersion\":0,\"scalingFactors\":[\"0xc9f2c9cd04674edea40000000\",\"0xde0b6b3a7640000\"],\"sqrtParameters\":[\"0xddef04b92227207\",\"0xde27dca5c29b233\"],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}","blockNumber":51305088}`
		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		amountIn, _ := new(big.Int).SetString("39962531222", 10)
		params := poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0x37b8e1152fb90a867f3dcca6e8d537681b04705e",
				Amount: amountIn,
			},
			TokenOut: "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
		}

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "0"

		// actual
		actual, err := s.CalcAmountOut(params)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expectedAmountOut, actual.TokenAmountOut.Amount.String())
	})

	t.Run("6. should return correct result", func(t *testing.T) {
		// input
		poolStr := `{"address":"0x918390ee7d83e79e3020a7f72df3f181cc9c029d","exchange":"gyroscope-2clp","type":"gyroscope-2clp","timestamp":1702978154,"reserves":["5001","4996253122268084"],"tokens":[{"address":"0x2791bca1f2de4661ed88a30c99a7a9449aa84174","weight":1,"swappable":true},{"address":"0x37b8e1152fb90a867f3dcca6e8d537681b04705e","weight":1,"swappable":true}],"extra":"{\"swapFeePercentage\":\"0xb5e620f48000\",\"paused\":false}","staticExtra":"{\"poolId\":\"0x918390ee7d83e79e3020a7f72df3f181cc9c029d000200000000000000000c0c\",\"poolType\":\"Gyro2\",\"poolTypeVersion\":0,\"scalingFactors\":[\"0xc9f2c9cd04674edea40000000\",\"0xde0b6b3a7640000\"],\"sqrtParameters\":[\"0xddef04b92227207\",\"0xde27dca5c29b233\"],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}","blockNumber":51305088}`
		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		amountIn, _ := new(big.Int).SetString("28721432574291", 10)
		params := poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0x37b8e1152fb90a867f3dcca6e8d537681b04705e",
				Amount: amountIn,
			},
			TokenOut: "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
		}

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "28"

		// actual
		actual, err := s.CalcAmountOut(params)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expectedAmountOut, actual.TokenAmountOut.Amount.String())
	})

	t.Run("7. should return error", func(t *testing.T) {
		// input
		poolStr := `{"address":"0xdac42eeb17758daa38caf9a3540c808247527ae3","exchange":"gyroscope-2clp","type":"gyroscope-2clp","timestamp":1702978154,"reserves":["41488841728","42841512988282624073636"],"tokens":[{"address":"0x2791bca1f2de4661ed88a30c99a7a9449aa84174","weight":1,"swappable":true},{"address":"0x8f3cf7ad23cd3cadbd9735aff958023239c6a063","weight":1,"swappable":true}],"extra":"{\"swapFeePercentage\":\"0xb5e620f48000\",\"paused\":false}","staticExtra":"{\"poolId\":\"0xdac42eeb17758daa38caf9a3540c808247527ae3000200000000000000000a2b\",\"poolType\":\"Gyro2\",\"poolTypeVersion\":0,\"scalingFactors\":[\"0xc9f2c9cd04674edea40000000\",\"0xde0b6b3a7640000\"],\"sqrtParameters\":[\"0xdd7d21d9fd0cd67\",\"0xde9959a7b067d3c\"],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}","blockNumber":51305088}`
		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		amountIn, _ := new(big.Int).SetString("41841512988282624073636", 10)
		params := poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0x8f3cf7ad23cd3cadbd9735aff958023239c6a063",
				Amount: amountIn,
			},
			TokenOut: "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
		}

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		_, err = s.CalcAmountOut(params)
		assert.Equal(t, ErrAssetBoundsExceeded, err)
	})
}
