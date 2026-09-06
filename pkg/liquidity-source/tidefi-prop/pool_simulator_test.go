package tidefiprop

import (
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// Synthetic pool: both ladders lie on a straight line through the origin
// (dir0 rate=2, dir1 rate=0.5), so the spline's implicit (0,0) anchor makes
// every point on the curve -- not just the sampled ones -- an exact linear
// interpolation, letting expected outputs be hand-computed.
var (
	entityAB entity.Pool
	_        = json.Unmarshal([]byte(`{
		"address":"0xdef100000000000000000000000000000000000c",
		"exchange":"tidefi-prop",
		"type":"tidefi-prop",
		"reserves":["1000000","1000000"],
		"tokens":[
			{"address":"0xdef100000000000000000000000000000000000a","symbol":"A","decimals":18,"swappable":true},
			{"address":"0xdef100000000000000000000000000000000000b","symbol":"B","decimals":6,"swappable":true}
		],
		"extra":"{\"l\":[[[1000,2000],[2000,4000],[3000,6000]],[[20,10],[40,20],[60,30]]]}",
		"staticExtra":"{\"a\":\"0x71def100007a540305dd65d1034d10e809679fd5\"}",
		"blockNumber":100
	}`), &entityAB)
	poolSimAB *PoolSimulator
)

func init() {
	entityAB.Timestamp = time.Now().Unix() // keep MaxAge's staleness check happy
	poolSimAB = lo.Must(NewPoolSimulator(entityAB))
}

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSimAB, map[int]map[int]map[string]string{
		0: {
			1: {
				"500":  "1000", // below first ladder entry -> spline toward origin
				"1000": "2000", // exact first ladder entry
				"1500": "3000", // between entries -> spline-interpolated
				"3000": "6000", // exact last ladder entry
				"3001": ladder.ErrAmountInTooLarge.Error(),
				"0":    "",
			},
		},
		1: {
			0: {
				"10": "5",  // below first ladder entry -> spline toward origin
				"20": "10", // exact first ladder entry
				"30": "15", // between entries -> spline-interpolated
				"60": "30", // exact last ladder entry
				"61": ladder.ErrAmountInTooLarge.Error(),
				"0":  "",
			},
		},
	})
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	t.Parallel()
	// Two sequential B->A swaps on a fresh clone; consumedIn/consumedOut track
	// correctly since the curve is linear (no depletion within the range tested).
	testutil.TestCalcAmountOutWithUpdateBalance(t, poolSimAB, map[int]map[int][][][2]string{
		1: {
			0: {{{"20", "10"}, {"20", "10"}}},
		},
	})
}

func TestPoolSimulator_GetMetaInfo(t *testing.T) {
	t.Parallel()
	meta := poolSimAB.GetMetaInfo("", "")
	assert.Equal(t, pool.MetaInfo{
		ApprovalAddress: "0x71def100007a540305dd65d1034d10e809679fd5",
		BlockNumber:     100,
	}, meta)
}
