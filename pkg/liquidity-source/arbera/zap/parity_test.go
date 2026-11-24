package arberazap

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// goldenCase is one recorded quote from the last legacy build (baseline 4145096b5).
type goldenCase struct {
	In        int    `json:"in"`
	Out       int    `json:"out"`
	AmountIn  string `json:"amountIn"`
	OK        bool   `json:"ok"`
	AmountOut string `json:"amountOut"`
	Fee       string `json:"fee"`
	IsBuy     bool   `json:"isBuy"`
}

// TestGoldenParity pins the native CalcAmtOut to the recorded legacy quotes, including which inputs error.
func TestGoldenParity(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/golden.json")
	require.NoError(t, err)
	var golden struct {
		Cases []goldenCase `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(raw, &golden))
	require.NotEmpty(t, golden.Cases)

	for _, c := range golden.Cases {
		t.Run(fmt.Sprintf("in%d_out%d_%s", c.In, c.Out, c.AmountIn), func(t *testing.T) {
			var amt uint256.Int
			require.NoError(t, amt.SetFromDecimal(c.AmountIn))
			res, err := zapSim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: zapSim.Info.Tokens[c.In], Amount: amt},
				TokenOut:      zapSim.Info.Tokens[c.Out],
			})
			if !c.OK {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.AmountOut, res.TokenAmountOut.Amount.Dec())
			require.Equal(t, c.Fee, res.Fee.Amount.Dec())
			require.Equal(t, c.IsBuy, res.SwapInfo.(SwapInfo).IsBuy)
		})
	}
}
