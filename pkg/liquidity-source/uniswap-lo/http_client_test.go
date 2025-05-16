package uniswaplo

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const UniSwapXBaseURL = "https://api.uniswap.org/v2"

func TestFetchDutchOrders(t *testing.T) {
	t.Skip("Skip for CI")
	client := NewUniSwapXClient(UniSwapXBaseURL)
	q := DutchOrderQuery{
		Limit:       2000,
		ChainID:     1,
		OrderStatus: OpenOrderStatus,
		// OrderType:   DutchV2OrderType,
	}
	q.AddSortByCreatedAtGreaterThan(time.Now().Add(-time.Hour).Unix())
	response, err := client.FetchDutchOrders(context.Background(), q)
	require.NoError(t, err)
	t.Log(len(response.Orders))
	for _, order := range response.Orders {
		t.Log(order.DecayStartTime, order.DecayEndTime, order.CreatedAt)
	}
}
