package lunarbase

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

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

func (t *PoolTracker) GetDependencies(_ context.Context, p entity.Pool) ([]string, bool, error) {
	return []string{strings.ToLower(p.Address)}, true, nil
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	state, err := fetchRPCState(ctx, t.config, t.ethrpcClient, overrides)
	if err != nil {
		return p, err
	}

	p.Reserves = entity.PoolReserves{
		state.reserveX.String(),
		state.reserveY.String(),
	}
	p.BlockNumber = state.blockNumber
	p.Timestamp = time.Now().Unix()

	updatedPool, err := buildEntityPool(t.config, state)
	if err != nil {
		return p, err
	}

	p.Extra = updatedPool.Extra
	p.StaticExtra = updatedPool.StaticExtra
	p.Tokens = updatedPool.Tokens

	return p, nil
}

func (t *PoolTracker) processLogs(p entity.Pool, logs []types.Log) (entity.Pool, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return p, err
	}

	changed := false
	var latestLogBlock uint64
	for _, log := range logs {
		if len(log.Topics) == 0 {
			continue
		}
		if log.BlockNumber > latestLogBlock {
			latestLogBlock = log.BlockNumber
		}

		switch log.Topics[0] {
		case topicStateUpdated:
			if err := t.processStateUpdated(&extra, log); err == nil {
				changed = true
			}
		case topicSync:
			if reserveX, reserveY, err := t.processSync(log); err == nil {
				p.Reserves = entity.PoolReserves{reserveX.String(), reserveY.String()}
				changed = true
			}
		case topicConcentrationKSet:
			if err := t.processConcentrationKSet(&extra, log); err == nil {
				changed = true
			}
		case topicBlockDelaySet:
			if err := t.processBlockDelaySet(&extra, log); err == nil {
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
	if len(values) < 1 {
		return ErrQuoteFailed
	}
	tuple := *abi.ConvertType(values[0], new(struct {
		PX96 *big.Int `abi:"pX96"`
		Fee  *big.Int `abi:"fee"`
	})).(*struct {
		PX96 *big.Int `abi:"pX96"`
		Fee  *big.Int `abi:"fee"`
	})
	if tuple.PX96 == nil || tuple.Fee == nil {
		return ErrQuoteFailed
	}

	extra.PX96 = uint256.MustFromBig(tuple.PX96)
	extra.Fee = tuple.Fee.Uint64()
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

	extra.PX96 = new(uint256.Int).Set(state.PX96)
	extra.Fee = state.FeeQ48
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
