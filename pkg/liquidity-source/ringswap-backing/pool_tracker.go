package ringswapbacking

import (
	"context"
	"errors"
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

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	if err := t.config.validate(); err != nil {
		return p, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}
	configured, ok := t.config.routerConfig(staticExtra.RouterAddress)
	if !ok {
		return p, ErrInvalidConfig
	}
	staticExtra.ReplaceOrdinaryPair = configured.ReplaceOrdinaryPair
	staticExtra.NoRecallGasToken0 = configured.NoRecallGasToken0
	staticExtra.NoRecallGasToken1 = configured.NoRecallGasToken1
	staticExtra.RecallGasToken0 = configured.RecallGasToken0
	staticExtra.RecallGasToken1 = configured.RecallGasToken1
	if !validStaticExtra(p, staticExtra) {
		return p, ErrInvalidState
	}

	var stateResult backingSourceStateResult
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI: routerABI, Target: staticExtra.RouterAddress, Method: "backingSourceState",
	}, []any{&stateResult}).TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	state := stateResult.State
	if !validState(state) || resp.BlockNumber == nil {
		return p, ErrInvalidState
	}
	if p.BlockNumber > resp.BlockNumber.Uint64() {
		return p, nil
	}
	if len(p.Tokens) != 2 {
		return p, errors.New("ringswap-backing requires exactly two origin tokens")
	}

	extraBytes, err := json.Marshal(Extra{
		WrapperBuffer0:  state.WrapperBuffer0,
		WrapperBuffer1:  state.WrapperBuffer1,
		RecallCapacity0: state.RecallCapacity0,
		RecallCapacity1: state.RecallCapacity1,
	})
	if err != nil {
		return p, err
	}
	p.Reserves = entity.PoolReserves{state.Reserve0.String(), state.Reserve1.String()}
	p.Extra = string(extraBytes)
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return p, err
	}
	p.StaticExtra = string(staticExtraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()
	return p, nil
}

func validStaticExtra(p entity.Pool, extra StaticExtra) bool {
	return common.IsHexAddress(extra.RouterAddress) && common.IsHexAddress(extra.PairAddress) &&
		common.IsHexAddress(extra.Wrapper0) && common.IsHexAddress(extra.Wrapper1) &&
		common.HexToAddress(extra.RouterAddress) != (common.Address{}) &&
		common.HexToAddress(extra.PairAddress) != (common.Address{}) &&
		common.HexToAddress(extra.Wrapper0) != (common.Address{}) &&
		common.HexToAddress(extra.Wrapper1) != (common.Address{}) &&
		!strings.EqualFold(extra.Wrapper0, extra.Wrapper1) &&
		strings.EqualFold(p.Address, extra.PairAddress) && extra.ReplaceOrdinaryPair &&
		extra.NoRecallGasToken0 > 0 && extra.NoRecallGasToken1 > 0 &&
		extra.RecallGasToken0 > 0 && extra.RecallGasToken1 > 0
}

func (c *Config) routerConfig(routerAddress string) (RouterConfig, bool) {
	for _, configured := range c.Routers {
		if strings.EqualFold(configured.Address, routerAddress) {
			return configured, true
		}
	}
	return RouterConfig{}, false
}

func validState(state BackingSourceState) bool {
	values := []*big.Int{
		state.Reserve0,
		state.Reserve1,
		state.WrapperBuffer0,
		state.WrapperBuffer1,
		state.RecallCapacity0,
		state.RecallCapacity1,
	}
	for _, value := range values {
		if value == nil || value.Sign() < 0 {
			return false
		}
	}
	return state.Reserve0.Sign() > 0 && state.Reserve1.Sign() > 0
}
