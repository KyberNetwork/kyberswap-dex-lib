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

type (
	DependencyConfig struct {
		Core  common.Address `json:"core"`
		Twamm common.Address `json:"twamm"`
	}

	EventParser struct {
		Core  string
		Twamm string
	}
)

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
	case e.Core:
		return e.handleCoreLog(log)
	case e.Twamm:
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

		orderKey, ok := values[2].(pools.TwammOrderKey)
		if !ok {
			return "", fmt.Errorf("failed to parse orderKey")
		}

		poolKey := pools.NewPoolKey(orderKey.Token0, orderKey.Token1, pools.NewPoolConfig(common.HexToAddress(e.Twamm), orderKey.Fee(), pools.NewFullRangePoolTypeConfig()))

		return poolKey.ToPoolAddress()
	}

	return "", nil
}

func (e *EventParser) GetKeys(_ context.Context) ([]string, error) {
	return []string{
		e.Core,
		e.Twamm,
	}, nil
}

func NewEventParser(config *Config) *EventParser {
	return &EventParser{
		Core:  hexutil.Encode(config.Core[:]),
		Twamm: hexutil.Encode(config.Twamm[:]),
	}
}
