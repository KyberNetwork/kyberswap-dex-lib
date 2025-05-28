package skypsm

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	poolEncoded = `{
		"address": "0x1601843c5e9bc251a3272907010afa41fa18347e",
		"exchange": "sky-psm",
		"type": "sky-psm",
		"timestamp": 1739765780,
		"reserves": [
			"14236841448487",
			"28946856661441273511196026",
			"27759833974904041860803040"
		],
		"tokens": [
			{
				"address": "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
				"name": "USD Coin",
				"symbol": "USDC",
				"decimals": 6,
				"swappable": true
			},
			{
				"address": "0x820c137fa70c8691f0e44dc420a5e53c168921dc",
				"name": "USDS Stablecoin",
				"symbol": "USDS",
				"decimals": 18,
				"swappable": true
			},
			{
				"address": "0x5875eee11cf8398102fdad704c9e96607675467a",
				"name": "Savings USDS",
				"symbol": "sUSDS",
				"decimals": 18,
				"swappable": true
			}
		],
		"extra": "{\"rate\":\"1038105872293887335025106342\",\"blockTimestamp\":1739765785}",
		"staticExtra": "{\"rateProvider\":\"0x65d946e533748a998b1f0e430803e39a6388f7a1\"}"
	}`
	poolEntity entity.Pool
	_          = lo.Must(0, json.Unmarshal([]byte(poolEncoded), &poolEntity))
	poolSim    = lo.Must(NewPoolSimulator(poolEntity))
)

func TestPoolSimulator_getSwapQuote(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		inIdx  int
		outIdx int
		rate   *uint256.Int
		amount *uint256.Int

		wantExactIn  *uint256.Int
		wantExactOut *uint256.Int
	}{
		{
			name:  "USDC to USDS",
			inIdx: 0, outIdx: 1,
			amount:       big256.NewUint256("4160700"),
			wantExactIn:  big256.NewUint256("4160700000000000000"),
			wantExactOut: big256.NewUint256("4160700"),
		},
		{
			name:  "USDC to sUSDS",
			inIdx: 0, outIdx: 2,
			rate:         big256.NewUint256("1036903527276524877455228177"),
			amount:       big256.NewUint256("37250726683"),
			wantExactIn:  big256.NewUint256("35924968623000000000000"),
			wantExactOut: big256.NewUint256("37250726683"),
		},
		{
			name:  "USDS to USDC",
			inIdx: 1, outIdx: 0,
			amount:       big256.NewUint256("200000000000000000000000"),
			wantExactIn:  big256.NewUint256("200000000000"),
			wantExactOut: big256.NewUint256("200000000000000000000000"),
		},
		{
			name:  "USDS to sUSDS",
			inIdx: 1, outIdx: 2,
			rate:         big256.NewUint256("1036950199588485300229215700"),
			amount:       big256.NewUint256("2200000000000000000000"),
			wantExactIn:  big256.NewUint256("2121606226483270025236"),
			wantExactOut: big256.NewUint256("2200000000000000000000"),
		},
		{
			name:  "sUSDS to USDC",
			inIdx: 2, outIdx: 0,
			rate:         big256.NewUint256("1036953443174346868582323752"),
			amount:       big256.NewUint256("295290546526000000000000"),
			wantExactIn:  big256.NewUint256("306202548956"),
			wantExactOut: big256.NewUint256("295290546526000000000000"),
		},
		{
			name:  "sUSDS to USDS",
			inIdx: 2, outIdx: 1,
			rate:         big256.NewUint256("1036944771569535666613153580"),
			amount:       big256.NewUint256("50236236932312720146"),
			wantExactIn:  big256.NewUint256("52092203230290084781"),
			wantExactOut: big256.NewUint256("50236236932312720146"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldRate := poolSim.rate
			defer func() { poolSim.rate = oldRate }()
			poolSim.rate = tt.rate
			gotExactIn, err := poolSim.getSwapQuote(tt.inIdx, tt.outIdx, tt.amount, false)
			require.NoError(t, err)
			require.Equal(t, tt.wantExactIn, gotExactIn)

			gotExactOut, err := poolSim.getSwapQuote(tt.outIdx, tt.inIdx, gotExactIn, true)
			require.NoError(t, err)
			require.Equal(t, tt.wantExactOut, gotExactOut)
		})
	}
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountIn(t, poolSim)
}
