package woofiv21

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

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
	var (
		quoteToken common.Address
		baseTokens = make([]common.Address, 0)
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    IntegrationHelperABI,
		Target: d.config.IntegrationHelperAddress,
		Method: integrationHelperMethodAllBaseTokens,
		Params: nil,
	}, []interface{}{&baseTokens})
	calls.AddCall(&ethrpc.Call{
		ABI:    WooPPV2ABI,
		Target: d.config.WooPPV2Address,
		Method: wooPPV2MethodQuoteToken,
		Params: nil,
	}, []interface{}{&quoteToken})

	if _, err := calls.Aggregate(); err != nil {
		logger.Errorf("failed to aggregate call with error %v", err)
		return nil, err
	}

	supportedToken := append(baseTokens, quoteToken)

	tokenDecimals := make([]uint8, len(supportedToken))
	decimalCalls := d.ethrpcClient.NewRequest().SetContext(ctx)
	for i, token := range supportedToken {
		decimalCalls.AddCall(&ethrpc.Call{
			ABI:    Erc20ABI,
			Target: token.Hex(),
			Method: erc20MethodDecimals,
			Params: nil,
		}, []interface{}{&tokenDecimals[i]})
	}
	if _, err := decimalCalls.Aggregate(); err != nil {
		logger.Errorf("failed to aggregate decimalCalls with error %v", err)
		return nil, err
	}

	var (
		tokens   = make([]*entity.PoolToken, len(supportedToken))
		reserves = make([]string, len(supportedToken))
	)
	for i, tokenAddress := range supportedToken {
		tokens[i] = &entity.PoolToken{
			Address:   strings.ToLower(tokenAddress.Hex()),
			Decimals:  tokenDecimals[i],
			Weight:    defaultWeight,
			Swappable: true,
		}
		reserves[i] = zeroString
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
