package llamma

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
		logger       logger.Logger
	}

	PoolsListUpdaterMetadata struct {
		Offset int `json:"offset"`
	}
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
		logger: logger.WithFields(logger.Fields{
			"dexId":   config.DexID,
			"dexType": DexType,
		}),
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	u.logger.Infof("Start updating pools list ...")
	defer func() {
		u.logger.Infof("Finish updating pools list.")
	}()

	nCollaterals, err := u.nCollaterals(ctx)
	if err != nil {
		u.logger.Errorf("failed to get n collaterals %v", err)
		return nil, metadataBytes, err
	}

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		u.logger.Errorf("failed to get offset %v", err)
		return nil, metadataBytes, err
	}

	batchSize := u.getBatchSize(nCollaterals, u.config.NewPoolLimit, offset)
	pools, err := u.getPools(ctx, offset, batchSize)
	if err != nil {
		u.logger.Errorf("failed to get pools %v", err)
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(offset + batchSize)
	if err != nil {
		u.logger.Errorf("failed to create new metadata %v", err)
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getPools(ctx context.Context, offset int, batchSize int) ([]entity.Pool, error) {
	stableCoin := strings.ToLower(u.config.StableCoin)

	amms := make([]common.Address, batchSize)
	collaterals := make([]common.Address, batchSize)
	factoryCalls := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := 0; i < batchSize; i++ {
		idx := big.NewInt(int64(offset + i))
		factoryCalls.AddCall(&ethrpc.Call{
			ABI:    curveControllerFactoryABI,
			Target: u.config.FactoryAddress,
			Method: factoryMethodAmms,
			Params: []interface{}{idx},
		}, []interface{}{&amms[i]})
		factoryCalls.AddCall(&ethrpc.Call{
			ABI:    curveControllerFactoryABI,
			Target: u.config.FactoryAddress,
			Method: factoryMethodCollaterals,
			Params: []interface{}{idx},
		}, []interface{}{&collaterals[i]})
	}
	if _, err := factoryCalls.Aggregate(); err != nil {
		return nil, err
	}

	var (
		decimals      = make([]uint8, batchSize+1)
		aCoefficients = make([]*big.Int, batchSize)
	)

	ammCalls := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := 0; i < batchSize; i++ {
		ammCalls.AddCall(&ethrpc.Call{
			ABI:    shared.ERC20ABI,
			Target: collaterals[i].String(),
			Method: shared.ERC20MethodDecimals,
		}, []interface{}{&decimals[i]})
		ammCalls.AddCall(&ethrpc.Call{
			ABI:    curveLlammaABI,
			Target: amms[i].String(),
			Method: llammaMethodA,
		}, []interface{}{&aCoefficients[i]})
	}
	ammCalls.AddCall(&ethrpc.Call{
		ABI:    shared.ERC20ABI,
		Target: u.config.StableCoin,
		Method: shared.ERC20MethodDecimals,
	}, []interface{}{&decimals[batchSize]})
	if _, err := ammCalls.Aggregate(); err != nil {
		return nil, err
	}

	var pools = make([]entity.Pool, 0, batchSize)
	for i, amm := range amms {
		var staticExtra = StaticExtra{
			A: uint256.MustFromBig(aCoefficients[i]),
		}

		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			u.logger.Errorf("failed to marshal static extra data")
			return nil, err
		}

		newPool := entity.Pool{
			Address:   strings.ToLower(amm.String()),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{
					Address:   stableCoin,
					Decimals:  decimals[batchSize],
					Swappable: true,
				},
				{
					Address:   strings.ToLower(collaterals[i].String()),
					Decimals:  decimals[i],
					Swappable: true,
				},
			},
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}

func (u *PoolsListUpdater) nCollaterals(ctx context.Context) (int, error) {
	var nCollaterals *big.Int
	calls := u.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    curveControllerFactoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryMethodNCollaterals,
	}, []interface{}{&nCollaterals})
	if _, err := calls.TryAggregate(); err != nil {
		return 0, err
	}

	return int(nCollaterals.Int64()), nil
}

func (u *PoolsListUpdater) getOffset(metadataBytes []byte) (int, error) {
	if len(metadataBytes) == 0 {
		return 0, nil
	}

	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return 0, err
	}

	return metadata.Offset, nil
}

func (u *PoolsListUpdater) getBatchSize(length int, limit int, offset int) int {
	if offset == length {
		return 0
	}

	if offset+limit >= length {
		if offset > length {
			u.logger.Warn("offset is greater than length")
		}
		return max(length-offset, 0)
	}

	return limit
}

func (u *PoolsListUpdater) newMetadata(newOffset int) ([]byte, error) {
	metadataBytes, err := json.Marshal(PoolsListUpdaterMetadata{Offset: newOffset})
	if err != nil {
		return nil, err
	}

	return metadataBytes, nil
}
