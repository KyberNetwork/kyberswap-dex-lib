package friendtech

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

type LogDecoder struct{}

func NewLogDecoder() *LogDecoder {
	return &LogDecoder{}
}

func (d *LogDecoder) Decode(logs []types.Log) (ReserveData, *big.Int, error) {
	latestSyncEvent := d.findLatestSyncEvent(logs)

	if len(latestSyncEvent.Data) == 0 {
		return ReserveData{}, nil, nil
	}

	filterer, err := NewUniswapFilterer(latestSyncEvent.Address, nil)
	if err != nil {
		return ReserveData{}, nil, err
	}

	syncEvent, err := filterer.ParseSync(latestSyncEvent)
	if err != nil {
		return ReserveData{}, nil, err
	}

	return ReserveData{
		Reserve0: syncEvent.Reserve0,
		Reserve1: syncEvent.Reserve1,
	}, new(big.Int).SetUint64(syncEvent.Raw.BlockNumber), nil
}

func (d *LogDecoder) findLatestSyncEvent(logs []types.Log) types.Log {
	var latestEvent types.Log

	for _, log := range logs {
		if log.Removed {
			continue
		}

		if !d.isSyncEvent(log) {
			continue
		}

		if latestEvent.BlockNumber < log.BlockNumber ||
			(latestEvent.BlockNumber == log.BlockNumber && latestEvent.TxIndex < log.TxIndex) ||
			(latestEvent.BlockNumber == log.BlockNumber && latestEvent.TxIndex == log.TxIndex && latestEvent.Index < log.Index) {
			latestEvent = log
		}
	}

	return latestEvent
}

// isSyncEvent returns true if the first topic is a uniswap-v2 sync event
func (d *LogDecoder) isSyncEvent(log types.Log) bool {
	if len(log.Topics) == 0 {
		return false
	}

	return log.Topics[0] == bunnySwapABI.Events["Sync"].ID
}
