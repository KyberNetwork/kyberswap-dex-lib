package trackexecutor

import (
	"context"
	"fmt"
	"strconv"

	"github.com/machinebox/graphql"
)

func fetchNewExecutorExchangeEvents(
	ctx context.Context,
	aggregatorGraphQLClient *graphql.Client,
	executorAddress string,
	lastBlockNumber uint64,
) ([]ExchangeEvent, error) {
	var exchangeEvents []ExchangeEvent
	var pageIndex int
	for {
		req := graphql.NewRequest(getExecutorExchangeEventsQuery(executorAddress, lastBlockNumber, pageIndex*graphQLPageSize))
		var res SubgraphAggregatorResponse
		if err := aggregatorGraphQLClient.Run(ctx, req, &res); err != nil {
			return nil, err
		}
		exchangeEvents = append(exchangeEvents, res.ExecutorExchanges...)

		pageIndex += 1
		if pageIndex*graphQLPageSize > graphQLMaxOffset {
			break
		}
	}

	return exchangeEvents, nil
}

func fetchLatestEventBlockNumber(
	ctx context.Context,
	aggregatorGraphQLClient *graphql.Client,
	executorAddress string,
) (uint64, error) {
	req := graphql.NewRequest(getExecutorExchangeLatestEventQuery(executorAddress))
	var res SubgraphAggregatorResponse
	if err := aggregatorGraphQLClient.Run(ctx, req, &res); err != nil {
		return 0, err
	}

	if len(res.ExecutorExchanges) != 1 {
		return 0, fmt.Errorf("should return one latest event, received %d", len(res.ExecutorExchanges))
	}

	lastBlockNumberStr := res.ExecutorExchanges[0].BlockNumber
	return strconv.ParseUint(lastBlockNumberStr, 10, 64)
}

func getExecutorExchangeEventsQuery(executorAddress string, lastBlockNumber uint64, offset int) string {
	// Intentionally use `blockNumber_gte` instead of `blockNumber_gt`,
	// to cover case event logs of a single block number spread between multiple graphQL pages.
	return fmt.Sprintf(`{
		executorExchanges(
			where: {
				executor: "%s"
				blockNumber_gte: %d
			}
			first: %d
			skip: %d
			orderBy: blockNumber
			orderDirection: asc
		) {
			executor
			pair
			token
			blockNumber
		}
	}`, executorAddress, lastBlockNumber, graphQLPageSize, offset)
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
