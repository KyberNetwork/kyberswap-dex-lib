package curve

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func (d *PoolsListUpdater) getNewPoolsTypeTricrypto(
	ctx context.Context,
	poolAndRegistries []PoolAndRegistries,
) ([]entity.Pool, error) {
	var (
		coins    = make([][8]common.Address, len(poolAndRegistries))
		decimals = make([][8]*big.Int, len(poolAndRegistries))
		lpTokens = make([]common.Address, len(poolAndRegistries))

		maHalfTime         = make([]*big.Int, len(poolAndRegistries))
		allowedExtraProfit = make([]*big.Int, len(poolAndRegistries))
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

		calls.AddCall(&ethrpc.Call{
			ABI:    tricryptoABI,
			Target: poolAndRegistry.PoolAddress.Hex(),
			Method: poolMethodMaHalfTime,
			Params: nil,
		}, []interface{}{&maHalfTime[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    tricryptoABI,
			Target: poolAndRegistry.PoolAddress.Hex(),
			Method: poolMethodAllowedExtraProfit,
			Params: nil,
		}, []interface{}{&allowedExtraProfit[i]})
	}

	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate call to get pool data")
		return nil, err
	}

	var pools = make([]entity.Pool, 0, len(poolAndRegistries))
	for i := range poolAndRegistries {
		if allowedExtraProfit[i] == nil || allowedExtraProfit[i].Cmp(bignumber.ZeroBI) == 0 {
			// ignore old tricrypto pool with hardcoded allowed_extra_profit
			// for example https://etherscan.io/address/0x80466c64868e1ab14a1ddf27a676c3fcbe638fe5#readContract
			logger.WithFields(logger.Fields{
				"poolAddress": poolAndRegistries[i].PoolAddress,
			}).Warn("ignore pool without allowed_extra_profit")
			continue
		}

		var reserves entity.PoolReserves
		var tokens []*entity.PoolToken

		// tricrypto-ng is a special version optimized for ETH
		isTricryptoNg := maHalfTime[i] == nil || maHalfTime[i].Cmp(bignumber.ZeroBI) == 0
		var staticExtra = PoolTricryptoStaticExtra{
			LpToken:       strings.ToLower(lpTokens[i].Hex()),
			IsTricryptoNg: isTricryptoNg,
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

		pools = append(pools, entity.Pool{
			Address:     strings.ToLower(poolAndRegistries[i].PoolAddress.Hex()),
			Exchange:    DexTypeCurve,
			Type:        poolTypeTricrypto,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
		})
	}

	return pools, nil
}

// Smart contract code: https://arbiscan.io/address/0x960ea3e3c7fb317332d990873d354e18d7645590#code
func (d *PoolTracker) getNewPoolStateTypeTricrypto(
	ctx context.Context,
	p entity.Pool,
) (entity.Pool, error) {
	logger.Infof("[Curve] Start getting new state of pool %v with type %v", p.Address, p.Type)

	var (
		a, dExtra, gamma, feeGamma, midFee, outFee, futureAGammaTime, futureAGamma, initialAGammaTime, initialAGamma *big.Int

		lastPriceTimestamp, xcpProfit, virtualPrice, allowedExtraProfit, adjustmentStep, maHalfTime, maTime, lpSupply *big.Int

		balances = make([]*big.Int, len(p.Tokens))

		// These 3 slices only has length = number of tokens - 1 (check in the contract)
		priceScales  = make([]*big.Int, len(p.Tokens)-1)
		priceOracles = make([]*big.Int, len(p.Tokens)-1)
		lastPrices   = make([]*big.Int, len(p.Tokens)-1)
	)

	var staticExtra PoolTricryptoStaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"poolType":    p.Type,
			"error":       err,
		}).Errorf("failed to unmarshal static extra data")
		return entity.Pool{}, err
	}

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

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

	if staticExtra.IsTricryptoNg {
		calls.AddCall(&ethrpc.Call{
			ABI:    tricryptoABI,
			Target: p.Address,
			Method: poolMethodMaTime,
			Params: nil,
		}, []interface{}{&maTime})
	} else {
		calls.AddCall(&ethrpc.Call{
			ABI:    tricryptoABI,
			Target: p.Address,
			Method: poolMethodMaHalfTime,
			Params: nil,
		}, []interface{}{&maHalfTime})
	}

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

	var maHalfTimeS string
	if staticExtra.IsTricryptoNg {
		maHalfTimeS = maTime.String()
	} else {
		maHalfTimeS = maHalfTime.String()
	}

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
		MaHalfTime:          maHalfTimeS,

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

	// debug to get testcase data
	if strings.EqualFold(p.Address, "0xf5f5b97624542d72a9e06f04804bf81baa15e2b4") {
		x, _ := json.Marshal(p)
		fmt.Println("---", p.Address)
		fmt.Println("---", string(x))

		var r, r1 *big.Int
		request := d.ethrpcClient.NewRequest().
			AddCall(&ethrpc.Call{
				ABI:    tricryptoABI,
				Target: p.Address,
				Method: "get_dy",
				Params: []interface{}{big.NewInt(0), big.NewInt(2), big.NewInt(1)},
			}, []interface{}{&r}).
			AddCall(&ethrpc.Call{
				ABI:    tricryptoABI,
				Target: p.Address,
				Method: "get_dy",
				Params: []interface{}{big.NewInt(1), big.NewInt(2), big.NewInt(1)},
			}, []interface{}{&r1})

		if _, err := request.Aggregate(); err != nil {
			fmt.Println("err---", err)
		}
		fmt.Println("amount out---", r.String(), r1.String())
	}
	// end of debug

	logger.Infof("[Curve] Finish getting new state of pool %v with type %v", p.Address, p.Type)

	return p, nil
}
