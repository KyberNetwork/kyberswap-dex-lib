package ticks

import (
	"errors"
	"maps"
	"math/big"
	"strconv"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/logger"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/metrics"
)

var ErrTickNetDeltaZero = errors.New("tick net delta must be zero")

type ExtraTickU256 struct {
	Liquidity    *uint256.Int `json:"liquidity"`
	SqrtPriceX96 *uint256.Int `json:"sqrtPriceX96"`
	TickSpacing  uint64       `json:"tickSpacing"`
	Tick         *int         `json:"tick"`
	Ticks        []TickU256   `json:"ticks"`
	LensBlock    uint64       `json:"lensBlock,omitempty"`
}

type Tick struct {
	TickIdx        int      `json:"tickIdx"`
	LiquidityGross *big.Int `json:"liquidityGross"`
	LiquidityNet   *big.Int `json:"liquidityNet"`
}

type TickU256 struct {
	Index          int          `json:"index"`
	LiquidityGross *uint256.Int `json:"liquidityGross"`
	LiquidityNet   *int256.Int  `json:"liquidityNet"`
}

type TicksBasedPool struct {
	Address     string       `json:"address"`
	Ticks       map[int]Tick `json:"ticks"`
	BlockNumber uint64       `json:"blockNumber"`
	Exchange    string       `json:"exchange"`
	// LensBlock is the block the ticks were last read whole at through a lens, 0 if never: every
	// log at or below it is already in them.
	LensBlock uint64 `json:"lensBlock,omitempty"`
}

// LensDue is the block a lens snapshot of a pool still without state must reach: the newest block
// of the logs the pool has seen without state, which a snapshot below it would not hold. 0 if none.
func LensDue(p entity.Pool) uint64 {
	var extra struct {
		LensDue uint64 `json:"lensDue"`
	}
	_ = json.Unmarshal([]byte(p.Extra), &extra)
	return extra.LensDue
}

// WithLensDue records in p's Extra that a snapshot must reach due, the newest block of logs, and
// keeps the rest of the Extra, such as what a pool factory filled in, as it is.
func WithLensDue(p entity.Pool, logs []ethtypes.Log) entity.Pool {
	return editExtra(p, func(extra map[string]json.RawMessage) {
		extra["lensDue"] = json.RawMessage(strconv.FormatUint(lensDueOf(p, logs), 10))
	})
}

// WithoutState drops the state of p its logs built, its liquidity, ticks and lens block, keeping
// the rest of the Extra as it is, and records that a lens snapshot must reach due: p is then a
// pool without state, which a tracker with useLens reads whole on its next update.
func WithoutState(p entity.Pool, due uint64) entity.Pool {
	return editExtra(p, func(extra map[string]json.RawMessage) {
		delete(extra, "liquidity")
		delete(extra, "ticks")
		delete(extra, "lensBlock")
		extra["lensDue"] = json.RawMessage(strconv.FormatUint(max(LensDue(p), due), 10))
	})
}

// lensDueOf is the block a lens snapshot of p must reach once it has seen logs: the newer of the
// one recorded and the newest of logs, removed ones aside.
func lensDueOf(p entity.Pool, logs []ethtypes.Log) uint64 {
	due := LensDue(p)
	for _, l := range logs {
		if !l.Removed {
			due = max(due, l.BlockNumber)
		}
	}
	return due
}

// editExtra applies edit to p's Extra as raw values, so the fields it leaves, such as a price
// wider than a float64, are kept to the digit.
func editExtra(p entity.Pool, edit func(map[string]json.RawMessage)) entity.Pool {
	extra := map[string]json.RawMessage{}
	_ = json.Unmarshal([]byte(p.Extra), &extra)
	edit(extra)
	if b, err := json.Marshal(extra); err == nil {
		p.Extra = string(b)
	}
	return p
}

// Unapplied drops the logs the lens snapshot at LensBlock already holds. Such logs keep arriving
// after the snapshot, from a backfill that replays older blocks or a replay after a crash, and
// applying them again would count their liquidity twice. A removed log and a log carrying no block,
// the empty one that triggers an update, are always kept.
//
// A reorg that reverts a block at or below LensBlock means the snapshot was read on the abandoned
// fork from that block on: LensBlock drops to the block before, and the replacing blocks' logs are
// kept.
func (t *TicksBasedPool) Unapplied(logs []ethtypes.Log) []ethtypes.Log {
	for _, l := range logs {
		if l.Removed && l.BlockNumber > 0 && l.BlockNumber <= t.LensBlock {
			t.LensBlock = l.BlockNumber - 1
		}
	}
	if t.LensBlock == 0 {
		return logs
	}
	kept := logs[:0:0]
	for _, l := range logs {
		if l.Removed || l.BlockNumber == 0 || l.BlockNumber > t.LensBlock {
			kept = append(kept, l)
		}
	}
	return kept
}

// ReadBlock is the block at which an update reads again the ticks its logs touch: the newest of
// the logs' blocks, or the pool's own when that is newer, as it is for logs retried or replayed
// after newer ones were applied, whose own block would put those ticks back to their values there.
// A reorg reads at its logs' block, since the pool's may be one of the abandoned fork, past the new
// head. 0 when the logs carry no block.
func ReadBlock(logs []ethtypes.Log, poolBlock uint64) uint64 {
	block := eth.GetBlockNumberFromLogs(logs)
	if block == 0 || eth.HasRevertedLog(logs) {
		return block
	}
	return max(block, poolBlock)
}

func NewTicksBasedPool(pool entity.Pool) (TicksBasedPool, error) {
	var extra ExtraTickU256
	if len(pool.Extra) > 0 {
		err := json.Unmarshal([]byte(pool.Extra), &extra)
		if err != nil {
			return TicksBasedPool{}, err
		}
	}

	ticks := make(map[int]Tick, len(extra.Ticks))
	for _, tick := range extra.Ticks {
		ticks[tick.Index] = Tick{
			TickIdx:        tick.Index,
			LiquidityGross: tick.LiquidityGross.ToBig(),
			LiquidityNet:   tick.LiquidityNet.ToBig(),
		}
	}

	return TicksBasedPool{
		Address:   pool.Address,
		Exchange:  pool.Exchange,
		Ticks:     ticks,
		LensBlock: extra.LensBlock,
	}, nil
}

func (t TicksBasedPool) HasAllValidTicks() bool {
	if !t.HasValidTicks() {
		metrics.IncrInvalidPoolTicks(t.Exchange)
		return false
	}
	return true
}

func (t TicksBasedPool) HasValidTicks() bool {
	var sum big.Int
	for _, t := range t.Ticks {
		sum.Add(&sum, t.LiquidityNet)
	}
	return sum.Sign() == 0
}

func ValidateAllPoolTicks(pool TicksBasedPool, ticks []Tick) error {
	if err := ValidatePoolTicks(pool, ticks); err != nil {
		metrics.IncrInvalidPoolTicks(pool.Exchange)
		return err
	}
	return nil
}

func ValidatePoolTicks(pool TicksBasedPool, ticks []Tick) error {
	tickMap := make(map[int]Tick, len(pool.Ticks))
	maps.Copy(tickMap, pool.Ticks)

	oldTicks := make([]Tick, 0, len(ticks))
	for _, tick := range ticks {
		oldTicks = append(oldTicks, tickMap[tick.TickIdx])
		tickMap[tick.TickIdx] = tick
	}

	sum := integer.Zero()
	for _, tick := range tickMap {
		sum.Add(sum, tick.LiquidityNet)
	}

	if sum.Sign() != 0 {
		if len(ticks) < 10 {
			logFields := logger.Fields{
				"address":         pool.Address,
				"oldTicks":        oldTicks,
				"newTicks":        ticks,
				"sumLiquidityNet": sum,
			}
			if len(pool.Ticks) < 10 {
				logFields["poolTicks"] = pool.Ticks
			}

			logger.WithFields(logFields).Warn("tick net delta must be zero")
		}

		logger.WithFields(logger.Fields{
			"oldTicks": oldTicks,
			"newTicks": ticks,
		}).Debug("tick net delta must be zero")
		return ErrTickNetDeltaZero
	}

	return nil
}

// EstimateLastActivityTime derives a pool's last-activity timestamp from the chain
// timestamp of the latest log being applied, not from wall-clock processing time, so
// replaying an old (e.g. backfilled) event can never make an abandoned pool look
// freshly active - and a genuinely fresh event still advances the timestamp normally,
// since max-wins is the only rule needed for both directions. Shared by every
// ticks-based tracker's incremental update path.
func EstimateLastActivityTime(p *entity.Pool, logs []ethtypes.Log,
	blockHeaders map[uint64]entity.BlockHeader) int64 {
	if len(logs) > 0 && blockHeaders != nil {
		latestLog := logs[len(logs)-1]
		if blockHeader, ok := blockHeaders[latestLog.BlockNumber]; ok {
			return max(p.Timestamp, int64(blockHeader.Timestamp))
		}
	}

	// Do not update the timestamp as the pool triggered state update via a custom empty log.
	return p.Timestamp
}

// IsMissingTrieNodeError reports whether err indicates the RPC node can't serve state
// for the requested historical block (pruned, not an archive node, etc). Delegates the
// actual classification to eth.IsPrunedStateError, shared with pool-service, so the two
// can't drift out of sync on which error strings different RPC providers use for the
// same underlying condition.
func IsMissingTrieNodeError(err error) bool {
	if !eth.IsPrunedStateError(err) {
		return false
	}
	metrics.IncrMissingTrieNode()
	return true
}
