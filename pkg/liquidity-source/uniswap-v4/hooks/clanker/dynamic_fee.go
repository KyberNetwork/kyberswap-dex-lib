// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package clanker

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// HooksPermissions is an auto generated low-level Go binding around an user-defined struct.
type HooksPermissions struct {
	BeforeInitialize                bool
	AfterInitialize                 bool
	BeforeAddLiquidity              bool
	AfterAddLiquidity               bool
	BeforeRemoveLiquidity           bool
	AfterRemoveLiquidity            bool
	BeforeSwap                      bool
	AfterSwap                       bool
	BeforeDonate                    bool
	AfterDonate                     bool
	BeforeSwapReturnDelta           bool
	AfterSwapReturnDelta            bool
	AfterAddLiquidityReturnDelta    bool
	AfterRemoveLiquidityReturnDelta bool
}

// IClankerHookDynamicFeePoolDynamicConfigVars is an auto generated low-level Go binding around an user-defined struct.
type IClankerHookDynamicFeePoolDynamicConfigVars struct {
	BaseFee                   *big.Int
	MaxLpFee                  *big.Int
	ReferenceTickFilterPeriod *big.Int
	ResetPeriod               *big.Int
	ResetTickFilter           *big.Int
	FeeControlNumerator       *big.Int
	DecayFilterBps            *big.Int
}

// IClankerHookDynamicFeePoolDynamicFeeVars is an auto generated low-level Go binding around an user-defined struct.
type IClankerHookDynamicFeePoolDynamicFeeVars struct {
	ReferenceTick      *big.Int
	ResetTick          *big.Int
	ResetTickTimestamp *big.Int
	LastSwapTimestamp  *big.Int
	AppliedVR          *big.Int
	PrevVA             *big.Int
}

// IPoolManagerModifyLiquidityParams is an auto generated low-level Go binding around an user-defined struct.
type IPoolManagerModifyLiquidityParams struct {
	TickLower      *big.Int
	TickUpper      *big.Int
	LiquidityDelta *big.Int
	Salt           [32]byte
}

// IPoolManagerSwapParams is an auto generated low-level Go binding around an user-defined struct.
type IPoolManagerSwapParams struct {
	ZeroForOne        bool
	AmountSpecified   *big.Int
	SqrtPriceLimitX96 *big.Int
}

// PoolKey is an auto generated low-level Go binding around an user-defined struct.
type PoolKey struct {
	Currency0   common.Address
	Currency1   common.Address
	Fee         *big.Int
	TickSpacing *big.Int
	Hooks       common.Address
}

// ClankerMetaData contains all meta data concerning the Clanker contract.
var ClankerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_poolManager\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_factory\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_weth\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"BaseFeeGreaterThanMaxLpFee\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BaseFeeTooLow\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ETHPoolNotAllowed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"HookNotImplemented\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MaxLpFeeTooHigh\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MevModuleEnabled\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotPoolManager\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OnlyFactory\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"OwnableInvalidOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"OwnableUnauthorizedAccount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PastCreationTimestamp\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"TickReturned\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"UnsupportedInitializePath\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"WethCannotBeClanker\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"ClaimProtocolFees\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"beforeTick\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"afterTick\",\"type\":\"int24\"}],\"name\":\"EstimatedTickDifference\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"PoolId\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"MevModuleDisabled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"pairedToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"clanker\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tickIfToken0IsClanker\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"locker\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"mevModule\",\"type\":\"address\"}],\"name\":\"PoolCreatedFactory\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"pairedToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"clanker\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tickIfToken0IsClanker\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"}],\"name\":\"PoolCreatedOpen\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"baseFee\",\"type\":\"uint24\"},{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"maxLpFee\",\"type\":\"uint24\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"referenceTickFilterPeriod\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"resetPeriod\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"resetTickFilter\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"feeControlNumerator\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"decayFilterBps\",\"type\":\"uint24\"}],\"name\":\"PoolInitialized\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"BPS_DENOMINATOR\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"FEE_CONTROL_DENOMINATOR\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"FEE_DENOMINATOR\",\"outputs\":[{\"internalType\":\"int128\",\"name\":\"\",\"type\":\"int128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MAX_DECAY_FILTER_BPS\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MAX_LP_FEE\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MAX_MEV_MODULE_DELAY\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_BASE_FEE\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"PROTOCOL_FEE_NUMERATOR\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"int256\",\"name\":\"liquidityDelta\",\"type\":\"int256\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"}],\"internalType\":\"structIPoolManager.ModifyLiquidityParams\",\"name\":\"params\",\"type\":\"tuple\"},{\"internalType\":\"BalanceDelta\",\"name\":\"delta\",\"type\":\"int256\"},{\"internalType\":\"BalanceDelta\",\"name\":\"feesAccrued\",\"type\":\"int256\"},{\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"afterAddLiquidity\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"},{\"internalType\":\"BalanceDelta\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"afterDonate\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"afterInitialize\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"int256\",\"name\":\"liquidityDelta\",\"type\":\"int256\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"}],\"internalType\":\"structIPoolManager.ModifyLiquidityParams\",\"name\":\"params\",\"type\":\"tuple\"},{\"internalType\":\"BalanceDelta\",\"name\":\"delta\",\"type\":\"int256\"},{\"internalType\":\"BalanceDelta\",\"name\":\"feesAccrued\",\"type\":\"int256\"},{\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"afterRemoveLiquidity\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"},{\"internalType\":\"BalanceDelta\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountSpecified\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceLimitX96\",\"type\":\"uint160\"}],\"internalType\":\"structIPoolManager.SwapParams\",\"name\":\"params\",\"type\":\"tuple\"},{\"internalType\":\"BalanceDelta\",\"name\":\"delta\",\"type\":\"int256\"},{\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"afterSwap\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"},{\"internalType\":\"int128\",\"name\":\"\",\"type\":\"int128\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"int256\",\"name\":\"liquidityDelta\",\"type\":\"int256\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"}],\"internalType\":\"structIPoolManager.ModifyLiquidityParams\",\"name\":\"params\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"beforeAddLiquidity\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"beforeDonate\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"}],\"name\":\"beforeInitialize\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"int256\",\"name\":\"liquidityDelta\",\"type\":\"int256\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"}],\"internalType\":\"structIPoolManager.ModifyLiquidityParams\",\"name\":\"params\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"beforeRemoveLiquidity\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountSpecified\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceLimitX96\",\"type\":\"uint160\"}],\"internalType\":\"structIPoolManager.SwapParams\",\"name\":\"params\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"beforeSwap\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"},{\"internalType\":\"BeforeSwapDelta\",\"name\":\"\",\"type\":\"int256\"},{\"internalType\":\"uint24\",\"name\":\"\",\"type\":\"uint24\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"factory\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getHookPermissions\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"beforeInitialize\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"afterInitialize\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"beforeAddLiquidity\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"afterAddLiquidity\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"beforeRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"afterRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"beforeSwap\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"afterSwap\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"beforeDonate\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"afterDonate\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"beforeSwapReturnDelta\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"afterSwapReturnDelta\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"afterAddLiquidityReturnDelta\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"afterRemoveLiquidityReturnDelta\",\"type\":\"bool\"}],\"internalType\":\"structHooks.Permissions\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"poolKey\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"mevModuleData\",\"type\":\"bytes\"}],\"name\":\"initializeMevModule\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"clanker\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"pairedToken\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"tickIfToken0IsClanker\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"address\",\"name\":\"_locker\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_mevModule\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"poolData\",\"type\":\"bytes\"}],\"name\":\"initializePool\",\"outputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"clanker\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"pairedToken\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"tickIfToken0IsClanker\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"bytes\",\"name\":\"poolData\",\"type\":\"bytes\"}],\"name\":\"initializePoolOpen\",\"outputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"mevModule\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"mevModuleEnabled\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"poolConfigVars\",\"outputs\":[{\"components\":[{\"internalType\":\"uint24\",\"name\":\"baseFee\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"maxLpFee\",\"type\":\"uint24\"},{\"internalType\":\"uint256\",\"name\":\"referenceTickFilterPeriod\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"resetPeriod\",\"type\":\"uint256\"},{\"internalType\":\"int24\",\"name\":\"resetTickFilter\",\"type\":\"int24\"},{\"internalType\":\"uint256\",\"name\":\"feeControlNumerator\",\"type\":\"uint256\"},{\"internalType\":\"uint24\",\"name\":\"decayFilterBps\",\"type\":\"uint24\"}],\"internalType\":\"structIClankerHookDynamicFee.PoolDynamicConfigVars\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"poolCreationTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"poolFeeVars\",\"outputs\":[{\"components\":[{\"internalType\":\"int24\",\"name\":\"referenceTick\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"resetTick\",\"type\":\"int24\"},{\"internalType\":\"uint256\",\"name\":\"resetTickTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lastSwapTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint24\",\"name\":\"appliedVR\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"prevVA\",\"type\":\"uint24\"}],\"internalType\":\"structIClankerHookDynamicFee.PoolDynamicFeeVars\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"poolManager\",\"outputs\":[{\"internalType\":\"contractIPoolManager\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"protocolFee\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"poolKey\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountSpecified\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceLimitX96\",\"type\":\"uint160\"}],\"internalType\":\"structIPoolManager.SwapParams\",\"name\":\"swapParams\",\"type\":\"tuple\"}],\"name\":\"simulateSwap\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"weth\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// ClankerABI is the input ABI used to generate the binding from.
// Deprecated: Use ClankerMetaData.ABI instead.
var ClankerABI = ClankerMetaData.ABI

// Clanker is an auto generated Go binding around an Ethereum contract.
type Clanker struct {
	ClankerCaller     // Read-only binding to the contract
	ClankerTransactor // Write-only binding to the contract
	ClankerFilterer   // Log filterer for contract events
}

// ClankerCaller is an auto generated read-only Go binding around an Ethereum contract.
type ClankerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClankerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ClankerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClankerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ClankerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClankerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ClankerSession struct {
	Contract     *Clanker          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ClankerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ClankerCallerSession struct {
	Contract *ClankerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// ClankerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ClankerTransactorSession struct {
	Contract     *ClankerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// ClankerRaw is an auto generated low-level Go binding around an Ethereum contract.
type ClankerRaw struct {
	Contract *Clanker // Generic contract binding to access the raw methods on
}

// ClankerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ClankerCallerRaw struct {
	Contract *ClankerCaller // Generic read-only contract binding to access the raw methods on
}

// ClankerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ClankerTransactorRaw struct {
	Contract *ClankerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewClanker creates a new instance of Clanker, bound to a specific deployed contract.
func NewClanker(address common.Address, backend bind.ContractBackend) (*Clanker, error) {
	contract, err := bindClanker(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Clanker{ClankerCaller: ClankerCaller{contract: contract}, ClankerTransactor: ClankerTransactor{contract: contract}, ClankerFilterer: ClankerFilterer{contract: contract}}, nil
}

// NewClankerCaller creates a new read-only instance of Clanker, bound to a specific deployed contract.
func NewClankerCaller(address common.Address, caller bind.ContractCaller) (*ClankerCaller, error) {
	contract, err := bindClanker(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ClankerCaller{contract: contract}, nil
}

// NewClankerTransactor creates a new write-only instance of Clanker, bound to a specific deployed contract.
func NewClankerTransactor(address common.Address, transactor bind.ContractTransactor) (*ClankerTransactor, error) {
	contract, err := bindClanker(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ClankerTransactor{contract: contract}, nil
}

// NewClankerFilterer creates a new log filterer instance of Clanker, bound to a specific deployed contract.
func NewClankerFilterer(address common.Address, filterer bind.ContractFilterer) (*ClankerFilterer, error) {
	contract, err := bindClanker(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ClankerFilterer{contract: contract}, nil
}

// bindClanker binds a generic wrapper to an already deployed contract.
func bindClanker(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ClankerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Clanker *ClankerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Clanker.Contract.ClankerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Clanker *ClankerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Clanker.Contract.ClankerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Clanker *ClankerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Clanker.Contract.ClankerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Clanker *ClankerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Clanker.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Clanker *ClankerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Clanker.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Clanker *ClankerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Clanker.Contract.contract.Transact(opts, method, params...)
}

// BPSDENOMINATOR is a free data retrieval call binding the contract method 0xe1a45218.
//
// Solidity: function BPS_DENOMINATOR() view returns(uint24)
func (_Clanker *ClankerCaller) BPSDENOMINATOR(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "BPS_DENOMINATOR")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BPSDENOMINATOR is a free data retrieval call binding the contract method 0xe1a45218.
//
// Solidity: function BPS_DENOMINATOR() view returns(uint24)
func (_Clanker *ClankerSession) BPSDENOMINATOR() (*big.Int, error) {
	return _Clanker.Contract.BPSDENOMINATOR(&_Clanker.CallOpts)
}

// BPSDENOMINATOR is a free data retrieval call binding the contract method 0xe1a45218.
//
// Solidity: function BPS_DENOMINATOR() view returns(uint24)
func (_Clanker *ClankerCallerSession) BPSDENOMINATOR() (*big.Int, error) {
	return _Clanker.Contract.BPSDENOMINATOR(&_Clanker.CallOpts)
}

// FEECONTROLDENOMINATOR is a free data retrieval call binding the contract method 0x5339b03b.
//
// Solidity: function FEE_CONTROL_DENOMINATOR() view returns(uint256)
func (_Clanker *ClankerCaller) FEECONTROLDENOMINATOR(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "FEE_CONTROL_DENOMINATOR")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FEECONTROLDENOMINATOR is a free data retrieval call binding the contract method 0x5339b03b.
//
// Solidity: function FEE_CONTROL_DENOMINATOR() view returns(uint256)
func (_Clanker *ClankerSession) FEECONTROLDENOMINATOR() (*big.Int, error) {
	return _Clanker.Contract.FEECONTROLDENOMINATOR(&_Clanker.CallOpts)
}

// FEECONTROLDENOMINATOR is a free data retrieval call binding the contract method 0x5339b03b.
//
// Solidity: function FEE_CONTROL_DENOMINATOR() view returns(uint256)
func (_Clanker *ClankerCallerSession) FEECONTROLDENOMINATOR() (*big.Int, error) {
	return _Clanker.Contract.FEECONTROLDENOMINATOR(&_Clanker.CallOpts)
}

// FEEDENOMINATOR is a free data retrieval call binding the contract method 0xd73792a9.
//
// Solidity: function FEE_DENOMINATOR() view returns(int128)
func (_Clanker *ClankerCaller) FEEDENOMINATOR(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "FEE_DENOMINATOR")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FEEDENOMINATOR is a free data retrieval call binding the contract method 0xd73792a9.
//
// Solidity: function FEE_DENOMINATOR() view returns(int128)
func (_Clanker *ClankerSession) FEEDENOMINATOR() (*big.Int, error) {
	return _Clanker.Contract.FEEDENOMINATOR(&_Clanker.CallOpts)
}

// FEEDENOMINATOR is a free data retrieval call binding the contract method 0xd73792a9.
//
// Solidity: function FEE_DENOMINATOR() view returns(int128)
func (_Clanker *ClankerCallerSession) FEEDENOMINATOR() (*big.Int, error) {
	return _Clanker.Contract.FEEDENOMINATOR(&_Clanker.CallOpts)
}

// MAXDECAYFILTERBPS is a free data retrieval call binding the contract method 0x69b84eb1.
//
// Solidity: function MAX_DECAY_FILTER_BPS() view returns(uint256)
func (_Clanker *ClankerCaller) MAXDECAYFILTERBPS(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "MAX_DECAY_FILTER_BPS")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXDECAYFILTERBPS is a free data retrieval call binding the contract method 0x69b84eb1.
//
// Solidity: function MAX_DECAY_FILTER_BPS() view returns(uint256)
func (_Clanker *ClankerSession) MAXDECAYFILTERBPS() (*big.Int, error) {
	return _Clanker.Contract.MAXDECAYFILTERBPS(&_Clanker.CallOpts)
}

// MAXDECAYFILTERBPS is a free data retrieval call binding the contract method 0x69b84eb1.
//
// Solidity: function MAX_DECAY_FILTER_BPS() view returns(uint256)
func (_Clanker *ClankerCallerSession) MAXDECAYFILTERBPS() (*big.Int, error) {
	return _Clanker.Contract.MAXDECAYFILTERBPS(&_Clanker.CallOpts)
}

// MAXLPFEE is a free data retrieval call binding the contract method 0x3fc48eba.
//
// Solidity: function MAX_LP_FEE() view returns(uint24)
func (_Clanker *ClankerCaller) MAXLPFEE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "MAX_LP_FEE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXLPFEE is a free data retrieval call binding the contract method 0x3fc48eba.
//
// Solidity: function MAX_LP_FEE() view returns(uint24)
func (_Clanker *ClankerSession) MAXLPFEE() (*big.Int, error) {
	return _Clanker.Contract.MAXLPFEE(&_Clanker.CallOpts)
}

// MAXLPFEE is a free data retrieval call binding the contract method 0x3fc48eba.
//
// Solidity: function MAX_LP_FEE() view returns(uint24)
func (_Clanker *ClankerCallerSession) MAXLPFEE() (*big.Int, error) {
	return _Clanker.Contract.MAXLPFEE(&_Clanker.CallOpts)
}

// MAXMEVMODULEDELAY is a free data retrieval call binding the contract method 0xccc8dc43.
//
// Solidity: function MAX_MEV_MODULE_DELAY() view returns(uint256)
func (_Clanker *ClankerCaller) MAXMEVMODULEDELAY(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "MAX_MEV_MODULE_DELAY")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXMEVMODULEDELAY is a free data retrieval call binding the contract method 0xccc8dc43.
//
// Solidity: function MAX_MEV_MODULE_DELAY() view returns(uint256)
func (_Clanker *ClankerSession) MAXMEVMODULEDELAY() (*big.Int, error) {
	return _Clanker.Contract.MAXMEVMODULEDELAY(&_Clanker.CallOpts)
}

// MAXMEVMODULEDELAY is a free data retrieval call binding the contract method 0xccc8dc43.
//
// Solidity: function MAX_MEV_MODULE_DELAY() view returns(uint256)
func (_Clanker *ClankerCallerSession) MAXMEVMODULEDELAY() (*big.Int, error) {
	return _Clanker.Contract.MAXMEVMODULEDELAY(&_Clanker.CallOpts)
}

// MINBASEFEE is a free data retrieval call binding the contract method 0xea78e61a.
//
// Solidity: function MIN_BASE_FEE() view returns(uint24)
func (_Clanker *ClankerCaller) MINBASEFEE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "MIN_BASE_FEE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINBASEFEE is a free data retrieval call binding the contract method 0xea78e61a.
//
// Solidity: function MIN_BASE_FEE() view returns(uint24)
func (_Clanker *ClankerSession) MINBASEFEE() (*big.Int, error) {
	return _Clanker.Contract.MINBASEFEE(&_Clanker.CallOpts)
}

// MINBASEFEE is a free data retrieval call binding the contract method 0xea78e61a.
//
// Solidity: function MIN_BASE_FEE() view returns(uint24)
func (_Clanker *ClankerCallerSession) MINBASEFEE() (*big.Int, error) {
	return _Clanker.Contract.MINBASEFEE(&_Clanker.CallOpts)
}

// PROTOCOLFEENUMERATOR is a free data retrieval call binding the contract method 0x334e8367.
//
// Solidity: function PROTOCOL_FEE_NUMERATOR() view returns(uint256)
func (_Clanker *ClankerCaller) PROTOCOLFEENUMERATOR(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "PROTOCOL_FEE_NUMERATOR")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PROTOCOLFEENUMERATOR is a free data retrieval call binding the contract method 0x334e8367.
//
// Solidity: function PROTOCOL_FEE_NUMERATOR() view returns(uint256)
func (_Clanker *ClankerSession) PROTOCOLFEENUMERATOR() (*big.Int, error) {
	return _Clanker.Contract.PROTOCOLFEENUMERATOR(&_Clanker.CallOpts)
}

// PROTOCOLFEENUMERATOR is a free data retrieval call binding the contract method 0x334e8367.
//
// Solidity: function PROTOCOL_FEE_NUMERATOR() view returns(uint256)
func (_Clanker *ClankerCallerSession) PROTOCOLFEENUMERATOR() (*big.Int, error) {
	return _Clanker.Contract.PROTOCOLFEENUMERATOR(&_Clanker.CallOpts)
}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_Clanker *ClankerCaller) Factory(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "factory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_Clanker *ClankerSession) Factory() (common.Address, error) {
	return _Clanker.Contract.Factory(&_Clanker.CallOpts)
}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_Clanker *ClankerCallerSession) Factory() (common.Address, error) {
	return _Clanker.Contract.Factory(&_Clanker.CallOpts)
}

// GetHookPermissions is a free data retrieval call binding the contract method 0xc4e833ce.
//
// Solidity: function getHookPermissions() pure returns((bool,bool,bool,bool,bool,bool,bool,bool,bool,bool,bool,bool,bool,bool))
func (_Clanker *ClankerCaller) GetHookPermissions(opts *bind.CallOpts) (HooksPermissions, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "getHookPermissions")

	if err != nil {
		return *new(HooksPermissions), err
	}

	out0 := *abi.ConvertType(out[0], new(HooksPermissions)).(*HooksPermissions)

	return out0, err

}

// GetHookPermissions is a free data retrieval call binding the contract method 0xc4e833ce.
//
// Solidity: function getHookPermissions() pure returns((bool,bool,bool,bool,bool,bool,bool,bool,bool,bool,bool,bool,bool,bool))
func (_Clanker *ClankerSession) GetHookPermissions() (HooksPermissions, error) {
	return _Clanker.Contract.GetHookPermissions(&_Clanker.CallOpts)
}

// GetHookPermissions is a free data retrieval call binding the contract method 0xc4e833ce.
//
// Solidity: function getHookPermissions() pure returns((bool,bool,bool,bool,bool,bool,bool,bool,bool,bool,bool,bool,bool,bool))
func (_Clanker *ClankerCallerSession) GetHookPermissions() (HooksPermissions, error) {
	return _Clanker.Contract.GetHookPermissions(&_Clanker.CallOpts)
}

// MevModule is a free data retrieval call binding the contract method 0xbdf22863.
//
// Solidity: function mevModule(bytes32 ) view returns(address)
func (_Clanker *ClankerCaller) MevModule(opts *bind.CallOpts, arg0 [32]byte) (common.Address, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "mevModule", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MevModule is a free data retrieval call binding the contract method 0xbdf22863.
//
// Solidity: function mevModule(bytes32 ) view returns(address)
func (_Clanker *ClankerSession) MevModule(arg0 [32]byte) (common.Address, error) {
	return _Clanker.Contract.MevModule(&_Clanker.CallOpts, arg0)
}

// MevModule is a free data retrieval call binding the contract method 0xbdf22863.
//
// Solidity: function mevModule(bytes32 ) view returns(address)
func (_Clanker *ClankerCallerSession) MevModule(arg0 [32]byte) (common.Address, error) {
	return _Clanker.Contract.MevModule(&_Clanker.CallOpts, arg0)
}

// MevModuleEnabled is a free data retrieval call binding the contract method 0x2b887e0f.
//
// Solidity: function mevModuleEnabled(bytes32 ) view returns(bool)
func (_Clanker *ClankerCaller) MevModuleEnabled(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "mevModuleEnabled", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// MevModuleEnabled is a free data retrieval call binding the contract method 0x2b887e0f.
//
// Solidity: function mevModuleEnabled(bytes32 ) view returns(bool)
func (_Clanker *ClankerSession) MevModuleEnabled(arg0 [32]byte) (bool, error) {
	return _Clanker.Contract.MevModuleEnabled(&_Clanker.CallOpts, arg0)
}

// MevModuleEnabled is a free data retrieval call binding the contract method 0x2b887e0f.
//
// Solidity: function mevModuleEnabled(bytes32 ) view returns(bool)
func (_Clanker *ClankerCallerSession) MevModuleEnabled(arg0 [32]byte) (bool, error) {
	return _Clanker.Contract.MevModuleEnabled(&_Clanker.CallOpts, arg0)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Clanker *ClankerCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Clanker *ClankerSession) Owner() (common.Address, error) {
	return _Clanker.Contract.Owner(&_Clanker.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Clanker *ClankerCallerSession) Owner() (common.Address, error) {
	return _Clanker.Contract.Owner(&_Clanker.CallOpts)
}

// PoolConfigVars is a free data retrieval call binding the contract method 0xca946313.
//
// Solidity: function poolConfigVars(bytes32 poolId) view returns((uint24,uint24,uint256,uint256,int24,uint256,uint24))
func (_Clanker *ClankerCaller) PoolConfigVars(opts *bind.CallOpts, poolId [32]byte) (IClankerHookDynamicFeePoolDynamicConfigVars, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "poolConfigVars", poolId)

	if err != nil {
		return *new(IClankerHookDynamicFeePoolDynamicConfigVars), err
	}

	out0 := *abi.ConvertType(out[0], new(IClankerHookDynamicFeePoolDynamicConfigVars)).(*IClankerHookDynamicFeePoolDynamicConfigVars)

	return out0, err

}

// PoolConfigVars is a free data retrieval call binding the contract method 0xca946313.
//
// Solidity: function poolConfigVars(bytes32 poolId) view returns((uint24,uint24,uint256,uint256,int24,uint256,uint24))
func (_Clanker *ClankerSession) PoolConfigVars(poolId [32]byte) (IClankerHookDynamicFeePoolDynamicConfigVars, error) {
	return _Clanker.Contract.PoolConfigVars(&_Clanker.CallOpts, poolId)
}

// PoolConfigVars is a free data retrieval call binding the contract method 0xca946313.
//
// Solidity: function poolConfigVars(bytes32 poolId) view returns((uint24,uint24,uint256,uint256,int24,uint256,uint24))
func (_Clanker *ClankerCallerSession) PoolConfigVars(poolId [32]byte) (IClankerHookDynamicFeePoolDynamicConfigVars, error) {
	return _Clanker.Contract.PoolConfigVars(&_Clanker.CallOpts, poolId)
}

// PoolCreationTimestamp is a free data retrieval call binding the contract method 0xee4d96cf.
//
// Solidity: function poolCreationTimestamp(bytes32 ) view returns(uint256)
func (_Clanker *ClankerCaller) PoolCreationTimestamp(opts *bind.CallOpts, arg0 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "poolCreationTimestamp", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PoolCreationTimestamp is a free data retrieval call binding the contract method 0xee4d96cf.
//
// Solidity: function poolCreationTimestamp(bytes32 ) view returns(uint256)
func (_Clanker *ClankerSession) PoolCreationTimestamp(arg0 [32]byte) (*big.Int, error) {
	return _Clanker.Contract.PoolCreationTimestamp(&_Clanker.CallOpts, arg0)
}

// PoolCreationTimestamp is a free data retrieval call binding the contract method 0xee4d96cf.
//
// Solidity: function poolCreationTimestamp(bytes32 ) view returns(uint256)
func (_Clanker *ClankerCallerSession) PoolCreationTimestamp(arg0 [32]byte) (*big.Int, error) {
	return _Clanker.Contract.PoolCreationTimestamp(&_Clanker.CallOpts, arg0)
}

// PoolFeeVars is a free data retrieval call binding the contract method 0x844dbde6.
//
// Solidity: function poolFeeVars(bytes32 poolId) view returns((int24,int24,uint256,uint256,uint24,uint24))
func (_Clanker *ClankerCaller) PoolFeeVars(opts *bind.CallOpts, poolId [32]byte) (IClankerHookDynamicFeePoolDynamicFeeVars, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "poolFeeVars", poolId)

	if err != nil {
		return *new(IClankerHookDynamicFeePoolDynamicFeeVars), err
	}

	out0 := *abi.ConvertType(out[0], new(IClankerHookDynamicFeePoolDynamicFeeVars)).(*IClankerHookDynamicFeePoolDynamicFeeVars)

	return out0, err

}

// PoolFeeVars is a free data retrieval call binding the contract method 0x844dbde6.
//
// Solidity: function poolFeeVars(bytes32 poolId) view returns((int24,int24,uint256,uint256,uint24,uint24))
func (_Clanker *ClankerSession) PoolFeeVars(poolId [32]byte) (IClankerHookDynamicFeePoolDynamicFeeVars, error) {
	return _Clanker.Contract.PoolFeeVars(&_Clanker.CallOpts, poolId)
}

// PoolFeeVars is a free data retrieval call binding the contract method 0x844dbde6.
//
// Solidity: function poolFeeVars(bytes32 poolId) view returns((int24,int24,uint256,uint256,uint24,uint24))
func (_Clanker *ClankerCallerSession) PoolFeeVars(poolId [32]byte) (IClankerHookDynamicFeePoolDynamicFeeVars, error) {
	return _Clanker.Contract.PoolFeeVars(&_Clanker.CallOpts, poolId)
}

// PoolManager is a free data retrieval call binding the contract method 0xdc4c90d3.
//
// Solidity: function poolManager() view returns(address)
func (_Clanker *ClankerCaller) PoolManager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "poolManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PoolManager is a free data retrieval call binding the contract method 0xdc4c90d3.
//
// Solidity: function poolManager() view returns(address)
func (_Clanker *ClankerSession) PoolManager() (common.Address, error) {
	return _Clanker.Contract.PoolManager(&_Clanker.CallOpts)
}

// PoolManager is a free data retrieval call binding the contract method 0xdc4c90d3.
//
// Solidity: function poolManager() view returns(address)
func (_Clanker *ClankerCallerSession) PoolManager() (common.Address, error) {
	return _Clanker.Contract.PoolManager(&_Clanker.CallOpts)
}

// ProtocolFee is a free data retrieval call binding the contract method 0xb0e21e8a.
//
// Solidity: function protocolFee() view returns(uint24)
func (_Clanker *ClankerCaller) ProtocolFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "protocolFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ProtocolFee is a free data retrieval call binding the contract method 0xb0e21e8a.
//
// Solidity: function protocolFee() view returns(uint24)
func (_Clanker *ClankerSession) ProtocolFee() (*big.Int, error) {
	return _Clanker.Contract.ProtocolFee(&_Clanker.CallOpts)
}

// ProtocolFee is a free data retrieval call binding the contract method 0xb0e21e8a.
//
// Solidity: function protocolFee() view returns(uint24)
func (_Clanker *ClankerCallerSession) ProtocolFee() (*big.Int, error) {
	return _Clanker.Contract.ProtocolFee(&_Clanker.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) pure returns(bool)
func (_Clanker *ClankerCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) pure returns(bool)
func (_Clanker *ClankerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Clanker.Contract.SupportsInterface(&_Clanker.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) pure returns(bool)
func (_Clanker *ClankerCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Clanker.Contract.SupportsInterface(&_Clanker.CallOpts, interfaceId)
}

// Weth is a free data retrieval call binding the contract method 0x3fc8cef3.
//
// Solidity: function weth() view returns(address)
func (_Clanker *ClankerCaller) Weth(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Clanker.contract.Call(opts, &out, "weth")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Weth is a free data retrieval call binding the contract method 0x3fc8cef3.
//
// Solidity: function weth() view returns(address)
func (_Clanker *ClankerSession) Weth() (common.Address, error) {
	return _Clanker.Contract.Weth(&_Clanker.CallOpts)
}

// Weth is a free data retrieval call binding the contract method 0x3fc8cef3.
//
// Solidity: function weth() view returns(address)
func (_Clanker *ClankerCallerSession) Weth() (common.Address, error) {
	return _Clanker.Contract.Weth(&_Clanker.CallOpts)
}

// AfterAddLiquidity is a paid mutator transaction binding the contract method 0x9f063efc.
//
// Solidity: function afterAddLiquidity(address sender, (address,address,uint24,int24,address) key, (int24,int24,int256,bytes32) params, int256 delta, int256 feesAccrued, bytes hookData) returns(bytes4, int256)
func (_Clanker *ClankerTransactor) AfterAddLiquidity(opts *bind.TransactOpts, sender common.Address, key PoolKey, params IPoolManagerModifyLiquidityParams, delta *big.Int, feesAccrued *big.Int, hookData []byte) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "afterAddLiquidity", sender, key, params, delta, feesAccrued, hookData)
}

// AfterAddLiquidity is a paid mutator transaction binding the contract method 0x9f063efc.
//
// Solidity: function afterAddLiquidity(address sender, (address,address,uint24,int24,address) key, (int24,int24,int256,bytes32) params, int256 delta, int256 feesAccrued, bytes hookData) returns(bytes4, int256)
func (_Clanker *ClankerSession) AfterAddLiquidity(sender common.Address, key PoolKey, params IPoolManagerModifyLiquidityParams, delta *big.Int, feesAccrued *big.Int, hookData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.AfterAddLiquidity(&_Clanker.TransactOpts, sender, key, params, delta, feesAccrued, hookData)
}

// AfterAddLiquidity is a paid mutator transaction binding the contract method 0x9f063efc.
//
// Solidity: function afterAddLiquidity(address sender, (address,address,uint24,int24,address) key, (int24,int24,int256,bytes32) params, int256 delta, int256 feesAccrued, bytes hookData) returns(bytes4, int256)
func (_Clanker *ClankerTransactorSession) AfterAddLiquidity(sender common.Address, key PoolKey, params IPoolManagerModifyLiquidityParams, delta *big.Int, feesAccrued *big.Int, hookData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.AfterAddLiquidity(&_Clanker.TransactOpts, sender, key, params, delta, feesAccrued, hookData)
}

// AfterDonate is a paid mutator transaction binding the contract method 0xe1b4af69.
//
// Solidity: function afterDonate(address sender, (address,address,uint24,int24,address) key, uint256 amount0, uint256 amount1, bytes hookData) returns(bytes4)
func (_Clanker *ClankerTransactor) AfterDonate(opts *bind.TransactOpts, sender common.Address, key PoolKey, amount0 *big.Int, amount1 *big.Int, hookData []byte) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "afterDonate", sender, key, amount0, amount1, hookData)
}

// AfterDonate is a paid mutator transaction binding the contract method 0xe1b4af69.
//
// Solidity: function afterDonate(address sender, (address,address,uint24,int24,address) key, uint256 amount0, uint256 amount1, bytes hookData) returns(bytes4)
func (_Clanker *ClankerSession) AfterDonate(sender common.Address, key PoolKey, amount0 *big.Int, amount1 *big.Int, hookData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.AfterDonate(&_Clanker.TransactOpts, sender, key, amount0, amount1, hookData)
}

// AfterDonate is a paid mutator transaction binding the contract method 0xe1b4af69.
//
// Solidity: function afterDonate(address sender, (address,address,uint24,int24,address) key, uint256 amount0, uint256 amount1, bytes hookData) returns(bytes4)
func (_Clanker *ClankerTransactorSession) AfterDonate(sender common.Address, key PoolKey, amount0 *big.Int, amount1 *big.Int, hookData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.AfterDonate(&_Clanker.TransactOpts, sender, key, amount0, amount1, hookData)
}

// AfterInitialize is a paid mutator transaction binding the contract method 0x6fe7e6eb.
//
// Solidity: function afterInitialize(address sender, (address,address,uint24,int24,address) key, uint160 sqrtPriceX96, int24 tick) returns(bytes4)
func (_Clanker *ClankerTransactor) AfterInitialize(opts *bind.TransactOpts, sender common.Address, key PoolKey, sqrtPriceX96 *big.Int, tick *big.Int) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "afterInitialize", sender, key, sqrtPriceX96, tick)
}

// AfterInitialize is a paid mutator transaction binding the contract method 0x6fe7e6eb.
//
// Solidity: function afterInitialize(address sender, (address,address,uint24,int24,address) key, uint160 sqrtPriceX96, int24 tick) returns(bytes4)
func (_Clanker *ClankerSession) AfterInitialize(sender common.Address, key PoolKey, sqrtPriceX96 *big.Int, tick *big.Int) (*types.Transaction, error) {
	return _Clanker.Contract.AfterInitialize(&_Clanker.TransactOpts, sender, key, sqrtPriceX96, tick)
}

// AfterInitialize is a paid mutator transaction binding the contract method 0x6fe7e6eb.
//
// Solidity: function afterInitialize(address sender, (address,address,uint24,int24,address) key, uint160 sqrtPriceX96, int24 tick) returns(bytes4)
func (_Clanker *ClankerTransactorSession) AfterInitialize(sender common.Address, key PoolKey, sqrtPriceX96 *big.Int, tick *big.Int) (*types.Transaction, error) {
	return _Clanker.Contract.AfterInitialize(&_Clanker.TransactOpts, sender, key, sqrtPriceX96, tick)
}

// AfterRemoveLiquidity is a paid mutator transaction binding the contract method 0x6c2bbe7e.
//
// Solidity: function afterRemoveLiquidity(address sender, (address,address,uint24,int24,address) key, (int24,int24,int256,bytes32) params, int256 delta, int256 feesAccrued, bytes hookData) returns(bytes4, int256)
func (_Clanker *ClankerTransactor) AfterRemoveLiquidity(opts *bind.TransactOpts, sender common.Address, key PoolKey, params IPoolManagerModifyLiquidityParams, delta *big.Int, feesAccrued *big.Int, hookData []byte) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "afterRemoveLiquidity", sender, key, params, delta, feesAccrued, hookData)
}

// AfterRemoveLiquidity is a paid mutator transaction binding the contract method 0x6c2bbe7e.
//
// Solidity: function afterRemoveLiquidity(address sender, (address,address,uint24,int24,address) key, (int24,int24,int256,bytes32) params, int256 delta, int256 feesAccrued, bytes hookData) returns(bytes4, int256)
func (_Clanker *ClankerSession) AfterRemoveLiquidity(sender common.Address, key PoolKey, params IPoolManagerModifyLiquidityParams, delta *big.Int, feesAccrued *big.Int, hookData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.AfterRemoveLiquidity(&_Clanker.TransactOpts, sender, key, params, delta, feesAccrued, hookData)
}

// AfterRemoveLiquidity is a paid mutator transaction binding the contract method 0x6c2bbe7e.
//
// Solidity: function afterRemoveLiquidity(address sender, (address,address,uint24,int24,address) key, (int24,int24,int256,bytes32) params, int256 delta, int256 feesAccrued, bytes hookData) returns(bytes4, int256)
func (_Clanker *ClankerTransactorSession) AfterRemoveLiquidity(sender common.Address, key PoolKey, params IPoolManagerModifyLiquidityParams, delta *big.Int, feesAccrued *big.Int, hookData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.AfterRemoveLiquidity(&_Clanker.TransactOpts, sender, key, params, delta, feesAccrued, hookData)
}

// AfterSwap is a paid mutator transaction binding the contract method 0xb47b2fb1.
//
// Solidity: function afterSwap(address sender, (address,address,uint24,int24,address) key, (bool,int256,uint160) params, int256 delta, bytes hookData) returns(bytes4, int128)
func (_Clanker *ClankerTransactor) AfterSwap(opts *bind.TransactOpts, sender common.Address, key PoolKey, params IPoolManagerSwapParams, delta *big.Int, hookData []byte) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "afterSwap", sender, key, params, delta, hookData)
}

// AfterSwap is a paid mutator transaction binding the contract method 0xb47b2fb1.
//
// Solidity: function afterSwap(address sender, (address,address,uint24,int24,address) key, (bool,int256,uint160) params, int256 delta, bytes hookData) returns(bytes4, int128)
func (_Clanker *ClankerSession) AfterSwap(sender common.Address, key PoolKey, params IPoolManagerSwapParams, delta *big.Int, hookData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.AfterSwap(&_Clanker.TransactOpts, sender, key, params, delta, hookData)
}

// AfterSwap is a paid mutator transaction binding the contract method 0xb47b2fb1.
//
// Solidity: function afterSwap(address sender, (address,address,uint24,int24,address) key, (bool,int256,uint160) params, int256 delta, bytes hookData) returns(bytes4, int128)
func (_Clanker *ClankerTransactorSession) AfterSwap(sender common.Address, key PoolKey, params IPoolManagerSwapParams, delta *big.Int, hookData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.AfterSwap(&_Clanker.TransactOpts, sender, key, params, delta, hookData)
}

// BeforeAddLiquidity is a paid mutator transaction binding the contract method 0x259982e5.
//
// Solidity: function beforeAddLiquidity(address sender, (address,address,uint24,int24,address) key, (int24,int24,int256,bytes32) params, bytes hookData) returns(bytes4)
func (_Clanker *ClankerTransactor) BeforeAddLiquidity(opts *bind.TransactOpts, sender common.Address, key PoolKey, params IPoolManagerModifyLiquidityParams, hookData []byte) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "beforeAddLiquidity", sender, key, params, hookData)
}

// BeforeAddLiquidity is a paid mutator transaction binding the contract method 0x259982e5.
//
// Solidity: function beforeAddLiquidity(address sender, (address,address,uint24,int24,address) key, (int24,int24,int256,bytes32) params, bytes hookData) returns(bytes4)
func (_Clanker *ClankerSession) BeforeAddLiquidity(sender common.Address, key PoolKey, params IPoolManagerModifyLiquidityParams, hookData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.BeforeAddLiquidity(&_Clanker.TransactOpts, sender, key, params, hookData)
}

// BeforeAddLiquidity is a paid mutator transaction binding the contract method 0x259982e5.
//
// Solidity: function beforeAddLiquidity(address sender, (address,address,uint24,int24,address) key, (int24,int24,int256,bytes32) params, bytes hookData) returns(bytes4)
func (_Clanker *ClankerTransactorSession) BeforeAddLiquidity(sender common.Address, key PoolKey, params IPoolManagerModifyLiquidityParams, hookData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.BeforeAddLiquidity(&_Clanker.TransactOpts, sender, key, params, hookData)
}

// BeforeDonate is a paid mutator transaction binding the contract method 0xb6a8b0fa.
//
// Solidity: function beforeDonate(address sender, (address,address,uint24,int24,address) key, uint256 amount0, uint256 amount1, bytes hookData) returns(bytes4)
func (_Clanker *ClankerTransactor) BeforeDonate(opts *bind.TransactOpts, sender common.Address, key PoolKey, amount0 *big.Int, amount1 *big.Int, hookData []byte) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "beforeDonate", sender, key, amount0, amount1, hookData)
}

// BeforeDonate is a paid mutator transaction binding the contract method 0xb6a8b0fa.
//
// Solidity: function beforeDonate(address sender, (address,address,uint24,int24,address) key, uint256 amount0, uint256 amount1, bytes hookData) returns(bytes4)
func (_Clanker *ClankerSession) BeforeDonate(sender common.Address, key PoolKey, amount0 *big.Int, amount1 *big.Int, hookData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.BeforeDonate(&_Clanker.TransactOpts, sender, key, amount0, amount1, hookData)
}

// BeforeDonate is a paid mutator transaction binding the contract method 0xb6a8b0fa.
//
// Solidity: function beforeDonate(address sender, (address,address,uint24,int24,address) key, uint256 amount0, uint256 amount1, bytes hookData) returns(bytes4)
func (_Clanker *ClankerTransactorSession) BeforeDonate(sender common.Address, key PoolKey, amount0 *big.Int, amount1 *big.Int, hookData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.BeforeDonate(&_Clanker.TransactOpts, sender, key, amount0, amount1, hookData)
}

// BeforeInitialize is a paid mutator transaction binding the contract method 0xdc98354e.
//
// Solidity: function beforeInitialize(address sender, (address,address,uint24,int24,address) key, uint160 sqrtPriceX96) returns(bytes4)
func (_Clanker *ClankerTransactor) BeforeInitialize(opts *bind.TransactOpts, sender common.Address, key PoolKey, sqrtPriceX96 *big.Int) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "beforeInitialize", sender, key, sqrtPriceX96)
}

// BeforeInitialize is a paid mutator transaction binding the contract method 0xdc98354e.
//
// Solidity: function beforeInitialize(address sender, (address,address,uint24,int24,address) key, uint160 sqrtPriceX96) returns(bytes4)
func (_Clanker *ClankerSession) BeforeInitialize(sender common.Address, key PoolKey, sqrtPriceX96 *big.Int) (*types.Transaction, error) {
	return _Clanker.Contract.BeforeInitialize(&_Clanker.TransactOpts, sender, key, sqrtPriceX96)
}

// BeforeInitialize is a paid mutator transaction binding the contract method 0xdc98354e.
//
// Solidity: function beforeInitialize(address sender, (address,address,uint24,int24,address) key, uint160 sqrtPriceX96) returns(bytes4)
func (_Clanker *ClankerTransactorSession) BeforeInitialize(sender common.Address, key PoolKey, sqrtPriceX96 *big.Int) (*types.Transaction, error) {
	return _Clanker.Contract.BeforeInitialize(&_Clanker.TransactOpts, sender, key, sqrtPriceX96)
}

// BeforeRemoveLiquidity is a paid mutator transaction binding the contract method 0x21d0ee70.
//
// Solidity: function beforeRemoveLiquidity(address sender, (address,address,uint24,int24,address) key, (int24,int24,int256,bytes32) params, bytes hookData) returns(bytes4)
func (_Clanker *ClankerTransactor) BeforeRemoveLiquidity(opts *bind.TransactOpts, sender common.Address, key PoolKey, params IPoolManagerModifyLiquidityParams, hookData []byte) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "beforeRemoveLiquidity", sender, key, params, hookData)
}

// BeforeRemoveLiquidity is a paid mutator transaction binding the contract method 0x21d0ee70.
//
// Solidity: function beforeRemoveLiquidity(address sender, (address,address,uint24,int24,address) key, (int24,int24,int256,bytes32) params, bytes hookData) returns(bytes4)
func (_Clanker *ClankerSession) BeforeRemoveLiquidity(sender common.Address, key PoolKey, params IPoolManagerModifyLiquidityParams, hookData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.BeforeRemoveLiquidity(&_Clanker.TransactOpts, sender, key, params, hookData)
}

// BeforeRemoveLiquidity is a paid mutator transaction binding the contract method 0x21d0ee70.
//
// Solidity: function beforeRemoveLiquidity(address sender, (address,address,uint24,int24,address) key, (int24,int24,int256,bytes32) params, bytes hookData) returns(bytes4)
func (_Clanker *ClankerTransactorSession) BeforeRemoveLiquidity(sender common.Address, key PoolKey, params IPoolManagerModifyLiquidityParams, hookData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.BeforeRemoveLiquidity(&_Clanker.TransactOpts, sender, key, params, hookData)
}

// BeforeSwap is a paid mutator transaction binding the contract method 0x575e24b4.
//
// Solidity: function beforeSwap(address sender, (address,address,uint24,int24,address) key, (bool,int256,uint160) params, bytes hookData) returns(bytes4, int256, uint24)
func (_Clanker *ClankerTransactor) BeforeSwap(opts *bind.TransactOpts, sender common.Address, key PoolKey, params IPoolManagerSwapParams, hookData []byte) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "beforeSwap", sender, key, params, hookData)
}

// BeforeSwap is a paid mutator transaction binding the contract method 0x575e24b4.
//
// Solidity: function beforeSwap(address sender, (address,address,uint24,int24,address) key, (bool,int256,uint160) params, bytes hookData) returns(bytes4, int256, uint24)
func (_Clanker *ClankerSession) BeforeSwap(sender common.Address, key PoolKey, params IPoolManagerSwapParams, hookData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.BeforeSwap(&_Clanker.TransactOpts, sender, key, params, hookData)
}

// BeforeSwap is a paid mutator transaction binding the contract method 0x575e24b4.
//
// Solidity: function beforeSwap(address sender, (address,address,uint24,int24,address) key, (bool,int256,uint160) params, bytes hookData) returns(bytes4, int256, uint24)
func (_Clanker *ClankerTransactorSession) BeforeSwap(sender common.Address, key PoolKey, params IPoolManagerSwapParams, hookData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.BeforeSwap(&_Clanker.TransactOpts, sender, key, params, hookData)
}

// InitializeMevModule is a paid mutator transaction binding the contract method 0x015e3c29.
//
// Solidity: function initializeMevModule((address,address,uint24,int24,address) poolKey, bytes mevModuleData) returns()
func (_Clanker *ClankerTransactor) InitializeMevModule(opts *bind.TransactOpts, poolKey PoolKey, mevModuleData []byte) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "initializeMevModule", poolKey, mevModuleData)
}

// InitializeMevModule is a paid mutator transaction binding the contract method 0x015e3c29.
//
// Solidity: function initializeMevModule((address,address,uint24,int24,address) poolKey, bytes mevModuleData) returns()
func (_Clanker *ClankerSession) InitializeMevModule(poolKey PoolKey, mevModuleData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.InitializeMevModule(&_Clanker.TransactOpts, poolKey, mevModuleData)
}

// InitializeMevModule is a paid mutator transaction binding the contract method 0x015e3c29.
//
// Solidity: function initializeMevModule((address,address,uint24,int24,address) poolKey, bytes mevModuleData) returns()
func (_Clanker *ClankerTransactorSession) InitializeMevModule(poolKey PoolKey, mevModuleData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.InitializeMevModule(&_Clanker.TransactOpts, poolKey, mevModuleData)
}

// InitializePool is a paid mutator transaction binding the contract method 0x564d49e7.
//
// Solidity: function initializePool(address clanker, address pairedToken, int24 tickIfToken0IsClanker, int24 tickSpacing, address _locker, address _mevModule, bytes poolData) returns((address,address,uint24,int24,address))
func (_Clanker *ClankerTransactor) InitializePool(opts *bind.TransactOpts, clanker common.Address, pairedToken common.Address, tickIfToken0IsClanker *big.Int, tickSpacing *big.Int, _locker common.Address, _mevModule common.Address, poolData []byte) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "initializePool", clanker, pairedToken, tickIfToken0IsClanker, tickSpacing, _locker, _mevModule, poolData)
}

// InitializePool is a paid mutator transaction binding the contract method 0x564d49e7.
//
// Solidity: function initializePool(address clanker, address pairedToken, int24 tickIfToken0IsClanker, int24 tickSpacing, address _locker, address _mevModule, bytes poolData) returns((address,address,uint24,int24,address))
func (_Clanker *ClankerSession) InitializePool(clanker common.Address, pairedToken common.Address, tickIfToken0IsClanker *big.Int, tickSpacing *big.Int, _locker common.Address, _mevModule common.Address, poolData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.InitializePool(&_Clanker.TransactOpts, clanker, pairedToken, tickIfToken0IsClanker, tickSpacing, _locker, _mevModule, poolData)
}

// InitializePool is a paid mutator transaction binding the contract method 0x564d49e7.
//
// Solidity: function initializePool(address clanker, address pairedToken, int24 tickIfToken0IsClanker, int24 tickSpacing, address _locker, address _mevModule, bytes poolData) returns((address,address,uint24,int24,address))
func (_Clanker *ClankerTransactorSession) InitializePool(clanker common.Address, pairedToken common.Address, tickIfToken0IsClanker *big.Int, tickSpacing *big.Int, _locker common.Address, _mevModule common.Address, poolData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.InitializePool(&_Clanker.TransactOpts, clanker, pairedToken, tickIfToken0IsClanker, tickSpacing, _locker, _mevModule, poolData)
}

// InitializePoolOpen is a paid mutator transaction binding the contract method 0x4c2b2876.
//
// Solidity: function initializePoolOpen(address clanker, address pairedToken, int24 tickIfToken0IsClanker, int24 tickSpacing, bytes poolData) returns((address,address,uint24,int24,address))
func (_Clanker *ClankerTransactor) InitializePoolOpen(opts *bind.TransactOpts, clanker common.Address, pairedToken common.Address, tickIfToken0IsClanker *big.Int, tickSpacing *big.Int, poolData []byte) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "initializePoolOpen", clanker, pairedToken, tickIfToken0IsClanker, tickSpacing, poolData)
}

// InitializePoolOpen is a paid mutator transaction binding the contract method 0x4c2b2876.
//
// Solidity: function initializePoolOpen(address clanker, address pairedToken, int24 tickIfToken0IsClanker, int24 tickSpacing, bytes poolData) returns((address,address,uint24,int24,address))
func (_Clanker *ClankerSession) InitializePoolOpen(clanker common.Address, pairedToken common.Address, tickIfToken0IsClanker *big.Int, tickSpacing *big.Int, poolData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.InitializePoolOpen(&_Clanker.TransactOpts, clanker, pairedToken, tickIfToken0IsClanker, tickSpacing, poolData)
}

// InitializePoolOpen is a paid mutator transaction binding the contract method 0x4c2b2876.
//
// Solidity: function initializePoolOpen(address clanker, address pairedToken, int24 tickIfToken0IsClanker, int24 tickSpacing, bytes poolData) returns((address,address,uint24,int24,address))
func (_Clanker *ClankerTransactorSession) InitializePoolOpen(clanker common.Address, pairedToken common.Address, tickIfToken0IsClanker *big.Int, tickSpacing *big.Int, poolData []byte) (*types.Transaction, error) {
	return _Clanker.Contract.InitializePoolOpen(&_Clanker.TransactOpts, clanker, pairedToken, tickIfToken0IsClanker, tickSpacing, poolData)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Clanker *ClankerTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Clanker *ClankerSession) RenounceOwnership() (*types.Transaction, error) {
	return _Clanker.Contract.RenounceOwnership(&_Clanker.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Clanker *ClankerTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Clanker.Contract.RenounceOwnership(&_Clanker.TransactOpts)
}

// SimulateSwap is a paid mutator transaction binding the contract method 0xb3066515.
//
// Solidity: function simulateSwap((address,address,uint24,int24,address) poolKey, (bool,int256,uint160) swapParams) returns()
func (_Clanker *ClankerTransactor) SimulateSwap(opts *bind.TransactOpts, poolKey PoolKey, swapParams IPoolManagerSwapParams) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "simulateSwap", poolKey, swapParams)
}

// SimulateSwap is a paid mutator transaction binding the contract method 0xb3066515.
//
// Solidity: function simulateSwap((address,address,uint24,int24,address) poolKey, (bool,int256,uint160) swapParams) returns()
func (_Clanker *ClankerSession) SimulateSwap(poolKey PoolKey, swapParams IPoolManagerSwapParams) (*types.Transaction, error) {
	return _Clanker.Contract.SimulateSwap(&_Clanker.TransactOpts, poolKey, swapParams)
}

// SimulateSwap is a paid mutator transaction binding the contract method 0xb3066515.
//
// Solidity: function simulateSwap((address,address,uint24,int24,address) poolKey, (bool,int256,uint160) swapParams) returns()
func (_Clanker *ClankerTransactorSession) SimulateSwap(poolKey PoolKey, swapParams IPoolManagerSwapParams) (*types.Transaction, error) {
	return _Clanker.Contract.SimulateSwap(&_Clanker.TransactOpts, poolKey, swapParams)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Clanker *ClankerTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Clanker.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Clanker *ClankerSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Clanker.Contract.TransferOwnership(&_Clanker.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Clanker *ClankerTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Clanker.Contract.TransferOwnership(&_Clanker.TransactOpts, newOwner)
}

// ClankerClaimProtocolFeesIterator is returned from FilterClaimProtocolFees and is used to iterate over the raw logs and unpacked data for ClaimProtocolFees events raised by the Clanker contract.
type ClankerClaimProtocolFeesIterator struct {
	Event *ClankerClaimProtocolFees // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ClankerClaimProtocolFeesIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerClaimProtocolFees)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ClankerClaimProtocolFees)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ClankerClaimProtocolFeesIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerClaimProtocolFeesIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerClaimProtocolFees represents a ClaimProtocolFees event raised by the Clanker contract.
type ClankerClaimProtocolFees struct {
	Token  common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterClaimProtocolFees is a free log retrieval operation binding the contract event 0x175b790d44599ca70432cc8d1406504cb3a28fc13ff995c06dde6663412b211a.
//
// Solidity: event ClaimProtocolFees(address indexed token, uint256 amount)
func (_Clanker *ClankerFilterer) FilterClaimProtocolFees(opts *bind.FilterOpts, token []common.Address) (*ClankerClaimProtocolFeesIterator, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "ClaimProtocolFees", tokenRule)
	if err != nil {
		return nil, err
	}
	return &ClankerClaimProtocolFeesIterator{contract: _Clanker.contract, event: "ClaimProtocolFees", logs: logs, sub: sub}, nil
}

// WatchClaimProtocolFees is a free log subscription operation binding the contract event 0x175b790d44599ca70432cc8d1406504cb3a28fc13ff995c06dde6663412b211a.
//
// Solidity: event ClaimProtocolFees(address indexed token, uint256 amount)
func (_Clanker *ClankerFilterer) WatchClaimProtocolFees(opts *bind.WatchOpts, sink chan<- *ClankerClaimProtocolFees, token []common.Address) (event.Subscription, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "ClaimProtocolFees", tokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerClaimProtocolFees)
				if err := _Clanker.contract.UnpackLog(event, "ClaimProtocolFees", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseClaimProtocolFees is a log parse operation binding the contract event 0x175b790d44599ca70432cc8d1406504cb3a28fc13ff995c06dde6663412b211a.
//
// Solidity: event ClaimProtocolFees(address indexed token, uint256 amount)
func (_Clanker *ClankerFilterer) ParseClaimProtocolFees(log types.Log) (*ClankerClaimProtocolFees, error) {
	event := new(ClankerClaimProtocolFees)
	if err := _Clanker.contract.UnpackLog(event, "ClaimProtocolFees", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClankerEstimatedTickDifferenceIterator is returned from FilterEstimatedTickDifference and is used to iterate over the raw logs and unpacked data for EstimatedTickDifference events raised by the Clanker contract.
type ClankerEstimatedTickDifferenceIterator struct {
	Event *ClankerEstimatedTickDifference // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ClankerEstimatedTickDifferenceIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerEstimatedTickDifference)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ClankerEstimatedTickDifference)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ClankerEstimatedTickDifferenceIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerEstimatedTickDifferenceIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerEstimatedTickDifference represents a EstimatedTickDifference event raised by the Clanker contract.
type ClankerEstimatedTickDifference struct {
	BeforeTick *big.Int
	AfterTick  *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterEstimatedTickDifference is a free log retrieval operation binding the contract event 0x98185ee38983c6cdd0d24b51d3401877d4098922144329a08d2ce3cbd38e49a3.
//
// Solidity: event EstimatedTickDifference(int24 beforeTick, int24 afterTick)
func (_Clanker *ClankerFilterer) FilterEstimatedTickDifference(opts *bind.FilterOpts) (*ClankerEstimatedTickDifferenceIterator, error) {

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "EstimatedTickDifference")
	if err != nil {
		return nil, err
	}
	return &ClankerEstimatedTickDifferenceIterator{contract: _Clanker.contract, event: "EstimatedTickDifference", logs: logs, sub: sub}, nil
}

// WatchEstimatedTickDifference is a free log subscription operation binding the contract event 0x98185ee38983c6cdd0d24b51d3401877d4098922144329a08d2ce3cbd38e49a3.
//
// Solidity: event EstimatedTickDifference(int24 beforeTick, int24 afterTick)
func (_Clanker *ClankerFilterer) WatchEstimatedTickDifference(opts *bind.WatchOpts, sink chan<- *ClankerEstimatedTickDifference) (event.Subscription, error) {

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "EstimatedTickDifference")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerEstimatedTickDifference)
				if err := _Clanker.contract.UnpackLog(event, "EstimatedTickDifference", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseEstimatedTickDifference is a log parse operation binding the contract event 0x98185ee38983c6cdd0d24b51d3401877d4098922144329a08d2ce3cbd38e49a3.
//
// Solidity: event EstimatedTickDifference(int24 beforeTick, int24 afterTick)
func (_Clanker *ClankerFilterer) ParseEstimatedTickDifference(log types.Log) (*ClankerEstimatedTickDifference, error) {
	event := new(ClankerEstimatedTickDifference)
	if err := _Clanker.contract.UnpackLog(event, "EstimatedTickDifference", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClankerMevModuleDisabledIterator is returned from FilterMevModuleDisabled and is used to iterate over the raw logs and unpacked data for MevModuleDisabled events raised by the Clanker contract.
type ClankerMevModuleDisabledIterator struct {
	Event *ClankerMevModuleDisabled // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ClankerMevModuleDisabledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerMevModuleDisabled)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ClankerMevModuleDisabled)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ClankerMevModuleDisabledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerMevModuleDisabledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerMevModuleDisabled represents a MevModuleDisabled event raised by the Clanker contract.
type ClankerMevModuleDisabled struct {
	Arg0 [32]byte
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterMevModuleDisabled is a free log retrieval operation binding the contract event 0x5fd57032db0db564bdfe22ff91033c11d319a1e6b3677e723545a7ba3e304b93.
//
// Solidity: event MevModuleDisabled(bytes32 arg0)
func (_Clanker *ClankerFilterer) FilterMevModuleDisabled(opts *bind.FilterOpts) (*ClankerMevModuleDisabledIterator, error) {

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "MevModuleDisabled")
	if err != nil {
		return nil, err
	}
	return &ClankerMevModuleDisabledIterator{contract: _Clanker.contract, event: "MevModuleDisabled", logs: logs, sub: sub}, nil
}

// WatchMevModuleDisabled is a free log subscription operation binding the contract event 0x5fd57032db0db564bdfe22ff91033c11d319a1e6b3677e723545a7ba3e304b93.
//
// Solidity: event MevModuleDisabled(bytes32 arg0)
func (_Clanker *ClankerFilterer) WatchMevModuleDisabled(opts *bind.WatchOpts, sink chan<- *ClankerMevModuleDisabled) (event.Subscription, error) {

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "MevModuleDisabled")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerMevModuleDisabled)
				if err := _Clanker.contract.UnpackLog(event, "MevModuleDisabled", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseMevModuleDisabled is a log parse operation binding the contract event 0x5fd57032db0db564bdfe22ff91033c11d319a1e6b3677e723545a7ba3e304b93.
//
// Solidity: event MevModuleDisabled(bytes32 arg0)
func (_Clanker *ClankerFilterer) ParseMevModuleDisabled(log types.Log) (*ClankerMevModuleDisabled, error) {
	event := new(ClankerMevModuleDisabled)
	if err := _Clanker.contract.UnpackLog(event, "MevModuleDisabled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClankerOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Clanker contract.
type ClankerOwnershipTransferredIterator struct {
	Event *ClankerOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ClankerOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ClankerOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ClankerOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerOwnershipTransferred represents a OwnershipTransferred event raised by the Clanker contract.
type ClankerOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Clanker *ClankerFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*ClankerOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ClankerOwnershipTransferredIterator{contract: _Clanker.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Clanker *ClankerFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *ClankerOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerOwnershipTransferred)
				if err := _Clanker.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Clanker *ClankerFilterer) ParseOwnershipTransferred(log types.Log) (*ClankerOwnershipTransferred, error) {
	event := new(ClankerOwnershipTransferred)
	if err := _Clanker.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClankerPoolCreatedFactoryIterator is returned from FilterPoolCreatedFactory and is used to iterate over the raw logs and unpacked data for PoolCreatedFactory events raised by the Clanker contract.
type ClankerPoolCreatedFactoryIterator struct {
	Event *ClankerPoolCreatedFactory // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ClankerPoolCreatedFactoryIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerPoolCreatedFactory)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ClankerPoolCreatedFactory)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ClankerPoolCreatedFactoryIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerPoolCreatedFactoryIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerPoolCreatedFactory represents a PoolCreatedFactory event raised by the Clanker contract.
type ClankerPoolCreatedFactory struct {
	PairedToken           common.Address
	Clanker               common.Address
	PoolId                [32]byte
	TickIfToken0IsClanker *big.Int
	TickSpacing           *big.Int
	Locker                common.Address
	MevModule             common.Address
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterPoolCreatedFactory is a free log retrieval operation binding the contract event 0xc2cc0ae95873d7fa4e50998bc075ab443517931580a05c3f76d9fc983c666a5c.
//
// Solidity: event PoolCreatedFactory(address indexed pairedToken, address indexed clanker, bytes32 poolId, int24 tickIfToken0IsClanker, int24 tickSpacing, address locker, address mevModule)
func (_Clanker *ClankerFilterer) FilterPoolCreatedFactory(opts *bind.FilterOpts, pairedToken []common.Address, clanker []common.Address) (*ClankerPoolCreatedFactoryIterator, error) {

	var pairedTokenRule []interface{}
	for _, pairedTokenItem := range pairedToken {
		pairedTokenRule = append(pairedTokenRule, pairedTokenItem)
	}
	var clankerRule []interface{}
	for _, clankerItem := range clanker {
		clankerRule = append(clankerRule, clankerItem)
	}

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "PoolCreatedFactory", pairedTokenRule, clankerRule)
	if err != nil {
		return nil, err
	}
	return &ClankerPoolCreatedFactoryIterator{contract: _Clanker.contract, event: "PoolCreatedFactory", logs: logs, sub: sub}, nil
}

// WatchPoolCreatedFactory is a free log subscription operation binding the contract event 0xc2cc0ae95873d7fa4e50998bc075ab443517931580a05c3f76d9fc983c666a5c.
//
// Solidity: event PoolCreatedFactory(address indexed pairedToken, address indexed clanker, bytes32 poolId, int24 tickIfToken0IsClanker, int24 tickSpacing, address locker, address mevModule)
func (_Clanker *ClankerFilterer) WatchPoolCreatedFactory(opts *bind.WatchOpts, sink chan<- *ClankerPoolCreatedFactory, pairedToken []common.Address, clanker []common.Address) (event.Subscription, error) {

	var pairedTokenRule []interface{}
	for _, pairedTokenItem := range pairedToken {
		pairedTokenRule = append(pairedTokenRule, pairedTokenItem)
	}
	var clankerRule []interface{}
	for _, clankerItem := range clanker {
		clankerRule = append(clankerRule, clankerItem)
	}

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "PoolCreatedFactory", pairedTokenRule, clankerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerPoolCreatedFactory)
				if err := _Clanker.contract.UnpackLog(event, "PoolCreatedFactory", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePoolCreatedFactory is a log parse operation binding the contract event 0xc2cc0ae95873d7fa4e50998bc075ab443517931580a05c3f76d9fc983c666a5c.
//
// Solidity: event PoolCreatedFactory(address indexed pairedToken, address indexed clanker, bytes32 poolId, int24 tickIfToken0IsClanker, int24 tickSpacing, address locker, address mevModule)
func (_Clanker *ClankerFilterer) ParsePoolCreatedFactory(log types.Log) (*ClankerPoolCreatedFactory, error) {
	event := new(ClankerPoolCreatedFactory)
	if err := _Clanker.contract.UnpackLog(event, "PoolCreatedFactory", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClankerPoolCreatedOpenIterator is returned from FilterPoolCreatedOpen and is used to iterate over the raw logs and unpacked data for PoolCreatedOpen events raised by the Clanker contract.
type ClankerPoolCreatedOpenIterator struct {
	Event *ClankerPoolCreatedOpen // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ClankerPoolCreatedOpenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerPoolCreatedOpen)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ClankerPoolCreatedOpen)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ClankerPoolCreatedOpenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerPoolCreatedOpenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerPoolCreatedOpen represents a PoolCreatedOpen event raised by the Clanker contract.
type ClankerPoolCreatedOpen struct {
	PairedToken           common.Address
	Clanker               common.Address
	PoolId                [32]byte
	TickIfToken0IsClanker *big.Int
	TickSpacing           *big.Int
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterPoolCreatedOpen is a free log retrieval operation binding the contract event 0x09cc09ed021cf0d9112cf6d1bdc8689bfedd367df1ac5a550dfd6a07955b8d19.
//
// Solidity: event PoolCreatedOpen(address indexed pairedToken, address indexed clanker, bytes32 poolId, int24 tickIfToken0IsClanker, int24 tickSpacing)
func (_Clanker *ClankerFilterer) FilterPoolCreatedOpen(opts *bind.FilterOpts, pairedToken []common.Address, clanker []common.Address) (*ClankerPoolCreatedOpenIterator, error) {

	var pairedTokenRule []interface{}
	for _, pairedTokenItem := range pairedToken {
		pairedTokenRule = append(pairedTokenRule, pairedTokenItem)
	}
	var clankerRule []interface{}
	for _, clankerItem := range clanker {
		clankerRule = append(clankerRule, clankerItem)
	}

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "PoolCreatedOpen", pairedTokenRule, clankerRule)
	if err != nil {
		return nil, err
	}
	return &ClankerPoolCreatedOpenIterator{contract: _Clanker.contract, event: "PoolCreatedOpen", logs: logs, sub: sub}, nil
}

// WatchPoolCreatedOpen is a free log subscription operation binding the contract event 0x09cc09ed021cf0d9112cf6d1bdc8689bfedd367df1ac5a550dfd6a07955b8d19.
//
// Solidity: event PoolCreatedOpen(address indexed pairedToken, address indexed clanker, bytes32 poolId, int24 tickIfToken0IsClanker, int24 tickSpacing)
func (_Clanker *ClankerFilterer) WatchPoolCreatedOpen(opts *bind.WatchOpts, sink chan<- *ClankerPoolCreatedOpen, pairedToken []common.Address, clanker []common.Address) (event.Subscription, error) {

	var pairedTokenRule []interface{}
	for _, pairedTokenItem := range pairedToken {
		pairedTokenRule = append(pairedTokenRule, pairedTokenItem)
	}
	var clankerRule []interface{}
	for _, clankerItem := range clanker {
		clankerRule = append(clankerRule, clankerItem)
	}

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "PoolCreatedOpen", pairedTokenRule, clankerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerPoolCreatedOpen)
				if err := _Clanker.contract.UnpackLog(event, "PoolCreatedOpen", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePoolCreatedOpen is a log parse operation binding the contract event 0x09cc09ed021cf0d9112cf6d1bdc8689bfedd367df1ac5a550dfd6a07955b8d19.
//
// Solidity: event PoolCreatedOpen(address indexed pairedToken, address indexed clanker, bytes32 poolId, int24 tickIfToken0IsClanker, int24 tickSpacing)
func (_Clanker *ClankerFilterer) ParsePoolCreatedOpen(log types.Log) (*ClankerPoolCreatedOpen, error) {
	event := new(ClankerPoolCreatedOpen)
	if err := _Clanker.contract.UnpackLog(event, "PoolCreatedOpen", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClankerPoolInitializedIterator is returned from FilterPoolInitialized and is used to iterate over the raw logs and unpacked data for PoolInitialized events raised by the Clanker contract.
type ClankerPoolInitializedIterator struct {
	Event *ClankerPoolInitialized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ClankerPoolInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClankerPoolInitialized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ClankerPoolInitialized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ClankerPoolInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClankerPoolInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClankerPoolInitialized represents a PoolInitialized event raised by the Clanker contract.
type ClankerPoolInitialized struct {
	PoolId                    [32]byte
	BaseFee                   *big.Int
	MaxLpFee                  *big.Int
	ReferenceTickFilterPeriod *big.Int
	ResetPeriod               *big.Int
	ResetTickFilter           *big.Int
	FeeControlNumerator       *big.Int
	DecayFilterBps            *big.Int
	Raw                       types.Log // Blockchain specific contextual infos
}

// FilterPoolInitialized is a free log retrieval operation binding the contract event 0x432632506f74db6583b1c612fbc7f00dc5ed2b2cdb9aeb8080ce8dca32dca49d.
//
// Solidity: event PoolInitialized(bytes32 poolId, uint24 baseFee, uint24 maxLpFee, uint256 referenceTickFilterPeriod, uint256 resetPeriod, int24 resetTickFilter, uint256 feeControlNumerator, uint24 decayFilterBps)
func (_Clanker *ClankerFilterer) FilterPoolInitialized(opts *bind.FilterOpts) (*ClankerPoolInitializedIterator, error) {

	logs, sub, err := _Clanker.contract.FilterLogs(opts, "PoolInitialized")
	if err != nil {
		return nil, err
	}
	return &ClankerPoolInitializedIterator{contract: _Clanker.contract, event: "PoolInitialized", logs: logs, sub: sub}, nil
}

// WatchPoolInitialized is a free log subscription operation binding the contract event 0x432632506f74db6583b1c612fbc7f00dc5ed2b2cdb9aeb8080ce8dca32dca49d.
//
// Solidity: event PoolInitialized(bytes32 poolId, uint24 baseFee, uint24 maxLpFee, uint256 referenceTickFilterPeriod, uint256 resetPeriod, int24 resetTickFilter, uint256 feeControlNumerator, uint24 decayFilterBps)
func (_Clanker *ClankerFilterer) WatchPoolInitialized(opts *bind.WatchOpts, sink chan<- *ClankerPoolInitialized) (event.Subscription, error) {

	logs, sub, err := _Clanker.contract.WatchLogs(opts, "PoolInitialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClankerPoolInitialized)
				if err := _Clanker.contract.UnpackLog(event, "PoolInitialized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePoolInitialized is a log parse operation binding the contract event 0x432632506f74db6583b1c612fbc7f00dc5ed2b2cdb9aeb8080ce8dca32dca49d.
//
// Solidity: event PoolInitialized(bytes32 poolId, uint24 baseFee, uint24 maxLpFee, uint256 referenceTickFilterPeriod, uint256 resetPeriod, int24 resetTickFilter, uint256 feeControlNumerator, uint24 decayFilterBps)
func (_Clanker *ClankerFilterer) ParsePoolInitialized(log types.Log) (*ClankerPoolInitialized, error) {
	event := new(ClankerPoolInitialized)
	if err := _Clanker.contract.UnpackLog(event, "PoolInitialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
