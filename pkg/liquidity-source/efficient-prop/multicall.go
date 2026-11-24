package efficientprop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
	abiutil "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

// call is one call of a multicall the tracker sends itself, as one element of a JSON-RPC batch, so that it can carry
// a block override, which ethrpc's requests cannot.
type call struct {
	abi    *abi.ABI
	target common.Address
	method string
	params []any
}

type mcCall struct {
	Target   common.Address
	CallData []byte
}

type mcResult struct {
	Success    bool
	ReturnData []byte
}

// override is an eth_call block override: the block's unix second and, where not zero, its number; where baseTimeMs
// is not zero, a state override of BaseTime's slot 0 (BaseTime at baseTime), the sub-second it adds to the block's
// second, so BaseTime answers baseTimeMs (frontClock.place); and, where state is set, a /state answer's overrides,
// the slots its target block's updates write (a front read through /state). The state is otherwise the pinned
// block's: the chain writes BaseTime's sub-second each block, and a block override does not move it.
type override struct {
	time, number uint64
	baseTimeMs   uint64
	baseTime     common.Address
	state        *titan.State
}

// multicallElem is an eth_call of Multicall3's tryAggregate(false, calls) pinned to block, run in the block o names
// where o.time is not zero (a block override), with BaseTime set as o says, its raw answer into out.
func multicallElem(multicall common.Address, calls []call, block *big.Int, o override,
	out *hexutil.Bytes) (rpc.BatchElem, error) {
	mc := make([]mcCall, len(calls))
	for k, c := range calls {
		data, err := c.abi.Pack(c.method, c.params...)
		if err != nil {
			return rpc.BatchElem{}, err
		}
		mc[k] = mcCall{Target: c.target, CallData: data}
	}
	data, err := abiutil.Multicall3ABI.Pack("tryAggregate", false, mc)
	if err != nil {
		return rpc.BatchElem{}, err
	}
	args := []any{map[string]any{"to": multicall, "data": hexutil.Bytes(data)}, hexutil.EncodeBig(block)}
	return rpc.BatchElem{Method: "eth_call", Args: append(args, overrideArgs(o)...), Result: out}, nil
}

// overrideArgs are an eth_call's state and block overrides for o; none where o.time is zero.
func overrideArgs(o override) []any {
	if o.time == 0 {
		return nil
	}
	blockOverride := map[string]any{"time": hexutil.Uint64(o.time)}
	if o.number != 0 {
		blockOverride["number"] = hexutil.Uint64(o.number)
	}
	state := map[string]any{}
	if o.state != nil {
		for a, acct := range o.state.Overrides {
			state[a.Hex()] = acct
		}
	}
	if o.baseTimeMs != 0 {
		state[o.baseTime.Hex()] = map[string]any{"stateDiff": map[string]any{
			common.Hash{}.Hex(): common.BigToHash(new(big.Int).SetUint64(o.baseTimeMs % 1000)).Hex()}}
	}
	return []any{state, blockOverride}
}

// unpackTry is a tryAggregate answer of n calls: each call's return data, nil where it reverted.
func unpackTry(raw []byte, n int) ([][]byte, error) {
	vals, err := abiutil.Multicall3ABI.Unpack("tryAggregate", raw)
	if err != nil || len(vals) != 1 {
		return nil, ErrInvalidState
	}
	res := *abi.ConvertType(vals[0], new([]mcResult)).(*[]mcResult)
	if len(res) != n {
		return nil, ErrInvalidState
	}
	out := make([][]byte, n)
	for k, r := range res {
		if r.Success {
			out[k] = r.ReturnData
			if out[k] == nil {
				out[k] = []byte{}
			}
		}
	}
	return out, nil
}

// unpackUint is a call's one uint256 answer; nil where it reverted or answered something else.
func unpackUint(a *abi.ABI, method string, ret []byte) *big.Int {
	if ret == nil {
		return nil
	}
	vals, err := a.Methods[method].Outputs.Unpack(ret)
	if err != nil || len(vals) == 0 {
		return nil
	}
	v, _ := vals[0].(*big.Int)
	return v
}
