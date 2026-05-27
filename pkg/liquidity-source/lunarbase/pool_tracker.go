package lunarbase

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	if config.DexID == "" {
		config.DexID = DexType
	}
	if config.ChainID == 0 {
		config.ChainID = valueobject.ChainIDBase
	}

	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	if len(params.Logs) > 0 {
		return t.processLogs(p, params.Logs)
	}

	if sub := GetFlashBlockSubscriber(); sub != nil {
		if state := sub.GetLatestState(); state != nil && !state.IsStale() {
			return t.buildPoolFromCachedState(p, state)
		}
	}

	return t.getNewPoolState(ctx, p, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	log.Ctx(ctx).Info().Str("pool", p.Address).Msg("getting new state")
	defer log.Ctx(ctx).Info().Str("pool", p.Address).Msg("finished getting new state")

	state, err := fetchRPCState(ctx, &p, t.config, t.ethrpcClient, overrides)
	if err != nil {
		return p, err
	}

	_, err = buildEntityPool(&p, t.config, state)
	return p, err
}

func (t *PoolTracker) processLogs(p entity.Pool, logs []types.Log) (entity.Pool, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return p, err
	}

	changed := false
	var latestLogBlock uint64
	for _, lg := range logs {
		if len(lg.Topics) == 0 {
			continue
		}
		if lg.BlockNumber > latestLogBlock {
			latestLogBlock = lg.BlockNumber
		}

		switch lg.Topics[0] {
		case topicStateUpdated:
			if err := t.processStateUpdated(&extra, lg); err == nil {
				changed = true
			}
		case topicSync:
			if reserveX, reserveY, err := t.processSync(lg); err == nil {
				p.Reserves = entity.PoolReserves{reserveX.String(), reserveY.String()}
				changed = true
			}
		case topicSwapExecuted:
			// On the fix/incident contract a swap does not mutate the
			// operator-set sqrt-price; reserves change and arrive via a
			// trailing Sync log. We apply the (dx, dy) deltas pessimistically
			// so quote routing has a coherent post-swap view until Sync lands.
			if err := t.processSwapExecuted(&p, lg); err == nil {
				changed = true
			}
		case topicConcentrationKSet:
			if err := t.processConcentrationKSet(&extra, lg); err == nil {
				changed = true
			}
		case topicBlockDelaySet:
			if err := t.processBlockDelaySet(&extra, lg); err == nil {
				changed = true
			}
		}
	}

	if changed {
		extraBytes, err := json.Marshal(extra)
		if err != nil {
			return p, err
		}
		p.Extra = string(extraBytes)
		if latestLogBlock > 0 {
			p.BlockNumber = latestLogBlock
		}
		p.Timestamp = time.Now().Unix()
	}

	return p, nil
}

func (t *PoolTracker) processStateUpdated(extra *Extra, log types.Log) error {
	values, err := coreABI.Events["StateUpdated"].Inputs.Unpack(log.Data)
	if err != nil {
		return err
	}
	if len(values) < 3 {
		return ErrQuoteFailed
	}

	anchorBig, ok1 := values[0].(*big.Int)
	feeAskBig, ok2 := values[1].(*big.Int)
	feeBidBig, ok3 := values[2].(*big.Int)
	if !ok1 || !ok2 || !ok3 {
		return ErrQuoteFailed
	}

	extra.SqrtPriceX96 = uint256.MustFromBig(anchorBig)
	extra.FeeAskX24 = uint32(feeAskBig.Uint64())
	extra.FeeBidX24 = uint32(feeBidBig.Uint64())
	extra.LatestUpdateBlock = log.BlockNumber
	return nil
}

func (t *PoolTracker) processSync(log types.Log) (*big.Int, *big.Int, error) {
	values, err := coreABI.Events["Sync"].Inputs.Unpack(log.Data)
	if err != nil {
		return nil, nil, err
	}
	if len(values) < 2 {
		return nil, nil, ErrQuoteFailed
	}

	reserveX, ok1 := values[0].(*big.Int)
	reserveY, ok2 := values[1].(*big.Int)
	if !ok1 || !ok2 {
		return nil, nil, ErrQuoteFailed
	}

	return reserveX, reserveY, nil
}

func (t *PoolTracker) processConcentrationKSet(extra *Extra, log types.Log) error {
	values, err := coreABI.Events["ConcentrationKSet"].Inputs.Unpack(log.Data)
	if err != nil {
		return err
	}
	if len(values) < 1 {
		return ErrQuoteFailed
	}

	k, ok := values[0].(uint32)
	if !ok {
		return ErrQuoteFailed
	}

	extra.ConcentrationK = k

	return nil
}

// processSwapExecuted projects the swap's (dx, dy) deltas onto cached
// reserves. The matching Sync log lands later in the same tx and overwrites
// reserves with the authoritative post-swap values.
func (t *PoolTracker) processSwapExecuted(p *entity.Pool, log types.Log) error {
	values, err := coreABI.Events["SwapExecuted"].Inputs.Unpack(log.Data)
	if err != nil {
		return err
	}
	if len(values) < 5 {
		return ErrQuoteFailed
	}

	xToY, ok := values[1].(bool)
	if !ok {
		return ErrQuoteFailed
	}
	dxBig, ok1 := values[2].(*big.Int)
	dyBig, ok2 := values[3].(*big.Int)
	if !ok1 || !ok2 {
		return ErrQuoteFailed
	}

	if len(p.Reserves) < 2 {
		return ErrQuoteFailed
	}
	reserveX, err := uint256.FromDecimal(p.Reserves[0])
	if err != nil {
		return ErrQuoteFailed
	}
	reserveY, err := uint256.FromDecimal(p.Reserves[1])
	if err != nil {
		return ErrQuoteFailed
	}

	dx, dy := uint256.MustFromBig(dxBig), uint256.MustFromBig(dyBig)
	if xToY {
		reserveX.Add(reserveX, dx)
		if reserveY.Lt(dy) {
			return ErrQuoteFailed
		}
		reserveY.Sub(reserveY, dy)
	} else {
		reserveY.Add(reserveY, dy)
		if reserveX.Lt(dx) {
			return ErrQuoteFailed
		}
		reserveX.Sub(reserveX, dx)
	}

	p.Reserves = entity.PoolReserves{reserveX.Dec(), reserveY.Dec()}
	return nil
}

func (t *PoolTracker) processBlockDelaySet(extra *Extra, log types.Log) error {
	values, err := coreABI.Events["BlockDelaySet"].Inputs.Unpack(log.Data)
	if err != nil {
		return err
	}
	if len(values) < 1 {
		return ErrQuoteFailed
	}

	bd, ok := values[0].(uint64)
	if !ok {
		return ErrQuoteFailed
	}

	extra.BlockDelay = bd

	return nil
}

func (t *PoolTracker) buildPoolFromCachedState(p entity.Pool, state *poolState) (entity.Pool, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return p, err
	}

	if state.SqrtPriceX96 != nil {
		extra.SqrtPriceX96 = new(uint256.Int).Set(state.SqrtPriceX96)
	}
	extra.FeeAskX24 = state.FeeAskX24
	extra.FeeBidX24 = state.FeeBidX24
	if state.LatestUpdateBlock > 0 {
		extra.LatestUpdateBlock = state.LatestUpdateBlock
	}
	if state.BlockDelay > 0 {
		extra.BlockDelay = state.BlockDelay
	}
	if state.ConcentrationK > 0 {
		extra.ConcentrationK = state.ConcentrationK
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{
		state.ReserveX.ToBig().String(),
		state.ReserveY.ToBig().String(),
	}
	p.BlockNumber = state.BlockNumber
	p.Timestamp = time.Now().Unix()

	return p, nil
}
