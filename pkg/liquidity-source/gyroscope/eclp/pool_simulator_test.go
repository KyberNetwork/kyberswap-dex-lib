package gyroeclp

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestCalcAmountOut(t *testing.T) {
	t.Run("1. should return correct result", func(t *testing.T) {
		// ethereum
		p := `{
			"address": "0xe0e8ac08de6708603cfd3d23b613d2f80e3b7afb",
			"exchange": "gyroscope-eclp",
			"type": "gyroscope-eclp",
			"timestamp": 1705566147,
			"reserves": [
			  "1432237821990898965",
			  "2685567802993977683"
			],
			"tokens": [
			  {
				"address": "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
				"weight": 1,
				"swappable": true
			  },
			  {
				"address": "0xf951e335afb289353dc249e82926178eac7ded78",
				"weight": 1,
				"swappable": true
			  }
			],
			"extra": "{\"paused\":false,\"swapFeePercentage\":\"0x5af3107a4000\",\"paramsAlpha\":\"999500249875062469\",\"paramsBeta\":\"1010101010101010101\",\"paramsC\":\"705688316491160463\",\"paramsS\":\"708522406115622955\",\"paramsLambda\":\"500000000000000000000\",\"tauAlphaX\":\"-74798712145497721414789338637153095764\",\"tauAlphaY\":\"66371324089360848501248857841320837382\",\"tauBetaX\":\"83383678297259876539161659077817401265\",\"tauBetaY\":\"55201106815163337488949515922664830840\",\"u\":\"79090559955836985090533912561030798620\",\"v\":\"60763830337203480831932978680724761109\",\"w\":\"-5585063777148738251815296884188865778\",\"z\":\"3975485570515915653508200992108476537\",\"dSq\":\"100000000000000000082596734413730639400\",\"tokenRates\":[\"0x1003dadd43ba4f85\",\"0xe8c3a22e66c5342\"]}",
			"staticExtra": "{\"poolId\":\"0xe0e8ac08de6708603cfd3d23b613d2f80e3b7afb00020000000000000000058a\",\"poolType\":\"GyroE\",\"poolTypeVersion\":2,\"tokenDecimals\":[18,18],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}",
			"blockNumber": 19032529
		  }`
		var pool entity.Pool
		err := json.Unmarshal([]byte(p), &pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "2475693422207386"

		// calculation
		simulator, err := NewPoolSimulator(pool)
		assert.Nil(t, err)
		amountIn, _ := new(big.Int).SetString("2237821990898965", 10)
		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
				Amount: amountIn,
			},
			TokenOut: "0xf951e335afb289353dc249e82926178eac7ded78",
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("2. should return correct result", func(t *testing.T) {
		// ethereum
		p := `{
			"address": "0xe0e8ac08de6708603cfd3d23b613d2f80e3b7afb",
			"exchange": "gyroscope-eclp",
			"type": "gyroscope-eclp",
			"timestamp": 1705566195,
			"reserves": [
			  "1432237821990898965",
			  "2685567802993977683"
			],
			"tokens": [
			  {
				"address": "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
				"weight": 1,
				"swappable": true
			  },
			  {
				"address": "0xf951e335afb289353dc249e82926178eac7ded78",
				"weight": 1,
				"swappable": true
			  }
			],
			"extra": "{\"paused\":false,\"swapFeePercentage\":\"0x5af3107a4000\",\"paramsAlpha\":\"999500249875062469\",\"paramsBeta\":\"1010101010101010101\",\"paramsC\":\"705688316491160463\",\"paramsS\":\"708522406115622955\",\"paramsLambda\":\"500000000000000000000\",\"tauAlphaX\":\"-74798712145497721414789338637153095764\",\"tauAlphaY\":\"66371324089360848501248857841320837382\",\"tauBetaX\":\"83383678297259876539161659077817401265\",\"tauBetaY\":\"55201106815163337488949515922664830840\",\"u\":\"79090559955836985090533912561030798620\",\"v\":\"60763830337203480831932978680724761109\",\"w\":\"-5585063777148738251815296884188865778\",\"z\":\"3975485570515915653508200992108476537\",\"dSq\":\"100000000000000000082596734413730639400\",\"tokenRates\":[\"0x1003dadd43ba4f85\",\"0xe8c3a22e66c5342\"]}",
			"staticExtra": "{\"poolId\":\"0xe0e8ac08de6708603cfd3d23b613d2f80e3b7afb00020000000000000000058a\",\"poolType\":\"GyroE\",\"poolTypeVersion\":2,\"tokenDecimals\":[18,18],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}",
			"blockNumber": 19032533
		  }`
		var pool entity.Pool
		err := json.Unmarshal([]byte(p), &pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "2475693422206503"

		// calculation
		simulator, err := NewPoolSimulator(pool)
		assert.Nil(t, err)
		amountIn, _ := new(big.Int).SetString("2237821990898165", 10)
		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
				Amount: amountIn,
			},
			TokenOut: "0xf951e335afb289353dc249e82926178eac7ded78",
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("3. should return correct result", func(t *testing.T) {
		// ethereum
		p := `{
			"address": "0x317994cba902be6633de043a6bf05f4f08f43702",
			"exchange": "gyroscope-eclp",
			"type": "gyroscope-eclp",
			"timestamp": 1705566219,
			"reserves": [
			  "10999999999569018079",
			  "461505253532688956"
			],
			"tokens": [
			  {
				"address": "0x5f98805a4e8be255a32880fdec7f6728c6568ba0",
				"weight": 1,
				"swappable": true
			  },
			  {
				"address": "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
				"weight": 1,
				"swappable": true
			  }
			],
			"extra": "{\"paused\":false,\"swapFeePercentage\":\"0xb5e620f48000\",\"paramsAlpha\":\"990000000000000000\",\"paramsBeta\":\"1030000000000000000\",\"paramsC\":\"705341229421805917\",\"paramsS\":\"708867935568914946\",\"paramsLambda\":\"300000000000000000000\",\"tauAlphaX\":\"-91419190325524419061563454825339849034\",\"tauAlphaY\":\"40528158608867766625743607016709959954\",\"tauBetaX\":\"96509962886466721906116316748865363834\",\"tauBetaY\":\"26188300129119032331830137555207094394\",\"u\":\"93963407906892250949294611952201863577\",\"v\":\"33322469345794812145697093295638243947\",\"w\":\"-7169840062759158701297470491722166024\",\"z\":\"2076737940040009800988554195145513263\",\"dSq\":\"100000000000000000021740145339839380500\",\"tokenRates\":[\"0xde0b6b3a7640000\",\"0xde0b6b3a7640000\"]}",
			"staticExtra": "{\"poolId\":\"0x317994cba902be6633de043a6bf05f4f08f4370200020000000000000000060b\",\"poolType\":\"GyroE\",\"poolTypeVersion\":2,\"tokenDecimals\":[18,18],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}",
			"blockNumber": 19032535
		  }`
		var pool entity.Pool
		err := json.Unmarshal([]byte(p), &pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "463249688358555467"

		// calculation
		simulator, err := NewPoolSimulator(pool)
		assert.Nil(t, err)
		amountIn, _ := new(big.Int).SetString("461505253532188956", 10)
		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
				Amount: amountIn,
			},
			TokenOut: "0x5f98805a4e8be255a32880fdec7f6728c6568ba0",
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("4. should return correct result", func(t *testing.T) {
		p := `{
			"address": "0x317994cba902be6633de043a6bf05f4f08f43702",
			"exchange": "gyroscope-eclp",
			"type": "gyroscope-eclp",
			"timestamp": 1705566243,
			"reserves": [
			  "10999999999569018079",
			  "461505253532688956"
			],
			"tokens": [
			  {
				"address": "0x5f98805a4e8be255a32880fdec7f6728c6568ba0",
				"weight": 1,
				"swappable": true
			  },
			  {
				"address": "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
				"weight": 1,
				"swappable": true
			  }
			],
			"extra": "{\"paused\":false,\"swapFeePercentage\":\"0xb5e620f48000\",\"paramsAlpha\":\"990000000000000000\",\"paramsBeta\":\"1030000000000000000\",\"paramsC\":\"705341229421805917\",\"paramsS\":\"708867935568914946\",\"paramsLambda\":\"300000000000000000000\",\"tauAlphaX\":\"-91419190325524419061563454825339849034\",\"tauAlphaY\":\"40528158608867766625743607016709959954\",\"tauBetaX\":\"96509962886466721906116316748865363834\",\"tauBetaY\":\"26188300129119032331830137555207094394\",\"u\":\"93963407906892250949294611952201863577\",\"v\":\"33322469345794812145697093295638243947\",\"w\":\"-7169840062759158701297470491722166024\",\"z\":\"2076737940040009800988554195145513263\",\"dSq\":\"100000000000000000021740145339839380500\",\"tokenRates\":[\"0xde0b6b3a7640000\",\"0xde0b6b3a7640000\"]}",
			"staticExtra": "{\"poolId\":\"0x317994cba902be6633de043a6bf05f4f08f4370200020000000000000000060b\",\"poolType\":\"GyroE\",\"poolTypeVersion\":2,\"tokenDecimals\":[18,18],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}",
			"blockNumber": 19032537
		  }`
		var pool entity.Pool
		err := json.Unmarshal([]byte(p), &pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "198761416187986611"

		// calculation
		simulator, err := NewPoolSimulator(pool)
		assert.Nil(t, err)
		amountIn, _ := new(big.Int).SetString("199999999569018079", 10)
		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0x5f98805a4e8be255a32880fdec7f6728c6568ba0",
				Amount: amountIn,
			},
			TokenOut: "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("5. should return correct result", func(t *testing.T) {
		p := `{
			"address": "0x2191df821c198600499aa1f0031b1a7514d7a7d9",
			"exchange": "gyroscope-eclp",
			"type": "gyroscope-eclp",
			"timestamp": 1705568655,
			"reserves": [
			  "2225282408141747152381921",
			  "17675159398351800484082150"
			],
			"tokens": [
			  {
				"address": "0x83f20f44975d03b1b09e64809b757c47f942beea",
				"weight": 1,
				"swappable": true
			  },
			  {
				"address": "0xe07f9d810a48ab5c3c914ba3ca53af14e4491e8a",
				"weight": 1,
				"swappable": true
			  }
			],
			"extra": "{\"paused\":false,\"swapFeePercentage\":\"0x9184e72a000\",\"paramsAlpha\":\"998502246630054917\",\"paramsBeta\":\"1000200040008001600\",\"paramsC\":\"707106781186547524\",\"paramsS\":\"707106781186547524\",\"paramsLambda\":\"4000000000000000000000\",\"tauAlphaX\":\"-94861212813096057289512505574275160547\",\"tauAlphaY\":\"31644119574235279926451292677567331630\",\"tauBetaX\":\"37142269533113549537591131345643981951\",\"tauBetaY\":\"92846388265400743995957747409218517601\",\"u\":\"66001741173104803338721745994955553010\",\"v\":\"62245253919818011890633399060291020887\",\"w\":\"30601134345582732000058913853921008022\",\"z\":\"-28859471639991253843240999485797747790\",\"dSq\":\"99999999999999999886624093342106115200\",\"tokenRates\":[\"0xe992599f5322745\",\"0xde0b6b3a7640000\"]}",
			"staticExtra": "{\"poolId\":\"0x2191df821c198600499aa1f0031b1a7514d7a7d9000200000000000000000639\",\"poolType\":\"GyroE\",\"poolTypeVersion\":2,\"tokenDecimals\":[18,18],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}",
			"blockNumber": 19032737
		  }`
		var pool entity.Pool
		err := json.Unmarshal([]byte(p), &pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "23410401128383078500983"

		// calculation
		simulator, err := NewPoolSimulator(pool)
		assert.Nil(t, err)
		amountIn, _ := new(big.Int).SetString("22252824081417471523809", 10)
		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0x83f20f44975d03b1b09e64809b757c47f942beea",
				Amount: amountIn,
			},
			TokenOut: "0xe07f9d810a48ab5c3c914ba3ca53af14e4491e8a",
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("6. should return correct result", func(t *testing.T) {
		p := `{
			"address": "0x8d2ea84b1bb33d956096e5d4dccbc8e6dbe8dbbc",
			"exchange": "gyroscope-eclp",
			"type": "gyroscope-eclp",
			"timestamp": 1705571365,
			"reserves": [
			  "1999799893965774007",
			  "2000399787392128640"
			],
			"tokens": [
			  {
				"address": "0x15e86be6084c6a5a8c17732d398dfbc2ec574cec",
				"weight": 1,
				"swappable": true
			  },
			  {
				"address": "0x37b8e1152fb90a867f3dcca6e8d537681b04705e",
				"weight": 1,
				"swappable": true
			  }
			],
			"extra": "{\"paused\":false,\"swapFeePercentage\":\"0x5af3107a4000\",\"paramsAlpha\":\"999000999000999000\",\"paramsBeta\":\"1001001001001001001\",\"paramsC\":\"707106781186547524\",\"paramsS\":\"707106781186547524\",\"paramsLambda\":\"3500000000000000000000\",\"tauAlphaX\":\"-86813627444881502269005830178453760968\",\"tauAlphaY\":\"49632591004916489211256720438426352580\",\"tauBetaX\":\"86834999582601090763959787675832339395\",\"tauBetaY\":\"49595189761605594460186269922864521705\",\"u\":\"86824313513741296418044956281446571633\",\"v\":\"49613890383261041779471297130369607773\",\"w\":\"-18700621655447375514023258428389530\",\"z\":\"10686068859794247464863321233410621\",\"dSq\":\"99999999999999999886624093342106115200\",\"tokenRates\":[\"0xe0e6fbed38485a6\",\"0xde0b6b3a7640000\"]}",
			"staticExtra": "{\"poolId\":\"0x8d2ea84b1bb33d956096e5d4dccbc8e6dbe8dbbc000200000000000000000c78\",\"poolType\":\"GyroE\",\"poolTypeVersion\":2,\"tokenDecimals\":[18,18],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}",
			"blockNumber": 52464093
		  }`
		var pool entity.Pool
		err := json.Unmarshal([]byte(p), &pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "23410401128383078500983"

		// calculation
		simulator, err := NewPoolSimulator(pool)
		assert.Nil(t, err)
		amountIn, _ := new(big.Int).SetString("22252824081417471523809", 10)
		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0x83f20f44975d03b1b09e64809b757c47f942beea",
				Amount: amountIn,
			},
			TokenOut: "0xe07f9d810a48ab5c3c914ba3ca53af14e4491e8a",
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("7. should return correct result", func(t *testing.T) {
		p := `{
			"address": "0x97469e6236bd467cd147065f77752b00efadce8a",
			"exchange": "gyroscope-eclp",
			"type": "gyroscope-eclp",
			"timestamp": 1705572412,
			"reserves": [
			  "1892570",
			  "15002094566676268805213"
			],
			"tokens": [
			  {
				"address": "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
				"weight": 1,
				"swappable": true
			  },
			  {
				"address": "0x2e1ad108ff1d8c782fcbbb89aad783ac49586756",
				"weight": 1,
				"swappable": true
			  }
			],
			"extra": "{\"paused\":false,\"swapFeePercentage\":\"0xb5e620f48000\",\"paramsAlpha\":\"980000000000000000\",\"paramsBeta\":\"1020408163265306122\",\"paramsC\":\"707106781186547524\",\"paramsS\":\"707106781186547524\",\"paramsLambda\":\"2500000000000000000000\",\"tauAlphaX\":\"-99921684096872623630266893017017594088\",\"tauAlphaY\":\"3956898690236155895758568963473896725\",\"tauBetaX\":\"99921684096872623626859806443439155895\",\"tauBetaY\":\"3956898690236155981796108700303143085\",\"u\":\"99921684096872623515276234437562471024\",\"v\":\"3956898690236155934291169066298950059\",\"w\":\"43018769868414623130\",\"z\":\"-1703543286789219094\",\"dSq\":\"99999999999999999886624093342106115200\",\"tokenRates\":null}",
			"staticExtra": "{\"poolId\":\"0x97469e6236bd467cd147065f77752b00efadce8a0002000000000000000008c0\",\"poolType\":\"GyroE\",\"poolTypeVersion\":1,\"tokenDecimals\":[6,18],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}",
			"blockNumber": 52464697
		  }`
		var pool entity.Pool
		err := json.Unmarshal([]byte(p), &pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "602743318267653364"

		// calculation
		simulator, err := NewPoolSimulator(pool)
		assert.Nil(t, err)
		amountIn, _ := new(big.Int).SetString("592570", 10)
		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
				Amount: amountIn,
			},
			TokenOut: "0x2e1ad108ff1d8c782fcbbb89aad783ac49586756",
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("8. should return correct result", func(t *testing.T) {
		p := `{
			"address": "0x97469e6236bd467cd147065f77752b00efadce8a",
			"exchange": "gyroscope-eclp",
			"type": "gyroscope-eclp",
			"timestamp": 1705572412,
			"reserves": [
			  "1892570",
			  "15002094566676268805213"
			],
			"tokens": [
			  {
				"address": "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
				"weight": 1,
				"swappable": true
			  },
			  {
				"address": "0x2e1ad108ff1d8c782fcbbb89aad783ac49586756",
				"weight": 1,
				"swappable": true
			  }
			],
			"extra": "{\"paused\":false,\"swapFeePercentage\":\"0xb5e620f48000\",\"paramsAlpha\":\"980000000000000000\",\"paramsBeta\":\"1020408163265306122\",\"paramsC\":\"707106781186547524\",\"paramsS\":\"707106781186547524\",\"paramsLambda\":\"2500000000000000000000\",\"tauAlphaX\":\"-99921684096872623630266893017017594088\",\"tauAlphaY\":\"3956898690236155895758568963473896725\",\"tauBetaX\":\"99921684096872623626859806443439155895\",\"tauBetaY\":\"3956898690236155981796108700303143085\",\"u\":\"99921684096872623515276234437562471024\",\"v\":\"3956898690236155934291169066298950059\",\"w\":\"43018769868414623130\",\"z\":\"-1703543286789219094\",\"dSq\":\"99999999999999999886624093342106115200\",\"tokenRates\":null}",
			"staticExtra": "{\"poolId\":\"0x97469e6236bd467cd147065f77752b00efadce8a0002000000000000000008c0\",\"poolType\":\"GyroE\",\"poolTypeVersion\":1,\"tokenDecimals\":[6,18],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}",
			"blockNumber": 52464697
		  }`
		var pool entity.Pool
		err := json.Unmarshal([]byte(p), &pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "1472449"

		// calculation
		simulator, err := NewPoolSimulator(pool)
		assert.Nil(t, err)
		amountIn, _ := new(big.Int).SetString("1500209456667626880", 10)
		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0x2e1ad108ff1d8c782fcbbb89aad783ac49586756",
				Amount: amountIn,
			},
			TokenOut: "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})
}
