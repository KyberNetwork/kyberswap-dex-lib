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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
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

	psmConfigs, err := d.initializeDexConfig()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not initialize dex config")
		return nil, nil, err
	}

	pools := make([]entity.Pool, 0, len(d.cfg.ConfigPath))
	for _, psmCfg := range psmConfigs {
		newPool, err := d.newPool(ctx, psmCfg)
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

func (d *PoolsListUpdater) newPool(ctx context.Context, psmCfg PSMConfig) (entity.Pool, error) {
	var (
		gemPocket                           common.Address
		debtTokenDecimals, gemTokenDecimals uint8
	)

	req := d.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: psmCfg.DebtToken,
			Method: abi.Erc20DecimalsMethod,
		}, []any{&debtTokenDecimals}).
		AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: psmCfg.GemToken,
			Method: abi.Erc20DecimalsMethod,
			Params: nil,
		}, []any{&gemTokenDecimals}).
		AddCall(&ethrpc.Call{
			ABI:    LitePSMABI,
			Target: psmCfg.PoolAddress,
			Method: litePSMMethodPocket,
		}, []any{&gemPocket})

	_, err := req.Aggregate()
	if err != nil {
		return entity.Pool{}, err
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{
		Pocket: gemPocket,
		Psm:    common.HexToAddress(psmCfg.PsmAddress),
		Dai:    common.HexToAddress(psmCfg.DaiToken),
	})
	if err != nil {
		panic(err)
	}

	return entity.Pool{
		Address:  strings.ToLower(psmCfg.PoolAddress),
		Exchange: d.cfg.DexID,
		Type:     DexTypeLitePSM,
		Tokens: []*entity.PoolToken{
			{
				Address:   strings.ToLower(psmCfg.DebtToken),
				Decimals:  debtTokenDecimals,
				Swappable: true,
			},
			{
				Address:   strings.ToLower(psmCfg.GemToken),
				Decimals:  gemTokenDecimals,
				Swappable: true,
			},
		},
		Reserves:    entity.PoolReserves{"0", "0"},
		Timestamp:   time.Now().Unix(),
		StaticExtra: string(staticExtraBytes),
	}, nil
}

func (d *PoolsListUpdater) initializeDexConfig() ([]PSMConfig, error) {
	dexConfigBytes, ok := bytesByPath[d.cfg.ConfigPath]
	if !ok {
		err := fmt.Errorf("key %s not found", d.cfg.ConfigPath)
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not find dex config")
		return nil, err
	}

	var psmConfigs []PSMConfig
	err := json.Unmarshal(dexConfigBytes, &psmConfigs)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not unmarshal dex config")
		return nil, err
	}

	return psmConfigs, nil
}
