package efficientprop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/big"
)

// Early close: the merge window is over once url 0 has answered with an account for the highest block seen.
func TestClosesEarly(t *testing.T) {
	at := func(rank int, n int64, withAccount bool) answer {
		a := answer{rank: rank, accounts: map[common.Address]gethclient.OverrideAccount{}}
		if n >= 0 {
			a.number = big.NewInt(n)
		}
		if withAccount {
			a.accounts[cellOf(1)] = gethclient.OverrideAccount{}
		}
		return a
	}
	for _, c := range []struct {
		name string
		in   []answer
		want bool
	}{
		{"none", nil, false},
		{"url 1 alone", []answer{at(1, 5, true)}, false},
		{"url 0 with an account", []answer{at(0, 5, true)}, true},
		{"url 0 empty", []answer{at(0, 5, false)}, false},
		{"url 0 without a block", []answer{at(0, -1, true)}, false},
		{"url 0 behind url 1", []answer{at(1, 6, true), at(0, 5, true)}, false},
		{"url 0 level with url 1", []answer{at(1, 5, true), at(0, 5, true)}, true},
		{"url 0 ahead", []answer{at(1, 4, true), at(0, 5, true)}, true},
		{"url 0 has two blocks, one the highest", []answer{at(0, 4, true), at(0, 6, true), at(1, 6, true)}, true},
	} {
		assert.Equal(t, c.want, closesEarly(c.in), c.name)
	}
}

// blockedServer answers a /state call with result once release is closed, and a fast one at once.
func answerServer(t *testing.T, result any, release <-chan struct{}) *httptest.Server {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&msg)
		if release != nil {
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": msg.ID, "result": result})
	}))
	t.Cleanup(s.Close)
	return s
}

// A second url that never answers does not hold the fetch for the merge window once url 0 has answered for the highest
// block: the window here is a minute and the fetch returns with url 0's answer alone, the second call cancelled.
func TestFetchAnswersClosesWhenURLZeroAnswered(t *testing.T) {
	cell := cellOf(1)
	fast := answerServer(t, map[string]any{"slot": 7, "blockNumber": "0x4d3",
		"e": map[string]any{"stateOverride": map[string]any{cell.Hex(): map[string]any{
			"stateDiff": map[string]string{quoteKey(0).Hex(): w1}, "balance": "0x0", "nonce": "0x1"}}}}, nil)
	never := make(chan struct{})
	t.Cleanup(func() { close(never) })
	slow := answerServer(t, map[string]any{}, never)
	done := make(chan []answer, 1)
	go func() {
		done <- fetchAnswers(context.Background(), dialClients(t, fast.URL, slow.URL), time.Minute, time.Minute, nil)
	}()
	select {
	case got := <-done:
		require.Len(t, got, 1)
		assert.Equal(t, 0, got[0].rank)
	case <-time.After(5 * time.Second):
		t.Fatal("the merge window was not closed by url 0's answer")
	}
}
