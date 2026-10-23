package liquiditybookv21

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var aggregateABI = lo.Must(abi.JSON(strings.NewReader(`[{"name":"aggregate","type":"function",
"stateMutability":"view","inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target",
"type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber",
"type":"uint256"},{"name":"returnData","type":"bytes[]"}]}]`))).Methods["aggregate"]

// zeroNode answers every multicall with zeros, and records the block each one asked for.
func zeroNode(t *testing.T) (*httptest.Server, func() []string) {
	var mu sync.Mutex
	var blocks []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &req))
		reply := func(v any) {
			b, _ := json.Marshal(v)
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, b)
		}
		if req.Method != "eth_call" {
			t.Errorf("unexpected %s", req.Method)
			return
		}
		var msg struct {
			Data  hexutil.Bytes `json:"data"`
			Input hexutil.Bytes `json:"input"`
		}
		require.NoError(t, json.Unmarshal(req.Params[0], &msg))
		data := lo.Ternary(len(msg.Data) > 0, msg.Data, msg.Input)
		if string(data[:4]) != string(aggregateABI.ID) {
			// A call of the multicall's own, such as its block timestamp.
			reply(hexutil.Bytes(make([]byte, 32)))
			return
		}
		var block string
		require.NoError(t, json.Unmarshal(req.Params[1], &block))
		mu.Lock()
		blocks = append(blocks, block)
		mu.Unlock()
		args, err := aggregateABI.Inputs.Unpack(data[4:])
		require.NoError(t, err)
		results := make([][]byte, reflect.ValueOf(args[0]).Len())
		for i := range results {
			results[i] = make([]byte, 512)
		}
		out, err := aggregateABI.Outputs.Pack(big.NewInt(0), results)
		require.NoError(t, err)
		reply(hexutil.Bytes(out))
	}))
	return server, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), blocks...)
	}
}

// swapLog is a Swap of pair in bin 8388608 at block.
func swapLog(t *testing.T, pair common.Address, block uint64, removed bool) ethtypes.Log {
	event := pairABI.Events["Swap"]
	topics := []common.Hash{event.ID}
	var values []any
	for _, input := range event.Inputs {
		if input.Indexed {
			topics = append(topics, common.Hash{})
			continue
		}
		switch input.Type.T {
		case abi.FixedBytesTy:
			values = append(values, [32]byte{})
		case abi.AddressTy:
			values = append(values, common.Address{})
		default:
			values = append(values, big.NewInt(8388608))
		}
	}
	data, err := event.Inputs.NonIndexed().Pack(values...)
	require.NoError(t, err)
	return ethtypes.Log{Address: pair, BlockNumber: block, Topics: topics, Data: data, Removed: removed}
}

// TestReorgReadsBinsWithThePair: a reorg under a pair at block 120 reverting block 110 reads the
// pair's reserves and active bin at 110, its logs' block, and its bins there too, so they add up to
// the reserves read with them.
func TestReorgReadsBinsWithThePair(t *testing.T) {
	node, blocks := zeroNode(t)
	defer node.Close()
	client := ethrpc.New(node.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	tracker := NewPoolTracker(&Config{}, client, nil)

	pair := common.HexToAddress("0x1111111111111111111111111111111111111111")
	p := entity.Pool{Address: strings.ToLower(pair.Hex()), BlockNumber: 120, Extra: `{}`}
	_, _ = tracker.GetNewPoolState(t.Context(), p, poolpkg.GetNewPoolStateParams{
		Logs: []ethtypes.Log{swapLog(t, pair, 110, true), swapLog(t, pair, 110, false)},
	})

	read := blocks()
	require.GreaterOrEqual(t, len(read), 2, "the pair's state and its bins are read")
	for _, block := range read {
		assert.Equal(t, hexutil.EncodeUint64(110), block)
	}
}
