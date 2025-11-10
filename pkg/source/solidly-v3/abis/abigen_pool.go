// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package abis

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

// PoolMetaData contains all meta data concerning the Pool contract.
var PoolMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"name\":\"Burn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount0\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount1\",\"type\":\"uint128\"}],\"name\":\"Collect\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount0\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount1\",\"type\":\"uint128\"}],\"name\":\"CollectProtocol\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"paid0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"paid1\",\"type\":\"uint256\"}],\"name\":\"Flash\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"Initialize\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"name\":\"Mint\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"feeOld\",\"type\":\"uint24\"},{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"feeNew\",\"type\":\"uint24\"}],\"name\":\"SetFee\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"amount0\",\"type\":\"int256\"},{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"amount1\",\"type\":\"int256\"},{\"indexed\":false,\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"liquidity\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"Swap\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"},{\"internalType\":\"uint256\",\"name\":\"amount0Min\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1Min\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"name\":\"burn\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"}],\"name\":\"burn\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amountToBurn\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount0ToCollect\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1ToCollect\",\"type\":\"uint128\"}],\"name\":\"burnAndCollect\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount0FromBurn\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1FromBurn\",\"type\":\"uint256\"},{\"internalType\":\"uint128\",\"name\":\"amount0Collected\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1Collected\",\"type\":\"uint128\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amountToBurn\",\"type\":\"uint128\"},{\"internalType\":\"uint256\",\"name\":\"amount0FromBurnMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1FromBurnMin\",\"type\":\"uint256\"},{\"internalType\":\"uint128\",\"name\":\"amount0ToCollect\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1ToCollect\",\"type\":\"uint128\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"name\":\"burnAndCollect\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount0FromBurn\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1FromBurn\",\"type\":\"uint256\"},{\"internalType\":\"uint128\",\"name\":\"amount0Collected\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1Collected\",\"type\":\"uint128\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amount0Requested\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1Requested\",\"type\":\"uint128\"}],\"name\":\"collect\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"amount0\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1\",\"type\":\"uint128\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint128\",\"name\":\"amount0Requested\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1Requested\",\"type\":\"uint128\"}],\"name\":\"collectProtocol\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"amount0\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1\",\"type\":\"uint128\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"factory\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"flash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"liquidity\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxLiquidityPerTick\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"mint\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"},{\"internalType\":\"uint256\",\"name\":\"amount0Min\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1Min\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"name\":\"mint\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"},{\"internalType\":\"uint256\",\"name\":\"amount0Min\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1Min\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"mint\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"}],\"name\":\"mint\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"poolFees\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"token0\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"token1\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"positions\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"liquidity\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"tokensOwed0\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"tokensOwed1\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountSpecified\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceLimitX96\",\"type\":\"uint160\"}],\"name\":\"quoteSwap\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"amount0\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"amount1\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96After\",\"type\":\"uint160\"},{\"internalType\":\"int24\",\"name\":\"tickAfter\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"liquidityAfter\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"}],\"name\":\"setFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"slot0\",\"outputs\":[{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"bool\",\"name\":\"unlocked\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountSpecified\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceLimitX96\",\"type\":\"uint160\"},{\"internalType\":\"uint256\",\"name\":\"amountLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"swap\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"amount0\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"amount1\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountSpecified\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceLimitX96\",\"type\":\"uint160\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"swap\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"amount0\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"amount1\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountSpecified\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceLimitX96\",\"type\":\"uint160\"},{\"internalType\":\"uint256\",\"name\":\"amountLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"trackingCode\",\"type\":\"uint256\"}],\"name\":\"swap\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"amount0\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"amount1\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountSpecified\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceLimitX96\",\"type\":\"uint160\"},{\"internalType\":\"uint256\",\"name\":\"trackingCode\",\"type\":\"uint256\"}],\"name\":\"swap\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"amount0\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"amount1\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountSpecified\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceLimitX96\",\"type\":\"uint160\"},{\"internalType\":\"uint256\",\"name\":\"amountLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"name\":\"swap\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"amount0\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"amount1\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountSpecified\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceLimitX96\",\"type\":\"uint160\"},{\"internalType\":\"uint256\",\"name\":\"amountLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"trackingCode\",\"type\":\"uint256\"}],\"name\":\"swap\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"amount0\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"amount1\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountSpecified\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceLimitX96\",\"type\":\"uint160\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"trackingCode\",\"type\":\"uint256\"}],\"name\":\"swap\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"amount0\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"amount1\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountSpecified\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceLimitX96\",\"type\":\"uint160\"}],\"name\":\"swap\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"amount0\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"amount1\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int16\",\"name\":\"\",\"type\":\"int16\"}],\"name\":\"tickBitmap\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"tickSpacing\",\"outputs\":[{\"internalType\":\"int24\",\"name\":\"\",\"type\":\"int24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int24\",\"name\":\"\",\"type\":\"int24\"}],\"name\":\"ticks\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"liquidityGross\",\"type\":\"uint128\"},{\"internalType\":\"int128\",\"name\":\"liquidityNet\",\"type\":\"int128\"},{\"internalType\":\"bool\",\"name\":\"initialized\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"token0\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"token1\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// PoolABI is the input ABI used to generate the binding from.
// Deprecated: Use PoolMetaData.ABI instead.
var PoolABI = PoolMetaData.ABI

// Pool is an auto generated Go binding around an Ethereum contract.
type Pool struct {
	PoolCaller     // Read-only binding to the contract
	PoolTransactor // Write-only binding to the contract
	PoolFilterer   // Log filterer for contract events
}

// PoolCaller is an auto generated read-only Go binding around an Ethereum contract.
type PoolCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PoolTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PoolTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PoolFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PoolFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PoolSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PoolSession struct {
	Contract     *Pool             // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PoolCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PoolCallerSession struct {
	Contract *PoolCaller   // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// PoolTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PoolTransactorSession struct {
	Contract     *PoolTransactor   // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PoolRaw is an auto generated low-level Go binding around an Ethereum contract.
type PoolRaw struct {
	Contract *Pool // Generic contract binding to access the raw methods on
}

// PoolCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PoolCallerRaw struct {
	Contract *PoolCaller // Generic read-only contract binding to access the raw methods on
}

// PoolTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PoolTransactorRaw struct {
	Contract *PoolTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPool creates a new instance of Pool, bound to a specific deployed contract.
func NewPool(address common.Address, backend bind.ContractBackend) (*Pool, error) {
	contract, err := bindPool(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Pool{PoolCaller: PoolCaller{contract: contract}, PoolTransactor: PoolTransactor{contract: contract}, PoolFilterer: PoolFilterer{contract: contract}}, nil
}

// NewPoolCaller creates a new read-only instance of Pool, bound to a specific deployed contract.
func NewPoolCaller(address common.Address, caller bind.ContractCaller) (*PoolCaller, error) {
	contract, err := bindPool(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PoolCaller{contract: contract}, nil
}

// NewPoolTransactor creates a new write-only instance of Pool, bound to a specific deployed contract.
func NewPoolTransactor(address common.Address, transactor bind.ContractTransactor) (*PoolTransactor, error) {
	contract, err := bindPool(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PoolTransactor{contract: contract}, nil
}

// NewPoolFilterer creates a new log filterer instance of Pool, bound to a specific deployed contract.
func NewPoolFilterer(address common.Address, filterer bind.ContractFilterer) (*PoolFilterer, error) {
	contract, err := bindPool(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PoolFilterer{contract: contract}, nil
}

// bindPool binds a generic wrapper to an already deployed contract.
func bindPool(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := PoolMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Pool *PoolRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Pool.Contract.PoolCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Pool *PoolRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Pool.Contract.PoolTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Pool *PoolRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Pool.Contract.PoolTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Pool *PoolCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Pool.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Pool *PoolTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Pool.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Pool *PoolTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Pool.Contract.contract.Transact(opts, method, params...)
}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_Pool *PoolCaller) Factory(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "factory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_Pool *PoolSession) Factory() (common.Address, error) {
	return _Pool.Contract.Factory(&_Pool.CallOpts)
}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_Pool *PoolCallerSession) Factory() (common.Address, error) {
	return _Pool.Contract.Factory(&_Pool.CallOpts)
}

// Liquidity is a free data retrieval call binding the contract method 0x1a686502.
//
// Solidity: function liquidity() view returns(uint128)
func (_Pool *PoolCaller) Liquidity(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "liquidity")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Liquidity is a free data retrieval call binding the contract method 0x1a686502.
//
// Solidity: function liquidity() view returns(uint128)
func (_Pool *PoolSession) Liquidity() (*big.Int, error) {
	return _Pool.Contract.Liquidity(&_Pool.CallOpts)
}

// Liquidity is a free data retrieval call binding the contract method 0x1a686502.
//
// Solidity: function liquidity() view returns(uint128)
func (_Pool *PoolCallerSession) Liquidity() (*big.Int, error) {
	return _Pool.Contract.Liquidity(&_Pool.CallOpts)
}

// MaxLiquidityPerTick is a free data retrieval call binding the contract method 0x70cf754a.
//
// Solidity: function maxLiquidityPerTick() view returns(uint128)
func (_Pool *PoolCaller) MaxLiquidityPerTick(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "maxLiquidityPerTick")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxLiquidityPerTick is a free data retrieval call binding the contract method 0x70cf754a.
//
// Solidity: function maxLiquidityPerTick() view returns(uint128)
func (_Pool *PoolSession) MaxLiquidityPerTick() (*big.Int, error) {
	return _Pool.Contract.MaxLiquidityPerTick(&_Pool.CallOpts)
}

// MaxLiquidityPerTick is a free data retrieval call binding the contract method 0x70cf754a.
//
// Solidity: function maxLiquidityPerTick() view returns(uint128)
func (_Pool *PoolCallerSession) MaxLiquidityPerTick() (*big.Int, error) {
	return _Pool.Contract.MaxLiquidityPerTick(&_Pool.CallOpts)
}

// PoolFees is a free data retrieval call binding the contract method 0x33580959.
//
// Solidity: function poolFees() view returns(uint128 token0, uint128 token1)
func (_Pool *PoolCaller) PoolFees(opts *bind.CallOpts) (struct {
	Token0 *big.Int
	Token1 *big.Int
}, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "poolFees")

	outstruct := new(struct {
		Token0 *big.Int
		Token1 *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Token0 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Token1 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// PoolFees is a free data retrieval call binding the contract method 0x33580959.
//
// Solidity: function poolFees() view returns(uint128 token0, uint128 token1)
func (_Pool *PoolSession) PoolFees() (struct {
	Token0 *big.Int
	Token1 *big.Int
}, error) {
	return _Pool.Contract.PoolFees(&_Pool.CallOpts)
}

// PoolFees is a free data retrieval call binding the contract method 0x33580959.
//
// Solidity: function poolFees() view returns(uint128 token0, uint128 token1)
func (_Pool *PoolCallerSession) PoolFees() (struct {
	Token0 *big.Int
	Token1 *big.Int
}, error) {
	return _Pool.Contract.PoolFees(&_Pool.CallOpts)
}

// Positions is a free data retrieval call binding the contract method 0x514ea4bf.
//
// Solidity: function positions(bytes32 ) view returns(uint128 liquidity, uint128 tokensOwed0, uint128 tokensOwed1)
func (_Pool *PoolCaller) Positions(opts *bind.CallOpts, arg0 [32]byte) (struct {
	Liquidity   *big.Int
	TokensOwed0 *big.Int
	TokensOwed1 *big.Int
}, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "positions", arg0)

	outstruct := new(struct {
		Liquidity   *big.Int
		TokensOwed0 *big.Int
		TokensOwed1 *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Liquidity = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.TokensOwed0 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.TokensOwed1 = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Positions is a free data retrieval call binding the contract method 0x514ea4bf.
//
// Solidity: function positions(bytes32 ) view returns(uint128 liquidity, uint128 tokensOwed0, uint128 tokensOwed1)
func (_Pool *PoolSession) Positions(arg0 [32]byte) (struct {
	Liquidity   *big.Int
	TokensOwed0 *big.Int
	TokensOwed1 *big.Int
}, error) {
	return _Pool.Contract.Positions(&_Pool.CallOpts, arg0)
}

// Positions is a free data retrieval call binding the contract method 0x514ea4bf.
//
// Solidity: function positions(bytes32 ) view returns(uint128 liquidity, uint128 tokensOwed0, uint128 tokensOwed1)
func (_Pool *PoolCallerSession) Positions(arg0 [32]byte) (struct {
	Liquidity   *big.Int
	TokensOwed0 *big.Int
	TokensOwed1 *big.Int
}, error) {
	return _Pool.Contract.Positions(&_Pool.CallOpts, arg0)
}

// QuoteSwap is a free data retrieval call binding the contract method 0x5d41514b.
//
// Solidity: function quoteSwap(bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96) view returns(int256 amount0, int256 amount1, uint160 sqrtPriceX96After, int24 tickAfter, uint128 liquidityAfter)
func (_Pool *PoolCaller) QuoteSwap(opts *bind.CallOpts, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int) (struct {
	Amount0           *big.Int
	Amount1           *big.Int
	SqrtPriceX96After *big.Int
	TickAfter         *big.Int
	LiquidityAfter    *big.Int
}, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "quoteSwap", zeroForOne, amountSpecified, sqrtPriceLimitX96)

	outstruct := new(struct {
		Amount0           *big.Int
		Amount1           *big.Int
		SqrtPriceX96After *big.Int
		TickAfter         *big.Int
		LiquidityAfter    *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Amount0 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Amount1 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.SqrtPriceX96After = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.TickAfter = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.LiquidityAfter = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// QuoteSwap is a free data retrieval call binding the contract method 0x5d41514b.
//
// Solidity: function quoteSwap(bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96) view returns(int256 amount0, int256 amount1, uint160 sqrtPriceX96After, int24 tickAfter, uint128 liquidityAfter)
func (_Pool *PoolSession) QuoteSwap(zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int) (struct {
	Amount0           *big.Int
	Amount1           *big.Int
	SqrtPriceX96After *big.Int
	TickAfter         *big.Int
	LiquidityAfter    *big.Int
}, error) {
	return _Pool.Contract.QuoteSwap(&_Pool.CallOpts, zeroForOne, amountSpecified, sqrtPriceLimitX96)
}

// QuoteSwap is a free data retrieval call binding the contract method 0x5d41514b.
//
// Solidity: function quoteSwap(bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96) view returns(int256 amount0, int256 amount1, uint160 sqrtPriceX96After, int24 tickAfter, uint128 liquidityAfter)
func (_Pool *PoolCallerSession) QuoteSwap(zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int) (struct {
	Amount0           *big.Int
	Amount1           *big.Int
	SqrtPriceX96After *big.Int
	TickAfter         *big.Int
	LiquidityAfter    *big.Int
}, error) {
	return _Pool.Contract.QuoteSwap(&_Pool.CallOpts, zeroForOne, amountSpecified, sqrtPriceLimitX96)
}

// Slot0 is a free data retrieval call binding the contract method 0x3850c7bd.
//
// Solidity: function slot0() view returns(uint160 sqrtPriceX96, int24 tick, uint24 fee, bool unlocked)
func (_Pool *PoolCaller) Slot0(opts *bind.CallOpts) (struct {
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	Fee          *big.Int
	Unlocked     bool
}, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "slot0")

	outstruct := new(struct {
		SqrtPriceX96 *big.Int
		Tick         *big.Int
		Fee          *big.Int
		Unlocked     bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.SqrtPriceX96 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Tick = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Fee = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.Unlocked = *abi.ConvertType(out[3], new(bool)).(*bool)

	return *outstruct, err

}

// Slot0 is a free data retrieval call binding the contract method 0x3850c7bd.
//
// Solidity: function slot0() view returns(uint160 sqrtPriceX96, int24 tick, uint24 fee, bool unlocked)
func (_Pool *PoolSession) Slot0() (struct {
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	Fee          *big.Int
	Unlocked     bool
}, error) {
	return _Pool.Contract.Slot0(&_Pool.CallOpts)
}

// Slot0 is a free data retrieval call binding the contract method 0x3850c7bd.
//
// Solidity: function slot0() view returns(uint160 sqrtPriceX96, int24 tick, uint24 fee, bool unlocked)
func (_Pool *PoolCallerSession) Slot0() (struct {
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	Fee          *big.Int
	Unlocked     bool
}, error) {
	return _Pool.Contract.Slot0(&_Pool.CallOpts)
}

// TickBitmap is a free data retrieval call binding the contract method 0x5339c296.
//
// Solidity: function tickBitmap(int16 ) view returns(uint256)
func (_Pool *PoolCaller) TickBitmap(opts *bind.CallOpts, arg0 int16) (*big.Int, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "tickBitmap", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TickBitmap is a free data retrieval call binding the contract method 0x5339c296.
//
// Solidity: function tickBitmap(int16 ) view returns(uint256)
func (_Pool *PoolSession) TickBitmap(arg0 int16) (*big.Int, error) {
	return _Pool.Contract.TickBitmap(&_Pool.CallOpts, arg0)
}

// TickBitmap is a free data retrieval call binding the contract method 0x5339c296.
//
// Solidity: function tickBitmap(int16 ) view returns(uint256)
func (_Pool *PoolCallerSession) TickBitmap(arg0 int16) (*big.Int, error) {
	return _Pool.Contract.TickBitmap(&_Pool.CallOpts, arg0)
}

// TickSpacing is a free data retrieval call binding the contract method 0xd0c93a7c.
//
// Solidity: function tickSpacing() view returns(int24)
func (_Pool *PoolCaller) TickSpacing(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "tickSpacing")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TickSpacing is a free data retrieval call binding the contract method 0xd0c93a7c.
//
// Solidity: function tickSpacing() view returns(int24)
func (_Pool *PoolSession) TickSpacing() (*big.Int, error) {
	return _Pool.Contract.TickSpacing(&_Pool.CallOpts)
}

// TickSpacing is a free data retrieval call binding the contract method 0xd0c93a7c.
//
// Solidity: function tickSpacing() view returns(int24)
func (_Pool *PoolCallerSession) TickSpacing() (*big.Int, error) {
	return _Pool.Contract.TickSpacing(&_Pool.CallOpts)
}

// Ticks is a free data retrieval call binding the contract method 0xf30dba93.
//
// Solidity: function ticks(int24 ) view returns(uint128 liquidityGross, int128 liquidityNet, bool initialized)
func (_Pool *PoolCaller) Ticks(opts *bind.CallOpts, arg0 *big.Int) (struct {
	LiquidityGross *big.Int
	LiquidityNet   *big.Int
	Initialized    bool
}, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "ticks", arg0)

	outstruct := new(struct {
		LiquidityGross *big.Int
		LiquidityNet   *big.Int
		Initialized    bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.LiquidityGross = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.LiquidityNet = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Initialized = *abi.ConvertType(out[2], new(bool)).(*bool)

	return *outstruct, err

}

// Ticks is a free data retrieval call binding the contract method 0xf30dba93.
//
// Solidity: function ticks(int24 ) view returns(uint128 liquidityGross, int128 liquidityNet, bool initialized)
func (_Pool *PoolSession) Ticks(arg0 *big.Int) (struct {
	LiquidityGross *big.Int
	LiquidityNet   *big.Int
	Initialized    bool
}, error) {
	return _Pool.Contract.Ticks(&_Pool.CallOpts, arg0)
}

// Ticks is a free data retrieval call binding the contract method 0xf30dba93.
//
// Solidity: function ticks(int24 ) view returns(uint128 liquidityGross, int128 liquidityNet, bool initialized)
func (_Pool *PoolCallerSession) Ticks(arg0 *big.Int) (struct {
	LiquidityGross *big.Int
	LiquidityNet   *big.Int
	Initialized    bool
}, error) {
	return _Pool.Contract.Ticks(&_Pool.CallOpts, arg0)
}

// Token0 is a free data retrieval call binding the contract method 0x0dfe1681.
//
// Solidity: function token0() view returns(address)
func (_Pool *PoolCaller) Token0(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "token0")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Token0 is a free data retrieval call binding the contract method 0x0dfe1681.
//
// Solidity: function token0() view returns(address)
func (_Pool *PoolSession) Token0() (common.Address, error) {
	return _Pool.Contract.Token0(&_Pool.CallOpts)
}

// Token0 is a free data retrieval call binding the contract method 0x0dfe1681.
//
// Solidity: function token0() view returns(address)
func (_Pool *PoolCallerSession) Token0() (common.Address, error) {
	return _Pool.Contract.Token0(&_Pool.CallOpts)
}

// Token1 is a free data retrieval call binding the contract method 0xd21220a7.
//
// Solidity: function token1() view returns(address)
func (_Pool *PoolCaller) Token1(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "token1")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Token1 is a free data retrieval call binding the contract method 0xd21220a7.
//
// Solidity: function token1() view returns(address)
func (_Pool *PoolSession) Token1() (common.Address, error) {
	return _Pool.Contract.Token1(&_Pool.CallOpts)
}

// Token1 is a free data retrieval call binding the contract method 0xd21220a7.
//
// Solidity: function token1() view returns(address)
func (_Pool *PoolCallerSession) Token1() (common.Address, error) {
	return _Pool.Contract.Token1(&_Pool.CallOpts)
}

// Burn is a paid mutator transaction binding the contract method 0x7074d2df.
//
// Solidity: function burn(int24 tickLower, int24 tickUpper, uint128 amount, uint256 amount0Min, uint256 amount1Min, uint256 deadline) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolTransactor) Burn(opts *bind.TransactOpts, tickLower *big.Int, tickUpper *big.Int, amount *big.Int, amount0Min *big.Int, amount1Min *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "burn", tickLower, tickUpper, amount, amount0Min, amount1Min, deadline)
}

// Burn is a paid mutator transaction binding the contract method 0x7074d2df.
//
// Solidity: function burn(int24 tickLower, int24 tickUpper, uint128 amount, uint256 amount0Min, uint256 amount1Min, uint256 deadline) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolSession) Burn(tickLower *big.Int, tickUpper *big.Int, amount *big.Int, amount0Min *big.Int, amount1Min *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Burn(&_Pool.TransactOpts, tickLower, tickUpper, amount, amount0Min, amount1Min, deadline)
}

// Burn is a paid mutator transaction binding the contract method 0x7074d2df.
//
// Solidity: function burn(int24 tickLower, int24 tickUpper, uint128 amount, uint256 amount0Min, uint256 amount1Min, uint256 deadline) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolTransactorSession) Burn(tickLower *big.Int, tickUpper *big.Int, amount *big.Int, amount0Min *big.Int, amount1Min *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Burn(&_Pool.TransactOpts, tickLower, tickUpper, amount, amount0Min, amount1Min, deadline)
}

// Burn0 is a paid mutator transaction binding the contract method 0xa34123a7.
//
// Solidity: function burn(int24 tickLower, int24 tickUpper, uint128 amount) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolTransactor) Burn0(opts *bind.TransactOpts, tickLower *big.Int, tickUpper *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "burn0", tickLower, tickUpper, amount)
}

// Burn0 is a paid mutator transaction binding the contract method 0xa34123a7.
//
// Solidity: function burn(int24 tickLower, int24 tickUpper, uint128 amount) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolSession) Burn0(tickLower *big.Int, tickUpper *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Burn0(&_Pool.TransactOpts, tickLower, tickUpper, amount)
}

// Burn0 is a paid mutator transaction binding the contract method 0xa34123a7.
//
// Solidity: function burn(int24 tickLower, int24 tickUpper, uint128 amount) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolTransactorSession) Burn0(tickLower *big.Int, tickUpper *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Burn0(&_Pool.TransactOpts, tickLower, tickUpper, amount)
}

// BurnAndCollect is a paid mutator transaction binding the contract method 0x9c421449.
//
// Solidity: function burnAndCollect(address recipient, int24 tickLower, int24 tickUpper, uint128 amountToBurn, uint128 amount0ToCollect, uint128 amount1ToCollect) returns(uint256 amount0FromBurn, uint256 amount1FromBurn, uint128 amount0Collected, uint128 amount1Collected)
func (_Pool *PoolTransactor) BurnAndCollect(opts *bind.TransactOpts, recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amountToBurn *big.Int, amount0ToCollect *big.Int, amount1ToCollect *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "burnAndCollect", recipient, tickLower, tickUpper, amountToBurn, amount0ToCollect, amount1ToCollect)
}

// BurnAndCollect is a paid mutator transaction binding the contract method 0x9c421449.
//
// Solidity: function burnAndCollect(address recipient, int24 tickLower, int24 tickUpper, uint128 amountToBurn, uint128 amount0ToCollect, uint128 amount1ToCollect) returns(uint256 amount0FromBurn, uint256 amount1FromBurn, uint128 amount0Collected, uint128 amount1Collected)
func (_Pool *PoolSession) BurnAndCollect(recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amountToBurn *big.Int, amount0ToCollect *big.Int, amount1ToCollect *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.BurnAndCollect(&_Pool.TransactOpts, recipient, tickLower, tickUpper, amountToBurn, amount0ToCollect, amount1ToCollect)
}

// BurnAndCollect is a paid mutator transaction binding the contract method 0x9c421449.
//
// Solidity: function burnAndCollect(address recipient, int24 tickLower, int24 tickUpper, uint128 amountToBurn, uint128 amount0ToCollect, uint128 amount1ToCollect) returns(uint256 amount0FromBurn, uint256 amount1FromBurn, uint128 amount0Collected, uint128 amount1Collected)
func (_Pool *PoolTransactorSession) BurnAndCollect(recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amountToBurn *big.Int, amount0ToCollect *big.Int, amount1ToCollect *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.BurnAndCollect(&_Pool.TransactOpts, recipient, tickLower, tickUpper, amountToBurn, amount0ToCollect, amount1ToCollect)
}

// BurnAndCollect0 is a paid mutator transaction binding the contract method 0xb6dc010d.
//
// Solidity: function burnAndCollect(address recipient, int24 tickLower, int24 tickUpper, uint128 amountToBurn, uint256 amount0FromBurnMin, uint256 amount1FromBurnMin, uint128 amount0ToCollect, uint128 amount1ToCollect, uint256 deadline) returns(uint256 amount0FromBurn, uint256 amount1FromBurn, uint128 amount0Collected, uint128 amount1Collected)
func (_Pool *PoolTransactor) BurnAndCollect0(opts *bind.TransactOpts, recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amountToBurn *big.Int, amount0FromBurnMin *big.Int, amount1FromBurnMin *big.Int, amount0ToCollect *big.Int, amount1ToCollect *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "burnAndCollect0", recipient, tickLower, tickUpper, amountToBurn, amount0FromBurnMin, amount1FromBurnMin, amount0ToCollect, amount1ToCollect, deadline)
}

// BurnAndCollect0 is a paid mutator transaction binding the contract method 0xb6dc010d.
//
// Solidity: function burnAndCollect(address recipient, int24 tickLower, int24 tickUpper, uint128 amountToBurn, uint256 amount0FromBurnMin, uint256 amount1FromBurnMin, uint128 amount0ToCollect, uint128 amount1ToCollect, uint256 deadline) returns(uint256 amount0FromBurn, uint256 amount1FromBurn, uint128 amount0Collected, uint128 amount1Collected)
func (_Pool *PoolSession) BurnAndCollect0(recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amountToBurn *big.Int, amount0FromBurnMin *big.Int, amount1FromBurnMin *big.Int, amount0ToCollect *big.Int, amount1ToCollect *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.BurnAndCollect0(&_Pool.TransactOpts, recipient, tickLower, tickUpper, amountToBurn, amount0FromBurnMin, amount1FromBurnMin, amount0ToCollect, amount1ToCollect, deadline)
}

// BurnAndCollect0 is a paid mutator transaction binding the contract method 0xb6dc010d.
//
// Solidity: function burnAndCollect(address recipient, int24 tickLower, int24 tickUpper, uint128 amountToBurn, uint256 amount0FromBurnMin, uint256 amount1FromBurnMin, uint128 amount0ToCollect, uint128 amount1ToCollect, uint256 deadline) returns(uint256 amount0FromBurn, uint256 amount1FromBurn, uint128 amount0Collected, uint128 amount1Collected)
func (_Pool *PoolTransactorSession) BurnAndCollect0(recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amountToBurn *big.Int, amount0FromBurnMin *big.Int, amount1FromBurnMin *big.Int, amount0ToCollect *big.Int, amount1ToCollect *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.BurnAndCollect0(&_Pool.TransactOpts, recipient, tickLower, tickUpper, amountToBurn, amount0FromBurnMin, amount1FromBurnMin, amount0ToCollect, amount1ToCollect, deadline)
}

// Collect is a paid mutator transaction binding the contract method 0x4f1eb3d8.
//
// Solidity: function collect(address recipient, int24 tickLower, int24 tickUpper, uint128 amount0Requested, uint128 amount1Requested) returns(uint128 amount0, uint128 amount1)
func (_Pool *PoolTransactor) Collect(opts *bind.TransactOpts, recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amount0Requested *big.Int, amount1Requested *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "collect", recipient, tickLower, tickUpper, amount0Requested, amount1Requested)
}

// Collect is a paid mutator transaction binding the contract method 0x4f1eb3d8.
//
// Solidity: function collect(address recipient, int24 tickLower, int24 tickUpper, uint128 amount0Requested, uint128 amount1Requested) returns(uint128 amount0, uint128 amount1)
func (_Pool *PoolSession) Collect(recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amount0Requested *big.Int, amount1Requested *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Collect(&_Pool.TransactOpts, recipient, tickLower, tickUpper, amount0Requested, amount1Requested)
}

// Collect is a paid mutator transaction binding the contract method 0x4f1eb3d8.
//
// Solidity: function collect(address recipient, int24 tickLower, int24 tickUpper, uint128 amount0Requested, uint128 amount1Requested) returns(uint128 amount0, uint128 amount1)
func (_Pool *PoolTransactorSession) Collect(recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amount0Requested *big.Int, amount1Requested *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Collect(&_Pool.TransactOpts, recipient, tickLower, tickUpper, amount0Requested, amount1Requested)
}

// CollectProtocol is a paid mutator transaction binding the contract method 0x85b66729.
//
// Solidity: function collectProtocol(address recipient, uint128 amount0Requested, uint128 amount1Requested) returns(uint128 amount0, uint128 amount1)
func (_Pool *PoolTransactor) CollectProtocol(opts *bind.TransactOpts, recipient common.Address, amount0Requested *big.Int, amount1Requested *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "collectProtocol", recipient, amount0Requested, amount1Requested)
}

// CollectProtocol is a paid mutator transaction binding the contract method 0x85b66729.
//
// Solidity: function collectProtocol(address recipient, uint128 amount0Requested, uint128 amount1Requested) returns(uint128 amount0, uint128 amount1)
func (_Pool *PoolSession) CollectProtocol(recipient common.Address, amount0Requested *big.Int, amount1Requested *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.CollectProtocol(&_Pool.TransactOpts, recipient, amount0Requested, amount1Requested)
}

// CollectProtocol is a paid mutator transaction binding the contract method 0x85b66729.
//
// Solidity: function collectProtocol(address recipient, uint128 amount0Requested, uint128 amount1Requested) returns(uint128 amount0, uint128 amount1)
func (_Pool *PoolTransactorSession) CollectProtocol(recipient common.Address, amount0Requested *big.Int, amount1Requested *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.CollectProtocol(&_Pool.TransactOpts, recipient, amount0Requested, amount1Requested)
}

// Flash is a paid mutator transaction binding the contract method 0x490e6cbc.
//
// Solidity: function flash(address recipient, uint256 amount0, uint256 amount1, bytes data) returns()
func (_Pool *PoolTransactor) Flash(opts *bind.TransactOpts, recipient common.Address, amount0 *big.Int, amount1 *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "flash", recipient, amount0, amount1, data)
}

// Flash is a paid mutator transaction binding the contract method 0x490e6cbc.
//
// Solidity: function flash(address recipient, uint256 amount0, uint256 amount1, bytes data) returns()
func (_Pool *PoolSession) Flash(recipient common.Address, amount0 *big.Int, amount1 *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.Flash(&_Pool.TransactOpts, recipient, amount0, amount1, data)
}

// Flash is a paid mutator transaction binding the contract method 0x490e6cbc.
//
// Solidity: function flash(address recipient, uint256 amount0, uint256 amount1, bytes data) returns()
func (_Pool *PoolTransactorSession) Flash(recipient common.Address, amount0 *big.Int, amount1 *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.Flash(&_Pool.TransactOpts, recipient, amount0, amount1, data)
}

// Initialize is a paid mutator transaction binding the contract method 0xf637731d.
//
// Solidity: function initialize(uint160 sqrtPriceX96) returns()
func (_Pool *PoolTransactor) Initialize(opts *bind.TransactOpts, sqrtPriceX96 *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "initialize", sqrtPriceX96)
}

// Initialize is a paid mutator transaction binding the contract method 0xf637731d.
//
// Solidity: function initialize(uint160 sqrtPriceX96) returns()
func (_Pool *PoolSession) Initialize(sqrtPriceX96 *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Initialize(&_Pool.TransactOpts, sqrtPriceX96)
}

// Initialize is a paid mutator transaction binding the contract method 0xf637731d.
//
// Solidity: function initialize(uint160 sqrtPriceX96) returns()
func (_Pool *PoolTransactorSession) Initialize(sqrtPriceX96 *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Initialize(&_Pool.TransactOpts, sqrtPriceX96)
}

// Mint is a paid mutator transaction binding the contract method 0x3c8a7d8d.
//
// Solidity: function mint(address recipient, int24 tickLower, int24 tickUpper, uint128 amount, bytes data) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolTransactor) Mint(opts *bind.TransactOpts, recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amount *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "mint", recipient, tickLower, tickUpper, amount, data)
}

// Mint is a paid mutator transaction binding the contract method 0x3c8a7d8d.
//
// Solidity: function mint(address recipient, int24 tickLower, int24 tickUpper, uint128 amount, bytes data) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolSession) Mint(recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amount *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.Mint(&_Pool.TransactOpts, recipient, tickLower, tickUpper, amount, data)
}

// Mint is a paid mutator transaction binding the contract method 0x3c8a7d8d.
//
// Solidity: function mint(address recipient, int24 tickLower, int24 tickUpper, uint128 amount, bytes data) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolTransactorSession) Mint(recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amount *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.Mint(&_Pool.TransactOpts, recipient, tickLower, tickUpper, amount, data)
}

// Mint0 is a paid mutator transaction binding the contract method 0x82cee55c.
//
// Solidity: function mint(address recipient, int24 tickLower, int24 tickUpper, uint128 amount, uint256 amount0Min, uint256 amount1Min, uint256 deadline) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolTransactor) Mint0(opts *bind.TransactOpts, recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amount *big.Int, amount0Min *big.Int, amount1Min *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "mint0", recipient, tickLower, tickUpper, amount, amount0Min, amount1Min, deadline)
}

// Mint0 is a paid mutator transaction binding the contract method 0x82cee55c.
//
// Solidity: function mint(address recipient, int24 tickLower, int24 tickUpper, uint128 amount, uint256 amount0Min, uint256 amount1Min, uint256 deadline) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolSession) Mint0(recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amount *big.Int, amount0Min *big.Int, amount1Min *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Mint0(&_Pool.TransactOpts, recipient, tickLower, tickUpper, amount, amount0Min, amount1Min, deadline)
}

// Mint0 is a paid mutator transaction binding the contract method 0x82cee55c.
//
// Solidity: function mint(address recipient, int24 tickLower, int24 tickUpper, uint128 amount, uint256 amount0Min, uint256 amount1Min, uint256 deadline) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolTransactorSession) Mint0(recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amount *big.Int, amount0Min *big.Int, amount1Min *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Mint0(&_Pool.TransactOpts, recipient, tickLower, tickUpper, amount, amount0Min, amount1Min, deadline)
}

// Mint1 is a paid mutator transaction binding the contract method 0x93ffcbb8.
//
// Solidity: function mint(address recipient, int24 tickLower, int24 tickUpper, uint128 amount, uint256 amount0Min, uint256 amount1Min, uint256 deadline, bytes data) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolTransactor) Mint1(opts *bind.TransactOpts, recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amount *big.Int, amount0Min *big.Int, amount1Min *big.Int, deadline *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "mint1", recipient, tickLower, tickUpper, amount, amount0Min, amount1Min, deadline, data)
}

// Mint1 is a paid mutator transaction binding the contract method 0x93ffcbb8.
//
// Solidity: function mint(address recipient, int24 tickLower, int24 tickUpper, uint128 amount, uint256 amount0Min, uint256 amount1Min, uint256 deadline, bytes data) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolSession) Mint1(recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amount *big.Int, amount0Min *big.Int, amount1Min *big.Int, deadline *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.Mint1(&_Pool.TransactOpts, recipient, tickLower, tickUpper, amount, amount0Min, amount1Min, deadline, data)
}

// Mint1 is a paid mutator transaction binding the contract method 0x93ffcbb8.
//
// Solidity: function mint(address recipient, int24 tickLower, int24 tickUpper, uint128 amount, uint256 amount0Min, uint256 amount1Min, uint256 deadline, bytes data) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolTransactorSession) Mint1(recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amount *big.Int, amount0Min *big.Int, amount1Min *big.Int, deadline *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.Mint1(&_Pool.TransactOpts, recipient, tickLower, tickUpper, amount, amount0Min, amount1Min, deadline, data)
}

// Mint2 is a paid mutator transaction binding the contract method 0xb8cca34e.
//
// Solidity: function mint(address recipient, int24 tickLower, int24 tickUpper, uint128 amount) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolTransactor) Mint2(opts *bind.TransactOpts, recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "mint2", recipient, tickLower, tickUpper, amount)
}

// Mint2 is a paid mutator transaction binding the contract method 0xb8cca34e.
//
// Solidity: function mint(address recipient, int24 tickLower, int24 tickUpper, uint128 amount) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolSession) Mint2(recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Mint2(&_Pool.TransactOpts, recipient, tickLower, tickUpper, amount)
}

// Mint2 is a paid mutator transaction binding the contract method 0xb8cca34e.
//
// Solidity: function mint(address recipient, int24 tickLower, int24 tickUpper, uint128 amount) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolTransactorSession) Mint2(recipient common.Address, tickLower *big.Int, tickUpper *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Mint2(&_Pool.TransactOpts, recipient, tickLower, tickUpper, amount)
}

// SetFee is a paid mutator transaction binding the contract method 0xeabb5622.
//
// Solidity: function setFee(uint24 fee) returns()
func (_Pool *PoolTransactor) SetFee(opts *bind.TransactOpts, fee *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "setFee", fee)
}

// SetFee is a paid mutator transaction binding the contract method 0xeabb5622.
//
// Solidity: function setFee(uint24 fee) returns()
func (_Pool *PoolSession) SetFee(fee *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.SetFee(&_Pool.TransactOpts, fee)
}

// SetFee is a paid mutator transaction binding the contract method 0xeabb5622.
//
// Solidity: function setFee(uint24 fee) returns()
func (_Pool *PoolTransactorSession) SetFee(fee *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.SetFee(&_Pool.TransactOpts, fee)
}

// Swap is a paid mutator transaction binding the contract method 0x0c033b9b.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, uint256 amountLimit, uint256 deadline, bytes data) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactor) Swap(opts *bind.TransactOpts, recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, amountLimit *big.Int, deadline *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "swap", recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, amountLimit, deadline, data)
}

// Swap is a paid mutator transaction binding the contract method 0x0c033b9b.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, uint256 amountLimit, uint256 deadline, bytes data) returns(int256 amount0, int256 amount1)
func (_Pool *PoolSession) Swap(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, amountLimit *big.Int, deadline *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.Swap(&_Pool.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, amountLimit, deadline, data)
}

// Swap is a paid mutator transaction binding the contract method 0x0c033b9b.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, uint256 amountLimit, uint256 deadline, bytes data) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactorSession) Swap(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, amountLimit *big.Int, deadline *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.Swap(&_Pool.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, amountLimit, deadline, data)
}

// Swap0 is a paid mutator transaction binding the contract method 0x128acb08.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, bytes data) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactor) Swap0(opts *bind.TransactOpts, recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "swap0", recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, data)
}

// Swap0 is a paid mutator transaction binding the contract method 0x128acb08.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, bytes data) returns(int256 amount0, int256 amount1)
func (_Pool *PoolSession) Swap0(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.Swap0(&_Pool.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, data)
}

// Swap0 is a paid mutator transaction binding the contract method 0x128acb08.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, bytes data) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactorSession) Swap0(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.Swap0(&_Pool.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, data)
}

// Swap1 is a paid mutator transaction binding the contract method 0x3f442864.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, uint256 amountLimit, uint256 deadline, bytes data, uint256 trackingCode) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactor) Swap1(opts *bind.TransactOpts, recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, amountLimit *big.Int, deadline *big.Int, data []byte, trackingCode *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "swap1", recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, amountLimit, deadline, data, trackingCode)
}

// Swap1 is a paid mutator transaction binding the contract method 0x3f442864.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, uint256 amountLimit, uint256 deadline, bytes data, uint256 trackingCode) returns(int256 amount0, int256 amount1)
func (_Pool *PoolSession) Swap1(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, amountLimit *big.Int, deadline *big.Int, data []byte, trackingCode *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Swap1(&_Pool.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, amountLimit, deadline, data, trackingCode)
}

// Swap1 is a paid mutator transaction binding the contract method 0x3f442864.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, uint256 amountLimit, uint256 deadline, bytes data, uint256 trackingCode) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactorSession) Swap1(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, amountLimit *big.Int, deadline *big.Int, data []byte, trackingCode *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Swap1(&_Pool.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, amountLimit, deadline, data, trackingCode)
}

// Swap2 is a paid mutator transaction binding the contract method 0x460985e8.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, uint256 trackingCode) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactor) Swap2(opts *bind.TransactOpts, recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, trackingCode *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "swap2", recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, trackingCode)
}

// Swap2 is a paid mutator transaction binding the contract method 0x460985e8.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, uint256 trackingCode) returns(int256 amount0, int256 amount1)
func (_Pool *PoolSession) Swap2(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, trackingCode *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Swap2(&_Pool.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, trackingCode)
}

// Swap2 is a paid mutator transaction binding the contract method 0x460985e8.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, uint256 trackingCode) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactorSession) Swap2(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, trackingCode *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Swap2(&_Pool.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, trackingCode)
}

// Swap3 is a paid mutator transaction binding the contract method 0x75d39ecb.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, uint256 amountLimit, uint256 deadline) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactor) Swap3(opts *bind.TransactOpts, recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, amountLimit *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "swap3", recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, amountLimit, deadline)
}

// Swap3 is a paid mutator transaction binding the contract method 0x75d39ecb.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, uint256 amountLimit, uint256 deadline) returns(int256 amount0, int256 amount1)
func (_Pool *PoolSession) Swap3(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, amountLimit *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Swap3(&_Pool.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, amountLimit, deadline)
}

// Swap3 is a paid mutator transaction binding the contract method 0x75d39ecb.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, uint256 amountLimit, uint256 deadline) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactorSession) Swap3(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, amountLimit *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Swap3(&_Pool.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, amountLimit, deadline)
}

// Swap4 is a paid mutator transaction binding the contract method 0xa23ba09c.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, uint256 amountLimit, uint256 deadline, uint256 trackingCode) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactor) Swap4(opts *bind.TransactOpts, recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, amountLimit *big.Int, deadline *big.Int, trackingCode *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "swap4", recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, amountLimit, deadline, trackingCode)
}

// Swap4 is a paid mutator transaction binding the contract method 0xa23ba09c.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, uint256 amountLimit, uint256 deadline, uint256 trackingCode) returns(int256 amount0, int256 amount1)
func (_Pool *PoolSession) Swap4(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, amountLimit *big.Int, deadline *big.Int, trackingCode *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Swap4(&_Pool.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, amountLimit, deadline, trackingCode)
}

// Swap4 is a paid mutator transaction binding the contract method 0xa23ba09c.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, uint256 amountLimit, uint256 deadline, uint256 trackingCode) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactorSession) Swap4(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, amountLimit *big.Int, deadline *big.Int, trackingCode *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Swap4(&_Pool.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, amountLimit, deadline, trackingCode)
}

// Swap5 is a paid mutator transaction binding the contract method 0xb0344304.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, bytes data, uint256 trackingCode) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactor) Swap5(opts *bind.TransactOpts, recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, data []byte, trackingCode *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "swap5", recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, data, trackingCode)
}

// Swap5 is a paid mutator transaction binding the contract method 0xb0344304.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, bytes data, uint256 trackingCode) returns(int256 amount0, int256 amount1)
func (_Pool *PoolSession) Swap5(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, data []byte, trackingCode *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Swap5(&_Pool.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, data, trackingCode)
}

// Swap5 is a paid mutator transaction binding the contract method 0xb0344304.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, bytes data, uint256 trackingCode) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactorSession) Swap5(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, data []byte, trackingCode *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Swap5(&_Pool.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, data, trackingCode)
}

// Swap6 is a paid mutator transaction binding the contract method 0xc31b8d7a.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactor) Swap6(opts *bind.TransactOpts, recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "swap6", recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96)
}

// Swap6 is a paid mutator transaction binding the contract method 0xc31b8d7a.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96) returns(int256 amount0, int256 amount1)
func (_Pool *PoolSession) Swap6(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Swap6(&_Pool.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96)
}

// Swap6 is a paid mutator transaction binding the contract method 0xc31b8d7a.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactorSession) Swap6(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Swap6(&_Pool.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96)
}

// PoolBurnIterator is returned from FilterBurn and is used to iterate over the raw logs and unpacked data for Burn events raised by the Pool contract.
type PoolBurnIterator struct {
	Event *PoolBurn // Event containing the contract specifics and raw log

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
func (it *PoolBurnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolBurn)
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
		it.Event = new(PoolBurn)
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
func (it *PoolBurnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolBurnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolBurn represents a Burn event raised by the Pool contract.
type PoolBurn struct {
	Owner     common.Address
	TickLower *big.Int
	TickUpper *big.Int
	Amount    *big.Int
	Amount0   *big.Int
	Amount1   *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterBurn is a free log retrieval operation binding the contract event 0x0c396cd989a39f4459b5fa1aed6a9a8dcdbc45908acfd67e028cd568da98982c.
//
// Solidity: event Burn(address indexed owner, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) FilterBurn(opts *bind.FilterOpts, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (*PoolBurnIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var tickLowerRule []interface{}
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []interface{}
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _Pool.contract.FilterLogs(opts, "Burn", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return &PoolBurnIterator{contract: _Pool.contract, event: "Burn", logs: logs, sub: sub}, nil
}

// WatchBurn is a free log subscription operation binding the contract event 0x0c396cd989a39f4459b5fa1aed6a9a8dcdbc45908acfd67e028cd568da98982c.
//
// Solidity: event Burn(address indexed owner, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) WatchBurn(opts *bind.WatchOpts, sink chan<- *PoolBurn, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var tickLowerRule []interface{}
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []interface{}
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _Pool.contract.WatchLogs(opts, "Burn", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolBurn)
				if err := _Pool.contract.UnpackLog(event, "Burn", log); err != nil {
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

// ParseBurn is a log parse operation binding the contract event 0x0c396cd989a39f4459b5fa1aed6a9a8dcdbc45908acfd67e028cd568da98982c.
//
// Solidity: event Burn(address indexed owner, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) ParseBurn(log types.Log) (*PoolBurn, error) {
	event := new(PoolBurn)
	if err := _Pool.contract.UnpackLog(event, "Burn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolCollectIterator is returned from FilterCollect and is used to iterate over the raw logs and unpacked data for Collect events raised by the Pool contract.
type PoolCollectIterator struct {
	Event *PoolCollect // Event containing the contract specifics and raw log

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
func (it *PoolCollectIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolCollect)
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
		it.Event = new(PoolCollect)
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
func (it *PoolCollectIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolCollectIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolCollect represents a Collect event raised by the Pool contract.
type PoolCollect struct {
	Owner     common.Address
	Recipient common.Address
	TickLower *big.Int
	TickUpper *big.Int
	Amount0   *big.Int
	Amount1   *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterCollect is a free log retrieval operation binding the contract event 0x70935338e69775456a85ddef226c395fb668b63fa0115f5f20610b388e6ca9c0.
//
// Solidity: event Collect(address indexed owner, address recipient, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount0, uint128 amount1)
func (_Pool *PoolFilterer) FilterCollect(opts *bind.FilterOpts, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (*PoolCollectIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	var tickLowerRule []interface{}
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []interface{}
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _Pool.contract.FilterLogs(opts, "Collect", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return &PoolCollectIterator{contract: _Pool.contract, event: "Collect", logs: logs, sub: sub}, nil
}

// WatchCollect is a free log subscription operation binding the contract event 0x70935338e69775456a85ddef226c395fb668b63fa0115f5f20610b388e6ca9c0.
//
// Solidity: event Collect(address indexed owner, address recipient, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount0, uint128 amount1)
func (_Pool *PoolFilterer) WatchCollect(opts *bind.WatchOpts, sink chan<- *PoolCollect, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	var tickLowerRule []interface{}
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []interface{}
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _Pool.contract.WatchLogs(opts, "Collect", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolCollect)
				if err := _Pool.contract.UnpackLog(event, "Collect", log); err != nil {
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

// ParseCollect is a log parse operation binding the contract event 0x70935338e69775456a85ddef226c395fb668b63fa0115f5f20610b388e6ca9c0.
//
// Solidity: event Collect(address indexed owner, address recipient, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount0, uint128 amount1)
func (_Pool *PoolFilterer) ParseCollect(log types.Log) (*PoolCollect, error) {
	event := new(PoolCollect)
	if err := _Pool.contract.UnpackLog(event, "Collect", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolCollectProtocolIterator is returned from FilterCollectProtocol and is used to iterate over the raw logs and unpacked data for CollectProtocol events raised by the Pool contract.
type PoolCollectProtocolIterator struct {
	Event *PoolCollectProtocol // Event containing the contract specifics and raw log

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
func (it *PoolCollectProtocolIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolCollectProtocol)
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
		it.Event = new(PoolCollectProtocol)
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
func (it *PoolCollectProtocolIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolCollectProtocolIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolCollectProtocol represents a CollectProtocol event raised by the Pool contract.
type PoolCollectProtocol struct {
	Sender    common.Address
	Recipient common.Address
	Amount0   *big.Int
	Amount1   *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterCollectProtocol is a free log retrieval operation binding the contract event 0x596b573906218d3411850b26a6b437d6c4522fdb43d2d2386263f86d50b8b151.
//
// Solidity: event CollectProtocol(address indexed sender, address indexed recipient, uint128 amount0, uint128 amount1)
func (_Pool *PoolFilterer) FilterCollectProtocol(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address) (*PoolCollectProtocolIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Pool.contract.FilterLogs(opts, "CollectProtocol", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &PoolCollectProtocolIterator{contract: _Pool.contract, event: "CollectProtocol", logs: logs, sub: sub}, nil
}

// WatchCollectProtocol is a free log subscription operation binding the contract event 0x596b573906218d3411850b26a6b437d6c4522fdb43d2d2386263f86d50b8b151.
//
// Solidity: event CollectProtocol(address indexed sender, address indexed recipient, uint128 amount0, uint128 amount1)
func (_Pool *PoolFilterer) WatchCollectProtocol(opts *bind.WatchOpts, sink chan<- *PoolCollectProtocol, sender []common.Address, recipient []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Pool.contract.WatchLogs(opts, "CollectProtocol", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolCollectProtocol)
				if err := _Pool.contract.UnpackLog(event, "CollectProtocol", log); err != nil {
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

// ParseCollectProtocol is a log parse operation binding the contract event 0x596b573906218d3411850b26a6b437d6c4522fdb43d2d2386263f86d50b8b151.
//
// Solidity: event CollectProtocol(address indexed sender, address indexed recipient, uint128 amount0, uint128 amount1)
func (_Pool *PoolFilterer) ParseCollectProtocol(log types.Log) (*PoolCollectProtocol, error) {
	event := new(PoolCollectProtocol)
	if err := _Pool.contract.UnpackLog(event, "CollectProtocol", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolFlashIterator is returned from FilterFlash and is used to iterate over the raw logs and unpacked data for Flash events raised by the Pool contract.
type PoolFlashIterator struct {
	Event *PoolFlash // Event containing the contract specifics and raw log

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
func (it *PoolFlashIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolFlash)
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
		it.Event = new(PoolFlash)
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
func (it *PoolFlashIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolFlashIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolFlash represents a Flash event raised by the Pool contract.
type PoolFlash struct {
	Sender    common.Address
	Recipient common.Address
	Amount0   *big.Int
	Amount1   *big.Int
	Paid0     *big.Int
	Paid1     *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterFlash is a free log retrieval operation binding the contract event 0xbdbdb71d7860376ba52b25a5028beea23581364a40522f6bcfb86bb1f2dca633.
//
// Solidity: event Flash(address indexed sender, address indexed recipient, uint256 amount0, uint256 amount1, uint256 paid0, uint256 paid1)
func (_Pool *PoolFilterer) FilterFlash(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address) (*PoolFlashIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Pool.contract.FilterLogs(opts, "Flash", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &PoolFlashIterator{contract: _Pool.contract, event: "Flash", logs: logs, sub: sub}, nil
}

// WatchFlash is a free log subscription operation binding the contract event 0xbdbdb71d7860376ba52b25a5028beea23581364a40522f6bcfb86bb1f2dca633.
//
// Solidity: event Flash(address indexed sender, address indexed recipient, uint256 amount0, uint256 amount1, uint256 paid0, uint256 paid1)
func (_Pool *PoolFilterer) WatchFlash(opts *bind.WatchOpts, sink chan<- *PoolFlash, sender []common.Address, recipient []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Pool.contract.WatchLogs(opts, "Flash", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolFlash)
				if err := _Pool.contract.UnpackLog(event, "Flash", log); err != nil {
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

// ParseFlash is a log parse operation binding the contract event 0xbdbdb71d7860376ba52b25a5028beea23581364a40522f6bcfb86bb1f2dca633.
//
// Solidity: event Flash(address indexed sender, address indexed recipient, uint256 amount0, uint256 amount1, uint256 paid0, uint256 paid1)
func (_Pool *PoolFilterer) ParseFlash(log types.Log) (*PoolFlash, error) {
	event := new(PoolFlash)
	if err := _Pool.contract.UnpackLog(event, "Flash", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolInitializeIterator is returned from FilterInitialize and is used to iterate over the raw logs and unpacked data for Initialize events raised by the Pool contract.
type PoolInitializeIterator struct {
	Event *PoolInitialize // Event containing the contract specifics and raw log

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
func (it *PoolInitializeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolInitialize)
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
		it.Event = new(PoolInitialize)
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
func (it *PoolInitializeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolInitializeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolInitialize represents a Initialize event raised by the Pool contract.
type PoolInitialize struct {
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterInitialize is a free log retrieval operation binding the contract event 0x98636036cb66a9c19a37435efc1e90142190214e8abeb821bdba3f2990dd4c95.
//
// Solidity: event Initialize(uint160 sqrtPriceX96, int24 tick)
func (_Pool *PoolFilterer) FilterInitialize(opts *bind.FilterOpts) (*PoolInitializeIterator, error) {

	logs, sub, err := _Pool.contract.FilterLogs(opts, "Initialize")
	if err != nil {
		return nil, err
	}
	return &PoolInitializeIterator{contract: _Pool.contract, event: "Initialize", logs: logs, sub: sub}, nil
}

// WatchInitialize is a free log subscription operation binding the contract event 0x98636036cb66a9c19a37435efc1e90142190214e8abeb821bdba3f2990dd4c95.
//
// Solidity: event Initialize(uint160 sqrtPriceX96, int24 tick)
func (_Pool *PoolFilterer) WatchInitialize(opts *bind.WatchOpts, sink chan<- *PoolInitialize) (event.Subscription, error) {

	logs, sub, err := _Pool.contract.WatchLogs(opts, "Initialize")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolInitialize)
				if err := _Pool.contract.UnpackLog(event, "Initialize", log); err != nil {
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

// ParseInitialize is a log parse operation binding the contract event 0x98636036cb66a9c19a37435efc1e90142190214e8abeb821bdba3f2990dd4c95.
//
// Solidity: event Initialize(uint160 sqrtPriceX96, int24 tick)
func (_Pool *PoolFilterer) ParseInitialize(log types.Log) (*PoolInitialize, error) {
	event := new(PoolInitialize)
	if err := _Pool.contract.UnpackLog(event, "Initialize", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolMintIterator is returned from FilterMint and is used to iterate over the raw logs and unpacked data for Mint events raised by the Pool contract.
type PoolMintIterator struct {
	Event *PoolMint // Event containing the contract specifics and raw log

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
func (it *PoolMintIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolMint)
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
		it.Event = new(PoolMint)
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
func (it *PoolMintIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolMintIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolMint represents a Mint event raised by the Pool contract.
type PoolMint struct {
	Sender    common.Address
	Owner     common.Address
	TickLower *big.Int
	TickUpper *big.Int
	Amount    *big.Int
	Amount0   *big.Int
	Amount1   *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterMint is a free log retrieval operation binding the contract event 0x7a53080ba414158be7ec69b987b5fb7d07dee101fe85488f0853ae16239d0bde.
//
// Solidity: event Mint(address sender, address indexed owner, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) FilterMint(opts *bind.FilterOpts, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (*PoolMintIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var tickLowerRule []interface{}
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []interface{}
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _Pool.contract.FilterLogs(opts, "Mint", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return &PoolMintIterator{contract: _Pool.contract, event: "Mint", logs: logs, sub: sub}, nil
}

// WatchMint is a free log subscription operation binding the contract event 0x7a53080ba414158be7ec69b987b5fb7d07dee101fe85488f0853ae16239d0bde.
//
// Solidity: event Mint(address sender, address indexed owner, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) WatchMint(opts *bind.WatchOpts, sink chan<- *PoolMint, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var tickLowerRule []interface{}
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []interface{}
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _Pool.contract.WatchLogs(opts, "Mint", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolMint)
				if err := _Pool.contract.UnpackLog(event, "Mint", log); err != nil {
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

// ParseMint is a log parse operation binding the contract event 0x7a53080ba414158be7ec69b987b5fb7d07dee101fe85488f0853ae16239d0bde.
//
// Solidity: event Mint(address sender, address indexed owner, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) ParseMint(log types.Log) (*PoolMint, error) {
	event := new(PoolMint)
	if err := _Pool.contract.UnpackLog(event, "Mint", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolSetFeeIterator is returned from FilterSetFee and is used to iterate over the raw logs and unpacked data for SetFee events raised by the Pool contract.
type PoolSetFeeIterator struct {
	Event *PoolSetFee // Event containing the contract specifics and raw log

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
func (it *PoolSetFeeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolSetFee)
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
		it.Event = new(PoolSetFee)
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
func (it *PoolSetFeeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolSetFeeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolSetFee represents a SetFee event raised by the Pool contract.
type PoolSetFee struct {
	FeeOld *big.Int
	FeeNew *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterSetFee is a free log retrieval operation binding the contract event 0x0eb63f4a36d6bdeee05aa00020a97d80c3e84f1b5b3ebf345fb67262e62b0f33.
//
// Solidity: event SetFee(uint24 feeOld, uint24 feeNew)
func (_Pool *PoolFilterer) FilterSetFee(opts *bind.FilterOpts) (*PoolSetFeeIterator, error) {

	logs, sub, err := _Pool.contract.FilterLogs(opts, "SetFee")
	if err != nil {
		return nil, err
	}
	return &PoolSetFeeIterator{contract: _Pool.contract, event: "SetFee", logs: logs, sub: sub}, nil
}

// WatchSetFee is a free log subscription operation binding the contract event 0x0eb63f4a36d6bdeee05aa00020a97d80c3e84f1b5b3ebf345fb67262e62b0f33.
//
// Solidity: event SetFee(uint24 feeOld, uint24 feeNew)
func (_Pool *PoolFilterer) WatchSetFee(opts *bind.WatchOpts, sink chan<- *PoolSetFee) (event.Subscription, error) {

	logs, sub, err := _Pool.contract.WatchLogs(opts, "SetFee")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolSetFee)
				if err := _Pool.contract.UnpackLog(event, "SetFee", log); err != nil {
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

// ParseSetFee is a log parse operation binding the contract event 0x0eb63f4a36d6bdeee05aa00020a97d80c3e84f1b5b3ebf345fb67262e62b0f33.
//
// Solidity: event SetFee(uint24 feeOld, uint24 feeNew)
func (_Pool *PoolFilterer) ParseSetFee(log types.Log) (*PoolSetFee, error) {
	event := new(PoolSetFee)
	if err := _Pool.contract.UnpackLog(event, "SetFee", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolSwapIterator is returned from FilterSwap and is used to iterate over the raw logs and unpacked data for Swap events raised by the Pool contract.
type PoolSwapIterator struct {
	Event *PoolSwap // Event containing the contract specifics and raw log

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
func (it *PoolSwapIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolSwap)
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
		it.Event = new(PoolSwap)
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
func (it *PoolSwapIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolSwapIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolSwap represents a Swap event raised by the Pool contract.
type PoolSwap struct {
	Sender       common.Address
	Recipient    common.Address
	Amount0      *big.Int
	Amount1      *big.Int
	SqrtPriceX96 *big.Int
	Liquidity    *big.Int
	Tick         *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterSwap is a free log retrieval operation binding the contract event 0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67.
//
// Solidity: event Swap(address indexed sender, address indexed recipient, int256 amount0, int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick)
func (_Pool *PoolFilterer) FilterSwap(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address) (*PoolSwapIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Pool.contract.FilterLogs(opts, "Swap", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &PoolSwapIterator{contract: _Pool.contract, event: "Swap", logs: logs, sub: sub}, nil
}

// WatchSwap is a free log subscription operation binding the contract event 0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67.
//
// Solidity: event Swap(address indexed sender, address indexed recipient, int256 amount0, int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick)
func (_Pool *PoolFilterer) WatchSwap(opts *bind.WatchOpts, sink chan<- *PoolSwap, sender []common.Address, recipient []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Pool.contract.WatchLogs(opts, "Swap", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolSwap)
				if err := _Pool.contract.UnpackLog(event, "Swap", log); err != nil {
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

// ParseSwap is a log parse operation binding the contract event 0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67.
//
// Solidity: event Swap(address indexed sender, address indexed recipient, int256 amount0, int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick)
func (_Pool *PoolFilterer) ParseSwap(log types.Log) (*PoolSwap, error) {
	event := new(PoolSwap)
	if err := _Pool.contract.UnpackLog(event, "Swap", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
