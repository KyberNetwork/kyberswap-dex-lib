package nabla

import (
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/stretchr/testify/require"
)

func monadSnapshotPools() (NablaPool, NablaPool) {
	wmonPool := NablaPool{
		Meta: NablaPoolMeta{
			CurveBeta:                 int256.MustFromDec("10000000000000000"),
			CurveC:                    int256.MustFromDec("16110498756211208902"),
			LpFee:                     int256.NewInt(200),
			BackstopFee:               int256.NewInt(1500),
			ProtocolFee:               int256.NewInt(300),
			MaxCoverageRatioForSwapIn: int256.NewInt(200),
		},
		State: NablaPoolState{
			Reserve:             int256.MustFromDec("725325240298071518380205"),
			ReserveWithSlippage: int256.MustFromDec("725338516452075133857542"),
			TotalLiabilities:    int256.MustFromDec("607213562903825489532767"),
			Price:               int256.MustFromDec("32538620000000000"),
		},
	}

	usdcPool := NablaPool{
		Meta: NablaPoolMeta{
			CurveBeta:                 int256.MustFromDec("5000000000000000"),
			CurveC:                    int256.MustFromDec("17075887234393789126"),
			LpFee:                     int256.NewInt(200),
			BackstopFee:               int256.NewInt(1500),
			ProtocolFee:               int256.NewInt(300),
			MaxCoverageRatioForSwapIn: int256.NewInt(200),
		},
		State: NablaPoolState{
			Reserve:             int256.MustFromDec("12691554632"),
			ReserveWithSlippage: int256.MustFromDec("12691697065"),
			TotalLiabilities:    int256.MustFromDec("15502707444"),
			Price:               int256.MustFromDec("999789830000000000"),
		},
	}

	return wmonPool, usdcPool
}

// Test_sell_monadSnapshot reproduces a fixed Monad WMON/USDC pool snapshot and
// checks the simulator against live router quotes captured from that state.
func Test_sell_monadSnapshot(t *testing.T) {
	wmonPool, usdcPool := monadSnapshotPools()

	tests := []struct {
		name     string
		amountIn string
		want     string
	}{
		{"0.1 WMON -> USDC", "100000000000000000", "3249"},
		{"1 WMON -> USDC", "1000000000000000000", "32472"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := sell(wmonPool, usdcPool, int256.MustFromDec(tt.amountIn), 18, 6)
			require.NoError(t, err)
			require.Equal(t, tt.want, got.Dec())
		})
	}
}
