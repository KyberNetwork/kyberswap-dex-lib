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

// DPPFactoryMetaData contains all meta data concerning the DPPFactory contract.
var DPPFactoryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"cloneFactory\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"dppTemplate\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"dppAdminTemplate\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"defaultMaintainer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"defaultMtFeeRateModel\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"dodoApproveProxy\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"dpp\",\"type\":\"address\"}],\"name\":\"NewDPP\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferPrepared\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"dpp\",\"type\":\"address\"}],\"name\":\"RemoveDPP\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"admin\",\"type\":\"address\"}],\"name\":\"addAdmin\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"admin\",\"type\":\"address\"}],\"name\":\"removeAdmin\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"_CLONE_FACTORY_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_DEFAULT_MAINTAINER_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_DEFAULT_MT_FEE_RATE_MODEL_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_DODO_APPROVE_PROXY_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_DPP_ADMIN_TEMPLATE_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_DPP_TEMPLATE_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_NEW_OWNER_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_OWNER_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"_REGISTRY_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"_USER_REGISTRY_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"contractAddr\",\"type\":\"address\"}],\"name\":\"addAdminList\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"addPoolByAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"creators\",\"type\":\"address[]\"},{\"internalType\":\"address[]\",\"name\":\"baseTokens\",\"type\":\"address[]\"},{\"internalType\":\"address[]\",\"name\":\"quoteTokens\",\"type\":\"address[]\"},{\"internalType\":\"address[]\",\"name\":\"pools\",\"type\":\"address[]\"}],\"name\":\"batchAddPoolByAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"claimOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"createDODOPrivatePool\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"newPrivatePool\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"}],\"name\":\"getDODOPool\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"pools\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"}],\"name\":\"getDODOPoolBidirection\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"baseToken0Pool\",\"type\":\"address[]\"},{\"internalType\":\"address[]\",\"name\":\"baseToken1Pool\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"}],\"name\":\"getDODOPoolByUser\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"pools\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"dppAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"lpFeeRate\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"k\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"i\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isOpenTwap\",\"type\":\"bool\"}],\"name\":\"initDODOPrivatePool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"initOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isAdminListed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"contractAddr\",\"type\":\"address\"}],\"name\":\"removeAdminList\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"removePoolByAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newDPPAdminTemplate\",\"type\":\"address\"}],\"name\":\"updateAdminTemplate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newMaintainer\",\"type\":\"address\"}],\"name\":\"updateDefaultMaintainer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newDPPTemplate\",\"type\":\"address\"}],\"name\":\"updateDppTemplate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// DPPFactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use DPPFactoryMetaData.ABI instead.
var DPPFactoryABI = DPPFactoryMetaData.ABI

// DPPFactory is an auto generated Go binding around an Ethereum contract.
type DPPFactory struct {
	DPPFactoryCaller     // Read-only binding to the contract
	DPPFactoryTransactor // Write-only binding to the contract
	DPPFactoryFilterer   // Log filterer for contract events
}

// DPPFactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type DPPFactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DPPFactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DPPFactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DPPFactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DPPFactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DPPFactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DPPFactorySession struct {
	Contract     *DPPFactory       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DPPFactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DPPFactoryCallerSession struct {
	Contract *DPPFactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// DPPFactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DPPFactoryTransactorSession struct {
	Contract     *DPPFactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// DPPFactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type DPPFactoryRaw struct {
	Contract *DPPFactory // Generic contract binding to access the raw methods on
}

// DPPFactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DPPFactoryCallerRaw struct {
	Contract *DPPFactoryCaller // Generic read-only contract binding to access the raw methods on
}

// DPPFactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DPPFactoryTransactorRaw struct {
	Contract *DPPFactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDPPFactory creates a new instance of DPPFactory, bound to a specific deployed contract.
func NewDPPFactory(address common.Address, backend bind.ContractBackend) (*DPPFactory, error) {
	contract, err := bindDPPFactory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DPPFactory{DPPFactoryCaller: DPPFactoryCaller{contract: contract}, DPPFactoryTransactor: DPPFactoryTransactor{contract: contract}, DPPFactoryFilterer: DPPFactoryFilterer{contract: contract}}, nil
}

// NewDPPFactoryCaller creates a new read-only instance of DPPFactory, bound to a specific deployed contract.
func NewDPPFactoryCaller(address common.Address, caller bind.ContractCaller) (*DPPFactoryCaller, error) {
	contract, err := bindDPPFactory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DPPFactoryCaller{contract: contract}, nil
}

// NewDPPFactoryTransactor creates a new write-only instance of DPPFactory, bound to a specific deployed contract.
func NewDPPFactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*DPPFactoryTransactor, error) {
	contract, err := bindDPPFactory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DPPFactoryTransactor{contract: contract}, nil
}

// NewDPPFactoryFilterer creates a new log filterer instance of DPPFactory, bound to a specific deployed contract.
func NewDPPFactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*DPPFactoryFilterer, error) {
	contract, err := bindDPPFactory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DPPFactoryFilterer{contract: contract}, nil
}

// bindDPPFactory binds a generic wrapper to an already deployed contract.
func bindDPPFactory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := DPPFactoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DPPFactory *DPPFactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DPPFactory.Contract.DPPFactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DPPFactory *DPPFactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DPPFactory.Contract.DPPFactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DPPFactory *DPPFactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DPPFactory.Contract.DPPFactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DPPFactory *DPPFactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DPPFactory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DPPFactory *DPPFactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DPPFactory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DPPFactory *DPPFactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DPPFactory.Contract.contract.Transact(opts, method, params...)
}

// CLONEFACTORY is a free data retrieval call binding the contract method 0xeb774d05.
//
// Solidity: function _CLONE_FACTORY_() view returns(address)
func (_DPPFactory *DPPFactoryCaller) CLONEFACTORY(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DPPFactory.contract.Call(opts, &out, "_CLONE_FACTORY_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CLONEFACTORY is a free data retrieval call binding the contract method 0xeb774d05.
//
// Solidity: function _CLONE_FACTORY_() view returns(address)
func (_DPPFactory *DPPFactorySession) CLONEFACTORY() (common.Address, error) {
	return _DPPFactory.Contract.CLONEFACTORY(&_DPPFactory.CallOpts)
}

// CLONEFACTORY is a free data retrieval call binding the contract method 0xeb774d05.
//
// Solidity: function _CLONE_FACTORY_() view returns(address)
func (_DPPFactory *DPPFactoryCallerSession) CLONEFACTORY() (common.Address, error) {
	return _DPPFactory.Contract.CLONEFACTORY(&_DPPFactory.CallOpts)
}

// DEFAULTMAINTAINER is a free data retrieval call binding the contract method 0x81ab4d0a.
//
// Solidity: function _DEFAULT_MAINTAINER_() view returns(address)
func (_DPPFactory *DPPFactoryCaller) DEFAULTMAINTAINER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DPPFactory.contract.Call(opts, &out, "_DEFAULT_MAINTAINER_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DEFAULTMAINTAINER is a free data retrieval call binding the contract method 0x81ab4d0a.
//
// Solidity: function _DEFAULT_MAINTAINER_() view returns(address)
func (_DPPFactory *DPPFactorySession) DEFAULTMAINTAINER() (common.Address, error) {
	return _DPPFactory.Contract.DEFAULTMAINTAINER(&_DPPFactory.CallOpts)
}

// DEFAULTMAINTAINER is a free data retrieval call binding the contract method 0x81ab4d0a.
//
// Solidity: function _DEFAULT_MAINTAINER_() view returns(address)
func (_DPPFactory *DPPFactoryCallerSession) DEFAULTMAINTAINER() (common.Address, error) {
	return _DPPFactory.Contract.DEFAULTMAINTAINER(&_DPPFactory.CallOpts)
}

// DEFAULTMTFEERATEMODEL is a free data retrieval call binding the contract method 0x6c5ccb9b.
//
// Solidity: function _DEFAULT_MT_FEE_RATE_MODEL_() view returns(address)
func (_DPPFactory *DPPFactoryCaller) DEFAULTMTFEERATEMODEL(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DPPFactory.contract.Call(opts, &out, "_DEFAULT_MT_FEE_RATE_MODEL_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DEFAULTMTFEERATEMODEL is a free data retrieval call binding the contract method 0x6c5ccb9b.
//
// Solidity: function _DEFAULT_MT_FEE_RATE_MODEL_() view returns(address)
func (_DPPFactory *DPPFactorySession) DEFAULTMTFEERATEMODEL() (common.Address, error) {
	return _DPPFactory.Contract.DEFAULTMTFEERATEMODEL(&_DPPFactory.CallOpts)
}

// DEFAULTMTFEERATEMODEL is a free data retrieval call binding the contract method 0x6c5ccb9b.
//
// Solidity: function _DEFAULT_MT_FEE_RATE_MODEL_() view returns(address)
func (_DPPFactory *DPPFactoryCallerSession) DEFAULTMTFEERATEMODEL() (common.Address, error) {
	return _DPPFactory.Contract.DEFAULTMTFEERATEMODEL(&_DPPFactory.CallOpts)
}

// DODOAPPROVEPROXY is a free data retrieval call binding the contract method 0xeb99be12.
//
// Solidity: function _DODO_APPROVE_PROXY_() view returns(address)
func (_DPPFactory *DPPFactoryCaller) DODOAPPROVEPROXY(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DPPFactory.contract.Call(opts, &out, "_DODO_APPROVE_PROXY_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DODOAPPROVEPROXY is a free data retrieval call binding the contract method 0xeb99be12.
//
// Solidity: function _DODO_APPROVE_PROXY_() view returns(address)
func (_DPPFactory *DPPFactorySession) DODOAPPROVEPROXY() (common.Address, error) {
	return _DPPFactory.Contract.DODOAPPROVEPROXY(&_DPPFactory.CallOpts)
}

// DODOAPPROVEPROXY is a free data retrieval call binding the contract method 0xeb99be12.
//
// Solidity: function _DODO_APPROVE_PROXY_() view returns(address)
func (_DPPFactory *DPPFactoryCallerSession) DODOAPPROVEPROXY() (common.Address, error) {
	return _DPPFactory.Contract.DODOAPPROVEPROXY(&_DPPFactory.CallOpts)
}

// DPPADMINTEMPLATE is a free data retrieval call binding the contract method 0x633644d6.
//
// Solidity: function _DPP_ADMIN_TEMPLATE_() view returns(address)
func (_DPPFactory *DPPFactoryCaller) DPPADMINTEMPLATE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DPPFactory.contract.Call(opts, &out, "_DPP_ADMIN_TEMPLATE_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DPPADMINTEMPLATE is a free data retrieval call binding the contract method 0x633644d6.
//
// Solidity: function _DPP_ADMIN_TEMPLATE_() view returns(address)
func (_DPPFactory *DPPFactorySession) DPPADMINTEMPLATE() (common.Address, error) {
	return _DPPFactory.Contract.DPPADMINTEMPLATE(&_DPPFactory.CallOpts)
}

// DPPADMINTEMPLATE is a free data retrieval call binding the contract method 0x633644d6.
//
// Solidity: function _DPP_ADMIN_TEMPLATE_() view returns(address)
func (_DPPFactory *DPPFactoryCallerSession) DPPADMINTEMPLATE() (common.Address, error) {
	return _DPPFactory.Contract.DPPADMINTEMPLATE(&_DPPFactory.CallOpts)
}

// DPPTEMPLATE is a free data retrieval call binding the contract method 0xace378ca.
//
// Solidity: function _DPP_TEMPLATE_() view returns(address)
func (_DPPFactory *DPPFactoryCaller) DPPTEMPLATE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DPPFactory.contract.Call(opts, &out, "_DPP_TEMPLATE_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DPPTEMPLATE is a free data retrieval call binding the contract method 0xace378ca.
//
// Solidity: function _DPP_TEMPLATE_() view returns(address)
func (_DPPFactory *DPPFactorySession) DPPTEMPLATE() (common.Address, error) {
	return _DPPFactory.Contract.DPPTEMPLATE(&_DPPFactory.CallOpts)
}

// DPPTEMPLATE is a free data retrieval call binding the contract method 0xace378ca.
//
// Solidity: function _DPP_TEMPLATE_() view returns(address)
func (_DPPFactory *DPPFactoryCallerSession) DPPTEMPLATE() (common.Address, error) {
	return _DPPFactory.Contract.DPPTEMPLATE(&_DPPFactory.CallOpts)
}

// NEWOWNER is a free data retrieval call binding the contract method 0x8456db15.
//
// Solidity: function _NEW_OWNER_() view returns(address)
func (_DPPFactory *DPPFactoryCaller) NEWOWNER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DPPFactory.contract.Call(opts, &out, "_NEW_OWNER_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NEWOWNER is a free data retrieval call binding the contract method 0x8456db15.
//
// Solidity: function _NEW_OWNER_() view returns(address)
func (_DPPFactory *DPPFactorySession) NEWOWNER() (common.Address, error) {
	return _DPPFactory.Contract.NEWOWNER(&_DPPFactory.CallOpts)
}

// NEWOWNER is a free data retrieval call binding the contract method 0x8456db15.
//
// Solidity: function _NEW_OWNER_() view returns(address)
func (_DPPFactory *DPPFactoryCallerSession) NEWOWNER() (common.Address, error) {
	return _DPPFactory.Contract.NEWOWNER(&_DPPFactory.CallOpts)
}

// OWNER is a free data retrieval call binding the contract method 0x16048bc4.
//
// Solidity: function _OWNER_() view returns(address)
func (_DPPFactory *DPPFactoryCaller) OWNER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DPPFactory.contract.Call(opts, &out, "_OWNER_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OWNER is a free data retrieval call binding the contract method 0x16048bc4.
//
// Solidity: function _OWNER_() view returns(address)
func (_DPPFactory *DPPFactorySession) OWNER() (common.Address, error) {
	return _DPPFactory.Contract.OWNER(&_DPPFactory.CallOpts)
}

// OWNER is a free data retrieval call binding the contract method 0x16048bc4.
//
// Solidity: function _OWNER_() view returns(address)
func (_DPPFactory *DPPFactoryCallerSession) OWNER() (common.Address, error) {
	return _DPPFactory.Contract.OWNER(&_DPPFactory.CallOpts)
}

// REGISTRY is a free data retrieval call binding the contract method 0xbdeb0a91.
//
// Solidity: function _REGISTRY_(address , address , uint256 ) view returns(address)
func (_DPPFactory *DPPFactoryCaller) REGISTRY(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _DPPFactory.contract.Call(opts, &out, "_REGISTRY_", arg0, arg1, arg2)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// REGISTRY is a free data retrieval call binding the contract method 0xbdeb0a91.
//
// Solidity: function _REGISTRY_(address , address , uint256 ) view returns(address)
func (_DPPFactory *DPPFactorySession) REGISTRY(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	return _DPPFactory.Contract.REGISTRY(&_DPPFactory.CallOpts, arg0, arg1, arg2)
}

// REGISTRY is a free data retrieval call binding the contract method 0xbdeb0a91.
//
// Solidity: function _REGISTRY_(address , address , uint256 ) view returns(address)
func (_DPPFactory *DPPFactoryCallerSession) REGISTRY(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	return _DPPFactory.Contract.REGISTRY(&_DPPFactory.CallOpts, arg0, arg1, arg2)
}

// USERREGISTRY is a free data retrieval call binding the contract method 0xa58888db.
//
// Solidity: function _USER_REGISTRY_(address , uint256 ) view returns(address)
func (_DPPFactory *DPPFactoryCaller) USERREGISTRY(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _DPPFactory.contract.Call(opts, &out, "_USER_REGISTRY_", arg0, arg1)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// USERREGISTRY is a free data retrieval call binding the contract method 0xa58888db.
//
// Solidity: function _USER_REGISTRY_(address , uint256 ) view returns(address)
func (_DPPFactory *DPPFactorySession) USERREGISTRY(arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	return _DPPFactory.Contract.USERREGISTRY(&_DPPFactory.CallOpts, arg0, arg1)
}

// USERREGISTRY is a free data retrieval call binding the contract method 0xa58888db.
//
// Solidity: function _USER_REGISTRY_(address , uint256 ) view returns(address)
func (_DPPFactory *DPPFactoryCallerSession) USERREGISTRY(arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	return _DPPFactory.Contract.USERREGISTRY(&_DPPFactory.CallOpts, arg0, arg1)
}

// GetDODOPool is a free data retrieval call binding the contract method 0x57a281dc.
//
// Solidity: function getDODOPool(address baseToken, address quoteToken) view returns(address[] pools)
func (_DPPFactory *DPPFactoryCaller) GetDODOPool(opts *bind.CallOpts, baseToken common.Address, quoteToken common.Address) ([]common.Address, error) {
	var out []interface{}
	err := _DPPFactory.contract.Call(opts, &out, "getDODOPool", baseToken, quoteToken)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetDODOPool is a free data retrieval call binding the contract method 0x57a281dc.
//
// Solidity: function getDODOPool(address baseToken, address quoteToken) view returns(address[] pools)
func (_DPPFactory *DPPFactorySession) GetDODOPool(baseToken common.Address, quoteToken common.Address) ([]common.Address, error) {
	return _DPPFactory.Contract.GetDODOPool(&_DPPFactory.CallOpts, baseToken, quoteToken)
}

// GetDODOPool is a free data retrieval call binding the contract method 0x57a281dc.
//
// Solidity: function getDODOPool(address baseToken, address quoteToken) view returns(address[] pools)
func (_DPPFactory *DPPFactoryCallerSession) GetDODOPool(baseToken common.Address, quoteToken common.Address) ([]common.Address, error) {
	return _DPPFactory.Contract.GetDODOPool(&_DPPFactory.CallOpts, baseToken, quoteToken)
}

// GetDODOPoolBidirection is a free data retrieval call binding the contract method 0x794e5538.
//
// Solidity: function getDODOPoolBidirection(address token0, address token1) view returns(address[] baseToken0Pool, address[] baseToken1Pool)
func (_DPPFactory *DPPFactoryCaller) GetDODOPoolBidirection(opts *bind.CallOpts, token0 common.Address, token1 common.Address) (struct {
	BaseToken0Pool []common.Address
	BaseToken1Pool []common.Address
}, error) {
	var out []interface{}
	err := _DPPFactory.contract.Call(opts, &out, "getDODOPoolBidirection", token0, token1)

	outstruct := new(struct {
		BaseToken0Pool []common.Address
		BaseToken1Pool []common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.BaseToken0Pool = *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)
	outstruct.BaseToken1Pool = *abi.ConvertType(out[1], new([]common.Address)).(*[]common.Address)

	return *outstruct, err

}

// GetDODOPoolBidirection is a free data retrieval call binding the contract method 0x794e5538.
//
// Solidity: function getDODOPoolBidirection(address token0, address token1) view returns(address[] baseToken0Pool, address[] baseToken1Pool)
func (_DPPFactory *DPPFactorySession) GetDODOPoolBidirection(token0 common.Address, token1 common.Address) (struct {
	BaseToken0Pool []common.Address
	BaseToken1Pool []common.Address
}, error) {
	return _DPPFactory.Contract.GetDODOPoolBidirection(&_DPPFactory.CallOpts, token0, token1)
}

// GetDODOPoolBidirection is a free data retrieval call binding the contract method 0x794e5538.
//
// Solidity: function getDODOPoolBidirection(address token0, address token1) view returns(address[] baseToken0Pool, address[] baseToken1Pool)
func (_DPPFactory *DPPFactoryCallerSession) GetDODOPoolBidirection(token0 common.Address, token1 common.Address) (struct {
	BaseToken0Pool []common.Address
	BaseToken1Pool []common.Address
}, error) {
	return _DPPFactory.Contract.GetDODOPoolBidirection(&_DPPFactory.CallOpts, token0, token1)
}

// GetDODOPoolByUser is a free data retrieval call binding the contract method 0xe65f7029.
//
// Solidity: function getDODOPoolByUser(address user) view returns(address[] pools)
func (_DPPFactory *DPPFactoryCaller) GetDODOPoolByUser(opts *bind.CallOpts, user common.Address) ([]common.Address, error) {
	var out []interface{}
	err := _DPPFactory.contract.Call(opts, &out, "getDODOPoolByUser", user)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetDODOPoolByUser is a free data retrieval call binding the contract method 0xe65f7029.
//
// Solidity: function getDODOPoolByUser(address user) view returns(address[] pools)
func (_DPPFactory *DPPFactorySession) GetDODOPoolByUser(user common.Address) ([]common.Address, error) {
	return _DPPFactory.Contract.GetDODOPoolByUser(&_DPPFactory.CallOpts, user)
}

// GetDODOPoolByUser is a free data retrieval call binding the contract method 0xe65f7029.
//
// Solidity: function getDODOPoolByUser(address user) view returns(address[] pools)
func (_DPPFactory *DPPFactoryCallerSession) GetDODOPoolByUser(user common.Address) ([]common.Address, error) {
	return _DPPFactory.Contract.GetDODOPoolByUser(&_DPPFactory.CallOpts, user)
}

// IsAdminListed is a free data retrieval call binding the contract method 0x1822c0c0.
//
// Solidity: function isAdminListed(address ) view returns(bool)
func (_DPPFactory *DPPFactoryCaller) IsAdminListed(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _DPPFactory.contract.Call(opts, &out, "isAdminListed", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAdminListed is a free data retrieval call binding the contract method 0x1822c0c0.
//
// Solidity: function isAdminListed(address ) view returns(bool)
func (_DPPFactory *DPPFactorySession) IsAdminListed(arg0 common.Address) (bool, error) {
	return _DPPFactory.Contract.IsAdminListed(&_DPPFactory.CallOpts, arg0)
}

// IsAdminListed is a free data retrieval call binding the contract method 0x1822c0c0.
//
// Solidity: function isAdminListed(address ) view returns(bool)
func (_DPPFactory *DPPFactoryCallerSession) IsAdminListed(arg0 common.Address) (bool, error) {
	return _DPPFactory.Contract.IsAdminListed(&_DPPFactory.CallOpts, arg0)
}

// AddAdminList is a paid mutator transaction binding the contract method 0xae52aae7.
//
// Solidity: function addAdminList(address contractAddr) returns()
func (_DPPFactory *DPPFactoryTransactor) AddAdminList(opts *bind.TransactOpts, contractAddr common.Address) (*types.Transaction, error) {
	return _DPPFactory.contract.Transact(opts, "addAdminList", contractAddr)
}

// AddAdminList is a paid mutator transaction binding the contract method 0xae52aae7.
//
// Solidity: function addAdminList(address contractAddr) returns()
func (_DPPFactory *DPPFactorySession) AddAdminList(contractAddr common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.AddAdminList(&_DPPFactory.TransactOpts, contractAddr)
}

// AddAdminList is a paid mutator transaction binding the contract method 0xae52aae7.
//
// Solidity: function addAdminList(address contractAddr) returns()
func (_DPPFactory *DPPFactoryTransactorSession) AddAdminList(contractAddr common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.AddAdminList(&_DPPFactory.TransactOpts, contractAddr)
}

// AddPoolByAdmin is a paid mutator transaction binding the contract method 0x39d00ef9.
//
// Solidity: function addPoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DPPFactory *DPPFactoryTransactor) AddPoolByAdmin(opts *bind.TransactOpts, creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DPPFactory.contract.Transact(opts, "addPoolByAdmin", creator, baseToken, quoteToken, pool)
}

// AddPoolByAdmin is a paid mutator transaction binding the contract method 0x39d00ef9.
//
// Solidity: function addPoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DPPFactory *DPPFactorySession) AddPoolByAdmin(creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.AddPoolByAdmin(&_DPPFactory.TransactOpts, creator, baseToken, quoteToken, pool)
}

// AddPoolByAdmin is a paid mutator transaction binding the contract method 0x39d00ef9.
//
// Solidity: function addPoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DPPFactory *DPPFactoryTransactorSession) AddPoolByAdmin(creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.AddPoolByAdmin(&_DPPFactory.TransactOpts, creator, baseToken, quoteToken, pool)
}

// BatchAddPoolByAdmin is a paid mutator transaction binding the contract method 0x182e8dbb.
//
// Solidity: function batchAddPoolByAdmin(address[] creators, address[] baseTokens, address[] quoteTokens, address[] pools) returns()
func (_DPPFactory *DPPFactoryTransactor) BatchAddPoolByAdmin(opts *bind.TransactOpts, creators []common.Address, baseTokens []common.Address, quoteTokens []common.Address, pools []common.Address) (*types.Transaction, error) {
	return _DPPFactory.contract.Transact(opts, "batchAddPoolByAdmin", creators, baseTokens, quoteTokens, pools)
}

// BatchAddPoolByAdmin is a paid mutator transaction binding the contract method 0x182e8dbb.
//
// Solidity: function batchAddPoolByAdmin(address[] creators, address[] baseTokens, address[] quoteTokens, address[] pools) returns()
func (_DPPFactory *DPPFactorySession) BatchAddPoolByAdmin(creators []common.Address, baseTokens []common.Address, quoteTokens []common.Address, pools []common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.BatchAddPoolByAdmin(&_DPPFactory.TransactOpts, creators, baseTokens, quoteTokens, pools)
}

// BatchAddPoolByAdmin is a paid mutator transaction binding the contract method 0x182e8dbb.
//
// Solidity: function batchAddPoolByAdmin(address[] creators, address[] baseTokens, address[] quoteTokens, address[] pools) returns()
func (_DPPFactory *DPPFactoryTransactorSession) BatchAddPoolByAdmin(creators []common.Address, baseTokens []common.Address, quoteTokens []common.Address, pools []common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.BatchAddPoolByAdmin(&_DPPFactory.TransactOpts, creators, baseTokens, quoteTokens, pools)
}

// ClaimOwnership is a paid mutator transaction binding the contract method 0x4e71e0c8.
//
// Solidity: function claimOwnership() returns()
func (_DPPFactory *DPPFactoryTransactor) ClaimOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DPPFactory.contract.Transact(opts, "claimOwnership")
}

// ClaimOwnership is a paid mutator transaction binding the contract method 0x4e71e0c8.
//
// Solidity: function claimOwnership() returns()
func (_DPPFactory *DPPFactorySession) ClaimOwnership() (*types.Transaction, error) {
	return _DPPFactory.Contract.ClaimOwnership(&_DPPFactory.TransactOpts)
}

// ClaimOwnership is a paid mutator transaction binding the contract method 0x4e71e0c8.
//
// Solidity: function claimOwnership() returns()
func (_DPPFactory *DPPFactoryTransactorSession) ClaimOwnership() (*types.Transaction, error) {
	return _DPPFactory.Contract.ClaimOwnership(&_DPPFactory.TransactOpts)
}

// CreateDODOPrivatePool is a paid mutator transaction binding the contract method 0x09b8adb8.
//
// Solidity: function createDODOPrivatePool() returns(address newPrivatePool)
func (_DPPFactory *DPPFactoryTransactor) CreateDODOPrivatePool(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DPPFactory.contract.Transact(opts, "createDODOPrivatePool")
}

// CreateDODOPrivatePool is a paid mutator transaction binding the contract method 0x09b8adb8.
//
// Solidity: function createDODOPrivatePool() returns(address newPrivatePool)
func (_DPPFactory *DPPFactorySession) CreateDODOPrivatePool() (*types.Transaction, error) {
	return _DPPFactory.Contract.CreateDODOPrivatePool(&_DPPFactory.TransactOpts)
}

// CreateDODOPrivatePool is a paid mutator transaction binding the contract method 0x09b8adb8.
//
// Solidity: function createDODOPrivatePool() returns(address newPrivatePool)
func (_DPPFactory *DPPFactoryTransactorSession) CreateDODOPrivatePool() (*types.Transaction, error) {
	return _DPPFactory.Contract.CreateDODOPrivatePool(&_DPPFactory.TransactOpts)
}

// InitDODOPrivatePool is a paid mutator transaction binding the contract method 0x195eefe0.
//
// Solidity: function initDODOPrivatePool(address dppAddress, address creator, address baseToken, address quoteToken, uint256 lpFeeRate, uint256 k, uint256 i, bool isOpenTwap) returns()
func (_DPPFactory *DPPFactoryTransactor) InitDODOPrivatePool(opts *bind.TransactOpts, dppAddress common.Address, creator common.Address, baseToken common.Address, quoteToken common.Address, lpFeeRate *big.Int, k *big.Int, i *big.Int, isOpenTwap bool) (*types.Transaction, error) {
	return _DPPFactory.contract.Transact(opts, "initDODOPrivatePool", dppAddress, creator, baseToken, quoteToken, lpFeeRate, k, i, isOpenTwap)
}

// InitDODOPrivatePool is a paid mutator transaction binding the contract method 0x195eefe0.
//
// Solidity: function initDODOPrivatePool(address dppAddress, address creator, address baseToken, address quoteToken, uint256 lpFeeRate, uint256 k, uint256 i, bool isOpenTwap) returns()
func (_DPPFactory *DPPFactorySession) InitDODOPrivatePool(dppAddress common.Address, creator common.Address, baseToken common.Address, quoteToken common.Address, lpFeeRate *big.Int, k *big.Int, i *big.Int, isOpenTwap bool) (*types.Transaction, error) {
	return _DPPFactory.Contract.InitDODOPrivatePool(&_DPPFactory.TransactOpts, dppAddress, creator, baseToken, quoteToken, lpFeeRate, k, i, isOpenTwap)
}

// InitDODOPrivatePool is a paid mutator transaction binding the contract method 0x195eefe0.
//
// Solidity: function initDODOPrivatePool(address dppAddress, address creator, address baseToken, address quoteToken, uint256 lpFeeRate, uint256 k, uint256 i, bool isOpenTwap) returns()
func (_DPPFactory *DPPFactoryTransactorSession) InitDODOPrivatePool(dppAddress common.Address, creator common.Address, baseToken common.Address, quoteToken common.Address, lpFeeRate *big.Int, k *big.Int, i *big.Int, isOpenTwap bool) (*types.Transaction, error) {
	return _DPPFactory.Contract.InitDODOPrivatePool(&_DPPFactory.TransactOpts, dppAddress, creator, baseToken, quoteToken, lpFeeRate, k, i, isOpenTwap)
}

// InitOwner is a paid mutator transaction binding the contract method 0x0d009297.
//
// Solidity: function initOwner(address newOwner) returns()
func (_DPPFactory *DPPFactoryTransactor) InitOwner(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _DPPFactory.contract.Transact(opts, "initOwner", newOwner)
}

// InitOwner is a paid mutator transaction binding the contract method 0x0d009297.
//
// Solidity: function initOwner(address newOwner) returns()
func (_DPPFactory *DPPFactorySession) InitOwner(newOwner common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.InitOwner(&_DPPFactory.TransactOpts, newOwner)
}

// InitOwner is a paid mutator transaction binding the contract method 0x0d009297.
//
// Solidity: function initOwner(address newOwner) returns()
func (_DPPFactory *DPPFactoryTransactorSession) InitOwner(newOwner common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.InitOwner(&_DPPFactory.TransactOpts, newOwner)
}

// RemoveAdminList is a paid mutator transaction binding the contract method 0xfd8bd849.
//
// Solidity: function removeAdminList(address contractAddr) returns()
func (_DPPFactory *DPPFactoryTransactor) RemoveAdminList(opts *bind.TransactOpts, contractAddr common.Address) (*types.Transaction, error) {
	return _DPPFactory.contract.Transact(opts, "removeAdminList", contractAddr)
}

// RemoveAdminList is a paid mutator transaction binding the contract method 0xfd8bd849.
//
// Solidity: function removeAdminList(address contractAddr) returns()
func (_DPPFactory *DPPFactorySession) RemoveAdminList(contractAddr common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.RemoveAdminList(&_DPPFactory.TransactOpts, contractAddr)
}

// RemoveAdminList is a paid mutator transaction binding the contract method 0xfd8bd849.
//
// Solidity: function removeAdminList(address contractAddr) returns()
func (_DPPFactory *DPPFactoryTransactorSession) RemoveAdminList(contractAddr common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.RemoveAdminList(&_DPPFactory.TransactOpts, contractAddr)
}

// RemovePoolByAdmin is a paid mutator transaction binding the contract method 0x43274b82.
//
// Solidity: function removePoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DPPFactory *DPPFactoryTransactor) RemovePoolByAdmin(opts *bind.TransactOpts, creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DPPFactory.contract.Transact(opts, "removePoolByAdmin", creator, baseToken, quoteToken, pool)
}

// RemovePoolByAdmin is a paid mutator transaction binding the contract method 0x43274b82.
//
// Solidity: function removePoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DPPFactory *DPPFactorySession) RemovePoolByAdmin(creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.RemovePoolByAdmin(&_DPPFactory.TransactOpts, creator, baseToken, quoteToken, pool)
}

// RemovePoolByAdmin is a paid mutator transaction binding the contract method 0x43274b82.
//
// Solidity: function removePoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DPPFactory *DPPFactoryTransactorSession) RemovePoolByAdmin(creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.RemovePoolByAdmin(&_DPPFactory.TransactOpts, creator, baseToken, quoteToken, pool)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DPPFactory *DPPFactoryTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _DPPFactory.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DPPFactory *DPPFactorySession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.TransferOwnership(&_DPPFactory.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DPPFactory *DPPFactoryTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.TransferOwnership(&_DPPFactory.TransactOpts, newOwner)
}

// UpdateAdminTemplate is a paid mutator transaction binding the contract method 0x7d2c82d7.
//
// Solidity: function updateAdminTemplate(address _newDPPAdminTemplate) returns()
func (_DPPFactory *DPPFactoryTransactor) UpdateAdminTemplate(opts *bind.TransactOpts, _newDPPAdminTemplate common.Address) (*types.Transaction, error) {
	return _DPPFactory.contract.Transact(opts, "updateAdminTemplate", _newDPPAdminTemplate)
}

// UpdateAdminTemplate is a paid mutator transaction binding the contract method 0x7d2c82d7.
//
// Solidity: function updateAdminTemplate(address _newDPPAdminTemplate) returns()
func (_DPPFactory *DPPFactorySession) UpdateAdminTemplate(_newDPPAdminTemplate common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.UpdateAdminTemplate(&_DPPFactory.TransactOpts, _newDPPAdminTemplate)
}

// UpdateAdminTemplate is a paid mutator transaction binding the contract method 0x7d2c82d7.
//
// Solidity: function updateAdminTemplate(address _newDPPAdminTemplate) returns()
func (_DPPFactory *DPPFactoryTransactorSession) UpdateAdminTemplate(_newDPPAdminTemplate common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.UpdateAdminTemplate(&_DPPFactory.TransactOpts, _newDPPAdminTemplate)
}

// UpdateDefaultMaintainer is a paid mutator transaction binding the contract method 0x9e988ee3.
//
// Solidity: function updateDefaultMaintainer(address _newMaintainer) returns()
func (_DPPFactory *DPPFactoryTransactor) UpdateDefaultMaintainer(opts *bind.TransactOpts, _newMaintainer common.Address) (*types.Transaction, error) {
	return _DPPFactory.contract.Transact(opts, "updateDefaultMaintainer", _newMaintainer)
}

// UpdateDefaultMaintainer is a paid mutator transaction binding the contract method 0x9e988ee3.
//
// Solidity: function updateDefaultMaintainer(address _newMaintainer) returns()
func (_DPPFactory *DPPFactorySession) UpdateDefaultMaintainer(_newMaintainer common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.UpdateDefaultMaintainer(&_DPPFactory.TransactOpts, _newMaintainer)
}

// UpdateDefaultMaintainer is a paid mutator transaction binding the contract method 0x9e988ee3.
//
// Solidity: function updateDefaultMaintainer(address _newMaintainer) returns()
func (_DPPFactory *DPPFactoryTransactorSession) UpdateDefaultMaintainer(_newMaintainer common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.UpdateDefaultMaintainer(&_DPPFactory.TransactOpts, _newMaintainer)
}

// UpdateDppTemplate is a paid mutator transaction binding the contract method 0x44b7f78e.
//
// Solidity: function updateDppTemplate(address _newDPPTemplate) returns()
func (_DPPFactory *DPPFactoryTransactor) UpdateDppTemplate(opts *bind.TransactOpts, _newDPPTemplate common.Address) (*types.Transaction, error) {
	return _DPPFactory.contract.Transact(opts, "updateDppTemplate", _newDPPTemplate)
}

// UpdateDppTemplate is a paid mutator transaction binding the contract method 0x44b7f78e.
//
// Solidity: function updateDppTemplate(address _newDPPTemplate) returns()
func (_DPPFactory *DPPFactorySession) UpdateDppTemplate(_newDPPTemplate common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.UpdateDppTemplate(&_DPPFactory.TransactOpts, _newDPPTemplate)
}

// UpdateDppTemplate is a paid mutator transaction binding the contract method 0x44b7f78e.
//
// Solidity: function updateDppTemplate(address _newDPPTemplate) returns()
func (_DPPFactory *DPPFactoryTransactorSession) UpdateDppTemplate(_newDPPTemplate common.Address) (*types.Transaction, error) {
	return _DPPFactory.Contract.UpdateDppTemplate(&_DPPFactory.TransactOpts, _newDPPTemplate)
}

// DPPFactoryNewDPPIterator is returned from FilterNewDPP and is used to iterate over the raw logs and unpacked data for NewDPP events raised by the DPPFactory contract.
type DPPFactoryNewDPPIterator struct {
	Event *DPPFactoryNewDPP // Event containing the contract specifics and raw log

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
func (it *DPPFactoryNewDPPIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DPPFactoryNewDPP)
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
		it.Event = new(DPPFactoryNewDPP)
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
func (it *DPPFactoryNewDPPIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DPPFactoryNewDPPIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DPPFactoryNewDPP represents a NewDPP event raised by the DPPFactory contract.
type DPPFactoryNewDPP struct {
	BaseToken  common.Address
	QuoteToken common.Address
	Creator    common.Address
	Dpp        common.Address
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterNewDPP is a free log retrieval operation binding the contract event 0x8494fe594cd5087021d4b11758a2bbc7be28a430e94f2b268d668e5991ed3b8a.
//
// Solidity: event NewDPP(address baseToken, address quoteToken, address creator, address dpp)
func (_DPPFactory *DPPFactoryFilterer) FilterNewDPP(opts *bind.FilterOpts) (*DPPFactoryNewDPPIterator, error) {

	logs, sub, err := _DPPFactory.contract.FilterLogs(opts, "NewDPP")
	if err != nil {
		return nil, err
	}
	return &DPPFactoryNewDPPIterator{contract: _DPPFactory.contract, event: "NewDPP", logs: logs, sub: sub}, nil
}

// WatchNewDPP is a free log subscription operation binding the contract event 0x8494fe594cd5087021d4b11758a2bbc7be28a430e94f2b268d668e5991ed3b8a.
//
// Solidity: event NewDPP(address baseToken, address quoteToken, address creator, address dpp)
func (_DPPFactory *DPPFactoryFilterer) WatchNewDPP(opts *bind.WatchOpts, sink chan<- *DPPFactoryNewDPP) (event.Subscription, error) {

	logs, sub, err := _DPPFactory.contract.WatchLogs(opts, "NewDPP")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DPPFactoryNewDPP)
				if err := _DPPFactory.contract.UnpackLog(event, "NewDPP", log); err != nil {
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

// ParseNewDPP is a log parse operation binding the contract event 0x8494fe594cd5087021d4b11758a2bbc7be28a430e94f2b268d668e5991ed3b8a.
//
// Solidity: event NewDPP(address baseToken, address quoteToken, address creator, address dpp)
func (_DPPFactory *DPPFactoryFilterer) ParseNewDPP(log types.Log) (*DPPFactoryNewDPP, error) {
	event := new(DPPFactoryNewDPP)
	if err := _DPPFactory.contract.UnpackLog(event, "NewDPP", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DPPFactoryOwnershipTransferPreparedIterator is returned from FilterOwnershipTransferPrepared and is used to iterate over the raw logs and unpacked data for OwnershipTransferPrepared events raised by the DPPFactory contract.
type DPPFactoryOwnershipTransferPreparedIterator struct {
	Event *DPPFactoryOwnershipTransferPrepared // Event containing the contract specifics and raw log

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
func (it *DPPFactoryOwnershipTransferPreparedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DPPFactoryOwnershipTransferPrepared)
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
		it.Event = new(DPPFactoryOwnershipTransferPrepared)
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
func (it *DPPFactoryOwnershipTransferPreparedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DPPFactoryOwnershipTransferPreparedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DPPFactoryOwnershipTransferPrepared represents a OwnershipTransferPrepared event raised by the DPPFactory contract.
type DPPFactoryOwnershipTransferPrepared struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferPrepared is a free log retrieval operation binding the contract event 0xdcf55418cee3220104fef63f979ff3c4097ad240c0c43dcb33ce837748983e62.
//
// Solidity: event OwnershipTransferPrepared(address indexed previousOwner, address indexed newOwner)
func (_DPPFactory *DPPFactoryFilterer) FilterOwnershipTransferPrepared(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*DPPFactoryOwnershipTransferPreparedIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DPPFactory.contract.FilterLogs(opts, "OwnershipTransferPrepared", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &DPPFactoryOwnershipTransferPreparedIterator{contract: _DPPFactory.contract, event: "OwnershipTransferPrepared", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferPrepared is a free log subscription operation binding the contract event 0xdcf55418cee3220104fef63f979ff3c4097ad240c0c43dcb33ce837748983e62.
//
// Solidity: event OwnershipTransferPrepared(address indexed previousOwner, address indexed newOwner)
func (_DPPFactory *DPPFactoryFilterer) WatchOwnershipTransferPrepared(opts *bind.WatchOpts, sink chan<- *DPPFactoryOwnershipTransferPrepared, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DPPFactory.contract.WatchLogs(opts, "OwnershipTransferPrepared", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DPPFactoryOwnershipTransferPrepared)
				if err := _DPPFactory.contract.UnpackLog(event, "OwnershipTransferPrepared", log); err != nil {
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

// ParseOwnershipTransferPrepared is a log parse operation binding the contract event 0xdcf55418cee3220104fef63f979ff3c4097ad240c0c43dcb33ce837748983e62.
//
// Solidity: event OwnershipTransferPrepared(address indexed previousOwner, address indexed newOwner)
func (_DPPFactory *DPPFactoryFilterer) ParseOwnershipTransferPrepared(log types.Log) (*DPPFactoryOwnershipTransferPrepared, error) {
	event := new(DPPFactoryOwnershipTransferPrepared)
	if err := _DPPFactory.contract.UnpackLog(event, "OwnershipTransferPrepared", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DPPFactoryOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the DPPFactory contract.
type DPPFactoryOwnershipTransferredIterator struct {
	Event *DPPFactoryOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *DPPFactoryOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DPPFactoryOwnershipTransferred)
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
		it.Event = new(DPPFactoryOwnershipTransferred)
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
func (it *DPPFactoryOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DPPFactoryOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DPPFactoryOwnershipTransferred represents a OwnershipTransferred event raised by the DPPFactory contract.
type DPPFactoryOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_DPPFactory *DPPFactoryFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*DPPFactoryOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DPPFactory.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &DPPFactoryOwnershipTransferredIterator{contract: _DPPFactory.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_DPPFactory *DPPFactoryFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *DPPFactoryOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DPPFactory.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DPPFactoryOwnershipTransferred)
				if err := _DPPFactory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_DPPFactory *DPPFactoryFilterer) ParseOwnershipTransferred(log types.Log) (*DPPFactoryOwnershipTransferred, error) {
	event := new(DPPFactoryOwnershipTransferred)
	if err := _DPPFactory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DPPFactoryRemoveDPPIterator is returned from FilterRemoveDPP and is used to iterate over the raw logs and unpacked data for RemoveDPP events raised by the DPPFactory contract.
type DPPFactoryRemoveDPPIterator struct {
	Event *DPPFactoryRemoveDPP // Event containing the contract specifics and raw log

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
func (it *DPPFactoryRemoveDPPIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DPPFactoryRemoveDPP)
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
		it.Event = new(DPPFactoryRemoveDPP)
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
func (it *DPPFactoryRemoveDPPIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DPPFactoryRemoveDPPIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DPPFactoryRemoveDPP represents a RemoveDPP event raised by the DPPFactory contract.
type DPPFactoryRemoveDPP struct {
	Dpp common.Address
	Raw types.Log // Blockchain specific contextual infos
}

// FilterRemoveDPP is a free log retrieval operation binding the contract event 0xafed4f6701f0e943874e915e49e2e7b67ae31f18e083f540e890af2275aa116d.
//
// Solidity: event RemoveDPP(address dpp)
func (_DPPFactory *DPPFactoryFilterer) FilterRemoveDPP(opts *bind.FilterOpts) (*DPPFactoryRemoveDPPIterator, error) {

	logs, sub, err := _DPPFactory.contract.FilterLogs(opts, "RemoveDPP")
	if err != nil {
		return nil, err
	}
	return &DPPFactoryRemoveDPPIterator{contract: _DPPFactory.contract, event: "RemoveDPP", logs: logs, sub: sub}, nil
}

// WatchRemoveDPP is a free log subscription operation binding the contract event 0xafed4f6701f0e943874e915e49e2e7b67ae31f18e083f540e890af2275aa116d.
//
// Solidity: event RemoveDPP(address dpp)
func (_DPPFactory *DPPFactoryFilterer) WatchRemoveDPP(opts *bind.WatchOpts, sink chan<- *DPPFactoryRemoveDPP) (event.Subscription, error) {

	logs, sub, err := _DPPFactory.contract.WatchLogs(opts, "RemoveDPP")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DPPFactoryRemoveDPP)
				if err := _DPPFactory.contract.UnpackLog(event, "RemoveDPP", log); err != nil {
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

// ParseRemoveDPP is a log parse operation binding the contract event 0xafed4f6701f0e943874e915e49e2e7b67ae31f18e083f540e890af2275aa116d.
//
// Solidity: event RemoveDPP(address dpp)
func (_DPPFactory *DPPFactoryFilterer) ParseRemoveDPP(log types.Log) (*DPPFactoryRemoveDPP, error) {
	event := new(DPPFactoryRemoveDPP)
	if err := _DPPFactory.contract.UnpackLog(event, "RemoveDPP", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DPPFactoryAddAdminIterator is returned from FilterAddAdmin and is used to iterate over the raw logs and unpacked data for AddAdmin events raised by the DPPFactory contract.
type DPPFactoryAddAdminIterator struct {
	Event *DPPFactoryAddAdmin // Event containing the contract specifics and raw log

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
func (it *DPPFactoryAddAdminIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DPPFactoryAddAdmin)
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
		it.Event = new(DPPFactoryAddAdmin)
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
func (it *DPPFactoryAddAdminIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DPPFactoryAddAdminIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DPPFactoryAddAdmin represents a AddAdmin event raised by the DPPFactory contract.
type DPPFactoryAddAdmin struct {
	Admin common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterAddAdmin is a free log retrieval operation binding the contract event 0x7048027520ecbaa8947764cd502c5c78c2c53bbd902e06b108da1cbdf98c6fc4.
//
// Solidity: event addAdmin(address admin)
func (_DPPFactory *DPPFactoryFilterer) FilterAddAdmin(opts *bind.FilterOpts) (*DPPFactoryAddAdminIterator, error) {

	logs, sub, err := _DPPFactory.contract.FilterLogs(opts, "addAdmin")
	if err != nil {
		return nil, err
	}
	return &DPPFactoryAddAdminIterator{contract: _DPPFactory.contract, event: "addAdmin", logs: logs, sub: sub}, nil
}

// WatchAddAdmin is a free log subscription operation binding the contract event 0x7048027520ecbaa8947764cd502c5c78c2c53bbd902e06b108da1cbdf98c6fc4.
//
// Solidity: event addAdmin(address admin)
func (_DPPFactory *DPPFactoryFilterer) WatchAddAdmin(opts *bind.WatchOpts, sink chan<- *DPPFactoryAddAdmin) (event.Subscription, error) {

	logs, sub, err := _DPPFactory.contract.WatchLogs(opts, "addAdmin")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DPPFactoryAddAdmin)
				if err := _DPPFactory.contract.UnpackLog(event, "addAdmin", log); err != nil {
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

// ParseAddAdmin is a log parse operation binding the contract event 0x7048027520ecbaa8947764cd502c5c78c2c53bbd902e06b108da1cbdf98c6fc4.
//
// Solidity: event addAdmin(address admin)
func (_DPPFactory *DPPFactoryFilterer) ParseAddAdmin(log types.Log) (*DPPFactoryAddAdmin, error) {
	event := new(DPPFactoryAddAdmin)
	if err := _DPPFactory.contract.UnpackLog(event, "addAdmin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DPPFactoryRemoveAdminIterator is returned from FilterRemoveAdmin and is used to iterate over the raw logs and unpacked data for RemoveAdmin events raised by the DPPFactory contract.
type DPPFactoryRemoveAdminIterator struct {
	Event *DPPFactoryRemoveAdmin // Event containing the contract specifics and raw log

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
func (it *DPPFactoryRemoveAdminIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DPPFactoryRemoveAdmin)
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
		it.Event = new(DPPFactoryRemoveAdmin)
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
func (it *DPPFactoryRemoveAdminIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DPPFactoryRemoveAdminIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DPPFactoryRemoveAdmin represents a RemoveAdmin event raised by the DPPFactory contract.
type DPPFactoryRemoveAdmin struct {
	Admin common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterRemoveAdmin is a free log retrieval operation binding the contract event 0x1785f53c768259a7ab38ed67e958aab075b56ff206e3d7f29ea4ca203d1a9774.
//
// Solidity: event removeAdmin(address admin)
func (_DPPFactory *DPPFactoryFilterer) FilterRemoveAdmin(opts *bind.FilterOpts) (*DPPFactoryRemoveAdminIterator, error) {

	logs, sub, err := _DPPFactory.contract.FilterLogs(opts, "removeAdmin")
	if err != nil {
		return nil, err
	}
	return &DPPFactoryRemoveAdminIterator{contract: _DPPFactory.contract, event: "removeAdmin", logs: logs, sub: sub}, nil
}

// WatchRemoveAdmin is a free log subscription operation binding the contract event 0x1785f53c768259a7ab38ed67e958aab075b56ff206e3d7f29ea4ca203d1a9774.
//
// Solidity: event removeAdmin(address admin)
func (_DPPFactory *DPPFactoryFilterer) WatchRemoveAdmin(opts *bind.WatchOpts, sink chan<- *DPPFactoryRemoveAdmin) (event.Subscription, error) {

	logs, sub, err := _DPPFactory.contract.WatchLogs(opts, "removeAdmin")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DPPFactoryRemoveAdmin)
				if err := _DPPFactory.contract.UnpackLog(event, "removeAdmin", log); err != nil {
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

// ParseRemoveAdmin is a log parse operation binding the contract event 0x1785f53c768259a7ab38ed67e958aab075b56ff206e3d7f29ea4ca203d1a9774.
//
// Solidity: event removeAdmin(address admin)
func (_DPPFactory *DPPFactoryFilterer) ParseRemoveAdmin(log types.Log) (*DPPFactoryRemoveAdmin, error) {
	event := new(DPPFactoryRemoveAdmin)
	if err := _DPPFactory.contract.UnpackLog(event, "removeAdmin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
