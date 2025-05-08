package ekubo

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/pools"
	pooldecoder "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/decode"
)

type DependencyConfig struct {
	Core  common.Address `json:"core"`
	Twamm common.Address `json:"twamm"`
}

type EventParser struct {
	Core  string
	Twamm string
}

var _ = pooldecoder.RegisterFactoryC(DexType, NewEventParser)

func NewEventParser(config *Config) *EventParser {
	return &EventParser{
		Core:  strings.ToLower(config.Core.String()),
		Twamm: strings.ToLower(config.Twamm.String()),
	}
}

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
	logAddress := strings.ToLower(log.Address.String())

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

		orderKey, ok := values[2].(struct {
			SellToken common.Address `json:"sellToken"`
			BuyToken  common.Address `json:"buyToken"`
			Fee       uint64         `json:"fee"`
			StartTime *big.Int       `json:"startTime"`
			EndTime   *big.Int       `json:"endTime"`
		})
		if !ok {
			return "", fmt.Errorf("failed to parse orderKey")
		}

		token0, token1 := sortTokens(orderKey.SellToken, orderKey.BuyToken)

		poolKey := pools.NewPoolKey(
			token0,
			token1,
			pools.PoolConfig{
				Fee:       orderKey.Fee,
				Extension: common.HexToAddress(e.Twamm),
			},
		)

		return poolKey.ToPoolAddress()
	}

	return "", nil
}

func sortTokens(tokenA, tokenB common.Address) (common.Address, common.Address) {
	if tokenB.Cmp(tokenA) == 1 {
		return tokenA, tokenB
	}
	return tokenB, tokenA
}

func (e *EventParser) GetKeys(_ context.Context) ([]string, error) {
	return []string{
		e.Core,
		e.Twamm,
	}, nil
}
