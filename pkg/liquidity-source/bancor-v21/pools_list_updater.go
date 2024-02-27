package bancor_v21

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

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}

	PoolsListUpdaterMetadata struct {
		Offset int `json:"offset"`
	}
)

// getAllPairsLength gets number of pairs from the factory contracts
func (u *PoolsListUpdater) getAllPairsLength(ctx context.Context) (int, error) {
	var allPairsLength *big.Int
	//
	getAllPairsLengthRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

	getAllPairsLengthRequest.AddCall(&ethrpc.Call{
		ABI:    converterRegistryABI,
		Target: u.config.ConverterRegistry,
		Method: getAnchorCount,
		Params: nil,
	}, []interface{}{&allPairsLength})

	if _, err := getAllPairsLengthRequest.Call(); err != nil {
		return 0, err
	}

	return int(allPairsLength.Int64()), nil
}

// getOffset gets index of the last pair that is fetched
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

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var (
		dexID     = u.config.DexID
		startTime = time.Now()
	)

	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new innerPools")
	ctx = util.NewContextWithTimestamp(ctx)

	allPairsLength, err := u.getAllPairsLength(ctx)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID}).
			Error("getAllPairsLength failed")

		return nil, metadataBytes, err
	}

	pairAddresses, anchors, err := u.listPairAddresses(ctx, allPairsLength)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("listPairAddresses failed")

		return nil, metadataBytes, err
	}
	anchorMap := make(map[string]struct{})
	for _, anchor := range anchors {
		anchorMap[anchor.Hex()] = struct{}{}
	}

	innerPools, err := u.initInnerPools(ctx, pairAddresses, anchors)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("initInnerPools failed")

		return nil, metadataBytes, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      dexID,
				"pools_len":   len(innerPools),
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new innerPools")

	onePool := entity.Pool{
		Address:      u.config.BancorNetworkAddress,
		ReserveUsd:   0,
		AmplifiedTvl: 0,
		SwapFee:      0,
		Exchange:     DexTypeBancorV21,
		Type:         DexTypeBancorV21,
		Timestamp:    time.Now().Unix(),
		Reserves:     nil,
		Tokens:       nil,
		Extra:        "",
		StaticExtra:  "",
		TotalSupply:  "",
		BlockNumber:  0,
	}
	return innerPools, metadataBytes, nil
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

// listPairTokens receives list of pair addresses and returns their tokens
func (u *PoolsListUpdater) listPairTokens(ctx context.Context, pairAddresses []common.Address) ([][]common.Address, error) {
	listTokensRequest := u.ethrpcClient.NewRequest().SetContext(ctx)
	tokens := make([][]common.Address, len(pairAddresses))

	for index, pairAddress := range pairAddresses {
		var numToken *big.Int
		if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
			ABI:    converterABI,
			Target: pairAddress.Hex(),
			Method: converterGetTokenCount,
			Params: nil,
		}, []interface{}{&numToken}).Call(); err != nil {
			return nil, err
		}
		nTokens := int(numToken.Int64())
		tokens[index] = make([]common.Address, nTokens)

		for i := 0; i < nTokens; i++ {
			listTokensRequest.AddCall(&ethrpc.Call{
				ABI:    converterABI,
				Target: pairAddress.Hex(),
				Method: converterGetTokens,
				Params: []interface{}{big.NewInt(int64(i))},
			}, []interface{}{&tokens[index][i]})
		}
	}

	if _, err := listTokensRequest.Aggregate(); err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": u.config.DexID}).
			Error("Get tokens list for pool failed")
		return nil, err
	}

	return tokens, nil
}
func (u *PoolsListUpdater) newExtraInner(anchorAddress string) ([]byte, error) {
	extra := ExtraInner{
		anchorAddress: anchorAddress,
	}

	return json.Marshal(extra)
}

func (u *PoolsListUpdater) buildConvertibleTokensAnchorState(ctx context.Context) (map[string][]string, error) {
	convertibleTokens := make([]common.Address, 0)
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    converterRegistryABI,
		Target: u.config.ConverterRegistry,
		Method: getConvertibleTokens,
		Params: nil,
	}, []interface{}{&convertibleTokens}).Call(); err != nil {
		return nil, err
	}

	anchorsByConvertibleTokens := make(map[string][]string)
	anchorsRequest := u.ethrpcClient.NewRequest().SetContext(ctx)
	anchors := make([][]common.Address, len(convertibleTokens))

	for i, convertibleToken := range convertibleTokens {
		anchors[i] = make([]common.Address, 0)
		anchorsRequest.AddCall(&ethrpc.Call{
			ABI:    converterRegistryABI,
			Target: u.config.ConverterRegistry,
			Method: getConvertibleTokenAnchors,
			Params: []interface{}{convertibleToken},
		}, []interface{}{&anchors[i]})
	}

	if _, err := anchorsRequest.Aggregate(); err != nil {
		return nil, err
	}

	for i, convertibleToken := range convertibleTokens {
		anchorsByConvertibleTokens[convertibleToken.Hex()] = make([]string, len(anchors[i]))
		for j, anchor := range anchors[i] {
			anchorsByConvertibleTokens[convertibleToken.Hex()][j] = anchor.Hex()
		}
	}

	return anchorsByConvertibleTokens, nil
}

// initInnerPools fetches token data and initializes pools
func (u *PoolsListUpdater) initInnerPools(ctx context.Context, pairAddresses, anchors []common.Address) ([]entity.Pool, error) {
	tokens, err := u.listPairTokens(ctx, pairAddresses)
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(pairAddresses))

	for i, pairAddress := range pairAddresses {
		entityTokens := make([]*entity.PoolToken, len(tokens[i]))
		for tokenIndex := 0; tokenIndex < len(tokens[i]); tokenIndex++ {
			entityTokens[tokenIndex] = &entity.PoolToken{
				Address:   strings.ToLower(tokens[i][tokenIndex].Hex()),
				Swappable: true,
			}
		}

		extra, err := u.newExtraInner(anchors[i].Hex())
		if err != nil {
			return nil, err
		}

		var newPool = entity.Pool{
			Address:   strings.ToLower(pairAddress.Hex()),
			Exchange:  u.config.DexID,
			Type:      DexTypeBancorV21InnerPool,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{reserveZero, reserveZero},
			Tokens:    entityTokens,
			Extra:     string(extra),
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}

// listPairAddresses lists address of pairs from offset
// return: poolAddresses, lpAddresses, error
func (u *PoolsListUpdater) listPairAddresses(ctx context.Context, allPairLength int) ([]common.Address, []common.Address, error) {
	anchors := make([]common.Address, allPairLength)
	listAnchorAddressesRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

	listAnchorAddressesRequest.AddCall(&ethrpc.Call{
		ABI:    converterRegistryABI,
		Target: u.config.ConverterRegistry,
		Method: registryGetAnchors,
	}, []interface{}{&anchors})

	_, err := listAnchorAddressesRequest.TryAggregate()
	if err != nil {
		return nil, nil, err
	}

	// get pool address (converters) from anchorResults (lp address)
	poolAddresses := make([]common.Address, allPairLength)
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(
		&ethrpc.Call{
			ABI:    converterRegistryABI,
			Target: u.config.ConverterRegistry,
			Method: getConvertersByAnchors,
			Params: []interface{}{anchors},
		}, []interface{}{&poolAddresses}).Call(); err != nil {
	}

	return poolAddresses, anchors, nil
}

// getBatchSize
// @params length number of pairs (factory tracked)
// @params limit number of pairs to be fetched in one run
// @params offset index of the last pair has been fetched
// @returns batchSize
func getBatchSize(length int, limit int, offset int) int {
	if offset == length {
		return 0
	}

	if offset+limit >= length {
		return length - offset
	}

	return limit
}
