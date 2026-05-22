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

// V2PoolMetaData contains all meta data concerning the V2Pool contract.
var V2PoolMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"},{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"name\":\"Burn\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amount0\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1\",\"type\":\"uint128\"}],\"name\":\"Collect\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint128\",\"name\":\"amount0\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1\",\"type\":\"uint128\"}],\"name\":\"CollectProtocol\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"paid0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"paid1\",\"type\":\"uint256\"}],\"name\":\"Flash\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"observationCardinalityNextOld\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"observationCardinalityNextNew\",\"type\":\"uint16\"}],\"name\":\"IncreaseObservationCardinalityNext\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"Initialize\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"},{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"name\":\"Mint\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"feeProtocol0Old\",\"type\":\"uint8\"},{\"internalType\":\"uint8\",\"name\":\"feeProtocol1Old\",\"type\":\"uint8\"},{\"internalType\":\"uint8\",\"name\":\"feeProtocol0New\",\"type\":\"uint8\"},{\"internalType\":\"uint8\",\"name\":\"feeProtocol1New\",\"type\":\"uint8\"}],\"name\":\"SetFeeProtocol\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"int256\",\"name\":\"amount0\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"amount1\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"internalType\":\"uint128\",\"name\":\"liquidity\",\"type\":\"uint128\"},{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"Swap\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"factory\",\"outputs\":[{\"internalType\":\"address\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"fee\",\"outputs\":[{\"internalType\":\"uint24\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"feeGrowthGlobal0X128\",\"outputs\":[{\"internalType\":\"uint256\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"feeGrowthGlobal1X128\",\"outputs\":[{\"internalType\":\"uint256\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"lastPeriod\",\"outputs\":[{\"internalType\":\"uint256\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"liquidity\",\"outputs\":[{\"internalType\":\"uint128\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxLiquidityPerTick\",\"outputs\":[{\"internalType\":\"uint128\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"observations\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"blockTimestamp\",\"type\":\"uint32\"},{\"internalType\":\"int56\",\"name\":\"tickCumulative\",\"type\":\"int56\"},{\"internalType\":\"uint160\",\"name\":\"secondsPerLiquidityCumulativeX128\",\"type\":\"uint160\"},{\"internalType\":\"bool\",\"name\":\"initialized\",\"type\":\"bool\"},{\"internalType\":\"uint160\",\"name\":\"secondsPerBoostedLiquidityPeriodX128\",\"type\":\"uint160\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32[]\",\"name\":\"secondsAgos\",\"type\":\"uint32[]\"}],\"name\":\"observe\",\"outputs\":[{\"internalType\":\"int56[]\",\"name\":\"tickCumulatives\",\"type\":\"int56[]\"},{\"internalType\":\"uint160[]\",\"name\":\"secondsPerLiquidityCumulativeX128s\",\"type\":\"uint160[]\"},{\"internalType\":\"uint160[]\",\"name\":\"secondsPerBoostedLiquidityPeriodX128s\",\"type\":\"uint160[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"period\",\"type\":\"uint32\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"}],\"name\":\"periodCumulativesInside\",\"outputs\":[{\"internalType\":\"uint160\",\"name\":\"secondsPerLiquidityInsideX128\",\"type\":\"uint160\"},{\"internalType\":\"uint160\",\"name\":\"secondsPerBoostedLiquidityInsideX128\",\"type\":\"uint160\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"period\",\"type\":\"uint256\"}],\"name\":\"periods\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"previousPeriod\",\"type\":\"uint32\"},{\"internalType\":\"int24\",\"name\":\"startTick\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"lastTick\",\"type\":\"int24\"},{\"internalType\":\"uint160\",\"name\":\"endSecondsPerLiquidityPeriodX128\",\"type\":\"uint160\"},{\"internalType\":\"uint160\",\"name\":\"endSecondsPerBoostedLiquidityPeriodX128\",\"type\":\"uint160\"},{\"internalType\":\"uint32\",\"name\":\"boostedInRange\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"period\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"}],\"name\":\"positionPeriodDebt\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"secondsDebtX96\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"boostedSecondsDebtX96\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"period\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"}],\"name\":\"positionPeriodSecondsInRange\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"periodSecondsInsideX96\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"periodBoostedSecondsInsideX96\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"key\",\"type\":\"bytes32\"}],\"name\":\"positions\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"liquidity\",\"type\":\"uint128\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthInside0LastX128\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthInside1LastX128\",\"type\":\"uint256\"},{\"internalType\":\"uint128\",\"name\":\"tokensOwed0\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"tokensOwed1\",\"type\":\"uint128\"},{\"internalType\":\"uint256\",\"name\":\"attachedVeRamId\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"protocolFees\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"token0\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"token1\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"slots\",\"type\":\"bytes32[]\"}],\"name\":\"readStorage\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"returnData\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"slot0\",\"outputs\":[{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"},{\"internalType\":\"uint16\",\"name\":\"observationIndex\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"observationCardinality\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"observationCardinalityNext\",\"type\":\"uint16\"},{\"internalType\":\"uint32\",\"name\":\"feeProtocol\",\"type\":\"uint32\"},{\"internalType\":\"bool\",\"name\":\"unlocked\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"}],\"name\":\"snapshotCumulativesInside\",\"outputs\":[{\"internalType\":\"int56\",\"name\":\"tickCumulativeInside\",\"type\":\"int56\"},{\"internalType\":\"uint160\",\"name\":\"secondsPerLiquidityInsideX128\",\"type\":\"uint160\"},{\"internalType\":\"uint160\",\"name\":\"secondsPerBoostedLiquidityInsideX128\",\"type\":\"uint160\"},{\"internalType\":\"uint32\",\"name\":\"secondsInside\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int16\",\"name\":\"tick\",\"type\":\"int16\"}],\"name\":\"tickBitmap\",\"outputs\":[{\"internalType\":\"uint256\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"tickSpacing\",\"outputs\":[{\"internalType\":\"int24\",\"type\":\"int24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"ticks\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"liquidityGross\",\"type\":\"uint128\"},{\"internalType\":\"int128\",\"name\":\"liquidityNet\",\"type\":\"int128\"},{\"internalType\":\"uint128\",\"name\":\"boostedLiquidityGross\",\"type\":\"uint128\"},{\"internalType\":\"int128\",\"name\":\"boostedLiquidityNet\",\"type\":\"int128\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthOutside0X128\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthOutside1X128\",\"type\":\"uint256\"},{\"internalType\":\"int56\",\"name\":\"tickCumulativeOutside\",\"type\":\"int56\"},{\"internalType\":\"uint160\",\"name\":\"secondsPerLiquidityOutsideX128\",\"type\":\"uint160\"},{\"internalType\":\"uint32\",\"name\":\"secondsOutside\",\"type\":\"uint32\"},{\"internalType\":\"bool\",\"name\":\"initialized\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"token0\",\"outputs\":[{\"internalType\":\"address\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"token1\",\"outputs\":[{\"internalType\":\"address\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentFee\",\"outputs\":[{\"internalType\":\"uint24\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// V2PoolABI is the input ABI used to generate the binding from.
// Deprecated: Use V2PoolMetaData.ABI instead.
var V2PoolABI = V2PoolMetaData.ABI

// V2Pool is an auto generated Go binding around an Ethereum contract.
type V2Pool struct {
	V2PoolCaller     // Read-only binding to the contract
	V2PoolTransactor // Write-only binding to the contract
	V2PoolFilterer   // Log filterer for contract events
}

// V2PoolCaller is an auto generated read-only Go binding around an Ethereum contract.
type V2PoolCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// V2PoolTransactor is an auto generated write-only Go binding around an Ethereum contract.
type V2PoolTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// V2PoolFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type V2PoolFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// V2PoolSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type V2PoolSession struct {
	Contract     *V2Pool           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// V2PoolCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type V2PoolCallerSession struct {
	Contract *V2PoolCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// V2PoolTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type V2PoolTransactorSession struct {
	Contract     *V2PoolTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// V2PoolRaw is an auto generated low-level Go binding around an Ethereum contract.
type V2PoolRaw struct {
	Contract *V2Pool // Generic contract binding to access the raw methods on
}

// V2PoolCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type V2PoolCallerRaw struct {
	Contract *V2PoolCaller // Generic read-only contract binding to access the raw methods on
}

// V2PoolTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type V2PoolTransactorRaw struct {
	Contract *V2PoolTransactor // Generic write-only contract binding to access the raw methods on
}

// NewV2Pool creates a new instance of V2Pool, bound to a specific deployed contract.
func NewV2Pool(address common.Address, backend bind.ContractBackend) (*V2Pool, error) {
	contract, err := bindV2Pool(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &V2Pool{V2PoolCaller: V2PoolCaller{contract: contract}, V2PoolTransactor: V2PoolTransactor{contract: contract}, V2PoolFilterer: V2PoolFilterer{contract: contract}}, nil
}

// NewV2PoolCaller creates a new read-only instance of V2Pool, bound to a specific deployed contract.
func NewV2PoolCaller(address common.Address, caller bind.ContractCaller) (*V2PoolCaller, error) {
	contract, err := bindV2Pool(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &V2PoolCaller{contract: contract}, nil
}

// NewV2PoolTransactor creates a new write-only instance of V2Pool, bound to a specific deployed contract.
func NewV2PoolTransactor(address common.Address, transactor bind.ContractTransactor) (*V2PoolTransactor, error) {
	contract, err := bindV2Pool(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &V2PoolTransactor{contract: contract}, nil
}

// NewV2PoolFilterer creates a new log filterer instance of V2Pool, bound to a specific deployed contract.
func NewV2PoolFilterer(address common.Address, filterer bind.ContractFilterer) (*V2PoolFilterer, error) {
	contract, err := bindV2Pool(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &V2PoolFilterer{contract: contract}, nil
}

// bindV2Pool binds a generic wrapper to an already deployed contract.
func bindV2Pool(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := V2PoolMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_V2Pool *V2PoolRaw) Call(opts *bind.CallOpts, result *[]any, method string, params ...any) error {
	return _V2Pool.Contract.V2PoolCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_V2Pool *V2PoolRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _V2Pool.Contract.V2PoolTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_V2Pool *V2PoolRaw) Transact(opts *bind.TransactOpts, method string, params ...any) (*types.Transaction, error) {
	return _V2Pool.Contract.V2PoolTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_V2Pool *V2PoolCallerRaw) Call(opts *bind.CallOpts, result *[]any, method string, params ...any) error {
	return _V2Pool.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_V2Pool *V2PoolTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _V2Pool.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_V2Pool *V2PoolTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...any) (*types.Transaction, error) {
	return _V2Pool.Contract.contract.Transact(opts, method, params...)
}

// CurrentFee is a free data retrieval call binding the contract method 0xda3c300d.
//
// Solidity: function currentFee() view returns(uint24)
func (_V2Pool *V2PoolCaller) CurrentFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "currentFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentFee is a free data retrieval call binding the contract method 0xda3c300d.
//
// Solidity: function currentFee() view returns(uint24)
func (_V2Pool *V2PoolSession) CurrentFee() (*big.Int, error) {
	return _V2Pool.Contract.CurrentFee(&_V2Pool.CallOpts)
}

// CurrentFee is a free data retrieval call binding the contract method 0xda3c300d.
//
// Solidity: function currentFee() view returns(uint24)
func (_V2Pool *V2PoolCallerSession) CurrentFee() (*big.Int, error) {
	return _V2Pool.Contract.CurrentFee(&_V2Pool.CallOpts)
}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_V2Pool *V2PoolCaller) Factory(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "factory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_V2Pool *V2PoolSession) Factory() (common.Address, error) {
	return _V2Pool.Contract.Factory(&_V2Pool.CallOpts)
}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_V2Pool *V2PoolCallerSession) Factory() (common.Address, error) {
	return _V2Pool.Contract.Factory(&_V2Pool.CallOpts)
}

// Fee is a free data retrieval call binding the contract method 0xddca3f43.
//
// Solidity: function fee() view returns(uint24)
func (_V2Pool *V2PoolCaller) Fee(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "fee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Fee is a free data retrieval call binding the contract method 0xddca3f43.
//
// Solidity: function fee() view returns(uint24)
func (_V2Pool *V2PoolSession) Fee() (*big.Int, error) {
	return _V2Pool.Contract.Fee(&_V2Pool.CallOpts)
}

// Fee is a free data retrieval call binding the contract method 0xddca3f43.
//
// Solidity: function fee() view returns(uint24)
func (_V2Pool *V2PoolCallerSession) Fee() (*big.Int, error) {
	return _V2Pool.Contract.Fee(&_V2Pool.CallOpts)
}

// FeeGrowthGlobal0X128 is a free data retrieval call binding the contract method 0xf3058399.
//
// Solidity: function feeGrowthGlobal0X128() view returns(uint256)
func (_V2Pool *V2PoolCaller) FeeGrowthGlobal0X128(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "feeGrowthGlobal0X128")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FeeGrowthGlobal0X128 is a free data retrieval call binding the contract method 0xf3058399.
//
// Solidity: function feeGrowthGlobal0X128() view returns(uint256)
func (_V2Pool *V2PoolSession) FeeGrowthGlobal0X128() (*big.Int, error) {
	return _V2Pool.Contract.FeeGrowthGlobal0X128(&_V2Pool.CallOpts)
}

// FeeGrowthGlobal0X128 is a free data retrieval call binding the contract method 0xf3058399.
//
// Solidity: function feeGrowthGlobal0X128() view returns(uint256)
func (_V2Pool *V2PoolCallerSession) FeeGrowthGlobal0X128() (*big.Int, error) {
	return _V2Pool.Contract.FeeGrowthGlobal0X128(&_V2Pool.CallOpts)
}

// FeeGrowthGlobal1X128 is a free data retrieval call binding the contract method 0x46141319.
//
// Solidity: function feeGrowthGlobal1X128() view returns(uint256)
func (_V2Pool *V2PoolCaller) FeeGrowthGlobal1X128(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "feeGrowthGlobal1X128")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FeeGrowthGlobal1X128 is a free data retrieval call binding the contract method 0x46141319.
//
// Solidity: function feeGrowthGlobal1X128() view returns(uint256)
func (_V2Pool *V2PoolSession) FeeGrowthGlobal1X128() (*big.Int, error) {
	return _V2Pool.Contract.FeeGrowthGlobal1X128(&_V2Pool.CallOpts)
}

// FeeGrowthGlobal1X128 is a free data retrieval call binding the contract method 0x46141319.
//
// Solidity: function feeGrowthGlobal1X128() view returns(uint256)
func (_V2Pool *V2PoolCallerSession) FeeGrowthGlobal1X128() (*big.Int, error) {
	return _V2Pool.Contract.FeeGrowthGlobal1X128(&_V2Pool.CallOpts)
}

// LastPeriod is a free data retrieval call binding the contract method 0xd340ef8a.
//
// Solidity: function lastPeriod() view returns(uint256)
func (_V2Pool *V2PoolCaller) LastPeriod(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "lastPeriod")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LastPeriod is a free data retrieval call binding the contract method 0xd340ef8a.
//
// Solidity: function lastPeriod() view returns(uint256)
func (_V2Pool *V2PoolSession) LastPeriod() (*big.Int, error) {
	return _V2Pool.Contract.LastPeriod(&_V2Pool.CallOpts)
}

// LastPeriod is a free data retrieval call binding the contract method 0xd340ef8a.
//
// Solidity: function lastPeriod() view returns(uint256)
func (_V2Pool *V2PoolCallerSession) LastPeriod() (*big.Int, error) {
	return _V2Pool.Contract.LastPeriod(&_V2Pool.CallOpts)
}

// Liquidity is a free data retrieval call binding the contract method 0x1a686502.
//
// Solidity: function liquidity() view returns(uint128)
func (_V2Pool *V2PoolCaller) Liquidity(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "liquidity")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Liquidity is a free data retrieval call binding the contract method 0x1a686502.
//
// Solidity: function liquidity() view returns(uint128)
func (_V2Pool *V2PoolSession) Liquidity() (*big.Int, error) {
	return _V2Pool.Contract.Liquidity(&_V2Pool.CallOpts)
}

// Liquidity is a free data retrieval call binding the contract method 0x1a686502.
//
// Solidity: function liquidity() view returns(uint128)
func (_V2Pool *V2PoolCallerSession) Liquidity() (*big.Int, error) {
	return _V2Pool.Contract.Liquidity(&_V2Pool.CallOpts)
}

// MaxLiquidityPerTick is a free data retrieval call binding the contract method 0x70cf754a.
//
// Solidity: function maxLiquidityPerTick() view returns(uint128)
func (_V2Pool *V2PoolCaller) MaxLiquidityPerTick(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "maxLiquidityPerTick")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxLiquidityPerTick is a free data retrieval call binding the contract method 0x70cf754a.
//
// Solidity: function maxLiquidityPerTick() view returns(uint128)
func (_V2Pool *V2PoolSession) MaxLiquidityPerTick() (*big.Int, error) {
	return _V2Pool.Contract.MaxLiquidityPerTick(&_V2Pool.CallOpts)
}

// MaxLiquidityPerTick is a free data retrieval call binding the contract method 0x70cf754a.
//
// Solidity: function maxLiquidityPerTick() view returns(uint128)
func (_V2Pool *V2PoolCallerSession) MaxLiquidityPerTick() (*big.Int, error) {
	return _V2Pool.Contract.MaxLiquidityPerTick(&_V2Pool.CallOpts)
}

// Observations is a free data retrieval call binding the contract method 0x252c09d7.
//
// Solidity: function observations(uint256 index) view returns(uint32 blockTimestamp, int56 tickCumulative, uint160 secondsPerLiquidityCumulativeX128, bool initialized, uint160 secondsPerBoostedLiquidityPeriodX128)
func (_V2Pool *V2PoolCaller) Observations(opts *bind.CallOpts, index *big.Int) (struct {
	BlockTimestamp                       uint32
	TickCumulative                       *big.Int
	SecondsPerLiquidityCumulativeX128    *big.Int
	Initialized                          bool
	SecondsPerBoostedLiquidityPeriodX128 *big.Int
}, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "observations", index)

	outstruct := new(struct {
		BlockTimestamp                       uint32
		TickCumulative                       *big.Int
		SecondsPerLiquidityCumulativeX128    *big.Int
		Initialized                          bool
		SecondsPerBoostedLiquidityPeriodX128 *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.BlockTimestamp = *abi.ConvertType(out[0], new(uint32)).(*uint32)
	outstruct.TickCumulative = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.SecondsPerLiquidityCumulativeX128 = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.Initialized = *abi.ConvertType(out[3], new(bool)).(*bool)
	outstruct.SecondsPerBoostedLiquidityPeriodX128 = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Observations is a free data retrieval call binding the contract method 0x252c09d7.
//
// Solidity: function observations(uint256 index) view returns(uint32 blockTimestamp, int56 tickCumulative, uint160 secondsPerLiquidityCumulativeX128, bool initialized, uint160 secondsPerBoostedLiquidityPeriodX128)
func (_V2Pool *V2PoolSession) Observations(index *big.Int) (struct {
	BlockTimestamp                       uint32
	TickCumulative                       *big.Int
	SecondsPerLiquidityCumulativeX128    *big.Int
	Initialized                          bool
	SecondsPerBoostedLiquidityPeriodX128 *big.Int
}, error) {
	return _V2Pool.Contract.Observations(&_V2Pool.CallOpts, index)
}

// Observations is a free data retrieval call binding the contract method 0x252c09d7.
//
// Solidity: function observations(uint256 index) view returns(uint32 blockTimestamp, int56 tickCumulative, uint160 secondsPerLiquidityCumulativeX128, bool initialized, uint160 secondsPerBoostedLiquidityPeriodX128)
func (_V2Pool *V2PoolCallerSession) Observations(index *big.Int) (struct {
	BlockTimestamp                       uint32
	TickCumulative                       *big.Int
	SecondsPerLiquidityCumulativeX128    *big.Int
	Initialized                          bool
	SecondsPerBoostedLiquidityPeriodX128 *big.Int
}, error) {
	return _V2Pool.Contract.Observations(&_V2Pool.CallOpts, index)
}

// Observe is a free data retrieval call binding the contract method 0x883bdbfd.
//
// Solidity: function observe(uint32[] secondsAgos) view returns(int56[] tickCumulatives, uint160[] secondsPerLiquidityCumulativeX128s, uint160[] secondsPerBoostedLiquidityPeriodX128s)
func (_V2Pool *V2PoolCaller) Observe(opts *bind.CallOpts, secondsAgos []uint32) (struct {
	TickCumulatives                       []*big.Int
	SecondsPerLiquidityCumulativeX128s    []*big.Int
	SecondsPerBoostedLiquidityPeriodX128s []*big.Int
}, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "observe", secondsAgos)

	outstruct := new(struct {
		TickCumulatives                       []*big.Int
		SecondsPerLiquidityCumulativeX128s    []*big.Int
		SecondsPerBoostedLiquidityPeriodX128s []*big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.TickCumulatives = *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)
	outstruct.SecondsPerLiquidityCumulativeX128s = *abi.ConvertType(out[1], new([]*big.Int)).(*[]*big.Int)
	outstruct.SecondsPerBoostedLiquidityPeriodX128s = *abi.ConvertType(out[2], new([]*big.Int)).(*[]*big.Int)

	return *outstruct, err

}

// Observe is a free data retrieval call binding the contract method 0x883bdbfd.
//
// Solidity: function observe(uint32[] secondsAgos) view returns(int56[] tickCumulatives, uint160[] secondsPerLiquidityCumulativeX128s, uint160[] secondsPerBoostedLiquidityPeriodX128s)
func (_V2Pool *V2PoolSession) Observe(secondsAgos []uint32) (struct {
	TickCumulatives                       []*big.Int
	SecondsPerLiquidityCumulativeX128s    []*big.Int
	SecondsPerBoostedLiquidityPeriodX128s []*big.Int
}, error) {
	return _V2Pool.Contract.Observe(&_V2Pool.CallOpts, secondsAgos)
}

// Observe is a free data retrieval call binding the contract method 0x883bdbfd.
//
// Solidity: function observe(uint32[] secondsAgos) view returns(int56[] tickCumulatives, uint160[] secondsPerLiquidityCumulativeX128s, uint160[] secondsPerBoostedLiquidityPeriodX128s)
func (_V2Pool *V2PoolCallerSession) Observe(secondsAgos []uint32) (struct {
	TickCumulatives                       []*big.Int
	SecondsPerLiquidityCumulativeX128s    []*big.Int
	SecondsPerBoostedLiquidityPeriodX128s []*big.Int
}, error) {
	return _V2Pool.Contract.Observe(&_V2Pool.CallOpts, secondsAgos)
}

// PeriodCumulativesInside is a free data retrieval call binding the contract method 0xadd5887e.
//
// Solidity: function periodCumulativesInside(uint32 period, int24 tickLower, int24 tickUpper) view returns(uint160 secondsPerLiquidityInsideX128, uint160 secondsPerBoostedLiquidityInsideX128)
func (_V2Pool *V2PoolCaller) PeriodCumulativesInside(opts *bind.CallOpts, period uint32, tickLower *big.Int, tickUpper *big.Int) (struct {
	SecondsPerLiquidityInsideX128        *big.Int
	SecondsPerBoostedLiquidityInsideX128 *big.Int
}, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "periodCumulativesInside", period, tickLower, tickUpper)

	outstruct := new(struct {
		SecondsPerLiquidityInsideX128        *big.Int
		SecondsPerBoostedLiquidityInsideX128 *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.SecondsPerLiquidityInsideX128 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.SecondsPerBoostedLiquidityInsideX128 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// PeriodCumulativesInside is a free data retrieval call binding the contract method 0xadd5887e.
//
// Solidity: function periodCumulativesInside(uint32 period, int24 tickLower, int24 tickUpper) view returns(uint160 secondsPerLiquidityInsideX128, uint160 secondsPerBoostedLiquidityInsideX128)
func (_V2Pool *V2PoolSession) PeriodCumulativesInside(period uint32, tickLower *big.Int, tickUpper *big.Int) (struct {
	SecondsPerLiquidityInsideX128        *big.Int
	SecondsPerBoostedLiquidityInsideX128 *big.Int
}, error) {
	return _V2Pool.Contract.PeriodCumulativesInside(&_V2Pool.CallOpts, period, tickLower, tickUpper)
}

// PeriodCumulativesInside is a free data retrieval call binding the contract method 0xadd5887e.
//
// Solidity: function periodCumulativesInside(uint32 period, int24 tickLower, int24 tickUpper) view returns(uint160 secondsPerLiquidityInsideX128, uint160 secondsPerBoostedLiquidityInsideX128)
func (_V2Pool *V2PoolCallerSession) PeriodCumulativesInside(period uint32, tickLower *big.Int, tickUpper *big.Int) (struct {
	SecondsPerLiquidityInsideX128        *big.Int
	SecondsPerBoostedLiquidityInsideX128 *big.Int
}, error) {
	return _V2Pool.Contract.PeriodCumulativesInside(&_V2Pool.CallOpts, period, tickLower, tickUpper)
}

// Periods is a free data retrieval call binding the contract method 0xea4a1104.
//
// Solidity: function periods(uint256 period) view returns(uint32 previousPeriod, int24 startTick, int24 lastTick, uint160 endSecondsPerLiquidityPeriodX128, uint160 endSecondsPerBoostedLiquidityPeriodX128, uint32 boostedInRange)
func (_V2Pool *V2PoolCaller) Periods(opts *bind.CallOpts, period *big.Int) (struct {
	PreviousPeriod                          uint32
	StartTick                               *big.Int
	LastTick                                *big.Int
	EndSecondsPerLiquidityPeriodX128        *big.Int
	EndSecondsPerBoostedLiquidityPeriodX128 *big.Int
	BoostedInRange                          uint32
}, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "periods", period)

	outstruct := new(struct {
		PreviousPeriod                          uint32
		StartTick                               *big.Int
		LastTick                                *big.Int
		EndSecondsPerLiquidityPeriodX128        *big.Int
		EndSecondsPerBoostedLiquidityPeriodX128 *big.Int
		BoostedInRange                          uint32
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.PreviousPeriod = *abi.ConvertType(out[0], new(uint32)).(*uint32)
	outstruct.StartTick = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.LastTick = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.EndSecondsPerLiquidityPeriodX128 = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.EndSecondsPerBoostedLiquidityPeriodX128 = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.BoostedInRange = *abi.ConvertType(out[5], new(uint32)).(*uint32)

	return *outstruct, err

}

// Periods is a free data retrieval call binding the contract method 0xea4a1104.
//
// Solidity: function periods(uint256 period) view returns(uint32 previousPeriod, int24 startTick, int24 lastTick, uint160 endSecondsPerLiquidityPeriodX128, uint160 endSecondsPerBoostedLiquidityPeriodX128, uint32 boostedInRange)
func (_V2Pool *V2PoolSession) Periods(period *big.Int) (struct {
	PreviousPeriod                          uint32
	StartTick                               *big.Int
	LastTick                                *big.Int
	EndSecondsPerLiquidityPeriodX128        *big.Int
	EndSecondsPerBoostedLiquidityPeriodX128 *big.Int
	BoostedInRange                          uint32
}, error) {
	return _V2Pool.Contract.Periods(&_V2Pool.CallOpts, period)
}

// Periods is a free data retrieval call binding the contract method 0xea4a1104.
//
// Solidity: function periods(uint256 period) view returns(uint32 previousPeriod, int24 startTick, int24 lastTick, uint160 endSecondsPerLiquidityPeriodX128, uint160 endSecondsPerBoostedLiquidityPeriodX128, uint32 boostedInRange)
func (_V2Pool *V2PoolCallerSession) Periods(period *big.Int) (struct {
	PreviousPeriod                          uint32
	StartTick                               *big.Int
	LastTick                                *big.Int
	EndSecondsPerLiquidityPeriodX128        *big.Int
	EndSecondsPerBoostedLiquidityPeriodX128 *big.Int
	BoostedInRange                          uint32
}, error) {
	return _V2Pool.Contract.Periods(&_V2Pool.CallOpts, period)
}

// PositionPeriodDebt is a free data retrieval call binding the contract method 0xdfc8b615.
//
// Solidity: function positionPeriodDebt(uint256 period, address owner, uint256 index, int24 tickLower, int24 tickUpper) view returns(int256 secondsDebtX96, int256 boostedSecondsDebtX96)
func (_V2Pool *V2PoolCaller) PositionPeriodDebt(opts *bind.CallOpts, period *big.Int, owner common.Address, index *big.Int, tickLower *big.Int, tickUpper *big.Int) (struct {
	SecondsDebtX96        *big.Int
	BoostedSecondsDebtX96 *big.Int
}, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "positionPeriodDebt", period, owner, index, tickLower, tickUpper)

	outstruct := new(struct {
		SecondsDebtX96        *big.Int
		BoostedSecondsDebtX96 *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.SecondsDebtX96 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.BoostedSecondsDebtX96 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// PositionPeriodDebt is a free data retrieval call binding the contract method 0xdfc8b615.
//
// Solidity: function positionPeriodDebt(uint256 period, address owner, uint256 index, int24 tickLower, int24 tickUpper) view returns(int256 secondsDebtX96, int256 boostedSecondsDebtX96)
func (_V2Pool *V2PoolSession) PositionPeriodDebt(period *big.Int, owner common.Address, index *big.Int, tickLower *big.Int, tickUpper *big.Int) (struct {
	SecondsDebtX96        *big.Int
	BoostedSecondsDebtX96 *big.Int
}, error) {
	return _V2Pool.Contract.PositionPeriodDebt(&_V2Pool.CallOpts, period, owner, index, tickLower, tickUpper)
}

// PositionPeriodDebt is a free data retrieval call binding the contract method 0xdfc8b615.
//
// Solidity: function positionPeriodDebt(uint256 period, address owner, uint256 index, int24 tickLower, int24 tickUpper) view returns(int256 secondsDebtX96, int256 boostedSecondsDebtX96)
func (_V2Pool *V2PoolCallerSession) PositionPeriodDebt(period *big.Int, owner common.Address, index *big.Int, tickLower *big.Int, tickUpper *big.Int) (struct {
	SecondsDebtX96        *big.Int
	BoostedSecondsDebtX96 *big.Int
}, error) {
	return _V2Pool.Contract.PositionPeriodDebt(&_V2Pool.CallOpts, period, owner, index, tickLower, tickUpper)
}

// PositionPeriodSecondsInRange is a free data retrieval call binding the contract method 0x9918fbb6.
//
// Solidity: function positionPeriodSecondsInRange(uint256 period, address owner, uint256 index, int24 tickLower, int24 tickUpper) view returns(uint256 periodSecondsInsideX96, uint256 periodBoostedSecondsInsideX96)
func (_V2Pool *V2PoolCaller) PositionPeriodSecondsInRange(opts *bind.CallOpts, period *big.Int, owner common.Address, index *big.Int, tickLower *big.Int, tickUpper *big.Int) (struct {
	PeriodSecondsInsideX96        *big.Int
	PeriodBoostedSecondsInsideX96 *big.Int
}, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "positionPeriodSecondsInRange", period, owner, index, tickLower, tickUpper)

	outstruct := new(struct {
		PeriodSecondsInsideX96        *big.Int
		PeriodBoostedSecondsInsideX96 *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.PeriodSecondsInsideX96 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.PeriodBoostedSecondsInsideX96 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// PositionPeriodSecondsInRange is a free data retrieval call binding the contract method 0x9918fbb6.
//
// Solidity: function positionPeriodSecondsInRange(uint256 period, address owner, uint256 index, int24 tickLower, int24 tickUpper) view returns(uint256 periodSecondsInsideX96, uint256 periodBoostedSecondsInsideX96)
func (_V2Pool *V2PoolSession) PositionPeriodSecondsInRange(period *big.Int, owner common.Address, index *big.Int, tickLower *big.Int, tickUpper *big.Int) (struct {
	PeriodSecondsInsideX96        *big.Int
	PeriodBoostedSecondsInsideX96 *big.Int
}, error) {
	return _V2Pool.Contract.PositionPeriodSecondsInRange(&_V2Pool.CallOpts, period, owner, index, tickLower, tickUpper)
}

// PositionPeriodSecondsInRange is a free data retrieval call binding the contract method 0x9918fbb6.
//
// Solidity: function positionPeriodSecondsInRange(uint256 period, address owner, uint256 index, int24 tickLower, int24 tickUpper) view returns(uint256 periodSecondsInsideX96, uint256 periodBoostedSecondsInsideX96)
func (_V2Pool *V2PoolCallerSession) PositionPeriodSecondsInRange(period *big.Int, owner common.Address, index *big.Int, tickLower *big.Int, tickUpper *big.Int) (struct {
	PeriodSecondsInsideX96        *big.Int
	PeriodBoostedSecondsInsideX96 *big.Int
}, error) {
	return _V2Pool.Contract.PositionPeriodSecondsInRange(&_V2Pool.CallOpts, period, owner, index, tickLower, tickUpper)
}

// Positions is a free data retrieval call binding the contract method 0x514ea4bf.
//
// Solidity: function positions(bytes32 key) view returns(uint128 liquidity, uint256 feeGrowthInside0LastX128, uint256 feeGrowthInside1LastX128, uint128 tokensOwed0, uint128 tokensOwed1, uint256 attachedVeRamId)
func (_V2Pool *V2PoolCaller) Positions(opts *bind.CallOpts, key [32]byte) (struct {
	Liquidity                *big.Int
	FeeGrowthInside0LastX128 *big.Int
	FeeGrowthInside1LastX128 *big.Int
	TokensOwed0              *big.Int
	TokensOwed1              *big.Int
	AttachedVeRamId          *big.Int
}, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "positions", key)

	outstruct := new(struct {
		Liquidity                *big.Int
		FeeGrowthInside0LastX128 *big.Int
		FeeGrowthInside1LastX128 *big.Int
		TokensOwed0              *big.Int
		TokensOwed1              *big.Int
		AttachedVeRamId          *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Liquidity = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.FeeGrowthInside0LastX128 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.FeeGrowthInside1LastX128 = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.TokensOwed0 = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.TokensOwed1 = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.AttachedVeRamId = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Positions is a free data retrieval call binding the contract method 0x514ea4bf.
//
// Solidity: function positions(bytes32 key) view returns(uint128 liquidity, uint256 feeGrowthInside0LastX128, uint256 feeGrowthInside1LastX128, uint128 tokensOwed0, uint128 tokensOwed1, uint256 attachedVeRamId)
func (_V2Pool *V2PoolSession) Positions(key [32]byte) (struct {
	Liquidity                *big.Int
	FeeGrowthInside0LastX128 *big.Int
	FeeGrowthInside1LastX128 *big.Int
	TokensOwed0              *big.Int
	TokensOwed1              *big.Int
	AttachedVeRamId          *big.Int
}, error) {
	return _V2Pool.Contract.Positions(&_V2Pool.CallOpts, key)
}

// Positions is a free data retrieval call binding the contract method 0x514ea4bf.
//
// Solidity: function positions(bytes32 key) view returns(uint128 liquidity, uint256 feeGrowthInside0LastX128, uint256 feeGrowthInside1LastX128, uint128 tokensOwed0, uint128 tokensOwed1, uint256 attachedVeRamId)
func (_V2Pool *V2PoolCallerSession) Positions(key [32]byte) (struct {
	Liquidity                *big.Int
	FeeGrowthInside0LastX128 *big.Int
	FeeGrowthInside1LastX128 *big.Int
	TokensOwed0              *big.Int
	TokensOwed1              *big.Int
	AttachedVeRamId          *big.Int
}, error) {
	return _V2Pool.Contract.Positions(&_V2Pool.CallOpts, key)
}

// ProtocolFees is a free data retrieval call binding the contract method 0x1ad8b03b.
//
// Solidity: function protocolFees() view returns(uint128 token0, uint128 token1)
func (_V2Pool *V2PoolCaller) ProtocolFees(opts *bind.CallOpts) (struct {
	Token0 *big.Int
	Token1 *big.Int
}, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "protocolFees")

	outstruct := new(struct {
		Token0 *big.Int
		Token1 *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Token0 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Token1 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// ProtocolFees is a free data retrieval call binding the contract method 0x1ad8b03b.
//
// Solidity: function protocolFees() view returns(uint128 token0, uint128 token1)
func (_V2Pool *V2PoolSession) ProtocolFees() (struct {
	Token0 *big.Int
	Token1 *big.Int
}, error) {
	return _V2Pool.Contract.ProtocolFees(&_V2Pool.CallOpts)
}

// ProtocolFees is a free data retrieval call binding the contract method 0x1ad8b03b.
//
// Solidity: function protocolFees() view returns(uint128 token0, uint128 token1)
func (_V2Pool *V2PoolCallerSession) ProtocolFees() (struct {
	Token0 *big.Int
	Token1 *big.Int
}, error) {
	return _V2Pool.Contract.ProtocolFees(&_V2Pool.CallOpts)
}

// ReadStorage is a free data retrieval call binding the contract method 0xe57c0ca9.
//
// Solidity: function readStorage(bytes32[] slots) view returns(bytes32[] returnData)
func (_V2Pool *V2PoolCaller) ReadStorage(opts *bind.CallOpts, slots [][32]byte) ([][32]byte, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "readStorage", slots)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// ReadStorage is a free data retrieval call binding the contract method 0xe57c0ca9.
//
// Solidity: function readStorage(bytes32[] slots) view returns(bytes32[] returnData)
func (_V2Pool *V2PoolSession) ReadStorage(slots [][32]byte) ([][32]byte, error) {
	return _V2Pool.Contract.ReadStorage(&_V2Pool.CallOpts, slots)
}

// ReadStorage is a free data retrieval call binding the contract method 0xe57c0ca9.
//
// Solidity: function readStorage(bytes32[] slots) view returns(bytes32[] returnData)
func (_V2Pool *V2PoolCallerSession) ReadStorage(slots [][32]byte) ([][32]byte, error) {
	return _V2Pool.Contract.ReadStorage(&_V2Pool.CallOpts, slots)
}

// Slot0 is a free data retrieval call binding the contract method 0x3850c7bd.
//
// Solidity: function slot0() view returns(uint160 sqrtPriceX96, int24 tick, uint16 observationIndex, uint16 observationCardinality, uint16 observationCardinalityNext, uint32 feeProtocol, bool unlocked)
func (_V2Pool *V2PoolCaller) Slot0(opts *bind.CallOpts) (struct {
	SqrtPriceX96               *big.Int
	Tick                       *big.Int
	ObservationIndex           uint16
	ObservationCardinality     uint16
	ObservationCardinalityNext uint16
	FeeProtocol                uint32
	Unlocked                   bool
}, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "slot0")

	outstruct := new(struct {
		SqrtPriceX96               *big.Int
		Tick                       *big.Int
		ObservationIndex           uint16
		ObservationCardinality     uint16
		ObservationCardinalityNext uint16
		FeeProtocol                uint32
		Unlocked                   bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.SqrtPriceX96 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Tick = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.ObservationIndex = *abi.ConvertType(out[2], new(uint16)).(*uint16)
	outstruct.ObservationCardinality = *abi.ConvertType(out[3], new(uint16)).(*uint16)
	outstruct.ObservationCardinalityNext = *abi.ConvertType(out[4], new(uint16)).(*uint16)
	outstruct.FeeProtocol = *abi.ConvertType(out[5], new(uint32)).(*uint32)
	outstruct.Unlocked = *abi.ConvertType(out[6], new(bool)).(*bool)

	return *outstruct, err

}

// Slot0 is a free data retrieval call binding the contract method 0x3850c7bd.
//
// Solidity: function slot0() view returns(uint160 sqrtPriceX96, int24 tick, uint16 observationIndex, uint16 observationCardinality, uint16 observationCardinalityNext, uint32 feeProtocol, bool unlocked)
func (_V2Pool *V2PoolSession) Slot0() (struct {
	SqrtPriceX96               *big.Int
	Tick                       *big.Int
	ObservationIndex           uint16
	ObservationCardinality     uint16
	ObservationCardinalityNext uint16
	FeeProtocol                uint32
	Unlocked                   bool
}, error) {
	return _V2Pool.Contract.Slot0(&_V2Pool.CallOpts)
}

// Slot0 is a free data retrieval call binding the contract method 0x3850c7bd.
//
// Solidity: function slot0() view returns(uint160 sqrtPriceX96, int24 tick, uint16 observationIndex, uint16 observationCardinality, uint16 observationCardinalityNext, uint32 feeProtocol, bool unlocked)
func (_V2Pool *V2PoolCallerSession) Slot0() (struct {
	SqrtPriceX96               *big.Int
	Tick                       *big.Int
	ObservationIndex           uint16
	ObservationCardinality     uint16
	ObservationCardinalityNext uint16
	FeeProtocol                uint32
	Unlocked                   bool
}, error) {
	return _V2Pool.Contract.Slot0(&_V2Pool.CallOpts)
}

// SnapshotCumulativesInside is a free data retrieval call binding the contract method 0xa38807f2.
//
// Solidity: function snapshotCumulativesInside(int24 tickLower, int24 tickUpper) view returns(int56 tickCumulativeInside, uint160 secondsPerLiquidityInsideX128, uint160 secondsPerBoostedLiquidityInsideX128, uint32 secondsInside)
func (_V2Pool *V2PoolCaller) SnapshotCumulativesInside(opts *bind.CallOpts, tickLower *big.Int, tickUpper *big.Int) (struct {
	TickCumulativeInside                 *big.Int
	SecondsPerLiquidityInsideX128        *big.Int
	SecondsPerBoostedLiquidityInsideX128 *big.Int
	SecondsInside                        uint32
}, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "snapshotCumulativesInside", tickLower, tickUpper)

	outstruct := new(struct {
		TickCumulativeInside                 *big.Int
		SecondsPerLiquidityInsideX128        *big.Int
		SecondsPerBoostedLiquidityInsideX128 *big.Int
		SecondsInside                        uint32
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.TickCumulativeInside = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.SecondsPerLiquidityInsideX128 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.SecondsPerBoostedLiquidityInsideX128 = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.SecondsInside = *abi.ConvertType(out[3], new(uint32)).(*uint32)

	return *outstruct, err

}

// SnapshotCumulativesInside is a free data retrieval call binding the contract method 0xa38807f2.
//
// Solidity: function snapshotCumulativesInside(int24 tickLower, int24 tickUpper) view returns(int56 tickCumulativeInside, uint160 secondsPerLiquidityInsideX128, uint160 secondsPerBoostedLiquidityInsideX128, uint32 secondsInside)
func (_V2Pool *V2PoolSession) SnapshotCumulativesInside(tickLower *big.Int, tickUpper *big.Int) (struct {
	TickCumulativeInside                 *big.Int
	SecondsPerLiquidityInsideX128        *big.Int
	SecondsPerBoostedLiquidityInsideX128 *big.Int
	SecondsInside                        uint32
}, error) {
	return _V2Pool.Contract.SnapshotCumulativesInside(&_V2Pool.CallOpts, tickLower, tickUpper)
}

// SnapshotCumulativesInside is a free data retrieval call binding the contract method 0xa38807f2.
//
// Solidity: function snapshotCumulativesInside(int24 tickLower, int24 tickUpper) view returns(int56 tickCumulativeInside, uint160 secondsPerLiquidityInsideX128, uint160 secondsPerBoostedLiquidityInsideX128, uint32 secondsInside)
func (_V2Pool *V2PoolCallerSession) SnapshotCumulativesInside(tickLower *big.Int, tickUpper *big.Int) (struct {
	TickCumulativeInside                 *big.Int
	SecondsPerLiquidityInsideX128        *big.Int
	SecondsPerBoostedLiquidityInsideX128 *big.Int
	SecondsInside                        uint32
}, error) {
	return _V2Pool.Contract.SnapshotCumulativesInside(&_V2Pool.CallOpts, tickLower, tickUpper)
}

// TickBitmap is a free data retrieval call binding the contract method 0x5339c296.
//
// Solidity: function tickBitmap(int16 tick) view returns(uint256)
func (_V2Pool *V2PoolCaller) TickBitmap(opts *bind.CallOpts, tick int16) (*big.Int, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "tickBitmap", tick)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TickBitmap is a free data retrieval call binding the contract method 0x5339c296.
//
// Solidity: function tickBitmap(int16 tick) view returns(uint256)
func (_V2Pool *V2PoolSession) TickBitmap(tick int16) (*big.Int, error) {
	return _V2Pool.Contract.TickBitmap(&_V2Pool.CallOpts, tick)
}

// TickBitmap is a free data retrieval call binding the contract method 0x5339c296.
//
// Solidity: function tickBitmap(int16 tick) view returns(uint256)
func (_V2Pool *V2PoolCallerSession) TickBitmap(tick int16) (*big.Int, error) {
	return _V2Pool.Contract.TickBitmap(&_V2Pool.CallOpts, tick)
}

// TickSpacing is a free data retrieval call binding the contract method 0xd0c93a7c.
//
// Solidity: function tickSpacing() view returns(int24)
func (_V2Pool *V2PoolCaller) TickSpacing(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "tickSpacing")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TickSpacing is a free data retrieval call binding the contract method 0xd0c93a7c.
//
// Solidity: function tickSpacing() view returns(int24)
func (_V2Pool *V2PoolSession) TickSpacing() (*big.Int, error) {
	return _V2Pool.Contract.TickSpacing(&_V2Pool.CallOpts)
}

// TickSpacing is a free data retrieval call binding the contract method 0xd0c93a7c.
//
// Solidity: function tickSpacing() view returns(int24)
func (_V2Pool *V2PoolCallerSession) TickSpacing() (*big.Int, error) {
	return _V2Pool.Contract.TickSpacing(&_V2Pool.CallOpts)
}

// Ticks is a free data retrieval call binding the contract method 0xf30dba93.
//
// Solidity: function ticks(int24 tick) view returns(uint128 liquidityGross, int128 liquidityNet, uint128 boostedLiquidityGross, int128 boostedLiquidityNet, uint256 feeGrowthOutside0X128, uint256 feeGrowthOutside1X128, int56 tickCumulativeOutside, uint160 secondsPerLiquidityOutsideX128, uint32 secondsOutside, bool initialized)
func (_V2Pool *V2PoolCaller) Ticks(opts *bind.CallOpts, tick *big.Int) (struct {
	LiquidityGross                 *big.Int
	LiquidityNet                   *big.Int
	BoostedLiquidityGross          *big.Int
	BoostedLiquidityNet            *big.Int
	FeeGrowthOutside0X128          *big.Int
	FeeGrowthOutside1X128          *big.Int
	TickCumulativeOutside          *big.Int
	SecondsPerLiquidityOutsideX128 *big.Int
	SecondsOutside                 uint32
	Initialized                    bool
}, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "ticks", tick)

	outstruct := new(struct {
		LiquidityGross                 *big.Int
		LiquidityNet                   *big.Int
		BoostedLiquidityGross          *big.Int
		BoostedLiquidityNet            *big.Int
		FeeGrowthOutside0X128          *big.Int
		FeeGrowthOutside1X128          *big.Int
		TickCumulativeOutside          *big.Int
		SecondsPerLiquidityOutsideX128 *big.Int
		SecondsOutside                 uint32
		Initialized                    bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.LiquidityGross = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.LiquidityNet = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.BoostedLiquidityGross = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.BoostedLiquidityNet = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.FeeGrowthOutside0X128 = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.FeeGrowthOutside1X128 = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)
	outstruct.TickCumulativeOutside = *abi.ConvertType(out[6], new(*big.Int)).(**big.Int)
	outstruct.SecondsPerLiquidityOutsideX128 = *abi.ConvertType(out[7], new(*big.Int)).(**big.Int)
	outstruct.SecondsOutside = *abi.ConvertType(out[8], new(uint32)).(*uint32)
	outstruct.Initialized = *abi.ConvertType(out[9], new(bool)).(*bool)

	return *outstruct, err

}

// Ticks is a free data retrieval call binding the contract method 0xf30dba93.
//
// Solidity: function ticks(int24 tick) view returns(uint128 liquidityGross, int128 liquidityNet, uint128 boostedLiquidityGross, int128 boostedLiquidityNet, uint256 feeGrowthOutside0X128, uint256 feeGrowthOutside1X128, int56 tickCumulativeOutside, uint160 secondsPerLiquidityOutsideX128, uint32 secondsOutside, bool initialized)
func (_V2Pool *V2PoolSession) Ticks(tick *big.Int) (struct {
	LiquidityGross                 *big.Int
	LiquidityNet                   *big.Int
	BoostedLiquidityGross          *big.Int
	BoostedLiquidityNet            *big.Int
	FeeGrowthOutside0X128          *big.Int
	FeeGrowthOutside1X128          *big.Int
	TickCumulativeOutside          *big.Int
	SecondsPerLiquidityOutsideX128 *big.Int
	SecondsOutside                 uint32
	Initialized                    bool
}, error) {
	return _V2Pool.Contract.Ticks(&_V2Pool.CallOpts, tick)
}

// Ticks is a free data retrieval call binding the contract method 0xf30dba93.
//
// Solidity: function ticks(int24 tick) view returns(uint128 liquidityGross, int128 liquidityNet, uint128 boostedLiquidityGross, int128 boostedLiquidityNet, uint256 feeGrowthOutside0X128, uint256 feeGrowthOutside1X128, int56 tickCumulativeOutside, uint160 secondsPerLiquidityOutsideX128, uint32 secondsOutside, bool initialized)
func (_V2Pool *V2PoolCallerSession) Ticks(tick *big.Int) (struct {
	LiquidityGross                 *big.Int
	LiquidityNet                   *big.Int
	BoostedLiquidityGross          *big.Int
	BoostedLiquidityNet            *big.Int
	FeeGrowthOutside0X128          *big.Int
	FeeGrowthOutside1X128          *big.Int
	TickCumulativeOutside          *big.Int
	SecondsPerLiquidityOutsideX128 *big.Int
	SecondsOutside                 uint32
	Initialized                    bool
}, error) {
	return _V2Pool.Contract.Ticks(&_V2Pool.CallOpts, tick)
}

// Token0 is a free data retrieval call binding the contract method 0x0dfe1681.
//
// Solidity: function token0() view returns(address)
func (_V2Pool *V2PoolCaller) Token0(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "token0")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Token0 is a free data retrieval call binding the contract method 0x0dfe1681.
//
// Solidity: function token0() view returns(address)
func (_V2Pool *V2PoolSession) Token0() (common.Address, error) {
	return _V2Pool.Contract.Token0(&_V2Pool.CallOpts)
}

// Token0 is a free data retrieval call binding the contract method 0x0dfe1681.
//
// Solidity: function token0() view returns(address)
func (_V2Pool *V2PoolCallerSession) Token0() (common.Address, error) {
	return _V2Pool.Contract.Token0(&_V2Pool.CallOpts)
}

// Token1 is a free data retrieval call binding the contract method 0xd21220a7.
//
// Solidity: function token1() view returns(address)
func (_V2Pool *V2PoolCaller) Token1(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _V2Pool.contract.Call(opts, &out, "token1")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Token1 is a free data retrieval call binding the contract method 0xd21220a7.
//
// Solidity: function token1() view returns(address)
func (_V2Pool *V2PoolSession) Token1() (common.Address, error) {
	return _V2Pool.Contract.Token1(&_V2Pool.CallOpts)
}

// Token1 is a free data retrieval call binding the contract method 0xd21220a7.
//
// Solidity: function token1() view returns(address)
func (_V2Pool *V2PoolCallerSession) Token1() (common.Address, error) {
	return _V2Pool.Contract.Token1(&_V2Pool.CallOpts)
}

// V2PoolBurnIterator is returned from FilterBurn and is used to iterate over the raw logs and unpacked data for Burn events raised by the V2Pool contract.
type V2PoolBurnIterator struct {
	Event *V2PoolBurn // Event containing the contract specifics and raw log

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
func (it *V2PoolBurnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(V2PoolBurn)
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
		it.Event = new(V2PoolBurn)
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
func (it *V2PoolBurnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *V2PoolBurnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// V2PoolBurn represents a Burn event raised by the V2Pool contract.
type V2PoolBurn struct {
	Owner     common.Address
	TickLower *big.Int
	TickUpper *big.Int
	Amount    *big.Int
	Amount0   *big.Int
	Amount1   *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterBurn is a free log retrieval operation binding the contract event 0x0c396cd989a39f4459b5fa1aed6a9a8dcdbc45908acfd67e028cd568da98982c.
//
// Solidity: event Burn(address indexed owner, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
func (_V2Pool *V2PoolFilterer) FilterBurn(opts *bind.FilterOpts, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (*V2PoolBurnIterator, error) {

	var ownerRule []any
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var tickLowerRule []any
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []any
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _V2Pool.contract.FilterLogs(opts, "Burn", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return &V2PoolBurnIterator{contract: _V2Pool.contract, event: "Burn", logs: logs, sub: sub}, nil
}

// WatchBurn is a free log subscription operation binding the contract event 0x0c396cd989a39f4459b5fa1aed6a9a8dcdbc45908acfd67e028cd568da98982c.
//
// Solidity: event Burn(address indexed owner, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
func (_V2Pool *V2PoolFilterer) WatchBurn(opts *bind.WatchOpts, sink chan<- *V2PoolBurn, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (event.Subscription, error) {

	var ownerRule []any
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var tickLowerRule []any
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []any
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _V2Pool.contract.WatchLogs(opts, "Burn", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(V2PoolBurn)
				if err := _V2Pool.contract.UnpackLog(event, "Burn", log); err != nil {
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

// ParseBurn is a log parse operation binding the contract event 0x0c396cd989a39f4459b5fa1aed6a9a8dcdbc45908acfd67e028cd568da98982c.
//
// Solidity: event Burn(address indexed owner, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
func (_V2Pool *V2PoolFilterer) ParseBurn(log types.Log) (*V2PoolBurn, error) {
	event := new(V2PoolBurn)
	if err := _V2Pool.contract.UnpackLog(event, "Burn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// V2PoolCollectIterator is returned from FilterCollect and is used to iterate over the raw logs and unpacked data for Collect events raised by the V2Pool contract.
type V2PoolCollectIterator struct {
	Event *V2PoolCollect // Event containing the contract specifics and raw log

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
func (it *V2PoolCollectIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(V2PoolCollect)
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
		it.Event = new(V2PoolCollect)
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
func (it *V2PoolCollectIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *V2PoolCollectIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// V2PoolCollect represents a Collect event raised by the V2Pool contract.
type V2PoolCollect struct {
	Owner     common.Address
	Recipient common.Address
	TickLower *big.Int
	TickUpper *big.Int
	Amount0   *big.Int
	Amount1   *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterCollect is a free log retrieval operation binding the contract event 0x70935338e69775456a85ddef226c395fb668b63fa0115f5f20610b388e6ca9c0.
//
// Solidity: event Collect(address indexed owner, address recipient, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount0, uint128 amount1)
func (_V2Pool *V2PoolFilterer) FilterCollect(opts *bind.FilterOpts, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (*V2PoolCollectIterator, error) {

	var ownerRule []any
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	var tickLowerRule []any
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []any
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _V2Pool.contract.FilterLogs(opts, "Collect", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return &V2PoolCollectIterator{contract: _V2Pool.contract, event: "Collect", logs: logs, sub: sub}, nil
}

// WatchCollect is a free log subscription operation binding the contract event 0x70935338e69775456a85ddef226c395fb668b63fa0115f5f20610b388e6ca9c0.
//
// Solidity: event Collect(address indexed owner, address recipient, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount0, uint128 amount1)
func (_V2Pool *V2PoolFilterer) WatchCollect(opts *bind.WatchOpts, sink chan<- *V2PoolCollect, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (event.Subscription, error) {

	var ownerRule []any
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	var tickLowerRule []any
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []any
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _V2Pool.contract.WatchLogs(opts, "Collect", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(V2PoolCollect)
				if err := _V2Pool.contract.UnpackLog(event, "Collect", log); err != nil {
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

// ParseCollect is a log parse operation binding the contract event 0x70935338e69775456a85ddef226c395fb668b63fa0115f5f20610b388e6ca9c0.
//
// Solidity: event Collect(address indexed owner, address recipient, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount0, uint128 amount1)
func (_V2Pool *V2PoolFilterer) ParseCollect(log types.Log) (*V2PoolCollect, error) {
	event := new(V2PoolCollect)
	if err := _V2Pool.contract.UnpackLog(event, "Collect", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// V2PoolCollectProtocolIterator is returned from FilterCollectProtocol and is used to iterate over the raw logs and unpacked data for CollectProtocol events raised by the V2Pool contract.
type V2PoolCollectProtocolIterator struct {
	Event *V2PoolCollectProtocol // Event containing the contract specifics and raw log

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
func (it *V2PoolCollectProtocolIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(V2PoolCollectProtocol)
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
		it.Event = new(V2PoolCollectProtocol)
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
func (it *V2PoolCollectProtocolIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *V2PoolCollectProtocolIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// V2PoolCollectProtocol represents a CollectProtocol event raised by the V2Pool contract.
type V2PoolCollectProtocol struct {
	Sender    common.Address
	Recipient common.Address
	Amount0   *big.Int
	Amount1   *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterCollectProtocol is a free log retrieval operation binding the contract event 0x596b573906218d3411850b26a6b437d6c4522fdb43d2d2386263f86d50b8b151.
//
// Solidity: event CollectProtocol(address indexed sender, address indexed recipient, uint128 amount0, uint128 amount1)
func (_V2Pool *V2PoolFilterer) FilterCollectProtocol(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address) (*V2PoolCollectProtocolIterator, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _V2Pool.contract.FilterLogs(opts, "CollectProtocol", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &V2PoolCollectProtocolIterator{contract: _V2Pool.contract, event: "CollectProtocol", logs: logs, sub: sub}, nil
}

// WatchCollectProtocol is a free log subscription operation binding the contract event 0x596b573906218d3411850b26a6b437d6c4522fdb43d2d2386263f86d50b8b151.
//
// Solidity: event CollectProtocol(address indexed sender, address indexed recipient, uint128 amount0, uint128 amount1)
func (_V2Pool *V2PoolFilterer) WatchCollectProtocol(opts *bind.WatchOpts, sink chan<- *V2PoolCollectProtocol, sender []common.Address, recipient []common.Address) (event.Subscription, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _V2Pool.contract.WatchLogs(opts, "CollectProtocol", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(V2PoolCollectProtocol)
				if err := _V2Pool.contract.UnpackLog(event, "CollectProtocol", log); err != nil {
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

// ParseCollectProtocol is a log parse operation binding the contract event 0x596b573906218d3411850b26a6b437d6c4522fdb43d2d2386263f86d50b8b151.
//
// Solidity: event CollectProtocol(address indexed sender, address indexed recipient, uint128 amount0, uint128 amount1)
func (_V2Pool *V2PoolFilterer) ParseCollectProtocol(log types.Log) (*V2PoolCollectProtocol, error) {
	event := new(V2PoolCollectProtocol)
	if err := _V2Pool.contract.UnpackLog(event, "CollectProtocol", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// V2PoolFlashIterator is returned from FilterFlash and is used to iterate over the raw logs and unpacked data for Flash events raised by the V2Pool contract.
type V2PoolFlashIterator struct {
	Event *V2PoolFlash // Event containing the contract specifics and raw log

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
func (it *V2PoolFlashIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(V2PoolFlash)
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
		it.Event = new(V2PoolFlash)
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
func (it *V2PoolFlashIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *V2PoolFlashIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// V2PoolFlash represents a Flash event raised by the V2Pool contract.
type V2PoolFlash struct {
	Sender    common.Address
	Recipient common.Address
	Amount0   *big.Int
	Amount1   *big.Int
	Paid0     *big.Int
	Paid1     *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterFlash is a free log retrieval operation binding the contract event 0xbdbdb71d7860376ba52b25a5028beea23581364a40522f6bcfb86bb1f2dca633.
//
// Solidity: event Flash(address indexed sender, address indexed recipient, uint256 amount0, uint256 amount1, uint256 paid0, uint256 paid1)
func (_V2Pool *V2PoolFilterer) FilterFlash(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address) (*V2PoolFlashIterator, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _V2Pool.contract.FilterLogs(opts, "Flash", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &V2PoolFlashIterator{contract: _V2Pool.contract, event: "Flash", logs: logs, sub: sub}, nil
}

// WatchFlash is a free log subscription operation binding the contract event 0xbdbdb71d7860376ba52b25a5028beea23581364a40522f6bcfb86bb1f2dca633.
//
// Solidity: event Flash(address indexed sender, address indexed recipient, uint256 amount0, uint256 amount1, uint256 paid0, uint256 paid1)
func (_V2Pool *V2PoolFilterer) WatchFlash(opts *bind.WatchOpts, sink chan<- *V2PoolFlash, sender []common.Address, recipient []common.Address) (event.Subscription, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _V2Pool.contract.WatchLogs(opts, "Flash", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(V2PoolFlash)
				if err := _V2Pool.contract.UnpackLog(event, "Flash", log); err != nil {
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

// ParseFlash is a log parse operation binding the contract event 0xbdbdb71d7860376ba52b25a5028beea23581364a40522f6bcfb86bb1f2dca633.
//
// Solidity: event Flash(address indexed sender, address indexed recipient, uint256 amount0, uint256 amount1, uint256 paid0, uint256 paid1)
func (_V2Pool *V2PoolFilterer) ParseFlash(log types.Log) (*V2PoolFlash, error) {
	event := new(V2PoolFlash)
	if err := _V2Pool.contract.UnpackLog(event, "Flash", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// V2PoolIncreaseObservationCardinalityNextIterator is returned from FilterIncreaseObservationCardinalityNext and is used to iterate over the raw logs and unpacked data for IncreaseObservationCardinalityNext events raised by the V2Pool contract.
type V2PoolIncreaseObservationCardinalityNextIterator struct {
	Event *V2PoolIncreaseObservationCardinalityNext // Event containing the contract specifics and raw log

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
func (it *V2PoolIncreaseObservationCardinalityNextIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(V2PoolIncreaseObservationCardinalityNext)
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
		it.Event = new(V2PoolIncreaseObservationCardinalityNext)
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
func (it *V2PoolIncreaseObservationCardinalityNextIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *V2PoolIncreaseObservationCardinalityNextIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// V2PoolIncreaseObservationCardinalityNext represents a IncreaseObservationCardinalityNext event raised by the V2Pool contract.
type V2PoolIncreaseObservationCardinalityNext struct {
	ObservationCardinalityNextOld uint16
	ObservationCardinalityNextNew uint16
	Raw                           types.Log // Blockchain specific contextual infos
}

// FilterIncreaseObservationCardinalityNext is a free log retrieval operation binding the contract event 0xac49e518f90a358f652e4400164f05a5d8f7e35e7747279bc3a93dbf584e125a.
//
// Solidity: event IncreaseObservationCardinalityNext(uint16 observationCardinalityNextOld, uint16 observationCardinalityNextNew)
func (_V2Pool *V2PoolFilterer) FilterIncreaseObservationCardinalityNext(opts *bind.FilterOpts) (*V2PoolIncreaseObservationCardinalityNextIterator, error) {

	logs, sub, err := _V2Pool.contract.FilterLogs(opts, "IncreaseObservationCardinalityNext")
	if err != nil {
		return nil, err
	}
	return &V2PoolIncreaseObservationCardinalityNextIterator{contract: _V2Pool.contract, event: "IncreaseObservationCardinalityNext", logs: logs, sub: sub}, nil
}

// WatchIncreaseObservationCardinalityNext is a free log subscription operation binding the contract event 0xac49e518f90a358f652e4400164f05a5d8f7e35e7747279bc3a93dbf584e125a.
//
// Solidity: event IncreaseObservationCardinalityNext(uint16 observationCardinalityNextOld, uint16 observationCardinalityNextNew)
func (_V2Pool *V2PoolFilterer) WatchIncreaseObservationCardinalityNext(opts *bind.WatchOpts, sink chan<- *V2PoolIncreaseObservationCardinalityNext) (event.Subscription, error) {

	logs, sub, err := _V2Pool.contract.WatchLogs(opts, "IncreaseObservationCardinalityNext")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(V2PoolIncreaseObservationCardinalityNext)
				if err := _V2Pool.contract.UnpackLog(event, "IncreaseObservationCardinalityNext", log); err != nil {
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

// ParseIncreaseObservationCardinalityNext is a log parse operation binding the contract event 0xac49e518f90a358f652e4400164f05a5d8f7e35e7747279bc3a93dbf584e125a.
//
// Solidity: event IncreaseObservationCardinalityNext(uint16 observationCardinalityNextOld, uint16 observationCardinalityNextNew)
func (_V2Pool *V2PoolFilterer) ParseIncreaseObservationCardinalityNext(log types.Log) (*V2PoolIncreaseObservationCardinalityNext, error) {
	event := new(V2PoolIncreaseObservationCardinalityNext)
	if err := _V2Pool.contract.UnpackLog(event, "IncreaseObservationCardinalityNext", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// V2PoolInitializeIterator is returned from FilterInitialize and is used to iterate over the raw logs and unpacked data for Initialize events raised by the V2Pool contract.
type V2PoolInitializeIterator struct {
	Event *V2PoolInitialize // Event containing the contract specifics and raw log

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
func (it *V2PoolInitializeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(V2PoolInitialize)
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
		it.Event = new(V2PoolInitialize)
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
func (it *V2PoolInitializeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *V2PoolInitializeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// V2PoolInitialize represents a Initialize event raised by the V2Pool contract.
type V2PoolInitialize struct {
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterInitialize is a free log retrieval operation binding the contract event 0x98636036cb66a9c19a37435efc1e90142190214e8abeb821bdba3f2990dd4c95.
//
// Solidity: event Initialize(uint160 sqrtPriceX96, int24 tick)
func (_V2Pool *V2PoolFilterer) FilterInitialize(opts *bind.FilterOpts) (*V2PoolInitializeIterator, error) {

	logs, sub, err := _V2Pool.contract.FilterLogs(opts, "Initialize")
	if err != nil {
		return nil, err
	}
	return &V2PoolInitializeIterator{contract: _V2Pool.contract, event: "Initialize", logs: logs, sub: sub}, nil
}

// WatchInitialize is a free log subscription operation binding the contract event 0x98636036cb66a9c19a37435efc1e90142190214e8abeb821bdba3f2990dd4c95.
//
// Solidity: event Initialize(uint160 sqrtPriceX96, int24 tick)
func (_V2Pool *V2PoolFilterer) WatchInitialize(opts *bind.WatchOpts, sink chan<- *V2PoolInitialize) (event.Subscription, error) {

	logs, sub, err := _V2Pool.contract.WatchLogs(opts, "Initialize")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(V2PoolInitialize)
				if err := _V2Pool.contract.UnpackLog(event, "Initialize", log); err != nil {
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

// ParseInitialize is a log parse operation binding the contract event 0x98636036cb66a9c19a37435efc1e90142190214e8abeb821bdba3f2990dd4c95.
//
// Solidity: event Initialize(uint160 sqrtPriceX96, int24 tick)
func (_V2Pool *V2PoolFilterer) ParseInitialize(log types.Log) (*V2PoolInitialize, error) {
	event := new(V2PoolInitialize)
	if err := _V2Pool.contract.UnpackLog(event, "Initialize", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// V2PoolMintIterator is returned from FilterMint and is used to iterate over the raw logs and unpacked data for Mint events raised by the V2Pool contract.
type V2PoolMintIterator struct {
	Event *V2PoolMint // Event containing the contract specifics and raw log

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
func (it *V2PoolMintIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(V2PoolMint)
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
		it.Event = new(V2PoolMint)
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
func (it *V2PoolMintIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *V2PoolMintIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// V2PoolMint represents a Mint event raised by the V2Pool contract.
type V2PoolMint struct {
	Sender    common.Address
	Owner     common.Address
	TickLower *big.Int
	TickUpper *big.Int
	Amount    *big.Int
	Amount0   *big.Int
	Amount1   *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterMint is a free log retrieval operation binding the contract event 0x7a53080ba414158be7ec69b987b5fb7d07dee101fe85488f0853ae16239d0bde.
//
// Solidity: event Mint(address sender, address indexed owner, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
func (_V2Pool *V2PoolFilterer) FilterMint(opts *bind.FilterOpts, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (*V2PoolMintIterator, error) {

	var ownerRule []any
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var tickLowerRule []any
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []any
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _V2Pool.contract.FilterLogs(opts, "Mint", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return &V2PoolMintIterator{contract: _V2Pool.contract, event: "Mint", logs: logs, sub: sub}, nil
}

// WatchMint is a free log subscription operation binding the contract event 0x7a53080ba414158be7ec69b987b5fb7d07dee101fe85488f0853ae16239d0bde.
//
// Solidity: event Mint(address sender, address indexed owner, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
func (_V2Pool *V2PoolFilterer) WatchMint(opts *bind.WatchOpts, sink chan<- *V2PoolMint, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (event.Subscription, error) {

	var ownerRule []any
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var tickLowerRule []any
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []any
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _V2Pool.contract.WatchLogs(opts, "Mint", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(V2PoolMint)
				if err := _V2Pool.contract.UnpackLog(event, "Mint", log); err != nil {
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

// ParseMint is a log parse operation binding the contract event 0x7a53080ba414158be7ec69b987b5fb7d07dee101fe85488f0853ae16239d0bde.
//
// Solidity: event Mint(address sender, address indexed owner, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
func (_V2Pool *V2PoolFilterer) ParseMint(log types.Log) (*V2PoolMint, error) {
	event := new(V2PoolMint)
	if err := _V2Pool.contract.UnpackLog(event, "Mint", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// V2PoolSetFeeProtocolIterator is returned from FilterSetFeeProtocol and is used to iterate over the raw logs and unpacked data for SetFeeProtocol events raised by the V2Pool contract.
type V2PoolSetFeeProtocolIterator struct {
	Event *V2PoolSetFeeProtocol // Event containing the contract specifics and raw log

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
func (it *V2PoolSetFeeProtocolIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(V2PoolSetFeeProtocol)
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
		it.Event = new(V2PoolSetFeeProtocol)
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
func (it *V2PoolSetFeeProtocolIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *V2PoolSetFeeProtocolIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// V2PoolSetFeeProtocol represents a SetFeeProtocol event raised by the V2Pool contract.
type V2PoolSetFeeProtocol struct {
	FeeProtocol0Old uint8
	FeeProtocol1Old uint8
	FeeProtocol0New uint8
	FeeProtocol1New uint8
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterSetFeeProtocol is a free log retrieval operation binding the contract event 0x973d8d92bb299f4af6ce49b52a8adb85ae46b9f214c4c4fc06ac77401237b133.
//
// Solidity: event SetFeeProtocol(uint8 feeProtocol0Old, uint8 feeProtocol1Old, uint8 feeProtocol0New, uint8 feeProtocol1New)
func (_V2Pool *V2PoolFilterer) FilterSetFeeProtocol(opts *bind.FilterOpts) (*V2PoolSetFeeProtocolIterator, error) {

	logs, sub, err := _V2Pool.contract.FilterLogs(opts, "SetFeeProtocol")
	if err != nil {
		return nil, err
	}
	return &V2PoolSetFeeProtocolIterator{contract: _V2Pool.contract, event: "SetFeeProtocol", logs: logs, sub: sub}, nil
}

// WatchSetFeeProtocol is a free log subscription operation binding the contract event 0x973d8d92bb299f4af6ce49b52a8adb85ae46b9f214c4c4fc06ac77401237b133.
//
// Solidity: event SetFeeProtocol(uint8 feeProtocol0Old, uint8 feeProtocol1Old, uint8 feeProtocol0New, uint8 feeProtocol1New)
func (_V2Pool *V2PoolFilterer) WatchSetFeeProtocol(opts *bind.WatchOpts, sink chan<- *V2PoolSetFeeProtocol) (event.Subscription, error) {

	logs, sub, err := _V2Pool.contract.WatchLogs(opts, "SetFeeProtocol")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(V2PoolSetFeeProtocol)
				if err := _V2Pool.contract.UnpackLog(event, "SetFeeProtocol", log); err != nil {
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

// ParseSetFeeProtocol is a log parse operation binding the contract event 0x973d8d92bb299f4af6ce49b52a8adb85ae46b9f214c4c4fc06ac77401237b133.
//
// Solidity: event SetFeeProtocol(uint8 feeProtocol0Old, uint8 feeProtocol1Old, uint8 feeProtocol0New, uint8 feeProtocol1New)
func (_V2Pool *V2PoolFilterer) ParseSetFeeProtocol(log types.Log) (*V2PoolSetFeeProtocol, error) {
	event := new(V2PoolSetFeeProtocol)
	if err := _V2Pool.contract.UnpackLog(event, "SetFeeProtocol", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// V2PoolSwapIterator is returned from FilterSwap and is used to iterate over the raw logs and unpacked data for Swap events raised by the V2Pool contract.
type V2PoolSwapIterator struct {
	Event *V2PoolSwap // Event containing the contract specifics and raw log

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
func (it *V2PoolSwapIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(V2PoolSwap)
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
		it.Event = new(V2PoolSwap)
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
func (it *V2PoolSwapIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *V2PoolSwapIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// V2PoolSwap represents a Swap event raised by the V2Pool contract.
type V2PoolSwap struct {
	Sender       common.Address
	Recipient    common.Address
	Amount0      *big.Int
	Amount1      *big.Int
	SqrtPriceX96 *big.Int
	Liquidity    *big.Int
	Tick         *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterSwap is a free log retrieval operation binding the contract event 0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67.
//
// Solidity: event Swap(address indexed sender, address indexed recipient, int256 amount0, int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick)
func (_V2Pool *V2PoolFilterer) FilterSwap(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address) (*V2PoolSwapIterator, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _V2Pool.contract.FilterLogs(opts, "Swap", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &V2PoolSwapIterator{contract: _V2Pool.contract, event: "Swap", logs: logs, sub: sub}, nil
}

// WatchSwap is a free log subscription operation binding the contract event 0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67.
//
// Solidity: event Swap(address indexed sender, address indexed recipient, int256 amount0, int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick)
func (_V2Pool *V2PoolFilterer) WatchSwap(opts *bind.WatchOpts, sink chan<- *V2PoolSwap, sender []common.Address, recipient []common.Address) (event.Subscription, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _V2Pool.contract.WatchLogs(opts, "Swap", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(V2PoolSwap)
				if err := _V2Pool.contract.UnpackLog(event, "Swap", log); err != nil {
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

// ParseSwap is a log parse operation binding the contract event 0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67.
//
// Solidity: event Swap(address indexed sender, address indexed recipient, int256 amount0, int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick)
func (_V2Pool *V2PoolFilterer) ParseSwap(log types.Log) (*V2PoolSwap, error) {
	event := new(V2PoolSwap)
	if err := _V2Pool.contract.UnpackLog(event, "Swap", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
