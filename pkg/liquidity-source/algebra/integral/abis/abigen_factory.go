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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_poolDeployer\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"deployer\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"CustomPool\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"newDefaultCommunityFee\",\"type\":\"uint16\"}],\"name\":\"DefaultCommunityFee\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"newDefaultFee\",\"type\":\"uint16\"}],\"name\":\"DefaultFee\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"defaultPluginFactoryAddress\",\"type\":\"address\"}],\"name\":\"DefaultPluginFactory\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"newDefaultTickspacing\",\"type\":\"int24\"}],\"name\":\"DefaultTickspacing\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferStarted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"name\":\"Pool\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"RenounceOwnershipFinish\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"finishTimestamp\",\"type\":\"uint256\"}],\"name\":\"RenounceOwnershipStart\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"RenounceOwnershipStop\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"previousAdminRole\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"newAdminRole\",\"type\":\"bytes32\"}],\"name\":\"RoleAdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleGranted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newVaultFactory\",\"type\":\"address\"}],\"name\":\"VaultFactory\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"CUSTOM_POOL_DEPLOYER\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"DEFAULT_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"POOLS_ADMINISTRATOR_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"POOL_INIT_CODE_HASH\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"acceptOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"deployer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"}],\"name\":\"computeCustomPoolAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"customPool\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"}],\"name\":\"computePoolAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"deployer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"createCustomPool\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"customPool\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"createPool\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pool\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"customPoolByPair\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"defaultCommunityFee\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"defaultConfigurationForPool\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"communityFee\",\"type\":\"uint16\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"uint16\",\"name\":\"fee\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"defaultFee\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"defaultPluginFactory\",\"outputs\":[{\"internalType\":\"contractIAlgebraPluginFactory\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"defaultTickspacing\",\"outputs\":[{\"internalType\":\"int24\",\"name\":\"\",\"type\":\"int24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleAdmin\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"getRoleMember\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleMemberCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"grantRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasRole\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasRoleOrOwner\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pendingOwner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"poolByPair\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"poolDeployer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnershipStartTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"renounceRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"newDefaultCommunityFee\",\"type\":\"uint16\"}],\"name\":\"setDefaultCommunityFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"newDefaultFee\",\"type\":\"uint16\"}],\"name\":\"setDefaultFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newDefaultPluginFactory\",\"type\":\"address\"}],\"name\":\"setDefaultPluginFactory\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int24\",\"name\":\"newDefaultTickspacing\",\"type\":\"int24\"}],\"name\":\"setDefaultTickspacing\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newVaultFactory\",\"type\":\"address\"}],\"name\":\"setVaultFactory\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"startRenounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"stopRenounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"vaultFactory\",\"outputs\":[{\"internalType\":\"contractIAlgebraVaultFactory\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
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

// CUSTOMPOOLDEPLOYER is a free data retrieval call binding the contract method 0x07810754.
//
// Solidity: function CUSTOM_POOL_DEPLOYER() view returns(bytes32)
func (_Factory *FactoryCaller) CUSTOMPOOLDEPLOYER(opts *bind.CallOpts) ([32]byte, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "CUSTOM_POOL_DEPLOYER")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CUSTOMPOOLDEPLOYER is a free data retrieval call binding the contract method 0x07810754.
//
// Solidity: function CUSTOM_POOL_DEPLOYER() view returns(bytes32)
func (_Factory *FactorySession) CUSTOMPOOLDEPLOYER() ([32]byte, error) {
	return _Factory.Contract.CUSTOMPOOLDEPLOYER(&_Factory.CallOpts)
}

// CUSTOMPOOLDEPLOYER is a free data retrieval call binding the contract method 0x07810754.
//
// Solidity: function CUSTOM_POOL_DEPLOYER() view returns(bytes32)
func (_Factory *FactoryCallerSession) CUSTOMPOOLDEPLOYER() ([32]byte, error) {
	return _Factory.Contract.CUSTOMPOOLDEPLOYER(&_Factory.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Factory *FactoryCaller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "DEFAULT_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Factory *FactorySession) DEFAULTADMINROLE() ([32]byte, error) {
	return _Factory.Contract.DEFAULTADMINROLE(&_Factory.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Factory *FactoryCallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _Factory.Contract.DEFAULTADMINROLE(&_Factory.CallOpts)
}

// POOLSADMINISTRATORROLE is a free data retrieval call binding the contract method 0xb500a48b.
//
// Solidity: function POOLS_ADMINISTRATOR_ROLE() view returns(bytes32)
func (_Factory *FactoryCaller) POOLSADMINISTRATORROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "POOLS_ADMINISTRATOR_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// POOLSADMINISTRATORROLE is a free data retrieval call binding the contract method 0xb500a48b.
//
// Solidity: function POOLS_ADMINISTRATOR_ROLE() view returns(bytes32)
func (_Factory *FactorySession) POOLSADMINISTRATORROLE() ([32]byte, error) {
	return _Factory.Contract.POOLSADMINISTRATORROLE(&_Factory.CallOpts)
}

// POOLSADMINISTRATORROLE is a free data retrieval call binding the contract method 0xb500a48b.
//
// Solidity: function POOLS_ADMINISTRATOR_ROLE() view returns(bytes32)
func (_Factory *FactoryCallerSession) POOLSADMINISTRATORROLE() ([32]byte, error) {
	return _Factory.Contract.POOLSADMINISTRATORROLE(&_Factory.CallOpts)
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

// ComputeCustomPoolAddress is a free data retrieval call binding the contract method 0x1ba89df4.
//
// Solidity: function computeCustomPoolAddress(address deployer, address token0, address token1) view returns(address customPool)
func (_Factory *FactoryCaller) ComputeCustomPoolAddress(opts *bind.CallOpts, deployer common.Address, token0 common.Address, token1 common.Address) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "computeCustomPoolAddress", deployer, token0, token1)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ComputeCustomPoolAddress is a free data retrieval call binding the contract method 0x1ba89df4.
//
// Solidity: function computeCustomPoolAddress(address deployer, address token0, address token1) view returns(address customPool)
func (_Factory *FactorySession) ComputeCustomPoolAddress(deployer common.Address, token0 common.Address, token1 common.Address) (common.Address, error) {
	return _Factory.Contract.ComputeCustomPoolAddress(&_Factory.CallOpts, deployer, token0, token1)
}

// ComputeCustomPoolAddress is a free data retrieval call binding the contract method 0x1ba89df4.
//
// Solidity: function computeCustomPoolAddress(address deployer, address token0, address token1) view returns(address customPool)
func (_Factory *FactoryCallerSession) ComputeCustomPoolAddress(deployer common.Address, token0 common.Address, token1 common.Address) (common.Address, error) {
	return _Factory.Contract.ComputeCustomPoolAddress(&_Factory.CallOpts, deployer, token0, token1)
}

// ComputePoolAddress is a free data retrieval call binding the contract method 0xd8ed2241.
//
// Solidity: function computePoolAddress(address token0, address token1) view returns(address pool)
func (_Factory *FactoryCaller) ComputePoolAddress(opts *bind.CallOpts, token0 common.Address, token1 common.Address) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "computePoolAddress", token0, token1)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ComputePoolAddress is a free data retrieval call binding the contract method 0xd8ed2241.
//
// Solidity: function computePoolAddress(address token0, address token1) view returns(address pool)
func (_Factory *FactorySession) ComputePoolAddress(token0 common.Address, token1 common.Address) (common.Address, error) {
	return _Factory.Contract.ComputePoolAddress(&_Factory.CallOpts, token0, token1)
}

// ComputePoolAddress is a free data retrieval call binding the contract method 0xd8ed2241.
//
// Solidity: function computePoolAddress(address token0, address token1) view returns(address pool)
func (_Factory *FactoryCallerSession) ComputePoolAddress(token0 common.Address, token1 common.Address) (common.Address, error) {
	return _Factory.Contract.ComputePoolAddress(&_Factory.CallOpts, token0, token1)
}

// CustomPoolByPair is a free data retrieval call binding the contract method 0x23da36cc.
//
// Solidity: function customPoolByPair(address , address , address ) view returns(address)
func (_Factory *FactoryCaller) CustomPoolByPair(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 common.Address) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "customPoolByPair", arg0, arg1, arg2)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CustomPoolByPair is a free data retrieval call binding the contract method 0x23da36cc.
//
// Solidity: function customPoolByPair(address , address , address ) view returns(address)
func (_Factory *FactorySession) CustomPoolByPair(arg0 common.Address, arg1 common.Address, arg2 common.Address) (common.Address, error) {
	return _Factory.Contract.CustomPoolByPair(&_Factory.CallOpts, arg0, arg1, arg2)
}

// CustomPoolByPair is a free data retrieval call binding the contract method 0x23da36cc.
//
// Solidity: function customPoolByPair(address , address , address ) view returns(address)
func (_Factory *FactoryCallerSession) CustomPoolByPair(arg0 common.Address, arg1 common.Address, arg2 common.Address) (common.Address, error) {
	return _Factory.Contract.CustomPoolByPair(&_Factory.CallOpts, arg0, arg1, arg2)
}

// DefaultCommunityFee is a free data retrieval call binding the contract method 0x2f8a39dd.
//
// Solidity: function defaultCommunityFee() view returns(uint16)
func (_Factory *FactoryCaller) DefaultCommunityFee(opts *bind.CallOpts) (uint16, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "defaultCommunityFee")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// DefaultCommunityFee is a free data retrieval call binding the contract method 0x2f8a39dd.
//
// Solidity: function defaultCommunityFee() view returns(uint16)
func (_Factory *FactorySession) DefaultCommunityFee() (uint16, error) {
	return _Factory.Contract.DefaultCommunityFee(&_Factory.CallOpts)
}

// DefaultCommunityFee is a free data retrieval call binding the contract method 0x2f8a39dd.
//
// Solidity: function defaultCommunityFee() view returns(uint16)
func (_Factory *FactoryCallerSession) DefaultCommunityFee() (uint16, error) {
	return _Factory.Contract.DefaultCommunityFee(&_Factory.CallOpts)
}

// DefaultConfigurationForPool is a free data retrieval call binding the contract method 0x25b355d6.
//
// Solidity: function defaultConfigurationForPool() view returns(uint16 communityFee, int24 tickSpacing, uint16 fee)
func (_Factory *FactoryCaller) DefaultConfigurationForPool(opts *bind.CallOpts) (struct {
	CommunityFee uint16
	TickSpacing  *big.Int
	Fee          uint16
}, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "defaultConfigurationForPool")

	outstruct := new(struct {
		CommunityFee uint16
		TickSpacing  *big.Int
		Fee          uint16
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.CommunityFee = *abi.ConvertType(out[0], new(uint16)).(*uint16)
	outstruct.TickSpacing = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Fee = *abi.ConvertType(out[2], new(uint16)).(*uint16)

	return *outstruct, err

}

// DefaultConfigurationForPool is a free data retrieval call binding the contract method 0x25b355d6.
//
// Solidity: function defaultConfigurationForPool() view returns(uint16 communityFee, int24 tickSpacing, uint16 fee)
func (_Factory *FactorySession) DefaultConfigurationForPool() (struct {
	CommunityFee uint16
	TickSpacing  *big.Int
	Fee          uint16
}, error) {
	return _Factory.Contract.DefaultConfigurationForPool(&_Factory.CallOpts)
}

// DefaultConfigurationForPool is a free data retrieval call binding the contract method 0x25b355d6.
//
// Solidity: function defaultConfigurationForPool() view returns(uint16 communityFee, int24 tickSpacing, uint16 fee)
func (_Factory *FactoryCallerSession) DefaultConfigurationForPool() (struct {
	CommunityFee uint16
	TickSpacing  *big.Int
	Fee          uint16
}, error) {
	return _Factory.Contract.DefaultConfigurationForPool(&_Factory.CallOpts)
}

// DefaultFee is a free data retrieval call binding the contract method 0x5a6c72d0.
//
// Solidity: function defaultFee() view returns(uint16)
func (_Factory *FactoryCaller) DefaultFee(opts *bind.CallOpts) (uint16, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "defaultFee")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// DefaultFee is a free data retrieval call binding the contract method 0x5a6c72d0.
//
// Solidity: function defaultFee() view returns(uint16)
func (_Factory *FactorySession) DefaultFee() (uint16, error) {
	return _Factory.Contract.DefaultFee(&_Factory.CallOpts)
}

// DefaultFee is a free data retrieval call binding the contract method 0x5a6c72d0.
//
// Solidity: function defaultFee() view returns(uint16)
func (_Factory *FactoryCallerSession) DefaultFee() (uint16, error) {
	return _Factory.Contract.DefaultFee(&_Factory.CallOpts)
}

// DefaultPluginFactory is a free data retrieval call binding the contract method 0xd0ad2792.
//
// Solidity: function defaultPluginFactory() view returns(address)
func (_Factory *FactoryCaller) DefaultPluginFactory(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "defaultPluginFactory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DefaultPluginFactory is a free data retrieval call binding the contract method 0xd0ad2792.
//
// Solidity: function defaultPluginFactory() view returns(address)
func (_Factory *FactorySession) DefaultPluginFactory() (common.Address, error) {
	return _Factory.Contract.DefaultPluginFactory(&_Factory.CallOpts)
}

// DefaultPluginFactory is a free data retrieval call binding the contract method 0xd0ad2792.
//
// Solidity: function defaultPluginFactory() view returns(address)
func (_Factory *FactoryCallerSession) DefaultPluginFactory() (common.Address, error) {
	return _Factory.Contract.DefaultPluginFactory(&_Factory.CallOpts)
}

// DefaultTickspacing is a free data retrieval call binding the contract method 0x29bc3446.
//
// Solidity: function defaultTickspacing() view returns(int24)
func (_Factory *FactoryCaller) DefaultTickspacing(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "defaultTickspacing")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DefaultTickspacing is a free data retrieval call binding the contract method 0x29bc3446.
//
// Solidity: function defaultTickspacing() view returns(int24)
func (_Factory *FactorySession) DefaultTickspacing() (*big.Int, error) {
	return _Factory.Contract.DefaultTickspacing(&_Factory.CallOpts)
}

// DefaultTickspacing is a free data retrieval call binding the contract method 0x29bc3446.
//
// Solidity: function defaultTickspacing() view returns(int24)
func (_Factory *FactoryCallerSession) DefaultTickspacing() (*big.Int, error) {
	return _Factory.Contract.DefaultTickspacing(&_Factory.CallOpts)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Factory *FactoryCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Factory *FactorySession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _Factory.Contract.GetRoleAdmin(&_Factory.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Factory *FactoryCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _Factory.Contract.GetRoleAdmin(&_Factory.CallOpts, role)
}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) view returns(address)
func (_Factory *FactoryCaller) GetRoleMember(opts *bind.CallOpts, role [32]byte, index *big.Int) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "getRoleMember", role, index)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) view returns(address)
func (_Factory *FactorySession) GetRoleMember(role [32]byte, index *big.Int) (common.Address, error) {
	return _Factory.Contract.GetRoleMember(&_Factory.CallOpts, role, index)
}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) view returns(address)
func (_Factory *FactoryCallerSession) GetRoleMember(role [32]byte, index *big.Int) (common.Address, error) {
	return _Factory.Contract.GetRoleMember(&_Factory.CallOpts, role, index)
}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) view returns(uint256)
func (_Factory *FactoryCaller) GetRoleMemberCount(opts *bind.CallOpts, role [32]byte) (*big.Int, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "getRoleMemberCount", role)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) view returns(uint256)
func (_Factory *FactorySession) GetRoleMemberCount(role [32]byte) (*big.Int, error) {
	return _Factory.Contract.GetRoleMemberCount(&_Factory.CallOpts, role)
}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) view returns(uint256)
func (_Factory *FactoryCallerSession) GetRoleMemberCount(role [32]byte) (*big.Int, error) {
	return _Factory.Contract.GetRoleMemberCount(&_Factory.CallOpts, role)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Factory *FactoryCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Factory *FactorySession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _Factory.Contract.HasRole(&_Factory.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Factory *FactoryCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _Factory.Contract.HasRole(&_Factory.CallOpts, role, account)
}

// HasRoleOrOwner is a free data retrieval call binding the contract method 0xe8ae2b69.
//
// Solidity: function hasRoleOrOwner(bytes32 role, address account) view returns(bool)
func (_Factory *FactoryCaller) HasRoleOrOwner(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "hasRoleOrOwner", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRoleOrOwner is a free data retrieval call binding the contract method 0xe8ae2b69.
//
// Solidity: function hasRoleOrOwner(bytes32 role, address account) view returns(bool)
func (_Factory *FactorySession) HasRoleOrOwner(role [32]byte, account common.Address) (bool, error) {
	return _Factory.Contract.HasRoleOrOwner(&_Factory.CallOpts, role, account)
}

// HasRoleOrOwner is a free data retrieval call binding the contract method 0xe8ae2b69.
//
// Solidity: function hasRoleOrOwner(bytes32 role, address account) view returns(bool)
func (_Factory *FactoryCallerSession) HasRoleOrOwner(role [32]byte, account common.Address) (bool, error) {
	return _Factory.Contract.HasRoleOrOwner(&_Factory.CallOpts, role, account)
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

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_Factory *FactoryCaller) PendingOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "pendingOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_Factory *FactorySession) PendingOwner() (common.Address, error) {
	return _Factory.Contract.PendingOwner(&_Factory.CallOpts)
}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_Factory *FactoryCallerSession) PendingOwner() (common.Address, error) {
	return _Factory.Contract.PendingOwner(&_Factory.CallOpts)
}

// PoolByPair is a free data retrieval call binding the contract method 0xd9a641e1.
//
// Solidity: function poolByPair(address , address ) view returns(address)
func (_Factory *FactoryCaller) PoolByPair(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "poolByPair", arg0, arg1)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PoolByPair is a free data retrieval call binding the contract method 0xd9a641e1.
//
// Solidity: function poolByPair(address , address ) view returns(address)
func (_Factory *FactorySession) PoolByPair(arg0 common.Address, arg1 common.Address) (common.Address, error) {
	return _Factory.Contract.PoolByPair(&_Factory.CallOpts, arg0, arg1)
}

// PoolByPair is a free data retrieval call binding the contract method 0xd9a641e1.
//
// Solidity: function poolByPair(address , address ) view returns(address)
func (_Factory *FactoryCallerSession) PoolByPair(arg0 common.Address, arg1 common.Address) (common.Address, error) {
	return _Factory.Contract.PoolByPair(&_Factory.CallOpts, arg0, arg1)
}

// PoolDeployer is a free data retrieval call binding the contract method 0x3119049a.
//
// Solidity: function poolDeployer() view returns(address)
func (_Factory *FactoryCaller) PoolDeployer(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "poolDeployer")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PoolDeployer is a free data retrieval call binding the contract method 0x3119049a.
//
// Solidity: function poolDeployer() view returns(address)
func (_Factory *FactorySession) PoolDeployer() (common.Address, error) {
	return _Factory.Contract.PoolDeployer(&_Factory.CallOpts)
}

// PoolDeployer is a free data retrieval call binding the contract method 0x3119049a.
//
// Solidity: function poolDeployer() view returns(address)
func (_Factory *FactoryCallerSession) PoolDeployer() (common.Address, error) {
	return _Factory.Contract.PoolDeployer(&_Factory.CallOpts)
}

// RenounceOwnershipStartTimestamp is a free data retrieval call binding the contract method 0x084bfff9.
//
// Solidity: function renounceOwnershipStartTimestamp() view returns(uint256)
func (_Factory *FactoryCaller) RenounceOwnershipStartTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "renounceOwnershipStartTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// RenounceOwnershipStartTimestamp is a free data retrieval call binding the contract method 0x084bfff9.
//
// Solidity: function renounceOwnershipStartTimestamp() view returns(uint256)
func (_Factory *FactorySession) RenounceOwnershipStartTimestamp() (*big.Int, error) {
	return _Factory.Contract.RenounceOwnershipStartTimestamp(&_Factory.CallOpts)
}

// RenounceOwnershipStartTimestamp is a free data retrieval call binding the contract method 0x084bfff9.
//
// Solidity: function renounceOwnershipStartTimestamp() view returns(uint256)
func (_Factory *FactoryCallerSession) RenounceOwnershipStartTimestamp() (*big.Int, error) {
	return _Factory.Contract.RenounceOwnershipStartTimestamp(&_Factory.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Factory *FactoryCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Factory *FactorySession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Factory.Contract.SupportsInterface(&_Factory.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Factory *FactoryCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Factory.Contract.SupportsInterface(&_Factory.CallOpts, interfaceId)
}

// VaultFactory is a free data retrieval call binding the contract method 0xd8a06f73.
//
// Solidity: function vaultFactory() view returns(address)
func (_Factory *FactoryCaller) VaultFactory(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _Factory.contract.Call(opts, &out, "vaultFactory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// VaultFactory is a free data retrieval call binding the contract method 0xd8a06f73.
//
// Solidity: function vaultFactory() view returns(address)
func (_Factory *FactorySession) VaultFactory() (common.Address, error) {
	return _Factory.Contract.VaultFactory(&_Factory.CallOpts)
}

// VaultFactory is a free data retrieval call binding the contract method 0xd8a06f73.
//
// Solidity: function vaultFactory() view returns(address)
func (_Factory *FactoryCallerSession) VaultFactory() (common.Address, error) {
	return _Factory.Contract.VaultFactory(&_Factory.CallOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_Factory *FactoryTransactor) AcceptOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "acceptOwnership")
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_Factory *FactorySession) AcceptOwnership() (*types.Transaction, error) {
	return _Factory.Contract.AcceptOwnership(&_Factory.TransactOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_Factory *FactoryTransactorSession) AcceptOwnership() (*types.Transaction, error) {
	return _Factory.Contract.AcceptOwnership(&_Factory.TransactOpts)
}

// CreateCustomPool is a paid mutator transaction binding the contract method 0xdbbf3db4.
//
// Solidity: function createCustomPool(address deployer, address creator, address tokenA, address tokenB, bytes data) returns(address customPool)
func (_Factory *FactoryTransactor) CreateCustomPool(opts *bind.TransactOpts, deployer common.Address, creator common.Address, tokenA common.Address, tokenB common.Address, data []byte) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "createCustomPool", deployer, creator, tokenA, tokenB, data)
}

// CreateCustomPool is a paid mutator transaction binding the contract method 0xdbbf3db4.
//
// Solidity: function createCustomPool(address deployer, address creator, address tokenA, address tokenB, bytes data) returns(address customPool)
func (_Factory *FactorySession) CreateCustomPool(deployer common.Address, creator common.Address, tokenA common.Address, tokenB common.Address, data []byte) (*types.Transaction, error) {
	return _Factory.Contract.CreateCustomPool(&_Factory.TransactOpts, deployer, creator, tokenA, tokenB, data)
}

// CreateCustomPool is a paid mutator transaction binding the contract method 0xdbbf3db4.
//
// Solidity: function createCustomPool(address deployer, address creator, address tokenA, address tokenB, bytes data) returns(address customPool)
func (_Factory *FactoryTransactorSession) CreateCustomPool(deployer common.Address, creator common.Address, tokenA common.Address, tokenB common.Address, data []byte) (*types.Transaction, error) {
	return _Factory.Contract.CreateCustomPool(&_Factory.TransactOpts, deployer, creator, tokenA, tokenB, data)
}

// CreatePool is a paid mutator transaction binding the contract method 0x321935c6.
//
// Solidity: function createPool(address tokenA, address tokenB, bytes data) returns(address pool)
func (_Factory *FactoryTransactor) CreatePool(opts *bind.TransactOpts, tokenA common.Address, tokenB common.Address, data []byte) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "createPool", tokenA, tokenB, data)
}

// CreatePool is a paid mutator transaction binding the contract method 0x321935c6.
//
// Solidity: function createPool(address tokenA, address tokenB, bytes data) returns(address pool)
func (_Factory *FactorySession) CreatePool(tokenA common.Address, tokenB common.Address, data []byte) (*types.Transaction, error) {
	return _Factory.Contract.CreatePool(&_Factory.TransactOpts, tokenA, tokenB, data)
}

// CreatePool is a paid mutator transaction binding the contract method 0x321935c6.
//
// Solidity: function createPool(address tokenA, address tokenB, bytes data) returns(address pool)
func (_Factory *FactoryTransactorSession) CreatePool(tokenA common.Address, tokenB common.Address, data []byte) (*types.Transaction, error) {
	return _Factory.Contract.CreatePool(&_Factory.TransactOpts, tokenA, tokenB, data)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Factory *FactoryTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Factory *FactorySession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Factory.Contract.GrantRole(&_Factory.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Factory *FactoryTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Factory.Contract.GrantRole(&_Factory.TransactOpts, role, account)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Factory *FactoryTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Factory *FactorySession) RenounceOwnership() (*types.Transaction, error) {
	return _Factory.Contract.RenounceOwnership(&_Factory.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Factory *FactoryTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Factory.Contract.RenounceOwnership(&_Factory.TransactOpts)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Factory *FactoryTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "renounceRole", role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Factory *FactorySession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Factory.Contract.RenounceRole(&_Factory.TransactOpts, role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Factory *FactoryTransactorSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Factory.Contract.RenounceRole(&_Factory.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Factory *FactoryTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Factory *FactorySession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Factory.Contract.RevokeRole(&_Factory.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Factory *FactoryTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Factory.Contract.RevokeRole(&_Factory.TransactOpts, role, account)
}

// SetDefaultCommunityFee is a paid mutator transaction binding the contract method 0x8d5a8711.
//
// Solidity: function setDefaultCommunityFee(uint16 newDefaultCommunityFee) returns()
func (_Factory *FactoryTransactor) SetDefaultCommunityFee(opts *bind.TransactOpts, newDefaultCommunityFee uint16) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setDefaultCommunityFee", newDefaultCommunityFee)
}

// SetDefaultCommunityFee is a paid mutator transaction binding the contract method 0x8d5a8711.
//
// Solidity: function setDefaultCommunityFee(uint16 newDefaultCommunityFee) returns()
func (_Factory *FactorySession) SetDefaultCommunityFee(newDefaultCommunityFee uint16) (*types.Transaction, error) {
	return _Factory.Contract.SetDefaultCommunityFee(&_Factory.TransactOpts, newDefaultCommunityFee)
}

// SetDefaultCommunityFee is a paid mutator transaction binding the contract method 0x8d5a8711.
//
// Solidity: function setDefaultCommunityFee(uint16 newDefaultCommunityFee) returns()
func (_Factory *FactoryTransactorSession) SetDefaultCommunityFee(newDefaultCommunityFee uint16) (*types.Transaction, error) {
	return _Factory.Contract.SetDefaultCommunityFee(&_Factory.TransactOpts, newDefaultCommunityFee)
}

// SetDefaultFee is a paid mutator transaction binding the contract method 0x77326584.
//
// Solidity: function setDefaultFee(uint16 newDefaultFee) returns()
func (_Factory *FactoryTransactor) SetDefaultFee(opts *bind.TransactOpts, newDefaultFee uint16) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setDefaultFee", newDefaultFee)
}

// SetDefaultFee is a paid mutator transaction binding the contract method 0x77326584.
//
// Solidity: function setDefaultFee(uint16 newDefaultFee) returns()
func (_Factory *FactorySession) SetDefaultFee(newDefaultFee uint16) (*types.Transaction, error) {
	return _Factory.Contract.SetDefaultFee(&_Factory.TransactOpts, newDefaultFee)
}

// SetDefaultFee is a paid mutator transaction binding the contract method 0x77326584.
//
// Solidity: function setDefaultFee(uint16 newDefaultFee) returns()
func (_Factory *FactoryTransactorSession) SetDefaultFee(newDefaultFee uint16) (*types.Transaction, error) {
	return _Factory.Contract.SetDefaultFee(&_Factory.TransactOpts, newDefaultFee)
}

// SetDefaultPluginFactory is a paid mutator transaction binding the contract method 0x2939dd97.
//
// Solidity: function setDefaultPluginFactory(address newDefaultPluginFactory) returns()
func (_Factory *FactoryTransactor) SetDefaultPluginFactory(opts *bind.TransactOpts, newDefaultPluginFactory common.Address) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setDefaultPluginFactory", newDefaultPluginFactory)
}

// SetDefaultPluginFactory is a paid mutator transaction binding the contract method 0x2939dd97.
//
// Solidity: function setDefaultPluginFactory(address newDefaultPluginFactory) returns()
func (_Factory *FactorySession) SetDefaultPluginFactory(newDefaultPluginFactory common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetDefaultPluginFactory(&_Factory.TransactOpts, newDefaultPluginFactory)
}

// SetDefaultPluginFactory is a paid mutator transaction binding the contract method 0x2939dd97.
//
// Solidity: function setDefaultPluginFactory(address newDefaultPluginFactory) returns()
func (_Factory *FactoryTransactorSession) SetDefaultPluginFactory(newDefaultPluginFactory common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetDefaultPluginFactory(&_Factory.TransactOpts, newDefaultPluginFactory)
}

// SetDefaultTickspacing is a paid mutator transaction binding the contract method 0xf09489ac.
//
// Solidity: function setDefaultTickspacing(int24 newDefaultTickspacing) returns()
func (_Factory *FactoryTransactor) SetDefaultTickspacing(opts *bind.TransactOpts, newDefaultTickspacing *big.Int) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setDefaultTickspacing", newDefaultTickspacing)
}

// SetDefaultTickspacing is a paid mutator transaction binding the contract method 0xf09489ac.
//
// Solidity: function setDefaultTickspacing(int24 newDefaultTickspacing) returns()
func (_Factory *FactorySession) SetDefaultTickspacing(newDefaultTickspacing *big.Int) (*types.Transaction, error) {
	return _Factory.Contract.SetDefaultTickspacing(&_Factory.TransactOpts, newDefaultTickspacing)
}

// SetDefaultTickspacing is a paid mutator transaction binding the contract method 0xf09489ac.
//
// Solidity: function setDefaultTickspacing(int24 newDefaultTickspacing) returns()
func (_Factory *FactoryTransactorSession) SetDefaultTickspacing(newDefaultTickspacing *big.Int) (*types.Transaction, error) {
	return _Factory.Contract.SetDefaultTickspacing(&_Factory.TransactOpts, newDefaultTickspacing)
}

// SetVaultFactory is a paid mutator transaction binding the contract method 0x3ea7fbdb.
//
// Solidity: function setVaultFactory(address newVaultFactory) returns()
func (_Factory *FactoryTransactor) SetVaultFactory(opts *bind.TransactOpts, newVaultFactory common.Address) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "setVaultFactory", newVaultFactory)
}

// SetVaultFactory is a paid mutator transaction binding the contract method 0x3ea7fbdb.
//
// Solidity: function setVaultFactory(address newVaultFactory) returns()
func (_Factory *FactorySession) SetVaultFactory(newVaultFactory common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetVaultFactory(&_Factory.TransactOpts, newVaultFactory)
}

// SetVaultFactory is a paid mutator transaction binding the contract method 0x3ea7fbdb.
//
// Solidity: function setVaultFactory(address newVaultFactory) returns()
func (_Factory *FactoryTransactorSession) SetVaultFactory(newVaultFactory common.Address) (*types.Transaction, error) {
	return _Factory.Contract.SetVaultFactory(&_Factory.TransactOpts, newVaultFactory)
}

// StartRenounceOwnership is a paid mutator transaction binding the contract method 0x469388c4.
//
// Solidity: function startRenounceOwnership() returns()
func (_Factory *FactoryTransactor) StartRenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "startRenounceOwnership")
}

// StartRenounceOwnership is a paid mutator transaction binding the contract method 0x469388c4.
//
// Solidity: function startRenounceOwnership() returns()
func (_Factory *FactorySession) StartRenounceOwnership() (*types.Transaction, error) {
	return _Factory.Contract.StartRenounceOwnership(&_Factory.TransactOpts)
}

// StartRenounceOwnership is a paid mutator transaction binding the contract method 0x469388c4.
//
// Solidity: function startRenounceOwnership() returns()
func (_Factory *FactoryTransactorSession) StartRenounceOwnership() (*types.Transaction, error) {
	return _Factory.Contract.StartRenounceOwnership(&_Factory.TransactOpts)
}

// StopRenounceOwnership is a paid mutator transaction binding the contract method 0x238a1d74.
//
// Solidity: function stopRenounceOwnership() returns()
func (_Factory *FactoryTransactor) StopRenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "stopRenounceOwnership")
}

// StopRenounceOwnership is a paid mutator transaction binding the contract method 0x238a1d74.
//
// Solidity: function stopRenounceOwnership() returns()
func (_Factory *FactorySession) StopRenounceOwnership() (*types.Transaction, error) {
	return _Factory.Contract.StopRenounceOwnership(&_Factory.TransactOpts)
}

// StopRenounceOwnership is a paid mutator transaction binding the contract method 0x238a1d74.
//
// Solidity: function stopRenounceOwnership() returns()
func (_Factory *FactoryTransactorSession) StopRenounceOwnership() (*types.Transaction, error) {
	return _Factory.Contract.StopRenounceOwnership(&_Factory.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Factory *FactoryTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Factory.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Factory *FactorySession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Factory.Contract.TransferOwnership(&_Factory.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Factory *FactoryTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Factory.Contract.TransferOwnership(&_Factory.TransactOpts, newOwner)
}

// FactoryCustomPoolIterator is returned from FilterCustomPool and is used to iterate over the raw logs and unpacked data for CustomPool events raised by the Factory contract.
type FactoryCustomPoolIterator struct {
	Event *FactoryCustomPool // Event containing the contract specifics and raw log

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
func (it *FactoryCustomPoolIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryCustomPool)
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
		it.Event = new(FactoryCustomPool)
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
func (it *FactoryCustomPoolIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryCustomPoolIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryCustomPool represents a CustomPool event raised by the Factory contract.
type FactoryCustomPool struct {
	Deployer common.Address
	Token0   common.Address
	Token1   common.Address
	Pool     common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterCustomPool is a free log retrieval operation binding the contract event 0x8a5f030f5fc13b04a1e4ef7c47177e3d76b0e80e1d9be9843db37caa5b7b9b8f.
//
// Solidity: event CustomPool(address indexed deployer, address indexed token0, address indexed token1, address pool)
func (_Factory *FactoryFilterer) FilterCustomPool(opts *bind.FilterOpts, deployer []common.Address, token0 []common.Address, token1 []common.Address) (*FactoryCustomPoolIterator, error) {

	var deployerRule []any
	for _, deployerItem := range deployer {
		deployerRule = append(deployerRule, deployerItem)
	}
	var token0Rule []any
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []any
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "CustomPool", deployerRule, token0Rule, token1Rule)
	if err != nil {
		return nil, err
	}
	return &FactoryCustomPoolIterator{contract: _Factory.contract, event: "CustomPool", logs: logs, sub: sub}, nil
}

// WatchCustomPool is a free log subscription operation binding the contract event 0x8a5f030f5fc13b04a1e4ef7c47177e3d76b0e80e1d9be9843db37caa5b7b9b8f.
//
// Solidity: event CustomPool(address indexed deployer, address indexed token0, address indexed token1, address pool)
func (_Factory *FactoryFilterer) WatchCustomPool(opts *bind.WatchOpts, sink chan<- *FactoryCustomPool, deployer []common.Address, token0 []common.Address, token1 []common.Address) (event.Subscription, error) {

	var deployerRule []any
	for _, deployerItem := range deployer {
		deployerRule = append(deployerRule, deployerItem)
	}
	var token0Rule []any
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []any
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "CustomPool", deployerRule, token0Rule, token1Rule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryCustomPool)
				if err := _Factory.contract.UnpackLog(event, "CustomPool", log); err != nil {
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

// ParseCustomPool is a log parse operation binding the contract event 0x8a5f030f5fc13b04a1e4ef7c47177e3d76b0e80e1d9be9843db37caa5b7b9b8f.
//
// Solidity: event CustomPool(address indexed deployer, address indexed token0, address indexed token1, address pool)
func (_Factory *FactoryFilterer) ParseCustomPool(log types.Log) (*FactoryCustomPool, error) {
	event := new(FactoryCustomPool)
	if err := _Factory.contract.UnpackLog(event, "CustomPool", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryDefaultCommunityFeeIterator is returned from FilterDefaultCommunityFee and is used to iterate over the raw logs and unpacked data for DefaultCommunityFee events raised by the Factory contract.
type FactoryDefaultCommunityFeeIterator struct {
	Event *FactoryDefaultCommunityFee // Event containing the contract specifics and raw log

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
func (it *FactoryDefaultCommunityFeeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryDefaultCommunityFee)
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
		it.Event = new(FactoryDefaultCommunityFee)
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
func (it *FactoryDefaultCommunityFeeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryDefaultCommunityFeeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryDefaultCommunityFee represents a DefaultCommunityFee event raised by the Factory contract.
type FactoryDefaultCommunityFee struct {
	NewDefaultCommunityFee uint16
	Raw                    types.Log // Blockchain specific contextual infos
}

// FilterDefaultCommunityFee is a free log retrieval operation binding the contract event 0x6b5c342391f543846fce47a925e7eba910f7bec232b08633308ca93fdd0fdf0d.
//
// Solidity: event DefaultCommunityFee(uint16 newDefaultCommunityFee)
func (_Factory *FactoryFilterer) FilterDefaultCommunityFee(opts *bind.FilterOpts) (*FactoryDefaultCommunityFeeIterator, error) {

	logs, sub, err := _Factory.contract.FilterLogs(opts, "DefaultCommunityFee")
	if err != nil {
		return nil, err
	}
	return &FactoryDefaultCommunityFeeIterator{contract: _Factory.contract, event: "DefaultCommunityFee", logs: logs, sub: sub}, nil
}

// WatchDefaultCommunityFee is a free log subscription operation binding the contract event 0x6b5c342391f543846fce47a925e7eba910f7bec232b08633308ca93fdd0fdf0d.
//
// Solidity: event DefaultCommunityFee(uint16 newDefaultCommunityFee)
func (_Factory *FactoryFilterer) WatchDefaultCommunityFee(opts *bind.WatchOpts, sink chan<- *FactoryDefaultCommunityFee) (event.Subscription, error) {

	logs, sub, err := _Factory.contract.WatchLogs(opts, "DefaultCommunityFee")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryDefaultCommunityFee)
				if err := _Factory.contract.UnpackLog(event, "DefaultCommunityFee", log); err != nil {
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

// ParseDefaultCommunityFee is a log parse operation binding the contract event 0x6b5c342391f543846fce47a925e7eba910f7bec232b08633308ca93fdd0fdf0d.
//
// Solidity: event DefaultCommunityFee(uint16 newDefaultCommunityFee)
func (_Factory *FactoryFilterer) ParseDefaultCommunityFee(log types.Log) (*FactoryDefaultCommunityFee, error) {
	event := new(FactoryDefaultCommunityFee)
	if err := _Factory.contract.UnpackLog(event, "DefaultCommunityFee", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryDefaultFeeIterator is returned from FilterDefaultFee and is used to iterate over the raw logs and unpacked data for DefaultFee events raised by the Factory contract.
type FactoryDefaultFeeIterator struct {
	Event *FactoryDefaultFee // Event containing the contract specifics and raw log

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
func (it *FactoryDefaultFeeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryDefaultFee)
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
		it.Event = new(FactoryDefaultFee)
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
func (it *FactoryDefaultFeeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryDefaultFeeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryDefaultFee represents a DefaultFee event raised by the Factory contract.
type FactoryDefaultFee struct {
	NewDefaultFee uint16
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterDefaultFee is a free log retrieval operation binding the contract event 0xddc0c6f0b581e0d51bfe90ff138e4a548f94515c4dbcb12f5e98fdf0f7503983.
//
// Solidity: event DefaultFee(uint16 newDefaultFee)
func (_Factory *FactoryFilterer) FilterDefaultFee(opts *bind.FilterOpts) (*FactoryDefaultFeeIterator, error) {

	logs, sub, err := _Factory.contract.FilterLogs(opts, "DefaultFee")
	if err != nil {
		return nil, err
	}
	return &FactoryDefaultFeeIterator{contract: _Factory.contract, event: "DefaultFee", logs: logs, sub: sub}, nil
}

// WatchDefaultFee is a free log subscription operation binding the contract event 0xddc0c6f0b581e0d51bfe90ff138e4a548f94515c4dbcb12f5e98fdf0f7503983.
//
// Solidity: event DefaultFee(uint16 newDefaultFee)
func (_Factory *FactoryFilterer) WatchDefaultFee(opts *bind.WatchOpts, sink chan<- *FactoryDefaultFee) (event.Subscription, error) {

	logs, sub, err := _Factory.contract.WatchLogs(opts, "DefaultFee")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryDefaultFee)
				if err := _Factory.contract.UnpackLog(event, "DefaultFee", log); err != nil {
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

// ParseDefaultFee is a log parse operation binding the contract event 0xddc0c6f0b581e0d51bfe90ff138e4a548f94515c4dbcb12f5e98fdf0f7503983.
//
// Solidity: event DefaultFee(uint16 newDefaultFee)
func (_Factory *FactoryFilterer) ParseDefaultFee(log types.Log) (*FactoryDefaultFee, error) {
	event := new(FactoryDefaultFee)
	if err := _Factory.contract.UnpackLog(event, "DefaultFee", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryDefaultPluginFactoryIterator is returned from FilterDefaultPluginFactory and is used to iterate over the raw logs and unpacked data for DefaultPluginFactory events raised by the Factory contract.
type FactoryDefaultPluginFactoryIterator struct {
	Event *FactoryDefaultPluginFactory // Event containing the contract specifics and raw log

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
func (it *FactoryDefaultPluginFactoryIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryDefaultPluginFactory)
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
		it.Event = new(FactoryDefaultPluginFactory)
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
func (it *FactoryDefaultPluginFactoryIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryDefaultPluginFactoryIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryDefaultPluginFactory represents a DefaultPluginFactory event raised by the Factory contract.
type FactoryDefaultPluginFactory struct {
	DefaultPluginFactoryAddress common.Address
	Raw                         types.Log // Blockchain specific contextual infos
}

// FilterDefaultPluginFactory is a free log retrieval operation binding the contract event 0x5e38e259ec1f8a38b98fc65a27e266bb9cc87c76eb8c96c957450d1cff4591ef.
//
// Solidity: event DefaultPluginFactory(address defaultPluginFactoryAddress)
func (_Factory *FactoryFilterer) FilterDefaultPluginFactory(opts *bind.FilterOpts) (*FactoryDefaultPluginFactoryIterator, error) {

	logs, sub, err := _Factory.contract.FilterLogs(opts, "DefaultPluginFactory")
	if err != nil {
		return nil, err
	}
	return &FactoryDefaultPluginFactoryIterator{contract: _Factory.contract, event: "DefaultPluginFactory", logs: logs, sub: sub}, nil
}

// WatchDefaultPluginFactory is a free log subscription operation binding the contract event 0x5e38e259ec1f8a38b98fc65a27e266bb9cc87c76eb8c96c957450d1cff4591ef.
//
// Solidity: event DefaultPluginFactory(address defaultPluginFactoryAddress)
func (_Factory *FactoryFilterer) WatchDefaultPluginFactory(opts *bind.WatchOpts, sink chan<- *FactoryDefaultPluginFactory) (event.Subscription, error) {

	logs, sub, err := _Factory.contract.WatchLogs(opts, "DefaultPluginFactory")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryDefaultPluginFactory)
				if err := _Factory.contract.UnpackLog(event, "DefaultPluginFactory", log); err != nil {
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

// ParseDefaultPluginFactory is a log parse operation binding the contract event 0x5e38e259ec1f8a38b98fc65a27e266bb9cc87c76eb8c96c957450d1cff4591ef.
//
// Solidity: event DefaultPluginFactory(address defaultPluginFactoryAddress)
func (_Factory *FactoryFilterer) ParseDefaultPluginFactory(log types.Log) (*FactoryDefaultPluginFactory, error) {
	event := new(FactoryDefaultPluginFactory)
	if err := _Factory.contract.UnpackLog(event, "DefaultPluginFactory", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryDefaultTickspacingIterator is returned from FilterDefaultTickspacing and is used to iterate over the raw logs and unpacked data for DefaultTickspacing events raised by the Factory contract.
type FactoryDefaultTickspacingIterator struct {
	Event *FactoryDefaultTickspacing // Event containing the contract specifics and raw log

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
func (it *FactoryDefaultTickspacingIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryDefaultTickspacing)
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
		it.Event = new(FactoryDefaultTickspacing)
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
func (it *FactoryDefaultTickspacingIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryDefaultTickspacingIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryDefaultTickspacing represents a DefaultTickspacing event raised by the Factory contract.
type FactoryDefaultTickspacing struct {
	NewDefaultTickspacing *big.Int
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterDefaultTickspacing is a free log retrieval operation binding the contract event 0x7d7979096f943139ebee59f01c077a0f0766d06c40c86d596f23ed2561547cce.
//
// Solidity: event DefaultTickspacing(int24 newDefaultTickspacing)
func (_Factory *FactoryFilterer) FilterDefaultTickspacing(opts *bind.FilterOpts) (*FactoryDefaultTickspacingIterator, error) {

	logs, sub, err := _Factory.contract.FilterLogs(opts, "DefaultTickspacing")
	if err != nil {
		return nil, err
	}
	return &FactoryDefaultTickspacingIterator{contract: _Factory.contract, event: "DefaultTickspacing", logs: logs, sub: sub}, nil
}

// WatchDefaultTickspacing is a free log subscription operation binding the contract event 0x7d7979096f943139ebee59f01c077a0f0766d06c40c86d596f23ed2561547cce.
//
// Solidity: event DefaultTickspacing(int24 newDefaultTickspacing)
func (_Factory *FactoryFilterer) WatchDefaultTickspacing(opts *bind.WatchOpts, sink chan<- *FactoryDefaultTickspacing) (event.Subscription, error) {

	logs, sub, err := _Factory.contract.WatchLogs(opts, "DefaultTickspacing")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryDefaultTickspacing)
				if err := _Factory.contract.UnpackLog(event, "DefaultTickspacing", log); err != nil {
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

// ParseDefaultTickspacing is a log parse operation binding the contract event 0x7d7979096f943139ebee59f01c077a0f0766d06c40c86d596f23ed2561547cce.
//
// Solidity: event DefaultTickspacing(int24 newDefaultTickspacing)
func (_Factory *FactoryFilterer) ParseDefaultTickspacing(log types.Log) (*FactoryDefaultTickspacing, error) {
	event := new(FactoryDefaultTickspacing)
	if err := _Factory.contract.UnpackLog(event, "DefaultTickspacing", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryOwnershipTransferStartedIterator is returned from FilterOwnershipTransferStarted and is used to iterate over the raw logs and unpacked data for OwnershipTransferStarted events raised by the Factory contract.
type FactoryOwnershipTransferStartedIterator struct {
	Event *FactoryOwnershipTransferStarted // Event containing the contract specifics and raw log

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
func (it *FactoryOwnershipTransferStartedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryOwnershipTransferStarted)
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
		it.Event = new(FactoryOwnershipTransferStarted)
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
func (it *FactoryOwnershipTransferStartedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryOwnershipTransferStartedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryOwnershipTransferStarted represents a OwnershipTransferStarted event raised by the Factory contract.
type FactoryOwnershipTransferStarted struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferStarted is a free log retrieval operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_Factory *FactoryFilterer) FilterOwnershipTransferStarted(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*FactoryOwnershipTransferStartedIterator, error) {

	var previousOwnerRule []any
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []any
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "OwnershipTransferStarted", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &FactoryOwnershipTransferStartedIterator{contract: _Factory.contract, event: "OwnershipTransferStarted", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferStarted is a free log subscription operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_Factory *FactoryFilterer) WatchOwnershipTransferStarted(opts *bind.WatchOpts, sink chan<- *FactoryOwnershipTransferStarted, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []any
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []any
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "OwnershipTransferStarted", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryOwnershipTransferStarted)
				if err := _Factory.contract.UnpackLog(event, "OwnershipTransferStarted", log); err != nil {
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

// ParseOwnershipTransferStarted is a log parse operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_Factory *FactoryFilterer) ParseOwnershipTransferStarted(log types.Log) (*FactoryOwnershipTransferStarted, error) {
	event := new(FactoryOwnershipTransferStarted)
	if err := _Factory.contract.UnpackLog(event, "OwnershipTransferStarted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Factory contract.
type FactoryOwnershipTransferredIterator struct {
	Event *FactoryOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *FactoryOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryOwnershipTransferred)
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
		it.Event = new(FactoryOwnershipTransferred)
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
func (it *FactoryOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryOwnershipTransferred represents a OwnershipTransferred event raised by the Factory contract.
type FactoryOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Factory *FactoryFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*FactoryOwnershipTransferredIterator, error) {

	var previousOwnerRule []any
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []any
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &FactoryOwnershipTransferredIterator{contract: _Factory.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Factory *FactoryFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *FactoryOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []any
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []any
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryOwnershipTransferred)
				if err := _Factory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_Factory *FactoryFilterer) ParseOwnershipTransferred(log types.Log) (*FactoryOwnershipTransferred, error) {
	event := new(FactoryOwnershipTransferred)
	if err := _Factory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryPoolIterator is returned from FilterPool and is used to iterate over the raw logs and unpacked data for Pool events raised by the Factory contract.
type FactoryPoolIterator struct {
	Event *FactoryPool // Event containing the contract specifics and raw log

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
func (it *FactoryPoolIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryPool)
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
		it.Event = new(FactoryPool)
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
func (it *FactoryPoolIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryPoolIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryPool represents a Pool event raised by the Factory contract.
type FactoryPool struct {
	Token0 common.Address
	Token1 common.Address
	Pool   common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterPool is a free log retrieval operation binding the contract event 0x91ccaa7a278130b65168c3a0c8d3bcae84cf5e43704342bd3ec0b59e59c036db.
//
// Solidity: event Pool(address indexed token0, address indexed token1, address pool)
func (_Factory *FactoryFilterer) FilterPool(opts *bind.FilterOpts, token0 []common.Address, token1 []common.Address) (*FactoryPoolIterator, error) {

	var token0Rule []any
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []any
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "Pool", token0Rule, token1Rule)
	if err != nil {
		return nil, err
	}
	return &FactoryPoolIterator{contract: _Factory.contract, event: "Pool", logs: logs, sub: sub}, nil
}

// WatchPool is a free log subscription operation binding the contract event 0x91ccaa7a278130b65168c3a0c8d3bcae84cf5e43704342bd3ec0b59e59c036db.
//
// Solidity: event Pool(address indexed token0, address indexed token1, address pool)
func (_Factory *FactoryFilterer) WatchPool(opts *bind.WatchOpts, sink chan<- *FactoryPool, token0 []common.Address, token1 []common.Address) (event.Subscription, error) {

	var token0Rule []any
	for _, token0Item := range token0 {
		token0Rule = append(token0Rule, token0Item)
	}
	var token1Rule []any
	for _, token1Item := range token1 {
		token1Rule = append(token1Rule, token1Item)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "Pool", token0Rule, token1Rule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryPool)
				if err := _Factory.contract.UnpackLog(event, "Pool", log); err != nil {
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

// ParsePool is a log parse operation binding the contract event 0x91ccaa7a278130b65168c3a0c8d3bcae84cf5e43704342bd3ec0b59e59c036db.
//
// Solidity: event Pool(address indexed token0, address indexed token1, address pool)
func (_Factory *FactoryFilterer) ParsePool(log types.Log) (*FactoryPool, error) {
	event := new(FactoryPool)
	if err := _Factory.contract.UnpackLog(event, "Pool", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryRenounceOwnershipFinishIterator is returned from FilterRenounceOwnershipFinish and is used to iterate over the raw logs and unpacked data for RenounceOwnershipFinish events raised by the Factory contract.
type FactoryRenounceOwnershipFinishIterator struct {
	Event *FactoryRenounceOwnershipFinish // Event containing the contract specifics and raw log

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
func (it *FactoryRenounceOwnershipFinishIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryRenounceOwnershipFinish)
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
		it.Event = new(FactoryRenounceOwnershipFinish)
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
func (it *FactoryRenounceOwnershipFinishIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryRenounceOwnershipFinishIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryRenounceOwnershipFinish represents a RenounceOwnershipFinish event raised by the Factory contract.
type FactoryRenounceOwnershipFinish struct {
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterRenounceOwnershipFinish is a free log retrieval operation binding the contract event 0xa24203c457ce43a097fa0c491fc9cf5e0a893af87a5e0a9785f29491deb11e23.
//
// Solidity: event RenounceOwnershipFinish(uint256 timestamp)
func (_Factory *FactoryFilterer) FilterRenounceOwnershipFinish(opts *bind.FilterOpts) (*FactoryRenounceOwnershipFinishIterator, error) {

	logs, sub, err := _Factory.contract.FilterLogs(opts, "RenounceOwnershipFinish")
	if err != nil {
		return nil, err
	}
	return &FactoryRenounceOwnershipFinishIterator{contract: _Factory.contract, event: "RenounceOwnershipFinish", logs: logs, sub: sub}, nil
}

// WatchRenounceOwnershipFinish is a free log subscription operation binding the contract event 0xa24203c457ce43a097fa0c491fc9cf5e0a893af87a5e0a9785f29491deb11e23.
//
// Solidity: event RenounceOwnershipFinish(uint256 timestamp)
func (_Factory *FactoryFilterer) WatchRenounceOwnershipFinish(opts *bind.WatchOpts, sink chan<- *FactoryRenounceOwnershipFinish) (event.Subscription, error) {

	logs, sub, err := _Factory.contract.WatchLogs(opts, "RenounceOwnershipFinish")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryRenounceOwnershipFinish)
				if err := _Factory.contract.UnpackLog(event, "RenounceOwnershipFinish", log); err != nil {
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

// ParseRenounceOwnershipFinish is a log parse operation binding the contract event 0xa24203c457ce43a097fa0c491fc9cf5e0a893af87a5e0a9785f29491deb11e23.
//
// Solidity: event RenounceOwnershipFinish(uint256 timestamp)
func (_Factory *FactoryFilterer) ParseRenounceOwnershipFinish(log types.Log) (*FactoryRenounceOwnershipFinish, error) {
	event := new(FactoryRenounceOwnershipFinish)
	if err := _Factory.contract.UnpackLog(event, "RenounceOwnershipFinish", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryRenounceOwnershipStartIterator is returned from FilterRenounceOwnershipStart and is used to iterate over the raw logs and unpacked data for RenounceOwnershipStart events raised by the Factory contract.
type FactoryRenounceOwnershipStartIterator struct {
	Event *FactoryRenounceOwnershipStart // Event containing the contract specifics and raw log

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
func (it *FactoryRenounceOwnershipStartIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryRenounceOwnershipStart)
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
		it.Event = new(FactoryRenounceOwnershipStart)
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
func (it *FactoryRenounceOwnershipStartIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryRenounceOwnershipStartIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryRenounceOwnershipStart represents a RenounceOwnershipStart event raised by the Factory contract.
type FactoryRenounceOwnershipStart struct {
	Timestamp       *big.Int
	FinishTimestamp *big.Int
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterRenounceOwnershipStart is a free log retrieval operation binding the contract event 0xcd60f5d54996130c21c3f063279b39230bcbafc12f763a1ac1dfaec2e9b61d29.
//
// Solidity: event RenounceOwnershipStart(uint256 timestamp, uint256 finishTimestamp)
func (_Factory *FactoryFilterer) FilterRenounceOwnershipStart(opts *bind.FilterOpts) (*FactoryRenounceOwnershipStartIterator, error) {

	logs, sub, err := _Factory.contract.FilterLogs(opts, "RenounceOwnershipStart")
	if err != nil {
		return nil, err
	}
	return &FactoryRenounceOwnershipStartIterator{contract: _Factory.contract, event: "RenounceOwnershipStart", logs: logs, sub: sub}, nil
}

// WatchRenounceOwnershipStart is a free log subscription operation binding the contract event 0xcd60f5d54996130c21c3f063279b39230bcbafc12f763a1ac1dfaec2e9b61d29.
//
// Solidity: event RenounceOwnershipStart(uint256 timestamp, uint256 finishTimestamp)
func (_Factory *FactoryFilterer) WatchRenounceOwnershipStart(opts *bind.WatchOpts, sink chan<- *FactoryRenounceOwnershipStart) (event.Subscription, error) {

	logs, sub, err := _Factory.contract.WatchLogs(opts, "RenounceOwnershipStart")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryRenounceOwnershipStart)
				if err := _Factory.contract.UnpackLog(event, "RenounceOwnershipStart", log); err != nil {
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

// ParseRenounceOwnershipStart is a log parse operation binding the contract event 0xcd60f5d54996130c21c3f063279b39230bcbafc12f763a1ac1dfaec2e9b61d29.
//
// Solidity: event RenounceOwnershipStart(uint256 timestamp, uint256 finishTimestamp)
func (_Factory *FactoryFilterer) ParseRenounceOwnershipStart(log types.Log) (*FactoryRenounceOwnershipStart, error) {
	event := new(FactoryRenounceOwnershipStart)
	if err := _Factory.contract.UnpackLog(event, "RenounceOwnershipStart", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryRenounceOwnershipStopIterator is returned from FilterRenounceOwnershipStop and is used to iterate over the raw logs and unpacked data for RenounceOwnershipStop events raised by the Factory contract.
type FactoryRenounceOwnershipStopIterator struct {
	Event *FactoryRenounceOwnershipStop // Event containing the contract specifics and raw log

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
func (it *FactoryRenounceOwnershipStopIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryRenounceOwnershipStop)
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
		it.Event = new(FactoryRenounceOwnershipStop)
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
func (it *FactoryRenounceOwnershipStopIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryRenounceOwnershipStopIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryRenounceOwnershipStop represents a RenounceOwnershipStop event raised by the Factory contract.
type FactoryRenounceOwnershipStop struct {
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterRenounceOwnershipStop is a free log retrieval operation binding the contract event 0xa2492902a0a1d28dc73e6ab22e473239ef077bb7bc8174dc7dab9fc0818e7135.
//
// Solidity: event RenounceOwnershipStop(uint256 timestamp)
func (_Factory *FactoryFilterer) FilterRenounceOwnershipStop(opts *bind.FilterOpts) (*FactoryRenounceOwnershipStopIterator, error) {

	logs, sub, err := _Factory.contract.FilterLogs(opts, "RenounceOwnershipStop")
	if err != nil {
		return nil, err
	}
	return &FactoryRenounceOwnershipStopIterator{contract: _Factory.contract, event: "RenounceOwnershipStop", logs: logs, sub: sub}, nil
}

// WatchRenounceOwnershipStop is a free log subscription operation binding the contract event 0xa2492902a0a1d28dc73e6ab22e473239ef077bb7bc8174dc7dab9fc0818e7135.
//
// Solidity: event RenounceOwnershipStop(uint256 timestamp)
func (_Factory *FactoryFilterer) WatchRenounceOwnershipStop(opts *bind.WatchOpts, sink chan<- *FactoryRenounceOwnershipStop) (event.Subscription, error) {

	logs, sub, err := _Factory.contract.WatchLogs(opts, "RenounceOwnershipStop")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryRenounceOwnershipStop)
				if err := _Factory.contract.UnpackLog(event, "RenounceOwnershipStop", log); err != nil {
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

// ParseRenounceOwnershipStop is a log parse operation binding the contract event 0xa2492902a0a1d28dc73e6ab22e473239ef077bb7bc8174dc7dab9fc0818e7135.
//
// Solidity: event RenounceOwnershipStop(uint256 timestamp)
func (_Factory *FactoryFilterer) ParseRenounceOwnershipStop(log types.Log) (*FactoryRenounceOwnershipStop, error) {
	event := new(FactoryRenounceOwnershipStop)
	if err := _Factory.contract.UnpackLog(event, "RenounceOwnershipStop", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the Factory contract.
type FactoryRoleAdminChangedIterator struct {
	Event *FactoryRoleAdminChanged // Event containing the contract specifics and raw log

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
func (it *FactoryRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryRoleAdminChanged)
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
		it.Event = new(FactoryRoleAdminChanged)
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
func (it *FactoryRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryRoleAdminChanged represents a RoleAdminChanged event raised by the Factory contract.
type FactoryRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_Factory *FactoryFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*FactoryRoleAdminChangedIterator, error) {

	var roleRule []any
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []any
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []any
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &FactoryRoleAdminChangedIterator{contract: _Factory.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_Factory *FactoryFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *FactoryRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

	var roleRule []any
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []any
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []any
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryRoleAdminChanged)
				if err := _Factory.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
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

// ParseRoleAdminChanged is a log parse operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_Factory *FactoryFilterer) ParseRoleAdminChanged(log types.Log) (*FactoryRoleAdminChanged, error) {
	event := new(FactoryRoleAdminChanged)
	if err := _Factory.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the Factory contract.
type FactoryRoleGrantedIterator struct {
	Event *FactoryRoleGranted // Event containing the contract specifics and raw log

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
func (it *FactoryRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryRoleGranted)
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
		it.Event = new(FactoryRoleGranted)
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
func (it *FactoryRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryRoleGranted represents a RoleGranted event raised by the Factory contract.
type FactoryRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_Factory *FactoryFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*FactoryRoleGrantedIterator, error) {

	var roleRule []any
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []any
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &FactoryRoleGrantedIterator{contract: _Factory.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_Factory *FactoryFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *FactoryRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []any
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []any
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryRoleGranted)
				if err := _Factory.contract.UnpackLog(event, "RoleGranted", log); err != nil {
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

// ParseRoleGranted is a log parse operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_Factory *FactoryFilterer) ParseRoleGranted(log types.Log) (*FactoryRoleGranted, error) {
	event := new(FactoryRoleGranted)
	if err := _Factory.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the Factory contract.
type FactoryRoleRevokedIterator struct {
	Event *FactoryRoleRevoked // Event containing the contract specifics and raw log

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
func (it *FactoryRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryRoleRevoked)
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
		it.Event = new(FactoryRoleRevoked)
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
func (it *FactoryRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryRoleRevoked represents a RoleRevoked event raised by the Factory contract.
type FactoryRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_Factory *FactoryFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*FactoryRoleRevokedIterator, error) {

	var roleRule []any
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []any
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Factory.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &FactoryRoleRevokedIterator{contract: _Factory.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_Factory *FactoryFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *FactoryRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []any
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []any
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Factory.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryRoleRevoked)
				if err := _Factory.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
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

// ParseRoleRevoked is a log parse operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_Factory *FactoryFilterer) ParseRoleRevoked(log types.Log) (*FactoryRoleRevoked, error) {
	event := new(FactoryRoleRevoked)
	if err := _Factory.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryVaultFactoryIterator is returned from FilterVaultFactory and is used to iterate over the raw logs and unpacked data for VaultFactory events raised by the Factory contract.
type FactoryVaultFactoryIterator struct {
	Event *FactoryVaultFactory // Event containing the contract specifics and raw log

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
func (it *FactoryVaultFactoryIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryVaultFactory)
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
		it.Event = new(FactoryVaultFactory)
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
func (it *FactoryVaultFactoryIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryVaultFactoryIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryVaultFactory represents a VaultFactory event raised by the Factory contract.
type FactoryVaultFactory struct {
	NewVaultFactory common.Address
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterVaultFactory is a free log retrieval operation binding the contract event 0xa006ea05a14783821b0248e75d2342cd1681b07509e10a0f08487b080c29dea8.
//
// Solidity: event VaultFactory(address newVaultFactory)
func (_Factory *FactoryFilterer) FilterVaultFactory(opts *bind.FilterOpts) (*FactoryVaultFactoryIterator, error) {

	logs, sub, err := _Factory.contract.FilterLogs(opts, "VaultFactory")
	if err != nil {
		return nil, err
	}
	return &FactoryVaultFactoryIterator{contract: _Factory.contract, event: "VaultFactory", logs: logs, sub: sub}, nil
}

// WatchVaultFactory is a free log subscription operation binding the contract event 0xa006ea05a14783821b0248e75d2342cd1681b07509e10a0f08487b080c29dea8.
//
// Solidity: event VaultFactory(address newVaultFactory)
func (_Factory *FactoryFilterer) WatchVaultFactory(opts *bind.WatchOpts, sink chan<- *FactoryVaultFactory) (event.Subscription, error) {

	logs, sub, err := _Factory.contract.WatchLogs(opts, "VaultFactory")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryVaultFactory)
				if err := _Factory.contract.UnpackLog(event, "VaultFactory", log); err != nil {
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

// ParseVaultFactory is a log parse operation binding the contract event 0xa006ea05a14783821b0248e75d2342cd1681b07509e10a0f08487b080c29dea8.
//
// Solidity: event VaultFactory(address newVaultFactory)
func (_Factory *FactoryFilterer) ParseVaultFactory(log types.Log) (*FactoryVaultFactory, error) {
	event := new(FactoryVaultFactory)
	if err := _Factory.contract.UnpackLog(event, "VaultFactory", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
