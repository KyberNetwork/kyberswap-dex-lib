package oneinchaqua

import (
	"net/url"
	"strings"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config is the deployment properties of the source, one set per chain. Every field that differs by chain or by
// operator choice comes from there; the constructors refuse a config that leaves one out.
type Config struct {
	DexId   string              `json:"dexID"`
	ChainId valueobject.ChainID `json:"chainId"`

	// AquaAddress and RouterAddress are required. LaunchRouterAddress is optional: it is set only on the
	// chains where the launch router has code, and a strategy whose app is the launch address is refused
	// everywhere else.
	AquaAddress         string `json:"aquaAddress"`
	RouterAddress       string `json:"routerAddress"`
	LaunchRouterAddress string `json:"launchRouterAddress"`

	// StartBlock is the first block the lister scans (required, above zero). LogWindow is the largest
	// eth_getLogs range the chain's nodes serve and MaxWindowsPerRun bounds the windows one GetNewPools call
	// scans (both required, above zero). Confirmations is how many blocks the cursor stays behind head
	// (required; 0 on chains with instant finality).
	StartBlock       uint64  `json:"startBlock"`
	LogWindow        uint64  `json:"logWindow"`
	MaxWindowsPerRun int     `json:"maxWindowsPerRun"`
	Confirmations    *uint64 `json:"confirmations"`

	// ExcludedTokens is optional: a strategy with a listed token is not admitted.
	ExcludedTokens []string `json:"excludedTokens"`

	// Portal discovery (lister only). HTTPConfig is the Portal API client; an empty base_url means the
	// Portal is not used and discovery is the on-chain log scan alone. When it is set, the paging fields are
	// required too: PortalPageLimit (items per request) and PortalMaxPages (a sweep that needs more pages
	// fails and falls back to logs).
	HTTPConfig      HTTPConfig `json:"http_config"`
	PortalPageLimit int        `json:"portalPageLimit"`
	PortalMaxPages  int        `json:"portalMaxPages"`
}

// HTTPConfig is the 1inch Business Portal API client. With BaseURL set ("https://...") every other field is
// required: Timeout (per request), RetryCount (how many times a 429 is retried; 0 retries none) and
// RetryWaitTime (the wait before retry n is n times this; required when RetryCount is above zero). ApiKey
// is a secret: keep it out of committed config and supply it at deploy time; with a base URL and no key the
// lister warns once and uses logs only.
type HTTPConfig struct {
	BaseURL       string                `json:"base_url"`
	ApiKey        string                `json:"api_key"`
	Timeout       durationjson.Duration `json:"timeout"`
	RetryCount    *int                  `json:"retry_count"`
	RetryWaitTime durationjson.Duration `json:"retry_wait_time"`
}

// settings is Config validated, with addresses parsed and token keys lowercased.
type settings struct {
	dexId         string
	aqua          common.Address
	router        common.Address
	launch        common.Address
	hasLaunch     bool
	startBlock    uint64
	logWindow     uint64
	maxWindows    int
	confirmations uint64
	excluded      map[common.Address]struct{}
	// portal is nil when Portal discovery is off: no URL, or a URL with no key.
	portal *portalSettings
	// portalNoKey is set when a URL is configured without a key: each run counts a portal_fallback of reason
	// no_key, so the missing secret is visible.
	portalNoKey bool
}

// portalSettings is the validated Portal configuration. The key is behind a pointer to a type that prints as
// redacted, so a settings value that reaches a log line cannot show it.
type portalSettings struct {
	baseURL   string
	key       *secret
	chainId   uint64
	pageLimit int
	maxPages  int
	timeout   time.Duration
	retries   int
	backoff   time.Duration
}

// secret is a string that never prints.
type secret struct{ value string }

func (*secret) String() string               { return "[redacted]" }
func (*secret) GoString() string             { return "[redacted]" }
func (*secret) MarshalJSON() ([]byte, error) { return []byte(`"[redacted]"`), nil }

// scrub removes the key from a string that may carry it (an error from an HTTP client).
func (p *portalSettings) scrub(s string) string {
	if p.key.value == "" {
		return s
	}
	return strings.ReplaceAll(s, p.key.value, "[redacted]")
}

func requiredAddress(field, value string) (common.Address, error) {
	if value == "" {
		return common.Address{}, errors.Errorf("1inch-aqua: config %s is required", field)
	}
	return parseAddress(field, value)
}

func parseAddress(field, value string) (common.Address, error) {
	if !common.IsHexAddress(value) {
		return common.Address{}, errors.Errorf("1inch-aqua: config %s %q is not an address", field, value)
	}
	addr := common.HexToAddress(value)
	if addr == (common.Address{}) {
		return common.Address{}, errors.Errorf("1inch-aqua: config %s is the zero address", field)
	}
	return addr, nil
}

// trackerSettings validates what the tracker and the simulator factory need: the registry, the router and
// confirmations (how deep a docked read must be before the dock is final).
func (c *Config) trackerSettings() (*settings, error) {
	var err error
	if c.Confirmations == nil {
		return nil, errors.New("1inch-aqua: config confirmations is required (0 where finality is instant)")
	}
	s := &settings{dexId: c.DexId, confirmations: *c.Confirmations}
	if s.aqua, err = requiredAddress("aquaAddress", c.AquaAddress); err != nil {
		return nil, err
	}
	if s.router, err = requiredAddress("routerAddress", c.RouterAddress); err != nil {
		return nil, err
	}
	if c.LaunchRouterAddress != "" {
		if s.launch, err = parseAddress("launchRouterAddress", c.LaunchRouterAddress); err != nil {
			return nil, err
		}
		switch s.launch {
		case s.router:
			return nil, errors.New("1inch-aqua: config launchRouterAddress equals routerAddress")
		case s.aqua:
			return nil, errors.New("1inch-aqua: config launchRouterAddress equals aquaAddress")
		}
		s.hasLaunch = true
	}
	return s, nil
}

// settings validates the whole config for the lister.
func (c *Config) settings() (*settings, error) {
	s, err := c.trackerSettings()
	if err != nil {
		return nil, err
	}
	switch {
	case c.StartBlock == 0:
		return nil, errors.New("1inch-aqua: config startBlock is required and must be above zero")
	case c.LogWindow == 0:
		return nil, errors.New("1inch-aqua: config logWindow is required and must be above zero")
	case c.MaxWindowsPerRun <= 0:
		return nil, errors.New("1inch-aqua: config maxWindowsPerRun is required and must be above zero")
	}
	s.startBlock, s.logWindow, s.maxWindows = c.StartBlock, c.LogWindow, c.MaxWindowsPerRun
	if s.portal, err = c.portalSettings(); err != nil {
		return nil, err
	}
	s.portalNoKey = c.HTTPConfig.BaseURL != "" && s.portal == nil
	s.excluded = make(map[common.Address]struct{}, len(c.ExcludedTokens))
	for _, token := range c.ExcludedTokens {
		addr, err := parseAddress("excludedTokens", token)
		if err != nil {
			return nil, err
		}
		s.excluded[addr] = struct{}{}
	}
	return s, nil
}

// portalSettings validates the Portal fields. A nil result with no error means the Portal is not used: no URL,
// or a URL with no key (a warning, not a failure: a deployment without the key must still run on logs).
func (c *Config) portalSettings() (*portalSettings, error) {
	h := c.HTTPConfig
	if h.BaseURL == "" {
		return nil, nil
	}
	u, err := url.Parse(h.BaseURL)
	switch {
	case err != nil || u.Host == "":
		return nil, errors.Errorf("1inch-aqua: config http_config.base_url is not a URL")
	case u.Scheme != "https":
		return nil, errors.New("1inch-aqua: config http_config.base_url must be https")
	case u.User != nil || u.RawQuery != "" || u.Fragment != "":
		return nil, errors.New("1inch-aqua: config http_config.base_url must be a base URL without credentials, query or fragment")
	case c.ChainId == 0:
		return nil, errors.New("1inch-aqua: config chainId is required with http_config.base_url")
	case c.PortalPageLimit <= 0:
		return nil, errors.New("1inch-aqua: config portalPageLimit is required and must be above zero")
	case c.PortalMaxPages <= 0:
		return nil, errors.New("1inch-aqua: config portalMaxPages is required and must be above zero")
	case h.Timeout.Duration <= 0:
		return nil, errors.New("1inch-aqua: config http_config.timeout is required and must be above zero")
	case h.RetryCount == nil || *h.RetryCount < 0:
		return nil, errors.New("1inch-aqua: config http_config.retry_count is required and must not be negative")
	case *h.RetryCount > 0 && h.RetryWaitTime.Duration <= 0:
		return nil, errors.New("1inch-aqua: config http_config.retry_wait_time is required above zero when retry_count is above zero")
	}
	if strings.TrimSpace(h.ApiKey) == "" {
		logger.Warnf("1inch-aqua: http_config.base_url is set but http_config.api_key is empty: discovery uses the on-chain logs only")
		return nil, nil
	}
	return &portalSettings{
		baseURL: strings.TrimRight(u.String(), "/"), key: &secret{value: strings.TrimSpace(h.ApiKey)},
		chainId: uint64(c.ChainId), pageLimit: c.PortalPageLimit, maxPages: c.PortalMaxPages,
		timeout: h.Timeout.Duration, retries: *h.RetryCount, backoff: h.RetryWaitTime.Duration,
	}, nil
}
