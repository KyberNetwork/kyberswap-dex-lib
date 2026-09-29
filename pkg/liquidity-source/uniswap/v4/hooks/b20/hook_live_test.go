package b20

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

// TestTrack_Live decodes poolConfig against a real B20 launch on Base (found via a
// live launch tx, see explorer.md), confirming LaunchHookABI's field order/types
// actually match the deployed contract -- not just the ABI JSON compiling.
func TestTrack_Live(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://mainnet.base.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	hookAddr := common.HexToAddress("0x985c14baa2A18316ffDA0AefB3a632faDFCA2acc")

	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	_, err := h.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: hookAddr,
		Pool: &entity.Pool{
			Address: "0x68d39022eee9e18f82fe929f70dd8d2009e442e6d80c6c1ffc170c32b7d3b671",
		},
	})
	require.NoError(t, err)

	// Values cross-checked by hand against `cast call ... poolConfig(bytes32)` on 2026-08-25.
	assert.False(t, h.TokenIsCurrency0)
	assert.Equal(t, int64(100), h.BaseFeeBps)
	assert.Equal(t, int64(9900), h.AntiSnipeStartTotalBps)
	assert.Equal(t, int64(16), h.AntiSnipeWindowSeconds)
	assert.Equal(t, int64(1786986263), h.LaunchTime)
}
