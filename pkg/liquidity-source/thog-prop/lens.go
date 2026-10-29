package thogprop

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

var (
	snapshotLensABI    = mustLensABI()
	snapshotLensCode   = common.FromHex(strings.TrimSpace(snapshotLensBytecodeHex))
	snapshotLensResult = lensResultArguments()
)

func mustLensABI() abi.ABI {
	a, err := abi.JSON(bytes.NewReader(snapshotLensABIBytes))
	if err != nil {
		panic(err)
	}
	return a
}

func lensResultArguments() abi.Arguments {
	var args abi.Arguments
	for _, name := range []string{"bytes4", "uint256", "uint256", "bytes", "address[]", "uint256[]", "uint256[]"} {
		t, err := abi.NewType(name, "", nil)
		if err != nil {
			panic(err)
		}
		args = append(args, abi.Argument{Type: t})
	}
	return args
}

func readSnapshotLens(ctx context.Context, client *rpc.Client, proxy common.Address) (State, uint64, error) {
	args, err := snapshotLensABI.Pack("", proxy)
	if err != nil {
		return State{}, 0, err
	}
	initCode := append(append([]byte{}, snapshotLensCode...), args...)
	gasPrice := (*hexutil.Big)(big.NewInt(1))
	data, err := eth.DeploylessCallWithGasPrice(ctx, client, initCode, snapshotLensGas, gasPrice, nil, nil)
	if err != nil {
		return State{}, 0, err
	}
	return decodeSnapshotLens(data, proxy)
}

func decodeSnapshotLens(data []byte, proxy common.Address) (State, uint64, error) {
	failed := snapshotLensABI.Errors["ReadFailed"]
	if len(data) >= 4 && bytes.Equal(data[:4], failed.ID[:4]) {
		values, err := failed.Inputs.Unpack(data[4:])
		if err != nil {
			return State{}, 0, refreshFailure("validation", "lens failure payload", err)
		}
		target := values[0].(common.Address)
		kind := "issuer"
		if target == proxy {
			kind = "contract"
		}
		return State{}, 0, refreshFailure(kind, "lens read", fmt.Errorf("target %s selector %x: 0x%x", target, values[1].([4]byte), values[2].([]byte)))
	}
	values, err := snapshotLensResult.Unpack(data)
	if err != nil {
		return State{}, 0, refreshFailure("validation", "lens result", err)
	}
	if values[0].([4]byte) != [4]byte{'T', 'H', 'G', '8'} {
		return State{}, 0, refreshFailure("validation", "lens tag", ErrInvalidState)
	}
	block, timestamp := values[1].(*big.Int), values[2].(*big.Int)
	if !block.IsUint64() || !timestamp.IsUint64() || timestamp.Sign() <= 0 {
		return State{}, 0, refreshFailure("validation", "lens context", ErrInvalidState)
	}
	page, err := decodeSnapshot(values[3].([]byte))
	if err != nil {
		return State{}, 0, refreshFailure("validation", "lens page", err)
	}
	if page.BlockNumber == nil || page.BlockNumber.Cmp(block) != 0 {
		return State{}, 0, refreshFailure("validation", "lens block", ErrInvalidState)
	}
	issuers, amounts, multipliers := values[4].([]common.Address), values[5].([]*big.Int), values[6].([]*big.Int)
	if len(issuers) != len(amounts) || len(issuers) != len(multipliers) || len(issuers) > maxTokens {
		return State{}, 0, refreshFailure("validation", "lens issuers", ErrInvalidState)
	}
	shares := make(map[string]ShareBalance, len(issuers))
	for i, issuer := range issuers {
		key := hexutil.Encode(issuer[:])
		if _, exists := shares[key]; exists || !isXStock(key) {
			return State{}, 0, refreshFailure("validation", "lens issuer identity", ErrInvalidState)
		}
		var holding ShareBalance
		holding.Shares.SetBytes(amounts[i].Bytes())
		holding.Multiplier.SetBytes(multipliers[i].Bytes())
		shares[key] = holding
	}
	state, err := snapshotState(page, shares)
	if err != nil {
		return State{}, 0, refreshFailure("validation", "lens snapshot", err)
	}
	return state, timestamp.Uint64(), nil
}

// Only a provider capability/gas rejection selects the number-pinned reader.
// Contract/issuer errors and malformed payloads remain explicit refresh failures.
func lensUnsupported(err error) bool {
	var readFailure *refreshError
	if errors.As(err, &readFailure) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "gas") && (strings.Contains(message, "out of gas") || strings.Contains(message, "exceed") || strings.Contains(message, "limit") || strings.Contains(message, "allowance")) {
		return true
	}
	return strings.Contains(message, "creat") && (strings.Contains(message, "unsupported") || strings.Contains(message, "not supported") || strings.Contains(message, "not allowed") || strings.Contains(message, "disabled"))
}
