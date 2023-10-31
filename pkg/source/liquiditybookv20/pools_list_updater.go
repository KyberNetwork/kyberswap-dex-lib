package liquiditybookv20

import (
	"context"
	"encoding/json"
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

func (p *PoolsListUpdater) InitPool(_ context.Context) error {
	return nil
}

func (p *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
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

	getNumPoolsRequest := p.ethrpcClient.NewRequest()
	getNumPoolsRequest.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: p.config.FactoryAddress,
		Method: factoryMethodGetNumberOfLBPairs,
	}, []interface{}{&lengthBI})

	if _, err := getNumPoolsRequest.Call(); err != nil {
		logger.Errorf("failed to get number of pairs from factory, err: %v", err)
		return nil, metadataBytes, err
	}

	totalNumberOfPools := int(lengthBI.Int64())

	currentOffset := metadata.Offset
	batchSize := p.config.NewPoolLimit
	if currentOffset+batchSize > totalNumberOfPools {
		batchSize = totalNumberOfPools - currentOffset
		if batchSize <= 0 {
			return nil, metadataBytes, nil
		}
	}

	getPairAddressRequest := p.ethrpcClient.NewRequest()

	var pairAddresses = make([]common.Address, batchSize)
	for j := 0; j < batchSize; j++ {
		getPairAddressRequest.AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: p.config.FactoryAddress,
			Method: factoryMethodAllLBPairs,
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

	pools, err := p.processBatch(ctx, successPairAddresses)
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
		logger.Infof("scan Liquidity Book V2.1 LBFactory with batch size %v, progress: %d/%d", batchSize, currentOffset+batchSize, totalNumberOfPools)
	}

	return pools, newMetadataBytes, nil
}

func (p *PoolsListUpdater) processBatch(ctx context.Context, pairAddresses []common.Address) ([]entity.Pool, error) {
	var tokenXAddresses = make([]common.Address, len(pairAddresses))
	var tokenYAddresses = make([]common.Address, len(pairAddresses))

	rpcRequest := p.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	for i := 0; i < len(pairAddresses); i++ {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddresses[i].Hex(),
			Method: pairMethodTokenX,
		}, []interface{}{&tokenXAddresses[i]})

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddresses[i].Hex(),
			Method: pairMethodTokenY,
		}, []interface{}{&tokenYAddresses[i]})
	}

	if _, err := rpcRequest.Aggregate(); err != nil {
		logger.Errorf("failed to process aggregate to get 2 tokens from pair contract, err: %v", err)
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(pairAddresses))

	for i, pairAddress := range pairAddresses {
		address := strings.ToLower(pairAddress.Hex())
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
			Address:   address,
			Exchange:  p.config.DexID,
			Type:      DexTypeLiquidityBookV20,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens:    []*entity.PoolToken{&tokenX, &tokenY},
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}
