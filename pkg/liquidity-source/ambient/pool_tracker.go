package ambient

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
	stateTracker *StateTracker
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
		stateTracker: NewStateTracker(ethrpcClient.GetETHClient(), cfg.SwapDex),
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()
	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("started getting new pool state")
	defer func() {
		logger.WithFields(logger.Fields{
			"pool_id":     p.Address,
			"duration_ms": time.Since(startTime).Milliseconds(),
		}).Info("finished getting new pool state")
	}()

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}
	if staticExtra.PoolIdx == 0 {
		return p, fmt.Errorf("StaticExtra.PoolIdx is zero")
	}

	var extra Extra
	if len(p.Extra) != 0 {
		if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
			return p, err
		}
	}

	blockNumber, err := t.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		return p, err
	}
	blockNumBI := new(big.Int).SetUint64(blockNumber)

	if err := t.fetchReserves(ctx, &p, &staticExtra, blockNumBI); err != nil {
		return p, err
	}

	var state *TrackerExtra
	if extra.State == nil {
		state, err = t.stateTracker.LoadCentered(
			ctx,
			common.HexToAddress(staticExtra.Base),
			common.HexToAddress(staticExtra.Quote),
			staticExtra.PoolIdx, blockNumBI,
			t.cfg.TickRange,
		)
	} else {
		window := t.tickWindow(&extra.State.Curve)
		var changed bool
		state, changed, err = t.stateTracker.Refresh(ctx, extra.State, blockNumBI, window)
		if err == nil && !changed {
			state = extra.State
		}
	}
	if err != nil {
		return p, err
	}
	extra.State = state

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = blockNumber

	return p, nil
}

func (t *PoolTracker) fetchReserves(ctx context.Context, p *entity.Pool, sE *StaticExtra, blockNum *big.Int) error {
	reserves := make([]*big.Int, 2)
	swapDex := common.HexToAddress(t.cfg.SwapDex)

	req := t.ethrpcClient.R().SetContext(ctx).SetBlockNumber(blockNum)
	for i, addr := range [2]string{sE.Base, sE.Quote} {
		if valueobject.IsZero(addr) {
			req.AddCall(&ethrpc.Call{
				ABI:    abi.Multicall3ABI,
				Target: t.cfg.Multicall3,
				Method: abi.Multicall3GetEthBalance,
				Params: []any{swapDex},
			}, []any{&reserves[i]})
		} else {
			req.AddCall(&ethrpc.Call{
				ABI:    abi.Erc20ABI,
				Target: addr,
				Method: abi.Erc20BalanceOfMethod,
				Params: []any{swapDex},
			}, []any{&reserves[i]})
		}
	}
	if _, err := req.Aggregate(); err != nil {
		return err
	}

	p.Reserves = bignum.ToStrings(reserves)
	return nil
}

func (t *PoolTracker) tickWindow(prevCurve *CurveState) TickWindow {
	if t.cfg.TickRange <= 0 {
		return FullTickWindow
	}
	if prevCurve == nil || prevCurve.PriceRoot == nil || prevCurve.PriceRoot.Sign() == 0 {
		return FullTickWindow
	}

	center := GetTickAtSqrtRatio(prevCurve.PriceRoot)
	minTick := center - t.cfg.TickRange
	maxTick := center + t.cfg.TickRange
	if minTick < FullTickWindow.MinTick {
		minTick = FullTickWindow.MinTick
	}
	if maxTick > FullTickWindow.MaxTick {
		maxTick = FullTickWindow.MaxTick
	}

	return TickWindow{MinTick: minTick, MaxTick: maxTick}
}
