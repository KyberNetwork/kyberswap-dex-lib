package traderjoecommon

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

type PoolsListUpdater struct {
	Config       *Config
	EthrpcClient *ethrpc.Client

	FactoryABI                 abi.ABI
	FactoryNumberOfPairsMethod string
	FactoryGetPairMethod       string

	PairABI          abi.ABI
	PairTokenXMethod string
	PairTokenYMethod string

	DexType            string
	DefaultTokenWeight uint
}

// GetNewPools gets TraderJoe pools with offset and limit.
// TraderJoe v2.0 and v2.1 stores list of pools in the factory contract by a append-only list.
// The offset is store in Metadata struct and advanced after each call of GetNewPools.
// The limit is configured in Config field.
// Functions to get the pools list's length and get each pool address are documented in
//
// * TraderJoe v2.0: https://docs.traderjoexyz.com/V2/contracts/LBFactory
//
// * TraderJoe v2.1: https://docs.traderjoexyz.com/contracts/LBFactory
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

	getNumPoolsRequest := d.EthrpcClient.NewRequest()
	getNumPoolsRequest.AddCall(&ethrpc.Call{
		ABI:    d.FactoryABI,
		Target: d.Config.FactoryAddress,
		Method: d.FactoryNumberOfPairsMethod,
	}, []interface{}{&lengthBI})

	if _, err := getNumPoolsRequest.Call(); err != nil {
		logger.Errorf("failed to get number of pairs from factory, err: %v", err)
		return nil, metadataBytes, err
	}

	totalNumberOfPools := int(lengthBI.Int64())

	currentOffset := metadata.Offset
	batchSize := d.Config.NewPoolLimit
	if currentOffset+batchSize > totalNumberOfPools {
		batchSize = totalNumberOfPools - currentOffset
		if batchSize <= 0 {
			return nil, metadataBytes, nil
		}
	}

	getPairAddressRequest := d.EthrpcClient.NewRequest()

	var pairAddresses = make([]common.Address, batchSize)
	for j := 0; j < batchSize; j++ {
		getPairAddressRequest.AddCall(&ethrpc.Call{
			ABI:    d.FactoryABI,
			Target: d.Config.FactoryAddress,
			Method: d.FactoryGetPairMethod,
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
		logger.Infof("scan TraderJoe LBFactory with batch size %v, progress: %d/%d", batchSize, currentOffset+batchSize, totalNumberOfPools)
	}

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) processBatch(ctx context.Context, pairAddresses []common.Address) ([]entity.Pool, error) {
	var tokenXAddresses = make([]common.Address, len(pairAddresses))
	var tokenYAddresses = make([]common.Address, len(pairAddresses))

	rpcRequest := d.EthrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	for i := 0; i < len(pairAddresses); i++ {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    d.PairABI,
			Target: pairAddresses[i].Hex(),
			Method: d.PairTokenXMethod,
		}, []interface{}{&tokenXAddresses[i]})

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    d.PairABI,
			Target: pairAddresses[i].Hex(),
			Method: d.PairTokenYMethod,
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
			Weight:    d.DefaultTokenWeight,
			Swappable: true,
		}
		var tokenY = entity.PoolToken{
			Address:   tokenYAddress,
			Weight:    d.DefaultTokenWeight,
			Swappable: true,
		}

		var newPool = entity.Pool{
			Address:   p,
			Exchange:  d.Config.DexID,
			Type:      d.DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens:    []*entity.PoolToken{&tokenX, &tokenY},
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}
