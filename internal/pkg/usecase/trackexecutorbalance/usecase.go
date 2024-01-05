package trackexecutor

import (
	"context"
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/machinebox/graphql"
	"github.com/samber/lo"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/ethrpc"
	graphqlPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type useCase struct {
	ethClient                 *ethrpc.Client
	graphQLClient             *graphql.Client
	poolFactory               IPoolFactory
	poolRepository            IPoolRepository
	executorBalanceRepository IExecutorBalanceRepository
	config                    Config
}

func NewUseCase(
	ethClient *ethrpc.Client,
	poolFactory IPoolFactory,
	poolRepository IPoolRepository,
	executorBalanceRepository IExecutorBalanceRepository,
	config Config,
) *useCase {
	graphQLClient := graphqlPkg.NewWithTimeout(config.SubgraphURL, graphQLRequestTimeout)

	return &useCase{
		ethClient:                 ethClient,
		graphQLClient:             graphQLClient,
		poolFactory:               poolFactory,
		poolRepository:            poolRepository,
		executorBalanceRepository: executorBalanceRepository,
		config:                    config,
	}
}

func (u *useCase) Handle(ctx context.Context) error {
	var wg sync.WaitGroup
	for _, executorAddress := range u.config.ExecutorAddresses {
		wg.Add(1)
		go func(executorAddress string) {
			defer wg.Done()

			err := u.trackExecutor(ctx, executorAddress)
			if err != nil {
				logger.Errorf("fail to track executor %s, err: %v", executorAddress, err)
			}
		}(executorAddress)
	}
	wg.Wait()

	return nil
}

func (u *useCase) trackExecutor(ctx context.Context, executorAddress string) error {
	executorAddress = strings.ToLower(executorAddress)
	blockNumberCheckpoint, err := fetchLatestEventBlockNumber(ctx, u.graphQLClient, executorAddress)
	if err != nil {
		return err
	}

	for {
		blockNumber, err := u.executorBalanceRepository.GetLatestProcessedBlockNumber(executorAddress)
		if err != nil {
			return err
		}

		events, err := fetchNewExecutorExchangeEvents(ctx, u.graphQLClient, executorAddress, blockNumber)
		if err != nil {
			return err
		}
		logger.WithFields(logger.Fields{
			"executor":         executorAddress,
			"startBlockNumber": blockNumber,
			"numEvents":        len(events),
		}).Info("Fetch Exchange events from executor")

		if len(events) == 0 {
			logger.Info("No new Exchange events, skip to the next interval")
			return nil
		}

		if err := u.trackExecutorBalance(ctx, executorAddress, events); err != nil {
			return err
		}
		if err := u.trackExecutorPoolApproval(ctx, executorAddress, events); err != nil {
			return err
		}

		// Persist new block number into data store.
		lastBlockNumberStr := events[len(events)-1].BlockNumber
		lastBlockNumber, err := strconv.ParseUint(lastBlockNumberStr, 10, 64)
		if err != nil {
			return err
		}
		err = u.executorBalanceRepository.UpdateLatestProcessedBlockNumber(executorAddress, lastBlockNumber)
		if err != nil {
			return err
		}

		if lastBlockNumber >= blockNumberCheckpoint {
			break
		} else {
			logger.WithFields(logger.Fields{
				"currentBlock": lastBlockNumber,
				"latestBlock":  blockNumberCheckpoint,
			}).Info("Continue catching up with the latest events")
			time.Sleep(intervalDelay)
		}
	}

	return nil
}

func (u *useCase) trackExecutorBalance(ctx context.Context, executorAddress string, events []ExchangeEvent) error {
	tokenOutSet := mapset.NewSet[string]()
	for _, event := range events {
		if event.Token == EtherAddress {
			tokenOutSet.Add(u.config.GasTokenAddress)
			continue
		}
		tokenOutSet.Add(event.Token)
	}
	tokenOutSlice := tokenOutSet.ToSlice() // To persist the index of elements.

	existed, err := u.executorBalanceRepository.HasToken(executorAddress, tokenOutSlice)
	if err != nil {
		return err
	}

	candidateTokenOuts := lo.Filter(tokenOutSlice, func(item string, index int) bool {
		return !existed[index]
	})

	if len(candidateTokenOuts) == 0 {
		logger.Info("No new tokens to track, skip to the next interval")
		return nil
	}

	rpcRequest := u.ethClient.NewRequest()
	rpcRequest.SetContext(ctx)

	tokenBalances := make([]*big.Int, len(candidateTokenOuts))
	for i, token := range candidateTokenOuts {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: token,
			Method: erc20MethodGetBalanceOf,
			Params: []interface{}{common.HexToAddress(executorAddress)},
		}, []interface{}{&tokenBalances[i]})
	}
	rpcResponse, err := rpcRequest.TryAggregate()
	if err != nil {
		return err
	}

	updateTokens := lo.Filter(candidateTokenOuts, func(item string, idx int) bool {
		return rpcResponse.Result[idx] && tokenBalances[idx] != nil && tokenBalances[idx].Cmp(integer.Zero()) > 0
	})

	// missTokens are tokens that have Exchange events,
	// however executor still does not have 1 wei of them.
	missTokens := lo.Filter(candidateTokenOuts, func(item string, idx int) bool {
		return !rpcResponse.Result[idx] || tokenBalances[idx] == nil || tokenBalances[idx].Cmp(integer.Zero()) == 0
	})

	logger.WithFields(logger.Fields{
		"executor":        executorAddress,
		"numUpdateTokens": len(updateTokens),
		"missTokens":      missTokens,
	}).Info("Add tokens that executor has balance")

	if err := u.executorBalanceRepository.AddToken(executorAddress, updateTokens); err != nil {
		return err
	}

	return nil
}

func (u *useCase) trackExecutorPoolApproval(ctx context.Context, executorAddress string, events []ExchangeEvent) error {
	poolAddressSet := mapset.NewSet[string]()
	for _, event := range events {
		poolAddressSet.Add(event.Pair)
	}
	poolAddresses := poolAddressSet.ToSlice()

	poolEntities, err := u.poolRepository.FindByAddresses(ctx, poolAddresses)
	if err != nil {
		return err
	}

	poolInfo := map[string]*PoolInfo{}
	for _, poolEntity := range poolEntities {
		if _, ok := poolInfo[poolEntity.Address]; !ok {
			poolInfo[poolEntity.Address] = &PoolInfo{}
		}
		poolInfo[poolEntity.Address].entity = poolEntity
	}

	poolSimulators := u.poolFactory.NewPools(ctx, poolEntities, common.Hash{})
	logger.WithFields(logger.Fields{
		"executor":          executorAddress,
		"numPoolAddresses":  len(poolAddresses),
		"numPoolEntities":   len(poolEntities),
		"numPoolSimulators": len(poolSimulators),
	}).Info("Fetch pool info")
	for _, poolSimulator := range poolSimulators {
		if _, ok := poolInfo[poolSimulator.GetAddress()]; !ok {
			poolInfo[poolSimulator.GetAddress()] = &PoolInfo{}
		}
		poolInfo[poolSimulator.GetAddress()].simulator = poolSimulator
	}

	poolApprovalSet := mapset.NewSet[dto.PoolApprovalQuery]()
	for _, pool := range poolInfo {
		if pool.entity == nil || pool.simulator == nil {
			continue
		}

		if !valueobject.IsApproveMaxExchange(valueobject.Exchange(pool.entity.Exchange)) {
			continue
		}

		for _, token := range pool.simulator.GetTokens() {
			approveAddress, err := getAddressToApproveMax(pool)
			if err != nil {
				continue
			}

			poolApprovalSet.Add(dto.PoolApprovalQuery{
				TokenIn:     token,
				PoolAddress: approveAddress,
			})
		}
	}

	poolApprovalQueries := poolApprovalSet.ToSlice()
	existed, err := u.executorBalanceRepository.HasPoolApproval(executorAddress, poolApprovalQueries)
	if err != nil {
		return err
	}

	poolApprovalCandidates := lo.Filter(poolApprovalQueries, func(item dto.PoolApprovalQuery, index int) bool {
		return !existed[index]
	})

	if len(poolApprovalCandidates) == 0 {
		logger.Info("No new pool approvals to track, skip to the next interval")
		return nil
	}

	rpcRequest := u.ethClient.NewRequest()
	rpcRequest.SetContext(ctx)

	poolApprovals := make([]*big.Int, len(poolApprovalCandidates))
	for i, query := range poolApprovalCandidates {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: query.TokenIn,
			Method: erc20MethodGetAllowance,
			Params: []interface{}{
				common.HexToAddress(executorAddress),
				common.HexToAddress(query.PoolAddress),
			},
		}, []interface{}{&poolApprovals[i]})
	}
	rpcResponse, err := rpcRequest.TryAggregate()
	if err != nil {
		return err
	}

	updatePoolApprovals := lo.Filter(poolApprovalCandidates, func(item dto.PoolApprovalQuery, idx int) bool {
		return rpcResponse.Result[idx] && poolApprovals[idx] != nil && poolApprovals[idx].Cmp(integer.Zero()) > 0
	})

	logger.WithFields(logger.Fields{
		"executor":               executorAddress,
		"numUpdatePoolApprovals": len(updatePoolApprovals),
		"numMissPoolApprovals":   len(poolApprovalCandidates) - len(updatePoolApprovals),
	}).Info("Add pool approvals for executor")

	if err := u.executorBalanceRepository.ApprovePool(executorAddress, updatePoolApprovals); err != nil {
		return err
	}

	return nil
}

// For some dexes, instead of approving max for pool address,
// executor approves max for other address (e.g: vault address).
// `getAddressToApproveMax` receives the pool info, then return the address
// which executor should approve max for. By default, it returns pool address.
func getAddressToApproveMax(pool *PoolInfo) (string, error) {
	switch valueobject.Exchange(pool.simulator.GetExchange()) {
	case
		valueobject.ExchangeBalancerV2Weighted,
		valueobject.ExchangeBalancerV2Stable,
		valueobject.ExchangeBalancerV2ComposableStable,
		valueobject.ExchangeBeethovenXWeighted,
		valueobject.ExchangeBeethovenXStable,
		valueobject.ExchangeBeethovenXComposableStable,
		valueobject.ExchangeVelocoreV2CPMM,
		valueobject.ExchangeVelocoreV2WombatStable:
		{
			var staticExtra struct {
				Vault string `json:"vault"`
			}
			if err := json.Unmarshal([]byte(pool.entity.StaticExtra), &staticExtra); err != nil {
				return "", err
			}

			return staticExtra.Vault, nil
		}
	default:
		return pool.entity.Address, nil
	}
}
