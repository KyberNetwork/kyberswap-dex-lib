package service

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/config"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/redis"
)

const gasPriceJobIntervalSec = 2
const GasPriceKey = "GAS_PRICE"
const ConfigKey = "configs"

type CommonService struct {
	db         *redis.Redis
	config     *config.Common
	rpcService IRPCService
}

func NewCommon(db *redis.Redis, config *config.Common, rpcService IRPCService) *CommonService {
	var ret = CommonService{
		db,
		config,
		rpcService,
	}
	return &ret
}

func (t *CommonService) UpdateData(ctx context.Context) {
	go t.updateGasPriceJob(ctx)
}

func (t *CommonService) updateGasPriceJob(ctx context.Context) {
	run := func() {
		logger.Debugf("start updating ...")
		defaultRPC, err := ethclient.Dial(t.config.PublicRPC)
		if err != nil {
			logger.Errorf("failed to load default rpc")
			return
		}
		gasPrice, err := defaultRPC.SuggestGasPrice(ctx)
		if err != nil {
			logger.Errorf("failed to get gas price: %v", err)
			return
		}
		t.db.Client.HSet(ctx, t.db.FormatKey(ConfigKey), GasPriceKey, gasPrice.String())
		logger.Infof("gas price: %v", gasPrice.String())
	}
	for {
		run()
		time.Sleep(gasPriceJobIntervalSec * time.Second)
	}
}
