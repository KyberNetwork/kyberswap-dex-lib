package coocoo

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

// deployments are CooCoo's hooks with the fee escrow each was constructed with -- the one
// immutable that differs between deployments of PonsV2MemeHook.
var deployments = []struct {
	name, rpc    string
	hook, escrow common.Address
}{
	{"robinhood", "https://rpc.mainnet.chain.robinhood.com", HookAddresses[0],
		common.HexToAddress("0xB1703f631084fec914cE2834F29EBEb8e07d3fb4")},
	{"arc", "https://rpc.mainnet.arc.io", HookAddresses[1],
		common.HexToAddress("0x21524873EFaB3Af236A35eA287099eD1fBe4c4cD")},
}

// Pons' own hook (hooks/pons-v2) and the escrow it was constructed with, on Robinhood chain.
var (
	ponsRPC    = "https://rpc.mainnet.chain.robinhood.com"
	ponsEscrow = common.HexToAddress("0xd3AFEB2a57f70eF218Aa82451c51B2fb0416Ac9e")
)

const multicall3 = "0xcA11bde05977b3631167028862bE2a173976CA11"

// TestBytecodeParity_Live pins the claim this package rests on: each CooCoo hook's runtime
// code equals Pons' hook's, once the compiler metadata trailer is dropped and the three
// immutable references to the fee escrow are zeroed. Anything else differing would mean a
// different contract, and hooks/pons-v2's pricing could not be assumed.
func TestBytecodeParity_Live(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	ctx := context.Background()

	pons := normalizedCode(t, ctx, ponsRPC, ponsv2.HookAddresses[0], ponsEscrow)
	for _, d := range deployments {
		ours := normalizedCode(t, ctx, d.rpc, d.hook, d.escrow)
		assert.Equal(t, len(pons), len(ours), "%s: code length", d.name)
		assert.True(t, bytes.Equal(pons, ours),
			"%s: runtime code differs from Pons' hook beyond the escrow immutables and metadata", d.name)
	}
}

// normalizedCode is a hook's runtime code without its CBOR metadata trailer and with every
// occurrence of its fee-escrow address zeroed, so two deployments of one compilation compare
// equal. It also requires exactly three escrow references, PonsV2MemeHook's immutable count.
func normalizedCode(t *testing.T, ctx context.Context, url string, hook, escrow common.Address) []byte {
	rpcClient, err := rpc.DialOptions(ctx, url, rpc.WithHeader("User-Agent", "kyberswap-dex-lib live test"))
	require.NoError(t, err)
	code, err := ethclient.NewClient(rpcClient).CodeAt(ctx, hook, nil)
	require.NoError(t, err)
	require.Greater(t, len(code), 2, "%s: no code", hook)

	metadataLen := int(code[len(code)-2])<<8 | int(code[len(code)-1])
	require.Less(t, metadataLen+2, len(code), "%s: metadata trailer longer than code", hook)
	code = code[:len(code)-metadataLen-2]

	require.Equal(t, 3, bytes.Count(code, escrow.Bytes()), "%s: immutable escrow references", hook)
	return bytes.ReplaceAll(code, escrow.Bytes(), make([]byte, common.AddressLength))
}

// TestTrack_Live decodes `launches` against both CooCoo hooks, confirming hooks/pons-v2's
// ABI matches these deployments too. No CooCoo launch has graduated yet, so the pool id is
// one the hook has never seen: the getter answers, and Extra stays unregistered.
func TestTrack_Live(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	for _, d := range deployments {
		rpcClient := ethrpc.New(d.rpc).SetMulticallContract(common.HexToAddress(multicall3))
		h := &Hook{Hook: &uniswapv4.BaseHook{}}
		raw, err := h.Track(context.Background(), &uniswapv4.HookParam{
			RpcClient:   rpcClient,
			HookAddress: d.hook,
			Pool:        &entity.Pool{Address: crypto.Keccak256Hash([]byte("coocoo: never registered")).Hex()},
		})
		require.NoError(t, err, d.name)
		assert.False(t, h.Registered, d.name)
		assert.Equal(t, int64(0), h.FeeBps, d.name)
		assert.JSONEq(t, `{}`, string(raw), d.name)
	}
}
