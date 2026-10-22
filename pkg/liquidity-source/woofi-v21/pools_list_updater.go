package woofiv21

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexTypeWooFiV21, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:         cfg,
		ethrpcClient:   ethrpcClient,
		hasInitialized: false,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if d.hasInitialized {
		return nil, nil, nil
	}

	pools, err := d.init(ctx)
	if err != nil {
		return nil, nil, err
	}

	d.hasInitialized = true

	return pools, nil, nil
}

func (d *PoolsListUpdater) init(ctx context.Context) ([]entity.Pool, error) {
	tokens, err := loadTokens(ctx, d.ethrpcClient, d.config, nil)
	if err != nil {
		return nil, err
	}

	reserves := make([]string, len(tokens))
	for i := range reserves {
		reserves[i] = "0"
	}

	var newPool = entity.Pool{
		Address:   strings.ToLower(d.config.WooPPV2Address),
		Exchange:  d.config.DexID,
		Type:      DexTypeWooFiV21,
		Timestamp: time.Now().Unix(),
		Reserves:  reserves,
		Tokens:    tokens,
	}

	logger.Infof("[%s] got pool %v from config", d.config.DexID, newPool.Address)

	return []entity.Pool{newPool}, nil
}

// loadTokens reads the pool's base tokens and quote token (in that order). Tokens already in
// old are reused as is; decimals are fetched only for the others.
func loadTokens(ctx context.Context, client *ethrpc.Client, cfg *Config,
	old []*entity.PoolToken) ([]*entity.PoolToken, error) {
	var (
		quoteToken common.Address
		baseTokens = make([]common.Address, 0)
	)

	calls := client.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    IntegrationHelperABI,
		Target: cfg.IntegrationHelperAddress,
		Method: integrationHelperMethodAllBaseTokens,
	}, []any{&baseTokens})
	calls.AddCall(&ethrpc.Call{
		ABI:    WooPPV2ABI,
		Target: cfg.WooPPV2Address,
		Method: wooPPV2MethodQuoteToken,
	}, []any{&quoteToken})

	if _, err := calls.Aggregate(); err != nil {
		logger.Errorf("failed to aggregate call with error %v", err)
		return nil, err
	}

	supportedToken := append(baseTokens, quoteToken)
	oldByAddr := make(map[string]*entity.PoolToken, len(old))
	for _, t := range old {
		oldByAddr[t.Address] = t
	}

	var (
		tokens      = make([]*entity.PoolToken, len(supportedToken))
		decimalsIdx []int
	)
	for i, tokenAddress := range supportedToken {
		addr := hexutil.Encode(tokenAddress[:])
		if t := oldByAddr[addr]; t != nil {
			tokens[i] = t
			continue
		}
		tokens[i] = &entity.PoolToken{Address: addr, Swappable: true}
		decimalsIdx = append(decimalsIdx, i)
	}

	if len(decimalsIdx) == 0 {
		return tokens, nil
	}
	decimalCalls := client.NewRequest().SetContext(ctx)
	for _, i := range decimalsIdx {
		decimalCalls.AddCall(&ethrpc.Call{
			ABI:    Erc20ABI,
			Target: supportedToken[i].Hex(),
			Method: erc20MethodDecimals,
		}, []any{&tokens[i].Decimals})
	}
	if _, err := decimalCalls.Aggregate(); err != nil {
		logger.Errorf("failed to aggregate decimalCalls with error %v", err)
		return nil, err
	}

	return tokens, nil
}
