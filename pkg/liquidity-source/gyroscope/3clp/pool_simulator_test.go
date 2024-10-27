package gyro3clp

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestCalcAmoutOut(t *testing.T) {
	t.Run("1. should return correct result", func(t *testing.T) {
		// polygon, block 51380313

		// input
		poolStr := `{"address":"0x17f1ef81707811ea15d9ee7c741179bbe2a63887","exchange":"gyroscope-3clp","type":"gyroscope-3clp","timestamp":1703150040,"reserves":["23020440114","1126110825231923552925","19544825382"],"tokens":[{"address":"0x2791bca1f2de4661ed88a30c99a7a9449aa84174","weight":1,"swappable":true},{"address":"0x9c9e5fd8bbc25984b178fdce6117defa39d2db39","weight":1,"swappable":true},{"address":"0xc2132d05d31c914a87c6611c10748aeb04b58e8f","weight":1,"swappable":true}],"extra":"{\"poolTokenInfos\":[{\"cash\":\"0x55c200a32\",\"managed\":\"0x0\",\"lastChangeBlock\":51379111,\"assetManager\":\"0x0000000000000000000000000000000000000000\"},{\"cash\":\"0x3d0bed552856cc229d\",\"managed\":\"0x0\",\"lastChangeBlock\":51378988,\"assetManager\":\"0x0000000000000000000000000000000000000000\"},{\"cash\":\"0x48cf65e26\",\"managed\":\"0x0\",\"lastChangeBlock\":51379111,\"assetManager\":\"0x0000000000000000000000000000000000000000\"}],\"swapFeePercentage\":\"0x110d9316ec000\",\"paused\":false}","staticExtra":"{\"poolId\":\"0x17f1ef81707811ea15d9ee7c741179bbe2a63887000100000000000000000799\",\"poolType\":\"Gyro3\",\"poolTypeVersion\":0,\"scalingFactors\":[\"0xc9f2c9cd04674edea40000000\",\"0xde0b6b3a7640000\",\"0xc9f2c9cd04674edea40000000\"],\"root3Alpha\":\"0xddeeff45500c000\",\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}","blockNumber":51380313}`
		var pool entity.Pool
		err := sonic.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
			Amount: bignumber.NewBig("10440114"),
		}
		tokenOut := "0x9c9e5fd8bbc25984b178fdce6117defa39d2db39"

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		// expected
		expected := "10429133523081407408"

		// actual
		actual, err := s.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, actual.TokenAmountOut.Amount.String())
	})

	t.Run("2. should return correct result", func(t *testing.T) {
		// polygon, block 51380313

		// input
		poolStr := `{"address":"0x17f1ef81707811ea15d9ee7c741179bbe2a63887","exchange":"gyroscope-3clp","type":"gyroscope-3clp","timestamp":1703150040,"reserves":["23020440114","1126110825231923552925","19544825382"],"tokens":[{"address":"0x2791bca1f2de4661ed88a30c99a7a9449aa84174","weight":1,"swappable":true},{"address":"0x9c9e5fd8bbc25984b178fdce6117defa39d2db39","weight":1,"swappable":true},{"address":"0xc2132d05d31c914a87c6611c10748aeb04b58e8f","weight":1,"swappable":true}],"extra":"{\"poolTokenInfos\":[{\"cash\":\"0x55c200a32\",\"managed\":\"0x0\",\"lastChangeBlock\":51379111,\"assetManager\":\"0x0000000000000000000000000000000000000000\"},{\"cash\":\"0x3d0bed552856cc229d\",\"managed\":\"0x0\",\"lastChangeBlock\":51378988,\"assetManager\":\"0x0000000000000000000000000000000000000000\"},{\"cash\":\"0x48cf65e26\",\"managed\":\"0x0\",\"lastChangeBlock\":51379111,\"assetManager\":\"0x0000000000000000000000000000000000000000\"}],\"swapFeePercentage\":\"0x110d9316ec000\",\"paused\":false}","staticExtra":"{\"poolId\":\"0x17f1ef81707811ea15d9ee7c741179bbe2a63887000100000000000000000799\",\"poolType\":\"Gyro3\",\"poolTypeVersion\":0,\"scalingFactors\":[\"0xc9f2c9cd04674edea40000000\",\"0xde0b6b3a7640000\",\"0xc9f2c9cd04674edea40000000\"],\"root3Alpha\":\"0xddeeff45500c000\",\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}","blockNumber":51380313}`
		var pool entity.Pool
		err := sonic.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x9c9e5fd8bbc25984b178fdce6117defa39d2db39",
			Amount: bignumber.NewBig("6110825231923552925"),
		}
		tokenOut := "0xc2132d05d31c914a87c6611c10748aeb04b58e8f"

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		// expected
		expected := "6112855"

		// actual
		actual, err := s.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, actual.TokenAmountOut.Amount.String())
	})

	t.Run("3. should return correct result", func(t *testing.T) {
		// polygon, block 51380313

		// input
		poolStr := `{"address":"0x1a076c59321a38bf48431081e8fe3420de67de8f","exchange":"gyroscope-3clp","type":"gyroscope-3clp","timestamp":1703150040,"reserves":["36664","76675558717198560","36664888493720408"],"tokens":[{"address":"0x2791bca1f2de4661ed88a30c99a7a9449aa84174","weight":1,"swappable":true},{"address":"0x2e1ad108ff1d8c782fcbbb89aad783ac49586756","weight":1,"swappable":true},{"address":"0x8f3cf7ad23cd3cadbd9735aff958023239c6a063","weight":1,"swappable":true}],"extra":"{\"poolTokenInfos\":[{\"cash\":\"0x8f38\",\"managed\":\"0x0\",\"lastChangeBlock\":33051429,\"assetManager\":\"0x0000000000000000000000000000000000000000\"},{\"cash\":\"0x1106803b04b10e0\",\"managed\":\"0x0\",\"lastChangeBlock\":33051429,\"assetManager\":\"0x0000000000000000000000000000000000000000\"},{\"cash\":\"0x8242859665c358\",\"managed\":\"0x0\",\"lastChangeBlock\":33051429,\"assetManager\":\"0x0000000000000000000000000000000000000000\"}],\"swapFeePercentage\":\"0x110d9316ec000\",\"paused\":false}","staticExtra":"{\"poolId\":\"0x1a076c59321a38bf48431081e8fe3420de67de8f000100000000000000000771\",\"poolType\":\"Gyro3\",\"poolTypeVersion\":0,\"scalingFactors\":[\"0xc9f2c9cd04674edea40000000\",\"0xde0b6b3a7640000\",\"0xde0b6b3a7640000\"],\"root3Alpha\":\"0xddeeff45500c000\",\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}","blockNumber":51380313}`
		var pool entity.Pool
		err := sonic.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x2e1ad108ff1d8c782fcbbb89aad783ac49586756",
			Amount: bignumber.NewBig("6675558717198560"),
		}
		tokenOut := "0x8f3cf7ad23cd3cadbd9735aff958023239c6a063"

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		// {
		// 	"kind": 0,
		// 	"tokenIn": "0x2e1ad108ff1d8c782fcbbb89aad783ac49586756",
		// 	"tokenOut": "0x8f3cf7ad23cd3cadbd9735aff958023239c6a063",
		// 	"amount": "6675558717198560",
		// 	"poolId": "0x1a076c59321a38bf48431081e8fe3420de67de8f000100000000000000000771",
		// 	"lastChangeBlock": 0,
		// 	"from": "0xdac42eeb17758daa38caf9a3540c808247527ae3",
		// 	"to": "0xdac42eeb17758daa38caf9a3540c808247527ae3",
		// 	"userData": "0x"
		// }

		// expected
		expected := "6670441571785677"

		// actual
		actual, err := s.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, actual.TokenAmountOut.Amount.String())
	})

	t.Run("4. should return correct result", func(t *testing.T) {
		// polygon, block 51380313

		// input
		poolStr := `{"address":"0x1a076c59321a38bf48431081e8fe3420de67de8f","exchange":"gyroscope-3clp","type":"gyroscope-3clp","timestamp":1703150040,"reserves":["36664","76675558717198560","36664888493720408"],"tokens":[{"address":"0x2791bca1f2de4661ed88a30c99a7a9449aa84174","weight":1,"swappable":true},{"address":"0x2e1ad108ff1d8c782fcbbb89aad783ac49586756","weight":1,"swappable":true},{"address":"0x8f3cf7ad23cd3cadbd9735aff958023239c6a063","weight":1,"swappable":true}],"extra":"{\"poolTokenInfos\":[{\"cash\":\"0x8f38\",\"managed\":\"0x0\",\"lastChangeBlock\":33051429,\"assetManager\":\"0x0000000000000000000000000000000000000000\"},{\"cash\":\"0x1106803b04b10e0\",\"managed\":\"0x0\",\"lastChangeBlock\":33051429,\"assetManager\":\"0x0000000000000000000000000000000000000000\"},{\"cash\":\"0x8242859665c358\",\"managed\":\"0x0\",\"lastChangeBlock\":33051429,\"assetManager\":\"0x0000000000000000000000000000000000000000\"}],\"swapFeePercentage\":\"0x110d9316ec000\",\"paused\":false}","staticExtra":"{\"poolId\":\"0x1a076c59321a38bf48431081e8fe3420de67de8f000100000000000000000771\",\"poolType\":\"Gyro3\",\"poolTypeVersion\":0,\"scalingFactors\":[\"0xc9f2c9cd04674edea40000000\",\"0xde0b6b3a7640000\",\"0xde0b6b3a7640000\"],\"root3Alpha\":\"0xddeeff45500c000\",\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}","blockNumber":51380313}`
		var pool entity.Pool
		err := sonic.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x8f3cf7ad23cd3cadbd9735aff958023239c6a063",
			Amount: bignumber.NewBig("64888493720408"),
		}
		tokenOut := "0x2791bca1f2de4661ed88a30c99a7a9449aa84174"

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		// {
		// 	"kind": 0,
		// 	"tokenIn": "0x8f3cf7ad23cd3cadbd9735aff958023239c6a063",
		// 	"tokenOut": "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
		// 	"amount": "64888493720408",
		// 	"poolId": "0x1a076c59321a38bf48431081e8fe3420de67de8f000100000000000000000771",
		// 	"lastChangeBlock": 0,
		// 	"from": "0xdac42eeb17758daa38caf9a3540c808247527ae3",
		// 	"to": "0xdac42eeb17758daa38caf9a3540c808247527ae3",
		// 	"userData": "0x"
		// }

		// expected
		expected := "64"

		// actual
		actual, err := s.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, actual.TokenAmountOut.Amount.String())
	})

	t.Run("5. should return correct result", func(t *testing.T) {
		// polygon, block 51380313

		// input
		poolStr := `{"address":"0x1a076c59321a38bf48431081e8fe3420de67de8f","exchange":"gyroscope-3clp","type":"gyroscope-3clp","timestamp":1703150040,"reserves":["36664","76675558717198560","36664888493720408"],"tokens":[{"address":"0x2791bca1f2de4661ed88a30c99a7a9449aa84174","weight":1,"swappable":true},{"address":"0x2e1ad108ff1d8c782fcbbb89aad783ac49586756","weight":1,"swappable":true},{"address":"0x8f3cf7ad23cd3cadbd9735aff958023239c6a063","weight":1,"swappable":true}],"extra":"{\"poolTokenInfos\":[{\"cash\":\"0x8f38\",\"managed\":\"0x0\",\"lastChangeBlock\":33051429,\"assetManager\":\"0x0000000000000000000000000000000000000000\"},{\"cash\":\"0x1106803b04b10e0\",\"managed\":\"0x0\",\"lastChangeBlock\":33051429,\"assetManager\":\"0x0000000000000000000000000000000000000000\"},{\"cash\":\"0x8242859665c358\",\"managed\":\"0x0\",\"lastChangeBlock\":33051429,\"assetManager\":\"0x0000000000000000000000000000000000000000\"}],\"swapFeePercentage\":\"0x110d9316ec000\",\"paused\":false}","staticExtra":"{\"poolId\":\"0x1a076c59321a38bf48431081e8fe3420de67de8f000100000000000000000771\",\"poolType\":\"Gyro3\",\"poolTypeVersion\":0,\"scalingFactors\":[\"0xc9f2c9cd04674edea40000000\",\"0xde0b6b3a7640000\",\"0xde0b6b3a7640000\"],\"root3Alpha\":\"0xddeeff45500c000\",\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}","blockNumber":51380313}`
		var pool entity.Pool
		err := sonic.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x8f3cf7ad23cd3cadbd9735aff958023239c6a063",
			Amount: bignumber.NewBig("36664888493720408"),
		}
		tokenOut := "0x2791bca1f2de4661ed88a30c99a7a9449aa84174"

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		// expected
		expected := "36640"

		// actual
		actual, err := s.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, actual.TokenAmountOut.Amount.String())
	})

	t.Run("6. should return correct result", func(t *testing.T) {
		// polygon, block 51380313

		// input
		poolStr := `{"address":"0x1a076c59321a38bf48431081e8fe3420de67de8f","exchange":"gyroscope-3clp","type":"gyroscope-3clp","timestamp":1703150040,"reserves":["36664","76675558717198560","36664888493720408"],"tokens":[{"address":"0x2791bca1f2de4661ed88a30c99a7a9449aa84174","weight":1,"swappable":true},{"address":"0x2e1ad108ff1d8c782fcbbb89aad783ac49586756","weight":1,"swappable":true},{"address":"0x8f3cf7ad23cd3cadbd9735aff958023239c6a063","weight":1,"swappable":true}],"extra":"{\"poolTokenInfos\":[{\"cash\":\"0x8f38\",\"managed\":\"0x0\",\"lastChangeBlock\":33051429,\"assetManager\":\"0x0000000000000000000000000000000000000000\"},{\"cash\":\"0x1106803b04b10e0\",\"managed\":\"0x0\",\"lastChangeBlock\":33051429,\"assetManager\":\"0x0000000000000000000000000000000000000000\"},{\"cash\":\"0x8242859665c358\",\"managed\":\"0x0\",\"lastChangeBlock\":33051429,\"assetManager\":\"0x0000000000000000000000000000000000000000\"}],\"swapFeePercentage\":\"0x110d9316ec000\",\"paused\":false}","staticExtra":"{\"poolId\":\"0x1a076c59321a38bf48431081e8fe3420de67de8f000100000000000000000771\",\"poolType\":\"Gyro3\",\"poolTypeVersion\":0,\"scalingFactors\":[\"0xc9f2c9cd04674edea40000000\",\"0xde0b6b3a7640000\",\"0xde0b6b3a7640000\"],\"root3Alpha\":\"0xddeeff45500c000\",\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}","blockNumber":51380313}`
		var pool entity.Pool
		err := sonic.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x8f3cf7ad23cd3cadbd9735aff958023239c6a063",
			Amount: bignumber.NewBig("1664888493720408"),
		}
		tokenOut := "0x2e1ad108ff1d8c782fcbbb89aad783ac49586756"

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		// expected
		expected := "1665027336743831"

		// actual
		actual, err := s.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, actual.TokenAmountOut.Amount.String())
	})
}
