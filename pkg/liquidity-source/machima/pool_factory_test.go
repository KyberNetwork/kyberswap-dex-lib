package machima

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFactoryConfig() *Config {
	return &Config{
		DexID:         DexType,
		RouterAddress: routerAddr,
		WETH:          wethAddr,
		USDC:          usdcAddr,
		XMA:           xmaAddr,
	}
}

// poolCreatedLog is a real Machima PoolCreated log from Base (factory
// 0xadd30837a707cce4567eea2c27d0617270d54c75, tx 0xa017d159…3839, block 0x2e4107f):
// token0 = WETH, token1 = the launched token, fee = 0x2710 (1%), tickSpacing = 0xc8 (200).
func poolCreatedLog() ethtypes.Log {
	data, _ := hexutil.Decode(
		"0x00000000000000000000000000000000000000000000000000000000000000c8" +
			"000000000000000000000000d4829d181e93059ae602ce5a5b59ff4d6736a4a8")
	return ethtypes.Log{
		Address: common.HexToAddress("0xadd30837a707cce4567eea2c27d0617270d54c75"),
		Topics: []common.Hash{
			common.HexToHash("0x783cca1c0412dd0d695e784568c96da2e9c22ff989357a2e8b1d9b2b4e6b7118"),
			common.HexToHash("0x0000000000000000000000004200000000000000000000000000000000000006"),
			common.HexToHash("0x0000000000000000000000008f49e45a2f08aa7aee54d9453944fed1e2841975"),
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000002710"),
		},
		Data:        data,
		BlockNumber: 0x2e4107f,
	}
}

// TestDecodePoolCreated proves the UniV3 factory ABI really does decode a Machima PoolCreated log,
// which is what lets this package reuse it instead of shipping its own factory ABI.
func TestDecodePoolCreated(t *testing.T) {
	f := NewPoolFactory(testFactoryConfig())

	log := poolCreatedLog()
	require.True(t, f.IsEventSupported(log.Topics[0]), "PoolCreated topic must be recognised")

	p, err := f.DecodePoolCreated(log)
	require.NoError(t, err)

	assert.Equal(t, "0xd4829d181e93059ae602ce5a5b59ff4d6736a4a8", p.Address)
	assert.Equal(t, DexType, p.Exchange)
	assert.Equal(t, DexType, p.Type)
	assert.Equal(t, float64(defaultFee), p.SwapFee, "fee 0x2710 is the 1%% tier")
	assert.Equal(t, uint64(0x2e4107f), p.BlockNumber)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)

	require.Len(t, p.Tokens, 2)
	assert.Equal(t, wethAddr, p.Tokens[0].Address)
	assert.Equal(t, "0x8f49e45a2f08aa7aee54d9453944fed1e2841975", p.Tokens[1].Address)

	// tickSpacing must be seeded, otherwise the tracker's first tick sweep has nothing to scan.
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	assert.Equal(t, defaultTickSpacing, extra.TickSpacing)

	// WETH is a counter asset, so the other side is the launched token.
	var staticExtra StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &staticExtra))
	assert.Equal(t, "0x8f49e45a2f08aa7aee54d9453944fed1e2841975", staticExtra.Token)
	assert.Equal(t, routerAddr, staticExtra.RouterAddress)
	assert.Equal(t, xmaAddr, staticExtra.XMA)
}

// TestResolveTradedToken covers the classification the event itself does not carry.
func TestResolveTradedToken(t *testing.T) {
	f := NewPoolFactory(testFactoryConfig())
	const launched = "0x8f49e45a2f08aa7aee54d9453944fed1e2841975"

	for _, tc := range []struct {
		name           string
		token0, token1 string
		want           string
		ok             bool
	}{
		{"counter first", wethAddr, launched, launched, true},
		{"counter second", launched, usdcAddr, launched, true},
		{"XMA is traded against WETH", wethAddr, xmaAddr, xmaAddr, true},
		{"XMA is traded, ordered first", xmaAddr, usdcAddr, xmaAddr, true},
		{"neither side is a counter asset", launched, "0xdead", "", false},
		{"two counter assets, neither is XMA", wethAddr, usdcAddr, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := f.resolveTradedToken(tc.token0, tc.token1)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestDecodePoolCreatedRejectsUnclassifiablePair keeps a pool the router could never route from
// entering the pool list.
func TestDecodePoolCreatedRejectsUnclassifiablePair(t *testing.T) {
	f := NewPoolFactory(testFactoryConfig())

	log := poolCreatedLog()
	// WETH/USDC: both counter assets, neither is XMA.
	log.Topics[2] = common.HexToHash("0x000000000000000000000000833589fcd6edb6e08f4c7c32d4f71b54bda02913")

	_, err := f.DecodePoolCreated(log)
	assert.ErrorIs(t, err, ErrInvalidPair)
}

// TestMetadataMatchesTicksBasedCheckpoint pins the checkpoint wire format. The ticks-based
// bootstrap persists its own {offset,lastCreatedAtTimestamp} over whatever the lister returns and
// feeds that back in, so a different field name here means the cursor silently never loads.
func TestMetadataMatchesTicksBasedCheckpoint(t *testing.T) {
	// Exactly what pool-service wrote to Redis for machima.
	const persisted = `{"offset":1783793595,"lastCreatedAtTimestamp":1783793595}`

	var m Metadata
	require.NoError(t, json.Unmarshal([]byte(persisted), &m))
	require.NotNil(t, m.LastCreatedAtTimestamp, "checkpoint must load, otherwise every bootstrap rescans from 0")
	assert.Equal(t, "1783793595", m.LastCreatedAtTimestamp.String())

	// And what we emit must be readable back by the same shape.
	out, err := json.Marshal(Metadata{LastCreatedAtTimestamp: big.NewInt(42)})
	require.NoError(t, err)
	var back Metadata
	require.NoError(t, json.Unmarshal(out, &back))
	assert.Equal(t, "42", back.LastCreatedAtTimestamp.String())
}
