package bin

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/bin/hooks/thanos"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/internal/fakerpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
)

// Thanos pools are dynamic-fee with swap callbacks. Their LP fee is the slot0 value (500), not the
// 100% BaseHook default, and the hook's swapFees(poolId) must be read in the same multicall.
// Fails if either the slot0 pass-through or the SwapFeeHook read is reverted.
func TestTrackerThanosDynamicFee(t *testing.T) {
	const mgr = "0xc697d2898e0d09264376196696c51d7abbbaa4a9"
	const poolID = "0xa859b22e97f32d4c7b1d9788044697712cb7183d294ed7aa1832799c1739e5cf"
	hook := thanos.BinHookAddresses[0]

	slot0, err := shared.BinPoolManagerABI.Methods["getSlot0"].Outputs.Pack(uint32(8388609), uint32(0), uint32(500))
	require.NoError(t, err)
	sf := (&thanos.Hook{}).SwapFeeCall(hook, poolID)
	swapFees, err := sf.ABI.Methods[sf.Method].Outputs.Pack(uint32(777))
	require.NoError(t, err)
	rpc := fakerpc.New(t)
	rpc.Calls[fakerpc.Key(mgr, shared.BinPoolManagerABI, "getSlot0")] = slot0
	rpc.Calls[fakerpc.Key(sf.Target, sf.ABI, sf.Method)] = swapFees
	se, _ := json.Marshal(StaticExtra{HasSwapPermissions: true, Fee: shared.DYNAMIC_FEE_FLAG, HooksAddress: hook})

	res, err := NewPoolTracker(&Config{BinPoolManagerAddress: mgr}, rpc.Client(), nil).
		FetchRPCData(t.Context(), &entity.Pool{Address: poolID, StaticExtra: string(se)}, 0)
	require.NoError(t, err)
	require.EqualValues(t, 500, res.SwapFee)
	require.EqualValues(t, 777, res.HookSwapFee)
}
