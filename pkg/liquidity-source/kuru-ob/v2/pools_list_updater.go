package kuruobv2

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"

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
	return &PoolListUpdater{
		config: cfg,
		httpClient: resty.NewWithClient(lo.ToPtr(lo.FromPtr(http.DefaultClient))).
			SetBaseURL(cfg.HTTPConfig.BaseURL).
			SetTimeout(cfg.HTTPConfig.Timeout.Duration).
			SetRetryCount(cfg.HTTPConfig.RetryCount),
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}

	markets, err := u.getMarkets(ctx)
	if err != nil {
		return nil, metadataBytes, err
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

	params, err := u.getMarketParams(ctx, markets)
	if err != nil {
		return nil, metadataBytes, err
	}
	log.Info().Str("dexID", u.config.DexID).Int("count", len(markets)).Msg("fetched new markets")

	pools := make([]entity.Pool, len(markets))
	for i, market := range markets {
		var native bool
		tokens := lo.Map([]string{market.BaseToken.TokenAddress, market.QuoteToken.TokenAddress},
			func(token string, _ int) *entity.PoolToken {
				if valueobject.IsZero(token) { // V2 uses address(0) for native MON
					token, native = valueobject.WrappedNativeMap[u.config.ChainId], true
				}
				return &entity.PoolToken{Address: strings.ToLower(token), Swappable: true}
			})
		staticExtraBytes, _ := json.Marshal(StaticExtra{
			PricePrecision:     uint64(params[i].PricePrecision),
			SizePrecision:      uint256.MustFromBig(params[i].SizePrecision),
			BaseSizeMultiplier: uint256.MustFromBig(params[i].BaseSizeMultiplier),
			QuoteDecimals:      params[i].QuoteTokenDecimals,
			HasNative:          native,
		})
		pools[i] = entity.Pool{
			Address:     strings.ToLower(market.MarketAddress),
			Exchange:    u.config.DexID,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    entity.PoolReserves{"0", "0"},
			Tokens:      tokens,
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		}
	}

	metadata.LastCount, metadata.LastPoolsChecksum = len(markets), poolsChecksum
	metadataBytes, _ = json.Marshal(metadata)
	return pools, metadataBytes, nil
}

// ponytail: single page of 500 (API max); follow pagination.nextCursor if Kuru ever lists more.
func (u *PoolListUpdater) getMarkets(ctx context.Context) ([]*MarketInfo, error) {
	var result struct{ Data []*MarketInfo }
	if resp, err := u.httpClient.R().SetContext(ctx).SetResult(&result).
		SetQueryParams(map[string]string{"status": "active", "limit": "500"}).
		Get("/api/v1/markets"); err != nil || !resp.IsSuccess() {
		return nil, errors.Errorf("failed to get markets: %v, resp=%v", err, resp)
	}
	return result.Data, nil
}

func (u *PoolListUpdater) getMarketParams(ctx context.Context, markets []*MarketInfo) ([]MarketParamsRPC, error) {
	params := make([]MarketParamsRPC, len(markets))
	for start := 0; start < len(markets); start += marketBatchSize {
		req := u.ethrpcClient.NewRequest().SetContext(ctx)
		for i := start; i < min(start+marketBatchSize, len(markets)); i++ {
			target := markets[i].MarketAddress
			req.AddCall(&ethrpc.Call{ABI: kuruV2ABI, Target: target, Method: "pricePrecision"},
				[]any{&params[i].PricePrecision}).
				AddCall(&ethrpc.Call{ABI: kuruV2ABI, Target: target, Method: "sizePrecision"},
					[]any{&params[i].SizePrecision}).
				AddCall(&ethrpc.Call{ABI: kuruV2ABI, Target: target, Method: "baseSizeMultiplier"},
					[]any{&params[i].BaseSizeMultiplier}).
				AddCall(&ethrpc.Call{ABI: kuruV2ABI, Target: target, Method: "quoteTokenDecimals"},
					[]any{&params[i].QuoteTokenDecimals})
		}
		if _, err := req.Aggregate(); err != nil {
			return nil, err
		}
	}
	return params, nil
}
