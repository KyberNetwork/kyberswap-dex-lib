package synthereum

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseFinder    = "0x3b05B902Fe763AD87Aa755Fab70F86c76Bf331F4"
	baseMulticall = "0xcA11bde05977b3631167028862bE2a173976CA11"

	jEURAddr = "0x4154550f4db74dc38d1fe98e1f3f28ed6dad627d"
	usdcAddr = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
	eurcAddr = "0x60a3e35cc302bfa44cb288bc5a4f316fdb1adb42"
)

func TestGetNewPools_RequiresFinder(t *testing.T) {
	t.Parallel()

	for _, finder := range []string{"", "not-an-address", "0x123"} {
		u := NewPoolsListUpdater(&Config{DexID: DexType, Finder: finder}, nil)
		_, _, err := u.GetNewPools(context.Background(), nil)
		assert.ErrorIs(t, err, ErrMissingFinder, "finder %q", finder)
	}
}

// interfaceName must reproduce Solidity's bytes32 encoding of a short string
// literal (raw ASCII, left-aligned, zero-padded) -- not keccak256 of the name.
// A wrong encoding resolves the Finder to the zero address and silently discovers
// nothing, so pin the exact bytes.
func TestInterfaceNameEncoding(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		"0x5072696365466565640000000000000000000000000000000000000000000000",
		hexOf(interfaceName("PriceFeed")))
	assert.Equal(t,
		"0x506f6f6c52656769737472790000000000000000000000000000000000000000",
		hexOf(registriesByPoolType[PoolTypeMultiLP]))
	assert.Equal(t,
		"0x4669786564526174655265676973747279000000000000000000000000000000",
		hexOf(registriesByPoolType[PoolTypeWrapper]))
}

func hexOf(b [32]byte) string {
	return "0x" + common.Bytes2Hex(b[:])
}

func TestConfig_UnmarshalsFromProperties(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(map[string]any{"dexID": "synthereum", "finder": baseFinder})
	require.NoError(t, err)

	var cfg Config
	require.NoError(t, json.Unmarshal(raw, &cfg))
	assert.Equal(t, "synthereum", cfg.DexID)
	assert.Equal(t, baseFinder, cfg.Finder)
}

// Live discovery against Base. Opt-in: set SYNTHEREUM_RPC to run it.
func TestGetNewPools_LiveBase(t *testing.T) {
	rpcURL := os.Getenv("SYNTHEREUM_RPC")
	if rpcURL == "" {
		t.Skip("set SYNTHEREUM_RPC to run live discovery")
	}

	client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	rpcClient := ethrpc.NewWithClient(client).
		SetMulticallContract(common.HexToAddress(baseMulticall))

	u := NewPoolsListUpdater(&Config{DexID: DexType, Finder: baseFinder}, rpcClient)
	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)

	byAddress := make(map[string]struct {
		poolType string
		vault    string
		tokens   []string
	}, len(pools))
	for _, p := range pools {
		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))

		// discovery must not invent reserves or token metadata
		assert.Equal(t, []string{reserveZero, reserveZero}, []string(p.Reserves), p.Address)
		require.Len(t, p.Tokens, 2, p.Address)
		for _, tok := range p.Tokens {
			assert.True(t, tok.Swappable)
			assert.Empty(t, tok.Symbol)
			assert.Zero(t, tok.Decimals)
			assert.Equal(t, strings.ToLower(tok.Address), tok.Address,
				"token address must be lowercased")
		}

		byAddress[p.Address] = struct {
			poolType string
			vault    string
			tokens   []string
		}{se.PoolType, se.Vault, []string{p.Tokens[0].Address, p.Tokens[1].Address}}
	}

	// the two pools the integration originally shipped, now discovered rather than configured
	multiLp, ok := byAddress["0x67aefc812ec0a83a327c05d6e7913c35b48bfb94"]
	require.True(t, ok, "jEUR/USDC multi-lp pool must be discovered")
	assert.Equal(t, PoolTypeMultiLP, multiLp.poolType)
	assert.Equal(t, []string{usdcAddr, jEURAddr}, multiLp.tokens)
	assert.Empty(t, multiLp.vault)

	wrapper, ok := byAddress["0x41b0667ea45a5401d95f9a5d281287630704b798"]
	require.True(t, ok, "jEUR/EURC wrapper must be discovered")
	assert.Equal(t, PoolTypeWrapper, wrapper.poolType)
	assert.Equal(t, []string{eurcAddr, jEURAddr}, wrapper.tokens)
	assert.Equal(t, "0xbeef086b8807dc5e5a1740c5e3a7c4c366ea6ab5", wrapper.vault,
		"vault must come from lendingModule(), not config")

	// the pool the hardcoded list missed
	jbrl, ok := byAddress["0xd1e358d4f3157b86b6bac7447318c7ee7402e8a5"]
	require.True(t, ok, "jBRL/USDC pool must be discovered")
	assert.Equal(t, PoolTypeMultiLP, jbrl.poolType)
	assert.Equal(t, usdcAddr, jbrl.tokens[0])
}
