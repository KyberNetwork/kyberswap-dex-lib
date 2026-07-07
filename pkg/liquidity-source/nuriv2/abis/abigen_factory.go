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
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"}],\"name\":\"FeeAmountEnabled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldFeeCollector\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newFeeCollector\",\"type\":\"address\"}],\"name\":\"FeeCollectorChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldSetter\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newSetter\",\"type\":\"address\"}],\"name\":\"FeeSetterChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldImplementation\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newImplementation\",\"type\":\"address\"}],\"name\":\"ImplementationChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnerChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"PoolCreated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"feeProtocol0Old\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"feeProtocol1Old\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"feeProtocol0New\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"feeProtocol1New\",\"type\":\"uint8\"}],\"name\":\"SetFeeProtocol\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"feeProtocol0Old\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"feeProtocol1Old\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"feeProtocol0New\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"feeProtocol1New\",\"type\":\"uint8\"}],\"name\":\"SetPoolFeeProtocol\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"POOL_INIT_CODE_HASH\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"}],\"name\":\"createPool\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"}],\"name\":\"enableFeeAmount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint24\",\"name\":\"\",\"type\":\"uint24\"}],\"name\":\"feeAmountTickSpacing\",\"outputs\":[{\"internalType\":\"int24\",\"name\":\"\",\"type\":\"int24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"feeCollector\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"feeProtocol\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"feeSetter\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"\",\"type\":\"uint24\"}],\"name\":\"getPool\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"implementation\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_nfpManager\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_veRam\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_voter\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_implementation\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nfpManager\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"poolFeeProtocol\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"__poolFeeProtocol\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_pool\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"_fee\",\"type\":\"uint24\"}],\"name\":\"setFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_feeCollector\",\"type\":\"address\"}],\"name\":\"setFeeCollector\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"_feeProtocol\",\"type\":\"uint8\"}],\"name\":\"setFeeProtocol\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newFeeSetter\",\"type\":\"address\"}],\"name\":\"setFeeSetter\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_implementation\",\"type\":\"address\"}],\"name\":\"setImplementation\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"setOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"feeProtocol0\",\"type\":\"uint8\"},{\"internalType\":\"uint8\",\"name\":\"feeProtocol1\",\"type\":\"uint8\"}],\"name\":\"setPoolFeeProtocol\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"veRam\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"voter\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
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

// POOLINITCODEHASH is a free data retrieval call binding the contract method 0xdc6fd8ab.
//
// Solidity: function POOL_INIT_CODE_HASH() view returns(bytes32)
func (_Factory *FactoryCaller) POOLINITCODEHASH(opts *bind.CallOpts) ([32]byte, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "POOL_INIT_CODE_HASH")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// POOLINITCODEHASH is a free data retrieval call binding the contract method 0xdc6fd8ab.
//
// Solidity: function POOL_INIT_CODE_HASH() view returns(bytes32)
func (_Factory *FactorySession) POOLINITCODEHASH() ([32]byte, error) {
	return _Factory.Contract.POOLINITCODEHASH(&_Factory.CallOpts)
}

// POOLINITCODEHASH is a free data retrieval call binding the contract method 0xdc6fd8ab.
//
// Solidity: function POOL_INIT_CODE_HASH() view returns(bytes32)
func (_Factory *FactoryCallerSession) POOLINITCODEHASH() ([32]byte, error) {
	return _Factory.Contract.POOLINITCODEHASH(&_Factory.CallOpts)
}

// FeeAmountTickSpacing is a free data retrieval call binding the contract method 0x22afcccb.
//
// Solidity: function feeAmountTickSpacing(uint24 ) view returns(int24)
func (_Factory *FactoryCaller) FeeAmountTickSpacing(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "feeAmountTickSpacing", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FeeAmountTickSpacing is a free data retrieval call binding the contract method 0x22afcccb.
//
// Solidity: function feeAmountTickSpacing(uint24 ) view returns(int24)
func (_Factory *FactorySession) FeeAmountTickSpacing(arg0 *big.Int) (*big.Int, error) {
	return _Factory.Contract.FeeAmountTickSpacing(&_Factory.CallOpts, arg0)
}

// FeeAmountTickSpacing is a free data retrieval call binding the contract method 0x22afcccb.
//
// Solidity: function feeAmountTickSpacing(uint24 ) view returns(int24)
func (_Factory *FactoryCallerSession) FeeAmountTickSpacing(arg0 *big.Int) (*big.Int, error) {
	return _Factory.Contract.FeeAmountTickSpacing(&_Factory.CallOpts, arg0)
}

// FeeCollector is a free data retrieval call binding the contract method 0xc415b95c.
//
// Solidity: function feeCollector() view returns(address)
func (_Factory *FactoryCaller) FeeCollector(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "feeCollector")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FeeCollector is a free data retrieval call binding the contract method 0xc415b95c.
//
// Solidity: function feeCollector() view returns(address)
func (_Factory *FactorySession) FeeCollector() (common.Address, error) {
	return _Factory.Contract.FeeCollector(&_Factory.CallOpts)
}

// FeeCollector is a free data retrieval call binding the contract method 0xc415b95c.
//
// Solidity: function feeCollector() view returns(address)
func (_Factory *FactoryCallerSession) FeeCollector() (common.Address, error) {
	return _Factory.Contract.FeeCollector(&_Factory.CallOpts)
}

// FeeProtocol is a free data retrieval call binding the contract method 0x527eb4bc.
//
// Solidity: function feeProtocol() view returns(uint8)
func (_Factory *FactoryCaller) FeeProtocol(opts *bind.CallOpts) (uint8, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "feeProtocol")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// FeeProtocol is a free data retrieval call binding the contract method 0x527eb4bc.
//
// Solidity: function feeProtocol() view returns(uint8)
func (_Factory *FactorySession) FeeProtocol() (uint8, error) {
	return _Factory.Contract.FeeProtocol(&_Factory.CallOpts)
}

// FeeProtocol is a free data retrieval call binding the contract method 0x527eb4bc.
//
// Solidity: function feeProtocol() view returns(uint8)
func (_Factory *FactoryCallerSession) FeeProtocol() (uint8, error) {
	return _Factory.Contract.FeeProtocol(&_Factory.CallOpts)
}

// FeeSetter is a free data retrieval call binding the contract method 0x87cf3ef4.
//
// Solidity: function feeSetter() view returns(address)
func (_Factory *FactoryCaller) FeeSetter(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "feeSetter")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FeeSetter is a free data retrieval call binding the contract method 0x87cf3ef4.
//
// Solidity: function feeSetter() view returns(address)
func (_Factory *FactorySession) FeeSetter() (common.Address, error) {
	return _Factory.Contract.FeeSetter(&_Factory.CallOpts)
}

// FeeSetter is a free data retrieval call binding the contract method 0x87cf3ef4.
//
// Solidity: function feeSetter() view returns(address)
func (_Factory *FactoryCallerSession) FeeSetter() (common.Address, error) {
	return _Factory.Contract.FeeSetter(&_Factory.CallOpts)
}

// GetPool is a free data retrieval call binding the contract method 0x1698ee82.
//
// Solidity: function getPool(address , address , uint24 ) view returns(address)
func (_Factory *FactoryCaller) GetPool(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "getPool", arg0, arg1, arg2)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetPool is a free data retrieval call binding the contract method 0x1698ee82.
//
// Solidity: function getPool(address , address , uint24 ) view returns(address)
func (_Factory *FactorySession) GetPool(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	return _Factory.Contract.GetPool(&_Factory.CallOpts, arg0, arg1, arg2)
}

// GetPool is a free data retrieval call binding the contract method 0x1698ee82.
//
// Solidity: function getPool(address , address , uint24 ) view returns(address)
func (_Factory *FactoryCallerSession) GetPool(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	return _Factory.Contract.GetPool(&_Factory.CallOpts, arg0, arg1, arg2)
}

// Implementation is a free data retrieval call binding the contract method 0x5c60da1b.
//
// Solidity: function implementation() view returns(address)
func (_Factory *FactoryCaller) Implementation(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "implementation")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Implementation is a free data retrieval call binding the contract method 0x5c60da1b.
//
// Solidity: function implementation() view returns(address)
func (_Factory *FactorySession) Implementation() (common.Address, error) {
	return _Factory.Contract.Implementation(&_Factory.CallOpts)
}

// Implementation is a free data retrieval call binding the contract method 0x5c60da1b.
//
// Solidity: function implementation() view returns(address)
func (_Factory *FactoryCallerSession) Implementation() (common.Address, error) {
	return _Factory.Contract.Implementation(&_Factory.CallOpts)
}

// NfpManager is a free data retrieval call binding the contract method 0x98bbc3c7.
//
// Solidity: function nfpManager() view returns(address)
func (_Factory *FactoryCaller) NfpManager(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "nfpManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NfpManager is a free data retrieval call binding the contract method 0x98bbc3c7.
//
// Solidity: function nfpManager() view returns(address)
func (_Factory *FactorySession) NfpManager() (common.Address, error) {
	return _Factory.Contract.NfpManager(&_Factory.CallOpts)
}

// NfpManager is a free data retrieval call binding the contract method 0x98bbc3c7.
//
// Solidity: function nfpManager() view returns(address)
func (_Factory *FactoryCallerSession) NfpManager() (common.Address, error) {
	return _Factory.Contract.NfpManager(&_Factory.CallOpts)
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

// PoolFeeProtocol is a free data retrieval call binding the contract method 0xebb0d9f7.
//
// Solidity: function poolFeeProtocol(address pool) view returns(uint8 __poolFeeProtocol)
func (_Factory *FactoryCaller) PoolFeeProtocol(opts *bind.CallOpts, pool common.Address) (uint8, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "poolFeeProtocol", pool)

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// PoolFeeProtocol is a free data retrieval call binding the contract method 0xebb0d9f7.
//
// Solidity: function poolFeeProtocol(address pool) view returns(uint8 __poolFeeProtocol)
func (_Factory *FactorySession) PoolFeeProtocol(pool common.Address) (uint8, error) {
	return _Factory.Contract.PoolFeeProtocol(&_Factory.CallOpts, pool)
}

// PoolFeeProtocol is a free data retrieval call binding the contract method 0xebb0d9f7.
//
// Solidity: function poolFeeProtocol(address pool) view returns(uint8 __poolFeeProtocol)
func (_Factory *FactoryCallerSession) PoolFeeProtocol(pool common.Address) (uint8, error) {
	return _Factory.Contract.PoolFeeProtocol(&_Factory.CallOpts, pool)
}

// VeRam is a free data retrieval call binding the contract method 0x97e9dc31.
//
// Solidity: function veRam() view returns(address)
func (_Factory *FactoryCaller) VeRam(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "veRam")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// VeRam is a free data retrieval call binding the contract method 0x97e9dc31.
//
// Solidity: function veRam() view returns(address)
func (_Factory *FactorySession) VeRam() (common.Address, error) {
	return _Factory.Contract.VeRam(&_Factory.CallOpts)
}

// VeRam is a free data retrieval call binding the contract method 0x97e9dc31.
//
// Solidity: function veRam() view returns(address)
func (_Factory *FactoryCallerSession) VeRam() (common.Address, error) {
	return _Factory.Contract.VeRam(&_Factory.CallOpts)
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

// CreatePool is a paid mutator transaction binding the contract method 0xa1671295.
//
// Solidity: function createPool(address tokenA, address tokenB, uint24 fee) returns(address pool)
func (_Factory *FactoryTransactor) CreatePool(opts *bind.TransactOpts, tokenA common.Address, tokenB common.Address, fee *big.Int) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "createPool", tokenA, tokenB, fee)
}

// CreatePool is a paid mutator transaction binding the contract method 0xa1671295.
//
// Solidity: function createPool(address tokenA, address tokenB, uint24 fee) returns(address pool)
func (_Factory *FactorySession) CreatePool(tokenA common.Address, tokenB common.Address, fee *big.Int) (*types.Transaction, error) {
	return _Factory.Contract.CreatePool(&_Factory.TransactOpts, tokenA, tokenB, fee)
}

// CreatePool is a paid mutator transaction binding the contract method 0xa1671295.
//
// Solidity: function createPool(address tokenA, address tokenB, uint24 fee) returns(address pool)
func (_Factory *FactoryTransactorSession) CreatePool(tokenA common.Address, tokenB common.Address, fee *big.Int) (*types.Transaction, error) {
	return _Factory.Contract.CreatePool(&_Factory.TransactOpts, tokenA, tokenB, fee)
}

// EnableFeeAmount is a paid mutator transaction binding the contract method 0x8a7c195f.
//
// Solidity: function enableFeeAmount(uint24 fee, int24 tickSpacing) returns()
func (_Factory *FactoryTransactor) EnableFeeAmount(opts *bind.TransactOpts, fee *big.Int, tickSpacing *big.Int) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "enableFeeAmount", fee, tickSpacing)
}

// EnableFeeAmount is a paid mutator transaction binding the contract method 0x8a7c195f.
//
// Solidity: function enableFeeAmount(uint24 fee, int24 tickSpacing) returns()
func (_Factory *FactorySession) EnableFeeAmount(fee *big.Int, tickSpacing *big.Int) (*types.Transaction, error) {
	return _Factory.Contract.EnableFeeAmount(&_Factory.TransactOpts, fee, tickSpacing)
}

// EnableFeeAmount is a paid mutator transaction binding the contract method 0x8a7c195f.
//
// Solidity: function enableFeeAmount(uint24 fee, int24 tickSpacing) returns()
func (_Factory *FactoryTransactorSession) EnableFeeAmount(fee *big.Int, tickSpacing *big.Int) (*types.Transaction, error) {
	return _Factory.Contract.EnableFeeAmount(&_Factory.TransactOpts, fee, tickSpacing)
}

// Initialize is a paid mutator transaction binding the contract method 0xf8c8765e.
//
// Solidity: function initialize(address _nfpManager, address _veRam, address _voter, address _implementation) returns()
func (_Factory *FactoryTransactor) Initialize(opts *bind.TransactOpts, _nfpManager common.Address, _veRam common.Address, _voter common.Address, _implementation common.Address) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "initialize", _nfpManager, _veRam, _voter, _implementation)
}

// Initialize is a paid mutator transaction binding the contract method 0xf8c8765e.
//
// Solidity: function initialize(address _nfpManager, address _veRam, address _voter, address _implementation) returns()
func (_Factory *FactorySession) Initialize(_nfpManager common.Address, _veRam common.Address, _voter common.Address, _implementation common.Address) (*types.Transaction, error) {
	return _Factory.Contract.Initialize(&_Factory.TransactOpts, _nfpManager, _veRam, _voter, _implementation)
}

// Initialize is a paid mutator transaction binding the contract method 0xf8c8765e.
//
// Solidity: function initialize(address _nfpManager, address _veRam, address _voter, address _implementation) returns()
func (_Factory *FactoryTransactorSession) Initialize(_nfpManager common.Address, _veRam common.Address, _voter common.Address, _implementation common.Address) (*types.Transaction, error) {
	return _Factory.Contract.Initialize(&_Factory.TransactOpts, _nfpManager, _veRam, _voter, _implementation)
}

// SetFee is a paid mutator transaction binding the contract method 0xba364c3d.
//
// Solidity: function setFee(address _pool, uint24 _fee) returns()
func (_Factory *FactoryTransactor) SetFee(opts *bind.TransactOpts, _pool common.Address, _fee *big.Int) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setFee", _pool, _fee)
}

// SetFee is a paid mutator transaction binding the contract method 0xba364c3d.
//
// Solidity: function setFee(address _pool, uint24 _fee) returns()
func (_Factory *FactorySession) SetFee(_pool common.Address, _fee *big.Int) (*types.Transaction, error) {
	return _Factory.Contract.SetFee(&_Factory.TransactOpts, _pool, _fee)
}

// SetFee is a paid mutator transaction binding the contract method 0xba364c3d.
//
// Solidity: function setFee(address _pool, uint24 _fee) returns()
func (_Factory *FactoryTransactorSession) SetFee(_pool common.Address, _fee *big.Int) (*types.Transaction, error) {
	return _Factory.Contract.SetFee(&_Factory.TransactOpts, _pool, _fee)
}

// SetFeeCollector is a paid mutator transaction binding the contract method 0xa42dce80.
//
// Solidity: function setFeeCollector(address _feeCollector) returns()
func (_Factory *FactoryTransactor) SetFeeCollector(opts *bind.TransactOpts, _feeCollector common.Address) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setFeeCollector", _feeCollector)
}

// SetFeeCollector is a paid mutator transaction binding the contract method 0xa42dce80.
//
// Solidity: function setFeeCollector(address _feeCollector) returns()
func (_Factory *FactorySession) SetFeeCollector(_feeCollector common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetFeeCollector(&_Factory.TransactOpts, _feeCollector)
}

// SetFeeCollector is a paid mutator transaction binding the contract method 0xa42dce80.
//
// Solidity: function setFeeCollector(address _feeCollector) returns()
func (_Factory *FactoryTransactorSession) SetFeeCollector(_feeCollector common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetFeeCollector(&_Factory.TransactOpts, _feeCollector)
}

// SetFeeProtocol is a paid mutator transaction binding the contract method 0xb613a141.
//
// Solidity: function setFeeProtocol(uint8 _feeProtocol) returns()
func (_Factory *FactoryTransactor) SetFeeProtocol(opts *bind.TransactOpts, _feeProtocol uint8) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setFeeProtocol", _feeProtocol)
}

// SetFeeProtocol is a paid mutator transaction binding the contract method 0xb613a141.
//
// Solidity: function setFeeProtocol(uint8 _feeProtocol) returns()
func (_Factory *FactorySession) SetFeeProtocol(_feeProtocol uint8) (*types.Transaction, error) {
	return _Factory.Contract.SetFeeProtocol(&_Factory.TransactOpts, _feeProtocol)
}

// SetFeeProtocol is a paid mutator transaction binding the contract method 0xb613a141.
//
// Solidity: function setFeeProtocol(uint8 _feeProtocol) returns()
func (_Factory *FactoryTransactorSession) SetFeeProtocol(_feeProtocol uint8) (*types.Transaction, error) {
	return _Factory.Contract.SetFeeProtocol(&_Factory.TransactOpts, _feeProtocol)
}

// SetFeeSetter is a paid mutator transaction binding the contract method 0xb19805af.
//
// Solidity: function setFeeSetter(address _newFeeSetter) returns()
func (_Factory *FactoryTransactor) SetFeeSetter(opts *bind.TransactOpts, _newFeeSetter common.Address) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setFeeSetter", _newFeeSetter)
}

// SetFeeSetter is a paid mutator transaction binding the contract method 0xb19805af.
//
// Solidity: function setFeeSetter(address _newFeeSetter) returns()
func (_Factory *FactorySession) SetFeeSetter(_newFeeSetter common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetFeeSetter(&_Factory.TransactOpts, _newFeeSetter)
}

// SetFeeSetter is a paid mutator transaction binding the contract method 0xb19805af.
//
// Solidity: function setFeeSetter(address _newFeeSetter) returns()
func (_Factory *FactoryTransactorSession) SetFeeSetter(_newFeeSetter common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetFeeSetter(&_Factory.TransactOpts, _newFeeSetter)
}

// SetImplementation is a paid mutator transaction binding the contract method 0xd784d426.
//
// Solidity: function setImplementation(address _implementation) returns()
func (_Factory *FactoryTransactor) SetImplementation(opts *bind.TransactOpts, _implementation common.Address) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setImplementation", _implementation)
}

// SetImplementation is a paid mutator transaction binding the contract method 0xd784d426.
//
// Solidity: function setImplementation(address _implementation) returns()
func (_Factory *FactorySession) SetImplementation(_implementation common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetImplementation(&_Factory.TransactOpts, _implementation)
}

// SetImplementation is a paid mutator transaction binding the contract method 0xd784d426.
//
// Solidity: function setImplementation(address _implementation) returns()
func (_Factory *FactoryTransactorSession) SetImplementation(_implementation common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetImplementation(&_Factory.TransactOpts, _implementation)
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

// SetPoolFeeProtocol is a paid mutator transaction binding the contract method 0x8a703fa2.
//
// Solidity: function setPoolFeeProtocol(address pool, uint8 feeProtocol0, uint8 feeProtocol1) returns()
func (_Factory *FactoryTransactor) SetPoolFeeProtocol(opts *bind.TransactOpts, pool common.Address, feeProtocol0 uint8, feeProtocol1 uint8) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setPoolFeeProtocol", pool, feeProtocol0, feeProtocol1)
}

// SetPoolFeeProtocol is a paid mutator transaction binding the contract method 0x8a703fa2.
//
// Solidity: function setPoolFeeProtocol(address pool, uint8 feeProtocol0, uint8 feeProtocol1) returns()
func (_Factory *FactorySession) SetPoolFeeProtocol(pool common.Address, feeProtocol0 uint8, feeProtocol1 uint8) (*types.Transaction, error) {
	return _Factory.Contract.SetPoolFeeProtocol(&_Factory.TransactOpts, pool, feeProtocol0, feeProtocol1)
}

// SetPoolFeeProtocol is a paid mutator transaction binding the contract method 0x8a703fa2.
//
// Solidity: function setPoolFeeProtocol(address pool, uint8 feeProtocol0, uint8 feeProtocol1) returns()
func (_Factory *FactoryTransactorSession) SetPoolFeeProtocol(pool common.Address, feeProtocol0 uint8, feeProtocol1 uint8) (*types.Transaction, error) {
	return _Factory.Contract.SetPoolFeeProtocol(&_Factory.TransactOpts, pool, feeProtocol0, feeProtocol1)
}

// FactoryFeeAmountEnabledIterator is returned from FilterFeeAmountEnabled and is used to iterate over the raw logs and unpacked data for FeeAmountEnabled events raised by the Factory contract.
type FactoryFeeAmountEnabledIterator struct {
	Event *FactoryFeeAmountEnabled // Event containing the contract specifics and raw log

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
func (it *FactoryFeeAmountEnabledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryFeeAmountEnabled)
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
		it.Event = new(FactoryFeeAmountEnabled)
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
func (it *FactoryFeeAmountEnabledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryFeeAmountEnabledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryFeeAmountEnabled represents a FeeAmountEnabled event raised by the Factory contract.
type FactoryFeeAmountEnabled struct {
	Fee         *big.Int
	TickSpacing *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterFeeAmountEnabled is a free log retrieval operation binding the contract event 0xc66a3fdf07232cdd185febcc6579d408c241b47ae2f9907d84be655141eeaecc.
//
// Solidity: event FeeAmountEnabled(uint24 indexed fee, int24 indexed tickSpacing)
func (_Factory *FactoryFilterer) FilterFeeAmountEnabled(opts *bind.FilterOpts, fee []*big.Int, tickSpacing []*big.Int) (*FactoryFeeAmountEnabledIterator, error) {

	var feeRule []any
	for _, feeItem := range fee {
		feeRule = append(feeRule, feeItem)
	}
	var tickSpacingRule []any
	for _, tickSpacingItem := range tickSpacing {
		tickSpacingRule = append(tickSpacingRule, tickSpacingItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "FeeAmountEnabled", feeRule, tickSpacingRule)
	if err != nil {
		return nil, err
	}
	return &FactoryFeeAmountEnabledIterator{contract: _Factory.contract, event: "FeeAmountEnabled", logs: logs, sub: sub}, nil
}

// WatchFeeAmountEnabled is a free log subscription operation binding the contract event 0xc66a3fdf07232cdd185febcc6579d408c241b47ae2f9907d84be655141eeaecc.
//
// Solidity: event FeeAmountEnabled(uint24 indexed fee, int24 indexed tickSpacing)
func (_Factory *FactoryFilterer) WatchFeeAmountEnabled(opts *bind.WatchOpts, sink chan<- *FactoryFeeAmountEnabled, fee []*big.Int, tickSpacing []*big.Int) (event.Subscription, error) {

	var feeRule []any
	for _, feeItem := range fee {
		feeRule = append(feeRule, feeItem)
	}
	var tickSpacingRule []any
	for _, tickSpacingItem := range tickSpacing {
		tickSpacingRule = append(tickSpacingRule, tickSpacingItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "FeeAmountEnabled", feeRule, tickSpacingRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryFeeAmountEnabled)
				if err := _Factory.contract.UnpackLog(event, "FeeAmountEnabled", log); err != nil {
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

// ParseFeeAmountEnabled is a log parse operation binding the contract event 0xc66a3fdf07232cdd185febcc6579d408c241b47ae2f9907d84be655141eeaecc.
//
// Solidity: event FeeAmountEnabled(uint24 indexed fee, int24 indexed tickSpacing)
func (_Factory *FactoryFilterer) ParseFeeAmountEnabled(log types.Log) (*FactoryFeeAmountEnabled, error) {
	event := new(FactoryFeeAmountEnabled)
	if err := _Factory.contract.UnpackLog(event, "FeeAmountEnabled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryFeeCollectorChangedIterator is returned from FilterFeeCollectorChanged and is used to iterate over the raw logs and unpacked data for FeeCollectorChanged events raised by the Factory contract.
type FactoryFeeCollectorChangedIterator struct {
	Event *FactoryFeeCollectorChanged // Event containing the contract specifics and raw log

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
func (it *FactoryFeeCollectorChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryFeeCollectorChanged)
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
		it.Event = new(FactoryFeeCollectorChanged)
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
func (it *FactoryFeeCollectorChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryFeeCollectorChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryFeeCollectorChanged represents a FeeCollectorChanged event raised by the Factory contract.
type FactoryFeeCollectorChanged struct {
	OldFeeCollector common.Address
	NewFeeCollector common.Address
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterFeeCollectorChanged is a free log retrieval operation binding the contract event 0x649c5e3d0ed183894196148e193af316452b0037e77d2ff0fef23b7dc722bed0.
//
// Solidity: event FeeCollectorChanged(address indexed oldFeeCollector, address indexed newFeeCollector)
func (_Factory *FactoryFilterer) FilterFeeCollectorChanged(opts *bind.FilterOpts, oldFeeCollector []common.Address, newFeeCollector []common.Address) (*FactoryFeeCollectorChangedIterator, error) {

	var oldFeeCollectorRule []any
	for _, oldFeeCollectorItem := range oldFeeCollector {
		oldFeeCollectorRule = append(oldFeeCollectorRule, oldFeeCollectorItem)
	}
	var newFeeCollectorRule []any
	for _, newFeeCollectorItem := range newFeeCollector {
		newFeeCollectorRule = append(newFeeCollectorRule, newFeeCollectorItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "FeeCollectorChanged", oldFeeCollectorRule, newFeeCollectorRule)
	if err != nil {
		return nil, err
	}
	return &FactoryFeeCollectorChangedIterator{contract: _Factory.contract, event: "FeeCollectorChanged", logs: logs, sub: sub}, nil
}

// WatchFeeCollectorChanged is a free log subscription operation binding the contract event 0x649c5e3d0ed183894196148e193af316452b0037e77d2ff0fef23b7dc722bed0.
//
// Solidity: event FeeCollectorChanged(address indexed oldFeeCollector, address indexed newFeeCollector)
func (_Factory *FactoryFilterer) WatchFeeCollectorChanged(opts *bind.WatchOpts, sink chan<- *FactoryFeeCollectorChanged, oldFeeCollector []common.Address, newFeeCollector []common.Address) (event.Subscription, error) {

	var oldFeeCollectorRule []any
	for _, oldFeeCollectorItem := range oldFeeCollector {
		oldFeeCollectorRule = append(oldFeeCollectorRule, oldFeeCollectorItem)
	}
	var newFeeCollectorRule []any
	for _, newFeeCollectorItem := range newFeeCollector {
		newFeeCollectorRule = append(newFeeCollectorRule, newFeeCollectorItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "FeeCollectorChanged", oldFeeCollectorRule, newFeeCollectorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryFeeCollectorChanged)
				if err := _Factory.contract.UnpackLog(event, "FeeCollectorChanged", log); err != nil {
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

// ParseFeeCollectorChanged is a log parse operation binding the contract event 0x649c5e3d0ed183894196148e193af316452b0037e77d2ff0fef23b7dc722bed0.
//
// Solidity: event FeeCollectorChanged(address indexed oldFeeCollector, address indexed newFeeCollector)
func (_Factory *FactoryFilterer) ParseFeeCollectorChanged(log types.Log) (*FactoryFeeCollectorChanged, error) {
	event := new(FactoryFeeCollectorChanged)
	if err := _Factory.contract.UnpackLog(event, "FeeCollectorChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryFeeSetterChangedIterator is returned from FilterFeeSetterChanged and is used to iterate over the raw logs and unpacked data for FeeSetterChanged events raised by the Factory contract.
type FactoryFeeSetterChangedIterator struct {
	Event *FactoryFeeSetterChanged // Event containing the contract specifics and raw log

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
func (it *FactoryFeeSetterChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryFeeSetterChanged)
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
		it.Event = new(FactoryFeeSetterChanged)
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
func (it *FactoryFeeSetterChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryFeeSetterChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryFeeSetterChanged represents a FeeSetterChanged event raised by the Factory contract.
type FactoryFeeSetterChanged struct {
	OldSetter common.Address
	NewSetter common.Address
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterFeeSetterChanged is a free log retrieval operation binding the contract event 0x774b126b94b3cc801460a024dd575406c3ebf27affd7c36198a53ac6655f056d.
//
// Solidity: event FeeSetterChanged(address indexed oldSetter, address indexed newSetter)
func (_Factory *FactoryFilterer) FilterFeeSetterChanged(opts *bind.FilterOpts, oldSetter []common.Address, newSetter []common.Address) (*FactoryFeeSetterChangedIterator, error) {

	var oldSetterRule []any
	for _, oldSetterItem := range oldSetter {
		oldSetterRule = append(oldSetterRule, oldSetterItem)
	}
	var newSetterRule []any
	for _, newSetterItem := range newSetter {
		newSetterRule = append(newSetterRule, newSetterItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "FeeSetterChanged", oldSetterRule, newSetterRule)
	if err != nil {
		return nil, err
	}
	return &FactoryFeeSetterChangedIterator{contract: _Factory.contract, event: "FeeSetterChanged", logs: logs, sub: sub}, nil
}

// WatchFeeSetterChanged is a free log subscription operation binding the contract event 0x774b126b94b3cc801460a024dd575406c3ebf27affd7c36198a53ac6655f056d.
//
// Solidity: event FeeSetterChanged(address indexed oldSetter, address indexed newSetter)
func (_Factory *FactoryFilterer) WatchFeeSetterChanged(opts *bind.WatchOpts, sink chan<- *FactoryFeeSetterChanged, oldSetter []common.Address, newSetter []common.Address) (event.Subscription, error) {

	var oldSetterRule []any
	for _, oldSetterItem := range oldSetter {
		oldSetterRule = append(oldSetterRule, oldSetterItem)
	}
	var newSetterRule []any
	for _, newSetterItem := range newSetter {
		newSetterRule = append(newSetterRule, newSetterItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "FeeSetterChanged", oldSetterRule, newSetterRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryFeeSetterChanged)
				if err := _Factory.contract.UnpackLog(event, "FeeSetterChanged", log); err != nil {
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

// ParseFeeSetterChanged is a log parse operation binding the contract event 0x774b126b94b3cc801460a024dd575406c3ebf27affd7c36198a53ac6655f056d.
//
// Solidity: event FeeSetterChanged(address indexed oldSetter, address indexed newSetter)
func (_Factory *FactoryFilterer) ParseFeeSetterChanged(log types.Log) (*FactoryFeeSetterChanged, error) {
	event := new(FactoryFeeSetterChanged)
	if err := _Factory.contract.UnpackLog(event, "FeeSetterChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryImplementationChangedIterator is returned from FilterImplementationChanged and is used to iterate over the raw logs and unpacked data for ImplementationChanged events raised by the Factory contract.
type FactoryImplementationChangedIterator struct {
	Event *FactoryImplementationChanged // Event containing the contract specifics and raw log

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
func (it *FactoryImplementationChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryImplementationChanged)
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
		it.Event = new(FactoryImplementationChanged)
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
func (it *FactoryImplementationChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryImplementationChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryImplementationChanged represents a ImplementationChanged event raised by the Factory contract.
type FactoryImplementationChanged struct {
	OldImplementation common.Address
	NewImplementation common.Address
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterImplementationChanged is a free log retrieval operation binding the contract event 0xcfbf4028add9318bbf716f08c348595afb063b0e9feed1f86d33681a4b3ed4d3.
//
// Solidity: event ImplementationChanged(address indexed oldImplementation, address indexed newImplementation)
func (_Factory *FactoryFilterer) FilterImplementationChanged(opts *bind.FilterOpts, oldImplementation []common.Address, newImplementation []common.Address) (*FactoryImplementationChangedIterator, error) {

	var oldImplementationRule []any
	for _, oldImplementationItem := range oldImplementation {
		oldImplementationRule = append(oldImplementationRule, oldImplementationItem)
	}
	var newImplementationRule []any
	for _, newImplementationItem := range newImplementation {
		newImplementationRule = append(newImplementationRule, newImplementationItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "ImplementationChanged", oldImplementationRule, newImplementationRule)
	if err != nil {
		return nil, err
	}
	return &FactoryImplementationChangedIterator{contract: _Factory.contract, event: "ImplementationChanged", logs: logs, sub: sub}, nil
}

// WatchImplementationChanged is a free log subscription operation binding the contract event 0xcfbf4028add9318bbf716f08c348595afb063b0e9feed1f86d33681a4b3ed4d3.
//
// Solidity: event ImplementationChanged(address indexed oldImplementation, address indexed newImplementation)
func (_Factory *FactoryFilterer) WatchImplementationChanged(opts *bind.WatchOpts, sink chan<- *FactoryImplementationChanged, oldImplementation []common.Address, newImplementation []common.Address) (event.Subscription, error) {

	var oldImplementationRule []any
	for _, oldImplementationItem := range oldImplementation {
		oldImplementationRule = append(oldImplementationRule, oldImplementationItem)
	}
	var newImplementationRule []any
	for _, newImplementationItem := range newImplementation {
		newImplementationRule = append(newImplementationRule, newImplementationItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "ImplementationChanged", oldImplementationRule, newImplementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryImplementationChanged)
				if err := _Factory.contract.UnpackLog(event, "ImplementationChanged", log); err != nil {
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

// ParseImplementationChanged is a log parse operation binding the contract event 0xcfbf4028add9318bbf716f08c348595afb063b0e9feed1f86d33681a4b3ed4d3.
//
// Solidity: event ImplementationChanged(address indexed oldImplementation, address indexed newImplementation)
func (_Factory *FactoryFilterer) ParseImplementationChanged(log types.Log) (*FactoryImplementationChanged, error) {
	event := new(FactoryImplementationChanged)
	if err := _Factory.contract.UnpackLog(event, "ImplementationChanged", log); err != nil {
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
	Fee         *big.Int
	TickSpacing *big.Int
	Pool        common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterPoolCreated is a free log retrieval operation binding the contract event 0x783cca1c0412dd0d695e784568c96da2e9c22ff989357a2e8b1d9b2b4e6b7118.
//
// Solidity: event PoolCreated(address indexed token0, address indexed token1, uint24 indexed fee, int24 tickSpacing, address pool)
func (_Factory *FactoryFilterer) FilterPoolCreated(opts *bind.FilterOpts, token0 []common.Address, token1 []common.Address, fee []*big.Int) (*FactoryPoolCreatedIterator, error) {

	var token0Rule []any
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []any
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}
	var feeRule []any
	for _, feeItem := range fee {
		feeRule = append(feeRule, feeItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "PoolCreated", token0Rule, token1Rule, feeRule)
	if err != nil {
		return nil, err
	}
	return &FactoryPoolCreatedIterator{contract: _Factory.contract, event: "PoolCreated", logs: logs, sub: sub}, nil
}

// WatchPoolCreated is a free log subscription operation binding the contract event 0x783cca1c0412dd0d695e784568c96da2e9c22ff989357a2e8b1d9b2b4e6b7118.
//
// Solidity: event PoolCreated(address indexed token0, address indexed token1, uint24 indexed fee, int24 tickSpacing, address pool)
func (_Factory *FactoryFilterer) WatchPoolCreated(opts *bind.WatchOpts, sink chan<- *FactoryPoolCreated, token0 []common.Address, token1 []common.Address, fee []*big.Int) (event.Subscription, error) {

	var token0Rule []any
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []any
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}
	var feeRule []any
	for _, feeItem := range fee {
		feeRule = append(feeRule, feeItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "PoolCreated", token0Rule, token1Rule, feeRule)
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

// ParsePoolCreated is a log parse operation binding the contract event 0x783cca1c0412dd0d695e784568c96da2e9c22ff989357a2e8b1d9b2b4e6b7118.
//
// Solidity: event PoolCreated(address indexed token0, address indexed token1, uint24 indexed fee, int24 tickSpacing, address pool)
func (_Factory *FactoryFilterer) ParsePoolCreated(log types.Log) (*FactoryPoolCreated, error) {
	event := new(FactoryPoolCreated)
	if err := _Factory.contract.UnpackLog(event, "PoolCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactorySetFeeProtocolIterator is returned from FilterSetFeeProtocol and is used to iterate over the raw logs and unpacked data for SetFeeProtocol events raised by the Factory contract.
type FactorySetFeeProtocolIterator struct {
	Event *FactorySetFeeProtocol // Event containing the contract specifics and raw log

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
func (it *FactorySetFeeProtocolIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactorySetFeeProtocol)
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
		it.Event = new(FactorySetFeeProtocol)
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
func (it *FactorySetFeeProtocolIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactorySetFeeProtocolIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactorySetFeeProtocol represents a SetFeeProtocol event raised by the Factory contract.
type FactorySetFeeProtocol struct {
	FeeProtocol0Old uint8
	FeeProtocol1Old uint8
	FeeProtocol0New uint8
	FeeProtocol1New uint8
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterSetFeeProtocol is a free log retrieval operation binding the contract event 0x973d8d92bb299f4af6ce49b52a8adb85ae46b9f214c4c4fc06ac77401237b133.
//
// Solidity: event SetFeeProtocol(uint8 feeProtocol0Old, uint8 feeProtocol1Old, uint8 feeProtocol0New, uint8 feeProtocol1New)
func (_Factory *FactoryFilterer) FilterSetFeeProtocol(opts *bind.FilterOpts) (*FactorySetFeeProtocolIterator, error) {

	logs, sub, err := _Factory.contract.FilterLogs(opts, "SetFeeProtocol")
	if err != nil {
		return nil, err
	}
	return &FactorySetFeeProtocolIterator{contract: _Factory.contract, event: "SetFeeProtocol", logs: logs, sub: sub}, nil
}

// WatchSetFeeProtocol is a free log subscription operation binding the contract event 0x973d8d92bb299f4af6ce49b52a8adb85ae46b9f214c4c4fc06ac77401237b133.
//
// Solidity: event SetFeeProtocol(uint8 feeProtocol0Old, uint8 feeProtocol1Old, uint8 feeProtocol0New, uint8 feeProtocol1New)
func (_Factory *FactoryFilterer) WatchSetFeeProtocol(opts *bind.WatchOpts, sink chan<- *FactorySetFeeProtocol) (event.Subscription, error) {

	logs, sub, err := _Factory.contract.WatchLogs(opts, "SetFeeProtocol")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactorySetFeeProtocol)
				if err := _Factory.contract.UnpackLog(event, "SetFeeProtocol", log); err != nil {
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

// ParseSetFeeProtocol is a log parse operation binding the contract event 0x973d8d92bb299f4af6ce49b52a8adb85ae46b9f214c4c4fc06ac77401237b133.
//
// Solidity: event SetFeeProtocol(uint8 feeProtocol0Old, uint8 feeProtocol1Old, uint8 feeProtocol0New, uint8 feeProtocol1New)
func (_Factory *FactoryFilterer) ParseSetFeeProtocol(log types.Log) (*FactorySetFeeProtocol, error) {
	event := new(FactorySetFeeProtocol)
	if err := _Factory.contract.UnpackLog(event, "SetFeeProtocol", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactorySetPoolFeeProtocolIterator is returned from FilterSetPoolFeeProtocol and is used to iterate over the raw logs and unpacked data for SetPoolFeeProtocol events raised by the Factory contract.
type FactorySetPoolFeeProtocolIterator struct {
	Event *FactorySetPoolFeeProtocol // Event containing the contract specifics and raw log

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
func (it *FactorySetPoolFeeProtocolIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactorySetPoolFeeProtocol)
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
		it.Event = new(FactorySetPoolFeeProtocol)
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
func (it *FactorySetPoolFeeProtocolIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactorySetPoolFeeProtocolIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactorySetPoolFeeProtocol represents a SetPoolFeeProtocol event raised by the Factory contract.
type FactorySetPoolFeeProtocol struct {
	Pool            common.Address
	FeeProtocol0Old uint8
	FeeProtocol1Old uint8
	FeeProtocol0New uint8
	FeeProtocol1New uint8
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterSetPoolFeeProtocol is a free log retrieval operation binding the contract event 0xc79f8f26ea41a4b5cdad3c4ba9a1c7e86474a1f3a1fb31a80e1112122cb4ec4d.
//
// Solidity: event SetPoolFeeProtocol(address pool, uint8 feeProtocol0Old, uint8 feeProtocol1Old, uint8 feeProtocol0New, uint8 feeProtocol1New)
func (_Factory *FactoryFilterer) FilterSetPoolFeeProtocol(opts *bind.FilterOpts) (*FactorySetPoolFeeProtocolIterator, error) {

	logs, sub, err := _Factory.contract.FilterLogs(opts, "SetPoolFeeProtocol")
	if err != nil {
		return nil, err
	}
	return &FactorySetPoolFeeProtocolIterator{contract: _Factory.contract, event: "SetPoolFeeProtocol", logs: logs, sub: sub}, nil
}

// WatchSetPoolFeeProtocol is a free log subscription operation binding the contract event 0xc79f8f26ea41a4b5cdad3c4ba9a1c7e86474a1f3a1fb31a80e1112122cb4ec4d.
//
// Solidity: event SetPoolFeeProtocol(address pool, uint8 feeProtocol0Old, uint8 feeProtocol1Old, uint8 feeProtocol0New, uint8 feeProtocol1New)
func (_Factory *FactoryFilterer) WatchSetPoolFeeProtocol(opts *bind.WatchOpts, sink chan<- *FactorySetPoolFeeProtocol) (event.Subscription, error) {

	logs, sub, err := _Factory.contract.WatchLogs(opts, "SetPoolFeeProtocol")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactorySetPoolFeeProtocol)
				if err := _Factory.contract.UnpackLog(event, "SetPoolFeeProtocol", log); err != nil {
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

// ParseSetPoolFeeProtocol is a log parse operation binding the contract event 0xc79f8f26ea41a4b5cdad3c4ba9a1c7e86474a1f3a1fb31a80e1112122cb4ec4d.
//
// Solidity: event SetPoolFeeProtocol(address pool, uint8 feeProtocol0Old, uint8 feeProtocol1Old, uint8 feeProtocol0New, uint8 feeProtocol1New)
func (_Factory *FactoryFilterer) ParseSetPoolFeeProtocol(log types.Log) (*FactorySetPoolFeeProtocol, error) {
	event := new(FactorySetPoolFeeProtocol)
	if err := _Factory.contract.UnpackLog(event, "SetPoolFeeProtocol", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
