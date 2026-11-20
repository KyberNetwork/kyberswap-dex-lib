package doppler

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// A fee-charging swap on the robinhood rehype hook costs its measured gas; other rehype hooks and fee-free swaps keep
// the shared default.
func TestRehypeGasByHook(t *testing.T) {
	robinhood := common.HexToAddress("0x6f02324d20CC679d0E585290CAa6b16baCbC0F77")
	base := common.HexToAddress("0x3Ec4798A9B11e8243A8Db99687f7A23597B96623")
	params := &uniswapv4.AfterSwapParams{BeforeSwapParams: &uniswapv4.BeforeSwapParams{}, AmountIn: big.NewInt(1_000_000)}

	for _, tc := range []struct {
		name  string
		dHook common.Address
		extra string
		gas   int64
	}{
		{"robinhood with fee", robinhood, `{"s":10500}`, 660_000},
		{"robinhood without fee", robinhood, `{}`, 198835},
		{"other rehype hook with fee", base, `{"s":10500}`, 198835},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hook := &DHook{DHook: tc.dHook, IDHook: DHooks[tc.dHook](json.RawMessage(tc.extra))}
			res, err := hook.AfterSwap(params)
			require.NoError(t, err)
			assert.Equal(t, tc.gas, res.Gas)
		})
	}
}
