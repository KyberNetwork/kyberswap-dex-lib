// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package pandafun

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

// IPandaStructsPandaFees is an auto generated low-level Go binding around an user-defined struct.
type IPandaStructsPandaFees struct {
	BuyFee           uint16
	SellFee          uint16
	GraduationFee    uint16
	DeployerFeeShare uint16
}

// FactoryContractMetaData contains all meta data concerning the FactoryContract contract.
var FactoryContractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_treasury\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_dexFactory\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"_initCodeHash\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"_wbera\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"allowed\",\"type\":\"bool\"}],\"name\":\"AllowedImplementationSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"factory\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"initCodeHash\",\"type\":\"bytes32\"}],\"name\":\"FactorySet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"pandaPool\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"IncentiveClaimed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"IncentiveSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"minEndPrice\",\"type\":\"uint256\"}],\"name\":\"MinRaiseSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"minTradeSize\",\"type\":\"uint256\"}],\"name\":\"MinTradeSizeSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"pandaPool\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"PandaDeployed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"buyFee\",\"type\":\"uint16\"},{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"sellFee\",\"type\":\"uint16\"},{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"graduationFee\",\"type\":\"uint16\"},{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"deployerFeeShare\",\"type\":\"uint16\"}],\"name\":\"PandaPoolFeesSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"treasury\",\"type\":\"address\"}],\"name\":\"TreasurySet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"wbera\",\"type\":\"address\"}],\"name\":\"WberaSet\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEPLOYER_MAX_BPS\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MAX_SQRTP_MULTIPLE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MAX_TOKENSINPOOL_SHARE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_SQRTP_MULTIPLE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_TOKENSINPOOL_SHARE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"TOKEN_SUPPLY\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allPools\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"allPoolsLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allowedImplementations\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"allowedImplementationsLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_pandaPool\",\"type\":\"address\"}],\"name\":\"claimIncentive\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"sqrtPa\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"sqrtPb\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"vestingPeriod\",\"type\":\"uint256\"}],\"internalType\":\"structIPandaStructs.PandaPoolParams\",\"name\":\"pp\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"totalTokens\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"pandaToken\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"deployPandaPool\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"sqrtPa\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"sqrtPb\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"vestingPeriod\",\"type\":\"uint256\"}],\"internalType\":\"structIPandaStructs.PandaPoolParams\",\"name\":\"pp\",\"type\":\"tuple\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint16\",\"name\":\"deployerSupplyBps\",\"type\":\"uint16\"}],\"name\":\"deployPandaToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pandaToken\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"sqrtPa\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"sqrtPb\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"vestingPeriod\",\"type\":\"uint256\"}],\"internalType\":\"structIPandaStructs.PandaPoolParams\",\"name\":\"pp\",\"type\":\"tuple\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint16\",\"name\":\"deployerSupplyBps\",\"type\":\"uint16\"}],\"name\":\"deployPandaTokenWithBera\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pandaToken\",\"type\":\"address\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"deployerNonce\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"dexFactory\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getPoolFees\",\"outputs\":[{\"components\":[{\"internalType\":\"uint16\",\"name\":\"buyFee\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"sellFee\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"graduationFee\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"deployerFeeShare\",\"type\":\"uint16\"}],\"internalType\":\"structIPandaStructs.PandaFees\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"scaledPrice\",\"type\":\"uint256\"}],\"name\":\"getSqrtP\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"incentiveAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"incentiveToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initCodeHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isImplementationAllowed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_pandaPool\",\"type\":\"address\"}],\"name\":\"isLegitPool\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"minRaise\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"minTradeSize\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"poolToImplementation\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"poolToIncentiveClaimed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"deployer\",\"type\":\"address\"}],\"name\":\"predictPoolAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_implementation\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"_allowed\",\"type\":\"bool\"}],\"name\":\"setAllowedImplementation\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_factory\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"_initCodeHash\",\"type\":\"bytes32\"}],\"name\":\"setDexFactory\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_incentiveToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_incentiveAmount\",\"type\":\"uint256\"}],\"name\":\"setIncentive\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_minRaise\",\"type\":\"uint256\"}],\"name\":\"setMinRaise\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_baseToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_minTradeSize\",\"type\":\"uint256\"}],\"name\":\"setMinTradeSize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"_buyFee\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"_sellFee\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"_graduationFee\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"_deployerFeeShare\",\"type\":\"uint16\"}],\"name\":\"setPandaPoolFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_treasury\",\"type\":\"address\"}],\"name\":\"setTreasury\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_wbera\",\"type\":\"address\"}],\"name\":\"setWbera\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"treasury\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wbera\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// FactoryContractABI is the input ABI used to generate the binding from.
// Deprecated: Use FactoryContractMetaData.ABI instead.
var FactoryContractABI = FactoryContractMetaData.ABI

// FactoryContract is an auto generated Go binding around an Ethereum contract.
type FactoryContract struct {
	FactoryContractCaller     // Read-only binding to the contract
	FactoryContractTransactor // Write-only binding to the contract
	FactoryContractFilterer   // Log filterer for contract events
}

// FactoryContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type FactoryContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FactoryContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type FactoryContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FactoryContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type FactoryContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FactoryContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type FactoryContractSession struct {
	Contract     *FactoryContract  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// FactoryContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type FactoryContractCallerSession struct {
	Contract *FactoryContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// FactoryContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type FactoryContractTransactorSession struct {
	Contract     *FactoryContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// FactoryContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type FactoryContractRaw struct {
	Contract *FactoryContract // Generic contract binding to access the raw methods on
}

// FactoryContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type FactoryContractCallerRaw struct {
	Contract *FactoryContractCaller // Generic read-only contract binding to access the raw methods on
}

// FactoryContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type FactoryContractTransactorRaw struct {
	Contract *FactoryContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewFactoryContract creates a new instance of FactoryContract, bound to a specific deployed contract.
func NewFactoryContract(address common.Address, backend bind.ContractBackend) (*FactoryContract, error) {
	contract, err := bindFactoryContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &FactoryContract{FactoryContractCaller: FactoryContractCaller{contract: contract}, FactoryContractTransactor: FactoryContractTransactor{contract: contract}, FactoryContractFilterer: FactoryContractFilterer{contract: contract}}, nil
}

// NewFactoryContractCaller creates a new read-only instance of FactoryContract, bound to a specific deployed contract.
func NewFactoryContractCaller(address common.Address, caller bind.ContractCaller) (*FactoryContractCaller, error) {
	contract, err := bindFactoryContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &FactoryContractCaller{contract: contract}, nil
}

// NewFactoryContractTransactor creates a new write-only instance of FactoryContract, bound to a specific deployed contract.
func NewFactoryContractTransactor(address common.Address, transactor bind.ContractTransactor) (*FactoryContractTransactor, error) {
	contract, err := bindFactoryContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &FactoryContractTransactor{contract: contract}, nil
}

// NewFactoryContractFilterer creates a new log filterer instance of FactoryContract, bound to a specific deployed contract.
func NewFactoryContractFilterer(address common.Address, filterer bind.ContractFilterer) (*FactoryContractFilterer, error) {
	contract, err := bindFactoryContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &FactoryContractFilterer{contract: contract}, nil
}

// bindFactoryContract binds a generic wrapper to an already deployed contract.
func bindFactoryContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := FactoryContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FactoryContract *FactoryContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FactoryContract.Contract.FactoryContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FactoryContract *FactoryContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FactoryContract.Contract.FactoryContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FactoryContract *FactoryContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FactoryContract.Contract.FactoryContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FactoryContract *FactoryContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FactoryContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FactoryContract *FactoryContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FactoryContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FactoryContract *FactoryContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FactoryContract.Contract.contract.Transact(opts, method, params...)
}

// DEPLOYERMAXBPS is a free data retrieval call binding the contract method 0x8b1ac4d4.
//
// Solidity: function DEPLOYER_MAX_BPS() view returns(uint16)
func (_FactoryContract *FactoryContractCaller) DEPLOYERMAXBPS(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "DEPLOYER_MAX_BPS")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// DEPLOYERMAXBPS is a free data retrieval call binding the contract method 0x8b1ac4d4.
//
// Solidity: function DEPLOYER_MAX_BPS() view returns(uint16)
func (_FactoryContract *FactoryContractSession) DEPLOYERMAXBPS() (uint16, error) {
	return _FactoryContract.Contract.DEPLOYERMAXBPS(&_FactoryContract.CallOpts)
}

// DEPLOYERMAXBPS is a free data retrieval call binding the contract method 0x8b1ac4d4.
//
// Solidity: function DEPLOYER_MAX_BPS() view returns(uint16)
func (_FactoryContract *FactoryContractCallerSession) DEPLOYERMAXBPS() (uint16, error) {
	return _FactoryContract.Contract.DEPLOYERMAXBPS(&_FactoryContract.CallOpts)
}

// MAXSQRTPMULTIPLE is a free data retrieval call binding the contract method 0xf60ce2c7.
//
// Solidity: function MAX_SQRTP_MULTIPLE() view returns(uint256)
func (_FactoryContract *FactoryContractCaller) MAXSQRTPMULTIPLE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "MAX_SQRTP_MULTIPLE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXSQRTPMULTIPLE is a free data retrieval call binding the contract method 0xf60ce2c7.
//
// Solidity: function MAX_SQRTP_MULTIPLE() view returns(uint256)
func (_FactoryContract *FactoryContractSession) MAXSQRTPMULTIPLE() (*big.Int, error) {
	return _FactoryContract.Contract.MAXSQRTPMULTIPLE(&_FactoryContract.CallOpts)
}

// MAXSQRTPMULTIPLE is a free data retrieval call binding the contract method 0xf60ce2c7.
//
// Solidity: function MAX_SQRTP_MULTIPLE() view returns(uint256)
func (_FactoryContract *FactoryContractCallerSession) MAXSQRTPMULTIPLE() (*big.Int, error) {
	return _FactoryContract.Contract.MAXSQRTPMULTIPLE(&_FactoryContract.CallOpts)
}

// MAXTOKENSINPOOLSHARE is a free data retrieval call binding the contract method 0x4ce6090f.
//
// Solidity: function MAX_TOKENSINPOOL_SHARE() view returns(uint256)
func (_FactoryContract *FactoryContractCaller) MAXTOKENSINPOOLSHARE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "MAX_TOKENSINPOOL_SHARE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXTOKENSINPOOLSHARE is a free data retrieval call binding the contract method 0x4ce6090f.
//
// Solidity: function MAX_TOKENSINPOOL_SHARE() view returns(uint256)
func (_FactoryContract *FactoryContractSession) MAXTOKENSINPOOLSHARE() (*big.Int, error) {
	return _FactoryContract.Contract.MAXTOKENSINPOOLSHARE(&_FactoryContract.CallOpts)
}

// MAXTOKENSINPOOLSHARE is a free data retrieval call binding the contract method 0x4ce6090f.
//
// Solidity: function MAX_TOKENSINPOOL_SHARE() view returns(uint256)
func (_FactoryContract *FactoryContractCallerSession) MAXTOKENSINPOOLSHARE() (*big.Int, error) {
	return _FactoryContract.Contract.MAXTOKENSINPOOLSHARE(&_FactoryContract.CallOpts)
}

// MINSQRTPMULTIPLE is a free data retrieval call binding the contract method 0x8ee221d1.
//
// Solidity: function MIN_SQRTP_MULTIPLE() view returns(uint256)
func (_FactoryContract *FactoryContractCaller) MINSQRTPMULTIPLE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "MIN_SQRTP_MULTIPLE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINSQRTPMULTIPLE is a free data retrieval call binding the contract method 0x8ee221d1.
//
// Solidity: function MIN_SQRTP_MULTIPLE() view returns(uint256)
func (_FactoryContract *FactoryContractSession) MINSQRTPMULTIPLE() (*big.Int, error) {
	return _FactoryContract.Contract.MINSQRTPMULTIPLE(&_FactoryContract.CallOpts)
}

// MINSQRTPMULTIPLE is a free data retrieval call binding the contract method 0x8ee221d1.
//
// Solidity: function MIN_SQRTP_MULTIPLE() view returns(uint256)
func (_FactoryContract *FactoryContractCallerSession) MINSQRTPMULTIPLE() (*big.Int, error) {
	return _FactoryContract.Contract.MINSQRTPMULTIPLE(&_FactoryContract.CallOpts)
}

// MINTOKENSINPOOLSHARE is a free data retrieval call binding the contract method 0x9e9ac16a.
//
// Solidity: function MIN_TOKENSINPOOL_SHARE() view returns(uint256)
func (_FactoryContract *FactoryContractCaller) MINTOKENSINPOOLSHARE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "MIN_TOKENSINPOOL_SHARE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINTOKENSINPOOLSHARE is a free data retrieval call binding the contract method 0x9e9ac16a.
//
// Solidity: function MIN_TOKENSINPOOL_SHARE() view returns(uint256)
func (_FactoryContract *FactoryContractSession) MINTOKENSINPOOLSHARE() (*big.Int, error) {
	return _FactoryContract.Contract.MINTOKENSINPOOLSHARE(&_FactoryContract.CallOpts)
}

// MINTOKENSINPOOLSHARE is a free data retrieval call binding the contract method 0x9e9ac16a.
//
// Solidity: function MIN_TOKENSINPOOL_SHARE() view returns(uint256)
func (_FactoryContract *FactoryContractCallerSession) MINTOKENSINPOOLSHARE() (*big.Int, error) {
	return _FactoryContract.Contract.MINTOKENSINPOOLSHARE(&_FactoryContract.CallOpts)
}

// TOKENSUPPLY is a free data retrieval call binding the contract method 0xb152f6cf.
//
// Solidity: function TOKEN_SUPPLY() view returns(uint256)
func (_FactoryContract *FactoryContractCaller) TOKENSUPPLY(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "TOKEN_SUPPLY")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TOKENSUPPLY is a free data retrieval call binding the contract method 0xb152f6cf.
//
// Solidity: function TOKEN_SUPPLY() view returns(uint256)
func (_FactoryContract *FactoryContractSession) TOKENSUPPLY() (*big.Int, error) {
	return _FactoryContract.Contract.TOKENSUPPLY(&_FactoryContract.CallOpts)
}

// TOKENSUPPLY is a free data retrieval call binding the contract method 0xb152f6cf.
//
// Solidity: function TOKEN_SUPPLY() view returns(uint256)
func (_FactoryContract *FactoryContractCallerSession) TOKENSUPPLY() (*big.Int, error) {
	return _FactoryContract.Contract.TOKENSUPPLY(&_FactoryContract.CallOpts)
}

// AllPools is a free data retrieval call binding the contract method 0x41d1de97.
//
// Solidity: function allPools(uint256 ) view returns(address)
func (_FactoryContract *FactoryContractCaller) AllPools(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "allPools", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AllPools is a free data retrieval call binding the contract method 0x41d1de97.
//
// Solidity: function allPools(uint256 ) view returns(address)
func (_FactoryContract *FactoryContractSession) AllPools(arg0 *big.Int) (common.Address, error) {
	return _FactoryContract.Contract.AllPools(&_FactoryContract.CallOpts, arg0)
}

// AllPools is a free data retrieval call binding the contract method 0x41d1de97.
//
// Solidity: function allPools(uint256 ) view returns(address)
func (_FactoryContract *FactoryContractCallerSession) AllPools(arg0 *big.Int) (common.Address, error) {
	return _FactoryContract.Contract.AllPools(&_FactoryContract.CallOpts, arg0)
}

// AllPoolsLength is a free data retrieval call binding the contract method 0xefde4e64.
//
// Solidity: function allPoolsLength() view returns(uint256)
func (_FactoryContract *FactoryContractCaller) AllPoolsLength(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "allPoolsLength")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// AllPoolsLength is a free data retrieval call binding the contract method 0xefde4e64.
//
// Solidity: function allPoolsLength() view returns(uint256)
func (_FactoryContract *FactoryContractSession) AllPoolsLength() (*big.Int, error) {
	return _FactoryContract.Contract.AllPoolsLength(&_FactoryContract.CallOpts)
}

// AllPoolsLength is a free data retrieval call binding the contract method 0xefde4e64.
//
// Solidity: function allPoolsLength() view returns(uint256)
func (_FactoryContract *FactoryContractCallerSession) AllPoolsLength() (*big.Int, error) {
	return _FactoryContract.Contract.AllPoolsLength(&_FactoryContract.CallOpts)
}

// AllowedImplementations is a free data retrieval call binding the contract method 0x3144e853.
//
// Solidity: function allowedImplementations(uint256 ) view returns(address)
func (_FactoryContract *FactoryContractCaller) AllowedImplementations(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "allowedImplementations", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AllowedImplementations is a free data retrieval call binding the contract method 0x3144e853.
//
// Solidity: function allowedImplementations(uint256 ) view returns(address)
func (_FactoryContract *FactoryContractSession) AllowedImplementations(arg0 *big.Int) (common.Address, error) {
	return _FactoryContract.Contract.AllowedImplementations(&_FactoryContract.CallOpts, arg0)
}

// AllowedImplementations is a free data retrieval call binding the contract method 0x3144e853.
//
// Solidity: function allowedImplementations(uint256 ) view returns(address)
func (_FactoryContract *FactoryContractCallerSession) AllowedImplementations(arg0 *big.Int) (common.Address, error) {
	return _FactoryContract.Contract.AllowedImplementations(&_FactoryContract.CallOpts, arg0)
}

// AllowedImplementationsLength is a free data retrieval call binding the contract method 0x2e852a62.
//
// Solidity: function allowedImplementationsLength() view returns(uint256)
func (_FactoryContract *FactoryContractCaller) AllowedImplementationsLength(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "allowedImplementationsLength")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// AllowedImplementationsLength is a free data retrieval call binding the contract method 0x2e852a62.
//
// Solidity: function allowedImplementationsLength() view returns(uint256)
func (_FactoryContract *FactoryContractSession) AllowedImplementationsLength() (*big.Int, error) {
	return _FactoryContract.Contract.AllowedImplementationsLength(&_FactoryContract.CallOpts)
}

// AllowedImplementationsLength is a free data retrieval call binding the contract method 0x2e852a62.
//
// Solidity: function allowedImplementationsLength() view returns(uint256)
func (_FactoryContract *FactoryContractCallerSession) AllowedImplementationsLength() (*big.Int, error) {
	return _FactoryContract.Contract.AllowedImplementationsLength(&_FactoryContract.CallOpts)
}

// DeployerNonce is a free data retrieval call binding the contract method 0x8b415158.
//
// Solidity: function deployerNonce(address ) view returns(uint256)
func (_FactoryContract *FactoryContractCaller) DeployerNonce(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "deployerNonce", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DeployerNonce is a free data retrieval call binding the contract method 0x8b415158.
//
// Solidity: function deployerNonce(address ) view returns(uint256)
func (_FactoryContract *FactoryContractSession) DeployerNonce(arg0 common.Address) (*big.Int, error) {
	return _FactoryContract.Contract.DeployerNonce(&_FactoryContract.CallOpts, arg0)
}

// DeployerNonce is a free data retrieval call binding the contract method 0x8b415158.
//
// Solidity: function deployerNonce(address ) view returns(uint256)
func (_FactoryContract *FactoryContractCallerSession) DeployerNonce(arg0 common.Address) (*big.Int, error) {
	return _FactoryContract.Contract.DeployerNonce(&_FactoryContract.CallOpts, arg0)
}

// DexFactory is a free data retrieval call binding the contract method 0xb8d8fbb4.
//
// Solidity: function dexFactory() view returns(address)
func (_FactoryContract *FactoryContractCaller) DexFactory(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "dexFactory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DexFactory is a free data retrieval call binding the contract method 0xb8d8fbb4.
//
// Solidity: function dexFactory() view returns(address)
func (_FactoryContract *FactoryContractSession) DexFactory() (common.Address, error) {
	return _FactoryContract.Contract.DexFactory(&_FactoryContract.CallOpts)
}

// DexFactory is a free data retrieval call binding the contract method 0xb8d8fbb4.
//
// Solidity: function dexFactory() view returns(address)
func (_FactoryContract *FactoryContractCallerSession) DexFactory() (common.Address, error) {
	return _FactoryContract.Contract.DexFactory(&_FactoryContract.CallOpts)
}

// GetPoolFees is a free data retrieval call binding the contract method 0xd1d8d060.
//
// Solidity: function getPoolFees() view returns((uint16,uint16,uint16,uint16))
func (_FactoryContract *FactoryContractCaller) GetPoolFees(opts *bind.CallOpts) (IPandaStructsPandaFees, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "getPoolFees")

	if err != nil {
		return *new(IPandaStructsPandaFees), err
	}

	out0 := *abi.ConvertType(out[0], new(IPandaStructsPandaFees)).(*IPandaStructsPandaFees)

	return out0, err

}

// GetPoolFees is a free data retrieval call binding the contract method 0xd1d8d060.
//
// Solidity: function getPoolFees() view returns((uint16,uint16,uint16,uint16))
func (_FactoryContract *FactoryContractSession) GetPoolFees() (IPandaStructsPandaFees, error) {
	return _FactoryContract.Contract.GetPoolFees(&_FactoryContract.CallOpts)
}

// GetPoolFees is a free data retrieval call binding the contract method 0xd1d8d060.
//
// Solidity: function getPoolFees() view returns((uint16,uint16,uint16,uint16))
func (_FactoryContract *FactoryContractCallerSession) GetPoolFees() (IPandaStructsPandaFees, error) {
	return _FactoryContract.Contract.GetPoolFees(&_FactoryContract.CallOpts)
}

// GetSqrtP is a free data retrieval call binding the contract method 0x24282fda.
//
// Solidity: function getSqrtP(uint256 scaledPrice) pure returns(uint256)
func (_FactoryContract *FactoryContractCaller) GetSqrtP(opts *bind.CallOpts, scaledPrice *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "getSqrtP", scaledPrice)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetSqrtP is a free data retrieval call binding the contract method 0x24282fda.
//
// Solidity: function getSqrtP(uint256 scaledPrice) pure returns(uint256)
func (_FactoryContract *FactoryContractSession) GetSqrtP(scaledPrice *big.Int) (*big.Int, error) {
	return _FactoryContract.Contract.GetSqrtP(&_FactoryContract.CallOpts, scaledPrice)
}

// GetSqrtP is a free data retrieval call binding the contract method 0x24282fda.
//
// Solidity: function getSqrtP(uint256 scaledPrice) pure returns(uint256)
func (_FactoryContract *FactoryContractCallerSession) GetSqrtP(scaledPrice *big.Int) (*big.Int, error) {
	return _FactoryContract.Contract.GetSqrtP(&_FactoryContract.CallOpts, scaledPrice)
}

// IncentiveAmount is a free data retrieval call binding the contract method 0xcae5d73b.
//
// Solidity: function incentiveAmount() view returns(uint256)
func (_FactoryContract *FactoryContractCaller) IncentiveAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "incentiveAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// IncentiveAmount is a free data retrieval call binding the contract method 0xcae5d73b.
//
// Solidity: function incentiveAmount() view returns(uint256)
func (_FactoryContract *FactoryContractSession) IncentiveAmount() (*big.Int, error) {
	return _FactoryContract.Contract.IncentiveAmount(&_FactoryContract.CallOpts)
}

// IncentiveAmount is a free data retrieval call binding the contract method 0xcae5d73b.
//
// Solidity: function incentiveAmount() view returns(uint256)
func (_FactoryContract *FactoryContractCallerSession) IncentiveAmount() (*big.Int, error) {
	return _FactoryContract.Contract.IncentiveAmount(&_FactoryContract.CallOpts)
}

// IncentiveToken is a free data retrieval call binding the contract method 0x638126f8.
//
// Solidity: function incentiveToken() view returns(address)
func (_FactoryContract *FactoryContractCaller) IncentiveToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "incentiveToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// IncentiveToken is a free data retrieval call binding the contract method 0x638126f8.
//
// Solidity: function incentiveToken() view returns(address)
func (_FactoryContract *FactoryContractSession) IncentiveToken() (common.Address, error) {
	return _FactoryContract.Contract.IncentiveToken(&_FactoryContract.CallOpts)
}

// IncentiveToken is a free data retrieval call binding the contract method 0x638126f8.
//
// Solidity: function incentiveToken() view returns(address)
func (_FactoryContract *FactoryContractCallerSession) IncentiveToken() (common.Address, error) {
	return _FactoryContract.Contract.IncentiveToken(&_FactoryContract.CallOpts)
}

// InitCodeHash is a free data retrieval call binding the contract method 0xdb4c545e.
//
// Solidity: function initCodeHash() view returns(bytes32)
func (_FactoryContract *FactoryContractCaller) InitCodeHash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "initCodeHash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// InitCodeHash is a free data retrieval call binding the contract method 0xdb4c545e.
//
// Solidity: function initCodeHash() view returns(bytes32)
func (_FactoryContract *FactoryContractSession) InitCodeHash() ([32]byte, error) {
	return _FactoryContract.Contract.InitCodeHash(&_FactoryContract.CallOpts)
}

// InitCodeHash is a free data retrieval call binding the contract method 0xdb4c545e.
//
// Solidity: function initCodeHash() view returns(bytes32)
func (_FactoryContract *FactoryContractCallerSession) InitCodeHash() ([32]byte, error) {
	return _FactoryContract.Contract.InitCodeHash(&_FactoryContract.CallOpts)
}

// IsImplementationAllowed is a free data retrieval call binding the contract method 0x0d4b4bda.
//
// Solidity: function isImplementationAllowed(address ) view returns(bool)
func (_FactoryContract *FactoryContractCaller) IsImplementationAllowed(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "isImplementationAllowed", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsImplementationAllowed is a free data retrieval call binding the contract method 0x0d4b4bda.
//
// Solidity: function isImplementationAllowed(address ) view returns(bool)
func (_FactoryContract *FactoryContractSession) IsImplementationAllowed(arg0 common.Address) (bool, error) {
	return _FactoryContract.Contract.IsImplementationAllowed(&_FactoryContract.CallOpts, arg0)
}

// IsImplementationAllowed is a free data retrieval call binding the contract method 0x0d4b4bda.
//
// Solidity: function isImplementationAllowed(address ) view returns(bool)
func (_FactoryContract *FactoryContractCallerSession) IsImplementationAllowed(arg0 common.Address) (bool, error) {
	return _FactoryContract.Contract.IsImplementationAllowed(&_FactoryContract.CallOpts, arg0)
}

// IsLegitPool is a free data retrieval call binding the contract method 0xe20497e3.
//
// Solidity: function isLegitPool(address _pandaPool) view returns(bool)
func (_FactoryContract *FactoryContractCaller) IsLegitPool(opts *bind.CallOpts, _pandaPool common.Address) (bool, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "isLegitPool", _pandaPool)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsLegitPool is a free data retrieval call binding the contract method 0xe20497e3.
//
// Solidity: function isLegitPool(address _pandaPool) view returns(bool)
func (_FactoryContract *FactoryContractSession) IsLegitPool(_pandaPool common.Address) (bool, error) {
	return _FactoryContract.Contract.IsLegitPool(&_FactoryContract.CallOpts, _pandaPool)
}

// IsLegitPool is a free data retrieval call binding the contract method 0xe20497e3.
//
// Solidity: function isLegitPool(address _pandaPool) view returns(bool)
func (_FactoryContract *FactoryContractCallerSession) IsLegitPool(_pandaPool common.Address) (bool, error) {
	return _FactoryContract.Contract.IsLegitPool(&_FactoryContract.CallOpts, _pandaPool)
}

// MinRaise is a free data retrieval call binding the contract method 0x056a35df.
//
// Solidity: function minRaise(address ) view returns(uint256)
func (_FactoryContract *FactoryContractCaller) MinRaise(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "minRaise", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinRaise is a free data retrieval call binding the contract method 0x056a35df.
//
// Solidity: function minRaise(address ) view returns(uint256)
func (_FactoryContract *FactoryContractSession) MinRaise(arg0 common.Address) (*big.Int, error) {
	return _FactoryContract.Contract.MinRaise(&_FactoryContract.CallOpts, arg0)
}

// MinRaise is a free data retrieval call binding the contract method 0x056a35df.
//
// Solidity: function minRaise(address ) view returns(uint256)
func (_FactoryContract *FactoryContractCallerSession) MinRaise(arg0 common.Address) (*big.Int, error) {
	return _FactoryContract.Contract.MinRaise(&_FactoryContract.CallOpts, arg0)
}

// MinTradeSize is a free data retrieval call binding the contract method 0x48f7e4d1.
//
// Solidity: function minTradeSize(address ) view returns(uint256)
func (_FactoryContract *FactoryContractCaller) MinTradeSize(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "minTradeSize", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinTradeSize is a free data retrieval call binding the contract method 0x48f7e4d1.
//
// Solidity: function minTradeSize(address ) view returns(uint256)
func (_FactoryContract *FactoryContractSession) MinTradeSize(arg0 common.Address) (*big.Int, error) {
	return _FactoryContract.Contract.MinTradeSize(&_FactoryContract.CallOpts, arg0)
}

// MinTradeSize is a free data retrieval call binding the contract method 0x48f7e4d1.
//
// Solidity: function minTradeSize(address ) view returns(uint256)
func (_FactoryContract *FactoryContractCallerSession) MinTradeSize(arg0 common.Address) (*big.Int, error) {
	return _FactoryContract.Contract.MinTradeSize(&_FactoryContract.CallOpts, arg0)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_FactoryContract *FactoryContractCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_FactoryContract *FactoryContractSession) Owner() (common.Address, error) {
	return _FactoryContract.Contract.Owner(&_FactoryContract.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_FactoryContract *FactoryContractCallerSession) Owner() (common.Address, error) {
	return _FactoryContract.Contract.Owner(&_FactoryContract.CallOpts)
}

// PoolToImplementation is a free data retrieval call binding the contract method 0xb7761f92.
//
// Solidity: function poolToImplementation(address ) view returns(address)
func (_FactoryContract *FactoryContractCaller) PoolToImplementation(opts *bind.CallOpts, arg0 common.Address) (common.Address, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "poolToImplementation", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PoolToImplementation is a free data retrieval call binding the contract method 0xb7761f92.
//
// Solidity: function poolToImplementation(address ) view returns(address)
func (_FactoryContract *FactoryContractSession) PoolToImplementation(arg0 common.Address) (common.Address, error) {
	return _FactoryContract.Contract.PoolToImplementation(&_FactoryContract.CallOpts, arg0)
}

// PoolToImplementation is a free data retrieval call binding the contract method 0xb7761f92.
//
// Solidity: function poolToImplementation(address ) view returns(address)
func (_FactoryContract *FactoryContractCallerSession) PoolToImplementation(arg0 common.Address) (common.Address, error) {
	return _FactoryContract.Contract.PoolToImplementation(&_FactoryContract.CallOpts, arg0)
}

// PoolToIncentiveClaimed is a free data retrieval call binding the contract method 0x508dbc84.
//
// Solidity: function poolToIncentiveClaimed(address ) view returns(bool)
func (_FactoryContract *FactoryContractCaller) PoolToIncentiveClaimed(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "poolToIncentiveClaimed", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// PoolToIncentiveClaimed is a free data retrieval call binding the contract method 0x508dbc84.
//
// Solidity: function poolToIncentiveClaimed(address ) view returns(bool)
func (_FactoryContract *FactoryContractSession) PoolToIncentiveClaimed(arg0 common.Address) (bool, error) {
	return _FactoryContract.Contract.PoolToIncentiveClaimed(&_FactoryContract.CallOpts, arg0)
}

// PoolToIncentiveClaimed is a free data retrieval call binding the contract method 0x508dbc84.
//
// Solidity: function poolToIncentiveClaimed(address ) view returns(bool)
func (_FactoryContract *FactoryContractCallerSession) PoolToIncentiveClaimed(arg0 common.Address) (bool, error) {
	return _FactoryContract.Contract.PoolToIncentiveClaimed(&_FactoryContract.CallOpts, arg0)
}

// PredictPoolAddress is a free data retrieval call binding the contract method 0x3dd680e6.
//
// Solidity: function predictPoolAddress(address implementation, address deployer) view returns(address)
func (_FactoryContract *FactoryContractCaller) PredictPoolAddress(opts *bind.CallOpts, implementation common.Address, deployer common.Address) (common.Address, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "predictPoolAddress", implementation, deployer)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PredictPoolAddress is a free data retrieval call binding the contract method 0x3dd680e6.
//
// Solidity: function predictPoolAddress(address implementation, address deployer) view returns(address)
func (_FactoryContract *FactoryContractSession) PredictPoolAddress(implementation common.Address, deployer common.Address) (common.Address, error) {
	return _FactoryContract.Contract.PredictPoolAddress(&_FactoryContract.CallOpts, implementation, deployer)
}

// PredictPoolAddress is a free data retrieval call binding the contract method 0x3dd680e6.
//
// Solidity: function predictPoolAddress(address implementation, address deployer) view returns(address)
func (_FactoryContract *FactoryContractCallerSession) PredictPoolAddress(implementation common.Address, deployer common.Address) (common.Address, error) {
	return _FactoryContract.Contract.PredictPoolAddress(&_FactoryContract.CallOpts, implementation, deployer)
}

// Treasury is a free data retrieval call binding the contract method 0x61d027b3.
//
// Solidity: function treasury() view returns(address)
func (_FactoryContract *FactoryContractCaller) Treasury(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "treasury")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Treasury is a free data retrieval call binding the contract method 0x61d027b3.
//
// Solidity: function treasury() view returns(address)
func (_FactoryContract *FactoryContractSession) Treasury() (common.Address, error) {
	return _FactoryContract.Contract.Treasury(&_FactoryContract.CallOpts)
}

// Treasury is a free data retrieval call binding the contract method 0x61d027b3.
//
// Solidity: function treasury() view returns(address)
func (_FactoryContract *FactoryContractCallerSession) Treasury() (common.Address, error) {
	return _FactoryContract.Contract.Treasury(&_FactoryContract.CallOpts)
}

// Wbera is a free data retrieval call binding the contract method 0x31f41a33.
//
// Solidity: function wbera() view returns(address)
func (_FactoryContract *FactoryContractCaller) Wbera(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FactoryContract.contract.Call(opts, &out, "wbera")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Wbera is a free data retrieval call binding the contract method 0x31f41a33.
//
// Solidity: function wbera() view returns(address)
func (_FactoryContract *FactoryContractSession) Wbera() (common.Address, error) {
	return _FactoryContract.Contract.Wbera(&_FactoryContract.CallOpts)
}

// Wbera is a free data retrieval call binding the contract method 0x31f41a33.
//
// Solidity: function wbera() view returns(address)
func (_FactoryContract *FactoryContractCallerSession) Wbera() (common.Address, error) {
	return _FactoryContract.Contract.Wbera(&_FactoryContract.CallOpts)
}

// ClaimIncentive is a paid mutator transaction binding the contract method 0xdfa9f205.
//
// Solidity: function claimIncentive(address _pandaPool) returns()
func (_FactoryContract *FactoryContractTransactor) ClaimIncentive(opts *bind.TransactOpts, _pandaPool common.Address) (*types.Transaction, error) {
	return _FactoryContract.contract.Transact(opts, "claimIncentive", _pandaPool)
}

// ClaimIncentive is a paid mutator transaction binding the contract method 0xdfa9f205.
//
// Solidity: function claimIncentive(address _pandaPool) returns()
func (_FactoryContract *FactoryContractSession) ClaimIncentive(_pandaPool common.Address) (*types.Transaction, error) {
	return _FactoryContract.Contract.ClaimIncentive(&_FactoryContract.TransactOpts, _pandaPool)
}

// ClaimIncentive is a paid mutator transaction binding the contract method 0xdfa9f205.
//
// Solidity: function claimIncentive(address _pandaPool) returns()
func (_FactoryContract *FactoryContractTransactorSession) ClaimIncentive(_pandaPool common.Address) (*types.Transaction, error) {
	return _FactoryContract.Contract.ClaimIncentive(&_FactoryContract.TransactOpts, _pandaPool)
}

// DeployPandaPool is a paid mutator transaction binding the contract method 0xb4d35a5d.
//
// Solidity: function deployPandaPool(address implementation, (address,uint256,uint256,uint256) pp, uint256 totalTokens, address pandaToken, bytes data) returns(address)
func (_FactoryContract *FactoryContractTransactor) DeployPandaPool(opts *bind.TransactOpts, implementation common.Address, pp IPandaStructsPandaPoolParams, totalTokens *big.Int, pandaToken common.Address, data []byte) (*types.Transaction, error) {
	return _FactoryContract.contract.Transact(opts, "deployPandaPool", implementation, pp, totalTokens, pandaToken, data)
}

// DeployPandaPool is a paid mutator transaction binding the contract method 0xb4d35a5d.
//
// Solidity: function deployPandaPool(address implementation, (address,uint256,uint256,uint256) pp, uint256 totalTokens, address pandaToken, bytes data) returns(address)
func (_FactoryContract *FactoryContractSession) DeployPandaPool(implementation common.Address, pp IPandaStructsPandaPoolParams, totalTokens *big.Int, pandaToken common.Address, data []byte) (*types.Transaction, error) {
	return _FactoryContract.Contract.DeployPandaPool(&_FactoryContract.TransactOpts, implementation, pp, totalTokens, pandaToken, data)
}

// DeployPandaPool is a paid mutator transaction binding the contract method 0xb4d35a5d.
//
// Solidity: function deployPandaPool(address implementation, (address,uint256,uint256,uint256) pp, uint256 totalTokens, address pandaToken, bytes data) returns(address)
func (_FactoryContract *FactoryContractTransactorSession) DeployPandaPool(implementation common.Address, pp IPandaStructsPandaPoolParams, totalTokens *big.Int, pandaToken common.Address, data []byte) (*types.Transaction, error) {
	return _FactoryContract.Contract.DeployPandaPool(&_FactoryContract.TransactOpts, implementation, pp, totalTokens, pandaToken, data)
}

// DeployPandaToken is a paid mutator transaction binding the contract method 0x08d103f1.
//
// Solidity: function deployPandaToken(address implementation, (address,uint256,uint256,uint256) pp, string name, string symbol, uint16 deployerSupplyBps) returns(address pandaToken)
func (_FactoryContract *FactoryContractTransactor) DeployPandaToken(opts *bind.TransactOpts, implementation common.Address, pp IPandaStructsPandaPoolParams, name string, symbol string, deployerSupplyBps uint16) (*types.Transaction, error) {
	return _FactoryContract.contract.Transact(opts, "deployPandaToken", implementation, pp, name, symbol, deployerSupplyBps)
}

// DeployPandaToken is a paid mutator transaction binding the contract method 0x08d103f1.
//
// Solidity: function deployPandaToken(address implementation, (address,uint256,uint256,uint256) pp, string name, string symbol, uint16 deployerSupplyBps) returns(address pandaToken)
func (_FactoryContract *FactoryContractSession) DeployPandaToken(implementation common.Address, pp IPandaStructsPandaPoolParams, name string, symbol string, deployerSupplyBps uint16) (*types.Transaction, error) {
	return _FactoryContract.Contract.DeployPandaToken(&_FactoryContract.TransactOpts, implementation, pp, name, symbol, deployerSupplyBps)
}

// DeployPandaToken is a paid mutator transaction binding the contract method 0x08d103f1.
//
// Solidity: function deployPandaToken(address implementation, (address,uint256,uint256,uint256) pp, string name, string symbol, uint16 deployerSupplyBps) returns(address pandaToken)
func (_FactoryContract *FactoryContractTransactorSession) DeployPandaToken(implementation common.Address, pp IPandaStructsPandaPoolParams, name string, symbol string, deployerSupplyBps uint16) (*types.Transaction, error) {
	return _FactoryContract.Contract.DeployPandaToken(&_FactoryContract.TransactOpts, implementation, pp, name, symbol, deployerSupplyBps)
}

// DeployPandaTokenWithBera is a paid mutator transaction binding the contract method 0xec2334e5.
//
// Solidity: function deployPandaTokenWithBera(address implementation, (address,uint256,uint256,uint256) pp, string name, string symbol, uint16 deployerSupplyBps) payable returns(address pandaToken)
func (_FactoryContract *FactoryContractTransactor) DeployPandaTokenWithBera(opts *bind.TransactOpts, implementation common.Address, pp IPandaStructsPandaPoolParams, name string, symbol string, deployerSupplyBps uint16) (*types.Transaction, error) {
	return _FactoryContract.contract.Transact(opts, "deployPandaTokenWithBera", implementation, pp, name, symbol, deployerSupplyBps)
}

// DeployPandaTokenWithBera is a paid mutator transaction binding the contract method 0xec2334e5.
//
// Solidity: function deployPandaTokenWithBera(address implementation, (address,uint256,uint256,uint256) pp, string name, string symbol, uint16 deployerSupplyBps) payable returns(address pandaToken)
func (_FactoryContract *FactoryContractSession) DeployPandaTokenWithBera(implementation common.Address, pp IPandaStructsPandaPoolParams, name string, symbol string, deployerSupplyBps uint16) (*types.Transaction, error) {
	return _FactoryContract.Contract.DeployPandaTokenWithBera(&_FactoryContract.TransactOpts, implementation, pp, name, symbol, deployerSupplyBps)
}

// DeployPandaTokenWithBera is a paid mutator transaction binding the contract method 0xec2334e5.
//
// Solidity: function deployPandaTokenWithBera(address implementation, (address,uint256,uint256,uint256) pp, string name, string symbol, uint16 deployerSupplyBps) payable returns(address pandaToken)
func (_FactoryContract *FactoryContractTransactorSession) DeployPandaTokenWithBera(implementation common.Address, pp IPandaStructsPandaPoolParams, name string, symbol string, deployerSupplyBps uint16) (*types.Transaction, error) {
	return _FactoryContract.Contract.DeployPandaTokenWithBera(&_FactoryContract.TransactOpts, implementation, pp, name, symbol, deployerSupplyBps)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_FactoryContract *FactoryContractTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FactoryContract.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_FactoryContract *FactoryContractSession) RenounceOwnership() (*types.Transaction, error) {
	return _FactoryContract.Contract.RenounceOwnership(&_FactoryContract.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_FactoryContract *FactoryContractTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _FactoryContract.Contract.RenounceOwnership(&_FactoryContract.TransactOpts)
}

// SetAllowedImplementation is a paid mutator transaction binding the contract method 0x92469457.
//
// Solidity: function setAllowedImplementation(address _implementation, bool _allowed) returns()
func (_FactoryContract *FactoryContractTransactor) SetAllowedImplementation(opts *bind.TransactOpts, _implementation common.Address, _allowed bool) (*types.Transaction, error) {
	return _FactoryContract.contract.Transact(opts, "setAllowedImplementation", _implementation, _allowed)
}

// SetAllowedImplementation is a paid mutator transaction binding the contract method 0x92469457.
//
// Solidity: function setAllowedImplementation(address _implementation, bool _allowed) returns()
func (_FactoryContract *FactoryContractSession) SetAllowedImplementation(_implementation common.Address, _allowed bool) (*types.Transaction, error) {
	return _FactoryContract.Contract.SetAllowedImplementation(&_FactoryContract.TransactOpts, _implementation, _allowed)
}

// SetAllowedImplementation is a paid mutator transaction binding the contract method 0x92469457.
//
// Solidity: function setAllowedImplementation(address _implementation, bool _allowed) returns()
func (_FactoryContract *FactoryContractTransactorSession) SetAllowedImplementation(_implementation common.Address, _allowed bool) (*types.Transaction, error) {
	return _FactoryContract.Contract.SetAllowedImplementation(&_FactoryContract.TransactOpts, _implementation, _allowed)
}

// SetDexFactory is a paid mutator transaction binding the contract method 0x09f26d7f.
//
// Solidity: function setDexFactory(address _factory, bytes32 _initCodeHash) returns()
func (_FactoryContract *FactoryContractTransactor) SetDexFactory(opts *bind.TransactOpts, _factory common.Address, _initCodeHash [32]byte) (*types.Transaction, error) {
	return _FactoryContract.contract.Transact(opts, "setDexFactory", _factory, _initCodeHash)
}

// SetDexFactory is a paid mutator transaction binding the contract method 0x09f26d7f.
//
// Solidity: function setDexFactory(address _factory, bytes32 _initCodeHash) returns()
func (_FactoryContract *FactoryContractSession) SetDexFactory(_factory common.Address, _initCodeHash [32]byte) (*types.Transaction, error) {
	return _FactoryContract.Contract.SetDexFactory(&_FactoryContract.TransactOpts, _factory, _initCodeHash)
}

// SetDexFactory is a paid mutator transaction binding the contract method 0x09f26d7f.
//
// Solidity: function setDexFactory(address _factory, bytes32 _initCodeHash) returns()
func (_FactoryContract *FactoryContractTransactorSession) SetDexFactory(_factory common.Address, _initCodeHash [32]byte) (*types.Transaction, error) {
	return _FactoryContract.Contract.SetDexFactory(&_FactoryContract.TransactOpts, _factory, _initCodeHash)
}

// SetIncentive is a paid mutator transaction binding the contract method 0xe8efb832.
//
// Solidity: function setIncentive(address _incentiveToken, uint256 _incentiveAmount) returns()
func (_FactoryContract *FactoryContractTransactor) SetIncentive(opts *bind.TransactOpts, _incentiveToken common.Address, _incentiveAmount *big.Int) (*types.Transaction, error) {
	return _FactoryContract.contract.Transact(opts, "setIncentive", _incentiveToken, _incentiveAmount)
}

// SetIncentive is a paid mutator transaction binding the contract method 0xe8efb832.
//
// Solidity: function setIncentive(address _incentiveToken, uint256 _incentiveAmount) returns()
func (_FactoryContract *FactoryContractSession) SetIncentive(_incentiveToken common.Address, _incentiveAmount *big.Int) (*types.Transaction, error) {
	return _FactoryContract.Contract.SetIncentive(&_FactoryContract.TransactOpts, _incentiveToken, _incentiveAmount)
}

// SetIncentive is a paid mutator transaction binding the contract method 0xe8efb832.
//
// Solidity: function setIncentive(address _incentiveToken, uint256 _incentiveAmount) returns()
func (_FactoryContract *FactoryContractTransactorSession) SetIncentive(_incentiveToken common.Address, _incentiveAmount *big.Int) (*types.Transaction, error) {
	return _FactoryContract.Contract.SetIncentive(&_FactoryContract.TransactOpts, _incentiveToken, _incentiveAmount)
}

// SetMinRaise is a paid mutator transaction binding the contract method 0xbe7b7905.
//
// Solidity: function setMinRaise(address baseToken, uint256 _minRaise) returns()
func (_FactoryContract *FactoryContractTransactor) SetMinRaise(opts *bind.TransactOpts, baseToken common.Address, _minRaise *big.Int) (*types.Transaction, error) {
	return _FactoryContract.contract.Transact(opts, "setMinRaise", baseToken, _minRaise)
}

// SetMinRaise is a paid mutator transaction binding the contract method 0xbe7b7905.
//
// Solidity: function setMinRaise(address baseToken, uint256 _minRaise) returns()
func (_FactoryContract *FactoryContractSession) SetMinRaise(baseToken common.Address, _minRaise *big.Int) (*types.Transaction, error) {
	return _FactoryContract.Contract.SetMinRaise(&_FactoryContract.TransactOpts, baseToken, _minRaise)
}

// SetMinRaise is a paid mutator transaction binding the contract method 0xbe7b7905.
//
// Solidity: function setMinRaise(address baseToken, uint256 _minRaise) returns()
func (_FactoryContract *FactoryContractTransactorSession) SetMinRaise(baseToken common.Address, _minRaise *big.Int) (*types.Transaction, error) {
	return _FactoryContract.Contract.SetMinRaise(&_FactoryContract.TransactOpts, baseToken, _minRaise)
}

// SetMinTradeSize is a paid mutator transaction binding the contract method 0xf442c1c5.
//
// Solidity: function setMinTradeSize(address _baseToken, uint256 _minTradeSize) returns()
func (_FactoryContract *FactoryContractTransactor) SetMinTradeSize(opts *bind.TransactOpts, _baseToken common.Address, _minTradeSize *big.Int) (*types.Transaction, error) {
	return _FactoryContract.contract.Transact(opts, "setMinTradeSize", _baseToken, _minTradeSize)
}

// SetMinTradeSize is a paid mutator transaction binding the contract method 0xf442c1c5.
//
// Solidity: function setMinTradeSize(address _baseToken, uint256 _minTradeSize) returns()
func (_FactoryContract *FactoryContractSession) SetMinTradeSize(_baseToken common.Address, _minTradeSize *big.Int) (*types.Transaction, error) {
	return _FactoryContract.Contract.SetMinTradeSize(&_FactoryContract.TransactOpts, _baseToken, _minTradeSize)
}

// SetMinTradeSize is a paid mutator transaction binding the contract method 0xf442c1c5.
//
// Solidity: function setMinTradeSize(address _baseToken, uint256 _minTradeSize) returns()
func (_FactoryContract *FactoryContractTransactorSession) SetMinTradeSize(_baseToken common.Address, _minTradeSize *big.Int) (*types.Transaction, error) {
	return _FactoryContract.Contract.SetMinTradeSize(&_FactoryContract.TransactOpts, _baseToken, _minTradeSize)
}

// SetPandaPoolFees is a paid mutator transaction binding the contract method 0x3dad9b70.
//
// Solidity: function setPandaPoolFees(uint16 _buyFee, uint16 _sellFee, uint16 _graduationFee, uint16 _deployerFeeShare) returns()
func (_FactoryContract *FactoryContractTransactor) SetPandaPoolFees(opts *bind.TransactOpts, _buyFee uint16, _sellFee uint16, _graduationFee uint16, _deployerFeeShare uint16) (*types.Transaction, error) {
	return _FactoryContract.contract.Transact(opts, "setPandaPoolFees", _buyFee, _sellFee, _graduationFee, _deployerFeeShare)
}

// SetPandaPoolFees is a paid mutator transaction binding the contract method 0x3dad9b70.
//
// Solidity: function setPandaPoolFees(uint16 _buyFee, uint16 _sellFee, uint16 _graduationFee, uint16 _deployerFeeShare) returns()
func (_FactoryContract *FactoryContractSession) SetPandaPoolFees(_buyFee uint16, _sellFee uint16, _graduationFee uint16, _deployerFeeShare uint16) (*types.Transaction, error) {
	return _FactoryContract.Contract.SetPandaPoolFees(&_FactoryContract.TransactOpts, _buyFee, _sellFee, _graduationFee, _deployerFeeShare)
}

// SetPandaPoolFees is a paid mutator transaction binding the contract method 0x3dad9b70.
//
// Solidity: function setPandaPoolFees(uint16 _buyFee, uint16 _sellFee, uint16 _graduationFee, uint16 _deployerFeeShare) returns()
func (_FactoryContract *FactoryContractTransactorSession) SetPandaPoolFees(_buyFee uint16, _sellFee uint16, _graduationFee uint16, _deployerFeeShare uint16) (*types.Transaction, error) {
	return _FactoryContract.Contract.SetPandaPoolFees(&_FactoryContract.TransactOpts, _buyFee, _sellFee, _graduationFee, _deployerFeeShare)
}

// SetTreasury is a paid mutator transaction binding the contract method 0xf0f44260.
//
// Solidity: function setTreasury(address _treasury) returns()
func (_FactoryContract *FactoryContractTransactor) SetTreasury(opts *bind.TransactOpts, _treasury common.Address) (*types.Transaction, error) {
	return _FactoryContract.contract.Transact(opts, "setTreasury", _treasury)
}

// SetTreasury is a paid mutator transaction binding the contract method 0xf0f44260.
//
// Solidity: function setTreasury(address _treasury) returns()
func (_FactoryContract *FactoryContractSession) SetTreasury(_treasury common.Address) (*types.Transaction, error) {
	return _FactoryContract.Contract.SetTreasury(&_FactoryContract.TransactOpts, _treasury)
}

// SetTreasury is a paid mutator transaction binding the contract method 0xf0f44260.
//
// Solidity: function setTreasury(address _treasury) returns()
func (_FactoryContract *FactoryContractTransactorSession) SetTreasury(_treasury common.Address) (*types.Transaction, error) {
	return _FactoryContract.Contract.SetTreasury(&_FactoryContract.TransactOpts, _treasury)
}

// SetWbera is a paid mutator transaction binding the contract method 0x9389e3e4.
//
// Solidity: function setWbera(address _wbera) returns()
func (_FactoryContract *FactoryContractTransactor) SetWbera(opts *bind.TransactOpts, _wbera common.Address) (*types.Transaction, error) {
	return _FactoryContract.contract.Transact(opts, "setWbera", _wbera)
}

// SetWbera is a paid mutator transaction binding the contract method 0x9389e3e4.
//
// Solidity: function setWbera(address _wbera) returns()
func (_FactoryContract *FactoryContractSession) SetWbera(_wbera common.Address) (*types.Transaction, error) {
	return _FactoryContract.Contract.SetWbera(&_FactoryContract.TransactOpts, _wbera)
}

// SetWbera is a paid mutator transaction binding the contract method 0x9389e3e4.
//
// Solidity: function setWbera(address _wbera) returns()
func (_FactoryContract *FactoryContractTransactorSession) SetWbera(_wbera common.Address) (*types.Transaction, error) {
	return _FactoryContract.Contract.SetWbera(&_FactoryContract.TransactOpts, _wbera)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_FactoryContract *FactoryContractTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _FactoryContract.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_FactoryContract *FactoryContractSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _FactoryContract.Contract.TransferOwnership(&_FactoryContract.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_FactoryContract *FactoryContractTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _FactoryContract.Contract.TransferOwnership(&_FactoryContract.TransactOpts, newOwner)
}

// FactoryContractAllowedImplementationSetIterator is returned from FilterAllowedImplementationSet and is used to iterate over the raw logs and unpacked data for AllowedImplementationSet events raised by the FactoryContract contract.
type FactoryContractAllowedImplementationSetIterator struct {
	Event *FactoryContractAllowedImplementationSet // Event containing the contract specifics and raw log

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
func (it *FactoryContractAllowedImplementationSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryContractAllowedImplementationSet)
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
		it.Event = new(FactoryContractAllowedImplementationSet)
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
func (it *FactoryContractAllowedImplementationSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryContractAllowedImplementationSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryContractAllowedImplementationSet represents a AllowedImplementationSet event raised by the FactoryContract contract.
type FactoryContractAllowedImplementationSet struct {
	Implementation common.Address
	Allowed        bool
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterAllowedImplementationSet is a free log retrieval operation binding the contract event 0xedb3ccc8722c14f1eafe86c77c0852c4296c22b9da711af50a4e4e94401a3a79.
//
// Solidity: event AllowedImplementationSet(address indexed implementation, bool allowed)
func (_FactoryContract *FactoryContractFilterer) FilterAllowedImplementationSet(opts *bind.FilterOpts, implementation []common.Address) (*FactoryContractAllowedImplementationSetIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _FactoryContract.contract.FilterLogs(opts, "AllowedImplementationSet", implementationRule)
	if err != nil {
		return nil, err
	}
	return &FactoryContractAllowedImplementationSetIterator{contract: _FactoryContract.contract, event: "AllowedImplementationSet", logs: logs, sub: sub}, nil
}

// WatchAllowedImplementationSet is a free log subscription operation binding the contract event 0xedb3ccc8722c14f1eafe86c77c0852c4296c22b9da711af50a4e4e94401a3a79.
//
// Solidity: event AllowedImplementationSet(address indexed implementation, bool allowed)
func (_FactoryContract *FactoryContractFilterer) WatchAllowedImplementationSet(opts *bind.WatchOpts, sink chan<- *FactoryContractAllowedImplementationSet, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _FactoryContract.contract.WatchLogs(opts, "AllowedImplementationSet", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryContractAllowedImplementationSet)
				if err := _FactoryContract.contract.UnpackLog(event, "AllowedImplementationSet", log); err != nil {
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

// ParseAllowedImplementationSet is a log parse operation binding the contract event 0xedb3ccc8722c14f1eafe86c77c0852c4296c22b9da711af50a4e4e94401a3a79.
//
// Solidity: event AllowedImplementationSet(address indexed implementation, bool allowed)
func (_FactoryContract *FactoryContractFilterer) ParseAllowedImplementationSet(log types.Log) (*FactoryContractAllowedImplementationSet, error) {
	event := new(FactoryContractAllowedImplementationSet)
	if err := _FactoryContract.contract.UnpackLog(event, "AllowedImplementationSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryContractFactorySetIterator is returned from FilterFactorySet and is used to iterate over the raw logs and unpacked data for FactorySet events raised by the FactoryContract contract.
type FactoryContractFactorySetIterator struct {
	Event *FactoryContractFactorySet // Event containing the contract specifics and raw log

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
func (it *FactoryContractFactorySetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryContractFactorySet)
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
		it.Event = new(FactoryContractFactorySet)
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
func (it *FactoryContractFactorySetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryContractFactorySetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryContractFactorySet represents a FactorySet event raised by the FactoryContract contract.
type FactoryContractFactorySet struct {
	Factory      common.Address
	InitCodeHash [32]byte
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterFactorySet is a free log retrieval operation binding the contract event 0xc75374bcfd658d0f23178bb5576c83c89dce9577017f91d6ac123eb9fa139c1b.
//
// Solidity: event FactorySet(address indexed factory, bytes32 initCodeHash)
func (_FactoryContract *FactoryContractFilterer) FilterFactorySet(opts *bind.FilterOpts, factory []common.Address) (*FactoryContractFactorySetIterator, error) {

	var factoryRule []interface{}
	for _, factoryItem := range factory {
		factoryRule = append(factoryRule, factoryItem)
	}

	logs, sub, err := _FactoryContract.contract.FilterLogs(opts, "FactorySet", factoryRule)
	if err != nil {
		return nil, err
	}
	return &FactoryContractFactorySetIterator{contract: _FactoryContract.contract, event: "FactorySet", logs: logs, sub: sub}, nil
}

// WatchFactorySet is a free log subscription operation binding the contract event 0xc75374bcfd658d0f23178bb5576c83c89dce9577017f91d6ac123eb9fa139c1b.
//
// Solidity: event FactorySet(address indexed factory, bytes32 initCodeHash)
func (_FactoryContract *FactoryContractFilterer) WatchFactorySet(opts *bind.WatchOpts, sink chan<- *FactoryContractFactorySet, factory []common.Address) (event.Subscription, error) {

	var factoryRule []interface{}
	for _, factoryItem := range factory {
		factoryRule = append(factoryRule, factoryItem)
	}

	logs, sub, err := _FactoryContract.contract.WatchLogs(opts, "FactorySet", factoryRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryContractFactorySet)
				if err := _FactoryContract.contract.UnpackLog(event, "FactorySet", log); err != nil {
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

// ParseFactorySet is a log parse operation binding the contract event 0xc75374bcfd658d0f23178bb5576c83c89dce9577017f91d6ac123eb9fa139c1b.
//
// Solidity: event FactorySet(address indexed factory, bytes32 initCodeHash)
func (_FactoryContract *FactoryContractFilterer) ParseFactorySet(log types.Log) (*FactoryContractFactorySet, error) {
	event := new(FactoryContractFactorySet)
	if err := _FactoryContract.contract.UnpackLog(event, "FactorySet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryContractIncentiveClaimedIterator is returned from FilterIncentiveClaimed and is used to iterate over the raw logs and unpacked data for IncentiveClaimed events raised by the FactoryContract contract.
type FactoryContractIncentiveClaimedIterator struct {
	Event *FactoryContractIncentiveClaimed // Event containing the contract specifics and raw log

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
func (it *FactoryContractIncentiveClaimedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryContractIncentiveClaimed)
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
		it.Event = new(FactoryContractIncentiveClaimed)
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
func (it *FactoryContractIncentiveClaimedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryContractIncentiveClaimedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryContractIncentiveClaimed represents a IncentiveClaimed event raised by the FactoryContract contract.
type FactoryContractIncentiveClaimed struct {
	PandaPool common.Address
	Amount    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterIncentiveClaimed is a free log retrieval operation binding the contract event 0xf383c0546bb6669c3250601085779bc0b63da3e0fb8e4881605db6fb1a9d4e82.
//
// Solidity: event IncentiveClaimed(address indexed pandaPool, uint256 amount)
func (_FactoryContract *FactoryContractFilterer) FilterIncentiveClaimed(opts *bind.FilterOpts, pandaPool []common.Address) (*FactoryContractIncentiveClaimedIterator, error) {

	var pandaPoolRule []interface{}
	for _, pandaPoolItem := range pandaPool {
		pandaPoolRule = append(pandaPoolRule, pandaPoolItem)
	}

	logs, sub, err := _FactoryContract.contract.FilterLogs(opts, "IncentiveClaimed", pandaPoolRule)
	if err != nil {
		return nil, err
	}
	return &FactoryContractIncentiveClaimedIterator{contract: _FactoryContract.contract, event: "IncentiveClaimed", logs: logs, sub: sub}, nil
}

// WatchIncentiveClaimed is a free log subscription operation binding the contract event 0xf383c0546bb6669c3250601085779bc0b63da3e0fb8e4881605db6fb1a9d4e82.
//
// Solidity: event IncentiveClaimed(address indexed pandaPool, uint256 amount)
func (_FactoryContract *FactoryContractFilterer) WatchIncentiveClaimed(opts *bind.WatchOpts, sink chan<- *FactoryContractIncentiveClaimed, pandaPool []common.Address) (event.Subscription, error) {

	var pandaPoolRule []interface{}
	for _, pandaPoolItem := range pandaPool {
		pandaPoolRule = append(pandaPoolRule, pandaPoolItem)
	}

	logs, sub, err := _FactoryContract.contract.WatchLogs(opts, "IncentiveClaimed", pandaPoolRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryContractIncentiveClaimed)
				if err := _FactoryContract.contract.UnpackLog(event, "IncentiveClaimed", log); err != nil {
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

// ParseIncentiveClaimed is a log parse operation binding the contract event 0xf383c0546bb6669c3250601085779bc0b63da3e0fb8e4881605db6fb1a9d4e82.
//
// Solidity: event IncentiveClaimed(address indexed pandaPool, uint256 amount)
func (_FactoryContract *FactoryContractFilterer) ParseIncentiveClaimed(log types.Log) (*FactoryContractIncentiveClaimed, error) {
	event := new(FactoryContractIncentiveClaimed)
	if err := _FactoryContract.contract.UnpackLog(event, "IncentiveClaimed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryContractIncentiveSetIterator is returned from FilterIncentiveSet and is used to iterate over the raw logs and unpacked data for IncentiveSet events raised by the FactoryContract contract.
type FactoryContractIncentiveSetIterator struct {
	Event *FactoryContractIncentiveSet // Event containing the contract specifics and raw log

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
func (it *FactoryContractIncentiveSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryContractIncentiveSet)
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
		it.Event = new(FactoryContractIncentiveSet)
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
func (it *FactoryContractIncentiveSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryContractIncentiveSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryContractIncentiveSet represents a IncentiveSet event raised by the FactoryContract contract.
type FactoryContractIncentiveSet struct {
	Token  common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterIncentiveSet is a free log retrieval operation binding the contract event 0xaa9255dac8b4081b70188b64255b52c5fff255d69448fa11c229de0162b82231.
//
// Solidity: event IncentiveSet(address indexed token, uint256 amount)
func (_FactoryContract *FactoryContractFilterer) FilterIncentiveSet(opts *bind.FilterOpts, token []common.Address) (*FactoryContractIncentiveSetIterator, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}

	logs, sub, err := _FactoryContract.contract.FilterLogs(opts, "IncentiveSet", tokenRule)
	if err != nil {
		return nil, err
	}
	return &FactoryContractIncentiveSetIterator{contract: _FactoryContract.contract, event: "IncentiveSet", logs: logs, sub: sub}, nil
}

// WatchIncentiveSet is a free log subscription operation binding the contract event 0xaa9255dac8b4081b70188b64255b52c5fff255d69448fa11c229de0162b82231.
//
// Solidity: event IncentiveSet(address indexed token, uint256 amount)
func (_FactoryContract *FactoryContractFilterer) WatchIncentiveSet(opts *bind.WatchOpts, sink chan<- *FactoryContractIncentiveSet, token []common.Address) (event.Subscription, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}

	logs, sub, err := _FactoryContract.contract.WatchLogs(opts, "IncentiveSet", tokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryContractIncentiveSet)
				if err := _FactoryContract.contract.UnpackLog(event, "IncentiveSet", log); err != nil {
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

// ParseIncentiveSet is a log parse operation binding the contract event 0xaa9255dac8b4081b70188b64255b52c5fff255d69448fa11c229de0162b82231.
//
// Solidity: event IncentiveSet(address indexed token, uint256 amount)
func (_FactoryContract *FactoryContractFilterer) ParseIncentiveSet(log types.Log) (*FactoryContractIncentiveSet, error) {
	event := new(FactoryContractIncentiveSet)
	if err := _FactoryContract.contract.UnpackLog(event, "IncentiveSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryContractMinRaiseSetIterator is returned from FilterMinRaiseSet and is used to iterate over the raw logs and unpacked data for MinRaiseSet events raised by the FactoryContract contract.
type FactoryContractMinRaiseSetIterator struct {
	Event *FactoryContractMinRaiseSet // Event containing the contract specifics and raw log

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
func (it *FactoryContractMinRaiseSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryContractMinRaiseSet)
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
		it.Event = new(FactoryContractMinRaiseSet)
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
func (it *FactoryContractMinRaiseSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryContractMinRaiseSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryContractMinRaiseSet represents a MinRaiseSet event raised by the FactoryContract contract.
type FactoryContractMinRaiseSet struct {
	BaseToken   common.Address
	MinEndPrice *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterMinRaiseSet is a free log retrieval operation binding the contract event 0x4b09097f0f737584174ffbbe48feb8f302a2c11c5a576b8b4951af01694bfeac.
//
// Solidity: event MinRaiseSet(address indexed baseToken, uint256 minEndPrice)
func (_FactoryContract *FactoryContractFilterer) FilterMinRaiseSet(opts *bind.FilterOpts, baseToken []common.Address) (*FactoryContractMinRaiseSetIterator, error) {

	var baseTokenRule []interface{}
	for _, baseTokenItem := range baseToken {
		baseTokenRule = append(baseTokenRule, baseTokenItem)
	}

	logs, sub, err := _FactoryContract.contract.FilterLogs(opts, "MinRaiseSet", baseTokenRule)
	if err != nil {
		return nil, err
	}
	return &FactoryContractMinRaiseSetIterator{contract: _FactoryContract.contract, event: "MinRaiseSet", logs: logs, sub: sub}, nil
}

// WatchMinRaiseSet is a free log subscription operation binding the contract event 0x4b09097f0f737584174ffbbe48feb8f302a2c11c5a576b8b4951af01694bfeac.
//
// Solidity: event MinRaiseSet(address indexed baseToken, uint256 minEndPrice)
func (_FactoryContract *FactoryContractFilterer) WatchMinRaiseSet(opts *bind.WatchOpts, sink chan<- *FactoryContractMinRaiseSet, baseToken []common.Address) (event.Subscription, error) {

	var baseTokenRule []interface{}
	for _, baseTokenItem := range baseToken {
		baseTokenRule = append(baseTokenRule, baseTokenItem)
	}

	logs, sub, err := _FactoryContract.contract.WatchLogs(opts, "MinRaiseSet", baseTokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryContractMinRaiseSet)
				if err := _FactoryContract.contract.UnpackLog(event, "MinRaiseSet", log); err != nil {
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

// ParseMinRaiseSet is a log parse operation binding the contract event 0x4b09097f0f737584174ffbbe48feb8f302a2c11c5a576b8b4951af01694bfeac.
//
// Solidity: event MinRaiseSet(address indexed baseToken, uint256 minEndPrice)
func (_FactoryContract *FactoryContractFilterer) ParseMinRaiseSet(log types.Log) (*FactoryContractMinRaiseSet, error) {
	event := new(FactoryContractMinRaiseSet)
	if err := _FactoryContract.contract.UnpackLog(event, "MinRaiseSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryContractMinTradeSizeSetIterator is returned from FilterMinTradeSizeSet and is used to iterate over the raw logs and unpacked data for MinTradeSizeSet events raised by the FactoryContract contract.
type FactoryContractMinTradeSizeSetIterator struct {
	Event *FactoryContractMinTradeSizeSet // Event containing the contract specifics and raw log

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
func (it *FactoryContractMinTradeSizeSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryContractMinTradeSizeSet)
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
		it.Event = new(FactoryContractMinTradeSizeSet)
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
func (it *FactoryContractMinTradeSizeSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryContractMinTradeSizeSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryContractMinTradeSizeSet represents a MinTradeSizeSet event raised by the FactoryContract contract.
type FactoryContractMinTradeSizeSet struct {
	BaseToken    common.Address
	MinTradeSize *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterMinTradeSizeSet is a free log retrieval operation binding the contract event 0x1f2e1e307c3f45252af2c75628b94beb277b891400fec0c43665374d6e97bb1a.
//
// Solidity: event MinTradeSizeSet(address indexed baseToken, uint256 minTradeSize)
func (_FactoryContract *FactoryContractFilterer) FilterMinTradeSizeSet(opts *bind.FilterOpts, baseToken []common.Address) (*FactoryContractMinTradeSizeSetIterator, error) {

	var baseTokenRule []interface{}
	for _, baseTokenItem := range baseToken {
		baseTokenRule = append(baseTokenRule, baseTokenItem)
	}

	logs, sub, err := _FactoryContract.contract.FilterLogs(opts, "MinTradeSizeSet", baseTokenRule)
	if err != nil {
		return nil, err
	}
	return &FactoryContractMinTradeSizeSetIterator{contract: _FactoryContract.contract, event: "MinTradeSizeSet", logs: logs, sub: sub}, nil
}

// WatchMinTradeSizeSet is a free log subscription operation binding the contract event 0x1f2e1e307c3f45252af2c75628b94beb277b891400fec0c43665374d6e97bb1a.
//
// Solidity: event MinTradeSizeSet(address indexed baseToken, uint256 minTradeSize)
func (_FactoryContract *FactoryContractFilterer) WatchMinTradeSizeSet(opts *bind.WatchOpts, sink chan<- *FactoryContractMinTradeSizeSet, baseToken []common.Address) (event.Subscription, error) {

	var baseTokenRule []interface{}
	for _, baseTokenItem := range baseToken {
		baseTokenRule = append(baseTokenRule, baseTokenItem)
	}

	logs, sub, err := _FactoryContract.contract.WatchLogs(opts, "MinTradeSizeSet", baseTokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryContractMinTradeSizeSet)
				if err := _FactoryContract.contract.UnpackLog(event, "MinTradeSizeSet", log); err != nil {
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

// ParseMinTradeSizeSet is a log parse operation binding the contract event 0x1f2e1e307c3f45252af2c75628b94beb277b891400fec0c43665374d6e97bb1a.
//
// Solidity: event MinTradeSizeSet(address indexed baseToken, uint256 minTradeSize)
func (_FactoryContract *FactoryContractFilterer) ParseMinTradeSizeSet(log types.Log) (*FactoryContractMinTradeSizeSet, error) {
	event := new(FactoryContractMinTradeSizeSet)
	if err := _FactoryContract.contract.UnpackLog(event, "MinTradeSizeSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryContractOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the FactoryContract contract.
type FactoryContractOwnershipTransferredIterator struct {
	Event *FactoryContractOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *FactoryContractOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryContractOwnershipTransferred)
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
		it.Event = new(FactoryContractOwnershipTransferred)
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
func (it *FactoryContractOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryContractOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryContractOwnershipTransferred represents a OwnershipTransferred event raised by the FactoryContract contract.
type FactoryContractOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_FactoryContract *FactoryContractFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*FactoryContractOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _FactoryContract.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &FactoryContractOwnershipTransferredIterator{contract: _FactoryContract.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_FactoryContract *FactoryContractFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *FactoryContractOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _FactoryContract.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryContractOwnershipTransferred)
				if err := _FactoryContract.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_FactoryContract *FactoryContractFilterer) ParseOwnershipTransferred(log types.Log) (*FactoryContractOwnershipTransferred, error) {
	event := new(FactoryContractOwnershipTransferred)
	if err := _FactoryContract.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryContractPandaDeployedIterator is returned from FilterPandaDeployed and is used to iterate over the raw logs and unpacked data for PandaDeployed events raised by the FactoryContract contract.
type FactoryContractPandaDeployedIterator struct {
	Event *FactoryContractPandaDeployed // Event containing the contract specifics and raw log

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
func (it *FactoryContractPandaDeployedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryContractPandaDeployed)
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
		it.Event = new(FactoryContractPandaDeployed)
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
func (it *FactoryContractPandaDeployedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryContractPandaDeployedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryContractPandaDeployed represents a PandaDeployed event raised by the FactoryContract contract.
type FactoryContractPandaDeployed struct {
	PandaPool      common.Address
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterPandaDeployed is a free log retrieval operation binding the contract event 0xba7af04a26315a022cff47a81c16f0e8b2762aca49ff74fdc6722f543eec3ee2.
//
// Solidity: event PandaDeployed(address indexed pandaPool, address indexed implementation)
func (_FactoryContract *FactoryContractFilterer) FilterPandaDeployed(opts *bind.FilterOpts, pandaPool []common.Address, implementation []common.Address) (*FactoryContractPandaDeployedIterator, error) {

	var pandaPoolRule []interface{}
	for _, pandaPoolItem := range pandaPool {
		pandaPoolRule = append(pandaPoolRule, pandaPoolItem)
	}
	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _FactoryContract.contract.FilterLogs(opts, "PandaDeployed", pandaPoolRule, implementationRule)
	if err != nil {
		return nil, err
	}
	return &FactoryContractPandaDeployedIterator{contract: _FactoryContract.contract, event: "PandaDeployed", logs: logs, sub: sub}, nil
}

// WatchPandaDeployed is a free log subscription operation binding the contract event 0xba7af04a26315a022cff47a81c16f0e8b2762aca49ff74fdc6722f543eec3ee2.
//
// Solidity: event PandaDeployed(address indexed pandaPool, address indexed implementation)
func (_FactoryContract *FactoryContractFilterer) WatchPandaDeployed(opts *bind.WatchOpts, sink chan<- *FactoryContractPandaDeployed, pandaPool []common.Address, implementation []common.Address) (event.Subscription, error) {

	var pandaPoolRule []interface{}
	for _, pandaPoolItem := range pandaPool {
		pandaPoolRule = append(pandaPoolRule, pandaPoolItem)
	}
	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _FactoryContract.contract.WatchLogs(opts, "PandaDeployed", pandaPoolRule, implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryContractPandaDeployed)
				if err := _FactoryContract.contract.UnpackLog(event, "PandaDeployed", log); err != nil {
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

// ParsePandaDeployed is a log parse operation binding the contract event 0xba7af04a26315a022cff47a81c16f0e8b2762aca49ff74fdc6722f543eec3ee2.
//
// Solidity: event PandaDeployed(address indexed pandaPool, address indexed implementation)
func (_FactoryContract *FactoryContractFilterer) ParsePandaDeployed(log types.Log) (*FactoryContractPandaDeployed, error) {
	event := new(FactoryContractPandaDeployed)
	if err := _FactoryContract.contract.UnpackLog(event, "PandaDeployed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryContractPandaPoolFeesSetIterator is returned from FilterPandaPoolFeesSet and is used to iterate over the raw logs and unpacked data for PandaPoolFeesSet events raised by the FactoryContract contract.
type FactoryContractPandaPoolFeesSetIterator struct {
	Event *FactoryContractPandaPoolFeesSet // Event containing the contract specifics and raw log

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
func (it *FactoryContractPandaPoolFeesSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryContractPandaPoolFeesSet)
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
		it.Event = new(FactoryContractPandaPoolFeesSet)
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
func (it *FactoryContractPandaPoolFeesSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryContractPandaPoolFeesSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryContractPandaPoolFeesSet represents a PandaPoolFeesSet event raised by the FactoryContract contract.
type FactoryContractPandaPoolFeesSet struct {
	BuyFee           uint16
	SellFee          uint16
	GraduationFee    uint16
	DeployerFeeShare uint16
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterPandaPoolFeesSet is a free log retrieval operation binding the contract event 0x58d7aaf8ba19daf3e0f75422a6740ce19f413c120df7e261514abffe3c808c3e.
//
// Solidity: event PandaPoolFeesSet(uint16 buyFee, uint16 sellFee, uint16 graduationFee, uint16 deployerFeeShare)
func (_FactoryContract *FactoryContractFilterer) FilterPandaPoolFeesSet(opts *bind.FilterOpts) (*FactoryContractPandaPoolFeesSetIterator, error) {

	logs, sub, err := _FactoryContract.contract.FilterLogs(opts, "PandaPoolFeesSet")
	if err != nil {
		return nil, err
	}
	return &FactoryContractPandaPoolFeesSetIterator{contract: _FactoryContract.contract, event: "PandaPoolFeesSet", logs: logs, sub: sub}, nil
}

// WatchPandaPoolFeesSet is a free log subscription operation binding the contract event 0x58d7aaf8ba19daf3e0f75422a6740ce19f413c120df7e261514abffe3c808c3e.
//
// Solidity: event PandaPoolFeesSet(uint16 buyFee, uint16 sellFee, uint16 graduationFee, uint16 deployerFeeShare)
func (_FactoryContract *FactoryContractFilterer) WatchPandaPoolFeesSet(opts *bind.WatchOpts, sink chan<- *FactoryContractPandaPoolFeesSet) (event.Subscription, error) {

	logs, sub, err := _FactoryContract.contract.WatchLogs(opts, "PandaPoolFeesSet")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryContractPandaPoolFeesSet)
				if err := _FactoryContract.contract.UnpackLog(event, "PandaPoolFeesSet", log); err != nil {
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

// ParsePandaPoolFeesSet is a log parse operation binding the contract event 0x58d7aaf8ba19daf3e0f75422a6740ce19f413c120df7e261514abffe3c808c3e.
//
// Solidity: event PandaPoolFeesSet(uint16 buyFee, uint16 sellFee, uint16 graduationFee, uint16 deployerFeeShare)
func (_FactoryContract *FactoryContractFilterer) ParsePandaPoolFeesSet(log types.Log) (*FactoryContractPandaPoolFeesSet, error) {
	event := new(FactoryContractPandaPoolFeesSet)
	if err := _FactoryContract.contract.UnpackLog(event, "PandaPoolFeesSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryContractTreasurySetIterator is returned from FilterTreasurySet and is used to iterate over the raw logs and unpacked data for TreasurySet events raised by the FactoryContract contract.
type FactoryContractTreasurySetIterator struct {
	Event *FactoryContractTreasurySet // Event containing the contract specifics and raw log

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
func (it *FactoryContractTreasurySetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryContractTreasurySet)
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
		it.Event = new(FactoryContractTreasurySet)
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
func (it *FactoryContractTreasurySetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryContractTreasurySetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryContractTreasurySet represents a TreasurySet event raised by the FactoryContract contract.
type FactoryContractTreasurySet struct {
	Treasury common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterTreasurySet is a free log retrieval operation binding the contract event 0x3c864541ef71378c6229510ed90f376565ee42d9c5e0904a984a9e863e6db44f.
//
// Solidity: event TreasurySet(address indexed treasury)
func (_FactoryContract *FactoryContractFilterer) FilterTreasurySet(opts *bind.FilterOpts, treasury []common.Address) (*FactoryContractTreasurySetIterator, error) {

	var treasuryRule []interface{}
	for _, treasuryItem := range treasury {
		treasuryRule = append(treasuryRule, treasuryItem)
	}

	logs, sub, err := _FactoryContract.contract.FilterLogs(opts, "TreasurySet", treasuryRule)
	if err != nil {
		return nil, err
	}
	return &FactoryContractTreasurySetIterator{contract: _FactoryContract.contract, event: "TreasurySet", logs: logs, sub: sub}, nil
}

// WatchTreasurySet is a free log subscription operation binding the contract event 0x3c864541ef71378c6229510ed90f376565ee42d9c5e0904a984a9e863e6db44f.
//
// Solidity: event TreasurySet(address indexed treasury)
func (_FactoryContract *FactoryContractFilterer) WatchTreasurySet(opts *bind.WatchOpts, sink chan<- *FactoryContractTreasurySet, treasury []common.Address) (event.Subscription, error) {

	var treasuryRule []interface{}
	for _, treasuryItem := range treasury {
		treasuryRule = append(treasuryRule, treasuryItem)
	}

	logs, sub, err := _FactoryContract.contract.WatchLogs(opts, "TreasurySet", treasuryRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryContractTreasurySet)
				if err := _FactoryContract.contract.UnpackLog(event, "TreasurySet", log); err != nil {
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

// ParseTreasurySet is a log parse operation binding the contract event 0x3c864541ef71378c6229510ed90f376565ee42d9c5e0904a984a9e863e6db44f.
//
// Solidity: event TreasurySet(address indexed treasury)
func (_FactoryContract *FactoryContractFilterer) ParseTreasurySet(log types.Log) (*FactoryContractTreasurySet, error) {
	event := new(FactoryContractTreasurySet)
	if err := _FactoryContract.contract.UnpackLog(event, "TreasurySet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FactoryContractWberaSetIterator is returned from FilterWberaSet and is used to iterate over the raw logs and unpacked data for WberaSet events raised by the FactoryContract contract.
type FactoryContractWberaSetIterator struct {
	Event *FactoryContractWberaSet // Event containing the contract specifics and raw log

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
func (it *FactoryContractWberaSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FactoryContractWberaSet)
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
		it.Event = new(FactoryContractWberaSet)
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
func (it *FactoryContractWberaSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FactoryContractWberaSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FactoryContractWberaSet represents a WberaSet event raised by the FactoryContract contract.
type FactoryContractWberaSet struct {
	Wbera common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterWberaSet is a free log retrieval operation binding the contract event 0xb9715cec0f54e0d4ebd68cf0a8943c128514bdae25921462c4acad0d8903f3ec.
//
// Solidity: event WberaSet(address indexed wbera)
func (_FactoryContract *FactoryContractFilterer) FilterWberaSet(opts *bind.FilterOpts, wbera []common.Address) (*FactoryContractWberaSetIterator, error) {

	var wberaRule []interface{}
	for _, wberaItem := range wbera {
		wberaRule = append(wberaRule, wberaItem)
	}

	logs, sub, err := _FactoryContract.contract.FilterLogs(opts, "WberaSet", wberaRule)
	if err != nil {
		return nil, err
	}
	return &FactoryContractWberaSetIterator{contract: _FactoryContract.contract, event: "WberaSet", logs: logs, sub: sub}, nil
}

// WatchWberaSet is a free log subscription operation binding the contract event 0xb9715cec0f54e0d4ebd68cf0a8943c128514bdae25921462c4acad0d8903f3ec.
//
// Solidity: event WberaSet(address indexed wbera)
func (_FactoryContract *FactoryContractFilterer) WatchWberaSet(opts *bind.WatchOpts, sink chan<- *FactoryContractWberaSet, wbera []common.Address) (event.Subscription, error) {

	var wberaRule []interface{}
	for _, wberaItem := range wbera {
		wberaRule = append(wberaRule, wberaItem)
	}

	logs, sub, err := _FactoryContract.contract.WatchLogs(opts, "WberaSet", wberaRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FactoryContractWberaSet)
				if err := _FactoryContract.contract.UnpackLog(event, "WberaSet", log); err != nil {
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

// ParseWberaSet is a log parse operation binding the contract event 0xb9715cec0f54e0d4ebd68cf0a8943c128514bdae25921462c4acad0d8903f3ec.
//
// Solidity: event WberaSet(address indexed wbera)
func (_FactoryContract *FactoryContractFilterer) ParseWberaSet(log types.Log) (*FactoryContractWberaSet, error) {
	event := new(FactoryContractWberaSet)
	if err := _FactoryContract.contract.UnpackLog(event, "WberaSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
