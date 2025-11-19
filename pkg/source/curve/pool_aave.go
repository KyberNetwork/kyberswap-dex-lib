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

func (d *PoolsListUpdater) getNewPoolsTypeAave(
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
			Params: []any{poolAndRegistry.PoolAddress},
		}, []any{&coins[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    poolAndRegistry.RegistryOrFactoryABI,
			Target: poolAndRegistry.RegistryOrFactoryAddress,
			Method: registryOrFactoryMethodGetUnderlyingCoins,
			Params: []any{poolAndRegistry.PoolAddress},
		}, []any{&underlyingCoins[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    poolAndRegistry.RegistryOrFactoryABI,
			Target: poolAndRegistry.RegistryOrFactoryAddress,
			Method: registryOrFactoryMethodGetUnderDecimals,
			Params: []any{poolAndRegistry.PoolAddress},
		}, []any{&decimals[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    poolAndRegistry.RegistryOrFactoryABI,
			Target: poolAndRegistry.RegistryOrFactoryAddress,
			Method: registryOrFactoryMethodGetLpToken,
			Params: []any{poolAndRegistry.PoolAddress},
		}, []any{&lpAddresses[i]})
	}
	if _, err := calls.TryAggregate(); err != nil {
		logger.Errorf("failed to aggregate call to get pool data, err: %v", err)
		return nil, err
	}

	var pools = make([]entity.Pool, 0, len(poolAndRegistries))
	for i := range poolAndRegistries {
		var reserves entity.PoolReserves
		var tokens []*entity.PoolToken
		var staticExtra = PoolAaveStaticExtra{
			LpToken:          hexutil.Encode(lpAddresses[i][:]),
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
				Swappable: true,
			})
		}
		reserves = append(reserves, zeroString)
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.Errorf("failed to marshal static extra data, err: %v", err)
			return nil, err
		}

		pools = append(pools, entity.Pool{
			Address:     hexutil.Encode(poolAndRegistries[i].PoolAddress[:]),
			Exchange:    DexTypeCurve,
			Type:        PoolTypeAave,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
		})
	}

	return pools, nil
}

func (d *PoolTracker) getNewPoolStateTypeAave(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.Infof("[Curve] Start getting new state of pool %v with type %v", p.Address, p.Type)

	var (
		initialA, futureA, initialATime, futureATime, swapFee, adminFee, offpegFee *big.Int
		lpSupply                                                                   *big.Int
		balances                                                                   = make([]*big.Int, len(p.Tokens))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    aaveABI,
		Target: p.Address,
		Method: poolMethodInitialA,
		Params: nil,
	}, []any{&initialA})

	calls.AddCall(&ethrpc.Call{
		ABI:    aaveABI,
		Target: p.Address,
		Method: poolMethodFutureA,
		Params: nil,
	}, []any{&futureA})

	calls.AddCall(&ethrpc.Call{
		ABI:    aaveABI,
		Target: p.Address,
		Method: poolMethodInitialATime,
		Params: nil,
	}, []any{&initialATime})

	calls.AddCall(&ethrpc.Call{
		ABI:    aaveABI,
		Target: p.Address,
		Method: poolMethodFutureATime,
		Params: nil,
	}, []any{&futureATime})

	calls.AddCall(&ethrpc.Call{
		ABI:    aaveABI,
		Target: p.Address,
		Method: poolMethodFee,
		Params: nil,
	}, []any{&swapFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    aaveABI,
		Target: p.Address,
		Method: poolMethodAdminFee,
		Params: nil,
	}, []any{&adminFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    aaveABI,
		Target: p.Address,
		Method: aaveMethodOffpegFeeMultiplier,
		Params: nil,
	}, []any{&offpegFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: p.GetLpToken(),
		Method: erc20MethodTotalSupply,
		Params: nil,
	}, []any{&lpSupply})

	for i := range p.Tokens {
		calls.AddCall(&ethrpc.Call{
			ABI:    aaveABI,
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

	var extra = PoolAaveExtra{
		InitialA:            initialA.String(),
		FutureA:             futureA.String(),
		InitialATime:        initialATime.Int64(),
		FutureATime:         futureATime.Int64(),
		SwapFee:             swapFee.String(),
		AdminFee:            adminFee.String(),
		OffpegFeeMultiplier: offpegFee.String(),
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

	var reserves = make(entity.PoolReserves, 0, len(balances)+1)
	for _, balance := range balances {
		reserves = append(reserves, balance.String())
	}
	reserves = append(reserves, safeCastBigIntToReserve(lpSupply))

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	logger.Infof("[Curve] Finish getting new state of pool %v with type %v", p.Address, p.Type)

	return p, nil
}
