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

// NomiStablePoolMetaData contains all meta data concerning the NomiStablePool contract.
var NomiStablePoolMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_token1\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"Burn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"name\":\"Mint\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"oldA\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newA\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"initialTime\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"futureTime\",\"type\":\"uint256\"}],\"name\":\"RampA\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"A\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"t\",\"type\":\"uint256\"}],\"name\":\"StopRampA\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0In\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1In\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0Out\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1Out\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"Swap\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"indexed\":false,\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"}],\"name\":\"Sync\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DOMAIN_SEPARATOR\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MINIMUM_LIQUIDITY\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"PERMIT_TYPEHASH\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"adminFee\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"burn\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"devFee\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"factory\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getA\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenIn\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amountOut\",\"type\":\"uint256\"}],\"name\":\"getAmountIn\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"finalAmountIn\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenIn\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amountIn\",\"type\":\"uint256\"}],\"name\":\"getAmountOut\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"finalAmountOut\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getReserves\",\"outputs\":[{\"internalType\":\"uint112\",\"name\":\"_reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"_reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"_blockTimestampLast\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"mint\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"liquidity\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"nonces\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"name\":\"permit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"_futureA\",\"type\":\"uint32\"},{\"internalType\":\"uint40\",\"name\":\"_futureTime\",\"type\":\"uint40\"}],\"name\":\"rampA\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint128\",\"name\":\"_devFee\",\"type\":\"uint128\"}],\"name\":\"setDevFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"_swapFee\",\"type\":\"uint32\"}],\"name\":\"setSwapFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"skim\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"stopRampA\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount0Out\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1Out\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"swap\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"swapFee\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sync\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"token0\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"token0PrecisionMultiplier\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"token1\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"token1PrecisionMultiplier\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// NomiStablePoolABI is the input ABI used to generate the binding from.
// Deprecated: Use NomiStablePoolMetaData.ABI instead.
var NomiStablePoolABI = NomiStablePoolMetaData.ABI

// NomiStablePool is an auto generated Go binding around an Ethereum contract.
type NomiStablePool struct {
	NomiStablePoolCaller     // Read-only binding to the contract
	NomiStablePoolTransactor // Write-only binding to the contract
	NomiStablePoolFilterer   // Log filterer for contract events
}

// NomiStablePoolCaller is an auto generated read-only Go binding around an Ethereum contract.
type NomiStablePoolCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NomiStablePoolTransactor is an auto generated write-only Go binding around an Ethereum contract.
type NomiStablePoolTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NomiStablePoolFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type NomiStablePoolFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NomiStablePoolSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type NomiStablePoolSession struct {
	Contract     *NomiStablePool   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// NomiStablePoolCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type NomiStablePoolCallerSession struct {
	Contract *NomiStablePoolCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// NomiStablePoolTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type NomiStablePoolTransactorSession struct {
	Contract     *NomiStablePoolTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// NomiStablePoolRaw is an auto generated low-level Go binding around an Ethereum contract.
type NomiStablePoolRaw struct {
	Contract *NomiStablePool // Generic contract binding to access the raw methods on
}

// NomiStablePoolCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type NomiStablePoolCallerRaw struct {
	Contract *NomiStablePoolCaller // Generic read-only contract binding to access the raw methods on
}

// NomiStablePoolTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type NomiStablePoolTransactorRaw struct {
	Contract *NomiStablePoolTransactor // Generic write-only contract binding to access the raw methods on
}

// NewNomiStablePool creates a new instance of NomiStablePool, bound to a specific deployed contract.
func NewNomiStablePool(address common.Address, backend bind.ContractBackend) (*NomiStablePool, error) {
	contract, err := bindNomiStablePool(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &NomiStablePool{NomiStablePoolCaller: NomiStablePoolCaller{contract: contract}, NomiStablePoolTransactor: NomiStablePoolTransactor{contract: contract}, NomiStablePoolFilterer: NomiStablePoolFilterer{contract: contract}}, nil
}

// NewNomiStablePoolCaller creates a new read-only instance of NomiStablePool, bound to a specific deployed contract.
func NewNomiStablePoolCaller(address common.Address, caller bind.ContractCaller) (*NomiStablePoolCaller, error) {
	contract, err := bindNomiStablePool(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &NomiStablePoolCaller{contract: contract}, nil
}

// NewNomiStablePoolTransactor creates a new write-only instance of NomiStablePool, bound to a specific deployed contract.
func NewNomiStablePoolTransactor(address common.Address, transactor bind.ContractTransactor) (*NomiStablePoolTransactor, error) {
	contract, err := bindNomiStablePool(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &NomiStablePoolTransactor{contract: contract}, nil
}

// NewNomiStablePoolFilterer creates a new log filterer instance of NomiStablePool, bound to a specific deployed contract.
func NewNomiStablePoolFilterer(address common.Address, filterer bind.ContractFilterer) (*NomiStablePoolFilterer, error) {
	contract, err := bindNomiStablePool(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &NomiStablePoolFilterer{contract: contract}, nil
}

// bindNomiStablePool binds a generic wrapper to an already deployed contract.
func bindNomiStablePool(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := NomiStablePoolMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NomiStablePool *NomiStablePoolRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NomiStablePool.Contract.NomiStablePoolCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NomiStablePool *NomiStablePoolRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NomiStablePool.Contract.NomiStablePoolTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NomiStablePool *NomiStablePoolRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NomiStablePool.Contract.NomiStablePoolTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NomiStablePool *NomiStablePoolCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NomiStablePool.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NomiStablePool *NomiStablePoolTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NomiStablePool.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NomiStablePool *NomiStablePoolTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NomiStablePool.Contract.contract.Transact(opts, method, params...)
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_NomiStablePool *NomiStablePoolCaller) DOMAINSEPARATOR(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "DOMAIN_SEPARATOR")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_NomiStablePool *NomiStablePoolSession) DOMAINSEPARATOR() ([32]byte, error) {
	return _NomiStablePool.Contract.DOMAINSEPARATOR(&_NomiStablePool.CallOpts)
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_NomiStablePool *NomiStablePoolCallerSession) DOMAINSEPARATOR() ([32]byte, error) {
	return _NomiStablePool.Contract.DOMAINSEPARATOR(&_NomiStablePool.CallOpts)
}

// MINIMUMLIQUIDITY is a free data retrieval call binding the contract method 0xba9a7a56.
//
// Solidity: function MINIMUM_LIQUIDITY() view returns(uint256)
func (_NomiStablePool *NomiStablePoolCaller) MINIMUMLIQUIDITY(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "MINIMUM_LIQUIDITY")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINIMUMLIQUIDITY is a free data retrieval call binding the contract method 0xba9a7a56.
//
// Solidity: function MINIMUM_LIQUIDITY() view returns(uint256)
func (_NomiStablePool *NomiStablePoolSession) MINIMUMLIQUIDITY() (*big.Int, error) {
	return _NomiStablePool.Contract.MINIMUMLIQUIDITY(&_NomiStablePool.CallOpts)
}

// MINIMUMLIQUIDITY is a free data retrieval call binding the contract method 0xba9a7a56.
//
// Solidity: function MINIMUM_LIQUIDITY() view returns(uint256)
func (_NomiStablePool *NomiStablePoolCallerSession) MINIMUMLIQUIDITY() (*big.Int, error) {
	return _NomiStablePool.Contract.MINIMUMLIQUIDITY(&_NomiStablePool.CallOpts)
}

// PERMITTYPEHASH is a free data retrieval call binding the contract method 0x30adf81f.
//
// Solidity: function PERMIT_TYPEHASH() view returns(bytes32)
func (_NomiStablePool *NomiStablePoolCaller) PERMITTYPEHASH(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "PERMIT_TYPEHASH")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// PERMITTYPEHASH is a free data retrieval call binding the contract method 0x30adf81f.
//
// Solidity: function PERMIT_TYPEHASH() view returns(bytes32)
func (_NomiStablePool *NomiStablePoolSession) PERMITTYPEHASH() ([32]byte, error) {
	return _NomiStablePool.Contract.PERMITTYPEHASH(&_NomiStablePool.CallOpts)
}

// PERMITTYPEHASH is a free data retrieval call binding the contract method 0x30adf81f.
//
// Solidity: function PERMIT_TYPEHASH() view returns(bytes32)
func (_NomiStablePool *NomiStablePoolCallerSession) PERMITTYPEHASH() ([32]byte, error) {
	return _NomiStablePool.Contract.PERMITTYPEHASH(&_NomiStablePool.CallOpts)
}

// AdminFee is a free data retrieval call binding the contract method 0xa0be06f9.
//
// Solidity: function adminFee() view returns(uint128)
func (_NomiStablePool *NomiStablePoolCaller) AdminFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "adminFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// AdminFee is a free data retrieval call binding the contract method 0xa0be06f9.
//
// Solidity: function adminFee() view returns(uint128)
func (_NomiStablePool *NomiStablePoolSession) AdminFee() (*big.Int, error) {
	return _NomiStablePool.Contract.AdminFee(&_NomiStablePool.CallOpts)
}

// AdminFee is a free data retrieval call binding the contract method 0xa0be06f9.
//
// Solidity: function adminFee() view returns(uint128)
func (_NomiStablePool *NomiStablePoolCallerSession) AdminFee() (*big.Int, error) {
	return _NomiStablePool.Contract.AdminFee(&_NomiStablePool.CallOpts)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address , address ) view returns(uint256)
func (_NomiStablePool *NomiStablePoolCaller) Allowance(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "allowance", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address , address ) view returns(uint256)
func (_NomiStablePool *NomiStablePoolSession) Allowance(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _NomiStablePool.Contract.Allowance(&_NomiStablePool.CallOpts, arg0, arg1)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address , address ) view returns(uint256)
func (_NomiStablePool *NomiStablePoolCallerSession) Allowance(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _NomiStablePool.Contract.Allowance(&_NomiStablePool.CallOpts, arg0, arg1)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) view returns(uint256)
func (_NomiStablePool *NomiStablePoolCaller) BalanceOf(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "balanceOf", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) view returns(uint256)
func (_NomiStablePool *NomiStablePoolSession) BalanceOf(arg0 common.Address) (*big.Int, error) {
	return _NomiStablePool.Contract.BalanceOf(&_NomiStablePool.CallOpts, arg0)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) view returns(uint256)
func (_NomiStablePool *NomiStablePoolCallerSession) BalanceOf(arg0 common.Address) (*big.Int, error) {
	return _NomiStablePool.Contract.BalanceOf(&_NomiStablePool.CallOpts, arg0)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_NomiStablePool *NomiStablePoolCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_NomiStablePool *NomiStablePoolSession) Decimals() (uint8, error) {
	return _NomiStablePool.Contract.Decimals(&_NomiStablePool.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_NomiStablePool *NomiStablePoolCallerSession) Decimals() (uint8, error) {
	return _NomiStablePool.Contract.Decimals(&_NomiStablePool.CallOpts)
}

// DevFee is a free data retrieval call binding the contract method 0x6827e764.
//
// Solidity: function devFee() view returns(uint128)
func (_NomiStablePool *NomiStablePoolCaller) DevFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "devFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DevFee is a free data retrieval call binding the contract method 0x6827e764.
//
// Solidity: function devFee() view returns(uint128)
func (_NomiStablePool *NomiStablePoolSession) DevFee() (*big.Int, error) {
	return _NomiStablePool.Contract.DevFee(&_NomiStablePool.CallOpts)
}

// DevFee is a free data retrieval call binding the contract method 0x6827e764.
//
// Solidity: function devFee() view returns(uint128)
func (_NomiStablePool *NomiStablePoolCallerSession) DevFee() (*big.Int, error) {
	return _NomiStablePool.Contract.DevFee(&_NomiStablePool.CallOpts)
}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_NomiStablePool *NomiStablePoolCaller) Factory(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "factory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_NomiStablePool *NomiStablePoolSession) Factory() (common.Address, error) {
	return _NomiStablePool.Contract.Factory(&_NomiStablePool.CallOpts)
}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_NomiStablePool *NomiStablePoolCallerSession) Factory() (common.Address, error) {
	return _NomiStablePool.Contract.Factory(&_NomiStablePool.CallOpts)
}

// GetA is a free data retrieval call binding the contract method 0xd46300fd.
//
// Solidity: function getA() view returns(uint256)
func (_NomiStablePool *NomiStablePoolCaller) GetA(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "getA")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetA is a free data retrieval call binding the contract method 0xd46300fd.
//
// Solidity: function getA() view returns(uint256)
func (_NomiStablePool *NomiStablePoolSession) GetA() (*big.Int, error) {
	return _NomiStablePool.Contract.GetA(&_NomiStablePool.CallOpts)
}

// GetA is a free data retrieval call binding the contract method 0xd46300fd.
//
// Solidity: function getA() view returns(uint256)
func (_NomiStablePool *NomiStablePoolCallerSession) GetA() (*big.Int, error) {
	return _NomiStablePool.Contract.GetA(&_NomiStablePool.CallOpts)
}

// GetAmountIn is a free data retrieval call binding the contract method 0x632db21c.
//
// Solidity: function getAmountIn(address tokenIn, uint256 amountOut) view returns(uint256 finalAmountIn)
func (_NomiStablePool *NomiStablePoolCaller) GetAmountIn(opts *bind.CallOpts, tokenIn common.Address, amountOut *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "getAmountIn", tokenIn, amountOut)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetAmountIn is a free data retrieval call binding the contract method 0x632db21c.
//
// Solidity: function getAmountIn(address tokenIn, uint256 amountOut) view returns(uint256 finalAmountIn)
func (_NomiStablePool *NomiStablePoolSession) GetAmountIn(tokenIn common.Address, amountOut *big.Int) (*big.Int, error) {
	return _NomiStablePool.Contract.GetAmountIn(&_NomiStablePool.CallOpts, tokenIn, amountOut)
}

// GetAmountIn is a free data retrieval call binding the contract method 0x632db21c.
//
// Solidity: function getAmountIn(address tokenIn, uint256 amountOut) view returns(uint256 finalAmountIn)
func (_NomiStablePool *NomiStablePoolCallerSession) GetAmountIn(tokenIn common.Address, amountOut *big.Int) (*big.Int, error) {
	return _NomiStablePool.Contract.GetAmountIn(&_NomiStablePool.CallOpts, tokenIn, amountOut)
}

// GetAmountOut is a free data retrieval call binding the contract method 0xca706bcf.
//
// Solidity: function getAmountOut(address tokenIn, uint256 amountIn) view returns(uint256 finalAmountOut)
func (_NomiStablePool *NomiStablePoolCaller) GetAmountOut(opts *bind.CallOpts, tokenIn common.Address, amountIn *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "getAmountOut", tokenIn, amountIn)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetAmountOut is a free data retrieval call binding the contract method 0xca706bcf.
//
// Solidity: function getAmountOut(address tokenIn, uint256 amountIn) view returns(uint256 finalAmountOut)
func (_NomiStablePool *NomiStablePoolSession) GetAmountOut(tokenIn common.Address, amountIn *big.Int) (*big.Int, error) {
	return _NomiStablePool.Contract.GetAmountOut(&_NomiStablePool.CallOpts, tokenIn, amountIn)
}

// GetAmountOut is a free data retrieval call binding the contract method 0xca706bcf.
//
// Solidity: function getAmountOut(address tokenIn, uint256 amountIn) view returns(uint256 finalAmountOut)
func (_NomiStablePool *NomiStablePoolCallerSession) GetAmountOut(tokenIn common.Address, amountIn *big.Int) (*big.Int, error) {
	return _NomiStablePool.Contract.GetAmountOut(&_NomiStablePool.CallOpts, tokenIn, amountIn)
}

// GetReserves is a free data retrieval call binding the contract method 0x0902f1ac.
//
// Solidity: function getReserves() view returns(uint112 _reserve0, uint112 _reserve1, uint32 _blockTimestampLast)
func (_NomiStablePool *NomiStablePoolCaller) GetReserves(opts *bind.CallOpts) (struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "getReserves")

	outstruct := new(struct {
		Reserve0           *big.Int
		Reserve1           *big.Int
		BlockTimestampLast uint32
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Reserve0 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Reserve1 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.BlockTimestampLast = *abi.ConvertType(out[2], new(uint32)).(*uint32)

	return *outstruct, err

}

// GetReserves is a free data retrieval call binding the contract method 0x0902f1ac.
//
// Solidity: function getReserves() view returns(uint112 _reserve0, uint112 _reserve1, uint32 _blockTimestampLast)
func (_NomiStablePool *NomiStablePoolSession) GetReserves() (struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}, error) {
	return _NomiStablePool.Contract.GetReserves(&_NomiStablePool.CallOpts)
}

// GetReserves is a free data retrieval call binding the contract method 0x0902f1ac.
//
// Solidity: function getReserves() view returns(uint112 _reserve0, uint112 _reserve1, uint32 _blockTimestampLast)
func (_NomiStablePool *NomiStablePoolCallerSession) GetReserves() (struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}, error) {
	return _NomiStablePool.Contract.GetReserves(&_NomiStablePool.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_NomiStablePool *NomiStablePoolCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_NomiStablePool *NomiStablePoolSession) Name() (string, error) {
	return _NomiStablePool.Contract.Name(&_NomiStablePool.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_NomiStablePool *NomiStablePoolCallerSession) Name() (string, error) {
	return _NomiStablePool.Contract.Name(&_NomiStablePool.CallOpts)
}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces(address ) view returns(uint256)
func (_NomiStablePool *NomiStablePoolCaller) Nonces(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "nonces", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces(address ) view returns(uint256)
func (_NomiStablePool *NomiStablePoolSession) Nonces(arg0 common.Address) (*big.Int, error) {
	return _NomiStablePool.Contract.Nonces(&_NomiStablePool.CallOpts, arg0)
}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces(address ) view returns(uint256)
func (_NomiStablePool *NomiStablePoolCallerSession) Nonces(arg0 common.Address) (*big.Int, error) {
	return _NomiStablePool.Contract.Nonces(&_NomiStablePool.CallOpts, arg0)
}

// SwapFee is a free data retrieval call binding the contract method 0x54cf2aeb.
//
// Solidity: function swapFee() view returns(uint32)
func (_NomiStablePool *NomiStablePoolCaller) SwapFee(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "swapFee")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// SwapFee is a free data retrieval call binding the contract method 0x54cf2aeb.
//
// Solidity: function swapFee() view returns(uint32)
func (_NomiStablePool *NomiStablePoolSession) SwapFee() (uint32, error) {
	return _NomiStablePool.Contract.SwapFee(&_NomiStablePool.CallOpts)
}

// SwapFee is a free data retrieval call binding the contract method 0x54cf2aeb.
//
// Solidity: function swapFee() view returns(uint32)
func (_NomiStablePool *NomiStablePoolCallerSession) SwapFee() (uint32, error) {
	return _NomiStablePool.Contract.SwapFee(&_NomiStablePool.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_NomiStablePool *NomiStablePoolCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_NomiStablePool *NomiStablePoolSession) Symbol() (string, error) {
	return _NomiStablePool.Contract.Symbol(&_NomiStablePool.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_NomiStablePool *NomiStablePoolCallerSession) Symbol() (string, error) {
	return _NomiStablePool.Contract.Symbol(&_NomiStablePool.CallOpts)
}

// Token0 is a free data retrieval call binding the contract method 0x0dfe1681.
//
// Solidity: function token0() view returns(address)
func (_NomiStablePool *NomiStablePoolCaller) Token0(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "token0")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Token0 is a free data retrieval call binding the contract method 0x0dfe1681.
//
// Solidity: function token0() view returns(address)
func (_NomiStablePool *NomiStablePoolSession) Token0() (common.Address, error) {
	return _NomiStablePool.Contract.Token0(&_NomiStablePool.CallOpts)
}

// Token0 is a free data retrieval call binding the contract method 0x0dfe1681.
//
// Solidity: function token0() view returns(address)
func (_NomiStablePool *NomiStablePoolCallerSession) Token0() (common.Address, error) {
	return _NomiStablePool.Contract.Token0(&_NomiStablePool.CallOpts)
}

// Token0PrecisionMultiplier is a free data retrieval call binding the contract method 0xbaa8c7cb.
//
// Solidity: function token0PrecisionMultiplier() view returns(uint128)
func (_NomiStablePool *NomiStablePoolCaller) Token0PrecisionMultiplier(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "token0PrecisionMultiplier")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Token0PrecisionMultiplier is a free data retrieval call binding the contract method 0xbaa8c7cb.
//
// Solidity: function token0PrecisionMultiplier() view returns(uint128)
func (_NomiStablePool *NomiStablePoolSession) Token0PrecisionMultiplier() (*big.Int, error) {
	return _NomiStablePool.Contract.Token0PrecisionMultiplier(&_NomiStablePool.CallOpts)
}

// Token0PrecisionMultiplier is a free data retrieval call binding the contract method 0xbaa8c7cb.
//
// Solidity: function token0PrecisionMultiplier() view returns(uint128)
func (_NomiStablePool *NomiStablePoolCallerSession) Token0PrecisionMultiplier() (*big.Int, error) {
	return _NomiStablePool.Contract.Token0PrecisionMultiplier(&_NomiStablePool.CallOpts)
}

// Token1 is a free data retrieval call binding the contract method 0xd21220a7.
//
// Solidity: function token1() view returns(address)
func (_NomiStablePool *NomiStablePoolCaller) Token1(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "token1")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Token1 is a free data retrieval call binding the contract method 0xd21220a7.
//
// Solidity: function token1() view returns(address)
func (_NomiStablePool *NomiStablePoolSession) Token1() (common.Address, error) {
	return _NomiStablePool.Contract.Token1(&_NomiStablePool.CallOpts)
}

// Token1 is a free data retrieval call binding the contract method 0xd21220a7.
//
// Solidity: function token1() view returns(address)
func (_NomiStablePool *NomiStablePoolCallerSession) Token1() (common.Address, error) {
	return _NomiStablePool.Contract.Token1(&_NomiStablePool.CallOpts)
}

// Token1PrecisionMultiplier is a free data retrieval call binding the contract method 0x4e25dc47.
//
// Solidity: function token1PrecisionMultiplier() view returns(uint128)
func (_NomiStablePool *NomiStablePoolCaller) Token1PrecisionMultiplier(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "token1PrecisionMultiplier")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Token1PrecisionMultiplier is a free data retrieval call binding the contract method 0x4e25dc47.
//
// Solidity: function token1PrecisionMultiplier() view returns(uint128)
func (_NomiStablePool *NomiStablePoolSession) Token1PrecisionMultiplier() (*big.Int, error) {
	return _NomiStablePool.Contract.Token1PrecisionMultiplier(&_NomiStablePool.CallOpts)
}

// Token1PrecisionMultiplier is a free data retrieval call binding the contract method 0x4e25dc47.
//
// Solidity: function token1PrecisionMultiplier() view returns(uint128)
func (_NomiStablePool *NomiStablePoolCallerSession) Token1PrecisionMultiplier() (*big.Int, error) {
	return _NomiStablePool.Contract.Token1PrecisionMultiplier(&_NomiStablePool.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_NomiStablePool *NomiStablePoolCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NomiStablePool.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_NomiStablePool *NomiStablePoolSession) TotalSupply() (*big.Int, error) {
	return _NomiStablePool.Contract.TotalSupply(&_NomiStablePool.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_NomiStablePool *NomiStablePoolCallerSession) TotalSupply() (*big.Int, error) {
	return _NomiStablePool.Contract.TotalSupply(&_NomiStablePool.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_NomiStablePool *NomiStablePoolTransactor) Approve(opts *bind.TransactOpts, spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.contract.Transact(opts, "approve", spender, value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_NomiStablePool *NomiStablePoolSession) Approve(spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.Contract.Approve(&_NomiStablePool.TransactOpts, spender, value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_NomiStablePool *NomiStablePoolTransactorSession) Approve(spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.Contract.Approve(&_NomiStablePool.TransactOpts, spender, value)
}

// Burn is a paid mutator transaction binding the contract method 0x89afcb44.
//
// Solidity: function burn(address to) returns(uint256 amount0, uint256 amount1)
func (_NomiStablePool *NomiStablePoolTransactor) Burn(opts *bind.TransactOpts, to common.Address) (*types.Transaction, error) {
	return _NomiStablePool.contract.Transact(opts, "burn", to)
}

// Burn is a paid mutator transaction binding the contract method 0x89afcb44.
//
// Solidity: function burn(address to) returns(uint256 amount0, uint256 amount1)
func (_NomiStablePool *NomiStablePoolSession) Burn(to common.Address) (*types.Transaction, error) {
	return _NomiStablePool.Contract.Burn(&_NomiStablePool.TransactOpts, to)
}

// Burn is a paid mutator transaction binding the contract method 0x89afcb44.
//
// Solidity: function burn(address to) returns(uint256 amount0, uint256 amount1)
func (_NomiStablePool *NomiStablePoolTransactorSession) Burn(to common.Address) (*types.Transaction, error) {
	return _NomiStablePool.Contract.Burn(&_NomiStablePool.TransactOpts, to)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_NomiStablePool *NomiStablePoolTransactor) DecreaseAllowance(opts *bind.TransactOpts, spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.contract.Transact(opts, "decreaseAllowance", spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_NomiStablePool *NomiStablePoolSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.Contract.DecreaseAllowance(&_NomiStablePool.TransactOpts, spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_NomiStablePool *NomiStablePoolTransactorSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.Contract.DecreaseAllowance(&_NomiStablePool.TransactOpts, spender, subtractedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_NomiStablePool *NomiStablePoolTransactor) IncreaseAllowance(opts *bind.TransactOpts, spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.contract.Transact(opts, "increaseAllowance", spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_NomiStablePool *NomiStablePoolSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.Contract.IncreaseAllowance(&_NomiStablePool.TransactOpts, spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_NomiStablePool *NomiStablePoolTransactorSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.Contract.IncreaseAllowance(&_NomiStablePool.TransactOpts, spender, addedValue)
}

// Mint is a paid mutator transaction binding the contract method 0x6a627842.
//
// Solidity: function mint(address to) returns(uint256 liquidity)
func (_NomiStablePool *NomiStablePoolTransactor) Mint(opts *bind.TransactOpts, to common.Address) (*types.Transaction, error) {
	return _NomiStablePool.contract.Transact(opts, "mint", to)
}

// Mint is a paid mutator transaction binding the contract method 0x6a627842.
//
// Solidity: function mint(address to) returns(uint256 liquidity)
func (_NomiStablePool *NomiStablePoolSession) Mint(to common.Address) (*types.Transaction, error) {
	return _NomiStablePool.Contract.Mint(&_NomiStablePool.TransactOpts, to)
}

// Mint is a paid mutator transaction binding the contract method 0x6a627842.
//
// Solidity: function mint(address to) returns(uint256 liquidity)
func (_NomiStablePool *NomiStablePoolTransactorSession) Mint(to common.Address) (*types.Transaction, error) {
	return _NomiStablePool.Contract.Mint(&_NomiStablePool.TransactOpts, to)
}

// Permit is a paid mutator transaction binding the contract method 0xd505accf.
//
// Solidity: function permit(address owner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_NomiStablePool *NomiStablePoolTransactor) Permit(opts *bind.TransactOpts, owner common.Address, spender common.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _NomiStablePool.contract.Transact(opts, "permit", owner, spender, value, deadline, v, r, s)
}

// Permit is a paid mutator transaction binding the contract method 0xd505accf.
//
// Solidity: function permit(address owner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_NomiStablePool *NomiStablePoolSession) Permit(owner common.Address, spender common.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _NomiStablePool.Contract.Permit(&_NomiStablePool.TransactOpts, owner, spender, value, deadline, v, r, s)
}

// Permit is a paid mutator transaction binding the contract method 0xd505accf.
//
// Solidity: function permit(address owner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_NomiStablePool *NomiStablePoolTransactorSession) Permit(owner common.Address, spender common.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _NomiStablePool.Contract.Permit(&_NomiStablePool.TransactOpts, owner, spender, value, deadline, v, r, s)
}

// RampA is a paid mutator transaction binding the contract method 0x73c48bb5.
//
// Solidity: function rampA(uint32 _futureA, uint40 _futureTime) returns()
func (_NomiStablePool *NomiStablePoolTransactor) RampA(opts *bind.TransactOpts, _futureA uint32, _futureTime *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.contract.Transact(opts, "rampA", _futureA, _futureTime)
}

// RampA is a paid mutator transaction binding the contract method 0x73c48bb5.
//
// Solidity: function rampA(uint32 _futureA, uint40 _futureTime) returns()
func (_NomiStablePool *NomiStablePoolSession) RampA(_futureA uint32, _futureTime *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.Contract.RampA(&_NomiStablePool.TransactOpts, _futureA, _futureTime)
}

// RampA is a paid mutator transaction binding the contract method 0x73c48bb5.
//
// Solidity: function rampA(uint32 _futureA, uint40 _futureTime) returns()
func (_NomiStablePool *NomiStablePoolTransactorSession) RampA(_futureA uint32, _futureTime *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.Contract.RampA(&_NomiStablePool.TransactOpts, _futureA, _futureTime)
}

// SetDevFee is a paid mutator transaction binding the contract method 0x111f8ef3.
//
// Solidity: function setDevFee(uint128 _devFee) returns()
func (_NomiStablePool *NomiStablePoolTransactor) SetDevFee(opts *bind.TransactOpts, _devFee *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.contract.Transact(opts, "setDevFee", _devFee)
}

// SetDevFee is a paid mutator transaction binding the contract method 0x111f8ef3.
//
// Solidity: function setDevFee(uint128 _devFee) returns()
func (_NomiStablePool *NomiStablePoolSession) SetDevFee(_devFee *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.Contract.SetDevFee(&_NomiStablePool.TransactOpts, _devFee)
}

// SetDevFee is a paid mutator transaction binding the contract method 0x111f8ef3.
//
// Solidity: function setDevFee(uint128 _devFee) returns()
func (_NomiStablePool *NomiStablePoolTransactorSession) SetDevFee(_devFee *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.Contract.SetDevFee(&_NomiStablePool.TransactOpts, _devFee)
}

// SetSwapFee is a paid mutator transaction binding the contract method 0xd6d788c3.
//
// Solidity: function setSwapFee(uint32 _swapFee) returns()
func (_NomiStablePool *NomiStablePoolTransactor) SetSwapFee(opts *bind.TransactOpts, _swapFee uint32) (*types.Transaction, error) {
	return _NomiStablePool.contract.Transact(opts, "setSwapFee", _swapFee)
}

// SetSwapFee is a paid mutator transaction binding the contract method 0xd6d788c3.
//
// Solidity: function setSwapFee(uint32 _swapFee) returns()
func (_NomiStablePool *NomiStablePoolSession) SetSwapFee(_swapFee uint32) (*types.Transaction, error) {
	return _NomiStablePool.Contract.SetSwapFee(&_NomiStablePool.TransactOpts, _swapFee)
}

// SetSwapFee is a paid mutator transaction binding the contract method 0xd6d788c3.
//
// Solidity: function setSwapFee(uint32 _swapFee) returns()
func (_NomiStablePool *NomiStablePoolTransactorSession) SetSwapFee(_swapFee uint32) (*types.Transaction, error) {
	return _NomiStablePool.Contract.SetSwapFee(&_NomiStablePool.TransactOpts, _swapFee)
}

// Skim is a paid mutator transaction binding the contract method 0xbc25cf77.
//
// Solidity: function skim(address to) returns()
func (_NomiStablePool *NomiStablePoolTransactor) Skim(opts *bind.TransactOpts, to common.Address) (*types.Transaction, error) {
	return _NomiStablePool.contract.Transact(opts, "skim", to)
}

// Skim is a paid mutator transaction binding the contract method 0xbc25cf77.
//
// Solidity: function skim(address to) returns()
func (_NomiStablePool *NomiStablePoolSession) Skim(to common.Address) (*types.Transaction, error) {
	return _NomiStablePool.Contract.Skim(&_NomiStablePool.TransactOpts, to)
}

// Skim is a paid mutator transaction binding the contract method 0xbc25cf77.
//
// Solidity: function skim(address to) returns()
func (_NomiStablePool *NomiStablePoolTransactorSession) Skim(to common.Address) (*types.Transaction, error) {
	return _NomiStablePool.Contract.Skim(&_NomiStablePool.TransactOpts, to)
}

// StopRampA is a paid mutator transaction binding the contract method 0xc4db7fa0.
//
// Solidity: function stopRampA() returns()
func (_NomiStablePool *NomiStablePoolTransactor) StopRampA(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NomiStablePool.contract.Transact(opts, "stopRampA")
}

// StopRampA is a paid mutator transaction binding the contract method 0xc4db7fa0.
//
// Solidity: function stopRampA() returns()
func (_NomiStablePool *NomiStablePoolSession) StopRampA() (*types.Transaction, error) {
	return _NomiStablePool.Contract.StopRampA(&_NomiStablePool.TransactOpts)
}

// StopRampA is a paid mutator transaction binding the contract method 0xc4db7fa0.
//
// Solidity: function stopRampA() returns()
func (_NomiStablePool *NomiStablePoolTransactorSession) StopRampA() (*types.Transaction, error) {
	return _NomiStablePool.Contract.StopRampA(&_NomiStablePool.TransactOpts)
}

// Swap is a paid mutator transaction binding the contract method 0x022c0d9f.
//
// Solidity: function swap(uint256 amount0Out, uint256 amount1Out, address to, bytes data) returns()
func (_NomiStablePool *NomiStablePoolTransactor) Swap(opts *bind.TransactOpts, amount0Out *big.Int, amount1Out *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _NomiStablePool.contract.Transact(opts, "swap", amount0Out, amount1Out, to, data)
}

// Swap is a paid mutator transaction binding the contract method 0x022c0d9f.
//
// Solidity: function swap(uint256 amount0Out, uint256 amount1Out, address to, bytes data) returns()
func (_NomiStablePool *NomiStablePoolSession) Swap(amount0Out *big.Int, amount1Out *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _NomiStablePool.Contract.Swap(&_NomiStablePool.TransactOpts, amount0Out, amount1Out, to, data)
}

// Swap is a paid mutator transaction binding the contract method 0x022c0d9f.
//
// Solidity: function swap(uint256 amount0Out, uint256 amount1Out, address to, bytes data) returns()
func (_NomiStablePool *NomiStablePoolTransactorSession) Swap(amount0Out *big.Int, amount1Out *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _NomiStablePool.Contract.Swap(&_NomiStablePool.TransactOpts, amount0Out, amount1Out, to, data)
}

// Sync is a paid mutator transaction binding the contract method 0xfff6cae9.
//
// Solidity: function sync() returns()
func (_NomiStablePool *NomiStablePoolTransactor) Sync(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NomiStablePool.contract.Transact(opts, "sync")
}

// Sync is a paid mutator transaction binding the contract method 0xfff6cae9.
//
// Solidity: function sync() returns()
func (_NomiStablePool *NomiStablePoolSession) Sync() (*types.Transaction, error) {
	return _NomiStablePool.Contract.Sync(&_NomiStablePool.TransactOpts)
}

// Sync is a paid mutator transaction binding the contract method 0xfff6cae9.
//
// Solidity: function sync() returns()
func (_NomiStablePool *NomiStablePoolTransactorSession) Sync() (*types.Transaction, error) {
	return _NomiStablePool.Contract.Sync(&_NomiStablePool.TransactOpts)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_NomiStablePool *NomiStablePoolTransactor) Transfer(opts *bind.TransactOpts, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.contract.Transact(opts, "transfer", to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_NomiStablePool *NomiStablePoolSession) Transfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.Contract.Transfer(&_NomiStablePool.TransactOpts, to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_NomiStablePool *NomiStablePoolTransactorSession) Transfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.Contract.Transfer(&_NomiStablePool.TransactOpts, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_NomiStablePool *NomiStablePoolTransactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.contract.Transact(opts, "transferFrom", from, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_NomiStablePool *NomiStablePoolSession) TransferFrom(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.Contract.TransferFrom(&_NomiStablePool.TransactOpts, from, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_NomiStablePool *NomiStablePoolTransactorSession) TransferFrom(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _NomiStablePool.Contract.TransferFrom(&_NomiStablePool.TransactOpts, from, to, value)
}

// NomiStablePoolApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the NomiStablePool contract.
type NomiStablePoolApprovalIterator struct {
	Event *NomiStablePoolApproval // Event containing the contract specifics and raw log

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
func (it *NomiStablePoolApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NomiStablePoolApproval)
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
		it.Event = new(NomiStablePoolApproval)
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
func (it *NomiStablePoolApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NomiStablePoolApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NomiStablePoolApproval represents a Approval event raised by the NomiStablePool contract.
type NomiStablePoolApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_NomiStablePool *NomiStablePoolFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*NomiStablePoolApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _NomiStablePool.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &NomiStablePoolApprovalIterator{contract: _NomiStablePool.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_NomiStablePool *NomiStablePoolFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *NomiStablePoolApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _NomiStablePool.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NomiStablePoolApproval)
				if err := _NomiStablePool.contract.UnpackLog(event, "Approval", log); err != nil {
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

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_NomiStablePool *NomiStablePoolFilterer) ParseApproval(log types.Log) (*NomiStablePoolApproval, error) {
	event := new(NomiStablePoolApproval)
	if err := _NomiStablePool.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NomiStablePoolBurnIterator is returned from FilterBurn and is used to iterate over the raw logs and unpacked data for Burn events raised by the NomiStablePool contract.
type NomiStablePoolBurnIterator struct {
	Event *NomiStablePoolBurn // Event containing the contract specifics and raw log

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
func (it *NomiStablePoolBurnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NomiStablePoolBurn)
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
		it.Event = new(NomiStablePoolBurn)
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
func (it *NomiStablePoolBurnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NomiStablePoolBurnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NomiStablePoolBurn represents a Burn event raised by the NomiStablePool contract.
type NomiStablePoolBurn struct {
	Sender  common.Address
	Amount0 *big.Int
	Amount1 *big.Int
	To      common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterBurn is a free log retrieval operation binding the contract event 0xdccd412f0b1252819cb1fd330b93224ca42612892bb3f4f789976e6d81936496.
//
// Solidity: event Burn(address indexed sender, uint256 amount0, uint256 amount1, address indexed to)
func (_NomiStablePool *NomiStablePoolFilterer) FilterBurn(opts *bind.FilterOpts, sender []common.Address, to []common.Address) (*NomiStablePoolBurnIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _NomiStablePool.contract.FilterLogs(opts, "Burn", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return &NomiStablePoolBurnIterator{contract: _NomiStablePool.contract, event: "Burn", logs: logs, sub: sub}, nil
}

// WatchBurn is a free log subscription operation binding the contract event 0xdccd412f0b1252819cb1fd330b93224ca42612892bb3f4f789976e6d81936496.
//
// Solidity: event Burn(address indexed sender, uint256 amount0, uint256 amount1, address indexed to)
func (_NomiStablePool *NomiStablePoolFilterer) WatchBurn(opts *bind.WatchOpts, sink chan<- *NomiStablePoolBurn, sender []common.Address, to []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _NomiStablePool.contract.WatchLogs(opts, "Burn", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NomiStablePoolBurn)
				if err := _NomiStablePool.contract.UnpackLog(event, "Burn", log); err != nil {
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

// ParseBurn is a log parse operation binding the contract event 0xdccd412f0b1252819cb1fd330b93224ca42612892bb3f4f789976e6d81936496.
//
// Solidity: event Burn(address indexed sender, uint256 amount0, uint256 amount1, address indexed to)
func (_NomiStablePool *NomiStablePoolFilterer) ParseBurn(log types.Log) (*NomiStablePoolBurn, error) {
	event := new(NomiStablePoolBurn)
	if err := _NomiStablePool.contract.UnpackLog(event, "Burn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NomiStablePoolMintIterator is returned from FilterMint and is used to iterate over the raw logs and unpacked data for Mint events raised by the NomiStablePool contract.
type NomiStablePoolMintIterator struct {
	Event *NomiStablePoolMint // Event containing the contract specifics and raw log

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
func (it *NomiStablePoolMintIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NomiStablePoolMint)
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
		it.Event = new(NomiStablePoolMint)
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
func (it *NomiStablePoolMintIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NomiStablePoolMintIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NomiStablePoolMint represents a Mint event raised by the NomiStablePool contract.
type NomiStablePoolMint struct {
	Sender  common.Address
	Amount0 *big.Int
	Amount1 *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterMint is a free log retrieval operation binding the contract event 0x4c209b5fc8ad50758f13e2e1088ba56a560dff690a1c6fef26394f4c03821c4f.
//
// Solidity: event Mint(address indexed sender, uint256 amount0, uint256 amount1)
func (_NomiStablePool *NomiStablePoolFilterer) FilterMint(opts *bind.FilterOpts, sender []common.Address) (*NomiStablePoolMintIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NomiStablePool.contract.FilterLogs(opts, "Mint", senderRule)
	if err != nil {
		return nil, err
	}
	return &NomiStablePoolMintIterator{contract: _NomiStablePool.contract, event: "Mint", logs: logs, sub: sub}, nil
}

// WatchMint is a free log subscription operation binding the contract event 0x4c209b5fc8ad50758f13e2e1088ba56a560dff690a1c6fef26394f4c03821c4f.
//
// Solidity: event Mint(address indexed sender, uint256 amount0, uint256 amount1)
func (_NomiStablePool *NomiStablePoolFilterer) WatchMint(opts *bind.WatchOpts, sink chan<- *NomiStablePoolMint, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NomiStablePool.contract.WatchLogs(opts, "Mint", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NomiStablePoolMint)
				if err := _NomiStablePool.contract.UnpackLog(event, "Mint", log); err != nil {
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

// ParseMint is a log parse operation binding the contract event 0x4c209b5fc8ad50758f13e2e1088ba56a560dff690a1c6fef26394f4c03821c4f.
//
// Solidity: event Mint(address indexed sender, uint256 amount0, uint256 amount1)
func (_NomiStablePool *NomiStablePoolFilterer) ParseMint(log types.Log) (*NomiStablePoolMint, error) {
	event := new(NomiStablePoolMint)
	if err := _NomiStablePool.contract.UnpackLog(event, "Mint", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NomiStablePoolRampAIterator is returned from FilterRampA and is used to iterate over the raw logs and unpacked data for RampA events raised by the NomiStablePool contract.
type NomiStablePoolRampAIterator struct {
	Event *NomiStablePoolRampA // Event containing the contract specifics and raw log

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
func (it *NomiStablePoolRampAIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NomiStablePoolRampA)
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
		it.Event = new(NomiStablePoolRampA)
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
func (it *NomiStablePoolRampAIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NomiStablePoolRampAIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NomiStablePoolRampA represents a RampA event raised by the NomiStablePool contract.
type NomiStablePoolRampA struct {
	OldA        *big.Int
	NewA        *big.Int
	InitialTime *big.Int
	FutureTime  *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterRampA is a free log retrieval operation binding the contract event 0xa2b71ec6df949300b59aab36b55e189697b750119dd349fcfa8c0f779e83c254.
//
// Solidity: event RampA(uint256 oldA, uint256 newA, uint256 initialTime, uint256 futureTime)
func (_NomiStablePool *NomiStablePoolFilterer) FilterRampA(opts *bind.FilterOpts) (*NomiStablePoolRampAIterator, error) {

	logs, sub, err := _NomiStablePool.contract.FilterLogs(opts, "RampA")
	if err != nil {
		return nil, err
	}
	return &NomiStablePoolRampAIterator{contract: _NomiStablePool.contract, event: "RampA", logs: logs, sub: sub}, nil
}

// WatchRampA is a free log subscription operation binding the contract event 0xa2b71ec6df949300b59aab36b55e189697b750119dd349fcfa8c0f779e83c254.
//
// Solidity: event RampA(uint256 oldA, uint256 newA, uint256 initialTime, uint256 futureTime)
func (_NomiStablePool *NomiStablePoolFilterer) WatchRampA(opts *bind.WatchOpts, sink chan<- *NomiStablePoolRampA) (event.Subscription, error) {

	logs, sub, err := _NomiStablePool.contract.WatchLogs(opts, "RampA")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NomiStablePoolRampA)
				if err := _NomiStablePool.contract.UnpackLog(event, "RampA", log); err != nil {
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

// ParseRampA is a log parse operation binding the contract event 0xa2b71ec6df949300b59aab36b55e189697b750119dd349fcfa8c0f779e83c254.
//
// Solidity: event RampA(uint256 oldA, uint256 newA, uint256 initialTime, uint256 futureTime)
func (_NomiStablePool *NomiStablePoolFilterer) ParseRampA(log types.Log) (*NomiStablePoolRampA, error) {
	event := new(NomiStablePoolRampA)
	if err := _NomiStablePool.contract.UnpackLog(event, "RampA", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NomiStablePoolStopRampAIterator is returned from FilterStopRampA and is used to iterate over the raw logs and unpacked data for StopRampA events raised by the NomiStablePool contract.
type NomiStablePoolStopRampAIterator struct {
	Event *NomiStablePoolStopRampA // Event containing the contract specifics and raw log

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
func (it *NomiStablePoolStopRampAIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NomiStablePoolStopRampA)
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
		it.Event = new(NomiStablePoolStopRampA)
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
func (it *NomiStablePoolStopRampAIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NomiStablePoolStopRampAIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NomiStablePoolStopRampA represents a StopRampA event raised by the NomiStablePool contract.
type NomiStablePoolStopRampA struct {
	A   *big.Int
	T   *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterStopRampA is a free log retrieval operation binding the contract event 0x46e22fb3709ad289f62ce63d469248536dbc78d82b84a3d7e74ad606dc201938.
//
// Solidity: event StopRampA(uint256 A, uint256 t)
func (_NomiStablePool *NomiStablePoolFilterer) FilterStopRampA(opts *bind.FilterOpts) (*NomiStablePoolStopRampAIterator, error) {

	logs, sub, err := _NomiStablePool.contract.FilterLogs(opts, "StopRampA")
	if err != nil {
		return nil, err
	}
	return &NomiStablePoolStopRampAIterator{contract: _NomiStablePool.contract, event: "StopRampA", logs: logs, sub: sub}, nil
}

// WatchStopRampA is a free log subscription operation binding the contract event 0x46e22fb3709ad289f62ce63d469248536dbc78d82b84a3d7e74ad606dc201938.
//
// Solidity: event StopRampA(uint256 A, uint256 t)
func (_NomiStablePool *NomiStablePoolFilterer) WatchStopRampA(opts *bind.WatchOpts, sink chan<- *NomiStablePoolStopRampA) (event.Subscription, error) {

	logs, sub, err := _NomiStablePool.contract.WatchLogs(opts, "StopRampA")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NomiStablePoolStopRampA)
				if err := _NomiStablePool.contract.UnpackLog(event, "StopRampA", log); err != nil {
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

// ParseStopRampA is a log parse operation binding the contract event 0x46e22fb3709ad289f62ce63d469248536dbc78d82b84a3d7e74ad606dc201938.
//
// Solidity: event StopRampA(uint256 A, uint256 t)
func (_NomiStablePool *NomiStablePoolFilterer) ParseStopRampA(log types.Log) (*NomiStablePoolStopRampA, error) {
	event := new(NomiStablePoolStopRampA)
	if err := _NomiStablePool.contract.UnpackLog(event, "StopRampA", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NomiStablePoolSwapIterator is returned from FilterSwap and is used to iterate over the raw logs and unpacked data for Swap events raised by the NomiStablePool contract.
type NomiStablePoolSwapIterator struct {
	Event *NomiStablePoolSwap // Event containing the contract specifics and raw log

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
func (it *NomiStablePoolSwapIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NomiStablePoolSwap)
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
		it.Event = new(NomiStablePoolSwap)
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
func (it *NomiStablePoolSwapIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NomiStablePoolSwapIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NomiStablePoolSwap represents a Swap event raised by the NomiStablePool contract.
type NomiStablePoolSwap struct {
	Sender     common.Address
	Amount0In  *big.Int
	Amount1In  *big.Int
	Amount0Out *big.Int
	Amount1Out *big.Int
	To         common.Address
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterSwap is a free log retrieval operation binding the contract event 0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822.
//
// Solidity: event Swap(address indexed sender, uint256 amount0In, uint256 amount1In, uint256 amount0Out, uint256 amount1Out, address indexed to)
func (_NomiStablePool *NomiStablePoolFilterer) FilterSwap(opts *bind.FilterOpts, sender []common.Address, to []common.Address) (*NomiStablePoolSwapIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _NomiStablePool.contract.FilterLogs(opts, "Swap", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return &NomiStablePoolSwapIterator{contract: _NomiStablePool.contract, event: "Swap", logs: logs, sub: sub}, nil
}

// WatchSwap is a free log subscription operation binding the contract event 0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822.
//
// Solidity: event Swap(address indexed sender, uint256 amount0In, uint256 amount1In, uint256 amount0Out, uint256 amount1Out, address indexed to)
func (_NomiStablePool *NomiStablePoolFilterer) WatchSwap(opts *bind.WatchOpts, sink chan<- *NomiStablePoolSwap, sender []common.Address, to []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _NomiStablePool.contract.WatchLogs(opts, "Swap", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NomiStablePoolSwap)
				if err := _NomiStablePool.contract.UnpackLog(event, "Swap", log); err != nil {
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

// ParseSwap is a log parse operation binding the contract event 0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822.
//
// Solidity: event Swap(address indexed sender, uint256 amount0In, uint256 amount1In, uint256 amount0Out, uint256 amount1Out, address indexed to)
func (_NomiStablePool *NomiStablePoolFilterer) ParseSwap(log types.Log) (*NomiStablePoolSwap, error) {
	event := new(NomiStablePoolSwap)
	if err := _NomiStablePool.contract.UnpackLog(event, "Swap", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NomiStablePoolSyncIterator is returned from FilterSync and is used to iterate over the raw logs and unpacked data for Sync events raised by the NomiStablePool contract.
type NomiStablePoolSyncIterator struct {
	Event *NomiStablePoolSync // Event containing the contract specifics and raw log

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
func (it *NomiStablePoolSyncIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NomiStablePoolSync)
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
		it.Event = new(NomiStablePoolSync)
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
func (it *NomiStablePoolSyncIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NomiStablePoolSyncIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NomiStablePoolSync represents a Sync event raised by the NomiStablePool contract.
type NomiStablePoolSync struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterSync is a free log retrieval operation binding the contract event 0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1.
//
// Solidity: event Sync(uint112 reserve0, uint112 reserve1)
func (_NomiStablePool *NomiStablePoolFilterer) FilterSync(opts *bind.FilterOpts) (*NomiStablePoolSyncIterator, error) {

	logs, sub, err := _NomiStablePool.contract.FilterLogs(opts, "Sync")
	if err != nil {
		return nil, err
	}
	return &NomiStablePoolSyncIterator{contract: _NomiStablePool.contract, event: "Sync", logs: logs, sub: sub}, nil
}

// WatchSync is a free log subscription operation binding the contract event 0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1.
//
// Solidity: event Sync(uint112 reserve0, uint112 reserve1)
func (_NomiStablePool *NomiStablePoolFilterer) WatchSync(opts *bind.WatchOpts, sink chan<- *NomiStablePoolSync) (event.Subscription, error) {

	logs, sub, err := _NomiStablePool.contract.WatchLogs(opts, "Sync")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NomiStablePoolSync)
				if err := _NomiStablePool.contract.UnpackLog(event, "Sync", log); err != nil {
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

// ParseSync is a log parse operation binding the contract event 0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1.
//
// Solidity: event Sync(uint112 reserve0, uint112 reserve1)
func (_NomiStablePool *NomiStablePoolFilterer) ParseSync(log types.Log) (*NomiStablePoolSync, error) {
	event := new(NomiStablePoolSync)
	if err := _NomiStablePool.contract.UnpackLog(event, "Sync", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NomiStablePoolTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the NomiStablePool contract.
type NomiStablePoolTransferIterator struct {
	Event *NomiStablePoolTransfer // Event containing the contract specifics and raw log

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
func (it *NomiStablePoolTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NomiStablePoolTransfer)
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
		it.Event = new(NomiStablePoolTransfer)
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
func (it *NomiStablePoolTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NomiStablePoolTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NomiStablePoolTransfer represents a Transfer event raised by the NomiStablePool contract.
type NomiStablePoolTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_NomiStablePool *NomiStablePoolFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*NomiStablePoolTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _NomiStablePool.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &NomiStablePoolTransferIterator{contract: _NomiStablePool.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_NomiStablePool *NomiStablePoolFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *NomiStablePoolTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _NomiStablePool.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NomiStablePoolTransfer)
				if err := _NomiStablePool.contract.UnpackLog(event, "Transfer", log); err != nil {
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

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_NomiStablePool *NomiStablePoolFilterer) ParseTransfer(log types.Log) (*NomiStablePoolTransfer, error) {
	event := new(NomiStablePoolTransfer)
	if err := _NomiStablePool.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
