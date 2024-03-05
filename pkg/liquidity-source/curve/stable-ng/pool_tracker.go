package stableng

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"
)

type PoolTracker struct {
	config       shared.Config
	ethrpcClient *ethrpc.Client
	logger       logger.Logger
}

func NewPoolTracker(
	config shared.Config,
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
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	lg := t.logger.WithFields(logger.Fields{"poolAddress": p.Address})
	lg.Info("Start updating state ...")
	defer func() { lg.Info("Finish updating state.") }()

	var (
		initialA, futureA, initialATime, futureATime, swapFee, adminFee, lpSupply *big.Int

		numTokens = len(p.Tokens)

		balances = make([]*big.Int, numTokens)

		storedRates [shared.MaxTokenCount]*big.Int
	)

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return entity.Pool{}, err
	}

	calls := t.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    curveStableNGABI,
		Target: p.Address,
		Method: poolMethodInitialA,
		Params: nil,
	}, []interface{}{&initialA})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveStableNGABI,
		Target: p.Address,
		Method: poolMethodFutureA,
		Params: nil,
	}, []interface{}{&futureA})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveStableNGABI,
		Target: p.Address,
		Method: poolMethodInitialATime,
		Params: nil,
	}, []interface{}{&initialATime})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveStableNGABI,
		Target: p.Address,
		Method: poolMethodFutureATime,
		Params: nil,
	}, []interface{}{&futureATime})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveStableNGABI,
		Target: p.Address,
		Method: poolMethodFee,
		Params: nil,
	}, []interface{}{&swapFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveStableNGABI,
		Target: p.Address,
		Method: poolMethodAdminFee,
		Params: nil,
	}, []interface{}{&adminFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveStableNGABI,
		Target: p.Address,
		Method: shared.ERC20MethodTotalSupply,
		Params: nil,
	}, []interface{}{&lpSupply})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveStableNGABI,
		Target: p.Address,
		Method: poolMethodStoredRates,
		Params: nil,
	}, []interface{}{&storedRates})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveStableNGABI,
		Target: p.Address,
		Method: poolMethodGetBalances,
		Params: nil,
	}, []interface{}{&balances})

	if res, err := calls.TryBlockAndAggregate(); err != nil {
		lg.WithFields(logger.Fields{"error": err}).Error("failed to aggregate call pool data")
		return entity.Pool{}, err
	} else if res.BlockNumber != nil {
		p.BlockNumber = res.BlockNumber.Uint64()
	}

	var extra = Extra{
		InitialA:     number.SetFromBig(initialA),
		FutureA:      number.SetFromBig(futureA),
		InitialATime: initialATime.Int64(),
		FutureATime:  futureATime.Int64(),
		SwapFee:      number.SetFromBig(swapFee),
		AdminFee:     number.SetFromBig(adminFee),
	}

	if err := t.updateRateMultipliers(lg, &extra, numTokens, storedRates[:numTokens]); err != nil {
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		lg.WithFields(logger.Fields{"error": err}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	var reserves = make(entity.PoolReserves, 0, len(balances)+1)
	for i := range balances {
		reserves = append(reserves, balances[i].String())
	}
	reserves = append(reserves, lpSupply.String())

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	return p, nil
}

func (t *PoolTracker) updateRateMultipliers(lg logger.Logger, extra *Extra, numTokens int, customRates []*big.Int) error {
	extra.RateMultipliers = make([]uint256.Int, numTokens)
	lg.Debugf("pool use stored rate %v", customRates)

	for i := 0; i < numTokens; i++ {
		if overflow := extra.RateMultipliers[i].SetFromBig(customRates[i]); overflow {
			lg.WithFields(logger.Fields{"storedRates": customRates}).Error("invalid stored rates")
			return ErrInvalidStoredRates
		}
	}
	return nil
}
