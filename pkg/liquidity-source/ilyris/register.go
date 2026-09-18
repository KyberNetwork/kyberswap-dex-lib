package ilyris

import (
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	poolfactory "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

// Registration, in the four places their service looks.
//
// Each is a package-level `var _ =` so it runs on import and nothing has to remember to call
// it. Their generic helpers pin the constructor signatures, which means these lines are also
// a compile-time check that our constructors match what the service will actually call --
// a wrong shape fails to build here rather than at service start.
var (
	_ = pool.RegisterFactory0(DexType, NewPoolSimulator)
	_ = poolfactory.RegisterFactoryC(DexType, newPoolFactoryFromConfig)
	_ = poollist.RegisterFactoryCE(DexType, newPoolsListUpdaterFromConfig)
	_ = pooltrack.RegisterFactoryCEG0(DexType, func(
		cfg *Config, client *ethrpc.Client, _ *graphqlpkg.Client,
	) *PoolTracker {
		return NewPoolTrackerFromConfig(cfg, client)
	})
)

// The adapters below exist because their factories hand you a *Config and an *ethrpc.Client,
// while our types take a chainReader. That indirection is deliberate -- see chain.go: it is
// what makes cold-start, fail-closed and cursor behaviour testable without a node -- so this
// is the one place the two shapes are joined.

func newPoolFactoryFromConfig(cfg *Config) *PoolFactory {
	id := string(DexType)
	if cfg != nil && cfg.DexID != "" {
		id = cfg.DexID
	}
	return NewPoolFactory(id)
}

func newPoolsListUpdaterFromConfig(cfg *Config, client *ethrpc.Client) *PoolsListUpdater {
	if cfg == nil {
		cfg = &Config{}
	}
	u := NewPoolsListUpdater(NewEthrpcChain(client, cfg.LensAddress), cfg.FactoryAddress, cfg.DexID)
	if cfg.NewPoolLimit > 0 {
		u.limit = cfg.NewPoolLimit
	}
	return u
}

// NewPoolTrackerFromConfig is the tracker's config-shaped constructor.
// A nil client must not panic: factory_test constructs trackers with a nil ethrpc client.
func NewPoolTrackerFromConfig(cfg *Config, client *ethrpc.Client) *PoolTracker {
	if cfg == nil {
		cfg = &Config{}
	}
	return NewPoolTracker(NewEthrpcChain(client, cfg.LensAddress))
}
