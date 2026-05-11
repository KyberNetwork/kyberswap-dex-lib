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

// DSPFactoryMetaData contains all meta data concerning the DSPFactory contract.
var DSPFactoryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"cloneFactory\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"DSPTemplate\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"defaultMaintainer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"defaultMtFeeRateModel\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"DSP\",\"type\":\"address\"}],\"name\":\"NewDSP\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferPrepared\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"DSP\",\"type\":\"address\"}],\"name\":\"RemoveDSP\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"_CLONE_FACTORY_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_DEFAULT_MAINTAINER_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_DEFAULT_MT_FEE_RATE_MODEL_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_DSP_TEMPLATE_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_NEW_OWNER_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_OWNER_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"_REGISTRY_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"_USER_REGISTRY_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"addPoolByAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"claimOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"lpFeeRate\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"i\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"k\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isOpenTWAP\",\"type\":\"bool\"}],\"name\":\"createDODOStablePool\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"newStablePool\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"}],\"name\":\"getDODOPool\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"machines\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"}],\"name\":\"getDODOPoolBidirection\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"baseToken0Machines\",\"type\":\"address[]\"},{\"internalType\":\"address[]\",\"name\":\"baseToken1Machines\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"}],\"name\":\"getDODOPoolByUser\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"machines\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"initOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"removePoolByAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newDSPTemplate\",\"type\":\"address\"}],\"name\":\"updateDSPTemplate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newMaintainer\",\"type\":\"address\"}],\"name\":\"updateDefaultMaintainer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// DSPFactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use DSPFactoryMetaData.ABI instead.
var DSPFactoryABI = DSPFactoryMetaData.ABI

// DSPFactory is an auto generated Go binding around an Ethereum contract.
type DSPFactory struct {
	DSPFactoryCaller     // Read-only binding to the contract
	DSPFactoryTransactor // Write-only binding to the contract
	DSPFactoryFilterer   // Log filterer for contract events
}

// DSPFactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type DSPFactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DSPFactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DSPFactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DSPFactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DSPFactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DSPFactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DSPFactorySession struct {
	Contract     *DSPFactory       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DSPFactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DSPFactoryCallerSession struct {
	Contract *DSPFactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// DSPFactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DSPFactoryTransactorSession struct {
	Contract     *DSPFactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// DSPFactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type DSPFactoryRaw struct {
	Contract *DSPFactory // Generic contract binding to access the raw methods on
}

// DSPFactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DSPFactoryCallerRaw struct {
	Contract *DSPFactoryCaller // Generic read-only contract binding to access the raw methods on
}

// DSPFactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DSPFactoryTransactorRaw struct {
	Contract *DSPFactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDSPFactory creates a new instance of DSPFactory, bound to a specific deployed contract.
func NewDSPFactory(address common.Address, backend bind.ContractBackend) (*DSPFactory, error) {
	contract, err := bindDSPFactory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DSPFactory{DSPFactoryCaller: DSPFactoryCaller{contract: contract}, DSPFactoryTransactor: DSPFactoryTransactor{contract: contract}, DSPFactoryFilterer: DSPFactoryFilterer{contract: contract}}, nil
}

// NewDSPFactoryCaller creates a new read-only instance of DSPFactory, bound to a specific deployed contract.
func NewDSPFactoryCaller(address common.Address, caller bind.ContractCaller) (*DSPFactoryCaller, error) {
	contract, err := bindDSPFactory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DSPFactoryCaller{contract: contract}, nil
}

// NewDSPFactoryTransactor creates a new write-only instance of DSPFactory, bound to a specific deployed contract.
func NewDSPFactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*DSPFactoryTransactor, error) {
	contract, err := bindDSPFactory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DSPFactoryTransactor{contract: contract}, nil
}

// NewDSPFactoryFilterer creates a new log filterer instance of DSPFactory, bound to a specific deployed contract.
func NewDSPFactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*DSPFactoryFilterer, error) {
	contract, err := bindDSPFactory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DSPFactoryFilterer{contract: contract}, nil
}

// bindDSPFactory binds a generic wrapper to an already deployed contract.
func bindDSPFactory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := DSPFactoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DSPFactory *DSPFactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DSPFactory.Contract.DSPFactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DSPFactory *DSPFactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DSPFactory.Contract.DSPFactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DSPFactory *DSPFactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DSPFactory.Contract.DSPFactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DSPFactory *DSPFactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DSPFactory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DSPFactory *DSPFactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DSPFactory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DSPFactory *DSPFactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DSPFactory.Contract.contract.Transact(opts, method, params...)
}

// CLONEFACTORY is a free data retrieval call binding the contract method 0xeb774d05.
//
// Solidity: function _CLONE_FACTORY_() view returns(address)
func (_DSPFactory *DSPFactoryCaller) CLONEFACTORY(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DSPFactory.contract.Call(opts, &out, "_CLONE_FACTORY_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CLONEFACTORY is a free data retrieval call binding the contract method 0xeb774d05.
//
// Solidity: function _CLONE_FACTORY_() view returns(address)
func (_DSPFactory *DSPFactorySession) CLONEFACTORY() (common.Address, error) {
	return _DSPFactory.Contract.CLONEFACTORY(&_DSPFactory.CallOpts)
}

// CLONEFACTORY is a free data retrieval call binding the contract method 0xeb774d05.
//
// Solidity: function _CLONE_FACTORY_() view returns(address)
func (_DSPFactory *DSPFactoryCallerSession) CLONEFACTORY() (common.Address, error) {
	return _DSPFactory.Contract.CLONEFACTORY(&_DSPFactory.CallOpts)
}

// DEFAULTMAINTAINER is a free data retrieval call binding the contract method 0x81ab4d0a.
//
// Solidity: function _DEFAULT_MAINTAINER_() view returns(address)
func (_DSPFactory *DSPFactoryCaller) DEFAULTMAINTAINER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DSPFactory.contract.Call(opts, &out, "_DEFAULT_MAINTAINER_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DEFAULTMAINTAINER is a free data retrieval call binding the contract method 0x81ab4d0a.
//
// Solidity: function _DEFAULT_MAINTAINER_() view returns(address)
func (_DSPFactory *DSPFactorySession) DEFAULTMAINTAINER() (common.Address, error) {
	return _DSPFactory.Contract.DEFAULTMAINTAINER(&_DSPFactory.CallOpts)
}

// DEFAULTMAINTAINER is a free data retrieval call binding the contract method 0x81ab4d0a.
//
// Solidity: function _DEFAULT_MAINTAINER_() view returns(address)
func (_DSPFactory *DSPFactoryCallerSession) DEFAULTMAINTAINER() (common.Address, error) {
	return _DSPFactory.Contract.DEFAULTMAINTAINER(&_DSPFactory.CallOpts)
}

// DEFAULTMTFEERATEMODEL is a free data retrieval call binding the contract method 0x6c5ccb9b.
//
// Solidity: function _DEFAULT_MT_FEE_RATE_MODEL_() view returns(address)
func (_DSPFactory *DSPFactoryCaller) DEFAULTMTFEERATEMODEL(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DSPFactory.contract.Call(opts, &out, "_DEFAULT_MT_FEE_RATE_MODEL_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DEFAULTMTFEERATEMODEL is a free data retrieval call binding the contract method 0x6c5ccb9b.
//
// Solidity: function _DEFAULT_MT_FEE_RATE_MODEL_() view returns(address)
func (_DSPFactory *DSPFactorySession) DEFAULTMTFEERATEMODEL() (common.Address, error) {
	return _DSPFactory.Contract.DEFAULTMTFEERATEMODEL(&_DSPFactory.CallOpts)
}

// DEFAULTMTFEERATEMODEL is a free data retrieval call binding the contract method 0x6c5ccb9b.
//
// Solidity: function _DEFAULT_MT_FEE_RATE_MODEL_() view returns(address)
func (_DSPFactory *DSPFactoryCallerSession) DEFAULTMTFEERATEMODEL() (common.Address, error) {
	return _DSPFactory.Contract.DEFAULTMTFEERATEMODEL(&_DSPFactory.CallOpts)
}

// DSPTEMPLATE is a free data retrieval call binding the contract method 0x59358068.
//
// Solidity: function _DSP_TEMPLATE_() view returns(address)
func (_DSPFactory *DSPFactoryCaller) DSPTEMPLATE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DSPFactory.contract.Call(opts, &out, "_DSP_TEMPLATE_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DSPTEMPLATE is a free data retrieval call binding the contract method 0x59358068.
//
// Solidity: function _DSP_TEMPLATE_() view returns(address)
func (_DSPFactory *DSPFactorySession) DSPTEMPLATE() (common.Address, error) {
	return _DSPFactory.Contract.DSPTEMPLATE(&_DSPFactory.CallOpts)
}

// DSPTEMPLATE is a free data retrieval call binding the contract method 0x59358068.
//
// Solidity: function _DSP_TEMPLATE_() view returns(address)
func (_DSPFactory *DSPFactoryCallerSession) DSPTEMPLATE() (common.Address, error) {
	return _DSPFactory.Contract.DSPTEMPLATE(&_DSPFactory.CallOpts)
}

// NEWOWNER is a free data retrieval call binding the contract method 0x8456db15.
//
// Solidity: function _NEW_OWNER_() view returns(address)
func (_DSPFactory *DSPFactoryCaller) NEWOWNER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DSPFactory.contract.Call(opts, &out, "_NEW_OWNER_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NEWOWNER is a free data retrieval call binding the contract method 0x8456db15.
//
// Solidity: function _NEW_OWNER_() view returns(address)
func (_DSPFactory *DSPFactorySession) NEWOWNER() (common.Address, error) {
	return _DSPFactory.Contract.NEWOWNER(&_DSPFactory.CallOpts)
}

// NEWOWNER is a free data retrieval call binding the contract method 0x8456db15.
//
// Solidity: function _NEW_OWNER_() view returns(address)
func (_DSPFactory *DSPFactoryCallerSession) NEWOWNER() (common.Address, error) {
	return _DSPFactory.Contract.NEWOWNER(&_DSPFactory.CallOpts)
}

// OWNER is a free data retrieval call binding the contract method 0x16048bc4.
//
// Solidity: function _OWNER_() view returns(address)
func (_DSPFactory *DSPFactoryCaller) OWNER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DSPFactory.contract.Call(opts, &out, "_OWNER_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OWNER is a free data retrieval call binding the contract method 0x16048bc4.
//
// Solidity: function _OWNER_() view returns(address)
func (_DSPFactory *DSPFactorySession) OWNER() (common.Address, error) {
	return _DSPFactory.Contract.OWNER(&_DSPFactory.CallOpts)
}

// OWNER is a free data retrieval call binding the contract method 0x16048bc4.
//
// Solidity: function _OWNER_() view returns(address)
func (_DSPFactory *DSPFactoryCallerSession) OWNER() (common.Address, error) {
	return _DSPFactory.Contract.OWNER(&_DSPFactory.CallOpts)
}

// REGISTRY is a free data retrieval call binding the contract method 0xbdeb0a91.
//
// Solidity: function _REGISTRY_(address , address , uint256 ) view returns(address)
func (_DSPFactory *DSPFactoryCaller) REGISTRY(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _DSPFactory.contract.Call(opts, &out, "_REGISTRY_", arg0, arg1, arg2)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// REGISTRY is a free data retrieval call binding the contract method 0xbdeb0a91.
//
// Solidity: function _REGISTRY_(address , address , uint256 ) view returns(address)
func (_DSPFactory *DSPFactorySession) REGISTRY(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	return _DSPFactory.Contract.REGISTRY(&_DSPFactory.CallOpts, arg0, arg1, arg2)
}

// REGISTRY is a free data retrieval call binding the contract method 0xbdeb0a91.
//
// Solidity: function _REGISTRY_(address , address , uint256 ) view returns(address)
func (_DSPFactory *DSPFactoryCallerSession) REGISTRY(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	return _DSPFactory.Contract.REGISTRY(&_DSPFactory.CallOpts, arg0, arg1, arg2)
}

// USERREGISTRY is a free data retrieval call binding the contract method 0xa58888db.
//
// Solidity: function _USER_REGISTRY_(address , uint256 ) view returns(address)
func (_DSPFactory *DSPFactoryCaller) USERREGISTRY(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _DSPFactory.contract.Call(opts, &out, "_USER_REGISTRY_", arg0, arg1)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// USERREGISTRY is a free data retrieval call binding the contract method 0xa58888db.
//
// Solidity: function _USER_REGISTRY_(address , uint256 ) view returns(address)
func (_DSPFactory *DSPFactorySession) USERREGISTRY(arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	return _DSPFactory.Contract.USERREGISTRY(&_DSPFactory.CallOpts, arg0, arg1)
}

// USERREGISTRY is a free data retrieval call binding the contract method 0xa58888db.
//
// Solidity: function _USER_REGISTRY_(address , uint256 ) view returns(address)
func (_DSPFactory *DSPFactoryCallerSession) USERREGISTRY(arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	return _DSPFactory.Contract.USERREGISTRY(&_DSPFactory.CallOpts, arg0, arg1)
}

// GetDODOPool is a free data retrieval call binding the contract method 0x57a281dc.
//
// Solidity: function getDODOPool(address baseToken, address quoteToken) view returns(address[] machines)
func (_DSPFactory *DSPFactoryCaller) GetDODOPool(opts *bind.CallOpts, baseToken common.Address, quoteToken common.Address) ([]common.Address, error) {
	var out []interface{}
	err := _DSPFactory.contract.Call(opts, &out, "getDODOPool", baseToken, quoteToken)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetDODOPool is a free data retrieval call binding the contract method 0x57a281dc.
//
// Solidity: function getDODOPool(address baseToken, address quoteToken) view returns(address[] machines)
func (_DSPFactory *DSPFactorySession) GetDODOPool(baseToken common.Address, quoteToken common.Address) ([]common.Address, error) {
	return _DSPFactory.Contract.GetDODOPool(&_DSPFactory.CallOpts, baseToken, quoteToken)
}

// GetDODOPool is a free data retrieval call binding the contract method 0x57a281dc.
//
// Solidity: function getDODOPool(address baseToken, address quoteToken) view returns(address[] machines)
func (_DSPFactory *DSPFactoryCallerSession) GetDODOPool(baseToken common.Address, quoteToken common.Address) ([]common.Address, error) {
	return _DSPFactory.Contract.GetDODOPool(&_DSPFactory.CallOpts, baseToken, quoteToken)
}

// GetDODOPoolBidirection is a free data retrieval call binding the contract method 0x794e5538.
//
// Solidity: function getDODOPoolBidirection(address token0, address token1) view returns(address[] baseToken0Machines, address[] baseToken1Machines)
func (_DSPFactory *DSPFactoryCaller) GetDODOPoolBidirection(opts *bind.CallOpts, token0 common.Address, token1 common.Address) (struct {
	BaseToken0Machines []common.Address
	BaseToken1Machines []common.Address
}, error) {
	var out []interface{}
	err := _DSPFactory.contract.Call(opts, &out, "getDODOPoolBidirection", token0, token1)

	outstruct := new(struct {
		BaseToken0Machines []common.Address
		BaseToken1Machines []common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.BaseToken0Machines = *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)
	outstruct.BaseToken1Machines = *abi.ConvertType(out[1], new([]common.Address)).(*[]common.Address)

	return *outstruct, err

}

// GetDODOPoolBidirection is a free data retrieval call binding the contract method 0x794e5538.
//
// Solidity: function getDODOPoolBidirection(address token0, address token1) view returns(address[] baseToken0Machines, address[] baseToken1Machines)
func (_DSPFactory *DSPFactorySession) GetDODOPoolBidirection(token0 common.Address, token1 common.Address) (struct {
	BaseToken0Machines []common.Address
	BaseToken1Machines []common.Address
}, error) {
	return _DSPFactory.Contract.GetDODOPoolBidirection(&_DSPFactory.CallOpts, token0, token1)
}

// GetDODOPoolBidirection is a free data retrieval call binding the contract method 0x794e5538.
//
// Solidity: function getDODOPoolBidirection(address token0, address token1) view returns(address[] baseToken0Machines, address[] baseToken1Machines)
func (_DSPFactory *DSPFactoryCallerSession) GetDODOPoolBidirection(token0 common.Address, token1 common.Address) (struct {
	BaseToken0Machines []common.Address
	BaseToken1Machines []common.Address
}, error) {
	return _DSPFactory.Contract.GetDODOPoolBidirection(&_DSPFactory.CallOpts, token0, token1)
}

// GetDODOPoolByUser is a free data retrieval call binding the contract method 0xe65f7029.
//
// Solidity: function getDODOPoolByUser(address user) view returns(address[] machines)
func (_DSPFactory *DSPFactoryCaller) GetDODOPoolByUser(opts *bind.CallOpts, user common.Address) ([]common.Address, error) {
	var out []interface{}
	err := _DSPFactory.contract.Call(opts, &out, "getDODOPoolByUser", user)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetDODOPoolByUser is a free data retrieval call binding the contract method 0xe65f7029.
//
// Solidity: function getDODOPoolByUser(address user) view returns(address[] machines)
func (_DSPFactory *DSPFactorySession) GetDODOPoolByUser(user common.Address) ([]common.Address, error) {
	return _DSPFactory.Contract.GetDODOPoolByUser(&_DSPFactory.CallOpts, user)
}

// GetDODOPoolByUser is a free data retrieval call binding the contract method 0xe65f7029.
//
// Solidity: function getDODOPoolByUser(address user) view returns(address[] machines)
func (_DSPFactory *DSPFactoryCallerSession) GetDODOPoolByUser(user common.Address) ([]common.Address, error) {
	return _DSPFactory.Contract.GetDODOPoolByUser(&_DSPFactory.CallOpts, user)
}

// AddPoolByAdmin is a paid mutator transaction binding the contract method 0x39d00ef9.
//
// Solidity: function addPoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DSPFactory *DSPFactoryTransactor) AddPoolByAdmin(opts *bind.TransactOpts, creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DSPFactory.contract.Transact(opts, "addPoolByAdmin", creator, baseToken, quoteToken, pool)
}

// AddPoolByAdmin is a paid mutator transaction binding the contract method 0x39d00ef9.
//
// Solidity: function addPoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DSPFactory *DSPFactorySession) AddPoolByAdmin(creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DSPFactory.Contract.AddPoolByAdmin(&_DSPFactory.TransactOpts, creator, baseToken, quoteToken, pool)
}

// AddPoolByAdmin is a paid mutator transaction binding the contract method 0x39d00ef9.
//
// Solidity: function addPoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DSPFactory *DSPFactoryTransactorSession) AddPoolByAdmin(creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DSPFactory.Contract.AddPoolByAdmin(&_DSPFactory.TransactOpts, creator, baseToken, quoteToken, pool)
}

// ClaimOwnership is a paid mutator transaction binding the contract method 0x4e71e0c8.
//
// Solidity: function claimOwnership() returns()
func (_DSPFactory *DSPFactoryTransactor) ClaimOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DSPFactory.contract.Transact(opts, "claimOwnership")
}

// ClaimOwnership is a paid mutator transaction binding the contract method 0x4e71e0c8.
//
// Solidity: function claimOwnership() returns()
func (_DSPFactory *DSPFactorySession) ClaimOwnership() (*types.Transaction, error) {
	return _DSPFactory.Contract.ClaimOwnership(&_DSPFactory.TransactOpts)
}

// ClaimOwnership is a paid mutator transaction binding the contract method 0x4e71e0c8.
//
// Solidity: function claimOwnership() returns()
func (_DSPFactory *DSPFactoryTransactorSession) ClaimOwnership() (*types.Transaction, error) {
	return _DSPFactory.Contract.ClaimOwnership(&_DSPFactory.TransactOpts)
}

// CreateDODOStablePool is a paid mutator transaction binding the contract method 0xcf5c2f10.
//
// Solidity: function createDODOStablePool(address baseToken, address quoteToken, uint256 lpFeeRate, uint256 i, uint256 k, bool isOpenTWAP) returns(address newStablePool)
func (_DSPFactory *DSPFactoryTransactor) CreateDODOStablePool(opts *bind.TransactOpts, baseToken common.Address, quoteToken common.Address, lpFeeRate *big.Int, i *big.Int, k *big.Int, isOpenTWAP bool) (*types.Transaction, error) {
	return _DSPFactory.contract.Transact(opts, "createDODOStablePool", baseToken, quoteToken, lpFeeRate, i, k, isOpenTWAP)
}

// CreateDODOStablePool is a paid mutator transaction binding the contract method 0xcf5c2f10.
//
// Solidity: function createDODOStablePool(address baseToken, address quoteToken, uint256 lpFeeRate, uint256 i, uint256 k, bool isOpenTWAP) returns(address newStablePool)
func (_DSPFactory *DSPFactorySession) CreateDODOStablePool(baseToken common.Address, quoteToken common.Address, lpFeeRate *big.Int, i *big.Int, k *big.Int, isOpenTWAP bool) (*types.Transaction, error) {
	return _DSPFactory.Contract.CreateDODOStablePool(&_DSPFactory.TransactOpts, baseToken, quoteToken, lpFeeRate, i, k, isOpenTWAP)
}

// CreateDODOStablePool is a paid mutator transaction binding the contract method 0xcf5c2f10.
//
// Solidity: function createDODOStablePool(address baseToken, address quoteToken, uint256 lpFeeRate, uint256 i, uint256 k, bool isOpenTWAP) returns(address newStablePool)
func (_DSPFactory *DSPFactoryTransactorSession) CreateDODOStablePool(baseToken common.Address, quoteToken common.Address, lpFeeRate *big.Int, i *big.Int, k *big.Int, isOpenTWAP bool) (*types.Transaction, error) {
	return _DSPFactory.Contract.CreateDODOStablePool(&_DSPFactory.TransactOpts, baseToken, quoteToken, lpFeeRate, i, k, isOpenTWAP)
}

// InitOwner is a paid mutator transaction binding the contract method 0x0d009297.
//
// Solidity: function initOwner(address newOwner) returns()
func (_DSPFactory *DSPFactoryTransactor) InitOwner(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _DSPFactory.contract.Transact(opts, "initOwner", newOwner)
}

// InitOwner is a paid mutator transaction binding the contract method 0x0d009297.
//
// Solidity: function initOwner(address newOwner) returns()
func (_DSPFactory *DSPFactorySession) InitOwner(newOwner common.Address) (*types.Transaction, error) {
	return _DSPFactory.Contract.InitOwner(&_DSPFactory.TransactOpts, newOwner)
}

// InitOwner is a paid mutator transaction binding the contract method 0x0d009297.
//
// Solidity: function initOwner(address newOwner) returns()
func (_DSPFactory *DSPFactoryTransactorSession) InitOwner(newOwner common.Address) (*types.Transaction, error) {
	return _DSPFactory.Contract.InitOwner(&_DSPFactory.TransactOpts, newOwner)
}

// RemovePoolByAdmin is a paid mutator transaction binding the contract method 0x43274b82.
//
// Solidity: function removePoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DSPFactory *DSPFactoryTransactor) RemovePoolByAdmin(opts *bind.TransactOpts, creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DSPFactory.contract.Transact(opts, "removePoolByAdmin", creator, baseToken, quoteToken, pool)
}

// RemovePoolByAdmin is a paid mutator transaction binding the contract method 0x43274b82.
//
// Solidity: function removePoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DSPFactory *DSPFactorySession) RemovePoolByAdmin(creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DSPFactory.Contract.RemovePoolByAdmin(&_DSPFactory.TransactOpts, creator, baseToken, quoteToken, pool)
}

// RemovePoolByAdmin is a paid mutator transaction binding the contract method 0x43274b82.
//
// Solidity: function removePoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DSPFactory *DSPFactoryTransactorSession) RemovePoolByAdmin(creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DSPFactory.Contract.RemovePoolByAdmin(&_DSPFactory.TransactOpts, creator, baseToken, quoteToken, pool)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DSPFactory *DSPFactoryTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _DSPFactory.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DSPFactory *DSPFactorySession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _DSPFactory.Contract.TransferOwnership(&_DSPFactory.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DSPFactory *DSPFactoryTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _DSPFactory.Contract.TransferOwnership(&_DSPFactory.TransactOpts, newOwner)
}

// UpdateDSPTemplate is a paid mutator transaction binding the contract method 0x1cdd5094.
//
// Solidity: function updateDSPTemplate(address _newDSPTemplate) returns()
func (_DSPFactory *DSPFactoryTransactor) UpdateDSPTemplate(opts *bind.TransactOpts, _newDSPTemplate common.Address) (*types.Transaction, error) {
	return _DSPFactory.contract.Transact(opts, "updateDSPTemplate", _newDSPTemplate)
}

// UpdateDSPTemplate is a paid mutator transaction binding the contract method 0x1cdd5094.
//
// Solidity: function updateDSPTemplate(address _newDSPTemplate) returns()
func (_DSPFactory *DSPFactorySession) UpdateDSPTemplate(_newDSPTemplate common.Address) (*types.Transaction, error) {
	return _DSPFactory.Contract.UpdateDSPTemplate(&_DSPFactory.TransactOpts, _newDSPTemplate)
}

// UpdateDSPTemplate is a paid mutator transaction binding the contract method 0x1cdd5094.
//
// Solidity: function updateDSPTemplate(address _newDSPTemplate) returns()
func (_DSPFactory *DSPFactoryTransactorSession) UpdateDSPTemplate(_newDSPTemplate common.Address) (*types.Transaction, error) {
	return _DSPFactory.Contract.UpdateDSPTemplate(&_DSPFactory.TransactOpts, _newDSPTemplate)
}

// UpdateDefaultMaintainer is a paid mutator transaction binding the contract method 0x9e988ee3.
//
// Solidity: function updateDefaultMaintainer(address _newMaintainer) returns()
func (_DSPFactory *DSPFactoryTransactor) UpdateDefaultMaintainer(opts *bind.TransactOpts, _newMaintainer common.Address) (*types.Transaction, error) {
	return _DSPFactory.contract.Transact(opts, "updateDefaultMaintainer", _newMaintainer)
}

// UpdateDefaultMaintainer is a paid mutator transaction binding the contract method 0x9e988ee3.
//
// Solidity: function updateDefaultMaintainer(address _newMaintainer) returns()
func (_DSPFactory *DSPFactorySession) UpdateDefaultMaintainer(_newMaintainer common.Address) (*types.Transaction, error) {
	return _DSPFactory.Contract.UpdateDefaultMaintainer(&_DSPFactory.TransactOpts, _newMaintainer)
}

// UpdateDefaultMaintainer is a paid mutator transaction binding the contract method 0x9e988ee3.
//
// Solidity: function updateDefaultMaintainer(address _newMaintainer) returns()
func (_DSPFactory *DSPFactoryTransactorSession) UpdateDefaultMaintainer(_newMaintainer common.Address) (*types.Transaction, error) {
	return _DSPFactory.Contract.UpdateDefaultMaintainer(&_DSPFactory.TransactOpts, _newMaintainer)
}

// DSPFactoryNewDSPIterator is returned from FilterNewDSP and is used to iterate over the raw logs and unpacked data for NewDSP events raised by the DSPFactory contract.
type DSPFactoryNewDSPIterator struct {
	Event *DSPFactoryNewDSP // Event containing the contract specifics and raw log

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
func (it *DSPFactoryNewDSPIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DSPFactoryNewDSP)
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
		it.Event = new(DSPFactoryNewDSP)
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
func (it *DSPFactoryNewDSPIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DSPFactoryNewDSPIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DSPFactoryNewDSP represents a NewDSP event raised by the DSPFactory contract.
type DSPFactoryNewDSP struct {
	BaseToken  common.Address
	QuoteToken common.Address
	Creator    common.Address
	DSP        common.Address
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterNewDSP is a free log retrieval operation binding the contract event 0xbc1083a2c1c5ef31e13fb436953d22b47880cf7db279c2c5666b16083afd6b9d.
//
// Solidity: event NewDSP(address baseToken, address quoteToken, address creator, address DSP)
func (_DSPFactory *DSPFactoryFilterer) FilterNewDSP(opts *bind.FilterOpts) (*DSPFactoryNewDSPIterator, error) {

	logs, sub, err := _DSPFactory.contract.FilterLogs(opts, "NewDSP")
	if err != nil {
		return nil, err
	}
	return &DSPFactoryNewDSPIterator{contract: _DSPFactory.contract, event: "NewDSP", logs: logs, sub: sub}, nil
}

// WatchNewDSP is a free log subscription operation binding the contract event 0xbc1083a2c1c5ef31e13fb436953d22b47880cf7db279c2c5666b16083afd6b9d.
//
// Solidity: event NewDSP(address baseToken, address quoteToken, address creator, address DSP)
func (_DSPFactory *DSPFactoryFilterer) WatchNewDSP(opts *bind.WatchOpts, sink chan<- *DSPFactoryNewDSP) (event.Subscription, error) {

	logs, sub, err := _DSPFactory.contract.WatchLogs(opts, "NewDSP")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DSPFactoryNewDSP)
				if err := _DSPFactory.contract.UnpackLog(event, "NewDSP", log); err != nil {
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

// ParseNewDSP is a log parse operation binding the contract event 0xbc1083a2c1c5ef31e13fb436953d22b47880cf7db279c2c5666b16083afd6b9d.
//
// Solidity: event NewDSP(address baseToken, address quoteToken, address creator, address DSP)
func (_DSPFactory *DSPFactoryFilterer) ParseNewDSP(log types.Log) (*DSPFactoryNewDSP, error) {
	event := new(DSPFactoryNewDSP)
	if err := _DSPFactory.contract.UnpackLog(event, "NewDSP", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DSPFactoryOwnershipTransferPreparedIterator is returned from FilterOwnershipTransferPrepared and is used to iterate over the raw logs and unpacked data for OwnershipTransferPrepared events raised by the DSPFactory contract.
type DSPFactoryOwnershipTransferPreparedIterator struct {
	Event *DSPFactoryOwnershipTransferPrepared // Event containing the contract specifics and raw log

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
func (it *DSPFactoryOwnershipTransferPreparedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DSPFactoryOwnershipTransferPrepared)
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
		it.Event = new(DSPFactoryOwnershipTransferPrepared)
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
func (it *DSPFactoryOwnershipTransferPreparedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DSPFactoryOwnershipTransferPreparedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DSPFactoryOwnershipTransferPrepared represents a OwnershipTransferPrepared event raised by the DSPFactory contract.
type DSPFactoryOwnershipTransferPrepared struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferPrepared is a free log retrieval operation binding the contract event 0xdcf55418cee3220104fef63f979ff3c4097ad240c0c43dcb33ce837748983e62.
//
// Solidity: event OwnershipTransferPrepared(address indexed previousOwner, address indexed newOwner)
func (_DSPFactory *DSPFactoryFilterer) FilterOwnershipTransferPrepared(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*DSPFactoryOwnershipTransferPreparedIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DSPFactory.contract.FilterLogs(opts, "OwnershipTransferPrepared", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &DSPFactoryOwnershipTransferPreparedIterator{contract: _DSPFactory.contract, event: "OwnershipTransferPrepared", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferPrepared is a free log subscription operation binding the contract event 0xdcf55418cee3220104fef63f979ff3c4097ad240c0c43dcb33ce837748983e62.
//
// Solidity: event OwnershipTransferPrepared(address indexed previousOwner, address indexed newOwner)
func (_DSPFactory *DSPFactoryFilterer) WatchOwnershipTransferPrepared(opts *bind.WatchOpts, sink chan<- *DSPFactoryOwnershipTransferPrepared, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DSPFactory.contract.WatchLogs(opts, "OwnershipTransferPrepared", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DSPFactoryOwnershipTransferPrepared)
				if err := _DSPFactory.contract.UnpackLog(event, "OwnershipTransferPrepared", log); err != nil {
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
func (_DSPFactory *DSPFactoryFilterer) ParseOwnershipTransferPrepared(log types.Log) (*DSPFactoryOwnershipTransferPrepared, error) {
	event := new(DSPFactoryOwnershipTransferPrepared)
	if err := _DSPFactory.contract.UnpackLog(event, "OwnershipTransferPrepared", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DSPFactoryOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the DSPFactory contract.
type DSPFactoryOwnershipTransferredIterator struct {
	Event *DSPFactoryOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *DSPFactoryOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DSPFactoryOwnershipTransferred)
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
		it.Event = new(DSPFactoryOwnershipTransferred)
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
func (it *DSPFactoryOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DSPFactoryOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DSPFactoryOwnershipTransferred represents a OwnershipTransferred event raised by the DSPFactory contract.
type DSPFactoryOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_DSPFactory *DSPFactoryFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*DSPFactoryOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DSPFactory.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &DSPFactoryOwnershipTransferredIterator{contract: _DSPFactory.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_DSPFactory *DSPFactoryFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *DSPFactoryOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DSPFactory.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DSPFactoryOwnershipTransferred)
				if err := _DSPFactory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_DSPFactory *DSPFactoryFilterer) ParseOwnershipTransferred(log types.Log) (*DSPFactoryOwnershipTransferred, error) {
	event := new(DSPFactoryOwnershipTransferred)
	if err := _DSPFactory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DSPFactoryRemoveDSPIterator is returned from FilterRemoveDSP and is used to iterate over the raw logs and unpacked data for RemoveDSP events raised by the DSPFactory contract.
type DSPFactoryRemoveDSPIterator struct {
	Event *DSPFactoryRemoveDSP // Event containing the contract specifics and raw log

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
func (it *DSPFactoryRemoveDSPIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DSPFactoryRemoveDSP)
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
		it.Event = new(DSPFactoryRemoveDSP)
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
func (it *DSPFactoryRemoveDSPIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DSPFactoryRemoveDSPIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DSPFactoryRemoveDSP represents a RemoveDSP event raised by the DSPFactory contract.
type DSPFactoryRemoveDSP struct {
	DSP common.Address
	Raw types.Log // Blockchain specific contextual infos
}

// FilterRemoveDSP is a free log retrieval operation binding the contract event 0x896d6f4e7068e6d194a5b84bf222d513973a0dc5ae9687123d5c61329d4489f2.
//
// Solidity: event RemoveDSP(address DSP)
func (_DSPFactory *DSPFactoryFilterer) FilterRemoveDSP(opts *bind.FilterOpts) (*DSPFactoryRemoveDSPIterator, error) {

	logs, sub, err := _DSPFactory.contract.FilterLogs(opts, "RemoveDSP")
	if err != nil {
		return nil, err
	}
	return &DSPFactoryRemoveDSPIterator{contract: _DSPFactory.contract, event: "RemoveDSP", logs: logs, sub: sub}, nil
}

// WatchRemoveDSP is a free log subscription operation binding the contract event 0x896d6f4e7068e6d194a5b84bf222d513973a0dc5ae9687123d5c61329d4489f2.
//
// Solidity: event RemoveDSP(address DSP)
func (_DSPFactory *DSPFactoryFilterer) WatchRemoveDSP(opts *bind.WatchOpts, sink chan<- *DSPFactoryRemoveDSP) (event.Subscription, error) {

	logs, sub, err := _DSPFactory.contract.WatchLogs(opts, "RemoveDSP")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DSPFactoryRemoveDSP)
				if err := _DSPFactory.contract.UnpackLog(event, "RemoveDSP", log); err != nil {
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

// ParseRemoveDSP is a log parse operation binding the contract event 0x896d6f4e7068e6d194a5b84bf222d513973a0dc5ae9687123d5c61329d4489f2.
//
// Solidity: event RemoveDSP(address DSP)
func (_DSPFactory *DSPFactoryFilterer) ParseRemoveDSP(log types.Log) (*DSPFactoryRemoveDSP, error) {
	event := new(DSPFactoryRemoveDSP)
	if err := _DSPFactory.contract.UnpackLog(event, "RemoveDSP", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
