package tessera

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

const (
	batchSize = 50
)

type (
	PoolsListUpdater struct {
		cfg          *Config
		ethrpcClient *ethrpc.Client
	}

	PoolsListUpdaterMetadata struct {
		Offset int `json:"offset"`
	}
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	log := logger.WithFields(logger.Fields{
		"dexId": u.cfg.DexId,
	})

	log.Info("Start getting new pools.")

	metadata, err := u.getMetadata(metadataBytes)
	if err != nil {
		log.Warnf("getMetadata failed: %v", err)
	}

	var allPairs [][]common.Address
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    tesseraIndexerABI,
		Target: u.cfg.TesseraIndexer,
		Method: "getTesseraPairs",
	}, []any{&allPairs})

	if _, err := req.Call(); err != nil {
		log.Errorf("Indexer call failed: %v", err)
		return nil, metadataBytes, err
	}

	totalPairs := len(allPairs)
	if totalPairs == 0 {
		log.Info("No pairs found from Indexer")
		return nil, metadataBytes, nil
	}

	if metadata.Offset > totalPairs {
		log.Infof("Resetting offset to 0 (metadata.Offset %d > totalPairs %d)", metadata.Offset, totalPairs)
		metadata.Offset = 0
	}

	if metadata.Offset == totalPairs {
		log.Info("No new pairs to track")
		return nil, metadataBytes, nil
	}

	numToProcess := totalPairs - metadata.Offset
	if numToProcess > batchSize {
		numToProcess = batchSize
	}

	pairsToProcess := allPairs[metadata.Offset : metadata.Offset+numToProcess]

	type getPoolResp struct {
		Exists bool           `abi:"exists"`
		Pool   common.Address `abi:"pool"`
	}

	var pools []entity.Pool
	engineReq := u.ethrpcClient.NewRequest().SetContext(ctx)
	resps := make([]getPoolResp, len(pairsToProcess))

	for i, pair := range pairsToProcess {
		engineReq.AddCall(&ethrpc.Call{
			ABI:    tesseraEngineABI,
			Target: u.cfg.TesseraEngine,
			Method: "getTesseraPool",
			Params: []any{pair[0], pair[1]},
		}, []any{&resps[i]})
	}

	resp, err := engineReq.TryAggregate()
	if err != nil {
		log.Errorf("Engine call failed: %v", err)
		return nil, metadataBytes, err
	}

	for i, r := range resps {
		if !resp.Result[i] || !r.Exists || r.Pool == (common.Address{}) {
			log.Debugf("Skipping pair %v: success=%v, exists=%v, addr=%s", pairsToProcess[i], resp.Result[i], r.Exists, r.Pool.Hex())
			continue
		}

		tokens := []*entity.PoolToken{
			{Address: strings.ToLower(pairsToProcess[i][0].Hex()), Swappable: true},
			{Address: strings.ToLower(pairsToProcess[i][1].Hex()), Swappable: true},
		}

		staticExtra, _ := json.Marshal(StaticExtra{
			TesseraSwap: u.cfg.TesseraSwap,
		})

		p := entity.Pool{
			Address:     strings.ToLower(r.Pool.Hex()),
			Exchange:    u.cfg.DexId,
			Type:        "tessera",
			Timestamp:   time.Now().Unix(),
			Reserves:    entity.PoolReserves{"0", "0"},
			Tokens:      tokens,
			StaticExtra: string(staticExtra),
		}
		pools = append(pools, p)
	}

	newOffset := metadata.Offset + numToProcess
	newMetadataBytes, err := u.newMetadata(newOffset)
	if err != nil {
		log.Errorf("newMetadata failed: %v", err)
		return pools, metadataBytes, nil
	}

	log.Infof("Found %d valid pools from batch of %d (new offset: %d/%d)", len(pools), numToProcess, newOffset, totalPairs)
	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getMetadata(metadataBytes []byte) (PoolsListUpdaterMetadata, error) {
	if len(metadataBytes) == 0 {
		return PoolsListUpdaterMetadata{}, nil
	}

	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return PoolsListUpdaterMetadata{}, err
	}

	return metadata, nil
}

func (u *PoolsListUpdater) newMetadata(newOffset int) ([]byte, error) {
	metadata := PoolsListUpdaterMetadata{
		Offset: newOffset,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	return metadataBytes, nil
}
