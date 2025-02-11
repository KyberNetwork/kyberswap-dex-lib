package litepsm

import (
	"context"
	"fmt"
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
	cfg            *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexTypeLitePSM, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{"dexID": d.cfg.DexID}).Info("get new pools")

	if d.hasInitialized {
		return nil, nil, nil
	}

	err := d.initializeDexConfig()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not initialize dex config")
		return nil, nil, err
	}

	pools := make([]entity.Pool, 0, len(d.cfg.DexConfig.PSMs))
	for _, psmCfg := range d.cfg.DexConfig.PSMs {
		newPool, err := d.newPool(psmCfg)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexID": d.cfg.DexID,
				"error": err,
			}).Error("can not create new pool")
			return nil, nil, err
		}

		pools = append(pools, newPool)
	}

	logger.WithFields(logger.Fields{"dexID": d.cfg.DexID}).Info("get new pools successfully")

	d.hasInitialized = true

	return pools, nil, nil
}

func (d *PoolsListUpdater) newPool(psmCfg PSMConfig) (entity.Pool, error) {
	token0 := &entity.PoolToken{
		Address:   d.cfg.DexConfig.Dai.Address,
		Decimals:  d.cfg.DexConfig.Dai.Decimals,
		Swappable: true,
	}

	token1 := &entity.PoolToken{
		Address:   psmCfg.Gem.Address,
		Decimals:  psmCfg.Gem.Decimals,
		Swappable: true,
	}

	var gemPocket common.Address
	req := d.ethrpcClient.
		NewRequest().
		AddCall(&ethrpc.Call{
			ABI:    litePSMABI,
			Target: psmCfg.Address,
			Method: litePSMMethodPocket,
			Params: nil,
		}, []interface{}{&gemPocket})

	_, err := req.Call()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": DexTypeLitePSM,
			"error": err,
		}).Error("[getReserves] eth rpc call error")
		return entity.Pool{}, err
	}

	staticExtra := StaticExtra{
		Gem:    psmCfg.Gem,
		Pocket: gemPocket,
	}

	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		panic(err)
	}

	return entity.Pool{
		Address:     strings.ToLower(psmCfg.Address),
		Exchange:    d.cfg.DexID,
		Type:        DexTypeLitePSM,
		Tokens:      []*entity.PoolToken{token0, token1},
		Reserves:    entity.PoolReserves{"0", "0"},
		Timestamp:   time.Now().Unix(),
		StaticExtra: string(staticExtraBytes),
	}, nil
}

func (d *PoolsListUpdater) initializeDexConfig() error {
	dexConfigBytes, ok := bytesByPath[d.cfg.ConfigPath]
	if !ok {
		err := fmt.Errorf("key %s not found", d.cfg.ConfigPath)
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not find dex config")
		return err
	}

	err := json.Unmarshal(dexConfigBytes, &d.cfg.DexConfig)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not unmarshal dex config")
		return err
	}

	return nil
}
