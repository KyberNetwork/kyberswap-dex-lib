package bin

import (
	"math/big"

	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Legacy big.Int swap, CalcAmountOut, CalcAmountIn and UpdateBalance, kept for parity tests until the golden data is removed.

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := params.TokenAmountIn, params.TokenOut
	indexIn, indexOut := p.GetTokenIndex(tokenIn.Token), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, shared.ErrInvalidToken
	}

	if p.activeId == 0 {
		return nil, shared.ErrUninitializedPool
	}

	amountIn := tokenIn.Amount
	// A native-flagged side trades at native's own decimals on-chain; only Arc's differ (18 vs 6).
	if p.isNative[indexIn] {
		amountIn = valueobject.UnwrapNativeAmount(p.chainID, amountIn)
	}

	res, err := p.swap(true, indexIn == 0, amountIn)
	if err != nil {
		return nil, err
	}

	amountOut := res.Amount.ToBig()
	if p.isNative[indexOut] {
		amountOut = valueobject.WrapNativeAmount(p.chainID, amountOut)
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn.Token,
			Amount: bignumber.ZeroBI,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn.Token,
			Amount: res.Fee.ToBig(),
		},
		Gas: p.gas(),
		SwapInfo: SwapInfo{
			NewActiveID:        res.NewActiveID,
			BinsReserveChanges: res.BinsReserveChanges,
		},
	}, nil
}

func (p *PoolSimulator) swap(exactIn, swapForY bool, amountIn *big.Int) (*swapResult, error) {
	amountsLeft, overflow := uint256.FromBig(amountIn)
	if overflow {
		return nil, shared.ErrInvalidAmountIn
	}
	return p.swapU(exactIn, swapForY, amountsLeft)
}

func (p *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn, tokenOut := params.TokenIn, params.TokenAmountOut
	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, shared.ErrInvalidToken
	}

	if p.activeId == 0 {
		return nil, shared.ErrUninitializedPool
	}

	amountOut := tokenOut.Amount
	// A native-flagged side trades at native's own decimals on-chain; only Arc's differ (18 vs 6).
	if p.isNative[indexOut] {
		amountOut = valueobject.UnwrapNativeAmount(p.chainID, amountOut)
	}

	res, err := p.swap(false, indexIn == 0, amountOut)
	if err != nil {
		return nil, err
	}

	amountIn := res.Amount.ToBig()
	if p.isNative[indexIn] {
		amountIn = valueobject.WrapNativeAmount(p.chainID, amountIn)
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn,
		},
		RemainingTokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut.Token,
			Amount: bignumber.ZeroBI,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: res.Fee.ToBig(),
		},
		Gas: p.gas(),
		SwapInfo: SwapInfo{
			NewActiveID:        res.NewActiveID,
			BinsReserveChanges: res.BinsReserveChanges,
		},
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.WithFields(logger.Fields{
			"address": p.Info.Address,
		}).Warn("invalid swap info")
		return
	}

	// active bin ID
	p.activeId = swapInfo.NewActiveID

	// update reserves of bins
	totalBinReserveChanges := make(map[uint32][2]*uint256.Int)
	for _, b := range swapInfo.BinsReserveChanges {
		changes, ok := totalBinReserveChanges[b.BinID]
		if !ok {
			totalBinReserveChanges[b.BinID] = [2]*uint256.Int{
				new(uint256.Int).Sub(b.AmountXIn, b.AmountXOut),
				new(uint256.Int).Sub(b.AmountYIn, b.AmountYOut),
			}
			continue
		}

		changes[0].Add(changes[0], b.AmountXIn).Sub(changes[0], b.AmountXOut)
		changes[1].Add(changes[1], b.AmountYIn).Sub(changes[1], b.AmountYOut)
	}
	newBins := p.bins[:0]
	for _, b := range p.bins {
		if changes, ok := totalBinReserveChanges[b.ID]; ok {
			b.ReserveX = changes[0].Add(b.ReserveX, changes[0])
			b.ReserveY = changes[1].Add(b.ReserveY, changes[1])
		}

		newBins = append(newBins, b)
	}
	p.bins = newBins
}
