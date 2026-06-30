package stablemetang

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
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *shared.Config
	ethrpcClient *ethrpc.Client
	logger       logger.Logger
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *shared.Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	lg := logger.WithFields(logger.Fields{
		"dexId":   config.DexID,
		"dexType": DexType,
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

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return entity.Pool{}, err
	}

	d := &rpcData{balances: make([]*big.Int, len(p.Tokens))}
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).
		SetFrom(shared.AddrDummy) // poolMethodStoredRates behaves differently for tx.origin == 0
	addRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) }, p.Address, d)

	if res, err := req.TryBlockAndAggregate(); err != nil {
		lg.WithFields(logger.Fields{"error": err}).Error("failed to aggregate call pool data")
		return entity.Pool{}, err
	} else if res.BlockNumber != nil {
		p.BlockNumber = res.BlockNumber.Uint64()
	}

	return buildPoolState(lg, p, d)
}

type rpcData struct {
	initialA, futureA, initialATime, futureATime, swapFee, adminFee, lpSupply *big.Int
	storedRates                                                                [shared.MaxTokenCount]*big.Int
	balances                                                                   []*big.Int
}

func addRPCCalls(addFn func(*ethrpc.Call, []any), poolAddress string, d *rpcData) {
	addFn(&ethrpc.Call{ABI: curveStableMetaNGABI, Target: poolAddress, Method: poolMethodInitialA}, []any{&d.initialA})
	addFn(&ethrpc.Call{ABI: curveStableMetaNGABI, Target: poolAddress, Method: poolMethodFutureA}, []any{&d.futureA})
	addFn(&ethrpc.Call{ABI: curveStableMetaNGABI, Target: poolAddress, Method: poolMethodInitialATime}, []any{&d.initialATime})
	addFn(&ethrpc.Call{ABI: curveStableMetaNGABI, Target: poolAddress, Method: poolMethodFutureATime}, []any{&d.futureATime})
	addFn(&ethrpc.Call{ABI: curveStableMetaNGABI, Target: poolAddress, Method: poolMethodFee}, []any{&d.swapFee})
	addFn(&ethrpc.Call{ABI: curveStableMetaNGABI, Target: poolAddress, Method: poolMethodAdminFee}, []any{&d.adminFee})
	addFn(&ethrpc.Call{ABI: curveStableMetaNGABI, Target: poolAddress, Method: shared.ERC20MethodTotalSupply}, []any{&d.lpSupply})
	addFn(&ethrpc.Call{ABI: curveStableMetaNGABI, Target: poolAddress, Method: poolMethodStoredRates}, []any{&d.storedRates})
	addFn(&ethrpc.Call{ABI: curveStableMetaNGABI, Target: poolAddress, Method: poolMethodGetBalances}, []any{&d.balances})
}

func buildPoolState(lg logger.Logger, p entity.Pool, d *rpcData) (entity.Pool, error) {
	numTokens := len(p.Tokens)
	var extra = Extra{
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
		p.Reserves = make(entity.PoolReserves, len(d.balances)+1)
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

	var reserves = make(entity.PoolReserves, 0, len(d.balances)+1)
	for i := range d.balances {
		reserves = append(reserves, d.balances[i].String())
	}
	reserves = append(reserves, d.lpSupply.String())

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	return p, nil
}

func updateRateMultipliers(lg logger.Logger, extra *Extra, numTokens int, customRates []*big.Int) error {
	extra.RateMultipliers = make([]uint256.Int, numTokens)
	lg.Debugf("pool use stored rate %v", customRates)

	for i := range numTokens {
		if customRates[i] == nil {
			return ErrInvalidStoredRates
		}
		if overflow := extra.RateMultipliers[i].SetFromBig(customRates[i]); overflow {
			lg.WithFields(logger.Fields{"storedRates": customRates}).Error("invalid stored rates")
			return ErrInvalidStoredRates
		}
	}
	return nil
}
