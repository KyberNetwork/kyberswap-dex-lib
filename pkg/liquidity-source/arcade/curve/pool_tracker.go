package curve

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}, nil
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}
	hook := staticExtra.Hook
	if hook == "" {
		hook = t.config.Hook
	}
	poolID := common.HexToHash(PoolIDFromAddress(p.Address))
	token := common.HexToAddress(p.Tokens[1].Address)

	var (
		state  curveStateResult
		snipe  snipeConfigResult
		paused bool
	)
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: arcadeHookABI, Target: hook, Method: "curveStates", Params: []any{poolID}}, []any{&state}).
		AddCall(&ethrpc.Call{ABI: arcadeHookABI, Target: hook, Method: "snipeConfigs", Params: []any{token}}, []any{&snipe}).
		AddCall(&ethrpc.Call{ABI: arcadeHookABI, Target: hook, Method: "paused"}, []any{&paused}).
		Aggregate()
	if err != nil {
		return p, err
	}
	if state.TokensSold == nil || state.RealUsdcReserve == nil {
		return p, ErrNotTracked
	}

	extra := Extra{
		Tracked:           true,
		TokensSold:        uint256.MustFromBig(state.TokensSold),
		RealUsdcReserve:   uint256.MustFromBig(state.RealUsdcReserve),
		Mode:              state.Mode,
		Status:            state.Status,
		Paused:            paused,
		SnipeStartBps:     snipe.StartBps,
		SnipeDecaySeconds: snipe.DecaySeconds,
		SnipeLaunchedAt:   snipe.LaunchedAt,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)

	// Reserves: the real USDC collected and the tokens the curve can still sell.
	remaining := new(uint256.Int)
	if extra.TokensSold.Cmp(curveSupply) < 0 {
		remaining.Sub(curveSupply, extra.TokensSold)
	}
	p.Reserves = entity.PoolReserves{extra.RealUsdcReserve.Dec(), remaining.Dec()}

	if extra.Status == statusCurving && extra.Mode == modePump {
		p.Timestamp = time.Now().Unix()
	} else {
		// Graduated (or not a curve launch): permanently done here, the launch now
		// trades on its uniswap-v4 pool. Timestamp 1 lets pool-service archive it.
		p.Timestamp = 1
	}
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}
	return p, nil
}
