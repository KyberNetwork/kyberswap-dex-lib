package sfrxeth

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	frax_common "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/frax/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client

		hasInitialized bool
	}
)

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

	extra, blockNumber, err := getExtra(ctx, poolItem.FrxETHMinterAddress, poolItem.SfrxETHAddress, u.ethrpcClient)
	if err != nil {
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(extra)
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
			Address:   poolItem.FrxETHMinterAddress,
			Reserves:  []string{defaultReserves, defaultReserves},
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{
					Address:   valueobject.WrapETHLower(valueobject.EtherAddress, u.config.ChainID),
					Swappable: true,
				},
				{
					Address:   strings.ToLower(poolItem.SfrxETHAddress),
					Swappable: true,
				},
			},
			BlockNumber: blockNumber,
			Extra:       string(extraBytes),
		},
	}, nil, nil
}

func getExtra(
	ctx context.Context,
	minterAddress string,
	sfrxETHAddress string,
	ethrpcClient *ethrpc.Client,
) (PoolExtra, uint64, error) {

	var (
		submitPaused bool
		totalSupply  *big.Int
		totalAssets  *big.Int
	)

	calls := ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    frax_common.FrxETHMinterABI,
		Target: minterAddress,
		Method: minterMethodSubmitPaused,
	}, []interface{}{&submitPaused})
	calls.AddCall(&ethrpc.Call{
		ABI:    frax_common.SfrxETHABI,
		Target: sfrxETHAddress,
		Method: sfrxETHMethodTotalSupply,
	}, []interface{}{&totalSupply})
	calls.AddCall(&ethrpc.Call{
		ABI:    frax_common.SfrxETHABI,
		Target: sfrxETHAddress,
		Method: sfrxETHMethodTotalAssets,
	}, []interface{}{&totalAssets})

	resp, err := calls.Aggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}

	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	poolExtra := PoolExtra{
		SubmitPaused: submitPaused,
		TotalSupply:  uint256.MustFromBig(totalSupply),
		TotalAssets:  uint256.MustFromBig(totalAssets),
	}

	return poolExtra, resp.BlockNumber.Uint64(), nil
}
