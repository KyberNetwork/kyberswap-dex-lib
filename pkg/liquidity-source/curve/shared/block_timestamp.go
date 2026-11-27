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

// DefaultMulticall3 is Multicall3's address on nearly every EVM chain (one deterministic deployment). ethrpc's
// client has a multicall too but does not expose its address, so a config without multicall_address uses this one.
const DefaultMulticall3 = "0xcA11bde05977b3631167028862bE2a173976CA11"

// AddBlockTimestampCall adds Multicall3's getCurrentBlockTimestamp() to a refresh's batch, at the config's
// multicall_address or else DefaultMulticall3, so the timestamp is the block.timestamp of the batch's own block.
// The batch tolerates a failing call: on a chain with no Multicall3 there, BlockTimestamp reads it at the block.
func (c *Config) AddBlockTimestampCall(add func(*ethrpc.Call, []any), ts **big.Int) {
	target := c.MulticallAddress
	if target == "" {
		target = DefaultMulticall3
	}
	add(&ethrpc.Call{ABI: abi.Multicall3ABI, Target: target, Method: ethrpc.MethodGetCurrentBlockTimestamp},
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
