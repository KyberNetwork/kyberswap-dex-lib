// Package ephemeral runs contracts that are never deployed. Their creation code and constructor
// arguments are the data of an eth_call with no to address, and the constructor reverts with the
// result, so one call reads what would otherwise take many, all from one block.
//
// The contracts come from KyberNetwork/ephemeral-contracts, whose release writes each one, its ABI
// and creation code, to a JSON file next to the tracker that uses it (lens.json for its one lens). This package
// is the protocol-agnostic part: loading such a file, running a call, telling a result from a
// failure, and paging a scan that does not fit one call. To read a new dex through such a contract:
//
//  1. Release the contract from ephemeral-contracts with a target in the dex's package.
//  2. Write the dex's adapter next to its tracker: embed the file and Load it, declare the result
//     struct, run the call with Call, and page it with Scan when the contract returns a cursor.
//  3. For a tick-based dex, check the ticks with CheckTicks before trusting them.
package ephemeral

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// Caller runs an eth_call. *ethclient.Client is one.
type Caller interface {
	CallContract(ctx context.Context, msg ethereum.CallMsg, block *big.Int) ([]byte, error)
}

var (
	// ErrNotReverted means the call returned normally, which an ephemeral contract never does: the
	// data is not the contract it was meant to be.
	ErrNotReverted = errors.New("ephemeral: call returned instead of reverting with its result")
	// ErrNoRevertData means the call reverted with nothing: it ran out of gas, or the node strips
	// revert data. Retrying with more gas, or a smaller range, tells the two apart.
	ErrNoRevertData = errors.New("ephemeral: revert carried no data")
)

// RevertError is a revert that is not a result: one of the contract's own errors, a *ContractError,
// or the reason of an Error(string) passed up from a contract it called.
type RevertError struct {
	Err any
}

func (e *RevertError) Error() string {
	return fmt.Sprintf("ephemeral: reverted with %v", e.Err)
}

// Call runs contract with args at block (nil for the latest) under gas (0 for the node's own cap),
// and decodes the result into out, as Contract.UnpackCall does.
//
// An error the node returns without revert data, such as a rate limit or a pruned block, is
// returned as it is: the call never ran.
func Call(ctx context.Context, c Caller, contract *Contract, out any, gas uint64, block *big.Int, args ...any) error {
	data, err := contract.PackCall(args...)
	if err != nil {
		return err
	}
	_, err = c.CallContract(ctx, ethereum.CallMsg{Data: data, Gas: gas}, block)
	if err == nil {
		return ErrNotReverted
	}
	payload, ok := revertData(err)
	if !ok {
		return err
	}
	if len(payload) == 0 {
		return fmt.Errorf("%w: %v", ErrNoRevertData, err)
	}

	unpackErr := contract.UnpackCall(payload, out)
	if unpackErr == nil {
		return nil
	}
	if e, err := contract.UnpackRevert(payload); err == nil {
		return &RevertError{Err: e}
	}
	if reason, err := abi.UnpackRevert(payload); err == nil {
		return &RevertError{Err: reason}
	}
	return fmt.Errorf("ephemeral: %s reverted with neither a result nor a known error: %w", contract.Name, unpackErr)
}

// revertData returns the data of an error the node reports as a revert, and whether the call ran
// and failed at all. Most nodes give the data as a hex string; some wrap it in an object, as
// {"data": "0x..."} or {"result": "0x..."}.
//
// go-ethereum reports every JSON-RPC error as a DataError, those of calls that never ran (an
// unknown block, a rate limit) included, so an error without data is a failed call only when its
// message says the call reverted or ran out of gas.
func revertData(err error) ([]byte, bool) {
	de, ok := errors.AsType[rpc.DataError](err)
	if !ok {
		return nil, false
	}
	data := de.ErrorData()
	if wrapped, ok := data.(map[string]any); ok {
		data = wrapped["data"]
		if _, ok := data.(string); !ok {
			data = wrapped["result"]
		}
	}
	if data == nil {
		msg := strings.ToLower(err.Error())
		return nil, strings.Contains(msg, "execution reverted") || strings.Contains(msg, "out of gas")
	}
	s, ok := data.(string)
	if !ok {
		return nil, true
	}
	b, decodeErr := hexutil.Decode(s)
	if decodeErr != nil {
		return nil, true
	}
	return b, true
}
