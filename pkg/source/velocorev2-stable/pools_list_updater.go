package velocorev2stable

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

type PoolsListUpdater struct {
	Config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		Config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (p *PoolsListUpdater) InitPool(_ context.Context) error {
	return nil
}

func (p *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	// Add timestamp to the context so that each run iteration will have something different
	ctx = util.NewContextWithTimestamp(ctx)

	newPoolAddresses, err := p.getNewPoolAddresses(ctx, metadata)
	if err != nil {
		return nil, metadataBytes, err
	}

	if len(newPoolAddresses) == 0 {
		logger.Infof("no new pool")
		return nil, metadataBytes, nil
	}

	if len(newPoolAddresses) > p.Config.NewPoolLimit {
		newPoolAddresses = newPoolAddresses[:p.Config.NewPoolLimit]
	}

	pools, err := p.getPools(ctx, newPoolAddresses)
	if err != nil {
		return nil, metadataBytes, err
	}

	newMetadata := Metadata{
		Offset: metadata.Offset + len(newPoolAddresses),
	}
	newMetadataBytes, err := json.Marshal(newMetadata)
	if err != nil {
		return nil, metadataBytes, err
	}

	if len(pools) > 0 {
		logger.Infof("got %d new pools", len(pools))
	}

	return pools, newMetadataBytes, nil
}

func (p *PoolsListUpdater) getNewPoolAddresses(ctx context.Context, metadata Metadata) ([]common.Address, error) {
	var addresses []common.Address

	req := p.ethrpcClient.R()
	req.AddCall(&ethrpc.Call{
		ABI:    registryABI,
		Target: p.Config.RegistryAddress,
		Method: registryMethodGetPools,
	}, []interface{}{&addresses})

	if _, err := req.Call(); err != nil {
		logger.Errorf("failed to get pool addresses from registry, err: %v", err)
		return nil, err
	}

	var newAddresses []common.Address
	for i := metadata.Offset; i < len(addresses); i++ {
		newAddresses = append(newAddresses, addresses[i])
	}

	return newAddresses, nil
}

func (p *PoolsListUpdater) getPools(ctx context.Context, poolAddreses []common.Address) ([]entity.Pool, error) {
	var (
		pools    = make([]entity.Pool, len(poolAddreses))
		poolData = make([]poolData, len(poolAddreses))
	)

	req := p.ethrpcClient.NewRequest()
	for i, poolAddress := range poolAddreses {
		req.AddCall(&ethrpc.Call{
			ABI:    lensABI,
			Target: p.Config.LensAddress,
			Method: lensMethodQueryPool,
			Params: []interface{}{poolAddress},
		}, []interface{}{&poolData[i]})
	}

	if _, err := req.Aggregate(); err != nil {
		logger.Errorf("failed to get pools, err: %v", err)
		return nil, err
	}

	for i, poolAddress := range poolAddreses {
		var (
			poolAddr = strings.ToLower(poolAddress.Hex())
			poolDat  = poolData[i].Data

			poolTokens      = []*entity.PoolToken{}
			poolReserves    = []string{}
			lpTokenBalances = make(map[string]*big.Int)
		)

		for j, token := range poolDat.ListedTokens {
			t := strings.ToLower(common.BytesToAddress(token[:]).Hex())
			poolTokens = append(poolTokens, &entity.PoolToken{
				Address: t,
				Weight:  defaultWeight,
			})
			poolReserves = append(poolReserves, poolDat.Reserves[j].String())
			lpTokenBalances[t] = new(big.Int).Sub(maxUint128, poolDat.MintedLPTokens[j])
		}

		extra := Extra{
			Amp:             nil,
			Fee1e18:         nil,
			LpTokenBalances: lpTokenBalances,
			TokenInfo:       nil,
		}
		extraBytes, err := json.Marshal(extra)
		if err != nil {
			logger.Errorf("failed to marshal extra data, err: %v", err)
			return nil, err
		}

		pools[i] = entity.Pool{
			Address:   poolAddr,
			Exchange:  p.Config.DexID,
			Type:      DexTypeVelocoreV2Stable,
			Timestamp: time.Now().Unix(),
			Reserves:  poolReserves,
			Tokens:    poolTokens,
			Extra:     string(extraBytes),
		}
	}

	return pools, nil
}
