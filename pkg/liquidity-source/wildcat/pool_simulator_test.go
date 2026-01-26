package wildcat

import (
	"encoding/json"
	"testing"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0xcc8e05f2d5e28b70214d3d384bada2bd0ae99cda","exchange":"wildcat","type":"wildcat","timestamp":1767595770,"reserves":["30699993136664026623","31490510717"],"tokens":[{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true},{"address":"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"rates\":[18332721688954665668,70366686069],\"isNative\":[true,false],\"samples\":[[[1000000000000000,3154701],[10000000000000000,31547016],[100000000000000000,315470163],[1000000000000000000,3154701633],[10000000000000000000,0],[100000000000000000000,0],[1000000000000000000000,0]],[[1000,0],[10000,0],[100000,31698338375796],[1000000,316983383757966],[10000000,3169833837579669],[100000000,31698338375796699],[1000000000,316983383757966994]]]}"}`),
		&entityPool)
	poolSim = lo.Must(NewPoolSimulator(entityPool))
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		1: {
			0: {
				"3005810": "952791824733531",
			},
		},
		0: {
			1: {
				"1000000000000000": "3154701",
			},
		},
	})
}
