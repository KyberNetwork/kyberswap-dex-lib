package shared

import (
	"context"
	"errors"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

// ErrNoReadBlock means a refresh has neither a block timestamp from its batch nor a block to read it at.
var ErrNoReadBlock = errors.New("no block to read the block timestamp at")

// ValidateMulticallAddress checks the optional multicall_address.
func (c *Config) ValidateMulticallAddress() error {
	if c.MulticallAddress != "" && !common.IsHexAddress(c.MulticallAddress) {
		return errors.New("invalid multicall_address: " + c.MulticallAddress)
	}
	return nil
}

// AddBlockTimestampCall adds Multicall3's getCurrentBlockTimestamp() to a refresh's batch when the config names
// the multicall, so the timestamp is the block.timestamp of the batch's own block. Without one it adds nothing
// and BlockTimestamp reads it at the batch's block.
func (c *Config) AddBlockTimestampCall(add func(*ethrpc.Call, []any), ts **big.Int) {
	if c.MulticallAddress == "" {
		return
	}
	add(&ethrpc.Call{ABI: abi.Multicall3ABI, Target: c.MulticallAddress, Method: ethrpc.MethodGetCurrentBlockTimestamp},
		[]any{ts})
}

// BlockTimestamp is the block.timestamp of the block a refresh read at: the one its batch returned, else one
// read at that block.
func BlockTimestamp(ctx context.Context, client *ethrpc.Client,
	overrides map[common.Address]gethclient.OverrideAccount, block, ts *big.Int) (int64, error) {
	if ts != nil && ts.Sign() > 0 {
		return ts.Int64(), nil
	}
	if block == nil {
		return 0, ErrNoReadBlock
	}
	req := client.NewRequest().SetContext(ctx).SetBlockNumber(block)
	if overrides != nil {
		req.SetOverrides(overrides)
	}
	read, err := req.GetCurrentBlockTimestamp()
	if err != nil {
		return 0, err
	}
	return int64(read), nil
}
