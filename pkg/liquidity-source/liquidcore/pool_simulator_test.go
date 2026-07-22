package liquidcore

import (
	"math"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// UBTC(8)/UETH(18) pool — synthetic ladder fixture (values are not on-chain data;
// this exercises the ladder-quoting wiring, not the pool's actual fee curve).
var (
	entityUBTCUETH entity.Pool
	_              = json.Unmarshal([]byte(`{
		"address":"0x437bccdb2875aace0f685fc7e730b0a758346e5e",
		"exchange":"liquidcore",
		"type":"liquidcore",
		"reserves":["79582613","8203190593497158815"],
		"tokens":[
			{"address":"0x9fdbda0a5e284c32744d2f17ee5c74b284993463","decimals":8,"swappable":true},
			{"address":"0xbe6727b535545c67d5caa73dea54865b92cf7907","decimals":18,"swappable":true}
		],
		"extra":"{\"l\":[[[1000000,365186231364257547],[10000000,3651862313642575470],[79582613,8203190593497158815]],[[100000000000000000,900000],[1000000000000000000,9000000],[8000000000000000000,72000000]]]}",
		"blockNumber":36634918
	}`), &entityUBTCUETH)
	poolSimUBTCUETH = lo.Must(NewPoolSimulatorWith(entityUBTCUETH, math.MaxInt64))
)

func TestPoolSimulator_CalcAmountOut_UBTCUETH(t *testing.T) {
	t.Parallel()
	// token[0] = UBTC (8-dec), token[1] = UETH (18-dec)
	testutil.TestCalcAmountOut(t, poolSimUBTCUETH, map[int]map[int]map[string]string{
		0: {
			1: {
				// exact first ladder entry
				"1000000": "365186231364257536",
				// between entries 0 and 1 → spline-interpolated
				"5000000": "1995840362197584128",
				// zero → error
				"0": "",
			},
		},
		1: {
			0: {
				// exact second ladder entry
				"1000000000000000000": "9000000",
				// exceeds max ladder entry → error
				"9000000000000000000": ladder.ErrAmountInTooLarge.Error(),
			},
		},
	})
}

func TestPoolSimulator_Staleness(t *testing.T) {
	t.Parallel()

	stale := entityUBTCUETH
	stale.Timestamp = 0
	if _, err := NewPoolSimulatorWith(stale, MaxAge); err == nil {
		t.Fatal("expected staleness error for zero timestamp")
	}
}

func TestPoolSimulator_CloneState(t *testing.T) {
	t.Parallel()
	testutil.TestCloneState(t, poolSimUBTCUETH, pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x9fdbda0a5e284c32744d2f17ee5c74b284993463",
			Amount: bignumber.NewBig10("1000000"),
		},
		TokenOut: "0xbe6727b535545c67d5caa73dea54865b92cf7907",
	}, nil)
}
