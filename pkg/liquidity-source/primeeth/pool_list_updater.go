package primeeth

import (
	"context"
	"encoding/json"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/logger"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"math/big"
	"time"
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

	logger.WithFields(logger.Fields{"dex_id": u.config.DexID}).Debug("Start getting new pools")

	startTime := time.Now()
	u.hasInitialized = true

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
			Address:   primeZapper,
			Reserves:  []string{defaultReserves, defaultReserves},
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{
					Address:   WETH,
					Symbol:    "WETH",
					Decimals:  18,
					Name:      "Wrapped Ether",
					Swappable: true,
				},
				{
					Address:   PrimeETH,
					Symbol:    "primeETH",
					Decimals:  18,
					Name:      "Prime Staked ETH",
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
	ethrpcClient *ethrpc.Client,
	overrides map[gethcommon.Address]gethclient.OverrideAccount,
) (PoolExtra, uint64, error) {

	var (
		paused              bool
		totalAssetDeposit   *big.Int
		depositLimitByAsset *big.Int
		minAmountToDeposit  *big.Int
		primeETHPrice       *big.Int
	)

	calls := ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    lrtDepositPoolABI,
		Target: lrtDepositPool,
		Method: lrtDepositPoolMethodPaused,
		Params: []interface{}{},
	}, []interface{}{&paused})
	calls.AddCall(&ethrpc.Call{
		ABI:    lrtDepositPoolABI,
		Target: lrtDepositPool,
		Method: lrtDepositPoolMethodMinAmountToDeposit,
		Params: []interface{}{},
	}, []interface{}{&minAmountToDeposit})
	calls.AddCall(&ethrpc.Call{
		ABI:    lrtDepositPoolABI,
		Target: lrtDepositPool,
		Method: lrtDepositPoolMethodGetTotalAssetDeposits,
		Params: []interface{}{gethcommon.HexToAddress(WETH)},
	}, []interface{}{&totalAssetDeposit})
	calls.AddCall(&ethrpc.Call{
		ABI:    lrtConfigABI,
		Target: lrtConfig,
		Method: lrtConfigMethodDepositLimitByAsset,
		Params: []interface{}{gethcommon.HexToAddress(WETH)},
	}, []interface{}{&depositLimitByAsset})
	calls.AddCall(&ethrpc.Call{
		ABI:    lrtOracleABI,
		Target: lrtOracle,
		Method: lrtOracleMethodPrimeETHPrice,
		Params: []interface{}{},
	}, []interface{}{&primeETHPrice})

	resp, err := calls.Aggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}
	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	poolExtra := PoolExtra{
		Paused:              paused,
		TotalAssetDeposit:   totalAssetDeposit,
		DepositLimitByAsset: depositLimitByAsset,
		MinAmountToDeposit:  minAmountToDeposit,
		PrimeETHPrice:       primeETHPrice,
	}

	return poolExtra, resp.BlockNumber.Uint64(), nil
}
