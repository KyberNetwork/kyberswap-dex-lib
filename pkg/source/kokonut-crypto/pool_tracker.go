package kokonutcrypto

import (
	"context"
	"encoding/json"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"math/big"
	"strconv"
	"time"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	logger.Infof("[kokonut] Start getting new state of pool %v with type %v", p.Address, p.Type)

	var (
		a, dExtra, gamma, feeGamma, midFee, outFee                                                            *big.Int
		lastPriceTimestamp, lpSupply, xcpProfit, virtualPrice, allowedExtraProfit, adjustmentStep, maHalfTime *big.Int
		priceScale, priceOracle, lastPrices, minRemainingPostRebalanceRatio                                   *big.Int
		futureAGammaTime, initialAGammaTime, futureA, initialA                                                uint32
		futureGamma, initialGamma                                                                             uint64
		balances                                                                                              = make([]*big.Int, len(p.Tokens))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodA,
		Params: nil,
	}, []interface{}{&a})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodD,
		Params: nil,
	}, []interface{}{&dExtra})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodGamma,
		Params: nil,
	}, []interface{}{&gamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodFeeGamma,
		Params: nil,
	}, []interface{}{&feeGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodMidFee,
		Params: nil,
	}, []interface{}{&midFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodOutFee,
		Params: nil,
	}, []interface{}{&outFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodFutureAGammaTime,
		Params: nil,
	}, []interface{}{&futureAGammaTime})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodFutureA,
		Params: nil,
	}, []interface{}{&futureA})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodFutureGamma,
		Params: nil,
	}, []interface{}{&futureGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodInitialAGammaTime,
		Params: nil,
	}, []interface{}{&initialAGammaTime})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodInitialA,
		Params: nil,
	}, []interface{}{&initialA})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodInitialGamma,
		Params: nil,
	}, []interface{}{&initialGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodLastPricesTimestamp,
		Params: nil,
	}, []interface{}{&lastPriceTimestamp})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodXcpProfit,
		Params: nil,
	}, []interface{}{&xcpProfit})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodVirtualPrice,
		Params: nil,
	}, []interface{}{&virtualPrice})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodAllowedExtraProfit,
		Params: nil,
	}, []interface{}{&allowedExtraProfit})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodAdjustmentStep,
		Params: nil,
	}, []interface{}{&adjustmentStep})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodMaHalfTime,
		Params: nil,
	}, []interface{}{&maHalfTime})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodPriceScale,
		Params: nil,
	}, []interface{}{&priceScale})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodPriceOracle,
		Params: nil,
	}, []interface{}{&priceOracle})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodLastPrices,
		Params: nil,
	}, []interface{}{&lastPrices})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodMinRemainingPostRebalanceRatio,
		Params: nil,
	}, []interface{}{&minRemainingPostRebalanceRatio})

	lpToken := p.GetLpToken()
	if len(lpToken) > 0 {
		calls.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: lpToken,
			Method: erc20MethodTotalSupply,
			Params: nil,
		}, []interface{}{&lpSupply})
	}

	for i := range p.Tokens {
		calls.AddCall(&ethrpc.Call{
			ABI:    cryptoSwap2PoolABI,
			Target: p.Address,
			Method: poolMethodBalances,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&balances[i]})
	}

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"poolType":    p.Type,
			"error":       err,
		}).Errorf("failed to aggregate call pool data")
		return entity.Pool{}, err
	}

	var (
		reserves = make(entity.PoolReserves, len(balances))
	)
	for i := range p.Tokens {
		reserves[i] = balances[i].String()
	}

	var extra = Extra{
		A:                              a.String(),
		D:                              dExtra.String(),
		Gamma:                          gamma.String(),
		FeeGamma:                       feeGamma.String(),
		MidFee:                         midFee.String(),
		OutFee:                         outFee.String(),
		FutureAGammaTime:               int64(futureAGammaTime),
		FutureA:                        strconv.FormatUint(uint64(futureA), 10),
		FutureGamma:                    strconv.FormatUint(futureGamma, 10),
		InitialAGammaTime:              int64(initialAGammaTime),
		InitialA:                       strconv.FormatUint(uint64(initialA), 10),
		InitialGamma:                   strconv.FormatUint(initialGamma, 10),
		PriceScale:                     priceScale.String(),
		LastPrices:                     lastPrices.String(),
		PriceOracle:                    priceOracle.String(),
		LpSupply:                       lpSupply.String(),
		XcpProfit:                      xcpProfit.String(),
		VirtualPrice:                   virtualPrice.String(),
		AllowedExtraProfit:             allowedExtraProfit.String(),
		AdjustmentStep:                 adjustmentStep.String(),
		MaHalfTime:                     maHalfTime.String(),
		LastPricesTimestamp:            lastPriceTimestamp.Int64(),
		MinRemainingPostRebalanceRatio: minRemainingPostRebalanceRatio.String(),
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"poolType":    p.Type,
			"error":       err,
		}).Errorf("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	logger.Infof("[kokonut] Finish getting new state of pool %v with type %v", p.Address, p.Type)

	return p, nil
}
