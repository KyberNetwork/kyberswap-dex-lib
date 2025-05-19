package trackexecutor

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/elastic-go-sdk/v2/constants"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kutils"
	graphqlPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	dexValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/machinebox/graphql"
	"github.com/samber/lo"

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
	gasTokenAddress           string
}

func NewUseCase(
	ethClient *ethrpc.Client,
	poolFactory IPoolFactory,
	poolRepository IPoolRepository,
	executorBalanceRepository IExecutorBalanceRepository,
	config Config,
) *useCase {
	graphQLClient := graphqlPkg.New(graphqlPkg.Config{
		Url:     config.SubgraphURL,
		Timeout: graphQLRequestTimeout,
	})

	return &useCase{
		ethClient:                 ethClient,
		graphQLClient:             graphQLClient,
		poolFactory:               poolFactory,
		poolRepository:            poolRepository,
		executorBalanceRepository: executorBalanceRepository,
		config:                    config,
		gasTokenAddress:           valueobject.WrapNativeLower(dexValueObject.NativeAddress, config.ChainID),
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
				logger.Errorf(ctx, "fail to track executor %s, err: %v", executorAddress, err)
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

	lg := logger.WithFields(ctx, logger.Fields{
		"executor": executorAddress,
	})

	for {
		blockNumber, err := u.executorBalanceRepository.GetLatestProcessedBlockNumber(ctx, executorAddress)
		if err != nil {
			return err
		}

		blockNumber = max(blockNumber, u.config.StartBlock)

		lg.WithFields(logger.Fields{
			"currentBlock": blockNumber,
			"latestBlock":  blockNumberCheckpoint,
		}).Info("Start fetch events.")

		swappedEvents, err := fetchNewRouterSwappedEvents(ctx, u.graphQLClient, blockNumber)
		if err != nil {
			return err
		}

		if len(swappedEvents) == 0 {
			return nil
		}

		lg.WithFields(logger.Fields{
			"currentBlock": blockNumber,
			"latestBlock":  blockNumberCheckpoint,
		}).Infof("Fetched %d Swapped events", len(swappedEvents))

		lastBlockNumber, err := kutils.Atou[uint64](swappedEvents[len(swappedEvents)-1].BlockNumber)
		if err != nil {
			return fmt.Errorf("failed to convert block number to uint64: %v", err)
		}

		exchangeEvents, err := fetchNewExecutorExchangeEvents(ctx, u.graphQLClient, executorAddress, blockNumber, lastBlockNumber)
		if err != nil {
			return err
		}

		if len(exchangeEvents) == 0 {
			err = u.executorBalanceRepository.UpdateLatestProcessedBlockNumber(ctx, executorAddress, lastBlockNumber)
			if err != nil {
				return err
			}

			lg.WithFields(logger.Fields{
				"currentBlock":          blockNumber,
				"lastestProcessedBlock": lastBlockNumber,
			}).Info("No new Exchange events, skip to the next interval")

			return nil
		}

		lg.WithFields(logger.Fields{
			"currentBlock": blockNumber,
			"toBlock":      lastBlockNumber,
		}).Infof("Fetched %d Exchange events", len(exchangeEvents))

		if err := u.trackExecutorBalance(ctx, executorAddress, exchangeEvents); err != nil {
			return err
		}

		if err := u.trackExecutorPoolApproval(ctx, executorAddress, swappedEvents, exchangeEvents); err != nil {
			return err
		}

		err = u.executorBalanceRepository.UpdateLatestProcessedBlockNumber(ctx, executorAddress, lastBlockNumber)
		if err != nil {
			return err
		}

		if lastBlockNumber >= blockNumberCheckpoint {
			break
		} else {
			logger.WithFields(ctx, logger.Fields{
				"currentBlock": lastBlockNumber,
				"latestBlock":  blockNumberCheckpoint,
			}).Debug("Continue catching up with the latest events")
			time.Sleep(intervalDelay)
		}
	}

	return nil
}

func (u *useCase) trackExecutorBalance(ctx context.Context, executorAddress string, events []ExchangeEvent) error {
	tokenOutSet := mapset.NewThreadUnsafeSet[string]()
	for _, event := range events {
		tokenOutSet.Add(u.formatToken(event.Token))
	}
	tokenOutSlice := tokenOutSet.ToSlice() // To persist the index of elements.

	existed, err := u.executorBalanceRepository.HasToken(ctx, executorAddress, tokenOutSlice)
	if err != nil {
		return err
	}

	candidateTokenOuts := lo.Filter(tokenOutSlice, func(item string, index int) bool {
		return !existed[index]
	})

	if len(candidateTokenOuts) == 0 {
		logger.Debug(ctx, "No new tokens to track, skip to the next interval")
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
		return rpcResponse.Result[idx] && tokenBalances[idx] != nil && tokenBalances[idx].Cmp(constants.Zero) > 0
	})

	// missTokens are tokens that have Exchange events,
	// however executor still does not have 1 wei of them.
	// This can happen after the Merge swap feature, since
	// executor will use all the current balance of the token
	// to swap.
	missTokens := lo.Filter(candidateTokenOuts, func(item string, idx int) bool {
		return !rpcResponse.Result[idx] || tokenBalances[idx] == nil || tokenBalances[idx].Cmp(constants.Zero) == 0
	})

	logger.WithFields(ctx, logger.Fields{
		"executor":        executorAddress,
		"numUpdateTokens": len(updateTokens),
		"missTokens":      len(missTokens),
	}).Info("Track tokens that executor has balance")

	if err := u.executorBalanceRepository.AddToken(ctx, executorAddress, updateTokens); err != nil {
		return err
	}

	if err := u.executorBalanceRepository.RemoveToken(ctx, executorAddress, missTokens); err != nil {
		return err
	}

	return nil
}

func (u *useCase) trackExecutorPoolApproval(ctx context.Context, executorAddress string,
	swappedEvents []SwappedEvent, exchangeEvents []ExchangeEvent) error {

	swappedEventByTxHash := lo.KeyBy(swappedEvents, func(event SwappedEvent) string { return event.Tx })
	pairAddressSet := mapset.NewThreadUnsafeSet[string]()

	for i, event := range exchangeEvents {
		idParts := strings.Split(event.Id, "-")
		if len(idParts) != 2 {
			return fmt.Errorf("invalid event id: %s", event.Id)
		}
		logIndex, err := kutils.Atou[uint32](idParts[1])
		if err != nil {
			return fmt.Errorf("invalid log index: %s", idParts[1])
		}
		exchangeEvents[i].LogIndex = logIndex
		pairAddressSet.Add(event.Pair)
	}

	poolEntities, err := u.poolRepository.FindByAddresses(ctx, pairAddressSet.ToSlice())
	if err != nil {
		return err
	}
	poolSimMap := u.poolFactory.NewPoolByAddress(ctx, poolEntities, common.Hash{})

	// Group Exchange events by tx hash and sort them by log index ascending.
	exchangeEventsByTxHash := lo.MapValues(
		lo.GroupBy(exchangeEvents, func(event ExchangeEvent) string { return event.Tx }),
		func(events []ExchangeEvent, _ string) []ExchangeEvent {
			slices.SortFunc(events, func(a, b ExchangeEvent) int { return int(a.LogIndex - b.LogIndex) })
			return events
		},
	)

	swapInfoSet := mapset.NewThreadUnsafeSet[SwapInfo]()
	for txHash, events := range exchangeEventsByTxHash {
		var (
			swappedEvent = swappedEventByTxHash[txHash]
			tokenIn      = u.formatToken(swappedEvent.TokenIn)
			tokenOut     = u.formatToken(swappedEvent.TokenOut)

			currentTokenIn = tokenIn
		)

		slices.SortFunc(events, func(a, b ExchangeEvent) int {
			return int(a.LogIndex - b.LogIndex)
		})

		for _, event := range events {
			token := u.formatToken(event.Token)

			swapInfoSet.Add(SwapInfo{
				Pair:     event.Pair,
				TokenIn:  currentTokenIn,
				TokenOut: token,
			})

			// Check if last swap in a path.
			if strings.EqualFold(token, tokenOut) {
				currentTokenIn = tokenIn
			} else {
				currentTokenIn = token
			}
		}
	}
	swapInfos := swapInfoSet.ToSlice()

	updatePoolApprovalSet := mapset.NewThreadUnsafeSet[dto.PoolApprovalQuery]()
	for _, swapInfo := range swapInfos {
		approvalAddress := swapInfo.Pair
		if sim, ok := poolSimMap[swapInfo.Pair]; ok {
			exchange := valueobject.Exchange(sim.GetExchange())
			isApproveMaxExchange, usePoolAsApprovalAddress := valueobject.IsApproveMaxExchange(exchange)

			if !isApproveMaxExchange {
				continue
			}

			if !usePoolAsApprovalAddress {
				approvalAddress = sim.GetApprovalAddress(swapInfo.TokenIn, swapInfo.TokenOut)
			}
		}

		updatePoolApprovalSet.Add(dto.PoolApprovalQuery{
			TokenIn:     swapInfo.TokenIn,
			PoolAddress: strings.ToLower(approvalAddress),
		})
	}
	updatePoolApprovals := updatePoolApprovalSet.ToSlice()

	if err := u.executorBalanceRepository.ApprovePool(ctx, executorAddress, updatePoolApprovals); err != nil {
		return err
	}

	logger.WithFields(ctx, logger.Fields{
		"executor":               executorAddress,
		"numUpdatePoolApprovals": len(updatePoolApprovals),
	}).Info("Add pool approvals for executor")

	return nil
}

func (u *useCase) formatToken(token string) string {
	if strings.EqualFold(token, valueobject.EtherAddress) ||
		strings.EqualFold(token, valueobject.ZeroAddress) {
		return u.gasTokenAddress
	}
	return token
}
