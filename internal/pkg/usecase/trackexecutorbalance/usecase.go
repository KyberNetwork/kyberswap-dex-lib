package trackexecutor

import (
	"context"
	"fmt"
	"math/big"
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
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type useCase struct {
	ethClient                 *ethrpc.Client
	aggregatorGraphQLClient   *graphql.Client
	poolApprovalGraphQLClient *graphql.Client
	executorBalanceRepository IExecutorBalanceRepository
	config                    Config
	gasTokenAddress           string
}

func NewUseCase(
	ethClient *ethrpc.Client,
	executorBalanceRepository IExecutorBalanceRepository,
	config Config,
) *useCase {
	aggregatorGraphQLClient := graphqlPkg.New(graphqlPkg.Config{
		Url:     config.AggregatorSubgraphURL,
		Timeout: graphQLRequestTimeout,
	})

	poolApprovalGraphQLClient := graphqlPkg.New(graphqlPkg.Config{
		Url:     config.PoolApprovalSubgraphURL,
		Timeout: graphQLRequestTimeout,
	})

	return &useCase{
		ethClient:                 ethClient,
		aggregatorGraphQLClient:   aggregatorGraphQLClient,
		poolApprovalGraphQLClient: poolApprovalGraphQLClient,
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
				log.Ctx(ctx).Err(err).Str("executor", executorAddress).Msg("failed to track")
			}
		}(executorAddress)
	}
	wg.Wait()

	return nil
}

func (u *useCase) trackExecutor(ctx context.Context, executorAddress string) error {
	executorAddress = strings.ToLower(executorAddress)

	lg := log.Ctx(ctx).With().Str("executor", executorAddress).Logger()

	g, ctx := errgroup.WithContext(ctx)

	// Track executor balance.
	g.Go(func() error {
		blockNumberCheckpoint, err := fetchLatestEventBlockNumber(ctx, u.aggregatorGraphQLClient, executorAddress)
		if err != nil {
			return err
		}

		for {
			blockNumber, err := u.executorBalanceRepository.GetBalanceTrackerProcessedBlockNumber(ctx, executorAddress)
			if err != nil {
				return err
			}

			blockNumber = max(blockNumber, u.config.StartBlock)

			lg.Info().
				Uint64("currentBlock", blockNumber).
				Uint64("latestBlock", blockNumberCheckpoint).
				Msg("Start fetch events.")

			exchangeEvents, err := fetchNewExecutorExchangeEvents(ctx, u.aggregatorGraphQLClient, executorAddress, blockNumber)
			if err != nil {
				return err
			}

			lg.Info().
				Uint64("currentBlock", blockNumber).
				Uint64("toBlock", blockNumberCheckpoint).
				Msgf("Fetched %d Exchange events", len(exchangeEvents))

			if len(exchangeEvents) == 0 {
				return nil
			}

			lastBlockNumber, err := kutils.Atou[uint64](exchangeEvents[len(exchangeEvents)-1].BlockNumber)
			if err != nil {
				return fmt.Errorf("failed to convert block number to uint64: %v", err)
			}

			if err := u.trackExecutorBalance(ctx, executorAddress, exchangeEvents); err != nil {
				return err
			}

			err = u.executorBalanceRepository.UpdateBalanceTrackerProcessedBlockNumber(ctx, executorAddress, lastBlockNumber)
			if err != nil {
				return err
			}

			if lastBlockNumber >= blockNumberCheckpoint {
				break
			} else {
				log.Ctx(ctx).Debug().
					Uint64("currentBlock", lastBlockNumber).
					Uint64("latestBlock", blockNumberCheckpoint).
					Msg("Continue catching up with the latest events")
				time.Sleep(intervalDelay)
			}
		}
		return nil
	})

	// Track executor pool approval.
	g.Go(func() error {
		return u.trackExecutorPoolApproval(ctx, executorAddress)
	})

	return g.Wait()
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
		log.Ctx(ctx).Debug().Msg("No new tokens to track, skip to the next interval")
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

	log.Ctx(ctx).Info().
		Str("executor", executorAddress).
		Int("numUpdateTokens", len(updateTokens)).
		Int("missTokens", len(missTokens)).
		Msg("Track tokens that executor has balance")

	if err := u.executorBalanceRepository.AddToken(ctx, executorAddress, updateTokens); err != nil {
		return err
	}

	if err := u.executorBalanceRepository.RemoveToken(ctx, executorAddress, missTokens); err != nil {
		return err
	}

	return nil
}

func (u *useCase) trackExecutorPoolApproval(ctx context.Context, executorAddress string) error {
	blockNumber, err := u.executorBalanceRepository.GetPoolApprovalTrackerProcessedBlockNumber(ctx, executorAddress)
	if err != nil {
		return err
	}
	blockNumber = max(blockNumber, u.config.StartBlock)
	events := fetchNewPoolApprovalEvents(ctx, u.poolApprovalGraphQLClient, executorAddress, blockNumber)

	if len(events) == 0 {
		return nil
	}

	approvals := lo.Map(events, func(event PoolApprovalEvent, _ int) dto.PoolApprovalQuery {
		return dto.PoolApprovalQuery{
			TokenIn:     event.Token,
			PoolAddress: event.Spender,
		}
	})

	if err := u.executorBalanceRepository.ApprovePool(ctx, executorAddress, approvals); err != nil {
		return err
	}

	lastBlockNumber, err := kutils.Atou[uint64](events[len(events)-1].BlockNumber)
	if err != nil {
		return err
	}

	if err := u.executorBalanceRepository.UpdatePoolApprovalTrackerProcessedBlockNumber(ctx, executorAddress, lastBlockNumber); err != nil {
		return err
	}

	log.Ctx(ctx).Info().
		Str("executor", executorAddress).
		Uint64("currentBlock", blockNumber).
		Uint64("toBlock", lastBlockNumber).
		Int("numApprovals", len(approvals)).
		Msg("Track pool approvals for executor")

	return nil
}

func (u *useCase) formatToken(token string) string {
	if strings.EqualFold(token, valueobject.EtherAddress) ||
		strings.EqualFold(token, valueobject.ZeroAddress) {
		return u.gasTokenAddress
	}
	return token
}
