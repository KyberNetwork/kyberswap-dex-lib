package syncswapv2aqua

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/syncswapv2"
	syncswapv2shared "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/syncswapv2/shared"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	syncswapv2shared.PoolsListUpdater
}

var _ = poollist.RegisterFactoryCE(DexTypeSyncSwapV2Aqua, NewPoolsListUpdater)

func NewPoolsListUpdater(
	config *syncswapv2.Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		PoolsListUpdater: syncswapv2shared.PoolsListUpdater{
			Config:       config,
			EthrpcClient: ethrpcClient,
		},
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	return d.GetPools(ctx, metadataBytes, d.processBatch)
}

func (d *PoolsListUpdater) processBatch(ctx context.Context, poolAddresses []common.Address, masterAddresses []string) ([]entity.Pool, error) {
	var (
		poolTypes   = make([]uint16, len(poolAddresses))
		assets      = make([][2]common.Address, len(poolAddresses))
		feeManagers = make([]common.Address, len(poolAddresses))
	)

	calls := d.PoolsListUpdater.EthrpcClient.NewRequest().SetContext(ctx)
	for i := 0; i < len(poolAddresses); i++ {
		calls.AddCall(&ethrpc.Call{
			ABI:    aquaPoolABI,
			Target: poolAddresses[i].Hex(),
			Method: poolMethodPoolType,
			Params: nil,
		}, []interface{}{&poolTypes[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    aquaPoolABI,
			Target: poolAddresses[i].Hex(),
			Method: poolMethodGetAssets,
			Params: nil,
		}, []interface{}{&assets[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    masterABI,
			Target: masterAddresses[i],
			Method: poolMethodGetFeeManager,
			Params: nil,
		}, []interface{}{&feeManagers[i]})
	}
	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get pool type and assets")

		return nil, err
	}

	var pools = make([]entity.Pool, 0, len(poolAddresses))
	for i := 0; i < len(poolAddresses); i++ {
		extra := ""
		poolAddress := strings.ToLower(poolAddresses[i].Hex())
		token0Address := strings.ToLower(assets[i][0].Hex())
		token1Address := strings.ToLower(assets[i][1].Hex())
		if int(poolTypes[i]) != poolTypeSyncSwapV2AquaInContract {
			continue
		}
		temp, err := json.Marshal(ExtraAquaPool{
			FeeManagerAddress: feeManagers[i].Hex(),
			MasterAddress:     masterAddresses[i],
		})
		if err != nil {
			return nil, err
		}
		extra = string(temp)

		var token0 = entity.PoolToken{
			Address:   token0Address,
			Weight:    defaultTokenWeight,
			Swappable: true,
		}
		var token1 = entity.PoolToken{
			Address:   token1Address,
			Weight:    defaultTokenWeight,
			Swappable: true,
		}

		newPool := entity.Pool{
			Address:   poolAddress,
			Exchange:  d.PoolsListUpdater.Config.DexID,
			Type:      PoolTypeSyncSwapV2Aqua,
			Timestamp: time.Now().Unix(),
			Reserves:  entity.PoolReserves{reserveZero, reserveZero},
			Tokens:    []*entity.PoolToken{&token0, &token1},
			Extra:     extra,
		}
		pools = append(pools, newPool)
	}

	return pools, nil
}
