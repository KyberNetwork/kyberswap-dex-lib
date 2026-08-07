package flap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// On the real graduatinghot board, curve-stage tokens are always "listed":false ("listed" means listed
// on a DEX, i.e. already graduated) and "progress" is always < 100. Both are exercised here as skip
// conditions: Graduated is listed=true (crossed over to DEX), EdgeCase100 covers the defensive
// progress==100 check even though listed=false, and OnCurve is the one entry that should survive.
const page1 = `{
	"category":"graduatinghot","sort":"volume24h_desc","nextCursor":"cursor-page-2",
	"items":[
		{"coin":{"address":"0xAAAA000000000000000000000000000000AAAA","name":"Graduated","symbol":"GRAD"},
		 "listed":true,"quoteToken":"0x0000000000000000000000000000000000000000","progress":"100.00",
		 "tax":{"hasTax":false,"buyTaxBps":null,"sellTaxBps":null}},
		{"coin":{"address":"0xBBBB000000000000000000000000000000BBBB","name":"OnCurve","symbol":"CURVE"},
		 "listed":false,"quoteToken":"0x0000000000000000000000000000000000000000","progress":"42.00",
		 "tax":{"hasTax":true,"buyTaxBps":100,"sellTaxBps":100}},
		{"coin":{"address":"0xCCCC000000000000000000000000000000CCCC","name":"EdgeCase100","symbol":"EC"},
		 "listed":false,"quoteToken":"0x0000000000000000000000000000000000000000","progress":"100.00",
		 "tax":{"hasTax":false,"buyTaxBps":null,"sellTaxBps":null}}
	]
}`

const page2 = `{
	"category":"graduatinghot","sort":"volume24h_desc","nextCursor":"",
	"items":[
		{"coin":{"address":"0xDDDD000000000000000000000000000000DDDD","name":"ErcQuote","symbol":"EQ"},
		 "listed":false,"quoteToken":"0xbe9d156892e55e7154bcd3cb0fea677f9d3103e1","progress":"10.00",
		 "tax":{"hasTax":false,"buyTaxBps":null,"sellTaxBps":null}}
	]
}`

func TestGetNewPools_PagesThroughCursorThenStops(t *testing.T) {
	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("cursor") == "cursor-page-2" {
			_, _ = w.Write([]byte(page2))
			return
		}
		_, _ = w.Write([]byte(page1))
	}))
	defer srv.Close()

	config := &Config{
		DexID:      DexType,
		ChainID:    valueobject.ChainIDBSC,
		APIBaseURL: srv.URL,
		APIKey:     "test-key",
	}
	updater := NewPoolsListUpdater(config)

	// Page 1: 1 of 3 items qualifies (listed=true and progress==100 are both filtered out).
	pools, metadataBytes, err := updater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	assert.Equal(t, "0xbbbb000000000000000000000000000000bbbb", pools[0].Address)
	assert.EqualValues(t, []string{"0", "0"}, pools[0].Reserves)
	assert.Equal(t, DexType, pools[0].Type)
	require.Len(t, pools[0].Tokens, 2)
	wrappedNative := valueobject.ZeroToWrappedLower(valueobject.ZeroAddress, valueobject.ChainIDBSC)
	assert.Equal(t, wrappedNative, pools[0].Tokens[0].Address)
	assert.Equal(t, "0xbbbb000000000000000000000000000000bbbb", pools[0].Tokens[1].Address)

	var metadata Metadata
	require.NoError(t, json.Unmarshal(metadataBytes, &metadata))
	assert.Equal(t, "cursor-page-2", metadata.Cursor)
	assert.False(t, metadata.Done)

	// Page 2: last page, nextCursor empty -> Done.
	pools, metadataBytes, err = updater.GetNewPools(context.Background(), metadataBytes)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	assert.Equal(t, "0xdddd000000000000000000000000000000dddd", pools[0].Address)

	require.NoError(t, json.Unmarshal(metadataBytes, &metadata))
	assert.True(t, metadata.Done)

	// Further calls are no-ops and must not hit the server again.
	requestsBefore := len(requests)
	pools, _, err = updater.GetNewPools(context.Background(), metadataBytes)
	require.NoError(t, err)
	assert.Nil(t, pools)
	assert.Len(t, requests, requestsBefore)
}
