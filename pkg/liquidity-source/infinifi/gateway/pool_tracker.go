package gateway

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type bucket struct {
	Index       uint32     `json:"index"`
	TotalSupply *big.Int   `json:"totalSupply"`
	BucketData  bucketData `json:"bucketData"`
}
type bucketData struct {
	ShareToken         common.Address `json:"shareToken"`
	TotalReceiptTokens *big.Int       `json:"totalReceiptTokens"`
	Multiplier         *big.Int       `json:"multiplier"`
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ pool.GetNewPoolStateParams) (entity.Pool, error) {
	return getPoolState(ctx, t.ethrpcClient, t.config, &p)
}

// getPoolState fetches all necessary state from the blockchain
func getPoolState(ctx context.Context, ethrpcClient *ethrpc.Client, cfg *Config, p *entity.Pool) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"gateway": p.Address,
	}).Infof("fetching new pool state")
	var extra Extra
	err := json.Unmarshal([]byte(p.Extra), &extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to unmarshal extra data")
		return entity.Pool{}, err
	}

	var (
		liusdBuckets     []bucket
		isPaused         bool
		iusdSupply       *big.Int
		siusdTotalAssets *big.Int
		siusdSupply      *big.Int
		enabledBuckets   []uint32
	)

	// Build batched RPC request
	req := ethrpcClient.NewRequest().SetContext(ctx)

	// Check if gateway is paused
	req.AddCall(&ethrpc.Call{
		ABI:    gatewayABI,
		Target: cfg.Gateway,
		Method: coreControlledPausedMethod,
	}, []any{&isPaused})

	// Get iUSD total supply
	req.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: cfg.IUSD,
		Method: erc20TotalSupplyMethod,
	}, []any{&iusdSupply})

	// Get siUSD vault info (ERC4626)
	req.AddCall(&ethrpc.Call{
		ABI:    erc4626ABI,
		Target: cfg.SIUSD,
		Method: erc4626TotalAssetsMethod,
	}, []any{&siusdTotalAssets})

	req.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: cfg.SIUSD,
		Method: erc20TotalSupplyMethod,
	}, []any{&siusdSupply})

	req.AddCall(&ethrpc.Call{
		ABI:    lockingControllerABI,
		Target: cfg.LockingController,
		Method: "getEnabledBuckets",
	}, []any{&enabledBuckets})
	if len(extra.LIUSDBuckets) > 0 {
		// Prepare slices for liUSD bucket data
		liusdBuckets = extra.LIUSDBuckets
		ReqLiusdSupplies(req, cfg, liusdBuckets)
		ReqLiusdBuckets(req, cfg, liusdBuckets)
	}

	// Execute batched call
	resp, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"gateway": cfg.Gateway,
			"error":   err,
		}).Errorf("failed to aggregate RPC calls")
		return entity.Pool{}, err
	}

	currentBuckets := lo.Map(extra.LIUSDBuckets, func(bucket bucket, _ int) uint32 { return bucket.Index })
	if len(enabledBuckets) != len(currentBuckets) || !lo.Every(enabledBuckets, currentBuckets) {
		newBucketIndexes, _ := lo.Difference(enabledBuckets, currentBuckets)
		newLiusdBuckets := lo.Map(newBucketIndexes, func(index uint32, _ int) bucket { return bucket{Index: index} })
		req := ethrpcClient.NewRequest().SetContext(ctx)
		ReqLiusdBuckets(req, cfg, newLiusdBuckets)
		if _, err := req.Aggregate(); err != nil {
			return entity.Pool{}, err
		}
		req = ethrpcClient.NewRequest().SetContext(ctx)
		ReqLiusdSupplies(req, cfg, newLiusdBuckets)
		if _, err := req.Aggregate(); err != nil {
			return entity.Pool{}, err
		}
		p.Tokens = append(p.Tokens, lo.Map(newLiusdBuckets, func(bucket bucket, _ int) *entity.PoolToken {
			return &entity.PoolToken{
				Address:   strings.ToLower(bucket.BucketData.ShareToken.Hex()),
				Swappable: true,
			}
		})...)
		liusdBuckets = append(liusdBuckets, newLiusdBuckets...)
	}

	// Update block number
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}

	extra.IsPaused = isPaused
	extra.IUSDSupply = iusdSupply
	extra.SIUSDTotalAssets = siusdTotalAssets
	extra.SIUSDSupply = siusdSupply
	extra.LIUSDBuckets = liusdBuckets

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)

	// Update reserves (for display/informational purposes)
	// Format: [siUSD total assets, siUSD shares, liUSD1 supply, liUSD2 supply, ...]
	reserves := []string{
		defaultReserves,
		siusdTotalAssets.String(),
		siusdSupply.String(),
	}
	reserves = append(reserves, lo.Map(liusdBuckets, func(bucket bucket, _ int) string {
		return bucket.TotalSupply.String()
	})...)
	p.Reserves = reserves

	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"gateway":     p.Address,
		"blockNumber": p.BlockNumber,
		"paused":      isPaused,
	}).Infof("successfully fetched pool state")

	return *p, nil
}

func ReqLiusdBuckets(req *ethrpc.Request, cfg *Config, liusdBuckets []bucket) {
	for i, bucket := range liusdBuckets {
		req.AddCall(&ethrpc.Call{
			ABI:    lockingControllerABI,
			Target: cfg.LockingController,
			Method: lockingControllerBucketsMethod,
			Params: []any{bucket.Index},
		}, []any{&liusdBuckets[i].BucketData})
	}
}

func ReqLiusdSupplies(req *ethrpc.Request, cfg *Config, liusdBuckets []bucket) {
	for i, bucket := range liusdBuckets {
		req.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: bucket.BucketData.ShareToken.Hex(),
			Method: erc20TotalSupplyMethod,
		}, []any{&liusdBuckets[i].TotalSupply})
	}
}
