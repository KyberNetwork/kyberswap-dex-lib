package liquidcore

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// UBTC(8)/UETH(18) pool — 0x437bccdb2875aace0f685fc7e730b0a758346e5e @ block 36634918 on HyperEVM
// estimateSwap(UBTC, UETH, 1_000_000) on-chain = 367393297419180338 (fee included)
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
		"extra":"{\"s\":\"272151553\"}"
	}`), &entityUBTCUETH)
	poolSimUBTCUETH = lo.Must(NewPoolSimulator(entityUBTCUETH))
)

func TestPoolSimulator_CalcAmountOut_UBTCUETH(t *testing.T) {
	t.Parallel()
	// token[0] = UBTC (8-dec), token[1] = UETH (18-dec)
	testutil.TestCalcAmountOut(t, poolSimUBTCUETH, map[int]map[int]map[string]string{
		0: {
			1: {
				// 0.01 UBTC → ~0.3652 UETH; on-chain estimateSwap = 367393297419180338
				"1000000": "365186231364257547",
				"0":       "",
			},
		},
		1: {
			0: {
				// ~1 UETH → UBTC
				"1000000000000000000": "2720835",
			},
		},
	})
}
