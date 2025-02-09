package skypsm

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	bignumber "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestPoolSimulator_getSwapQuote(t *testing.T) {

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
			p := &PoolSimulator{
				rate:           tt.rate,
				usdcPrecision:  bignumber.TenPowInt(6),
				usdsPrecision:  bignumber.TenPowInt(18),
				susdsPrecision: bignumber.TenPowInt(18),
			}
			gotExactIn, err := p.getSwapQuote(tt.inIdx, tt.outIdx, tt.amount, false)
			require.NoError(t, err)
			require.Equal(t, tt.wantExactIn, gotExactIn)

			gotExactOut, err := p.getSwapQuote(tt.outIdx, tt.inIdx, gotExactIn, true)
			require.NoError(t, err)
			require.Equal(t, tt.wantExactOut, gotExactOut)
		})
	}
}
