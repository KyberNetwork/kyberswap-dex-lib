// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package abi

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

// BinPoolMintArrays is an auto generated low-level Go binding around an user-defined struct.
type BinPoolMintArrays struct {
	Ids             []*big.Int
	Amounts         [][32]byte
	LiquidityMinted []*big.Int
}

// BinPositionInfo is an auto generated low-level Go binding around an user-defined struct.
type BinPositionInfo struct {
	Share *big.Int
}

// IBinPoolManagerBurnParams is an auto generated low-level Go binding around an user-defined struct.
type IBinPoolManagerBurnParams struct {
	Ids           []*big.Int
	AmountsToBurn []*big.Int
	Salt          [32]byte
}

// IBinPoolManagerMintParams is an auto generated low-level Go binding around an user-defined struct.
type IBinPoolManagerMintParams struct {
	LiquidityConfigs [][32]byte
	AmountIn         [32]byte
	Salt             [32]byte
}

// PoolKey is an auto generated low-level Go binding around an user-defined struct.
type PoolKey struct {
	Currency0   common.Address
	Currency1   common.Address
	Hooks       common.Address
	PoolManager common.Address
	Fee         *big.Int
	Parameters  [32]byte
}

// PancakeInfinityPoolManagerMetaData contains all meta data concerning the PancakeInfinityPoolManager contract.
var PancakeInfinityPoolManagerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIVault\",\"name\":\"vault\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AmountSpecifiedIsZero\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint24\",\"name\":\"id\",\"type\":\"uint24\"}],\"name\":\"BinHelper__CompositionFactorFlawed\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint24\",\"name\":\"id\",\"type\":\"uint24\"}],\"name\":\"BinPool__BurnZeroAmount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BinPool__EmptyLiquidityConfigs\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BinPool__InsufficientAmountUnSpecified\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BinPool__InvalidBurnInput\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BinPool__MaxLiquidityPerBinExceeded\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BinPool__NoLiquidityToReceiveFees\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint24\",\"name\":\"id\",\"type\":\"uint24\"}],\"name\":\"BinPool__ZeroAmountsOut\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint24\",\"name\":\"id\",\"type\":\"uint24\"}],\"name\":\"BinPool__ZeroShares\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"binStep\",\"type\":\"uint16\"}],\"name\":\"BinStepTooLarge\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"binStep\",\"type\":\"uint16\"}],\"name\":\"BinStepTooSmall\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"currency1\",\"type\":\"address\"}],\"name\":\"CurrenciesInitializedOutOfOrder\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EnforcedPause\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"HookConfigValidationError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"HookDeltaExceedsSwapAmount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"HookPermissionsValidationError\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"shares\",\"type\":\"uint256\"}],\"name\":\"InsufficientBinShareForDonate\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidCaller\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidHookResponse\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"}],\"name\":\"LPFeeTooLarge\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LiquidityConfigurations__InvalidConfig\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"maxBinStep\",\"type\":\"uint16\"}],\"name\":\"MaxBinStepTooSmall\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PackedUint128Math__AddOverflow\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PackedUint128Math__SubUnderflow\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PoolAlreadyInitialized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PoolManagerMismatch\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PoolNotInitialized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ProtocolFeeCannotBeFetched\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"}],\"name\":\"ProtocolFeeTooLarge\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"},{\"internalType\":\"int256\",\"name\":\"y\",\"type\":\"int256\"}],\"name\":\"Uint128x128Math__PowUnderflow\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Uint256x256Math__MulDivOverflow\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Uint256x256Math__MulShiftOverflow\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"UnauthorizedDynamicLPFeeUpdate\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"UnusedBitsNonZero\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32[]\",\"name\":\"amounts\",\"type\":\"bytes32[]\"}],\"name\":\"Burn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"int128\",\"name\":\"amount0\",\"type\":\"int128\"},{\"indexed\":false,\"internalType\":\"int128\",\"name\":\"amount1\",\"type\":\"int128\"},{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"binId\",\"type\":\"uint24\"}],\"name\":\"Donate\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"dynamicLPFee\",\"type\":\"uint24\"}],\"name\":\"DynamicLPFeeUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"parameters\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"activeId\",\"type\":\"uint24\"}],\"name\":\"Initialize\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32[]\",\"name\":\"amounts\",\"type\":\"bytes32[]\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"compositionFeeAmount\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"feeAmountToProtocol\",\"type\":\"bytes32\"}],\"name\":\"Mint\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"protocolFeeController\",\"type\":\"address\"}],\"name\":\"ProtocolFeeControllerUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"protocolFee\",\"type\":\"uint24\"}],\"name\":\"ProtocolFeeUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"maxBinStep\",\"type\":\"uint16\"}],\"name\":\"SetMaxBinStep\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"minLiquidity\",\"type\":\"uint256\"}],\"name\":\"SetMinBinSharesForDonate\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"int128\",\"name\":\"amount0\",\"type\":\"int128\"},{\"indexed\":false,\"internalType\":\"int128\",\"name\":\"amount1\",\"type\":\"int128\"},{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"activeId\",\"type\":\"uint24\"},{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"protocolFee\",\"type\":\"uint16\"}],\"name\":\"Swap\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MIN_BIN_STEP\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"},{\"internalType\":\"contractIPoolManager\",\"name\":\"poolManager\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"parameters\",\"type\":\"bytes32\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"amountsToBurn\",\"type\":\"uint256[]\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"}],\"internalType\":\"structIBinPoolManager.BurnParams\",\"name\":\"params\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"burn\",\"outputs\":[{\"internalType\":\"BalanceDelta\",\"name\":\"delta\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"collectProtocolFees\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountCollected\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"},{\"internalType\":\"contractIPoolManager\",\"name\":\"poolManager\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"parameters\",\"type\":\"bytes32\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint128\",\"name\":\"amount0\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1\",\"type\":\"uint128\"},{\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"donate\",\"outputs\":[{\"internalType\":\"BalanceDelta\",\"name\":\"delta\",\"type\":\"int256\"},{\"internalType\":\"uint24\",\"name\":\"binId\",\"type\":\"uint24\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"slot\",\"type\":\"bytes32\"}],\"name\":\"extsload\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"slots\",\"type\":\"bytes32[]\"}],\"name\":\"extsload\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"uint24\",\"name\":\"binId\",\"type\":\"uint24\"}],\"name\":\"getBin\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"binReserveX\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"binReserveY\",\"type\":\"uint128\"},{\"internalType\":\"uint256\",\"name\":\"binLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalShares\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"bool\",\"name\":\"swapForY\",\"type\":\"bool\"},{\"internalType\":\"uint24\",\"name\":\"binId\",\"type\":\"uint24\"}],\"name\":\"getNextNonEmptyBin\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"nextId\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"binId\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"}],\"name\":\"getPosition\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"share\",\"type\":\"uint256\"}],\"internalType\":\"structBinPosition.Info\",\"name\":\"position\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"getSlot0\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"activeId\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"protocolFee\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"lpFee\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"},{\"internalType\":\"contractIPoolManager\",\"name\":\"poolManager\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"parameters\",\"type\":\"bytes32\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint24\",\"name\":\"activeId\",\"type\":\"uint24\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxBinStep\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minBinShareForDonate\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"},{\"internalType\":\"contractIPoolManager\",\"name\":\"poolManager\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"parameters\",\"type\":\"bytes32\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32[]\",\"name\":\"liquidityConfigs\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32\",\"name\":\"amountIn\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"}],\"internalType\":\"structIBinPoolManager.MintParams\",\"name\":\"params\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"mint\",\"outputs\":[{\"internalType\":\"BalanceDelta\",\"name\":\"delta\",\"type\":\"int256\"},{\"components\":[{\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"internalType\":\"bytes32[]\",\"name\":\"amounts\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256[]\",\"name\":\"liquidityMinted\",\"type\":\"uint256[]\"}],\"internalType\":\"structBinPool.MintArrays\",\"name\":\"mintArray\",\"type\":\"tuple\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"res\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"poolIdToPoolKey\",\"outputs\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"},{\"internalType\":\"contractIPoolManager\",\"name\":\"poolManager\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"parameters\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"pools\",\"outputs\":[{\"internalType\":\"BinSlot0\",\"name\":\"slot0\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"level0\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"protocolFeeController\",\"outputs\":[{\"internalType\":\"contractIProtocolFeeController\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Currency\",\"name\":\"currency\",\"type\":\"address\"}],\"name\":\"protocolFeesAccrued\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"newMaxBinStep\",\"type\":\"uint16\"}],\"name\":\"setMaxBinStep\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"minBinShare\",\"type\":\"uint256\"}],\"name\":\"setMinBinSharesForDonate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"},{\"internalType\":\"contractIPoolManager\",\"name\":\"poolManager\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"parameters\",\"type\":\"bytes32\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint24\",\"name\":\"newProtocolFee\",\"type\":\"uint24\"}],\"name\":\"setProtocolFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIProtocolFeeController\",\"name\":\"controller\",\"type\":\"address\"}],\"name\":\"setProtocolFeeController\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"},{\"internalType\":\"contractIPoolManager\",\"name\":\"poolManager\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"parameters\",\"type\":\"bytes32\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"bool\",\"name\":\"swapForY\",\"type\":\"bool\"},{\"internalType\":\"int128\",\"name\":\"amountSpecified\",\"type\":\"int128\"},{\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"swap\",\"outputs\":[{\"internalType\":\"BalanceDelta\",\"name\":\"delta\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"},{\"internalType\":\"contractIPoolManager\",\"name\":\"poolManager\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"parameters\",\"type\":\"bytes32\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint24\",\"name\":\"newDynamicLPFee\",\"type\":\"uint24\"}],\"name\":\"updateDynamicLPFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"vault\",\"outputs\":[{\"internalType\":\"contractIVault\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// PancakeInfinityPoolManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use PancakeInfinityPoolManagerMetaData.ABI instead.
var PancakeInfinityPoolManagerABI = PancakeInfinityPoolManagerMetaData.ABI

// PancakeInfinityPoolManager is an auto generated Go binding around an Ethereum contract.
type PancakeInfinityPoolManager struct {
	PancakeInfinityPoolManagerCaller     // Read-only binding to the contract
	PancakeInfinityPoolManagerTransactor // Write-only binding to the contract
	PancakeInfinityPoolManagerFilterer   // Log filterer for contract events
}

// PancakeInfinityPoolManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type PancakeInfinityPoolManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PancakeInfinityPoolManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PancakeInfinityPoolManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PancakeInfinityPoolManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PancakeInfinityPoolManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PancakeInfinityPoolManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PancakeInfinityPoolManagerSession struct {
	Contract     *PancakeInfinityPoolManager // Generic contract binding to set the session for
	CallOpts     bind.CallOpts               // Call options to use throughout this session
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// PancakeInfinityPoolManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PancakeInfinityPoolManagerCallerSession struct {
	Contract *PancakeInfinityPoolManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                     // Call options to use throughout this session
}

// PancakeInfinityPoolManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PancakeInfinityPoolManagerTransactorSession struct {
	Contract     *PancakeInfinityPoolManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                     // Transaction auth options to use throughout this session
}

// PancakeInfinityPoolManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type PancakeInfinityPoolManagerRaw struct {
	Contract *PancakeInfinityPoolManager // Generic contract binding to access the raw methods on
}

// PancakeInfinityPoolManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PancakeInfinityPoolManagerCallerRaw struct {
	Contract *PancakeInfinityPoolManagerCaller // Generic read-only contract binding to access the raw methods on
}

// PancakeInfinityPoolManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PancakeInfinityPoolManagerTransactorRaw struct {
	Contract *PancakeInfinityPoolManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPancakeInfinityPoolManager creates a new instance of PancakeInfinityPoolManager, bound to a specific deployed contract.
func NewPancakeInfinityPoolManager(address common.Address, backend bind.ContractBackend) (*PancakeInfinityPoolManager, error) {
	contract, err := bindPancakeInfinityPoolManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManager{PancakeInfinityPoolManagerCaller: PancakeInfinityPoolManagerCaller{contract: contract}, PancakeInfinityPoolManagerTransactor: PancakeInfinityPoolManagerTransactor{contract: contract}, PancakeInfinityPoolManagerFilterer: PancakeInfinityPoolManagerFilterer{contract: contract}}, nil
}

// NewPancakeInfinityPoolManagerCaller creates a new read-only instance of PancakeInfinityPoolManager, bound to a specific deployed contract.
func NewPancakeInfinityPoolManagerCaller(address common.Address, caller bind.ContractCaller) (*PancakeInfinityPoolManagerCaller, error) {
	contract, err := bindPancakeInfinityPoolManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManagerCaller{contract: contract}, nil
}

// NewPancakeInfinityPoolManagerTransactor creates a new write-only instance of PancakeInfinityPoolManager, bound to a specific deployed contract.
func NewPancakeInfinityPoolManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*PancakeInfinityPoolManagerTransactor, error) {
	contract, err := bindPancakeInfinityPoolManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManagerTransactor{contract: contract}, nil
}

// NewPancakeInfinityPoolManagerFilterer creates a new log filterer instance of PancakeInfinityPoolManager, bound to a specific deployed contract.
func NewPancakeInfinityPoolManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*PancakeInfinityPoolManagerFilterer, error) {
	contract, err := bindPancakeInfinityPoolManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManagerFilterer{contract: contract}, nil
}

// bindPancakeInfinityPoolManager binds a generic wrapper to an already deployed contract.
func bindPancakeInfinityPoolManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := PancakeInfinityPoolManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerRaw) Call(opts *bind.CallOpts, result *[]any, method string, params ...any) error {
	return _PancakeInfinityPoolManager.Contract.PancakeInfinityPoolManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.PancakeInfinityPoolManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...any) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.PancakeInfinityPoolManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerRaw) Call(opts *bind.CallOpts, result *[]any, method string, params ...any) error {
	return _PancakeInfinityPoolManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...any) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.contract.Transact(opts, method, params...)
}

// MINBINSTEP is a free data retrieval call binding the contract method 0x7df880e3.
//
// Solidity: function MIN_BIN_STEP() view returns(uint16)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCaller) MINBINSTEP(opts *bind.CallOpts) (uint16, error) {
	var out []any
	err := _PancakeInfinityPoolManager.contract.Call(opts, &out, "MIN_BIN_STEP")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// MINBINSTEP is a free data retrieval call binding the contract method 0x7df880e3.
//
// Solidity: function MIN_BIN_STEP() view returns(uint16)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) MINBINSTEP() (uint16, error) {
	return _PancakeInfinityPoolManager.Contract.MINBINSTEP(&_PancakeInfinityPoolManager.CallOpts)
}

// MINBINSTEP is a free data retrieval call binding the contract method 0x7df880e3.
//
// Solidity: function MIN_BIN_STEP() view returns(uint16)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerSession) MINBINSTEP() (uint16, error) {
	return _PancakeInfinityPoolManager.Contract.MINBINSTEP(&_PancakeInfinityPoolManager.CallOpts)
}

// Extsload is a free data retrieval call binding the contract method 0x1e2eaeaf.
//
// Solidity: function extsload(bytes32 slot) view returns(bytes32)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCaller) Extsload(opts *bind.CallOpts, slot [32]byte) ([32]byte, error) {
	var out []any
	err := _PancakeInfinityPoolManager.contract.Call(opts, &out, "extsload", slot)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Extsload is a free data retrieval call binding the contract method 0x1e2eaeaf.
//
// Solidity: function extsload(bytes32 slot) view returns(bytes32)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) Extsload(slot [32]byte) ([32]byte, error) {
	return _PancakeInfinityPoolManager.Contract.Extsload(&_PancakeInfinityPoolManager.CallOpts, slot)
}

// Extsload is a free data retrieval call binding the contract method 0x1e2eaeaf.
//
// Solidity: function extsload(bytes32 slot) view returns(bytes32)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerSession) Extsload(slot [32]byte) ([32]byte, error) {
	return _PancakeInfinityPoolManager.Contract.Extsload(&_PancakeInfinityPoolManager.CallOpts, slot)
}

// Extsload0 is a free data retrieval call binding the contract method 0xdbd035ff.
//
// Solidity: function extsload(bytes32[] slots) view returns(bytes32[])
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCaller) Extsload0(opts *bind.CallOpts, slots [][32]byte) ([][32]byte, error) {
	var out []any
	err := _PancakeInfinityPoolManager.contract.Call(opts, &out, "extsload0", slots)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// Extsload0 is a free data retrieval call binding the contract method 0xdbd035ff.
//
// Solidity: function extsload(bytes32[] slots) view returns(bytes32[])
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) Extsload0(slots [][32]byte) ([][32]byte, error) {
	return _PancakeInfinityPoolManager.Contract.Extsload0(&_PancakeInfinityPoolManager.CallOpts, slots)
}

// Extsload0 is a free data retrieval call binding the contract method 0xdbd035ff.
//
// Solidity: function extsload(bytes32[] slots) view returns(bytes32[])
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerSession) Extsload0(slots [][32]byte) ([][32]byte, error) {
	return _PancakeInfinityPoolManager.Contract.Extsload0(&_PancakeInfinityPoolManager.CallOpts, slots)
}

// GetBin is a free data retrieval call binding the contract method 0x4580c3c5.
//
// Solidity: function getBin(bytes32 id, uint24 binId) view returns(uint128 binReserveX, uint128 binReserveY, uint256 binLiquidity, uint256 totalShares)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCaller) GetBin(opts *bind.CallOpts, id [32]byte, binId *big.Int) (struct {
	BinReserveX  *big.Int
	BinReserveY  *big.Int
	BinLiquidity *big.Int
	TotalShares  *big.Int
}, error) {
	var out []any
	err := _PancakeInfinityPoolManager.contract.Call(opts, &out, "getBin", id, binId)

	outstruct := new(struct {
		BinReserveX  *big.Int
		BinReserveY  *big.Int
		BinLiquidity *big.Int
		TotalShares  *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.BinReserveX = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.BinReserveY = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.BinLiquidity = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.TotalShares = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetBin is a free data retrieval call binding the contract method 0x4580c3c5.
//
// Solidity: function getBin(bytes32 id, uint24 binId) view returns(uint128 binReserveX, uint128 binReserveY, uint256 binLiquidity, uint256 totalShares)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) GetBin(id [32]byte, binId *big.Int) (struct {
	BinReserveX  *big.Int
	BinReserveY  *big.Int
	BinLiquidity *big.Int
	TotalShares  *big.Int
}, error) {
	return _PancakeInfinityPoolManager.Contract.GetBin(&_PancakeInfinityPoolManager.CallOpts, id, binId)
}

// GetBin is a free data retrieval call binding the contract method 0x4580c3c5.
//
// Solidity: function getBin(bytes32 id, uint24 binId) view returns(uint128 binReserveX, uint128 binReserveY, uint256 binLiquidity, uint256 totalShares)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerSession) GetBin(id [32]byte, binId *big.Int) (struct {
	BinReserveX  *big.Int
	BinReserveY  *big.Int
	BinLiquidity *big.Int
	TotalShares  *big.Int
}, error) {
	return _PancakeInfinityPoolManager.Contract.GetBin(&_PancakeInfinityPoolManager.CallOpts, id, binId)
}

// GetNextNonEmptyBin is a free data retrieval call binding the contract method 0xbf7b7286.
//
// Solidity: function getNextNonEmptyBin(bytes32 id, bool swapForY, uint24 binId) view returns(uint24 nextId)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCaller) GetNextNonEmptyBin(opts *bind.CallOpts, id [32]byte, swapForY bool, binId *big.Int) (*big.Int, error) {
	var out []any
	err := _PancakeInfinityPoolManager.contract.Call(opts, &out, "getNextNonEmptyBin", id, swapForY, binId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNextNonEmptyBin is a free data retrieval call binding the contract method 0xbf7b7286.
//
// Solidity: function getNextNonEmptyBin(bytes32 id, bool swapForY, uint24 binId) view returns(uint24 nextId)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) GetNextNonEmptyBin(id [32]byte, swapForY bool, binId *big.Int) (*big.Int, error) {
	return _PancakeInfinityPoolManager.Contract.GetNextNonEmptyBin(&_PancakeInfinityPoolManager.CallOpts, id, swapForY, binId)
}

// GetNextNonEmptyBin is a free data retrieval call binding the contract method 0xbf7b7286.
//
// Solidity: function getNextNonEmptyBin(bytes32 id, bool swapForY, uint24 binId) view returns(uint24 nextId)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerSession) GetNextNonEmptyBin(id [32]byte, swapForY bool, binId *big.Int) (*big.Int, error) {
	return _PancakeInfinityPoolManager.Contract.GetNextNonEmptyBin(&_PancakeInfinityPoolManager.CallOpts, id, swapForY, binId)
}

// GetPosition is a free data retrieval call binding the contract method 0xa4a9a549.
//
// Solidity: function getPosition(bytes32 id, address owner, uint24 binId, bytes32 salt) view returns((uint256) position)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCaller) GetPosition(opts *bind.CallOpts, id [32]byte, owner common.Address, binId *big.Int, salt [32]byte) (BinPositionInfo, error) {
	var out []any
	err := _PancakeInfinityPoolManager.contract.Call(opts, &out, "getPosition", id, owner, binId, salt)

	if err != nil {
		return *new(BinPositionInfo), err
	}

	out0 := *abi.ConvertType(out[0], new(BinPositionInfo)).(*BinPositionInfo)

	return out0, err

}

// GetPosition is a free data retrieval call binding the contract method 0xa4a9a549.
//
// Solidity: function getPosition(bytes32 id, address owner, uint24 binId, bytes32 salt) view returns((uint256) position)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) GetPosition(id [32]byte, owner common.Address, binId *big.Int, salt [32]byte) (BinPositionInfo, error) {
	return _PancakeInfinityPoolManager.Contract.GetPosition(&_PancakeInfinityPoolManager.CallOpts, id, owner, binId, salt)
}

// GetPosition is a free data retrieval call binding the contract method 0xa4a9a549.
//
// Solidity: function getPosition(bytes32 id, address owner, uint24 binId, bytes32 salt) view returns((uint256) position)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerSession) GetPosition(id [32]byte, owner common.Address, binId *big.Int, salt [32]byte) (BinPositionInfo, error) {
	return _PancakeInfinityPoolManager.Contract.GetPosition(&_PancakeInfinityPoolManager.CallOpts, id, owner, binId, salt)
}

// GetSlot0 is a free data retrieval call binding the contract method 0xc815641c.
//
// Solidity: function getSlot0(bytes32 id) view returns(uint24 activeId, uint24 protocolFee, uint24 lpFee)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCaller) GetSlot0(opts *bind.CallOpts, id [32]byte) (struct {
	ActiveId    *big.Int
	ProtocolFee *big.Int
	LpFee       *big.Int
}, error) {
	var out []any
	err := _PancakeInfinityPoolManager.contract.Call(opts, &out, "getSlot0", id)

	outstruct := new(struct {
		ActiveId    *big.Int
		ProtocolFee *big.Int
		LpFee       *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.ActiveId = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.ProtocolFee = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.LpFee = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetSlot0 is a free data retrieval call binding the contract method 0xc815641c.
//
// Solidity: function getSlot0(bytes32 id) view returns(uint24 activeId, uint24 protocolFee, uint24 lpFee)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) GetSlot0(id [32]byte) (struct {
	ActiveId    *big.Int
	ProtocolFee *big.Int
	LpFee       *big.Int
}, error) {
	return _PancakeInfinityPoolManager.Contract.GetSlot0(&_PancakeInfinityPoolManager.CallOpts, id)
}

// GetSlot0 is a free data retrieval call binding the contract method 0xc815641c.
//
// Solidity: function getSlot0(bytes32 id) view returns(uint24 activeId, uint24 protocolFee, uint24 lpFee)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerSession) GetSlot0(id [32]byte) (struct {
	ActiveId    *big.Int
	ProtocolFee *big.Int
	LpFee       *big.Int
}, error) {
	return _PancakeInfinityPoolManager.Contract.GetSlot0(&_PancakeInfinityPoolManager.CallOpts, id)
}

// MaxBinStep is a free data retrieval call binding the contract method 0x615188b7.
//
// Solidity: function maxBinStep() view returns(uint16)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCaller) MaxBinStep(opts *bind.CallOpts) (uint16, error) {
	var out []any
	err := _PancakeInfinityPoolManager.contract.Call(opts, &out, "maxBinStep")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// MaxBinStep is a free data retrieval call binding the contract method 0x615188b7.
//
// Solidity: function maxBinStep() view returns(uint16)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) MaxBinStep() (uint16, error) {
	return _PancakeInfinityPoolManager.Contract.MaxBinStep(&_PancakeInfinityPoolManager.CallOpts)
}

// MaxBinStep is a free data retrieval call binding the contract method 0x615188b7.
//
// Solidity: function maxBinStep() view returns(uint16)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerSession) MaxBinStep() (uint16, error) {
	return _PancakeInfinityPoolManager.Contract.MaxBinStep(&_PancakeInfinityPoolManager.CallOpts)
}

// MinBinShareForDonate is a free data retrieval call binding the contract method 0xb1def7f4.
//
// Solidity: function minBinShareForDonate() view returns(uint256)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCaller) MinBinShareForDonate(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _PancakeInfinityPoolManager.contract.Call(opts, &out, "minBinShareForDonate")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinBinShareForDonate is a free data retrieval call binding the contract method 0xb1def7f4.
//
// Solidity: function minBinShareForDonate() view returns(uint256)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) MinBinShareForDonate() (*big.Int, error) {
	return _PancakeInfinityPoolManager.Contract.MinBinShareForDonate(&_PancakeInfinityPoolManager.CallOpts)
}

// MinBinShareForDonate is a free data retrieval call binding the contract method 0xb1def7f4.
//
// Solidity: function minBinShareForDonate() view returns(uint256)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerSession) MinBinShareForDonate() (*big.Int, error) {
	return _PancakeInfinityPoolManager.Contract.MinBinShareForDonate(&_PancakeInfinityPoolManager.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _PancakeInfinityPoolManager.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) Owner() (common.Address, error) {
	return _PancakeInfinityPoolManager.Contract.Owner(&_PancakeInfinityPoolManager.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerSession) Owner() (common.Address, error) {
	return _PancakeInfinityPoolManager.Contract.Owner(&_PancakeInfinityPoolManager.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool res)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []any
	err := _PancakeInfinityPoolManager.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool res)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) Paused() (bool, error) {
	return _PancakeInfinityPoolManager.Contract.Paused(&_PancakeInfinityPoolManager.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool res)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerSession) Paused() (bool, error) {
	return _PancakeInfinityPoolManager.Contract.Paused(&_PancakeInfinityPoolManager.CallOpts)
}

// PoolIdToPoolKey is a free data retrieval call binding the contract method 0x0e2d484a.
//
// Solidity: function poolIdToPoolKey(bytes32 id) view returns(address currency0, address currency1, address hooks, address poolManager, uint24 fee, bytes32 parameters)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCaller) PoolIdToPoolKey(opts *bind.CallOpts, id [32]byte) (struct {
	Currency0   common.Address
	Currency1   common.Address
	Hooks       common.Address
	PoolManager common.Address
	Fee         *big.Int
	Parameters  [32]byte
}, error) {
	var out []any
	err := _PancakeInfinityPoolManager.contract.Call(opts, &out, "poolIdToPoolKey", id)

	outstruct := new(struct {
		Currency0   common.Address
		Currency1   common.Address
		Hooks       common.Address
		PoolManager common.Address
		Fee         *big.Int
		Parameters  [32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Currency0 = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Currency1 = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.Hooks = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	outstruct.PoolManager = *abi.ConvertType(out[3], new(common.Address)).(*common.Address)
	outstruct.Fee = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.Parameters = *abi.ConvertType(out[5], new([32]byte)).(*[32]byte)

	return *outstruct, err

}

// PoolIdToPoolKey is a free data retrieval call binding the contract method 0x0e2d484a.
//
// Solidity: function poolIdToPoolKey(bytes32 id) view returns(address currency0, address currency1, address hooks, address poolManager, uint24 fee, bytes32 parameters)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) PoolIdToPoolKey(id [32]byte) (struct {
	Currency0   common.Address
	Currency1   common.Address
	Hooks       common.Address
	PoolManager common.Address
	Fee         *big.Int
	Parameters  [32]byte
}, error) {
	return _PancakeInfinityPoolManager.Contract.PoolIdToPoolKey(&_PancakeInfinityPoolManager.CallOpts, id)
}

// PoolIdToPoolKey is a free data retrieval call binding the contract method 0x0e2d484a.
//
// Solidity: function poolIdToPoolKey(bytes32 id) view returns(address currency0, address currency1, address hooks, address poolManager, uint24 fee, bytes32 parameters)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerSession) PoolIdToPoolKey(id [32]byte) (struct {
	Currency0   common.Address
	Currency1   common.Address
	Hooks       common.Address
	PoolManager common.Address
	Fee         *big.Int
	Parameters  [32]byte
}, error) {
	return _PancakeInfinityPoolManager.Contract.PoolIdToPoolKey(&_PancakeInfinityPoolManager.CallOpts, id)
}

// Pools is a free data retrieval call binding the contract method 0xb5217bb4.
//
// Solidity: function pools(bytes32 id) view returns(bytes32 slot0, bytes32 level0)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCaller) Pools(opts *bind.CallOpts, id [32]byte) (struct {
	Slot0  [32]byte
	Level0 [32]byte
}, error) {
	var out []any
	err := _PancakeInfinityPoolManager.contract.Call(opts, &out, "pools", id)

	outstruct := new(struct {
		Slot0  [32]byte
		Level0 [32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Slot0 = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.Level0 = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)

	return *outstruct, err

}

// Pools is a free data retrieval call binding the contract method 0xb5217bb4.
//
// Solidity: function pools(bytes32 id) view returns(bytes32 slot0, bytes32 level0)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) Pools(id [32]byte) (struct {
	Slot0  [32]byte
	Level0 [32]byte
}, error) {
	return _PancakeInfinityPoolManager.Contract.Pools(&_PancakeInfinityPoolManager.CallOpts, id)
}

// Pools is a free data retrieval call binding the contract method 0xb5217bb4.
//
// Solidity: function pools(bytes32 id) view returns(bytes32 slot0, bytes32 level0)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerSession) Pools(id [32]byte) (struct {
	Slot0  [32]byte
	Level0 [32]byte
}, error) {
	return _PancakeInfinityPoolManager.Contract.Pools(&_PancakeInfinityPoolManager.CallOpts, id)
}

// ProtocolFeeController is a free data retrieval call binding the contract method 0xf02de3b2.
//
// Solidity: function protocolFeeController() view returns(address)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCaller) ProtocolFeeController(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _PancakeInfinityPoolManager.contract.Call(opts, &out, "protocolFeeController")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ProtocolFeeController is a free data retrieval call binding the contract method 0xf02de3b2.
//
// Solidity: function protocolFeeController() view returns(address)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) ProtocolFeeController() (common.Address, error) {
	return _PancakeInfinityPoolManager.Contract.ProtocolFeeController(&_PancakeInfinityPoolManager.CallOpts)
}

// ProtocolFeeController is a free data retrieval call binding the contract method 0xf02de3b2.
//
// Solidity: function protocolFeeController() view returns(address)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerSession) ProtocolFeeController() (common.Address, error) {
	return _PancakeInfinityPoolManager.Contract.ProtocolFeeController(&_PancakeInfinityPoolManager.CallOpts)
}

// ProtocolFeesAccrued is a free data retrieval call binding the contract method 0x97e8cd4e.
//
// Solidity: function protocolFeesAccrued(address currency) view returns(uint256 amount)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCaller) ProtocolFeesAccrued(opts *bind.CallOpts, currency common.Address) (*big.Int, error) {
	var out []any
	err := _PancakeInfinityPoolManager.contract.Call(opts, &out, "protocolFeesAccrued", currency)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ProtocolFeesAccrued is a free data retrieval call binding the contract method 0x97e8cd4e.
//
// Solidity: function protocolFeesAccrued(address currency) view returns(uint256 amount)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) ProtocolFeesAccrued(currency common.Address) (*big.Int, error) {
	return _PancakeInfinityPoolManager.Contract.ProtocolFeesAccrued(&_PancakeInfinityPoolManager.CallOpts, currency)
}

// ProtocolFeesAccrued is a free data retrieval call binding the contract method 0x97e8cd4e.
//
// Solidity: function protocolFeesAccrued(address currency) view returns(uint256 amount)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerSession) ProtocolFeesAccrued(currency common.Address) (*big.Int, error) {
	return _PancakeInfinityPoolManager.Contract.ProtocolFeesAccrued(&_PancakeInfinityPoolManager.CallOpts, currency)
}

// Vault is a free data retrieval call binding the contract method 0xfbfa77cf.
//
// Solidity: function vault() view returns(address)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCaller) Vault(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _PancakeInfinityPoolManager.contract.Call(opts, &out, "vault")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Vault is a free data retrieval call binding the contract method 0xfbfa77cf.
//
// Solidity: function vault() view returns(address)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) Vault() (common.Address, error) {
	return _PancakeInfinityPoolManager.Contract.Vault(&_PancakeInfinityPoolManager.CallOpts)
}

// Vault is a free data retrieval call binding the contract method 0xfbfa77cf.
//
// Solidity: function vault() view returns(address)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerCallerSession) Vault() (common.Address, error) {
	return _PancakeInfinityPoolManager.Contract.Vault(&_PancakeInfinityPoolManager.CallOpts)
}

// Burn is a paid mutator transaction binding the contract method 0xbc5348f2.
//
// Solidity: function burn((address,address,address,address,uint24,bytes32) key, (uint256[],uint256[],bytes32) params, bytes hookData) returns(int256 delta)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactor) Burn(opts *bind.TransactOpts, key PoolKey, params IBinPoolManagerBurnParams, hookData []byte) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.contract.Transact(opts, "burn", key, params, hookData)
}

// Burn is a paid mutator transaction binding the contract method 0xbc5348f2.
//
// Solidity: function burn((address,address,address,address,uint24,bytes32) key, (uint256[],uint256[],bytes32) params, bytes hookData) returns(int256 delta)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) Burn(key PoolKey, params IBinPoolManagerBurnParams, hookData []byte) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.Burn(&_PancakeInfinityPoolManager.TransactOpts, key, params, hookData)
}

// Burn is a paid mutator transaction binding the contract method 0xbc5348f2.
//
// Solidity: function burn((address,address,address,address,uint24,bytes32) key, (uint256[],uint256[],bytes32) params, bytes hookData) returns(int256 delta)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactorSession) Burn(key PoolKey, params IBinPoolManagerBurnParams, hookData []byte) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.Burn(&_PancakeInfinityPoolManager.TransactOpts, key, params, hookData)
}

// CollectProtocolFees is a paid mutator transaction binding the contract method 0x8161b874.
//
// Solidity: function collectProtocolFees(address recipient, address currency, uint256 amount) returns(uint256 amountCollected)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactor) CollectProtocolFees(opts *bind.TransactOpts, recipient common.Address, currency common.Address, amount *big.Int) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.contract.Transact(opts, "collectProtocolFees", recipient, currency, amount)
}

// CollectProtocolFees is a paid mutator transaction binding the contract method 0x8161b874.
//
// Solidity: function collectProtocolFees(address recipient, address currency, uint256 amount) returns(uint256 amountCollected)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) CollectProtocolFees(recipient common.Address, currency common.Address, amount *big.Int) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.CollectProtocolFees(&_PancakeInfinityPoolManager.TransactOpts, recipient, currency, amount)
}

// CollectProtocolFees is a paid mutator transaction binding the contract method 0x8161b874.
//
// Solidity: function collectProtocolFees(address recipient, address currency, uint256 amount) returns(uint256 amountCollected)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactorSession) CollectProtocolFees(recipient common.Address, currency common.Address, amount *big.Int) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.CollectProtocolFees(&_PancakeInfinityPoolManager.TransactOpts, recipient, currency, amount)
}

// Donate is a paid mutator transaction binding the contract method 0x57738174.
//
// Solidity: function donate((address,address,address,address,uint24,bytes32) key, uint128 amount0, uint128 amount1, bytes hookData) returns(int256 delta, uint24 binId)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactor) Donate(opts *bind.TransactOpts, key PoolKey, amount0 *big.Int, amount1 *big.Int, hookData []byte) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.contract.Transact(opts, "donate", key, amount0, amount1, hookData)
}

// Donate is a paid mutator transaction binding the contract method 0x57738174.
//
// Solidity: function donate((address,address,address,address,uint24,bytes32) key, uint128 amount0, uint128 amount1, bytes hookData) returns(int256 delta, uint24 binId)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) Donate(key PoolKey, amount0 *big.Int, amount1 *big.Int, hookData []byte) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.Donate(&_PancakeInfinityPoolManager.TransactOpts, key, amount0, amount1, hookData)
}

// Donate is a paid mutator transaction binding the contract method 0x57738174.
//
// Solidity: function donate((address,address,address,address,uint24,bytes32) key, uint128 amount0, uint128 amount1, bytes hookData) returns(int256 delta, uint24 binId)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactorSession) Donate(key PoolKey, amount0 *big.Int, amount1 *big.Int, hookData []byte) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.Donate(&_PancakeInfinityPoolManager.TransactOpts, key, amount0, amount1, hookData)
}

// Initialize is a paid mutator transaction binding the contract method 0xdc2ab18c.
//
// Solidity: function initialize((address,address,address,address,uint24,bytes32) key, uint24 activeId) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactor) Initialize(opts *bind.TransactOpts, key PoolKey, activeId *big.Int) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.contract.Transact(opts, "initialize", key, activeId)
}

// Initialize is a paid mutator transaction binding the contract method 0xdc2ab18c.
//
// Solidity: function initialize((address,address,address,address,uint24,bytes32) key, uint24 activeId) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) Initialize(key PoolKey, activeId *big.Int) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.Initialize(&_PancakeInfinityPoolManager.TransactOpts, key, activeId)
}

// Initialize is a paid mutator transaction binding the contract method 0xdc2ab18c.
//
// Solidity: function initialize((address,address,address,address,uint24,bytes32) key, uint24 activeId) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactorSession) Initialize(key PoolKey, activeId *big.Int) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.Initialize(&_PancakeInfinityPoolManager.TransactOpts, key, activeId)
}

// Mint is a paid mutator transaction binding the contract method 0xebc413e0.
//
// Solidity: function mint((address,address,address,address,uint24,bytes32) key, (bytes32[],bytes32,bytes32) params, bytes hookData) returns(int256 delta, (uint256[],bytes32[],uint256[]) mintArray)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactor) Mint(opts *bind.TransactOpts, key PoolKey, params IBinPoolManagerMintParams, hookData []byte) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.contract.Transact(opts, "mint", key, params, hookData)
}

// Mint is a paid mutator transaction binding the contract method 0xebc413e0.
//
// Solidity: function mint((address,address,address,address,uint24,bytes32) key, (bytes32[],bytes32,bytes32) params, bytes hookData) returns(int256 delta, (uint256[],bytes32[],uint256[]) mintArray)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) Mint(key PoolKey, params IBinPoolManagerMintParams, hookData []byte) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.Mint(&_PancakeInfinityPoolManager.TransactOpts, key, params, hookData)
}

// Mint is a paid mutator transaction binding the contract method 0xebc413e0.
//
// Solidity: function mint((address,address,address,address,uint24,bytes32) key, (bytes32[],bytes32,bytes32) params, bytes hookData) returns(int256 delta, (uint256[],bytes32[],uint256[]) mintArray)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactorSession) Mint(key PoolKey, params IBinPoolManagerMintParams, hookData []byte) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.Mint(&_PancakeInfinityPoolManager.TransactOpts, key, params, hookData)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) Pause() (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.Pause(&_PancakeInfinityPoolManager.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactorSession) Pause() (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.Pause(&_PancakeInfinityPoolManager.TransactOpts)
}

// SetMaxBinStep is a paid mutator transaction binding the contract method 0x5cf7d255.
//
// Solidity: function setMaxBinStep(uint16 newMaxBinStep) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactor) SetMaxBinStep(opts *bind.TransactOpts, newMaxBinStep uint16) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.contract.Transact(opts, "setMaxBinStep", newMaxBinStep)
}

// SetMaxBinStep is a paid mutator transaction binding the contract method 0x5cf7d255.
//
// Solidity: function setMaxBinStep(uint16 newMaxBinStep) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) SetMaxBinStep(newMaxBinStep uint16) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.SetMaxBinStep(&_PancakeInfinityPoolManager.TransactOpts, newMaxBinStep)
}

// SetMaxBinStep is a paid mutator transaction binding the contract method 0x5cf7d255.
//
// Solidity: function setMaxBinStep(uint16 newMaxBinStep) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactorSession) SetMaxBinStep(newMaxBinStep uint16) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.SetMaxBinStep(&_PancakeInfinityPoolManager.TransactOpts, newMaxBinStep)
}

// SetMinBinSharesForDonate is a paid mutator transaction binding the contract method 0xc3b226f7.
//
// Solidity: function setMinBinSharesForDonate(uint256 minBinShare) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactor) SetMinBinSharesForDonate(opts *bind.TransactOpts, minBinShare *big.Int) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.contract.Transact(opts, "setMinBinSharesForDonate", minBinShare)
}

// SetMinBinSharesForDonate is a paid mutator transaction binding the contract method 0xc3b226f7.
//
// Solidity: function setMinBinSharesForDonate(uint256 minBinShare) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) SetMinBinSharesForDonate(minBinShare *big.Int) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.SetMinBinSharesForDonate(&_PancakeInfinityPoolManager.TransactOpts, minBinShare)
}

// SetMinBinSharesForDonate is a paid mutator transaction binding the contract method 0xc3b226f7.
//
// Solidity: function setMinBinSharesForDonate(uint256 minBinShare) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactorSession) SetMinBinSharesForDonate(minBinShare *big.Int) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.SetMinBinSharesForDonate(&_PancakeInfinityPoolManager.TransactOpts, minBinShare)
}

// SetProtocolFee is a paid mutator transaction binding the contract method 0x81a250a1.
//
// Solidity: function setProtocolFee((address,address,address,address,uint24,bytes32) key, uint24 newProtocolFee) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactor) SetProtocolFee(opts *bind.TransactOpts, key PoolKey, newProtocolFee *big.Int) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.contract.Transact(opts, "setProtocolFee", key, newProtocolFee)
}

// SetProtocolFee is a paid mutator transaction binding the contract method 0x81a250a1.
//
// Solidity: function setProtocolFee((address,address,address,address,uint24,bytes32) key, uint24 newProtocolFee) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) SetProtocolFee(key PoolKey, newProtocolFee *big.Int) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.SetProtocolFee(&_PancakeInfinityPoolManager.TransactOpts, key, newProtocolFee)
}

// SetProtocolFee is a paid mutator transaction binding the contract method 0x81a250a1.
//
// Solidity: function setProtocolFee((address,address,address,address,uint24,bytes32) key, uint24 newProtocolFee) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactorSession) SetProtocolFee(key PoolKey, newProtocolFee *big.Int) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.SetProtocolFee(&_PancakeInfinityPoolManager.TransactOpts, key, newProtocolFee)
}

// SetProtocolFeeController is a paid mutator transaction binding the contract method 0x2d771389.
//
// Solidity: function setProtocolFeeController(address controller) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactor) SetProtocolFeeController(opts *bind.TransactOpts, controller common.Address) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.contract.Transact(opts, "setProtocolFeeController", controller)
}

// SetProtocolFeeController is a paid mutator transaction binding the contract method 0x2d771389.
//
// Solidity: function setProtocolFeeController(address controller) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) SetProtocolFeeController(controller common.Address) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.SetProtocolFeeController(&_PancakeInfinityPoolManager.TransactOpts, controller)
}

// SetProtocolFeeController is a paid mutator transaction binding the contract method 0x2d771389.
//
// Solidity: function setProtocolFeeController(address controller) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactorSession) SetProtocolFeeController(controller common.Address) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.SetProtocolFeeController(&_PancakeInfinityPoolManager.TransactOpts, controller)
}

// Swap is a paid mutator transaction binding the contract method 0x911a63b7.
//
// Solidity: function swap((address,address,address,address,uint24,bytes32) key, bool swapForY, int128 amountSpecified, bytes hookData) returns(int256 delta)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactor) Swap(opts *bind.TransactOpts, key PoolKey, swapForY bool, amountSpecified *big.Int, hookData []byte) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.contract.Transact(opts, "swap", key, swapForY, amountSpecified, hookData)
}

// Swap is a paid mutator transaction binding the contract method 0x911a63b7.
//
// Solidity: function swap((address,address,address,address,uint24,bytes32) key, bool swapForY, int128 amountSpecified, bytes hookData) returns(int256 delta)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) Swap(key PoolKey, swapForY bool, amountSpecified *big.Int, hookData []byte) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.Swap(&_PancakeInfinityPoolManager.TransactOpts, key, swapForY, amountSpecified, hookData)
}

// Swap is a paid mutator transaction binding the contract method 0x911a63b7.
//
// Solidity: function swap((address,address,address,address,uint24,bytes32) key, bool swapForY, int128 amountSpecified, bytes hookData) returns(int256 delta)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactorSession) Swap(key PoolKey, swapForY bool, amountSpecified *big.Int, hookData []byte) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.Swap(&_PancakeInfinityPoolManager.TransactOpts, key, swapForY, amountSpecified, hookData)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.TransferOwnership(&_PancakeInfinityPoolManager.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.TransferOwnership(&_PancakeInfinityPoolManager.TransactOpts, newOwner)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) Unpause() (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.Unpause(&_PancakeInfinityPoolManager.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactorSession) Unpause() (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.Unpause(&_PancakeInfinityPoolManager.TransactOpts)
}

// UpdateDynamicLPFee is a paid mutator transaction binding the contract method 0xad4cc2d3.
//
// Solidity: function updateDynamicLPFee((address,address,address,address,uint24,bytes32) key, uint24 newDynamicLPFee) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactor) UpdateDynamicLPFee(opts *bind.TransactOpts, key PoolKey, newDynamicLPFee *big.Int) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.contract.Transact(opts, "updateDynamicLPFee", key, newDynamicLPFee)
}

// UpdateDynamicLPFee is a paid mutator transaction binding the contract method 0xad4cc2d3.
//
// Solidity: function updateDynamicLPFee((address,address,address,address,uint24,bytes32) key, uint24 newDynamicLPFee) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerSession) UpdateDynamicLPFee(key PoolKey, newDynamicLPFee *big.Int) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.UpdateDynamicLPFee(&_PancakeInfinityPoolManager.TransactOpts, key, newDynamicLPFee)
}

// UpdateDynamicLPFee is a paid mutator transaction binding the contract method 0xad4cc2d3.
//
// Solidity: function updateDynamicLPFee((address,address,address,address,uint24,bytes32) key, uint24 newDynamicLPFee) returns()
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerTransactorSession) UpdateDynamicLPFee(key PoolKey, newDynamicLPFee *big.Int) (*types.Transaction, error) {
	return _PancakeInfinityPoolManager.Contract.UpdateDynamicLPFee(&_PancakeInfinityPoolManager.TransactOpts, key, newDynamicLPFee)
}

// PancakeInfinityPoolManagerBurnIterator is returned from FilterBurn and is used to iterate over the raw logs and unpacked data for Burn events raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerBurnIterator struct {
	Event *PancakeInfinityPoolManagerBurn // Event containing the contract specifics and raw log

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
func (it *PancakeInfinityPoolManagerBurnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PancakeInfinityPoolManagerBurn)
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
		it.Event = new(PancakeInfinityPoolManagerBurn)
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
func (it *PancakeInfinityPoolManagerBurnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PancakeInfinityPoolManagerBurnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PancakeInfinityPoolManagerBurn represents a Burn event raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerBurn struct {
	Id      [32]byte
	Sender  common.Address
	Ids     []*big.Int
	Salt    [32]byte
	Amounts [][32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterBurn is a free log retrieval operation binding the contract event 0x16d40aa4e497175b58e47cbf101544758de8c01d92a760c738943044c758df8a.
//
// Solidity: event Burn(bytes32 indexed id, address indexed sender, uint256[] ids, bytes32 salt, bytes32[] amounts)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) FilterBurn(opts *bind.FilterOpts, id [][32]byte, sender []common.Address) (*PancakeInfinityPoolManagerBurnIterator, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.FilterLogs(opts, "Burn", idRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManagerBurnIterator{contract: _PancakeInfinityPoolManager.contract, event: "Burn", logs: logs, sub: sub}, nil
}

// WatchBurn is a free log subscription operation binding the contract event 0x16d40aa4e497175b58e47cbf101544758de8c01d92a760c738943044c758df8a.
//
// Solidity: event Burn(bytes32 indexed id, address indexed sender, uint256[] ids, bytes32 salt, bytes32[] amounts)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) WatchBurn(opts *bind.WatchOpts, sink chan<- *PancakeInfinityPoolManagerBurn, id [][32]byte, sender []common.Address) (event.Subscription, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.WatchLogs(opts, "Burn", idRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PancakeInfinityPoolManagerBurn)
				if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "Burn", log); err != nil {
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

// ParseBurn is a log parse operation binding the contract event 0x16d40aa4e497175b58e47cbf101544758de8c01d92a760c738943044c758df8a.
//
// Solidity: event Burn(bytes32 indexed id, address indexed sender, uint256[] ids, bytes32 salt, bytes32[] amounts)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) ParseBurn(log types.Log) (*PancakeInfinityPoolManagerBurn, error) {
	event := new(PancakeInfinityPoolManagerBurn)
	if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "Burn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PancakeInfinityPoolManagerDonateIterator is returned from FilterDonate and is used to iterate over the raw logs and unpacked data for Donate events raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerDonateIterator struct {
	Event *PancakeInfinityPoolManagerDonate // Event containing the contract specifics and raw log

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
func (it *PancakeInfinityPoolManagerDonateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PancakeInfinityPoolManagerDonate)
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
		it.Event = new(PancakeInfinityPoolManagerDonate)
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
func (it *PancakeInfinityPoolManagerDonateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PancakeInfinityPoolManagerDonateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PancakeInfinityPoolManagerDonate represents a Donate event raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerDonate struct {
	Id      [32]byte
	Sender  common.Address
	Amount0 *big.Int
	Amount1 *big.Int
	BinId   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterDonate is a free log retrieval operation binding the contract event 0xfc18146d5586318640b3febea90b094b834f09982812258534a1c07bdda12954.
//
// Solidity: event Donate(bytes32 indexed id, address indexed sender, int128 amount0, int128 amount1, uint24 binId)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) FilterDonate(opts *bind.FilterOpts, id [][32]byte, sender []common.Address) (*PancakeInfinityPoolManagerDonateIterator, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.FilterLogs(opts, "Donate", idRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManagerDonateIterator{contract: _PancakeInfinityPoolManager.contract, event: "Donate", logs: logs, sub: sub}, nil
}

// WatchDonate is a free log subscription operation binding the contract event 0xfc18146d5586318640b3febea90b094b834f09982812258534a1c07bdda12954.
//
// Solidity: event Donate(bytes32 indexed id, address indexed sender, int128 amount0, int128 amount1, uint24 binId)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) WatchDonate(opts *bind.WatchOpts, sink chan<- *PancakeInfinityPoolManagerDonate, id [][32]byte, sender []common.Address) (event.Subscription, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.WatchLogs(opts, "Donate", idRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PancakeInfinityPoolManagerDonate)
				if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "Donate", log); err != nil {
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

// ParseDonate is a log parse operation binding the contract event 0xfc18146d5586318640b3febea90b094b834f09982812258534a1c07bdda12954.
//
// Solidity: event Donate(bytes32 indexed id, address indexed sender, int128 amount0, int128 amount1, uint24 binId)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) ParseDonate(log types.Log) (*PancakeInfinityPoolManagerDonate, error) {
	event := new(PancakeInfinityPoolManagerDonate)
	if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "Donate", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PancakeInfinityPoolManagerDynamicLPFeeUpdatedIterator is returned from FilterDynamicLPFeeUpdated and is used to iterate over the raw logs and unpacked data for DynamicLPFeeUpdated events raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerDynamicLPFeeUpdatedIterator struct {
	Event *PancakeInfinityPoolManagerDynamicLPFeeUpdated // Event containing the contract specifics and raw log

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
func (it *PancakeInfinityPoolManagerDynamicLPFeeUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PancakeInfinityPoolManagerDynamicLPFeeUpdated)
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
		it.Event = new(PancakeInfinityPoolManagerDynamicLPFeeUpdated)
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
func (it *PancakeInfinityPoolManagerDynamicLPFeeUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PancakeInfinityPoolManagerDynamicLPFeeUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PancakeInfinityPoolManagerDynamicLPFeeUpdated represents a DynamicLPFeeUpdated event raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerDynamicLPFeeUpdated struct {
	Id           [32]byte
	DynamicLPFee *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterDynamicLPFeeUpdated is a free log retrieval operation binding the contract event 0x14b2b80e0d62303dc85494859f35a84579160aafbd650180ddf526b1ab547bd6.
//
// Solidity: event DynamicLPFeeUpdated(bytes32 indexed id, uint24 dynamicLPFee)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) FilterDynamicLPFeeUpdated(opts *bind.FilterOpts, id [][32]byte) (*PancakeInfinityPoolManagerDynamicLPFeeUpdatedIterator, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.FilterLogs(opts, "DynamicLPFeeUpdated", idRule)
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManagerDynamicLPFeeUpdatedIterator{contract: _PancakeInfinityPoolManager.contract, event: "DynamicLPFeeUpdated", logs: logs, sub: sub}, nil
}

// WatchDynamicLPFeeUpdated is a free log subscription operation binding the contract event 0x14b2b80e0d62303dc85494859f35a84579160aafbd650180ddf526b1ab547bd6.
//
// Solidity: event DynamicLPFeeUpdated(bytes32 indexed id, uint24 dynamicLPFee)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) WatchDynamicLPFeeUpdated(opts *bind.WatchOpts, sink chan<- *PancakeInfinityPoolManagerDynamicLPFeeUpdated, id [][32]byte) (event.Subscription, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.WatchLogs(opts, "DynamicLPFeeUpdated", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PancakeInfinityPoolManagerDynamicLPFeeUpdated)
				if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "DynamicLPFeeUpdated", log); err != nil {
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

// ParseDynamicLPFeeUpdated is a log parse operation binding the contract event 0x14b2b80e0d62303dc85494859f35a84579160aafbd650180ddf526b1ab547bd6.
//
// Solidity: event DynamicLPFeeUpdated(bytes32 indexed id, uint24 dynamicLPFee)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) ParseDynamicLPFeeUpdated(log types.Log) (*PancakeInfinityPoolManagerDynamicLPFeeUpdated, error) {
	event := new(PancakeInfinityPoolManagerDynamicLPFeeUpdated)
	if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "DynamicLPFeeUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PancakeInfinityPoolManagerInitializeIterator is returned from FilterInitialize and is used to iterate over the raw logs and unpacked data for Initialize events raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerInitializeIterator struct {
	Event *PancakeInfinityPoolManagerInitialize // Event containing the contract specifics and raw log

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
func (it *PancakeInfinityPoolManagerInitializeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PancakeInfinityPoolManagerInitialize)
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
		it.Event = new(PancakeInfinityPoolManagerInitialize)
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
func (it *PancakeInfinityPoolManagerInitializeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PancakeInfinityPoolManagerInitializeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PancakeInfinityPoolManagerInitialize represents a Initialize event raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerInitialize struct {
	Id         [32]byte
	Currency0  common.Address
	Currency1  common.Address
	Hooks      common.Address
	Fee        *big.Int
	Parameters [32]byte
	ActiveId   *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInitialize is a free log retrieval operation binding the contract event 0xddfde5903015c0eb1671976c6c8f760f1328bec57f15286b6bdab2f955cab9c9.
//
// Solidity: event Initialize(bytes32 indexed id, address indexed currency0, address indexed currency1, address hooks, uint24 fee, bytes32 parameters, uint24 activeId)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) FilterInitialize(opts *bind.FilterOpts, id [][32]byte, currency0 []common.Address, currency1 []common.Address) (*PancakeInfinityPoolManagerInitializeIterator, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var currency0Rule []any
	for _, currency0Item := range currency0 {
		currency0Rule = append(currency0Rule, currency0Item)
	}
	var currency1Rule []any
	for _, currency1Item := range currency1 {
		currency1Rule = append(currency1Rule, currency1Item)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.FilterLogs(opts, "Initialize", idRule, currency0Rule, currency1Rule)
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManagerInitializeIterator{contract: _PancakeInfinityPoolManager.contract, event: "Initialize", logs: logs, sub: sub}, nil
}

// WatchInitialize is a free log subscription operation binding the contract event 0xddfde5903015c0eb1671976c6c8f760f1328bec57f15286b6bdab2f955cab9c9.
//
// Solidity: event Initialize(bytes32 indexed id, address indexed currency0, address indexed currency1, address hooks, uint24 fee, bytes32 parameters, uint24 activeId)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) WatchInitialize(opts *bind.WatchOpts, sink chan<- *PancakeInfinityPoolManagerInitialize, id [][32]byte, currency0 []common.Address, currency1 []common.Address) (event.Subscription, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var currency0Rule []any
	for _, currency0Item := range currency0 {
		currency0Rule = append(currency0Rule, currency0Item)
	}
	var currency1Rule []any
	for _, currency1Item := range currency1 {
		currency1Rule = append(currency1Rule, currency1Item)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.WatchLogs(opts, "Initialize", idRule, currency0Rule, currency1Rule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PancakeInfinityPoolManagerInitialize)
				if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "Initialize", log); err != nil {
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

// ParseInitialize is a log parse operation binding the contract event 0xddfde5903015c0eb1671976c6c8f760f1328bec57f15286b6bdab2f955cab9c9.
//
// Solidity: event Initialize(bytes32 indexed id, address indexed currency0, address indexed currency1, address hooks, uint24 fee, bytes32 parameters, uint24 activeId)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) ParseInitialize(log types.Log) (*PancakeInfinityPoolManagerInitialize, error) {
	event := new(PancakeInfinityPoolManagerInitialize)
	if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "Initialize", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PancakeInfinityPoolManagerMintIterator is returned from FilterMint and is used to iterate over the raw logs and unpacked data for Mint events raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerMintIterator struct {
	Event *PancakeInfinityPoolManagerMint // Event containing the contract specifics and raw log

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
func (it *PancakeInfinityPoolManagerMintIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PancakeInfinityPoolManagerMint)
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
		it.Event = new(PancakeInfinityPoolManagerMint)
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
func (it *PancakeInfinityPoolManagerMintIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PancakeInfinityPoolManagerMintIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PancakeInfinityPoolManagerMint represents a Mint event raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerMint struct {
	Id                   [32]byte
	Sender               common.Address
	Ids                  []*big.Int
	Salt                 [32]byte
	Amounts              [][32]byte
	CompositionFeeAmount [32]byte
	FeeAmountToProtocol  [32]byte
	Raw                  types.Log // Blockchain specific contextual infos
}

// FilterMint is a free log retrieval operation binding the contract event 0x7b6bc49b385af8644341f07a67cd976bf9daf2bdd5d71668e651a3a792e318e1.
//
// Solidity: event Mint(bytes32 indexed id, address indexed sender, uint256[] ids, bytes32 salt, bytes32[] amounts, bytes32 compositionFeeAmount, bytes32 feeAmountToProtocol)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) FilterMint(opts *bind.FilterOpts, id [][32]byte, sender []common.Address) (*PancakeInfinityPoolManagerMintIterator, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.FilterLogs(opts, "Mint", idRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManagerMintIterator{contract: _PancakeInfinityPoolManager.contract, event: "Mint", logs: logs, sub: sub}, nil
}

// WatchMint is a free log subscription operation binding the contract event 0x7b6bc49b385af8644341f07a67cd976bf9daf2bdd5d71668e651a3a792e318e1.
//
// Solidity: event Mint(bytes32 indexed id, address indexed sender, uint256[] ids, bytes32 salt, bytes32[] amounts, bytes32 compositionFeeAmount, bytes32 feeAmountToProtocol)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) WatchMint(opts *bind.WatchOpts, sink chan<- *PancakeInfinityPoolManagerMint, id [][32]byte, sender []common.Address) (event.Subscription, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.WatchLogs(opts, "Mint", idRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PancakeInfinityPoolManagerMint)
				if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "Mint", log); err != nil {
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

// ParseMint is a log parse operation binding the contract event 0x7b6bc49b385af8644341f07a67cd976bf9daf2bdd5d71668e651a3a792e318e1.
//
// Solidity: event Mint(bytes32 indexed id, address indexed sender, uint256[] ids, bytes32 salt, bytes32[] amounts, bytes32 compositionFeeAmount, bytes32 feeAmountToProtocol)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) ParseMint(log types.Log) (*PancakeInfinityPoolManagerMint, error) {
	event := new(PancakeInfinityPoolManagerMint)
	if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "Mint", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PancakeInfinityPoolManagerOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerOwnershipTransferredIterator struct {
	Event *PancakeInfinityPoolManagerOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *PancakeInfinityPoolManagerOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PancakeInfinityPoolManagerOwnershipTransferred)
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
		it.Event = new(PancakeInfinityPoolManagerOwnershipTransferred)
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
func (it *PancakeInfinityPoolManagerOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PancakeInfinityPoolManagerOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PancakeInfinityPoolManagerOwnershipTransferred represents a OwnershipTransferred event raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*PancakeInfinityPoolManagerOwnershipTransferredIterator, error) {

	var previousOwnerRule []any
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []any
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManagerOwnershipTransferredIterator{contract: _PancakeInfinityPoolManager.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *PancakeInfinityPoolManagerOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []any
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []any
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PancakeInfinityPoolManagerOwnershipTransferred)
				if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) ParseOwnershipTransferred(log types.Log) (*PancakeInfinityPoolManagerOwnershipTransferred, error) {
	event := new(PancakeInfinityPoolManagerOwnershipTransferred)
	if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PancakeInfinityPoolManagerPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerPausedIterator struct {
	Event *PancakeInfinityPoolManagerPaused // Event containing the contract specifics and raw log

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
func (it *PancakeInfinityPoolManagerPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PancakeInfinityPoolManagerPaused)
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
		it.Event = new(PancakeInfinityPoolManagerPaused)
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
func (it *PancakeInfinityPoolManagerPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PancakeInfinityPoolManagerPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PancakeInfinityPoolManagerPaused represents a Paused event raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerPaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) FilterPaused(opts *bind.FilterOpts) (*PancakeInfinityPoolManagerPausedIterator, error) {

	logs, sub, err := _PancakeInfinityPoolManager.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManagerPausedIterator{contract: _PancakeInfinityPoolManager.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *PancakeInfinityPoolManagerPaused) (event.Subscription, error) {

	logs, sub, err := _PancakeInfinityPoolManager.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PancakeInfinityPoolManagerPaused)
				if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "Paused", log); err != nil {
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

// ParsePaused is a log parse operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) ParsePaused(log types.Log) (*PancakeInfinityPoolManagerPaused, error) {
	event := new(PancakeInfinityPoolManagerPaused)
	if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PancakeInfinityPoolManagerProtocolFeeControllerUpdatedIterator is returned from FilterProtocolFeeControllerUpdated and is used to iterate over the raw logs and unpacked data for ProtocolFeeControllerUpdated events raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerProtocolFeeControllerUpdatedIterator struct {
	Event *PancakeInfinityPoolManagerProtocolFeeControllerUpdated // Event containing the contract specifics and raw log

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
func (it *PancakeInfinityPoolManagerProtocolFeeControllerUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PancakeInfinityPoolManagerProtocolFeeControllerUpdated)
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
		it.Event = new(PancakeInfinityPoolManagerProtocolFeeControllerUpdated)
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
func (it *PancakeInfinityPoolManagerProtocolFeeControllerUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PancakeInfinityPoolManagerProtocolFeeControllerUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PancakeInfinityPoolManagerProtocolFeeControllerUpdated represents a ProtocolFeeControllerUpdated event raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerProtocolFeeControllerUpdated struct {
	ProtocolFeeController common.Address
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterProtocolFeeControllerUpdated is a free log retrieval operation binding the contract event 0xb4bd8ef53df690b9943d3318996006dbb82a25f54719d8c8035b516a2a5b8acc.
//
// Solidity: event ProtocolFeeControllerUpdated(address indexed protocolFeeController)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) FilterProtocolFeeControllerUpdated(opts *bind.FilterOpts, protocolFeeController []common.Address) (*PancakeInfinityPoolManagerProtocolFeeControllerUpdatedIterator, error) {

	var protocolFeeControllerRule []any
	for _, protocolFeeControllerItem := range protocolFeeController {
		protocolFeeControllerRule = append(protocolFeeControllerRule, protocolFeeControllerItem)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.FilterLogs(opts, "ProtocolFeeControllerUpdated", protocolFeeControllerRule)
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManagerProtocolFeeControllerUpdatedIterator{contract: _PancakeInfinityPoolManager.contract, event: "ProtocolFeeControllerUpdated", logs: logs, sub: sub}, nil
}

// WatchProtocolFeeControllerUpdated is a free log subscription operation binding the contract event 0xb4bd8ef53df690b9943d3318996006dbb82a25f54719d8c8035b516a2a5b8acc.
//
// Solidity: event ProtocolFeeControllerUpdated(address indexed protocolFeeController)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) WatchProtocolFeeControllerUpdated(opts *bind.WatchOpts, sink chan<- *PancakeInfinityPoolManagerProtocolFeeControllerUpdated, protocolFeeController []common.Address) (event.Subscription, error) {

	var protocolFeeControllerRule []any
	for _, protocolFeeControllerItem := range protocolFeeController {
		protocolFeeControllerRule = append(protocolFeeControllerRule, protocolFeeControllerItem)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.WatchLogs(opts, "ProtocolFeeControllerUpdated", protocolFeeControllerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PancakeInfinityPoolManagerProtocolFeeControllerUpdated)
				if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "ProtocolFeeControllerUpdated", log); err != nil {
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

// ParseProtocolFeeControllerUpdated is a log parse operation binding the contract event 0xb4bd8ef53df690b9943d3318996006dbb82a25f54719d8c8035b516a2a5b8acc.
//
// Solidity: event ProtocolFeeControllerUpdated(address indexed protocolFeeController)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) ParseProtocolFeeControllerUpdated(log types.Log) (*PancakeInfinityPoolManagerProtocolFeeControllerUpdated, error) {
	event := new(PancakeInfinityPoolManagerProtocolFeeControllerUpdated)
	if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "ProtocolFeeControllerUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PancakeInfinityPoolManagerProtocolFeeUpdatedIterator is returned from FilterProtocolFeeUpdated and is used to iterate over the raw logs and unpacked data for ProtocolFeeUpdated events raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerProtocolFeeUpdatedIterator struct {
	Event *PancakeInfinityPoolManagerProtocolFeeUpdated // Event containing the contract specifics and raw log

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
func (it *PancakeInfinityPoolManagerProtocolFeeUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PancakeInfinityPoolManagerProtocolFeeUpdated)
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
		it.Event = new(PancakeInfinityPoolManagerProtocolFeeUpdated)
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
func (it *PancakeInfinityPoolManagerProtocolFeeUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PancakeInfinityPoolManagerProtocolFeeUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PancakeInfinityPoolManagerProtocolFeeUpdated represents a ProtocolFeeUpdated event raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerProtocolFeeUpdated struct {
	Id          [32]byte
	ProtocolFee *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterProtocolFeeUpdated is a free log retrieval operation binding the contract event 0xe9c42593e71f84403b84352cd168d693e2c9fcd1fdbcc3feb21d92b43e6696f9.
//
// Solidity: event ProtocolFeeUpdated(bytes32 indexed id, uint24 protocolFee)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) FilterProtocolFeeUpdated(opts *bind.FilterOpts, id [][32]byte) (*PancakeInfinityPoolManagerProtocolFeeUpdatedIterator, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.FilterLogs(opts, "ProtocolFeeUpdated", idRule)
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManagerProtocolFeeUpdatedIterator{contract: _PancakeInfinityPoolManager.contract, event: "ProtocolFeeUpdated", logs: logs, sub: sub}, nil
}

// WatchProtocolFeeUpdated is a free log subscription operation binding the contract event 0xe9c42593e71f84403b84352cd168d693e2c9fcd1fdbcc3feb21d92b43e6696f9.
//
// Solidity: event ProtocolFeeUpdated(bytes32 indexed id, uint24 protocolFee)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) WatchProtocolFeeUpdated(opts *bind.WatchOpts, sink chan<- *PancakeInfinityPoolManagerProtocolFeeUpdated, id [][32]byte) (event.Subscription, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.WatchLogs(opts, "ProtocolFeeUpdated", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PancakeInfinityPoolManagerProtocolFeeUpdated)
				if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "ProtocolFeeUpdated", log); err != nil {
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

// ParseProtocolFeeUpdated is a log parse operation binding the contract event 0xe9c42593e71f84403b84352cd168d693e2c9fcd1fdbcc3feb21d92b43e6696f9.
//
// Solidity: event ProtocolFeeUpdated(bytes32 indexed id, uint24 protocolFee)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) ParseProtocolFeeUpdated(log types.Log) (*PancakeInfinityPoolManagerProtocolFeeUpdated, error) {
	event := new(PancakeInfinityPoolManagerProtocolFeeUpdated)
	if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "ProtocolFeeUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PancakeInfinityPoolManagerSetMaxBinStepIterator is returned from FilterSetMaxBinStep and is used to iterate over the raw logs and unpacked data for SetMaxBinStep events raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerSetMaxBinStepIterator struct {
	Event *PancakeInfinityPoolManagerSetMaxBinStep // Event containing the contract specifics and raw log

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
func (it *PancakeInfinityPoolManagerSetMaxBinStepIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PancakeInfinityPoolManagerSetMaxBinStep)
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
		it.Event = new(PancakeInfinityPoolManagerSetMaxBinStep)
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
func (it *PancakeInfinityPoolManagerSetMaxBinStepIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PancakeInfinityPoolManagerSetMaxBinStepIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PancakeInfinityPoolManagerSetMaxBinStep represents a SetMaxBinStep event raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerSetMaxBinStep struct {
	MaxBinStep uint16
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterSetMaxBinStep is a free log retrieval operation binding the contract event 0x02172f85720dcdece86093a50de7b9578583a2b7a567992be89d92868feba494.
//
// Solidity: event SetMaxBinStep(uint16 maxBinStep)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) FilterSetMaxBinStep(opts *bind.FilterOpts) (*PancakeInfinityPoolManagerSetMaxBinStepIterator, error) {

	logs, sub, err := _PancakeInfinityPoolManager.contract.FilterLogs(opts, "SetMaxBinStep")
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManagerSetMaxBinStepIterator{contract: _PancakeInfinityPoolManager.contract, event: "SetMaxBinStep", logs: logs, sub: sub}, nil
}

// WatchSetMaxBinStep is a free log subscription operation binding the contract event 0x02172f85720dcdece86093a50de7b9578583a2b7a567992be89d92868feba494.
//
// Solidity: event SetMaxBinStep(uint16 maxBinStep)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) WatchSetMaxBinStep(opts *bind.WatchOpts, sink chan<- *PancakeInfinityPoolManagerSetMaxBinStep) (event.Subscription, error) {

	logs, sub, err := _PancakeInfinityPoolManager.contract.WatchLogs(opts, "SetMaxBinStep")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PancakeInfinityPoolManagerSetMaxBinStep)
				if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "SetMaxBinStep", log); err != nil {
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

// ParseSetMaxBinStep is a log parse operation binding the contract event 0x02172f85720dcdece86093a50de7b9578583a2b7a567992be89d92868feba494.
//
// Solidity: event SetMaxBinStep(uint16 maxBinStep)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) ParseSetMaxBinStep(log types.Log) (*PancakeInfinityPoolManagerSetMaxBinStep, error) {
	event := new(PancakeInfinityPoolManagerSetMaxBinStep)
	if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "SetMaxBinStep", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PancakeInfinityPoolManagerSetMinBinSharesForDonateIterator is returned from FilterSetMinBinSharesForDonate and is used to iterate over the raw logs and unpacked data for SetMinBinSharesForDonate events raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerSetMinBinSharesForDonateIterator struct {
	Event *PancakeInfinityPoolManagerSetMinBinSharesForDonate // Event containing the contract specifics and raw log

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
func (it *PancakeInfinityPoolManagerSetMinBinSharesForDonateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PancakeInfinityPoolManagerSetMinBinSharesForDonate)
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
		it.Event = new(PancakeInfinityPoolManagerSetMinBinSharesForDonate)
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
func (it *PancakeInfinityPoolManagerSetMinBinSharesForDonateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PancakeInfinityPoolManagerSetMinBinSharesForDonateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PancakeInfinityPoolManagerSetMinBinSharesForDonate represents a SetMinBinSharesForDonate event raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerSetMinBinSharesForDonate struct {
	MinLiquidity *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterSetMinBinSharesForDonate is a free log retrieval operation binding the contract event 0xd752b38d4cbf2c0d2ceecee3d2f43840ec77bd9cde6b733b94cc065bdab931a3.
//
// Solidity: event SetMinBinSharesForDonate(uint256 minLiquidity)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) FilterSetMinBinSharesForDonate(opts *bind.FilterOpts) (*PancakeInfinityPoolManagerSetMinBinSharesForDonateIterator, error) {

	logs, sub, err := _PancakeInfinityPoolManager.contract.FilterLogs(opts, "SetMinBinSharesForDonate")
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManagerSetMinBinSharesForDonateIterator{contract: _PancakeInfinityPoolManager.contract, event: "SetMinBinSharesForDonate", logs: logs, sub: sub}, nil
}

// WatchSetMinBinSharesForDonate is a free log subscription operation binding the contract event 0xd752b38d4cbf2c0d2ceecee3d2f43840ec77bd9cde6b733b94cc065bdab931a3.
//
// Solidity: event SetMinBinSharesForDonate(uint256 minLiquidity)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) WatchSetMinBinSharesForDonate(opts *bind.WatchOpts, sink chan<- *PancakeInfinityPoolManagerSetMinBinSharesForDonate) (event.Subscription, error) {

	logs, sub, err := _PancakeInfinityPoolManager.contract.WatchLogs(opts, "SetMinBinSharesForDonate")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PancakeInfinityPoolManagerSetMinBinSharesForDonate)
				if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "SetMinBinSharesForDonate", log); err != nil {
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

// ParseSetMinBinSharesForDonate is a log parse operation binding the contract event 0xd752b38d4cbf2c0d2ceecee3d2f43840ec77bd9cde6b733b94cc065bdab931a3.
//
// Solidity: event SetMinBinSharesForDonate(uint256 minLiquidity)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) ParseSetMinBinSharesForDonate(log types.Log) (*PancakeInfinityPoolManagerSetMinBinSharesForDonate, error) {
	event := new(PancakeInfinityPoolManagerSetMinBinSharesForDonate)
	if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "SetMinBinSharesForDonate", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PancakeInfinityPoolManagerSwapIterator is returned from FilterSwap and is used to iterate over the raw logs and unpacked data for Swap events raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerSwapIterator struct {
	Event *PancakeInfinityPoolManagerSwap // Event containing the contract specifics and raw log

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
func (it *PancakeInfinityPoolManagerSwapIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PancakeInfinityPoolManagerSwap)
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
		it.Event = new(PancakeInfinityPoolManagerSwap)
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
func (it *PancakeInfinityPoolManagerSwapIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PancakeInfinityPoolManagerSwapIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PancakeInfinityPoolManagerSwap represents a Swap event raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerSwap struct {
	Id          [32]byte
	Sender      common.Address
	Amount0     *big.Int
	Amount1     *big.Int
	ActiveId    *big.Int
	Fee         *big.Int
	ProtocolFee uint16
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterSwap is a free log retrieval operation binding the contract event 0x3e8aae37f890eb1f9d63dd4d2062f3f0be757848a0f0760e4f3e53dad556e861.
//
// Solidity: event Swap(bytes32 indexed id, address indexed sender, int128 amount0, int128 amount1, uint24 activeId, uint24 fee, uint16 protocolFee)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) FilterSwap(opts *bind.FilterOpts, id [][32]byte, sender []common.Address) (*PancakeInfinityPoolManagerSwapIterator, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.FilterLogs(opts, "Swap", idRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManagerSwapIterator{contract: _PancakeInfinityPoolManager.contract, event: "Swap", logs: logs, sub: sub}, nil
}

// WatchSwap is a free log subscription operation binding the contract event 0x3e8aae37f890eb1f9d63dd4d2062f3f0be757848a0f0760e4f3e53dad556e861.
//
// Solidity: event Swap(bytes32 indexed id, address indexed sender, int128 amount0, int128 amount1, uint24 activeId, uint24 fee, uint16 protocolFee)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) WatchSwap(opts *bind.WatchOpts, sink chan<- *PancakeInfinityPoolManagerSwap, id [][32]byte, sender []common.Address) (event.Subscription, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _PancakeInfinityPoolManager.contract.WatchLogs(opts, "Swap", idRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PancakeInfinityPoolManagerSwap)
				if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "Swap", log); err != nil {
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

// ParseSwap is a log parse operation binding the contract event 0x3e8aae37f890eb1f9d63dd4d2062f3f0be757848a0f0760e4f3e53dad556e861.
//
// Solidity: event Swap(bytes32 indexed id, address indexed sender, int128 amount0, int128 amount1, uint24 activeId, uint24 fee, uint16 protocolFee)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) ParseSwap(log types.Log) (*PancakeInfinityPoolManagerSwap, error) {
	event := new(PancakeInfinityPoolManagerSwap)
	if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "Swap", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PancakeInfinityPoolManagerUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerUnpausedIterator struct {
	Event *PancakeInfinityPoolManagerUnpaused // Event containing the contract specifics and raw log

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
func (it *PancakeInfinityPoolManagerUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PancakeInfinityPoolManagerUnpaused)
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
		it.Event = new(PancakeInfinityPoolManagerUnpaused)
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
func (it *PancakeInfinityPoolManagerUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PancakeInfinityPoolManagerUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PancakeInfinityPoolManagerUnpaused represents a Unpaused event raised by the PancakeInfinityPoolManager contract.
type PancakeInfinityPoolManagerUnpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) FilterUnpaused(opts *bind.FilterOpts) (*PancakeInfinityPoolManagerUnpausedIterator, error) {

	logs, sub, err := _PancakeInfinityPoolManager.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &PancakeInfinityPoolManagerUnpausedIterator{contract: _PancakeInfinityPoolManager.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *PancakeInfinityPoolManagerUnpaused) (event.Subscription, error) {

	logs, sub, err := _PancakeInfinityPoolManager.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PancakeInfinityPoolManagerUnpaused)
				if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "Unpaused", log); err != nil {
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

// ParseUnpaused is a log parse operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_PancakeInfinityPoolManager *PancakeInfinityPoolManagerFilterer) ParseUnpaused(log types.Log) (*PancakeInfinityPoolManagerUnpaused, error) {
	event := new(PancakeInfinityPoolManagerUnpaused)
	if err := _PancakeInfinityPoolManager.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
