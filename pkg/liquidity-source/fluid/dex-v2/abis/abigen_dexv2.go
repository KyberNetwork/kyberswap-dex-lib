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

// DexKey is an auto generated low-level Go binding around an user-defined struct.
type DexKey struct {
	Token0      common.Address
	Token1      common.Address
	Fee         *big.Int
	TickSpacing *big.Int
	Controller  common.Address
}

// FluidDexV2MetaData contains all meta data concerning the FluidDexV2 contract.
var FluidDexV2MetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"dexType\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"dexId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"positionSalt\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"feeAccruedToken0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"feeAccruedToken1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"liquidityIncreaseRaw\",\"type\":\"uint256\"}],\"name\":\"LogDeposit\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"dexType\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"dexId\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"tickSpacing\",\"type\":\"uint24\"},{\"internalType\":\"address\",\"name\":\"controller\",\"type\":\"address\"}],\"indexed\":false,\"internalType\":\"structDexKey\",\"name\":\"dexKey\",\"type\":\"tuple\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"sqrtPriceX96\",\"type\":\"uint256\"}],\"name\":\"LogInitialize\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"dexType\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"dexId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"positionSalt\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"feeAccruedToken0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"feeAccruedToken1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"liquidityDecreaseRaw\",\"type\":\"uint256\"}],\"name\":\"LogWithdraw\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"dexType\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"dexId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"positionSalt\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"feeAccruedToken0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"feeAccruedToken1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"liquidityIncreaseRaw\",\"type\":\"uint256\"}],\"name\":\"LogBorrow\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"dexType\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"dexId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"positionSalt\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"feeAccruedToken0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"feeAccruedToken1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"liquidityDecreaseRaw\",\"type\":\"uint256\"}],\"name\":\"LogPayback\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"liquidity_\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"errorId_\",\"type\":\"uint256\"}],\"name\":\"FluidDexV2Error\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"errorId_\",\"type\":\"uint256\"}],\"name\":\"FluidSafeTransferError\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"amount\",\"type\":\"int256\"}],\"name\":\"LogAddOrRemoveTokens\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"dexType\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"implementationId\",\"type\":\"uint256\"}],\"name\":\"LogOperate\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"dexType\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"implementationId\",\"type\":\"uint256\"}],\"name\":\"LogOperateAdmin\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"supplyAmount\",\"type\":\"int256\"},{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"borrowAmount\",\"type\":\"int256\"}],\"name\":\"LogRebalance\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"supplyAmount\",\"type\":\"int256\"},{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"borrowAmount\",\"type\":\"int256\"},{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"storeAmount\",\"type\":\"int256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"LogSettle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"auth\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bool\",\"name\":\"isAuth\",\"type\":\"bool\"}],\"name\":\"LogUpdateAuth\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"dexType\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"adminImplementationId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"adminImplementation\",\"type\":\"address\"}],\"name\":\"LogUpdateDexTypeToAdminImplementation\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"LogUpgraded\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token_\",\"type\":\"address\"},{\"internalType\":\"int256\",\"name\":\"amount_\",\"type\":\"int256\"}],\"name\":\"addOrRemoveTokens\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token_\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount_\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data_\",\"type\":\"bytes\"}],\"name\":\"liquidityCallback\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"dexType_\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"implementationId_\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data_\",\"type\":\"bytes\"}],\"name\":\"operate\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"returnData_\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"dexType_\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"implementationId_\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data_\",\"type\":\"bytes\"}],\"name\":\"operateAdmin\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"returnData_\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proxiableUUID\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"slot_\",\"type\":\"bytes32\"}],\"name\":\"readFromStorage\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"result_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"slot_\",\"type\":\"bytes32\"}],\"name\":\"readFromTransientStorage\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"result_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token_\",\"type\":\"address\"}],\"name\":\"rebalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token_\",\"type\":\"address\"},{\"internalType\":\"int256\",\"name\":\"supplyAmount_\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"borrowAmount_\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"storeAmount_\",\"type\":\"int256\"},{\"internalType\":\"address\",\"name\":\"to_\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isCallback_\",\"type\":\"bool\"}],\"name\":\"settle\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data_\",\"type\":\"bytes\"}],\"name\":\"startOperation\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"result_\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"auth_\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isAuth_\",\"type\":\"bool\"}],\"name\":\"updateAuth\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"dexType_\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"adminImplementationId_\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"adminImplementation_\",\"type\":\"address\"}],\"name\":\"updateDexTypeToAdminImplementation\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newImplementation_\",\"type\":\"address\"}],\"name\":\"upgradeTo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newImplementation_\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data_\",\"type\":\"bytes\"}],\"name\":\"upgradeToAndCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
}

// FluidDexV2ABI is the input ABI used to generate the binding from.
// Deprecated: Use FluidDexV2MetaData.ABI instead.
var FluidDexV2ABI = FluidDexV2MetaData.ABI

// FluidDexV2 is an auto generated Go binding around an Ethereum contract.
type FluidDexV2 struct {
	FluidDexV2Caller     // Read-only binding to the contract
	FluidDexV2Transactor // Write-only binding to the contract
	FluidDexV2Filterer   // Log filterer for contract events
}

// FluidDexV2Caller is an auto generated read-only Go binding around an Ethereum contract.
type FluidDexV2Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FluidDexV2Transactor is an auto generated write-only Go binding around an Ethereum contract.
type FluidDexV2Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FluidDexV2Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type FluidDexV2Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FluidDexV2Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type FluidDexV2Session struct {
	Contract     *FluidDexV2       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// FluidDexV2CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type FluidDexV2CallerSession struct {
	Contract *FluidDexV2Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// FluidDexV2TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type FluidDexV2TransactorSession struct {
	Contract     *FluidDexV2Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// FluidDexV2Raw is an auto generated low-level Go binding around an Ethereum contract.
type FluidDexV2Raw struct {
	Contract *FluidDexV2 // Generic contract binding to access the raw methods on
}

// FluidDexV2CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type FluidDexV2CallerRaw struct {
	Contract *FluidDexV2Caller // Generic read-only contract binding to access the raw methods on
}

// FluidDexV2TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type FluidDexV2TransactorRaw struct {
	Contract *FluidDexV2Transactor // Generic write-only contract binding to access the raw methods on
}

// NewFluidDexV2 creates a new instance of FluidDexV2, bound to a specific deployed contract.
func NewFluidDexV2(address common.Address, backend bind.ContractBackend) (*FluidDexV2, error) {
	contract, err := bindFluidDexV2(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &FluidDexV2{FluidDexV2Caller: FluidDexV2Caller{contract: contract}, FluidDexV2Transactor: FluidDexV2Transactor{contract: contract}, FluidDexV2Filterer: FluidDexV2Filterer{contract: contract}}, nil
}

// NewFluidDexV2Caller creates a new read-only instance of FluidDexV2, bound to a specific deployed contract.
func NewFluidDexV2Caller(address common.Address, caller bind.ContractCaller) (*FluidDexV2Caller, error) {
	contract, err := bindFluidDexV2(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &FluidDexV2Caller{contract: contract}, nil
}

// NewFluidDexV2Transactor creates a new write-only instance of FluidDexV2, bound to a specific deployed contract.
func NewFluidDexV2Transactor(address common.Address, transactor bind.ContractTransactor) (*FluidDexV2Transactor, error) {
	contract, err := bindFluidDexV2(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &FluidDexV2Transactor{contract: contract}, nil
}

// NewFluidDexV2Filterer creates a new log filterer instance of FluidDexV2, bound to a specific deployed contract.
func NewFluidDexV2Filterer(address common.Address, filterer bind.ContractFilterer) (*FluidDexV2Filterer, error) {
	contract, err := bindFluidDexV2(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &FluidDexV2Filterer{contract: contract}, nil
}

// bindFluidDexV2 binds a generic wrapper to an already deployed contract.
func bindFluidDexV2(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := FluidDexV2MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FluidDexV2 *FluidDexV2Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FluidDexV2.Contract.FluidDexV2Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FluidDexV2 *FluidDexV2Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FluidDexV2.Contract.FluidDexV2Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FluidDexV2 *FluidDexV2Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FluidDexV2.Contract.FluidDexV2Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FluidDexV2 *FluidDexV2CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FluidDexV2.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FluidDexV2 *FluidDexV2TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FluidDexV2.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FluidDexV2 *FluidDexV2TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FluidDexV2.Contract.contract.Transact(opts, method, params...)
}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() pure returns(bytes32)
func (_FluidDexV2 *FluidDexV2Caller) ProxiableUUID(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _FluidDexV2.contract.Call(opts, &out, "proxiableUUID")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() pure returns(bytes32)
func (_FluidDexV2 *FluidDexV2Session) ProxiableUUID() ([32]byte, error) {
	return _FluidDexV2.Contract.ProxiableUUID(&_FluidDexV2.CallOpts)
}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() pure returns(bytes32)
func (_FluidDexV2 *FluidDexV2CallerSession) ProxiableUUID() ([32]byte, error) {
	return _FluidDexV2.Contract.ProxiableUUID(&_FluidDexV2.CallOpts)
}

// ReadFromStorage is a free data retrieval call binding the contract method 0xb5c736e4.
//
// Solidity: function readFromStorage(bytes32 slot_) view returns(uint256 result_)
func (_FluidDexV2 *FluidDexV2Caller) ReadFromStorage(opts *bind.CallOpts, slot_ [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _FluidDexV2.contract.Call(opts, &out, "readFromStorage", slot_)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ReadFromStorage is a free data retrieval call binding the contract method 0xb5c736e4.
//
// Solidity: function readFromStorage(bytes32 slot_) view returns(uint256 result_)
func (_FluidDexV2 *FluidDexV2Session) ReadFromStorage(slot_ [32]byte) (*big.Int, error) {
	return _FluidDexV2.Contract.ReadFromStorage(&_FluidDexV2.CallOpts, slot_)
}

// ReadFromStorage is a free data retrieval call binding the contract method 0xb5c736e4.
//
// Solidity: function readFromStorage(bytes32 slot_) view returns(uint256 result_)
func (_FluidDexV2 *FluidDexV2CallerSession) ReadFromStorage(slot_ [32]byte) (*big.Int, error) {
	return _FluidDexV2.Contract.ReadFromStorage(&_FluidDexV2.CallOpts, slot_)
}

// ReadFromTransientStorage is a free data retrieval call binding the contract method 0x11c5f016.
//
// Solidity: function readFromTransientStorage(bytes32 slot_) view returns(uint256 result_)
func (_FluidDexV2 *FluidDexV2Caller) ReadFromTransientStorage(opts *bind.CallOpts, slot_ [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _FluidDexV2.contract.Call(opts, &out, "readFromTransientStorage", slot_)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ReadFromTransientStorage is a free data retrieval call binding the contract method 0x11c5f016.
//
// Solidity: function readFromTransientStorage(bytes32 slot_) view returns(uint256 result_)
func (_FluidDexV2 *FluidDexV2Session) ReadFromTransientStorage(slot_ [32]byte) (*big.Int, error) {
	return _FluidDexV2.Contract.ReadFromTransientStorage(&_FluidDexV2.CallOpts, slot_)
}

// ReadFromTransientStorage is a free data retrieval call binding the contract method 0x11c5f016.
//
// Solidity: function readFromTransientStorage(bytes32 slot_) view returns(uint256 result_)
func (_FluidDexV2 *FluidDexV2CallerSession) ReadFromTransientStorage(slot_ [32]byte) (*big.Int, error) {
	return _FluidDexV2.Contract.ReadFromTransientStorage(&_FluidDexV2.CallOpts, slot_)
}

// AddOrRemoveTokens is a paid mutator transaction binding the contract method 0x779fd5d8.
//
// Solidity: function addOrRemoveTokens(address token_, int256 amount_) payable returns()
func (_FluidDexV2 *FluidDexV2Transactor) AddOrRemoveTokens(opts *bind.TransactOpts, token_ common.Address, amount_ *big.Int) (*types.Transaction, error) {
	return _FluidDexV2.contract.Transact(opts, "addOrRemoveTokens", token_, amount_)
}

// AddOrRemoveTokens is a paid mutator transaction binding the contract method 0x779fd5d8.
//
// Solidity: function addOrRemoveTokens(address token_, int256 amount_) payable returns()
func (_FluidDexV2 *FluidDexV2Session) AddOrRemoveTokens(token_ common.Address, amount_ *big.Int) (*types.Transaction, error) {
	return _FluidDexV2.Contract.AddOrRemoveTokens(&_FluidDexV2.TransactOpts, token_, amount_)
}

// AddOrRemoveTokens is a paid mutator transaction binding the contract method 0x779fd5d8.
//
// Solidity: function addOrRemoveTokens(address token_, int256 amount_) payable returns()
func (_FluidDexV2 *FluidDexV2TransactorSession) AddOrRemoveTokens(token_ common.Address, amount_ *big.Int) (*types.Transaction, error) {
	return _FluidDexV2.Contract.AddOrRemoveTokens(&_FluidDexV2.TransactOpts, token_, amount_)
}

// LiquidityCallback is a paid mutator transaction binding the contract method 0xad207501.
//
// Solidity: function liquidityCallback(address token_, uint256 amount_, bytes data_) returns()
func (_FluidDexV2 *FluidDexV2Transactor) LiquidityCallback(opts *bind.TransactOpts, token_ common.Address, amount_ *big.Int, data_ []byte) (*types.Transaction, error) {
	return _FluidDexV2.contract.Transact(opts, "liquidityCallback", token_, amount_, data_)
}

// LiquidityCallback is a paid mutator transaction binding the contract method 0xad207501.
//
// Solidity: function liquidityCallback(address token_, uint256 amount_, bytes data_) returns()
func (_FluidDexV2 *FluidDexV2Session) LiquidityCallback(token_ common.Address, amount_ *big.Int, data_ []byte) (*types.Transaction, error) {
	return _FluidDexV2.Contract.LiquidityCallback(&_FluidDexV2.TransactOpts, token_, amount_, data_)
}

// LiquidityCallback is a paid mutator transaction binding the contract method 0xad207501.
//
// Solidity: function liquidityCallback(address token_, uint256 amount_, bytes data_) returns()
func (_FluidDexV2 *FluidDexV2TransactorSession) LiquidityCallback(token_ common.Address, amount_ *big.Int, data_ []byte) (*types.Transaction, error) {
	return _FluidDexV2.Contract.LiquidityCallback(&_FluidDexV2.TransactOpts, token_, amount_, data_)
}

// Operate is a paid mutator transaction binding the contract method 0xbfa5a352.
//
// Solidity: function operate(uint256 dexType_, uint256 implementationId_, bytes data_) returns(bytes returnData_)
func (_FluidDexV2 *FluidDexV2Transactor) Operate(opts *bind.TransactOpts, dexType_ *big.Int, implementationId_ *big.Int, data_ []byte) (*types.Transaction, error) {
	return _FluidDexV2.contract.Transact(opts, "operate", dexType_, implementationId_, data_)
}

// Operate is a paid mutator transaction binding the contract method 0xbfa5a352.
//
// Solidity: function operate(uint256 dexType_, uint256 implementationId_, bytes data_) returns(bytes returnData_)
func (_FluidDexV2 *FluidDexV2Session) Operate(dexType_ *big.Int, implementationId_ *big.Int, data_ []byte) (*types.Transaction, error) {
	return _FluidDexV2.Contract.Operate(&_FluidDexV2.TransactOpts, dexType_, implementationId_, data_)
}

// Operate is a paid mutator transaction binding the contract method 0xbfa5a352.
//
// Solidity: function operate(uint256 dexType_, uint256 implementationId_, bytes data_) returns(bytes returnData_)
func (_FluidDexV2 *FluidDexV2TransactorSession) Operate(dexType_ *big.Int, implementationId_ *big.Int, data_ []byte) (*types.Transaction, error) {
	return _FluidDexV2.Contract.Operate(&_FluidDexV2.TransactOpts, dexType_, implementationId_, data_)
}

// OperateAdmin is a paid mutator transaction binding the contract method 0xc9046d88.
//
// Solidity: function operateAdmin(uint256 dexType_, uint256 implementationId_, bytes data_) returns(bytes returnData_)
func (_FluidDexV2 *FluidDexV2Transactor) OperateAdmin(opts *bind.TransactOpts, dexType_ *big.Int, implementationId_ *big.Int, data_ []byte) (*types.Transaction, error) {
	return _FluidDexV2.contract.Transact(opts, "operateAdmin", dexType_, implementationId_, data_)
}

// OperateAdmin is a paid mutator transaction binding the contract method 0xc9046d88.
//
// Solidity: function operateAdmin(uint256 dexType_, uint256 implementationId_, bytes data_) returns(bytes returnData_)
func (_FluidDexV2 *FluidDexV2Session) OperateAdmin(dexType_ *big.Int, implementationId_ *big.Int, data_ []byte) (*types.Transaction, error) {
	return _FluidDexV2.Contract.OperateAdmin(&_FluidDexV2.TransactOpts, dexType_, implementationId_, data_)
}

// OperateAdmin is a paid mutator transaction binding the contract method 0xc9046d88.
//
// Solidity: function operateAdmin(uint256 dexType_, uint256 implementationId_, bytes data_) returns(bytes returnData_)
func (_FluidDexV2 *FluidDexV2TransactorSession) OperateAdmin(dexType_ *big.Int, implementationId_ *big.Int, data_ []byte) (*types.Transaction, error) {
	return _FluidDexV2.Contract.OperateAdmin(&_FluidDexV2.TransactOpts, dexType_, implementationId_, data_)
}

// Rebalance is a paid mutator transaction binding the contract method 0x21c28191.
//
// Solidity: function rebalance(address token_) returns()
func (_FluidDexV2 *FluidDexV2Transactor) Rebalance(opts *bind.TransactOpts, token_ common.Address) (*types.Transaction, error) {
	return _FluidDexV2.contract.Transact(opts, "rebalance", token_)
}

// Rebalance is a paid mutator transaction binding the contract method 0x21c28191.
//
// Solidity: function rebalance(address token_) returns()
func (_FluidDexV2 *FluidDexV2Session) Rebalance(token_ common.Address) (*types.Transaction, error) {
	return _FluidDexV2.Contract.Rebalance(&_FluidDexV2.TransactOpts, token_)
}

// Rebalance is a paid mutator transaction binding the contract method 0x21c28191.
//
// Solidity: function rebalance(address token_) returns()
func (_FluidDexV2 *FluidDexV2TransactorSession) Rebalance(token_ common.Address) (*types.Transaction, error) {
	return _FluidDexV2.Contract.Rebalance(&_FluidDexV2.TransactOpts, token_)
}

// Settle is a paid mutator transaction binding the contract method 0x5aa62d1b.
//
// Solidity: function settle(address token_, int256 supplyAmount_, int256 borrowAmount_, int256 storeAmount_, address to_, bool isCallback_) payable returns()
func (_FluidDexV2 *FluidDexV2Transactor) Settle(opts *bind.TransactOpts, token_ common.Address, supplyAmount_ *big.Int, borrowAmount_ *big.Int, storeAmount_ *big.Int, to_ common.Address, isCallback_ bool) (*types.Transaction, error) {
	return _FluidDexV2.contract.Transact(opts, "settle", token_, supplyAmount_, borrowAmount_, storeAmount_, to_, isCallback_)
}

// Settle is a paid mutator transaction binding the contract method 0x5aa62d1b.
//
// Solidity: function settle(address token_, int256 supplyAmount_, int256 borrowAmount_, int256 storeAmount_, address to_, bool isCallback_) payable returns()
func (_FluidDexV2 *FluidDexV2Session) Settle(token_ common.Address, supplyAmount_ *big.Int, borrowAmount_ *big.Int, storeAmount_ *big.Int, to_ common.Address, isCallback_ bool) (*types.Transaction, error) {
	return _FluidDexV2.Contract.Settle(&_FluidDexV2.TransactOpts, token_, supplyAmount_, borrowAmount_, storeAmount_, to_, isCallback_)
}

// Settle is a paid mutator transaction binding the contract method 0x5aa62d1b.
//
// Solidity: function settle(address token_, int256 supplyAmount_, int256 borrowAmount_, int256 storeAmount_, address to_, bool isCallback_) payable returns()
func (_FluidDexV2 *FluidDexV2TransactorSession) Settle(token_ common.Address, supplyAmount_ *big.Int, borrowAmount_ *big.Int, storeAmount_ *big.Int, to_ common.Address, isCallback_ bool) (*types.Transaction, error) {
	return _FluidDexV2.Contract.Settle(&_FluidDexV2.TransactOpts, token_, supplyAmount_, borrowAmount_, storeAmount_, to_, isCallback_)
}

// StartOperation is a paid mutator transaction binding the contract method 0xc9d5479f.
//
// Solidity: function startOperation(bytes data_) returns(bytes result_)
func (_FluidDexV2 *FluidDexV2Transactor) StartOperation(opts *bind.TransactOpts, data_ []byte) (*types.Transaction, error) {
	return _FluidDexV2.contract.Transact(opts, "startOperation", data_)
}

// StartOperation is a paid mutator transaction binding the contract method 0xc9d5479f.
//
// Solidity: function startOperation(bytes data_) returns(bytes result_)
func (_FluidDexV2 *FluidDexV2Session) StartOperation(data_ []byte) (*types.Transaction, error) {
	return _FluidDexV2.Contract.StartOperation(&_FluidDexV2.TransactOpts, data_)
}

// StartOperation is a paid mutator transaction binding the contract method 0xc9d5479f.
//
// Solidity: function startOperation(bytes data_) returns(bytes result_)
func (_FluidDexV2 *FluidDexV2TransactorSession) StartOperation(data_ []byte) (*types.Transaction, error) {
	return _FluidDexV2.Contract.StartOperation(&_FluidDexV2.TransactOpts, data_)
}

// UpdateAuth is a paid mutator transaction binding the contract method 0xe9c771b2.
//
// Solidity: function updateAuth(address auth_, bool isAuth_) returns()
func (_FluidDexV2 *FluidDexV2Transactor) UpdateAuth(opts *bind.TransactOpts, auth_ common.Address, isAuth_ bool) (*types.Transaction, error) {
	return _FluidDexV2.contract.Transact(opts, "updateAuth", auth_, isAuth_)
}

// UpdateAuth is a paid mutator transaction binding the contract method 0xe9c771b2.
//
// Solidity: function updateAuth(address auth_, bool isAuth_) returns()
func (_FluidDexV2 *FluidDexV2Session) UpdateAuth(auth_ common.Address, isAuth_ bool) (*types.Transaction, error) {
	return _FluidDexV2.Contract.UpdateAuth(&_FluidDexV2.TransactOpts, auth_, isAuth_)
}

// UpdateAuth is a paid mutator transaction binding the contract method 0xe9c771b2.
//
// Solidity: function updateAuth(address auth_, bool isAuth_) returns()
func (_FluidDexV2 *FluidDexV2TransactorSession) UpdateAuth(auth_ common.Address, isAuth_ bool) (*types.Transaction, error) {
	return _FluidDexV2.Contract.UpdateAuth(&_FluidDexV2.TransactOpts, auth_, isAuth_)
}

// UpdateDexTypeToAdminImplementation is a paid mutator transaction binding the contract method 0xb20aca3d.
//
// Solidity: function updateDexTypeToAdminImplementation(uint256 dexType_, uint256 adminImplementationId_, address adminImplementation_) returns()
func (_FluidDexV2 *FluidDexV2Transactor) UpdateDexTypeToAdminImplementation(opts *bind.TransactOpts, dexType_ *big.Int, adminImplementationId_ *big.Int, adminImplementation_ common.Address) (*types.Transaction, error) {
	return _FluidDexV2.contract.Transact(opts, "updateDexTypeToAdminImplementation", dexType_, adminImplementationId_, adminImplementation_)
}

// UpdateDexTypeToAdminImplementation is a paid mutator transaction binding the contract method 0xb20aca3d.
//
// Solidity: function updateDexTypeToAdminImplementation(uint256 dexType_, uint256 adminImplementationId_, address adminImplementation_) returns()
func (_FluidDexV2 *FluidDexV2Session) UpdateDexTypeToAdminImplementation(dexType_ *big.Int, adminImplementationId_ *big.Int, adminImplementation_ common.Address) (*types.Transaction, error) {
	return _FluidDexV2.Contract.UpdateDexTypeToAdminImplementation(&_FluidDexV2.TransactOpts, dexType_, adminImplementationId_, adminImplementation_)
}

// UpdateDexTypeToAdminImplementation is a paid mutator transaction binding the contract method 0xb20aca3d.
//
// Solidity: function updateDexTypeToAdminImplementation(uint256 dexType_, uint256 adminImplementationId_, address adminImplementation_) returns()
func (_FluidDexV2 *FluidDexV2TransactorSession) UpdateDexTypeToAdminImplementation(dexType_ *big.Int, adminImplementationId_ *big.Int, adminImplementation_ common.Address) (*types.Transaction, error) {
	return _FluidDexV2.Contract.UpdateDexTypeToAdminImplementation(&_FluidDexV2.TransactOpts, dexType_, adminImplementationId_, adminImplementation_)
}

// UpgradeTo is a paid mutator transaction binding the contract method 0x3659cfe6.
//
// Solidity: function upgradeTo(address newImplementation_) returns()
func (_FluidDexV2 *FluidDexV2Transactor) UpgradeTo(opts *bind.TransactOpts, newImplementation_ common.Address) (*types.Transaction, error) {
	return _FluidDexV2.contract.Transact(opts, "upgradeTo", newImplementation_)
}

// UpgradeTo is a paid mutator transaction binding the contract method 0x3659cfe6.
//
// Solidity: function upgradeTo(address newImplementation_) returns()
func (_FluidDexV2 *FluidDexV2Session) UpgradeTo(newImplementation_ common.Address) (*types.Transaction, error) {
	return _FluidDexV2.Contract.UpgradeTo(&_FluidDexV2.TransactOpts, newImplementation_)
}

// UpgradeTo is a paid mutator transaction binding the contract method 0x3659cfe6.
//
// Solidity: function upgradeTo(address newImplementation_) returns()
func (_FluidDexV2 *FluidDexV2TransactorSession) UpgradeTo(newImplementation_ common.Address) (*types.Transaction, error) {
	return _FluidDexV2.Contract.UpgradeTo(&_FluidDexV2.TransactOpts, newImplementation_)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation_, bytes data_) payable returns()
func (_FluidDexV2 *FluidDexV2Transactor) UpgradeToAndCall(opts *bind.TransactOpts, newImplementation_ common.Address, data_ []byte) (*types.Transaction, error) {
	return _FluidDexV2.contract.Transact(opts, "upgradeToAndCall", newImplementation_, data_)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation_, bytes data_) payable returns()
func (_FluidDexV2 *FluidDexV2Session) UpgradeToAndCall(newImplementation_ common.Address, data_ []byte) (*types.Transaction, error) {
	return _FluidDexV2.Contract.UpgradeToAndCall(&_FluidDexV2.TransactOpts, newImplementation_, data_)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation_, bytes data_) payable returns()
func (_FluidDexV2 *FluidDexV2TransactorSession) UpgradeToAndCall(newImplementation_ common.Address, data_ []byte) (*types.Transaction, error) {
	return _FluidDexV2.Contract.UpgradeToAndCall(&_FluidDexV2.TransactOpts, newImplementation_, data_)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_FluidDexV2 *FluidDexV2Transactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FluidDexV2.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_FluidDexV2 *FluidDexV2Session) Receive() (*types.Transaction, error) {
	return _FluidDexV2.Contract.Receive(&_FluidDexV2.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_FluidDexV2 *FluidDexV2TransactorSession) Receive() (*types.Transaction, error) {
	return _FluidDexV2.Contract.Receive(&_FluidDexV2.TransactOpts)
}

// FluidDexV2LogAddOrRemoveTokensIterator is returned from FilterLogAddOrRemoveTokens and is used to iterate over the raw logs and unpacked data for LogAddOrRemoveTokens events raised by the FluidDexV2 contract.
type FluidDexV2LogAddOrRemoveTokensIterator struct {
	Event *FluidDexV2LogAddOrRemoveTokens // Event containing the contract specifics and raw log

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
func (it *FluidDexV2LogAddOrRemoveTokensIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FluidDexV2LogAddOrRemoveTokens)
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
		it.Event = new(FluidDexV2LogAddOrRemoveTokens)
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
func (it *FluidDexV2LogAddOrRemoveTokensIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FluidDexV2LogAddOrRemoveTokensIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FluidDexV2LogAddOrRemoveTokens represents a LogAddOrRemoveTokens event raised by the FluidDexV2 contract.
type FluidDexV2LogAddOrRemoveTokens struct {
	Token  common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterLogAddOrRemoveTokens is a free log retrieval operation binding the contract event 0x2f84845af9d6e4715f7efcad312517d8ffb6daa89585ee278addc134a3ba92ce.
//
// Solidity: event LogAddOrRemoveTokens(address indexed token, int256 amount)
func (_FluidDexV2 *FluidDexV2Filterer) FilterLogAddOrRemoveTokens(opts *bind.FilterOpts, token []common.Address) (*FluidDexV2LogAddOrRemoveTokensIterator, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}

	logs, sub, err := _FluidDexV2.contract.FilterLogs(opts, "LogAddOrRemoveTokens", tokenRule)
	if err != nil {
		return nil, err
	}
	return &FluidDexV2LogAddOrRemoveTokensIterator{contract: _FluidDexV2.contract, event: "LogAddOrRemoveTokens", logs: logs, sub: sub}, nil
}

// WatchLogAddOrRemoveTokens is a free log subscription operation binding the contract event 0x2f84845af9d6e4715f7efcad312517d8ffb6daa89585ee278addc134a3ba92ce.
//
// Solidity: event LogAddOrRemoveTokens(address indexed token, int256 amount)
func (_FluidDexV2 *FluidDexV2Filterer) WatchLogAddOrRemoveTokens(opts *bind.WatchOpts, sink chan<- *FluidDexV2LogAddOrRemoveTokens, token []common.Address) (event.Subscription, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}

	logs, sub, err := _FluidDexV2.contract.WatchLogs(opts, "LogAddOrRemoveTokens", tokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FluidDexV2LogAddOrRemoveTokens)
				if err := _FluidDexV2.contract.UnpackLog(event, "LogAddOrRemoveTokens", log); err != nil {
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

// ParseLogAddOrRemoveTokens is a log parse operation binding the contract event 0x2f84845af9d6e4715f7efcad312517d8ffb6daa89585ee278addc134a3ba92ce.
//
// Solidity: event LogAddOrRemoveTokens(address indexed token, int256 amount)
func (_FluidDexV2 *FluidDexV2Filterer) ParseLogAddOrRemoveTokens(log types.Log) (*FluidDexV2LogAddOrRemoveTokens, error) {
	event := new(FluidDexV2LogAddOrRemoveTokens)
	if err := _FluidDexV2.contract.UnpackLog(event, "LogAddOrRemoveTokens", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FluidDexV2LogBorrowIterator is returned from FilterLogBorrow and is used to iterate over the raw logs and unpacked data for LogBorrow events raised by the FluidDexV2 contract.
type FluidDexV2LogBorrowIterator struct {
	Event *FluidDexV2LogBorrow // Event containing the contract specifics and raw log

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
func (it *FluidDexV2LogBorrowIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FluidDexV2LogBorrow)
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
		it.Event = new(FluidDexV2LogBorrow)
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
func (it *FluidDexV2LogBorrowIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FluidDexV2LogBorrowIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FluidDexV2LogBorrow represents a LogBorrow event raised by the FluidDexV2 contract.
type FluidDexV2LogBorrow struct {
	DexType              *big.Int
	DexId                [32]byte
	User                 common.Address
	TickLower            *big.Int
	TickUpper            *big.Int
	PositionSalt         [32]byte
	Amount0              *big.Int
	Amount1              *big.Int
	FeeAccruedToken0     *big.Int
	FeeAccruedToken1     *big.Int
	LiquidityIncreaseRaw *big.Int
	Raw                  types.Log // Blockchain specific contextual infos
}

// FilterLogBorrow is a free log retrieval operation binding the contract event 0x961b382cd4afde644c763913b46ab630ca88dbdf43335cdb7571872ec764678f.
//
// Solidity: event LogBorrow(uint256 dexType, bytes32 dexId, address user, int24 tickLower, int24 tickUpper, bytes32 positionSalt, uint256 amount0, uint256 amount1, uint256 feeAccruedToken0, uint256 feeAccruedToken1, uint256 liquidityIncreaseRaw)
func (_FluidDexV2 *FluidDexV2Filterer) FilterLogBorrow(opts *bind.FilterOpts) (*FluidDexV2LogBorrowIterator, error) {

	logs, sub, err := _FluidDexV2.contract.FilterLogs(opts, "LogBorrow")
	if err != nil {
		return nil, err
	}
	return &FluidDexV2LogBorrowIterator{contract: _FluidDexV2.contract, event: "LogBorrow", logs: logs, sub: sub}, nil
}

// WatchLogBorrow is a free log subscription operation binding the contract event 0x961b382cd4afde644c763913b46ab630ca88dbdf43335cdb7571872ec764678f.
//
// Solidity: event LogBorrow(uint256 dexType, bytes32 dexId, address user, int24 tickLower, int24 tickUpper, bytes32 positionSalt, uint256 amount0, uint256 amount1, uint256 feeAccruedToken0, uint256 feeAccruedToken1, uint256 liquidityIncreaseRaw)
func (_FluidDexV2 *FluidDexV2Filterer) WatchLogBorrow(opts *bind.WatchOpts, sink chan<- *FluidDexV2LogBorrow) (event.Subscription, error) {

	logs, sub, err := _FluidDexV2.contract.WatchLogs(opts, "LogBorrow")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FluidDexV2LogBorrow)
				if err := _FluidDexV2.contract.UnpackLog(event, "LogBorrow", log); err != nil {
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

// ParseLogBorrow is a log parse operation binding the contract event 0x961b382cd4afde644c763913b46ab630ca88dbdf43335cdb7571872ec764678f.
//
// Solidity: event LogBorrow(uint256 dexType, bytes32 dexId, address user, int24 tickLower, int24 tickUpper, bytes32 positionSalt, uint256 amount0, uint256 amount1, uint256 feeAccruedToken0, uint256 feeAccruedToken1, uint256 liquidityIncreaseRaw)
func (_FluidDexV2 *FluidDexV2Filterer) ParseLogBorrow(log types.Log) (*FluidDexV2LogBorrow, error) {
	event := new(FluidDexV2LogBorrow)
	if err := _FluidDexV2.contract.UnpackLog(event, "LogBorrow", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FluidDexV2LogDepositIterator is returned from FilterLogDeposit and is used to iterate over the raw logs and unpacked data for LogDeposit events raised by the FluidDexV2 contract.
type FluidDexV2LogDepositIterator struct {
	Event *FluidDexV2LogDeposit // Event containing the contract specifics and raw log

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
func (it *FluidDexV2LogDepositIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FluidDexV2LogDeposit)
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
		it.Event = new(FluidDexV2LogDeposit)
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
func (it *FluidDexV2LogDepositIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FluidDexV2LogDepositIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FluidDexV2LogDeposit represents a LogDeposit event raised by the FluidDexV2 contract.
type FluidDexV2LogDeposit struct {
	DexType              *big.Int
	DexId                [32]byte
	User                 common.Address
	TickLower            *big.Int
	TickUpper            *big.Int
	PositionSalt         [32]byte
	Amount0              *big.Int
	Amount1              *big.Int
	FeeAccruedToken0     *big.Int
	FeeAccruedToken1     *big.Int
	LiquidityIncreaseRaw *big.Int
	Raw                  types.Log // Blockchain specific contextual infos
}

// FilterLogDeposit is a free log retrieval operation binding the contract event 0x6001fc2c19e4cb6ddb86346c7f6f1aaa42ecf6d4b7277d0b5477d772f70614d8.
//
// Solidity: event LogDeposit(uint256 dexType, bytes32 dexId, address user, int24 tickLower, int24 tickUpper, bytes32 positionSalt, uint256 amount0, uint256 amount1, uint256 feeAccruedToken0, uint256 feeAccruedToken1, uint256 liquidityIncreaseRaw)
func (_FluidDexV2 *FluidDexV2Filterer) FilterLogDeposit(opts *bind.FilterOpts) (*FluidDexV2LogDepositIterator, error) {

	logs, sub, err := _FluidDexV2.contract.FilterLogs(opts, "LogDeposit")
	if err != nil {
		return nil, err
	}
	return &FluidDexV2LogDepositIterator{contract: _FluidDexV2.contract, event: "LogDeposit", logs: logs, sub: sub}, nil
}

// WatchLogDeposit is a free log subscription operation binding the contract event 0x6001fc2c19e4cb6ddb86346c7f6f1aaa42ecf6d4b7277d0b5477d772f70614d8.
//
// Solidity: event LogDeposit(uint256 dexType, bytes32 dexId, address user, int24 tickLower, int24 tickUpper, bytes32 positionSalt, uint256 amount0, uint256 amount1, uint256 feeAccruedToken0, uint256 feeAccruedToken1, uint256 liquidityIncreaseRaw)
func (_FluidDexV2 *FluidDexV2Filterer) WatchLogDeposit(opts *bind.WatchOpts, sink chan<- *FluidDexV2LogDeposit) (event.Subscription, error) {

	logs, sub, err := _FluidDexV2.contract.WatchLogs(opts, "LogDeposit")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FluidDexV2LogDeposit)
				if err := _FluidDexV2.contract.UnpackLog(event, "LogDeposit", log); err != nil {
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

// ParseLogDeposit is a log parse operation binding the contract event 0x6001fc2c19e4cb6ddb86346c7f6f1aaa42ecf6d4b7277d0b5477d772f70614d8.
//
// Solidity: event LogDeposit(uint256 dexType, bytes32 dexId, address user, int24 tickLower, int24 tickUpper, bytes32 positionSalt, uint256 amount0, uint256 amount1, uint256 feeAccruedToken0, uint256 feeAccruedToken1, uint256 liquidityIncreaseRaw)
func (_FluidDexV2 *FluidDexV2Filterer) ParseLogDeposit(log types.Log) (*FluidDexV2LogDeposit, error) {
	event := new(FluidDexV2LogDeposit)
	if err := _FluidDexV2.contract.UnpackLog(event, "LogDeposit", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FluidDexV2LogInitializeIterator is returned from FilterLogInitialize and is used to iterate over the raw logs and unpacked data for LogInitialize events raised by the FluidDexV2 contract.
type FluidDexV2LogInitializeIterator struct {
	Event *FluidDexV2LogInitialize // Event containing the contract specifics and raw log

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
func (it *FluidDexV2LogInitializeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FluidDexV2LogInitialize)
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
		it.Event = new(FluidDexV2LogInitialize)
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
func (it *FluidDexV2LogInitializeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FluidDexV2LogInitializeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FluidDexV2LogInitialize represents a LogInitialize event raised by the FluidDexV2 contract.
type FluidDexV2LogInitialize struct {
	DexType      *big.Int
	DexId        [32]byte
	DexKey       DexKey
	SqrtPriceX96 *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterLogInitialize is a free log retrieval operation binding the contract event 0x93aef7ffcc4aaa866f9c0d52506953eeb28f61d39ac4d5cc3d2ef59426e85603.
//
// Solidity: event LogInitialize(uint256 dexType, bytes32 dexId, (address,address,uint24,uint24,address) dexKey, uint256 sqrtPriceX96)
func (_FluidDexV2 *FluidDexV2Filterer) FilterLogInitialize(opts *bind.FilterOpts) (*FluidDexV2LogInitializeIterator, error) {

	logs, sub, err := _FluidDexV2.contract.FilterLogs(opts, "LogInitialize")
	if err != nil {
		return nil, err
	}
	return &FluidDexV2LogInitializeIterator{contract: _FluidDexV2.contract, event: "LogInitialize", logs: logs, sub: sub}, nil
}

// WatchLogInitialize is a free log subscription operation binding the contract event 0x93aef7ffcc4aaa866f9c0d52506953eeb28f61d39ac4d5cc3d2ef59426e85603.
//
// Solidity: event LogInitialize(uint256 dexType, bytes32 dexId, (address,address,uint24,uint24,address) dexKey, uint256 sqrtPriceX96)
func (_FluidDexV2 *FluidDexV2Filterer) WatchLogInitialize(opts *bind.WatchOpts, sink chan<- *FluidDexV2LogInitialize) (event.Subscription, error) {

	logs, sub, err := _FluidDexV2.contract.WatchLogs(opts, "LogInitialize")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FluidDexV2LogInitialize)
				if err := _FluidDexV2.contract.UnpackLog(event, "LogInitialize", log); err != nil {
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

// ParseLogInitialize is a log parse operation binding the contract event 0x93aef7ffcc4aaa866f9c0d52506953eeb28f61d39ac4d5cc3d2ef59426e85603.
//
// Solidity: event LogInitialize(uint256 dexType, bytes32 dexId, (address,address,uint24,uint24,address) dexKey, uint256 sqrtPriceX96)
func (_FluidDexV2 *FluidDexV2Filterer) ParseLogInitialize(log types.Log) (*FluidDexV2LogInitialize, error) {
	event := new(FluidDexV2LogInitialize)
	if err := _FluidDexV2.contract.UnpackLog(event, "LogInitialize", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FluidDexV2LogOperateIterator is returned from FilterLogOperate and is used to iterate over the raw logs and unpacked data for LogOperate events raised by the FluidDexV2 contract.
type FluidDexV2LogOperateIterator struct {
	Event *FluidDexV2LogOperate // Event containing the contract specifics and raw log

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
func (it *FluidDexV2LogOperateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FluidDexV2LogOperate)
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
		it.Event = new(FluidDexV2LogOperate)
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
func (it *FluidDexV2LogOperateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FluidDexV2LogOperateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FluidDexV2LogOperate represents a LogOperate event raised by the FluidDexV2 contract.
type FluidDexV2LogOperate struct {
	User             common.Address
	DexType          *big.Int
	ImplementationId *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterLogOperate is a free log retrieval operation binding the contract event 0xdd396b775fb83887efb821a7888d5c857ab6edbfaa689df3bf6e1e3244e7a5a6.
//
// Solidity: event LogOperate(address indexed user, uint256 indexed dexType, uint256 indexed implementationId)
func (_FluidDexV2 *FluidDexV2Filterer) FilterLogOperate(opts *bind.FilterOpts, user []common.Address, dexType []*big.Int, implementationId []*big.Int) (*FluidDexV2LogOperateIterator, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}
	var dexTypeRule []interface{}
	for _, dexTypeItem := range dexType {
		dexTypeRule = append(dexTypeRule, dexTypeItem)
	}
	var implementationIdRule []interface{}
	for _, implementationIdItem := range implementationId {
		implementationIdRule = append(implementationIdRule, implementationIdItem)
	}

	logs, sub, err := _FluidDexV2.contract.FilterLogs(opts, "LogOperate", userRule, dexTypeRule, implementationIdRule)
	if err != nil {
		return nil, err
	}
	return &FluidDexV2LogOperateIterator{contract: _FluidDexV2.contract, event: "LogOperate", logs: logs, sub: sub}, nil
}

// WatchLogOperate is a free log subscription operation binding the contract event 0xdd396b775fb83887efb821a7888d5c857ab6edbfaa689df3bf6e1e3244e7a5a6.
//
// Solidity: event LogOperate(address indexed user, uint256 indexed dexType, uint256 indexed implementationId)
func (_FluidDexV2 *FluidDexV2Filterer) WatchLogOperate(opts *bind.WatchOpts, sink chan<- *FluidDexV2LogOperate, user []common.Address, dexType []*big.Int, implementationId []*big.Int) (event.Subscription, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}
	var dexTypeRule []interface{}
	for _, dexTypeItem := range dexType {
		dexTypeRule = append(dexTypeRule, dexTypeItem)
	}
	var implementationIdRule []interface{}
	for _, implementationIdItem := range implementationId {
		implementationIdRule = append(implementationIdRule, implementationIdItem)
	}

	logs, sub, err := _FluidDexV2.contract.WatchLogs(opts, "LogOperate", userRule, dexTypeRule, implementationIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FluidDexV2LogOperate)
				if err := _FluidDexV2.contract.UnpackLog(event, "LogOperate", log); err != nil {
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

// ParseLogOperate is a log parse operation binding the contract event 0xdd396b775fb83887efb821a7888d5c857ab6edbfaa689df3bf6e1e3244e7a5a6.
//
// Solidity: event LogOperate(address indexed user, uint256 indexed dexType, uint256 indexed implementationId)
func (_FluidDexV2 *FluidDexV2Filterer) ParseLogOperate(log types.Log) (*FluidDexV2LogOperate, error) {
	event := new(FluidDexV2LogOperate)
	if err := _FluidDexV2.contract.UnpackLog(event, "LogOperate", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FluidDexV2LogOperateAdminIterator is returned from FilterLogOperateAdmin and is used to iterate over the raw logs and unpacked data for LogOperateAdmin events raised by the FluidDexV2 contract.
type FluidDexV2LogOperateAdminIterator struct {
	Event *FluidDexV2LogOperateAdmin // Event containing the contract specifics and raw log

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
func (it *FluidDexV2LogOperateAdminIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FluidDexV2LogOperateAdmin)
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
		it.Event = new(FluidDexV2LogOperateAdmin)
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
func (it *FluidDexV2LogOperateAdminIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FluidDexV2LogOperateAdminIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FluidDexV2LogOperateAdmin represents a LogOperateAdmin event raised by the FluidDexV2 contract.
type FluidDexV2LogOperateAdmin struct {
	User             common.Address
	DexType          *big.Int
	ImplementationId *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterLogOperateAdmin is a free log retrieval operation binding the contract event 0x4ca7efe2364a3910d784894066f98b6bc696da684f506b42278f3e1a1b6fb43b.
//
// Solidity: event LogOperateAdmin(address indexed user, uint256 indexed dexType, uint256 indexed implementationId)
func (_FluidDexV2 *FluidDexV2Filterer) FilterLogOperateAdmin(opts *bind.FilterOpts, user []common.Address, dexType []*big.Int, implementationId []*big.Int) (*FluidDexV2LogOperateAdminIterator, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}
	var dexTypeRule []interface{}
	for _, dexTypeItem := range dexType {
		dexTypeRule = append(dexTypeRule, dexTypeItem)
	}
	var implementationIdRule []interface{}
	for _, implementationIdItem := range implementationId {
		implementationIdRule = append(implementationIdRule, implementationIdItem)
	}

	logs, sub, err := _FluidDexV2.contract.FilterLogs(opts, "LogOperateAdmin", userRule, dexTypeRule, implementationIdRule)
	if err != nil {
		return nil, err
	}
	return &FluidDexV2LogOperateAdminIterator{contract: _FluidDexV2.contract, event: "LogOperateAdmin", logs: logs, sub: sub}, nil
}

// WatchLogOperateAdmin is a free log subscription operation binding the contract event 0x4ca7efe2364a3910d784894066f98b6bc696da684f506b42278f3e1a1b6fb43b.
//
// Solidity: event LogOperateAdmin(address indexed user, uint256 indexed dexType, uint256 indexed implementationId)
func (_FluidDexV2 *FluidDexV2Filterer) WatchLogOperateAdmin(opts *bind.WatchOpts, sink chan<- *FluidDexV2LogOperateAdmin, user []common.Address, dexType []*big.Int, implementationId []*big.Int) (event.Subscription, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}
	var dexTypeRule []interface{}
	for _, dexTypeItem := range dexType {
		dexTypeRule = append(dexTypeRule, dexTypeItem)
	}
	var implementationIdRule []interface{}
	for _, implementationIdItem := range implementationId {
		implementationIdRule = append(implementationIdRule, implementationIdItem)
	}

	logs, sub, err := _FluidDexV2.contract.WatchLogs(opts, "LogOperateAdmin", userRule, dexTypeRule, implementationIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FluidDexV2LogOperateAdmin)
				if err := _FluidDexV2.contract.UnpackLog(event, "LogOperateAdmin", log); err != nil {
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

// ParseLogOperateAdmin is a log parse operation binding the contract event 0x4ca7efe2364a3910d784894066f98b6bc696da684f506b42278f3e1a1b6fb43b.
//
// Solidity: event LogOperateAdmin(address indexed user, uint256 indexed dexType, uint256 indexed implementationId)
func (_FluidDexV2 *FluidDexV2Filterer) ParseLogOperateAdmin(log types.Log) (*FluidDexV2LogOperateAdmin, error) {
	event := new(FluidDexV2LogOperateAdmin)
	if err := _FluidDexV2.contract.UnpackLog(event, "LogOperateAdmin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FluidDexV2LogPaybackIterator is returned from FilterLogPayback and is used to iterate over the raw logs and unpacked data for LogPayback events raised by the FluidDexV2 contract.
type FluidDexV2LogPaybackIterator struct {
	Event *FluidDexV2LogPayback // Event containing the contract specifics and raw log

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
func (it *FluidDexV2LogPaybackIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FluidDexV2LogPayback)
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
		it.Event = new(FluidDexV2LogPayback)
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
func (it *FluidDexV2LogPaybackIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FluidDexV2LogPaybackIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FluidDexV2LogPayback represents a LogPayback event raised by the FluidDexV2 contract.
type FluidDexV2LogPayback struct {
	DexType              *big.Int
	DexId                [32]byte
	User                 common.Address
	TickLower            *big.Int
	TickUpper            *big.Int
	PositionSalt         [32]byte
	Amount0              *big.Int
	Amount1              *big.Int
	FeeAccruedToken0     *big.Int
	FeeAccruedToken1     *big.Int
	LiquidityDecreaseRaw *big.Int
	Raw                  types.Log // Blockchain specific contextual infos
}

// FilterLogPayback is a free log retrieval operation binding the contract event 0x1a99572922a55a0c300560a3c83b988bf35cf14bd2a21d142f7fbe7b151f7852.
//
// Solidity: event LogPayback(uint256 dexType, bytes32 dexId, address user, int24 tickLower, int24 tickUpper, bytes32 positionSalt, uint256 amount0, uint256 amount1, uint256 feeAccruedToken0, uint256 feeAccruedToken1, uint256 liquidityDecreaseRaw)
func (_FluidDexV2 *FluidDexV2Filterer) FilterLogPayback(opts *bind.FilterOpts) (*FluidDexV2LogPaybackIterator, error) {

	logs, sub, err := _FluidDexV2.contract.FilterLogs(opts, "LogPayback")
	if err != nil {
		return nil, err
	}
	return &FluidDexV2LogPaybackIterator{contract: _FluidDexV2.contract, event: "LogPayback", logs: logs, sub: sub}, nil
}

// WatchLogPayback is a free log subscription operation binding the contract event 0x1a99572922a55a0c300560a3c83b988bf35cf14bd2a21d142f7fbe7b151f7852.
//
// Solidity: event LogPayback(uint256 dexType, bytes32 dexId, address user, int24 tickLower, int24 tickUpper, bytes32 positionSalt, uint256 amount0, uint256 amount1, uint256 feeAccruedToken0, uint256 feeAccruedToken1, uint256 liquidityDecreaseRaw)
func (_FluidDexV2 *FluidDexV2Filterer) WatchLogPayback(opts *bind.WatchOpts, sink chan<- *FluidDexV2LogPayback) (event.Subscription, error) {

	logs, sub, err := _FluidDexV2.contract.WatchLogs(opts, "LogPayback")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FluidDexV2LogPayback)
				if err := _FluidDexV2.contract.UnpackLog(event, "LogPayback", log); err != nil {
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

// ParseLogPayback is a log parse operation binding the contract event 0x1a99572922a55a0c300560a3c83b988bf35cf14bd2a21d142f7fbe7b151f7852.
//
// Solidity: event LogPayback(uint256 dexType, bytes32 dexId, address user, int24 tickLower, int24 tickUpper, bytes32 positionSalt, uint256 amount0, uint256 amount1, uint256 feeAccruedToken0, uint256 feeAccruedToken1, uint256 liquidityDecreaseRaw)
func (_FluidDexV2 *FluidDexV2Filterer) ParseLogPayback(log types.Log) (*FluidDexV2LogPayback, error) {
	event := new(FluidDexV2LogPayback)
	if err := _FluidDexV2.contract.UnpackLog(event, "LogPayback", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FluidDexV2LogRebalanceIterator is returned from FilterLogRebalance and is used to iterate over the raw logs and unpacked data for LogRebalance events raised by the FluidDexV2 contract.
type FluidDexV2LogRebalanceIterator struct {
	Event *FluidDexV2LogRebalance // Event containing the contract specifics and raw log

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
func (it *FluidDexV2LogRebalanceIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FluidDexV2LogRebalance)
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
		it.Event = new(FluidDexV2LogRebalance)
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
func (it *FluidDexV2LogRebalanceIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FluidDexV2LogRebalanceIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FluidDexV2LogRebalance represents a LogRebalance event raised by the FluidDexV2 contract.
type FluidDexV2LogRebalance struct {
	Token        common.Address
	SupplyAmount *big.Int
	BorrowAmount *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterLogRebalance is a free log retrieval operation binding the contract event 0xcb0222eb32b6798438e843651d9bf7e3cfc5f1b63ceb7354c5801d43add3e325.
//
// Solidity: event LogRebalance(address indexed token, int256 supplyAmount, int256 borrowAmount)
func (_FluidDexV2 *FluidDexV2Filterer) FilterLogRebalance(opts *bind.FilterOpts, token []common.Address) (*FluidDexV2LogRebalanceIterator, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}

	logs, sub, err := _FluidDexV2.contract.FilterLogs(opts, "LogRebalance", tokenRule)
	if err != nil {
		return nil, err
	}
	return &FluidDexV2LogRebalanceIterator{contract: _FluidDexV2.contract, event: "LogRebalance", logs: logs, sub: sub}, nil
}

// WatchLogRebalance is a free log subscription operation binding the contract event 0xcb0222eb32b6798438e843651d9bf7e3cfc5f1b63ceb7354c5801d43add3e325.
//
// Solidity: event LogRebalance(address indexed token, int256 supplyAmount, int256 borrowAmount)
func (_FluidDexV2 *FluidDexV2Filterer) WatchLogRebalance(opts *bind.WatchOpts, sink chan<- *FluidDexV2LogRebalance, token []common.Address) (event.Subscription, error) {

	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}

	logs, sub, err := _FluidDexV2.contract.WatchLogs(opts, "LogRebalance", tokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FluidDexV2LogRebalance)
				if err := _FluidDexV2.contract.UnpackLog(event, "LogRebalance", log); err != nil {
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

// ParseLogRebalance is a log parse operation binding the contract event 0xcb0222eb32b6798438e843651d9bf7e3cfc5f1b63ceb7354c5801d43add3e325.
//
// Solidity: event LogRebalance(address indexed token, int256 supplyAmount, int256 borrowAmount)
func (_FluidDexV2 *FluidDexV2Filterer) ParseLogRebalance(log types.Log) (*FluidDexV2LogRebalance, error) {
	event := new(FluidDexV2LogRebalance)
	if err := _FluidDexV2.contract.UnpackLog(event, "LogRebalance", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FluidDexV2LogSettleIterator is returned from FilterLogSettle and is used to iterate over the raw logs and unpacked data for LogSettle events raised by the FluidDexV2 contract.
type FluidDexV2LogSettleIterator struct {
	Event *FluidDexV2LogSettle // Event containing the contract specifics and raw log

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
func (it *FluidDexV2LogSettleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FluidDexV2LogSettle)
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
		it.Event = new(FluidDexV2LogSettle)
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
func (it *FluidDexV2LogSettleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FluidDexV2LogSettleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FluidDexV2LogSettle represents a LogSettle event raised by the FluidDexV2 contract.
type FluidDexV2LogSettle struct {
	User         common.Address
	Token        common.Address
	SupplyAmount *big.Int
	BorrowAmount *big.Int
	StoreAmount  *big.Int
	To           common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterLogSettle is a free log retrieval operation binding the contract event 0xecae71474004c8d68ce703c971981cf96c1e462343c71e4c97677a5467981671.
//
// Solidity: event LogSettle(address indexed user, address indexed token, int256 supplyAmount, int256 borrowAmount, int256 storeAmount, address to)
func (_FluidDexV2 *FluidDexV2Filterer) FilterLogSettle(opts *bind.FilterOpts, user []common.Address, token []common.Address) (*FluidDexV2LogSettleIterator, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}
	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}

	logs, sub, err := _FluidDexV2.contract.FilterLogs(opts, "LogSettle", userRule, tokenRule)
	if err != nil {
		return nil, err
	}
	return &FluidDexV2LogSettleIterator{contract: _FluidDexV2.contract, event: "LogSettle", logs: logs, sub: sub}, nil
}

// WatchLogSettle is a free log subscription operation binding the contract event 0xecae71474004c8d68ce703c971981cf96c1e462343c71e4c97677a5467981671.
//
// Solidity: event LogSettle(address indexed user, address indexed token, int256 supplyAmount, int256 borrowAmount, int256 storeAmount, address to)
func (_FluidDexV2 *FluidDexV2Filterer) WatchLogSettle(opts *bind.WatchOpts, sink chan<- *FluidDexV2LogSettle, user []common.Address, token []common.Address) (event.Subscription, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}
	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}

	logs, sub, err := _FluidDexV2.contract.WatchLogs(opts, "LogSettle", userRule, tokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FluidDexV2LogSettle)
				if err := _FluidDexV2.contract.UnpackLog(event, "LogSettle", log); err != nil {
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

// ParseLogSettle is a log parse operation binding the contract event 0xecae71474004c8d68ce703c971981cf96c1e462343c71e4c97677a5467981671.
//
// Solidity: event LogSettle(address indexed user, address indexed token, int256 supplyAmount, int256 borrowAmount, int256 storeAmount, address to)
func (_FluidDexV2 *FluidDexV2Filterer) ParseLogSettle(log types.Log) (*FluidDexV2LogSettle, error) {
	event := new(FluidDexV2LogSettle)
	if err := _FluidDexV2.contract.UnpackLog(event, "LogSettle", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FluidDexV2LogUpdateAuthIterator is returned from FilterLogUpdateAuth and is used to iterate over the raw logs and unpacked data for LogUpdateAuth events raised by the FluidDexV2 contract.
type FluidDexV2LogUpdateAuthIterator struct {
	Event *FluidDexV2LogUpdateAuth // Event containing the contract specifics and raw log

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
func (it *FluidDexV2LogUpdateAuthIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FluidDexV2LogUpdateAuth)
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
		it.Event = new(FluidDexV2LogUpdateAuth)
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
func (it *FluidDexV2LogUpdateAuthIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FluidDexV2LogUpdateAuthIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FluidDexV2LogUpdateAuth represents a LogUpdateAuth event raised by the FluidDexV2 contract.
type FluidDexV2LogUpdateAuth struct {
	Auth   common.Address
	IsAuth bool
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterLogUpdateAuth is a free log retrieval operation binding the contract event 0xb873643b3104ddd26927dec7f9a08aa22d03ac20ada0295a2de7e1a6c60f2a51.
//
// Solidity: event LogUpdateAuth(address indexed auth, bool indexed isAuth)
func (_FluidDexV2 *FluidDexV2Filterer) FilterLogUpdateAuth(opts *bind.FilterOpts, auth []common.Address, isAuth []bool) (*FluidDexV2LogUpdateAuthIterator, error) {

	var authRule []interface{}
	for _, authItem := range auth {
		authRule = append(authRule, authItem)
	}
	var isAuthRule []interface{}
	for _, isAuthItem := range isAuth {
		isAuthRule = append(isAuthRule, isAuthItem)
	}

	logs, sub, err := _FluidDexV2.contract.FilterLogs(opts, "LogUpdateAuth", authRule, isAuthRule)
	if err != nil {
		return nil, err
	}
	return &FluidDexV2LogUpdateAuthIterator{contract: _FluidDexV2.contract, event: "LogUpdateAuth", logs: logs, sub: sub}, nil
}

// WatchLogUpdateAuth is a free log subscription operation binding the contract event 0xb873643b3104ddd26927dec7f9a08aa22d03ac20ada0295a2de7e1a6c60f2a51.
//
// Solidity: event LogUpdateAuth(address indexed auth, bool indexed isAuth)
func (_FluidDexV2 *FluidDexV2Filterer) WatchLogUpdateAuth(opts *bind.WatchOpts, sink chan<- *FluidDexV2LogUpdateAuth, auth []common.Address, isAuth []bool) (event.Subscription, error) {

	var authRule []interface{}
	for _, authItem := range auth {
		authRule = append(authRule, authItem)
	}
	var isAuthRule []interface{}
	for _, isAuthItem := range isAuth {
		isAuthRule = append(isAuthRule, isAuthItem)
	}

	logs, sub, err := _FluidDexV2.contract.WatchLogs(opts, "LogUpdateAuth", authRule, isAuthRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FluidDexV2LogUpdateAuth)
				if err := _FluidDexV2.contract.UnpackLog(event, "LogUpdateAuth", log); err != nil {
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

// ParseLogUpdateAuth is a log parse operation binding the contract event 0xb873643b3104ddd26927dec7f9a08aa22d03ac20ada0295a2de7e1a6c60f2a51.
//
// Solidity: event LogUpdateAuth(address indexed auth, bool indexed isAuth)
func (_FluidDexV2 *FluidDexV2Filterer) ParseLogUpdateAuth(log types.Log) (*FluidDexV2LogUpdateAuth, error) {
	event := new(FluidDexV2LogUpdateAuth)
	if err := _FluidDexV2.contract.UnpackLog(event, "LogUpdateAuth", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FluidDexV2LogUpdateDexTypeToAdminImplementationIterator is returned from FilterLogUpdateDexTypeToAdminImplementation and is used to iterate over the raw logs and unpacked data for LogUpdateDexTypeToAdminImplementation events raised by the FluidDexV2 contract.
type FluidDexV2LogUpdateDexTypeToAdminImplementationIterator struct {
	Event *FluidDexV2LogUpdateDexTypeToAdminImplementation // Event containing the contract specifics and raw log

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
func (it *FluidDexV2LogUpdateDexTypeToAdminImplementationIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FluidDexV2LogUpdateDexTypeToAdminImplementation)
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
		it.Event = new(FluidDexV2LogUpdateDexTypeToAdminImplementation)
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
func (it *FluidDexV2LogUpdateDexTypeToAdminImplementationIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FluidDexV2LogUpdateDexTypeToAdminImplementationIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FluidDexV2LogUpdateDexTypeToAdminImplementation represents a LogUpdateDexTypeToAdminImplementation event raised by the FluidDexV2 contract.
type FluidDexV2LogUpdateDexTypeToAdminImplementation struct {
	DexType               *big.Int
	AdminImplementationId *big.Int
	AdminImplementation   common.Address
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterLogUpdateDexTypeToAdminImplementation is a free log retrieval operation binding the contract event 0x287217d76664e35e2cf44549761a95036951c0bd9d735fb6742125c46a05c466.
//
// Solidity: event LogUpdateDexTypeToAdminImplementation(uint256 indexed dexType, uint256 indexed adminImplementationId, address indexed adminImplementation)
func (_FluidDexV2 *FluidDexV2Filterer) FilterLogUpdateDexTypeToAdminImplementation(opts *bind.FilterOpts, dexType []*big.Int, adminImplementationId []*big.Int, adminImplementation []common.Address) (*FluidDexV2LogUpdateDexTypeToAdminImplementationIterator, error) {

	var dexTypeRule []interface{}
	for _, dexTypeItem := range dexType {
		dexTypeRule = append(dexTypeRule, dexTypeItem)
	}
	var adminImplementationIdRule []interface{}
	for _, adminImplementationIdItem := range adminImplementationId {
		adminImplementationIdRule = append(adminImplementationIdRule, adminImplementationIdItem)
	}
	var adminImplementationRule []interface{}
	for _, adminImplementationItem := range adminImplementation {
		adminImplementationRule = append(adminImplementationRule, adminImplementationItem)
	}

	logs, sub, err := _FluidDexV2.contract.FilterLogs(opts, "LogUpdateDexTypeToAdminImplementation", dexTypeRule, adminImplementationIdRule, adminImplementationRule)
	if err != nil {
		return nil, err
	}
	return &FluidDexV2LogUpdateDexTypeToAdminImplementationIterator{contract: _FluidDexV2.contract, event: "LogUpdateDexTypeToAdminImplementation", logs: logs, sub: sub}, nil
}

// WatchLogUpdateDexTypeToAdminImplementation is a free log subscription operation binding the contract event 0x287217d76664e35e2cf44549761a95036951c0bd9d735fb6742125c46a05c466.
//
// Solidity: event LogUpdateDexTypeToAdminImplementation(uint256 indexed dexType, uint256 indexed adminImplementationId, address indexed adminImplementation)
func (_FluidDexV2 *FluidDexV2Filterer) WatchLogUpdateDexTypeToAdminImplementation(opts *bind.WatchOpts, sink chan<- *FluidDexV2LogUpdateDexTypeToAdminImplementation, dexType []*big.Int, adminImplementationId []*big.Int, adminImplementation []common.Address) (event.Subscription, error) {

	var dexTypeRule []interface{}
	for _, dexTypeItem := range dexType {
		dexTypeRule = append(dexTypeRule, dexTypeItem)
	}
	var adminImplementationIdRule []interface{}
	for _, adminImplementationIdItem := range adminImplementationId {
		adminImplementationIdRule = append(adminImplementationIdRule, adminImplementationIdItem)
	}
	var adminImplementationRule []interface{}
	for _, adminImplementationItem := range adminImplementation {
		adminImplementationRule = append(adminImplementationRule, adminImplementationItem)
	}

	logs, sub, err := _FluidDexV2.contract.WatchLogs(opts, "LogUpdateDexTypeToAdminImplementation", dexTypeRule, adminImplementationIdRule, adminImplementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FluidDexV2LogUpdateDexTypeToAdminImplementation)
				if err := _FluidDexV2.contract.UnpackLog(event, "LogUpdateDexTypeToAdminImplementation", log); err != nil {
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

// ParseLogUpdateDexTypeToAdminImplementation is a log parse operation binding the contract event 0x287217d76664e35e2cf44549761a95036951c0bd9d735fb6742125c46a05c466.
//
// Solidity: event LogUpdateDexTypeToAdminImplementation(uint256 indexed dexType, uint256 indexed adminImplementationId, address indexed adminImplementation)
func (_FluidDexV2 *FluidDexV2Filterer) ParseLogUpdateDexTypeToAdminImplementation(log types.Log) (*FluidDexV2LogUpdateDexTypeToAdminImplementation, error) {
	event := new(FluidDexV2LogUpdateDexTypeToAdminImplementation)
	if err := _FluidDexV2.contract.UnpackLog(event, "LogUpdateDexTypeToAdminImplementation", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FluidDexV2LogUpgradedIterator is returned from FilterLogUpgraded and is used to iterate over the raw logs and unpacked data for LogUpgraded events raised by the FluidDexV2 contract.
type FluidDexV2LogUpgradedIterator struct {
	Event *FluidDexV2LogUpgraded // Event containing the contract specifics and raw log

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
func (it *FluidDexV2LogUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FluidDexV2LogUpgraded)
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
		it.Event = new(FluidDexV2LogUpgraded)
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
func (it *FluidDexV2LogUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FluidDexV2LogUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FluidDexV2LogUpgraded represents a LogUpgraded event raised by the FluidDexV2 contract.
type FluidDexV2LogUpgraded struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterLogUpgraded is a free log retrieval operation binding the contract event 0x12f2b2c15c8ad2b6746bfa03dae35f5c77654a6048af918d27730c010cf338d3.
//
// Solidity: event LogUpgraded(address indexed implementation)
func (_FluidDexV2 *FluidDexV2Filterer) FilterLogUpgraded(opts *bind.FilterOpts, implementation []common.Address) (*FluidDexV2LogUpgradedIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _FluidDexV2.contract.FilterLogs(opts, "LogUpgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return &FluidDexV2LogUpgradedIterator{contract: _FluidDexV2.contract, event: "LogUpgraded", logs: logs, sub: sub}, nil
}

// WatchLogUpgraded is a free log subscription operation binding the contract event 0x12f2b2c15c8ad2b6746bfa03dae35f5c77654a6048af918d27730c010cf338d3.
//
// Solidity: event LogUpgraded(address indexed implementation)
func (_FluidDexV2 *FluidDexV2Filterer) WatchLogUpgraded(opts *bind.WatchOpts, sink chan<- *FluidDexV2LogUpgraded, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _FluidDexV2.contract.WatchLogs(opts, "LogUpgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FluidDexV2LogUpgraded)
				if err := _FluidDexV2.contract.UnpackLog(event, "LogUpgraded", log); err != nil {
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

// ParseLogUpgraded is a log parse operation binding the contract event 0x12f2b2c15c8ad2b6746bfa03dae35f5c77654a6048af918d27730c010cf338d3.
//
// Solidity: event LogUpgraded(address indexed implementation)
func (_FluidDexV2 *FluidDexV2Filterer) ParseLogUpgraded(log types.Log) (*FluidDexV2LogUpgraded, error) {
	event := new(FluidDexV2LogUpgraded)
	if err := _FluidDexV2.contract.UnpackLog(event, "LogUpgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FluidDexV2LogWithdrawIterator is returned from FilterLogWithdraw and is used to iterate over the raw logs and unpacked data for LogWithdraw events raised by the FluidDexV2 contract.
type FluidDexV2LogWithdrawIterator struct {
	Event *FluidDexV2LogWithdraw // Event containing the contract specifics and raw log

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
func (it *FluidDexV2LogWithdrawIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FluidDexV2LogWithdraw)
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
		it.Event = new(FluidDexV2LogWithdraw)
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
func (it *FluidDexV2LogWithdrawIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FluidDexV2LogWithdrawIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FluidDexV2LogWithdraw represents a LogWithdraw event raised by the FluidDexV2 contract.
type FluidDexV2LogWithdraw struct {
	DexType              *big.Int
	DexId                [32]byte
	User                 common.Address
	TickLower            *big.Int
	TickUpper            *big.Int
	PositionSalt         [32]byte
	Amount0              *big.Int
	Amount1              *big.Int
	FeeAccruedToken0     *big.Int
	FeeAccruedToken1     *big.Int
	LiquidityDecreaseRaw *big.Int
	Raw                  types.Log // Blockchain specific contextual infos
}

// FilterLogWithdraw is a free log retrieval operation binding the contract event 0x7246e8a3c99cbc9d465bee78847ddc3d3dbb043e83552f9afe368eccc29ffc18.
//
// Solidity: event LogWithdraw(uint256 dexType, bytes32 dexId, address user, int24 tickLower, int24 tickUpper, bytes32 positionSalt, uint256 amount0, uint256 amount1, uint256 feeAccruedToken0, uint256 feeAccruedToken1, uint256 liquidityDecreaseRaw)
func (_FluidDexV2 *FluidDexV2Filterer) FilterLogWithdraw(opts *bind.FilterOpts) (*FluidDexV2LogWithdrawIterator, error) {

	logs, sub, err := _FluidDexV2.contract.FilterLogs(opts, "LogWithdraw")
	if err != nil {
		return nil, err
	}
	return &FluidDexV2LogWithdrawIterator{contract: _FluidDexV2.contract, event: "LogWithdraw", logs: logs, sub: sub}, nil
}

// WatchLogWithdraw is a free log subscription operation binding the contract event 0x7246e8a3c99cbc9d465bee78847ddc3d3dbb043e83552f9afe368eccc29ffc18.
//
// Solidity: event LogWithdraw(uint256 dexType, bytes32 dexId, address user, int24 tickLower, int24 tickUpper, bytes32 positionSalt, uint256 amount0, uint256 amount1, uint256 feeAccruedToken0, uint256 feeAccruedToken1, uint256 liquidityDecreaseRaw)
func (_FluidDexV2 *FluidDexV2Filterer) WatchLogWithdraw(opts *bind.WatchOpts, sink chan<- *FluidDexV2LogWithdraw) (event.Subscription, error) {

	logs, sub, err := _FluidDexV2.contract.WatchLogs(opts, "LogWithdraw")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FluidDexV2LogWithdraw)
				if err := _FluidDexV2.contract.UnpackLog(event, "LogWithdraw", log); err != nil {
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

// ParseLogWithdraw is a log parse operation binding the contract event 0x7246e8a3c99cbc9d465bee78847ddc3d3dbb043e83552f9afe368eccc29ffc18.
//
// Solidity: event LogWithdraw(uint256 dexType, bytes32 dexId, address user, int24 tickLower, int24 tickUpper, bytes32 positionSalt, uint256 amount0, uint256 amount1, uint256 feeAccruedToken0, uint256 feeAccruedToken1, uint256 liquidityDecreaseRaw)
func (_FluidDexV2 *FluidDexV2Filterer) ParseLogWithdraw(log types.Log) (*FluidDexV2LogWithdraw, error) {
	event := new(FluidDexV2LogWithdraw)
	if err := _FluidDexV2.contract.UnpackLog(event, "LogWithdraw", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
