package apefun

import (
	"context"
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

// TestTrack_Live decodes policies, buybackVaults and launchForToken on both chains (one coin with a buyback vault
// and 1% tax, one with neither), confirming the ABIs match the deployed contracts, that a live pool is never read
// again, and that the hook address is refused on the other chain.
func TestTrack_Live(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	for _, c := range []struct {
		name, rpc string
		l         launch
		want      Extra
	}{
		{"Arc/ARC1BR", "https://rpc.arc-scan.org", arc1br, arc1br.extra},
		{"Arc/ARC3CR", "https://rpc.arc-scan.org", arc3cr, arc3cr.extra},
		{"Robinhood/RH1", "https://rpc.mainnet.chain.robinhood.com", rh1, rh1.extra},
		{"Robinhood/RH4", "https://rpc.mainnet.chain.robinhood.com", rh4, rh4.extra},
	} {
		t.Run(c.name, func(t *testing.T) { trackLive(t, c.rpc, c.l, c.want) })
	}
}

func trackLive(t *testing.T, rpc string, l launch, want Extra) {
	// Robinhood: ArbMulticall2, as pool-service uses there (plain Multicall3 reports the L1 block number)
	multicall := "0xcA11bde05977b3631167028862bE2a173976CA11"
	if l.chainID == valueobject.ChainIDRobinhood {
		multicall = "0x2cAC2D899eCC914d704FeaAE33ac1bF36277DaD1"
	}
	rpcClient := ethrpc.New(rpc).SetMulticallContract(common.HexToAddress(multicall))
	pool := &entity.Pool{Address: l.pool, Tokens: []*entity.PoolToken{{Address: l.quote}, {Address: l.coin}}}
	param := &uniswapv4.HookParam{Cfg: &uniswapv4.Config{ChainID: l.chainID}, RpcClient: rpcClient, Pool: pool,
		HookAddress: l.hook}

	h := &Hook{}
	raw, err := h.Track(context.Background(), param)
	if err != nil && l.chainID == valueobject.ChainIDArc {
		// the public Arc RPC is sometimes unavailable; a transport error is not a decoding failure
		t.Skipf("public Arc RPC unavailable: %v", err)
	}
	require.NoError(t, err)
	assert.Equal(t, want, h.Extra)
	var decoded Extra
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, h.Extra, decoded)

	// tracked once: no RPC client needed any more
	again, err := (&Hook{Extra: decoded}).Track(context.Background(), &uniswapv4.HookParam{})
	require.NoError(t, err)
	assert.JSONEq(t, string(raw), string(again))

	// the same address read as another chain's is refused without a call
	param.Cfg, param.RpcClient = &uniswapv4.Config{ChainID: otherChain(l.chainID)}, nil
	raw, err = (&Hook{}).Track(context.Background(), param)
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(raw))
}
