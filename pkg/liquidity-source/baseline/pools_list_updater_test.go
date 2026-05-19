package baseline

import (
	"context"
	"strings"
	"testing"
)

func TestPoolsListUpdaterReturnsErrorWithoutGraphQLClient(t *testing.T) {
	updater := NewPoolsListUpdater(&Config{ChainID: 1}, nil, nil)

	_, _, err := updater.GetNewPools(context.Background(), nil)
	if err == nil {
		t.Fatal("expected missing GraphQL client error")
	}
	if !strings.Contains(err.Error(), "graphql client is not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}
