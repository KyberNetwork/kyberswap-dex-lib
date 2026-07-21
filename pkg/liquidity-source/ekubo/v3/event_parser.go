package ekubov3

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = poolfactory.RegisterFactoryCE(DexType, NewPoolFactory)

type EventParser struct {
	config       *Config
	dataFetchers *dataFetchers
}

func NewPoolFactory(config *Config, ethrpcClient *ethrpc.Client) *EventParser {
	return &EventParser{
		config:       config,
		dataFetchers: NewDataFetchers(ethrpcClient, config),
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
	switch log.Address {
	case e.config.Core:
		return e.handleCoreLog(log)
	case e.config.Twamm.V1.Address, e.config.Twamm.V2.Address:
		return e.handleTwammLog(log, log.Address)
	case e.config.BoostedFeesConcentrated:
		return e.handleBoostedFeesLog(log)
	case e.config.Ve33:
		return e.handleVe33Log(log)
	default:
		return nil, nil
	}
}

func (e *EventParser) handleVe33Log(log types.Log) ([]string, error) {
	if len(log.Topics) == 0 || log.Topics[0] != abis.VoteWeightAppliedEvent.ID {
		return nil, nil
	}
	values, err := abis.VoteWeightAppliedEvent.Inputs.Unpack(log.Data)
	if err != nil {
		return nil, err
	}
	poolID, ok := values[2].([32]byte)
	if !ok {
		return nil, fmt.Errorf("failed to parse poolId from VoteWeightApplied event data")
	}

	return []string{"0x" + common.Bytes2Hex(poolID[:])}, nil
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

func (e *EventParser) handleTwammLog(log types.Log, twamm common.Address) ([]string, error) {
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
			pools.NewPoolConfig(twamm, orderKey.Fee(), pools.NewFullRangePoolTypeConfig()))

		poolAddress, err := poolKey.ToPoolAddress()
		if err != nil {
			return nil, err
		}

		return []string{poolAddress}, nil
	}

	return nil, nil
}

// handleBoostedFeesLog decodes FeesDonated (anonymous) and PoolBoosted events from the
// BoostedFeesConcentrated contract. Both events encode the poolId as their first 32 bytes,
// mirroring BoostedFeesPool.ApplyEvent's own parsing in pools/boosted_fees.go.
func (e *EventParser) handleBoostedFeesLog(log types.Log) ([]string, error) {
	if len(log.Topics) == 0 {
		if len(log.Data) < 60 {
			return nil, fmt.Errorf("invalid data length for FeesDonated event: %d", len(log.Data))
		}

		return []string{"0x" + common.Bytes2Hex(log.Data[0:32])}, nil
	}

	if log.Topics[0] == abis.PoolBoostedEvent.ID {
		if len(log.Data) < 32 {
			return nil, fmt.Errorf("invalid data length for PoolBoosted event")
		}

		return []string{"0x" + common.Bytes2Hex(log.Data[0:32])}, nil
	}

	return nil, nil
}

func (e *EventParser) IsEventSupported(event common.Hash) bool {
	return event == abis.PoolInitializedEvent.ID
}

// DecodePoolCreated decodes a PoolInitialized event from the Core contract.
//
// PoolInitialized ABI-encodes (non-indexed):
//
//	[0:32]   poolId    bytes32
//	[32:64]  token0    address (padded)
//	[64:96]  token1    address (padded)
//	[96:128] config    bytes32  — extension[0:20] | fee[20:28] | typeConfig[28:32]
//	[128:160] tick     int32 (padded)
//	[160:192] sqrtRatio uint96 (padded)
func (e *EventParser) DecodePoolCreated(log types.Log) (*entity.Pool, error) {
	if len(log.Data) < 192 {
		return nil, fmt.Errorf("invalid data length for PoolInitialized event: %d", len(log.Data))
	}

	token0 := common.BytesToAddress(log.Data[32:64])
	token1 := common.BytesToAddress(log.Data[64:96])

	var configBytes [32]byte
	copy(configBytes[:], log.Data[96:128])

	extension, fee, typeConfig := decodePoolConfig(configBytes)

	poolKey := pools.AnyPoolKey{
		PoolKey: pools.NewPoolKey(token0, token1, pools.NewPoolConfig(extension, fee, typeConfig)),
	}

	fetched, err := e.dataFetchers.fetchPools(context.Background(), []pools.AnyPoolKey{poolKey}, nil)
	if err != nil {
		return nil, err
	}
	if len(fetched) == 0 {
		return nil, fmt.Errorf("failed to fetch state for new pool")
	}
	pool := fetched[0]

	poolAddress, err := poolKey.ToPoolAddress()
	if err != nil {
		return nil, err
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{
		Core:          e.config.Core,
		ExtensionType: e.config.ExtensionType(extension),
		PoolKey:       poolKey,
	})
	if err != nil {
		return nil, err
	}

	extraBytes, err := json.Marshal(pool.GetState())
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:   poolAddress,
		Exchange:  string(e.config.DexId),
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  []string{"0", "0"},
		Tokens: []*entity.PoolToken{
			{
				Address:   valueobject.ZeroToWrappedLower(hexutil.Encode(token0[:]), e.config.ChainId),
				Swappable: true,
			},
			{
				Address:   valueobject.ZeroToWrappedLower(hexutil.Encode(token1[:]), e.config.ChainId),
				Swappable: true,
			},
		},
		StaticExtra: string(staticExtraBytes),
		Extra:       string(extraBytes),
		BlockNumber: pool.blockNumber,
	}, nil
}

// decodePoolConfig reverses PoolConfig.Compressed() from keys.go:205.
//
// Layout: extension[0:20] | fee[20:28] | typeConfig[28:32]
//
// typeConfig 4-byte discriminant:
//   - byte[0] & 0x80 != 0  → Concentrated; tickSpacing = Uint32(bytes) &^ 0x80000000
//   - all zeros             → FullRange
//   - otherwise             → Stableswap; byte[0] = ampFactor, bytes[1:4] = lower 24 bits
//     of CenterTick/16 (sign-extended)
func decodePoolConfig(config [32]byte) (extension common.Address, fee uint64, typeConfig pools.PoolTypeConfig) {
	extension = common.BytesToAddress(config[:20])
	fee = binary.BigEndian.Uint64(config[20:28])
	tb := config[28:32]

	switch {
	case tb[0]&0x80 != 0:
		tickSpacing := binary.BigEndian.Uint32(tb) &^ uint32(0x80000000)
		typeConfig = pools.NewConcentratedPoolTypeConfig(tickSpacing)
	case tb[0] == 0 && tb[1] == 0 && tb[2] == 0 && tb[3] == 0:
		typeConfig = pools.NewFullRangePoolTypeConfig()
	default:
		ampFactor := tb[0]
		raw24 := uint32(tb[1])<<16 | uint32(tb[2])<<8 | uint32(tb[3])
		var centerTickDiv16 int32
		if raw24&0x800000 != 0 {
			centerTickDiv16 = int32(raw24 | 0xFF000000)
		} else {
			centerTickDiv16 = int32(raw24)
		}
		typeConfig = pools.NewStableswapPoolTypeConfig(centerTickDiv16*16, ampFactor)
	}
	return
}
