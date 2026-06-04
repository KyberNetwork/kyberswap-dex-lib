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

// USDH(6)/WHYPE(18) pool — 0x305e5b1a81879aa0538338306cb9430a547e1eea @ block 36900535 on HyperEVM
// estimateSwap(WHYPE, USDH, 551123404101004495) on-chain = 40122543 (fee included)
// SpotPrice derived via calibration: estimateSwap(WHYPE, USDH, 1e17) = 7274640
// => SP = 7274640 * 1e18 / 1e17 = 72746400
var (
	entityUSDHWHYPE entity.Pool
	_               = json.Unmarshal([]byte(`{
		"address":"0x305e5b1a81879aa0538338306cb9430a547e1eea",
		"exchange":"liquidcore",
		"type":"liquidcore",
		"reserves":["28637107413","59376779733270420975"],
		"tokens":[
			{"address":"0x111111a1a0667d36bd57c0a9f569b98057111111","decimals":6,"swappable":true},
			{"address":"0x5555555555555555555555555555555555555555","decimals":18,"swappable":true}
		],
		"extra":"{\"s\":\"72746400\"}"
	}`), &entityUSDHWHYPE)
	poolSimUSDHWHYPE = lo.Must(NewPoolSimulator(entityUSDHWHYPE))
)

func TestPoolSimulator_CalcAmountOut_USDH_WHYPE(t *testing.T) {
	t.Parallel()
	// token[0] = USDH (6-dec), token[1] = WHYPE (18-dec)
	// getSpotPrices().forwardPrice at this block = 137534713 (1.89× too high — different oracle).
	// Calibrated SpotPrice = 72746400 gives ~0.1% match with on-chain 40122543.
	testutil.TestCalcAmountOut(t, poolSimUSDHWHYPE, map[int]map[int]map[string]string{
		1: {
			0: {
				// 0.551 WHYPE → USDH; on-chain estimateSwap = 40122543
				"551123404101004495": "40082220",
				"0":                  "",
			},
		},
		0: {
			1: {
				// ~1 USDH → WHYPE
				"1000000": "13612632377684669",
			},
		},
	})
}

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
