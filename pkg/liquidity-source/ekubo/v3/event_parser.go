package ekubov3

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)

type EventParser struct {
	Core  string
	Twamm string
}

func NewPoolFactory(config *Config) *EventParser {
	return &EventParser{
		Core:  hexutil.Encode(config.Core[:]),
		Twamm: hexutil.Encode(config.Twamm[:]),
	}
}

func (e *EventParser) Decode(ctx context.Context, logs []types.Log) (map[string][]types.Log, error) {
	addressLogs := make(map[string][]types.Log)
	for _, log := range logs {
		poolAddresses, err := e.DecodePoolAddressesFromFactoryLog(ctx, log)
		if err != nil {
			return nil, err
		}

		for _, poolAddress := range poolAddresses {
			addressLogs[poolAddress] = append(addressLogs[poolAddress], log)
		}
	}
	return addressLogs, nil
}

func (e *EventParser) DecodePoolAddressesFromFactoryLog(_ context.Context, log types.Log) ([]string, error) {
	logAddress := hexutil.Encode(log.Address[:])

	switch logAddress {
	case e.Core:
		return e.handleCoreLog(log)
	case e.Twamm:
		return e.handleTwammLog(log)
	default:
		return nil, nil
	}
}

func (e *EventParser) handleCoreLog(log types.Log) ([]string, error) {
	if len(log.Topics) == 0 {
		if len(log.Data) < 52 {
			return nil, fmt.Errorf("invalid data length for Swapped event")
		}

		return []string{"0x" + common.Bytes2Hex(log.Data[20:52])}, nil
	}

	if log.Topics[0] == abis.PositionUpdatedEvent.ID {
		values, err := abis.PositionUpdatedEvent.Inputs.Unpack(log.Data)
		if err != nil {
			return nil, err
		}

		poolId, ok := values[1].([32]byte)
		if !ok {
			return nil, fmt.Errorf("failed to parse poolId from PositionUpdated event data")
		}

		return []string{"0x" + common.Bytes2Hex(poolId[:])}, nil
	}

	return nil, nil
}

func (e *EventParser) handleTwammLog(log types.Log) ([]string, error) {
	if len(log.Topics) == 0 {
		if len(log.Data) < 32 {
			return nil, fmt.Errorf("invalid data length for VirtualOrdersExecuted event")
		}

		return []string{"0x" + common.Bytes2Hex(log.Data[0:32])}, nil
	}

	if log.Topics[0] == abis.OrderUpdatedEvent.ID {
		values, err := abis.OrderUpdatedEvent.Inputs.Unpack(log.Data)
		if err != nil {
			return nil, err
		}

		orderKeyAbi, ok := values[2].(pools.TwammOrderKeyAbi)
		if !ok {
			return nil, fmt.Errorf("failed to parse orderKey")
		}
		orderKey := pools.TwammOrderKey{TwammOrderKeyAbi: orderKeyAbi}

		poolKey := pools.NewPoolKey(orderKey.Token0, orderKey.Token1,
			pools.NewPoolConfig(common.HexToAddress(e.Twamm), orderKey.Fee(), pools.NewFullRangePoolTypeConfig()))

		poolAddress, err := poolKey.ToPoolAddress()
		if err != nil {
			return nil, err
		}

		return []string{poolAddress}, nil
	}

	return nil, nil
}

func (ep *EventParser) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	// TODO: Implement this (non tick-based pool creation)
	return nil, nil
}

func (ep *EventParser) IsEventSupported(event common.Hash) bool {
	// TODO: Implement this (non tick-based pool creation)
	return true
}
