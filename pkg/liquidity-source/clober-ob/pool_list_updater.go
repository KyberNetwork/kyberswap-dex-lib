package cloberob

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	cloberlib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clober-ob/libraries"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
)

type (
	PoolsListUpdater struct {
		config        *Config
		ethrpcClient  *ethrpc.Client
		graphqlClient *graphqlpkg.Client
	}

	Metadata struct {
		LastCreatedAtTimestamp int64  `json:"lastCreatedAtTimestamp"`
		LastProcessedPoolId    string `json:"lastProcessedPoolID"`
	}
)

var _ = poollist.RegisterFactoryCEG(DexType, NewPoolListUpdater)

func NewPoolListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:        config,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.Info("starting to get new pools")

	var metadata Metadata
	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	subgraphBooks, err := u.getBooks(ctx, metadata.LastCreatedAtTimestamp, u.config.NewPoolLimit)
	if err != nil {
		return nil, metadataBytes, err
	}

	pools := make([]entity.Pool, 0, len(subgraphBooks))
	for _, book := range subgraphBooks {
		baseDecimals, err := kutils.Atou[uint8](book.Base.Decimals)
		if err != nil {
			return nil, metadataBytes, err
		}
		quoteDecimals, err := kutils.Atou[uint8](book.Quote.Decimals)
		if err != nil {
			return nil, metadataBytes, err
		}

		tokens := []*entity.PoolToken{
			{
				Address:   valueobject.ZeroToWrappedLower(book.Base.Id, u.config.ChainId),
				Decimals:  baseDecimals,
				Symbol:    book.Base.Symbol,
				Swappable: true,
			},
			{
				Address:   valueobject.ZeroToWrappedLower(book.Quote.Id, u.config.ChainId),
				Decimals:  quoteDecimals,
				Symbol:    book.Quote.Symbol,
				Swappable: true,
			},
		}

		unitSize, err := kutils.Atou[uint64](book.UnitSize)
		if err != nil {
			return nil, metadataBytes, err
		}

		makerPolicy, err := kutils.Atou[cloberlib.FeePolicy](book.MakerPolicy)
		if err != nil {
			return nil, metadataBytes, err
		}
		takerPolicy, err := kutils.Atou[cloberlib.FeePolicy](book.TakerPolicy)
		if err != nil {
			return nil, metadataBytes, err
		}

		staticExtra := StaticExtra{
			Base:        common.HexToAddress(book.Base.Id),
			Quote:       common.HexToAddress(book.Quote.Id),
			UnitSize:    unitSize,
			MakerPolicy: makerPolicy,
			TakerPolicy: takerPolicy,
			Hooks:       common.HexToAddress(book.Hooks),
			BookManager: u.config.BookManager,
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			return nil, metadataBytes, err
		}

		pool := entity.Pool{
			Address:     strings.ToLower(book.Id),
			Exchange:    GetExchangeByHook(staticExtra.Hooks),
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    entity.PoolReserves{"0", "0"},
			Tokens:      tokens,
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, pool)
	}

	if len(subgraphBooks) > 0 {
		lastCreatedAtTimestamp, err := kutils.Atoi[int64](subgraphBooks[len(subgraphBooks)-1].CreatedAtTimestamp)
		if err != nil {
			return nil, metadataBytes, err
		}

		metadata.LastCreatedAtTimestamp = lastCreatedAtTimestamp
		metadata.LastProcessedPoolId = subgraphBooks[len(subgraphBooks)-1].Id
		metadataBytes, err = json.Marshal(metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	logger.WithFields(logger.Fields{
		"dexId": u.config.DexId,
		"pools": len(pools),
	}).Info("finished getting new pools")

	return pools, metadataBytes, nil
}

func (u *PoolsListUpdater) getBooks(ctx context.Context, lastCreatedAtTimestamp int64, first int) ([]SubgraphBook, error) {
	req := graphqlpkg.NewRequest(getBooksQuery(lastCreatedAtTimestamp, first))
	var resp struct {
		Books []SubgraphBook `json:"books"`
	}
	if err := u.graphqlClient.Run(ctx, req, &resp); err != nil {
		logger.WithFields(logger.Fields{
			"dexId": u.config.DexId,
			"error": err,
		}).Errorf("failed to query subgraph")
		return nil, err
	}

	return resp.Books, nil
}
