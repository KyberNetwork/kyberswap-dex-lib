// Package rpcreplay serves a tracker's multicall batches from a fixture of single calls, for tracker tests that
// replay a pool at one block. A fixture maps each (target, calldata) to its return data, or to a revert; a call the
// fixture does not hold answers as a revert, so the tracker sees what a node gives for a failing call.
package rpcreplay

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// Fixture is the state of one chain at one block, as the answers to single calls.
type Fixture struct {
	Block uint64             `json:"block"`
	Calls map[string]*string `json:"calls"` // "target:calldata" (lower-case hex) -> return data, null for a revert
}

// Call is one call the server was asked, whether or not its fixture held an answer.
type Call struct {
	Target common.Address
	Data   []byte
}

type Server struct {
	*testutil.NodeStub
	mu    sync.Mutex
	asked []Call
}

// New serves file at its block.
func New(t *testing.T, file string) *Server {
	t.Helper()
	raw, err := os.ReadFile(file)
	require.NoError(t, err)
	var fx Fixture
	require.NoError(t, json.Unmarshal(raw, &fx))
	s := &Server{}
	respond := func(target common.Address, data []byte) (bool, []byte) {
		s.mu.Lock()
		s.asked = append(s.asked, Call{Target: target, Data: bytes.Clone(data)})
		s.mu.Unlock()
		v := fx.Calls[strings.ToLower(target.Hex())+":"+hex.EncodeToString(data)]
		if v == nil {
			return false, nil
		}
		return true, hexutil.MustDecode(*v)
	}
	s.NodeStub = testutil.NewNodeStub(t, 1, fx.Block, func(c testutil.NodeCall) ([]byte, error) {
		ret, _, err := testutil.Multicall(*c.To, c.Data, fx.Block, 0, respond)
		return ret, err
	})
	return s
}

// Client returns an ethrpc client reading from the fixture.
func (s *Server) Client() *ethrpc.Client { return s.NodeStub.Client() }

// Calls returns every call asked so far, in order, each multicall's calls included.
func (s *Server) Calls() []Call {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.asked)
}

// Selectors counts the asked calls by their 4-byte selector, as lower-case hex.
func (s *Server) Selectors() map[string]int {
	out := map[string]int{}
	for _, c := range s.Calls() {
		if len(c.Data) >= 4 {
			out[hex.EncodeToString(c.Data[:4])]++
		}
	}
	return out
}

// GetQuotesSelector is the selector of the oracle router's getQuotes(uint256,address,address).
const GetQuotesSelector = "0579e61f"

// QuotedAmounts is the inAmount of every getQuotes call asked, in order.
func (s *Server) QuotedAmounts() []*big.Int {
	var out []*big.Int
	for _, c := range s.Calls() {
		if len(c.Data) >= 36 && hex.EncodeToString(c.Data[:4]) == GetQuotesSelector {
			out = append(out, new(big.Int).SetBytes(c.Data[4:36]))
		}
	}
	return out
}
