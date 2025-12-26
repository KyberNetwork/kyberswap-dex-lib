package kuruob

import (
	"context"
	"math"
	"net/http"
	"slices"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

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

func NewPoolListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
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
	l := log.With().Str("dexID", u.config.DexID).Logger()
	l.Info().Msg("Start getting new pools")

	var metadata Metadata
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}

	markets, err := u.getPoolsList(ctx)
	if err != nil {
		l.Err(err).Msg("failed to get pools list")
		return nil, nil, err
	}

	var poolsChecksum common.Address
	for _, market := range markets {
		poolAddr := common.HexToAddress(market.MarketAddress)
		for i := range common.AddressLength {
			poolsChecksum[i] ^= poolAddr[i]
		}
	}
	if metadata.LastCount == len(markets) && metadata.LastPoolsChecksum == poolsChecksum {
		return nil, metadataBytes, nil
	}
	metadata.LastCount, metadata.LastPoolsChecksum = len(markets), poolsChecksum
	l.Info().Int("count", metadata.LastCount).Msg("fetched new markets")

	pools := make([]entity.Pool, len(markets))
	marketParamsLst, err := u.getMarketParams(ctx, markets)
	if err != nil {
		return nil, nil, err
	}
	for i, market := range markets {
		marketParams := marketParamsLst[i]
		fee, _ := marketParams.TakerFeeBps.Float64()
		tokens := []*entity.PoolToken{
			{
				Address:   strings.ToLower(market.BaseToken.Address),
				Symbol:    market.BaseToken.Ticker,
				Decimals:  market.BaseToken.Decimal,
				Swappable: true,
			},
			{
				Address:   strings.ToLower(market.QuoteToken.Address),
				Symbol:    market.QuoteToken.Ticker,
				Decimals:  market.QuoteToken.Decimal,
				Swappable: true,
			},
		}
		var native bool
		for _, token := range tokens {
			if valueobject.IsZero(token.Address) {
				token.Address = strings.ToLower(valueobject.WrappedNativeMap[u.config.ChainId])
				token.Symbol = "W" + token.Symbol
				native = true
			}
		}
		sizePrecision, _ := marketParams.SizePrecision.Float64()
		staticExtraBytes, _ := json.Marshal(StaticExtra{
			PricePrecision: int(math.Round(math.Log10(float64(marketParams.PricePrecision)))),
			SizePrecision:  int(math.Round(math.Log10(sizePrecision))),
			HasNative:      native,
		})
		pools[i] = entity.Pool{
			Address:     strings.ToLower(market.MarketAddress),
			SwapFee:     fee / 10000,
			Exchange:    u.config.DexID,
			Type:        DexType,
			Tokens:      tokens,
			Reserves:    entity.PoolReserves{"0", "0"},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		}
	}

	metadataBytes, _ = json.Marshal(metadata)
	return pools, metadataBytes, nil
}

func (u *PoolListUpdater) getPoolsList(ctx context.Context) ([]*MarketInfo, error) {
	var result struct{ Data struct{ Data []*MarketInfo } }
	if resp, err := u.httpClient.NewRequest().
		SetContext(ctx).
		SetResult(&result).
		Get("/api/v2/vaults"); err != nil || !resp.IsSuccess() {
		return nil, errors.Errorf("failed to get pools list: %v, resp=%v", err, resp)
	}
	return result.Data.Data, nil
}

const MaxBatchSize = 64

func (u *PoolListUpdater) getMarketParams(ctx context.Context, markets []*MarketInfo) ([]*MarketParamsRPC, error) {
	if len(markets) > MaxBatchSize {
		marketParamsLst := make([]*MarketParamsRPC, 0, len(markets))
		for marketsChunk := range slices.Chunk(markets, MaxBatchSize) {
			marketParamsChunk, err := u.getMarketParams(ctx, marketsChunk)
			if err != nil {
				return nil, err
			}
			marketParamsLst = append(marketParamsLst, marketParamsChunk...)
		}
		return marketParamsLst, nil
	}

	marketParamsLst := make([]*MarketParamsRPC, len(markets))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, market := range markets {
		marketParamsLst[i] = new(MarketParamsRPC)
		req.AddCall(&ethrpc.Call{
			ABI:    orderBookABI,
			Target: market.MarketAddress,
			Method: "getMarketParams",
		}, []any{marketParamsLst[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	return marketParamsLst, nil
}
