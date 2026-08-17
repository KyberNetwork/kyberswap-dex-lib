package equalizer

import (
	"testing"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/require"
)

func TestReproReportedSlippage(t *testing.T) {
	poolEncoded := `{
		"address": "0xf97a0a96e66f664dcbc54944b9bca30cf3d97116",
		"swapFee": 0.0125,
		"exchange": "scale",
		"type": "equalizer",
		"blockNumber": 49928798,
		"reserves": ["10796764493308946194976356", "13688185853900865477"],
		"tokens": [
			{"address": "0x30c8cf6b46aa4df3f9fbc2857aca92f10a6cad7f", "symbol": "ELITE", "decimals": 18, "swappable": true},
			{"address": "0x4200000000000000000000000000000000000006", "symbol": "WETH", "decimals": 18, "swappable": true}
		],
		"staticExtra": "{\"stable\":false}"
	}`

	poolEntity := new(entity.Pool)
	if err := json.Unmarshal([]byte(poolEncoded), poolEntity); err != nil {
		t.Fatal(err)
	}

	poolSim, err := NewPoolSimulator(*poolEntity)
	if err != nil {
		t.Fatal(err)
	}

	tokenIn := "0x30c8cf6b46aa4df3f9fbc2857aca92f10a6cad7f"
	tokenOut := "0x4200000000000000000000000000000000000006"
	amountIn := "5179807500000000000000"

	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: bignumber.NewBig10(amountIn)},
		TokenOut:      tokenOut,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("amountIn          : %s", amountIn)
	t.Logf("offchain amountOut: %s", result.TokenAmountOut.Amount.String())
	t.Logf("onchain same-block: 6481824788515291")
	t.Logf("reported amountOut: 6481900482540010 (offchain)")
	t.Logf("reported onchain  : 6399038147692856 (swap block, different reserves/fee)")

	require.Equal(t, "6481824788515291", result.TokenAmountOut.Amount.String())
}
