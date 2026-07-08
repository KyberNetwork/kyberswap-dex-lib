package nadswap

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type ILogDecoder interface {
	Decode(logs []types.Log, blockHeaders map[uint64]entity.BlockHeader) (ReserveData, *big.Int, error)
}

type LogDecoder struct{}

func NewLogDecoder() *LogDecoder { return &LogDecoder{} }

func (d *LogDecoder) Decode(logs []types.Log, _ map[uint64]entity.BlockHeader) (ReserveData, *big.Int, error) {
	latest := d.findLatestSyncEvent(logs)
	if len(latest.Data) == 0 {
		return ReserveData{}, nil, nil
	}

	event := pairABI.Events["Sync"]
	unpacked, err := event.Inputs.Unpack(latest.Data)
	if err != nil {
		return ReserveData{}, nil, err
	}
	if len(unpacked) != 2 {
		return ReserveData{}, nil, nil
	}
	r0 := unpacked[0].(*big.Int)
	r1 := unpacked[1].(*big.Int)

	u0, overflow := uint256.FromBig(r0)
	if overflow {
		return ReserveData{}, nil, ErrOverflow
	}
	u1, overflow := uint256.FromBig(r1)
	if overflow {
		return ReserveData{}, nil, ErrOverflow
	}
	return ReserveData{Reserve0: u0, Reserve1: u1}, new(big.Int).SetUint64(latest.BlockNumber), nil
}

func (d *LogDecoder) findLatestSyncEvent(logs []types.Log) types.Log {
	var latest types.Log
	syncTopic := pairABI.Events["Sync"].ID
	for _, log := range logs {
		if log.Removed || len(log.Topics) == 0 || log.Topics[0] != syncTopic {
			continue
		}
		if latest.BlockNumber < log.BlockNumber ||
			(latest.BlockNumber == log.BlockNumber && latest.TxIndex < log.TxIndex) ||
			(latest.BlockNumber == log.BlockNumber && latest.TxIndex == log.TxIndex && latest.Index < log.Index) {
			latest = log
		}
	}
	return latest
}
