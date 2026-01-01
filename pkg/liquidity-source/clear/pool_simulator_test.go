package clear

import (
	"fmt"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func getPoolSim(reserveIn, reserveOut string) *PoolSimulator {
	extraStr := `{"address":"0xcac0fa2818aed2eea8b9f52ca411e6ec3e13d822","exchange":"clear","type":"clear","timestamp":1766474455,"reserves":["100000000000000000000000000","100000000000000000000000000"],"tokens":[{"address":"0x75faf114eafb1bdbe2f0316df893fd58ce46aa4d","symbol":"USDC","decimals":6,"swappable":true},{"address":"0x69cac783c212bfae06e3c1a9a2e6ae6b17ba0614","symbol":"GHO","decimals":18,"swappable":true}],"extra":"{\"swapAddress\":\"0xeb5ad3d93e59efcbc6934cad2b48eb33baf29745\",\"ious\":[\"0x1267a63dc2d3af46b1333326f49b4d746374ac2e\",\"0x50ca266a50c6531dce25ee7da0dfb57a06bd864e\"],\"reserves\":{\"0\":{\"1\":{\"AmountIn\":null,\"AmountOut\":null}}}}"}`
	extraStr = strings.Replace(extraStr, `\"AmountIn\":null`, fmt.Sprintf(`\"AmountIn\":%v`, reserveIn), 1)
	extraStr = strings.Replace(extraStr, `\"AmountOut\":null`, fmt.Sprintf(`\"AmountOut\":%v`, reserveOut), 1)
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
		{name: "USDC -> GHO", indexIn: 0, indexOut: 1, amountIn: "1000", amountOut: "2000", poolSim: getPoolSim("1000000", "2000000")},
		{name: "GHO -> USDC", indexIn: 1, indexOut: 0, amountIn: "2000", amountOut: "1000", poolSim: getPoolSim("1000000", "2000000")},
		{name: "USDC -> GHO", indexIn: 0, indexOut: 1, amountIn: "1000", amountOut: "0", poolSim: getPoolSim("1000000", "null")},
		{name: "GHO -> USDC", indexIn: 1, indexOut: 0, amountIn: "2000", amountOut: "0", poolSim: getPoolSim("1000000", "null")},
		{name: "USDC -> GHO", indexIn: 0, indexOut: 1, amountIn: "1000", amountOut: "0", poolSim: getPoolSim("null", "2000000")},
		{name: "GHO -> USDC", indexIn: 1, indexOut: 0, amountIn: "2000", amountOut: "0", poolSim: getPoolSim("null", "1000000")},
	}
	for _, tc := range testCases {
		testutil.TestCalcAmountOut(t, tc.poolSim, map[int]map[int]map[string]string{
			tc.indexIn: {tc.indexOut: {tc.amountIn: tc.amountOut}},
		})
	}
}
