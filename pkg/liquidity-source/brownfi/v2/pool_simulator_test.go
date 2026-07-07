package brownfiv2

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0xdc46421b43688fddbb6030aae761385782e84905","exchange":"brownfi-v2","type":"brownfi-v2","timestamp":1999999999,"reserves":["4524638806931956971","19730271656"],"tokens":[{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true},{"address":"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"f\":100000,\"l\":46116860184273880,\"k\":\"92233720368547760\",\"p\":[\"80320902643980172383261\",\"18444967098852931174\"]}","staticExtra":"{\"f\":[\"0xff61491a931112ddf1bd8147cd1b641375f79f5825126d665480874634fd0ace\",\"0xeaa020c61cc479712813461ce153894a96a6c00b21ed0cfc2798d1f9a9e9c94a\"]}","blockNumber":34897138}`),
		&entityPool)
	poolSim = lo.Must(NewPoolSimulator(pool.FactoryParams{EntityPool: entityPool, ChainID: valueobject.ChainIDBase}))
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		0: {
			1: {
				"100000000000000000":  "435004387",
				"1000000000000000000": "4347217764",
				"3666666666666666666": "",
			},
		},
		1: {
			0: {
				"435004387":   "99789048796252611",
				"4347217764":  "996594619925937188",
				"16666666666": "",
			},
		},
	})
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountIn(t, poolSim)
}
