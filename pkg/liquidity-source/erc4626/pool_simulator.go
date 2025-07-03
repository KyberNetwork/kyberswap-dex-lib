package erc4626

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		pool.Pool

		supportedSwapType   SwapType
		TotalAssets         *uint256.Int
		TotalSupply         *uint256.Int
		MaxDeposit          *uint256.Int
		MaxRedeem           *uint256.Int
		EntryFeeBasisPoints uint64
		ExitFeeBasisPoints  uint64

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
		supportedSwapType:   extra.SwapTypes,
		TotalSupply:         uint256.MustFromDecimal(p.Reserves[0]),
		TotalAssets:         uint256.MustFromDecimal(p.Reserves[1]),
		MaxDeposit:          extra.MaxDeposit,
		MaxRedeem:           extra.MaxRedeem,
		EntryFeeBasisPoints: extra.EntryFeeBps,
		ExitFeeBasisPoints:  extra.ExitFeeBps,
		gas:                 extra.Gas,
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

	var amountOut, assets *uint256.Int
	if isDeposit {
		amountOut, assets, err = s.deposit(amountIn, int64(s.EntryFeeBasisPoints), false)
	} else {
		amountOut, assets, err = s.redeem(amountIn, int64(s.ExitFeeBasisPoints), false)
	}
	if err != nil {
		return nil, err
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
		SwapInfo: SwapInfo{assets: assets},
		Gas:      int64(lo.Ternary(isDeposit, s.gas.Deposit, s.gas.Redeem)),
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

	var amountIn, assets *uint256.Int
	if isDeposit {
		amountIn, assets, err = s.redeem(amountOut, -int64(s.EntryFeeBasisPoints), true)
	} else {
		amountIn, assets, err = s.deposit(amountOut, -int64(s.ExitFeeBasisPoints), true)
	}
	if err != nil {
		return nil, err
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
		SwapInfo: SwapInfo{assets: assets},
		Gas:      int64(lo.Ternary(isDeposit, s.gas.Deposit, s.gas.Redeem)),
	}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.TotalAssets = new(uint256.Int).Set(s.TotalAssets)
	cloned.TotalSupply = new(uint256.Int).Set(s.TotalSupply)
	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenAmountIn, tokenAmountOut := params.TokenAmountIn, params.TokenAmountOut
	swapType, err := s.getSwapType(tokenAmountIn.Token, tokenAmountOut.Token)
	if err != nil {
		return
	}
	assetsWithoutFee := params.SwapInfo.(SwapInfo).assets
	if assetsWithoutFee == nil {
		assetsWithoutFee = uint256.MustFromBig(tokenAmountIn.Amount)
	}

	if swapType == Deposit {
		s.TotalAssets = new(uint256.Int).Add(s.TotalAssets, assetsWithoutFee)
		s.TotalSupply = new(uint256.Int).Add(s.TotalSupply, uint256.MustFromBig(tokenAmountOut.Amount))
	} else {
		s.TotalAssets = new(uint256.Int).Sub(s.TotalAssets, assetsWithoutFee)
		s.TotalSupply = new(uint256.Int).Sub(s.TotalSupply, uint256.MustFromBig(tokenAmountIn.Amount))
	}
}

func (s *PoolSimulator) deposit(assets *uint256.Int, feeBps int64, roundUp bool) (*uint256.Int, *uint256.Int, error) {
	if assets.Gt(s.maxDeposit()) {
		return nil, nil, ErrERC4626DepositMoreThanMax
	}

	return s.previewDeposit(assets, feeBps, roundUp)
}

func (s *PoolSimulator) previewDeposit(assets *uint256.Int, feeBps int64, roundUp bool) (*uint256.Int, *uint256.Int,
	error) {
	assets = deductFee(assets, feeBps)
	shares, err := lo.Ternary(roundUp, v3Utils.MulDivRoundingUp, v3Utils.MulDiv)(assets, s.TotalSupply, s.TotalAssets)
	return shares, assets, err
}

func (s *PoolSimulator) maxDeposit() *uint256.Int {
	if s.MaxDeposit == nil {
		return number.MaxU256
	}

	return s.MaxDeposit
}

func (s *PoolSimulator) redeem(shares *uint256.Int, feeBps int64, roundUp bool) (*uint256.Int, *uint256.Int, error) {
	if shares.Gt(s.maxRedeem()) {
		return nil, nil, ErrERC4626RedeemMoreThanMax
	}

	return s.previewRedeem(shares, feeBps, roundUp)
}

func (s *PoolSimulator) previewRedeem(shares *uint256.Int, feeBps int64, roundUp bool) (*uint256.Int, *uint256.Int,
	error) {
	assets, err := lo.Ternary(roundUp, v3Utils.MulDivRoundingUp, v3Utils.MulDiv)(shares, s.TotalAssets, s.TotalSupply)
	if err != nil {
		return nil, nil, err
	}

	return deductFee(assets, feeBps), assets, nil
}

func (s *PoolSimulator) maxRedeem() *uint256.Int {
	if s.MaxRedeem == nil || s.MaxRedeem.IsZero() {
		return number.MaxU256
	}

	return s.MaxRedeem
}

func deductFee(assets *uint256.Int, feeBps int64) *uint256.Int {
	if feeBps != 0 {
		var tmp uint256.Int
		if feeBps > 0 {
			if err := v3Utils.MulDivV2(assets, tmp.SubUint64(big256.BasisPointUint256, uint64(feeBps)),
				big256.BasisPointUint256, &tmp, nil); err == nil {
				assets = &tmp
			}
		} else if err := v3Utils.MulDivRoundingUpV2(assets, big256.BasisPointUint256,
			tmp.SubUint64(big256.BasisPointUint256, uint64(-feeBps)), &tmp); err == nil {
			assets = &tmp
		}
	}
	return assets
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
