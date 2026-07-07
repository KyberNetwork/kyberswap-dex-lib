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

// Order is an auto generated low-level Go binding around an user-defined struct.
type Order struct {
	Y *big.Int
	Z *big.Int
	A uint64
	B uint64
}

// Pair is an auto generated low-level Go binding around an user-defined struct.
type Pair struct {
	Id     *big.Int
	Tokens [2]common.Address
}

// Strategy is an auto generated low-level Go binding around an user-defined struct.
type Strategy struct {
	Id     *big.Int
	Owner  common.Address
	Tokens [2]common.Address
	Orders [2]Order
}

// TradeAction is an auto generated low-level Go binding around an user-defined struct.
type TradeAction struct {
	StrategyId *big.Int
	Amount     *big.Int
}

// ControllerMetaData contains all meta data concerning the Controller contract.
var ControllerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint128\",\"name\":\"pairId\",\"type\":\"uint128\"},{\"indexed\":true,\"internalType\":\"Token\",\"name\":\"token0\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"Token\",\"name\":\"token1\",\"type\":\"address\"}],\"name\":\"PairCreated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"Token\",\"name\":\"token0\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"Token\",\"name\":\"token1\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"prevFeePPM\",\"type\":\"uint32\"},{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"newFeePPM\",\"type\":\"uint32\"}],\"name\":\"PairTradingFeePPMUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"Token\",\"name\":\"token0\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"Token\",\"name\":\"token1\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint128\",\"name\":\"y\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"z\",\"type\":\"uint128\"},{\"internalType\":\"uint64\",\"name\":\"A\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"B\",\"type\":\"uint64\"}],\"indexed\":false,\"internalType\":\"structOrder\",\"name\":\"order0\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint128\",\"name\":\"y\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"z\",\"type\":\"uint128\"},{\"internalType\":\"uint64\",\"name\":\"A\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"B\",\"type\":\"uint64\"}],\"indexed\":false,\"internalType\":\"structOrder\",\"name\":\"order1\",\"type\":\"tuple\"}],\"name\":\"StrategyCreated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"Token\",\"name\":\"token0\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"Token\",\"name\":\"token1\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint128\",\"name\":\"y\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"z\",\"type\":\"uint128\"},{\"internalType\":\"uint64\",\"name\":\"A\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"B\",\"type\":\"uint64\"}],\"indexed\":false,\"internalType\":\"structOrder\",\"name\":\"order0\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint128\",\"name\":\"y\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"z\",\"type\":\"uint128\"},{\"internalType\":\"uint64\",\"name\":\"A\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"B\",\"type\":\"uint64\"}],\"indexed\":false,\"internalType\":\"structOrder\",\"name\":\"order1\",\"type\":\"tuple\"}],\"name\":\"StrategyDeleted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"Token\",\"name\":\"token0\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"Token\",\"name\":\"token1\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint128\",\"name\":\"y\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"z\",\"type\":\"uint128\"},{\"internalType\":\"uint64\",\"name\":\"A\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"B\",\"type\":\"uint64\"}],\"indexed\":false,\"internalType\":\"structOrder\",\"name\":\"order0\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint128\",\"name\":\"y\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"z\",\"type\":\"uint128\"},{\"internalType\":\"uint64\",\"name\":\"A\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"B\",\"type\":\"uint64\"}],\"indexed\":false,\"internalType\":\"structOrder\",\"name\":\"order1\",\"type\":\"tuple\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"reason\",\"type\":\"uint8\"}],\"name\":\"StrategyUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"trader\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"Token\",\"name\":\"sourceToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"Token\",\"name\":\"targetToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"sourceAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"targetAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"tradingFeeAmount\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"byTargetAmount\",\"type\":\"bool\"}],\"name\":\"TokensTraded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"prevFeePPM\",\"type\":\"uint32\"},{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"newFeePPM\",\"type\":\"uint32\"}],\"name\":\"TradingFeePPMUpdated\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"Token\",\"name\":\"sourceToken\",\"type\":\"address\"},{\"internalType\":\"Token\",\"name\":\"targetToken\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"strategyId\",\"type\":\"uint256\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"}],\"internalType\":\"structTradeAction[]\",\"name\":\"tradeActions\",\"type\":\"tuple[]\"}],\"name\":\"calculateTradeSourceAmount\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Token\",\"name\":\"sourceToken\",\"type\":\"address\"},{\"internalType\":\"Token\",\"name\":\"targetToken\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"strategyId\",\"type\":\"uint256\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"}],\"internalType\":\"structTradeAction[]\",\"name\":\"tradeActions\",\"type\":\"tuple[]\"}],\"name\":\"calculateTradeTargetAmount\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Token\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"Token\",\"name\":\"token1\",\"type\":\"address\"}],\"name\":\"pair\",\"outputs\":[{\"components\":[{\"internalType\":\"uint128\",\"name\":\"id\",\"type\":\"uint128\"},{\"internalType\":\"Token[2]\",\"name\":\"tokens\",\"type\":\"address[2]\"}],\"internalType\":\"structPair\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Token\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"Token\",\"name\":\"token1\",\"type\":\"address\"}],\"name\":\"pairTradingFeePPM\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pairs\",\"outputs\":[{\"internalType\":\"Token[2][]\",\"name\":\"\",\"type\":\"address[2][]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Token\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"Token\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"startIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"endIndex\",\"type\":\"uint256\"}],\"name\":\"strategiesByPair\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"Token[2]\",\"name\":\"tokens\",\"type\":\"address[2]\"},{\"components\":[{\"internalType\":\"uint128\",\"name\":\"y\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"z\",\"type\":\"uint128\"},{\"internalType\":\"uint64\",\"name\":\"A\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"B\",\"type\":\"uint64\"}],\"internalType\":\"structOrder[2]\",\"name\":\"orders\",\"type\":\"tuple[2]\"}],\"internalType\":\"structStrategy[]\",\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Token\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"Token\",\"name\":\"token1\",\"type\":\"address\"}],\"name\":\"strategiesByPairCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"strategy\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"Token[2]\",\"name\":\"tokens\",\"type\":\"address[2]\"},{\"components\":[{\"internalType\":\"uint128\",\"name\":\"y\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"z\",\"type\":\"uint128\"},{\"internalType\":\"uint64\",\"name\":\"A\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"B\",\"type\":\"uint64\"}],\"internalType\":\"structOrder[2]\",\"name\":\"orders\",\"type\":\"tuple[2]\"}],\"internalType\":\"structStrategy\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Token\",\"name\":\"sourceToken\",\"type\":\"address\"},{\"internalType\":\"Token\",\"name\":\"targetToken\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"strategyId\",\"type\":\"uint256\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"}],\"internalType\":\"structTradeAction[]\",\"name\":\"tradeActions\",\"type\":\"tuple[]\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"uint128\",\"name\":\"minReturn\",\"type\":\"uint128\"}],\"name\":\"tradeBySourceAmount\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Token\",\"name\":\"sourceToken\",\"type\":\"address\"},{\"internalType\":\"Token\",\"name\":\"targetToken\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"strategyId\",\"type\":\"uint256\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"}],\"internalType\":\"structTradeAction[]\",\"name\":\"tradeActions\",\"type\":\"tuple[]\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"uint128\",\"name\":\"maxInput\",\"type\":\"uint128\"}],\"name\":\"tradeByTargetAmount\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"tradingFeePPM\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// ControllerABI is the input ABI used to generate the binding from.
// Deprecated: Use ControllerMetaData.ABI instead.
var ControllerABI = ControllerMetaData.ABI

// Controller is an auto generated Go binding around an Ethereum contract.
type Controller struct {
	ControllerCaller     // Read-only binding to the contract
	ControllerTransactor // Write-only binding to the contract
	ControllerFilterer   // Log filterer for contract events
}

// ControllerCaller is an auto generated read-only Go binding around an Ethereum contract.
type ControllerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ControllerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ControllerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ControllerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ControllerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ControllerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ControllerSession struct {
	Contract     *Controller       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ControllerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ControllerCallerSession struct {
	Contract *ControllerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// ControllerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ControllerTransactorSession struct {
	Contract     *ControllerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// ControllerRaw is an auto generated low-level Go binding around an Ethereum contract.
type ControllerRaw struct {
	Contract *Controller // Generic contract binding to access the raw methods on
}

// ControllerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ControllerCallerRaw struct {
	Contract *ControllerCaller // Generic read-only contract binding to access the raw methods on
}

// ControllerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ControllerTransactorRaw struct {
	Contract *ControllerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewController creates a new instance of Controller, bound to a specific deployed contract.
func NewController(address common.Address, backend bind.ContractBackend) (*Controller, error) {
	contract, err := bindController(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Controller{ControllerCaller: ControllerCaller{contract: contract}, ControllerTransactor: ControllerTransactor{contract: contract}, ControllerFilterer: ControllerFilterer{contract: contract}}, nil
}

// NewControllerCaller creates a new read-only instance of Controller, bound to a specific deployed contract.
func NewControllerCaller(address common.Address, caller bind.ContractCaller) (*ControllerCaller, error) {
	contract, err := bindController(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ControllerCaller{contract: contract}, nil
}

// NewControllerTransactor creates a new write-only instance of Controller, bound to a specific deployed contract.
func NewControllerTransactor(address common.Address, transactor bind.ContractTransactor) (*ControllerTransactor, error) {
	contract, err := bindController(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ControllerTransactor{contract: contract}, nil
}

// NewControllerFilterer creates a new log filterer instance of Controller, bound to a specific deployed contract.
func NewControllerFilterer(address common.Address, filterer bind.ContractFilterer) (*ControllerFilterer, error) {
	contract, err := bindController(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ControllerFilterer{contract: contract}, nil
}

// bindController binds a generic wrapper to an already deployed contract.
func bindController(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ControllerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Controller *ControllerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Controller.Contract.ControllerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Controller *ControllerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Controller.Contract.ControllerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Controller *ControllerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Controller.Contract.ControllerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Controller *ControllerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Controller.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Controller *ControllerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Controller.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Controller *ControllerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Controller.Contract.contract.Transact(opts, method, params...)
}

// CalculateTradeSourceAmount is a free data retrieval call binding the contract method 0xf2bda26d.
//
// Solidity: function calculateTradeSourceAmount(address sourceToken, address targetToken, (uint256,uint128)[] tradeActions) view returns(uint128)
func (_Controller *ControllerCaller) CalculateTradeSourceAmount(opts *bind.CallOpts, sourceToken common.Address, targetToken common.Address, tradeActions []TradeAction) (*big.Int, error) {
	var out []interface{}
	err := _Controller.contract.Call(opts, &out, "calculateTradeSourceAmount", sourceToken, targetToken, tradeActions)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CalculateTradeSourceAmount is a free data retrieval call binding the contract method 0xf2bda26d.
//
// Solidity: function calculateTradeSourceAmount(address sourceToken, address targetToken, (uint256,uint128)[] tradeActions) view returns(uint128)
func (_Controller *ControllerSession) CalculateTradeSourceAmount(sourceToken common.Address, targetToken common.Address, tradeActions []TradeAction) (*big.Int, error) {
	return _Controller.Contract.CalculateTradeSourceAmount(&_Controller.CallOpts, sourceToken, targetToken, tradeActions)
}

// CalculateTradeSourceAmount is a free data retrieval call binding the contract method 0xf2bda26d.
//
// Solidity: function calculateTradeSourceAmount(address sourceToken, address targetToken, (uint256,uint128)[] tradeActions) view returns(uint128)
func (_Controller *ControllerCallerSession) CalculateTradeSourceAmount(sourceToken common.Address, targetToken common.Address, tradeActions []TradeAction) (*big.Int, error) {
	return _Controller.Contract.CalculateTradeSourceAmount(&_Controller.CallOpts, sourceToken, targetToken, tradeActions)
}

// CalculateTradeTargetAmount is a free data retrieval call binding the contract method 0x2ab2fad1.
//
// Solidity: function calculateTradeTargetAmount(address sourceToken, address targetToken, (uint256,uint128)[] tradeActions) view returns(uint128)
func (_Controller *ControllerCaller) CalculateTradeTargetAmount(opts *bind.CallOpts, sourceToken common.Address, targetToken common.Address, tradeActions []TradeAction) (*big.Int, error) {
	var out []interface{}
	err := _Controller.contract.Call(opts, &out, "calculateTradeTargetAmount", sourceToken, targetToken, tradeActions)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CalculateTradeTargetAmount is a free data retrieval call binding the contract method 0x2ab2fad1.
//
// Solidity: function calculateTradeTargetAmount(address sourceToken, address targetToken, (uint256,uint128)[] tradeActions) view returns(uint128)
func (_Controller *ControllerSession) CalculateTradeTargetAmount(sourceToken common.Address, targetToken common.Address, tradeActions []TradeAction) (*big.Int, error) {
	return _Controller.Contract.CalculateTradeTargetAmount(&_Controller.CallOpts, sourceToken, targetToken, tradeActions)
}

// CalculateTradeTargetAmount is a free data retrieval call binding the contract method 0x2ab2fad1.
//
// Solidity: function calculateTradeTargetAmount(address sourceToken, address targetToken, (uint256,uint128)[] tradeActions) view returns(uint128)
func (_Controller *ControllerCallerSession) CalculateTradeTargetAmount(sourceToken common.Address, targetToken common.Address, tradeActions []TradeAction) (*big.Int, error) {
	return _Controller.Contract.CalculateTradeTargetAmount(&_Controller.CallOpts, sourceToken, targetToken, tradeActions)
}

// Pair is a free data retrieval call binding the contract method 0x8672d545.
//
// Solidity: function pair(address token0, address token1) view returns((uint128,address[2]))
func (_Controller *ControllerCaller) Pair(opts *bind.CallOpts, token0 common.Address, token1 common.Address) (Pair, error) {
	var out []interface{}
	err := _Controller.contract.Call(opts, &out, "pair", token0, token1)

	if err != nil {
		return *new(Pair), err
	}

	out0 := *abi.ConvertType(out[0], new(Pair)).(*Pair)

	return out0, err

}

// Pair is a free data retrieval call binding the contract method 0x8672d545.
//
// Solidity: function pair(address token0, address token1) view returns((uint128,address[2]))
func (_Controller *ControllerSession) Pair(token0 common.Address, token1 common.Address) (Pair, error) {
	return _Controller.Contract.Pair(&_Controller.CallOpts, token0, token1)
}

// Pair is a free data retrieval call binding the contract method 0x8672d545.
//
// Solidity: function pair(address token0, address token1) view returns((uint128,address[2]))
func (_Controller *ControllerCallerSession) Pair(token0 common.Address, token1 common.Address) (Pair, error) {
	return _Controller.Contract.Pair(&_Controller.CallOpts, token0, token1)
}

// PairTradingFeePPM is a free data retrieval call binding the contract method 0xba0a868b.
//
// Solidity: function pairTradingFeePPM(address token0, address token1) view returns(uint32)
func (_Controller *ControllerCaller) PairTradingFeePPM(opts *bind.CallOpts, token0 common.Address, token1 common.Address) (uint32, error) {
	var out []interface{}
	err := _Controller.contract.Call(opts, &out, "pairTradingFeePPM", token0, token1)

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// PairTradingFeePPM is a free data retrieval call binding the contract method 0xba0a868b.
//
// Solidity: function pairTradingFeePPM(address token0, address token1) view returns(uint32)
func (_Controller *ControllerSession) PairTradingFeePPM(token0 common.Address, token1 common.Address) (uint32, error) {
	return _Controller.Contract.PairTradingFeePPM(&_Controller.CallOpts, token0, token1)
}

// PairTradingFeePPM is a free data retrieval call binding the contract method 0xba0a868b.
//
// Solidity: function pairTradingFeePPM(address token0, address token1) view returns(uint32)
func (_Controller *ControllerCallerSession) PairTradingFeePPM(token0 common.Address, token1 common.Address) (uint32, error) {
	return _Controller.Contract.PairTradingFeePPM(&_Controller.CallOpts, token0, token1)
}

// Pairs is a free data retrieval call binding the contract method 0xffb0a4a0.
//
// Solidity: function pairs() view returns(address[2][])
func (_Controller *ControllerCaller) Pairs(opts *bind.CallOpts) ([][2]common.Address, error) {
	var out []interface{}
	err := _Controller.contract.Call(opts, &out, "pairs")

	if err != nil {
		return *new([][2]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([][2]common.Address)).(*[][2]common.Address)

	return out0, err

}

// Pairs is a free data retrieval call binding the contract method 0xffb0a4a0.
//
// Solidity: function pairs() view returns(address[2][])
func (_Controller *ControllerSession) Pairs() ([][2]common.Address, error) {
	return _Controller.Contract.Pairs(&_Controller.CallOpts)
}

// Pairs is a free data retrieval call binding the contract method 0xffb0a4a0.
//
// Solidity: function pairs() view returns(address[2][])
func (_Controller *ControllerCallerSession) Pairs() ([][2]common.Address, error) {
	return _Controller.Contract.Pairs(&_Controller.CallOpts)
}

// StrategiesByPair is a free data retrieval call binding the contract method 0xf74dad81.
//
// Solidity: function strategiesByPair(address token0, address token1, uint256 startIndex, uint256 endIndex) view returns((uint256,address,address[2],(uint128,uint128,uint64,uint64)[2])[])
func (_Controller *ControllerCaller) StrategiesByPair(opts *bind.CallOpts, token0 common.Address, token1 common.Address, startIndex *big.Int, endIndex *big.Int) ([]Strategy, error) {
	var out []interface{}
	err := _Controller.contract.Call(opts, &out, "strategiesByPair", token0, token1, startIndex, endIndex)

	if err != nil {
		return *new([]Strategy), err
	}

	out0 := *abi.ConvertType(out[0], new([]Strategy)).(*[]Strategy)

	return out0, err

}

// StrategiesByPair is a free data retrieval call binding the contract method 0xf74dad81.
//
// Solidity: function strategiesByPair(address token0, address token1, uint256 startIndex, uint256 endIndex) view returns((uint256,address,address[2],(uint128,uint128,uint64,uint64)[2])[])
func (_Controller *ControllerSession) StrategiesByPair(token0 common.Address, token1 common.Address, startIndex *big.Int, endIndex *big.Int) ([]Strategy, error) {
	return _Controller.Contract.StrategiesByPair(&_Controller.CallOpts, token0, token1, startIndex, endIndex)
}

// StrategiesByPair is a free data retrieval call binding the contract method 0xf74dad81.
//
// Solidity: function strategiesByPair(address token0, address token1, uint256 startIndex, uint256 endIndex) view returns((uint256,address,address[2],(uint128,uint128,uint64,uint64)[2])[])
func (_Controller *ControllerCallerSession) StrategiesByPair(token0 common.Address, token1 common.Address, startIndex *big.Int, endIndex *big.Int) ([]Strategy, error) {
	return _Controller.Contract.StrategiesByPair(&_Controller.CallOpts, token0, token1, startIndex, endIndex)
}

// StrategiesByPairCount is a free data retrieval call binding the contract method 0x322cf844.
//
// Solidity: function strategiesByPairCount(address token0, address token1) view returns(uint256)
func (_Controller *ControllerCaller) StrategiesByPairCount(opts *bind.CallOpts, token0 common.Address, token1 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Controller.contract.Call(opts, &out, "strategiesByPairCount", token0, token1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StrategiesByPairCount is a free data retrieval call binding the contract method 0x322cf844.
//
// Solidity: function strategiesByPairCount(address token0, address token1) view returns(uint256)
func (_Controller *ControllerSession) StrategiesByPairCount(token0 common.Address, token1 common.Address) (*big.Int, error) {
	return _Controller.Contract.StrategiesByPairCount(&_Controller.CallOpts, token0, token1)
}

// StrategiesByPairCount is a free data retrieval call binding the contract method 0x322cf844.
//
// Solidity: function strategiesByPairCount(address token0, address token1) view returns(uint256)
func (_Controller *ControllerCallerSession) StrategiesByPairCount(token0 common.Address, token1 common.Address) (*big.Int, error) {
	return _Controller.Contract.StrategiesByPairCount(&_Controller.CallOpts, token0, token1)
}

// Strategy is a free data retrieval call binding the contract method 0xbc88d7e4.
//
// Solidity: function strategy(uint256 id) view returns((uint256,address,address[2],(uint128,uint128,uint64,uint64)[2]))
func (_Controller *ControllerCaller) Strategy(opts *bind.CallOpts, id *big.Int) (Strategy, error) {
	var out []interface{}
	err := _Controller.contract.Call(opts, &out, "strategy", id)

	if err != nil {
		return *new(Strategy), err
	}

	out0 := *abi.ConvertType(out[0], new(Strategy)).(*Strategy)

	return out0, err

}

// Strategy is a free data retrieval call binding the contract method 0xbc88d7e4.
//
// Solidity: function strategy(uint256 id) view returns((uint256,address,address[2],(uint128,uint128,uint64,uint64)[2]))
func (_Controller *ControllerSession) Strategy(id *big.Int) (Strategy, error) {
	return _Controller.Contract.Strategy(&_Controller.CallOpts, id)
}

// Strategy is a free data retrieval call binding the contract method 0xbc88d7e4.
//
// Solidity: function strategy(uint256 id) view returns((uint256,address,address[2],(uint128,uint128,uint64,uint64)[2]))
func (_Controller *ControllerCallerSession) Strategy(id *big.Int) (Strategy, error) {
	return _Controller.Contract.Strategy(&_Controller.CallOpts, id)
}

// TradingFeePPM is a free data retrieval call binding the contract method 0xf06f8acd.
//
// Solidity: function tradingFeePPM() view returns(uint32)
func (_Controller *ControllerCaller) TradingFeePPM(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _Controller.contract.Call(opts, &out, "tradingFeePPM")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// TradingFeePPM is a free data retrieval call binding the contract method 0xf06f8acd.
//
// Solidity: function tradingFeePPM() view returns(uint32)
func (_Controller *ControllerSession) TradingFeePPM() (uint32, error) {
	return _Controller.Contract.TradingFeePPM(&_Controller.CallOpts)
}

// TradingFeePPM is a free data retrieval call binding the contract method 0xf06f8acd.
//
// Solidity: function tradingFeePPM() view returns(uint32)
func (_Controller *ControllerCallerSession) TradingFeePPM() (uint32, error) {
	return _Controller.Contract.TradingFeePPM(&_Controller.CallOpts)
}

// TradeBySourceAmount is a paid mutator transaction binding the contract method 0xf1c5e014.
//
// Solidity: function tradeBySourceAmount(address sourceToken, address targetToken, (uint256,uint128)[] tradeActions, uint256 deadline, uint128 minReturn) payable returns(uint128)
func (_Controller *ControllerTransactor) TradeBySourceAmount(opts *bind.TransactOpts, sourceToken common.Address, targetToken common.Address, tradeActions []TradeAction, deadline *big.Int, minReturn *big.Int) (*types.Transaction, error) {
	return _Controller.contract.Transact(opts, "tradeBySourceAmount", sourceToken, targetToken, tradeActions, deadline, minReturn)
}

// TradeBySourceAmount is a paid mutator transaction binding the contract method 0xf1c5e014.
//
// Solidity: function tradeBySourceAmount(address sourceToken, address targetToken, (uint256,uint128)[] tradeActions, uint256 deadline, uint128 minReturn) payable returns(uint128)
func (_Controller *ControllerSession) TradeBySourceAmount(sourceToken common.Address, targetToken common.Address, tradeActions []TradeAction, deadline *big.Int, minReturn *big.Int) (*types.Transaction, error) {
	return _Controller.Contract.TradeBySourceAmount(&_Controller.TransactOpts, sourceToken, targetToken, tradeActions, deadline, minReturn)
}

// TradeBySourceAmount is a paid mutator transaction binding the contract method 0xf1c5e014.
//
// Solidity: function tradeBySourceAmount(address sourceToken, address targetToken, (uint256,uint128)[] tradeActions, uint256 deadline, uint128 minReturn) payable returns(uint128)
func (_Controller *ControllerTransactorSession) TradeBySourceAmount(sourceToken common.Address, targetToken common.Address, tradeActions []TradeAction, deadline *big.Int, minReturn *big.Int) (*types.Transaction, error) {
	return _Controller.Contract.TradeBySourceAmount(&_Controller.TransactOpts, sourceToken, targetToken, tradeActions, deadline, minReturn)
}

// TradeByTargetAmount is a paid mutator transaction binding the contract method 0x102ee9ba.
//
// Solidity: function tradeByTargetAmount(address sourceToken, address targetToken, (uint256,uint128)[] tradeActions, uint256 deadline, uint128 maxInput) payable returns(uint128)
func (_Controller *ControllerTransactor) TradeByTargetAmount(opts *bind.TransactOpts, sourceToken common.Address, targetToken common.Address, tradeActions []TradeAction, deadline *big.Int, maxInput *big.Int) (*types.Transaction, error) {
	return _Controller.contract.Transact(opts, "tradeByTargetAmount", sourceToken, targetToken, tradeActions, deadline, maxInput)
}

// TradeByTargetAmount is a paid mutator transaction binding the contract method 0x102ee9ba.
//
// Solidity: function tradeByTargetAmount(address sourceToken, address targetToken, (uint256,uint128)[] tradeActions, uint256 deadline, uint128 maxInput) payable returns(uint128)
func (_Controller *ControllerSession) TradeByTargetAmount(sourceToken common.Address, targetToken common.Address, tradeActions []TradeAction, deadline *big.Int, maxInput *big.Int) (*types.Transaction, error) {
	return _Controller.Contract.TradeByTargetAmount(&_Controller.TransactOpts, sourceToken, targetToken, tradeActions, deadline, maxInput)
}

// TradeByTargetAmount is a paid mutator transaction binding the contract method 0x102ee9ba.
//
// Solidity: function tradeByTargetAmount(address sourceToken, address targetToken, (uint256,uint128)[] tradeActions, uint256 deadline, uint128 maxInput) payable returns(uint128)
func (_Controller *ControllerTransactorSession) TradeByTargetAmount(sourceToken common.Address, targetToken common.Address, tradeActions []TradeAction, deadline *big.Int, maxInput *big.Int) (*types.Transaction, error) {
	return _Controller.Contract.TradeByTargetAmount(&_Controller.TransactOpts, sourceToken, targetToken, tradeActions, deadline, maxInput)
}

// ControllerPairCreatedIterator is returned from FilterPairCreated and is used to iterate over the raw logs and unpacked data for PairCreated events raised by the Controller contract.
type ControllerPairCreatedIterator struct {
	Event *ControllerPairCreated // Event containing the contract specifics and raw log

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
func (it *ControllerPairCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ControllerPairCreated)
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
		it.Event = new(ControllerPairCreated)
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
func (it *ControllerPairCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ControllerPairCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ControllerPairCreated represents a PairCreated event raised by the Controller contract.
type ControllerPairCreated struct {
	PairId *big.Int
	Token0 common.Address
	Token1 common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterPairCreated is a free log retrieval operation binding the contract event 0x6365c594f5448f79c1cc1e6f661bdbf1d16f2e8f85747e13f8e80f1fd168b7c3.
//
// Solidity: event PairCreated(uint128 indexed pairId, address indexed token0, address indexed token1)
func (_Controller *ControllerFilterer) FilterPairCreated(opts *bind.FilterOpts, pairId []*big.Int, token0 []common.Address, token1 []common.Address) (*ControllerPairCreatedIterator, error) {

	var pairIdRule []interface{}
	for _, pairIdItem := range pairId {
		pairIdRule = append(pairIdRule, pairIdItem)
	}
	var token0Rule []interface{}
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []interface{}
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}

	logs, sub, err := _Controller.contract.FilterLogs(opts, "PairCreated", pairIdRule, token0Rule, token1Rule)
	if err != nil {
		return nil, err
	}
	return &ControllerPairCreatedIterator{contract: _Controller.contract, event: "PairCreated", logs: logs, sub: sub}, nil
}

// WatchPairCreated is a free log subscription operation binding the contract event 0x6365c594f5448f79c1cc1e6f661bdbf1d16f2e8f85747e13f8e80f1fd168b7c3.
//
// Solidity: event PairCreated(uint128 indexed pairId, address indexed token0, address indexed token1)
func (_Controller *ControllerFilterer) WatchPairCreated(opts *bind.WatchOpts, sink chan<- *ControllerPairCreated, pairId []*big.Int, token0 []common.Address, token1 []common.Address) (event.Subscription, error) {

	var pairIdRule []interface{}
	for _, pairIdItem := range pairId {
		pairIdRule = append(pairIdRule, pairIdItem)
	}
	var token0Rule []interface{}
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []interface{}
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}

	logs, sub, err := _Controller.contract.WatchLogs(opts, "PairCreated", pairIdRule, token0Rule, token1Rule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ControllerPairCreated)
				if err := _Controller.contract.UnpackLog(event, "PairCreated", log); err != nil {
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

// ParsePairCreated is a log parse operation binding the contract event 0x6365c594f5448f79c1cc1e6f661bdbf1d16f2e8f85747e13f8e80f1fd168b7c3.
//
// Solidity: event PairCreated(uint128 indexed pairId, address indexed token0, address indexed token1)
func (_Controller *ControllerFilterer) ParsePairCreated(log types.Log) (*ControllerPairCreated, error) {
	event := new(ControllerPairCreated)
	if err := _Controller.contract.UnpackLog(event, "PairCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ControllerPairTradingFeePPMUpdatedIterator is returned from FilterPairTradingFeePPMUpdated and is used to iterate over the raw logs and unpacked data for PairTradingFeePPMUpdated events raised by the Controller contract.
type ControllerPairTradingFeePPMUpdatedIterator struct {
	Event *ControllerPairTradingFeePPMUpdated // Event containing the contract specifics and raw log

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
func (it *ControllerPairTradingFeePPMUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ControllerPairTradingFeePPMUpdated)
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
		it.Event = new(ControllerPairTradingFeePPMUpdated)
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
func (it *ControllerPairTradingFeePPMUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ControllerPairTradingFeePPMUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ControllerPairTradingFeePPMUpdated represents a PairTradingFeePPMUpdated event raised by the Controller contract.
type ControllerPairTradingFeePPMUpdated struct {
	Token0     common.Address
	Token1     common.Address
	PrevFeePPM uint32
	NewFeePPM  uint32
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterPairTradingFeePPMUpdated is a free log retrieval operation binding the contract event 0x831434d05f3ad5f63be733ea463b2933c70d2162697fd200a22b5d56f5c454b6.
//
// Solidity: event PairTradingFeePPMUpdated(address indexed token0, address indexed token1, uint32 prevFeePPM, uint32 newFeePPM)
func (_Controller *ControllerFilterer) FilterPairTradingFeePPMUpdated(opts *bind.FilterOpts, token0 []common.Address, token1 []common.Address) (*ControllerPairTradingFeePPMUpdatedIterator, error) {

	var token0Rule []interface{}
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []interface{}
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}

	logs, sub, err := _Controller.contract.FilterLogs(opts, "PairTradingFeePPMUpdated", token0Rule, token1Rule)
	if err != nil {
		return nil, err
	}
	return &ControllerPairTradingFeePPMUpdatedIterator{contract: _Controller.contract, event: "PairTradingFeePPMUpdated", logs: logs, sub: sub}, nil
}

// WatchPairTradingFeePPMUpdated is a free log subscription operation binding the contract event 0x831434d05f3ad5f63be733ea463b2933c70d2162697fd200a22b5d56f5c454b6.
//
// Solidity: event PairTradingFeePPMUpdated(address indexed token0, address indexed token1, uint32 prevFeePPM, uint32 newFeePPM)
func (_Controller *ControllerFilterer) WatchPairTradingFeePPMUpdated(opts *bind.WatchOpts, sink chan<- *ControllerPairTradingFeePPMUpdated, token0 []common.Address, token1 []common.Address) (event.Subscription, error) {

	var token0Rule []interface{}
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []interface{}
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}

	logs, sub, err := _Controller.contract.WatchLogs(opts, "PairTradingFeePPMUpdated", token0Rule, token1Rule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ControllerPairTradingFeePPMUpdated)
				if err := _Controller.contract.UnpackLog(event, "PairTradingFeePPMUpdated", log); err != nil {
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

// ParsePairTradingFeePPMUpdated is a log parse operation binding the contract event 0x831434d05f3ad5f63be733ea463b2933c70d2162697fd200a22b5d56f5c454b6.
//
// Solidity: event PairTradingFeePPMUpdated(address indexed token0, address indexed token1, uint32 prevFeePPM, uint32 newFeePPM)
func (_Controller *ControllerFilterer) ParsePairTradingFeePPMUpdated(log types.Log) (*ControllerPairTradingFeePPMUpdated, error) {
	event := new(ControllerPairTradingFeePPMUpdated)
	if err := _Controller.contract.UnpackLog(event, "PairTradingFeePPMUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ControllerStrategyCreatedIterator is returned from FilterStrategyCreated and is used to iterate over the raw logs and unpacked data for StrategyCreated events raised by the Controller contract.
type ControllerStrategyCreatedIterator struct {
	Event *ControllerStrategyCreated // Event containing the contract specifics and raw log

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
func (it *ControllerStrategyCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ControllerStrategyCreated)
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
		it.Event = new(ControllerStrategyCreated)
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
func (it *ControllerStrategyCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ControllerStrategyCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ControllerStrategyCreated represents a StrategyCreated event raised by the Controller contract.
type ControllerStrategyCreated struct {
	Id     *big.Int
	Owner  common.Address
	Token0 common.Address
	Token1 common.Address
	Order0 Order
	Order1 Order
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterStrategyCreated is a free log retrieval operation binding the contract event 0xff24554f8ccfe540435cfc8854831f8dcf1cf2068708cfaf46e8b52a4ccc4c8d.
//
// Solidity: event StrategyCreated(uint256 id, address indexed owner, address indexed token0, address indexed token1, (uint128,uint128,uint64,uint64) order0, (uint128,uint128,uint64,uint64) order1)
func (_Controller *ControllerFilterer) FilterStrategyCreated(opts *bind.FilterOpts, owner []common.Address, token0 []common.Address, token1 []common.Address) (*ControllerStrategyCreatedIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var token0Rule []interface{}
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []interface{}
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}

	logs, sub, err := _Controller.contract.FilterLogs(opts, "StrategyCreated", ownerRule, token0Rule, token1Rule)
	if err != nil {
		return nil, err
	}
	return &ControllerStrategyCreatedIterator{contract: _Controller.contract, event: "StrategyCreated", logs: logs, sub: sub}, nil
}

// WatchStrategyCreated is a free log subscription operation binding the contract event 0xff24554f8ccfe540435cfc8854831f8dcf1cf2068708cfaf46e8b52a4ccc4c8d.
//
// Solidity: event StrategyCreated(uint256 id, address indexed owner, address indexed token0, address indexed token1, (uint128,uint128,uint64,uint64) order0, (uint128,uint128,uint64,uint64) order1)
func (_Controller *ControllerFilterer) WatchStrategyCreated(opts *bind.WatchOpts, sink chan<- *ControllerStrategyCreated, owner []common.Address, token0 []common.Address, token1 []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var token0Rule []interface{}
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []interface{}
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}

	logs, sub, err := _Controller.contract.WatchLogs(opts, "StrategyCreated", ownerRule, token0Rule, token1Rule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ControllerStrategyCreated)
				if err := _Controller.contract.UnpackLog(event, "StrategyCreated", log); err != nil {
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

// ParseStrategyCreated is a log parse operation binding the contract event 0xff24554f8ccfe540435cfc8854831f8dcf1cf2068708cfaf46e8b52a4ccc4c8d.
//
// Solidity: event StrategyCreated(uint256 id, address indexed owner, address indexed token0, address indexed token1, (uint128,uint128,uint64,uint64) order0, (uint128,uint128,uint64,uint64) order1)
func (_Controller *ControllerFilterer) ParseStrategyCreated(log types.Log) (*ControllerStrategyCreated, error) {
	event := new(ControllerStrategyCreated)
	if err := _Controller.contract.UnpackLog(event, "StrategyCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ControllerStrategyDeletedIterator is returned from FilterStrategyDeleted and is used to iterate over the raw logs and unpacked data for StrategyDeleted events raised by the Controller contract.
type ControllerStrategyDeletedIterator struct {
	Event *ControllerStrategyDeleted // Event containing the contract specifics and raw log

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
func (it *ControllerStrategyDeletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ControllerStrategyDeleted)
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
		it.Event = new(ControllerStrategyDeleted)
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
func (it *ControllerStrategyDeletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ControllerStrategyDeletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ControllerStrategyDeleted represents a StrategyDeleted event raised by the Controller contract.
type ControllerStrategyDeleted struct {
	Id     *big.Int
	Owner  common.Address
	Token0 common.Address
	Token1 common.Address
	Order0 Order
	Order1 Order
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterStrategyDeleted is a free log retrieval operation binding the contract event 0x4d5b6e0627ea711d8e9312b6ba56f50e0b51d41816fd6fd38643495ac81d38b6.
//
// Solidity: event StrategyDeleted(uint256 id, address indexed owner, address indexed token0, address indexed token1, (uint128,uint128,uint64,uint64) order0, (uint128,uint128,uint64,uint64) order1)
func (_Controller *ControllerFilterer) FilterStrategyDeleted(opts *bind.FilterOpts, owner []common.Address, token0 []common.Address, token1 []common.Address) (*ControllerStrategyDeletedIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var token0Rule []interface{}
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []interface{}
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}

	logs, sub, err := _Controller.contract.FilterLogs(opts, "StrategyDeleted", ownerRule, token0Rule, token1Rule)
	if err != nil {
		return nil, err
	}
	return &ControllerStrategyDeletedIterator{contract: _Controller.contract, event: "StrategyDeleted", logs: logs, sub: sub}, nil
}

// WatchStrategyDeleted is a free log subscription operation binding the contract event 0x4d5b6e0627ea711d8e9312b6ba56f50e0b51d41816fd6fd38643495ac81d38b6.
//
// Solidity: event StrategyDeleted(uint256 id, address indexed owner, address indexed token0, address indexed token1, (uint128,uint128,uint64,uint64) order0, (uint128,uint128,uint64,uint64) order1)
func (_Controller *ControllerFilterer) WatchStrategyDeleted(opts *bind.WatchOpts, sink chan<- *ControllerStrategyDeleted, owner []common.Address, token0 []common.Address, token1 []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var token0Rule []interface{}
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []interface{}
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}

	logs, sub, err := _Controller.contract.WatchLogs(opts, "StrategyDeleted", ownerRule, token0Rule, token1Rule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ControllerStrategyDeleted)
				if err := _Controller.contract.UnpackLog(event, "StrategyDeleted", log); err != nil {
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

// ParseStrategyDeleted is a log parse operation binding the contract event 0x4d5b6e0627ea711d8e9312b6ba56f50e0b51d41816fd6fd38643495ac81d38b6.
//
// Solidity: event StrategyDeleted(uint256 id, address indexed owner, address indexed token0, address indexed token1, (uint128,uint128,uint64,uint64) order0, (uint128,uint128,uint64,uint64) order1)
func (_Controller *ControllerFilterer) ParseStrategyDeleted(log types.Log) (*ControllerStrategyDeleted, error) {
	event := new(ControllerStrategyDeleted)
	if err := _Controller.contract.UnpackLog(event, "StrategyDeleted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ControllerStrategyUpdatedIterator is returned from FilterStrategyUpdated and is used to iterate over the raw logs and unpacked data for StrategyUpdated events raised by the Controller contract.
type ControllerStrategyUpdatedIterator struct {
	Event *ControllerStrategyUpdated // Event containing the contract specifics and raw log

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
func (it *ControllerStrategyUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ControllerStrategyUpdated)
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
		it.Event = new(ControllerStrategyUpdated)
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
func (it *ControllerStrategyUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ControllerStrategyUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ControllerStrategyUpdated represents a StrategyUpdated event raised by the Controller contract.
type ControllerStrategyUpdated struct {
	Id     *big.Int
	Token0 common.Address
	Token1 common.Address
	Order0 Order
	Order1 Order
	Reason uint8
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterStrategyUpdated is a free log retrieval operation binding the contract event 0x720da23a5c920b1d8827ec83c4d3c4d90d9419eadb0036b88cb4c2ffa91aef7d.
//
// Solidity: event StrategyUpdated(uint256 indexed id, address indexed token0, address indexed token1, (uint128,uint128,uint64,uint64) order0, (uint128,uint128,uint64,uint64) order1, uint8 reason)
func (_Controller *ControllerFilterer) FilterStrategyUpdated(opts *bind.FilterOpts, id []*big.Int, token0 []common.Address, token1 []common.Address) (*ControllerStrategyUpdatedIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var token0Rule []interface{}
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []interface{}
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}

	logs, sub, err := _Controller.contract.FilterLogs(opts, "StrategyUpdated", idRule, token0Rule, token1Rule)
	if err != nil {
		return nil, err
	}
	return &ControllerStrategyUpdatedIterator{contract: _Controller.contract, event: "StrategyUpdated", logs: logs, sub: sub}, nil
}

// WatchStrategyUpdated is a free log subscription operation binding the contract event 0x720da23a5c920b1d8827ec83c4d3c4d90d9419eadb0036b88cb4c2ffa91aef7d.
//
// Solidity: event StrategyUpdated(uint256 indexed id, address indexed token0, address indexed token1, (uint128,uint128,uint64,uint64) order0, (uint128,uint128,uint64,uint64) order1, uint8 reason)
func (_Controller *ControllerFilterer) WatchStrategyUpdated(opts *bind.WatchOpts, sink chan<- *ControllerStrategyUpdated, id []*big.Int, token0 []common.Address, token1 []common.Address) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var token0Rule []interface{}
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []interface{}
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}

	logs, sub, err := _Controller.contract.WatchLogs(opts, "StrategyUpdated", idRule, token0Rule, token1Rule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ControllerStrategyUpdated)
				if err := _Controller.contract.UnpackLog(event, "StrategyUpdated", log); err != nil {
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

// ParseStrategyUpdated is a log parse operation binding the contract event 0x720da23a5c920b1d8827ec83c4d3c4d90d9419eadb0036b88cb4c2ffa91aef7d.
//
// Solidity: event StrategyUpdated(uint256 indexed id, address indexed token0, address indexed token1, (uint128,uint128,uint64,uint64) order0, (uint128,uint128,uint64,uint64) order1, uint8 reason)
func (_Controller *ControllerFilterer) ParseStrategyUpdated(log types.Log) (*ControllerStrategyUpdated, error) {
	event := new(ControllerStrategyUpdated)
	if err := _Controller.contract.UnpackLog(event, "StrategyUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ControllerTokensTradedIterator is returned from FilterTokensTraded and is used to iterate over the raw logs and unpacked data for TokensTraded events raised by the Controller contract.
type ControllerTokensTradedIterator struct {
	Event *ControllerTokensTraded // Event containing the contract specifics and raw log

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
func (it *ControllerTokensTradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ControllerTokensTraded)
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
		it.Event = new(ControllerTokensTraded)
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
func (it *ControllerTokensTradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ControllerTokensTradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ControllerTokensTraded represents a TokensTraded event raised by the Controller contract.
type ControllerTokensTraded struct {
	Trader           common.Address
	SourceToken      common.Address
	TargetToken      common.Address
	SourceAmount     *big.Int
	TargetAmount     *big.Int
	TradingFeeAmount *big.Int
	ByTargetAmount   bool
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterTokensTraded is a free log retrieval operation binding the contract event 0x95f3b01351225fea0e69a46f68b164c9dea10284f12cd4a907ce66510ab7af6a.
//
// Solidity: event TokensTraded(address indexed trader, address indexed sourceToken, address indexed targetToken, uint256 sourceAmount, uint256 targetAmount, uint128 tradingFeeAmount, bool byTargetAmount)
func (_Controller *ControllerFilterer) FilterTokensTraded(opts *bind.FilterOpts, trader []common.Address, sourceToken []common.Address, targetToken []common.Address) (*ControllerTokensTradedIterator, error) {

	var traderRule []interface{}
	for _, traderItem := range trader {
		traderRule = append(traderRule, traderItem)
	}
	var sourceTokenRule []interface{}
	for _, sourceTokenItem := range sourceToken {
		sourceTokenRule = append(sourceTokenRule, sourceTokenItem)
	}
	var targetTokenRule []interface{}
	for _, targetTokenItem := range targetToken {
		targetTokenRule = append(targetTokenRule, targetTokenItem)
	}

	logs, sub, err := _Controller.contract.FilterLogs(opts, "TokensTraded", traderRule, sourceTokenRule, targetTokenRule)
	if err != nil {
		return nil, err
	}
	return &ControllerTokensTradedIterator{contract: _Controller.contract, event: "TokensTraded", logs: logs, sub: sub}, nil
}

// WatchTokensTraded is a free log subscription operation binding the contract event 0x95f3b01351225fea0e69a46f68b164c9dea10284f12cd4a907ce66510ab7af6a.
//
// Solidity: event TokensTraded(address indexed trader, address indexed sourceToken, address indexed targetToken, uint256 sourceAmount, uint256 targetAmount, uint128 tradingFeeAmount, bool byTargetAmount)
func (_Controller *ControllerFilterer) WatchTokensTraded(opts *bind.WatchOpts, sink chan<- *ControllerTokensTraded, trader []common.Address, sourceToken []common.Address, targetToken []common.Address) (event.Subscription, error) {

	var traderRule []interface{}
	for _, traderItem := range trader {
		traderRule = append(traderRule, traderItem)
	}
	var sourceTokenRule []interface{}
	for _, sourceTokenItem := range sourceToken {
		sourceTokenRule = append(sourceTokenRule, sourceTokenItem)
	}
	var targetTokenRule []interface{}
	for _, targetTokenItem := range targetToken {
		targetTokenRule = append(targetTokenRule, targetTokenItem)
	}

	logs, sub, err := _Controller.contract.WatchLogs(opts, "TokensTraded", traderRule, sourceTokenRule, targetTokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ControllerTokensTraded)
				if err := _Controller.contract.UnpackLog(event, "TokensTraded", log); err != nil {
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

// ParseTokensTraded is a log parse operation binding the contract event 0x95f3b01351225fea0e69a46f68b164c9dea10284f12cd4a907ce66510ab7af6a.
//
// Solidity: event TokensTraded(address indexed trader, address indexed sourceToken, address indexed targetToken, uint256 sourceAmount, uint256 targetAmount, uint128 tradingFeeAmount, bool byTargetAmount)
func (_Controller *ControllerFilterer) ParseTokensTraded(log types.Log) (*ControllerTokensTraded, error) {
	event := new(ControllerTokensTraded)
	if err := _Controller.contract.UnpackLog(event, "TokensTraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ControllerTradingFeePPMUpdatedIterator is returned from FilterTradingFeePPMUpdated and is used to iterate over the raw logs and unpacked data for TradingFeePPMUpdated events raised by the Controller contract.
type ControllerTradingFeePPMUpdatedIterator struct {
	Event *ControllerTradingFeePPMUpdated // Event containing the contract specifics and raw log

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
func (it *ControllerTradingFeePPMUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ControllerTradingFeePPMUpdated)
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
		it.Event = new(ControllerTradingFeePPMUpdated)
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
func (it *ControllerTradingFeePPMUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ControllerTradingFeePPMUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ControllerTradingFeePPMUpdated represents a TradingFeePPMUpdated event raised by the Controller contract.
type ControllerTradingFeePPMUpdated struct {
	PrevFeePPM uint32
	NewFeePPM  uint32
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterTradingFeePPMUpdated is a free log retrieval operation binding the contract event 0x66db0986e1156e2e747795714bf0301c7e1c695c149a738cb01bcf5cfead8465.
//
// Solidity: event TradingFeePPMUpdated(uint32 prevFeePPM, uint32 newFeePPM)
func (_Controller *ControllerFilterer) FilterTradingFeePPMUpdated(opts *bind.FilterOpts) (*ControllerTradingFeePPMUpdatedIterator, error) {

	logs, sub, err := _Controller.contract.FilterLogs(opts, "TradingFeePPMUpdated")
	if err != nil {
		return nil, err
	}
	return &ControllerTradingFeePPMUpdatedIterator{contract: _Controller.contract, event: "TradingFeePPMUpdated", logs: logs, sub: sub}, nil
}

// WatchTradingFeePPMUpdated is a free log subscription operation binding the contract event 0x66db0986e1156e2e747795714bf0301c7e1c695c149a738cb01bcf5cfead8465.
//
// Solidity: event TradingFeePPMUpdated(uint32 prevFeePPM, uint32 newFeePPM)
func (_Controller *ControllerFilterer) WatchTradingFeePPMUpdated(opts *bind.WatchOpts, sink chan<- *ControllerTradingFeePPMUpdated) (event.Subscription, error) {

	logs, sub, err := _Controller.contract.WatchLogs(opts, "TradingFeePPMUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ControllerTradingFeePPMUpdated)
				if err := _Controller.contract.UnpackLog(event, "TradingFeePPMUpdated", log); err != nil {
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

// ParseTradingFeePPMUpdated is a log parse operation binding the contract event 0x66db0986e1156e2e747795714bf0301c7e1c695c149a738cb01bcf5cfead8465.
//
// Solidity: event TradingFeePPMUpdated(uint32 prevFeePPM, uint32 newFeePPM)
func (_Controller *ControllerFilterer) ParseTradingFeePPMUpdated(log types.Log) (*ControllerTradingFeePPMUpdated, error) {
	event := new(ControllerTradingFeePPMUpdated)
	if err := _Controller.contract.UnpackLog(event, "TradingFeePPMUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
