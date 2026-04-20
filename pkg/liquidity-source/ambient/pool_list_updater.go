package ambient

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdater struct {
	config       *Config
	httpClient   *resty.Client
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolListUpdater)

func NewPoolListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolListUpdater {
	httpClient := resty.NewWithClient(http.DefaultClient).
		SetBaseURL(cfg.HTTPConfig.BaseURL).
		SetTimeout(cfg.HTTPConfig.Timeout.Duration).
		SetRetryCount(cfg.HTTPConfig.RetryCount)

	return &PoolListUpdater{
		config:       cfg,
		httpClient:   httpClient,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.Info("started getting new pools")

	indexerPools, err := u.fetchIndexer(ctx)
	if err != nil {
		return nil, metadataBytes, err
	}

	newPools := make([]entity.Pool, 0, len(indexerPools))
	for _, p := range indexerPools {
		if p.PoolIdx != u.config.PoolIdx.Uint64() {
			continue
		}

		base, quote := normalizePair(p.Base, p.Quote)
		poolHash := EncodePoolHash(common.HexToAddress(base), common.HexToAddress(quote), p.PoolIdx)

		staticExtraBytes, err := json.Marshal(StaticExtra{
			NativeToken: valueobject.LowerWrapped(u.config.ChainId),
			PoolIdx:     p.PoolIdx,
			SwapDex:     u.config.SwapDex,
			Base:        strings.ToLower(base),
			Quote:       strings.ToLower(quote),
		})
		if err != nil {
			continue
		}

		newPools = append(newPools, entity.Pool{
			Address:     strings.ToLower(poolHash.Hex()),
			Exchange:    string(u.config.DexId),
			Type:        DexType,
			StaticExtra: string(staticExtraBytes),
			Tokens: []*entity.PoolToken{
				{Address: valueobject.ZeroToWrappedLower(base, u.config.ChainId), Swappable: true},
				{Address: valueobject.ZeroToWrappedLower(quote, u.config.ChainId), Swappable: true},
			},
			Reserves: []string{"0", "0"},
		})
	}

	logger.Info("finished getting new pools")

	return newPools, metadataBytes, nil
}

func (u *PoolListUpdater) fetchIndexer(ctx context.Context) ([]IndexerPool, error) {
	req := u.httpClient.R().SetContext(ctx)
	if u.config.IndexerChainId != "" {
		req.SetQueryParam("chainId", u.config.IndexerChainId)
	}

	resp, err := req.Get(indexerPoolListPath)
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("indexer status %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	var parsed IndexerPoolsResponse
	if err := json.Unmarshal(resp.Body(), &parsed); err != nil {
		return nil, fmt.Errorf("decode indexer response: %w", err)
	}

	return parsed.Data, nil
}
