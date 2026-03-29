package alphixlvrfee

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
		assert.Equal(t, valueobject.ExchangeUniswapV4AlphixLvrFee, hook.GetExchange())
	}

	unknownAddr := common.HexToAddress("0x0000000000000000000000000000000000000001")
	_, ok := uniswapv4.GetHook(unknownAddr, &uniswapv4.HookParam{})
	assert.False(t, ok, "unknown address should not be registered")
}

func TestHookFactory_WithExtra(t *testing.T) {
	t.Parallel()

	extra := Extra{SwapFee: 499, HookFee: 5000}
	extraBytes, _ := json.Marshal(extra)

	hook, ok := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{
		HookExtra: extraBytes,
	})
	require.True(t, ok)

	lvrHook, ok := hook.(*Hook)
	require.True(t, ok)
	assert.Equal(t, uniswapv4.FeeAmount(499), lvrHook.SwapFee)
	assert.Equal(t, int64(5000), lvrHook.HookFee)
}

func TestBeforeSwap_ReturnsDynamicFee(t *testing.T) {
	t.Parallel()

	extra := Extra{SwapFee: 499}
	extraBytes, _ := json.Marshal(extra)

	hook, _ := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{
		HookExtra: extraBytes,
	})

	result, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{
		CalcOut:         true,
		ZeroForOne:      true,
		AmountSpecified: big.NewInt(1_000_000),
	})
	require.NoError(t, err)
	assert.Equal(t, uniswapv4.FeeAmount(499), result.SwapFee)
	assert.Equal(t, int64(0), result.DeltaSpecified.Int64())
	assert.Equal(t, int64(0), result.DeltaUnspecified.Int64())
}

func TestAfterSwap_NoHookFee(t *testing.T) {
	t.Parallel()

	// HookFee = 0 means no fee taken
	hook, _ := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{})

	result, err := hook.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{
			CalcOut:         true,
			ZeroForOne:      true,
			AmountSpecified: big.NewInt(1_000_000),
		},
		AmountIn:  big.NewInt(1_000_000),
		AmountOut: big.NewInt(999_500),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.HookFee.Int64())
}

func TestAfterSwap_WithHookFee(t *testing.T) {
	t.Parallel()

	// HookFee = 5000 means 0.5% (5000/1_000_000)
	extra := Extra{SwapFee: 499, HookFee: 5000}
	extraBytes, _ := json.Marshal(extra)

	hook, _ := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{
		HookExtra: extraBytes,
	})

	result, err := hook.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{
			CalcOut:         true,
			ZeroForOne:      true,
			AmountSpecified: big.NewInt(1_000_000_000),
		},
		AmountIn:  big.NewInt(1_000_000_000),
		AmountOut: big.NewInt(999_500_000),
	})
	require.NoError(t, err)
	// 999_500_000 * 5000 / 1_000_000 = 4_997_500
	assert.Equal(t, int64(4_997_500), result.HookFee.Int64())
}

func TestCloneState_DeepCopy(t *testing.T) {
	t.Parallel()

	original := &Hook{
		Hook:    &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4AlphixLvrFee},
		SwapFee: 499,
		HookFee: 5000,
	}

	cloned := original.CloneState().(*Hook)
	cloned.SwapFee = 1000
	cloned.HookFee = 0

	assert.Equal(t, uniswapv4.FeeAmount(499), original.SwapFee)
	assert.Equal(t, int64(5000), original.HookFee)
}

func TestParseHookAddresses(t *testing.T) {
	t.Parallel()

	assert.Equal(t, common.HexToAddress("0x7cBbfF9C4fcd74B221C535F4fB4B1Db04F1B9044"), HookAddresses[0])
}

// --- Live RPC tests (skipped in CI) ---

func TestTrack_BaseETHUSDC(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping RPC test in CI")
	}

	rpcClient := ethrpc.New("https://mainnet.base.org").SetMulticallContract(multicall3)
	param := &uniswapv4.HookParam{
		Cfg:       &uniswapv4.Config{ChainID: 8453},
		RpcClient: rpcClient,
		Pool: &entity.Pool{
			Address: "0xebb666a5c6449b83536950b975d74deb32aca1537a501b58161a896816b04da6",
		},
	}
	hook, _ := uniswapv4.GetHook(HookAddresses[0], param)
	extraStr, err := hook.Track(t.Context(), param)
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal(extraStr, &extra))
	t.Logf("Base ETH/USDC (LVRFee): swapFee=%d hookFee=%d", extra.SwapFee, extra.HookFee)
}

func TestTrack_BaseETHcbBTC(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping RPC test in CI")
	}

	rpcClient := ethrpc.New("https://mainnet.base.org").SetMulticallContract(multicall3)
	param := &uniswapv4.HookParam{
		Cfg:       &uniswapv4.Config{ChainID: 8453},
		RpcClient: rpcClient,
		Pool: &entity.Pool{
			Address: "0x3860784278e9e481ffd0888430ab2af8f2bb1180069f31cde9e1066728bbe73b",
		},
	}
	hook, _ := uniswapv4.GetHook(HookAddresses[0], param)
	extraStr, err := hook.Track(t.Context(), param)
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal(extraStr, &extra))
	t.Logf("Base ETH/cbBTC (LVRFee): swapFee=%d hookFee=%d", extra.SwapFee, extra.HookFee)
}
