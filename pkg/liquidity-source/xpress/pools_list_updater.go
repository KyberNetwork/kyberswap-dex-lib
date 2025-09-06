package xpress

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-resty/resty/v2"
	"github.com/samber/lo"
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
	metadata := Metadata{
		Pools: []common.Address{},
	}

	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	markets, err := u.getPoolsList(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get pools list")
		return nil, nil, err
	}

	numMarkets := len(markets)
	logger.Infof("got %v markets", numMarkets)

	pools := make([]entity.Pool, 0, len(markets))

	for _, market := range markets {
		// skip pool if already processed
		if lo.Contains(metadata.Pools, common.HexToAddress(market.OrderbookAddress)) {
			continue
		}

		staticExtra, err := u.getLobConfig(ctx, &market)
		if err != nil {
			return nil, nil, err
		}

		// TODO: compare scaling factors from staticExtra with market info

		staticExtraBytes, _ := json.Marshal(staticExtra)

		var newPool = entity.Pool{
			Address:  market.OrderbookAddress,
			SwapFee:  market.AggressiveFee,
			Exchange: u.config.DexId,
			Type:     DexType,
			//Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{
					Address:   market.BaseToken.ContractAddress,
					Symbol:    market.BaseToken.Symbol,
					Decimals:  market.BaseToken.Decimals,
					Swappable: true,
				},
				{
					Address:   market.QuoteToken.ContractAddress,
					Symbol:    market.QuoteToken.Symbol,
					Decimals:  market.QuoteToken.Decimals,
					Swappable: true,
				},
			},
			Reserves:    entity.PoolReserves{"0", "0"},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, newPool)

		metadata.Pools = append(metadata.Pools, common.HexToAddress(market.OrderbookAddress))
	}

	metadataBytes, err = json.Marshal(metadata)
	if err != nil {
		return nil, metadataBytes, err
	}

	return pools, metadataBytes, nil
}

func (u *PoolListUpdater) getPoolsList(ctx context.Context) ([]MarketInfo, error) {
	var result []MarketInfo

	resp, err := u.httpClient.NewRequest().
		SetContext(ctx).
		SetResult(&result).
		Get("/markets")

	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, errors.New("failed to get pools list")
	}

	return result, nil
}

func (u *PoolListUpdater) getLobConfig(ctx context.Context, market *MarketInfo) (*LobConfig, error) {
	lobConfig := LobConfig{}
	rpcRequests := u.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequests.AddCall(&ethrpc.Call{
		ABI:    onchainClobABI,
		Target: market.OrderbookAddress,
		Method: "getConfig",
		Params: nil,
	}, []any{&lobConfig})

	_, err := rpcRequests.Call()
	if err != nil {
		return nil, err
	}

	return &lobConfig, nil
}
