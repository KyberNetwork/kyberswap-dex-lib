package zkswapfinance

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
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
		Method: factoryMethodAllPairsLength,
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
			Method: factoryMethodGetPair,
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
		logger.Infof("scan ZFFactory with batch size %v, progress: %d/%d", batchSize, currentOffset+batchSize, totalNumberOfPools)
	}

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) processBatch(ctx context.Context, pairAddresses []common.Address) ([]entity.Pool, error) {
	var (
		limit           = len(pairAddresses)
		token0Addresses = make([]common.Address, limit)
		token1Addresses = make([]common.Address, limit)
	)

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
	}

	if _, err := rpcRequest.Aggregate(); err != nil {
		logger.Errorf("failed to process aggregate to get 2 tokens from pair contract, err: %v", err)
		return nil, err
	}

	staticExtra := StaticExtra{
		FeePrecision: feePrecision.Uint64(),
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		logger.Errorf("failed to marshal static extra, err: %v", err)
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
			Address:      p,
			ReserveUsd:   0,
			AmplifiedTvl: 0,
			SwapFee:      0,
			Exchange:     d.config.DexID,
			Type:         DexTypeZkSwapFinance,
			Timestamp:    time.Now().Unix(),
			Reserves:     []string{reserveZero, reserveZero},
			Tokens:       []*entity.PoolToken{&token0, &token1},
			Extra:        "",
			StaticExtra:  string(staticExtraBytes),
			TotalSupply:  "",
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}
