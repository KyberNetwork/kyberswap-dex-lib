package doppleramm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

// Base deployment addresses, fed in as config the way pool-service yaml does.
const (
	testAMM          = "0x533b7ee11f4f9be09191ebf6667abe73c01fe2c2"
	testQuoter       = "0x384e4c09071f7e511be34e38b4fcba23be51ea55"
	testSpotModule   = "0x3d17adc45d3c40735e7fdd271710f90d84c1b838"
	testLaunchModule = "0xf2e28140520de30de998ea25f46923bb7bebd909"
	testFeesLocker   = "0x05cfa684b6f28c3dfddcd6f9b361c8f90cd09ead"
)

func testConfig() *Config {
	return &Config{AMMAddress: testAMM, QuoterAddress: testQuoter, SpotModuleAddress: testSpotModule,
		LaunchModuleAddress: testLaunchModule, FeesLockerAddress: testFeesLocker}
}

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
		Address: common.HexToAddress(testAMM),
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

func newTestFactory(t *testing.T) (*PoolFactory, *Config) {
	t.Helper()
	cfg := testConfig()
	f, err := NewPoolFactory(poolfactory.PoolsDecoderParams[Config]{Cfg: cfg})
	require.NoError(t, err)
	return f, cfg
}

func TestDecodeMarketCreatedKeepsPlainSpotMarkets(t *testing.T) {
	f, cfg := newTestFactory(t)
	marketID := big.NewInt(0x1234)
	spot := common.HexToAddress(cfg.SpotModuleAddress)

	pool, err := f.DecodePoolCreated(marketCreatedLog(t, spot, common.Address{}, marketID))
	require.NoError(t, err)
	require.NotNil(t, pool)
	require.Equal(t, common.BigToHash(marketID).Hex(), pool.Address)
	require.Len(t, pool.Tokens, 2)
	require.Equal(t, "0x0000000000000000000000000000000000000011", pool.Tokens[0].Address)
	require.Equal(t, "0x0000000000000000000000000000000000000022", pool.Tokens[1].Address)
	require.Equal(t, DexType, pool.Type)
	require.Equal(t, "doppler-amm", pool.Exchange)
	require.JSONEq(t, `{"tokenX":"0x0000000000000000000000000000000000000011","baseFeeBps":0,"lpFeeBps":0}`, pool.Extra)
}

// A market with an extension, or one on another module, can require component
// payloads that are not derivable from state. Quoting it with empty payloads
// would silently misprice, so discovery must drop it (nil, nil: an error would
// make a backfill retry the same window forever).
func TestDecodeMarketCreatedRejectsUnsupportedComposition(t *testing.T) {
	f, cfg := newTestFactory(t)
	spot := common.HexToAddress(cfg.SpotModuleAddress)
	ext := common.HexToAddress("0x00000000000000000000000000000000000000ee")

	pool, err := f.DecodePoolCreated(marketCreatedLog(t, spot, ext, big.NewInt(1)))
	require.NoError(t, err)
	require.Nil(t, pool, "market with an extension must be skipped")

	pool, err = f.DecodePoolCreated(marketCreatedLog(t, common.HexToAddress("0x00000000000000000000000000000000000000ff"), common.Address{}, big.NewInt(2)))
	require.NoError(t, err)
	require.Nil(t, pool, "market on a non-CPMM module must be skipped")
}

// Every address is required and validated: no chain defaults, so a yaml typo or omission must
// fail at construction instead of silently decoding against the wrong contract.
func TestConfigRequiresEveryAddress(t *testing.T) {
	mutate := map[string]func(*Config, string){
		"ammAddress":          func(c *Config, v string) { c.AMMAddress = v },
		"quoterAddress":       func(c *Config, v string) { c.QuoterAddress = v },
		"spotModuleAddress":   func(c *Config, v string) { c.SpotModuleAddress = v },
		"launchModuleAddress": func(c *Config, v string) { c.LaunchModuleAddress = v },
		"feesLockerAddress":   func(c *Config, v string) { c.FeesLockerAddress = v },
	}
	for name, set := range mutate {
		for _, bad := range []string{"", "0x1234", "not-an-address", "0x0000000000000000000000000000000000000000"} {
			cfg := testConfig()
			set(cfg, bad)
			_, err := NewPoolFactory(poolfactory.PoolsDecoderParams[Config]{Cfg: cfg})
			require.ErrorIs(t, err, ErrInvalidConfig, "%s=%q", name, bad)
			_, err = NewPoolTracker(cfg, nil)
			require.ErrorIs(t, err, ErrInvalidConfig, "%s=%q", name, bad)
		}
	}
	cfg := testConfig()
	cfg.AMMAddress = "0x533B7ee11f4F9be09191ebf6667abe73c01fe2C2"
	_, err := NewPoolFactory(poolfactory.PoolsDecoderParams[Config]{Cfg: cfg})
	require.NoError(t, err)
	require.Equal(t, testAMM, cfg.AMMAddress, "lowercased")
}

// One decoder lists both kinds and sets Exchange by module. pool-service keeps one decoder per
// factory address, so a per-kind decoder would list only one kind live (whichever registered last).
func TestDecodeMarketCreatedListsBothKindsByModule(t *testing.T) {
	f, cfg := newTestFactory(t)
	launch, locker := common.HexToAddress(cfg.LaunchModuleAddress), common.HexToAddress(cfg.FeesLockerAddress)

	pool, err := f.DecodePoolCreated(marketCreatedLog(t, launch, locker, big.NewInt(7)))
	require.NoError(t, err)
	require.NotNil(t, pool)
	require.Equal(t, "doppler-fun", pool.Exchange)
	require.Equal(t, DexType, pool.Type)
	require.Contains(t, pool.StaticExtra, `"module":"`+cfg.LaunchModuleAddress+`"`)

	pool, err = f.DecodePoolCreated(marketCreatedLog(t, launch, common.HexToAddress("0x00000000000000000000000000000000000000ee"), big.NewInt(8)))
	require.NoError(t, err)
	require.Nil(t, pool, "other extension")

	pool, err = f.DecodePoolCreated(marketCreatedLog(t, common.HexToAddress(cfg.SpotModuleAddress), common.Address{}, big.NewInt(9)))
	require.NoError(t, err)
	require.Equal(t, "doppler-amm", pool.Exchange)
}

func TestSupportedEventTopicsIsMarketCreated(t *testing.T) {
	f, _ := newTestFactory(t)
	id := ammABI.Events["MarketCreated"].ID
	require.Equal(t, []common.Hash{id}, f.SupportedEventTopics())
	require.True(t, f.IsEventSupported(id))
	require.False(t, f.IsEventSupported(common.HexToHash("0x07af4e23b8e693b486c551709747254c747c07d7f9b3930a72bb6b0c3ac5af83")))
}

// Listers must not scan blocks themselves: pools come from the poolfactory decoder via the
// consumer's backfill config. A polling lister registered for this source would bring back the
// per-tick FilterLogs cursor scan.
func TestNoManualBlockScanLister(t *testing.T) {
	require.Nil(t, poollist.Factory(DexType), "doppler-amm must not register a pools list updater")
	require.NotNil(t, poolfactory.Factory(DexType))
}
