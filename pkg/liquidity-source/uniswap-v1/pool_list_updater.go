package uniswapv1

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}

	PoolsListUpdaterMetadata struct {
		Offset int `json:"offset"`
	}

	ExchangeInfo struct {
		ExchangeAddress common.Address
		TokenAddress    common.Address
	}
)

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
	var startTime = time.Now()

	logger.WithFields(logger.Fields{"dex_id": DexType}).Info("Started getting new pools")

	ctx = util.NewContextWithTimestamp(ctx)

	totalExchanges, err := u.getTotalExchanges(ctx)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": DexType}).
			Error("getTotalExchanges failed")

		return nil, metadataBytes, err
	}

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": DexType, "err": err}).
			Warn("getOffset failed")
	}

	batchSize := getBatchSize(totalExchanges, u.config.NewPoolLimit, offset)

	exchanges, err := u.listExchanges(ctx, offset, batchSize)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": DexType, "err": err}).
			Error("listExchangeAddresses failed")

		return nil, metadataBytes, err
	}

	pools, err := u.initPools(exchanges)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": DexType, "err": err}).
			Error("initPools failed")

		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(offset + batchSize)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": DexType, "err": err}).
			Error("newMetadata failed")

		return nil, metadataBytes, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      DexType,
				"pools_len":   len(pools),
				"offset":      offset,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getTotalExchanges(ctx context.Context) (int, error) {
	var totalExchanges *big.Int

	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    uniswapFactoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryTokenCountMethod,
		Params: nil,
	}, []interface{}{&totalExchanges})

	if _, err := req.Call(); err != nil {
		return 0, err
	}

	return int(totalExchanges.Int64()), nil
}

func (u *PoolsListUpdater) getOffset(metadataBytes []byte) (int, error) {
	if len(metadataBytes) == 0 {
		return 0, nil
	}

	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return 0, err
	}

	return metadata.Offset, nil
}

func (u *PoolsListUpdater) listExchanges(ctx context.Context, offset int, batchSize int) ([]ExchangeInfo, error) {
	listTokenResult := make([]common.Address, batchSize)

	getTokensRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i := 0; i < batchSize; i++ {
		index := big.NewInt(int64(offset + i))

		getTokensRequest.AddCall(&ethrpc.Call{
			ABI:    uniswapFactoryABI,
			Target: u.config.FactoryAddress,
			Method: factoryGetTokenWithIDMethod,
			Params: []interface{}{index},
		}, []interface{}{&listTokenResult[i]})
	}

	_, err := getTokensRequest.TryAggregate()
	if err != nil {
		return nil, err
	}

	getExchangesRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

	listExchangeResult := make([]common.Address, len(listTokenResult))
	for i, tokenAddress := range listTokenResult {
		getExchangesRequest.AddCall(&ethrpc.Call{
			ABI:    uniswapFactoryABI,
			Target: u.config.FactoryAddress,
			Method: factoryGetExchangeMethod,
			Params: []interface{}{tokenAddress},
		}, []interface{}{&listExchangeResult[i]})
	}

	resp, err := getExchangesRequest.TryAggregate()
	if err != nil {
		return nil, err
	}

	var exchanges = make([]ExchangeInfo, 0, len(listExchangeResult))
	for i, isSuccess := range resp.Result {
		if !isSuccess || listExchangeResult[i] == ZERO_ADDRESS {
			continue
		}

		exchanges = append(exchanges, ExchangeInfo{
			ExchangeAddress: listExchangeResult[i],
			TokenAddress:    listTokenResult[i],
		})
	}

	return exchanges, nil
}

func (u *PoolsListUpdater) initPools(exchanges []ExchangeInfo) ([]entity.Pool, error) {
	pools := make([]entity.Pool, 0, len(exchanges))

	for _, exchange := range exchanges {
		token0 := &entity.PoolToken{
			Address:   strings.ToLower(valueobject.WETHByChainID[u.config.ChainID]),
			Swappable: true,
		}

		token1 := &entity.PoolToken{
			Address:   strings.ToLower(exchange.TokenAddress.Hex()),
			Swappable: true,
		}

		var newPool = entity.Pool{
			Address:   strings.ToLower(exchange.ExchangeAddress.Hex()),
			Exchange:  string(valueobject.ExchangeUniSwapV1),
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens:    []*entity.PoolToken{token0, token1},
			SwapFee:   DefaultSwapFee,
		}

		pools = append(pools, newPool)
	}

	return pools, nil
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

func getBatchSize(length int, limit int, offset int) int {
	if offset == length {
		return 0
	}

	if offset+limit >= length {
		return length - offset
	}

	return limit
}
