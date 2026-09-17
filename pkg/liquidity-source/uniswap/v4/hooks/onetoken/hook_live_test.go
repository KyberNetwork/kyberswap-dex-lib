package onetoken

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

// TestTrack_Live reads the OneToken hook against a real launch on Base (WEREY),
// confirming HookABI's method names / arg (bytes32 PoolId) / return types actually
// match the deployed contract, not just that the ABI JSON compiles. It also proves
// the multicall of the two per-pool getters plus the three globals decodes together.
func TestTrack_Live(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping live test in CI environment")
	}

	rpcClient := ethrpc.New("https://mainnet.base.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	hookAddr := common.HexToAddress("0x63e0Ff2e9c38dB24C56A075B69653D99c412C880") // Base OneToken hook

	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	_, err := h.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: hookAddr,
		Pool: &entity.Pool{
			// WEREY/WETH poolId
			Address: "0xe23622f7d41653c012cfb72e9d4addbffcfd67f7fb142b998cbd050a861d94fc",
		},
	})
	require.NoError(t, err)

	// launchTime and the override are frozen per pool, so they are stable assertions.
	// Values cross-checked with `cast call ... --rpc-url https://mainnet.base.org` on 2026-09-09.
	assert.Equal(t, int64(1788896153), h.LaunchTime)
	assert.Equal(t, int64(0), h.OverrideBps)

	// feeBps / snipeTaxBps / snipeWindow are hook-global and owner-settable, so these
	// are the values as of the cross-check date; if the owner later changes them, update
	// these three (the test failing here means the globals moved, not that decoding broke).
	assert.Equal(t, int64(300), h.FeeBps)
	assert.Equal(t, int64(1500), h.SnipeTaxBps)
	assert.Equal(t, int64(900), h.SnipeWindow)
}
