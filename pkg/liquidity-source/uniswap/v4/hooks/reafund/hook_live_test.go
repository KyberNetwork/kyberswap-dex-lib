package reafund

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	ponsv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/pons-v2"
)

const robinhoodRPC = "https://rpc.mainnet.chain.robinhood.com"

// The fee escrow each hook was constructed with: PonsV2MemeHook's only immutable that differs
// between deployments.
var (
	escrow     = common.HexToAddress("0x11a21c0639f38375f70B663d73fA2085e30d20c2")
	ponsEscrow = common.HexToAddress("0xd3AFEB2a57f70eF218Aa82451c51B2fb0416Ac9e")
)

// TestBytecodeParity_Live checks what this package assumes: REA.fund's hook runs Pons' hook's
// code. With the CBOR metadata trailer cut and the fee escrow zeroed, the two runtime codes must
// be the same bytes.
func TestBytecodeParity_Live(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	ctx := context.Background()
	rpcClient, err := rpc.DialOptions(ctx, robinhoodRPC, rpc.WithHeader("User-Agent", "kyberswap-dex-lib live test"))
	require.NoError(t, err)
	client := ethclient.NewClient(rpcClient)

	pons := strippedCode(t, ctx, client, ponsv2.HookAddresses[0], ponsEscrow)
	ours := strippedCode(t, ctx, client, HookAddresses[0], escrow)
	require.Equal(t, len(pons), len(ours), "code length")
	assert.True(t, bytes.Equal(pons, ours), "runtime code differs from Pons' hook beyond the escrow and metadata")
}

// strippedCode is a hook's runtime code without its metadata trailer and with its fee escrow
// zeroed. The escrow must appear exactly three times, PonsV2MemeHook's references to it.
func strippedCode(t *testing.T, ctx context.Context, client *ethclient.Client, hook, escrow common.Address) []byte {
	t.Helper()
	code, err := client.CodeAt(ctx, hook, nil)
	require.NoError(t, err)
	require.Greater(t, len(code), 2, "%s: no code", hook)

	metadataLen := int(code[len(code)-2])<<8 | int(code[len(code)-1])
	require.Less(t, metadataLen+2, len(code), "%s: metadata trailer longer than code", hook)
	code = code[:len(code)-metadataLen-2]

	require.Equal(t, 3, bytes.Count(code, escrow.Bytes()), "%s: escrow references", hook)
	return bytes.ReplaceAll(code, escrow.Bytes(), make([]byte, common.AddressLength))
}

// TestTrack_Live decodes the hook's `launches` getter with hooks/pons-v2's ABI. No REA.fund launch
// has graduated yet, so it asks about a pool the hook has never registered: the call succeeds and
// the pool stays unregistered and fee-free.
func TestTrack_Live(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	rpcClient := ethrpc.New(robinhoodRPC).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	h := &Hook{}
	raw, err := h.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: HookAddresses[0],
		Pool:        &entity.Pool{Address: crypto.Keccak256Hash([]byte("reafund: no such pool")).Hex()},
	})
	require.NoError(t, err)
	assert.False(t, h.Registered)
	assert.Zero(t, h.FeeBps)
	assert.JSONEq(t, `{}`, string(raw))
}
