package printr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	httpClient   *resty.Client
	logger       logger.Logger
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
		httpClient:   resty.New().SetTimeout(30 * time.Second),
		logger:       logger.WithFields(logger.Fields{"dex_id": cfg.DexId}),
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	startTime := time.Now()
	u.logger.Info("started getting new pools")

	var metadata PoolsListUpdaterMetadata
	if len(metadataBytes) > 0 {
		_ = json.Unmarshal(metadataBytes, &metadata)
	}

	list, err := u.fetchTokenList(ctx, u.config.NewPoolLimit, metadata.Skip)
	if err != nil {
		u.logger.WithFields(logger.Fields{"error": err}).Warn("failed to fetch token list")
		return nil, metadataBytes, nil
	}

	pools, err := u.buildPools(ctx, list)
	if err != nil {
		u.logger.WithFields(logger.Fields{"error": err}).Error("failed to build pools from token list and onchain getCurve")
		return nil, metadataBytes, err
	}

	newMetadata := PoolsListUpdaterMetadata{
		Skip: metadata.Skip + len(list.Tokens),
	}

	newMetadataBytes, err := json.Marshal(newMetadata)
	if err != nil {
		return nil, metadataBytes, err
	}

	u.logger.WithFields(logger.Fields{
		"new_pools":   len(pools),
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) fetchTokenList(ctx context.Context, size, skip int) (*TokenListResponse, error) {
	base := strings.TrimSuffix(u.config.TokenListAPI, "/")
	rawURL, err := url.JoinPath(base, "chains", strconv.FormatInt(int64(u.config.ChainId), 10), "tokenlist.json")
	if err != nil {
		return nil, err
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	q := parsed.Query()
	q.Set("size", strconv.Itoa(size))
	q.Set("skip", strconv.Itoa(skip))
	parsed.RawQuery = q.Encode()

	var tokenList TokenListResponse
	resp, err := u.httpClient.R().
		SetContext(ctx).
		SetResult(&tokenList).
		Get(parsed.String())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("token list API returned status %d", resp.StatusCode())
	}
	return &tokenList, nil
}

func (u *PoolsListUpdater) buildPools(ctx context.Context, tokenList *TokenListResponse) ([]entity.Pool, error) {
	var candidates []TokenListEntry
	for i := range tokenList.Tokens {
		t := &tokenList.Tokens[i]
		if t.ChainId != int(u.config.ChainId) {
			continue
		}
		isGraduated, _ := t.Extensions["isGraduated"].(bool)
		if isGraduated {
			continue
		}
		candidates = append(candidates, *t)
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	curveResults := make([]GetCurveResult, len(candidates))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := range candidates {
		tokenAddr := common.HexToAddress(candidates[i].Address)
		req.AddCall(&ethrpc.Call{
			ABI:    printrABI,
			Target: u.config.PrintrAddr,
			Method: printrMethodGetCurve,
			Params: []any{tokenAddr},
		}, []any{&curveResults[i]})
	}

	resp, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(candidates))
	for i, ok := range resp.Result {
		if !ok || curveResults[i].Data.MaxTokenSupply == nil || curveResults[i].Data.VirtualReserve == nil {
			continue
		}
		tokenAddr := strings.ToLower(candidates[i].Address)
		basePairAddr := strings.ToLower(curveResults[i].Data.BasePair.Hex())

		staticExtraBytes, _ := json.Marshal(StaticExtra{
			PrintrAddr:     strings.ToLower(u.config.PrintrAddr),
			Token:          tokenAddr,
			BasePair:       basePairAddr,
			TotalCurves:    curveResults[i].Data.TotalCurves,
			MaxTokenSupply: curveResults[i].Data.MaxTokenSupply.String(),
			VirtualReserve: curveResults[i].Data.VirtualReserve.String(),
		})

		p := entity.Pool{
			Address:     tokenAddr,
			Exchange:    u.config.DexId,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    []string{"0", "0"},
			StaticExtra: string(staticExtraBytes),
			Tokens: []*entity.PoolToken{
				{Address: basePairAddr, Swappable: true},
				{Address: tokenAddr, Swappable: true},
			},
		}
		pools = append(pools, p)
	}
	return pools, nil
}
