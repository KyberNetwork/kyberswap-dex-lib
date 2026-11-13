package doppleramm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func marketCreatedLog(t *testing.T, module, extension common.Address, marketID *big.Int) types.Log {
	t.Helper()
	event := ammABI.Events["MarketCreated"]
	data, err := event.Inputs.NonIndexed().Pack(
		marketID,
		common.HexToAddress("0x00000000000000000000000000000000000000c0"),
		common.HexToAddress("0x0000000000000000000000000000000000000011"),
		common.HexToAddress("0x0000000000000000000000000000000000000022"),
		big.NewInt(1e18),
		big.NewInt(7),
		big.NewInt(8),
		big.NewInt(1e18),
		bi("2000000000000000000000"),
	)
	require.NoError(t, err)
	creator := common.HexToAddress("0x00000000000000000000000000000000000000d0")
	return types.Log{
		Address: common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		Topics: []common.Hash{
			event.ID,
			common.BytesToHash(common.LeftPadBytes(module[:], 32)),
			common.BytesToHash(common.LeftPadBytes(extension[:], 32)),
			common.BytesToHash(common.LeftPadBytes(creator[:], 32)),
		},
		Data:        data,
		BlockNumber: 100,
	}
}

func TestDecodeMarketCreatedKeepsPlainSpotMarkets(t *testing.T) {
	cfg := &Config{DexID: "doppler-amm", ChainID: 8453}
	require.NoError(t, cfg.withDefaults())
	u := NewPoolsListUpdater(cfg, nil)

	marketID := big.NewInt(0x1234)
	spot := common.HexToAddress(cfg.SpotModuleAddress)

	pool, ok, err := u.decodeMarketCreated(marketCreatedLog(t, spot, common.Address{}, marketID))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, common.BigToHash(marketID).Hex(), pool.Address)
	require.Len(t, pool.Tokens, 2)
	require.Equal(t, "0x0000000000000000000000000000000000000011", pool.Tokens[0].Address)
	require.Equal(t, "0x0000000000000000000000000000000000000022", pool.Tokens[1].Address)
	require.Equal(t, DexType, pool.Type)
	require.JSONEq(t, `{"tokenX":"0x0000000000000000000000000000000000000011","baseFeeBps":0,"lpFeeBps":0}`, pool.Extra)
}

// A market with an extension, or one on another module, can require component
// payloads that are not derivable from state. Quoting it with empty payloads
// would silently misprice, so discovery must drop it.
func TestDecodeMarketCreatedRejectsUnsupportedComposition(t *testing.T) {
	cfg := &Config{DexID: "doppler-amm", ChainID: 8453}
	require.NoError(t, cfg.withDefaults())
	u := NewPoolsListUpdater(cfg, nil)

	spot := common.HexToAddress(cfg.SpotModuleAddress)
	_, ok, err := u.decodeMarketCreated(marketCreatedLog(t, spot, common.HexToAddress("0x00000000000000000000000000000000000000ee"), big.NewInt(1)))
	require.NoError(t, err)
	require.False(t, ok, "market with an extension must be skipped")

	_, ok, err = u.decodeMarketCreated(marketCreatedLog(t, common.HexToAddress("0x00000000000000000000000000000000000000ff"), common.Address{}, big.NewInt(2)))
	require.NoError(t, err)
	require.False(t, ok, "market on a non-CPMM module must be skipped")
}

func TestConfigDefaultsOnlyKnownDeployments(t *testing.T) {
	cfg := &Config{DexID: "doppler-amm", ChainID: 84532}
	require.NoError(t, cfg.withDefaults())
	require.Equal(t, deployments[84532].AMM, cfg.AMMAddress)
	require.Equal(t, deployments[84532].Quoter, cfg.QuoterAddress)
	require.Equal(t, deployments[84532].StartBlock, cfg.StartBlock)
	require.NotZero(t, cfg.MaxBlockRange)

	unknown := &Config{DexID: "doppler-amm", ChainID: 1}
	require.ErrorIs(t, unknown.withDefaults(), ErrUnsupportedChain)
}

// A launch source lists only LaunchModule + FeesLocker markets; any other extension
// (for example the fee-decay one) can change the quote and is dropped. Spot markets
// are not listed by it, and launch markets are not listed by the spot source.
func TestDecodeMarketCreatedLaunchKind(t *testing.T) {
	cfg := &Config{DexID: "doppler-fun", ChainID: 8453, MarketKind: MarketKindLaunch}
	require.NoError(t, cfg.withDefaults())
	u := NewPoolsListUpdater(cfg, nil)
	launch, locker := common.HexToAddress(cfg.LaunchModuleAddress), common.HexToAddress(cfg.FeesLockerAddress)

	pool, ok, err := u.decodeMarketCreated(marketCreatedLog(t, launch, locker, big.NewInt(7)))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "doppler-fun", pool.Exchange)
	require.Contains(t, pool.StaticExtra, `"module":"`+cfg.LaunchModuleAddress+`"`)

	_, ok, _ = u.decodeMarketCreated(marketCreatedLog(t, launch, common.HexToAddress("0x00000000000000000000000000000000000000ee"), big.NewInt(8)))
	require.False(t, ok, "other extension")
	_, ok, _ = u.decodeMarketCreated(marketCreatedLog(t, common.HexToAddress(cfg.SpotModuleAddress), common.Address{}, big.NewInt(9)))
	require.False(t, ok, "spot market")

	spotCfg := &Config{DexID: "doppler-amm", ChainID: 8453}
	require.NoError(t, spotCfg.withDefaults())
	_, ok, _ = NewPoolsListUpdater(spotCfg, nil).decodeMarketCreated(marketCreatedLog(t, launch, locker, big.NewInt(7)))
	require.False(t, ok, "spot source must not list launch markets")
}

// An unknown module is skipped and reported once per module/extension pair, not once per market.
func TestDecodeMarketCreatedLogsUnknownModuleOnce(t *testing.T) {
	cfg := &Config{DexID: "doppler-fun", ChainID: 8453, MarketKind: MarketKindLaunch}
	require.NoError(t, cfg.withDefaults())
	u := NewPoolsListUpdater(cfg, nil)
	unknown := common.HexToAddress("0x00000000000000000000000000000000000000ff")

	for id := int64(1); id <= 3; id++ {
		_, ok, err := u.decodeMarketCreated(marketCreatedLog(t, unknown, common.Address{}, big.NewInt(id)))
		require.NoError(t, err)
		require.False(t, ok)
	}
	n := 0
	u.warned.Range(func(_, _ any) bool { n++; return true })
	require.Equal(t, 1, n)
}
