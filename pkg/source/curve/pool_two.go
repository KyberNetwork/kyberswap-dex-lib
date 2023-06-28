package curve

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func (d *PoolsListUpdater) getNewPoolsTypeTwo(
	ctx context.Context,
	poolAndRegistries []PoolAndRegistries,
) ([]entity.Pool, error) {
	var (
		coins    = make([][8]common.Address, len(poolAndRegistries))
		decimals = make([][8]*big.Int, len(poolAndRegistries))
		lpTokens = make([]common.Address, len(poolAndRegistries))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	for i, poolAndRegistry := range poolAndRegistries {
		calls.AddCall(&ethrpc.Call{
			ABI:    poolAndRegistry.RegistryOrFactoryABI,
			Target: poolAndRegistry.RegistryOrFactoryAddress,
			Method: registryOrFactoryMethodGetCoins,
			Params: []interface{}{poolAndRegistry.PoolAddress},
		}, []interface{}{&coins[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    poolAndRegistry.RegistryOrFactoryABI,
			Target: poolAndRegistry.RegistryOrFactoryAddress,
			Method: registryOrFactoryMethodGetDecimals,
			Params: []interface{}{poolAndRegistry.PoolAddress},
		}, []interface{}{&decimals[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    twoABI,
			Target: poolAndRegistry.PoolAddress.Hex(),
			Method: poolMethodToken,
			Params: nil,
		}, []interface{}{&lpTokens[i]})
	}

	if _, err := calls.Aggregate(); err != nil {
		logger.Errorf("failed to aggregate call to get pool data, err: %v", err)
		return nil, err
	}

	var pools = make([]entity.Pool, len(poolAndRegistries))
	for i := range poolAndRegistries {
		var reserves entity.PoolReserves
		var tokens []*entity.PoolToken
		var staticExtra = PoolTwoStaticExtra{
			LpToken: strings.ToLower(lpTokens[i].Hex()),
		}
		for j := range coins[i] {
			coinAddress := convertToEtherAddress(coins[i][j].Hex(), d.config.ChainID)
			if strings.EqualFold(coinAddress, addressZero) {
				break
			}
			precision := new(big.Int).Exp(big.NewInt(10), new(big.Int).Sub(big.NewInt(18), decimals[i][j]), nil)
			staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, precision.String())
			reserves = append(reserves, zeroString)
			tokens = append(tokens, &entity.PoolToken{
				Address:   strings.ToLower(coinAddress),
				Weight:    defaultWeight,
				Swappable: true,
			})
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.Errorf("failed to marshal static extra data, err: %v", err)
			return nil, err
		}

		pools[i] = entity.Pool{
			Address:     strings.ToLower(poolAndRegistries[i].PoolAddress.Hex()),
			Exchange:    DexTypeCurve,
			Type:        poolTypeTwo,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
		}
	}

	return pools, nil
}

func (d *PoolTracker) getNewPoolStateTypeTwo(
	ctx context.Context,
	p entity.Pool,
) (entity.Pool, error) {
	logger.Infof("[Curve] Start getting new state of pool %v with type %v", p.Address, p.Type)

	var (
		a, dExtra, gamma, feeGamma, midFee, outFee, futureAGammaTime, futureAGamma, initialAGammaTime, initialAGamma *big.Int
		lastPriceTimestamp, lpSupply, xcpProfit, virtualPrice, allowedExtraProfit, adjustmentStep, maHalfTime        *big.Int
		priceScale, priceOracle, lastPrices                                                                          *big.Int

		balances = make([]*big.Int, len(p.Tokens))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodA,
		Params: nil,
	}, []interface{}{&a})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodD,
		Params: nil,
	}, []interface{}{&dExtra})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodGamma,
		Params: nil,
	}, []interface{}{&gamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodFeeGamma,
		Params: nil,
	}, []interface{}{&feeGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodMidFee,
		Params: nil,
	}, []interface{}{&midFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodOutFee,
		Params: nil,
	}, []interface{}{&outFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodFutureAGammaTime,
		Params: nil,
	}, []interface{}{&futureAGammaTime})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodFutureAGamma,
		Params: nil,
	}, []interface{}{&futureAGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodInitialAGammaTime,
		Params: nil,
	}, []interface{}{&initialAGammaTime})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodInitialAGamma,
		Params: nil,
	}, []interface{}{&initialAGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodLastPricesTimestamp,
		Params: nil,
	}, []interface{}{&lastPriceTimestamp})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodXcpProfit,
		Params: nil,
	}, []interface{}{&xcpProfit})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodVirtualPrice,
		Params: nil,
	}, []interface{}{&virtualPrice})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodAllowedExtraProfit,
		Params: nil,
	}, []interface{}{&allowedExtraProfit})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodAdjustmentStep,
		Params: nil,
	}, []interface{}{&adjustmentStep})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodMaHalfTime,
		Params: nil,
	}, []interface{}{&maHalfTime})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodPriceScale,
		Params: nil,
	}, []interface{}{&priceScale})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodPriceOracle,
		Params: nil,
	}, []interface{}{&priceOracle})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodLastPrices,
		Params: nil,
	}, []interface{}{&lastPrices})

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
			ABI:    twoABI,
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

	var extra = PoolTwoExtra{
		A:                  a.String(),
		D:                  dExtra.String(),
		Gamma:              gamma.String(),
		FeeGamma:           feeGamma.String(),
		MidFee:             midFee.String(),
		OutFee:             outFee.String(),
		FutureAGammaTime:   futureAGammaTime.Int64(),
		FutureAGamma:       futureAGamma.String(),
		InitialAGammaTime:  initialAGammaTime.Int64(),
		InitialAGamma:      initialAGamma.String(),
		PriceScale:         priceScale.String(),
		LastPrices:         lastPrices.String(),
		PriceOracle:        priceOracle.String(),
		LpSupply:           lpSupply.String(),
		XcpProfit:          xcpProfit.String(),
		VirtualPrice:       virtualPrice.String(),
		AllowedExtraProfit: allowedExtraProfit.String(),
		AdjustmentStep:     adjustmentStep.String(),
		MaHalfTime:         maHalfTime.String(),

		LastPricesTimestamp: lastPriceTimestamp.Int64(),
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

	logger.Infof("[Curve] Finish getting new state of pool %v with type %v", p.Address, p.Type)

	return p, nil
}
