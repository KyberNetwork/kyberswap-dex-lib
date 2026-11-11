package algebrav1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// rpcExchange is one recorded JSON-RPC call and the node's answer to it.
type rpcExchange struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`
}

type rpcRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// rpcKey keys a call by its method and its params, encoded again so that whitespace and key order do not count.
func rpcKey(method string, params json.RawMessage) string {
	var v any
	_ = json.Unmarshal(params, &v)
	b, _ := json.Marshal(v)
	return method + string(b)
}

// replayRPC serves the recorded exchanges of file until the test ends, at the URL it returns, and fails the test on a
// call it does not hold.
func replayRPC(t *testing.T, file string) string {
	byKey := make(map[string]rpcExchange)
	for _, x := range readJSON[[]rpcExchange](t, file) {
		byKey[rpcKey(x.Method, x.Params)] = x
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode call: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		x, ok := byKey[rpcKey(req.Method, req.Params)]
		if !ok {
			t.Errorf("unrecorded call %s %s", req.Method, req.Params)
			http.Error(w, "unrecorded", http.StatusInternalServerError)
			return
		}
		resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if x.Error != nil {
			resp["error"] = x.Error
		} else {
			resp["result"] = x.Result
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}
