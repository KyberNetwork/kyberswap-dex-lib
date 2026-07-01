package lazy

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	stableng "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-ng"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *shared.Config
	ethrpcClient *ethrpc.Client
	logger       logger.Logger
}

var _ = pooltrack.RegisterFactoryCE(stableng.DexType, NewPoolTracker)

func NewPoolTracker(
	config *shared.Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	lg := logger.WithFields(logger.Fields{
		"dexId":   config.DexID,
		"dexType": stableng.DexType,
	})

	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		logger:       lg,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, poolpkg.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	lg := t.logger.WithFields(logger.Fields{"poolAddress": p.Address})
	lg.Info("Start updating state ...")
	defer func() { lg.Info("Finish updating state.") }()

	var staticExtra stableng.StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return entity.Pool{}, err
	}

	d := &rpcData{balances: make([]*big.Int, len(p.Tokens))}
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).
		SetFrom(shared.AddrDummy) // poolMethodStoredRates behaves differently for tx.origin == 0
	addRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) }, p.Address, d)

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		lg.WithFields(logger.Fields{"error": err}).Error("failed to aggregate call pool data")
		return entity.Pool{}, err
	}

	return buildPoolState(lg, p, d, res.BlockNumber)
}

type rpcData struct {
	initialA     *big.Int
	futureA      *big.Int
	initialATime *big.Int
	futureATime  *big.Int
	swapFee      *big.Int
	adminFee     *big.Int
	lpSupply     *big.Int
	storedRates  [shared.MaxTokenCount]*big.Int
	balances     []*big.Int
}

func addRPCCalls(addFn func(*ethrpc.Call, []any), poolAddress string, d *rpcData) {
	addFn(&ethrpc.Call{ABI: stableng.CurveStableNGABI, Target: poolAddress, Method: stableng.PoolMethodInitialA}, []any{&d.initialA})
	addFn(&ethrpc.Call{ABI: stableng.CurveStableNGABI, Target: poolAddress, Method: stableng.PoolMethodFutureA}, []any{&d.futureA})
	addFn(&ethrpc.Call{ABI: stableng.CurveStableNGABI, Target: poolAddress, Method: stableng.PoolMethodInitialATime}, []any{&d.initialATime})
	addFn(&ethrpc.Call{ABI: stableng.CurveStableNGABI, Target: poolAddress, Method: stableng.PoolMethodFutureATime}, []any{&d.futureATime})
	addFn(&ethrpc.Call{ABI: stableng.CurveStableNGABI, Target: poolAddress, Method: stableng.PoolMethodFee}, []any{&d.swapFee})
	addFn(&ethrpc.Call{ABI: stableng.CurveStableNGABI, Target: poolAddress, Method: stableng.PoolMethodAdminFee}, []any{&d.adminFee})
	addFn(&ethrpc.Call{ABI: stableng.CurveStableNGABI, Target: poolAddress, Method: shared.ERC20MethodTotalSupply}, []any{&d.lpSupply})
	addFn(&ethrpc.Call{ABI: stableng.CurveStableNGABI, Target: poolAddress, Method: stableng.PoolMethodStoredRates}, []any{&d.storedRates})
	addFn(&ethrpc.Call{ABI: stableng.CurveStableNGABI, Target: poolAddress, Method: stableng.PoolMethodGetBalances}, []any{&d.balances})
}

func buildPoolState(lg logger.Logger, p entity.Pool, d *rpcData, blockNumber *big.Int) (entity.Pool, error) {
	numTokens := len(d.balances)
	extra := stableng.Extra{
		InitialA:     number.SetFromBig(d.initialA),
		FutureA:      number.SetFromBig(d.futureA),
		InitialATime: d.initialATime.Int64(),
		FutureATime:  d.futureATime.Int64(),
		SwapFee:      number.SetFromBig(d.swapFee),
		AdminFee:     number.SetFromBig(d.adminFee),
	}

	if err := updateRateMultipliers(lg, &extra, numTokens, d.storedRates[:numTokens]); err != nil {
		// if the rates is invalid then clear the pool and return err=nil
		p.Timestamp = time.Now().Unix()
		p.Reserves = make(entity.PoolReserves, numTokens+1)
		for i := range p.Reserves {
			p.Reserves[i] = "0"
		}
		return p, nil
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		lg.WithFields(logger.Fields{"error": err}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	reserves := make(entity.PoolReserves, 0, numTokens+1)
	for _, b := range d.balances {
		reserves = append(reserves, b.String())
	}
	reserves = append(reserves, d.lpSupply.String())

	if blockNumber != nil {
		p.BlockNumber = blockNumber.Uint64()
	}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	return p, nil
}

func updateRateMultipliers(lg logger.Logger, extra *stableng.Extra, numTokens int, customRates []*big.Int) error {
	extra.RateMultipliers = make([]uint256.Int, numTokens)
	lg.Debugf("pool use stored rate %v", customRates)

	for i := range numTokens {
		if customRates[i] == nil {
			return stableng.ErrInvalidStoredRates
		}
		if overflow := extra.RateMultipliers[i].SetFromBig(customRates[i]); overflow {
			lg.WithFields(logger.Fields{"storedRates": customRates}).Error("invalid stored rates")
			return stableng.ErrInvalidStoredRates
		}
	}
	return nil
}
