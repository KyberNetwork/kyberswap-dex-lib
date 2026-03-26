package canonic

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	initialized  bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{"dex_id": u.config.DexId}).Info("started getting new pools")

	if u.initialized {
		return nil, nil, nil
	}

	pools := make([]entity.Pool, 0, len(u.config.Pools))

	for _, maobAddr := range u.config.Pools {
		p, err := u.fetchPool(ctx, maobAddr)
		if err != nil {
			return nil, nil, err
		}
		pools = append(pools, p)
	}

	u.initialized = true

	logger.WithFields(logger.Fields{
		"dex_id":     u.config.DexId,
		"pool_count": len(pools),
	}).Info("finished getting new pools")

	return pools, nil, nil
}

func (u *PoolsListUpdater) fetchPool(ctx context.Context, maobAddr string) (entity.Pool, error) {
	var (
		baseTokenAddr   common.Address
		quoteTokenAddr  common.Address
		baseDecimalsBI  *big.Int
		quoteDecimalsBI *big.Int
		baseScale       *big.Int
		quoteScale      *big.Int
	)

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    maobABI,
		Target: maobAddr,
		Method: maobMethodBaseToken,
	}, []any{&baseTokenAddr})
	req.AddCall(&ethrpc.Call{
		ABI:    maobABI,
		Target: maobAddr,
		Method: maobMethodQuoteToken,
	}, []any{&quoteTokenAddr})
	req.AddCall(&ethrpc.Call{
		ABI:    maobABI,
		Target: maobAddr,
		Method: maobMethodBaseDecimals,
	}, []any{&baseDecimalsBI})
	req.AddCall(&ethrpc.Call{
		ABI:    maobABI,
		Target: maobAddr,
		Method: maobMethodQuoteDecimals,
	}, []any{&quoteDecimalsBI})
	req.AddCall(&ethrpc.Call{
		ABI:    maobABI,
		Target: maobAddr,
		Method: maobMethodBaseScale,
	}, []any{&baseScale})
	req.AddCall(&ethrpc.Call{
		ABI:    maobABI,
		Target: maobAddr,
		Method: maobMethodQuoteScale,
	}, []any{&quoteScale})

	if _, err := req.Aggregate(); err != nil {
		return entity.Pool{}, err
	}

	baseToken := strings.ToLower(baseTokenAddr.Hex())
	quoteToken := strings.ToLower(quoteTokenAddr.Hex())
	baseDecimals := uint8(baseDecimalsBI.Uint64())
	quoteDecimals := uint8(quoteDecimalsBI.Uint64())

	staticExtra := StaticExtra{
		BaseToken:     baseToken,
		QuoteToken:    quoteToken,
		BaseDecimals:  baseDecimals,
		QuoteDecimals: quoteDecimals,
		BaseScale:     baseScale.String(),
		QuoteScale:    quoteScale.String(),
	}

	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return entity.Pool{}, err
	}

	return entity.Pool{
		Address:     maobAddr,
		Exchange:    u.config.DexId,
		Type:        DexType,
		Reserves:    entity.PoolReserves{"0", "0"},
		StaticExtra: string(staticExtraBytes),
		Tokens: []*entity.PoolToken{
			{Address: baseToken, Decimals: baseDecimals, Swappable: true},
			{Address: quoteToken, Decimals: quoteDecimals, Swappable: true},
		},
		Timestamp: time.Now().Unix(),
	}, nil
}
