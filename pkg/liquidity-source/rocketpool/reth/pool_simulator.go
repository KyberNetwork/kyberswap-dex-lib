package reth

import (
	"errors"
	"math/big"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrDepositDisabled                          = errors.New("deposits into Rocket Pool are currently disabled")
	ErrDepositLessThanMinimum                   = errors.New("the deposited amount is less than the minimum deposit size")
	ErrDepositMatchWithMinipoolsMoreThanMaximum = errors.New("the deposit pool size after depositing exceeds the maximum size")
	ErrDepositMoreThanMaximum                   = errors.New("the deposit pool size after depositing exceeds the maximum size")
	ErrZeroNetworkBalance                       = errors.New("cannot calculate rETH token amount while total network balance is zero")

	ErrInsufficientETHBalance = errors.New("insufficient ETH balance for exchange")
)

var calcBase = new(big.Int).Set(bignumber.BONE)

type PoolSimulator struct {
	pool.Pool

	// depositEnabled: RocketDAOProtocolSettingsDeposit.getDepositEnabled
	depositEnabled bool

	// minimumDeposit: RocketDAOProtocolSettingsDeposit.getMinimumDeposit
	minimumDeposit *big.Int

	// maximumDepositPoolSize: RocketDAOProtocolSettingsDeposit.getMaximumDepositPoolSize
	maximumDepositPoolSize *big.Int

	// assignDepositsEnabled: RocketDAOProtocolSettingsDeposit.getAssignDepositsEnabled
	assignDepositsEnabled bool

	// depositFee: RocketDAOProtocolSettingsDeposit.getDepositFee
	depositFee *big.Int

	// balance: RocketVault.balanceOf("rocketDepositPool")
	balance *big.Int

	// effectiveCapacity: RocketMinipoolQueue.getEffectiveCapacity
	effectiveCapacity *big.Int

	// totalETHBalance: RocketNetworkBalances.getTotalETHBalance
	totalETHBalance *big.Int

	// totalRETHSupply: RocketNetworkBalances.getTotalRETHSupply
	totalRETHSupply *big.Int

	// excessBalance: RocketDepositPool.getExcessBalance
	excessBalance *big.Int

	// rETHBalance: BalanceAt(RocketTokenRETH)
	rETHBalance *big.Int

	gas Gas
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		depositEnabled:         extra.DepositEnabled,
		minimumDeposit:         extra.MinimumDeposit,
		maximumDepositPoolSize: extra.MaximumDepositPoolSize,
		assignDepositsEnabled:  extra.AssignDepositsEnabled,
		depositFee:             extra.DepositFee,
		balance:                extra.Balance,
		effectiveCapacity:      extra.EffectiveCapacity,
		totalETHBalance:        extra.TotalETHBalance,
		totalRETHSupply:        extra.TotalRETHSupply,
		excessBalance:          extra.ExcessBalance,
		rETHBalance:            extra.RETHBalance,
		gas:                    defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if param.TokenAmountIn.Token == s.Info.Tokens[0] && param.TokenOut == s.Info.Tokens[1] {
		// ETH -> rETH
		return s.deposit(param.TokenAmountIn.Amount)
	}

	// rETH -> ETH
	return s.burn(param.TokenAmountIn.Amount)
}

func (s *PoolSimulator) UpdateBalance(param pool.UpdateBalanceParams) {
	if param.TokenAmountIn.Token == s.Info.Tokens[0] && param.TokenAmountOut.Token == s.Info.Tokens[1] {

		return
	}

}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) interface{} {
	return PoolMeta{
		BlockNumber:     s.Pool.Info.BlockNumber,
		ApprovalAddress: s.GetApprovalAddress(tokenIn, tokenOut),
	}
}

func (s *PoolSimulator) GetApprovalAddress(tokenIn, tokenOut string) string {
	isDeposit := tokenIn == s.Info.Tokens[0] && tokenOut == s.Info.Tokens[1]
	return lo.Ternary(!isDeposit, tokenOut, "")
}

// deposit ETH and mint rETH
func (s *PoolSimulator) deposit(amount *big.Int) (*pool.CalcAmountOutResult, error) {
	if !s.depositEnabled {
		return nil, ErrDepositDisabled
	}

	if amount.Cmp(s.minimumDeposit) < 0 {
		return nil, ErrDepositLessThanMinimum
	}

	capacityNeeded := new(big.Int).Add(s.balance, amount)

	if capacityNeeded.Cmp(s.maximumDepositPoolSize) > 0 {
		if s.assignDepositsEnabled {
			if capacityNeeded.Cmp(new(big.Int).Add(s.maximumDepositPoolSize, s.effectiveCapacity)) > 0 {
				return nil, ErrDepositMatchWithMinipoolsMoreThanMaximum
			}
		} else {
			return nil, ErrDepositMoreThanMaximum
		}
	}

	depositFee := new(big.Int).Div(new(big.Int).Mul(amount, s.depositFee), calcBase)
	depositNet := new(big.Int).Sub(amount, depositFee)

	if s.totalRETHSupply.Cmp(bignumber.ZeroBI) == 0 {
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[1], Amount: depositNet},
			Fee:            &pool.TokenAmount{Token: s.Info.Tokens[1], Amount: bignumber.ZeroBI},
			Gas:            s.gas.Deposit,
		}, nil
	}

	if s.totalETHBalance.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrZeroNetworkBalance
	}

	amountOut := new(big.Int).Div(new(big.Int).Mul(depositNet, s.totalRETHSupply), s.totalETHBalance)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[1], Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[1], Amount: bignumber.ZeroBI},
		Gas:            s.gas.Deposit,
	}, nil
}

// burn rETH and withdraw ETH
func (s *PoolSimulator) burn(amount *big.Int) (*pool.CalcAmountOutResult, error) {
	ethAmount := s.getEthValue(amount)
	ethBalance := new(big.Int).Add(s.excessBalance, s.rETHBalance)

	if ethBalance.Cmp(ethAmount) < 0 {
		return nil, ErrInsufficientETHBalance
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: ethAmount},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: bignumber.ZeroBI},
		Gas:            s.gas.Burn,
	}, nil
}

func (s *PoolSimulator) getEthValue(rethAmount *big.Int) *big.Int {
	if s.totalRETHSupply.Cmp(bignumber.ZeroBI) == 0 {
		return rethAmount
	}

	return new(big.Int).Div(
		new(big.Int).Mul(rethAmount, s.totalETHBalance),
		s.totalRETHSupply,
	)
}
