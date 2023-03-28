package service

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	cmap "github.com/orcaman/concurrent-map"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type ScanEventService struct {
	cfg              *config.Common
	mintOrRemoveMap  cmap.ConcurrentMap
	scan             *ScanService
	rpcRepo          repository.IRPCRepository
	scannerStateRepo repository.IScannerStateRepository
}

func NewScanEventService(
	cfg *config.Common,
	scan *ScanService,
	rpcRepo repository.IRPCRepository,
	scannerStateRepo repository.IScannerStateRepository,
) *ScanEventService {
	return &ScanEventService{
		cfg:              cfg,
		mintOrRemoveMap:  cmap.New(),
		scan:             scan,
		rpcRepo:          rpcRepo,
		scannerStateRepo: scannerStateRepo,
	}
}

type eventInfo struct {
	TimeStamp int64
	Reserves  []string
	Log       types.Log
}

func (s *ScanEventService) UpdateData(ctx context.Context) {
	syncTopic := "0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1"
	mintTopic := "0x4c209b5fc8ad50758f13e2e1088ba56a560dff690a1c6fef26394f4c03821c4f"
	burnTopic := "0xdccd412f0b1252819cb1fd330b93224ca42612892bb3f4f789976e6d81936496"

	scanLog := func() error {
		start := time.Now()

		ethCli, err := eth.GetEthClient(ctx, s.cfg.RPCs)
		if err != nil {
			logger.Errorf("failed to get eth client, err: %v", err)
			return err
		}

		maxBlock, err := ethCli.BlockNumber(ctx)
		if err != nil {
			logger.Errorf("failed to get current block")
			return err
		}

		startBlock, err := s.scannerStateRepo.GetScanBlock(ctx)
		if err != nil {
			logger.Errorf("failed to get scan block from database, err: %v", err)
			return err
		}

		if maxBlock-startBlock > 5 {
			startBlock = maxBlock - 5
		}
		eventMap := make(map[string]eventInfo)
		changeLiquidityCount := 0
		for blockNumber := startBlock + 1; blockNumber <= maxBlock; blockNumber++ {
			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(int64(blockNumber - 1)),
				ToBlock:   big.NewInt(int64(blockNumber)),
				Topics: [][]common.Hash{{
					common.HexToHash(syncTopic),
					common.HexToHash(mintTopic),
					common.HexToHash(burnTopic),
				}},
			}

			block, err := ethCli.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
			if err != nil {
				logger.Errorf("failed to get block %d, err: %v", blockNumber, err)
				return err
			}

			logs, err := ethCli.FilterLogs(ctx, query)
			if err != nil {
				logger.Errorf("failed to get logs, err: %v", err)
				return err
			}

			for i := range logs {
				topic := logs[i].Topics[0].Hex()
				pairAddress := strings.ToLower(logs[i].Address.Hex())

				if topic == syncTopic {

					var reserve struct {
						Reserve0 *big.Int
						Reserve1 *big.Int
					}
					if err := abis.SushiswapPair.UnpackIntoInterface(&reserve, "Sync", logs[i].Data); err != nil {
						logger.Debugf("failed to unpack log data, err: %v", err)
						return err
					}
					logger.Debugf(">>>> logs[%d]: %v, %v, %+v", i, logs[i].BlockNumber, pairAddress, reserve)
					if o, ok := eventMap[pairAddress]; ok {
						if o.Log.BlockNumber > logs[i].BlockNumber {
							continue
						}
						if o.Log.TxIndex > logs[i].TxIndex {
							continue
						}
						if o.Log.Index > logs[i].Index {
							continue
						}
					}
					eventMap[pairAddress] = eventInfo{
						int64(block.Time()),
						[]string{
							reserve.Reserve0.String(),
							reserve.Reserve1.String(),
						},
						logs[i],
					}
				} else {
					if s.scan.ExistPool(ctx, pairAddress) {
						s.mintOrRemoveMap.Set(pairAddress, int64(block.Time()))
						changeLiquidityCount++
					}
				}

			}
		}
		for address, info := range eventMap {
			if s.scan.ExistPool(ctx, address) {
				pool, _ := s.scan.GetPoolByAddress(ctx, address)
				if pool.Type != constant.PoolTypes.Dmm {
					err := s.scan.UpdatePoolReserve(ctx, address, info.TimeStamp, info.Reserves)
					if err != nil {
						logger.Errorf("failed to save pool to database, err: %v", err)
					}
				}
			}
		}

		if err = s.scannerStateRepo.SetScanBlock(ctx, maxBlock); err != nil {
			logger.Errorf("failed to save scan block to database, err: %v", err)
			return err
		}

		metrics.GaugeAggregatorScanLatestBlock(maxBlock)

		logger.Infof("Update reserved=%d supply=%d pools from block %d->%d in %v", len(eventMap), changeLiquidityCount, startBlock, maxBlock, time.Since(start))

		return nil
	}

	for {
		err := scanLog()
		if err != nil {
			logger.Errorf("failed to update status rpc")
		}
		time.Sleep(time.Second)
	}
}
