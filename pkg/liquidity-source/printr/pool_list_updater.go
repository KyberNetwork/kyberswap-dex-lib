package printr

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	httpClient   *http.Client
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
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		logger:       logger.WithFields(logger.Fields{"dex_id": cfg.DexId}),
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	startTime := time.Now()
	u.logger.Info("started getting new pools")

	var metadata PoolsListUpdaterMetadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			u.logger.WithFields(logger.Fields{"error": err}).Warn("failed to unmarshal metadata")
		}
	}

	tokenList, err := u.fetchTokenList(ctx)
	if err != nil {
		u.logger.WithFields(logger.Fields{"error": err}).Error("failed to fetch token list")
		return nil, metadataBytes, err
	}

	// Check if token list version changed
	if tokenList.Version.Major == metadata.VersionMajor &&
		tokenList.Version.Minor == metadata.VersionMinor &&
		tokenList.Version.Patch == metadata.VersionPatch {
		u.logger.Info("token list unchanged, no new pools")
		return nil, metadataBytes, nil
	}

	pools := u.buildPools(tokenList)

	newMetadata := PoolsListUpdaterMetadata{
		VersionMajor: tokenList.Version.Major,
		VersionMinor: tokenList.Version.Minor,
		VersionPatch: tokenList.Version.Patch,
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

func (u *PoolsListUpdater) fetchTokenList(ctx context.Context) (*TokenListResponse, error) {
	url := fmt.Sprintf("%s/chains/%d/tokenlist.json", u.config.TokenListAPI, u.config.ChainId)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token list API returned status %d", resp.StatusCode)
	}

	var tokenList TokenListResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenList); err != nil {
		return nil, err
	}

	return &tokenList, nil
}

func (u *PoolsListUpdater) buildPools(tokenList *TokenListResponse) []entity.Pool {
	pools := make([]entity.Pool, 0, len(tokenList.Tokens))

	for _, token := range tokenList.Tokens {
		// Filter for non-graduated tokens on this chain
		if token.ChainId != int(u.config.ChainId) {
			continue
		}

		isGraduated, _ := token.Extensions["isGraduated"].(bool)
		if isGraduated {
			continue
		}

		basePair, _ := token.Extensions["basePair"].(string)
		if basePair == "" {
			continue
		}

		totalCurvesFloat, _ := token.Extensions["totalCurves"].(float64)
		totalCurves := uint16(totalCurvesFloat)
		if totalCurves == 0 {
			continue
		}

		maxTokenSupply, _ := token.Extensions["maxTokenSupply"].(string)
		virtualReserve, _ := token.Extensions["virtualReserve"].(string)
		if maxTokenSupply == "" || virtualReserve == "" {
			continue
		}

		tokenAddr := strings.ToLower(token.Address)
		basePairAddr := strings.ToLower(basePair)

		staticExtraBytes, _ := json.Marshal(StaticExtra{
			PrintrAddr:     u.config.PrintrAddr,
			Token:          tokenAddr,
			BasePair:       basePairAddr,
			TotalCurves:    totalCurves,
			MaxTokenSupply: maxTokenSupply,
			VirtualReserve: virtualReserve,
		})

		p := entity.Pool{
			Address:   tokenAddr,
			Exchange:  u.config.DexId,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: basePairAddr, Swappable: true},
				{Address: tokenAddr, Swappable: true},
			},
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, p)

		if u.config.NewPoolLimit > 0 && len(pools) >= u.config.NewPoolLimit {
			break
		}
	}

	return pools
}
