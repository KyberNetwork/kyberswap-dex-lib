package cl

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/internal/fakerpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
)

// A dynamic-fee pool charges slot0.lpFee unless the hook's beforeSwap returns an override
// (CLPool.swap: lpFeeOverride.isOverride() ? override : slot0.lpFee). Cases marked "golden" must
// match origin/main; the rest are the bug and fail if BaseHook.GetDynamicFee discards lpFee.

const (
	dynMgr      = "0xa0ffb9c1ce1fe56963b0321b32e7a0302114058b"
	dynPoolID   = "0x0000000000000000000000000000000000000000000000000000000000000abc"
	dynBaseHook = "0x00000000000000000000000000000000000d1001" // embeds BaseHook, like alpha/tax/genius
	dynOverride = "0x00000000000000000000000000000000000d1002" // overrides GetDynamicFee, like feemanager
	dynUnreg    = "0x00000000000000000000000000000000000d1003" // not registered
)

type overrideHook struct{ Hook }

func (overrideHook) GetDynamicFee(context.Context, *HookParam, uint32) uint32 { return 777 }

var _ = RegisterHooksFactory(func(p *HookParam) Hook { return NewBaseHook(DexType, p) }, common.HexToAddress(dynBaseHook))
var _ = RegisterHooksFactory(func(p *HookParam) Hook { return overrideHook{NewBaseHook(DexType, p)} },
	common.HexToAddress(dynOverride))

func dynSwapFee(t *testing.T, key uint32, hook string, slot0LpFee uint32) uint32 {
	t.Helper()
	rpc := fakerpc.New(t)
	slot0, err := shared.CLPoolManagerABI.Methods["getSlot0"].Outputs.Pack(
		new(big.Int).Lsh(big.NewInt(1), 96), big.NewInt(0), uint32(1000), slot0LpFee)
	require.NoError(t, err)
	liq, err := shared.CLPoolManagerABI.Methods["getLiquidity"].Outputs.Pack(big.NewInt(1000))
	require.NoError(t, err)
	rpc.Calls[fakerpc.Key(dynMgr, shared.CLPoolManagerABI, "getSlot0")] = slot0
	rpc.Calls[fakerpc.Key(dynMgr, shared.CLPoolManagerABI, "getLiquidity")] = liq

	tr := NewPoolTracker(&Config{CLPoolManagerAddress: dynMgr}, rpc.Client(), nil)
	p := entity.Pool{Address: dynPoolID, StaticExtra: fmt.Sprintf(
		`{"hsp":false,"0x0":[false,false],"fee":%d,"params":"0x0","tS":60,"pm":%q,"hooks":%q}`, key, dynMgr, hook)}
	res, _, _, err := tr.fetchOnchainState(t.Context(), &p, 0)
	require.NoError(t, err)
	return res.SwapFee
}

// swapFee = protocolFee(1000) + lpFee - protocolFee*lpFee/1e6.
func TestFetchOnchainStateDynamicFee(t *testing.T) {
	const dyn = 0x800000
	t.Run("golden static fee", func(t *testing.T) {
		require.EqualValues(t, 3997, dynSwapFee(t, 3000, dynUnreg, 3000))
		require.EqualValues(t, 3997, dynSwapFee(t, 3000, dynBaseHook, 3000))
	})
	t.Run("golden hook override wins", func(t *testing.T) {
		require.EqualValues(t, 1777, dynSwapFee(t, dyn, dynOverride, 3000)) // 1000+777-0
	})
	t.Run("golden dynamic with zero slot0", func(t *testing.T) {
		require.EqualValues(t, 1000, dynSwapFee(t, dyn, dynBaseHook, 0))
	})
	t.Run("unregistered hook falls back to slot0", func(t *testing.T) {
		require.EqualValues(t, 3997, dynSwapFee(t, dyn, dynUnreg, 3000))
	})
	t.Run("BaseHook-embedding hook falls back to slot0", func(t *testing.T) {
		require.EqualValues(t, 3997, dynSwapFee(t, dyn, dynBaseHook, 3000))
	})
}
