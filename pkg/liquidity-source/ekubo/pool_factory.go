package ekubo

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/pools"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

var _ = poolfactory.RegisterFactoryC(DexType, NewPoolFactory)

type PoolFactory struct {
	cfg   *Config
	Core  string
	Twamm string
}

func NewPoolFactory(config *Config) *PoolFactory {
	return &PoolFactory{
		cfg:   config,
		Core:  hexutil.Encode(config.Core[:]),
		Twamm: hexutil.Encode(config.Twamm[:]),
	}
}

func (f *PoolFactory) DecodePoolCreated(event types.Log) (*entity.Pool, error) {
	return nil, nil
}

func (f *PoolFactory) IsEventSupported(event common.Hash) bool {
	return true
}

func (f *PoolFactory) DecodePoolAddress(ctx context.Context, log types.Log) ([]string, error) {
	logAddress := hexutil.Encode(log.Address[:])

	switch logAddress {
	case f.Core:
		return f.handleCoreLog(log)
	case f.Twamm:
		return f.handleTwammLog(log)
	default:
		return nil, nil
	}
}

func (f *PoolFactory) handleCoreLog(log types.Log) ([]string, error) {
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

func (f *PoolFactory) handleTwammLog(log types.Log) ([]string, error) {
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

		orderKey, ok := values[2].(struct {
			SellToken common.Address `json:"sellToken"`
			BuyToken  common.Address `json:"buyToken"`
			Fee       uint64         `json:"fee"`
			StartTime *big.Int       `json:"startTime"`
			EndTime   *big.Int       `json:"endTime"`
		})
		if !ok {
			return nil, fmt.Errorf("failed to parse orderKey")
		}

		token0, token1 := sort(orderKey.SellToken, orderKey.BuyToken)

		poolKey := pools.NewPoolKey(
			token0,
			token1,
			pools.PoolConfig{
				Fee:       orderKey.Fee,
				Extension: common.HexToAddress(f.Twamm),
			},
		)

		poolAddress, err := poolKey.ToPoolAddress()
		if err != nil {
			return nil, err
		}

		return []string{poolAddress}, nil
	}

	return nil, nil
}

func sort(tokenA, tokenB common.Address) (common.Address, common.Address) {
	if tokenB.Cmp(tokenA) == 1 {
		return tokenA, tokenB
	}
	return tokenB, tokenA
}
