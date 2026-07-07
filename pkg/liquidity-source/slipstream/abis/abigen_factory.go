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

// FactoryMetaData contains all meta data concerning the Factory contract.
var FactoryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_voter\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_poolImplementation\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint24\",\"name\":\"oldUnstakedFee\",\"type\":\"uint24\"},{\"indexed\":true,\"internalType\":\"uint24\",\"name\":\"newUnstakedFee\",\"type\":\"uint24\"}],\"name\":\"DefaultUnstakedFeeChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnerChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"PoolCreated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldFeeManager\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newFeeManager\",\"type\":\"address\"}],\"name\":\"SwapFeeManagerChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldFeeModule\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newFeeModule\",\"type\":\"address\"}],\"name\":\"SwapFeeModuleChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"indexed\":true,\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"}],\"name\":\"TickSpacingEnabled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldFeeManager\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newFeeManager\",\"type\":\"address\"}],\"name\":\"UnstakedFeeManagerChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldFeeModule\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newFeeModule\",\"type\":\"address\"}],\"name\":\"UnstakedFeeModuleChanged\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allPools\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"allPoolsLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"}],\"name\":\"createPool\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"defaultUnstakedFee\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"}],\"name\":\"enableTickSpacing\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"factoryRegistry\",\"outputs\":[{\"internalType\":\"contractIFactoryRegistry\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"\",\"type\":\"int24\"}],\"name\":\"getPool\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"getSwapFee\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"getUnstakedFee\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"isPair\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"poolImplementation\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint24\",\"name\":\"_defaultUnstakedFee\",\"type\":\"uint24\"}],\"name\":\"setDefaultUnstakedFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"setOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_swapFeeManager\",\"type\":\"address\"}],\"name\":\"setSwapFeeManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_swapFeeModule\",\"type\":\"address\"}],\"name\":\"setSwapFeeModule\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_unstakedFeeManager\",\"type\":\"address\"}],\"name\":\"setUnstakedFeeManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_unstakedFeeModule\",\"type\":\"address\"}],\"name\":\"setUnstakedFeeModule\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"swapFeeManager\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"swapFeeModule\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int24\",\"name\":\"\",\"type\":\"int24\"}],\"name\":\"tickSpacingToFee\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"tickSpacings\",\"outputs\":[{\"internalType\":\"int24[]\",\"name\":\"\",\"type\":\"int24[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unstakedFeeManager\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unstakedFeeModule\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"voter\",\"outputs\":[{\"internalType\":\"contractIVoter\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// FactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use FactoryMetaData.ABI instead.
var FactoryABI = FactoryMetaData.ABI

// Factory is an auto generated Go binding around an Ethereum contract.
type Factory struct {
	FactoryCaller     // Read-only binding to the contract
	FactoryTransactor // Write-only binding to the contract
	FactoryFilterer   // Log filterer for contract events
}

// FactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type FactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type FactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type FactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type FactorySession struct {
	Contract     *Factory          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// FactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type FactoryCallerSession struct {
	Contract *FactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// FactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type FactoryTransactorSession struct {
	Contract     *FactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// FactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type FactoryRaw struct {
	Contract *Factory // Generic contract binding to access the raw methods on
}

// FactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type FactoryCallerRaw struct {
	Contract *FactoryCaller // Generic read-only contract binding to access the raw methods on
}

// FactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type FactoryTransactorRaw struct {
	Contract *FactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewFactory creates a new instance of Factory, bound to a specific deployed contract.
func NewFactory(address common.Address, backend bind.ContractBackend) (*Factory, error) {
	contract, err := bindFactory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Factory{FactoryCaller: FactoryCaller{contract: contract}, FactoryTransactor: FactoryTransactor{contract: contract}, FactoryFilterer: FactoryFilterer{contract: contract}}, nil
}

// NewFactoryCaller creates a new read-only instance of Factory, bound to a specific deployed contract.
func NewFactoryCaller(address common.Address, caller bind.ContractCaller) (*FactoryCaller, error) {
	contract, err := bindFactory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &FactoryCaller{contract: contract}, nil
}

// NewFactoryTransactor creates a new write-only instance of Factory, bound to a specific deployed contract.
func NewFactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*FactoryTransactor, error) {
	contract, err := bindFactory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &FactoryTransactor{contract: contract}, nil
}

// NewFactoryFilterer creates a new log filterer instance of Factory, bound to a specific deployed contract.
func NewFactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*FactoryFilterer, error) {
	contract, err := bindFactory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &FactoryFilterer{contract: contract}, nil
}

// bindFactory binds a generic wrapper to an already deployed contract.
func bindFactory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := FactoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Factory *FactoryRaw) Call(opts *bind.CallOpts, result *[]any, method string, params ...any) error {
	return _Factory.Contract.FactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Factory *FactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Factory.Contract.FactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Factory *FactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...any) (*types.Transaction, error) {
	return _Factory.Contract.FactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Factory *FactoryCallerRaw) Call(opts *bind.CallOpts, result *[]any, method string, params ...any) error {
	return _Factory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Factory *FactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Factory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Factory *FactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...any) (*types.Transaction, error) {
	return _Factory.Contract.contract.Transact(opts, method, params...)
}

// AllPools is a free data retrieval call binding the contract method 0x41d1de97.
//
// Solidity: function allPools(uint256 ) view returns(address)
func (_Factory *FactoryCaller) AllPools(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "allPools", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AllPools is a free data retrieval call binding the contract method 0x41d1de97.
//
// Solidity: function allPools(uint256 ) view returns(address)
func (_Factory *FactorySession) AllPools(arg0 *big.Int) (common.Address, error) {
	return _Factory.Contract.AllPools(&_Factory.CallOpts, arg0)
}

// AllPools is a free data retrieval call binding the contract method 0x41d1de97.
//
// Solidity: function allPools(uint256 ) view returns(address)
func (_Factory *FactoryCallerSession) AllPools(arg0 *big.Int) (common.Address, error) {
	return _Factory.Contract.AllPools(&_Factory.CallOpts, arg0)
}

// AllPoolsLength is a free data retrieval call binding the contract method 0xefde4e64.
//
// Solidity: function allPoolsLength() view returns(uint256)
func (_Factory *FactoryCaller) AllPoolsLength(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "allPoolsLength")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// AllPoolsLength is a free data retrieval call binding the contract method 0xefde4e64.
//
// Solidity: function allPoolsLength() view returns(uint256)
func (_Factory *FactorySession) AllPoolsLength() (*big.Int, error) {
	return _Factory.Contract.AllPoolsLength(&_Factory.CallOpts)
}

// AllPoolsLength is a free data retrieval call binding the contract method 0xefde4e64.
//
// Solidity: function allPoolsLength() view returns(uint256)
func (_Factory *FactoryCallerSession) AllPoolsLength() (*big.Int, error) {
	return _Factory.Contract.AllPoolsLength(&_Factory.CallOpts)
}

// DefaultUnstakedFee is a free data retrieval call binding the contract method 0xe2824832.
//
// Solidity: function defaultUnstakedFee() view returns(uint24)
func (_Factory *FactoryCaller) DefaultUnstakedFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "defaultUnstakedFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DefaultUnstakedFee is a free data retrieval call binding the contract method 0xe2824832.
//
// Solidity: function defaultUnstakedFee() view returns(uint24)
func (_Factory *FactorySession) DefaultUnstakedFee() (*big.Int, error) {
	return _Factory.Contract.DefaultUnstakedFee(&_Factory.CallOpts)
}

// DefaultUnstakedFee is a free data retrieval call binding the contract method 0xe2824832.
//
// Solidity: function defaultUnstakedFee() view returns(uint24)
func (_Factory *FactoryCallerSession) DefaultUnstakedFee() (*big.Int, error) {
	return _Factory.Contract.DefaultUnstakedFee(&_Factory.CallOpts)
}

// FactoryRegistry is a free data retrieval call binding the contract method 0x3bf0c9fb.
//
// Solidity: function factoryRegistry() view returns(address)
func (_Factory *FactoryCaller) FactoryRegistry(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "factoryRegistry")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FactoryRegistry is a free data retrieval call binding the contract method 0x3bf0c9fb.
//
// Solidity: function factoryRegistry() view returns(address)
func (_Factory *FactorySession) FactoryRegistry() (common.Address, error) {
	return _Factory.Contract.FactoryRegistry(&_Factory.CallOpts)
}

// FactoryRegistry is a free data retrieval call binding the contract method 0x3bf0c9fb.
//
// Solidity: function factoryRegistry() view returns(address)
func (_Factory *FactoryCallerSession) FactoryRegistry() (common.Address, error) {
	return _Factory.Contract.FactoryRegistry(&_Factory.CallOpts)
}

// GetPool is a free data retrieval call binding the contract method 0x28af8d0b.
//
// Solidity: function getPool(address , address , int24 ) view returns(address)
func (_Factory *FactoryCaller) GetPool(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "getPool", arg0, arg1, arg2)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetPool is a free data retrieval call binding the contract method 0x28af8d0b.
//
// Solidity: function getPool(address , address , int24 ) view returns(address)
func (_Factory *FactorySession) GetPool(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	return _Factory.Contract.GetPool(&_Factory.CallOpts, arg0, arg1, arg2)
}

// GetPool is a free data retrieval call binding the contract method 0x28af8d0b.
//
// Solidity: function getPool(address , address , int24 ) view returns(address)
func (_Factory *FactoryCallerSession) GetPool(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	return _Factory.Contract.GetPool(&_Factory.CallOpts, arg0, arg1, arg2)
}

// GetSwapFee is a free data retrieval call binding the contract method 0x35458dcc.
//
// Solidity: function getSwapFee(address pool) view returns(uint24)
func (_Factory *FactoryCaller) GetSwapFee(opts *bind.CallOpts, pool common.Address) (*big.Int, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "getSwapFee", pool)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetSwapFee is a free data retrieval call binding the contract method 0x35458dcc.
//
// Solidity: function getSwapFee(address pool) view returns(uint24)
func (_Factory *FactorySession) GetSwapFee(pool common.Address) (*big.Int, error) {
	return _Factory.Contract.GetSwapFee(&_Factory.CallOpts, pool)
}

// GetSwapFee is a free data retrieval call binding the contract method 0x35458dcc.
//
// Solidity: function getSwapFee(address pool) view returns(uint24)
func (_Factory *FactoryCallerSession) GetSwapFee(pool common.Address) (*big.Int, error) {
	return _Factory.Contract.GetSwapFee(&_Factory.CallOpts, pool)
}

// GetUnstakedFee is a free data retrieval call binding the contract method 0x48cf7a43.
//
// Solidity: function getUnstakedFee(address pool) view returns(uint24)
func (_Factory *FactoryCaller) GetUnstakedFee(opts *bind.CallOpts, pool common.Address) (*big.Int, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "getUnstakedFee", pool)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetUnstakedFee is a free data retrieval call binding the contract method 0x48cf7a43.
//
// Solidity: function getUnstakedFee(address pool) view returns(uint24)
func (_Factory *FactorySession) GetUnstakedFee(pool common.Address) (*big.Int, error) {
	return _Factory.Contract.GetUnstakedFee(&_Factory.CallOpts, pool)
}

// GetUnstakedFee is a free data retrieval call binding the contract method 0x48cf7a43.
//
// Solidity: function getUnstakedFee(address pool) view returns(uint24)
func (_Factory *FactoryCallerSession) GetUnstakedFee(pool common.Address) (*big.Int, error) {
	return _Factory.Contract.GetUnstakedFee(&_Factory.CallOpts, pool)
}

// IsPair is a free data retrieval call binding the contract method 0xe5e31b13.
//
// Solidity: function isPair(address pool) view returns(bool)
func (_Factory *FactoryCaller) IsPair(opts *bind.CallOpts, pool common.Address) (bool, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "isPair", pool)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsPair is a free data retrieval call binding the contract method 0xe5e31b13.
//
// Solidity: function isPair(address pool) view returns(bool)
func (_Factory *FactorySession) IsPair(pool common.Address) (bool, error) {
	return _Factory.Contract.IsPair(&_Factory.CallOpts, pool)
}

// IsPair is a free data retrieval call binding the contract method 0xe5e31b13.
//
// Solidity: function isPair(address pool) view returns(bool)
func (_Factory *FactoryCallerSession) IsPair(pool common.Address) (bool, error) {
	return _Factory.Contract.IsPair(&_Factory.CallOpts, pool)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Factory *FactoryCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Factory *FactorySession) Owner() (common.Address, error) {
	return _Factory.Contract.Owner(&_Factory.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Factory *FactoryCallerSession) Owner() (common.Address, error) {
	return _Factory.Contract.Owner(&_Factory.CallOpts)
}

// PoolImplementation is a free data retrieval call binding the contract method 0xcefa7799.
//
// Solidity: function poolImplementation() view returns(address)
func (_Factory *FactoryCaller) PoolImplementation(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "poolImplementation")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PoolImplementation is a free data retrieval call binding the contract method 0xcefa7799.
//
// Solidity: function poolImplementation() view returns(address)
func (_Factory *FactorySession) PoolImplementation() (common.Address, error) {
	return _Factory.Contract.PoolImplementation(&_Factory.CallOpts)
}

// PoolImplementation is a free data retrieval call binding the contract method 0xcefa7799.
//
// Solidity: function poolImplementation() view returns(address)
func (_Factory *FactoryCallerSession) PoolImplementation() (common.Address, error) {
	return _Factory.Contract.PoolImplementation(&_Factory.CallOpts)
}

// SwapFeeManager is a free data retrieval call binding the contract method 0xd574afa9.
//
// Solidity: function swapFeeManager() view returns(address)
func (_Factory *FactoryCaller) SwapFeeManager(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "swapFeeManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SwapFeeManager is a free data retrieval call binding the contract method 0xd574afa9.
//
// Solidity: function swapFeeManager() view returns(address)
func (_Factory *FactorySession) SwapFeeManager() (common.Address, error) {
	return _Factory.Contract.SwapFeeManager(&_Factory.CallOpts)
}

// SwapFeeManager is a free data retrieval call binding the contract method 0xd574afa9.
//
// Solidity: function swapFeeManager() view returns(address)
func (_Factory *FactoryCallerSession) SwapFeeManager() (common.Address, error) {
	return _Factory.Contract.SwapFeeManager(&_Factory.CallOpts)
}

// SwapFeeModule is a free data retrieval call binding the contract method 0x23c43a51.
//
// Solidity: function swapFeeModule() view returns(address)
func (_Factory *FactoryCaller) SwapFeeModule(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "swapFeeModule")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SwapFeeModule is a free data retrieval call binding the contract method 0x23c43a51.
//
// Solidity: function swapFeeModule() view returns(address)
func (_Factory *FactorySession) SwapFeeModule() (common.Address, error) {
	return _Factory.Contract.SwapFeeModule(&_Factory.CallOpts)
}

// SwapFeeModule is a free data retrieval call binding the contract method 0x23c43a51.
//
// Solidity: function swapFeeModule() view returns(address)
func (_Factory *FactoryCallerSession) SwapFeeModule() (common.Address, error) {
	return _Factory.Contract.SwapFeeModule(&_Factory.CallOpts)
}

// TickSpacingToFee is a free data retrieval call binding the contract method 0x380dc1c2.
//
// Solidity: function tickSpacingToFee(int24 ) view returns(uint24)
func (_Factory *FactoryCaller) TickSpacingToFee(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "tickSpacingToFee", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TickSpacingToFee is a free data retrieval call binding the contract method 0x380dc1c2.
//
// Solidity: function tickSpacingToFee(int24 ) view returns(uint24)
func (_Factory *FactorySession) TickSpacingToFee(arg0 *big.Int) (*big.Int, error) {
	return _Factory.Contract.TickSpacingToFee(&_Factory.CallOpts, arg0)
}

// TickSpacingToFee is a free data retrieval call binding the contract method 0x380dc1c2.
//
// Solidity: function tickSpacingToFee(int24 ) view returns(uint24)
func (_Factory *FactoryCallerSession) TickSpacingToFee(arg0 *big.Int) (*big.Int, error) {
	return _Factory.Contract.TickSpacingToFee(&_Factory.CallOpts, arg0)
}

// TickSpacings is a free data retrieval call binding the contract method 0x9cbbbe86.
//
// Solidity: function tickSpacings() view returns(int24[])
func (_Factory *FactoryCaller) TickSpacings(opts *bind.CallOpts) ([]*big.Int, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "tickSpacings")

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// TickSpacings is a free data retrieval call binding the contract method 0x9cbbbe86.
//
// Solidity: function tickSpacings() view returns(int24[])
func (_Factory *FactorySession) TickSpacings() ([]*big.Int, error) {
	return _Factory.Contract.TickSpacings(&_Factory.CallOpts)
}

// TickSpacings is a free data retrieval call binding the contract method 0x9cbbbe86.
//
// Solidity: function tickSpacings() view returns(int24[])
func (_Factory *FactoryCallerSession) TickSpacings() ([]*big.Int, error) {
	return _Factory.Contract.TickSpacings(&_Factory.CallOpts)
}

// UnstakedFeeManager is a free data retrieval call binding the contract method 0x82e189e0.
//
// Solidity: function unstakedFeeManager() view returns(address)
func (_Factory *FactoryCaller) UnstakedFeeManager(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "unstakedFeeManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// UnstakedFeeManager is a free data retrieval call binding the contract method 0x82e189e0.
//
// Solidity: function unstakedFeeManager() view returns(address)
func (_Factory *FactorySession) UnstakedFeeManager() (common.Address, error) {
	return _Factory.Contract.UnstakedFeeManager(&_Factory.CallOpts)
}

// UnstakedFeeManager is a free data retrieval call binding the contract method 0x82e189e0.
//
// Solidity: function unstakedFeeManager() view returns(address)
func (_Factory *FactoryCallerSession) UnstakedFeeManager() (common.Address, error) {
	return _Factory.Contract.UnstakedFeeManager(&_Factory.CallOpts)
}

// UnstakedFeeModule is a free data retrieval call binding the contract method 0x7693bc11.
//
// Solidity: function unstakedFeeModule() view returns(address)
func (_Factory *FactoryCaller) UnstakedFeeModule(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "unstakedFeeModule")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// UnstakedFeeModule is a free data retrieval call binding the contract method 0x7693bc11.
//
// Solidity: function unstakedFeeModule() view returns(address)
func (_Factory *FactorySession) UnstakedFeeModule() (common.Address, error) {
	return _Factory.Contract.UnstakedFeeModule(&_Factory.CallOpts)
}

// UnstakedFeeModule is a free data retrieval call binding the contract method 0x7693bc11.
//
// Solidity: function unstakedFeeModule() view returns(address)
func (_Factory *FactoryCallerSession) UnstakedFeeModule() (common.Address, error) {
	return _Factory.Contract.UnstakedFeeModule(&_Factory.CallOpts)
}

// Voter is a free data retrieval call binding the contract method 0x46c96aac.
//
// Solidity: function voter() view returns(address)
func (_Factory *FactoryCaller) Voter(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "voter")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Voter is a free data retrieval call binding the contract method 0x46c96aac.
//
// Solidity: function voter() view returns(address)
func (_Factory *FactorySession) Voter() (common.Address, error) {
	return _Factory.Contract.Voter(&_Factory.CallOpts)
}

// Voter is a free data retrieval call binding the contract method 0x46c96aac.
//
// Solidity: function voter() view returns(address)
func (_Factory *FactoryCallerSession) Voter() (common.Address, error) {
	return _Factory.Contract.Voter(&_Factory.CallOpts)
}

// CreatePool is a paid mutator transaction binding the contract method 0x232aa5ac.
//
// Solidity: function createPool(address tokenA, address tokenB, int24 tickSpacing, uint160 sqrtPriceX96) returns(address pool)
func (_Factory *FactoryTransactor) CreatePool(opts *bind.TransactOpts, tokenA common.Address, tokenB common.Address, tickSpacing *big.Int, sqrtPriceX96 *big.Int) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "createPool", tokenA, tokenB, tickSpacing, sqrtPriceX96)
}

// CreatePool is a paid mutator transaction binding the contract method 0x232aa5ac.
//
// Solidity: function createPool(address tokenA, address tokenB, int24 tickSpacing, uint160 sqrtPriceX96) returns(address pool)
func (_Factory *FactorySession) CreatePool(tokenA common.Address, tokenB common.Address, tickSpacing *big.Int, sqrtPriceX96 *big.Int) (*types.Transaction, error) {
	return _Factory.Contract.CreatePool(&_Factory.TransactOpts, tokenA, tokenB, tickSpacing, sqrtPriceX96)
}

// CreatePool is a paid mutator transaction binding the contract method 0x232aa5ac.
//
// Solidity: function createPool(address tokenA, address tokenB, int24 tickSpacing, uint160 sqrtPriceX96) returns(address pool)
func (_Factory *FactoryTransactorSession) CreatePool(tokenA common.Address, tokenB common.Address, tickSpacing *big.Int, sqrtPriceX96 *big.Int) (*types.Transaction, error) {
	return _Factory.Contract.CreatePool(&_Factory.TransactOpts, tokenA, tokenB, tickSpacing, sqrtPriceX96)
}

// EnableTickSpacing is a paid mutator transaction binding the contract method 0xeee0fdb4.
//
// Solidity: function enableTickSpacing(int24 tickSpacing, uint24 fee) returns()
func (_Factory *FactoryTransactor) EnableTickSpacing(opts *bind.TransactOpts, tickSpacing *big.Int, fee *big.Int) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "enableTickSpacing", tickSpacing, fee)
}

// EnableTickSpacing is a paid mutator transaction binding the contract method 0xeee0fdb4.
//
// Solidity: function enableTickSpacing(int24 tickSpacing, uint24 fee) returns()
func (_Factory *FactorySession) EnableTickSpacing(tickSpacing *big.Int, fee *big.Int) (*types.Transaction, error) {
	return _Factory.Contract.EnableTickSpacing(&_Factory.TransactOpts, tickSpacing, fee)
}

// EnableTickSpacing is a paid mutator transaction binding the contract method 0xeee0fdb4.
//
// Solidity: function enableTickSpacing(int24 tickSpacing, uint24 fee) returns()
func (_Factory *FactoryTransactorSession) EnableTickSpacing(tickSpacing *big.Int, fee *big.Int) (*types.Transaction, error) {
	return _Factory.Contract.EnableTickSpacing(&_Factory.TransactOpts, tickSpacing, fee)
}

// SetDefaultUnstakedFee is a paid mutator transaction binding the contract method 0xa2f97f42.
//
// Solidity: function setDefaultUnstakedFee(uint24 _defaultUnstakedFee) returns()
func (_Factory *FactoryTransactor) SetDefaultUnstakedFee(opts *bind.TransactOpts, _defaultUnstakedFee *big.Int) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setDefaultUnstakedFee", _defaultUnstakedFee)
}

// SetDefaultUnstakedFee is a paid mutator transaction binding the contract method 0xa2f97f42.
//
// Solidity: function setDefaultUnstakedFee(uint24 _defaultUnstakedFee) returns()
func (_Factory *FactorySession) SetDefaultUnstakedFee(_defaultUnstakedFee *big.Int) (*types.Transaction, error) {
	return _Factory.Contract.SetDefaultUnstakedFee(&_Factory.TransactOpts, _defaultUnstakedFee)
}

// SetDefaultUnstakedFee is a paid mutator transaction binding the contract method 0xa2f97f42.
//
// Solidity: function setDefaultUnstakedFee(uint24 _defaultUnstakedFee) returns()
func (_Factory *FactoryTransactorSession) SetDefaultUnstakedFee(_defaultUnstakedFee *big.Int) (*types.Transaction, error) {
	return _Factory.Contract.SetDefaultUnstakedFee(&_Factory.TransactOpts, _defaultUnstakedFee)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(address _owner) returns()
func (_Factory *FactoryTransactor) SetOwner(opts *bind.TransactOpts, _owner common.Address) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setOwner", _owner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(address _owner) returns()
func (_Factory *FactorySession) SetOwner(_owner common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetOwner(&_Factory.TransactOpts, _owner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(address _owner) returns()
func (_Factory *FactoryTransactorSession) SetOwner(_owner common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetOwner(&_Factory.TransactOpts, _owner)
}

// SetSwapFeeManager is a paid mutator transaction binding the contract method 0xffb4d9d1.
//
// Solidity: function setSwapFeeManager(address _swapFeeManager) returns()
func (_Factory *FactoryTransactor) SetSwapFeeManager(opts *bind.TransactOpts, _swapFeeManager common.Address) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setSwapFeeManager", _swapFeeManager)
}

// SetSwapFeeManager is a paid mutator transaction binding the contract method 0xffb4d9d1.
//
// Solidity: function setSwapFeeManager(address _swapFeeManager) returns()
func (_Factory *FactorySession) SetSwapFeeManager(_swapFeeManager common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetSwapFeeManager(&_Factory.TransactOpts, _swapFeeManager)
}

// SetSwapFeeManager is a paid mutator transaction binding the contract method 0xffb4d9d1.
//
// Solidity: function setSwapFeeManager(address _swapFeeManager) returns()
func (_Factory *FactoryTransactorSession) SetSwapFeeManager(_swapFeeManager common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetSwapFeeManager(&_Factory.TransactOpts, _swapFeeManager)
}

// SetSwapFeeModule is a paid mutator transaction binding the contract method 0x61b9c3ec.
//
// Solidity: function setSwapFeeModule(address _swapFeeModule) returns()
func (_Factory *FactoryTransactor) SetSwapFeeModule(opts *bind.TransactOpts, _swapFeeModule common.Address) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setSwapFeeModule", _swapFeeModule)
}

// SetSwapFeeModule is a paid mutator transaction binding the contract method 0x61b9c3ec.
//
// Solidity: function setSwapFeeModule(address _swapFeeModule) returns()
func (_Factory *FactorySession) SetSwapFeeModule(_swapFeeModule common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetSwapFeeModule(&_Factory.TransactOpts, _swapFeeModule)
}

// SetSwapFeeModule is a paid mutator transaction binding the contract method 0x61b9c3ec.
//
// Solidity: function setSwapFeeModule(address _swapFeeModule) returns()
func (_Factory *FactoryTransactorSession) SetSwapFeeModule(_swapFeeModule common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetSwapFeeModule(&_Factory.TransactOpts, _swapFeeModule)
}

// SetUnstakedFeeManager is a paid mutator transaction binding the contract method 0x93ce8627.
//
// Solidity: function setUnstakedFeeManager(address _unstakedFeeManager) returns()
func (_Factory *FactoryTransactor) SetUnstakedFeeManager(opts *bind.TransactOpts, _unstakedFeeManager common.Address) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setUnstakedFeeManager", _unstakedFeeManager)
}

// SetUnstakedFeeManager is a paid mutator transaction binding the contract method 0x93ce8627.
//
// Solidity: function setUnstakedFeeManager(address _unstakedFeeManager) returns()
func (_Factory *FactorySession) SetUnstakedFeeManager(_unstakedFeeManager common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetUnstakedFeeManager(&_Factory.TransactOpts, _unstakedFeeManager)
}

// SetUnstakedFeeManager is a paid mutator transaction binding the contract method 0x93ce8627.
//
// Solidity: function setUnstakedFeeManager(address _unstakedFeeManager) returns()
func (_Factory *FactoryTransactorSession) SetUnstakedFeeManager(_unstakedFeeManager common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetUnstakedFeeManager(&_Factory.TransactOpts, _unstakedFeeManager)
}

// SetUnstakedFeeModule is a paid mutator transaction binding the contract method 0x1b31d878.
//
// Solidity: function setUnstakedFeeModule(address _unstakedFeeModule) returns()
func (_Factory *FactoryTransactor) SetUnstakedFeeModule(opts *bind.TransactOpts, _unstakedFeeModule common.Address) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setUnstakedFeeModule", _unstakedFeeModule)
}

// SetUnstakedFeeModule is a paid mutator transaction binding the contract method 0x1b31d878.
//
// Solidity: function setUnstakedFeeModule(address _unstakedFeeModule) returns()
func (_Factory *FactorySession) SetUnstakedFeeModule(_unstakedFeeModule common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetUnstakedFeeModule(&_Factory.TransactOpts, _unstakedFeeModule)
}

// SetUnstakedFeeModule is a paid mutator transaction binding the contract method 0x1b31d878.
//
// Solidity: function setUnstakedFeeModule(address _unstakedFeeModule) returns()
func (_Factory *FactoryTransactorSession) SetUnstakedFeeModule(_unstakedFeeModule common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetUnstakedFeeModule(&_Factory.TransactOpts, _unstakedFeeModule)
}

// FactoryDefaultUnstakedFeeChangedIterator is returned from FilterDefaultUnstakedFeeChanged and is used to iterate over the raw logs and unpacked data for DefaultUnstakedFeeChanged events raised by the Factory contract.
type FactoryDefaultUnstakedFeeChangedIterator struct {
	Event *FactoryDefaultUnstakedFeeChanged // Event containing the contract specifics and raw log

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
func (it *FactoryDefaultUnstakedFeeChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryDefaultUnstakedFeeChanged)
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
		it.Event = new(FactoryDefaultUnstakedFeeChanged)
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
func (it *FactoryDefaultUnstakedFeeChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryDefaultUnstakedFeeChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryDefaultUnstakedFeeChanged represents a DefaultUnstakedFeeChanged event raised by the Factory contract.
type FactoryDefaultUnstakedFeeChanged struct {
	OldUnstakedFee *big.Int
	NewUnstakedFee *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterDefaultUnstakedFeeChanged is a free log retrieval operation binding the contract event 0xcbca61144322b913ada4febfb591864cad7617559d7ee0d3e29b48eb93fcc78e.
//
// Solidity: event DefaultUnstakedFeeChanged(uint24 indexed oldUnstakedFee, uint24 indexed newUnstakedFee)
func (_Factory *FactoryFilterer) FilterDefaultUnstakedFeeChanged(opts *bind.FilterOpts, oldUnstakedFee []*big.Int, newUnstakedFee []*big.Int) (*FactoryDefaultUnstakedFeeChangedIterator, error) {

	var oldUnstakedFeeRule []any
	for _, oldUnstakedFeeItem := range oldUnstakedFee {
		oldUnstakedFeeRule = append(oldUnstakedFeeRule, oldUnstakedFeeItem)
	}
	var newUnstakedFeeRule []any
	for _, newUnstakedFeeItem := range newUnstakedFee {
		newUnstakedFeeRule = append(newUnstakedFeeRule, newUnstakedFeeItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "DefaultUnstakedFeeChanged", oldUnstakedFeeRule, newUnstakedFeeRule)
	if err != nil {
		return nil, err
	}
	return &FactoryDefaultUnstakedFeeChangedIterator{contract: _Factory.contract, event: "DefaultUnstakedFeeChanged", logs: logs, sub: sub}, nil
}

// WatchDefaultUnstakedFeeChanged is a free log subscription operation binding the contract event 0xcbca61144322b913ada4febfb591864cad7617559d7ee0d3e29b48eb93fcc78e.
//
// Solidity: event DefaultUnstakedFeeChanged(uint24 indexed oldUnstakedFee, uint24 indexed newUnstakedFee)
func (_Factory *FactoryFilterer) WatchDefaultUnstakedFeeChanged(opts *bind.WatchOpts, sink chan<- *FactoryDefaultUnstakedFeeChanged, oldUnstakedFee []*big.Int, newUnstakedFee []*big.Int) (event.Subscription, error) {

	var oldUnstakedFeeRule []any
	for _, oldUnstakedFeeItem := range oldUnstakedFee {
		oldUnstakedFeeRule = append(oldUnstakedFeeRule, oldUnstakedFeeItem)
	}
	var newUnstakedFeeRule []any
	for _, newUnstakedFeeItem := range newUnstakedFee {
		newUnstakedFeeRule = append(newUnstakedFeeRule, newUnstakedFeeItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "DefaultUnstakedFeeChanged", oldUnstakedFeeRule, newUnstakedFeeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryDefaultUnstakedFeeChanged)
				if err := _Factory.contract.UnpackLog(event, "DefaultUnstakedFeeChanged", log); err != nil {
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

// ParseDefaultUnstakedFeeChanged is a log parse operation binding the contract event 0xcbca61144322b913ada4febfb591864cad7617559d7ee0d3e29b48eb93fcc78e.
//
// Solidity: event DefaultUnstakedFeeChanged(uint24 indexed oldUnstakedFee, uint24 indexed newUnstakedFee)
func (_Factory *FactoryFilterer) ParseDefaultUnstakedFeeChanged(log types.Log) (*FactoryDefaultUnstakedFeeChanged, error) {
	event := new(FactoryDefaultUnstakedFeeChanged)
	if err := _Factory.contract.UnpackLog(event, "DefaultUnstakedFeeChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryOwnerChangedIterator is returned from FilterOwnerChanged and is used to iterate over the raw logs and unpacked data for OwnerChanged events raised by the Factory contract.
type FactoryOwnerChangedIterator struct {
	Event *FactoryOwnerChanged // Event containing the contract specifics and raw log

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
func (it *FactoryOwnerChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryOwnerChanged)
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
		it.Event = new(FactoryOwnerChanged)
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
func (it *FactoryOwnerChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryOwnerChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryOwnerChanged represents a OwnerChanged event raised by the Factory contract.
type FactoryOwnerChanged struct {
	OldOwner common.Address
	NewOwner common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterOwnerChanged is a free log retrieval operation binding the contract event 0xb532073b38c83145e3e5135377a08bf9aab55bc0fd7c1179cd4fb995d2a5159c.
//
// Solidity: event OwnerChanged(address indexed oldOwner, address indexed newOwner)
func (_Factory *FactoryFilterer) FilterOwnerChanged(opts *bind.FilterOpts, oldOwner []common.Address, newOwner []common.Address) (*FactoryOwnerChangedIterator, error) {

	var oldOwnerRule []any
	for _, oldOwnerItem := range oldOwner {
		oldOwnerRule = append(oldOwnerRule, oldOwnerItem)
	}
	var newOwnerRule []any
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "OwnerChanged", oldOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &FactoryOwnerChangedIterator{contract: _Factory.contract, event: "OwnerChanged", logs: logs, sub: sub}, nil
}

// WatchOwnerChanged is a free log subscription operation binding the contract event 0xb532073b38c83145e3e5135377a08bf9aab55bc0fd7c1179cd4fb995d2a5159c.
//
// Solidity: event OwnerChanged(address indexed oldOwner, address indexed newOwner)
func (_Factory *FactoryFilterer) WatchOwnerChanged(opts *bind.WatchOpts, sink chan<- *FactoryOwnerChanged, oldOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var oldOwnerRule []any
	for _, oldOwnerItem := range oldOwner {
		oldOwnerRule = append(oldOwnerRule, oldOwnerItem)
	}
	var newOwnerRule []any
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "OwnerChanged", oldOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryOwnerChanged)
				if err := _Factory.contract.UnpackLog(event, "OwnerChanged", log); err != nil {
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

// ParseOwnerChanged is a log parse operation binding the contract event 0xb532073b38c83145e3e5135377a08bf9aab55bc0fd7c1179cd4fb995d2a5159c.
//
// Solidity: event OwnerChanged(address indexed oldOwner, address indexed newOwner)
func (_Factory *FactoryFilterer) ParseOwnerChanged(log types.Log) (*FactoryOwnerChanged, error) {
	event := new(FactoryOwnerChanged)
	if err := _Factory.contract.UnpackLog(event, "OwnerChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryPoolCreatedIterator is returned from FilterPoolCreated and is used to iterate over the raw logs and unpacked data for PoolCreated events raised by the Factory contract.
type FactoryPoolCreatedIterator struct {
	Event *FactoryPoolCreated // Event containing the contract specifics and raw log

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
func (it *FactoryPoolCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryPoolCreated)
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
		it.Event = new(FactoryPoolCreated)
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
func (it *FactoryPoolCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryPoolCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryPoolCreated represents a PoolCreated event raised by the Factory contract.
type FactoryPoolCreated struct {
	Token0      common.Address
	Token1      common.Address
	TickSpacing *big.Int
	Pool        common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterPoolCreated is a free log retrieval operation binding the contract event 0xab0d57f0df537bb25e80245ef7748fa62353808c54d6e528a9dd20887aed9ac2.
//
// Solidity: event PoolCreated(address indexed token0, address indexed token1, int24 indexed tickSpacing, address pool)
func (_Factory *FactoryFilterer) FilterPoolCreated(opts *bind.FilterOpts, token0 []common.Address, token1 []common.Address, tickSpacing []*big.Int) (*FactoryPoolCreatedIterator, error) {

	var token0Rule []any
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []any
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}
	var tickSpacingRule []any
	for _, tickSpacingItem := range tickSpacing {
		tickSpacingRule = append(tickSpacingRule, tickSpacingItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "PoolCreated", token0Rule, token1Rule, tickSpacingRule)
	if err != nil {
		return nil, err
	}
	return &FactoryPoolCreatedIterator{contract: _Factory.contract, event: "PoolCreated", logs: logs, sub: sub}, nil
}

// WatchPoolCreated is a free log subscription operation binding the contract event 0xab0d57f0df537bb25e80245ef7748fa62353808c54d6e528a9dd20887aed9ac2.
//
// Solidity: event PoolCreated(address indexed token0, address indexed token1, int24 indexed tickSpacing, address pool)
func (_Factory *FactoryFilterer) WatchPoolCreated(opts *bind.WatchOpts, sink chan<- *FactoryPoolCreated, token0 []common.Address, token1 []common.Address, tickSpacing []*big.Int) (event.Subscription, error) {

	var token0Rule []any
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []any
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}
	var tickSpacingRule []any
	for _, tickSpacingItem := range tickSpacing {
		tickSpacingRule = append(tickSpacingRule, tickSpacingItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "PoolCreated", token0Rule, token1Rule, tickSpacingRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryPoolCreated)
				if err := _Factory.contract.UnpackLog(event, "PoolCreated", log); err != nil {
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

// ParsePoolCreated is a log parse operation binding the contract event 0xab0d57f0df537bb25e80245ef7748fa62353808c54d6e528a9dd20887aed9ac2.
//
// Solidity: event PoolCreated(address indexed token0, address indexed token1, int24 indexed tickSpacing, address pool)
func (_Factory *FactoryFilterer) ParsePoolCreated(log types.Log) (*FactoryPoolCreated, error) {
	event := new(FactoryPoolCreated)
	if err := _Factory.contract.UnpackLog(event, "PoolCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactorySwapFeeManagerChangedIterator is returned from FilterSwapFeeManagerChanged and is used to iterate over the raw logs and unpacked data for SwapFeeManagerChanged events raised by the Factory contract.
type FactorySwapFeeManagerChangedIterator struct {
	Event *FactorySwapFeeManagerChanged // Event containing the contract specifics and raw log

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
func (it *FactorySwapFeeManagerChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactorySwapFeeManagerChanged)
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
		it.Event = new(FactorySwapFeeManagerChanged)
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
func (it *FactorySwapFeeManagerChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactorySwapFeeManagerChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactorySwapFeeManagerChanged represents a SwapFeeManagerChanged event raised by the Factory contract.
type FactorySwapFeeManagerChanged struct {
	OldFeeManager common.Address
	NewFeeManager common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterSwapFeeManagerChanged is a free log retrieval operation binding the contract event 0x7ae0007229b3333719d97e8ef5829c888f560776012974f87409c158e5b7eb91.
//
// Solidity: event SwapFeeManagerChanged(address indexed oldFeeManager, address indexed newFeeManager)
func (_Factory *FactoryFilterer) FilterSwapFeeManagerChanged(opts *bind.FilterOpts, oldFeeManager []common.Address, newFeeManager []common.Address) (*FactorySwapFeeManagerChangedIterator, error) {

	var oldFeeManagerRule []any
	for _, oldFeeManagerItem := range oldFeeManager {
		oldFeeManagerRule = append(oldFeeManagerRule, oldFeeManagerItem)
	}
	var newFeeManagerRule []any
	for _, newFeeManagerItem := range newFeeManager {
		newFeeManagerRule = append(newFeeManagerRule, newFeeManagerItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "SwapFeeManagerChanged", oldFeeManagerRule, newFeeManagerRule)
	if err != nil {
		return nil, err
	}
	return &FactorySwapFeeManagerChangedIterator{contract: _Factory.contract, event: "SwapFeeManagerChanged", logs: logs, sub: sub}, nil
}

// WatchSwapFeeManagerChanged is a free log subscription operation binding the contract event 0x7ae0007229b3333719d97e8ef5829c888f560776012974f87409c158e5b7eb91.
//
// Solidity: event SwapFeeManagerChanged(address indexed oldFeeManager, address indexed newFeeManager)
func (_Factory *FactoryFilterer) WatchSwapFeeManagerChanged(opts *bind.WatchOpts, sink chan<- *FactorySwapFeeManagerChanged, oldFeeManager []common.Address, newFeeManager []common.Address) (event.Subscription, error) {

	var oldFeeManagerRule []any
	for _, oldFeeManagerItem := range oldFeeManager {
		oldFeeManagerRule = append(oldFeeManagerRule, oldFeeManagerItem)
	}
	var newFeeManagerRule []any
	for _, newFeeManagerItem := range newFeeManager {
		newFeeManagerRule = append(newFeeManagerRule, newFeeManagerItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "SwapFeeManagerChanged", oldFeeManagerRule, newFeeManagerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactorySwapFeeManagerChanged)
				if err := _Factory.contract.UnpackLog(event, "SwapFeeManagerChanged", log); err != nil {
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

// ParseSwapFeeManagerChanged is a log parse operation binding the contract event 0x7ae0007229b3333719d97e8ef5829c888f560776012974f87409c158e5b7eb91.
//
// Solidity: event SwapFeeManagerChanged(address indexed oldFeeManager, address indexed newFeeManager)
func (_Factory *FactoryFilterer) ParseSwapFeeManagerChanged(log types.Log) (*FactorySwapFeeManagerChanged, error) {
	event := new(FactorySwapFeeManagerChanged)
	if err := _Factory.contract.UnpackLog(event, "SwapFeeManagerChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactorySwapFeeModuleChangedIterator is returned from FilterSwapFeeModuleChanged and is used to iterate over the raw logs and unpacked data for SwapFeeModuleChanged events raised by the Factory contract.
type FactorySwapFeeModuleChangedIterator struct {
	Event *FactorySwapFeeModuleChanged // Event containing the contract specifics and raw log

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
func (it *FactorySwapFeeModuleChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactorySwapFeeModuleChanged)
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
		it.Event = new(FactorySwapFeeModuleChanged)
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
func (it *FactorySwapFeeModuleChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactorySwapFeeModuleChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactorySwapFeeModuleChanged represents a SwapFeeModuleChanged event raised by the Factory contract.
type FactorySwapFeeModuleChanged struct {
	OldFeeModule common.Address
	NewFeeModule common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterSwapFeeModuleChanged is a free log retrieval operation binding the contract event 0xdf24ed64a7bcd761cf1132e79f94ea269a1d570e7a6ca0ab99a8f5ccd6f5022f.
//
// Solidity: event SwapFeeModuleChanged(address indexed oldFeeModule, address indexed newFeeModule)
func (_Factory *FactoryFilterer) FilterSwapFeeModuleChanged(opts *bind.FilterOpts, oldFeeModule []common.Address, newFeeModule []common.Address) (*FactorySwapFeeModuleChangedIterator, error) {

	var oldFeeModuleRule []any
	for _, oldFeeModuleItem := range oldFeeModule {
		oldFeeModuleRule = append(oldFeeModuleRule, oldFeeModuleItem)
	}
	var newFeeModuleRule []any
	for _, newFeeModuleItem := range newFeeModule {
		newFeeModuleRule = append(newFeeModuleRule, newFeeModuleItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "SwapFeeModuleChanged", oldFeeModuleRule, newFeeModuleRule)
	if err != nil {
		return nil, err
	}
	return &FactorySwapFeeModuleChangedIterator{contract: _Factory.contract, event: "SwapFeeModuleChanged", logs: logs, sub: sub}, nil
}

// WatchSwapFeeModuleChanged is a free log subscription operation binding the contract event 0xdf24ed64a7bcd761cf1132e79f94ea269a1d570e7a6ca0ab99a8f5ccd6f5022f.
//
// Solidity: event SwapFeeModuleChanged(address indexed oldFeeModule, address indexed newFeeModule)
func (_Factory *FactoryFilterer) WatchSwapFeeModuleChanged(opts *bind.WatchOpts, sink chan<- *FactorySwapFeeModuleChanged, oldFeeModule []common.Address, newFeeModule []common.Address) (event.Subscription, error) {

	var oldFeeModuleRule []any
	for _, oldFeeModuleItem := range oldFeeModule {
		oldFeeModuleRule = append(oldFeeModuleRule, oldFeeModuleItem)
	}
	var newFeeModuleRule []any
	for _, newFeeModuleItem := range newFeeModule {
		newFeeModuleRule = append(newFeeModuleRule, newFeeModuleItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "SwapFeeModuleChanged", oldFeeModuleRule, newFeeModuleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactorySwapFeeModuleChanged)
				if err := _Factory.contract.UnpackLog(event, "SwapFeeModuleChanged", log); err != nil {
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

// ParseSwapFeeModuleChanged is a log parse operation binding the contract event 0xdf24ed64a7bcd761cf1132e79f94ea269a1d570e7a6ca0ab99a8f5ccd6f5022f.
//
// Solidity: event SwapFeeModuleChanged(address indexed oldFeeModule, address indexed newFeeModule)
func (_Factory *FactoryFilterer) ParseSwapFeeModuleChanged(log types.Log) (*FactorySwapFeeModuleChanged, error) {
	event := new(FactorySwapFeeModuleChanged)
	if err := _Factory.contract.UnpackLog(event, "SwapFeeModuleChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryTickSpacingEnabledIterator is returned from FilterTickSpacingEnabled and is used to iterate over the raw logs and unpacked data for TickSpacingEnabled events raised by the Factory contract.
type FactoryTickSpacingEnabledIterator struct {
	Event *FactoryTickSpacingEnabled // Event containing the contract specifics and raw log

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
func (it *FactoryTickSpacingEnabledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryTickSpacingEnabled)
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
		it.Event = new(FactoryTickSpacingEnabled)
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
func (it *FactoryTickSpacingEnabledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryTickSpacingEnabledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryTickSpacingEnabled represents a TickSpacingEnabled event raised by the Factory contract.
type FactoryTickSpacingEnabled struct {
	TickSpacing *big.Int
	Fee         *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterTickSpacingEnabled is a free log retrieval operation binding the contract event 0xebafae466a4a780a1d87f5fab2f52fad33be9151a7f69d099e8934c8de85b747.
//
// Solidity: event TickSpacingEnabled(int24 indexed tickSpacing, uint24 indexed fee)
func (_Factory *FactoryFilterer) FilterTickSpacingEnabled(opts *bind.FilterOpts, tickSpacing []*big.Int, fee []*big.Int) (*FactoryTickSpacingEnabledIterator, error) {

	var tickSpacingRule []any
	for _, tickSpacingItem := range tickSpacing {
		tickSpacingRule = append(tickSpacingRule, tickSpacingItem)
	}
	var feeRule []any
	for _, feeItem := range fee {
		feeRule = append(feeRule, feeItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "TickSpacingEnabled", tickSpacingRule, feeRule)
	if err != nil {
		return nil, err
	}
	return &FactoryTickSpacingEnabledIterator{contract: _Factory.contract, event: "TickSpacingEnabled", logs: logs, sub: sub}, nil
}

// WatchTickSpacingEnabled is a free log subscription operation binding the contract event 0xebafae466a4a780a1d87f5fab2f52fad33be9151a7f69d099e8934c8de85b747.
//
// Solidity: event TickSpacingEnabled(int24 indexed tickSpacing, uint24 indexed fee)
func (_Factory *FactoryFilterer) WatchTickSpacingEnabled(opts *bind.WatchOpts, sink chan<- *FactoryTickSpacingEnabled, tickSpacing []*big.Int, fee []*big.Int) (event.Subscription, error) {

	var tickSpacingRule []any
	for _, tickSpacingItem := range tickSpacing {
		tickSpacingRule = append(tickSpacingRule, tickSpacingItem)
	}
	var feeRule []any
	for _, feeItem := range fee {
		feeRule = append(feeRule, feeItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "TickSpacingEnabled", tickSpacingRule, feeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryTickSpacingEnabled)
				if err := _Factory.contract.UnpackLog(event, "TickSpacingEnabled", log); err != nil {
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

// ParseTickSpacingEnabled is a log parse operation binding the contract event 0xebafae466a4a780a1d87f5fab2f52fad33be9151a7f69d099e8934c8de85b747.
//
// Solidity: event TickSpacingEnabled(int24 indexed tickSpacing, uint24 indexed fee)
func (_Factory *FactoryFilterer) ParseTickSpacingEnabled(log types.Log) (*FactoryTickSpacingEnabled, error) {
	event := new(FactoryTickSpacingEnabled)
	if err := _Factory.contract.UnpackLog(event, "TickSpacingEnabled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryUnstakedFeeManagerChangedIterator is returned from FilterUnstakedFeeManagerChanged and is used to iterate over the raw logs and unpacked data for UnstakedFeeManagerChanged events raised by the Factory contract.
type FactoryUnstakedFeeManagerChangedIterator struct {
	Event *FactoryUnstakedFeeManagerChanged // Event containing the contract specifics and raw log

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
func (it *FactoryUnstakedFeeManagerChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryUnstakedFeeManagerChanged)
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
		it.Event = new(FactoryUnstakedFeeManagerChanged)
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
func (it *FactoryUnstakedFeeManagerChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryUnstakedFeeManagerChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryUnstakedFeeManagerChanged represents a UnstakedFeeManagerChanged event raised by the Factory contract.
type FactoryUnstakedFeeManagerChanged struct {
	OldFeeManager common.Address
	NewFeeManager common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterUnstakedFeeManagerChanged is a free log retrieval operation binding the contract event 0x3d7ebe96182c99643ca0c997a416a2a3409baab225f85f50c29fcf0591c820c1.
//
// Solidity: event UnstakedFeeManagerChanged(address indexed oldFeeManager, address indexed newFeeManager)
func (_Factory *FactoryFilterer) FilterUnstakedFeeManagerChanged(opts *bind.FilterOpts, oldFeeManager []common.Address, newFeeManager []common.Address) (*FactoryUnstakedFeeManagerChangedIterator, error) {

	var oldFeeManagerRule []any
	for _, oldFeeManagerItem := range oldFeeManager {
		oldFeeManagerRule = append(oldFeeManagerRule, oldFeeManagerItem)
	}
	var newFeeManagerRule []any
	for _, newFeeManagerItem := range newFeeManager {
		newFeeManagerRule = append(newFeeManagerRule, newFeeManagerItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "UnstakedFeeManagerChanged", oldFeeManagerRule, newFeeManagerRule)
	if err != nil {
		return nil, err
	}
	return &FactoryUnstakedFeeManagerChangedIterator{contract: _Factory.contract, event: "UnstakedFeeManagerChanged", logs: logs, sub: sub}, nil
}

// WatchUnstakedFeeManagerChanged is a free log subscription operation binding the contract event 0x3d7ebe96182c99643ca0c997a416a2a3409baab225f85f50c29fcf0591c820c1.
//
// Solidity: event UnstakedFeeManagerChanged(address indexed oldFeeManager, address indexed newFeeManager)
func (_Factory *FactoryFilterer) WatchUnstakedFeeManagerChanged(opts *bind.WatchOpts, sink chan<- *FactoryUnstakedFeeManagerChanged, oldFeeManager []common.Address, newFeeManager []common.Address) (event.Subscription, error) {

	var oldFeeManagerRule []any
	for _, oldFeeManagerItem := range oldFeeManager {
		oldFeeManagerRule = append(oldFeeManagerRule, oldFeeManagerItem)
	}
	var newFeeManagerRule []any
	for _, newFeeManagerItem := range newFeeManager {
		newFeeManagerRule = append(newFeeManagerRule, newFeeManagerItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "UnstakedFeeManagerChanged", oldFeeManagerRule, newFeeManagerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryUnstakedFeeManagerChanged)
				if err := _Factory.contract.UnpackLog(event, "UnstakedFeeManagerChanged", log); err != nil {
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

// ParseUnstakedFeeManagerChanged is a log parse operation binding the contract event 0x3d7ebe96182c99643ca0c997a416a2a3409baab225f85f50c29fcf0591c820c1.
//
// Solidity: event UnstakedFeeManagerChanged(address indexed oldFeeManager, address indexed newFeeManager)
func (_Factory *FactoryFilterer) ParseUnstakedFeeManagerChanged(log types.Log) (*FactoryUnstakedFeeManagerChanged, error) {
	event := new(FactoryUnstakedFeeManagerChanged)
	if err := _Factory.contract.UnpackLog(event, "UnstakedFeeManagerChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryUnstakedFeeModuleChangedIterator is returned from FilterUnstakedFeeModuleChanged and is used to iterate over the raw logs and unpacked data for UnstakedFeeModuleChanged events raised by the Factory contract.
type FactoryUnstakedFeeModuleChangedIterator struct {
	Event *FactoryUnstakedFeeModuleChanged // Event containing the contract specifics and raw log

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
func (it *FactoryUnstakedFeeModuleChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryUnstakedFeeModuleChanged)
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
		it.Event = new(FactoryUnstakedFeeModuleChanged)
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
func (it *FactoryUnstakedFeeModuleChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryUnstakedFeeModuleChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryUnstakedFeeModuleChanged represents a UnstakedFeeModuleChanged event raised by the Factory contract.
type FactoryUnstakedFeeModuleChanged struct {
	OldFeeModule common.Address
	NewFeeModule common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterUnstakedFeeModuleChanged is a free log retrieval operation binding the contract event 0x6520f404f3831947cee8673060459cdfb181b7332aa7580bcce9bf90ef1f0e20.
//
// Solidity: event UnstakedFeeModuleChanged(address indexed oldFeeModule, address indexed newFeeModule)
func (_Factory *FactoryFilterer) FilterUnstakedFeeModuleChanged(opts *bind.FilterOpts, oldFeeModule []common.Address, newFeeModule []common.Address) (*FactoryUnstakedFeeModuleChangedIterator, error) {

	var oldFeeModuleRule []any
	for _, oldFeeModuleItem := range oldFeeModule {
		oldFeeModuleRule = append(oldFeeModuleRule, oldFeeModuleItem)
	}
	var newFeeModuleRule []any
	for _, newFeeModuleItem := range newFeeModule {
		newFeeModuleRule = append(newFeeModuleRule, newFeeModuleItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "UnstakedFeeModuleChanged", oldFeeModuleRule, newFeeModuleRule)
	if err != nil {
		return nil, err
	}
	return &FactoryUnstakedFeeModuleChangedIterator{contract: _Factory.contract, event: "UnstakedFeeModuleChanged", logs: logs, sub: sub}, nil
}

// WatchUnstakedFeeModuleChanged is a free log subscription operation binding the contract event 0x6520f404f3831947cee8673060459cdfb181b7332aa7580bcce9bf90ef1f0e20.
//
// Solidity: event UnstakedFeeModuleChanged(address indexed oldFeeModule, address indexed newFeeModule)
func (_Factory *FactoryFilterer) WatchUnstakedFeeModuleChanged(opts *bind.WatchOpts, sink chan<- *FactoryUnstakedFeeModuleChanged, oldFeeModule []common.Address, newFeeModule []common.Address) (event.Subscription, error) {

	var oldFeeModuleRule []any
	for _, oldFeeModuleItem := range oldFeeModule {
		oldFeeModuleRule = append(oldFeeModuleRule, oldFeeModuleItem)
	}
	var newFeeModuleRule []any
	for _, newFeeModuleItem := range newFeeModule {
		newFeeModuleRule = append(newFeeModuleRule, newFeeModuleItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "UnstakedFeeModuleChanged", oldFeeModuleRule, newFeeModuleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryUnstakedFeeModuleChanged)
				if err := _Factory.contract.UnpackLog(event, "UnstakedFeeModuleChanged", log); err != nil {
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

// ParseUnstakedFeeModuleChanged is a log parse operation binding the contract event 0x6520f404f3831947cee8673060459cdfb181b7332aa7580bcce9bf90ef1f0e20.
//
// Solidity: event UnstakedFeeModuleChanged(address indexed oldFeeModule, address indexed newFeeModule)
func (_Factory *FactoryFilterer) ParseUnstakedFeeModuleChanged(log types.Log) (*FactoryUnstakedFeeModuleChanged, error) {
	event := new(FactoryUnstakedFeeModuleChanged)
	if err := _Factory.contract.UnpackLog(event, "UnstakedFeeModuleChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
