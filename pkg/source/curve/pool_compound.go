package curve

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

func (d *PoolsListUpdater) getNewPoolsTypeCompound(
	ctx context.Context,
	poolAndRegistries []PoolAndRegistries,
) ([]entity.Pool, error) {
	var (
		coins           = make([][8]common.Address, len(poolAndRegistries))
		underlyingCoins = make([][8]common.Address, len(poolAndRegistries))
		decimals        = make([][8]*big.Int, len(poolAndRegistries))
		lpAddresses     = make([]common.Address, len(poolAndRegistries))
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
			Method: registryOrFactoryMethodGetUnderlyingCoins,
			Params: []interface{}{poolAndRegistry.PoolAddress},
		}, []interface{}{&underlyingCoins[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    poolAndRegistry.RegistryOrFactoryABI,
			Target: poolAndRegistry.RegistryOrFactoryAddress,
			Method: registryOrFactoryMethodGetUnderDecimals,
			Params: []interface{}{poolAndRegistry.PoolAddress},
		}, []interface{}{&decimals[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    poolAndRegistry.RegistryOrFactoryABI,
			Target: poolAndRegistry.RegistryOrFactoryAddress,
			Method: registryOrFactoryMethodGetLpToken,
			Params: []interface{}{poolAndRegistry.PoolAddress},
		}, []interface{}{&lpAddresses[i]})
	}
	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate call to get pool data")
		return nil, err
	}

	var pools = make([]entity.Pool, len(poolAndRegistries))
	for i := range poolAndRegistries {
		var reserves entity.PoolReserves
		var tokens []*entity.PoolToken
		var staticExtra = PoolCompoundStaticExtra{
			LpToken:          strings.ToLower(lpAddresses[i].Hex()),
			UnderlyingTokens: extractNonZeroAddressesToStrings(underlyingCoins[i]),
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
			Type:        PoolTypeCompound,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
		}
	}

	return pools, nil
}

func (d *PoolTracker) getNewPoolStateTypeCompound(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.Infof("[Curve] Start getting new state of pool %v with type %v", p.Address, p.Type)

	var (
		a, swapFee, adminFee *big.Int
		rates8               [8]*big.Int
		balances             = make([]*big.Int, len(p.Tokens))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    baseABI,
		Target: p.Address,
		Method: poolMethodA,
		Params: nil,
	}, []interface{}{&a})

	calls.AddCall(&ethrpc.Call{
		ABI:    compoundABI,
		Target: p.Address,
		Method: poolMethodFee,
		Params: nil,
	}, []interface{}{&swapFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    compoundABI,
		Target: p.Address,
		Method: poolMethodAdminFee,
		Params: nil,
	}, []interface{}{&adminFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    mainRegistryABI,
		Target: d.config.MainRegistryAddress,
		Method: registryOrFactoryMethodGetRates,
		Params: []interface{}{common.HexToAddress(p.Address)},
	}, []interface{}{&rates8})

	for i := range p.Tokens {
		calls.AddCall(&ethrpc.Call{
			ABI:    compoundABI,
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

	var rates = make([]string, len(p.Tokens))
	for i := range p.Tokens {
		if rates8[i] == zeroBI {
			logger.WithFields(logger.Fields{
				"poolAddress":  p.Address,
				"poolType":     p.Type,
				"tokenAddress": p.Tokens[i].Address,
			}).Errorf("token has no rate")
			return entity.Pool{}, errors.New("token has no rate")
		}
		rates[i] = rates8[i].String()
	}

	var extra = PoolCompoundExtra{
		A:        a.String(),
		SwapFee:  swapFee.String(),
		AdminFee: adminFee.String(),
		Rates:    rates,
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

	var reserves = make(entity.PoolReserves, len(balances))
	for i := range balances {
		reserves[i] = balances[i].String()
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	logger.Infof("[Curve] Finish getting new state of pool %v with type %v", p.Address, p.Type)

	return p, nil
}
