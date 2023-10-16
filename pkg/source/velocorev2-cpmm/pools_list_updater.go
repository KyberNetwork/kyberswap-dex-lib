package velocorev2cpmm

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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

func (d *PoolsListUpdater) InitPool(_ context.Context) error {
	return nil
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
		Method: factoryMethodPoolsLength,
		Params: nil,
	}, []interface{}{&lengthBI})
	if _, err := getNumPoolsRequest.Call(); err != nil {
		logger.Errorf("failed to get number of pairs from factory, err: %v", err)
		return nil, metadataBytes, err
	}
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
			Method: factoryMethodPoolList,
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
	newMetadataBytes, err := json.Marshal(Metadata{Offset: nextOffset})
	if err != nil {
		return nil, metadataBytes, err
	}

	if len(pools) > 0 {
		logger.Infof("scan VelocoreV2CPMM with batch size %v, progress: %d/%d", batchSize, currentOffset+batchSize, totalNumberOfPools)
	}

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) processBatch(ctx context.Context, poolAddresses []common.Address) ([]entity.Pool, error) {
	var (
		limit = len(poolAddresses)
		pools = make([]entity.Pool, 0, len(poolAddresses))

		tokens  = make([][maxPoolTokenNumber]bytes32, limit)
		weights = make([][maxPoolTokenNumber]*big.Int, limit)
	)

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)
	for i := 0; i < limit; i++ {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddresses[i].Hex(),
			Method: poolMethodRelevantTokens,
			Params: nil,
		}, []interface{}{&tokens[i]})

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddresses[i].Hex(),
			Method: poolMethodTokenWeights,
			Params: nil,
		}, []interface{}{&weights[i]})
	}
	if _, err := rpcRequest.Aggregate(); err != nil {
		logger.Errorf("failed to process aggregate to get tokens and weights from pool contract, err: %v", err)
		return nil, err
	}

	for i, poolAddress := range poolAddresses {
		p := strings.ToLower(poolAddress.Hex())
		poolTokens := []*entity.PoolToken{}
		reserves := []string{}

		for j := 0; j < maxPoolTokenNumber; j++ {
			t := tokens[i][j].unwrapToken()
			w := weights[i][j]
			if t == valueobject.ZeroAddress {
				break
			}
			poolTokens = append(poolTokens, &entity.PoolToken{
				Address:   t,
				Weight:    uint(w.Uint64()), // WARN: weight is uint64 in smart contract, but uint in entity
				Swappable: true,
			})
			reserves = append(reserves, reserveZero)
		}

		staticExtra := StaticExtra{
			PoolTokenNumber: uint(len(poolTokens)),
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.Errorf("failed to marshal static extra, err: %v", err)
			return nil, err
		}

		newPool := entity.Pool{
			Address:      p,
			ReserveUsd:   0,
			AmplifiedTvl: 0,
			Exchange:     d.config.DexID,
			Type:         DexTypeVelocoreV2CPMM,
			Timestamp:    time.Now().Unix(),
			Reserves:     reserves,
			Tokens:       poolTokens,
			StaticExtra:  string(staticExtraBytes),
		}
		pools = append(pools, newPool)
	}

	return pools, nil
}
