package shared

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// DefaultLogBlockRange is the max block span of one eth_getLogs call.
const DefaultLogBlockRange uint64 = 500_000

const logsMaxAttempts = 5

// FilterLogs reads the logs of `addr` matching `topics` in [from, to], chunked by blockRange blocks.
func FilterLogs(ctx context.Context, client *ethrpc.Client, addr common.Address, topics [][]common.Hash,
	from, to, blockRange uint64) ([]ethtypes.Log, error) {
	if blockRange == 0 {
		blockRange = DefaultLogBlockRange
	}
	var logs []ethtypes.Log
	for start := from; start <= to; start += blockRange {
		q := ethereum.FilterQuery{
			FromBlock: new(big.Int).SetUint64(start),
			ToBlock:   new(big.Int).SetUint64(min(start+blockRange-1, to)),
			Addresses: []common.Address{addr},
			Topics:    topics,
		}
		var chunk []ethtypes.Log
		var err error
		for attempt := range logsMaxAttempts { // rate-limited nodes answer 429
			if chunk, err = client.GetETHClient().FilterLogs(ctx, q); err == nil {
				break
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt+1) * 500 * time.Millisecond):
			}
		}
		if err != nil {
			return nil, err
		}
		logs = append(logs, chunk...)
	}
	return logs, nil
}
