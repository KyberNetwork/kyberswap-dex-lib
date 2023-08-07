package traderjoev20

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

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
	var metadata Metadata
	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	// Add timestamp to the context so that each run iteration will have something different
	ctx = util.NewContextWithTimestamp(ctx)

	var lengthBI *big.Int

	getNumPoolsRequest := d.ethrpcClient.NewRequest()
	getNumPoolsRequest.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: d.config.FactoryAddress,
		Method: factoryNumberOfPairsMethod,
	}, []interface{}{&lengthBI})

	if _, err := getNumPoolsRequest.Call(); err != nil {
		logger.Errorf("failed to get number of pairs from factory, err: %v", err)
		return nil, metadataBytes, err
	}

	fmt.Printf("lengthBI = %s\n", lengthBI)

	totalNumberOfPools := int(lengthBI.Int64())

	currentOffset := metadata.Offset
	batchSize := d.config.NewPoolLimit
	if currentOffset+batchSize > totalNumberOfPools {
		batchSize = totalNumberOfPools - currentOffset
		if batchSize <= 0 {
			return nil, metadataBytes, nil
		}
	}

	getPairAddressRequest := d.ethrpcClient.NewRequest()

	var pairAddresses = make([]common.Address, batchSize)
	for j := 0; j < batchSize; j++ {
		getPairAddressRequest.AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: d.config.FactoryAddress,
			Method: factoryGetPairMethod,
			Params: []interface{}{big.NewInt(int64(currentOffset + j))},
		}, []interface{}{&pairAddresses[j]})
	}
	resp, err := getPairAddressRequest.TryAggregate()
	if err != nil {
		logger.Errorf("failed to process aggregate, err: %v", err)
		return nil, metadataBytes, err
	}

	var successPairAddresses []common.Address
	for i, isSuccess := range resp.Result {
		if isSuccess {
			successPairAddresses = append(successPairAddresses, pairAddresses[i])
		}
	}

	pools, err := d.processBatch(ctx, successPairAddresses)
	if err != nil {
		logger.Errorf("failed to process update new pool, err: %v", err)
		return nil, metadataBytes, err
	}

	nextOffset := currentOffset + batchSize
	newMetadataBytes, err := json.Marshal(Metadata{
		Offset: nextOffset,
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	if len(pools) > 0 {
		logger.Infof("scan TraderJoe v2.0 LBFactory with batch size %v, progress: %d/%d", batchSize, currentOffset+batchSize, totalNumberOfPools)
	}

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) processBatch(ctx context.Context, pairAddresses []common.Address) ([]entity.Pool, error) {
	var tokenXAddresses = make([]common.Address, len(pairAddresses))
	var tokenYAddresses = make([]common.Address, len(pairAddresses))

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	for i := 0; i < len(pairAddresses); i++ {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddresses[i].Hex(),
			Method: pairTokenXMethod,
		}, []interface{}{&tokenXAddresses[i]})

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddresses[i].Hex(),
			Method: pairTokenYMethod,
		}, []interface{}{&tokenYAddresses[i]})
	}

	if _, err := rpcRequest.Aggregate(); err != nil {
		logger.Errorf("failed to process aggregate to get 2 tokens from pair contract, err: %v", err)
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(pairAddresses))

	for i, pairAddress := range pairAddresses {
		p := strings.ToLower(pairAddress.Hex())
		tokenXAddress := strings.ToLower(tokenXAddresses[i].Hex())
		tokenYAddress := strings.ToLower(tokenYAddresses[i].Hex())

		var tokenX = entity.PoolToken{
			Address:   tokenXAddress,
			Weight:    defaultTokenWeight,
			Swappable: true,
		}
		var tokenY = entity.PoolToken{
			Address:   tokenYAddress,
			Weight:    defaultTokenWeight,
			Swappable: true,
		}

		var newPool = entity.Pool{
			Address:   p,
			Exchange:  d.config.DexID,
			Type:      DexTypeTraderJoeV20,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens:    []*entity.PoolToken{&tokenX, &tokenY},
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}
