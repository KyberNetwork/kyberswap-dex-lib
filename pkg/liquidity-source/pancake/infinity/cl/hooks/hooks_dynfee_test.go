package hooks_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl"
	_ "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl/hooks/alpha"
	_ "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl/hooks/dynamicfee"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl/hooks/feemanager"
	_ "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl/hooks/geniusmeme"
	_ "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl/hooks/limitorder"
	_ "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl/hooks/tax"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/internal/fakerpc"
)

// Registered hooks that never return a beforeSwap fee override (verified on Sourcify: alpha returns 0
// as the override, genius registers no beforeSwap, dynamicfee afterInitialize stores baseLpFee in
// slot0) must resolve a dynamic pool's fee to slot0.lpFee.
func TestRegisteredHooksUseSlot0LpFee(t *testing.T) {
	for name, addr := range map[string]string{
		"alpha":      "0x9a9B5331ce8d74b2B721291D57DE696E878353fd",
		"tax":        "0x26F251A46D15f396d2095738ad19869a13d4c9fD",
		"geniusmeme": "0xFf17F41c5Efd6CCe944Af0912F300097D62df5c9",
		"limitorder": "0x6AdC560aF85377f9a73d17c658D798c9B39186e8",
		"dynamicfee": "0x32C59D556B16DB81DFc32525eFb3CB257f7e493d",
	} {
		t.Run(name, func(t *testing.T) {
			hook, ok := cl.GetHook(common.HexToAddress(addr), nil)
			require.True(t, ok)
			require.EqualValues(t, 3000, hook.GetDynamicFee(t.Context(), &cl.HookParam{}, 3000))
		})
	}
}

// feemanager: its beforeSwap overrides only while enableLPFeeOverride is set (golden: same on origin/main).
func TestFeeManagerDynamicFee(t *testing.T) {
	const addr = "0x51854e4baa4a7c9653b52e42021f1b2b35d31651"
	for _, tc := range []struct {
		enabled bool
		want    uint32
	}{{false, 3000}, {true, 555}} {
		rpc := fakerpc.New(t)
		en, err := feemanager.Abi.Methods["enableLPFeeOverride"].Outputs.Pack(tc.enabled)
		require.NoError(t, err)
		fee, err := feemanager.Abi.Methods["lpFees"].Outputs.Pack(uint32(555))
		require.NoError(t, err)
		rpc.Calls[fakerpc.Key(addr, feemanager.Abi, "enableLPFeeOverride")] = en
		rpc.Calls[fakerpc.Key(addr, feemanager.Abi, "lpFees")] = fee

		param := &cl.HookParam{RpcClient: rpc.Client(), HookAddress: common.HexToAddress(addr),
			Pool: &entity.Pool{Address: "0xabc"}}
		hook, ok := cl.GetHook(common.HexToAddress(addr), param)
		require.True(t, ok)
		require.EqualValues(t, tc.want, hook.GetDynamicFee(t.Context(), param, 3000), tc.enabled)
	}
}
