package obric

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
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
	logger.Infof("getting new pool state for %v", p.Address)

	if len(params.Logs) > 0 {
		if !t.shouldUpdate(p, params) {
			return p, nil
		}
	}

	var stateWrapper struct{ PoolState }
	req := t.ethrpcClient.R().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: "getState",
	}, []any{&stateWrapper})
	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}

	state := stateWrapper.PoolState

	extra := Extra{
		ReserveX:        state.ReserveX.String(),
		ReserveY:        state.ReserveY.String(),
		CurrentXK:       state.CurrentXK.String(),
		PreK:            state.PreK.String(),
		FeeMillionth:    state.FeeMillionth,
		PriceMaxAge:     state.PriceMaxAge.Uint64(),
		PriceUpdateTime: state.PriceUpdateTime.Uint64(),
		IsLocked:        state.IsLocked,
		Enable:          state.Enable,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{
		state.ReserveX.String(),
		state.ReserveY.String(),
	}
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()

	logger.Infof("finished getting pool state for %v", p.Address)

	return p, nil
}

func (t *PoolTracker) shouldUpdate(p entity.Pool, params pool.GetNewPoolStateParams) bool {
	factoryAddress := common.HexToAddress(t.config.Factory)

	for _, log := range params.Logs {
		if strings.EqualFold(log.Address.Hex(), p.Address) {
			return true
		}

		if log.Address.Cmp(factoryAddress) == 0 {
			event, err := registryFilterer.ParsePoolsPreKEvent(log)
			if err != nil {
				continue
			}

			if len(event.PreKs) > 0 {
				return true
			}
		}
	}

	return false
}

func (t *PoolTracker) GetDependencies(_ context.Context, p entity.Pool) ([]string, bool, error) {
	var staticExtra StaticExtra
	err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	if err != nil {
		return nil, false, err
	}

	return []string{strings.ToLower(t.config.Factory)}, staticExtra.DependenciesStored, nil
}

func (t *PoolTracker) SetDependenciesStored(p *entity.Pool, isStored bool) error {
	var staticExtra StaticExtra
	err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	if err != nil {
		return err
	}
	staticExtra.DependenciesStored = isStored
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return err
	}
	p.StaticExtra = string(staticExtraBytes)

	return err
}
