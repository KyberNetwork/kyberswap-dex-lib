package brownfiv2

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0xdc46421b43688fddbb6030aae761385782e84905","exchange":"brownfi-v2","type":"brownfi-v2","timestamp":1753733991,"reserves":["2454798995925182774","7926570577"],"tokens":[{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true},{"address":"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"f\":20000,\"l\":92233720368547760,\"k\":\"18446744073709551\",\"p\":[\"69773451254923957152643\",\"18444159869332265644\"]}","blockNumber":33472321}`),
		&entityPool)
	poolSim = lo.Must(NewPoolSimulator(entityPool))
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		0: {
			1: {
				"100000000000000000":  "377911852",
				"1000000000000000000": "3777493528",
				"2000000000000000000": "",
			},
		},
		1: {
			0: {
				"377339531":   "99804016287357869",
				"3773395313":  "997719722947226546",
				"37733953137": "",
			},
		},
	})
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountIn(t, poolSim)
}
