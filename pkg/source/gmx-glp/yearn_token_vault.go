package gmxglp

import (
	"fmt"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"math/big"
	"time"
)

var (
	MaxUint256             = bignumber.NewBig10("115792089237316195423570985008687907853269984665640564039457584007913129639935")
	DegradationCoefficient = bignumber.TenPowInt(18)
	MaxLoss                = big.NewInt(1)
)

type YearnStrategy struct {
	TotalDebt            *big.Int `json:"TotalDebt"`
	EstimatedTotalAssets *big.Int `json:"estimatedTotalAssets"`
}

type YearnTokenVault struct {
	Address                 string                    `json:"address"`
	TotalSupply             *big.Int                  `json:"totalSupply"`
	TotalAsset              *big.Int                  `json:"totalAsset"`
	LastReport              *big.Int                  `json:"lastReport"`
	LockedProfitDegradation *big.Int                  `json:"lockedProfitDegradation"`
	LockedProfit            *big.Int                  `json:"lockedProfit"`
	DepositLimit            *big.Int                  `json:"depositLimit"`
	TotalIdle               *big.Int                  `json:"totalIdle"`
	YearnStrategyMap        map[string]*YearnStrategy `json:"yearnStrategyMap"`
	WithdrawalQueue         []string                  `json:"withdrawalQueue"`
}

func (y *YearnTokenVault) GetStrategy(address string) (IStrategy, error) {
	strategy := y.YearnStrategyMap[address]
	switch address {
	case "0x321E9366a4Aaf40855713868710A306Ec665CA00":
		return NewStrategyBltStaker(address, strategy.EstimatedTotalAssets), nil
	}

	return nil, fmt.Errorf("not found strategy %v", address)
}

func (y *YearnTokenVault) Deposit(amount *big.Int) (*big.Int, error) {
	if new(big.Int).Add(y.TotalAsset, amount).Cmp(y.DepositLimit) > 0 {
		return nil, ErrYearnTokenVaultDepositNotRespected
	}
	if amount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrYearnTokenVaultDepositNothing
	}

	return y.issueSharesForAmount(amount), nil
}

func (y *YearnTokenVault) Withdraw(maxShares *big.Int) (*big.Int, error) {
	shares := new(big.Int).Set(maxShares)
	if shares.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrYearnTokenVaultWithdrawNothing
	}

	value := y.shareValue(shares)
	vaultBalance := new(big.Int).Set(y.TotalIdle)

	if value.Cmp(vaultBalance) > 0 {
		for _, strategy := range y.WithdrawalQueue {
			if value.Cmp(vaultBalance) <= 0 {
				break
			}
			amountNeeded := new(big.Int).Sub(value, vaultBalance)
			yStrategy := y.YearnStrategyMap[strategy]
			if amountNeeded.Cmp(yStrategy.TotalDebt) < 0 {
				amountNeeded = new(big.Int).Set(yStrategy.TotalDebt)
			}
			if amountNeeded.Cmp(bignumber.ZeroBI) == 0 {
				continue
			}

			withdrawalStrategy, err := y.GetStrategy(strategy)
			if err != nil {
				return nil, err
			}

			amountFreed, loss := withdrawalStrategy.Withdraw(amountNeeded)

			// withdrawn: uint256 = self.token.balanceOf(self) - preBalance
			withdrawn := new(big.Int).Set(amountFreed)
			vaultBalance = new(big.Int).Add(vaultBalance, withdrawn)

			if loss.Cmp(bignumber.ZeroBI) > 0 {
				value = new(big.Int).Set(loss)
			}
			yStrategy.TotalDebt = new(big.Int).Sub(yStrategy.TotalDebt, withdrawn)
		}

		y.TotalIdle = new(big.Int).Set(vaultBalance)
		if value.Cmp(vaultBalance) > 0 {
			value = new(big.Int).Set(vaultBalance)
		}
	}

	return value, nil
}

func (y *YearnTokenVault) issueSharesForAmount(amount *big.Int) *big.Int {
	if y.TotalSupply.Cmp(bignumber.ZeroBI) > 0 {
		return new(big.Int).Div(new(big.Int).Mul(amount, y.TotalSupply), y.freeFund())
	}

	return new(big.Int).Set(amount)
}

func (y *YearnTokenVault) freeFund() *big.Int {
	lockedProfit := y.calculateLockedProfit()
	return new(big.Int).Sub(y.TotalAsset, lockedProfit)
}

func (y *YearnTokenVault) calculateLockedProfit() *big.Int {
	blockTimestamp := big.NewInt(time.Now().Unix())
	lockedFundsRatio := new(big.Int).Mul(
		new(big.Int).Sub(blockTimestamp, y.LastReport),
		y.LockedProfitDegradation,
	)

	if lockedFundsRatio.Cmp(DegradationCoefficient) < 0 {
		return new(big.Int).Sub(
			y.LockedProfit,
			new(big.Int).Div(new(big.Int).Mul(lockedFundsRatio, y.LockedProfit), DegradationCoefficient),
		)
	}

	return big.NewInt(0)
}

func (y *YearnTokenVault) shareValue(shares *big.Int) *big.Int {
	if y.TotalSupply.Cmp(bignumber.ZeroBI) == 0 {
		return shares
	}

	return new(big.Int).Div(new(big.Int).Mul(shares, y.freeFund()), y.TotalSupply)
}
