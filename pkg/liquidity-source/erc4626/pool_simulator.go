package erc4626

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		pool.Pool

		supportedSwapType SwapType
		TotalAssets       *uint256.Int
		TotalSupply       *uint256.Int

		MaxDeposit *uint256.Int
		MaxRedeem  *uint256.Int

		gas Gas
	}
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	tokens := lo.Map(p.Tokens, func(e *entity.PoolToken, _ int) string { return e.Address })
	reserves := lo.Map(p.Reserves, func(e string, _ int) *big.Int { return bignum.NewBig(e) })

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     p.Address,
			Exchange:    p.Exchange,
			Type:        p.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: p.BlockNumber,
		}},
		supportedSwapType: extra.SwapTypes,
		TotalSupply:       uint256.MustFromDecimal(p.Reserves[0]),
		TotalAssets:       uint256.MustFromDecimal(p.Reserves[1]),
		MaxDeposit:        extra.MaxDeposit,
		MaxRedeem:         extra.MaxRedeem,
		gas:               extra.Gas,
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
	var postTotalShares, postTotalAssets *uint256.Int

	if isDeposit {
		amountOut, err = s.deposit(amountIn)
		if err != nil {
			return nil, err
		}

		postTotalShares = new(uint256.Int).Add(s.TotalSupply, amountOut)
		postTotalAssets = new(uint256.Int).Add(s.TotalAssets, amountIn)

	} else {
		amountOut, err = s.redeem(amountIn)
		if err != nil {
			return nil, err
		}

		postTotalShares = new(uint256.Int).Sub(s.TotalSupply, amountIn)
		postTotalAssets = new(uint256.Int).Sub(s.TotalAssets, amountOut)
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: integer.Zero(),
		},
		SwapInfo: PostSwapState{
			totalSupply: postTotalShares,
			totalAssets: postTotalAssets,
		},
		Gas: int64(lo.Ternary(isDeposit, s.gas.Deposit, s.gas.Redeem)),
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	postSwapState := params.SwapInfo.(PostSwapState)
	s.TotalSupply.Set(postSwapState.totalSupply)
	s.TotalAssets.Set(postSwapState.totalAssets)
}

func (s *PoolSimulator) maxDeposit() *uint256.Int {
	return s.MaxDeposit
}

func (s *PoolSimulator) deposit(asset *uint256.Int) (*uint256.Int, error) {
	if asset.Gt(s.maxDeposit()) {
		return nil, ErrERC4626DepositMoreThanMax
	}

	return s.previewDeposit(asset)
}

func (s *PoolSimulator) previewDeposit(asset *uint256.Int) (*uint256.Int, error) {
	shares, overflow := new(uint256.Int).MulDivOverflow(asset, s.TotalSupply, s.TotalAssets)
	if overflow {
		return nil, number.ErrOverflow
	}

	return shares, nil
}

func (s *PoolSimulator) redeem(shares *uint256.Int) (*uint256.Int, error) {
	if shares.Gt(s.maxRedeem()) {
		return nil, ErrERC4626RedeemMoreThanMax
	}

	return s.previewRedeem(shares)
}

func (s *PoolSimulator) previewRedeem(shares *uint256.Int) (*uint256.Int, error) {
	assets, overflow := new(uint256.Int).MulDivOverflow(shares, s.TotalAssets, s.TotalSupply)
	if overflow {
		return nil, number.ErrOverflow
	}

	return assets, nil
}

func (s *PoolSimulator) maxRedeem() *uint256.Int {
	return s.MaxRedeem
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) interface{} {
	return Meta{BlockNumber: s.Info.BlockNumber}
}

func (s *PoolSimulator) CanSwapTo(address string) []string {
	tokenOutIndex := s.GetTokenIndex(address)

	if tokenOutIndex < 0 {
		return []string{}
	}

	if s.supportedSwapType == Both ||
		(s.supportedSwapType == Deposit && tokenOutIndex == 0) ||
		(s.supportedSwapType == Redeem && tokenOutIndex == 1) {
		return []string{s.Info.Tokens[1-tokenOutIndex]}
	}

	return []string{}
}

func (s *PoolSimulator) CanSwapFrom(address string) []string {
	tokenInIndex := s.GetTokenIndex(address)

	if tokenInIndex < 0 {
		return []string{}
	}

	if s.supportedSwapType == Both ||
		(s.supportedSwapType == Deposit && tokenInIndex == 1) ||
		(s.supportedSwapType == Redeem && tokenInIndex == 0) {
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

	if s.supportedSwapType != swapType && s.supportedSwapType != Both {
		return None, errors.Wrapf(ErrUnsupportedSwap, "unsupported swap type: %v", swapType)
	}

	return swapType, nil
}
