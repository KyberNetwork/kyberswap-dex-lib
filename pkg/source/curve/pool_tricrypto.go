package curve

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func (d *PoolsListUpdater) getNewPoolsTypeTricrypto(
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
			ABI:    tricryptoABI,
			Target: poolAndRegistry.PoolAddress.Hex(),
			Method: poolMethodToken,
			Params: nil,
		}, []interface{}{&lpTokens[i]})
	}

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate call to get pool data")
		return nil, err
	}

	var pools = make([]entity.Pool, len(poolAndRegistries))
	for i := range poolAndRegistries {
		var reserves entity.PoolReserves
		var tokens []*entity.PoolToken
		var staticExtra = PoolTricryptoStaticExtra{
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
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("failed to marshal static extra data")
			return nil, err
		}

		pools[i] = entity.Pool{
			Address:     strings.ToLower(poolAndRegistries[i].PoolAddress.Hex()),
			Exchange:    DexTypeCurve,
			Type:        PoolTypeTricrypto,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
		}
	}

	return pools, nil
}

// Smart contract code: https://arbiscan.io/address/0x960ea3e3c7fb317332d990873d354e18d7645590#code
func (d *PoolTracker) getNewPoolStateTypeTricrypto(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.Infof("[Curve] Start getting new state of pool %v with type %v", p.Address, p.Type)

	var (
		a, dExtra, gamma, feeGamma, midFee, outFee, futureAGammaTime, futureAGamma, initialAGammaTime, initialAGamma *big.Int

		lastPriceTimestamp, xcpProfit, virtualPrice, allowedExtraProfit, adjustmentStep, maHalfTime, lpSupply *big.Int

		balances = make([]*big.Int, len(p.Tokens))

		// These 3 slices only has length = number of tokens - 1 (check in the contract)
		priceScales  = make([]*big.Int, len(p.Tokens)-1)
		priceOracles = make([]*big.Int, len(p.Tokens)-1)
		lastPrices   = make([]*big.Int, len(p.Tokens)-1)
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    tricryptoABI,
		Target: p.Address,
		Method: poolMethodA,
		Params: nil,
	}, []interface{}{&a})

	calls.AddCall(&ethrpc.Call{
		ABI:    tricryptoABI,
		Target: p.Address,
		Method: poolMethodD,
		Params: nil,
	}, []interface{}{&dExtra})

	calls.AddCall(&ethrpc.Call{
		ABI:    tricryptoABI,
		Target: p.Address,
		Method: poolMethodGamma,
		Params: nil,
	}, []interface{}{&gamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    tricryptoABI,
		Target: p.Address,
		Method: poolMethodFeeGamma,
		Params: nil,
	}, []interface{}{&feeGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    tricryptoABI,
		Target: p.Address,
		Method: poolMethodMidFee,
		Params: nil,
	}, []interface{}{&midFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    tricryptoABI,
		Target: p.Address,
		Method: poolMethodOutFee,
		Params: nil,
	}, []interface{}{&outFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    tricryptoABI,
		Target: p.Address,
		Method: poolMethodFutureAGammaTime,
		Params: nil,
	}, []interface{}{&futureAGammaTime})

	calls.AddCall(&ethrpc.Call{
		ABI:    tricryptoABI,
		Target: p.Address,
		Method: poolMethodFutureAGamma,
		Params: nil,
	}, []interface{}{&futureAGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    tricryptoABI,
		Target: p.Address,
		Method: poolMethodInitialAGammaTime,
		Params: nil,
	}, []interface{}{&initialAGammaTime})

	calls.AddCall(&ethrpc.Call{
		ABI:    tricryptoABI,
		Target: p.Address,
		Method: poolMethodInitialAGamma,
		Params: nil,
	}, []interface{}{&initialAGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    tricryptoABI,
		Target: p.Address,
		Method: poolMethodLastPricesTimestamp,
		Params: nil,
	}, []interface{}{&lastPriceTimestamp})

	calls.AddCall(&ethrpc.Call{
		ABI:    tricryptoABI,
		Target: p.Address,
		Method: poolMethodXcpProfit,
		Params: nil,
	}, []interface{}{&xcpProfit})

	calls.AddCall(&ethrpc.Call{
		ABI:    tricryptoABI,
		Target: p.Address,
		Method: poolMethodVirtualPrice,
		Params: nil,
	}, []interface{}{&virtualPrice})

	calls.AddCall(&ethrpc.Call{
		ABI:    tricryptoABI,
		Target: p.Address,
		Method: poolMethodAllowedExtraProfit,
		Params: nil,
	}, []interface{}{&allowedExtraProfit})

	calls.AddCall(&ethrpc.Call{
		ABI:    tricryptoABI,
		Target: p.Address,
		Method: poolMethodAdjustmentStep,
		Params: nil,
	}, []interface{}{&adjustmentStep})

	calls.AddCall(&ethrpc.Call{
		ABI:    tricryptoABI,
		Target: p.Address,
		Method: poolMethodMaHalfTime,
		Params: nil,
	}, []interface{}{&maHalfTime})

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
			ABI:    tricryptoABI,
			Target: p.Address,
			Method: poolMethodBalances,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&balances[i]})
	}

	for i := 0; i < len(p.Tokens)-1; i++ {
		calls.AddCall(&ethrpc.Call{
			ABI:    tricryptoABI,
			Target: p.Address,
			Method: poolMethodPriceScale,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&priceScales[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    tricryptoABI,
			Target: p.Address,
			Method: poolMethodPriceOracle,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&priceOracles[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    tricryptoABI,
			Target: p.Address,
			Method: poolMethodLastPrices,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&lastPrices[i]})
	}

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"poolType":    p.Type,
			"error":       err,
		}).Errorf("failed to aggregate call pool data")
		return entity.Pool{}, err
	}

	var reserves entity.PoolReserves = lo.Map(balances, func(value *big.Int, _ int) string {
		return value.String()
	})
	priceScalesStr := lo.Map(priceScales, func(value *big.Int, _ int) string {
		return value.String()
	})
	priceOraclesStr := lo.Map(priceOracles, func(value *big.Int, _ int) string {
		return value.String()
	})
	lastPricesStr := lo.Map(lastPrices, func(value *big.Int, _ int) string {
		return value.String()
	})

	var extra = PoolTricryptoExtra{
		A:                   a.String(),
		D:                   dExtra.String(),
		Gamma:               gamma.String(),
		FeeGamma:            feeGamma.String(),
		MidFee:              midFee.String(),
		OutFee:              outFee.String(),
		FutureAGammaTime:    futureAGammaTime.Int64(),
		FutureAGamma:        futureAGamma.String(),
		InitialAGammaTime:   initialAGammaTime.Int64(),
		InitialAGamma:       initialAGamma.String(),
		LastPricesTimestamp: lastPriceTimestamp.Int64(),
		XcpProfit:           xcpProfit.String(),
		VirtualPrice:        virtualPrice.String(),
		AllowedExtraProfit:  allowedExtraProfit.String(),
		AdjustmentStep:      adjustmentStep.String(),
		MaHalfTime:          maHalfTime.String(),

		PriceScale:  priceScalesStr,
		LastPrices:  lastPricesStr,
		PriceOracle: priceOraclesStr,
		LpSupply:    lpSupply.String(),
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
