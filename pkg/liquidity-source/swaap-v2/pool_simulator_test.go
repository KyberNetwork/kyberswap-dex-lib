package swaapv2

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	t.Run("1. should return OK", func(t *testing.T) {
		p := `{
			"address": "swaap_v2_0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			"exchange": "swaap-v2",
			"type": "swaap-v2",
			"timestamp": 1709711042,
			"reserves": [
			  "952034231656045615",
			  "1259118739"
			],
			"tokens": [
			  {
				"address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"symbol": "WETH",
				"decimals": 18,
				"swappable": true
			  },
			  {
				"address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"symbol": "USDC",
				"decimals": 6,
				"swappable": true
			  }
			],
			"extra": "{\"baseToQuotePriceLevels\":[{\"price\":3766.8762085558155,\"level\":0},{\"price\":3766.8762085558155,\"level\":0.0022288821657478614},{\"price\":3766.8490507666365,\"level\":0.01114440978035138},{\"price\":3766.8012247130564,\"level\":0.02228881956070276},{\"price\":3766.6965307059054,\"level\":0.0557220489017569},{\"price\":3766.4841226647645,\"level\":0.1114440978035138},{\"price\":3766.214521902419,\"level\":0.16716615928592582},{\"price\":3765.938538280598,\"level\":0.2228881956070276},{\"price\":3765.6575174450813,\"level\":0.2786102487023362},{\"price\":3765.3730881244423,\"level\":0.33433231857185164}],\"quoteToBasePriceLevels\":[{\"price\":0.00026532281648942546,\"level\":0},{\"price\":0.00026532281648942546,\"level\":17.941056366984068},{\"price\":0.0002653199249494233,\"level\":89.70605550508886},{\"price\":0.00026531464759751224,\"level\":179.41409087714086},{\"price\":0.0002653025212490239,\"level\":448.55049801735805},{\"price\":0.0002652764691449185,\"level\":897.1552285484877},{\"price\":0.00026524099477687863,\"level\":1345.8200585281966},{\"price\":0.00026520285689812795,\"level\":1794.54920677605},{\"price\":0.0002651613332028819,\"level\":2243.348760148422},{\"price\":0.0002651189910337773,\"level\":2692.2201264482296},{\"price\":0.0002650756079489333,\"level\":3141.1646861140907},{\"price\":0.0002650316307629569,\"level\":3590.1837399765}],\"priceTolerance\":10}"
		  }`
		var pool entity.Pool
		err := json.Unmarshal([]byte(p), &pool)
		assert.Nil(t, err)

		// expected amount
		expectedAmountOut := "1257859620"

		// calculation
		simulator, err := NewPoolSimulator(pool)
		assert.Nil(t, err)
		amountIn, _ := new(big.Int).SetString("334332318571851640", 10)
		result, err := testutil.MustConcurrentSafe[*poolpkg.CalcAmountOutResult](t, func() (any, error) {
			return simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					Amount: amountIn,
				},
				TokenOut: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			})
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

}
