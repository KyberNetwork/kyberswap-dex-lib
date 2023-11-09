package velocorev2stable

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
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
		pools  = make([]entity.Pool, len(poolAddreses))
		tokens = make([][]bytes32, len(poolAddreses))
	)

	req := p.ethrpcClient.R()
	for i, poolAddress := range poolAddreses {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress.Hex(),
			Method: poolMethodGetTokenList,
		}, []interface{}{&tokens[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		logger.Errorf("failed to get pools, err: %v", err)
		return nil, err
	}

	for i, poolAddress := range poolAddreses {
		var (
			poolAddr        = strings.ToLower(poolAddress.Hex())
			tokenNbr        = len(tokens[i])
			poolTokens      = []*entity.PoolToken{}
			poolReserves    = []string{}
			lpTokenBalances = make(map[string]*big.Int)
		)

		for j := 0; j < tokenNbr; j++ {
			t := common.BytesToAddress(tokens[i][j][:])
			addr := strings.ToLower(t.String())
			poolTokens = append(poolTokens, &entity.PoolToken{
				Address: addr,
				Weight:  defaultWeight,
			})
			poolReserves = append(poolReserves, "0")
			lpTokenBalances[addr] = integer.Zero()
		}

		extra := Extra{LpTokenBalances: lpTokenBalances}
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
