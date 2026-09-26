package everlongflamm

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/stretchr/testify/require"
)

// rpcTape replays recorded JSON-RPC answers so the lister and tracker run offline against the exact responses Base
// gave them. A tape is recorded by running the test with EVERLONG_FLAMM_RECORD=1 and BASE_RPC_URL set: every
// request the code makes is forwarded upstream once and stored under its method and canonical params. With
// EVERLONG_FLAMM_RECORD=missing the recorded answers are kept (the listing keeps its head block) and only requests
// the tape lacks are forwarded and added. Either way the saved tape holds exactly the requests the recording run
// made, so an answer no longer asked for is dropped. Like mainnet.base.org, the tape answers a batch of more than ten calls with
// a single error object, so every refresh exercises the chunked batches.

// rpcTapeMaxBatch is the largest JSON-RPC batch the tape answers, the cap mainnet.base.org enforces.
const rpcTapeMaxBatch = 10

type tapeEntry struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  *tapeError      `json:"error,omitempty"`
}

type tapeError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type rpcTape struct {
	t        *testing.T
	path     string
	upstream string
	mu       sync.Mutex
	entries  map[string]tapeEntry
	used     map[string]bool
	mutate   func(method string, params json.RawMessage, e *tapeEntry)
	srv      *httptest.Server
}

type tapeRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func tapeKey(method string, params json.RawMessage) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, params); err != nil {
		return method + "|" + string(params)
	}
	return method + "|" + buf.String()
}

// openTape serves the tape at path, recording it first when EVERLONG_FLAMM_RECORD is set.
func openTape(t *testing.T, path string) *rpcTape {
	t.Helper()
	tp := &rpcTape{t: t, path: path, entries: map[string]tapeEntry{}, used: map[string]bool{}}
	record := os.Getenv("EVERLONG_FLAMM_RECORD")
	if record != "1" && record != "missing" {
		record = "" // another recorder's mode (tracked, armed, sequence): replay the tape
	}
	if record != "" {
		tp.upstream = os.Getenv("BASE_RPC_URL")
		require.NotEmpty(t, tp.upstream, "recording needs BASE_RPC_URL")
	}
	if record == "" || record == "missing" {
		f, err := os.Open(path)
		require.NoError(t, err)
		defer func() { _ = f.Close() }()
		zr, err := gzip.NewReader(f)
		require.NoError(t, err)
		require.NoError(t, json.NewDecoder(zr).Decode(&tp.entries))
	}
	tp.srv = httptest.NewServer(http.HandlerFunc(tp.serve))
	t.Cleanup(tp.srv.Close)
	return tp
}

// recordTape records a new tape at path from upstream (an anvil fork's endpoint): every request is forwarded once and
// save writes the answers.
func recordTape(t *testing.T, path, upstream string) *rpcTape {
	t.Helper()
	tp := &rpcTape{t: t, path: path, upstream: upstream, entries: map[string]tapeEntry{}, used: map[string]bool{}}
	tp.srv = httptest.NewServer(http.HandlerFunc(tp.serve))
	t.Cleanup(tp.srv.Close)
	return tp
}

func (tp *rpcTape) client() *ethrpc.Client {
	return ethrpc.New(tp.srv.URL).SetMulticallContract(multicall3)
}

// save writes a recorded tape (deterministic: sorted keys, gzip without a timestamp).
func (tp *rpcTape) save() {
	if tp.upstream == "" {
		return
	}
	tp.mu.Lock()
	defer tp.mu.Unlock()
	keys := make([]string, 0, len(tp.used))
	for k := range tp.used {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var body bytes.Buffer
	body.WriteString("{\n")
	for i, k := range keys {
		kb, _ := json.Marshal(k)
		vb, _ := json.Marshal(tp.entries[k])
		sep := ","
		if i == len(keys)-1 {
			sep = ""
		}
		fmt.Fprintf(&body, "%s:%s%s\n", kb, vb, sep)
	}
	body.WriteString("}\n")
	var out bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&out, gzip.BestCompression)
	_, _ = zw.Write(body.Bytes())
	require.NoError(tp.t, zw.Close())
	require.NoError(tp.t, os.WriteFile(tp.path, out.Bytes(), 0o644))
}

func (tp *rpcTape) serve(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var batch []tapeRequest
	single := false
	if bytes.HasPrefix(bytes.TrimSpace(raw), []byte("[")) {
		err = json.Unmarshal(raw, &batch)
	} else {
		var one tapeRequest
		err = json.Unmarshal(raw, &one)
		batch, single = []tapeRequest{one}, true
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if len(batch) > rpcTapeMaxBatch {
		// mainnet.base.org's answer to an oversized batch: one error object, not an array.
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32014,"message":"maximum 10 calls in 1 batch"},"id":null}`))
		return
	}
	type response struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  json.RawMessage `json:"result,omitempty"`
		Error   *tapeError      `json:"error,omitempty"`
	}
	out := make([]response, len(batch))
	for i, req := range batch {
		e := tp.answer(req)
		out[i] = response{JSONRPC: "2.0", ID: req.ID, Result: e.Result, Error: e.Error}
	}
	if single {
		_ = json.NewEncoder(w).Encode(out[0])
		return
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (tp *rpcTape) answer(req tapeRequest) tapeEntry {
	key := tapeKey(req.Method, req.Params)
	tp.mu.Lock()
	e, ok := tp.entries[key]
	tp.used[key] = true
	tp.mu.Unlock()
	if !ok && tp.upstream != "" {
		e = tp.forward(req)
		tp.mu.Lock()
		tp.entries[key] = e
		tp.mu.Unlock()
	} else if !ok {
		e = tapeEntry{Error: &tapeError{Code: -32000, Message: "tape miss: " + key}}
	}
	if tp.mutate != nil {
		c := tapeEntry{Result: append(json.RawMessage(nil), e.Result...), Error: e.Error}
		tp.mutate(req.Method, req.Params, &c)
		return c
	}
	return e
}

func (tp *rpcTape) forward(req tapeRequest) tapeEntry {
	body, _ := json.Marshal(tapeRequest{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: req.Method, Params: req.Params})
	for attempt := 0; ; attempt++ {
		resp, err := http.Post(tp.upstream, "application/json", bytes.NewReader(body))
		require.NoError(tp.t, err)
		raw, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		require.NoError(tp.t, err)
		if resp.StatusCode == http.StatusTooManyRequests && attempt < 20 {
			time.Sleep(time.Duration(300*(attempt+1)) * time.Millisecond)
			continue
		}
		var e tapeEntry
		require.NoError(tp.t, json.Unmarshal(raw, &e), "upstream %s: %s", req.Method, raw)
		return e
	}
}
