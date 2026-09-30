package gmxglp

import (
	"context"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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
	strategyListEstimatedTotalAssetsResult := make([]*uint256.Int, len(strategyList))
	var (
		totalSupply             *uint256.Int
		totalAssets             *uint256.Int
		lastReport              *uint256.Int
		lockedProfitDegradation *uint256.Int
		lockedProfit            *uint256.Int
		depositLimit            *uint256.Int
		totalIdle               *uint256.Int
	)
	yearnTokenVault := &YearnTokenVault{
		Address:         strings.ToLower(address),
		WithdrawalQueue: make([]string, 0),
	}

	type GetStrategies struct {
		Strategies struct {
			PerformanceFee    *uint256.Int `json:"performanceFee"`
			Activation        *uint256.Int `json:"activation"`
			DebtRatio         *uint256.Int `json:"debtRatio"`
			MinDebtPerHarvest *uint256.Int `json:"minDebtPerHarvest"`
			MaxDebtPerHarvest *uint256.Int `json:"maxDebtPerHarvest"`
			LastReport        *uint256.Int `json:"lastReport"`
			TotalDebt         *uint256.Int `json:"totalDebt"`
			TotalGain         *uint256.Int `json:"totalGain"`
			TotalLoss         *uint256.Int `json:"totalLoss"`
		}
	}
	yearnStrategy := make([]GetStrategies, len(strategyList))

	calls := y.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    yearnTokenVaultABI,
		Target: address,
		Method: yearnTokenVaultMethodTotalSupply,
		Params: nil,
	}, []any{&totalSupply})
	calls.AddCall(&ethrpc.Call{
		ABI:    yearnTokenVaultABI,
		Target: address,
		Method: yearnTokenVaultMethodTotalAssets,
		Params: nil,
	}, []any{&totalAssets})
	calls.AddCall(&ethrpc.Call{
		ABI:    yearnTokenVaultABI,
		Target: address,
		Method: yearnTokenVaultMethodLastReport,
		Params: nil,
	}, []any{&lastReport})
	calls.AddCall(&ethrpc.Call{
		ABI:    yearnTokenVaultABI,
		Target: address,
		Method: yearnTokenVaultMethodLockedProfitDegradation,
		Params: nil,
	}, []any{&lockedProfitDegradation})
	calls.AddCall(&ethrpc.Call{
		ABI:    yearnTokenVaultABI,
		Target: address,
		Method: yearnTokenVaultMethodLockedProfit,
		Params: nil,
	}, []any{&lockedProfit})
	calls.AddCall(&ethrpc.Call{
		ABI:    yearnTokenVaultABI,
		Target: address,
		Method: yearnTokenVaultMethodDepositLimit,
		Params: nil,
	}, []any{&depositLimit})
	calls.AddCall(&ethrpc.Call{
		ABI:    yearnTokenVaultABI,
		Target: address,
		Method: yearnTokenVaultMethodTotalIdle,
		Params: nil,
	}, []any{&totalIdle})
	for i := 0; i < 10; i++ {
		calls.AddCall(&ethrpc.Call{
			ABI:    yearnTokenVaultABI,
			Target: address,
			Method: yearnTokenVaultMethodWithdrawalQueue,
			Params: []any{big.NewInt(int64(i))},
		}, []any{&withdrawalQueue[i]})
	}
	for i, strategyAddress := range strategyList {
		calls.AddCall(&ethrpc.Call{
			ABI:    strategyBLTStakerABI,
			Target: strategyAddress,
			Method: yearnTokenVaultStrategyMethodEstimatedTotalAssets,
			Params: nil,
		}, []any{&strategyListEstimatedTotalAssetsResult[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    yearnTokenVaultABI,
			Target: address,
			Method: yearnTokenVaultMethodStrategies,
			Params: []any{common.HexToAddress(strategyAddress)},
		}, []any{&yearnStrategy[i]})

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
			TotalDebt:            u256ToBig(yearnStrategy[i].Strategies.TotalDebt),
			EstimatedTotalAssets: u256ToBig(strategyListEstimatedTotalAssetsResult[i]),
		}
	}

	yearnTokenVault.TotalSupply = u256ToBig(totalSupply)
	yearnTokenVault.TotalAsset = u256ToBig(totalAssets)
	yearnTokenVault.LastReport = u256ToBig(lastReport)
	yearnTokenVault.LockedProfitDegradation = u256ToBig(lockedProfitDegradation)
	yearnTokenVault.LockedProfit = u256ToBig(lockedProfit)
	yearnTokenVault.DepositLimit = u256ToBig(depositLimit)
	yearnTokenVault.TotalIdle = u256ToBig(totalIdle)
	yearnTokenVault.WithdrawalQueue = withdrawalQueueResult
	yearnTokenVault.YearnStrategyMap = yearnStrategyMap

	return yearnTokenVault, nil
}
