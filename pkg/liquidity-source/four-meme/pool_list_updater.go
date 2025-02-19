package fourmeme

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
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

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
	var (
		dexID     = DexType
		startTime = time.Now()
	)

	logger.WithFields(logger.Fields{"exchange": dexID}).Info("Started getting new pools")

	ctx = util.NewContextWithTimestamp(ctx)

	allPairsLength, err := u.getAllPairsLength(ctx)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID}).
			Error("getAllPairsLength failed")

		return nil, metadataBytes, err
	}

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Warn("getOffset failed")
	}

	batchSize := u.getBatchSize(allPairsLength, u.config.NewPoolLimit, offset)

	tokenList, tokenInfoList, err := u.listPairs(ctx, offset, batchSize)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("listPairAddresses failed")

		return nil, metadataBytes, err
	}

	pools, err := u.initPools(tokenList, tokenInfoList)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("initPools failed")

		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(offset + batchSize)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("newMetadata failed")

		return nil, metadataBytes, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      dexID,
				"valid_pools": len(pools),
				"offset":      offset,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

// getAllPairsLength gets number of pairs from the factory contracts
func (u *PoolsListUpdater) getAllPairsLength(ctx context.Context) (int, error) {
	var allPairsLength *big.Int

	getAllPairsLengthRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

	getAllPairsLengthRequest.AddCall(&ethrpc.Call{
		ABI:    tokenManager2ABI,
		Target: u.config.TokenManagerV2,
		Method: tokenManager2TokenCountMethod,
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

// listPairAddresses lists address of pairs from offset
func (u *PoolsListUpdater) listPairs(ctx context.Context, offset int, batchSize int) ([]common.Address, []TokenInfo, error) {
	listTokens := make([]common.Address, batchSize)

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := 0; i < batchSize; i++ {
		index := big.NewInt(int64(offset + i))

		req.AddCall(&ethrpc.Call{
			ABI:    tokenManager2ABI,
			Target: u.config.TokenManagerV2,
			Method: tokenManager2TokensMethod,
			Params: []interface{}{index},
		}, []interface{}{&listTokens[i]})
	}
	_, err := req.Aggregate()
	if err != nil {
		return nil, nil, err
	}

	listQuotes := make([]TokenInfo, len(listTokens))

	req = u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := range listTokens {
		req.AddCall(&ethrpc.Call{
			ABI:    tokenManagerHelperABI,
			Target: u.config.TokenManagerHelperV3,
			Method: tokenManagerHelperGetTokenInfoMethod,
			Params: []interface{}{listTokens[i]},
		}, []interface{}{&listQuotes[i]})
	}

	_, err = req.Aggregate()
	if err != nil {
		return nil, nil, err
	}

	return listTokens, listQuotes, nil
}

// initPools fetches token data and initializes pools
func (u *PoolsListUpdater) initPools(tokenList []common.Address, tokenInfoList []TokenInfo) ([]entity.Pool, error) {
	pools := make([]entity.Pool, 0, len(tokenList))

	for i := range tokenList {
		token := tokenList[i].Hex()
		if strings.EqualFold(token, ZERO_ADDRESS) || tokenInfoList[i].LiquidityAdded {
			continue
		}

		raisedToken := tokenInfoList[i].Quote.Hex()
		if strings.EqualFold(raisedToken, ZERO_ADDRESS) {
			raisedToken = u.config.DefaultQuoteToken
		}

		extra, err := json.Marshal(&Extra{
			TradingFeeRate: tokenInfoList[i].TradingFeeRate,
		})
		if err != nil {
			return nil, err
		}

		var newPool = entity.Pool{
			Address:   strings.ToLower(tokenList[i].Hex()),
			Exchange:  string(valueobject.ExchangeFourMeme),
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(raisedToken),
					Swappable: true,
				},
				{
					Address:   strings.ToLower(token),
					Swappable: true,
				},
			},
			// StaticExtra: string(staticExtra),
			Extra: string(extra),
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

// getBatchSize
// @params length number of pairs (factory tracked)
// @params limit number of pairs to be fetched in one run
// @params offset index of the last pair has been fetched
// @returns batchSize
func (u *PoolsListUpdater) getBatchSize(length int, limit int, offset int) int {
	if offset == length {
		return 0
	}

	if offset+limit >= length {
		if offset > length {
			logger.WithFields(logger.Fields{
				"dex":    DexType,
				"offset": offset,
				"length": length,
			}).Warn("[getBatchSize] offset is greater than length")
		}
		return max(length-offset, 0)
	}

	return limit
}
