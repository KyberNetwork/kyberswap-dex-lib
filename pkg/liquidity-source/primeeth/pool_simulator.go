package primeeth

import (
	"errors"
	"math/big"

	"github.com/bytedance/sonic"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		poolpkg.Pool

		paused              bool
		totalAssetDeposit   *big.Int
		depositLimitByAsset *big.Int
		minAmountToDeposit  *big.Int
		primeETHPrice       *big.Int

		gas Gas
	}
)

var (
	ErrPoolPaused                   = errors.New("pool is paused")
	ErrInvalidTokenIn               = errors.New("invalid tokenIn")
	ErrInvalidTokenOut              = errors.New("invalid tokenOut")
	ErrInvalidAmountToDeposit       = errors.New("invalid amount to deposit")
	ErrMaximumDepositLimitReached   = errors.New("maximum deposit limit reached")
	ErrMinimumAmountToReceiveNotMet = errors.New("minimum amount to receive not met")
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := sonic.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		paused:              extra.Paused,
		totalAssetDeposit:   extra.TotalAssetDeposit,
		depositLimitByAsset: extra.DepositLimitByAsset,
		minAmountToDeposit:  extra.MinAmountToDeposit,
		primeETHPrice:       extra.PrimeETHPrice,
		gas:                 defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	if params.TokenAmountIn.Token != s.Pool.Info.Tokens[0] {
		return nil, ErrInvalidTokenIn
	}

	if params.TokenOut != s.Info.Tokens[1] {
		return nil, ErrInvalidTokenOut
	}

	primeETHAmount, err := s._beforeDeposit(params.TokenAmountIn.Amount)
	if err != nil {
		return nil, err
	}

	// amountOut = _mint(primeETHAmount)
	// IPrimeETH(primeETH).mint(msg.sender, amount);
	// PrimeStakedETH.mint(amount) rate is 1:1

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: primeETHAmount},
		Fee:            &poolpkg.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            s.gas.Deposit,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	s.totalAssetDeposit = new(big.Int).Add(s.totalAssetDeposit, params.TokenAmountIn.Amount)
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == PrimeETH {
		return []string{WETH}
	}
	return []string{}
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == WETH {
		return []string{PrimeETH}
	}
	return []string{}
}

func (s *PoolSimulator) _beforeDeposit(depositAmount *big.Int) (*big.Int, error) {
	if depositAmount.Cmp(bignumber.ZeroBI) == 0 || depositAmount.Cmp(s.minAmountToDeposit) < 0 {
		return nil, ErrInvalidAmountToDeposit
	}

	if depositAmount.Cmp(s.getAssetCurrentLimit()) > 0 {
		return nil, ErrMaximumDepositLimitReached
	}

	primeETHAmount := s.getMintAmount(depositAmount)

	if primeETHAmount.Cmp(bignumber.ZeroBI) < 0 {
		return nil, ErrMinimumAmountToReceiveNotMet
	}

	return primeETHAmount, nil
}

func (s *PoolSimulator) getAssetCurrentLimit() *big.Int {
	if s.totalAssetDeposit.Cmp(s.depositLimitByAsset) > 0 {
		return bignumber.ZeroBI
	}
	assetCurrentLimit := new(big.Int).Sub(s.depositLimitByAsset, s.totalAssetDeposit)
	return assetCurrentLimit
}

func (s *PoolSimulator) getMintAmount(amount *big.Int) *big.Int {
	// primeEthAmount = (amount * lrtOracle.getAssetPrice(asset)) / lrtOracle.primeETHPrice();
	// lrtOracle.getAssetPrice(WETH) = 10^18
	return new(big.Int).Div(new(big.Int).Mul(amount, bignumber.BONE), s.primeETHPrice)
}
