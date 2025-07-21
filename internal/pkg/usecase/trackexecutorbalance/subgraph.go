package trackexecutor

import (
	"context"
	"fmt"

	"github.com/KyberNetwork/kutils"
	"github.com/machinebox/graphql"
	"github.com/rs/zerolog/log"
)

func fetchNewExecutorExchangeEvents(
	ctx context.Context,
	aggregatorGraphQLClient *graphql.Client,
	executorAddress string,
	fromBlock uint64,
) ([]ExchangeEvent, error) {
	var exchangeEvents []ExchangeEvent
	var pageIndex int
	for {
		req := graphql.NewRequest(getExecutorExchangeEventsQuery(executorAddress, fromBlock, pageIndex*graphQLPageSize))
		var res SubgraphExecutorExchangesResponse
		if err := aggregatorGraphQLClient.Run(ctx, req, &res); err != nil {
			log.Ctx(ctx).Warn().
				Str("executor", executorAddress).
				Uint64("fromBlock", fromBlock).
				Int("pageIndex", pageIndex).
				Msg("fetch Exchange events from executor")
			break
		}
		exchangeEvents = append(exchangeEvents, res.ExecutorExchanges...)

		pageIndex += 1
		if pageIndex*graphQLPageSize > graphQLMaxOffset {
			break
		}

		if len(res.ExecutorExchanges) < graphQLPageSize {
			break
		}
	}

	return exchangeEvents, nil
}

func fetchNewPoolApprovalEvents(
	ctx context.Context,
	poolApprovalGraphQLClient *graphql.Client,
	executorAddress string,
	fromBlock uint64,
) []PoolApprovalEvent {
	var poolApprovalEvents []PoolApprovalEvent
	var pageIndex int
	for {
		req := graphql.NewRequest(getPoolApprovalEventsQuery(executorAddress, fromBlock, pageIndex*graphQLPageSize))
		var res SubgraphPoolApprovalsResponse
		if err := poolApprovalGraphQLClient.Run(ctx, req, &res); err != nil {
			log.Ctx(ctx).Warn().
				Str("executor", executorAddress).
				Uint64("fromBlock", fromBlock).
				Int("pageIndex", pageIndex).
				Msg("fetch Pool Approval events from executor")
			break
		}
		poolApprovalEvents = append(poolApprovalEvents, res.PoolApprovals...)

		pageIndex += 1
		if pageIndex*graphQLPageSize > graphQLMaxOffset {
			break
		}

		if len(res.PoolApprovals) < graphQLPageSize {
			break
		}
	}

	return poolApprovalEvents
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

func getExecutorExchangeEventsQuery(executorAddress string, fromBlock uint64, offset int) string {
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
			id
			pair
			executor
			tx
			token
			blockNumber
		}
	}`, executorAddress, fromBlock, graphQLPageSize, offset)
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

func getPoolApprovalEventsQuery(executorAddress string, fromBlock uint64, offset int) string {
	// Intentionally use `blockNumber_gte` instead of `blockNumber_gt`,
	// to cover case event logs of a single block number spread between multiple graphQL pages.
	return fmt.Sprintf(`{
		executorApprovals(
			where: {
				executor: "%s"
				blockNumber_gte: %d
			}
			first: %d
			skip: %d
			orderBy: blockNumber
			orderDirection: asc
		) {
			id
			token
			spender
			blockNumber
		}
	}`, executorAddress, fromBlock, graphQLPageSize, offset)
}
