package caliberprop

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// WETH(18)/USDC(6) pool on Optimism — block 153403874
// factory: 0x60a8fA0eB9eDBF97a7487f7163C793768385Adc4
var (
	entityWETHUSDC entity.Pool
	_              = json.Unmarshal([]byte(`{
		"address":"0x06a7db8a412ec8d78af6c10931818307161bf54f1021bb453433a14deb138b98",
		"exchange":"caliber-prop",
		"type":"caliber-prop",
		"reserves":["1000000000020000","10000"],
		"tokens":[
			{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true},
			{"address":"0x0b2c639c533813f4aa9d7837caf62653d097ff85","symbol":"USDC","decimals":6,"swappable":true}
		],
		"extra":"{\"l\":[[[1000000000020,1495],[5000000000100,7480],[25000000000500,10000],[50000000001000,10000],[100000000002000,10000],[200000000004000,10000],[300000000006000,10000],[500000000010000,10000],[700000000014000,10000],[900000000018000,10000],[990000000019800,10000]],[[10,6654830265],[50,33274151195],[250,166370752653],[500,332076014037],[1000,664151994972],[2000,1328303857539],[3000,1992455587700],[5000,3320758650805],[7000,4649061184287],[9000,5977363188147],[9900,6575631302465]]]}",
		"staticExtra":"{\"a\":\"0x60a8fA0eB9eDBF97a7487f7163C793768385Adc4\"}",
		"blockNumber":153403874
	}`), &entityWETHUSDC)
	poolSimWETHUSDC = lo.Must(NewPoolSimulator(entityWETHUSDC))
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	// token[0] = WETH (18-dec), token[1] = USDC (6-dec)
	// Ladder[0] (WETH→USDC): first entry {in:1000000000020, out:1495}
	// Ladder[1] (USDC→WETH): first entry {in:10, out:6654830265}
	testutil.TestCalcAmountOut(t, poolSimWETHUSDC, map[int]map[int]map[string]string{
		0: {
			1: {
				// 0.0000005 WETH → USDC; below first ladder entry → spline toward origin
				"500000000010": "747",
				// exact first ladder entry
				"1000000000020": "1495",
				// between entries 0 and 1 → spline-interpolated
				"2000000000040": "2991",
				// exceeds max ladder entry → error
				"990000000019801": ladder.ErrAmountInTooLarge.Error(),
				// zero → error
				"0": "",
			},
		},
		1: {
			0: {
				// below first ladder entry → spline toward origin
				"5": "3327415132",
				// exact first ladder entry
				"10": "6654830265",
				// between entries 1 and 2 → spline-interpolated
				"100": "66548302300",
				// exact entry 4
				"1000": "664151994972",
				// exceeds max ladder entry → error
				"9901": ladder.ErrAmountInTooLarge.Error(),
				// zero → error
				"0": "",
			},
		},
	})
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	t.Parallel()
	// Two sequential USDC→WETH swaps on a fresh clone; second quote reflects consumed amounts.
	// swap1: 10 USDC → 6654830265 WETH-wei  (exact first ladder entry)
	// swap2: 10 USDC → 6654830261 WETH-wei  (totalIn=20 spline-interpolated minus swap1's output)
	testutil.TestCalcAmountOutWithUpdateBalance(t, poolSimWETHUSDC, map[int]map[int][][][2]string{
		1: {
			0: {
				{
					{"10", "6654830265"},
					{"10", "6654830261"},
				},
			},
		},
	})
}

func TestPoolSimulator_CloneState(t *testing.T) {
	t.Parallel()
	testutil.TestCloneState(t, poolSimWETHUSDC, pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x0b2c639c533813f4aa9d7837caf62653d097ff85",
			Amount: bignumber.NewBig10("100"),
		},
		TokenOut: "0x4200000000000000000000000000000000000006",
	}, nil)
}
