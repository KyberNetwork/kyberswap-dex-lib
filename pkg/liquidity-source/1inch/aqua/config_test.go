package oneinchaqua

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/ethereum/go-ethereum/common"
	gojson "github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

func TestConfigSettings(t *testing.T) {
	t.Parallel()
	zero := uint64(0)
	for name, tc := range map[string]struct {
		mutate func(c *Config)
		want   string
	}{
		"missing aqua":          {func(c *Config) { c.AquaAddress = "" }, "aquaAddress is required"},
		"missing router":        {func(c *Config) { c.RouterAddress = "" }, "routerAddress is required"},
		"zero aqua":             {func(c *Config) { c.AquaAddress = "0x0000000000000000000000000000000000000000" }, "zero address"},
		"zero router":           {func(c *Config) { c.RouterAddress = "0x0000000000000000000000000000000000000000" }, "zero address"},
		"malformed aqua":        {func(c *Config) { c.AquaAddress = "0x1234" }, "not an address"},
		"malformed router":      {func(c *Config) { c.RouterAddress = "nope" }, "not an address"},
		"malformed launch":      {func(c *Config) { c.LaunchRouterAddress = "0xzz" }, "not an address"},
		"zero launch":           {func(c *Config) { c.LaunchRouterAddress = "0x0000000000000000000000000000000000000000" }, "zero address"},
		"missing startBlock":    {func(c *Config) { c.StartBlock = 0 }, "startBlock is required"},
		"window zero":           {func(c *Config) { c.LogWindow = 0 }, "logWindow is required"},
		"no windows per run":    {func(c *Config) { c.MaxWindowsPerRun = 0 }, "maxWindowsPerRun is required"},
		"negative windows":      {func(c *Config) { c.MaxWindowsPerRun = -1 }, "maxWindowsPerRun is required"},
		"missing confirmations": {func(c *Config) { c.Confirmations = nil }, "confirmations is required"},
		"launch is router":      {func(c *Config) { c.LaunchRouterAddress = strings.ToUpper(testRouterHex[:2]) + testRouterHex[2:] }, "equals routerAddress"},
		"launch is aqua":        {func(c *Config) { c.LaunchRouterAddress = testAquaHex }, "equals aquaAddress"},
		"excluded malformed":    {func(c *Config) { c.ExcludedTokens = []string{"0x12"} }, "not an address"},
		"excluded not hex":      {func(c *Config) { c.ExcludedTokens = []string{"dai"} }, "not an address"},
		"excluded zero":         {func(c *Config) { c.ExcludedTokens = []string{"0x0000000000000000000000000000000000000000"} }, "zero address"},
	} {
		cfg := testConfig(tc.mutate)
		_, err := cfg.settings()
		require.ErrorContains(t, err, tc.want, name)
		_, err = NewPoolsListUpdater(cfg, nil)
		require.ErrorContains(t, err, tc.want, name)
	}
	// The tracker and the simulator need only the addresses.
	for name, mutate := range map[string]func(c *Config){
		"missing aqua": func(c *Config) { c.AquaAddress = "" }, "missing router": func(c *Config) { c.RouterAddress = "" },
		"zero aqua": func(c *Config) { c.AquaAddress = "0x0000000000000000000000000000000000000000" }, "malformed launch": func(c *Config) { c.LaunchRouterAddress = "x" },
	} {
		_, err := NewPoolTracker(testConfig(mutate), nil)
		require.Error(t, err, name)
	}
	_, err := NewPoolTracker(testConfig(func(c *Config) { c.StartBlock, c.LogWindow, c.MaxWindowsPerRun = 0, 0, 0 }), nil)
	require.NoError(t, err, "the tracker does not read the lister window tunables")
	_, err = NewPoolTracker(testConfig(func(c *Config) { c.Confirmations = nil }), nil)
	require.ErrorContains(t, err, "confirmations is required", "the tracker needs the depth after which a dock is final")

	cfg := testConfig(func(c *Config) {
		c.AquaAddress, c.RouterAddress, c.LaunchRouterAddress = tokenA, tokenB, tokenC
		c.StartBlock, c.LogWindow, c.MaxWindowsPerRun, c.Confirmations = 7, 11, 3, &zero
		c.DexId = ""
		c.ExcludedTokens = []string{"0xAbC0000000000000000000000000000000000001", "0xabc0000000000000000000000000000000000001"}
	})
	s, err := cfg.settings()
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress(tokenA), s.aqua)
	require.Equal(t, common.HexToAddress(tokenB), s.router)
	require.True(t, s.hasLaunch)
	require.Equal(t, common.HexToAddress(tokenC), s.launch)
	require.Equal(t, uint64(7), s.startBlock)
	require.Equal(t, uint64(11), s.logWindow)
	require.Equal(t, 3, s.maxWindows)
	require.Equal(t, uint64(0), s.confirmations, "an explicit 0 is kept")
	require.Equal(t, map[common.Address]struct{}{common.HexToAddress("0xabc0000000000000000000000000000000000001"): {}}, s.excluded,
		"case-only duplicates are one token")
	noLaunch, err := testConfig().settings()
	require.NoError(t, err)
	require.False(t, noLaunch.hasLaunch)
	require.Empty(t, noLaunch.excluded, "excludedTokens is optional")
}

// Config validation: required fields, https only, and a missing key is a warning.
func TestPortalConfig(t *testing.T) {
	fp := &fakePortal{Server: &httptest.Server{URL: "https://portal.example"}}
	ok := portalConfig(fp, func(c *Config) {
		c.AquaAddress, c.RouterAddress = testAquaHex, testRouterHex
		c.Confirmations = new(uint64)
	})
	_, err := ok.settings()
	require.NoError(t, err)
	zero := 0
	neg := -1
	for name, tc := range map[string]struct {
		mutate func(*Config)
		want   string
	}{
		"http":             {func(c *Config) { c.PortalURL = "http://portal.example" }, "must be https"},
		"no scheme":        {func(c *Config) { c.PortalURL = "portal.example" }, "not a URL"},
		"no host":          {func(c *Config) { c.PortalURL = "https://" }, "not a URL"},
		"credentials":      {func(c *Config) { c.PortalURL = "https://user:pw@portal.example" }, "without credentials"},
		"query":            {func(c *Config) { c.PortalURL = "https://portal.example/?key=x" }, "without credentials, query"},
		"fragment":         {func(c *Config) { c.PortalURL = "https://portal.example/#x" }, "without credentials, query"},
		"no chain":         {func(c *Config) { c.ChainId = 0 }, "chainId is required"},
		"no page limit":    {func(c *Config) { c.PortalPageLimit = 0 }, "portalPageLimit"},
		"negative limit":   {func(c *Config) { c.PortalPageLimit = -1 }, "portalPageLimit"},
		"no max pages":     {func(c *Config) { c.PortalMaxPages = 0 }, "portalMaxPages"},
		"no timeout":       {func(c *Config) { c.PortalTimeout = durationjson.Duration{} }, "portalTimeout"},
		"no retries":       {func(c *Config) { c.PortalMaxRetries = nil }, "portalMaxRetries"},
		"negative retries": {func(c *Config) { c.PortalMaxRetries = &neg }, "portalMaxRetries"},
		"no backoff":       {func(c *Config) { c.PortalRetryBackoff = durationjson.Duration{} }, "portalRetryBackoff"},
	} {
		cfg := portalConfig(fp, tc.mutate)
		cfg.AquaAddress, cfg.RouterAddress = testAquaHex, testRouterHex
		confirm := uint64(0)
		cfg.Confirmations = &confirm
		_, err := cfg.settings()
		require.ErrorContains(t, err, tc.want, name)
		_, err = NewPoolsListUpdater(&cfg, nil)
		require.ErrorContains(t, err, tc.want, name)
	}
	// Zero retries need no backoff.
	cfg := portalConfig(fp, func(c *Config) { c.PortalMaxRetries = &zero; c.PortalRetryBackoff = durationjson.Duration{} })
	cfg.AquaAddress, cfg.RouterAddress = testAquaHex, testRouterHex
	confirm := uint64(0)
	cfg.Confirmations = &confirm
	s, err := cfg.settings()
	require.NoError(t, err)
	require.NotNil(t, s.portal)
	require.Zero(t, s.portal.retries)

	// URL and no key: not a failure, the Portal is off and the missing key is flagged.
	cfg.PortalApiKey = ""
	s, err = cfg.settings()
	require.NoError(t, err)
	require.Nil(t, s.portal)
	require.True(t, s.portalNoKey)
	// No URL: off, nothing flagged, and the other portal fields are not read.
	cfg = *testConfig()
	cfg.PortalPageLimit = -5
	s, err = cfg.settings()
	require.NoError(t, err)
	require.Nil(t, s.portal)
	require.False(t, s.portalNoKey)
	// The tracker never reads the Portal fields.
	bad := portalConfig(fp, func(c *Config) { c.PortalURL = "http://x" })
	bad.AquaAddress, bad.RouterAddress, bad.Confirmations = testAquaHex, testRouterHex, &confirm
	_, err = NewPoolTracker(&bad, nil)
	require.NoError(t, err)
	// the key is trimmed
	cfg = portalConfig(fp, func(c *Config) { c.PortalApiKey = "  " + testPortalKey + "\n" })
	cfg.AquaAddress, cfg.RouterAddress, cfg.Confirmations = testAquaHex, testRouterHex, &confirm
	s, err = cfg.settings()
	require.NoError(t, err)
	require.Equal(t, testPortalKey, s.portal.key.value)
	require.Equal(t, "https://portal.example", s.portal.baseURL)
	cfg.PortalURL = "https://portal.example/"
	s, err = cfg.settings()
	require.NoError(t, err)
	require.Equal(t, "https://portal.example", s.portal.baseURL, "a trailing slash is dropped")
}

// The properties arrive as a map with lower-cased keys and the timeouts as strings; they decode into
// the Config.
func TestPortalConfigFromProperties(t *testing.T) {
	props := map[string]any{
		"aquaaddress": testAquaHex, "routeraddress": testRouterHex, "startblock": 1, "logwindow": 50000, "maxwindowsperrun": 20,
		"confirmations": 12, "chainid": 1, "portalurl": "https://api.1inch.com", "portalapikey": "k", "portalpagelimit": 100,
		"portalmaxpages": 60, "portaltimeout": "10s", "portalmaxretries": 3, "portalretrybackoff": "2s",
		"excludedtokens": []any{tokenA},
	}
	b, err := json.Marshal(props)
	require.NoError(t, err)
	var c Config
	require.NoError(t, gojson.Unmarshal(b, &c))
	s, err := c.settings()
	require.NoError(t, err)
	require.NotNil(t, s.portal)
	require.Equal(t, 100, s.portal.pageLimit)
	require.Equal(t, 60, s.portal.maxPages)
	require.Equal(t, 10*time.Second, s.portal.timeout)
	require.Equal(t, 3, s.portal.retries)
	require.Equal(t, 2*time.Second, s.portal.backoff)
	require.Equal(t, uint64(1), s.portal.chainId)
	require.Contains(t, s.excluded, common.HexToAddress(tokenA))
}
