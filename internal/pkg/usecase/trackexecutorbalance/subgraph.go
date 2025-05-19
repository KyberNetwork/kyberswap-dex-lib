package trackexecutor

import (
	"context"
	"fmt"

	"github.com/KyberNetwork/kutils"
	"github.com/machinebox/graphql"

	"github.com/KyberNetwork/router-service/pkg/logger"
)

func fetchNewExecutorExchangeEvents(
	ctx context.Context,
	aggregatorGraphQLClient *graphql.Client,
	executorAddress string,
	fromBlock uint64,
	toBlock uint64,
) ([]ExchangeEvent, error) {
	var exchangeEvents []ExchangeEvent
	var pageIndex int
	for {
		req := graphql.NewRequest(getExecutorExchangeEventsQuery(executorAddress, fromBlock, toBlock, pageIndex*graphQLPageSize))
		var res SubgraphExecutorExchangesResponse
		if err := aggregatorGraphQLClient.Run(ctx, req, &res); err != nil {
			logger.WithFields(ctx, logger.Fields{
				"executor":  executorAddress,
				"fromBlock": fromBlock,
				"toBlock":   toBlock,
				"pageIndex": pageIndex,
			}).Warn("fetch Exchange events from executor")
			break
		}
		exchangeEvents = append(exchangeEvents, res.ExecutorExchanges...)

		pageIndex += 1
		if pageIndex*graphQLPageSize > graphQLMaxOffset {
			lastBlockStr := exchangeEvents[len(exchangeEvents)-1].BlockNumber
			blockNumber, err := kutils.Atou[uint64](lastBlockStr)
			if err != nil {
				logger.WithFields(ctx, logger.Fields{
					"executor":  executorAddress,
					"fromBlock": fromBlock,
					"toBlock":   toBlock,
					"pageIndex": pageIndex,
				}).Errorf("failed to convert block number to uint64: %v, %v", lastBlockStr, err)
				return nil, err
			}

			pageIndex = 0
			fromBlock = blockNumber + 1
		}

		if len(res.ExecutorExchanges) < graphQLPageSize {
			break
		}
	}

	return exchangeEvents, nil
}

func fetchNewRouterSwappedEvents(
	ctx context.Context,
	aggregatorGraphQLClient *graphql.Client,
	lastBlockNumber uint64,
) ([]SwappedEvent, error) {
	var swappedEvents []SwappedEvent
	var pageIndex int
	for {
		req := graphql.NewRequest(getRouterSwappedEventsQuery(lastBlockNumber, pageIndex*graphQLPageSize))
		var res SubgraphRouterSwappedResponse
		if err := aggregatorGraphQLClient.Run(ctx, req, &res); err != nil {
			logger.WithFields(ctx, logger.Fields{
				"lastBlockNumber": lastBlockNumber,
				"pageIndex":       pageIndex,
			}).Warn("fetch Swapped events from router")
			break
		}
		swappedEvents = append(swappedEvents, res.SwappedEvents...)

		if len(res.SwappedEvents) < graphQLPageSize {
			break
		}

		pageIndex += 1
		if pageIndex*graphQLPageSize > graphQLMaxOffset {
			break
		}
	}

	return swappedEvents, nil
}

func fetchLatestEventBlockNumber(
	ctx context.Context,
	aggregatorGraphQLClient *graphql.Client,
	executorAddress string,
) (uint64, error) {
	req := graphql.NewRequest(getExecutorExchangeLatestEventQuery(executorAddress))
	var res SubgraphExecutorExchangesResponse
	if err := aggregatorGraphQLClient.Run(ctx, req, &res); err != nil {
		return 0, err
	}

	if len(res.ExecutorExchanges) != 1 {
		return 0, fmt.Errorf("should return one latest event, received %d", len(res.ExecutorExchanges))
	}

	return kutils.Atou[uint64](res.ExecutorExchanges[0].BlockNumber)
}

func getExecutorExchangeEventsQuery(executorAddress string, fromBlock, toBlock uint64, offset int) string {
	// Intentionally use `blockNumber_gte` instead of `blockNumber_gt`,
	// to cover case event logs of a single block number spread between multiple graphQL pages.
	return fmt.Sprintf(`{
		executorExchanges(
			where: {
				executor: "%s"
				blockNumber_gte: %d
				blockNumber_lte: %d
			}
			first: %d
			skip: %d
			orderBy: blockNumber
			orderDirection: asc
		) {
			id
			pair
			executor
			tx
			token
			blockNumber
		}
	}`, executorAddress, fromBlock, toBlock, graphQLPageSize, offset)
}

func getExecutorExchangeLatestEventQuery(executorAddress string) string {
	return fmt.Sprintf(`{
		executorExchanges(
			where: {
				executor: "%s"
			}
			first: 1
			orderBy: blockNumber
			orderDirection: desc
		) {
			executor
			pair
			token
			blockNumber
		}
	}`, executorAddress)
}

func getRouterSwappedEventsQuery(lastBlockNumber uint64, offset int) string {
	return fmt.Sprintf(`{
		routerSwappeds(
			where: {
				blockNumber_gte: %d
			}
			first: %d
			skip: %d
			orderBy: blockNumber
			orderDirection: asc
		) {
			tx
			tokenIn
			tokenOut
			blockNumber
		}
	}`, lastBlockNumber, graphQLPageSize, offset)
}
