// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package nomiswap

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

// NomiStableFactoryMetaData contains all meta data concerning the NomiStableFactory contract.
var NomiStableFactoryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_feeToSetter\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"pair\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"PairCreated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"INIT_CODE_HASH\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allPairs\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"allPairsLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"}],\"name\":\"createPair\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pair\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"feeTo\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"feeToSetter\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"getPair\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_pair\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"_futureA\",\"type\":\"uint32\"},{\"internalType\":\"uint40\",\"name\":\"_futureTime\",\"type\":\"uint40\"}],\"name\":\"rampA\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_pair\",\"type\":\"address\"},{\"internalType\":\"uint128\",\"name\":\"_devFee\",\"type\":\"uint128\"}],\"name\":\"setDevFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_feeTo\",\"type\":\"address\"}],\"name\":\"setFeeTo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_feeToSetter\",\"type\":\"address\"}],\"name\":\"setFeeToSetter\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_pair\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"_swapFee\",\"type\":\"uint32\"}],\"name\":\"setSwapFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_pair\",\"type\":\"address\"}],\"name\":\"stopRampA\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// NomiStableFactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use NomiStableFactoryMetaData.ABI instead.
var NomiStableFactoryABI = NomiStableFactoryMetaData.ABI

// NomiStableFactory is an auto generated Go binding around an Ethereum contract.
type NomiStableFactory struct {
	NomiStableFactoryCaller     // Read-only binding to the contract
	NomiStableFactoryTransactor // Write-only binding to the contract
	NomiStableFactoryFilterer   // Log filterer for contract events
}

// NomiStableFactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type NomiStableFactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NomiStableFactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type NomiStableFactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NomiStableFactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type NomiStableFactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NomiStableFactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type NomiStableFactorySession struct {
	Contract     *NomiStableFactory // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// NomiStableFactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type NomiStableFactoryCallerSession struct {
	Contract *NomiStableFactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// NomiStableFactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type NomiStableFactoryTransactorSession struct {
	Contract     *NomiStableFactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// NomiStableFactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type NomiStableFactoryRaw struct {
	Contract *NomiStableFactory // Generic contract binding to access the raw methods on
}

// NomiStableFactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type NomiStableFactoryCallerRaw struct {
	Contract *NomiStableFactoryCaller // Generic read-only contract binding to access the raw methods on
}

// NomiStableFactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type NomiStableFactoryTransactorRaw struct {
	Contract *NomiStableFactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewNomiStableFactory creates a new instance of NomiStableFactory, bound to a specific deployed contract.
func NewNomiStableFactory(address common.Address, backend bind.ContractBackend) (*NomiStableFactory, error) {
	contract, err := bindNomiStableFactory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &NomiStableFactory{NomiStableFactoryCaller: NomiStableFactoryCaller{contract: contract}, NomiStableFactoryTransactor: NomiStableFactoryTransactor{contract: contract}, NomiStableFactoryFilterer: NomiStableFactoryFilterer{contract: contract}}, nil
}

// NewNomiStableFactoryCaller creates a new read-only instance of NomiStableFactory, bound to a specific deployed contract.
func NewNomiStableFactoryCaller(address common.Address, caller bind.ContractCaller) (*NomiStableFactoryCaller, error) {
	contract, err := bindNomiStableFactory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &NomiStableFactoryCaller{contract: contract}, nil
}

// NewNomiStableFactoryTransactor creates a new write-only instance of NomiStableFactory, bound to a specific deployed contract.
func NewNomiStableFactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*NomiStableFactoryTransactor, error) {
	contract, err := bindNomiStableFactory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &NomiStableFactoryTransactor{contract: contract}, nil
}

// NewNomiStableFactoryFilterer creates a new log filterer instance of NomiStableFactory, bound to a specific deployed contract.
func NewNomiStableFactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*NomiStableFactoryFilterer, error) {
	contract, err := bindNomiStableFactory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &NomiStableFactoryFilterer{contract: contract}, nil
}

// bindNomiStableFactory binds a generic wrapper to an already deployed contract.
func bindNomiStableFactory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := NomiStableFactoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NomiStableFactory *NomiStableFactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NomiStableFactory.Contract.NomiStableFactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NomiStableFactory *NomiStableFactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.NomiStableFactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NomiStableFactory *NomiStableFactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.NomiStableFactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NomiStableFactory *NomiStableFactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NomiStableFactory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NomiStableFactory *NomiStableFactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NomiStableFactory *NomiStableFactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.contract.Transact(opts, method, params...)
}

// INITCODEHASH is a free data retrieval call binding the contract method 0x257671f5.
//
// Solidity: function INIT_CODE_HASH() view returns(bytes32)
func (_NomiStableFactory *NomiStableFactoryCaller) INITCODEHASH(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _NomiStableFactory.contract.Call(opts, &out, "INIT_CODE_HASH")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// INITCODEHASH is a free data retrieval call binding the contract method 0x257671f5.
//
// Solidity: function INIT_CODE_HASH() view returns(bytes32)
func (_NomiStableFactory *NomiStableFactorySession) INITCODEHASH() ([32]byte, error) {
	return _NomiStableFactory.Contract.INITCODEHASH(&_NomiStableFactory.CallOpts)
}

// INITCODEHASH is a free data retrieval call binding the contract method 0x257671f5.
//
// Solidity: function INIT_CODE_HASH() view returns(bytes32)
func (_NomiStableFactory *NomiStableFactoryCallerSession) INITCODEHASH() ([32]byte, error) {
	return _NomiStableFactory.Contract.INITCODEHASH(&_NomiStableFactory.CallOpts)
}

// AllPairs is a free data retrieval call binding the contract method 0x1e3dd18b.
//
// Solidity: function allPairs(uint256 ) view returns(address)
func (_NomiStableFactory *NomiStableFactoryCaller) AllPairs(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _NomiStableFactory.contract.Call(opts, &out, "allPairs", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AllPairs is a free data retrieval call binding the contract method 0x1e3dd18b.
//
// Solidity: function allPairs(uint256 ) view returns(address)
func (_NomiStableFactory *NomiStableFactorySession) AllPairs(arg0 *big.Int) (common.Address, error) {
	return _NomiStableFactory.Contract.AllPairs(&_NomiStableFactory.CallOpts, arg0)
}

// AllPairs is a free data retrieval call binding the contract method 0x1e3dd18b.
//
// Solidity: function allPairs(uint256 ) view returns(address)
func (_NomiStableFactory *NomiStableFactoryCallerSession) AllPairs(arg0 *big.Int) (common.Address, error) {
	return _NomiStableFactory.Contract.AllPairs(&_NomiStableFactory.CallOpts, arg0)
}

// AllPairsLength is a free data retrieval call binding the contract method 0x574f2ba3.
//
// Solidity: function allPairsLength() view returns(uint256)
func (_NomiStableFactory *NomiStableFactoryCaller) AllPairsLength(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NomiStableFactory.contract.Call(opts, &out, "allPairsLength")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// AllPairsLength is a free data retrieval call binding the contract method 0x574f2ba3.
//
// Solidity: function allPairsLength() view returns(uint256)
func (_NomiStableFactory *NomiStableFactorySession) AllPairsLength() (*big.Int, error) {
	return _NomiStableFactory.Contract.AllPairsLength(&_NomiStableFactory.CallOpts)
}

// AllPairsLength is a free data retrieval call binding the contract method 0x574f2ba3.
//
// Solidity: function allPairsLength() view returns(uint256)
func (_NomiStableFactory *NomiStableFactoryCallerSession) AllPairsLength() (*big.Int, error) {
	return _NomiStableFactory.Contract.AllPairsLength(&_NomiStableFactory.CallOpts)
}

// FeeTo is a free data retrieval call binding the contract method 0x017e7e58.
//
// Solidity: function feeTo() view returns(address)
func (_NomiStableFactory *NomiStableFactoryCaller) FeeTo(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _NomiStableFactory.contract.Call(opts, &out, "feeTo")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FeeTo is a free data retrieval call binding the contract method 0x017e7e58.
//
// Solidity: function feeTo() view returns(address)
func (_NomiStableFactory *NomiStableFactorySession) FeeTo() (common.Address, error) {
	return _NomiStableFactory.Contract.FeeTo(&_NomiStableFactory.CallOpts)
}

// FeeTo is a free data retrieval call binding the contract method 0x017e7e58.
//
// Solidity: function feeTo() view returns(address)
func (_NomiStableFactory *NomiStableFactoryCallerSession) FeeTo() (common.Address, error) {
	return _NomiStableFactory.Contract.FeeTo(&_NomiStableFactory.CallOpts)
}

// FeeToSetter is a free data retrieval call binding the contract method 0x094b7415.
//
// Solidity: function feeToSetter() view returns(address)
func (_NomiStableFactory *NomiStableFactoryCaller) FeeToSetter(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _NomiStableFactory.contract.Call(opts, &out, "feeToSetter")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FeeToSetter is a free data retrieval call binding the contract method 0x094b7415.
//
// Solidity: function feeToSetter() view returns(address)
func (_NomiStableFactory *NomiStableFactorySession) FeeToSetter() (common.Address, error) {
	return _NomiStableFactory.Contract.FeeToSetter(&_NomiStableFactory.CallOpts)
}

// FeeToSetter is a free data retrieval call binding the contract method 0x094b7415.
//
// Solidity: function feeToSetter() view returns(address)
func (_NomiStableFactory *NomiStableFactoryCallerSession) FeeToSetter() (common.Address, error) {
	return _NomiStableFactory.Contract.FeeToSetter(&_NomiStableFactory.CallOpts)
}

// GetPair is a free data retrieval call binding the contract method 0xe6a43905.
//
// Solidity: function getPair(address , address ) view returns(address)
func (_NomiStableFactory *NomiStableFactoryCaller) GetPair(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (common.Address, error) {
	var out []interface{}
	err := _NomiStableFactory.contract.Call(opts, &out, "getPair", arg0, arg1)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetPair is a free data retrieval call binding the contract method 0xe6a43905.
//
// Solidity: function getPair(address , address ) view returns(address)
func (_NomiStableFactory *NomiStableFactorySession) GetPair(arg0 common.Address, arg1 common.Address) (common.Address, error) {
	return _NomiStableFactory.Contract.GetPair(&_NomiStableFactory.CallOpts, arg0, arg1)
}

// GetPair is a free data retrieval call binding the contract method 0xe6a43905.
//
// Solidity: function getPair(address , address ) view returns(address)
func (_NomiStableFactory *NomiStableFactoryCallerSession) GetPair(arg0 common.Address, arg1 common.Address) (common.Address, error) {
	return _NomiStableFactory.Contract.GetPair(&_NomiStableFactory.CallOpts, arg0, arg1)
}

// CreatePair is a paid mutator transaction binding the contract method 0xc9c65396.
//
// Solidity: function createPair(address tokenA, address tokenB) returns(address pair)
func (_NomiStableFactory *NomiStableFactoryTransactor) CreatePair(opts *bind.TransactOpts, tokenA common.Address, tokenB common.Address) (*types.Transaction, error) {
	return _NomiStableFactory.contract.Transact(opts, "createPair", tokenA, tokenB)
}

// CreatePair is a paid mutator transaction binding the contract method 0xc9c65396.
//
// Solidity: function createPair(address tokenA, address tokenB) returns(address pair)
func (_NomiStableFactory *NomiStableFactorySession) CreatePair(tokenA common.Address, tokenB common.Address) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.CreatePair(&_NomiStableFactory.TransactOpts, tokenA, tokenB)
}

// CreatePair is a paid mutator transaction binding the contract method 0xc9c65396.
//
// Solidity: function createPair(address tokenA, address tokenB) returns(address pair)
func (_NomiStableFactory *NomiStableFactoryTransactorSession) CreatePair(tokenA common.Address, tokenB common.Address) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.CreatePair(&_NomiStableFactory.TransactOpts, tokenA, tokenB)
}

// RampA is a paid mutator transaction binding the contract method 0xbcec1e4d.
//
// Solidity: function rampA(address _pair, uint32 _futureA, uint40 _futureTime) returns()
func (_NomiStableFactory *NomiStableFactoryTransactor) RampA(opts *bind.TransactOpts, _pair common.Address, _futureA uint32, _futureTime *big.Int) (*types.Transaction, error) {
	return _NomiStableFactory.contract.Transact(opts, "rampA", _pair, _futureA, _futureTime)
}

// RampA is a paid mutator transaction binding the contract method 0xbcec1e4d.
//
// Solidity: function rampA(address _pair, uint32 _futureA, uint40 _futureTime) returns()
func (_NomiStableFactory *NomiStableFactorySession) RampA(_pair common.Address, _futureA uint32, _futureTime *big.Int) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.RampA(&_NomiStableFactory.TransactOpts, _pair, _futureA, _futureTime)
}

// RampA is a paid mutator transaction binding the contract method 0xbcec1e4d.
//
// Solidity: function rampA(address _pair, uint32 _futureA, uint40 _futureTime) returns()
func (_NomiStableFactory *NomiStableFactoryTransactorSession) RampA(_pair common.Address, _futureA uint32, _futureTime *big.Int) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.RampA(&_NomiStableFactory.TransactOpts, _pair, _futureA, _futureTime)
}

// SetDevFee is a paid mutator transaction binding the contract method 0x52b5c71e.
//
// Solidity: function setDevFee(address _pair, uint128 _devFee) returns()
func (_NomiStableFactory *NomiStableFactoryTransactor) SetDevFee(opts *bind.TransactOpts, _pair common.Address, _devFee *big.Int) (*types.Transaction, error) {
	return _NomiStableFactory.contract.Transact(opts, "setDevFee", _pair, _devFee)
}

// SetDevFee is a paid mutator transaction binding the contract method 0x52b5c71e.
//
// Solidity: function setDevFee(address _pair, uint128 _devFee) returns()
func (_NomiStableFactory *NomiStableFactorySession) SetDevFee(_pair common.Address, _devFee *big.Int) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.SetDevFee(&_NomiStableFactory.TransactOpts, _pair, _devFee)
}

// SetDevFee is a paid mutator transaction binding the contract method 0x52b5c71e.
//
// Solidity: function setDevFee(address _pair, uint128 _devFee) returns()
func (_NomiStableFactory *NomiStableFactoryTransactorSession) SetDevFee(_pair common.Address, _devFee *big.Int) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.SetDevFee(&_NomiStableFactory.TransactOpts, _pair, _devFee)
}

// SetFeeTo is a paid mutator transaction binding the contract method 0xf46901ed.
//
// Solidity: function setFeeTo(address _feeTo) returns()
func (_NomiStableFactory *NomiStableFactoryTransactor) SetFeeTo(opts *bind.TransactOpts, _feeTo common.Address) (*types.Transaction, error) {
	return _NomiStableFactory.contract.Transact(opts, "setFeeTo", _feeTo)
}

// SetFeeTo is a paid mutator transaction binding the contract method 0xf46901ed.
//
// Solidity: function setFeeTo(address _feeTo) returns()
func (_NomiStableFactory *NomiStableFactorySession) SetFeeTo(_feeTo common.Address) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.SetFeeTo(&_NomiStableFactory.TransactOpts, _feeTo)
}

// SetFeeTo is a paid mutator transaction binding the contract method 0xf46901ed.
//
// Solidity: function setFeeTo(address _feeTo) returns()
func (_NomiStableFactory *NomiStableFactoryTransactorSession) SetFeeTo(_feeTo common.Address) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.SetFeeTo(&_NomiStableFactory.TransactOpts, _feeTo)
}

// SetFeeToSetter is a paid mutator transaction binding the contract method 0xa2e74af6.
//
// Solidity: function setFeeToSetter(address _feeToSetter) returns()
func (_NomiStableFactory *NomiStableFactoryTransactor) SetFeeToSetter(opts *bind.TransactOpts, _feeToSetter common.Address) (*types.Transaction, error) {
	return _NomiStableFactory.contract.Transact(opts, "setFeeToSetter", _feeToSetter)
}

// SetFeeToSetter is a paid mutator transaction binding the contract method 0xa2e74af6.
//
// Solidity: function setFeeToSetter(address _feeToSetter) returns()
func (_NomiStableFactory *NomiStableFactorySession) SetFeeToSetter(_feeToSetter common.Address) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.SetFeeToSetter(&_NomiStableFactory.TransactOpts, _feeToSetter)
}

// SetFeeToSetter is a paid mutator transaction binding the contract method 0xa2e74af6.
//
// Solidity: function setFeeToSetter(address _feeToSetter) returns()
func (_NomiStableFactory *NomiStableFactoryTransactorSession) SetFeeToSetter(_feeToSetter common.Address) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.SetFeeToSetter(&_NomiStableFactory.TransactOpts, _feeToSetter)
}

// SetSwapFee is a paid mutator transaction binding the contract method 0x9e68ceb8.
//
// Solidity: function setSwapFee(address _pair, uint32 _swapFee) returns()
func (_NomiStableFactory *NomiStableFactoryTransactor) SetSwapFee(opts *bind.TransactOpts, _pair common.Address, _swapFee uint32) (*types.Transaction, error) {
	return _NomiStableFactory.contract.Transact(opts, "setSwapFee", _pair, _swapFee)
}

// SetSwapFee is a paid mutator transaction binding the contract method 0x9e68ceb8.
//
// Solidity: function setSwapFee(address _pair, uint32 _swapFee) returns()
func (_NomiStableFactory *NomiStableFactorySession) SetSwapFee(_pair common.Address, _swapFee uint32) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.SetSwapFee(&_NomiStableFactory.TransactOpts, _pair, _swapFee)
}

// SetSwapFee is a paid mutator transaction binding the contract method 0x9e68ceb8.
//
// Solidity: function setSwapFee(address _pair, uint32 _swapFee) returns()
func (_NomiStableFactory *NomiStableFactoryTransactorSession) SetSwapFee(_pair common.Address, _swapFee uint32) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.SetSwapFee(&_NomiStableFactory.TransactOpts, _pair, _swapFee)
}

// StopRampA is a paid mutator transaction binding the contract method 0x6864a4b3.
//
// Solidity: function stopRampA(address _pair) returns()
func (_NomiStableFactory *NomiStableFactoryTransactor) StopRampA(opts *bind.TransactOpts, _pair common.Address) (*types.Transaction, error) {
	return _NomiStableFactory.contract.Transact(opts, "stopRampA", _pair)
}

// StopRampA is a paid mutator transaction binding the contract method 0x6864a4b3.
//
// Solidity: function stopRampA(address _pair) returns()
func (_NomiStableFactory *NomiStableFactorySession) StopRampA(_pair common.Address) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.StopRampA(&_NomiStableFactory.TransactOpts, _pair)
}

// StopRampA is a paid mutator transaction binding the contract method 0x6864a4b3.
//
// Solidity: function stopRampA(address _pair) returns()
func (_NomiStableFactory *NomiStableFactoryTransactorSession) StopRampA(_pair common.Address) (*types.Transaction, error) {
	return _NomiStableFactory.Contract.StopRampA(&_NomiStableFactory.TransactOpts, _pair)
}

// NomiStableFactoryPairCreatedIterator is returned from FilterPairCreated and is used to iterate over the raw logs and unpacked data for PairCreated events raised by the NomiStableFactory contract.
type NomiStableFactoryPairCreatedIterator struct {
	Event *NomiStableFactoryPairCreated // Event containing the contract specifics and raw log

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
func (it *NomiStableFactoryPairCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NomiStableFactoryPairCreated)
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
		it.Event = new(NomiStableFactoryPairCreated)
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
func (it *NomiStableFactoryPairCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NomiStableFactoryPairCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NomiStableFactoryPairCreated represents a PairCreated event raised by the NomiStableFactory contract.
type NomiStableFactoryPairCreated struct {
	Token0 common.Address
	Token1 common.Address
	Pair   common.Address
	Arg3   *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterPairCreated is a free log retrieval operation binding the contract event 0x0d3648bd0f6ba80134a33ba9275ac585d9d315f0ad8355cddefde31afa28d0e9.
//
// Solidity: event PairCreated(address indexed token0, address indexed token1, address pair, uint256 arg3)
func (_NomiStableFactory *NomiStableFactoryFilterer) FilterPairCreated(opts *bind.FilterOpts, token0 []common.Address, token1 []common.Address) (*NomiStableFactoryPairCreatedIterator, error) {

	var token0Rule []interface{}
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []interface{}
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}

	logs, sub, err := _NomiStableFactory.contract.FilterLogs(opts, "PairCreated", token0Rule, token1Rule)
	if err != nil {
		return nil, err
	}
	return &NomiStableFactoryPairCreatedIterator{contract: _NomiStableFactory.contract, event: "PairCreated", logs: logs, sub: sub}, nil
}

// WatchPairCreated is a free log subscription operation binding the contract event 0x0d3648bd0f6ba80134a33ba9275ac585d9d315f0ad8355cddefde31afa28d0e9.
//
// Solidity: event PairCreated(address indexed token0, address indexed token1, address pair, uint256 arg3)
func (_NomiStableFactory *NomiStableFactoryFilterer) WatchPairCreated(opts *bind.WatchOpts, sink chan<- *NomiStableFactoryPairCreated, token0 []common.Address, token1 []common.Address) (event.Subscription, error) {

	var token0Rule []interface{}
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []interface{}
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}

	logs, sub, err := _NomiStableFactory.contract.WatchLogs(opts, "PairCreated", token0Rule, token1Rule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NomiStableFactoryPairCreated)
				if err := _NomiStableFactory.contract.UnpackLog(event, "PairCreated", log); err != nil {
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

// ParsePairCreated is a log parse operation binding the contract event 0x0d3648bd0f6ba80134a33ba9275ac585d9d315f0ad8355cddefde31afa28d0e9.
//
// Solidity: event PairCreated(address indexed token0, address indexed token1, address pair, uint256 arg3)
func (_NomiStableFactory *NomiStableFactoryFilterer) ParsePairCreated(log types.Log) (*NomiStableFactoryPairCreated, error) {
	event := new(NomiStableFactoryPairCreated)
	if err := _NomiStableFactory.contract.UnpackLog(event, "PairCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
