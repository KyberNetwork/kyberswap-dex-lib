package ezeth

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdater struct {
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

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

	extra, blockNumber, err := getExtra(ctx, u.ethrpcClient)
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
	reserves := make([]string, len(extra.collaterals)+1)
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

func getExtra(ctx context.Context, ethrpcClient *ethrpc.Client) (PoolExtra, uint64, error) {
	var (
		calculateTVLsResult [3]interface{}
		calculateTVLs       struct {
			OperatorDelegatorTokenTVLs [][]*big.Int
			OperatorDelegatorTVLs      []*big.Int
			TotalTVL                   *big.Int
		}
		collateralTokenLength *big.Int
		maxDepositTVL         *big.Int
		paused                bool
		strategyManagerPaused *big.Int
		renzoOracle           common.Address

		operatorDelegatorsLength *big.Int
	)

	getPoolStateRequest := ethrpcClient.NewRequest().SetContext(ctx)

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

	// Get OperatorDelegators addresses
	operatorDelegatorsLen := operatorDelegatorsLength.Int64()
	var operatorDelegators = make([]common.Address, operatorDelegatorsLen)

	operatorDelegatorsRequest := ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
	for i := 0; i < int(operatorDelegatorsLen); i++ {
		operatorDelegatorsRequest.AddCall(&ethrpc.Call{
			ABI:    RestakeManagerABI,
			Target: RestakeManager,
			Method: RestakeManagerMethodOperatorDelegators,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&operatorDelegators[i]})
	}
	resp, err = operatorDelegatorsRequest.TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}

	// Get OperatorDelegatorAllocation & TokenStrategyMapping for each OperatorDelegator
	var (
		operatorDelegatorAllocations = make([]*big.Int, operatorDelegatorsLen)
		tokenStrategyMapping         = make([][]common.Address, operatorDelegatorsLen)
	)
	operatorDelegatorInfoRequest := ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
	for i := 0; i < int(operatorDelegatorsLen); i++ {
		operatorDelegatorInfoRequest.AddCall(&ethrpc.Call{
			ABI:    RestakeManagerABI,
			Target: RestakeManager,
			Method: RestakeManagerMethodOperatorDelegatorAllocations,
			Params: []interface{}{operatorDelegators[i]},
		}, []interface{}{&operatorDelegatorAllocations[i]})
	}

	for i := 0; i < int(operatorDelegatorsLen); i++ {
		tokenStrategyMapping[i] = make([]common.Address, collateralsLen)
		for j := 0; j < int(collateralsLen); j++ {
			operatorDelegatorInfoRequest.AddCall(&ethrpc.Call{
				ABI:    OperatorDelegatorABI,
				Target: operatorDelegators[i].String(),
				Method: OperatorDelegatorMethodTokenStrategyMapping,
				Params: []interface{}{collaterals[j]},
			}, []interface{}{&tokenStrategyMapping[i][j]})
		}
	}

	resp, err = operatorDelegatorInfoRequest.TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}

	poolExtra := PoolExtra{
		Paused:                     paused,
		OperatorDelegatorTokenTVLs: calculateTVLs.OperatorDelegatorTokenTVLs,
		OperatorDelegatorTVLs:      calculateTVLs.OperatorDelegatorTVLs,
		TotalTVL:                   calculateTVLs.TotalTVL,
		MaxDepositTVL:              maxDepositTVL,
	}

	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	return poolExtra, resp.BlockNumber.Uint64(), nil
}
