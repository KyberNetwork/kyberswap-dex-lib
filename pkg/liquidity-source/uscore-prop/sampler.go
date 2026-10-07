package uscoreprop

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"slices"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

// snapshot mirrors USCoreSampler.Snapshot field for field, so the unpacked tuple converts into it.
type snapshot struct {
	BlockNumber  *big.Int
	Timestamp    *big.Int
	ReserveQuote *big.Int
	ReserveBase  *big.Int
	Points0      []*big.Int
	Outs0        []*big.Int
	Status0      uint8
	Points1      []*big.Int
	Outs1        []*big.Int
	Status1      uint8
	Rounds       uint8
}

var (
	samplerABI      = lo.Must(abi.JSON(bytes.NewReader(samplerABIJson)))
	samplerBytecode = common.FromHex(samplerBytecodeHex)
	// sampleBps is the seed grid in bps of the basis, ladder.BuildSamplePointsN's for SampleSize.
	sampleBps = ladder.BuildSamplePointsN(big.NewInt(10000), ladder.SampleSize)
)

// sample runs USCoreSampler in one to-less eth_call: seed probes, grid and refinement rounds
// all happen on-chain at one block.
func sample(ctx context.Context, client *ethrpc.Client, pool common.Address, tokens [2]common.Address,
	units [2]*uint256.Int, overrides stateOverrides) (*snapshot, error) {
	args, err := samplerABI.Pack("", pool, tokens, [2]*big.Int{units[0].ToBig(), units[1].ToBig()}, sampleBps,
		big.NewInt(maxSamplePoints), big.NewInt(maxRefinementRounds))
	if err != nil {
		return nil, err
	}
	initCode := slices.Concat(samplerBytecode, args)
	data, err := eth.DeploylessCallWithGas(ctx, client.GetETHClient().Client(), initCode, samplerGas, overrides, nil)
	if isBareRevert(err) {
		// The public HyperEVM RPC sporadically (~5% of calls) reverts the sampler without data, and
		// the same call succeeds right after; retry once rather than skip the refresh.
		data, err = eth.DeploylessCallWithGas(ctx, client.GetETHClient().Client(), initCode, samplerGas, overrides, nil)
	}
	if err != nil {
		return nil, err
	}
	return decodeSnapshot(data)
}

func decodeSnapshot(data []byte) (*snapshot, error) {
	sampleErr := samplerABI.Errors["Sample"]
	if len(data) < 4 || !bytes.Equal(data[:4], sampleErr.ID[:4]) {
		return nil, ErrUnexpectedSamplerRevert
	}
	values, err := sampleErr.Inputs.Unpack(data[4:])
	if err != nil {
		return nil, err
	}
	return abi.ConvertType(values[0], new(snapshot)).(*snapshot), nil
}

// isBareRevert reports an execution revert that carries no data.
func isBareRevert(err error) bool {
	var rpcErr rpc.Error
	var dataErr rpc.DataError
	return errors.As(err, &rpcErr) && rpcErr.ErrorCode() == 3 && (!errors.As(err, &dataErr) || dataErr.ErrorData() == nil)
}
