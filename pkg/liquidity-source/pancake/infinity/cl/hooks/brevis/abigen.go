// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package brevis

import (
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
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

// BrevisMetaData contains all meta data concerning the Brevis contract.
var BrevisMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"origFee\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// BrevisABI is the input ABI used to generate the binding from.
// Deprecated: Use BrevisMetaData.ABI instead.
var BrevisABI = BrevisMetaData.ABI

// Brevis is an auto generated Go binding around an Ethereum contract.
type Brevis struct {
	BrevisCaller     // Read-only binding to the contract
	BrevisTransactor // Write-only binding to the contract
	BrevisFilterer   // Log filterer for contract events
}

// BrevisCaller is an auto generated read-only Go binding around an Ethereum contract.
type BrevisCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BrevisTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BrevisTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BrevisFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BrevisFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BrevisSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BrevisSession struct {
	Contract     *Brevis           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BrevisCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BrevisCallerSession struct {
	Contract *BrevisCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// BrevisTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BrevisTransactorSession struct {
	Contract     *BrevisTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BrevisRaw is an auto generated low-level Go binding around an Ethereum contract.
type BrevisRaw struct {
	Contract *Brevis // Generic contract binding to access the raw methods on
}

// BrevisCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BrevisCallerRaw struct {
	Contract *BrevisCaller // Generic read-only contract binding to access the raw methods on
}

// BrevisTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BrevisTransactorRaw struct {
	Contract *BrevisTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBrevis creates a new instance of Brevis, bound to a specific deployed contract.
func NewBrevis(address common.Address, backend bind.ContractBackend) (*Brevis, error) {
	contract, err := bindBrevis(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Brevis{BrevisCaller: BrevisCaller{contract: contract}, BrevisTransactor: BrevisTransactor{contract: contract}, BrevisFilterer: BrevisFilterer{contract: contract}}, nil
}

// NewBrevisCaller creates a new read-only instance of Brevis, bound to a specific deployed contract.
func NewBrevisCaller(address common.Address, caller bind.ContractCaller) (*BrevisCaller, error) {
	contract, err := bindBrevis(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BrevisCaller{contract: contract}, nil
}

// NewBrevisTransactor creates a new write-only instance of Brevis, bound to a specific deployed contract.
func NewBrevisTransactor(address common.Address, transactor bind.ContractTransactor) (*BrevisTransactor, error) {
	contract, err := bindBrevis(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BrevisTransactor{contract: contract}, nil
}

// NewBrevisFilterer creates a new log filterer instance of Brevis, bound to a specific deployed contract.
func NewBrevisFilterer(address common.Address, filterer bind.ContractFilterer) (*BrevisFilterer, error) {
	contract, err := bindBrevis(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BrevisFilterer{contract: contract}, nil
}

// bindBrevis binds a generic wrapper to an already deployed contract.
func bindBrevis(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BrevisMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Brevis *BrevisRaw) Call(opts *bind.CallOpts, result *[]any, method string, params ...any) error {
	return _Brevis.Contract.BrevisCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Brevis *BrevisRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Brevis.Contract.BrevisTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Brevis *BrevisRaw) Transact(opts *bind.TransactOpts, method string, params ...any) (*types.Transaction, error) {
	return _Brevis.Contract.BrevisTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Brevis *BrevisCallerRaw) Call(opts *bind.CallOpts, result *[]any, method string, params ...any) error {
	return _Brevis.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Brevis *BrevisTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Brevis.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Brevis *BrevisTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...any) (*types.Transaction, error) {
	return _Brevis.Contract.contract.Transact(opts, method, params...)
}

// OrigFee is a free data retrieval call binding the contract method 0x92b56fa7.
//
// Solidity: function origFee() view returns(uint24 fee)
func (_Brevis *BrevisCaller) OrigFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _Brevis.contract.Call(opts, &out, "origFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// OrigFee is a free data retrieval call binding the contract method 0x92b56fa7.
//
// Solidity: function origFee() view returns(uint24 fee)
func (_Brevis *BrevisSession) OrigFee() (*big.Int, error) {
	return _Brevis.Contract.OrigFee(&_Brevis.CallOpts)
}

// OrigFee is a free data retrieval call binding the contract method 0x92b56fa7.
//
// Solidity: function origFee() view returns(uint24 fee)
func (_Brevis *BrevisCallerSession) OrigFee() (*big.Int, error) {
	return _Brevis.Contract.OrigFee(&_Brevis.CallOpts)
}
