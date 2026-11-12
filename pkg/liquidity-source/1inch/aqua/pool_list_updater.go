package oneinchaqua

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	ethutil "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/metrics"
)

// Reasons the lister adds to the program-level discard reasons.
const (
	ReasonTokenCount     DiscardReason = "token_count"
	ReasonTokenExcluded  DiscardReason = "token_excluded"
	ReasonDockedInWindow DiscardReason = "docked_in_window"
)

var _ = poollist.RegisterFactoryCE1(DexType, NewPoolsListUpdater)

// PoolsListUpdater lists one pool per strategy from the Shipped, Pushed and Docked logs of the Aqua
// registry, or from the Business Portal. All of its memory is the Metadata blob.
type PoolsListUpdater struct {
	cfg          *settings
	ethrpcClient *ethrpc.Client
	// portal is nil when the Portal is not used (see Config.portalSettings).
	portal *portalClient
	// log is the lister's log sink; tests replace it to check what is written.
	log func(level logLevel, msg string, fields logger.Fields)
}

type logLevel int

const (
	logInfo logLevel = iota
	logWarn
)

func defaultLog(level logLevel, msg string, fields logger.Fields) {
	l := logger.WithFields(fields)
	if level == logWarn {
		l.Warn(msg)
		return
	}
	l.Info(msg)
}

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolsListUpdater, error) {
	s, err := cfg.settings()
	if err != nil {
		return nil, err
	}
	u := &PoolsListUpdater{cfg: s, ethrpcClient: ethrpcClient, log: defaultLog}
	if s.portal != nil {
		u.portal = newPortalClient(s.portal)
	}
	return u, nil
}

// shipped is a Shipped log with the Pushed logs that follow it in the same transaction.
type shipped struct {
	log      types.Log
	maker    common.Address
	app      common.Address
	hash     common.Hash
	strategy []byte
	tokens   []common.Address
}

type strategyKey struct {
	maker common.Address
	app   common.Address
	hash  common.Hash
}

type pushKey struct {
	tx common.Hash
	strategyKey
}

// scanResult is what a run of windows found.
type scanResult struct {
	shipped []*shipped
	docked  map[strategyKey]struct{}
	skipped uint64
}

// GetNewPools lists the new strategies. With the Portal configured (URL and key) it first sweeps the
// Portal; a full sweep is the whole run. Anything else (no Portal, no key, any Portal failure) runs the
// on-chain log scan from the stored cursor. The stored state is the cursor alone.
//
// The confirmed head H = head - confirmations is read before the sweep. After a full sweep the cursor is
// max(cursor, H_prev), where H_prev is the H of the previous full sweep (Metadata.LastPortalHead), and H is
// stored for the next one; the first sweep only stores H. The Portal can lag the chain: a strategy shipped
// at or before H may not be indexed when the sweep runs, so the cursor trails by one sweep and a log
// fallback re-scans at least one sweep interval. No logs are read on a full sweep. A sweep that fails keeps
// the metadata: the log scan starts where the cursor was, and the pools the sweep admitted before it
// failed are returned too (pool ids are the same on both paths, and the indexer skips ids it has).
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var meta Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &meta); err != nil {
			return nil, metadataBytes, err
		}
	}
	head, err := u.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		return nil, metadataBytes, err
	}
	if head < u.cfg.confirmations {
		return nil, metadataBytes, nil
	}
	target := head - u.cfg.confirmations

	var portalPools []entity.Pool
	switch {
	case u.portal != nil:
		var perr *portalError
		portalPools, perr = u.sweepPortal(ctx)
		if perr == nil {
			newMeta, err := json.Marshal(Metadata{LastScannedBlock: max(meta.LastScannedBlock, meta.LastPortalHead), LastPortalHead: target})
			if err != nil {
				return nil, metadataBytes, err
			}
			return portalPools, newMeta, nil
		}
		if ctx.Err() != nil {
			return portalPools, metadataBytes, ctx.Err()
		}
		u.fellBack(perr.reason, perr)
	case u.cfg.portalNoKey:
		u.fellBack(portalNoKey, nil)
	}

	logPools, newMeta, err := u.runLogs(ctx, meta, metadataBytes, target)
	if err != nil {
		return portalPools, metadataBytes, err
	}
	return mergePools(portalPools, logPools), newMeta, nil
}

// fellBack counts and logs a fall back from the Portal to the logs. The error text holds a status or a failure
// kind, never a URL query, a header or a body.
func (u *PoolsListUpdater) fellBack(reason string, err error) {
	metrics.IncrPortalFallback(u.cfg.dexId, reason)
	fields := logger.Fields{"exchange": u.cfg.dexId, "reason": reason}
	if err != nil {
		fields["error"] = err.Error()
	}
	u.log(logWarn, "1inch-aqua portal discovery fell back to logs", fields)
}

// mergePools appends the pools of b that a does not hold.
func mergePools(a, b []entity.Pool) []entity.Pool {
	if len(a) == 0 {
		return b
	}
	seen := make(map[string]struct{}, len(a))
	for _, p := range a {
		seen[p.Address] = struct{}{}
	}
	for _, p := range b {
		if _, dup := seen[p.Address]; !dup {
			a = append(a, p)
		}
	}
	return a
}

// sweepPortal reads the Portal and admits what it lists through the same admission as the log path. The pools
// are returned even when the sweep failed part-way or an item could not be read; the error says it did.
func (u *PoolsListUpdater) sweepPortal(ctx context.Context) ([]entity.Pool, *portalError) {
	items, perr := u.portal.sweep(ctx)
	ships, bad := u.portalShipped(items)
	pools, stats := u.admitAll(ships, nil)
	u.recordDiscards(sourcePortal, stats)
	u.log(logInfo, "1inch-aqua portal sweep", logger.Fields{"exchange": u.cfg.dexId, "items": len(items),
		"pools": len(pools), "discarded": stats, "badItems": bad, "ok": perr == nil && bad == 0})
	if perr == nil && bad > 0 {
		perr = &portalError{portalBadItem, errors.New("items the lister cannot read or without strategy bytes")}
	}
	return pools, perr
}

// portalShipped turns Portal items into the shipped form the log path admits. The Portal's balances are not
// read: the tracker reads state at one block. bad counts the items that cannot be read.
func (u *PoolsListUpdater) portalShipped(items []portalItem) (ships []*shipped, bad int) {
	for i := range items {
		if s, ok := items[i].shipped(); ok {
			ships = append(ships, s)
		} else {
			bad++
		}
	}
	return ships, bad
}

// Discovery sources, the source label of the discard metric.
const (
	sourceLogs   = "logs"
	sourcePortal = "portal"
)

func (u *PoolsListUpdater) recordDiscards(source string, stats map[DiscardReason]int) {
	for reason, count := range stats {
		metrics.IncrDiscardedStrategy(u.cfg.dexId, source, string(reason), float64(count))
	}
}

// runLogs is the on-chain log scan from the stored cursor up to target.
func (u *PoolsListUpdater) runLogs(ctx context.Context, meta Metadata, metadataBytes []byte, target uint64) ([]entity.Pool, []byte, error) {
	last := u.startCursor(meta)
	if last >= target {
		return nil, metadataBytes, nil
	}
	res, last, err := u.scan(ctx, last, target)
	if err != nil {
		return nil, metadataBytes, err
	}
	if res.skipped > 0 {
		metrics.IncrSkippedBlocks(u.cfg.dexId, float64(res.skipped))
	}

	pools, stats := u.admitAll(res.shipped, res.docked)
	u.recordDiscards(sourceLogs, stats)
	u.log(logInfo, "1inch-aqua lister run", logger.Fields{"exchange": u.cfg.dexId, "lastScannedBlock": last, "shipped": len(res.shipped),
		"pools": len(pools), "discarded": stats, "skippedBlocks": res.skipped})

	newMeta, err := json.Marshal(Metadata{LastScannedBlock: last, LastPortalHead: meta.LastPortalHead})
	if err != nil {
		return nil, metadataBytes, err
	}
	return pools, newMeta, nil
}

// startCursor is the last block already scanned: the stored cursor, else the block before startBlock (the
// config requires it above zero).
func (u *PoolsListUpdater) startCursor(meta Metadata) uint64 {
	if meta.LastScannedBlock > 0 {
		return meta.LastScannedBlock
	}
	return u.cfg.startBlock - 1
}

// scan reads up to maxWindows windows after last, up to target. It returns what they held and the end of
// the last window read. A failure in the first window is an error; a later one ends the scan and keeps
// what the finished windows found.
func (u *PoolsListUpdater) scan(ctx context.Context, last, target uint64) (scanResult, uint64, error) {
	res := scanResult{docked: map[strategyKey]struct{}{}}
	for window := 0; window < u.cfg.maxWindows && last < target; window++ {
		from, to := last+1, min(last+u.cfg.logWindow, target)
		logs, skipped, err := u.getLogs(ctx, from, to, target)
		if err != nil {
			if window == 0 {
				return res, last, err
			}
			break
		}
		res.skipped += skipped
		u.collect(&res, logs)
		last = to
	}
	return res, last, nil
}

// getLogs reads one window with a single eth_getLogs for the three events. A window the node refuses for
// its range or result size, or that times out on the server, is split in two. The only error skipped is a
// single block the node cannot serve (HyperEVM "invalid block range") that lies at least a log window
// below target: the same answer comes for a block above the serving node's head (HyperEVM, Polygon bor),
// so a block near target is an error and the cursor stays. Every other error, throttling, client timeouts
// and transport failures included, is returned, so the next run reads the window again.
func (u *PoolsListUpdater) getLogs(ctx context.Context, from, to, target uint64) ([]types.Log, uint64, error) {
	logs, err := u.ethrpcClient.GetETHClient().FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from),
		ToBlock:   new(big.Int).SetUint64(to),
		Addresses: []common.Address{u.cfg.aqua},
		Topics:    [][]common.Hash{{shippedEvent.ID, pushedEvent.ID, dockedEvent.ID}},
	})
	if err == nil {
		return logs, 0, nil
	}
	if ctx.Err() != nil {
		return nil, 0, err
	}
	if from == to {
		if isUnservableBlock(err) && to+u.cfg.logWindow <= target {
			return nil, 1, nil
		}
		return nil, 0, err
	}
	if !isRangeError(err) {
		return nil, 0, err
	}
	mid := from + (to-from)/2
	left, leftSkipped, err := u.getLogs(ctx, from, mid, target)
	if err != nil {
		return nil, 0, err
	}
	right, rightSkipped, err := u.getLogs(ctx, mid+1, to, target)
	if err != nil {
		return nil, 0, err
	}
	return append(left, right...), leftSkipped + rightSkipped, nil
}

var (
	rangeErrorMarkers = []string{"range", "limit", "too many", "exceed", "more than", "response size", "query returned", "too large",
		"gateway timeout", "query timeout"}
	// A node that throttles says "limit", "too many" or "exceeded" too, and a client that gives up says
	// "timeout", but splitting the window does not help either. Server-side timeouts that ask for a smaller
	// range ("Query timeout exceeded", "504 Gateway Timeout") are range errors, not listed here.
	transientErrorMarkers = []string{"too many requests", "rate limit", "rate-limit", "ratelimit", "throttl", "i/o timeout",
		"client.timeout", "tls handshake timeout", "timed out", "deadline exceeded", "compute units", "request limit", "request rate",
		"capacity", "quota", "connection"}
	// A word match: a block number in hex ("0x429") must not read as the status code.
	tooManyRequestsCode = regexp.MustCompile(`\b429\b`)
)

// isRangeError reports a refusal of the window's range, result size or server time, the only errors worth
// splitting for.
func isRangeError(err error) bool {
	if isTransientError(err) {
		return false
	}
	msg := strings.ToLower(err.Error())
	return lo.SomeBy(rangeErrorMarkers, func(m string) bool { return strings.Contains(msg, m) })
}

func isTransientError(err error) bool {
	var httpErr rpc.HTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusTooManyRequests {
		return true
	}
	var netErr net.Error
	if (errors.As(err, &netErr) && netErr.Timeout()) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.HasSuffix(msg, "eof") || tooManyRequestsCode.MatchString(msg) ||
		lo.SomeBy(transientErrorMarkers, func(m string) bool { return strings.Contains(msg, m) })
}

// isUnservableBlock reports the HyperEVM refusal of a block number its node does not have, for a block
// it pruned or has not synced yet. Callers must tell the two apart by distance from the head.
func isUnservableBlock(err error) bool {
	return !isTransientError(err) && strings.Contains(strings.ToLower(err.Error()), "invalid block range")
}

// collect decodes the logs of one window into the run's shipped strategies and docked set.
func (u *PoolsListUpdater) collect(res *scanResult, logs []types.Log) {
	ethutil.SortLogs(logs) // the join below relies on log order within a transaction
	pushes := map[pushKey][]*types.Log{}
	var ships []*shipped
	for i := range logs {
		lg := &logs[i]
		if lg.Removed || len(lg.Topics) == 0 {
			continue
		}
		switch lg.Topics[0] {
		case shippedEvent.ID:
			vals, err := shippedEvent.Inputs.Unpack(lg.Data)
			if err != nil || len(vals) != 4 {
				continue
			}
			ships = append(ships, &shipped{
				log:      *lg,
				maker:    vals[0].(common.Address),
				app:      vals[1].(common.Address),
				hash:     common.Hash(vals[2].([32]byte)),
				strategy: vals[3].([]byte),
			})
		case pushedEvent.ID:
			vals, err := pushedEvent.Inputs.Unpack(lg.Data)
			if err != nil || len(vals) != 5 {
				continue
			}
			k := pushKey{tx: lg.TxHash,
				maker: vals[0].(common.Address), app: vals[1].(common.Address), hash: common.Hash(vals[2].([32]byte))}
			pushes[k] = append(pushes[k], lg)
		case dockedEvent.ID:
			vals, err := dockedEvent.Inputs.Unpack(lg.Data)
			if err != nil || len(vals) != 3 {
				continue
			}
			res.docked[strategyKey{maker: vals[0].(common.Address), app: vals[1].(common.Address),
				hash: common.Hash(vals[2].([32]byte))}] = struct{}{}
		}
	}
	// The join is on (tx, maker, app, strategyHash) and a log index after the Shipped log: one tx can ship
	// several strategies or push into the same strategy again.
	for _, s := range ships {
		k := pushKey{tx: s.log.TxHash, maker: s.maker, app: s.app, hash: s.hash}
		for _, p := range pushes[k] {
			if p.Index <= s.log.Index {
				continue
			}
			vals, _ := pushedEvent.Inputs.Unpack(p.Data)
			if token := vals[3].(common.Address); !lo.Contains(s.tokens, token) {
				s.tokens = append(s.tokens, token)
			}
		}
		res.shipped = append(res.shipped, s)
	}
}

// admitAll runs admit over strategies from either source. docked is the Docked set of the log window, nil
// for the Portal, whose listing holds only open strategies.
func (u *PoolsListUpdater) admitAll(ships []*shipped, docked map[strategyKey]struct{}) ([]entity.Pool, map[DiscardReason]int) {
	stats := map[DiscardReason]int{}
	var (
		pools []entity.Pool
		seen  = map[string]struct{}{}
	)
	for _, s := range ships {
		pool, reason := u.admit(s, docked)
		if reason != "" {
			stats[reason]++
			continue
		}
		if _, dup := seen[pool.Address]; dup {
			continue
		}
		seen[pool.Address] = struct{}{}
		pools = append(pools, *pool)
	}
	return pools, stats
}

func (u *PoolsListUpdater) admit(s *shipped, docked map[strategyKey]struct{}) (*entity.Pool, DiscardReason) {
	launch := u.isLaunch(s.app)
	if s.app != u.cfg.router && !launch {
		return nil, ReasonApp
	}
	order, err := DecodeOrder(s.strategy)
	if err != nil {
		return nil, ReasonNonCanonical
	}
	if order.Maker != s.maker {
		return nil, ReasonMakerMismatch
	}
	if order.Hash() != s.hash {
		return nil, ReasonNonCanonical
	}
	if _, err := ParseOrder(order, launch); err != nil {
		return nil, ReasonOf(err)
	}
	if len(s.tokens) != 2 {
		return nil, ReasonTokenCount
	}

	if _, ok := docked[strategyKey{maker: s.maker, app: s.app, hash: s.hash}]; ok {
		return nil, ReasonDockedInWindow
	}

	// Canonical token order, ascending by address: the pool id does not depend on it and the order the
	// Pushed logs happen to come in does not either.
	pair := [2]common.Address{s.tokens[0], s.tokens[1]}
	if bytes.Compare(pair[0][:], pair[1][:]) > 0 {
		pair[0], pair[1] = pair[1], pair[0]
	}
	for _, token := range pair {
		if _, ok := u.cfg.excluded[token]; ok {
			return nil, ReasonTokenExcluded
		}
	}
	return u.newPool(s, order, pair), ""
}

// isLaunch reports whether app is the configured launch router.
func (u *PoolsListUpdater) isLaunch(app common.Address) bool {
	return u.cfg.hasLaunch && app == u.cfg.launch
}

func poolId(app, maker common.Address, hash common.Hash) string {
	return hexutil.Encode(crypto.Keccak256(app[:], maker[:], hash[:]))
}

func (u *PoolsListUpdater) newPool(s *shipped, order *Order, pair [2]common.Address) *entity.Pool {
	static, _ := json.Marshal(StaticExtra{
		Maker:        hexutil.Encode(s.maker[:]),
		App:          hexutil.Encode(s.app[:]),
		StrategyHash: s.hash.Hex(),
		Aqua:         hexutil.Encode(u.cfg.aqua[:]),
		Launch:       u.isLaunch(s.app),
		Traits:       order.Traits.Hex(),
		Data:         hexutil.Encode(order.Data),
	})
	zero := new(uint256.Int)
	extra, _ := json.Marshal(&Extra{RawBalance0: zero, RawBalance1: zero, WalletBalance0: zero, WalletBalance1: zero,
		Allowance0: zero, Allowance1: zero})
	return &entity.Pool{
		Address:   poolId(s.app, s.maker, s.hash),
		Exchange:  u.cfg.dexId,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: hexutil.Encode(pair[0][:]), Swappable: true},
			{Address: hexutil.Encode(pair[1][:]), Swappable: true},
		},
		Extra:       string(extra),
		StaticExtra: string(static),
	}
}
