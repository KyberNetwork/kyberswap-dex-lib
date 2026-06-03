package altfun

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

// apiToken is a single entry from GET /api/v1/tokens.
type apiToken struct {
	Address     string `json:"address"`
	BondingPair string `json:"bondingPair"` // bonding-curve Pair contract
	LTPair      string `json:"ltPair"`      // BounceTech LT address
	Leverage    uint64 `json:"leverage"`    // raw int (e.g. 3 for 3x)
	Status      string `json:"status"`      // "curve" | "graduated"
	CreatedAt   string `json:"createdAt"`   // RFC3339 timestamp
}

type apiResponse struct {
	Status string     `json:"status"`
	Data   []apiToken `json:"data"`
}

// ListMetadata is persisted between GetNewPools calls.
type ListMetadata struct {
	// LastCreatedAt is the RFC3339 timestamp of the newest token already discovered.
	// The API returns newest-first; we page from offset=0 and stop when we reach
	// tokens older than (or equal to) this value.
	LastCreatedAt string `json:"lastCreatedAt"`
}

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	httpClient   *resty.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	if config.APIURL == "" {
		config.APIURL = defaultAPIURL
	}
	if config.HTTPConfig.Timeout.Duration == 0 {
		config.HTTPConfig.Timeout = defaultTimeout
	}
	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
		httpClient:   resty.New().SetBaseURL(config.APIURL).SetTimeout(config.HTTPConfig.Timeout.Duration).SetRetryCount(config.HTTPConfig.RetryCount),
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var meta ListMetadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &meta); err != nil {
			return nil, metadataBytes, err
		}
	}

	// Fetch protocol-level constants once per batch.
	usdc, buyFeeBps, sellFeeBps, graduationThresholdUsd, err := u.fetchZapParams(ctx)
	if err != nil {
		return nil, metadataBytes, err
	}

	limit := u.config.NewPoolLimit
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	var (
		newPools        []entity.Pool
		newestCreatedAt string // createdAt of the first new token we see
		offset          int
	)

	for {
		tokens, err := u.fetchPage(ctx, limit, offset)
		if err != nil {
			return nil, metadataBytes, err
		}
		if len(tokens) == 0 {
			break
		}

		reachedCheckpoint := false
		for _, t := range tokens {
			if meta.LastCreatedAt != "" && t.CreatedAt <= meta.LastCreatedAt {
				// Reached already-known territory — stop scanning.
				reachedCheckpoint = true
				break
			}
			if t.Status != "curve" && t.Status != "" {
				// Skip graduated pools at discovery time; they appear as curve tokens
				// initially and only graduate later (tracker handles that transition).
				continue
			}
			if t.Address == "" || t.BondingPair == "" || t.LTPair == "" {
				continue
			}
			// Record the newest token (first one we see on offset=0).
			if newestCreatedAt == "" {
				newestCreatedAt = t.CreatedAt
			}
			p, err := u.newPool(t, usdc, buyFeeBps, sellFeeBps, graduationThresholdUsd)
			if err != nil {
				logger.WithFields(logger.Fields{"token": t.Address, "err": err}).
					Warn("[alt-fun] skipping pool")
				continue
			}
			newPools = append(newPools, p)
		}

		if reachedCheckpoint || len(tokens) < limit {
			break
		}
		offset += len(tokens)
	}

	// Update checkpoint to the newest createdAt we observed.
	if newestCreatedAt != "" {
		meta.LastCreatedAt = newestCreatedAt
	}
	newMetaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.WithFields(logger.Fields{
		"dexID":      u.config.DexID,
		"new_pools":  len(newPools),
		"checkpoint": meta.LastCreatedAt,
	}).Info("[alt-fun] finished getting new pools")

	return newPools, newMetaBytes, nil
}

func (u *PoolsListUpdater) fetchPage(ctx context.Context, limit, offset int) ([]apiToken, error) {
	var result apiResponse
	resp, err := u.httpClient.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"limit":  fmt.Sprintf("%d", limit),
			"offset": fmt.Sprintf("%d", offset),
		}).
		SetResult(&result).
		Get("")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("alt-fun API returned HTTP %d", resp.StatusCode())
	}
	return result.Data, nil
}

// fetchZapParams fetches Zap fees, USDC address and graduation threshold in one multicall.
func (u *PoolsListUpdater) fetchZapParams(ctx context.Context) (
	usdc string, buyFeeBps, sellFeeBps uint64, graduationThresholdUsd *big.Int, err error,
) {
	var (
		buyFee     = new(big.Int)
		sellFee    = new(big.Int)
		baseAsset  common.Address
		gradThresh = new(big.Int)
	)

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    zapABI,
		Target: u.config.ZapAddress,
		Method: "buyFeeBps",
	}, []any{&buyFee}).
		AddCall(&ethrpc.Call{
			ABI:    zapABI,
			Target: u.config.ZapAddress,
			Method: "sellFeeBps",
		}, []any{&sellFee}).
		AddCall(&ethrpc.Call{
			ABI:    globalStorageABI,
			Target: u.config.GlobalStorageAddress,
			Method: "baseAsset",
		}, []any{&baseAsset}).
		AddCall(&ethrpc.Call{
			ABI:    bondingABI,
			Target: u.config.BondingAddress,
			Method: "graduationThresholdUsd",
		}, []any{&gradThresh})

	if _, err = req.Aggregate(); err != nil {
		return
	}
	buyFeeBps = buyFee.Uint64()
	sellFeeBps = sellFee.Uint64()
	usdc = baseAsset.Hex()
	graduationThresholdUsd = gradThresh
	return
}

func (u *PoolsListUpdater) newPool(
	t apiToken, usdcAddr string, buyFeeBps, sellFeeBps uint64, graduationThresholdUsd *big.Int,
) (entity.Pool, error) {
	var gradThreshU *uint256.Int
	if graduationThresholdUsd != nil {
		gradThreshU = uint256.MustFromBig(graduationThresholdUsd)
	}
	ltAddr := strings.ToLower(t.LTPair)
	staticExtra := StaticExtra{
		PairAddress:            t.BondingPair,
		LTAddress:              ltAddr,
		USDC:                   usdcAddr,
		ZapAddress:             u.config.ZapAddress,
		BuyFeeBps:              buyFeeBps,
		SellFeeBps:             sellFeeBps,
		BasePool:               ltAddr,
		GraduationThresholdUsd: gradThreshU,
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return entity.Pool{}, err
	}

	return entity.Pool{
		Address:   t.Address,
		Exchange:  u.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: usdcAddr, Swappable: true},
			{Address: t.Address, Swappable: true},
		},
		StaticExtra: string(staticExtraBytes),
		Extra:       "{}",
	}, nil
}
