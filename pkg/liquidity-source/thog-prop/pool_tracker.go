package thogprop

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: cfg, ethrpcClient: ethrpcClient}, nil
}

// makerSnapshotResult mirrors THOGAMM_AGGREGATOR_INTEGRATION.md section 3's
// MakerSnapshot struct field-for-field and in order -- the doc states the
// ABI is append-only and must be decoded in full.
type makerSnapshotResult struct {
	BlockNumber    *big.Int
	V1             *big.Int
	V2             *big.Int
	R1             *big.Int
	R2             *big.Int
	I1             *big.Int
	GloballyPaused bool
	Tokens         []common.Address
	Balances       []*big.Int
	V3             *big.Int
	R3             *big.Int
	I2             *big.Int
	RiskV3Ready    bool
	P1             *big.Int
	P2             *big.Int
	PairRiskReady  bool
}

// GetNewPoolState polls makerSnapshot() and rewrites Extra wholesale from
// the result -- this package never tries to incrementally apply events (the
// doc's optional event-based mirror), only ever re-reads the full state.
// Token order is re-verified against tokenTable every refresh: an upgrade
// could in principle change the registry even though pool_list_updater.go
// already checked it once at discovery.
func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	// makerSnapshot() has exactly one return value that is itself a dynamic
	// tuple. go-ethereum's Arguments.copyAtomic assigns the whole unpacked
	// value into the destination struct's first field in that case, not the
	// struct itself -- passing &snapshot directly silently (or, here, loudly)
	// corrupts the decode. Wrap it and unwrap after.
	var out struct{ Snapshot makerSnapshotResult }
	if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: thogAMMABI, Target: t.config.Proxy, Method: methodMakerSnapshot}, []any{&out}).
		Call(); err != nil {
		return p, err
	}
	snapshot := out.Snapshot

	want := tokenAddresses()
	if len(snapshot.Tokens) != len(want) || len(snapshot.Balances) != len(want) {
		return p, ErrTokenOrderMismatch
	}
	reserves := make(entity.PoolReserves, len(want))
	for i, addr := range snapshot.Tokens {
		if !strings.EqualFold(addr.Hex(), want[i]) {
			return p, ErrTokenOrderMismatch
		}
		if snapshot.Balances[i] == nil {
			return p, ErrTokenOrderMismatch
		}
		reserves[i] = snapshot.Balances[i].String()
	}

	extraBytes, err := json.Marshal(Extra{
		V1:             snapshot.V1.String(),
		V2:             snapshot.V2.String(),
		V3:             snapshot.V3.String(),
		R1:             snapshot.R1.String(),
		R2:             snapshot.R2.String(),
		R3:             snapshot.R3.String(),
		I1:             snapshot.I1.String(),
		I2:             snapshot.I2.String(),
		P1:             snapshot.P1.String(),
		P2:             snapshot.P2.String(),
		GloballyPaused: snapshot.GloballyPaused,
		RiskV3Ready:    snapshot.RiskV3Ready,
		PairRiskReady:  snapshot.PairRiskReady,
		SnapshotBlock:  snapshot.BlockNumber.Uint64(),
	})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = reserves
	p.BlockNumber = snapshot.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()
	return p, nil
}
