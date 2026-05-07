package reth

import (
	"errors"
	"math/big"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
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

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if param.TokenAmountIn.Token == p.Info.Tokens[0] && param.TokenOut == p.Info.Tokens[1] {
		// ETH -> rETH
		return p.deposit(param.TokenAmountIn.Amount)
	}

	// rETH -> ETH
	return p.burn(param.TokenAmountIn.Amount)
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	return p
}

func (p *PoolSimulator) UpdateBalance(_ pool.UpdateBalanceParams) {
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{
		BlockNumber: p.Info.BlockNumber,
	}
}

func (s *PoolSimulator) SwapReceiveNativeIn(tokenIn, _ string, chainId valueobject.ChainID) bool {
	return valueobject.IsWrappedNative(tokenIn, chainId)
}

func (s *PoolSimulator) SwapReturnNativeOut(_, tokenOut string, chainId valueobject.ChainID) bool {
	return valueobject.IsWrappedNative(tokenOut, chainId)
}

// deposit ETH and mint rETH
func (p *PoolSimulator) deposit(amount *big.Int) (*pool.CalcAmountOutResult, error) {
	if !p.depositEnabled {
		return nil, ErrDepositDisabled
	} else if amount.Cmp(p.minimumDeposit) < 0 {
		return nil, ErrDepositLessThanMinimum
	}

	capacityNeeded := new(big.Int).Add(p.balance, amount)
	if capacityNeeded.Cmp(p.maximumDepositPoolSize) > 0 {
		if p.assignDepositsEnabled {
			if capacityNeeded.Cmp(new(big.Int).Add(p.maximumDepositPoolSize, p.effectiveCapacity)) > 0 {
				return nil, ErrDepositMatchWithMinipoolsMoreThanMaximum
			}
		} else {
			return nil, ErrDepositMoreThanMaximum
		}
	}

	depositFee := bignumber.MulDivDown(new(big.Int), amount, p.depositFee, calcBase)
	depositNet := depositFee.Sub(amount, depositFee)

	if p.totalRETHSupply.Sign() == 0 {
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{Token: p.Info.Tokens[1], Amount: depositNet},
			Fee:            &pool.TokenAmount{Token: p.Info.Tokens[1], Amount: bignumber.ZeroBI},
			Gas:            p.gas.Deposit,
		}, nil
	} else if p.totalETHBalance.Sign() <= 0 {
		return nil, ErrZeroNetworkBalance
	}

	amountOut := bignumber.MulDivDown(depositNet, depositNet, p.totalRETHSupply, p.totalETHBalance)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: p.Info.Tokens[1], Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: p.Info.Tokens[1], Amount: bignumber.ZeroBI},
		Gas:            p.gas.Deposit,
	}, nil
}

// burn rETH and withdraw ETH
func (p *PoolSimulator) burn(amount *big.Int) (*pool.CalcAmountOutResult, error) {
	ethAmount := p.getEthValue(amount)
	ethBalance := new(big.Int).Add(p.excessBalance, p.rETHBalance)
	if ethBalance.Cmp(ethAmount) < 0 {
		return nil, ErrInsufficientETHBalance
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: p.Info.Tokens[0], Amount: ethAmount},
		Fee:            &pool.TokenAmount{Token: p.Info.Tokens[0], Amount: bignumber.ZeroBI},
		Gas:            p.gas.Burn,
	}, nil
}

func (p *PoolSimulator) getEthValue(rethAmount *big.Int) *big.Int {
	if p.totalRETHSupply.Sign() == 0 {
		return rethAmount
	}

	return bignumber.MulDivDown(new(big.Int), rethAmount, p.totalETHBalance, p.totalRETHSupply)
}
