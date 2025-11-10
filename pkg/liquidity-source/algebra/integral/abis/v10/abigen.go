// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package intergralpoolv10

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
	ABI: "[{\"inputs\":[],\"name\":\"alreadyInitialized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"arithmeticError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"bottomTickLowerThanMIN\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"dynamicFeeActive\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"dynamicFeeDisabled\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"flashInsufficientPaid0\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"flashInsufficientPaid1\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"insufficientInputAmount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"invalidAmountRequired\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"expectedSelector\",\"type\":\"bytes4\"}],\"name\":\"invalidHookResponse\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"invalidLimitSqrtPrice\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"invalidNewCommunityFee\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"invalidNewTickSpacing\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"liquidityAdd\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"liquidityOverflow\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"liquiditySub\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"locked\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"notAllowed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"notInitialized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"pluginIsNotConnected\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"priceOutOfRange\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"tickInvalidLinks\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"tickIsNotInitialized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"tickIsNotSpaced\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"tickOutOfRange\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"topTickAboveMAX\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"topTickLowerOrEqBottomTick\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"transferFailed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"zeroAmountRequired\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"zeroLiquidityActual\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"zeroLiquidityDesired\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"bottomTick\",\"type\":\"int24\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"topTick\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"liquidityAmount\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"name\":\"Burn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"bottomTick\",\"type\":\"int24\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"topTick\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount0\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount1\",\"type\":\"uint128\"}],\"name\":\"Collect\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"communityFeeNew\",\"type\":\"uint16\"}],\"name\":\"CommunityFee\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newCommunityVault\",\"type\":\"address\"}],\"name\":\"CommunityVault\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"name\":\"ExcessTokens\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"fee\",\"type\":\"uint16\"}],\"name\":\"Fee\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"paid0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"paid1\",\"type\":\"uint256\"}],\"name\":\"Flash\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint160\",\"name\":\"price\",\"type\":\"uint160\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"Initialize\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"bottomTick\",\"type\":\"int24\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"topTick\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"liquidityAmount\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"name\":\"Mint\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newPluginAddress\",\"type\":\"address\"}],\"name\":\"Plugin\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"newPluginConfig\",\"type\":\"uint8\"}],\"name\":\"PluginConfig\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"name\":\"Skim\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"amount0\",\"type\":\"int256\"},{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"amount1\",\"type\":\"int256\"},{\"indexed\":false,\"internalType\":\"uint160\",\"name\":\"price\",\"type\":\"uint160\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"liquidity\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"Swap\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"newTickSpacing\",\"type\":\"int24\"}],\"name\":\"TickSpacing\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"int24\",\"name\":\"bottomTick\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"topTick\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"burn\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"bottomTick\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"topTick\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amount0Requested\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1Requested\",\"type\":\"uint128\"}],\"name\":\"collect\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"amount0\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1\",\"type\":\"uint128\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"communityFeeLastTimestamp\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"communityVault\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"factory\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"fee\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"currentFee\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"flash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCommunityFeePending\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getReserves\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"globalState\",\"outputs\":[{\"internalType\":\"uint160\",\"name\":\"price\",\"type\":\"uint160\"},{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"},{\"internalType\":\"uint16\",\"name\":\"lastFee\",\"type\":\"uint16\"},{\"internalType\":\"uint8\",\"name\":\"pluginConfig\",\"type\":\"uint8\"},{\"internalType\":\"uint16\",\"name\":\"communityFee\",\"type\":\"uint16\"},{\"internalType\":\"bool\",\"name\":\"unlocked\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint160\",\"name\":\"initialPrice\",\"type\":\"uint160\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isUnlocked\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"unlocked\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"liquidity\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxLiquidityPerTick\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"leftoversRecipient\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"bottomTick\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"topTick\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"liquidityDesired\",\"type\":\"uint128\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"mint\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"internalType\":\"uint128\",\"name\":\"liquidityActual\",\"type\":\"uint128\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nextTickGlobal\",\"outputs\":[{\"internalType\":\"int24\",\"name\":\"\",\"type\":\"int24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"plugin\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"positions\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"liquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"innerFeeGrowth0Token\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"innerFeeGrowth1Token\",\"type\":\"uint256\"},{\"internalType\":\"uint128\",\"name\":\"fees0\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"fees1\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"prevTickGlobal\",\"outputs\":[{\"internalType\":\"int24\",\"name\":\"\",\"type\":\"int24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"safelyGetStateOfAMM\",\"outputs\":[{\"internalType\":\"uint160\",\"name\":\"sqrtPrice\",\"type\":\"uint160\"},{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"},{\"internalType\":\"uint16\",\"name\":\"lastFee\",\"type\":\"uint16\"},{\"internalType\":\"uint8\",\"name\":\"pluginConfig\",\"type\":\"uint8\"},{\"internalType\":\"uint128\",\"name\":\"activeLiquidity\",\"type\":\"uint128\"},{\"internalType\":\"int24\",\"name\":\"nextTick\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"previousTick\",\"type\":\"int24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"newCommunityFee\",\"type\":\"uint16\"}],\"name\":\"setCommunityFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newCommunityVault\",\"type\":\"address\"}],\"name\":\"setCommunityVault\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"newFee\",\"type\":\"uint16\"}],\"name\":\"setFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newPluginAddress\",\"type\":\"address\"}],\"name\":\"setPlugin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"newConfig\",\"type\":\"uint8\"}],\"name\":\"setPluginConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int24\",\"name\":\"newTickSpacing\",\"type\":\"int24\"}],\"name\":\"setTickSpacing\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"skim\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"zeroToOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountRequired\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"limitSqrtPrice\",\"type\":\"uint160\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"swap\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"amount0\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"amount1\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"leftoversRecipient\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"zeroToOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountToSell\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"limitSqrtPrice\",\"type\":\"uint160\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"swapWithPaymentInAdvance\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"amount0\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"amount1\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sync\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"tickSpacing\",\"outputs\":[{\"internalType\":\"int24\",\"name\":\"\",\"type\":\"int24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int16\",\"name\":\"\",\"type\":\"int16\"}],\"name\":\"tickTable\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"tickTreeRoot\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int16\",\"name\":\"\",\"type\":\"int16\"}],\"name\":\"tickTreeSecondLayer\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int24\",\"name\":\"\",\"type\":\"int24\"}],\"name\":\"ticks\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"liquidityTotal\",\"type\":\"uint256\"},{\"internalType\":\"int128\",\"name\":\"liquidityDelta\",\"type\":\"int128\"},{\"internalType\":\"int24\",\"name\":\"prevTick\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"nextTick\",\"type\":\"int24\"},{\"internalType\":\"uint256\",\"name\":\"outerFeeGrowth0Token\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"outerFeeGrowth1Token\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"token0\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"token1\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalFeeGrowth0Token\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalFeeGrowth1Token\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
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

// CommunityFeeLastTimestamp is a free data retrieval call binding the contract method 0x1131b110.
//
// Solidity: function communityFeeLastTimestamp() view returns(uint32)
func (_Pool *PoolCaller) CommunityFeeLastTimestamp(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "communityFeeLastTimestamp")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// CommunityFeeLastTimestamp is a free data retrieval call binding the contract method 0x1131b110.
//
// Solidity: function communityFeeLastTimestamp() view returns(uint32)
func (_Pool *PoolSession) CommunityFeeLastTimestamp() (uint32, error) {
	return _Pool.Contract.CommunityFeeLastTimestamp(&_Pool.CallOpts)
}

// CommunityFeeLastTimestamp is a free data retrieval call binding the contract method 0x1131b110.
//
// Solidity: function communityFeeLastTimestamp() view returns(uint32)
func (_Pool *PoolCallerSession) CommunityFeeLastTimestamp() (uint32, error) {
	return _Pool.Contract.CommunityFeeLastTimestamp(&_Pool.CallOpts)
}

// CommunityVault is a free data retrieval call binding the contract method 0x53e97868.
//
// Solidity: function communityVault() view returns(address)
func (_Pool *PoolCaller) CommunityVault(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "communityVault")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CommunityVault is a free data retrieval call binding the contract method 0x53e97868.
//
// Solidity: function communityVault() view returns(address)
func (_Pool *PoolSession) CommunityVault() (common.Address, error) {
	return _Pool.Contract.CommunityVault(&_Pool.CallOpts)
}

// CommunityVault is a free data retrieval call binding the contract method 0x53e97868.
//
// Solidity: function communityVault() view returns(address)
func (_Pool *PoolCallerSession) CommunityVault() (common.Address, error) {
	return _Pool.Contract.CommunityVault(&_Pool.CallOpts)
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

// Fee is a free data retrieval call binding the contract method 0xddca3f43.
//
// Solidity: function fee() view returns(uint16 currentFee)
func (_Pool *PoolCaller) Fee(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "fee")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// Fee is a free data retrieval call binding the contract method 0xddca3f43.
//
// Solidity: function fee() view returns(uint16 currentFee)
func (_Pool *PoolSession) Fee() (uint16, error) {
	return _Pool.Contract.Fee(&_Pool.CallOpts)
}

// Fee is a free data retrieval call binding the contract method 0xddca3f43.
//
// Solidity: function fee() view returns(uint16 currentFee)
func (_Pool *PoolCallerSession) Fee() (uint16, error) {
	return _Pool.Contract.Fee(&_Pool.CallOpts)
}

// GetCommunityFeePending is a free data retrieval call binding the contract method 0x7bd78025.
//
// Solidity: function getCommunityFeePending() view returns(uint128, uint128)
func (_Pool *PoolCaller) GetCommunityFeePending(opts *bind.CallOpts) (*big.Int, *big.Int, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "getCommunityFeePending")

	if err != nil {
		return *new(*big.Int), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return out0, out1, err

}

// GetCommunityFeePending is a free data retrieval call binding the contract method 0x7bd78025.
//
// Solidity: function getCommunityFeePending() view returns(uint128, uint128)
func (_Pool *PoolSession) GetCommunityFeePending() (*big.Int, *big.Int, error) {
	return _Pool.Contract.GetCommunityFeePending(&_Pool.CallOpts)
}

// GetCommunityFeePending is a free data retrieval call binding the contract method 0x7bd78025.
//
// Solidity: function getCommunityFeePending() view returns(uint128, uint128)
func (_Pool *PoolCallerSession) GetCommunityFeePending() (*big.Int, *big.Int, error) {
	return _Pool.Contract.GetCommunityFeePending(&_Pool.CallOpts)
}

// GetReserves is a free data retrieval call binding the contract method 0x0902f1ac.
//
// Solidity: function getReserves() view returns(uint128, uint128)
func (_Pool *PoolCaller) GetReserves(opts *bind.CallOpts) (*big.Int, *big.Int, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "getReserves")

	if err != nil {
		return *new(*big.Int), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return out0, out1, err

}

// GetReserves is a free data retrieval call binding the contract method 0x0902f1ac.
//
// Solidity: function getReserves() view returns(uint128, uint128)
func (_Pool *PoolSession) GetReserves() (*big.Int, *big.Int, error) {
	return _Pool.Contract.GetReserves(&_Pool.CallOpts)
}

// GetReserves is a free data retrieval call binding the contract method 0x0902f1ac.
//
// Solidity: function getReserves() view returns(uint128, uint128)
func (_Pool *PoolCallerSession) GetReserves() (*big.Int, *big.Int, error) {
	return _Pool.Contract.GetReserves(&_Pool.CallOpts)
}

// GlobalState is a free data retrieval call binding the contract method 0xe76c01e4.
//
// Solidity: function globalState() view returns(uint160 price, int24 tick, uint16 lastFee, uint8 pluginConfig, uint16 communityFee, bool unlocked)
func (_Pool *PoolCaller) GlobalState(opts *bind.CallOpts) (struct {
	Price        *big.Int
	Tick         *big.Int
	LastFee      uint16
	PluginConfig uint8
	CommunityFee uint16
	Unlocked     bool
}, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "globalState")

	outstruct := new(struct {
		Price        *big.Int
		Tick         *big.Int
		LastFee      uint16
		PluginConfig uint8
		CommunityFee uint16
		Unlocked     bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Price = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Tick = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.LastFee = *abi.ConvertType(out[2], new(uint16)).(*uint16)
	outstruct.PluginConfig = *abi.ConvertType(out[3], new(uint8)).(*uint8)
	outstruct.CommunityFee = *abi.ConvertType(out[4], new(uint16)).(*uint16)
	outstruct.Unlocked = *abi.ConvertType(out[5], new(bool)).(*bool)

	return *outstruct, err

}

// GlobalState is a free data retrieval call binding the contract method 0xe76c01e4.
//
// Solidity: function globalState() view returns(uint160 price, int24 tick, uint16 lastFee, uint8 pluginConfig, uint16 communityFee, bool unlocked)
func (_Pool *PoolSession) GlobalState() (struct {
	Price        *big.Int
	Tick         *big.Int
	LastFee      uint16
	PluginConfig uint8
	CommunityFee uint16
	Unlocked     bool
}, error) {
	return _Pool.Contract.GlobalState(&_Pool.CallOpts)
}

// GlobalState is a free data retrieval call binding the contract method 0xe76c01e4.
//
// Solidity: function globalState() view returns(uint160 price, int24 tick, uint16 lastFee, uint8 pluginConfig, uint16 communityFee, bool unlocked)
func (_Pool *PoolCallerSession) GlobalState() (struct {
	Price        *big.Int
	Tick         *big.Int
	LastFee      uint16
	PluginConfig uint8
	CommunityFee uint16
	Unlocked     bool
}, error) {
	return _Pool.Contract.GlobalState(&_Pool.CallOpts)
}

// IsUnlocked is a free data retrieval call binding the contract method 0x8380edb7.
//
// Solidity: function isUnlocked() view returns(bool unlocked)
func (_Pool *PoolCaller) IsUnlocked(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "isUnlocked")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsUnlocked is a free data retrieval call binding the contract method 0x8380edb7.
//
// Solidity: function isUnlocked() view returns(bool unlocked)
func (_Pool *PoolSession) IsUnlocked() (bool, error) {
	return _Pool.Contract.IsUnlocked(&_Pool.CallOpts)
}

// IsUnlocked is a free data retrieval call binding the contract method 0x8380edb7.
//
// Solidity: function isUnlocked() view returns(bool unlocked)
func (_Pool *PoolCallerSession) IsUnlocked() (bool, error) {
	return _Pool.Contract.IsUnlocked(&_Pool.CallOpts)
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

// NextTickGlobal is a free data retrieval call binding the contract method 0xd5c35a7e.
//
// Solidity: function nextTickGlobal() view returns(int24)
func (_Pool *PoolCaller) NextTickGlobal(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "nextTickGlobal")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NextTickGlobal is a free data retrieval call binding the contract method 0xd5c35a7e.
//
// Solidity: function nextTickGlobal() view returns(int24)
func (_Pool *PoolSession) NextTickGlobal() (*big.Int, error) {
	return _Pool.Contract.NextTickGlobal(&_Pool.CallOpts)
}

// NextTickGlobal is a free data retrieval call binding the contract method 0xd5c35a7e.
//
// Solidity: function nextTickGlobal() view returns(int24)
func (_Pool *PoolCallerSession) NextTickGlobal() (*big.Int, error) {
	return _Pool.Contract.NextTickGlobal(&_Pool.CallOpts)
}

// Plugin is a free data retrieval call binding the contract method 0xef01df4f.
//
// Solidity: function plugin() view returns(address)
func (_Pool *PoolCaller) Plugin(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "plugin")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Plugin is a free data retrieval call binding the contract method 0xef01df4f.
//
// Solidity: function plugin() view returns(address)
func (_Pool *PoolSession) Plugin() (common.Address, error) {
	return _Pool.Contract.Plugin(&_Pool.CallOpts)
}

// Plugin is a free data retrieval call binding the contract method 0xef01df4f.
//
// Solidity: function plugin() view returns(address)
func (_Pool *PoolCallerSession) Plugin() (common.Address, error) {
	return _Pool.Contract.Plugin(&_Pool.CallOpts)
}

// Positions is a free data retrieval call binding the contract method 0x514ea4bf.
//
// Solidity: function positions(bytes32 ) view returns(uint256 liquidity, uint256 innerFeeGrowth0Token, uint256 innerFeeGrowth1Token, uint128 fees0, uint128 fees1)
func (_Pool *PoolCaller) Positions(opts *bind.CallOpts, arg0 [32]byte) (struct {
	Liquidity            *big.Int
	InnerFeeGrowth0Token *big.Int
	InnerFeeGrowth1Token *big.Int
	Fees0                *big.Int
	Fees1                *big.Int
}, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "positions", arg0)

	outstruct := new(struct {
		Liquidity            *big.Int
		InnerFeeGrowth0Token *big.Int
		InnerFeeGrowth1Token *big.Int
		Fees0                *big.Int
		Fees1                *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Liquidity = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.InnerFeeGrowth0Token = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.InnerFeeGrowth1Token = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.Fees0 = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.Fees1 = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Positions is a free data retrieval call binding the contract method 0x514ea4bf.
//
// Solidity: function positions(bytes32 ) view returns(uint256 liquidity, uint256 innerFeeGrowth0Token, uint256 innerFeeGrowth1Token, uint128 fees0, uint128 fees1)
func (_Pool *PoolSession) Positions(arg0 [32]byte) (struct {
	Liquidity            *big.Int
	InnerFeeGrowth0Token *big.Int
	InnerFeeGrowth1Token *big.Int
	Fees0                *big.Int
	Fees1                *big.Int
}, error) {
	return _Pool.Contract.Positions(&_Pool.CallOpts, arg0)
}

// Positions is a free data retrieval call binding the contract method 0x514ea4bf.
//
// Solidity: function positions(bytes32 ) view returns(uint256 liquidity, uint256 innerFeeGrowth0Token, uint256 innerFeeGrowth1Token, uint128 fees0, uint128 fees1)
func (_Pool *PoolCallerSession) Positions(arg0 [32]byte) (struct {
	Liquidity            *big.Int
	InnerFeeGrowth0Token *big.Int
	InnerFeeGrowth1Token *big.Int
	Fees0                *big.Int
	Fees1                *big.Int
}, error) {
	return _Pool.Contract.Positions(&_Pool.CallOpts, arg0)
}

// PrevTickGlobal is a free data retrieval call binding the contract method 0x050a4d21.
//
// Solidity: function prevTickGlobal() view returns(int24)
func (_Pool *PoolCaller) PrevTickGlobal(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "prevTickGlobal")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PrevTickGlobal is a free data retrieval call binding the contract method 0x050a4d21.
//
// Solidity: function prevTickGlobal() view returns(int24)
func (_Pool *PoolSession) PrevTickGlobal() (*big.Int, error) {
	return _Pool.Contract.PrevTickGlobal(&_Pool.CallOpts)
}

// PrevTickGlobal is a free data retrieval call binding the contract method 0x050a4d21.
//
// Solidity: function prevTickGlobal() view returns(int24)
func (_Pool *PoolCallerSession) PrevTickGlobal() (*big.Int, error) {
	return _Pool.Contract.PrevTickGlobal(&_Pool.CallOpts)
}

// SafelyGetStateOfAMM is a free data retrieval call binding the contract method 0x97ce1c51.
//
// Solidity: function safelyGetStateOfAMM() view returns(uint160 sqrtPrice, int24 tick, uint16 lastFee, uint8 pluginConfig, uint128 activeLiquidity, int24 nextTick, int24 previousTick)
func (_Pool *PoolCaller) SafelyGetStateOfAMM(opts *bind.CallOpts) (struct {
	SqrtPrice       *big.Int
	Tick            *big.Int
	LastFee         uint16
	PluginConfig    uint8
	ActiveLiquidity *big.Int
	NextTick        *big.Int
	PreviousTick    *big.Int
}, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "safelyGetStateOfAMM")

	outstruct := new(struct {
		SqrtPrice       *big.Int
		Tick            *big.Int
		LastFee         uint16
		PluginConfig    uint8
		ActiveLiquidity *big.Int
		NextTick        *big.Int
		PreviousTick    *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.SqrtPrice = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Tick = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.LastFee = *abi.ConvertType(out[2], new(uint16)).(*uint16)
	outstruct.PluginConfig = *abi.ConvertType(out[3], new(uint8)).(*uint8)
	outstruct.ActiveLiquidity = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.NextTick = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)
	outstruct.PreviousTick = *abi.ConvertType(out[6], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// SafelyGetStateOfAMM is a free data retrieval call binding the contract method 0x97ce1c51.
//
// Solidity: function safelyGetStateOfAMM() view returns(uint160 sqrtPrice, int24 tick, uint16 lastFee, uint8 pluginConfig, uint128 activeLiquidity, int24 nextTick, int24 previousTick)
func (_Pool *PoolSession) SafelyGetStateOfAMM() (struct {
	SqrtPrice       *big.Int
	Tick            *big.Int
	LastFee         uint16
	PluginConfig    uint8
	ActiveLiquidity *big.Int
	NextTick        *big.Int
	PreviousTick    *big.Int
}, error) {
	return _Pool.Contract.SafelyGetStateOfAMM(&_Pool.CallOpts)
}

// SafelyGetStateOfAMM is a free data retrieval call binding the contract method 0x97ce1c51.
//
// Solidity: function safelyGetStateOfAMM() view returns(uint160 sqrtPrice, int24 tick, uint16 lastFee, uint8 pluginConfig, uint128 activeLiquidity, int24 nextTick, int24 previousTick)
func (_Pool *PoolCallerSession) SafelyGetStateOfAMM() (struct {
	SqrtPrice       *big.Int
	Tick            *big.Int
	LastFee         uint16
	PluginConfig    uint8
	ActiveLiquidity *big.Int
	NextTick        *big.Int
	PreviousTick    *big.Int
}, error) {
	return _Pool.Contract.SafelyGetStateOfAMM(&_Pool.CallOpts)
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

// TickTable is a free data retrieval call binding the contract method 0xc677e3e0.
//
// Solidity: function tickTable(int16 ) view returns(uint256)
func (_Pool *PoolCaller) TickTable(opts *bind.CallOpts, arg0 int16) (*big.Int, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "tickTable", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TickTable is a free data retrieval call binding the contract method 0xc677e3e0.
//
// Solidity: function tickTable(int16 ) view returns(uint256)
func (_Pool *PoolSession) TickTable(arg0 int16) (*big.Int, error) {
	return _Pool.Contract.TickTable(&_Pool.CallOpts, arg0)
}

// TickTable is a free data retrieval call binding the contract method 0xc677e3e0.
//
// Solidity: function tickTable(int16 ) view returns(uint256)
func (_Pool *PoolCallerSession) TickTable(arg0 int16) (*big.Int, error) {
	return _Pool.Contract.TickTable(&_Pool.CallOpts, arg0)
}

// TickTreeRoot is a free data retrieval call binding the contract method 0x578b9a36.
//
// Solidity: function tickTreeRoot() view returns(uint32)
func (_Pool *PoolCaller) TickTreeRoot(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "tickTreeRoot")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// TickTreeRoot is a free data retrieval call binding the contract method 0x578b9a36.
//
// Solidity: function tickTreeRoot() view returns(uint32)
func (_Pool *PoolSession) TickTreeRoot() (uint32, error) {
	return _Pool.Contract.TickTreeRoot(&_Pool.CallOpts)
}

// TickTreeRoot is a free data retrieval call binding the contract method 0x578b9a36.
//
// Solidity: function tickTreeRoot() view returns(uint32)
func (_Pool *PoolCallerSession) TickTreeRoot() (uint32, error) {
	return _Pool.Contract.TickTreeRoot(&_Pool.CallOpts)
}

// TickTreeSecondLayer is a free data retrieval call binding the contract method 0xd8619037.
//
// Solidity: function tickTreeSecondLayer(int16 ) view returns(uint256)
func (_Pool *PoolCaller) TickTreeSecondLayer(opts *bind.CallOpts, arg0 int16) (*big.Int, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "tickTreeSecondLayer", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TickTreeSecondLayer is a free data retrieval call binding the contract method 0xd8619037.
//
// Solidity: function tickTreeSecondLayer(int16 ) view returns(uint256)
func (_Pool *PoolSession) TickTreeSecondLayer(arg0 int16) (*big.Int, error) {
	return _Pool.Contract.TickTreeSecondLayer(&_Pool.CallOpts, arg0)
}

// TickTreeSecondLayer is a free data retrieval call binding the contract method 0xd8619037.
//
// Solidity: function tickTreeSecondLayer(int16 ) view returns(uint256)
func (_Pool *PoolCallerSession) TickTreeSecondLayer(arg0 int16) (*big.Int, error) {
	return _Pool.Contract.TickTreeSecondLayer(&_Pool.CallOpts, arg0)
}

// Ticks is a free data retrieval call binding the contract method 0xf30dba93.
//
// Solidity: function ticks(int24 ) view returns(uint256 liquidityTotal, int128 liquidityDelta, int24 prevTick, int24 nextTick, uint256 outerFeeGrowth0Token, uint256 outerFeeGrowth1Token)
func (_Pool *PoolCaller) Ticks(opts *bind.CallOpts, arg0 *big.Int) (struct {
	LiquidityTotal       *big.Int
	LiquidityDelta       *big.Int
	PrevTick             *big.Int
	NextTick             *big.Int
	OuterFeeGrowth0Token *big.Int
	OuterFeeGrowth1Token *big.Int
}, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "ticks", arg0)

	outstruct := new(struct {
		LiquidityTotal       *big.Int
		LiquidityDelta       *big.Int
		PrevTick             *big.Int
		NextTick             *big.Int
		OuterFeeGrowth0Token *big.Int
		OuterFeeGrowth1Token *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.LiquidityTotal = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.LiquidityDelta = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.PrevTick = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.NextTick = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.OuterFeeGrowth0Token = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.OuterFeeGrowth1Token = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Ticks is a free data retrieval call binding the contract method 0xf30dba93.
//
// Solidity: function ticks(int24 ) view returns(uint256 liquidityTotal, int128 liquidityDelta, int24 prevTick, int24 nextTick, uint256 outerFeeGrowth0Token, uint256 outerFeeGrowth1Token)
func (_Pool *PoolSession) Ticks(arg0 *big.Int) (struct {
	LiquidityTotal       *big.Int
	LiquidityDelta       *big.Int
	PrevTick             *big.Int
	NextTick             *big.Int
	OuterFeeGrowth0Token *big.Int
	OuterFeeGrowth1Token *big.Int
}, error) {
	return _Pool.Contract.Ticks(&_Pool.CallOpts, arg0)
}

// Ticks is a free data retrieval call binding the contract method 0xf30dba93.
//
// Solidity: function ticks(int24 ) view returns(uint256 liquidityTotal, int128 liquidityDelta, int24 prevTick, int24 nextTick, uint256 outerFeeGrowth0Token, uint256 outerFeeGrowth1Token)
func (_Pool *PoolCallerSession) Ticks(arg0 *big.Int) (struct {
	LiquidityTotal       *big.Int
	LiquidityDelta       *big.Int
	PrevTick             *big.Int
	NextTick             *big.Int
	OuterFeeGrowth0Token *big.Int
	OuterFeeGrowth1Token *big.Int
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

// TotalFeeGrowth0Token is a free data retrieval call binding the contract method 0x6378ae44.
//
// Solidity: function totalFeeGrowth0Token() view returns(uint256)
func (_Pool *PoolCaller) TotalFeeGrowth0Token(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "totalFeeGrowth0Token")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalFeeGrowth0Token is a free data retrieval call binding the contract method 0x6378ae44.
//
// Solidity: function totalFeeGrowth0Token() view returns(uint256)
func (_Pool *PoolSession) TotalFeeGrowth0Token() (*big.Int, error) {
	return _Pool.Contract.TotalFeeGrowth0Token(&_Pool.CallOpts)
}

// TotalFeeGrowth0Token is a free data retrieval call binding the contract method 0x6378ae44.
//
// Solidity: function totalFeeGrowth0Token() view returns(uint256)
func (_Pool *PoolCallerSession) TotalFeeGrowth0Token() (*big.Int, error) {
	return _Pool.Contract.TotalFeeGrowth0Token(&_Pool.CallOpts)
}

// TotalFeeGrowth1Token is a free data retrieval call binding the contract method 0xecdecf42.
//
// Solidity: function totalFeeGrowth1Token() view returns(uint256)
func (_Pool *PoolCaller) TotalFeeGrowth1Token(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Pool.contract.Call(opts, &out, "totalFeeGrowth1Token")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalFeeGrowth1Token is a free data retrieval call binding the contract method 0xecdecf42.
//
// Solidity: function totalFeeGrowth1Token() view returns(uint256)
func (_Pool *PoolSession) TotalFeeGrowth1Token() (*big.Int, error) {
	return _Pool.Contract.TotalFeeGrowth1Token(&_Pool.CallOpts)
}

// TotalFeeGrowth1Token is a free data retrieval call binding the contract method 0xecdecf42.
//
// Solidity: function totalFeeGrowth1Token() view returns(uint256)
func (_Pool *PoolCallerSession) TotalFeeGrowth1Token() (*big.Int, error) {
	return _Pool.Contract.TotalFeeGrowth1Token(&_Pool.CallOpts)
}

// Burn is a paid mutator transaction binding the contract method 0x3b3bc70e.
//
// Solidity: function burn(int24 bottomTick, int24 topTick, uint128 amount, bytes data) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolTransactor) Burn(opts *bind.TransactOpts, bottomTick *big.Int, topTick *big.Int, amount *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "burn", bottomTick, topTick, amount, data)
}

// Burn is a paid mutator transaction binding the contract method 0x3b3bc70e.
//
// Solidity: function burn(int24 bottomTick, int24 topTick, uint128 amount, bytes data) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolSession) Burn(bottomTick *big.Int, topTick *big.Int, amount *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.Burn(&_Pool.TransactOpts, bottomTick, topTick, amount, data)
}

// Burn is a paid mutator transaction binding the contract method 0x3b3bc70e.
//
// Solidity: function burn(int24 bottomTick, int24 topTick, uint128 amount, bytes data) returns(uint256 amount0, uint256 amount1)
func (_Pool *PoolTransactorSession) Burn(bottomTick *big.Int, topTick *big.Int, amount *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.Burn(&_Pool.TransactOpts, bottomTick, topTick, amount, data)
}

// Collect is a paid mutator transaction binding the contract method 0x4f1eb3d8.
//
// Solidity: function collect(address recipient, int24 bottomTick, int24 topTick, uint128 amount0Requested, uint128 amount1Requested) returns(uint128 amount0, uint128 amount1)
func (_Pool *PoolTransactor) Collect(opts *bind.TransactOpts, recipient common.Address, bottomTick *big.Int, topTick *big.Int, amount0Requested *big.Int, amount1Requested *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "collect", recipient, bottomTick, topTick, amount0Requested, amount1Requested)
}

// Collect is a paid mutator transaction binding the contract method 0x4f1eb3d8.
//
// Solidity: function collect(address recipient, int24 bottomTick, int24 topTick, uint128 amount0Requested, uint128 amount1Requested) returns(uint128 amount0, uint128 amount1)
func (_Pool *PoolSession) Collect(recipient common.Address, bottomTick *big.Int, topTick *big.Int, amount0Requested *big.Int, amount1Requested *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Collect(&_Pool.TransactOpts, recipient, bottomTick, topTick, amount0Requested, amount1Requested)
}

// Collect is a paid mutator transaction binding the contract method 0x4f1eb3d8.
//
// Solidity: function collect(address recipient, int24 bottomTick, int24 topTick, uint128 amount0Requested, uint128 amount1Requested) returns(uint128 amount0, uint128 amount1)
func (_Pool *PoolTransactorSession) Collect(recipient common.Address, bottomTick *big.Int, topTick *big.Int, amount0Requested *big.Int, amount1Requested *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Collect(&_Pool.TransactOpts, recipient, bottomTick, topTick, amount0Requested, amount1Requested)
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
// Solidity: function initialize(uint160 initialPrice) returns()
func (_Pool *PoolTransactor) Initialize(opts *bind.TransactOpts, initialPrice *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "initialize", initialPrice)
}

// Initialize is a paid mutator transaction binding the contract method 0xf637731d.
//
// Solidity: function initialize(uint160 initialPrice) returns()
func (_Pool *PoolSession) Initialize(initialPrice *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Initialize(&_Pool.TransactOpts, initialPrice)
}

// Initialize is a paid mutator transaction binding the contract method 0xf637731d.
//
// Solidity: function initialize(uint160 initialPrice) returns()
func (_Pool *PoolTransactorSession) Initialize(initialPrice *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.Initialize(&_Pool.TransactOpts, initialPrice)
}

// Mint is a paid mutator transaction binding the contract method 0xaafe29c0.
//
// Solidity: function mint(address leftoversRecipient, address recipient, int24 bottomTick, int24 topTick, uint128 liquidityDesired, bytes data) returns(uint256 amount0, uint256 amount1, uint128 liquidityActual)
func (_Pool *PoolTransactor) Mint(opts *bind.TransactOpts, leftoversRecipient common.Address, recipient common.Address, bottomTick *big.Int, topTick *big.Int, liquidityDesired *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "mint", leftoversRecipient, recipient, bottomTick, topTick, liquidityDesired, data)
}

// Mint is a paid mutator transaction binding the contract method 0xaafe29c0.
//
// Solidity: function mint(address leftoversRecipient, address recipient, int24 bottomTick, int24 topTick, uint128 liquidityDesired, bytes data) returns(uint256 amount0, uint256 amount1, uint128 liquidityActual)
func (_Pool *PoolSession) Mint(leftoversRecipient common.Address, recipient common.Address, bottomTick *big.Int, topTick *big.Int, liquidityDesired *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.Mint(&_Pool.TransactOpts, leftoversRecipient, recipient, bottomTick, topTick, liquidityDesired, data)
}

// Mint is a paid mutator transaction binding the contract method 0xaafe29c0.
//
// Solidity: function mint(address leftoversRecipient, address recipient, int24 bottomTick, int24 topTick, uint128 liquidityDesired, bytes data) returns(uint256 amount0, uint256 amount1, uint128 liquidityActual)
func (_Pool *PoolTransactorSession) Mint(leftoversRecipient common.Address, recipient common.Address, bottomTick *big.Int, topTick *big.Int, liquidityDesired *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.Mint(&_Pool.TransactOpts, leftoversRecipient, recipient, bottomTick, topTick, liquidityDesired, data)
}

// SetCommunityFee is a paid mutator transaction binding the contract method 0x240a875a.
//
// Solidity: function setCommunityFee(uint16 newCommunityFee) returns()
func (_Pool *PoolTransactor) SetCommunityFee(opts *bind.TransactOpts, newCommunityFee uint16) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "setCommunityFee", newCommunityFee)
}

// SetCommunityFee is a paid mutator transaction binding the contract method 0x240a875a.
//
// Solidity: function setCommunityFee(uint16 newCommunityFee) returns()
func (_Pool *PoolSession) SetCommunityFee(newCommunityFee uint16) (*types.Transaction, error) {
	return _Pool.Contract.SetCommunityFee(&_Pool.TransactOpts, newCommunityFee)
}

// SetCommunityFee is a paid mutator transaction binding the contract method 0x240a875a.
//
// Solidity: function setCommunityFee(uint16 newCommunityFee) returns()
func (_Pool *PoolTransactorSession) SetCommunityFee(newCommunityFee uint16) (*types.Transaction, error) {
	return _Pool.Contract.SetCommunityFee(&_Pool.TransactOpts, newCommunityFee)
}

// SetCommunityVault is a paid mutator transaction binding the contract method 0xd8544cf3.
//
// Solidity: function setCommunityVault(address newCommunityVault) returns()
func (_Pool *PoolTransactor) SetCommunityVault(opts *bind.TransactOpts, newCommunityVault common.Address) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "setCommunityVault", newCommunityVault)
}

// SetCommunityVault is a paid mutator transaction binding the contract method 0xd8544cf3.
//
// Solidity: function setCommunityVault(address newCommunityVault) returns()
func (_Pool *PoolSession) SetCommunityVault(newCommunityVault common.Address) (*types.Transaction, error) {
	return _Pool.Contract.SetCommunityVault(&_Pool.TransactOpts, newCommunityVault)
}

// SetCommunityVault is a paid mutator transaction binding the contract method 0xd8544cf3.
//
// Solidity: function setCommunityVault(address newCommunityVault) returns()
func (_Pool *PoolTransactorSession) SetCommunityVault(newCommunityVault common.Address) (*types.Transaction, error) {
	return _Pool.Contract.SetCommunityVault(&_Pool.TransactOpts, newCommunityVault)
}

// SetFee is a paid mutator transaction binding the contract method 0x8e005553.
//
// Solidity: function setFee(uint16 newFee) returns()
func (_Pool *PoolTransactor) SetFee(opts *bind.TransactOpts, newFee uint16) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "setFee", newFee)
}

// SetFee is a paid mutator transaction binding the contract method 0x8e005553.
//
// Solidity: function setFee(uint16 newFee) returns()
func (_Pool *PoolSession) SetFee(newFee uint16) (*types.Transaction, error) {
	return _Pool.Contract.SetFee(&_Pool.TransactOpts, newFee)
}

// SetFee is a paid mutator transaction binding the contract method 0x8e005553.
//
// Solidity: function setFee(uint16 newFee) returns()
func (_Pool *PoolTransactorSession) SetFee(newFee uint16) (*types.Transaction, error) {
	return _Pool.Contract.SetFee(&_Pool.TransactOpts, newFee)
}

// SetPlugin is a paid mutator transaction binding the contract method 0xcc1f97cf.
//
// Solidity: function setPlugin(address newPluginAddress) returns()
func (_Pool *PoolTransactor) SetPlugin(opts *bind.TransactOpts, newPluginAddress common.Address) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "setPlugin", newPluginAddress)
}

// SetPlugin is a paid mutator transaction binding the contract method 0xcc1f97cf.
//
// Solidity: function setPlugin(address newPluginAddress) returns()
func (_Pool *PoolSession) SetPlugin(newPluginAddress common.Address) (*types.Transaction, error) {
	return _Pool.Contract.SetPlugin(&_Pool.TransactOpts, newPluginAddress)
}

// SetPlugin is a paid mutator transaction binding the contract method 0xcc1f97cf.
//
// Solidity: function setPlugin(address newPluginAddress) returns()
func (_Pool *PoolTransactorSession) SetPlugin(newPluginAddress common.Address) (*types.Transaction, error) {
	return _Pool.Contract.SetPlugin(&_Pool.TransactOpts, newPluginAddress)
}

// SetPluginConfig is a paid mutator transaction binding the contract method 0xbca57f81.
//
// Solidity: function setPluginConfig(uint8 newConfig) returns()
func (_Pool *PoolTransactor) SetPluginConfig(opts *bind.TransactOpts, newConfig uint8) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "setPluginConfig", newConfig)
}

// SetPluginConfig is a paid mutator transaction binding the contract method 0xbca57f81.
//
// Solidity: function setPluginConfig(uint8 newConfig) returns()
func (_Pool *PoolSession) SetPluginConfig(newConfig uint8) (*types.Transaction, error) {
	return _Pool.Contract.SetPluginConfig(&_Pool.TransactOpts, newConfig)
}

// SetPluginConfig is a paid mutator transaction binding the contract method 0xbca57f81.
//
// Solidity: function setPluginConfig(uint8 newConfig) returns()
func (_Pool *PoolTransactorSession) SetPluginConfig(newConfig uint8) (*types.Transaction, error) {
	return _Pool.Contract.SetPluginConfig(&_Pool.TransactOpts, newConfig)
}

// SetTickSpacing is a paid mutator transaction binding the contract method 0xf085a610.
//
// Solidity: function setTickSpacing(int24 newTickSpacing) returns()
func (_Pool *PoolTransactor) SetTickSpacing(opts *bind.TransactOpts, newTickSpacing *big.Int) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "setTickSpacing", newTickSpacing)
}

// SetTickSpacing is a paid mutator transaction binding the contract method 0xf085a610.
//
// Solidity: function setTickSpacing(int24 newTickSpacing) returns()
func (_Pool *PoolSession) SetTickSpacing(newTickSpacing *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.SetTickSpacing(&_Pool.TransactOpts, newTickSpacing)
}

// SetTickSpacing is a paid mutator transaction binding the contract method 0xf085a610.
//
// Solidity: function setTickSpacing(int24 newTickSpacing) returns()
func (_Pool *PoolTransactorSession) SetTickSpacing(newTickSpacing *big.Int) (*types.Transaction, error) {
	return _Pool.Contract.SetTickSpacing(&_Pool.TransactOpts, newTickSpacing)
}

// Skim is a paid mutator transaction binding the contract method 0x1dd19cb4.
//
// Solidity: function skim() returns()
func (_Pool *PoolTransactor) Skim(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "skim")
}

// Skim is a paid mutator transaction binding the contract method 0x1dd19cb4.
//
// Solidity: function skim() returns()
func (_Pool *PoolSession) Skim() (*types.Transaction, error) {
	return _Pool.Contract.Skim(&_Pool.TransactOpts)
}

// Skim is a paid mutator transaction binding the contract method 0x1dd19cb4.
//
// Solidity: function skim() returns()
func (_Pool *PoolTransactorSession) Skim() (*types.Transaction, error) {
	return _Pool.Contract.Skim(&_Pool.TransactOpts)
}

// Swap is a paid mutator transaction binding the contract method 0x128acb08.
//
// Solidity: function swap(address recipient, bool zeroToOne, int256 amountRequired, uint160 limitSqrtPrice, bytes data) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactor) Swap(opts *bind.TransactOpts, recipient common.Address, zeroToOne bool, amountRequired *big.Int, limitSqrtPrice *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "swap", recipient, zeroToOne, amountRequired, limitSqrtPrice, data)
}

// Swap is a paid mutator transaction binding the contract method 0x128acb08.
//
// Solidity: function swap(address recipient, bool zeroToOne, int256 amountRequired, uint160 limitSqrtPrice, bytes data) returns(int256 amount0, int256 amount1)
func (_Pool *PoolSession) Swap(recipient common.Address, zeroToOne bool, amountRequired *big.Int, limitSqrtPrice *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.Swap(&_Pool.TransactOpts, recipient, zeroToOne, amountRequired, limitSqrtPrice, data)
}

// Swap is a paid mutator transaction binding the contract method 0x128acb08.
//
// Solidity: function swap(address recipient, bool zeroToOne, int256 amountRequired, uint160 limitSqrtPrice, bytes data) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactorSession) Swap(recipient common.Address, zeroToOne bool, amountRequired *big.Int, limitSqrtPrice *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.Swap(&_Pool.TransactOpts, recipient, zeroToOne, amountRequired, limitSqrtPrice, data)
}

// SwapWithPaymentInAdvance is a paid mutator transaction binding the contract method 0x9e4e0227.
//
// Solidity: function swapWithPaymentInAdvance(address leftoversRecipient, address recipient, bool zeroToOne, int256 amountToSell, uint160 limitSqrtPrice, bytes data) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactor) SwapWithPaymentInAdvance(opts *bind.TransactOpts, leftoversRecipient common.Address, recipient common.Address, zeroToOne bool, amountToSell *big.Int, limitSqrtPrice *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "swapWithPaymentInAdvance", leftoversRecipient, recipient, zeroToOne, amountToSell, limitSqrtPrice, data)
}

// SwapWithPaymentInAdvance is a paid mutator transaction binding the contract method 0x9e4e0227.
//
// Solidity: function swapWithPaymentInAdvance(address leftoversRecipient, address recipient, bool zeroToOne, int256 amountToSell, uint160 limitSqrtPrice, bytes data) returns(int256 amount0, int256 amount1)
func (_Pool *PoolSession) SwapWithPaymentInAdvance(leftoversRecipient common.Address, recipient common.Address, zeroToOne bool, amountToSell *big.Int, limitSqrtPrice *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.SwapWithPaymentInAdvance(&_Pool.TransactOpts, leftoversRecipient, recipient, zeroToOne, amountToSell, limitSqrtPrice, data)
}

// SwapWithPaymentInAdvance is a paid mutator transaction binding the contract method 0x9e4e0227.
//
// Solidity: function swapWithPaymentInAdvance(address leftoversRecipient, address recipient, bool zeroToOne, int256 amountToSell, uint160 limitSqrtPrice, bytes data) returns(int256 amount0, int256 amount1)
func (_Pool *PoolTransactorSession) SwapWithPaymentInAdvance(leftoversRecipient common.Address, recipient common.Address, zeroToOne bool, amountToSell *big.Int, limitSqrtPrice *big.Int, data []byte) (*types.Transaction, error) {
	return _Pool.Contract.SwapWithPaymentInAdvance(&_Pool.TransactOpts, leftoversRecipient, recipient, zeroToOne, amountToSell, limitSqrtPrice, data)
}

// Sync is a paid mutator transaction binding the contract method 0xfff6cae9.
//
// Solidity: function sync() returns()
func (_Pool *PoolTransactor) Sync(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Pool.contract.Transact(opts, "sync")
}

// Sync is a paid mutator transaction binding the contract method 0xfff6cae9.
//
// Solidity: function sync() returns()
func (_Pool *PoolSession) Sync() (*types.Transaction, error) {
	return _Pool.Contract.Sync(&_Pool.TransactOpts)
}

// Sync is a paid mutator transaction binding the contract method 0xfff6cae9.
//
// Solidity: function sync() returns()
func (_Pool *PoolTransactorSession) Sync() (*types.Transaction, error) {
	return _Pool.Contract.Sync(&_Pool.TransactOpts)
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
	Owner           common.Address
	BottomTick      *big.Int
	TopTick         *big.Int
	LiquidityAmount *big.Int
	Amount0         *big.Int
	Amount1         *big.Int
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterBurn is a free log retrieval operation binding the contract event 0x0c396cd989a39f4459b5fa1aed6a9a8dcdbc45908acfd67e028cd568da98982c.
//
// Solidity: event Burn(address indexed owner, int24 indexed bottomTick, int24 indexed topTick, uint128 liquidityAmount, uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) FilterBurn(opts *bind.FilterOpts, owner []common.Address, bottomTick []*big.Int, topTick []*big.Int) (*PoolBurnIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var bottomTickRule []interface{}
	for _, bottomTickItem := range bottomTick {
		bottomTickRule = append(bottomTickRule, bottomTickItem)
	}
	var topTickRule []interface{}
	for _, topTickItem := range topTick {
		topTickRule = append(topTickRule, topTickItem)
	}

	logs, sub, err := _Pool.contract.FilterLogs(opts, "Burn", ownerRule, bottomTickRule, topTickRule)
	if err != nil {
		return nil, err
	}
	return &PoolBurnIterator{contract: _Pool.contract, event: "Burn", logs: logs, sub: sub}, nil
}

// WatchBurn is a free log subscription operation binding the contract event 0x0c396cd989a39f4459b5fa1aed6a9a8dcdbc45908acfd67e028cd568da98982c.
//
// Solidity: event Burn(address indexed owner, int24 indexed bottomTick, int24 indexed topTick, uint128 liquidityAmount, uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) WatchBurn(opts *bind.WatchOpts, sink chan<- *PoolBurn, owner []common.Address, bottomTick []*big.Int, topTick []*big.Int) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var bottomTickRule []interface{}
	for _, bottomTickItem := range bottomTick {
		bottomTickRule = append(bottomTickRule, bottomTickItem)
	}
	var topTickRule []interface{}
	for _, topTickItem := range topTick {
		topTickRule = append(topTickRule, topTickItem)
	}

	logs, sub, err := _Pool.contract.WatchLogs(opts, "Burn", ownerRule, bottomTickRule, topTickRule)
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
// Solidity: event Burn(address indexed owner, int24 indexed bottomTick, int24 indexed topTick, uint128 liquidityAmount, uint256 amount0, uint256 amount1)
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
	Owner      common.Address
	Recipient  common.Address
	BottomTick *big.Int
	TopTick    *big.Int
	Amount0    *big.Int
	Amount1    *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterCollect is a free log retrieval operation binding the contract event 0x70935338e69775456a85ddef226c395fb668b63fa0115f5f20610b388e6ca9c0.
//
// Solidity: event Collect(address indexed owner, address recipient, int24 indexed bottomTick, int24 indexed topTick, uint128 amount0, uint128 amount1)
func (_Pool *PoolFilterer) FilterCollect(opts *bind.FilterOpts, owner []common.Address, bottomTick []*big.Int, topTick []*big.Int) (*PoolCollectIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	var bottomTickRule []interface{}
	for _, bottomTickItem := range bottomTick {
		bottomTickRule = append(bottomTickRule, bottomTickItem)
	}
	var topTickRule []interface{}
	for _, topTickItem := range topTick {
		topTickRule = append(topTickRule, topTickItem)
	}

	logs, sub, err := _Pool.contract.FilterLogs(opts, "Collect", ownerRule, bottomTickRule, topTickRule)
	if err != nil {
		return nil, err
	}
	return &PoolCollectIterator{contract: _Pool.contract, event: "Collect", logs: logs, sub: sub}, nil
}

// WatchCollect is a free log subscription operation binding the contract event 0x70935338e69775456a85ddef226c395fb668b63fa0115f5f20610b388e6ca9c0.
//
// Solidity: event Collect(address indexed owner, address recipient, int24 indexed bottomTick, int24 indexed topTick, uint128 amount0, uint128 amount1)
func (_Pool *PoolFilterer) WatchCollect(opts *bind.WatchOpts, sink chan<- *PoolCollect, owner []common.Address, bottomTick []*big.Int, topTick []*big.Int) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	var bottomTickRule []interface{}
	for _, bottomTickItem := range bottomTick {
		bottomTickRule = append(bottomTickRule, bottomTickItem)
	}
	var topTickRule []interface{}
	for _, topTickItem := range topTick {
		topTickRule = append(topTickRule, topTickItem)
	}

	logs, sub, err := _Pool.contract.WatchLogs(opts, "Collect", ownerRule, bottomTickRule, topTickRule)
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
// Solidity: event Collect(address indexed owner, address recipient, int24 indexed bottomTick, int24 indexed topTick, uint128 amount0, uint128 amount1)
func (_Pool *PoolFilterer) ParseCollect(log types.Log) (*PoolCollect, error) {
	event := new(PoolCollect)
	if err := _Pool.contract.UnpackLog(event, "Collect", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolCommunityFeeIterator is returned from FilterCommunityFee and is used to iterate over the raw logs and unpacked data for CommunityFee events raised by the Pool contract.
type PoolCommunityFeeIterator struct {
	Event *PoolCommunityFee // Event containing the contract specifics and raw log

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
func (it *PoolCommunityFeeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolCommunityFee)
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
		it.Event = new(PoolCommunityFee)
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
func (it *PoolCommunityFeeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolCommunityFeeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolCommunityFee represents a CommunityFee event raised by the Pool contract.
type PoolCommunityFee struct {
	CommunityFeeNew uint16
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterCommunityFee is a free log retrieval operation binding the contract event 0x3647dccc990d4941b0b05b32527ef493a98d6187b20639ca2f9743f3b55ca5e1.
//
// Solidity: event CommunityFee(uint16 communityFeeNew)
func (_Pool *PoolFilterer) FilterCommunityFee(opts *bind.FilterOpts) (*PoolCommunityFeeIterator, error) {

	logs, sub, err := _Pool.contract.FilterLogs(opts, "CommunityFee")
	if err != nil {
		return nil, err
	}
	return &PoolCommunityFeeIterator{contract: _Pool.contract, event: "CommunityFee", logs: logs, sub: sub}, nil
}

// WatchCommunityFee is a free log subscription operation binding the contract event 0x3647dccc990d4941b0b05b32527ef493a98d6187b20639ca2f9743f3b55ca5e1.
//
// Solidity: event CommunityFee(uint16 communityFeeNew)
func (_Pool *PoolFilterer) WatchCommunityFee(opts *bind.WatchOpts, sink chan<- *PoolCommunityFee) (event.Subscription, error) {

	logs, sub, err := _Pool.contract.WatchLogs(opts, "CommunityFee")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolCommunityFee)
				if err := _Pool.contract.UnpackLog(event, "CommunityFee", log); err != nil {
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

// ParseCommunityFee is a log parse operation binding the contract event 0x3647dccc990d4941b0b05b32527ef493a98d6187b20639ca2f9743f3b55ca5e1.
//
// Solidity: event CommunityFee(uint16 communityFeeNew)
func (_Pool *PoolFilterer) ParseCommunityFee(log types.Log) (*PoolCommunityFee, error) {
	event := new(PoolCommunityFee)
	if err := _Pool.contract.UnpackLog(event, "CommunityFee", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolCommunityVaultIterator is returned from FilterCommunityVault and is used to iterate over the raw logs and unpacked data for CommunityVault events raised by the Pool contract.
type PoolCommunityVaultIterator struct {
	Event *PoolCommunityVault // Event containing the contract specifics and raw log

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
func (it *PoolCommunityVaultIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolCommunityVault)
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
		it.Event = new(PoolCommunityVault)
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
func (it *PoolCommunityVaultIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolCommunityVaultIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolCommunityVault represents a CommunityVault event raised by the Pool contract.
type PoolCommunityVault struct {
	NewCommunityVault common.Address
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterCommunityVault is a free log retrieval operation binding the contract event 0xb0b573c1f636e1f8bd9b415ba6c04d6dd49100bc25493fc6305b65ec0e581df3.
//
// Solidity: event CommunityVault(address newCommunityVault)
func (_Pool *PoolFilterer) FilterCommunityVault(opts *bind.FilterOpts) (*PoolCommunityVaultIterator, error) {

	logs, sub, err := _Pool.contract.FilterLogs(opts, "CommunityVault")
	if err != nil {
		return nil, err
	}
	return &PoolCommunityVaultIterator{contract: _Pool.contract, event: "CommunityVault", logs: logs, sub: sub}, nil
}

// WatchCommunityVault is a free log subscription operation binding the contract event 0xb0b573c1f636e1f8bd9b415ba6c04d6dd49100bc25493fc6305b65ec0e581df3.
//
// Solidity: event CommunityVault(address newCommunityVault)
func (_Pool *PoolFilterer) WatchCommunityVault(opts *bind.WatchOpts, sink chan<- *PoolCommunityVault) (event.Subscription, error) {

	logs, sub, err := _Pool.contract.WatchLogs(opts, "CommunityVault")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolCommunityVault)
				if err := _Pool.contract.UnpackLog(event, "CommunityVault", log); err != nil {
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

// ParseCommunityVault is a log parse operation binding the contract event 0xb0b573c1f636e1f8bd9b415ba6c04d6dd49100bc25493fc6305b65ec0e581df3.
//
// Solidity: event CommunityVault(address newCommunityVault)
func (_Pool *PoolFilterer) ParseCommunityVault(log types.Log) (*PoolCommunityVault, error) {
	event := new(PoolCommunityVault)
	if err := _Pool.contract.UnpackLog(event, "CommunityVault", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolExcessTokensIterator is returned from FilterExcessTokens and is used to iterate over the raw logs and unpacked data for ExcessTokens events raised by the Pool contract.
type PoolExcessTokensIterator struct {
	Event *PoolExcessTokens // Event containing the contract specifics and raw log

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
func (it *PoolExcessTokensIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolExcessTokens)
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
		it.Event = new(PoolExcessTokens)
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
func (it *PoolExcessTokensIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolExcessTokensIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolExcessTokens represents a ExcessTokens event raised by the Pool contract.
type PoolExcessTokens struct {
	Amount0 *big.Int
	Amount1 *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterExcessTokens is a free log retrieval operation binding the contract event 0xef10ebb00f0dbc72ad4602e94abbbda6f3d40632714f70e9c8fa30d5d44289c9.
//
// Solidity: event ExcessTokens(uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) FilterExcessTokens(opts *bind.FilterOpts) (*PoolExcessTokensIterator, error) {

	logs, sub, err := _Pool.contract.FilterLogs(opts, "ExcessTokens")
	if err != nil {
		return nil, err
	}
	return &PoolExcessTokensIterator{contract: _Pool.contract, event: "ExcessTokens", logs: logs, sub: sub}, nil
}

// WatchExcessTokens is a free log subscription operation binding the contract event 0xef10ebb00f0dbc72ad4602e94abbbda6f3d40632714f70e9c8fa30d5d44289c9.
//
// Solidity: event ExcessTokens(uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) WatchExcessTokens(opts *bind.WatchOpts, sink chan<- *PoolExcessTokens) (event.Subscription, error) {

	logs, sub, err := _Pool.contract.WatchLogs(opts, "ExcessTokens")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolExcessTokens)
				if err := _Pool.contract.UnpackLog(event, "ExcessTokens", log); err != nil {
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

// ParseExcessTokens is a log parse operation binding the contract event 0xef10ebb00f0dbc72ad4602e94abbbda6f3d40632714f70e9c8fa30d5d44289c9.
//
// Solidity: event ExcessTokens(uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) ParseExcessTokens(log types.Log) (*PoolExcessTokens, error) {
	event := new(PoolExcessTokens)
	if err := _Pool.contract.UnpackLog(event, "ExcessTokens", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolFeeIterator is returned from FilterFee and is used to iterate over the raw logs and unpacked data for Fee events raised by the Pool contract.
type PoolFeeIterator struct {
	Event *PoolFee // Event containing the contract specifics and raw log

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
func (it *PoolFeeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolFee)
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
		it.Event = new(PoolFee)
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
func (it *PoolFeeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolFeeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolFee represents a Fee event raised by the Pool contract.
type PoolFee struct {
	Fee uint16
	Raw types.Log // Blockchain specific contextual infos
}

// FilterFee is a free log retrieval operation binding the contract event 0x598b9f043c813aa6be3426ca60d1c65d17256312890be5118dab55b0775ebe2a.
//
// Solidity: event Fee(uint16 fee)
func (_Pool *PoolFilterer) FilterFee(opts *bind.FilterOpts) (*PoolFeeIterator, error) {

	logs, sub, err := _Pool.contract.FilterLogs(opts, "Fee")
	if err != nil {
		return nil, err
	}
	return &PoolFeeIterator{contract: _Pool.contract, event: "Fee", logs: logs, sub: sub}, nil
}

// WatchFee is a free log subscription operation binding the contract event 0x598b9f043c813aa6be3426ca60d1c65d17256312890be5118dab55b0775ebe2a.
//
// Solidity: event Fee(uint16 fee)
func (_Pool *PoolFilterer) WatchFee(opts *bind.WatchOpts, sink chan<- *PoolFee) (event.Subscription, error) {

	logs, sub, err := _Pool.contract.WatchLogs(opts, "Fee")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolFee)
				if err := _Pool.contract.UnpackLog(event, "Fee", log); err != nil {
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

// ParseFee is a log parse operation binding the contract event 0x598b9f043c813aa6be3426ca60d1c65d17256312890be5118dab55b0775ebe2a.
//
// Solidity: event Fee(uint16 fee)
func (_Pool *PoolFilterer) ParseFee(log types.Log) (*PoolFee, error) {
	event := new(PoolFee)
	if err := _Pool.contract.UnpackLog(event, "Fee", log); err != nil {
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
	Price *big.Int
	Tick  *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterInitialize is a free log retrieval operation binding the contract event 0x98636036cb66a9c19a37435efc1e90142190214e8abeb821bdba3f2990dd4c95.
//
// Solidity: event Initialize(uint160 price, int24 tick)
func (_Pool *PoolFilterer) FilterInitialize(opts *bind.FilterOpts) (*PoolInitializeIterator, error) {

	logs, sub, err := _Pool.contract.FilterLogs(opts, "Initialize")
	if err != nil {
		return nil, err
	}
	return &PoolInitializeIterator{contract: _Pool.contract, event: "Initialize", logs: logs, sub: sub}, nil
}

// WatchInitialize is a free log subscription operation binding the contract event 0x98636036cb66a9c19a37435efc1e90142190214e8abeb821bdba3f2990dd4c95.
//
// Solidity: event Initialize(uint160 price, int24 tick)
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
// Solidity: event Initialize(uint160 price, int24 tick)
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
	Sender          common.Address
	Owner           common.Address
	BottomTick      *big.Int
	TopTick         *big.Int
	LiquidityAmount *big.Int
	Amount0         *big.Int
	Amount1         *big.Int
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterMint is a free log retrieval operation binding the contract event 0x7a53080ba414158be7ec69b987b5fb7d07dee101fe85488f0853ae16239d0bde.
//
// Solidity: event Mint(address sender, address indexed owner, int24 indexed bottomTick, int24 indexed topTick, uint128 liquidityAmount, uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) FilterMint(opts *bind.FilterOpts, owner []common.Address, bottomTick []*big.Int, topTick []*big.Int) (*PoolMintIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var bottomTickRule []interface{}
	for _, bottomTickItem := range bottomTick {
		bottomTickRule = append(bottomTickRule, bottomTickItem)
	}
	var topTickRule []interface{}
	for _, topTickItem := range topTick {
		topTickRule = append(topTickRule, topTickItem)
	}

	logs, sub, err := _Pool.contract.FilterLogs(opts, "Mint", ownerRule, bottomTickRule, topTickRule)
	if err != nil {
		return nil, err
	}
	return &PoolMintIterator{contract: _Pool.contract, event: "Mint", logs: logs, sub: sub}, nil
}

// WatchMint is a free log subscription operation binding the contract event 0x7a53080ba414158be7ec69b987b5fb7d07dee101fe85488f0853ae16239d0bde.
//
// Solidity: event Mint(address sender, address indexed owner, int24 indexed bottomTick, int24 indexed topTick, uint128 liquidityAmount, uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) WatchMint(opts *bind.WatchOpts, sink chan<- *PoolMint, owner []common.Address, bottomTick []*big.Int, topTick []*big.Int) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var bottomTickRule []interface{}
	for _, bottomTickItem := range bottomTick {
		bottomTickRule = append(bottomTickRule, bottomTickItem)
	}
	var topTickRule []interface{}
	for _, topTickItem := range topTick {
		topTickRule = append(topTickRule, topTickItem)
	}

	logs, sub, err := _Pool.contract.WatchLogs(opts, "Mint", ownerRule, bottomTickRule, topTickRule)
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
// Solidity: event Mint(address sender, address indexed owner, int24 indexed bottomTick, int24 indexed topTick, uint128 liquidityAmount, uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) ParseMint(log types.Log) (*PoolMint, error) {
	event := new(PoolMint)
	if err := _Pool.contract.UnpackLog(event, "Mint", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolPluginIterator is returned from FilterPlugin and is used to iterate over the raw logs and unpacked data for Plugin events raised by the Pool contract.
type PoolPluginIterator struct {
	Event *PoolPlugin // Event containing the contract specifics and raw log

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
func (it *PoolPluginIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolPlugin)
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
		it.Event = new(PoolPlugin)
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
func (it *PoolPluginIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolPluginIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolPlugin represents a Plugin event raised by the Pool contract.
type PoolPlugin struct {
	NewPluginAddress common.Address
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterPlugin is a free log retrieval operation binding the contract event 0x27a3944eff2135a57675f17e72501038982b73620d01f794c72e93d61a3932a2.
//
// Solidity: event Plugin(address newPluginAddress)
func (_Pool *PoolFilterer) FilterPlugin(opts *bind.FilterOpts) (*PoolPluginIterator, error) {

	logs, sub, err := _Pool.contract.FilterLogs(opts, "Plugin")
	if err != nil {
		return nil, err
	}
	return &PoolPluginIterator{contract: _Pool.contract, event: "Plugin", logs: logs, sub: sub}, nil
}

// WatchPlugin is a free log subscription operation binding the contract event 0x27a3944eff2135a57675f17e72501038982b73620d01f794c72e93d61a3932a2.
//
// Solidity: event Plugin(address newPluginAddress)
func (_Pool *PoolFilterer) WatchPlugin(opts *bind.WatchOpts, sink chan<- *PoolPlugin) (event.Subscription, error) {

	logs, sub, err := _Pool.contract.WatchLogs(opts, "Plugin")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolPlugin)
				if err := _Pool.contract.UnpackLog(event, "Plugin", log); err != nil {
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

// ParsePlugin is a log parse operation binding the contract event 0x27a3944eff2135a57675f17e72501038982b73620d01f794c72e93d61a3932a2.
//
// Solidity: event Plugin(address newPluginAddress)
func (_Pool *PoolFilterer) ParsePlugin(log types.Log) (*PoolPlugin, error) {
	event := new(PoolPlugin)
	if err := _Pool.contract.UnpackLog(event, "Plugin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolPluginConfigIterator is returned from FilterPluginConfig and is used to iterate over the raw logs and unpacked data for PluginConfig events raised by the Pool contract.
type PoolPluginConfigIterator struct {
	Event *PoolPluginConfig // Event containing the contract specifics and raw log

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
func (it *PoolPluginConfigIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolPluginConfig)
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
		it.Event = new(PoolPluginConfig)
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
func (it *PoolPluginConfigIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolPluginConfigIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolPluginConfig represents a PluginConfig event raised by the Pool contract.
type PoolPluginConfig struct {
	NewPluginConfig uint8
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterPluginConfig is a free log retrieval operation binding the contract event 0x3a6271b36c1b44bd6a0a0d56230602dc6919b7c17af57254306fadf5fee69dc3.
//
// Solidity: event PluginConfig(uint8 newPluginConfig)
func (_Pool *PoolFilterer) FilterPluginConfig(opts *bind.FilterOpts) (*PoolPluginConfigIterator, error) {

	logs, sub, err := _Pool.contract.FilterLogs(opts, "PluginConfig")
	if err != nil {
		return nil, err
	}
	return &PoolPluginConfigIterator{contract: _Pool.contract, event: "PluginConfig", logs: logs, sub: sub}, nil
}

// WatchPluginConfig is a free log subscription operation binding the contract event 0x3a6271b36c1b44bd6a0a0d56230602dc6919b7c17af57254306fadf5fee69dc3.
//
// Solidity: event PluginConfig(uint8 newPluginConfig)
func (_Pool *PoolFilterer) WatchPluginConfig(opts *bind.WatchOpts, sink chan<- *PoolPluginConfig) (event.Subscription, error) {

	logs, sub, err := _Pool.contract.WatchLogs(opts, "PluginConfig")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolPluginConfig)
				if err := _Pool.contract.UnpackLog(event, "PluginConfig", log); err != nil {
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

// ParsePluginConfig is a log parse operation binding the contract event 0x3a6271b36c1b44bd6a0a0d56230602dc6919b7c17af57254306fadf5fee69dc3.
//
// Solidity: event PluginConfig(uint8 newPluginConfig)
func (_Pool *PoolFilterer) ParsePluginConfig(log types.Log) (*PoolPluginConfig, error) {
	event := new(PoolPluginConfig)
	if err := _Pool.contract.UnpackLog(event, "PluginConfig", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolSkimIterator is returned from FilterSkim and is used to iterate over the raw logs and unpacked data for Skim events raised by the Pool contract.
type PoolSkimIterator struct {
	Event *PoolSkim // Event containing the contract specifics and raw log

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
func (it *PoolSkimIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolSkim)
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
		it.Event = new(PoolSkim)
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
func (it *PoolSkimIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolSkimIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolSkim represents a Skim event raised by the Pool contract.
type PoolSkim struct {
	To      common.Address
	Amount0 *big.Int
	Amount1 *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterSkim is a free log retrieval operation binding the contract event 0xb94331e4420f16b156f53c397a8adcd09481283ee7830f7b688b22858e9db80b.
//
// Solidity: event Skim(address indexed to, uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) FilterSkim(opts *bind.FilterOpts, to []common.Address) (*PoolSkimIterator, error) {

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Pool.contract.FilterLogs(opts, "Skim", toRule)
	if err != nil {
		return nil, err
	}
	return &PoolSkimIterator{contract: _Pool.contract, event: "Skim", logs: logs, sub: sub}, nil
}

// WatchSkim is a free log subscription operation binding the contract event 0xb94331e4420f16b156f53c397a8adcd09481283ee7830f7b688b22858e9db80b.
//
// Solidity: event Skim(address indexed to, uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) WatchSkim(opts *bind.WatchOpts, sink chan<- *PoolSkim, to []common.Address) (event.Subscription, error) {

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Pool.contract.WatchLogs(opts, "Skim", toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolSkim)
				if err := _Pool.contract.UnpackLog(event, "Skim", log); err != nil {
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

// ParseSkim is a log parse operation binding the contract event 0xb94331e4420f16b156f53c397a8adcd09481283ee7830f7b688b22858e9db80b.
//
// Solidity: event Skim(address indexed to, uint256 amount0, uint256 amount1)
func (_Pool *PoolFilterer) ParseSkim(log types.Log) (*PoolSkim, error) {
	event := new(PoolSkim)
	if err := _Pool.contract.UnpackLog(event, "Skim", log); err != nil {
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
	Sender    common.Address
	Recipient common.Address
	Amount0   *big.Int
	Amount1   *big.Int
	Price     *big.Int
	Liquidity *big.Int
	Tick      *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterSwap is a free log retrieval operation binding the contract event 0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67.
//
// Solidity: event Swap(address indexed sender, address indexed recipient, int256 amount0, int256 amount1, uint160 price, uint128 liquidity, int24 tick)
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
// Solidity: event Swap(address indexed sender, address indexed recipient, int256 amount0, int256 amount1, uint160 price, uint128 liquidity, int24 tick)
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
// Solidity: event Swap(address indexed sender, address indexed recipient, int256 amount0, int256 amount1, uint160 price, uint128 liquidity, int24 tick)
func (_Pool *PoolFilterer) ParseSwap(log types.Log) (*PoolSwap, error) {
	event := new(PoolSwap)
	if err := _Pool.contract.UnpackLog(event, "Swap", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolTickSpacingIterator is returned from FilterTickSpacing and is used to iterate over the raw logs and unpacked data for TickSpacing events raised by the Pool contract.
type PoolTickSpacingIterator struct {
	Event *PoolTickSpacing // Event containing the contract specifics and raw log

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
func (it *PoolTickSpacingIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolTickSpacing)
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
		it.Event = new(PoolTickSpacing)
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
func (it *PoolTickSpacingIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolTickSpacingIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolTickSpacing represents a TickSpacing event raised by the Pool contract.
type PoolTickSpacing struct {
	NewTickSpacing *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterTickSpacing is a free log retrieval operation binding the contract event 0x01413b1d5d4c359e9a0daa7909ecda165f6e8c51fe2ff529d74b22a5a7c02645.
//
// Solidity: event TickSpacing(int24 newTickSpacing)
func (_Pool *PoolFilterer) FilterTickSpacing(opts *bind.FilterOpts) (*PoolTickSpacingIterator, error) {

	logs, sub, err := _Pool.contract.FilterLogs(opts, "TickSpacing")
	if err != nil {
		return nil, err
	}
	return &PoolTickSpacingIterator{contract: _Pool.contract, event: "TickSpacing", logs: logs, sub: sub}, nil
}

// WatchTickSpacing is a free log subscription operation binding the contract event 0x01413b1d5d4c359e9a0daa7909ecda165f6e8c51fe2ff529d74b22a5a7c02645.
//
// Solidity: event TickSpacing(int24 newTickSpacing)
func (_Pool *PoolFilterer) WatchTickSpacing(opts *bind.WatchOpts, sink chan<- *PoolTickSpacing) (event.Subscription, error) {

	logs, sub, err := _Pool.contract.WatchLogs(opts, "TickSpacing")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolTickSpacing)
				if err := _Pool.contract.UnpackLog(event, "TickSpacing", log); err != nil {
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

// ParseTickSpacing is a log parse operation binding the contract event 0x01413b1d5d4c359e9a0daa7909ecda165f6e8c51fe2ff529d74b22a5a7c02645.
//
// Solidity: event TickSpacing(int24 newTickSpacing)
func (_Pool *PoolFilterer) ParseTickSpacing(log types.Log) (*PoolTickSpacing, error) {
	event := new(PoolTickSpacing)
	if err := _Pool.contract.UnpackLog(event, "TickSpacing", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
