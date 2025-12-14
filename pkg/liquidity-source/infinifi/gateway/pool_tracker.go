package gateway

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
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

	var (
		isPaused         bool
		iusdSupply       *big.Int
		siusdTotalAssets *big.Int
		siusdSupply      *big.Int
	)

	// Prepare slice for liUSD supplies
	liusdSupplies := make([]*big.Int, len(cfg.LIUSDTokens))

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

	// Get total supply for each liUSD token
	for i, liusd := range cfg.LIUSDTokens {
		liusdSupplies[i] = new(big.Int)
		req.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: liusd.Address,
			Method: erc20TotalSupplyMethod,
		}, []any{&liusdSupplies[i]})
	}

	// Execute batched call
	resp, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"gateway": cfg.Gateway,
			"error":   err,
		}).Errorf("failed to aggregate RPC calls")
		return *p, err
	}

	// Update block number
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}

	// Convert liUSD supplies to strings
	liusdSupplyStrings := make([]string, len(liusdSupplies))
	for i, supply := range liusdSupplies {
		liusdSupplyStrings[i] = supply.String()
	}

	// Marshal extra data
	extra := Extra{
		IsPaused:         isPaused,
		IUSDSupply:       iusdSupply,
		SIUSDTotalAssets: siusdTotalAssets,
		SIUSDSupply:      siusdSupply,
		LIUSDSupplies:    liusdSupplyStrings,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to marshal extra data")
		return *p, err
	}

	p.Extra = string(extraBytes)

	// Update reserves (for display/informational purposes)
	// Format: [iUSD supply, siUSD total assets, siUSD shares, liUSD1 supply, liUSD2 supply, ...]
	reserves := []string{
		iusdSupply.String(),
		siusdTotalAssets.String(),
		siusdSupply.String(),
	}
	reserves = append(reserves, liusdSupplyStrings...)
	p.Reserves = reserves

	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"gateway":     p.Address,
		"blockNumber": p.BlockNumber,
		"paused":      isPaused,
	}).Infof("successfully fetched pool state")

	return *p, nil
}
