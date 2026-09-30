package clanker

import (
	"math/big"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// For msgpack registration
type Hook struct{}

// Exact-in takes the protocol fee in the paired token: from amountIn in beforeSwap when buying clanker,
// from amountOut in afterSwap when selling clanker. CalcIn reverses exact-in, so it inverts these two fees
// rather than following the contract's exact-out branches.

// beforeSwapDelta returns BeforeSwapResult.DeltaSpecified.
func beforeSwapDelta(params *uniswapv4.BeforeSwapParams, swappingForClanker bool, protocolFee *big.Int) *big.Int {
	if params.CalcOut != swappingForClanker {
		return bignumber.ZeroBI
	} else if params.CalcOut { // fee taken from paired amountIn
		var fee big.Int
		return bignumber.MulDivDown(&fee, params.AmountSpecified, scaledProtocolFee(protocolFee), bignumber.BONE)
	}
	// pool must give enough paired amountOut to leave AmountSpecified after the afterSwap fee
	gross := uniswapv4.GrossBeforeFee(params.AmountSpecified, protocolFee, FeeDenominator)
	return gross.Sub(gross, params.AmountSpecified)
}

// afterSwapHookFee returns AfterSwapResult.HookFee.
func afterSwapHookFee(params *uniswapv4.AfterSwapParams, swappingForClanker bool, protocolFee *big.Int) *big.Int {
	if params.CalcOut == swappingForClanker {
		return bignumber.ZeroBI
	} else if params.CalcOut { // fee taken from paired amountOut
		// https://basescan.org/address/0x34a45c6B61876d739400Bd71228CbcbD4F53E8cC#code#F2#L349
		var fee big.Int
		return bignumber.MulDivDown(&fee, params.AmountOut, protocolFee, FeeDenominator)
	}
	// paired amountIn whose beforeSwap fee leaves the pool's AmountIn
	gross := uniswapv4.GrossBeforeFee(params.AmountIn, scaledProtocolFee(protocolFee), bignumber.BONE)
	return gross.Sub(gross, params.AmountIn)
}

// scaledProtocolFee = protocolFee * 1e18 / (1e6 + protocolFee)
// https://basescan.org/address/0x34a45c6B61876d739400Bd71228CbcbD4F53E8cC#code#F2#L297
func scaledProtocolFee(protocolFee *big.Int) *big.Int {
	var fee, denom big.Int
	return bignumber.MulDivDown(&fee, protocolFee, bignumber.BONE, denom.Add(Million, protocolFee))
}
