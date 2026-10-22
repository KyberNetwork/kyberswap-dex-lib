package v2

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

// TestLensTooLarge: only running out of gas splits a batch; the lens's other error and a node
// failure do not, since a smaller batch cannot fix them.
func TestLensTooLarge(t *testing.T) {
	assert.True(t, lensTooLarge(&ephemeral.RevertError{Err: &ephemeral.ContractError{Name: "GasExhausted"}}))
	assert.True(t, lensTooLarge(fmt.Errorf("%w: out of gas", ephemeral.ErrNoRevertData)))
	assert.False(t, lensTooLarge(&ephemeral.RevertError{Err: &ephemeral.ContractError{Name: "MismatchedVaultInput"}}))
	assert.False(t, lensTooLarge(errors.New("429 too many requests")))
}

// lensChain is a node whose state lens reads every pool with empty rounds, but reverts with nothing
// on a call that holds bad, as a pool whose read always runs the lens out of gas makes it. It
// counts the lens calls and the multicalls it is sent; it answers no multicall.
type lensChain struct {
	*httptest.Server
	bad                   common.Address
	lensCalls, multicalls atomic.Int64
}

func newLensChain(t *testing.T, bad common.Address) *lensChain {
	var doc struct {
		ABI      json.RawMessage `json:"abi"`
		Bytecode hexutil.Bytes   `json:"bytecode"`
	}
	require.NoError(t, json.Unmarshal(lensJSON, &doc))
	parsed, err := abi.JSON(bytes.NewReader(doc.ABI))
	require.NoError(t, err)

	c := &lensChain{bad: bad}
	c.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &req))
		reply := func(field string, v any) {
			b, _ := json.Marshal(v)
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,%q:%s}`, req.ID, field, b)
		}
		switch req.Method {
		case "eth_chainId":
			reply("result", hexutil.Uint64(1))
		case "eth_blockNumber":
			reply("result", hexutil.Uint64(100))
		case "eth_call":
			var msg struct {
				To    *common.Address `json:"to"`
				Input hexutil.Bytes   `json:"input"`
				Data  hexutil.Bytes   `json:"data"`
				Gas   hexutil.Uint64  `json:"gas"`
			}
			require.NoError(t, json.Unmarshal(req.Params[0], &msg))
			if msg.To != nil {
				c.multicalls.Add(1)
				reply("error", map[string]any{"code": -32000, "message": "no multicall here"})
				return
			}
			data := append(msg.Data, msg.Input...)
			if bytes.Equal(data, []byte{0x5a, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}) {
				// gas cap probe: run with what it asked, capped at 50M
				g := min(uint64(msg.Gas), 50_000_000) - 53_124
				reply("result", hexutil.Bytes(common.BigToHash(new(big.Int).SetUint64(g)).Bytes()))
				return
			}
			c.lensCalls.Add(1)
			args, err := parsed.Constructor.Inputs.Unpack(data[len(doc.Bytecode):])
			require.NoError(t, err)
			part := *abi.ConvertType(args[0], new([]poolInput)).(*[]poolInput)
			for _, p := range part {
				if p.Pool == c.bad {
					reply("error", map[string]any{"code": 3, "message": "execution reverted"})
					return
				}
			}
			out, err := parsed.Methods["getPoolStates"].Outputs.Pack(big.NewInt(100), make([]poolReads, len(part)))
			require.NoError(t, err)
			reply("error", map[string]any{"code": 3, "message": "execution reverted", "data": hexutil.Bytes(out)})
		default:
			t.Errorf("unexpected %s", req.Method)
		}
	}))
	t.Cleanup(c.Close)
	return c
}

// TestLensReadsAroundAPoolItCannotRead: a pool the lens cannot read even alone gets its error, and
// the others are read through the lens all the same.
func TestLensReadsAroundAPoolItCannotRead(t *testing.T) {
	bad := common.HexToAddress("0xbad")
	chain := newLensChain(t, bad)
	tr, err := NewPoolTracker(&shared.Config{UseLens: true}, ethrpc.New(chain.URL))
	require.NoError(t, err)

	inputs := []poolInput{{Pool: common.HexToAddress("0x1")}, {Pool: bad}, {Pool: common.HexToAddress("0x3")}}
	block, reads, err := tr.readLens(t.Context(), inputs, nil, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 100, block)
	require.Len(t, reads, 3)
	assert.NoError(t, reads[0].err)
	assert.ErrorIs(t, reads[1].err, ephemeral.ErrNoRevertData)
	assert.NoError(t, reads[2].err)
}

// TestBatchWithoutLensReadsPoolsAlone: without UseLens a batch reads each pool the usual way and
// never calls the lens.
func TestBatchWithoutLensReadsPoolsAlone(t *testing.T) {
	chain := newLensChain(t, common.Address{})
	client := ethrpc.New(chain.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	tr, err := NewPoolTracker(&shared.Config{}, client)
	require.NoError(t, err)

	p := entity.Pool{
		Address:     "0x1",
		Tokens:      []*entity.PoolToken{{Address: "0xa", Decimals: 6}, {Address: "0xb", Decimals: 18}},
		StaticExtra: `{"sv0":"0x10","sv1":"0x11","ea":"0x12","evc":"0x13"}`,
	}
	_, errs := tr.GetNewPoolStates(t.Context(), []entity.Pool{p, p})
	assert.Error(t, errs[0])
	assert.Error(t, errs[1])
	assert.Zero(t, chain.lensCalls.Load(), "the lens was called")
	assert.EqualValues(t, 2, chain.multicalls.Load())
}
