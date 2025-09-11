package flaunch

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestHook_BeforeSwap_ExactIn(t *testing.T) {
	hook := &Hook{
		hook:    common.HexToAddress("0x23321f11a6d44Fd1ab790044FdFDE5758c902FDc"),
		swapFee: OnePercentFee,
	}

	// Test exact input swap with 1 ETH (1e18 wei)
	amountSpecified := big.NewInt(1e18)
	params := &uniswapv4.BeforeSwapParams{
		AmountSpecified: amountSpecified,
		ExactIn:         true,
	}

	result, err := hook.BeforeSwap(params)
	assert.NoError(t, err)
	
	// Flaunch does not take fees in BeforeSwap - all should be zero
	assert.Equal(t, uniswapv4.FeeAmount(0), result.SwapFee)
	assert.Equal(t, bignumber.ZeroBI, result.DeltaSpecific)
	assert.Equal(t, bignumber.ZeroBI, result.DeltaUnSpecific)
}

func TestHook_BeforeSwap_ExactOut(t *testing.T) {
	hook := &Hook{
		hook:    common.HexToAddress("0x23321f11a6d44Fd1ab790044FdFDE5758c902FDc"),
		swapFee: OnePercentFee,
	}

	// Test exact output swap
	amountSpecified := big.NewInt(1e18)
	params := &uniswapv4.BeforeSwapParams{
		AmountSpecified: amountSpecified,
		ExactIn:         false,
	}

	result, err := hook.BeforeSwap(params)
	assert.NoError(t, err)
	
	// Flaunch does not take fees in BeforeSwap - all should be zero
	assert.Equal(t, uniswapv4.FeeAmount(0), result.SwapFee)
	assert.Equal(t, bignumber.ZeroBI, result.DeltaSpecific)
	assert.Equal(t, bignumber.ZeroBI, result.DeltaUnSpecific)
}

func TestHook_AfterSwap_ExactIn(t *testing.T) {
	hook := &Hook{
		hook:    common.HexToAddress("0x23321f11a6d44Fd1ab790044FdFDE5758c902FDc"),
		swapFee: OnePercentFee,
	}

	// Test exact input swap - fee calculated based on output amount
	amountOut := big.NewInt(1e18)
	params := &uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{
			ExactIn: true,
		},
		AmountOut: amountOut,
	}

	result, err := hook.AfterSwap(params)
	assert.NoError(t, err)
	
	// Calculate expected 1% fee: 1e18 * 10000 / 1000000 = 1e16
	expectedFee := new(big.Int).Mul(amountOut, big.NewInt(10000))
	expectedFee.Div(expectedFee, big.NewInt(1000000))
	assert.Equal(t, expectedFee, result.HookFee)
}

func TestHook_AfterSwap_ExactOut(t *testing.T) {
	hook := &Hook{
		hook:    common.HexToAddress("0x23321f11a6d44Fd1ab790044FdFDE5758c902FDc"),
		swapFee: OnePercentFee,
	}

	// Test exact output swap with 1 ETH output
	amountOut := big.NewInt(1e18)
	params := &uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{
			ExactIn: false,
		},
		AmountOut: amountOut,
	}

	result, err := hook.AfterSwap(params)
	assert.NoError(t, err)
	
	// Calculate expected 1% fee: 1e18 * 10000 / 1000000 = 1e16
	expectedFee := new(big.Int).Mul(amountOut, big.NewInt(10000))
	expectedFee.Div(expectedFee, big.NewInt(1000000))
	assert.Equal(t, expectedFee, result.HookFee)
}

func TestHook_AfterSwap_ZeroFee(t *testing.T) {
	hook := &Hook{
		hook:    common.HexToAddress("0x23321f11a6d44Fd1ab790044FdFDE5758c902FDc"),
		swapFee: 0, // Zero fee
	}

	// Test with zero fee - should return zero hook fee
	amountOut := big.NewInt(1e18)
	params := &uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{
			ExactIn: true,
		},
		AmountOut: amountOut,
	}

	result, err := hook.AfterSwap(params)
	assert.NoError(t, err)
	assert.Equal(t, bignumber.ZeroBI, result.HookFee)
}

func TestHookAddresses(t *testing.T) {
	// Verify we have exactly 5 hook addresses
	assert.Len(t, HookAddresses, 5)
	
	// Verify all addresses are valid (not zero addresses)
	for i, addr := range HookAddresses {
		assert.NotEqual(t, common.Address{}, addr, "Hook address %d should not be zero address", i)
	}
}
