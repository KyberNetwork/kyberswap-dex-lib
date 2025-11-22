package curve

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

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
			Params: []any{poolAndRegistry.PoolAddress},
		}, []any{&coins[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    poolAndRegistry.RegistryOrFactoryABI,
			Target: poolAndRegistry.RegistryOrFactoryAddress,
			Method: registryOrFactoryMethodGetDecimals,
			Params: []any{poolAndRegistry.PoolAddress},
		}, []any{&decimals[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    twoABI,
			Target: poolAndRegistry.PoolAddress.Hex(),
			Method: poolMethodToken,
			Params: nil,
		}, []any{&lpTokens[i]})
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
			LpToken: hexutil.Encode(lpTokens[i][:]),
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
				Swappable: true,
			})
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.Errorf("failed to marshal static extra data, err: %v", err)
			return nil, err
		}

		pools[i] = entity.Pool{
			Address:     hexutil.Encode(poolAndRegistries[i].PoolAddress[:]),
			Exchange:    DexTypeCurve,
			Type:        PoolTypeTwo,
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
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.Infof("[Curve] Start getting new state of pool %v with type %v", p.Address, p.Type)

	var (
		a, dExtra, gamma, feeGamma, midFee, outFee, futureAGammaTime, futureAGamma, initialAGammaTime, initialAGamma *big.Int
		lastPriceTimestamp, lpSupply, xcpProfit, virtualPrice, allowedExtraProfit, adjustmentStep, maHalfTime        *big.Int
		priceScale, priceOracle, lastPrices                                                                          *big.Int

		balances = make([]*big.Int, len(p.Tokens))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodA,
		Params: nil,
	}, []any{&a})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodD,
		Params: nil,
	}, []any{&dExtra})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodGamma,
		Params: nil,
	}, []any{&gamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodFeeGamma,
		Params: nil,
	}, []any{&feeGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodMidFee,
		Params: nil,
	}, []any{&midFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodOutFee,
		Params: nil,
	}, []any{&outFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodFutureAGammaTime,
		Params: nil,
	}, []any{&futureAGammaTime})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodFutureAGamma,
		Params: nil,
	}, []any{&futureAGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodInitialAGammaTime,
		Params: nil,
	}, []any{&initialAGammaTime})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodInitialAGamma,
		Params: nil,
	}, []any{&initialAGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodLastPricesTimestamp,
		Params: nil,
	}, []any{&lastPriceTimestamp})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodXcpProfit,
		Params: nil,
	}, []any{&xcpProfit})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodVirtualPrice,
		Params: nil,
	}, []any{&virtualPrice})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodAllowedExtraProfit,
		Params: nil,
	}, []any{&allowedExtraProfit})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodAdjustmentStep,
		Params: nil,
	}, []any{&adjustmentStep})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodMaHalfTime,
		Params: nil,
	}, []any{&maHalfTime})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodPriceScale,
		Params: nil,
	}, []any{&priceScale})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodPriceOracle,
		Params: nil,
	}, []any{&priceOracle})

	calls.AddCall(&ethrpc.Call{
		ABI:    twoABI,
		Target: p.Address,
		Method: poolMethodLastPrices,
		Params: nil,
	}, []any{&lastPrices})

	lpToken := p.GetLpToken()
	if len(lpToken) > 0 {
		calls.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: lpToken,
			Method: erc20MethodTotalSupply,
			Params: nil,
		}, []any{&lpSupply})
	}

	for i := range p.Tokens {
		calls.AddCall(&ethrpc.Call{
			ABI:    twoABI,
			Target: p.Address,
			Method: poolMethodBalances,
			Params: []any{big.NewInt(int64(i))},
		}, []any{&balances[i]})
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
