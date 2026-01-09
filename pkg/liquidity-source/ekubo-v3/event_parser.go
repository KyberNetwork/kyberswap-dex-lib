package ekubov3

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo-v3/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo-v3/pools"
	pooldecode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/decode"
)

var _ = pooldecode.RegisterFactoryC(DexType, NewEventParser)

type EventParser struct{}

func (e *EventParser) Decode(_ context.Context, logs []types.Log) (map[string][]types.Log, error) {
	addressLogs := make(map[string][]types.Log)
	for _, log := range logs {
		poolAddress, err := e.getPoolAddress(log)
		if err != nil {
			return nil, err
		}

		if poolAddress != "" {
			addressLogs[poolAddress] = append(addressLogs[poolAddress], log)
		}
	}
	return addressLogs, nil
}

func (e *EventParser) getPoolAddress(log types.Log) (string, error) {
	logAddress := hexutil.Encode(log.Address[:])

	switch logAddress {
	case CoreAddressStrLower:
		return e.handleCoreLog(log)
	case TwammAddressStrLower:
		return e.handleTwammLog(log)
	default:
		return "", nil
	}
}

func (e *EventParser) handleCoreLog(log types.Log) (string, error) {
	if len(log.Topics) == 0 {
		if len(log.Data) < 52 {
			return "", fmt.Errorf("invalid data length for Swapped event")
		}

		return "0x" + common.Bytes2Hex(log.Data[20:52]), nil
	}

	if log.Topics[0] == abis.PositionUpdatedEvent.ID {
		values, err := abis.PositionUpdatedEvent.Inputs.Unpack(log.Data)
		if err != nil {
			return "", err
		}

		poolId, ok := values[1].([32]byte)
		if !ok {
			return "", fmt.Errorf("failed to parse poolId from PositionUpdated event data")
		}

		return "0x" + common.Bytes2Hex(poolId[:]), nil
	}

	return "", nil
}

func (e *EventParser) handleTwammLog(log types.Log) (string, error) {
	if len(log.Topics) == 0 {
		if len(log.Data) < 32 {
			return "", fmt.Errorf("invalid data length for VirtualOrdersExecuted event")
		}

		return "0x" + common.Bytes2Hex(log.Data[0:32]), nil
	}

	if log.Topics[0] == abis.OrderUpdatedEvent.ID {
		values, err := abis.OrderUpdatedEvent.Inputs.Unpack(log.Data)
		if err != nil {
			return "", err
		}

		orderKeyAbi, ok := values[2].(pools.TwammOrderKeyAbi)
		if !ok {
			return "", fmt.Errorf("failed to parse orderKey")
		}
		orderKey := pools.TwammOrderKey{TwammOrderKeyAbi: orderKeyAbi}

		poolKey := pools.NewPoolKey(orderKey.Token0, orderKey.Token1, pools.NewPoolConfig(TwammAddress, orderKey.Fee(), pools.NewFullRangePoolTypeConfig()))

		return poolKey.ToPoolAddress()
	}

	return "", nil
}

func (e *EventParser) GetKeys(_ context.Context) ([]string, error) {
	return []string{
		CoreAddressStrLower,
		TwammAddressStrLower,
	}, nil
}

func NewEventParser(cfg *struct{}) *EventParser {
	return &EventParser{}
}
