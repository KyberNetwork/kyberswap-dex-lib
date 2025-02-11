package bancorv3

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config       Config
	ethrpcClient *ethrpc.Client
	initialized  bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       *config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{
		"dexId":   u.config.DexID,
		"dexType": DexType,
	}).Infof("Start updating pools list ...")
	defer func() {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Infof("Finish updating pools list.")
	}()

	if u.initialized {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Infof("Pools have been initialized.")
		return nil, metadataBytes, nil
	}

	tokenAddresses, err := u.getTokenAddresses(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Error(err.Error())
		return nil, metadataBytes, err
	}

	var (
		poolTokens = make([]*entity.PoolToken, 0, len(tokenAddresses))
		reserves   = make([]string, 0, len(tokenAddresses))
	)

	for _, tokenAddress := range tokenAddresses {
		addr := valueobject.WrapNativeLower(tokenAddress, u.config.ChainID)
		poolTokens = append(poolTokens, &entity.PoolToken{Address: addr, Swappable: true})
		reserves = append(reserves, "0")
	}

	staticExtra := StaticExtra{
		BNT:     strings.ToLower(u.config.BNT),
		ChainID: valueobject.ChainID(u.config.ChainID),
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Error(err.Error())
		return nil, metadataBytes, err
	}

	p := entity.Pool{
		Address:     strings.ToLower(u.config.BancorNetwork),
		Exchange:    u.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      poolTokens,
		StaticExtra: string(staticExtraBytes),
	}

	u.initialized = true

	return []entity.Pool{p}, metadataBytes, nil
}

func (u *PoolsListUpdater) getTokenAddresses(ctx context.Context) ([]string, error) {
	var addresses []common.Address
	req := u.ethrpcClient.R().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    bancorNetworkABI,
		Target: u.config.BancorNetwork,
		Method: bancorNetworkMethodLiquidityPools,
	}, []interface{}{&addresses})

	_, err := req.Aggregate()
	if err != nil {
		return nil, err
	}

	ret := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		ret = append(ret, strings.ToLower(addr.Hex()))
	}

	return ret, nil
}
