package onetoken

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

	head, err := rpcClient.GetBlockNumber(context.Background())
	require.NoError(t, err)
	block := new(big.Int).SetUint64(head - 5)

	poolID := common.HexToHash("0xe23622f7d41653c012cfb72e9d4addbffcfd67f7fb142b998cbd050a861d94fc") // WEREY/WETH
	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	_, err = h.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: hookAddr,
		BlockNumber: block,
		Pool:        &entity.Pool{Address: poolID.Hex()},
	})
	require.NoError(t, err)

	// Reference: one direct call per getter at the same block. The globals and the override
	// are owner-settable, so compare with live values instead of ones frozen on one date.
	var (
		launchTime                    = new(big.Int)
		snipeWindow                   = new(big.Int)
		overrideRaw, feeBps, snipeTax uint16
	)
	target := hexutil.Encode(hookAddr[:])
	_, err = rpcClient.NewRequest().SetBlockNumber(block).
		AddCall(&ethrpc.Call{ABI: HookABI, Target: target, Method: "launchTime", Params: []any{poolID}}, []any{&launchTime}).
		AddCall(&ethrpc.Call{ABI: HookABI, Target: target, Method: "poolFeeBpsOverride", Params: []any{poolID}}, []any{&overrideRaw}).
		AddCall(&ethrpc.Call{ABI: HookABI, Target: target, Method: "feeBps"}, []any{&feeBps}).
		AddCall(&ethrpc.Call{ABI: HookABI, Target: target, Method: "snipeTaxBps"}, []any{&snipeTax}).
		AddCall(&ethrpc.Call{ABI: HookABI, Target: target, Method: "snipeWindow"}, []any{&snipeWindow}).
		Aggregate()
	require.NoError(t, err)

	// poolFeeBpsOverride stores bps+1, so 0 means unset.
	wantOverride := int64(0)
	if overrideRaw > 0 {
		wantOverride = int64(overrideRaw) - 1
	}
	assert.Equal(t, launchTime.Int64(), h.LaunchTime)
	assert.Equal(t, wantOverride, h.OverrideBps)
	assert.Equal(t, int64(feeBps), h.FeeBps)
	assert.Equal(t, int64(snipeTax), h.SnipeTaxBps)
	assert.Equal(t, snipeWindow.Int64(), h.SnipeWindow)

	// launchTime is written once at launch, so this value is stable.
	// Cross-checked with `cast call ... --rpc-url https://mainnet.base.org` on 2026-09-09.
	assert.Equal(t, int64(1788896153), h.LaunchTime)
}
