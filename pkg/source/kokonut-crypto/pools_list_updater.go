package kokonutcrypto

import (
	"context"
	"encoding/json"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strings"
	"time"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(
	ctx context.Context,
	metadataBytes []byte,
) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	// Add timestamp to the context so that each run iteration will have something different
	ctx = util.NewContextWithTimestamp(ctx)

	var lengthBI *big.Int

	getNumPoolsRequest := d.ethrpcClient.NewRequest()
	getNumPoolsRequest.AddCall(&ethrpc.Call{
		ABI:    poolRegistryABI,
		Target: d.config.RegistryAddress,
		Method: registryMethodPoolCount,
		Params: nil,
	}, []interface{}{&lengthBI})

	if _, err := getNumPoolsRequest.Call(); err != nil {
		logger.Errorf("failed to get number of pairs from factory, err: %v", err)
		return nil, metadataBytes, err
	}

	totalNumberOfPools := int(lengthBI.Int64())

	currentOffset := metadata.Offset
	batchSize := d.config.NewPoolLimit
	if currentOffset+batchSize > totalNumberOfPools {
		batchSize = totalNumberOfPools - currentOffset
		if batchSize <= 0 {
			return nil, metadataBytes, nil
		}
	}

	getPairAddressRequest := d.ethrpcClient.NewRequest()

	var pairAddresses = make([]common.Address, batchSize)
	for j := 0; j < batchSize; j++ {
		getPairAddressRequest.AddCall(&ethrpc.Call{
			ABI:    poolRegistryABI,
			Target: d.config.RegistryAddress,
			Method: registryMethodPoolList,
			Params: []interface{}{big.NewInt(int64(currentOffset + j))},
		}, []interface{}{&pairAddresses[j]})
	}
	if _, err := getPairAddressRequest.Aggregate(); err != nil {
		logger.Errorf("failed to process aggregate, err: %v", err)
		return nil, metadataBytes, err
	}

	pools, err := d.processBatch(ctx, pairAddresses)
	if err != nil {
		logger.Errorf("failed to process update new pool, err: %v", err)
		return nil, metadataBytes, err
	}

	numPools := len(pools)

	nextOffset := currentOffset + numPools
	newMetadataBytes, err := json.Marshal(Metadata{
		Offset: nextOffset,
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	if len(pools) > 0 {
		logger.WithFields(logger.Fields{
			"dexID": d.config.DexID,
		}).Infof("scan KokonutRegistry with batch size %v, progress: %d/%d", batchSize, currentOffset+numPools, totalNumberOfPools)
	}

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) processBatch(ctx context.Context, poolAddresses []common.Address) ([]entity.Pool, error) {
	var (
		coins    = make([][2]common.Address, len(poolAddresses))
		decimals = make([][2]uint8, len(poolAddresses))
		lpTokens = make([]common.Address, len(poolAddresses))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	for i, poolAddress := range poolAddresses {
		for j := 0; j < 2; j++ {
			calls.AddCall(&ethrpc.Call{
				ABI:    cryptoSwap2PoolABI,
				Target: poolAddress.Hex(),
				Method: poolMethodCoins,
				Params: []interface{}{big.NewInt(int64(j))},
			}, []interface{}{&coins[i][j]})
		}
	}
	if _, err := calls.Aggregate(); err != nil {
		logger.Errorf("failed to aggregate call to get pool data, err: %v", err)
		return nil, err
	}

	calls = d.ethrpcClient.NewRequest().SetContext(ctx)
	for i, poolAddress := range poolAddresses {
		for j := 0; j < 2; j++ {
			calls.AddCall(&ethrpc.Call{
				ABI:    erc20ABI,
				Target: coins[i][j].Hex(),
				Method: poolMethodDecimals,
				Params: nil,
			}, []interface{}{&decimals[i][j]})
		}
		calls.AddCall(&ethrpc.Call{
			ABI:    cryptoSwap2PoolABI,
			Target: poolAddress.Hex(),
			Method: poolMethodToken,
			Params: nil,
		}, []interface{}{&lpTokens[i]})
	}
	if _, err := calls.Aggregate(); err != nil {
		logger.Errorf("failed to aggregate call to get coin data, err: %v", err)
		return nil, err
	}

	var pools = make([]entity.Pool, len(poolAddresses))
	for i := range poolAddresses {
		var reserves entity.PoolReserves
		var tokens []*entity.PoolToken
		var staticExtra = StaticExtra{
			LpToken: strings.ToLower(lpTokens[i].Hex()),
		}
		for j := range coins[i] {
			precision := new(big.Int).Exp(big.NewInt(10), new(big.Int).Sub(big.NewInt(18), big.NewInt(int64(decimals[i][j]))), nil)
			staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, precision.String())
			reserves = append(reserves, zeroString)
			tokens = append(tokens, &entity.PoolToken{
				Address:   strings.ToLower(coins[i][j].Hex()),
				Decimals:  decimals[i][j],
				Weight:    defaultWeight,
				Swappable: true,
			})
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.Errorf("failed to marshal static extra data, err: %v", err)
			return nil, err
		}

		pools[i] = entity.Pool{
			Address:     strings.ToLower(poolAddresses[i].Hex()),
			Exchange:    d.config.DexID,
			Type:        DexTypeKokonutCrypto,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
		}
	}

	return pools, nil
}
