package eth

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var ErrDeploylessNoRevert = errors.New("deployless call: constructor did not revert")

// DeploylessCall runs initCode as a contract-creation eth_call and returns the
// constructor's revert data: a never-deployed helper can run arbitrary read
// logic in its constructor and revert with the result, collapsing several
// dependent round trips into one. "to" is omitted rather than sent as null,
// which not every provider accepts.
func DeploylessCall(
	ctx context.Context, client *rpc.Client, initCode []byte,
	overrides map[common.Address]gethclient.OverrideAccount, blockOverrides *gethclient.BlockOverrides,
) ([]byte, error) {
	return DeploylessCallWithGas(ctx, client, initCode, 0, overrides, blockOverrides)
}

// DeploylessCallWithGas is DeploylessCall with an explicit gas limit (0 leaves it to the node).
// Set it when the helper needs more than a node's default: some default to the block gas limit,
// and an out-of-gas sub-call then surfaces as a revert without data.
func DeploylessCallWithGas(
	ctx context.Context, client *rpc.Client, initCode []byte, gas uint64,
	overrides map[common.Address]gethclient.OverrideAccount, blockOverrides *gethclient.BlockOverrides,
) ([]byte, error) {
	call := map[string]any{"data": hexutil.Bytes(initCode)}
	if gas > 0 {
		call["gas"] = hexutil.Uint64(gas)
	}
	args := []any{call, "latest"}
	if len(overrides) > 0 || blockOverrides != nil {
		if overrides == nil {
			overrides = map[common.Address]gethclient.OverrideAccount{}
		}
		args = append(args, overrides)
	}
	if blockOverrides != nil {
		args = append(args, blockOverrides)
	}

	var res hexutil.Bytes
	err := client.CallContext(ctx, &res, "eth_call", args...)
	if err == nil {
		return nil, ErrDeploylessNoRevert
	}

	var dataErr rpc.DataError
	if !errors.As(err, &dataErr) {
		return nil, err
	}
	hexData, ok := dataErr.ErrorData().(string)
	if !ok {
		return nil, err
	}
	data, decodeErr := hexutil.Decode(hexData)
	if decodeErr != nil {
		return nil, err
	}
	return data, nil
}
