package ezeth

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
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdater struct {
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

var _ = poollist.RegisterFactoryE(DexType, NewPoolListUpdater)

func NewPoolListUpdater(
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	return &PoolListUpdater{
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}

	startTime := time.Now()
	u.hasInitialized = true

	extra, blockNumber, err := getExtra(ctx, u.ethrpcClient, nil)
	if err != nil {
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, nil, err
	}

	tokens := []*entity.PoolToken{
		{
			Address:   strings.ToLower(EzEthToken),
			Symbol:    "ezETH",
			Decimals:  18,
			Name:      "Renzo Restaked ETH",
			Swappable: true,
		},
		{
			Address:   strings.ToLower(WETH),
			Symbol:    "WETH",
			Decimals:  18,
			Name:      "Wrapped Ether",
			Swappable: true,
		},
	}
	tokens = append(tokens, extra.collaterals...)
	reserves := make([]string, len(extra.collaterals)+2)
	for i := 0; i < len(reserves); i++ {
		reserves[i] = defaultReserves
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      DexType,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return []entity.Pool{
		{
			Address:     strings.ToLower(RestakeManager),
			Exchange:    string(valueobject.ExchangeRenzoEZETH),
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
			BlockNumber: blockNumber,
			Extra:       string(extraBytes),
		},
	}, nil, nil
}

func getExtra(
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	overrides map[common.Address]gethclient.OverrideAccount,
) (PoolExtra, uint64, error) {
	var (
		calculateTVLsResult [3]interface{}
		calculateTVLs       struct {
			OperatorDelegatorTokenTVLs [][]*big.Int
			OperatorDelegatorTVLs      []*big.Int
			TotalTVL                   *big.Int
		}
		collateralTokenLength *big.Int
		totalSupply           *big.Int
		maxDepositTVL         *big.Int
		paused                bool
		strategyManagerPaused *big.Int
		renzoOracle           common.Address

		operatorDelegatorsLength *big.Int
	)

	getPoolStateRequest := ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		getPoolStateRequest.SetOverrides(overrides)
	}

	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    RestakeManagerABI,
		Target: RestakeManager,
		Method: RestakeManagerMethodCalculateTVLs,
		Params: []interface{}{},
	}, []interface{}{&calculateTVLsResult})
	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    RestakeManagerABI,
		Target: RestakeManager,
		Method: RestakeManagerMethodGetCollateralTokensLength,
		Params: []interface{}{},
	}, []interface{}{&collateralTokenLength})
	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    EzETHTokenABI,
		Target: EzEthToken,
		Method: EzEthTokenMethodTotalSupply,
		Params: []interface{}{},
	}, []interface{}{&totalSupply})
	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    RestakeManagerABI,
		Target: RestakeManager,
		Method: RestakeManagerMethodMaxDepositTVL,
		Params: []interface{}{},
	}, []interface{}{&maxDepositTVL})
	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    RestakeManagerABI,
		Target: RestakeManager,
		Method: RestakeManagerMethodPaused,
		Params: []interface{}{},
	}, []interface{}{&paused})
	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    RestakeManagerABI,
		Target: RestakeManager,
		Method: RestakeManagerMethodRenzoOracle,
		Params: []interface{}{},
	}, []interface{}{&renzoOracle})
	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    RestakeManagerABI,
		Target: RestakeManager,
		Method: RestakeManagerMethodGetOperatorDelegatorsLength,
		Params: []interface{}{},
	}, []interface{}{&operatorDelegatorsLength})
	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    StrategyManagerABI,
		Target: StrategyManager,
		Method: StrategyManagerMethodPaused,
		Params: []interface{}{},
	}, []interface{}{&strategyManagerPaused})

	resp, err := getPoolStateRequest.TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}
	calculateTVLs.OperatorDelegatorTokenTVLs = calculateTVLsResult[0].([][]*big.Int)
	calculateTVLs.OperatorDelegatorTVLs = calculateTVLsResult[1].([]*big.Int)
	calculateTVLs.TotalTVL = calculateTVLsResult[2].(*big.Int)

	collateralsLen := collateralTokenLength.Int64()

	var (
		collaterals = make([]common.Address, collateralsLen)
	)

	getCollateralsRequest := ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
	if overrides != nil {
		getCollateralsRequest.SetOverrides(overrides)
	}

	for i := 0; i < int(collateralsLen); i++ {
		getCollateralsRequest.AddCall(&ethrpc.Call{
			ABI:    RestakeManagerABI,
			Target: RestakeManager,
			Method: RestakeManagerMethodCollateralTokens,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&collaterals[i]})
	}
	resp, err = getCollateralsRequest.TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}

	// Get OperatorDelegators & Oracle addresses
	var (
		operatorDelegatorsLen    = operatorDelegatorsLength.Int64()
		operatorDelegators       = make([]common.Address, operatorDelegatorsLen)
		tokenOracleAddresses     = make([]common.Address, len(collaterals))
		collateralTokenTvlLimits = make([]*big.Int, len(collaterals))
	)

	operatorDelegatorsRequest := ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
	if overrides != nil {
		operatorDelegatorsRequest.SetOverrides(overrides)
	}

	for i := 0; i < int(operatorDelegatorsLen); i++ {
		operatorDelegatorsRequest.AddCall(&ethrpc.Call{
			ABI:    RestakeManagerABI,
			Target: RestakeManager,
			Method: RestakeManagerMethodOperatorDelegators,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&operatorDelegators[i]})
	}
	for i := 0; i < len(collaterals); i++ {
		operatorDelegatorsRequest.AddCall(&ethrpc.Call{
			ABI:    RenzoOracleABI,
			Target: renzoOracle.Hex(),
			Method: RenzoOracleMethodTokenOracleLookUp,
			Params: []interface{}{collaterals[i]},
		}, []interface{}{&tokenOracleAddresses[i]})
		operatorDelegatorsRequest.AddCall(&ethrpc.Call{
			ABI:    RestakeManagerABI,
			Target: RestakeManager,
			Method: RestakeManagerMethodCollateralTokenTvlLimits,
			Params: []interface{}{collaterals[i]},
		}, []interface{}{&collateralTokenTvlLimits[i]})
	}
	resp, err = operatorDelegatorsRequest.TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}

	// 1. Get OperatorDelegatorAllocation & TokenStrategyMapping for each OperatorDelegator
	// 2. Get TokenOracle.latestRoundData for each TokenOracleAddress
	var (
		operatorDelegatorAllocations = make([]*big.Int, operatorDelegatorsLen)
		tokenStrategies              = make([][]common.Address, operatorDelegatorsLen)
		oracleInfo                   = make([]Oracle, len(tokenOracleAddresses))
	)
	operatorDelegatorInfoRequest := ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
	if overrides != nil {
		operatorDelegatorInfoRequest.SetOverrides(overrides)
	}

	for i := 0; i < int(operatorDelegatorsLen); i++ {
		operatorDelegatorInfoRequest.AddCall(&ethrpc.Call{
			ABI:    RestakeManagerABI,
			Target: RestakeManager,
			Method: RestakeManagerMethodOperatorDelegatorAllocations,
			Params: []interface{}{operatorDelegators[i]},
		}, []interface{}{&operatorDelegatorAllocations[i]})
	}

	for i := 0; i < int(operatorDelegatorsLen); i++ {
		tokenStrategies[i] = make([]common.Address, collateralsLen)
		for j := 0; j < int(collateralsLen); j++ {
			operatorDelegatorInfoRequest.AddCall(&ethrpc.Call{
				ABI:    OperatorDelegatorABI,
				Target: operatorDelegators[i].String(),
				Method: OperatorDelegatorMethodTokenStrategyMapping,
				Params: []interface{}{collaterals[j]},
			}, []interface{}{&tokenStrategies[i][j]})
		}
	}

	for i := 0; i < len(tokenOracleAddresses); i++ {
		operatorDelegatorInfoRequest.AddCall(&ethrpc.Call{
			ABI:    TokenOracleABI,
			Target: tokenOracleAddresses[i].String(),
			Method: TokenOracleMethodLatestRoundData,
			Params: []interface{}{},
		}, []interface{}{&oracleInfo[i]})
	}

	resp, err = operatorDelegatorInfoRequest.TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}

	tokenStrategyMapping := make([]map[string]bool, operatorDelegatorsLen)
	for i := 0; i < int(operatorDelegatorsLen); i++ {
		tokenStrategyMapping[i] = map[string]bool{}
		for j := 0; j < len(tokenStrategies[i]); j++ {
			collateral := strings.ToLower(collaterals[j].Hex())
			address := tokenStrategies[i][j]
			hasTokenStrategyMapping := address.Hex() == valueobject.ZeroAddress
			tokenStrategyMapping[i][collateral] = hasTokenStrategyMapping
		}
	}

	var (
		collateralTokenIndex        = make(map[string]int, len(collaterals))
		tokenOracleLookup           = make(map[string]Oracle, len(collaterals))
		collateralTokenTvlLimitsMap = make(map[string]*big.Int, len(collaterals))
	)
	for i := 0; i < len(collaterals); i++ {
		address := strings.ToLower(collaterals[i].Hex())
		collateralTokenIndex[address] = i
		tokenOracleLookup[address] = oracleInfo[i]
		collateralTokenTvlLimitsMap[address] = collateralTokenTvlLimits[i]
	}

	poolExtra := PoolExtra{
		Paused:                       paused,
		StrategyManagerPaused:        strategyManagerPaused.Cmp(bignumber.One) > 0,
		CollateralTokenIndex:         collateralTokenIndex,
		OperatorDelegatorTokenTVLs:   calculateTVLs.OperatorDelegatorTokenTVLs,
		OperatorDelegatorTVLs:        calculateTVLs.OperatorDelegatorTVLs,
		TotalTVL:                     calculateTVLs.TotalTVL,
		OperatorDelegatorAllocations: operatorDelegatorAllocations,
		TokenStrategyMapping:         tokenStrategyMapping,
		TotalSupply:                  totalSupply,
		MaxDepositTVL:                maxDepositTVL,
		TokenOracleLookup:            tokenOracleLookup,
		CollateralTokenTvlLimits:     collateralTokenTvlLimitsMap,
		collaterals: lo.Map(collaterals,
			func(item common.Address, _ int) *entity.PoolToken {
				return &entity.PoolToken{
					Address:   strings.ToLower(item.Hex()),
					Swappable: true,
				}
			}),
	}

	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	return poolExtra, resp.BlockNumber.Uint64(), nil
}
