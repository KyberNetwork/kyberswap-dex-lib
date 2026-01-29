package someswapv2

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var (
		dexID     = DexType
		startTime = time.Now()
	)

	logger.WithFields(logger.Fields{"exchange": dexID}).Info("Started getting new pools")

	allPairsLength, err := u.getAllPairsLength(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID}).Error("getAllPairsLength failed")
		return nil, metadataBytes, err
	}

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Warn("getOffset failed")
	}

	if offset >= allPairsLength {
		return nil, metadataBytes, nil
	}

	poolsFromEvents, err := u.getPoolsFromEvents(ctx, offset, u.config.NewPoolLimit)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Error("getPoolsFromEvents failed")
		return nil, metadataBytes, err
	}

	if len(poolsFromEvents) == 0 {
		return nil, metadataBytes, nil
	}

	pools, err := u.initPools(ctx, poolsFromEvents)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Error("initPools failed")
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(offset + len(pools))
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Error("newMetadata failed")
		return nil, metadataBytes, err
	}

	logger.WithFields(
		logger.Fields{
			"dex_id":      dexID,
			"valid_pools": len(pools),
			"offset":      offset,
			"duration_ms": time.Since(startTime).Milliseconds(),
		},
	).Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getAllPairsLength(ctx context.Context) (int, error) {
	var allPairsLength *big.Int
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.Factory,
		Method: factoryMethodAllPairsLength,
	}, []any{&allPairsLength})

	if _, err := req.Call(); err != nil {
		return 0, err
	}
	if allPairsLength == nil {
		return 0, nil
	}
	return int(allPairsLength.Int64()), nil
}

func (u *PoolsListUpdater) getOffset(metadataBytes []byte) (int, error) {
	if len(metadataBytes) == 0 {
		return 0, nil
	}
	var metadata Metadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return 0, err
	}
	return metadata.Offset, nil
}

type poolFromEvent struct {
	Address common.Address
	Token0  common.Address
	Token1  common.Address
	BaseFee uint32
	WToken0 uint32
	WToken1 uint32
}

func (u *PoolsListUpdater) getPoolsFromEvents(ctx context.Context, offset, limit int) ([]poolFromEvent, error) {
	pairCreatedEvent := factoryABI.Events[factoryEventPairCreated]
	pairCreatedTopic := pairCreatedEvent.ID

	client := u.ethrpcClient.GetETHClient()
	currentBlock, err := client.BlockNumber(ctx)
	if err != nil {
		return nil, err
	}

	const blockStep = uint64(500)
	fromBlock := uint64(0)
	if currentBlock > 100000 {
		fromBlock = currentBlock - 100000
	}

	var allPools []poolFromEvent

	for start := fromBlock; start < currentBlock && len(allPools) < offset+limit; start += blockStep {
		end := start + blockStep - 1
		if end > currentBlock {
			end = currentBlock
		}

		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(start)),
			ToBlock:   big.NewInt(int64(end)),
			Addresses: []common.Address{common.HexToAddress(u.config.Factory)},
			Topics:    [][]common.Hash{{pairCreatedTopic}},
		}

		logs, err := client.FilterLogs(ctx, query)
		if err != nil {
			continue
		}

		for _, log := range logs {
			if len(log.Data) >= 192 {
				poolAddr := common.BytesToAddress(log.Data[:32])
				baseFee := uint32(log.Data[124])<<24 | uint32(log.Data[125])<<16 | uint32(log.Data[126])<<8 | uint32(log.Data[127])
				wToken0 := uint32(log.Data[156])<<24 | uint32(log.Data[157])<<16 | uint32(log.Data[158])<<8 | uint32(log.Data[159])
				wToken1 := uint32(log.Data[188])<<24 | uint32(log.Data[189])<<16 | uint32(log.Data[190])<<8 | uint32(log.Data[191])

				var token0, token1 common.Address
				if len(log.Topics) >= 3 {
					token0 = common.BytesToAddress(log.Topics[1].Bytes())
					token1 = common.BytesToAddress(log.Topics[2].Bytes())
				}

				allPools = append(allPools, poolFromEvent{
					Address: poolAddr,
					Token0:  token0,
					Token1:  token1,
					BaseFee: baseFee,
					WToken0: wToken0,
					WToken1: wToken1,
				})
			}
		}
	}

	if offset >= len(allPools) {
		return nil, nil
	}
	end := offset + limit
	if end > len(allPools) {
		end = len(allPools)
	}

	return allPools[offset:end], nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, poolsFromEvents []poolFromEvent) ([]entity.Pool, error) {
	pools := make([]entity.Pool, 0, len(poolsFromEvents))

	for _, pfe := range poolsFromEvents {
		token0 := &entity.PoolToken{
			Address:   hexutil.Encode(pfe.Token0[:]),
			Swappable: true,
		}
		token1 := &entity.PoolToken{
			Address:   hexutil.Encode(pfe.Token1[:]),
			Swappable: true,
		}

		staticExtraBytes, err := json.Marshal(StaticExtra{
			BaseFee: pfe.BaseFee,
			WToken0: pfe.WToken0,
			WToken1: pfe.WToken1,
		})
		if err != nil {
			return nil, err
		}

		swapFee := float64(pfe.BaseFee) / float64(feeDen.Int64())

		newPool := entity.Pool{
			Address:      hexutil.Encode(pfe.Address[:]),
			ReserveUsd:   0,
			AmplifiedTvl: 0,
			SwapFee:      swapFee,
			Exchange:     string(valueobject.ExchangeSomeSwapV2),
			Type:         DexType,
			Timestamp:    time.Now().Unix(),
			Reserves:     []string{reserveZero, reserveZero},
			Tokens:       []*entity.PoolToken{token0, token1},
			StaticExtra:  string(staticExtraBytes),
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}

func (u *PoolsListUpdater) newMetadata(offset int) ([]byte, error) {
	return json.Marshal(Metadata{Offset: offset})
}
