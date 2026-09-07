package fables

import (
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var multicall3 = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")

func TestHookRegistration(t *testing.T) {
	t.Parallel()

	for _, addr := range HookAddresses {
		hook, ok := uniswapv4.GetHook(addr, &uniswapv4.HookParam{})
		assert.True(t, ok, "hook should be registered for %s", addr.Hex())
		assert.Equal(t, valueobject.ExchangeUniswapV4Fables, hook.GetExchange())
	}

	unknownAddr := common.HexToAddress("0x0000000000000000000000000000000000000001")
	_, ok := uniswapv4.GetHook(unknownAddr, &uniswapv4.HookParam{})
	assert.False(t, ok, "unknown address should not be registered")
}

func TestHookFactory_WithExtra(t *testing.T) {
	t.Parallel()

	extra := Extra{Fee0For1: 400, Fee1For0: 800}
	extraBytes, _ := json.Marshal(extra)

	hook, ok := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{
		HookExtra: extraBytes,
	})
	require.True(t, ok)

	fablesHook, ok := hook.(*Hook)
	require.True(t, ok)
	assert.Equal(t, uniswapv4.FeeAmount(400), fablesHook.Fee0For1)
	assert.Equal(t, uniswapv4.FeeAmount(800), fablesHook.Fee1For0)
}

func TestBeforeSwap_ReturnsDirectionalFee(t *testing.T) {
	t.Parallel()

	extra := Extra{Fee0For1: 400, Fee1For0: 800}
	extraBytes, _ := json.Marshal(extra)
	hook, _ := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{HookExtra: extraBytes})

	// zeroForOne -> Fee0For1
	res, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{
		CalcOut:         true,
		ZeroForOne:      true,
		AmountSpecified: big.NewInt(1_000_000),
	})
	require.NoError(t, err)
	assert.Equal(t, uniswapv4.FeeAmount(400), res.SwapFee)
	assert.Equal(t, int64(0), res.DeltaSpecified.Int64())
	assert.Equal(t, int64(0), res.DeltaUnspecified.Int64())

	// oneForZero -> Fee1For0
	res, err = hook.BeforeSwap(&uniswapv4.BeforeSwapParams{
		CalcOut:         true,
		ZeroForOne:      false,
		AmountSpecified: big.NewInt(1_000_000),
	})
	require.NoError(t, err)
	assert.Equal(t, uniswapv4.FeeAmount(800), res.SwapFee)
}

func TestAfterSwap_Noop(t *testing.T) {
	t.Parallel()

	// Fables has no afterSwap permission and takes no hook fee; the base no-op must hold.
	hook, _ := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{})
	res, err := hook.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{
			CalcOut:         true,
			ZeroForOne:      true,
			AmountSpecified: big.NewInt(1_000_000),
		},
		AmountIn:  big.NewInt(1_000_000),
		AmountOut: big.NewInt(999_600),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), res.HookFee.Int64())
}

func TestCanSwapPermissions(t *testing.T) {
	t.Parallel()

	// Every Fables hook address carries the beforeSwap flag and no afterSwap flag.
	for _, addr := range HookAddresses {
		hook, _ := uniswapv4.GetHook(addr, &uniswapv4.HookParam{})
		assert.True(t, hook.CanBeforeSwap(addr), "beforeSwap flag expected on %s", addr.Hex())
		assert.False(t, hook.CanAfterSwap(addr), "no afterSwap flag expected on %s", addr.Hex())
	}
}

func TestHookAddresses_RegistrySnapshot(t *testing.T) {
	t.Parallel()

	// wantHooks is the registry's poolCount() at the last regeneration of HookAddresses, one
	// hook per pool, so the list must carry exactly that many distinct addresses.
	const wantHooks = 13
	assert.Len(t, HookAddresses, wantHooks)

	seen := make(map[common.Address]struct{}, len(HookAddresses))
	for _, addr := range HookAddresses {
		_, dup := seen[addr]
		assert.False(t, dup, "duplicate hook address %s", addr.Hex())
		seen[addr] = struct{}{}

		_, ok := uniswapv4.GetHook(addr, &uniswapv4.HookParam{})
		require.True(t, ok, "hook should be registered for %s", addr.Hex())
	}
}

func TestCloneState_DeepCopy(t *testing.T) {
	t.Parallel()

	original := &Hook{
		Hook:     &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Fables},
		Fee0For1: 400,
		Fee1For0: 800,
	}
	cloned := original.CloneState().(*Hook)
	cloned.Fee0For1 = 100
	cloned.Fee1For0 = 100

	assert.Equal(t, uniswapv4.FeeAmount(400), original.Fee0For1)
	assert.Equal(t, uniswapv4.FeeAmount(800), original.Fee1For0)
}

// --- Live RPC tests (skipped in CI) ---
//
// Reads the resolved fee for live Fables pools on Robinhood Chain (chainId 4663) through the
// public RPC (rate limited; each test issues a single multicall). The pool id below is registry
// index 0 (USDG/NVDA).
func TestTrack_RobinhoodChain(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping RPC test in CI")
	}

	rpcClient := ethrpc.New("https://rpc.mainnet.chain.robinhood.com").SetMulticallContract(multicall3)
	param := &uniswapv4.HookParam{
		Cfg:         &uniswapv4.Config{ChainID: 4663},
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0x66622f77B797D506e5376F7798b67ab288966080"), // NVDA hook
		Pool: &entity.Pool{
			Address: "0x7990aad9e8fb048f49a155a7df5603db0366f0657035b78eb4196395cccb3dcd",
		},
	}
	hook, _ := uniswapv4.GetHook(param.HookAddress, param)
	extraStr, err := hook.Track(t.Context(), param)
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal(extraStr, &extra))
	t.Logf("Fables NVDA fee: 0For1=%d 1For0=%d", extra.Fee0For1, extra.Fee1For0)
	assert.True(t, extra.Fee0For1 > 0, "resolved fee should be > 0")
	assert.True(t, extra.Fee1For0 > 0, "resolved fee should be > 0")
}

// Same read against a wave-4 pool: registry index 12, ETH/PONS on FablesRampETH (tick spacing
// 60, native ETH as currency0). This wave-4 build exposes the per-pool directional premium
// (poolAsymmetry), so the two sides are read independently and only asserted positive, never
// equal.
func TestTrack_RobinhoodChain_EthPons(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping RPC test in CI")
	}

	rpcClient := ethrpc.New("https://rpc.mainnet.chain.robinhood.com").SetMulticallContract(multicall3)
	param := &uniswapv4.HookParam{
		Cfg:         &uniswapv4.Config{ChainID: 4663},
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0x594e8e6281eDf2d363a0293a50004Cf868E7a080"), // ETH/PONS hook
		Pool: &entity.Pool{
			Address: "0xb59001413cb070e28433826f927b7265a0813213ba21f454d86896cee3cce674",
		},
	}
	hook, ok := uniswapv4.GetHook(param.HookAddress, param)
	require.True(t, ok, "ETH/PONS hook should be registered")
	extraStr, err := hook.Track(t.Context(), param)
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal(extraStr, &extra))
	t.Logf("Fables ETH/PONS fee: 0For1=%d 1For0=%d", extra.Fee0For1, extra.Fee1For0)
	assert.True(t, extra.Fee0For1 > 0, "resolved fee should be > 0")
	assert.True(t, extra.Fee1For0 > 0, "resolved fee should be > 0")
}
