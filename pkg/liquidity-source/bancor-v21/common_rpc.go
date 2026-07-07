package bancorv21

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// listPairAddresses lists address of pairs from offset
// return: poolAddresses, lpAddresses, error
func listPairAddresses(ctx context.Context, ethrpcClient *ethrpc.Client, converterRegistry string, allPairLength int) ([]common.Address, []common.Address, error) {
	anchors := make([]common.Address, allPairLength)
	listAnchorAddressesRequest := ethrpcClient.NewRequest().SetContext(ctx)

	listAnchorAddressesRequest.AddCall(&ethrpc.Call{
		ABI:    converterRegistryABI,
		Target: converterRegistry,
		Method: registryGetAnchors,
	}, []any{&anchors})

	_, err := listAnchorAddressesRequest.TryAggregate()
	if err != nil {
		return nil, nil, err
	}

	// get pool address (converters) from anchorResults (lp address)
	poolAddresses := make([]common.Address, allPairLength)
	if _, err := ethrpcClient.NewRequest().SetContext(ctx).AddCall(
		&ethrpc.Call{
			ABI:    converterRegistryABI,
			Target: converterRegistry,
			Method: getConvertersByAnchors,
			Params: []any{anchors},
		}, []any{&poolAddresses}).Call(); err != nil {
		return nil, nil, err
	}

	return poolAddresses, anchors, nil
}

func getConvertibleTokensAnchorState(ctx context.Context, ethrpcClient *ethrpc.Client, converterRegistry string) (map[string][]string, error) {
	convertibleTokens := make([]common.Address, 0)
	if _, err := ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    converterRegistryABI,
		Target: converterRegistry,
		Method: getConvertibleTokens,
		Params: nil,
	}, []any{&convertibleTokens}).Call(); err != nil {
		return nil, err
	}

	anchorsByConvertibleTokens := make(map[string][]string)
	anchorsRequest := ethrpcClient.NewRequest().SetContext(ctx)
	anchors := make([][]common.Address, len(convertibleTokens))

	for i, convertibleToken := range convertibleTokens {
		anchors[i] = make([]common.Address, 0)
		anchorsRequest.AddCall(&ethrpc.Call{
			ABI:    converterRegistryABI,
			Target: converterRegistry,
			Method: getConvertibleTokenAnchors,
			Params: []any{convertibleToken},
		}, []any{&anchors[i]})
	}

	if _, err := anchorsRequest.Aggregate(); err != nil {
		return nil, err
	}

	for i, convertibleToken := range convertibleTokens {
		convertibleTokenStr := hexutil.Encode(convertibleToken[:])
		anchorsByConvertibleTokens[convertibleTokenStr] = make([]string, len(anchors[i]))
		for j, anchor := range anchors[i] {
			anchorsByConvertibleTokens[convertibleTokenStr][j] = hexutil.Encode(anchor[:])
		}
	}

	return anchorsByConvertibleTokens, nil
}

// getAllPairsLength gets number of pairs from the factory contracts
func getAllPairsLength(ctx context.Context, ethrpcClient *ethrpc.Client, converterRegistry string) (int, error) {
	var allPairsLength *big.Int
	//
	getAllPairsLengthRequest := ethrpcClient.NewRequest().SetContext(ctx)

	getAllPairsLengthRequest.AddCall(&ethrpc.Call{
		ABI:    converterRegistryABI,
		Target: converterRegistry,
		Method: getAnchorCount,
		Params: nil,
	}, []any{&allPairsLength})

	if _, err := getAllPairsLengthRequest.Call(); err != nil {
		return 0, err
	}

	return int(allPairsLength.Int64()), nil
}

// listPairTokens receives list of pair addresses and returns their tokens
func listPairTokens(ctx context.Context, ethrpcClient *ethrpc.Client, pairAddresses []common.Address) ([][]common.Address, error) {
	listTokensRequest := ethrpcClient.NewRequest().SetContext(ctx)
	tokens := make([][]common.Address, len(pairAddresses))

	for index, pairAddress := range pairAddresses {
		var numToken uint16
		if _, err := ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
			ABI:    converterABI,
			Target: pairAddress.Hex(),
			Method: converterGetTokenCount,
			Params: nil,
		}, []any{&numToken}).Call(); err != nil {
			return nil, err
		}
		nTokens := int(numToken)
		tokens[index] = make([]common.Address, nTokens)

		for i := 0; i < nTokens; i++ {
			listTokensRequest.AddCall(&ethrpc.Call{
				ABI:    converterABI,
				Target: pairAddress.Hex(),
				Method: converterGetTokens,
				Params: []any{big.NewInt(int64(i))},
			}, []any{&tokens[index][i]})
		}
	}

	if _, err := listTokensRequest.Aggregate(); err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": "bancor-v21"}).
			Error("Get tokens list for pool failed")
		return nil, err
	}

	return tokens, nil
}

// initInnerPools fetches token data and initializes pools
func initInnerPools(ctx context.Context, ethrpcClient *ethrpc.Client, pairAddresses, anchors []common.Address) ([]entity.Pool, map[string][]string, error) {
	tokens, err := listPairTokens(ctx, ethrpcClient, pairAddresses)
	if err != nil {
		return nil, nil, err
	}

	tokensByAnchors := make(map[string][]string, len(anchors)*2)
	for i, anchor := range anchors {
		anchorAddress := hexutil.Encode(anchor[:])
		tokensByAnchors[anchorAddress] = make([]string, len(tokens[i]))
		for tokenIndex, token := range tokens[i] {
			tokensByAnchors[anchorAddress][tokenIndex] = hexutil.Encode(token[:])
		}
	}

	pools := make([]entity.Pool, 0, len(pairAddresses))

	for i, pairAddress := range pairAddresses {
		entityTokens := make([]*entity.PoolToken, len(tokens[i]))
		for tokenIndex := 0; tokenIndex < len(tokens[i]); tokenIndex++ {
			entityTokens[tokenIndex] = &entity.PoolToken{
				Address:   hexutil.Encode(tokens[i][tokenIndex][:]),
				Swappable: true,
			}
		}

		extra, err := newExtraInner(hexutil.Encode(anchors[i][:]))
		if err != nil {
			return nil, nil, err
		}

		var newPool = entity.Pool{
			Address:   hexutil.Encode(pairAddress[:]),
			Exchange:  DexTypeBancorV21InnerPool,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{reserveZero, reserveZero},
			Tokens:    entityTokens,
			Extra:     string(extra),
		}

		pools = append(pools, newPool)
	}

	return pools, tokensByAnchors, nil
}

func newExtraInner(anchorAddress string) ([]byte, error) {
	extra := ExtraInner{
		AnchorAddress: anchorAddress,
	}

	return json.Marshal(extra)
}
