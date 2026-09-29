package flowstatec1

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	loaded       bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.loaded || len(u.config.Pools) == 0 {
		return nil, metadataBytes, nil
	}

	pools, err := u.buildPools(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": u.config.DexID, "error": err}).
			Error("failed to build flowstate-c1 pools")
		return nil, metadataBytes, err
	}

	u.loaded = true

	logger.WithFields(logger.Fields{"dex_id": u.config.DexID, "pools": len(pools)}).
		Info("loaded flowstate-c1 static pools")

	return pools, metadataBytes, nil
}

// buildPools resolves poolByToken for every configured inventory token, then fans out
// one entity.Pool per (token, quoteAsset) pair, since one on-chain pool can be bought
// with any of the market's approved quote assets. Symbol/decimals are left unset --
// pool-service auto-populates token metadata from its own registry.
func (u *PoolsListUpdater) buildPools(ctx context.Context) ([]entity.Pool, error) {
	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	poolAddrs := make([]common.Address, len(u.config.Pools))

	for i, p := range u.config.Pools {
		req.AddCall(&ethrpc.Call{
			ABI:    marketABI,
			Target: u.config.Market,
			Method: "poolByToken",
			Params: []any{common.HexToAddress(p.Token)},
		}, []any{&poolAddrs[i]})
	}

	if _, err := req.TryBlockAndAggregate(); err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(u.config.Pools)*len(u.config.QuoteAssets))
	for i, p := range u.config.Pools {
		if poolAddrs[i] == (common.Address{}) {
			logger.WithFields(logger.Fields{"token": p.Token}).
				Warn("poolByToken returned zero address, skipping")
			continue
		}

		probeAmount := p.ProbeAmount
		if probeAmount == "" {
			probeAmount = defaultProbeAmount
		}

		for _, quoteAsset := range u.config.QuoteAssets {
			pools = append(pools, u.newPool(poolAddrs[i], p.Token, quoteAsset, probeAmount))
		}
	}

	return pools, nil
}

func (u *PoolsListUpdater) newPool(poolAddr common.Address, token, quoteAsset, probeAmount string) entity.Pool {
	poolAddrLower := strings.ToLower(hexutil.Encode(poolAddr[:]))
	quoteAssetLower := strings.ToLower(quoteAsset)
	tokenLower := strings.ToLower(token)

	staticExtra, _ := json.Marshal(StaticExtra{
		Market:     strings.ToLower(u.config.Market),
		Pool:       poolAddrLower,
		QuoteAsset: quoteAssetLower,
	})

	probe, err := uint256.FromDecimal(probeAmount)
	if err != nil {
		probe = new(uint256.Int)
	}
	extra, _ := json.Marshal(Extra{
		Available:      false,
		ProbeAmount:    probe,
		ProbeQuoteCost: new(uint256.Int),
		FillableAmount: new(uint256.Int),
	})

	return entity.Pool{
		// Synthetic id: the same on-chain pool is priced against every approved quote
		// asset, so the pool contract address alone is not a unique key.
		Address:   pairPoolAddress(poolAddr, common.HexToAddress(quoteAsset)),
		Exchange:  u.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  []string{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: quoteAssetLower, Swappable: true},
			{Address: tokenLower, Swappable: true},
		},
		StaticExtra: string(staticExtra),
		Extra:       string(extra),
	}
}

// pairPoolAddress derives a deterministic, address-shaped id from the real pool and
// the quote asset it's priced against, since one on-chain pool prices against every
// approved quote asset and needs a distinct entity.Pool per pair. XOR rather than a
// hash: collision-free for this scale (one pool times a handful of quote assets),
// no crypto dependency needed.
func pairPoolAddress(pool, quoteAsset common.Address) string {
	var xored common.Address
	for i := range xored {
		xored[i] = pool[i] ^ quoteAsset[i]
	}
	return strings.ToLower(hexutil.Encode(xored[:]))
}
