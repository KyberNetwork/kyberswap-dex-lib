package gmxglp

import (
	"context"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strings"
)

const (
	yearnTokenVaultMethodTotalSupply                  = "totalSupply"
	yearnTokenVaultMethodTotalAssets                  = "totalAssets"
	yearnTokenVaultMethodLastReport                   = "lastReport"
	yearnTokenVaultMethodLockedProfitDegradation      = "lockedProfitDegradation"
	yearnTokenVaultMethodLockedProfit                 = "lockedProfit"
	yearnTokenVaultMethodDepositLimit                 = "depositLimit"
	yearnTokenVaultMethodTotalIdle                    = "totalIdle"
	yearnTokenVaultMethodWithdrawalQueue              = "withdrawalQueue"
	yearnTokenVaultMethodStrategies                   = "strategies"
	yearnTokenVaultStrategyMethodEstimatedTotalAssets = "estimatedTotalAssets"
)

type YearnTokenVaultScanner struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewYearnTokenVaultScanner(config *Config, ethrpcClient *ethrpc.Client) *YearnTokenVaultScanner {
	return &YearnTokenVaultScanner{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (y *YearnTokenVaultScanner) getYearnTokenVaultScanner(ctx context.Context, address string) (*YearnTokenVault, error) {
	withdrawalQueue := make([]common.Address, 10)
	strategyList := []string{"0x321E9366a4Aaf40855713868710A306Ec665CA00"}
	strategyListEstimatedTotalAssetsResult := make([]*big.Int, len(strategyList))
	yearnTokenVault := &YearnTokenVault{
		Address:         strings.ToLower(address),
		WithdrawalQueue: make([]string, 0),
	}

	type GetStrategies struct {
		Strategies struct {
			PerformanceFee    *big.Int `json:"performanceFee"`
			Activation        *big.Int `json:"activation"`
			DebtRatio         *big.Int `json:"debtRatio"`
			MinDebtPerHarvest *big.Int `json:"minDebtPerHarvest"`
			MaxDebtPerHarvest *big.Int `json:"maxDebtPerHarvest"`
			LastReport        *big.Int `json:"lastReport"`
			TotalDebt         *big.Int `json:"totalDebt"`
			TotalGain         *big.Int `json:"totalGain"`
			TotalLoss         *big.Int `json:"totalLoss"`
		}
	}
	yearnStrategy := make([]GetStrategies, len(strategyList))

	calls := y.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    yearnTokenVaultABI,
		Target: address,
		Method: yearnTokenVaultMethodTotalSupply,
		Params: nil,
	}, []interface{}{&yearnTokenVault.TotalSupply})
	calls.AddCall(&ethrpc.Call{
		ABI:    yearnTokenVaultABI,
		Target: address,
		Method: yearnTokenVaultMethodTotalAssets,
		Params: nil,
	}, []interface{}{&yearnTokenVault.TotalAsset})
	calls.AddCall(&ethrpc.Call{
		ABI:    yearnTokenVaultABI,
		Target: address,
		Method: yearnTokenVaultMethodLastReport,
		Params: nil,
	}, []interface{}{&yearnTokenVault.LastReport})
	calls.AddCall(&ethrpc.Call{
		ABI:    yearnTokenVaultABI,
		Target: address,
		Method: yearnTokenVaultMethodLockedProfitDegradation,
		Params: nil,
	}, []interface{}{&yearnTokenVault.LockedProfitDegradation})
	calls.AddCall(&ethrpc.Call{
		ABI:    yearnTokenVaultABI,
		Target: address,
		Method: yearnTokenVaultMethodLockedProfit,
		Params: nil,
	}, []interface{}{&yearnTokenVault.LockedProfit})
	calls.AddCall(&ethrpc.Call{
		ABI:    yearnTokenVaultABI,
		Target: address,
		Method: yearnTokenVaultMethodDepositLimit,
		Params: nil,
	}, []interface{}{&yearnTokenVault.DepositLimit})
	calls.AddCall(&ethrpc.Call{
		ABI:    yearnTokenVaultABI,
		Target: address,
		Method: yearnTokenVaultMethodTotalIdle,
		Params: nil,
	}, []interface{}{&yearnTokenVault.TotalIdle})
	for i := 0; i < 10; i++ {
		calls.AddCall(&ethrpc.Call{
			ABI:    yearnTokenVaultABI,
			Target: address,
			Method: yearnTokenVaultMethodWithdrawalQueue,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&withdrawalQueue[i]})
	}
	for i, strategyAddress := range strategyList {
		calls.AddCall(&ethrpc.Call{
			ABI:    strategyBLTStakerABI,
			Target: strategyAddress,
			Method: yearnTokenVaultStrategyMethodEstimatedTotalAssets,
			Params: nil,
		}, []interface{}{&strategyListEstimatedTotalAssetsResult[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    yearnTokenVaultABI,
			Target: address,
			Method: yearnTokenVaultMethodStrategies,
			Params: []interface{}{common.HexToAddress(strategyAddress)},
		}, []interface{}{&yearnStrategy[i]})

	}
	if _, err := calls.TryAggregate(); err != nil {
		logger.Errorf("failed to aggregate calls address %v with err %v", y.config.YearnTokenVaultAddress, err)
		return nil, err
	}

	withdrawalQueueResult := make([]string, 0)
	for _, strategyAddress := range withdrawalQueue {
		if strategyAddress.Hex() == valueobject.ZeroAddress {
			break
		}
		withdrawalQueueResult = append(withdrawalQueueResult, strategyAddress.Hex())
	}

	yearnStrategyMap := make(map[string]*YearnStrategy, len(strategyList))
	for i, strategy := range strategyList {
		yearnStrategyMap[strategy] = &YearnStrategy{
			TotalDebt:            yearnStrategy[i].Strategies.TotalDebt,
			EstimatedTotalAssets: strategyListEstimatedTotalAssetsResult[i],
		}
	}

	yearnTokenVault.WithdrawalQueue = withdrawalQueueResult
	yearnTokenVault.YearnStrategyMap = yearnStrategyMap

	return yearnTokenVault, nil
}
