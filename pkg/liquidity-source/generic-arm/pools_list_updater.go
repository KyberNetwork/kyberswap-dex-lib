package genericarm

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
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

func (d *PoolsListUpdater) GetNewPools(_ context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if d.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	calls := d.ethrpcClient.NewRequest().SetContext(context.Background())
	var token0, token1 common.Address
	calls.AddCall(&ethrpc.Call{
		ABI:    lidoArmABI,
		Target: d.config.ArmAddress,
		Method: "token0",
	}, []interface{}{&token0}).
		AddCall(&ethrpc.Call{
			ABI:    lidoArmABI,
			Target: d.config.ArmAddress,
			Method: "token1",
		}, []interface{}{&token1})
	_, err := calls.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to initPool")
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(Extra{
		SwapType: d.config.SwapType,
		ArmType:  d.config.ArmType,
	})

	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to marshal extra")
		return nil, nil, err
	}

	pools := []entity.Pool{
		{
			Address:   d.config.ArmAddress,
			Exchange:  d.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(token0.Hex()),
					Swappable: true,
				},
				{
					Address:   strings.ToLower(token1.Hex()),
					Swappable: true,
				},
			},
			Extra: string(extraBytes),
		},
	}
	logger.WithFields(logger.Fields{"pool": pools}).Info("finish fetching pools")
	d.hasInitialized = true
	return pools, nil, nil
}
