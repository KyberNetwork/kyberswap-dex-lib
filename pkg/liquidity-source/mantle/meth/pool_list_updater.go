package meth

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
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

	startTime := time.Now()
	u.hasInitialized = true
	logger.WithFields(logger.Fields{"dex_id": u.config.DexID}).Debug("Start getting new pools")

	extra, blockNumber, err := getExtra(ctx, u.ethrpcClient, nil)
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
			Address:   MantleLSPStaking,
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{defaultReserves, defaultReserves},
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(WETH),
					Symbol:    "WETH",
					Decimals:  18,
					Name:      "Wrapped Ether",
					Swappable: true,
				},
				{
					Address:   strings.ToLower(METH),
					Symbol:    "mETH",
					Decimals:  18,
					Name:      "mETH",
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
	client *ethrpc.Client,
	overrides map[common.Address]gethclient.OverrideAccount,
) (PoolExtra, uint64, error) {
	var (
		isStakingPaused        bool
		minimumStakeBound      *big.Int
		maximumMETHSupply      *big.Int
		maximumDepositAmount   *big.Int
		totalControlled        *big.Int
		exchangeAdjustmentRate uint16
		mETHTotalSupply        *big.Int
	)

	calls := client.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    mantlePauserABI,
		Target: MantlePauser,
		Method: mantlePauserMethodIsStakingPaused,
		Params: []interface{}{},
	}, []interface{}{&isStakingPaused})
	calls.AddCall(&ethrpc.Call{
		ABI:    mantleLSPStakingABI,
		Target: MantleLSPStaking,
		Method: mantleLSPStakingMethodMinimumStakeBound,
		Params: []interface{}{},
	}, []interface{}{&minimumStakeBound})
	calls.AddCall(&ethrpc.Call{
		ABI:    mantleLSPStakingABI,
		Target: MantleLSPStaking,
		Method: mantleLSPStakingMethodExchangeAdjustmentRate,
		Params: []interface{}{},
	}, []interface{}{&exchangeAdjustmentRate})
	calls.AddCall(&ethrpc.Call{
		ABI:    mantleLSPStakingABI,
		Target: MantleLSPStaking,
		Method: mantleLSPStakingMethodMaximumDepositAmount,
		Params: []interface{}{},
	}, []interface{}{&maximumDepositAmount})
	calls.AddCall(&ethrpc.Call{
		ABI:    mantleLSPStakingABI,
		Target: MantleLSPStaking,
		Method: mantleLSPStakingMethodTotalControlled,
		Params: []interface{}{},
	}, []interface{}{&totalControlled})
	calls.AddCall(&ethrpc.Call{
		ABI:    mantleLSPStakingABI,
		Target: MantleLSPStaking,
		Method: mantleLSPStakingMethodMaximumMETHSupply,
		Params: []interface{}{},
	}, []interface{}{&maximumMETHSupply})
	calls.AddCall(&ethrpc.Call{
		ABI:    methABI,
		Target: METH,
		Method: mETHMethodTotalSupply,
		Params: []interface{}{},
	}, []interface{}{&mETHTotalSupply})

	resp, err := calls.Aggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}

	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	return PoolExtra{
		IsStakingPaused:        isStakingPaused,
		MinimumStakeBound:      uint256.MustFromBig(minimumStakeBound),
		MaximumMETHSupply:      uint256.MustFromBig(maximumMETHSupply),
		TotalControlled:        uint256.MustFromBig(totalControlled),
		ExchangeAdjustmentRate: exchangeAdjustmentRate,
		METHTotalSupply:        uint256.MustFromBig(mETHTotalSupply),
	}, resp.BlockNumber.Uint64(), nil
}
