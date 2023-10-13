// Package jsonrpc Add eth_call(callmsg, blockNumber, stateOverride) and debug_traceCall(callmsg, blockNumber, tracer) support.
package jsonrpc

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// EthCallCalldataParam eth_call's calldata param
type EthCallCalldataParam struct {
	From string `json:"from"`
	To   string `json:"to"`
	Gas  string `json:"gas"`
	Data string `json:"data"`
}

// EthCall eth_call wrapper
func EthCall(client *rpc.Client, calldata *EthCallCalldataParam, blockNumber string, override StateOverride) (*string, error) {
	resultHex := new(string)
	args := []interface{}{calldata, blockNumber}
	if override != nil {
		args = append(args, override)
	}
	err := client.Call(resultHex, "eth_call", args...)
	if err != nil {
		return nil, err
	}
	return resultHex, nil
}

// DebugTraceCallCalldataParam debug_traceCall's calldata param
type DebugTraceCallCalldataParam struct {
	From string `json:"from"`
	To   string `json:"to"`
	Gas  string `json:"gas"`
	Data string `json:"data"`
}

// DebugTraceCallTracerConfigParam debug_traceCall's tracer config param
type DebugTraceCallTracerConfigParam struct {
	Tracer         string          `json:"tracer"`
	TracerConfig   json.RawMessage `json:"tracerConfig,omitempty"`
	StateOverrides StateOverride   `json:"stateOverrides,omitempty"`
}

// DebugTraceCall debug_traceCall wrapper
func DebugTraceCall(
	client *rpc.Client,
	calldata *DebugTraceCallCalldataParam,
	blockNumber string,
	tracer *DebugTraceCallTracerConfigParam,
	result interface{},
) error {
	err := client.Call(result, "debug_traceCall", calldata, blockNumber, tracer)
	return err
}

// OverrideAccount similar to ethapi.OverrideAccount
type OverrideAccount struct {
	Nonce     *hexutil.Uint64             `json:"nonce,omitempty"`
	Code      hexutil.Bytes               `json:"code,omitempty"`
	Balance   *hexutil.Big                `json:"balance,omitempty"`
	State     map[common.Hash]common.Hash `json:"state,omitempty"`
	StateDiff map[common.Hash]common.Hash `json:"stateDiff,omitempty"`
}

// StateOverride similar to ethapi.StateOverride
type StateOverride = map[common.Address]OverrideAccount
