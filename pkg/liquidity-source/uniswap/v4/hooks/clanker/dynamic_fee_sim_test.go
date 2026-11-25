package clanker

import (
	"fmt"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// TestDynamicFeeBeforeSwapGolden pins BeforeSwap, which prices the swap with the pool simulator to get the
// volatility accumulator. The golden values were captured with the big.Int pool simulator path; they must not
// move when that simulation runs in uint256.
func TestDynamicFeeBeforeSwapGolden(t *testing.T) {
	t.Parallel()
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(dynamicFeePoolData), &p))
	var extra uniswapv4.ExtraU256
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	h := NewDynamicFeeHook(Clanker)(&uniswapv4.HookParam{
		Cfg:       &uniswapv4.Config{ChainID: valueobject.ChainIDArbitrumOne},
		Pool:      &p,
		HookExtra: uniswapv4.HookExtra(extra.HookExtra),
	}).(*DynamicFeeHook)

	var got []string
	for _, zeroForOne := range []bool{true, false} {
		for _, amt := range []string{"1", "1000000000", "1000000000000000", "1000000000000000000", "500000000000000000000", "10000000000000000000000"} {
			res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zeroForOne,
				AmountSpecified: uint256.MustFromDecimal(amt)})
			if err != nil {
				got = append(got, fmt.Sprintf("%v/%s err=%v", zeroForOne, amt, err))
				continue
			}
			got = append(got, fmt.Sprintf("%v/%s fee=%d spec=%s pfee=%s", zeroForOne, amt, res.SwapFee,
				res.DeltaSpecified.Dec(), res.SwapInfo.(*dynamicFeeSwapInfo).protocolFee))
		}
	}
	require.Equal(t, golden, got)
}

var golden = []string{
	"true/1 err=zero amount", "true/1000000000 err=zero amount",
	"true/1000000000000000 fee=5000 spec=0 pfee=1000",
	"true/1000000000000000000 fee=5000 spec=0 pfee=1000",
	"true/500000000000000000000 fee=5000 spec=0 pfee=1000",
	"true/10000000000000000000000 fee=5000 spec=0 pfee=1000",
	"false/1 err=zero amount",
	"false/1000000000 fee=5000 spec=999000 pfee=1000",
	"false/1000000000000000 fee=5000 spec=999000999000 pfee=1000",
	"false/1000000000000000000 fee=13241 spec=2641006614484844 pfee=2648",
	"false/500000000000000000000 fee=50000 spec=4950495049504950000 pfee=10000",
	"false/10000000000000000000000 err=at or above largest",
}
