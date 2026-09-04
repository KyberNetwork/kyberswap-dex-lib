package odysfun

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// TestTrack hits Arbitrum One mainnet and reads a real, previously-verified ODYS Elite pool
// (see PR description for the `cast`/tenderly trail): a 3% launch that settled to 1% on
// 2026-08-27, well in the past by the time this test runs, so currentFeeBps must read 100.
func TestTrack(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://arb1.arbitrum.io/rpc").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	h := NewHook(&uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0x18a6ba193352036ef8a7c22be2cb288bb26da8cc"),
	}).(*OdysFunHook)

	_, err := h.Track(context.Background(), &uniswapv4.HookParam{
		Cfg:         &uniswapv4.Config{ChainID: valueobject.ChainIDArbitrumOne},
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0x18a6ba193352036ef8a7c22be2cb288bb26da8cc"),
		Pool: &entity.Pool{
			Address: "0x1078939501c9115631ec4c38a8694e743d5393f20a96a32bc6e167869159db1b",
		},
	})
	require.NoError(t, err)

	assert.EqualValues(t, 1787655531, h.LaunchTime)
	assert.EqualValues(t, 300, h.InitialFeeBps)
	assert.EqualValues(t, 100, h.SettledFeeBps)
	assert.EqualValues(t, 1787828331, h.SettleTimestamp)
	assert.EqualValues(t, 100, h.currentFeeBps()) // launch and settleTimestamp are long past
}

// TestTrackLegacyHook hits the older OdysHook (first Elite line, static rate, one word fewer
// in pools() than OdysHook2 -- no settledFeeBps/settleTimestamp at all). Confirms Track uses
// the legacy 7-field ABI for this address instead of the OdysHook2 9-field one, which fails to
// decode against it.
func TestTrackLegacyHook(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://arb1.arbitrum.io/rpc").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	legacyHook := common.HexToAddress("0xBC40c7492F5823785c4dcC1EC2F9D1b7406b28Cc")
	h := NewHook(&uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: legacyHook,
	}).(*OdysFunHook)

	_, err := h.Track(context.Background(), &uniswapv4.HookParam{
		Cfg:         &uniswapv4.Config{ChainID: valueobject.ChainIDArbitrumOne},
		RpcClient:   rpcClient,
		HookAddress: legacyHook,
		Pool: &entity.Pool{
			// pool for allTokens(0) on OdysFactoryV4 (0x1C72a544F65F7b838f3b40ce144235135B6Bd154)
			Address: "0x089aeb60de799dac81d1090bee363dfc4af5c0dd9880509c786cfe97bca0bf4c",
		},
	})
	require.NoError(t, err)

	assert.EqualValues(t, 1787483996, h.LaunchTime)
	assert.EqualValues(t, 300, h.InitialFeeBps)
	assert.Zero(t, h.SettledFeeBps)
	assert.Zero(t, h.SettleTimestamp)
	assert.EqualValues(t, 300, h.currentFeeBps()) // no settle window -> initialFeeBps forever
}

func TestCurrentFeeBps(t *testing.T) {
	t.Parallel()

	// launchTime far enough in the past that every case below is outside the anti-snipe window
	// (elapsed >> 15s) unless a case overrides it to test that window specifically.
	longAgo := uint64(time.Now().Unix()) - 3600

	t.Run("no settle window -> initial forever", func(t *testing.T) {
		h := &OdysFunHook{LaunchTime: longAgo, InitialFeeBps: 300, SettledFeeBps: 300, SettleTimestamp: 0}
		assert.EqualValues(t, 300, h.currentFeeBps())
	})

	t.Run("before settle -> initial", func(t *testing.T) {
		h := &OdysFunHook{LaunchTime: longAgo, InitialFeeBps: 500, SettledFeeBps: 100, SettleTimestamp: ^uint64(0)}
		assert.EqualValues(t, 500, h.currentFeeBps())
	})

	t.Run("after settle -> settled", func(t *testing.T) {
		h := &OdysFunHook{LaunchTime: longAgo, InitialFeeBps: 500, SettledFeeBps: 100, SettleTimestamp: 1}
		assert.EqualValues(t, 100, h.currentFeeBps())
	})

	t.Run("elapsed < 5s -> flat 15% regardless of schedule", func(t *testing.T) {
		h := &OdysFunHook{
			LaunchTime: uint64(time.Now().Unix()) - 2, InitialFeeBps: 100, SettledFeeBps: 100,
		}
		assert.EqualValues(t, antiSnipeHardFeeBps, h.currentFeeBps())
	})

	t.Run("5s <= elapsed < 15s, schedule below floor -> 5% floor", func(t *testing.T) {
		h := &OdysFunHook{
			LaunchTime: uint64(time.Now().Unix()) - 10, InitialFeeBps: 100, SettledFeeBps: 100,
		}
		assert.EqualValues(t, antiSnipeFloorFeeBps, h.currentFeeBps())
	})

	t.Run("5s <= elapsed < 15s, schedule above floor -> schedule wins", func(t *testing.T) {
		h := &OdysFunHook{
			LaunchTime: uint64(time.Now().Unix()) - 10, InitialFeeBps: 900, SettledFeeBps: 900,
		}
		assert.EqualValues(t, 900, h.currentFeeBps())
	})

	t.Run("elapsed >= 15s -> schedule, no floor even if below 5%", func(t *testing.T) {
		h := &OdysFunHook{
			LaunchTime: uint64(time.Now().Unix()) - 20, InitialFeeBps: 100, SettledFeeBps: 100,
		}
		assert.EqualValues(t, 100, h.currentFeeBps())
	})
}

// TestBeforeAfterSwap covers all 4 buy/sell x CalcOut/CalcIn combinations at a fixed 5% rate,
// checking that the tax always lands on the ETH (currency0) leg and that the exactIn side
// takes fee-on-top while the exactOut side grosses up.
func TestBeforeAfterSwap(t *testing.T) {
	t.Parallel()

	h := &OdysFunHook{InitialFeeBps: 500, SettledFeeBps: 500, SettleTimestamp: 0} // fixed 5%
	amount := big.NewInt(1_000_000)

	t.Run("buy, CalcAmountOut (exactIn ETH)", func(t *testing.T) {
		res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{
			CalcOut: true, ZeroForOne: true, AmountSpecified: amount,
		})
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(50_000), res.DeltaSpecified) // 5% fee-on-top

		afterRes, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
			BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: amount},
			AmountOut:        big.NewInt(999_999_999), // token amount, not ETH; must not be taxed here
		})
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(0), afterRes.HookFee)
	})

	t.Run("sell, CalcAmountOut (exactIn token, ETH out taxed after)", func(t *testing.T) {
		res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{
			CalcOut: true, ZeroForOne: false, AmountSpecified: amount,
		})
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(0), res.DeltaSpecified)

		ethOut := big.NewInt(1_000_000)
		afterRes, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
			BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: false, AmountSpecified: amount},
			AmountOut:        ethOut,
		})
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(50_000), afterRes.HookFee) // 5% fee-on-top of the ETH output
	})

	t.Run("buy, CalcAmountIn (exactOut token, ETH in grossed up after)", func(t *testing.T) {
		res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{
			CalcOut: false, ZeroForOne: true, AmountSpecified: amount,
		})
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(0), res.DeltaSpecified)

		ethIn := big.NewInt(1_000_000)
		afterRes, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
			BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: false, ZeroForOne: true, AmountSpecified: amount},
			AmountIn:         ethIn,
		})
		require.NoError(t, err)
		// gross-up: fee = amount * bps / (10000 - bps) = 1_000_000 * 500 / 9500 = 52631 (floor)
		assert.Equal(t, big.NewInt(52631), afterRes.HookFee)
	})

	t.Run("sell, CalcAmountIn (exactOut ETH, grossed up before)", func(t *testing.T) {
		res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{
			CalcOut: false, ZeroForOne: false, AmountSpecified: amount,
		})
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(52631), res.DeltaSpecified) // gross-up on the fixed ETH output

		afterRes, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
			BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: false, ZeroForOne: false, AmountSpecified: amount},
			AmountIn:         big.NewInt(999_999_999), // token amount; must not be taxed here
		})
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(0), afterRes.HookFee)
	})
}
