package erc4626

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	Extra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     p.Address,
			Exchange:    p.Exchange,
			Type:        p.Type,
			Tokens:      lo.Map(p.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(p.Reserves, func(r string, _ int) *big.Int { return bignum.NewBig(r) }),
			BlockNumber: p.BlockNumber,
		}},
		Extra: extra,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenOut

	swapType, err := s.getSwapType(tokenIn, tokenOut)
	if err != nil {
		return nil, err
	}

	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)
	isDeposit := swapType == Deposit

	var amountOut *uint256.Int
	if isDeposit {
		if s.MaxDeposit != nil && amountIn.Gt(s.MaxDeposit) {
			return nil, ErrERC4626DepositMoreThanMax
		}
		amountOut, err = GetClosestRate(s.DepositRates, amountIn)
		if err != nil {
			return nil, ErrInvalidDepositRate
		}
	} else {
		if s.MaxRedeem != nil && amountIn.Gt(s.MaxRedeem) {
			return nil, ErrERC4626RedeemMoreThanMax
		}
		amountOut, err = GetClosestRate(s.RedeemRates, amountIn)
		if err != nil {
			return nil, ErrInvalidRedeemRate
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: bignum.ZeroBI,
		},
		Gas: int64(lo.Ternary(isDeposit, s.Gas.Deposit, s.Gas.Redeem)),
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn, tokenOut := params.TokenIn, params.TokenAmountOut.Token

	swapType, err := s.getSwapType(tokenIn, tokenOut)
	if err != nil {
		return nil, err
	}

	amountOut := uint256.MustFromBig(params.TokenAmountOut.Amount)
	isDeposit := swapType == Deposit

	var amountIn *uint256.Int
	if isDeposit {
		amountIn, err = GetClosestRate(s.DepositRates, amountOut)
		if err != nil {
			return nil, ErrInvalidDepositRate
		} else if s.MaxDeposit != nil && amountIn.Gt(s.MaxDeposit) {
			return nil, ErrERC4626DepositMoreThanMax
		}
	} else {
		amountIn, err = GetClosestRate(s.RedeemRates, amountOut)
		if err != nil {
			return nil, ErrInvalidRedeemRate
		} else if s.MaxRedeem != nil && amountIn.Gt(s.MaxRedeem) {
			return nil, ErrERC4626RedeemMoreThanMax
		}
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: bignum.ZeroBI,
		},
		Gas: int64(lo.Ternary(isDeposit, s.Gas.Deposit, s.Gas.Redeem)),
	}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	if s.MaxDeposit != nil {
		cloned.MaxDeposit = new(uint256.Int).Set(s.MaxDeposit)
	}
	if s.MaxRedeem != nil {
		cloned.MaxRedeem = new(uint256.Int).Set(s.MaxRedeem)
	}
	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenAmountIn, tokenAmountOut := params.TokenAmountIn, params.TokenAmountOut
	swapType, err := s.getSwapType(tokenAmountIn.Token, tokenAmountOut.Token)
	if err != nil {
		return
	}

	if swapType == Deposit {
		if s.MaxDeposit != nil {
			s.MaxDeposit = new(uint256.Int).Sub(s.MaxDeposit, uint256.MustFromBig(tokenAmountIn.Amount))
		}
	} else if s.MaxRedeem != nil {
		s.MaxRedeem = new(uint256.Int).Sub(s.MaxRedeem, uint256.MustFromBig(tokenAmountIn.Amount))
	}
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return Meta{
		BlockNumber: s.Info.BlockNumber,
	}
}

func (s *PoolSimulator) CanSwapTo(address string) []string {
	tokenOutIndex := s.GetTokenIndex(address)

	if tokenOutIndex < 0 {
		return []string{}
	}

	if s.SwapTypes == Both ||
		(s.SwapTypes == Deposit && tokenOutIndex == 0) ||
		(s.SwapTypes == Redeem && tokenOutIndex == 1) {
		return []string{s.Info.Tokens[1-tokenOutIndex]}
	}

	return []string{}
}

func (s *PoolSimulator) CanSwapFrom(address string) []string {
	tokenInIndex := s.GetTokenIndex(address)

	if tokenInIndex < 0 {
		return []string{}
	}

	if s.SwapTypes == Both ||
		(s.SwapTypes == Deposit && tokenInIndex == 1) ||
		(s.SwapTypes == Redeem && tokenInIndex == 0) {
		return []string{s.Info.Tokens[1-tokenInIndex]}
	}

	return []string{}
}

func (s *PoolSimulator) getSwapType(tokenIn string, tokenOut string) (SwapType, error) {
	tokenInIndex, tokenOutIndex := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 || tokenInIndex == tokenOutIndex {
		return None, errors.Wrapf(ErrInvalidToken, "tokenIn: %s, tokenOut: %s", tokenIn, tokenOut)
	}

	swapType := lo.Ternary(tokenInIndex < tokenOutIndex, Redeem, Deposit)

	if s.SwapTypes != swapType && s.SwapTypes != Both {
		return None, errors.Wrapf(ErrUnsupportedSwap, "unsupported swap type: %v", swapType)
	}

	return swapType, nil
}
