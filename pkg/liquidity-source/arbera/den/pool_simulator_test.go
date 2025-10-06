package arberaden

import (
	"encoding/json"
	"testing"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0x9d8890bb264de97bba37f4a512b2fc2fa08d06f0","exchange":"arbera-den","type":"arbera-den","timestamp":1759290395,"tokens":[{"address":"0x9d8890bb264de97bba37f4a512b2fc2fa08d06f0","symbol":"brNECT","decimals":18,"swappable":true},{"address":"0x1ce0a25d13ce4d52071ae7e02cf1f6606f4c79d3","symbol":"NECT","decimals":18,"swappable":true}],"extra":"{\"assets\":[{\"token\":\"0x1ce0a25d13ce4d52071ae7e02cf1f6606f4c79d3\",\"weighting\":\"1000000000000000000\",\"basePriceUSDX96\":\"0\",\"c1\":\"0x0000000000000000000000000000000000000000\",\"q1\":\"79228162514264337593543950336000000000000000000\"}],\"assetSupplies\":[\"177765524182485275394558\"],\"supply\":\"174697524527989303631620\",\"fee\":{\"bond\":\"20\",\"debond\":\"20\",\"burn\":\"7000\"}}"}`),
		&entityPool)
	poolSim = lo.Must(NewPoolSimulator(entityPool))
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		1: {
			0: {
				"1000000000000000000": "980775829738257783", // https://www.tdly.co/shared/simulation/55c5b682-43bb-4d91-8243-e37afe5f220c
			},
		},
		0: {
			1: {
				"1000000000000000000": "1015526657366552557", // https://www.tdly.co/shared/simulation/fa0f7349-e2e2-4095-ad56-89becb2241d2
			},
		},
	})
}

func TestPoolSimulator_CalcAmountOutWithUpdateBalance(t *testing.T) {
	// t.Parallel()
	testutil.TestCalcAmountOutWithUpdateBalance(t, poolSim, map[int]map[int][][][2]string{
		1: {
			0: [][][2]string{
				{{"1000000000000000000", "980775829738257783"}},
				{{"1000000000000000000", "980775829738257783"}, {"1000000000000000000", "980786856564798846"}},
			},
		},
		0: {
			1: [][][2]string{
				{{"1000000000000000000", "1015526657366552557"}},
				{{"1000000000000000000", "1015526657366552557"}, {"1000000000000000000", "1015526665504878830"}},
			},
		},
	})
}
