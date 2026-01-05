package clear

import (
	"fmt"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func getPoolSim(reserveIn01, reserveOut01, reserveIn10, reserveOut10 string) *PoolSimulator {
	extraStr := fmt.Sprintf(`{"address":"0x5cc8b3282dcc692532b857a68bc0fb07f45fbade","exchange":"clear","type":"clear","timestamp":1767580596,"reserves":["100000000000000000000000000","100000000000000000000000000"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0x40d16fc0246ad3160ccc09b8d0d3a2cd28ae6c2f","symbol":"GHO","decimals":18,"swappable":true}],"extra":"{\"swapAddress\":\"0xeb5AD3D93E59eFcbC6934caD2B48EB33BAf29745\",\"ious\":[\"0x1267a63dc2d3af46b1333326f49b4d746374ac2e\",\"0x50ca266a50c6531dce25ee7da0dfb57a06bd864e\"],\"reserves\":{\"0\":{\"1\":{\"AmountIn\":%v,\"AmountOut\":%v}},\"1\":{\"0\":{\"AmountIn\":%v,\"AmountOut\":%v}}}}"}`, reserveIn01, reserveOut01, reserveIn10, reserveOut10)
	var entityPool entity.Pool
	_ = json.Unmarshal([]byte(extraStr),
		&entityPool)
	return lo.Must(NewPoolSimulator(entityPool))
}

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	testCases := []struct {
		name      string
		indexIn   int
		indexOut  int
		amountIn  string
		amountOut string
		poolSim   *PoolSimulator
	}{
		{name: "USDC -> GHO", indexIn: 0, indexOut: 1, amountIn: "1000", amountOut: "2000", poolSim: getPoolSim("1000000", "2000000", "2000000", "1000000")},
		{name: "GHO -> USDC", indexIn: 1, indexOut: 0, amountIn: "2000", amountOut: "1000", poolSim: getPoolSim("1000000", "2000000", "2000000", "1000000")},
		{name: "USDC -> GHO", indexIn: 0, indexOut: 1, amountIn: "1000", amountOut: "0", poolSim: getPoolSim("1000000", "null", "2000000", "1000000")},
		{name: "GHO -> USDC", indexIn: 1, indexOut: 0, amountIn: "2000", amountOut: "0", poolSim: getPoolSim("1000000", "2000000", "2000000", "null")},
		{name: "USDC -> GHO", indexIn: 0, indexOut: 1, amountIn: "1000", amountOut: "0", poolSim: getPoolSim("null", "2000000", "2000000", "1000000")},
		{name: "GHO -> USDC", indexIn: 1, indexOut: 0, amountIn: "2000", amountOut: "0", poolSim: getPoolSim("1000000", "2000000", "null", "1000000")},
	}
	for _, tc := range testCases {
		testutil.TestCalcAmountOut(t, tc.poolSim, map[int]map[int]map[string]string{
			tc.indexIn: {tc.indexOut: {tc.amountIn: tc.amountOut}},
		})
	}
}
