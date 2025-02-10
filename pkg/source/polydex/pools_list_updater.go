package polydex

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexTypePolydex, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	log := logger.WithFields(logger.Fields{
		"liquiditySource": DexTypePolydex,
		"kind":            "getNewPools",
	})
	var metadata Metadata
	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			log.Errorf("error when unmarshal metadata: %s", err)
			return nil, metadataBytes, err
		}
	}

	// Add timestamp to the context so that each run iteration will have something different
	ctx = util.NewContextWithTimestamp(ctx)

	var lengthBI *big.Int

	getNumPoolsRequest := d.ethrpcClient.NewRequest()
	getNumPoolsRequest.AddCall(&ethrpc.Call{
		ABI:    polydexFactoryABI,
		Target: d.config.FactoryAddress,
		Method: factoryMethodAllPairsLength,
		Params: nil,
	}, []interface{}{&lengthBI})

	if _, err := getNumPoolsRequest.Call(); err != nil {
		log.Errorf("failed to get number of pairs from factory, err: %v", err)
		return nil, metadataBytes, err
	}

	totalNumberOfPools := int(lengthBI.Int64())

	currentOffset := metadata.Offset
	batchSize := d.config.NewPoolLimit
	if currentOffset+batchSize > totalNumberOfPools {
		batchSize = totalNumberOfPools - currentOffset
		if batchSize <= 0 {
			log.Info("no new pool. Ignore update pool")
			return nil, metadataBytes, nil
		}
	}

	getPairAddressRequest := d.ethrpcClient.NewRequest()

	var pairAddresses = make([]common.Address, batchSize)
	for j := 0; j < batchSize; j++ {
		getPairAddressRequest.AddCall(&ethrpc.Call{
			ABI:    polydexFactoryABI,
			Target: d.config.FactoryAddress,
			Method: factoryMethodGetPair,
			Params: []interface{}{big.NewInt(int64(currentOffset + j))},
		}, []interface{}{&pairAddresses[j]})
	}

	if _, err := getPairAddressRequest.Aggregate(); err != nil {
		log.Errorf("failed to process aggregate, err: %v", err)
		return nil, metadataBytes, err
	}

	pools, err := d.processBatch(ctx, pairAddresses)
	if err != nil {
		log.Errorf("failed to process update new pool, err: %v", err)
		return nil, metadataBytes, err
	}

	numPools := len(pools)

	nextOffset := currentOffset + numPools
	newMetadataBytes, err := json.Marshal(Metadata{
		Offset: nextOffset,
	})
	if err != nil {
		log.Errorf("error when marshal new metadata: %s", err)
		return nil, metadataBytes, err
	}

	if len(pools) > 0 {
		log.Infof("scan with batch size %v, progress: %d/%d", batchSize, currentOffset+numPools, totalNumberOfPools)
	}

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) processBatch(ctx context.Context, pairAddresses []common.Address) ([]entity.Pool, error) {
	var limit = len(pairAddresses)
	var token0Addresses = make([]common.Address, limit)
	var token1Addresses = make([]common.Address, limit)
	var swapFees = make([]uint32, limit)

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	for i := 0; i < limit; i++ {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddresses[i].Hex(),
			Method: pairMethodToken0,
			Params: nil,
		}, []interface{}{&token0Addresses[i]})

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddresses[i].Hex(),
			Method: pairMethodToken1,
			Params: nil,
		}, []interface{}{&token1Addresses[i]})

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddresses[i].Hex(),
			Method: pairMethodGetSwapFee,
			Params: nil,
		}, []interface{}{&swapFees[i]})
	}

	if _, err := rpcRequest.Aggregate(); err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(pairAddresses))

	for i, pairAddress := range pairAddresses {
		p := strings.ToLower(pairAddress.Hex())
		token0Address := strings.ToLower(token0Addresses[i].Hex())
		token1Address := strings.ToLower(token1Addresses[i].Hex())

		var token0 = entity.PoolToken{
			Address:   token0Address,
			Weight:    defaultTokenWeight,
			Swappable: true,
		}
		var token1 = entity.PoolToken{
			Address:   token1Address,
			Weight:    defaultTokenWeight,
			Swappable: true,
		}

		var newPool = entity.Pool{
			Address:   p,
			SwapFee:   float64(swapFees[i]) / bps,
			Exchange:  d.config.DexID,
			Type:      DexTypePolydex,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{reserveZero, reserveZero},
			Tokens:    []*entity.PoolToken{&token0, &token1},
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}
