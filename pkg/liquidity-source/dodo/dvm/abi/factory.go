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

// DVMFactoryMetaData contains all meta data concerning the DVMFactory contract.
var DVMFactoryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"cloneFactory\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"dvmTemplate\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"defaultMaintainer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"defaultMtFeeRateModel\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"dvm\",\"type\":\"address\"}],\"name\":\"NewDVM\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferPrepared\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"dvm\",\"type\":\"address\"}],\"name\":\"RemoveDVM\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"_CLONE_FACTORY_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_DEFAULT_MAINTAINER_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_DEFAULT_MT_FEE_RATE_MODEL_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_DVM_TEMPLATE_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_NEW_OWNER_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_OWNER_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"_REGISTRY_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"_USER_REGISTRY_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"addPoolByAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"claimOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"lpFeeRate\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"i\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"k\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isOpenTWAP\",\"type\":\"bool\"}],\"name\":\"createDODOVendingMachine\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"newVendingMachine\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"}],\"name\":\"getDODOPool\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"machines\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"}],\"name\":\"getDODOPoolBidirection\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"baseToken0Machines\",\"type\":\"address[]\"},{\"internalType\":\"address[]\",\"name\":\"baseToken1Machines\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"}],\"name\":\"getDODOPoolByUser\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"machines\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"initOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"removePoolByAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newMaintainer\",\"type\":\"address\"}],\"name\":\"updateDefaultMaintainer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newDVMTemplate\",\"type\":\"address\"}],\"name\":\"updateDvmTemplate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// DVMFactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use DVMFactoryMetaData.ABI instead.
var DVMFactoryABI = DVMFactoryMetaData.ABI

// DVMFactory is an auto generated Go binding around an Ethereum contract.
type DVMFactory struct {
	DVMFactoryCaller     // Read-only binding to the contract
	DVMFactoryTransactor // Write-only binding to the contract
	DVMFactoryFilterer   // Log filterer for contract events
}

// DVMFactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type DVMFactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DVMFactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DVMFactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DVMFactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DVMFactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DVMFactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DVMFactorySession struct {
	Contract     *DVMFactory       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DVMFactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DVMFactoryCallerSession struct {
	Contract *DVMFactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// DVMFactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DVMFactoryTransactorSession struct {
	Contract     *DVMFactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// DVMFactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type DVMFactoryRaw struct {
	Contract *DVMFactory // Generic contract binding to access the raw methods on
}

// DVMFactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DVMFactoryCallerRaw struct {
	Contract *DVMFactoryCaller // Generic read-only contract binding to access the raw methods on
}

// DVMFactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DVMFactoryTransactorRaw struct {
	Contract *DVMFactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDVMFactory creates a new instance of DVMFactory, bound to a specific deployed contract.
func NewDVMFactory(address common.Address, backend bind.ContractBackend) (*DVMFactory, error) {
	contract, err := bindDVMFactory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DVMFactory{DVMFactoryCaller: DVMFactoryCaller{contract: contract}, DVMFactoryTransactor: DVMFactoryTransactor{contract: contract}, DVMFactoryFilterer: DVMFactoryFilterer{contract: contract}}, nil
}

// NewDVMFactoryCaller creates a new read-only instance of DVMFactory, bound to a specific deployed contract.
func NewDVMFactoryCaller(address common.Address, caller bind.ContractCaller) (*DVMFactoryCaller, error) {
	contract, err := bindDVMFactory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DVMFactoryCaller{contract: contract}, nil
}

// NewDVMFactoryTransactor creates a new write-only instance of DVMFactory, bound to a specific deployed contract.
func NewDVMFactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*DVMFactoryTransactor, error) {
	contract, err := bindDVMFactory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DVMFactoryTransactor{contract: contract}, nil
}

// NewDVMFactoryFilterer creates a new log filterer instance of DVMFactory, bound to a specific deployed contract.
func NewDVMFactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*DVMFactoryFilterer, error) {
	contract, err := bindDVMFactory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DVMFactoryFilterer{contract: contract}, nil
}

// bindDVMFactory binds a generic wrapper to an already deployed contract.
func bindDVMFactory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := DVMFactoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DVMFactory *DVMFactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DVMFactory.Contract.DVMFactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DVMFactory *DVMFactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DVMFactory.Contract.DVMFactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DVMFactory *DVMFactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DVMFactory.Contract.DVMFactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DVMFactory *DVMFactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DVMFactory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DVMFactory *DVMFactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DVMFactory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DVMFactory *DVMFactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DVMFactory.Contract.contract.Transact(opts, method, params...)
}

// CLONEFACTORY is a free data retrieval call binding the contract method 0xeb774d05.
//
// Solidity: function _CLONE_FACTORY_() view returns(address)
func (_DVMFactory *DVMFactoryCaller) CLONEFACTORY(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DVMFactory.contract.Call(opts, &out, "_CLONE_FACTORY_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CLONEFACTORY is a free data retrieval call binding the contract method 0xeb774d05.
//
// Solidity: function _CLONE_FACTORY_() view returns(address)
func (_DVMFactory *DVMFactorySession) CLONEFACTORY() (common.Address, error) {
	return _DVMFactory.Contract.CLONEFACTORY(&_DVMFactory.CallOpts)
}

// CLONEFACTORY is a free data retrieval call binding the contract method 0xeb774d05.
//
// Solidity: function _CLONE_FACTORY_() view returns(address)
func (_DVMFactory *DVMFactoryCallerSession) CLONEFACTORY() (common.Address, error) {
	return _DVMFactory.Contract.CLONEFACTORY(&_DVMFactory.CallOpts)
}

// DEFAULTMAINTAINER is a free data retrieval call binding the contract method 0x81ab4d0a.
//
// Solidity: function _DEFAULT_MAINTAINER_() view returns(address)
func (_DVMFactory *DVMFactoryCaller) DEFAULTMAINTAINER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DVMFactory.contract.Call(opts, &out, "_DEFAULT_MAINTAINER_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DEFAULTMAINTAINER is a free data retrieval call binding the contract method 0x81ab4d0a.
//
// Solidity: function _DEFAULT_MAINTAINER_() view returns(address)
func (_DVMFactory *DVMFactorySession) DEFAULTMAINTAINER() (common.Address, error) {
	return _DVMFactory.Contract.DEFAULTMAINTAINER(&_DVMFactory.CallOpts)
}

// DEFAULTMAINTAINER is a free data retrieval call binding the contract method 0x81ab4d0a.
//
// Solidity: function _DEFAULT_MAINTAINER_() view returns(address)
func (_DVMFactory *DVMFactoryCallerSession) DEFAULTMAINTAINER() (common.Address, error) {
	return _DVMFactory.Contract.DEFAULTMAINTAINER(&_DVMFactory.CallOpts)
}

// DEFAULTMTFEERATEMODEL is a free data retrieval call binding the contract method 0x6c5ccb9b.
//
// Solidity: function _DEFAULT_MT_FEE_RATE_MODEL_() view returns(address)
func (_DVMFactory *DVMFactoryCaller) DEFAULTMTFEERATEMODEL(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DVMFactory.contract.Call(opts, &out, "_DEFAULT_MT_FEE_RATE_MODEL_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DEFAULTMTFEERATEMODEL is a free data retrieval call binding the contract method 0x6c5ccb9b.
//
// Solidity: function _DEFAULT_MT_FEE_RATE_MODEL_() view returns(address)
func (_DVMFactory *DVMFactorySession) DEFAULTMTFEERATEMODEL() (common.Address, error) {
	return _DVMFactory.Contract.DEFAULTMTFEERATEMODEL(&_DVMFactory.CallOpts)
}

// DEFAULTMTFEERATEMODEL is a free data retrieval call binding the contract method 0x6c5ccb9b.
//
// Solidity: function _DEFAULT_MT_FEE_RATE_MODEL_() view returns(address)
func (_DVMFactory *DVMFactoryCallerSession) DEFAULTMTFEERATEMODEL() (common.Address, error) {
	return _DVMFactory.Contract.DEFAULTMTFEERATEMODEL(&_DVMFactory.CallOpts)
}

// DVMTEMPLATE is a free data retrieval call binding the contract method 0xccf0c059.
//
// Solidity: function _DVM_TEMPLATE_() view returns(address)
func (_DVMFactory *DVMFactoryCaller) DVMTEMPLATE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DVMFactory.contract.Call(opts, &out, "_DVM_TEMPLATE_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DVMTEMPLATE is a free data retrieval call binding the contract method 0xccf0c059.
//
// Solidity: function _DVM_TEMPLATE_() view returns(address)
func (_DVMFactory *DVMFactorySession) DVMTEMPLATE() (common.Address, error) {
	return _DVMFactory.Contract.DVMTEMPLATE(&_DVMFactory.CallOpts)
}

// DVMTEMPLATE is a free data retrieval call binding the contract method 0xccf0c059.
//
// Solidity: function _DVM_TEMPLATE_() view returns(address)
func (_DVMFactory *DVMFactoryCallerSession) DVMTEMPLATE() (common.Address, error) {
	return _DVMFactory.Contract.DVMTEMPLATE(&_DVMFactory.CallOpts)
}

// NEWOWNER is a free data retrieval call binding the contract method 0x8456db15.
//
// Solidity: function _NEW_OWNER_() view returns(address)
func (_DVMFactory *DVMFactoryCaller) NEWOWNER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DVMFactory.contract.Call(opts, &out, "_NEW_OWNER_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NEWOWNER is a free data retrieval call binding the contract method 0x8456db15.
//
// Solidity: function _NEW_OWNER_() view returns(address)
func (_DVMFactory *DVMFactorySession) NEWOWNER() (common.Address, error) {
	return _DVMFactory.Contract.NEWOWNER(&_DVMFactory.CallOpts)
}

// NEWOWNER is a free data retrieval call binding the contract method 0x8456db15.
//
// Solidity: function _NEW_OWNER_() view returns(address)
func (_DVMFactory *DVMFactoryCallerSession) NEWOWNER() (common.Address, error) {
	return _DVMFactory.Contract.NEWOWNER(&_DVMFactory.CallOpts)
}

// OWNER is a free data retrieval call binding the contract method 0x16048bc4.
//
// Solidity: function _OWNER_() view returns(address)
func (_DVMFactory *DVMFactoryCaller) OWNER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DVMFactory.contract.Call(opts, &out, "_OWNER_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OWNER is a free data retrieval call binding the contract method 0x16048bc4.
//
// Solidity: function _OWNER_() view returns(address)
func (_DVMFactory *DVMFactorySession) OWNER() (common.Address, error) {
	return _DVMFactory.Contract.OWNER(&_DVMFactory.CallOpts)
}

// OWNER is a free data retrieval call binding the contract method 0x16048bc4.
//
// Solidity: function _OWNER_() view returns(address)
func (_DVMFactory *DVMFactoryCallerSession) OWNER() (common.Address, error) {
	return _DVMFactory.Contract.OWNER(&_DVMFactory.CallOpts)
}

// REGISTRY is a free data retrieval call binding the contract method 0xbdeb0a91.
//
// Solidity: function _REGISTRY_(address , address , uint256 ) view returns(address)
func (_DVMFactory *DVMFactoryCaller) REGISTRY(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _DVMFactory.contract.Call(opts, &out, "_REGISTRY_", arg0, arg1, arg2)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// REGISTRY is a free data retrieval call binding the contract method 0xbdeb0a91.
//
// Solidity: function _REGISTRY_(address , address , uint256 ) view returns(address)
func (_DVMFactory *DVMFactorySession) REGISTRY(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	return _DVMFactory.Contract.REGISTRY(&_DVMFactory.CallOpts, arg0, arg1, arg2)
}

// REGISTRY is a free data retrieval call binding the contract method 0xbdeb0a91.
//
// Solidity: function _REGISTRY_(address , address , uint256 ) view returns(address)
func (_DVMFactory *DVMFactoryCallerSession) REGISTRY(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	return _DVMFactory.Contract.REGISTRY(&_DVMFactory.CallOpts, arg0, arg1, arg2)
}

// USERREGISTRY is a free data retrieval call binding the contract method 0xa58888db.
//
// Solidity: function _USER_REGISTRY_(address , uint256 ) view returns(address)
func (_DVMFactory *DVMFactoryCaller) USERREGISTRY(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _DVMFactory.contract.Call(opts, &out, "_USER_REGISTRY_", arg0, arg1)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// USERREGISTRY is a free data retrieval call binding the contract method 0xa58888db.
//
// Solidity: function _USER_REGISTRY_(address , uint256 ) view returns(address)
func (_DVMFactory *DVMFactorySession) USERREGISTRY(arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	return _DVMFactory.Contract.USERREGISTRY(&_DVMFactory.CallOpts, arg0, arg1)
}

// USERREGISTRY is a free data retrieval call binding the contract method 0xa58888db.
//
// Solidity: function _USER_REGISTRY_(address , uint256 ) view returns(address)
func (_DVMFactory *DVMFactoryCallerSession) USERREGISTRY(arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	return _DVMFactory.Contract.USERREGISTRY(&_DVMFactory.CallOpts, arg0, arg1)
}

// GetDODOPool is a free data retrieval call binding the contract method 0x57a281dc.
//
// Solidity: function getDODOPool(address baseToken, address quoteToken) view returns(address[] machines)
func (_DVMFactory *DVMFactoryCaller) GetDODOPool(opts *bind.CallOpts, baseToken common.Address, quoteToken common.Address) ([]common.Address, error) {
	var out []interface{}
	err := _DVMFactory.contract.Call(opts, &out, "getDODOPool", baseToken, quoteToken)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetDODOPool is a free data retrieval call binding the contract method 0x57a281dc.
//
// Solidity: function getDODOPool(address baseToken, address quoteToken) view returns(address[] machines)
func (_DVMFactory *DVMFactorySession) GetDODOPool(baseToken common.Address, quoteToken common.Address) ([]common.Address, error) {
	return _DVMFactory.Contract.GetDODOPool(&_DVMFactory.CallOpts, baseToken, quoteToken)
}

// GetDODOPool is a free data retrieval call binding the contract method 0x57a281dc.
//
// Solidity: function getDODOPool(address baseToken, address quoteToken) view returns(address[] machines)
func (_DVMFactory *DVMFactoryCallerSession) GetDODOPool(baseToken common.Address, quoteToken common.Address) ([]common.Address, error) {
	return _DVMFactory.Contract.GetDODOPool(&_DVMFactory.CallOpts, baseToken, quoteToken)
}

// GetDODOPoolBidirection is a free data retrieval call binding the contract method 0x794e5538.
//
// Solidity: function getDODOPoolBidirection(address token0, address token1) view returns(address[] baseToken0Machines, address[] baseToken1Machines)
func (_DVMFactory *DVMFactoryCaller) GetDODOPoolBidirection(opts *bind.CallOpts, token0 common.Address, token1 common.Address) (struct {
	BaseToken0Machines []common.Address
	BaseToken1Machines []common.Address
}, error) {
	var out []interface{}
	err := _DVMFactory.contract.Call(opts, &out, "getDODOPoolBidirection", token0, token1)

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
func (_DVMFactory *DVMFactorySession) GetDODOPoolBidirection(token0 common.Address, token1 common.Address) (struct {
	BaseToken0Machines []common.Address
	BaseToken1Machines []common.Address
}, error) {
	return _DVMFactory.Contract.GetDODOPoolBidirection(&_DVMFactory.CallOpts, token0, token1)
}

// GetDODOPoolBidirection is a free data retrieval call binding the contract method 0x794e5538.
//
// Solidity: function getDODOPoolBidirection(address token0, address token1) view returns(address[] baseToken0Machines, address[] baseToken1Machines)
func (_DVMFactory *DVMFactoryCallerSession) GetDODOPoolBidirection(token0 common.Address, token1 common.Address) (struct {
	BaseToken0Machines []common.Address
	BaseToken1Machines []common.Address
}, error) {
	return _DVMFactory.Contract.GetDODOPoolBidirection(&_DVMFactory.CallOpts, token0, token1)
}

// GetDODOPoolByUser is a free data retrieval call binding the contract method 0xe65f7029.
//
// Solidity: function getDODOPoolByUser(address user) view returns(address[] machines)
func (_DVMFactory *DVMFactoryCaller) GetDODOPoolByUser(opts *bind.CallOpts, user common.Address) ([]common.Address, error) {
	var out []interface{}
	err := _DVMFactory.contract.Call(opts, &out, "getDODOPoolByUser", user)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetDODOPoolByUser is a free data retrieval call binding the contract method 0xe65f7029.
//
// Solidity: function getDODOPoolByUser(address user) view returns(address[] machines)
func (_DVMFactory *DVMFactorySession) GetDODOPoolByUser(user common.Address) ([]common.Address, error) {
	return _DVMFactory.Contract.GetDODOPoolByUser(&_DVMFactory.CallOpts, user)
}

// GetDODOPoolByUser is a free data retrieval call binding the contract method 0xe65f7029.
//
// Solidity: function getDODOPoolByUser(address user) view returns(address[] machines)
func (_DVMFactory *DVMFactoryCallerSession) GetDODOPoolByUser(user common.Address) ([]common.Address, error) {
	return _DVMFactory.Contract.GetDODOPoolByUser(&_DVMFactory.CallOpts, user)
}

// AddPoolByAdmin is a paid mutator transaction binding the contract method 0x39d00ef9.
//
// Solidity: function addPoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DVMFactory *DVMFactoryTransactor) AddPoolByAdmin(opts *bind.TransactOpts, creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DVMFactory.contract.Transact(opts, "addPoolByAdmin", creator, baseToken, quoteToken, pool)
}

// AddPoolByAdmin is a paid mutator transaction binding the contract method 0x39d00ef9.
//
// Solidity: function addPoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DVMFactory *DVMFactorySession) AddPoolByAdmin(creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DVMFactory.Contract.AddPoolByAdmin(&_DVMFactory.TransactOpts, creator, baseToken, quoteToken, pool)
}

// AddPoolByAdmin is a paid mutator transaction binding the contract method 0x39d00ef9.
//
// Solidity: function addPoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DVMFactory *DVMFactoryTransactorSession) AddPoolByAdmin(creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DVMFactory.Contract.AddPoolByAdmin(&_DVMFactory.TransactOpts, creator, baseToken, quoteToken, pool)
}

// ClaimOwnership is a paid mutator transaction binding the contract method 0x4e71e0c8.
//
// Solidity: function claimOwnership() returns()
func (_DVMFactory *DVMFactoryTransactor) ClaimOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DVMFactory.contract.Transact(opts, "claimOwnership")
}

// ClaimOwnership is a paid mutator transaction binding the contract method 0x4e71e0c8.
//
// Solidity: function claimOwnership() returns()
func (_DVMFactory *DVMFactorySession) ClaimOwnership() (*types.Transaction, error) {
	return _DVMFactory.Contract.ClaimOwnership(&_DVMFactory.TransactOpts)
}

// ClaimOwnership is a paid mutator transaction binding the contract method 0x4e71e0c8.
//
// Solidity: function claimOwnership() returns()
func (_DVMFactory *DVMFactoryTransactorSession) ClaimOwnership() (*types.Transaction, error) {
	return _DVMFactory.Contract.ClaimOwnership(&_DVMFactory.TransactOpts)
}

// CreateDODOVendingMachine is a paid mutator transaction binding the contract method 0xe18c40c7.
//
// Solidity: function createDODOVendingMachine(address baseToken, address quoteToken, uint256 lpFeeRate, uint256 i, uint256 k, bool isOpenTWAP) returns(address newVendingMachine)
func (_DVMFactory *DVMFactoryTransactor) CreateDODOVendingMachine(opts *bind.TransactOpts, baseToken common.Address, quoteToken common.Address, lpFeeRate *big.Int, i *big.Int, k *big.Int, isOpenTWAP bool) (*types.Transaction, error) {
	return _DVMFactory.contract.Transact(opts, "createDODOVendingMachine", baseToken, quoteToken, lpFeeRate, i, k, isOpenTWAP)
}

// CreateDODOVendingMachine is a paid mutator transaction binding the contract method 0xe18c40c7.
//
// Solidity: function createDODOVendingMachine(address baseToken, address quoteToken, uint256 lpFeeRate, uint256 i, uint256 k, bool isOpenTWAP) returns(address newVendingMachine)
func (_DVMFactory *DVMFactorySession) CreateDODOVendingMachine(baseToken common.Address, quoteToken common.Address, lpFeeRate *big.Int, i *big.Int, k *big.Int, isOpenTWAP bool) (*types.Transaction, error) {
	return _DVMFactory.Contract.CreateDODOVendingMachine(&_DVMFactory.TransactOpts, baseToken, quoteToken, lpFeeRate, i, k, isOpenTWAP)
}

// CreateDODOVendingMachine is a paid mutator transaction binding the contract method 0xe18c40c7.
//
// Solidity: function createDODOVendingMachine(address baseToken, address quoteToken, uint256 lpFeeRate, uint256 i, uint256 k, bool isOpenTWAP) returns(address newVendingMachine)
func (_DVMFactory *DVMFactoryTransactorSession) CreateDODOVendingMachine(baseToken common.Address, quoteToken common.Address, lpFeeRate *big.Int, i *big.Int, k *big.Int, isOpenTWAP bool) (*types.Transaction, error) {
	return _DVMFactory.Contract.CreateDODOVendingMachine(&_DVMFactory.TransactOpts, baseToken, quoteToken, lpFeeRate, i, k, isOpenTWAP)
}

// InitOwner is a paid mutator transaction binding the contract method 0x0d009297.
//
// Solidity: function initOwner(address newOwner) returns()
func (_DVMFactory *DVMFactoryTransactor) InitOwner(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _DVMFactory.contract.Transact(opts, "initOwner", newOwner)
}

// InitOwner is a paid mutator transaction binding the contract method 0x0d009297.
//
// Solidity: function initOwner(address newOwner) returns()
func (_DVMFactory *DVMFactorySession) InitOwner(newOwner common.Address) (*types.Transaction, error) {
	return _DVMFactory.Contract.InitOwner(&_DVMFactory.TransactOpts, newOwner)
}

// InitOwner is a paid mutator transaction binding the contract method 0x0d009297.
//
// Solidity: function initOwner(address newOwner) returns()
func (_DVMFactory *DVMFactoryTransactorSession) InitOwner(newOwner common.Address) (*types.Transaction, error) {
	return _DVMFactory.Contract.InitOwner(&_DVMFactory.TransactOpts, newOwner)
}

// RemovePoolByAdmin is a paid mutator transaction binding the contract method 0x43274b82.
//
// Solidity: function removePoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DVMFactory *DVMFactoryTransactor) RemovePoolByAdmin(opts *bind.TransactOpts, creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DVMFactory.contract.Transact(opts, "removePoolByAdmin", creator, baseToken, quoteToken, pool)
}

// RemovePoolByAdmin is a paid mutator transaction binding the contract method 0x43274b82.
//
// Solidity: function removePoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DVMFactory *DVMFactorySession) RemovePoolByAdmin(creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DVMFactory.Contract.RemovePoolByAdmin(&_DVMFactory.TransactOpts, creator, baseToken, quoteToken, pool)
}

// RemovePoolByAdmin is a paid mutator transaction binding the contract method 0x43274b82.
//
// Solidity: function removePoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_DVMFactory *DVMFactoryTransactorSession) RemovePoolByAdmin(creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _DVMFactory.Contract.RemovePoolByAdmin(&_DVMFactory.TransactOpts, creator, baseToken, quoteToken, pool)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DVMFactory *DVMFactoryTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _DVMFactory.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DVMFactory *DVMFactorySession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _DVMFactory.Contract.TransferOwnership(&_DVMFactory.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DVMFactory *DVMFactoryTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _DVMFactory.Contract.TransferOwnership(&_DVMFactory.TransactOpts, newOwner)
}

// UpdateDefaultMaintainer is a paid mutator transaction binding the contract method 0x9e988ee3.
//
// Solidity: function updateDefaultMaintainer(address _newMaintainer) returns()
func (_DVMFactory *DVMFactoryTransactor) UpdateDefaultMaintainer(opts *bind.TransactOpts, _newMaintainer common.Address) (*types.Transaction, error) {
	return _DVMFactory.contract.Transact(opts, "updateDefaultMaintainer", _newMaintainer)
}

// UpdateDefaultMaintainer is a paid mutator transaction binding the contract method 0x9e988ee3.
//
// Solidity: function updateDefaultMaintainer(address _newMaintainer) returns()
func (_DVMFactory *DVMFactorySession) UpdateDefaultMaintainer(_newMaintainer common.Address) (*types.Transaction, error) {
	return _DVMFactory.Contract.UpdateDefaultMaintainer(&_DVMFactory.TransactOpts, _newMaintainer)
}

// UpdateDefaultMaintainer is a paid mutator transaction binding the contract method 0x9e988ee3.
//
// Solidity: function updateDefaultMaintainer(address _newMaintainer) returns()
func (_DVMFactory *DVMFactoryTransactorSession) UpdateDefaultMaintainer(_newMaintainer common.Address) (*types.Transaction, error) {
	return _DVMFactory.Contract.UpdateDefaultMaintainer(&_DVMFactory.TransactOpts, _newMaintainer)
}

// UpdateDvmTemplate is a paid mutator transaction binding the contract method 0xd99b8ad4.
//
// Solidity: function updateDvmTemplate(address _newDVMTemplate) returns()
func (_DVMFactory *DVMFactoryTransactor) UpdateDvmTemplate(opts *bind.TransactOpts, _newDVMTemplate common.Address) (*types.Transaction, error) {
	return _DVMFactory.contract.Transact(opts, "updateDvmTemplate", _newDVMTemplate)
}

// UpdateDvmTemplate is a paid mutator transaction binding the contract method 0xd99b8ad4.
//
// Solidity: function updateDvmTemplate(address _newDVMTemplate) returns()
func (_DVMFactory *DVMFactorySession) UpdateDvmTemplate(_newDVMTemplate common.Address) (*types.Transaction, error) {
	return _DVMFactory.Contract.UpdateDvmTemplate(&_DVMFactory.TransactOpts, _newDVMTemplate)
}

// UpdateDvmTemplate is a paid mutator transaction binding the contract method 0xd99b8ad4.
//
// Solidity: function updateDvmTemplate(address _newDVMTemplate) returns()
func (_DVMFactory *DVMFactoryTransactorSession) UpdateDvmTemplate(_newDVMTemplate common.Address) (*types.Transaction, error) {
	return _DVMFactory.Contract.UpdateDvmTemplate(&_DVMFactory.TransactOpts, _newDVMTemplate)
}

// DVMFactoryNewDVMIterator is returned from FilterNewDVM and is used to iterate over the raw logs and unpacked data for NewDVM events raised by the DVMFactory contract.
type DVMFactoryNewDVMIterator struct {
	Event *DVMFactoryNewDVM // Event containing the contract specifics and raw log

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
func (it *DVMFactoryNewDVMIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DVMFactoryNewDVM)
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
		it.Event = new(DVMFactoryNewDVM)
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
func (it *DVMFactoryNewDVMIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DVMFactoryNewDVMIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DVMFactoryNewDVM represents a NewDVM event raised by the DVMFactory contract.
type DVMFactoryNewDVM struct {
	BaseToken  common.Address
	QuoteToken common.Address
	Creator    common.Address
	Dvm        common.Address
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterNewDVM is a free log retrieval operation binding the contract event 0xaf5c5f12a80fc937520df6fcaed66262a4cc775e0f3fceaf7a7cfe476d9a751d.
//
// Solidity: event NewDVM(address baseToken, address quoteToken, address creator, address dvm)
func (_DVMFactory *DVMFactoryFilterer) FilterNewDVM(opts *bind.FilterOpts) (*DVMFactoryNewDVMIterator, error) {

	logs, sub, err := _DVMFactory.contract.FilterLogs(opts, "NewDVM")
	if err != nil {
		return nil, err
	}
	return &DVMFactoryNewDVMIterator{contract: _DVMFactory.contract, event: "NewDVM", logs: logs, sub: sub}, nil
}

// WatchNewDVM is a free log subscription operation binding the contract event 0xaf5c5f12a80fc937520df6fcaed66262a4cc775e0f3fceaf7a7cfe476d9a751d.
//
// Solidity: event NewDVM(address baseToken, address quoteToken, address creator, address dvm)
func (_DVMFactory *DVMFactoryFilterer) WatchNewDVM(opts *bind.WatchOpts, sink chan<- *DVMFactoryNewDVM) (event.Subscription, error) {

	logs, sub, err := _DVMFactory.contract.WatchLogs(opts, "NewDVM")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DVMFactoryNewDVM)
				if err := _DVMFactory.contract.UnpackLog(event, "NewDVM", log); err != nil {
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

// ParseNewDVM is a log parse operation binding the contract event 0xaf5c5f12a80fc937520df6fcaed66262a4cc775e0f3fceaf7a7cfe476d9a751d.
//
// Solidity: event NewDVM(address baseToken, address quoteToken, address creator, address dvm)
func (_DVMFactory *DVMFactoryFilterer) ParseNewDVM(log types.Log) (*DVMFactoryNewDVM, error) {
	event := new(DVMFactoryNewDVM)
	if err := _DVMFactory.contract.UnpackLog(event, "NewDVM", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DVMFactoryOwnershipTransferPreparedIterator is returned from FilterOwnershipTransferPrepared and is used to iterate over the raw logs and unpacked data for OwnershipTransferPrepared events raised by the DVMFactory contract.
type DVMFactoryOwnershipTransferPreparedIterator struct {
	Event *DVMFactoryOwnershipTransferPrepared // Event containing the contract specifics and raw log

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
func (it *DVMFactoryOwnershipTransferPreparedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DVMFactoryOwnershipTransferPrepared)
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
		it.Event = new(DVMFactoryOwnershipTransferPrepared)
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
func (it *DVMFactoryOwnershipTransferPreparedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DVMFactoryOwnershipTransferPreparedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DVMFactoryOwnershipTransferPrepared represents a OwnershipTransferPrepared event raised by the DVMFactory contract.
type DVMFactoryOwnershipTransferPrepared struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferPrepared is a free log retrieval operation binding the contract event 0xdcf55418cee3220104fef63f979ff3c4097ad240c0c43dcb33ce837748983e62.
//
// Solidity: event OwnershipTransferPrepared(address indexed previousOwner, address indexed newOwner)
func (_DVMFactory *DVMFactoryFilterer) FilterOwnershipTransferPrepared(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*DVMFactoryOwnershipTransferPreparedIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DVMFactory.contract.FilterLogs(opts, "OwnershipTransferPrepared", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &DVMFactoryOwnershipTransferPreparedIterator{contract: _DVMFactory.contract, event: "OwnershipTransferPrepared", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferPrepared is a free log subscription operation binding the contract event 0xdcf55418cee3220104fef63f979ff3c4097ad240c0c43dcb33ce837748983e62.
//
// Solidity: event OwnershipTransferPrepared(address indexed previousOwner, address indexed newOwner)
func (_DVMFactory *DVMFactoryFilterer) WatchOwnershipTransferPrepared(opts *bind.WatchOpts, sink chan<- *DVMFactoryOwnershipTransferPrepared, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DVMFactory.contract.WatchLogs(opts, "OwnershipTransferPrepared", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DVMFactoryOwnershipTransferPrepared)
				if err := _DVMFactory.contract.UnpackLog(event, "OwnershipTransferPrepared", log); err != nil {
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
func (_DVMFactory *DVMFactoryFilterer) ParseOwnershipTransferPrepared(log types.Log) (*DVMFactoryOwnershipTransferPrepared, error) {
	event := new(DVMFactoryOwnershipTransferPrepared)
	if err := _DVMFactory.contract.UnpackLog(event, "OwnershipTransferPrepared", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DVMFactoryOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the DVMFactory contract.
type DVMFactoryOwnershipTransferredIterator struct {
	Event *DVMFactoryOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *DVMFactoryOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DVMFactoryOwnershipTransferred)
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
		it.Event = new(DVMFactoryOwnershipTransferred)
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
func (it *DVMFactoryOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DVMFactoryOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DVMFactoryOwnershipTransferred represents a OwnershipTransferred event raised by the DVMFactory contract.
type DVMFactoryOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_DVMFactory *DVMFactoryFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*DVMFactoryOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DVMFactory.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &DVMFactoryOwnershipTransferredIterator{contract: _DVMFactory.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_DVMFactory *DVMFactoryFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *DVMFactoryOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DVMFactory.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DVMFactoryOwnershipTransferred)
				if err := _DVMFactory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_DVMFactory *DVMFactoryFilterer) ParseOwnershipTransferred(log types.Log) (*DVMFactoryOwnershipTransferred, error) {
	event := new(DVMFactoryOwnershipTransferred)
	if err := _DVMFactory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DVMFactoryRemoveDVMIterator is returned from FilterRemoveDVM and is used to iterate over the raw logs and unpacked data for RemoveDVM events raised by the DVMFactory contract.
type DVMFactoryRemoveDVMIterator struct {
	Event *DVMFactoryRemoveDVM // Event containing the contract specifics and raw log

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
func (it *DVMFactoryRemoveDVMIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DVMFactoryRemoveDVM)
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
		it.Event = new(DVMFactoryRemoveDVM)
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
func (it *DVMFactoryRemoveDVMIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DVMFactoryRemoveDVMIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DVMFactoryRemoveDVM represents a RemoveDVM event raised by the DVMFactory contract.
type DVMFactoryRemoveDVM struct {
	Dvm common.Address
	Raw types.Log // Blockchain specific contextual infos
}

// FilterRemoveDVM is a free log retrieval operation binding the contract event 0x63971a172c674ce2e9da5e027e9e81a54fd3aa74a2c246a2eb473dc0aa7f5cdd.
//
// Solidity: event RemoveDVM(address dvm)
func (_DVMFactory *DVMFactoryFilterer) FilterRemoveDVM(opts *bind.FilterOpts) (*DVMFactoryRemoveDVMIterator, error) {

	logs, sub, err := _DVMFactory.contract.FilterLogs(opts, "RemoveDVM")
	if err != nil {
		return nil, err
	}
	return &DVMFactoryRemoveDVMIterator{contract: _DVMFactory.contract, event: "RemoveDVM", logs: logs, sub: sub}, nil
}

// WatchRemoveDVM is a free log subscription operation binding the contract event 0x63971a172c674ce2e9da5e027e9e81a54fd3aa74a2c246a2eb473dc0aa7f5cdd.
//
// Solidity: event RemoveDVM(address dvm)
func (_DVMFactory *DVMFactoryFilterer) WatchRemoveDVM(opts *bind.WatchOpts, sink chan<- *DVMFactoryRemoveDVM) (event.Subscription, error) {

	logs, sub, err := _DVMFactory.contract.WatchLogs(opts, "RemoveDVM")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DVMFactoryRemoveDVM)
				if err := _DVMFactory.contract.UnpackLog(event, "RemoveDVM", log); err != nil {
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

// ParseRemoveDVM is a log parse operation binding the contract event 0x63971a172c674ce2e9da5e027e9e81a54fd3aa74a2c246a2eb473dc0aa7f5cdd.
//
// Solidity: event RemoveDVM(address dvm)
func (_DVMFactory *DVMFactoryFilterer) ParseRemoveDVM(log types.Log) (*DVMFactoryRemoveDVM, error) {
	event := new(DVMFactoryRemoveDVM)
	if err := _DVMFactory.contract.UnpackLog(event, "RemoveDVM", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
