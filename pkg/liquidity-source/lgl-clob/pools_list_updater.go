package lglclob

import (
	"context"
	"slices"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
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
	httpClient := resty.New().
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
	l := logger.WithFields(logger.Fields{
		"dexID": u.config.DexID,
	})
	l.Info("Start getting new pools")

	var metadata Metadata
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}

	markets, err := u.getPoolsList(ctx)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get pools list")
		return nil, nil, err
	}

	var poolsChecksum common.Address
	for _, market := range markets {
		poolAddr := common.HexToAddress(market.OrderbookAddress)
		for i := range common.AddressLength {
			poolsChecksum[i] ^= poolAddr[i]
		}
	}
	if metadata.LastCount == len(markets) && metadata.LastPoolsChecksum == poolsChecksum {
		return nil, metadataBytes, nil
	}
	metadata.LastCount, metadata.LastPoolsChecksum = len(markets), poolsChecksum
	l.Infof("got %v markets", metadata.LastCount)

	pools := make([]entity.Pool, len(markets))
	staticExtras, err := u.getLobConfig(ctx, markets)
	if err != nil {
		return nil, nil, err
	}
	for i, market := range markets {
		staticExtraBytes, _ := json.Marshal(staticExtras[i])
		pools[i] = entity.Pool{
			Address:  strings.ToLower(market.OrderbookAddress),
			SwapFee:  market.AggressiveFee,
			Exchange: u.config.DexID,
			Type:     DexType,
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(market.BaseToken.ContractAddress), // X
					Symbol:    market.BaseToken.Symbol,
					Decimals:  market.BaseToken.Decimals,
					Swappable: true,
				},
				{
					Address:   strings.ToLower(market.QuoteToken.ContractAddress), // Y
					Symbol:    market.QuoteToken.Symbol,
					Decimals:  market.QuoteToken.Decimals,
					Swappable: true,
				},
			},
			Reserves:    entity.PoolReserves{"0", "0"},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		}
	}

	metadataBytes, _ = json.Marshal(metadata)
	return pools, metadataBytes, nil
}

func (u *PoolListUpdater) getPoolsList(ctx context.Context) ([]*MarketInfo, error) {
	var result []*MarketInfo
	if resp, err := u.httpClient.NewRequest().
		SetContext(ctx).
		SetResult(&result).
		Get("/markets"); err != nil || !resp.IsSuccess() {
		return nil, errors.Errorf("failed to get pools list: %v, resp=%v", err, resp)
	}
	return result, nil
}

const MaxBatchSize = 64

func (u *PoolListUpdater) getLobConfig(ctx context.Context, markets []*MarketInfo) ([]*StaticExtra, error) {
	if len(markets) > MaxBatchSize {
		staticExtras := make([]*StaticExtra, 0, len(markets))
		for marketsChunk := range slices.Chunk(markets, MaxBatchSize) {
			staticExtrasChunk, err := u.getLobConfig(ctx, marketsChunk)
			if err != nil {
				return nil, err
			}
			staticExtras = append(staticExtras, staticExtrasChunk...)
		}
		return staticExtras, nil
	}

	lobCfgs := make([]*LobConfig, len(markets))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, market := range markets {
		lobCfgs[i] = new(LobConfig)
		req.AddCall(&ethrpc.Call{
			ABI:    onchainClobABI,
			Target: market.OrderbookAddress,
			Method: "getConfig",
		}, []any{lobCfgs[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	return lo.Map(lobCfgs, func(lobCfg *LobConfig, _ int) *StaticExtra {
		return &StaticExtra{
			ScalingFactorX:    uint256.MustFromBig(lobCfg.ScalingFactorTokenX),
			ScalingFactorY:    uint256.MustFromBig(lobCfg.ScalingFactorTokenY),
			SupportsNativeEth: lobCfg.SupportsNativeEth,
		}
	}), nil
}
