package ticks

import (
	"errors"
	"maps"
	"math/big"

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
		Address:  pool.Address,
		Exchange: pool.Exchange,
		Ticks:    ticks,
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
