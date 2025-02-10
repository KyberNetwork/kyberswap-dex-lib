package sfrxeth_convertor

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	frax_common "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/frax/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/frax/sfrxeth"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client

		hasInitialized bool
	}
)

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

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	logger.WithFields(logger.Fields{"dex_id": u.config.DexID}).Info("Start getting new pools")

	startTime := time.Now()
	u.hasInitialized = true

	byteData, ok := bytesByPath[u.config.PoolPath]
	if !ok {
		logger.Errorf("misconfigured poolPath")
		return nil, nil, errors.New("misconfigured poolPath")
	}

	var poolItem PoolItem
	if err := json.Unmarshal(byteData, &poolItem); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to unmarshal poolData")
		return nil, nil, err
	}

	totalSupply, totalAssets, blockNumber, err := getReserves(ctx, poolItem.SfrxETHAddress, u.ethrpcClient, nil)
	if err != nil {
		return nil, nil, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      DexType,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return []entity.Pool{
		{
			Address:   poolItem.SfrxETHAddress,
			Reserves:  []string{totalAssets.String(), totalSupply.String()},
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(poolItem.FrxETHAddress),
					Swappable: true,
				},
				{
					Address:   strings.ToLower(poolItem.SfrxETHAddress),
					Swappable: true,
				},
			},
			BlockNumber: blockNumber,
		},
	}, nil, nil
}

func getReserves(
	ctx context.Context,
	poolAddress string,
	ethrpcClient *ethrpc.Client,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*big.Int, *big.Int, uint64, error) {
	var (
		totalSupply *big.Int
		totalAssets *big.Int
	)

	calls := ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    frax_common.SfrxETHABI,
		Target: poolAddress,
		Method: sfrxeth.SfrxETHMethodTotalAssets,
	}, []interface{}{&totalAssets})
	calls.AddCall(&ethrpc.Call{
		ABI:    frax_common.SfrxETHABI,
		Target: poolAddress,
		Method: sfrxeth.SfrxETHMethodTotalSupply,
	}, []interface{}{&totalSupply})

	resp, err := calls.Aggregate()
	if err != nil {
		return nil, nil, 0, err
	}

	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	return totalSupply, totalAssets, resp.BlockNumber.Uint64(), nil
}
