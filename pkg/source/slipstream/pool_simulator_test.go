package slipstream

import (
	"fmt"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var poolEncoded = `{
	"address": "0xb2cc224c1c9fee385f8ad6a55b4d94e92359dc59",
	"type": "slipstream",
	"timestamp": 1715918379,
	"reserves": [
		"2529981429777486647345",
		"4401629428817"
	],
	"tokens": [
		{
			"address": "0x4200000000000000000000000000000000000006",
			"name": "Wrapped Ether",
			"symbol": "WETH",
			"decimals": 18,
			"weight": 50,
			"swappable": true
		},
		{
			"address": "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
			"name": "USDC COIN",
			"symbol": "USDC",
			"decimals": 6,
			"weight": 50,
			"swappable": true
		}
	],
	"extra": "{\"liquidity\":7823851209968416017,\"sqrtPriceX96\":4304245232774846370939900,\"tickSpacing\":100,\"swapFee\":500,\"tick\":-196420,\"ticks\":[{\"index\":-203400,\"liquidityGross\":3190712423798,\"liquidityNet\":3190712423798},{\"index\":-203200,\"liquidityGross\":78178939718633,\"liquidityNet\":78178939718633},{\"index\":-201800,\"liquidityGross\":218205686587828,\"liquidityNet\":218205686587828},{\"index\":-200300,\"liquidityGross\":2514874768320,\"liquidityNet\":2514874768320},{\"index\":-199800,\"liquidityGross\":75014706551868298,\"liquidityNet\":75014706551868298},{\"index\":-199300,\"liquidityGross\":1649333900545095,\"liquidityNet\":1649333900545095},{\"index\":-199200,\"liquidityGross\":16699503668466505,\"liquidityNet\":16699503668466505},{\"index\":-198900,\"liquidityGross\":28053223894752,\"liquidityNet\":28053223894752},{\"index\":-198800,\"liquidityGross\":1149550250911,\"liquidityNet\":1149550250911},{\"index\":-198700,\"liquidityGross\":35911567114723,\"liquidityNet\":35911567114723},{\"index\":-198500,\"liquidityGross\":4810052491996793,\"liquidityNet\":4810052491996793},{\"index\":-198300,\"liquidityGross\":672060806119311,\"liquidityNet\":672060806119311},{\"index\":-198200,\"liquidityGross\":437669009136984,\"liquidityNet\":437669009136984},{\"index\":-198100,\"liquidityGross\":14727117423543357,\"liquidityNet\":14727117423543357},{\"index\":-198000,\"liquidityGross\":18337352672121977,\"liquidityNet\":18337352672121977},{\"index\":-197900,\"liquidityGross\":8833729510077,\"liquidityNet\":8833729510077},{\"index\":-197800,\"liquidityGross\":142722218901389,\"liquidityNet\":142722218901389},{\"index\":-197700,\"liquidityGross\":8498442497986201,\"liquidityNet\":8498442497986201},{\"index\":-197600,\"liquidityGross\":856458627723686,\"liquidityNet\":856458627723686},{\"index\":-197500,\"liquidityGross\":9269131738855720,\"liquidityNet\":9269131738855720},{\"index\":-197400,\"liquidityGross\":4267787987528332,\"liquidityNet\":4267787987528332},{\"index\":-197300,\"liquidityGross\":74742221740139124,\"liquidityNet\":74742221740139124},{\"index\":-197200,\"liquidityGross\":7936334644293724,\"liquidityNet\":7936334644293724},{\"index\":-197100,\"liquidityGross\":62097218472666426,\"liquidityNet\":62097218472666426},{\"index\":-197000,\"liquidityGross\":8542830590488775,\"liquidityNet\":8542830590488775},{\"index\":-196900,\"liquidityGross\":198447629887158657,\"liquidityNet\":198447629887158657},{\"index\":-196800,\"liquidityGross\":280144551006240520,\"liquidityNet\":280144551006240520},{\"index\":-196700,\"liquidityGross\":1278383086579354279,\"liquidityNet\":1278383086579354279},{\"index\":-196600,\"liquidityGross\":1103002533476292905,\"liquidityNet\":1102817192737446157},{\"index\":-196500,\"liquidityGross\":4705597080418081125,\"liquidityNet\":4654981766431565665},{\"index\":-196400,\"liquidityGross\":2174195520347156223,\"liquidityNet\":-1973506925447369185},{\"index\":-196300,\"liquidityGross\":2245897667420149090,\"liquidityNet\":-1659624197602771262},{\"index\":-196200,\"liquidityGross\":1246190397416708821,\"liquidityNet\":-1222112791770852809},{\"index\":-196100,\"liquidityGross\":585687873880609294,\"liquidityNet\":-569485392680523056},{\"index\":-196000,\"liquidityGross\":618781679401812731,\"liquidityNet\":-611155052320388843},{\"index\":-195900,\"liquidityGross\":481200909173620288,\"liquidityNet\":-426095416895726476},{\"index\":-195800,\"liquidityGross\":206238698490121323,\"liquidityNet\":-205210123372171791},{\"index\":-195700,\"liquidityGross\":180575225126506059,\"liquidityNet\":-180575225126506059},{\"index\":-195600,\"liquidityGross\":127482690277903598,\"liquidityNet\":-127482690277903598},{\"index\":-195500,\"liquidityGross\":26815896016605958,\"liquidityNet\":-26815896016605958},{\"index\":-195400,\"liquidityGross\":272928351700893832,\"liquidityNet\":-272928351700893832},{\"index\":-195300,\"liquidityGross\":115444009921720993,\"liquidityNet\":-115444009921720993},{\"index\":-195200,\"liquidityGross\":41092165004248132,\"liquidityNet\":-41092165004248132},{\"index\":-195100,\"liquidityGross\":20494090564882001,\"liquidityNet\":-20494090564882001},{\"index\":-195000,\"liquidityGross\":19850995940207168,\"liquidityNet\":-19850995940207168},{\"index\":-194900,\"liquidityGross\":9052601983147930,\"liquidityNet\":-9052601983147930},{\"index\":-194800,\"liquidityGross\":6210320582514992,\"liquidityNet\":-6210320582514992},{\"index\":-194700,\"liquidityGross\":136592386237949930,\"liquidityNet\":-136592386237949930},{\"index\":-194600,\"liquidityGross\":45944486964521762,\"liquidityNet\":-45944486964521762},{\"index\":-194500,\"liquidityGross\":7867144272703054,\"liquidityNet\":-7867144272703054},{\"index\":-194400,\"liquidityGross\":5305216864594265,\"liquidityNet\":-5305216864594265},{\"index\":-194300,\"liquidityGross\":2025030910005005,\"liquidityNet\":-2025030910005005},{\"index\":-194200,\"liquidityGross\":3146572785780255,\"liquidityNet\":-3146572785780255},{\"index\":-194100,\"liquidityGross\":1333534503184912,\"liquidityNet\":-1333534503184912},{\"index\":-194000,\"liquidityGross\":82247890879057509,\"liquidityNet\":-82247890879057509},{\"index\":-193900,\"liquidityGross\":449188003045857,\"liquidityNet\":-449188003045857},{\"index\":-193800,\"liquidityGross\":601611066278822,\"liquidityNet\":-601611066278822},{\"index\":-193700,\"liquidityGross\":140069444543675,\"liquidityNet\":-140069444543675},{\"index\":-193600,\"liquidityGross\":203978857503808,\"liquidityNet\":-203978857503808},{\"index\":-193500,\"liquidityGross\":18202134918676141,\"liquidityNet\":-18202134918676141},{\"index\":-193400,\"liquidityGross\":8388624988922592,\"liquidityNet\":-8388624988922592},{\"index\":-193300,\"liquidityGross\":18868446056413659,\"liquidityNet\":-18868446056413659},{\"index\":-193100,\"liquidityGross\":95336059832085,\"liquidityNet\":-95336059832085},{\"index\":-192900,\"liquidityGross\":442015849160807,\"liquidityNet\":-442015849160807},{\"index\":-192700,\"liquidityGross\":717573780571005,\"liquidityNet\":-717573780571005},{\"index\":-192400,\"liquidityGross\":2512829621645,\"liquidityNet\":-2512829621645},{\"index\":-192200,\"liquidityGross\":597758179068121,\"liquidityNet\":-597758179068121},{\"index\":-192100,\"liquidityGross\":99859893925866,\"liquidityNet\":-99859893925866},{\"index\":-191800,\"liquidityGross\":2834163652632,\"liquidityNet\":-2834163652632},{\"index\":-191100,\"liquidityGross\":10134851180132,\"liquidityNet\":-10134851180132},{\"index\":-191000,\"liquidityGross\":55246002733956,\"liquidityNet\":-55246002733956},{\"index\":-190400,\"liquidityGross\":2983363688886,\"liquidityNet\":-2983363688886},{\"index\":-190100,\"liquidityGross\":3293855544875752,\"liquidityNet\":-3293855544875752},{\"index\":-189300,\"liquidityGross\":75344776066001,\"liquidityNet\":-75344776066001},{\"index\":-181500,\"liquidityGross\":3190712423798,\"liquidityNet\":-3190712423798}]}"
}`

func TestCalcAmountOut(t *testing.T) {
	type testcase struct {
		name              string
		tokenIn           string
		amountIn          string
		tokenOut          string
		expectedAmountOut string
	}
	testcases := []testcase{
		{
			name:              "swap WETH for USDC",
			tokenIn:           "0x4200000000000000000000000000000000000006",
			tokenOut:          "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
			amountIn:          "1000000000000000000",
			expectedAmountOut: "2949949837",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			poolEntity := new(entity.Pool)
			err := json.Unmarshal([]byte(poolEncoded), poolEntity)
			require.NoError(t, err)

			poolSim, err := NewPoolSimulator(*poolEntity, valueobject.ChainIDBase)
			require.NoError(t, err)

			result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  tc.tokenIn,
					Amount: bignumber.NewBig10(tc.amountIn),
				},
				TokenOut: tc.tokenOut,
			})

			assert.NoError(t, err)
			fmt.Println(result.TokenAmountOut.Amount)
			// assert.Equal(t, result.TokenAmountOut.Amount, bignumber.NewBig(tc.expectedAmountOut))
		})
	}
}
