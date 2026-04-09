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

// GSPFactoryMetaData contains all meta data concerning the GSPFactory contract.
var GSPFactoryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"cloneFactory\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"GSPTemplate\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"defaultMaintainer\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"GSP\",\"type\":\"address\"}],\"name\":\"NewGSP\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferPrepared\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"GSP\",\"type\":\"address\"}],\"name\":\"RemoveGSP\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"_CLONE_FACTORY_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_DEFAULT_MAINTAINER_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_GSP_TEMPLATE_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_NEW_OWNER_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_OWNER_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"_REGISTRY_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"_USER_REGISTRY_\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"addPoolByAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"claimOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"admin\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"lpFeeRate\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"mtFeeRate\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"i\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"k\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"priceLimit\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isOpenTWAP\",\"type\":\"bool\"}],\"name\":\"createDODOGasSavingPool\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"newGasSavingPool\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"}],\"name\":\"getDODOPool\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"machines\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"}],\"name\":\"getDODOPoolBidirection\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"baseToken0Machines\",\"type\":\"address[]\"},{\"internalType\":\"address[]\",\"name\":\"baseToken1Machines\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"}],\"name\":\"getDODOPoolByUser\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"machines\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"initOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"quoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"removePoolByAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newMaintainer\",\"type\":\"address\"}],\"name\":\"updateDefaultMaintainer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newGSPTemplate\",\"type\":\"address\"}],\"name\":\"updateGSPTemplate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// GSPFactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use GSPFactoryMetaData.ABI instead.
var GSPFactoryABI = GSPFactoryMetaData.ABI

// GSPFactory is an auto generated Go binding around an Ethereum contract.
type GSPFactory struct {
	GSPFactoryCaller     // Read-only binding to the contract
	GSPFactoryTransactor // Write-only binding to the contract
	GSPFactoryFilterer   // Log filterer for contract events
}

// GSPFactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type GSPFactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GSPFactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type GSPFactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GSPFactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type GSPFactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GSPFactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type GSPFactorySession struct {
	Contract     *GSPFactory       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// GSPFactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type GSPFactoryCallerSession struct {
	Contract *GSPFactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// GSPFactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type GSPFactoryTransactorSession struct {
	Contract     *GSPFactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// GSPFactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type GSPFactoryRaw struct {
	Contract *GSPFactory // Generic contract binding to access the raw methods on
}

// GSPFactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type GSPFactoryCallerRaw struct {
	Contract *GSPFactoryCaller // Generic read-only contract binding to access the raw methods on
}

// GSPFactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type GSPFactoryTransactorRaw struct {
	Contract *GSPFactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewGSPFactory creates a new instance of GSPFactory, bound to a specific deployed contract.
func NewGSPFactory(address common.Address, backend bind.ContractBackend) (*GSPFactory, error) {
	contract, err := bindGSPFactory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &GSPFactory{GSPFactoryCaller: GSPFactoryCaller{contract: contract}, GSPFactoryTransactor: GSPFactoryTransactor{contract: contract}, GSPFactoryFilterer: GSPFactoryFilterer{contract: contract}}, nil
}

// NewGSPFactoryCaller creates a new read-only instance of GSPFactory, bound to a specific deployed contract.
func NewGSPFactoryCaller(address common.Address, caller bind.ContractCaller) (*GSPFactoryCaller, error) {
	contract, err := bindGSPFactory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &GSPFactoryCaller{contract: contract}, nil
}

// NewGSPFactoryTransactor creates a new write-only instance of GSPFactory, bound to a specific deployed contract.
func NewGSPFactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*GSPFactoryTransactor, error) {
	contract, err := bindGSPFactory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &GSPFactoryTransactor{contract: contract}, nil
}

// NewGSPFactoryFilterer creates a new log filterer instance of GSPFactory, bound to a specific deployed contract.
func NewGSPFactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*GSPFactoryFilterer, error) {
	contract, err := bindGSPFactory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &GSPFactoryFilterer{contract: contract}, nil
}

// bindGSPFactory binds a generic wrapper to an already deployed contract.
func bindGSPFactory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := GSPFactoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_GSPFactory *GSPFactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _GSPFactory.Contract.GSPFactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_GSPFactory *GSPFactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GSPFactory.Contract.GSPFactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_GSPFactory *GSPFactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _GSPFactory.Contract.GSPFactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_GSPFactory *GSPFactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _GSPFactory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_GSPFactory *GSPFactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GSPFactory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_GSPFactory *GSPFactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _GSPFactory.Contract.contract.Transact(opts, method, params...)
}

// CLONEFACTORY is a free data retrieval call binding the contract method 0xeb774d05.
//
// Solidity: function _CLONE_FACTORY_() view returns(address)
func (_GSPFactory *GSPFactoryCaller) CLONEFACTORY(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _GSPFactory.contract.Call(opts, &out, "_CLONE_FACTORY_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CLONEFACTORY is a free data retrieval call binding the contract method 0xeb774d05.
//
// Solidity: function _CLONE_FACTORY_() view returns(address)
func (_GSPFactory *GSPFactorySession) CLONEFACTORY() (common.Address, error) {
	return _GSPFactory.Contract.CLONEFACTORY(&_GSPFactory.CallOpts)
}

// CLONEFACTORY is a free data retrieval call binding the contract method 0xeb774d05.
//
// Solidity: function _CLONE_FACTORY_() view returns(address)
func (_GSPFactory *GSPFactoryCallerSession) CLONEFACTORY() (common.Address, error) {
	return _GSPFactory.Contract.CLONEFACTORY(&_GSPFactory.CallOpts)
}

// DEFAULTMAINTAINER is a free data retrieval call binding the contract method 0x81ab4d0a.
//
// Solidity: function _DEFAULT_MAINTAINER_() view returns(address)
func (_GSPFactory *GSPFactoryCaller) DEFAULTMAINTAINER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _GSPFactory.contract.Call(opts, &out, "_DEFAULT_MAINTAINER_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DEFAULTMAINTAINER is a free data retrieval call binding the contract method 0x81ab4d0a.
//
// Solidity: function _DEFAULT_MAINTAINER_() view returns(address)
func (_GSPFactory *GSPFactorySession) DEFAULTMAINTAINER() (common.Address, error) {
	return _GSPFactory.Contract.DEFAULTMAINTAINER(&_GSPFactory.CallOpts)
}

// DEFAULTMAINTAINER is a free data retrieval call binding the contract method 0x81ab4d0a.
//
// Solidity: function _DEFAULT_MAINTAINER_() view returns(address)
func (_GSPFactory *GSPFactoryCallerSession) DEFAULTMAINTAINER() (common.Address, error) {
	return _GSPFactory.Contract.DEFAULTMAINTAINER(&_GSPFactory.CallOpts)
}

// GSPTEMPLATE is a free data retrieval call binding the contract method 0x8483a1c9.
//
// Solidity: function _GSP_TEMPLATE_() view returns(address)
func (_GSPFactory *GSPFactoryCaller) GSPTEMPLATE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _GSPFactory.contract.Call(opts, &out, "_GSP_TEMPLATE_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GSPTEMPLATE is a free data retrieval call binding the contract method 0x8483a1c9.
//
// Solidity: function _GSP_TEMPLATE_() view returns(address)
func (_GSPFactory *GSPFactorySession) GSPTEMPLATE() (common.Address, error) {
	return _GSPFactory.Contract.GSPTEMPLATE(&_GSPFactory.CallOpts)
}

// GSPTEMPLATE is a free data retrieval call binding the contract method 0x8483a1c9.
//
// Solidity: function _GSP_TEMPLATE_() view returns(address)
func (_GSPFactory *GSPFactoryCallerSession) GSPTEMPLATE() (common.Address, error) {
	return _GSPFactory.Contract.GSPTEMPLATE(&_GSPFactory.CallOpts)
}

// NEWOWNER is a free data retrieval call binding the contract method 0x8456db15.
//
// Solidity: function _NEW_OWNER_() view returns(address)
func (_GSPFactory *GSPFactoryCaller) NEWOWNER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _GSPFactory.contract.Call(opts, &out, "_NEW_OWNER_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NEWOWNER is a free data retrieval call binding the contract method 0x8456db15.
//
// Solidity: function _NEW_OWNER_() view returns(address)
func (_GSPFactory *GSPFactorySession) NEWOWNER() (common.Address, error) {
	return _GSPFactory.Contract.NEWOWNER(&_GSPFactory.CallOpts)
}

// NEWOWNER is a free data retrieval call binding the contract method 0x8456db15.
//
// Solidity: function _NEW_OWNER_() view returns(address)
func (_GSPFactory *GSPFactoryCallerSession) NEWOWNER() (common.Address, error) {
	return _GSPFactory.Contract.NEWOWNER(&_GSPFactory.CallOpts)
}

// OWNER is a free data retrieval call binding the contract method 0x16048bc4.
//
// Solidity: function _OWNER_() view returns(address)
func (_GSPFactory *GSPFactoryCaller) OWNER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _GSPFactory.contract.Call(opts, &out, "_OWNER_")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OWNER is a free data retrieval call binding the contract method 0x16048bc4.
//
// Solidity: function _OWNER_() view returns(address)
func (_GSPFactory *GSPFactorySession) OWNER() (common.Address, error) {
	return _GSPFactory.Contract.OWNER(&_GSPFactory.CallOpts)
}

// OWNER is a free data retrieval call binding the contract method 0x16048bc4.
//
// Solidity: function _OWNER_() view returns(address)
func (_GSPFactory *GSPFactoryCallerSession) OWNER() (common.Address, error) {
	return _GSPFactory.Contract.OWNER(&_GSPFactory.CallOpts)
}

// REGISTRY is a free data retrieval call binding the contract method 0xbdeb0a91.
//
// Solidity: function _REGISTRY_(address , address , uint256 ) view returns(address)
func (_GSPFactory *GSPFactoryCaller) REGISTRY(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _GSPFactory.contract.Call(opts, &out, "_REGISTRY_", arg0, arg1, arg2)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// REGISTRY is a free data retrieval call binding the contract method 0xbdeb0a91.
//
// Solidity: function _REGISTRY_(address , address , uint256 ) view returns(address)
func (_GSPFactory *GSPFactorySession) REGISTRY(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	return _GSPFactory.Contract.REGISTRY(&_GSPFactory.CallOpts, arg0, arg1, arg2)
}

// REGISTRY is a free data retrieval call binding the contract method 0xbdeb0a91.
//
// Solidity: function _REGISTRY_(address , address , uint256 ) view returns(address)
func (_GSPFactory *GSPFactoryCallerSession) REGISTRY(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (common.Address, error) {
	return _GSPFactory.Contract.REGISTRY(&_GSPFactory.CallOpts, arg0, arg1, arg2)
}

// USERREGISTRY is a free data retrieval call binding the contract method 0xa58888db.
//
// Solidity: function _USER_REGISTRY_(address , uint256 ) view returns(address)
func (_GSPFactory *GSPFactoryCaller) USERREGISTRY(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _GSPFactory.contract.Call(opts, &out, "_USER_REGISTRY_", arg0, arg1)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// USERREGISTRY is a free data retrieval call binding the contract method 0xa58888db.
//
// Solidity: function _USER_REGISTRY_(address , uint256 ) view returns(address)
func (_GSPFactory *GSPFactorySession) USERREGISTRY(arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	return _GSPFactory.Contract.USERREGISTRY(&_GSPFactory.CallOpts, arg0, arg1)
}

// USERREGISTRY is a free data retrieval call binding the contract method 0xa58888db.
//
// Solidity: function _USER_REGISTRY_(address , uint256 ) view returns(address)
func (_GSPFactory *GSPFactoryCallerSession) USERREGISTRY(arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	return _GSPFactory.Contract.USERREGISTRY(&_GSPFactory.CallOpts, arg0, arg1)
}

// GetDODOPool is a free data retrieval call binding the contract method 0x57a281dc.
//
// Solidity: function getDODOPool(address baseToken, address quoteToken) view returns(address[] machines)
func (_GSPFactory *GSPFactoryCaller) GetDODOPool(opts *bind.CallOpts, baseToken common.Address, quoteToken common.Address) ([]common.Address, error) {
	var out []interface{}
	err := _GSPFactory.contract.Call(opts, &out, "getDODOPool", baseToken, quoteToken)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetDODOPool is a free data retrieval call binding the contract method 0x57a281dc.
//
// Solidity: function getDODOPool(address baseToken, address quoteToken) view returns(address[] machines)
func (_GSPFactory *GSPFactorySession) GetDODOPool(baseToken common.Address, quoteToken common.Address) ([]common.Address, error) {
	return _GSPFactory.Contract.GetDODOPool(&_GSPFactory.CallOpts, baseToken, quoteToken)
}

// GetDODOPool is a free data retrieval call binding the contract method 0x57a281dc.
//
// Solidity: function getDODOPool(address baseToken, address quoteToken) view returns(address[] machines)
func (_GSPFactory *GSPFactoryCallerSession) GetDODOPool(baseToken common.Address, quoteToken common.Address) ([]common.Address, error) {
	return _GSPFactory.Contract.GetDODOPool(&_GSPFactory.CallOpts, baseToken, quoteToken)
}

// GetDODOPoolBidirection is a free data retrieval call binding the contract method 0x794e5538.
//
// Solidity: function getDODOPoolBidirection(address token0, address token1) view returns(address[] baseToken0Machines, address[] baseToken1Machines)
func (_GSPFactory *GSPFactoryCaller) GetDODOPoolBidirection(opts *bind.CallOpts, token0 common.Address, token1 common.Address) (struct {
	BaseToken0Machines []common.Address
	BaseToken1Machines []common.Address
}, error) {
	var out []interface{}
	err := _GSPFactory.contract.Call(opts, &out, "getDODOPoolBidirection", token0, token1)

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
func (_GSPFactory *GSPFactorySession) GetDODOPoolBidirection(token0 common.Address, token1 common.Address) (struct {
	BaseToken0Machines []common.Address
	BaseToken1Machines []common.Address
}, error) {
	return _GSPFactory.Contract.GetDODOPoolBidirection(&_GSPFactory.CallOpts, token0, token1)
}

// GetDODOPoolBidirection is a free data retrieval call binding the contract method 0x794e5538.
//
// Solidity: function getDODOPoolBidirection(address token0, address token1) view returns(address[] baseToken0Machines, address[] baseToken1Machines)
func (_GSPFactory *GSPFactoryCallerSession) GetDODOPoolBidirection(token0 common.Address, token1 common.Address) (struct {
	BaseToken0Machines []common.Address
	BaseToken1Machines []common.Address
}, error) {
	return _GSPFactory.Contract.GetDODOPoolBidirection(&_GSPFactory.CallOpts, token0, token1)
}

// GetDODOPoolByUser is a free data retrieval call binding the contract method 0xe65f7029.
//
// Solidity: function getDODOPoolByUser(address user) view returns(address[] machines)
func (_GSPFactory *GSPFactoryCaller) GetDODOPoolByUser(opts *bind.CallOpts, user common.Address) ([]common.Address, error) {
	var out []interface{}
	err := _GSPFactory.contract.Call(opts, &out, "getDODOPoolByUser", user)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetDODOPoolByUser is a free data retrieval call binding the contract method 0xe65f7029.
//
// Solidity: function getDODOPoolByUser(address user) view returns(address[] machines)
func (_GSPFactory *GSPFactorySession) GetDODOPoolByUser(user common.Address) ([]common.Address, error) {
	return _GSPFactory.Contract.GetDODOPoolByUser(&_GSPFactory.CallOpts, user)
}

// GetDODOPoolByUser is a free data retrieval call binding the contract method 0xe65f7029.
//
// Solidity: function getDODOPoolByUser(address user) view returns(address[] machines)
func (_GSPFactory *GSPFactoryCallerSession) GetDODOPoolByUser(user common.Address) ([]common.Address, error) {
	return _GSPFactory.Contract.GetDODOPoolByUser(&_GSPFactory.CallOpts, user)
}

// AddPoolByAdmin is a paid mutator transaction binding the contract method 0x39d00ef9.
//
// Solidity: function addPoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_GSPFactory *GSPFactoryTransactor) AddPoolByAdmin(opts *bind.TransactOpts, creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _GSPFactory.contract.Transact(opts, "addPoolByAdmin", creator, baseToken, quoteToken, pool)
}

// AddPoolByAdmin is a paid mutator transaction binding the contract method 0x39d00ef9.
//
// Solidity: function addPoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_GSPFactory *GSPFactorySession) AddPoolByAdmin(creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _GSPFactory.Contract.AddPoolByAdmin(&_GSPFactory.TransactOpts, creator, baseToken, quoteToken, pool)
}

// AddPoolByAdmin is a paid mutator transaction binding the contract method 0x39d00ef9.
//
// Solidity: function addPoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_GSPFactory *GSPFactoryTransactorSession) AddPoolByAdmin(creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _GSPFactory.Contract.AddPoolByAdmin(&_GSPFactory.TransactOpts, creator, baseToken, quoteToken, pool)
}

// ClaimOwnership is a paid mutator transaction binding the contract method 0x4e71e0c8.
//
// Solidity: function claimOwnership() returns()
func (_GSPFactory *GSPFactoryTransactor) ClaimOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GSPFactory.contract.Transact(opts, "claimOwnership")
}

// ClaimOwnership is a paid mutator transaction binding the contract method 0x4e71e0c8.
//
// Solidity: function claimOwnership() returns()
func (_GSPFactory *GSPFactorySession) ClaimOwnership() (*types.Transaction, error) {
	return _GSPFactory.Contract.ClaimOwnership(&_GSPFactory.TransactOpts)
}

// ClaimOwnership is a paid mutator transaction binding the contract method 0x4e71e0c8.
//
// Solidity: function claimOwnership() returns()
func (_GSPFactory *GSPFactoryTransactorSession) ClaimOwnership() (*types.Transaction, error) {
	return _GSPFactory.Contract.ClaimOwnership(&_GSPFactory.TransactOpts)
}

// CreateDODOGasSavingPool is a paid mutator transaction binding the contract method 0x9f575593.
//
// Solidity: function createDODOGasSavingPool(address admin, address baseToken, address quoteToken, uint256 lpFeeRate, uint256 mtFeeRate, uint256 i, uint256 k, uint256 priceLimit, bool isOpenTWAP) returns(address newGasSavingPool)
func (_GSPFactory *GSPFactoryTransactor) CreateDODOGasSavingPool(opts *bind.TransactOpts, admin common.Address, baseToken common.Address, quoteToken common.Address, lpFeeRate *big.Int, mtFeeRate *big.Int, i *big.Int, k *big.Int, priceLimit *big.Int, isOpenTWAP bool) (*types.Transaction, error) {
	return _GSPFactory.contract.Transact(opts, "createDODOGasSavingPool", admin, baseToken, quoteToken, lpFeeRate, mtFeeRate, i, k, priceLimit, isOpenTWAP)
}

// CreateDODOGasSavingPool is a paid mutator transaction binding the contract method 0x9f575593.
//
// Solidity: function createDODOGasSavingPool(address admin, address baseToken, address quoteToken, uint256 lpFeeRate, uint256 mtFeeRate, uint256 i, uint256 k, uint256 priceLimit, bool isOpenTWAP) returns(address newGasSavingPool)
func (_GSPFactory *GSPFactorySession) CreateDODOGasSavingPool(admin common.Address, baseToken common.Address, quoteToken common.Address, lpFeeRate *big.Int, mtFeeRate *big.Int, i *big.Int, k *big.Int, priceLimit *big.Int, isOpenTWAP bool) (*types.Transaction, error) {
	return _GSPFactory.Contract.CreateDODOGasSavingPool(&_GSPFactory.TransactOpts, admin, baseToken, quoteToken, lpFeeRate, mtFeeRate, i, k, priceLimit, isOpenTWAP)
}

// CreateDODOGasSavingPool is a paid mutator transaction binding the contract method 0x9f575593.
//
// Solidity: function createDODOGasSavingPool(address admin, address baseToken, address quoteToken, uint256 lpFeeRate, uint256 mtFeeRate, uint256 i, uint256 k, uint256 priceLimit, bool isOpenTWAP) returns(address newGasSavingPool)
func (_GSPFactory *GSPFactoryTransactorSession) CreateDODOGasSavingPool(admin common.Address, baseToken common.Address, quoteToken common.Address, lpFeeRate *big.Int, mtFeeRate *big.Int, i *big.Int, k *big.Int, priceLimit *big.Int, isOpenTWAP bool) (*types.Transaction, error) {
	return _GSPFactory.Contract.CreateDODOGasSavingPool(&_GSPFactory.TransactOpts, admin, baseToken, quoteToken, lpFeeRate, mtFeeRate, i, k, priceLimit, isOpenTWAP)
}

// InitOwner is a paid mutator transaction binding the contract method 0x0d009297.
//
// Solidity: function initOwner(address newOwner) returns()
func (_GSPFactory *GSPFactoryTransactor) InitOwner(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _GSPFactory.contract.Transact(opts, "initOwner", newOwner)
}

// InitOwner is a paid mutator transaction binding the contract method 0x0d009297.
//
// Solidity: function initOwner(address newOwner) returns()
func (_GSPFactory *GSPFactorySession) InitOwner(newOwner common.Address) (*types.Transaction, error) {
	return _GSPFactory.Contract.InitOwner(&_GSPFactory.TransactOpts, newOwner)
}

// InitOwner is a paid mutator transaction binding the contract method 0x0d009297.
//
// Solidity: function initOwner(address newOwner) returns()
func (_GSPFactory *GSPFactoryTransactorSession) InitOwner(newOwner common.Address) (*types.Transaction, error) {
	return _GSPFactory.Contract.InitOwner(&_GSPFactory.TransactOpts, newOwner)
}

// RemovePoolByAdmin is a paid mutator transaction binding the contract method 0x43274b82.
//
// Solidity: function removePoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_GSPFactory *GSPFactoryTransactor) RemovePoolByAdmin(opts *bind.TransactOpts, creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _GSPFactory.contract.Transact(opts, "removePoolByAdmin", creator, baseToken, quoteToken, pool)
}

// RemovePoolByAdmin is a paid mutator transaction binding the contract method 0x43274b82.
//
// Solidity: function removePoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_GSPFactory *GSPFactorySession) RemovePoolByAdmin(creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _GSPFactory.Contract.RemovePoolByAdmin(&_GSPFactory.TransactOpts, creator, baseToken, quoteToken, pool)
}

// RemovePoolByAdmin is a paid mutator transaction binding the contract method 0x43274b82.
//
// Solidity: function removePoolByAdmin(address creator, address baseToken, address quoteToken, address pool) returns()
func (_GSPFactory *GSPFactoryTransactorSession) RemovePoolByAdmin(creator common.Address, baseToken common.Address, quoteToken common.Address, pool common.Address) (*types.Transaction, error) {
	return _GSPFactory.Contract.RemovePoolByAdmin(&_GSPFactory.TransactOpts, creator, baseToken, quoteToken, pool)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_GSPFactory *GSPFactoryTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _GSPFactory.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_GSPFactory *GSPFactorySession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _GSPFactory.Contract.TransferOwnership(&_GSPFactory.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_GSPFactory *GSPFactoryTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _GSPFactory.Contract.TransferOwnership(&_GSPFactory.TransactOpts, newOwner)
}

// UpdateDefaultMaintainer is a paid mutator transaction binding the contract method 0x9e988ee3.
//
// Solidity: function updateDefaultMaintainer(address _newMaintainer) returns()
func (_GSPFactory *GSPFactoryTransactor) UpdateDefaultMaintainer(opts *bind.TransactOpts, _newMaintainer common.Address) (*types.Transaction, error) {
	return _GSPFactory.contract.Transact(opts, "updateDefaultMaintainer", _newMaintainer)
}

// UpdateDefaultMaintainer is a paid mutator transaction binding the contract method 0x9e988ee3.
//
// Solidity: function updateDefaultMaintainer(address _newMaintainer) returns()
func (_GSPFactory *GSPFactorySession) UpdateDefaultMaintainer(_newMaintainer common.Address) (*types.Transaction, error) {
	return _GSPFactory.Contract.UpdateDefaultMaintainer(&_GSPFactory.TransactOpts, _newMaintainer)
}

// UpdateDefaultMaintainer is a paid mutator transaction binding the contract method 0x9e988ee3.
//
// Solidity: function updateDefaultMaintainer(address _newMaintainer) returns()
func (_GSPFactory *GSPFactoryTransactorSession) UpdateDefaultMaintainer(_newMaintainer common.Address) (*types.Transaction, error) {
	return _GSPFactory.Contract.UpdateDefaultMaintainer(&_GSPFactory.TransactOpts, _newMaintainer)
}

// UpdateGSPTemplate is a paid mutator transaction binding the contract method 0xe56b6a9b.
//
// Solidity: function updateGSPTemplate(address _newGSPTemplate) returns()
func (_GSPFactory *GSPFactoryTransactor) UpdateGSPTemplate(opts *bind.TransactOpts, _newGSPTemplate common.Address) (*types.Transaction, error) {
	return _GSPFactory.contract.Transact(opts, "updateGSPTemplate", _newGSPTemplate)
}

// UpdateGSPTemplate is a paid mutator transaction binding the contract method 0xe56b6a9b.
//
// Solidity: function updateGSPTemplate(address _newGSPTemplate) returns()
func (_GSPFactory *GSPFactorySession) UpdateGSPTemplate(_newGSPTemplate common.Address) (*types.Transaction, error) {
	return _GSPFactory.Contract.UpdateGSPTemplate(&_GSPFactory.TransactOpts, _newGSPTemplate)
}

// UpdateGSPTemplate is a paid mutator transaction binding the contract method 0xe56b6a9b.
//
// Solidity: function updateGSPTemplate(address _newGSPTemplate) returns()
func (_GSPFactory *GSPFactoryTransactorSession) UpdateGSPTemplate(_newGSPTemplate common.Address) (*types.Transaction, error) {
	return _GSPFactory.Contract.UpdateGSPTemplate(&_GSPFactory.TransactOpts, _newGSPTemplate)
}

// GSPFactoryNewGSPIterator is returned from FilterNewGSP and is used to iterate over the raw logs and unpacked data for NewGSP events raised by the GSPFactory contract.
type GSPFactoryNewGSPIterator struct {
	Event *GSPFactoryNewGSP // Event containing the contract specifics and raw log

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
func (it *GSPFactoryNewGSPIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GSPFactoryNewGSP)
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
		it.Event = new(GSPFactoryNewGSP)
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
func (it *GSPFactoryNewGSPIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GSPFactoryNewGSPIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GSPFactoryNewGSP represents a NewGSP event raised by the GSPFactory contract.
type GSPFactoryNewGSP struct {
	BaseToken  common.Address
	QuoteToken common.Address
	Creator    common.Address
	GSP        common.Address
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterNewGSP is a free log retrieval operation binding the contract event 0x9f20c9f8d79582ed814449029f45414a72f448edaa021dc66ebaa8855dbebf44.
//
// Solidity: event NewGSP(address baseToken, address quoteToken, address creator, address GSP)
func (_GSPFactory *GSPFactoryFilterer) FilterNewGSP(opts *bind.FilterOpts) (*GSPFactoryNewGSPIterator, error) {

	logs, sub, err := _GSPFactory.contract.FilterLogs(opts, "NewGSP")
	if err != nil {
		return nil, err
	}
	return &GSPFactoryNewGSPIterator{contract: _GSPFactory.contract, event: "NewGSP", logs: logs, sub: sub}, nil
}

// WatchNewGSP is a free log subscription operation binding the contract event 0x9f20c9f8d79582ed814449029f45414a72f448edaa021dc66ebaa8855dbebf44.
//
// Solidity: event NewGSP(address baseToken, address quoteToken, address creator, address GSP)
func (_GSPFactory *GSPFactoryFilterer) WatchNewGSP(opts *bind.WatchOpts, sink chan<- *GSPFactoryNewGSP) (event.Subscription, error) {

	logs, sub, err := _GSPFactory.contract.WatchLogs(opts, "NewGSP")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GSPFactoryNewGSP)
				if err := _GSPFactory.contract.UnpackLog(event, "NewGSP", log); err != nil {
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

// ParseNewGSP is a log parse operation binding the contract event 0x9f20c9f8d79582ed814449029f45414a72f448edaa021dc66ebaa8855dbebf44.
//
// Solidity: event NewGSP(address baseToken, address quoteToken, address creator, address GSP)
func (_GSPFactory *GSPFactoryFilterer) ParseNewGSP(log types.Log) (*GSPFactoryNewGSP, error) {
	event := new(GSPFactoryNewGSP)
	if err := _GSPFactory.contract.UnpackLog(event, "NewGSP", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// GSPFactoryOwnershipTransferPreparedIterator is returned from FilterOwnershipTransferPrepared and is used to iterate over the raw logs and unpacked data for OwnershipTransferPrepared events raised by the GSPFactory contract.
type GSPFactoryOwnershipTransferPreparedIterator struct {
	Event *GSPFactoryOwnershipTransferPrepared // Event containing the contract specifics and raw log

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
func (it *GSPFactoryOwnershipTransferPreparedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GSPFactoryOwnershipTransferPrepared)
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
		it.Event = new(GSPFactoryOwnershipTransferPrepared)
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
func (it *GSPFactoryOwnershipTransferPreparedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GSPFactoryOwnershipTransferPreparedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GSPFactoryOwnershipTransferPrepared represents a OwnershipTransferPrepared event raised by the GSPFactory contract.
type GSPFactoryOwnershipTransferPrepared struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferPrepared is a free log retrieval operation binding the contract event 0xdcf55418cee3220104fef63f979ff3c4097ad240c0c43dcb33ce837748983e62.
//
// Solidity: event OwnershipTransferPrepared(address indexed previousOwner, address indexed newOwner)
func (_GSPFactory *GSPFactoryFilterer) FilterOwnershipTransferPrepared(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*GSPFactoryOwnershipTransferPreparedIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _GSPFactory.contract.FilterLogs(opts, "OwnershipTransferPrepared", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &GSPFactoryOwnershipTransferPreparedIterator{contract: _GSPFactory.contract, event: "OwnershipTransferPrepared", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferPrepared is a free log subscription operation binding the contract event 0xdcf55418cee3220104fef63f979ff3c4097ad240c0c43dcb33ce837748983e62.
//
// Solidity: event OwnershipTransferPrepared(address indexed previousOwner, address indexed newOwner)
func (_GSPFactory *GSPFactoryFilterer) WatchOwnershipTransferPrepared(opts *bind.WatchOpts, sink chan<- *GSPFactoryOwnershipTransferPrepared, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _GSPFactory.contract.WatchLogs(opts, "OwnershipTransferPrepared", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GSPFactoryOwnershipTransferPrepared)
				if err := _GSPFactory.contract.UnpackLog(event, "OwnershipTransferPrepared", log); err != nil {
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
func (_GSPFactory *GSPFactoryFilterer) ParseOwnershipTransferPrepared(log types.Log) (*GSPFactoryOwnershipTransferPrepared, error) {
	event := new(GSPFactoryOwnershipTransferPrepared)
	if err := _GSPFactory.contract.UnpackLog(event, "OwnershipTransferPrepared", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// GSPFactoryOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the GSPFactory contract.
type GSPFactoryOwnershipTransferredIterator struct {
	Event *GSPFactoryOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *GSPFactoryOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GSPFactoryOwnershipTransferred)
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
		it.Event = new(GSPFactoryOwnershipTransferred)
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
func (it *GSPFactoryOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GSPFactoryOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GSPFactoryOwnershipTransferred represents a OwnershipTransferred event raised by the GSPFactory contract.
type GSPFactoryOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_GSPFactory *GSPFactoryFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*GSPFactoryOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _GSPFactory.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &GSPFactoryOwnershipTransferredIterator{contract: _GSPFactory.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_GSPFactory *GSPFactoryFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *GSPFactoryOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _GSPFactory.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GSPFactoryOwnershipTransferred)
				if err := _GSPFactory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_GSPFactory *GSPFactoryFilterer) ParseOwnershipTransferred(log types.Log) (*GSPFactoryOwnershipTransferred, error) {
	event := new(GSPFactoryOwnershipTransferred)
	if err := _GSPFactory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// GSPFactoryRemoveGSPIterator is returned from FilterRemoveGSP and is used to iterate over the raw logs and unpacked data for RemoveGSP events raised by the GSPFactory contract.
type GSPFactoryRemoveGSPIterator struct {
	Event *GSPFactoryRemoveGSP // Event containing the contract specifics and raw log

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
func (it *GSPFactoryRemoveGSPIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GSPFactoryRemoveGSP)
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
		it.Event = new(GSPFactoryRemoveGSP)
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
func (it *GSPFactoryRemoveGSPIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GSPFactoryRemoveGSPIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GSPFactoryRemoveGSP represents a RemoveGSP event raised by the GSPFactory contract.
type GSPFactoryRemoveGSP struct {
	GSP common.Address
	Raw types.Log // Blockchain specific contextual infos
}

// FilterRemoveGSP is a free log retrieval operation binding the contract event 0xb1c398f1e95b9c57ef15a49886bc596226b69ff354caa5e0fdfa26778c47ce7e.
//
// Solidity: event RemoveGSP(address GSP)
func (_GSPFactory *GSPFactoryFilterer) FilterRemoveGSP(opts *bind.FilterOpts) (*GSPFactoryRemoveGSPIterator, error) {

	logs, sub, err := _GSPFactory.contract.FilterLogs(opts, "RemoveGSP")
	if err != nil {
		return nil, err
	}
	return &GSPFactoryRemoveGSPIterator{contract: _GSPFactory.contract, event: "RemoveGSP", logs: logs, sub: sub}, nil
}

// WatchRemoveGSP is a free log subscription operation binding the contract event 0xb1c398f1e95b9c57ef15a49886bc596226b69ff354caa5e0fdfa26778c47ce7e.
//
// Solidity: event RemoveGSP(address GSP)
func (_GSPFactory *GSPFactoryFilterer) WatchRemoveGSP(opts *bind.WatchOpts, sink chan<- *GSPFactoryRemoveGSP) (event.Subscription, error) {

	logs, sub, err := _GSPFactory.contract.WatchLogs(opts, "RemoveGSP")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GSPFactoryRemoveGSP)
				if err := _GSPFactory.contract.UnpackLog(event, "RemoveGSP", log); err != nil {
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

// ParseRemoveGSP is a log parse operation binding the contract event 0xb1c398f1e95b9c57ef15a49886bc596226b69ff354caa5e0fdfa26778c47ce7e.
//
// Solidity: event RemoveGSP(address GSP)
func (_GSPFactory *GSPFactoryFilterer) ParseRemoveGSP(log types.Log) (*GSPFactoryRemoveGSP, error) {
	event := new(GSPFactoryRemoveGSP)
	if err := _GSPFactory.contract.UnpackLog(event, "RemoveGSP", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
