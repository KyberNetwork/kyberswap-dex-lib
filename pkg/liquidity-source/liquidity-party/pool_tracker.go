package liquidityparty

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, params, nil)
}

func (d *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (d *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	lg := logger.WithFields(logger.Fields{"dex_id": d.config.DexID, "pool_id": p.Address})
	lg.Info("Started getting new pool state")
	defer lg.Info("Finished getting new pool state")

	// Kill is irreversible. Once we know a pool is dead it is never routed and its swap state can
	// never change in a way that matters, so short-circuit with no RPC at all — the steady-state
	// fast path for killed pools.
	prevKilled, hadPrevExtra := previousKilled(p.Extra)
	if prevKilled {
		return p, nil
	}

	// A Killed() event in this window is the primary, authoritative kill signal (kill can't be
	// undone), so we act on it without polling killed() over RPC.
	killedFromLogs := hasKilledLog(params.Logs, p.Address)

	// Pin the swap-state snapshot to the block of the events we were told about so it is consistent
	// with them; otherwise read latest. We only pin forward (logBlock > p.BlockNumber) to avoid
	// re-reading state older than what we already hold. We deliberately do NOT gate the refresh on
	// log presence: effectiveSigmaQ (the EMA-driven price anchor) drifts on idle, event-less blocks,
	// so the tracker must refresh every block regardless of events.
	var pinBlock uint64
	if logBlock := eth.GetLatestBlockNumberFromLogs(params.Logs); logBlock > p.BlockNumber {
		pinBlock = logBlock
	}

	// killed() is consulted over RPC only as a one-time cold-start backstop: a pool pulled in by the
	// index backfill (rather than a PartyStarted event) may have been killed before we began
	// listening, so its kill predates any log we can see. Once we hold prior state the Killed event
	// is authoritative and we never poll killed() again.
	withKilledBackstop := !hadPrevExtra && !killedFromLogs

	snapshot, killedRPC, blockNumber, err := d.fetchPoolState(ctx, p.Address, pinBlock, withKilledBackstop, overrides)
	if err != nil {
		lg.WithFields(logger.Fields{"err": err}).Error("fetchPoolState failed")
		return p, err
	}

	if err := validateSnapshot(snapshot, len(p.Tokens)); err != nil {
		lg.WithFields(logger.Fields{"err": err}).Error("invalid snapshot")
		return p, err
	}

	extra := Extra{
		Kappa:           snapshot.Kappa,
		EffectiveSigmaQ: snapshot.EffectiveSigmaQ,
		QInternal:       snapshot.QInternal,
		Bases:           snapshot.Bases,
		FeesPpm:         make([]uint64, len(snapshot.FeesPpm)),
		Killed:          killedFromLogs || killedRPC,
	}
	for i, fee := range snapshot.FeesPpm {
		extra.FeesPpm[i] = fee.Uint64()
	}

	extraBytes, err := json.Marshal(&extra)
	if err != nil {
		return p, err
	}

	// Reserves come from LP-owned cached balances (TVL), which exclude pending protocol fees;
	// never raw balanceOf. Order is aligned with the pool token order.
	reserves := make(entity.PoolReserves, len(snapshot.CachedBalances))
	for i, bal := range snapshot.CachedBalances {
		reserves[i] = bal.String()
	}

	p.Extra = string(extraBytes)
	p.Reserves = reserves
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	return p, nil
}

// fetchPoolState reads the one-shot PartyInfo.fetchPoolState snapshot, optionally bundling the pool's
// killed() flag (only on the cold-start backstop path — see getNewPoolState). When pinBlock > 0 the
// read is pinned to that block for a view consistent with the events we were told about; pinBlock == 0
// reads latest. The returned killed value is meaningful only when withKilled is true.
func (d *PoolTracker) fetchPoolState(
	ctx context.Context,
	poolAddress string,
	pinBlock uint64,
	withKilled bool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*PoolStateSnapshotRPC, bool, uint64, error) {
	var (
		result fetchPoolStateResult
		killed bool
	)

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}
	req.AddCall(&ethrpc.Call{
		ABI:    partyInfoABI,
		Target: d.config.PartyInfoAddress,
		Method: infoMethodFetchPoolState,
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{&result})
	if withKilled {
		req.AddCall(&ethrpc.Call{
			ABI:    partyPoolABI,
			Target: poolAddress,
			Method: poolMethodKilled,
		}, []any{&killed})
	}

	if pinBlock > 0 {
		req.SetBlockNumber(new(big.Int).SetUint64(pinBlock))
		if _, err := req.Aggregate(); err != nil {
			return nil, false, 0, err
		}
		return &result.State, killed, pinBlock, nil
	}

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, false, 0, err
	}

	var blockNumber uint64
	if resp.BlockNumber != nil {
		blockNumber = resp.BlockNumber.Uint64()
	}

	return &result.State, killed, blockNumber, nil
}

// previousKilled reports the killed flag carried in the pool's prior Extra, and whether any prior
// Extra was present at all (hadExtra == false means this is the pool's first refresh, i.e. cold
// start). Unparseable Extra is treated as cold start so the killed() backstop still runs.
func previousKilled(extraStr string) (killed, hadExtra bool) {
	if extraStr == "" {
		return false, false
	}
	var e Extra
	if err := json.Unmarshal([]byte(extraStr), &e); err != nil {
		return false, false
	}
	return e.Killed, true
}

// hasKilledLog reports whether params.Logs contains a PartyPool Killed() event from poolAddress.
func hasKilledLog(logs []types.Log, poolAddress string) bool {
	target := common.HexToAddress(poolAddress)
	for i := range logs {
		l := &logs[i]
		if l.Removed || l.Address != target {
			continue
		}
		if len(l.Topics) > 0 && l.Topics[0] == killedEventTopic {
			return true
		}
	}
	return false
}

// validateSnapshot rejects a snapshot with missing/mismatched arrays so we never build corrupt state.
func validateSnapshot(s *PoolStateSnapshotRPC, nTokens int) error {
	if s == nil || s.Kappa == nil || s.EffectiveSigmaQ == nil {
		return fmt.Errorf("%w: nil scalar field", ErrInvalidExtra)
	}
	if len(s.QInternal) != nTokens || len(s.Bases) != nTokens ||
		len(s.FeesPpm) != nTokens || len(s.CachedBalances) != nTokens {
		return fmt.Errorf("%w: array length != %d tokens (q=%d bases=%d fees=%d bal=%d)",
			ErrInvalidExtra, nTokens, len(s.QInternal), len(s.Bases), len(s.FeesPpm), len(s.CachedBalances))
	}
	for i := 0; i < nTokens; i++ {
		if s.QInternal[i] == nil || s.Bases[i] == nil || s.FeesPpm[i] == nil || s.CachedBalances[i] == nil {
			return fmt.Errorf("%w: nil array element at %d", ErrInvalidExtra, i)
		}
	}
	return nil
}
