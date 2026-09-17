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
	args := []any{map[string]any{"data": hexutil.Bytes(initCode)}, "latest"}
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
