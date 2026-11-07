package flaunch

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// TestTrack_Live decodes getPoolFeeDistribution / feeCalculator / poolCalculator against the
// vested AnyPositionManager on Base and a real pool on it (VCANARY, launched 2026-09-17,
// tx 0xfc8f7ae0fd9714cfe65fb33c350ea0b37e1436a1b84f20bf011639fa7f919e56), confirming the ABI
// field order and the dispatcher walk against deployed contracts, not just compiling JSON.
func TestTrack_Live(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	rpcClient := ethrpc.New("https://mainnet.base.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	extra, err := h.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0xE753a351FB498051a09Dc130fcC29aEBc76525DC"),
		Pool:        &entity.Pool{Address: "0xd8c9dd1fbc32511cb94adff771fb4c0da20c3070c6e439665a96adeda361c5df"},
	})
	require.NoError(t, err)
	assert.Equal(t, uint32(DefaultSwapFee), h.SwapFee, "canary pool runs the protocol-default 1% swapFee")
	assert.False(t, h.GateEnabled, "canary pool is on the dispatcher's default calculator, no spend gate")
	assert.JSONEq(t, `{"f":100}`, string(extra))
}

// TestTrack_Live_V160 runs the same decode against the v1.6.0 PositionManager (CREATE3 parity on
// Base, Arbitrum, Robinhood and Ethereum) and a live Base pool on it, coin
// 0x8E7e6d5e10CF4406FEF62A41E411F1dA3e104696 paired with native ETH (fee 0, tickSpacing 60).
// v1.6.0 caps swapFee at 10% and stops reverting on fee-calculator failures, but the
// getPoolFeeDistribution / dispatcher surface Track reads is unchanged.
func TestTrack_Live_V160(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	rpcClient := ethrpc.New("https://mainnet.base.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	extra, err := h.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0x54bEDD41AfDf57722518cF6Dd826a70d81e9a5DC"),
		Pool:        &entity.Pool{Address: "0x336dcfd7a14da3083635ad739715d9ad588312df07b647e806789ebbcde816a9"},
	})
	require.NoError(t, err)
	assert.Equal(t, uint32(DefaultSwapFee), h.SwapFee, "v1.6.0 pool runs the protocol-default 1% swapFee")
	assert.False(t, h.GateEnabled, "v1.6.0 pool is on the dispatcher's default calculator, no spend gate")
	assert.JSONEq(t, `{"f":100}`, string(extra))
}

// The v1.0 PositionManager predates the dispatcher: its feeCalculator has no poolCalculator, so
// Track must still return the pool's fee and simply report no gate.
func TestTrack_Live_PreDispatcherGeneration(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	poolId := os.Getenv("FLAUNCH_V1_POOL_ID")
	if poolId == "" {
		t.Skip("set FLAUNCH_V1_POOL_ID to a pool on PositionManager 1.0 to run")
	}
	rpcClient := ethrpc.New("https://mainnet.base.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	_, err := h.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: HookAddresses[0],
		Pool:        &entity.Pool{Address: poolId},
	})
	require.NoError(t, err)
	assert.NotZero(t, h.SwapFee)
	assert.False(t, h.GateEnabled)
}
