package ekubov3

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const testFailingRPCURL = "http://127.0.0.1:1"

const (
	testCursorIDIndex          = 100
	testRefetchedPoolIDIndex   = 1
	testUnexpectedNextIDIndex  = 101
	testReorgRefetchReqCount   = 2
	testSingleReturnedPoolSize = 1

	testBlockHashPageOne       = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testBlockHashPageTwo       = "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	testBlockHashCursor        = "0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	testBlockHashMismatch      = "0xdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	testBlockHashAfterRefetch  = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	testPoolInitBlockNumberStr = "1"
	testTickSpacing            = 1000
	testFeeStr                 = "9223372036854775"
)

func TestPoolListUpdater(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	plUpdater := NewPoolListUpdater(MainnetConfig, ethrpc.New("https://ethereum.drpc.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
		graphql.NewClient(MainnetConfig.SubgraphAPI))

	newPools, _, err := plUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(newPools), 0)

	expected := []pools.AnyPoolKey{
		anyPoolKey(
			valueobject.ZeroAddress,
			"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			common.Address{}.Hex(),
			9223372036854775,
			pools.NewConcentratedPoolTypeConfig(1000),
		),
		// A stableswap `BoostedFees` pool, tracked through automatic support for extensions without `beforeSwap` and `afterSwap` call point
		anyPoolKey(
			"0x6440f144b7e50d6a8439336510312d2f54beb01d",
			"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			"0x948b9c2c99718034954110cb61a6e08e107745f9",
			3689348814741910,
			pools.NewStableswapPoolTypeConfig(-27631040, 14),
		),
	}

	filteredOut := []pools.AnyPoolKey{
		// An old `BoostedFees` extension
		anyPoolKey(
			valueobject.ZeroAddress,
			"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			"0xd48eb64c9c58cb3317f44551e80acc67b9f8ccae",
			9223372036854775,
			pools.NewConcentratedPoolTypeConfig(1000),
		),
	}

	containsPoolKey := func(testPk pools.AnyPoolKey) bool {
		return slices.ContainsFunc(newPools, func(el entity.Pool) bool {
			var staticExtra StaticExtra
			err := json.Unmarshal([]byte(el.StaticExtra), &staticExtra)
			require.NoError(t, err)

			pk := staticExtra.PoolKey

			return pk.Token0.Cmp(testPk.Token0) == 0 && pk.Token1.Cmp(testPk.Token1) == 0 &&
				pk.Config.Compressed() == testPk.Config.Compressed()
		})
	}

	for _, testPk := range expected {
		require.True(t, containsPoolKey(testPk), "missing pool key: %v", testPk)
	}

	for _, testPk := range filteredOut {
		require.False(t, containsPoolKey(testPk), "unexpected filtered pool key returned: %v", testPk)
	}
}

func TestGetNewPoolKeys_PaginatesAndSkipsCursorRow(t *testing.T) {
	t.Parallel()

	const firstPageSize = subgraphPageSize
	firstPage := make([]map[string]any, 0, firstPageSize)
	for i := range firstPageSize {
		firstPage = append(firstPage, makePoolInitialization(
			poolInitID(i+1),
			testBlockHashPageOne,
			poolAddress(i+1),
			poolAddress(i+2),
		))
	}
	lastFirstPage := firstPage[len(firstPage)-1]

	secondPage := []map[string]any{
		lastFirstPage, // repeated cursor row
		makePoolInitialization(poolInitID(firstPageSize+1), testBlockHashPageTwo, poolAddress(firstPageSize+3), poolAddress(firstPageSize+4)),
		makePoolInitialization(poolInitID(firstPageSize+2), testBlockHashPageTwo, poolAddress(firstPageSize+5), poolAddress(firstPageSize+6)),
	}

	var requestStartIDs []string
	srv := newSubgraphTestServer(t, func(startID string) []map[string]any {
		requestStartIDs = append(requestStartIDs, startID)
		switch startID {
		case subgraphInitialStartID:
			return firstPage
		case lastFirstPage["id"].(string):
			return secondPage
		default:
			t.Fatalf("unexpected startId: %s", startID)
			return nil
		}
	})
	defer srv.Close()

	u := NewPoolListUpdater(
		MainnetConfig, ethrpc.New(testFailingRPCURL), graphql.NewClient(srv.URL),
	)

	keys, cursor, err := u.getNewPoolKeys(context.Background())
	require.NoError(t, err)
	require.Len(t, keys, firstPageSize+2)
	require.Equal(t, []string{subgraphInitialStartID, lastFirstPage["id"].(string)}, requestStartIDs)
	require.Equal(t, secondPage[len(secondPage)-1]["id"], cursor.id)
}

func TestGetNewPoolKeys_ReorgWhenNonInitialCursorReturnsEmpty(t *testing.T) {
	t.Parallel()

	assertReorgRefetchFromNonInitialCursor(t, nil)
}

func TestGetNewPoolKeys_ReorgWhenCursorIDMatchesButBlockHashDiffers(t *testing.T) {
	t.Parallel()

	assertReorgRefetchFromNonInitialCursor(t, []map[string]any{
		makePoolInitialization(
			poolInitID(testCursorIDIndex),
			testBlockHashMismatch,
			poolAddress(10),
			poolAddress(11),
		),
	})
}

func TestGetNewPoolKeys_ReorgWhenCursorFirstRowIDDiffers(t *testing.T) {
	t.Parallel()

	assertReorgRefetchFromNonInitialCursor(t, []map[string]any{
		makePoolInitialization(
			poolInitID(testUnexpectedNextIDIndex),
			testBlockHashCursor,
			poolAddress(12),
			poolAddress(13),
		),
	})
}

func TestGetNewPools_DoesNotCommitCursorOnFetchPoolsError(t *testing.T) {
	t.Parallel()

	lastID := poolInitID(testRefetchedPoolIDIndex)
	srv := newSubgraphTestServer(t, func(startID string) []map[string]any {
		require.Equal(t, subgraphInitialStartID, startID)
		return []map[string]any{
			makePoolInitialization(lastID, testBlockHashAfterRefetch, poolAddress(1), poolAddress(2)),
		}
	})
	defer srv.Close()

	u := NewPoolListUpdater(
		MainnetConfig, ethrpc.New(testFailingRPCURL), graphql.NewClient(srv.URL), // force data fetcher RPC failure
	)

	_, _, err := u.GetNewPools(context.Background(), nil)
	require.Error(t, err)
	require.Equal(t, subgraphInitialStartID, u.subgraphCursor.id)
	require.Empty(t, u.subgraphCursor.blockHash)
}

func assertReorgRefetchFromNonInitialCursor(t *testing.T, firstPage []map[string]any) {
	t.Helper()

	reorgCursor := subgraphCursor{
		id:        poolInitID(testCursorIDIndex),
		blockHash: testBlockHashCursor,
	}

	requestCount := 0
	srv := newSubgraphTestServer(t, func(startID string) []map[string]any {
		requestCount++

		if requestCount == 1 {
			require.Equal(t, reorgCursor.id, startID)
			return firstPage
		}

		require.Equal(t, subgraphInitialStartID, startID)
		return []map[string]any{
			makePoolInitialization(poolInitID(testRefetchedPoolIDIndex), testBlockHashAfterRefetch, poolAddress(1), poolAddress(2)),
		}
	})
	defer srv.Close()

	u := NewPoolListUpdater(MainnetConfig, ethrpc.New(testFailingRPCURL), graphql.NewClient(srv.URL))
	u.subgraphCursor = reorgCursor

	keys, cursor, err := u.getNewPoolKeys(context.Background())
	require.NoError(t, err)
	require.Len(t, keys, testSingleReturnedPoolSize)
	require.Equal(t, testReorgRefetchReqCount, requestCount)
	require.Equal(t, poolInitID(testRefetchedPoolIDIndex), cursor.id)
}

func newSubgraphTestServer(t *testing.T, pageFn func(startID string) []map[string]any) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var req struct {
			Variables map[string]any `json:"variables"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))

		startID, ok := req.Variables["startId"].(string)
		require.True(t, ok, "startId is not a string: %T", req.Variables["startId"])

		resp := map[string]any{
			"data": map[string]any{
				"poolInitializations": pageFn(startID),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
}

func makePoolInitialization(id, blockHash, token0, token1 string) map[string]any {
	return map[string]any{
		"id":                      id,
		"blockNumber":             testPoolInitBlockNumberStr,
		"blockHash":               blockHash,
		"tickSpacing":             testTickSpacing,
		"stableswapCenterTick":    nil,
		"stableswapAmplification": nil,
		"extension":               common.Address{}.Hex(),
		"fee":                     testFeeStr,
		"poolId":                  idToHash(id),
		"token0":                  token0,
		"token1":                  token1,
	}
}

func poolInitID(i int) string {
	return fmt.Sprintf("0x%032x", i)
}

func idToHash(id string) string {
	trimmed := id[2:]
	return "0x" + strings.Repeat("0", 64-len(trimmed)) + trimmed
}

func poolAddress(i int) string {
	return fmt.Sprintf("0x%040x", i)
}
