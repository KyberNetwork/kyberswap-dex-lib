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

// StateViewMetaData contains all meta data concerning the StateView contract.
var StateViewMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIPoolManager\",\"name\":\"_poolManager\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"getFeeGrowthGlobals\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"feeGrowthGlobal0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthGlobal1\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"}],\"name\":\"getFeeGrowthInside\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"feeGrowthInside0X128\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthInside1X128\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"getLiquidity\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"liquidity\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"positionId\",\"type\":\"bytes32\"}],\"name\":\"getPositionInfo\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"liquidity\",\"type\":\"uint128\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthInside0LastX128\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthInside1LastX128\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"}],\"name\":\"getPositionInfo\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"liquidity\",\"type\":\"uint128\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthInside0LastX128\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthInside1LastX128\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"positionId\",\"type\":\"bytes32\"}],\"name\":\"getPositionLiquidity\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"liquidity\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"getSlot0\",\"outputs\":[{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"},{\"internalType\":\"uint24\",\"name\":\"protocolFee\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"lpFee\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"internalType\":\"int16\",\"name\":\"tick\",\"type\":\"int16\"}],\"name\":\"getTickBitmap\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"tickBitmap\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"getTickFeeGrowthOutside\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"feeGrowthOutside0X128\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthOutside1X128\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"getTickInfo\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"liquidityGross\",\"type\":\"uint128\"},{\"internalType\":\"int128\",\"name\":\"liquidityNet\",\"type\":\"int128\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthOutside0X128\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthOutside1X128\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"getTickLiquidity\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"liquidityGross\",\"type\":\"uint128\"},{\"internalType\":\"int128\",\"name\":\"liquidityNet\",\"type\":\"int128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"poolManager\",\"outputs\":[{\"internalType\":\"contractIPoolManager\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// StateViewABI is the input ABI used to generate the binding from.
// Deprecated: Use StateViewMetaData.ABI instead.
var StateViewABI = StateViewMetaData.ABI

// StateView is an auto generated Go binding around an Ethereum contract.
type StateView struct {
	StateViewCaller     // Read-only binding to the contract
	StateViewTransactor // Write-only binding to the contract
	StateViewFilterer   // Log filterer for contract events
}

// StateViewCaller is an auto generated read-only Go binding around an Ethereum contract.
type StateViewCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StateViewTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StateViewTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StateViewFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StateViewFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StateViewSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StateViewSession struct {
	Contract     *StateView        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StateViewCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StateViewCallerSession struct {
	Contract *StateViewCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// StateViewTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StateViewTransactorSession struct {
	Contract     *StateViewTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// StateViewRaw is an auto generated low-level Go binding around an Ethereum contract.
type StateViewRaw struct {
	Contract *StateView // Generic contract binding to access the raw methods on
}

// StateViewCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StateViewCallerRaw struct {
	Contract *StateViewCaller // Generic read-only contract binding to access the raw methods on
}

// StateViewTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StateViewTransactorRaw struct {
	Contract *StateViewTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStateView creates a new instance of StateView, bound to a specific deployed contract.
func NewStateView(address common.Address, backend bind.ContractBackend) (*StateView, error) {
	contract, err := bindStateView(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &StateView{StateViewCaller: StateViewCaller{contract: contract}, StateViewTransactor: StateViewTransactor{contract: contract}, StateViewFilterer: StateViewFilterer{contract: contract}}, nil
}

// NewStateViewCaller creates a new read-only instance of StateView, bound to a specific deployed contract.
func NewStateViewCaller(address common.Address, caller bind.ContractCaller) (*StateViewCaller, error) {
	contract, err := bindStateView(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StateViewCaller{contract: contract}, nil
}

// NewStateViewTransactor creates a new write-only instance of StateView, bound to a specific deployed contract.
func NewStateViewTransactor(address common.Address, transactor bind.ContractTransactor) (*StateViewTransactor, error) {
	contract, err := bindStateView(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StateViewTransactor{contract: contract}, nil
}

// NewStateViewFilterer creates a new log filterer instance of StateView, bound to a specific deployed contract.
func NewStateViewFilterer(address common.Address, filterer bind.ContractFilterer) (*StateViewFilterer, error) {
	contract, err := bindStateView(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StateViewFilterer{contract: contract}, nil
}

// bindStateView binds a generic wrapper to an already deployed contract.
func bindStateView(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := StateViewMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StateView *StateViewRaw) Call(opts *bind.CallOpts, result *[]any, method string, params ...any) error {
	return _StateView.Contract.StateViewCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StateView *StateViewRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StateView.Contract.StateViewTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StateView *StateViewRaw) Transact(opts *bind.TransactOpts, method string, params ...any) (*types.Transaction, error) {
	return _StateView.Contract.StateViewTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StateView *StateViewCallerRaw) Call(opts *bind.CallOpts, result *[]any, method string, params ...any) error {
	return _StateView.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StateView *StateViewTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StateView.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StateView *StateViewTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...any) (*types.Transaction, error) {
	return _StateView.Contract.contract.Transact(opts, method, params...)
}

// GetFeeGrowthGlobals is a free data retrieval call binding the contract method 0x9ec538c8.
//
// Solidity: function getFeeGrowthGlobals(bytes32 poolId) view returns(uint256 feeGrowthGlobal0, uint256 feeGrowthGlobal1)
func (_StateView *StateViewCaller) GetFeeGrowthGlobals(opts *bind.CallOpts, poolId [32]byte) (struct {
	FeeGrowthGlobal0 *big.Int
	FeeGrowthGlobal1 *big.Int
}, error) {
	var out []any
	err := _StateView.contract.Call(opts, &out, "getFeeGrowthGlobals", poolId)

	outstruct := new(struct {
		FeeGrowthGlobal0 *big.Int
		FeeGrowthGlobal1 *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.FeeGrowthGlobal0 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.FeeGrowthGlobal1 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetFeeGrowthGlobals is a free data retrieval call binding the contract method 0x9ec538c8.
//
// Solidity: function getFeeGrowthGlobals(bytes32 poolId) view returns(uint256 feeGrowthGlobal0, uint256 feeGrowthGlobal1)
func (_StateView *StateViewSession) GetFeeGrowthGlobals(poolId [32]byte) (struct {
	FeeGrowthGlobal0 *big.Int
	FeeGrowthGlobal1 *big.Int
}, error) {
	return _StateView.Contract.GetFeeGrowthGlobals(&_StateView.CallOpts, poolId)
}

// GetFeeGrowthGlobals is a free data retrieval call binding the contract method 0x9ec538c8.
//
// Solidity: function getFeeGrowthGlobals(bytes32 poolId) view returns(uint256 feeGrowthGlobal0, uint256 feeGrowthGlobal1)
func (_StateView *StateViewCallerSession) GetFeeGrowthGlobals(poolId [32]byte) (struct {
	FeeGrowthGlobal0 *big.Int
	FeeGrowthGlobal1 *big.Int
}, error) {
	return _StateView.Contract.GetFeeGrowthGlobals(&_StateView.CallOpts, poolId)
}

// GetFeeGrowthInside is a free data retrieval call binding the contract method 0x53e9c1fb.
//
// Solidity: function getFeeGrowthInside(bytes32 poolId, int24 tickLower, int24 tickUpper) view returns(uint256 feeGrowthInside0X128, uint256 feeGrowthInside1X128)
func (_StateView *StateViewCaller) GetFeeGrowthInside(opts *bind.CallOpts, poolId [32]byte, tickLower *big.Int, tickUpper *big.Int) (struct {
	FeeGrowthInside0X128 *big.Int
	FeeGrowthInside1X128 *big.Int
}, error) {
	var out []any
	err := _StateView.contract.Call(opts, &out, "getFeeGrowthInside", poolId, tickLower, tickUpper)

	outstruct := new(struct {
		FeeGrowthInside0X128 *big.Int
		FeeGrowthInside1X128 *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.FeeGrowthInside0X128 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.FeeGrowthInside1X128 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetFeeGrowthInside is a free data retrieval call binding the contract method 0x53e9c1fb.
//
// Solidity: function getFeeGrowthInside(bytes32 poolId, int24 tickLower, int24 tickUpper) view returns(uint256 feeGrowthInside0X128, uint256 feeGrowthInside1X128)
func (_StateView *StateViewSession) GetFeeGrowthInside(poolId [32]byte, tickLower *big.Int, tickUpper *big.Int) (struct {
	FeeGrowthInside0X128 *big.Int
	FeeGrowthInside1X128 *big.Int
}, error) {
	return _StateView.Contract.GetFeeGrowthInside(&_StateView.CallOpts, poolId, tickLower, tickUpper)
}

// GetFeeGrowthInside is a free data retrieval call binding the contract method 0x53e9c1fb.
//
// Solidity: function getFeeGrowthInside(bytes32 poolId, int24 tickLower, int24 tickUpper) view returns(uint256 feeGrowthInside0X128, uint256 feeGrowthInside1X128)
func (_StateView *StateViewCallerSession) GetFeeGrowthInside(poolId [32]byte, tickLower *big.Int, tickUpper *big.Int) (struct {
	FeeGrowthInside0X128 *big.Int
	FeeGrowthInside1X128 *big.Int
}, error) {
	return _StateView.Contract.GetFeeGrowthInside(&_StateView.CallOpts, poolId, tickLower, tickUpper)
}

// GetLiquidity is a free data retrieval call binding the contract method 0xfa6793d5.
//
// Solidity: function getLiquidity(bytes32 poolId) view returns(uint128 liquidity)
func (_StateView *StateViewCaller) GetLiquidity(opts *bind.CallOpts, poolId [32]byte) (*big.Int, error) {
	var out []any
	err := _StateView.contract.Call(opts, &out, "getLiquidity", poolId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetLiquidity is a free data retrieval call binding the contract method 0xfa6793d5.
//
// Solidity: function getLiquidity(bytes32 poolId) view returns(uint128 liquidity)
func (_StateView *StateViewSession) GetLiquidity(poolId [32]byte) (*big.Int, error) {
	return _StateView.Contract.GetLiquidity(&_StateView.CallOpts, poolId)
}

// GetLiquidity is a free data retrieval call binding the contract method 0xfa6793d5.
//
// Solidity: function getLiquidity(bytes32 poolId) view returns(uint128 liquidity)
func (_StateView *StateViewCallerSession) GetLiquidity(poolId [32]byte) (*big.Int, error) {
	return _StateView.Contract.GetLiquidity(&_StateView.CallOpts, poolId)
}

// GetPositionInfo is a free data retrieval call binding the contract method 0x97fd7b42.
//
// Solidity: function getPositionInfo(bytes32 poolId, bytes32 positionId) view returns(uint128 liquidity, uint256 feeGrowthInside0LastX128, uint256 feeGrowthInside1LastX128)
func (_StateView *StateViewCaller) GetPositionInfo(opts *bind.CallOpts, poolId [32]byte, positionId [32]byte) (struct {
	Liquidity                *big.Int
	FeeGrowthInside0LastX128 *big.Int
	FeeGrowthInside1LastX128 *big.Int
}, error) {
	var out []any
	err := _StateView.contract.Call(opts, &out, "getPositionInfo", poolId, positionId)

	outstruct := new(struct {
		Liquidity                *big.Int
		FeeGrowthInside0LastX128 *big.Int
		FeeGrowthInside1LastX128 *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Liquidity = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.FeeGrowthInside0LastX128 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.FeeGrowthInside1LastX128 = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetPositionInfo is a free data retrieval call binding the contract method 0x97fd7b42.
//
// Solidity: function getPositionInfo(bytes32 poolId, bytes32 positionId) view returns(uint128 liquidity, uint256 feeGrowthInside0LastX128, uint256 feeGrowthInside1LastX128)
func (_StateView *StateViewSession) GetPositionInfo(poolId [32]byte, positionId [32]byte) (struct {
	Liquidity                *big.Int
	FeeGrowthInside0LastX128 *big.Int
	FeeGrowthInside1LastX128 *big.Int
}, error) {
	return _StateView.Contract.GetPositionInfo(&_StateView.CallOpts, poolId, positionId)
}

// GetPositionInfo is a free data retrieval call binding the contract method 0x97fd7b42.
//
// Solidity: function getPositionInfo(bytes32 poolId, bytes32 positionId) view returns(uint128 liquidity, uint256 feeGrowthInside0LastX128, uint256 feeGrowthInside1LastX128)
func (_StateView *StateViewCallerSession) GetPositionInfo(poolId [32]byte, positionId [32]byte) (struct {
	Liquidity                *big.Int
	FeeGrowthInside0LastX128 *big.Int
	FeeGrowthInside1LastX128 *big.Int
}, error) {
	return _StateView.Contract.GetPositionInfo(&_StateView.CallOpts, poolId, positionId)
}

// GetPositionInfo0 is a free data retrieval call binding the contract method 0xdacf1d2f.
//
// Solidity: function getPositionInfo(bytes32 poolId, address owner, int24 tickLower, int24 tickUpper, bytes32 salt) view returns(uint128 liquidity, uint256 feeGrowthInside0LastX128, uint256 feeGrowthInside1LastX128)
func (_StateView *StateViewCaller) GetPositionInfo0(opts *bind.CallOpts, poolId [32]byte, owner common.Address, tickLower *big.Int, tickUpper *big.Int, salt [32]byte) (struct {
	Liquidity                *big.Int
	FeeGrowthInside0LastX128 *big.Int
	FeeGrowthInside1LastX128 *big.Int
}, error) {
	var out []any
	err := _StateView.contract.Call(opts, &out, "getPositionInfo0", poolId, owner, tickLower, tickUpper, salt)

	outstruct := new(struct {
		Liquidity                *big.Int
		FeeGrowthInside0LastX128 *big.Int
		FeeGrowthInside1LastX128 *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Liquidity = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.FeeGrowthInside0LastX128 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.FeeGrowthInside1LastX128 = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetPositionInfo0 is a free data retrieval call binding the contract method 0xdacf1d2f.
//
// Solidity: function getPositionInfo(bytes32 poolId, address owner, int24 tickLower, int24 tickUpper, bytes32 salt) view returns(uint128 liquidity, uint256 feeGrowthInside0LastX128, uint256 feeGrowthInside1LastX128)
func (_StateView *StateViewSession) GetPositionInfo0(poolId [32]byte, owner common.Address, tickLower *big.Int, tickUpper *big.Int, salt [32]byte) (struct {
	Liquidity                *big.Int
	FeeGrowthInside0LastX128 *big.Int
	FeeGrowthInside1LastX128 *big.Int
}, error) {
	return _StateView.Contract.GetPositionInfo0(&_StateView.CallOpts, poolId, owner, tickLower, tickUpper, salt)
}

// GetPositionInfo0 is a free data retrieval call binding the contract method 0xdacf1d2f.
//
// Solidity: function getPositionInfo(bytes32 poolId, address owner, int24 tickLower, int24 tickUpper, bytes32 salt) view returns(uint128 liquidity, uint256 feeGrowthInside0LastX128, uint256 feeGrowthInside1LastX128)
func (_StateView *StateViewCallerSession) GetPositionInfo0(poolId [32]byte, owner common.Address, tickLower *big.Int, tickUpper *big.Int, salt [32]byte) (struct {
	Liquidity                *big.Int
	FeeGrowthInside0LastX128 *big.Int
	FeeGrowthInside1LastX128 *big.Int
}, error) {
	return _StateView.Contract.GetPositionInfo0(&_StateView.CallOpts, poolId, owner, tickLower, tickUpper, salt)
}

// GetPositionLiquidity is a free data retrieval call binding the contract method 0xf0928f29.
//
// Solidity: function getPositionLiquidity(bytes32 poolId, bytes32 positionId) view returns(uint128 liquidity)
func (_StateView *StateViewCaller) GetPositionLiquidity(opts *bind.CallOpts, poolId [32]byte, positionId [32]byte) (*big.Int, error) {
	var out []any
	err := _StateView.contract.Call(opts, &out, "getPositionLiquidity", poolId, positionId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetPositionLiquidity is a free data retrieval call binding the contract method 0xf0928f29.
//
// Solidity: function getPositionLiquidity(bytes32 poolId, bytes32 positionId) view returns(uint128 liquidity)
func (_StateView *StateViewSession) GetPositionLiquidity(poolId [32]byte, positionId [32]byte) (*big.Int, error) {
	return _StateView.Contract.GetPositionLiquidity(&_StateView.CallOpts, poolId, positionId)
}

// GetPositionLiquidity is a free data retrieval call binding the contract method 0xf0928f29.
//
// Solidity: function getPositionLiquidity(bytes32 poolId, bytes32 positionId) view returns(uint128 liquidity)
func (_StateView *StateViewCallerSession) GetPositionLiquidity(poolId [32]byte, positionId [32]byte) (*big.Int, error) {
	return _StateView.Contract.GetPositionLiquidity(&_StateView.CallOpts, poolId, positionId)
}

// GetSlot0 is a free data retrieval call binding the contract method 0xc815641c.
//
// Solidity: function getSlot0(bytes32 poolId) view returns(uint160 sqrtPriceX96, int24 tick, uint24 protocolFee, uint24 lpFee)
func (_StateView *StateViewCaller) GetSlot0(opts *bind.CallOpts, poolId [32]byte) (struct {
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	ProtocolFee  *big.Int
	LpFee        *big.Int
}, error) {
	var out []any
	err := _StateView.contract.Call(opts, &out, "getSlot0", poolId)

	outstruct := new(struct {
		SqrtPriceX96 *big.Int
		Tick         *big.Int
		ProtocolFee  *big.Int
		LpFee        *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.SqrtPriceX96 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Tick = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.ProtocolFee = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.LpFee = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetSlot0 is a free data retrieval call binding the contract method 0xc815641c.
//
// Solidity: function getSlot0(bytes32 poolId) view returns(uint160 sqrtPriceX96, int24 tick, uint24 protocolFee, uint24 lpFee)
func (_StateView *StateViewSession) GetSlot0(poolId [32]byte) (struct {
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	ProtocolFee  *big.Int
	LpFee        *big.Int
}, error) {
	return _StateView.Contract.GetSlot0(&_StateView.CallOpts, poolId)
}

// GetSlot0 is a free data retrieval call binding the contract method 0xc815641c.
//
// Solidity: function getSlot0(bytes32 poolId) view returns(uint160 sqrtPriceX96, int24 tick, uint24 protocolFee, uint24 lpFee)
func (_StateView *StateViewCallerSession) GetSlot0(poolId [32]byte) (struct {
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	ProtocolFee  *big.Int
	LpFee        *big.Int
}, error) {
	return _StateView.Contract.GetSlot0(&_StateView.CallOpts, poolId)
}

// GetTickBitmap is a free data retrieval call binding the contract method 0x1c7ccb4c.
//
// Solidity: function getTickBitmap(bytes32 poolId, int16 tick) view returns(uint256 tickBitmap)
func (_StateView *StateViewCaller) GetTickBitmap(opts *bind.CallOpts, poolId [32]byte, tick int16) (*big.Int, error) {
	var out []any
	err := _StateView.contract.Call(opts, &out, "getTickBitmap", poolId, tick)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTickBitmap is a free data retrieval call binding the contract method 0x1c7ccb4c.
//
// Solidity: function getTickBitmap(bytes32 poolId, int16 tick) view returns(uint256 tickBitmap)
func (_StateView *StateViewSession) GetTickBitmap(poolId [32]byte, tick int16) (*big.Int, error) {
	return _StateView.Contract.GetTickBitmap(&_StateView.CallOpts, poolId, tick)
}

// GetTickBitmap is a free data retrieval call binding the contract method 0x1c7ccb4c.
//
// Solidity: function getTickBitmap(bytes32 poolId, int16 tick) view returns(uint256 tickBitmap)
func (_StateView *StateViewCallerSession) GetTickBitmap(poolId [32]byte, tick int16) (*big.Int, error) {
	return _StateView.Contract.GetTickBitmap(&_StateView.CallOpts, poolId, tick)
}

// GetTickFeeGrowthOutside is a free data retrieval call binding the contract method 0x8a2bb9e6.
//
// Solidity: function getTickFeeGrowthOutside(bytes32 poolId, int24 tick) view returns(uint256 feeGrowthOutside0X128, uint256 feeGrowthOutside1X128)
func (_StateView *StateViewCaller) GetTickFeeGrowthOutside(opts *bind.CallOpts, poolId [32]byte, tick *big.Int) (struct {
	FeeGrowthOutside0X128 *big.Int
	FeeGrowthOutside1X128 *big.Int
}, error) {
	var out []any
	err := _StateView.contract.Call(opts, &out, "getTickFeeGrowthOutside", poolId, tick)

	outstruct := new(struct {
		FeeGrowthOutside0X128 *big.Int
		FeeGrowthOutside1X128 *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.FeeGrowthOutside0X128 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.FeeGrowthOutside1X128 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetTickFeeGrowthOutside is a free data retrieval call binding the contract method 0x8a2bb9e6.
//
// Solidity: function getTickFeeGrowthOutside(bytes32 poolId, int24 tick) view returns(uint256 feeGrowthOutside0X128, uint256 feeGrowthOutside1X128)
func (_StateView *StateViewSession) GetTickFeeGrowthOutside(poolId [32]byte, tick *big.Int) (struct {
	FeeGrowthOutside0X128 *big.Int
	FeeGrowthOutside1X128 *big.Int
}, error) {
	return _StateView.Contract.GetTickFeeGrowthOutside(&_StateView.CallOpts, poolId, tick)
}

// GetTickFeeGrowthOutside is a free data retrieval call binding the contract method 0x8a2bb9e6.
//
// Solidity: function getTickFeeGrowthOutside(bytes32 poolId, int24 tick) view returns(uint256 feeGrowthOutside0X128, uint256 feeGrowthOutside1X128)
func (_StateView *StateViewCallerSession) GetTickFeeGrowthOutside(poolId [32]byte, tick *big.Int) (struct {
	FeeGrowthOutside0X128 *big.Int
	FeeGrowthOutside1X128 *big.Int
}, error) {
	return _StateView.Contract.GetTickFeeGrowthOutside(&_StateView.CallOpts, poolId, tick)
}

// GetTickInfo is a free data retrieval call binding the contract method 0x7c40f1fe.
//
// Solidity: function getTickInfo(bytes32 poolId, int24 tick) view returns(uint128 liquidityGross, int128 liquidityNet, uint256 feeGrowthOutside0X128, uint256 feeGrowthOutside1X128)
func (_StateView *StateViewCaller) GetTickInfo(opts *bind.CallOpts, poolId [32]byte, tick *big.Int) (struct {
	LiquidityGross        *big.Int
	LiquidityNet          *big.Int
	FeeGrowthOutside0X128 *big.Int
	FeeGrowthOutside1X128 *big.Int
}, error) {
	var out []any
	err := _StateView.contract.Call(opts, &out, "getTickInfo", poolId, tick)

	outstruct := new(struct {
		LiquidityGross        *big.Int
		LiquidityNet          *big.Int
		FeeGrowthOutside0X128 *big.Int
		FeeGrowthOutside1X128 *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.LiquidityGross = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.LiquidityNet = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.FeeGrowthOutside0X128 = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.FeeGrowthOutside1X128 = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetTickInfo is a free data retrieval call binding the contract method 0x7c40f1fe.
//
// Solidity: function getTickInfo(bytes32 poolId, int24 tick) view returns(uint128 liquidityGross, int128 liquidityNet, uint256 feeGrowthOutside0X128, uint256 feeGrowthOutside1X128)
func (_StateView *StateViewSession) GetTickInfo(poolId [32]byte, tick *big.Int) (struct {
	LiquidityGross        *big.Int
	LiquidityNet          *big.Int
	FeeGrowthOutside0X128 *big.Int
	FeeGrowthOutside1X128 *big.Int
}, error) {
	return _StateView.Contract.GetTickInfo(&_StateView.CallOpts, poolId, tick)
}

// GetTickInfo is a free data retrieval call binding the contract method 0x7c40f1fe.
//
// Solidity: function getTickInfo(bytes32 poolId, int24 tick) view returns(uint128 liquidityGross, int128 liquidityNet, uint256 feeGrowthOutside0X128, uint256 feeGrowthOutside1X128)
func (_StateView *StateViewCallerSession) GetTickInfo(poolId [32]byte, tick *big.Int) (struct {
	LiquidityGross        *big.Int
	LiquidityNet          *big.Int
	FeeGrowthOutside0X128 *big.Int
	FeeGrowthOutside1X128 *big.Int
}, error) {
	return _StateView.Contract.GetTickInfo(&_StateView.CallOpts, poolId, tick)
}

// GetTickLiquidity is a free data retrieval call binding the contract method 0xcaedab54.
//
// Solidity: function getTickLiquidity(bytes32 poolId, int24 tick) view returns(uint128 liquidityGross, int128 liquidityNet)
func (_StateView *StateViewCaller) GetTickLiquidity(opts *bind.CallOpts, poolId [32]byte, tick *big.Int) (struct {
	LiquidityGross *big.Int
	LiquidityNet   *big.Int
}, error) {
	var out []any
	err := _StateView.contract.Call(opts, &out, "getTickLiquidity", poolId, tick)

	outstruct := new(struct {
		LiquidityGross *big.Int
		LiquidityNet   *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.LiquidityGross = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.LiquidityNet = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetTickLiquidity is a free data retrieval call binding the contract method 0xcaedab54.
//
// Solidity: function getTickLiquidity(bytes32 poolId, int24 tick) view returns(uint128 liquidityGross, int128 liquidityNet)
func (_StateView *StateViewSession) GetTickLiquidity(poolId [32]byte, tick *big.Int) (struct {
	LiquidityGross *big.Int
	LiquidityNet   *big.Int
}, error) {
	return _StateView.Contract.GetTickLiquidity(&_StateView.CallOpts, poolId, tick)
}

// GetTickLiquidity is a free data retrieval call binding the contract method 0xcaedab54.
//
// Solidity: function getTickLiquidity(bytes32 poolId, int24 tick) view returns(uint128 liquidityGross, int128 liquidityNet)
func (_StateView *StateViewCallerSession) GetTickLiquidity(poolId [32]byte, tick *big.Int) (struct {
	LiquidityGross *big.Int
	LiquidityNet   *big.Int
}, error) {
	return _StateView.Contract.GetTickLiquidity(&_StateView.CallOpts, poolId, tick)
}

// PoolManager is a free data retrieval call binding the contract method 0xdc4c90d3.
//
// Solidity: function poolManager() view returns(address)
func (_StateView *StateViewCaller) PoolManager(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _StateView.contract.Call(opts, &out, "poolManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PoolManager is a free data retrieval call binding the contract method 0xdc4c90d3.
//
// Solidity: function poolManager() view returns(address)
func (_StateView *StateViewSession) PoolManager() (common.Address, error) {
	return _StateView.Contract.PoolManager(&_StateView.CallOpts)
}

// PoolManager is a free data retrieval call binding the contract method 0xdc4c90d3.
//
// Solidity: function poolManager() view returns(address)
func (_StateView *StateViewCallerSession) PoolManager() (common.Address, error) {
	return _StateView.Contract.PoolManager(&_StateView.CallOpts)
}
