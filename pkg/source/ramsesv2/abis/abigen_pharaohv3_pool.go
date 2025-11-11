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

// PharaohV3MetaData contains all meta data concerning the PharaohV3 contract.
var PharaohV3MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AI\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AS\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"F0\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"F1\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"I\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"IIA\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"L\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LOK\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"M0\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"M1\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NOT_AUTHORIZED\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OLD\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"R\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"SPL\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"T\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TF\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TLM\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TLU\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TUM\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"name\":\"Burn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount0\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount1\",\"type\":\"uint128\"}],\"name\":\"Collect\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount0\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount1\",\"type\":\"uint128\"}],\"name\":\"CollectProtocol\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"paid0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"paid1\",\"type\":\"uint256\"}],\"name\":\"Flash\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"observationCardinalityNextOld\",\"type\":\"uint16\"},{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"observationCardinalityNextNew\",\"type\":\"uint16\"}],\"name\":\"IncreaseObservationCardinalityNext\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"Initialize\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"indexed\":true,\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"name\":\"Mint\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"feeProtocol0Old\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"feeProtocol1Old\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"feeProtocol0New\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"feeProtocol1New\",\"type\":\"uint8\"}],\"name\":\"SetFeeProtocol\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"amount0\",\"type\":\"int256\"},{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"amount1\",\"type\":\"int256\"},{\"indexed\":false,\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"liquidity\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"Swap\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"_advancePeriod\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"}],\"name\":\"burn\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amount0Requested\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1Requested\",\"type\":\"uint128\"}],\"name\":\"collect\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"amount0\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1\",\"type\":\"uint128\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint128\",\"name\":\"amount0Requested\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1Requested\",\"type\":\"uint128\"}],\"name\":\"collectProtocol\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"amount0\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amount1\",\"type\":\"uint128\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"factory\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"fee\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"feeGrowthGlobal0X128\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"feeGrowthGlobal1X128\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"flash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"grossFeeGrowthGlobal0X128\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"grossFeeGrowthGlobal1X128\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"observationCardinalityNext\",\"type\":\"uint16\"}],\"name\":\"increaseObservationCardinalityNext\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"lastPeriod\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"liquidity\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxLiquidityPerTick\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"mint\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"observations\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"blockTimestamp\",\"type\":\"uint32\"},{\"internalType\":\"int56\",\"name\":\"tickCumulative\",\"type\":\"int56\"},{\"internalType\":\"uint160\",\"name\":\"secondsPerLiquidityCumulativeX128\",\"type\":\"uint160\"},{\"internalType\":\"bool\",\"name\":\"initialized\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32[]\",\"name\":\"secondsAgos\",\"type\":\"uint32[]\"}],\"name\":\"observe\",\"outputs\":[{\"internalType\":\"int56[]\",\"name\":\"tickCumulatives\",\"type\":\"int56[]\"},{\"internalType\":\"uint160[]\",\"name\":\"secondsPerLiquidityCumulativeX128s\",\"type\":\"uint160[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"period\",\"type\":\"uint256\"}],\"name\":\"periods\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"previousPeriod\",\"type\":\"uint32\"},{\"internalType\":\"int24\",\"name\":\"startTick\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"lastTick\",\"type\":\"int24\"},{\"internalType\":\"uint160\",\"name\":\"endSecondsPerLiquidityPeriodX128\",\"type\":\"uint160\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"period\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"}],\"name\":\"positionPeriodSecondsInRange\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"periodSecondsInsideX96\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"key\",\"type\":\"bytes32\"}],\"name\":\"positions\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"liquidity\",\"type\":\"uint128\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthInside0LastX128\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthInside1LastX128\",\"type\":\"uint256\"},{\"internalType\":\"uint128\",\"name\":\"tokensOwed0\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"tokensOwed1\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"protocolFees\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"slots\",\"type\":\"bytes32[]\"}],\"name\":\"readStorage\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"returnData\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint24\",\"name\":\"_fee\",\"type\":\"uint24\"}],\"name\":\"setFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"setFeeProtocol\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"slot0\",\"outputs\":[{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"},{\"internalType\":\"uint16\",\"name\":\"observationIndex\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"observationCardinality\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"observationCardinalityNext\",\"type\":\"uint16\"},{\"internalType\":\"uint24\",\"name\":\"feeProtocol\",\"type\":\"uint24\"},{\"internalType\":\"bool\",\"name\":\"unlocked\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"}],\"name\":\"snapshotCumulativesInside\",\"outputs\":[{\"internalType\":\"int56\",\"name\":\"tickCumulativeInside\",\"type\":\"int56\"},{\"internalType\":\"uint160\",\"name\":\"secondsPerLiquidityInsideX128\",\"type\":\"uint160\"},{\"internalType\":\"uint32\",\"name\":\"secondsInside\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountSpecified\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceLimitX96\",\"type\":\"uint160\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"swap\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"amount0\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"amount1\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int16\",\"name\":\"tick\",\"type\":\"int16\"}],\"name\":\"tickBitmap\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"tickSpacing\",\"outputs\":[{\"internalType\":\"int24\",\"name\":\"\",\"type\":\"int24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"ticks\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"liquidityGross\",\"type\":\"uint128\"},{\"internalType\":\"int128\",\"name\":\"liquidityNet\",\"type\":\"int128\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthOutside0X128\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"feeGrowthOutside1X128\",\"type\":\"uint256\"},{\"internalType\":\"int56\",\"name\":\"tickCumulativeOutside\",\"type\":\"int56\"},{\"internalType\":\"uint160\",\"name\":\"secondsPerLiquidityOutsideX128\",\"type\":\"uint160\"},{\"internalType\":\"uint32\",\"name\":\"secondsOutside\",\"type\":\"uint32\"},{\"internalType\":\"bool\",\"name\":\"initialized\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"token0\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"token1\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// PharaohV3ABI is the input ABI used to generate the binding from.
// Deprecated: Use PharaohV3MetaData.ABI instead.
var PharaohV3ABI = PharaohV3MetaData.ABI

// PharaohV3 is an auto generated Go binding around an Ethereum contract.
type PharaohV3 struct {
	PharaohV3Caller     // Read-only binding to the contract
	PharaohV3Transactor // Write-only binding to the contract
	PharaohV3Filterer   // Log filterer for contract events
}

// PharaohV3Caller is an auto generated read-only Go binding around an Ethereum contract.
type PharaohV3Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PharaohV3Transactor is an auto generated write-only Go binding around an Ethereum contract.
type PharaohV3Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PharaohV3Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PharaohV3Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PharaohV3Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PharaohV3Session struct {
	Contract     *PharaohV3        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PharaohV3CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PharaohV3CallerSession struct {
	Contract *PharaohV3Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// PharaohV3TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PharaohV3TransactorSession struct {
	Contract     *PharaohV3Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// PharaohV3Raw is an auto generated low-level Go binding around an Ethereum contract.
type PharaohV3Raw struct {
	Contract *PharaohV3 // Generic contract binding to access the raw methods on
}

// PharaohV3CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PharaohV3CallerRaw struct {
	Contract *PharaohV3Caller // Generic read-only contract binding to access the raw methods on
}

// PharaohV3TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PharaohV3TransactorRaw struct {
	Contract *PharaohV3Transactor // Generic write-only contract binding to access the raw methods on
}

// NewPharaohV3 creates a new instance of PharaohV3, bound to a specific deployed contract.
func NewPharaohV3(address common.Address, backend bind.ContractBackend) (*PharaohV3, error) {
	contract, err := bindPharaohV3(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PharaohV3{PharaohV3Caller: PharaohV3Caller{contract: contract}, PharaohV3Transactor: PharaohV3Transactor{contract: contract}, PharaohV3Filterer: PharaohV3Filterer{contract: contract}}, nil
}

// NewPharaohV3Caller creates a new read-only instance of PharaohV3, bound to a specific deployed contract.
func NewPharaohV3Caller(address common.Address, caller bind.ContractCaller) (*PharaohV3Caller, error) {
	contract, err := bindPharaohV3(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PharaohV3Caller{contract: contract}, nil
}

// NewPharaohV3Transactor creates a new write-only instance of PharaohV3, bound to a specific deployed contract.
func NewPharaohV3Transactor(address common.Address, transactor bind.ContractTransactor) (*PharaohV3Transactor, error) {
	contract, err := bindPharaohV3(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PharaohV3Transactor{contract: contract}, nil
}

// NewPharaohV3Filterer creates a new log filterer instance of PharaohV3, bound to a specific deployed contract.
func NewPharaohV3Filterer(address common.Address, filterer bind.ContractFilterer) (*PharaohV3Filterer, error) {
	contract, err := bindPharaohV3(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PharaohV3Filterer{contract: contract}, nil
}

// bindPharaohV3 binds a generic wrapper to an already deployed contract.
func bindPharaohV3(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := PharaohV3MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PharaohV3 *PharaohV3Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PharaohV3.Contract.PharaohV3Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PharaohV3 *PharaohV3Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PharaohV3.Contract.PharaohV3Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PharaohV3 *PharaohV3Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PharaohV3.Contract.PharaohV3Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PharaohV3 *PharaohV3CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PharaohV3.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PharaohV3 *PharaohV3TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PharaohV3.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PharaohV3 *PharaohV3TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PharaohV3.Contract.contract.Transact(opts, method, params...)
}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_PharaohV3 *PharaohV3Caller) Factory(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "factory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_PharaohV3 *PharaohV3Session) Factory() (common.Address, error) {
	return _PharaohV3.Contract.Factory(&_PharaohV3.CallOpts)
}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_PharaohV3 *PharaohV3CallerSession) Factory() (common.Address, error) {
	return _PharaohV3.Contract.Factory(&_PharaohV3.CallOpts)
}

// Fee is a free data retrieval call binding the contract method 0xddca3f43.
//
// Solidity: function fee() view returns(uint24)
func (_PharaohV3 *PharaohV3Caller) Fee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "fee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Fee is a free data retrieval call binding the contract method 0xddca3f43.
//
// Solidity: function fee() view returns(uint24)
func (_PharaohV3 *PharaohV3Session) Fee() (*big.Int, error) {
	return _PharaohV3.Contract.Fee(&_PharaohV3.CallOpts)
}

// Fee is a free data retrieval call binding the contract method 0xddca3f43.
//
// Solidity: function fee() view returns(uint24)
func (_PharaohV3 *PharaohV3CallerSession) Fee() (*big.Int, error) {
	return _PharaohV3.Contract.Fee(&_PharaohV3.CallOpts)
}

// FeeGrowthGlobal0X128 is a free data retrieval call binding the contract method 0xf3058399.
//
// Solidity: function feeGrowthGlobal0X128() view returns(uint256)
func (_PharaohV3 *PharaohV3Caller) FeeGrowthGlobal0X128(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "feeGrowthGlobal0X128")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FeeGrowthGlobal0X128 is a free data retrieval call binding the contract method 0xf3058399.
//
// Solidity: function feeGrowthGlobal0X128() view returns(uint256)
func (_PharaohV3 *PharaohV3Session) FeeGrowthGlobal0X128() (*big.Int, error) {
	return _PharaohV3.Contract.FeeGrowthGlobal0X128(&_PharaohV3.CallOpts)
}

// FeeGrowthGlobal0X128 is a free data retrieval call binding the contract method 0xf3058399.
//
// Solidity: function feeGrowthGlobal0X128() view returns(uint256)
func (_PharaohV3 *PharaohV3CallerSession) FeeGrowthGlobal0X128() (*big.Int, error) {
	return _PharaohV3.Contract.FeeGrowthGlobal0X128(&_PharaohV3.CallOpts)
}

// FeeGrowthGlobal1X128 is a free data retrieval call binding the contract method 0x46141319.
//
// Solidity: function feeGrowthGlobal1X128() view returns(uint256)
func (_PharaohV3 *PharaohV3Caller) FeeGrowthGlobal1X128(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "feeGrowthGlobal1X128")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FeeGrowthGlobal1X128 is a free data retrieval call binding the contract method 0x46141319.
//
// Solidity: function feeGrowthGlobal1X128() view returns(uint256)
func (_PharaohV3 *PharaohV3Session) FeeGrowthGlobal1X128() (*big.Int, error) {
	return _PharaohV3.Contract.FeeGrowthGlobal1X128(&_PharaohV3.CallOpts)
}

// FeeGrowthGlobal1X128 is a free data retrieval call binding the contract method 0x46141319.
//
// Solidity: function feeGrowthGlobal1X128() view returns(uint256)
func (_PharaohV3 *PharaohV3CallerSession) FeeGrowthGlobal1X128() (*big.Int, error) {
	return _PharaohV3.Contract.FeeGrowthGlobal1X128(&_PharaohV3.CallOpts)
}

// GrossFeeGrowthGlobal0X128 is a free data retrieval call binding the contract method 0x461dfd23.
//
// Solidity: function grossFeeGrowthGlobal0X128() view returns(uint256)
func (_PharaohV3 *PharaohV3Caller) GrossFeeGrowthGlobal0X128(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "grossFeeGrowthGlobal0X128")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GrossFeeGrowthGlobal0X128 is a free data retrieval call binding the contract method 0x461dfd23.
//
// Solidity: function grossFeeGrowthGlobal0X128() view returns(uint256)
func (_PharaohV3 *PharaohV3Session) GrossFeeGrowthGlobal0X128() (*big.Int, error) {
	return _PharaohV3.Contract.GrossFeeGrowthGlobal0X128(&_PharaohV3.CallOpts)
}

// GrossFeeGrowthGlobal0X128 is a free data retrieval call binding the contract method 0x461dfd23.
//
// Solidity: function grossFeeGrowthGlobal0X128() view returns(uint256)
func (_PharaohV3 *PharaohV3CallerSession) GrossFeeGrowthGlobal0X128() (*big.Int, error) {
	return _PharaohV3.Contract.GrossFeeGrowthGlobal0X128(&_PharaohV3.CallOpts)
}

// GrossFeeGrowthGlobal1X128 is a free data retrieval call binding the contract method 0xe8169d4b.
//
// Solidity: function grossFeeGrowthGlobal1X128() view returns(uint256)
func (_PharaohV3 *PharaohV3Caller) GrossFeeGrowthGlobal1X128(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "grossFeeGrowthGlobal1X128")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GrossFeeGrowthGlobal1X128 is a free data retrieval call binding the contract method 0xe8169d4b.
//
// Solidity: function grossFeeGrowthGlobal1X128() view returns(uint256)
func (_PharaohV3 *PharaohV3Session) GrossFeeGrowthGlobal1X128() (*big.Int, error) {
	return _PharaohV3.Contract.GrossFeeGrowthGlobal1X128(&_PharaohV3.CallOpts)
}

// GrossFeeGrowthGlobal1X128 is a free data retrieval call binding the contract method 0xe8169d4b.
//
// Solidity: function grossFeeGrowthGlobal1X128() view returns(uint256)
func (_PharaohV3 *PharaohV3CallerSession) GrossFeeGrowthGlobal1X128() (*big.Int, error) {
	return _PharaohV3.Contract.GrossFeeGrowthGlobal1X128(&_PharaohV3.CallOpts)
}

// LastPeriod is a free data retrieval call binding the contract method 0xd340ef8a.
//
// Solidity: function lastPeriod() view returns(uint256)
func (_PharaohV3 *PharaohV3Caller) LastPeriod(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "lastPeriod")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LastPeriod is a free data retrieval call binding the contract method 0xd340ef8a.
//
// Solidity: function lastPeriod() view returns(uint256)
func (_PharaohV3 *PharaohV3Session) LastPeriod() (*big.Int, error) {
	return _PharaohV3.Contract.LastPeriod(&_PharaohV3.CallOpts)
}

// LastPeriod is a free data retrieval call binding the contract method 0xd340ef8a.
//
// Solidity: function lastPeriod() view returns(uint256)
func (_PharaohV3 *PharaohV3CallerSession) LastPeriod() (*big.Int, error) {
	return _PharaohV3.Contract.LastPeriod(&_PharaohV3.CallOpts)
}

// Liquidity is a free data retrieval call binding the contract method 0x1a686502.
//
// Solidity: function liquidity() view returns(uint128)
func (_PharaohV3 *PharaohV3Caller) Liquidity(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "liquidity")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Liquidity is a free data retrieval call binding the contract method 0x1a686502.
//
// Solidity: function liquidity() view returns(uint128)
func (_PharaohV3 *PharaohV3Session) Liquidity() (*big.Int, error) {
	return _PharaohV3.Contract.Liquidity(&_PharaohV3.CallOpts)
}

// Liquidity is a free data retrieval call binding the contract method 0x1a686502.
//
// Solidity: function liquidity() view returns(uint128)
func (_PharaohV3 *PharaohV3CallerSession) Liquidity() (*big.Int, error) {
	return _PharaohV3.Contract.Liquidity(&_PharaohV3.CallOpts)
}

// MaxLiquidityPerTick is a free data retrieval call binding the contract method 0x70cf754a.
//
// Solidity: function maxLiquidityPerTick() view returns(uint128)
func (_PharaohV3 *PharaohV3Caller) MaxLiquidityPerTick(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "maxLiquidityPerTick")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxLiquidityPerTick is a free data retrieval call binding the contract method 0x70cf754a.
//
// Solidity: function maxLiquidityPerTick() view returns(uint128)
func (_PharaohV3 *PharaohV3Session) MaxLiquidityPerTick() (*big.Int, error) {
	return _PharaohV3.Contract.MaxLiquidityPerTick(&_PharaohV3.CallOpts)
}

// MaxLiquidityPerTick is a free data retrieval call binding the contract method 0x70cf754a.
//
// Solidity: function maxLiquidityPerTick() view returns(uint128)
func (_PharaohV3 *PharaohV3CallerSession) MaxLiquidityPerTick() (*big.Int, error) {
	return _PharaohV3.Contract.MaxLiquidityPerTick(&_PharaohV3.CallOpts)
}

// Observations is a free data retrieval call binding the contract method 0x252c09d7.
//
// Solidity: function observations(uint256 index) view returns(uint32 blockTimestamp, int56 tickCumulative, uint160 secondsPerLiquidityCumulativeX128, bool initialized)
func (_PharaohV3 *PharaohV3Caller) Observations(opts *bind.CallOpts, index *big.Int) (struct {
	BlockTimestamp                    uint32
	TickCumulative                    *big.Int
	SecondsPerLiquidityCumulativeX128 *big.Int
	Initialized                       bool
}, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "observations", index)

	outstruct := new(struct {
		BlockTimestamp                    uint32
		TickCumulative                    *big.Int
		SecondsPerLiquidityCumulativeX128 *big.Int
		Initialized                       bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.BlockTimestamp = *abi.ConvertType(out[0], new(uint32)).(*uint32)
	outstruct.TickCumulative = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.SecondsPerLiquidityCumulativeX128 = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.Initialized = *abi.ConvertType(out[3], new(bool)).(*bool)

	return *outstruct, err

}

// Observations is a free data retrieval call binding the contract method 0x252c09d7.
//
// Solidity: function observations(uint256 index) view returns(uint32 blockTimestamp, int56 tickCumulative, uint160 secondsPerLiquidityCumulativeX128, bool initialized)
func (_PharaohV3 *PharaohV3Session) Observations(index *big.Int) (struct {
	BlockTimestamp                    uint32
	TickCumulative                    *big.Int
	SecondsPerLiquidityCumulativeX128 *big.Int
	Initialized                       bool
}, error) {
	return _PharaohV3.Contract.Observations(&_PharaohV3.CallOpts, index)
}

// Observations is a free data retrieval call binding the contract method 0x252c09d7.
//
// Solidity: function observations(uint256 index) view returns(uint32 blockTimestamp, int56 tickCumulative, uint160 secondsPerLiquidityCumulativeX128, bool initialized)
func (_PharaohV3 *PharaohV3CallerSession) Observations(index *big.Int) (struct {
	BlockTimestamp                    uint32
	TickCumulative                    *big.Int
	SecondsPerLiquidityCumulativeX128 *big.Int
	Initialized                       bool
}, error) {
	return _PharaohV3.Contract.Observations(&_PharaohV3.CallOpts, index)
}

// Observe is a free data retrieval call binding the contract method 0x883bdbfd.
//
// Solidity: function observe(uint32[] secondsAgos) view returns(int56[] tickCumulatives, uint160[] secondsPerLiquidityCumulativeX128s)
func (_PharaohV3 *PharaohV3Caller) Observe(opts *bind.CallOpts, secondsAgos []uint32) (struct {
	TickCumulatives                    []*big.Int
	SecondsPerLiquidityCumulativeX128s []*big.Int
}, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "observe", secondsAgos)

	outstruct := new(struct {
		TickCumulatives                    []*big.Int
		SecondsPerLiquidityCumulativeX128s []*big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.TickCumulatives = *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)
	outstruct.SecondsPerLiquidityCumulativeX128s = *abi.ConvertType(out[1], new([]*big.Int)).(*[]*big.Int)

	return *outstruct, err

}

// Observe is a free data retrieval call binding the contract method 0x883bdbfd.
//
// Solidity: function observe(uint32[] secondsAgos) view returns(int56[] tickCumulatives, uint160[] secondsPerLiquidityCumulativeX128s)
func (_PharaohV3 *PharaohV3Session) Observe(secondsAgos []uint32) (struct {
	TickCumulatives                    []*big.Int
	SecondsPerLiquidityCumulativeX128s []*big.Int
}, error) {
	return _PharaohV3.Contract.Observe(&_PharaohV3.CallOpts, secondsAgos)
}

// Observe is a free data retrieval call binding the contract method 0x883bdbfd.
//
// Solidity: function observe(uint32[] secondsAgos) view returns(int56[] tickCumulatives, uint160[] secondsPerLiquidityCumulativeX128s)
func (_PharaohV3 *PharaohV3CallerSession) Observe(secondsAgos []uint32) (struct {
	TickCumulatives                    []*big.Int
	SecondsPerLiquidityCumulativeX128s []*big.Int
}, error) {
	return _PharaohV3.Contract.Observe(&_PharaohV3.CallOpts, secondsAgos)
}

// Periods is a free data retrieval call binding the contract method 0xea4a1104.
//
// Solidity: function periods(uint256 period) view returns(uint32 previousPeriod, int24 startTick, int24 lastTick, uint160 endSecondsPerLiquidityPeriodX128)
func (_PharaohV3 *PharaohV3Caller) Periods(opts *bind.CallOpts, period *big.Int) (struct {
	PreviousPeriod                   uint32
	StartTick                        *big.Int
	LastTick                         *big.Int
	EndSecondsPerLiquidityPeriodX128 *big.Int
}, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "periods", period)

	outstruct := new(struct {
		PreviousPeriod                   uint32
		StartTick                        *big.Int
		LastTick                         *big.Int
		EndSecondsPerLiquidityPeriodX128 *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.PreviousPeriod = *abi.ConvertType(out[0], new(uint32)).(*uint32)
	outstruct.StartTick = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.LastTick = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.EndSecondsPerLiquidityPeriodX128 = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Periods is a free data retrieval call binding the contract method 0xea4a1104.
//
// Solidity: function periods(uint256 period) view returns(uint32 previousPeriod, int24 startTick, int24 lastTick, uint160 endSecondsPerLiquidityPeriodX128)
func (_PharaohV3 *PharaohV3Session) Periods(period *big.Int) (struct {
	PreviousPeriod                   uint32
	StartTick                        *big.Int
	LastTick                         *big.Int
	EndSecondsPerLiquidityPeriodX128 *big.Int
}, error) {
	return _PharaohV3.Contract.Periods(&_PharaohV3.CallOpts, period)
}

// Periods is a free data retrieval call binding the contract method 0xea4a1104.
//
// Solidity: function periods(uint256 period) view returns(uint32 previousPeriod, int24 startTick, int24 lastTick, uint160 endSecondsPerLiquidityPeriodX128)
func (_PharaohV3 *PharaohV3CallerSession) Periods(period *big.Int) (struct {
	PreviousPeriod                   uint32
	StartTick                        *big.Int
	LastTick                         *big.Int
	EndSecondsPerLiquidityPeriodX128 *big.Int
}, error) {
	return _PharaohV3.Contract.Periods(&_PharaohV3.CallOpts, period)
}

// PositionPeriodSecondsInRange is a free data retrieval call binding the contract method 0x9918fbb6.
//
// Solidity: function positionPeriodSecondsInRange(uint256 period, address owner, uint256 index, int24 tickLower, int24 tickUpper) view returns(uint256 periodSecondsInsideX96)
func (_PharaohV3 *PharaohV3Caller) PositionPeriodSecondsInRange(opts *bind.CallOpts, period *big.Int, owner common.Address, index *big.Int, tickLower *big.Int, tickUpper *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "positionPeriodSecondsInRange", period, owner, index, tickLower, tickUpper)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PositionPeriodSecondsInRange is a free data retrieval call binding the contract method 0x9918fbb6.
//
// Solidity: function positionPeriodSecondsInRange(uint256 period, address owner, uint256 index, int24 tickLower, int24 tickUpper) view returns(uint256 periodSecondsInsideX96)
func (_PharaohV3 *PharaohV3Session) PositionPeriodSecondsInRange(period *big.Int, owner common.Address, index *big.Int, tickLower *big.Int, tickUpper *big.Int) (*big.Int, error) {
	return _PharaohV3.Contract.PositionPeriodSecondsInRange(&_PharaohV3.CallOpts, period, owner, index, tickLower, tickUpper)
}

// PositionPeriodSecondsInRange is a free data retrieval call binding the contract method 0x9918fbb6.
//
// Solidity: function positionPeriodSecondsInRange(uint256 period, address owner, uint256 index, int24 tickLower, int24 tickUpper) view returns(uint256 periodSecondsInsideX96)
func (_PharaohV3 *PharaohV3CallerSession) PositionPeriodSecondsInRange(period *big.Int, owner common.Address, index *big.Int, tickLower *big.Int, tickUpper *big.Int) (*big.Int, error) {
	return _PharaohV3.Contract.PositionPeriodSecondsInRange(&_PharaohV3.CallOpts, period, owner, index, tickLower, tickUpper)
}

// Positions is a free data retrieval call binding the contract method 0x514ea4bf.
//
// Solidity: function positions(bytes32 key) view returns(uint128 liquidity, uint256 feeGrowthInside0LastX128, uint256 feeGrowthInside1LastX128, uint128 tokensOwed0, uint128 tokensOwed1)
func (_PharaohV3 *PharaohV3Caller) Positions(opts *bind.CallOpts, key [32]byte) (struct {
	Liquidity                *big.Int
	FeeGrowthInside0LastX128 *big.Int
	FeeGrowthInside1LastX128 *big.Int
	TokensOwed0              *big.Int
	TokensOwed1              *big.Int
}, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "positions", key)

	outstruct := new(struct {
		Liquidity                *big.Int
		FeeGrowthInside0LastX128 *big.Int
		FeeGrowthInside1LastX128 *big.Int
		TokensOwed0              *big.Int
		TokensOwed1              *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Liquidity = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.FeeGrowthInside0LastX128 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.FeeGrowthInside1LastX128 = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.TokensOwed0 = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.TokensOwed1 = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Positions is a free data retrieval call binding the contract method 0x514ea4bf.
//
// Solidity: function positions(bytes32 key) view returns(uint128 liquidity, uint256 feeGrowthInside0LastX128, uint256 feeGrowthInside1LastX128, uint128 tokensOwed0, uint128 tokensOwed1)
func (_PharaohV3 *PharaohV3Session) Positions(key [32]byte) (struct {
	Liquidity                *big.Int
	FeeGrowthInside0LastX128 *big.Int
	FeeGrowthInside1LastX128 *big.Int
	TokensOwed0              *big.Int
	TokensOwed1              *big.Int
}, error) {
	return _PharaohV3.Contract.Positions(&_PharaohV3.CallOpts, key)
}

// Positions is a free data retrieval call binding the contract method 0x514ea4bf.
//
// Solidity: function positions(bytes32 key) view returns(uint128 liquidity, uint256 feeGrowthInside0LastX128, uint256 feeGrowthInside1LastX128, uint128 tokensOwed0, uint128 tokensOwed1)
func (_PharaohV3 *PharaohV3CallerSession) Positions(key [32]byte) (struct {
	Liquidity                *big.Int
	FeeGrowthInside0LastX128 *big.Int
	FeeGrowthInside1LastX128 *big.Int
	TokensOwed0              *big.Int
	TokensOwed1              *big.Int
}, error) {
	return _PharaohV3.Contract.Positions(&_PharaohV3.CallOpts, key)
}

// ProtocolFees is a free data retrieval call binding the contract method 0x1ad8b03b.
//
// Solidity: function protocolFees() view returns(uint128, uint128)
func (_PharaohV3 *PharaohV3Caller) ProtocolFees(opts *bind.CallOpts) (*big.Int, *big.Int, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "protocolFees")

	if err != nil {
		return *new(*big.Int), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return out0, out1, err

}

// ProtocolFees is a free data retrieval call binding the contract method 0x1ad8b03b.
//
// Solidity: function protocolFees() view returns(uint128, uint128)
func (_PharaohV3 *PharaohV3Session) ProtocolFees() (*big.Int, *big.Int, error) {
	return _PharaohV3.Contract.ProtocolFees(&_PharaohV3.CallOpts)
}

// ProtocolFees is a free data retrieval call binding the contract method 0x1ad8b03b.
//
// Solidity: function protocolFees() view returns(uint128, uint128)
func (_PharaohV3 *PharaohV3CallerSession) ProtocolFees() (*big.Int, *big.Int, error) {
	return _PharaohV3.Contract.ProtocolFees(&_PharaohV3.CallOpts)
}

// ReadStorage is a free data retrieval call binding the contract method 0xe57c0ca9.
//
// Solidity: function readStorage(bytes32[] slots) view returns(bytes32[] returnData)
func (_PharaohV3 *PharaohV3Caller) ReadStorage(opts *bind.CallOpts, slots [][32]byte) ([][32]byte, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "readStorage", slots)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// ReadStorage is a free data retrieval call binding the contract method 0xe57c0ca9.
//
// Solidity: function readStorage(bytes32[] slots) view returns(bytes32[] returnData)
func (_PharaohV3 *PharaohV3Session) ReadStorage(slots [][32]byte) ([][32]byte, error) {
	return _PharaohV3.Contract.ReadStorage(&_PharaohV3.CallOpts, slots)
}

// ReadStorage is a free data retrieval call binding the contract method 0xe57c0ca9.
//
// Solidity: function readStorage(bytes32[] slots) view returns(bytes32[] returnData)
func (_PharaohV3 *PharaohV3CallerSession) ReadStorage(slots [][32]byte) ([][32]byte, error) {
	return _PharaohV3.Contract.ReadStorage(&_PharaohV3.CallOpts, slots)
}

// Slot0 is a free data retrieval call binding the contract method 0x3850c7bd.
//
// Solidity: function slot0() view returns(uint160 sqrtPriceX96, int24 tick, uint16 observationIndex, uint16 observationCardinality, uint16 observationCardinalityNext, uint24 feeProtocol, bool unlocked)
func (_PharaohV3 *PharaohV3Caller) Slot0(opts *bind.CallOpts) (struct {
	SqrtPriceX96               *big.Int
	Tick                       *big.Int
	ObservationIndex           uint16
	ObservationCardinality     uint16
	ObservationCardinalityNext uint16
	FeeProtocol                *big.Int
	Unlocked                   bool
}, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "slot0")

	outstruct := new(struct {
		SqrtPriceX96               *big.Int
		Tick                       *big.Int
		ObservationIndex           uint16
		ObservationCardinality     uint16
		ObservationCardinalityNext uint16
		FeeProtocol                *big.Int
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
	outstruct.FeeProtocol = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)
	outstruct.Unlocked = *abi.ConvertType(out[6], new(bool)).(*bool)

	return *outstruct, err

}

// Slot0 is a free data retrieval call binding the contract method 0x3850c7bd.
//
// Solidity: function slot0() view returns(uint160 sqrtPriceX96, int24 tick, uint16 observationIndex, uint16 observationCardinality, uint16 observationCardinalityNext, uint24 feeProtocol, bool unlocked)
func (_PharaohV3 *PharaohV3Session) Slot0() (struct {
	SqrtPriceX96               *big.Int
	Tick                       *big.Int
	ObservationIndex           uint16
	ObservationCardinality     uint16
	ObservationCardinalityNext uint16
	FeeProtocol                *big.Int
	Unlocked                   bool
}, error) {
	return _PharaohV3.Contract.Slot0(&_PharaohV3.CallOpts)
}

// Slot0 is a free data retrieval call binding the contract method 0x3850c7bd.
//
// Solidity: function slot0() view returns(uint160 sqrtPriceX96, int24 tick, uint16 observationIndex, uint16 observationCardinality, uint16 observationCardinalityNext, uint24 feeProtocol, bool unlocked)
func (_PharaohV3 *PharaohV3CallerSession) Slot0() (struct {
	SqrtPriceX96               *big.Int
	Tick                       *big.Int
	ObservationIndex           uint16
	ObservationCardinality     uint16
	ObservationCardinalityNext uint16
	FeeProtocol                *big.Int
	Unlocked                   bool
}, error) {
	return _PharaohV3.Contract.Slot0(&_PharaohV3.CallOpts)
}

// SnapshotCumulativesInside is a free data retrieval call binding the contract method 0xa38807f2.
//
// Solidity: function snapshotCumulativesInside(int24 tickLower, int24 tickUpper) view returns(int56 tickCumulativeInside, uint160 secondsPerLiquidityInsideX128, uint32 secondsInside)
func (_PharaohV3 *PharaohV3Caller) SnapshotCumulativesInside(opts *bind.CallOpts, tickLower *big.Int, tickUpper *big.Int) (struct {
	TickCumulativeInside          *big.Int
	SecondsPerLiquidityInsideX128 *big.Int
	SecondsInside                 uint32
}, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "snapshotCumulativesInside", tickLower, tickUpper)

	outstruct := new(struct {
		TickCumulativeInside          *big.Int
		SecondsPerLiquidityInsideX128 *big.Int
		SecondsInside                 uint32
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.TickCumulativeInside = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.SecondsPerLiquidityInsideX128 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.SecondsInside = *abi.ConvertType(out[2], new(uint32)).(*uint32)

	return *outstruct, err

}

// SnapshotCumulativesInside is a free data retrieval call binding the contract method 0xa38807f2.
//
// Solidity: function snapshotCumulativesInside(int24 tickLower, int24 tickUpper) view returns(int56 tickCumulativeInside, uint160 secondsPerLiquidityInsideX128, uint32 secondsInside)
func (_PharaohV3 *PharaohV3Session) SnapshotCumulativesInside(tickLower *big.Int, tickUpper *big.Int) (struct {
	TickCumulativeInside          *big.Int
	SecondsPerLiquidityInsideX128 *big.Int
	SecondsInside                 uint32
}, error) {
	return _PharaohV3.Contract.SnapshotCumulativesInside(&_PharaohV3.CallOpts, tickLower, tickUpper)
}

// SnapshotCumulativesInside is a free data retrieval call binding the contract method 0xa38807f2.
//
// Solidity: function snapshotCumulativesInside(int24 tickLower, int24 tickUpper) view returns(int56 tickCumulativeInside, uint160 secondsPerLiquidityInsideX128, uint32 secondsInside)
func (_PharaohV3 *PharaohV3CallerSession) SnapshotCumulativesInside(tickLower *big.Int, tickUpper *big.Int) (struct {
	TickCumulativeInside          *big.Int
	SecondsPerLiquidityInsideX128 *big.Int
	SecondsInside                 uint32
}, error) {
	return _PharaohV3.Contract.SnapshotCumulativesInside(&_PharaohV3.CallOpts, tickLower, tickUpper)
}

// TickBitmap is a free data retrieval call binding the contract method 0x5339c296.
//
// Solidity: function tickBitmap(int16 tick) view returns(uint256)
func (_PharaohV3 *PharaohV3Caller) TickBitmap(opts *bind.CallOpts, tick int16) (*big.Int, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "tickBitmap", tick)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TickBitmap is a free data retrieval call binding the contract method 0x5339c296.
//
// Solidity: function tickBitmap(int16 tick) view returns(uint256)
func (_PharaohV3 *PharaohV3Session) TickBitmap(tick int16) (*big.Int, error) {
	return _PharaohV3.Contract.TickBitmap(&_PharaohV3.CallOpts, tick)
}

// TickBitmap is a free data retrieval call binding the contract method 0x5339c296.
//
// Solidity: function tickBitmap(int16 tick) view returns(uint256)
func (_PharaohV3 *PharaohV3CallerSession) TickBitmap(tick int16) (*big.Int, error) {
	return _PharaohV3.Contract.TickBitmap(&_PharaohV3.CallOpts, tick)
}

// TickSpacing is a free data retrieval call binding the contract method 0xd0c93a7c.
//
// Solidity: function tickSpacing() view returns(int24)
func (_PharaohV3 *PharaohV3Caller) TickSpacing(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "tickSpacing")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TickSpacing is a free data retrieval call binding the contract method 0xd0c93a7c.
//
// Solidity: function tickSpacing() view returns(int24)
func (_PharaohV3 *PharaohV3Session) TickSpacing() (*big.Int, error) {
	return _PharaohV3.Contract.TickSpacing(&_PharaohV3.CallOpts)
}

// TickSpacing is a free data retrieval call binding the contract method 0xd0c93a7c.
//
// Solidity: function tickSpacing() view returns(int24)
func (_PharaohV3 *PharaohV3CallerSession) TickSpacing() (*big.Int, error) {
	return _PharaohV3.Contract.TickSpacing(&_PharaohV3.CallOpts)
}

// Ticks is a free data retrieval call binding the contract method 0xf30dba93.
//
// Solidity: function ticks(int24 tick) view returns(uint128 liquidityGross, int128 liquidityNet, uint256 feeGrowthOutside0X128, uint256 feeGrowthOutside1X128, int56 tickCumulativeOutside, uint160 secondsPerLiquidityOutsideX128, uint32 secondsOutside, bool initialized)
func (_PharaohV3 *PharaohV3Caller) Ticks(opts *bind.CallOpts, tick *big.Int) (struct {
	LiquidityGross                 *big.Int
	LiquidityNet                   *big.Int
	FeeGrowthOutside0X128          *big.Int
	FeeGrowthOutside1X128          *big.Int
	TickCumulativeOutside          *big.Int
	SecondsPerLiquidityOutsideX128 *big.Int
	SecondsOutside                 uint32
	Initialized                    bool
}, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "ticks", tick)

	outstruct := new(struct {
		LiquidityGross                 *big.Int
		LiquidityNet                   *big.Int
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
	outstruct.FeeGrowthOutside0X128 = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.FeeGrowthOutside1X128 = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.TickCumulativeOutside = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.SecondsPerLiquidityOutsideX128 = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)
	outstruct.SecondsOutside = *abi.ConvertType(out[6], new(uint32)).(*uint32)
	outstruct.Initialized = *abi.ConvertType(out[7], new(bool)).(*bool)

	return *outstruct, err

}

// Ticks is a free data retrieval call binding the contract method 0xf30dba93.
//
// Solidity: function ticks(int24 tick) view returns(uint128 liquidityGross, int128 liquidityNet, uint256 feeGrowthOutside0X128, uint256 feeGrowthOutside1X128, int56 tickCumulativeOutside, uint160 secondsPerLiquidityOutsideX128, uint32 secondsOutside, bool initialized)
func (_PharaohV3 *PharaohV3Session) Ticks(tick *big.Int) (struct {
	LiquidityGross                 *big.Int
	LiquidityNet                   *big.Int
	FeeGrowthOutside0X128          *big.Int
	FeeGrowthOutside1X128          *big.Int
	TickCumulativeOutside          *big.Int
	SecondsPerLiquidityOutsideX128 *big.Int
	SecondsOutside                 uint32
	Initialized                    bool
}, error) {
	return _PharaohV3.Contract.Ticks(&_PharaohV3.CallOpts, tick)
}

// Ticks is a free data retrieval call binding the contract method 0xf30dba93.
//
// Solidity: function ticks(int24 tick) view returns(uint128 liquidityGross, int128 liquidityNet, uint256 feeGrowthOutside0X128, uint256 feeGrowthOutside1X128, int56 tickCumulativeOutside, uint160 secondsPerLiquidityOutsideX128, uint32 secondsOutside, bool initialized)
func (_PharaohV3 *PharaohV3CallerSession) Ticks(tick *big.Int) (struct {
	LiquidityGross                 *big.Int
	LiquidityNet                   *big.Int
	FeeGrowthOutside0X128          *big.Int
	FeeGrowthOutside1X128          *big.Int
	TickCumulativeOutside          *big.Int
	SecondsPerLiquidityOutsideX128 *big.Int
	SecondsOutside                 uint32
	Initialized                    bool
}, error) {
	return _PharaohV3.Contract.Ticks(&_PharaohV3.CallOpts, tick)
}

// Token0 is a free data retrieval call binding the contract method 0x0dfe1681.
//
// Solidity: function token0() view returns(address)
func (_PharaohV3 *PharaohV3Caller) Token0(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "token0")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Token0 is a free data retrieval call binding the contract method 0x0dfe1681.
//
// Solidity: function token0() view returns(address)
func (_PharaohV3 *PharaohV3Session) Token0() (common.Address, error) {
	return _PharaohV3.Contract.Token0(&_PharaohV3.CallOpts)
}

// Token0 is a free data retrieval call binding the contract method 0x0dfe1681.
//
// Solidity: function token0() view returns(address)
func (_PharaohV3 *PharaohV3CallerSession) Token0() (common.Address, error) {
	return _PharaohV3.Contract.Token0(&_PharaohV3.CallOpts)
}

// Token1 is a free data retrieval call binding the contract method 0xd21220a7.
//
// Solidity: function token1() view returns(address)
func (_PharaohV3 *PharaohV3Caller) Token1(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PharaohV3.contract.Call(opts, &out, "token1")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Token1 is a free data retrieval call binding the contract method 0xd21220a7.
//
// Solidity: function token1() view returns(address)
func (_PharaohV3 *PharaohV3Session) Token1() (common.Address, error) {
	return _PharaohV3.Contract.Token1(&_PharaohV3.CallOpts)
}

// Token1 is a free data retrieval call binding the contract method 0xd21220a7.
//
// Solidity: function token1() view returns(address)
func (_PharaohV3 *PharaohV3CallerSession) Token1() (common.Address, error) {
	return _PharaohV3.Contract.Token1(&_PharaohV3.CallOpts)
}

// AdvancePeriod is a paid mutator transaction binding the contract method 0xc2e0f9b2.
//
// Solidity: function _advancePeriod() returns()
func (_PharaohV3 *PharaohV3Transactor) AdvancePeriod(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PharaohV3.contract.Transact(opts, "_advancePeriod")
}

// AdvancePeriod is a paid mutator transaction binding the contract method 0xc2e0f9b2.
//
// Solidity: function _advancePeriod() returns()
func (_PharaohV3 *PharaohV3Session) AdvancePeriod() (*types.Transaction, error) {
	return _PharaohV3.Contract.AdvancePeriod(&_PharaohV3.TransactOpts)
}

// AdvancePeriod is a paid mutator transaction binding the contract method 0xc2e0f9b2.
//
// Solidity: function _advancePeriod() returns()
func (_PharaohV3 *PharaohV3TransactorSession) AdvancePeriod() (*types.Transaction, error) {
	return _PharaohV3.Contract.AdvancePeriod(&_PharaohV3.TransactOpts)
}

// Burn is a paid mutator transaction binding the contract method 0x6847456a.
//
// Solidity: function burn(uint256 index, int24 tickLower, int24 tickUpper, uint128 amount) returns(uint256 amount0, uint256 amount1)
func (_PharaohV3 *PharaohV3Transactor) Burn(opts *bind.TransactOpts, index *big.Int, tickLower *big.Int, tickUpper *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _PharaohV3.contract.Transact(opts, "burn", index, tickLower, tickUpper, amount)
}

// Burn is a paid mutator transaction binding the contract method 0x6847456a.
//
// Solidity: function burn(uint256 index, int24 tickLower, int24 tickUpper, uint128 amount) returns(uint256 amount0, uint256 amount1)
func (_PharaohV3 *PharaohV3Session) Burn(index *big.Int, tickLower *big.Int, tickUpper *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _PharaohV3.Contract.Burn(&_PharaohV3.TransactOpts, index, tickLower, tickUpper, amount)
}

// Burn is a paid mutator transaction binding the contract method 0x6847456a.
//
// Solidity: function burn(uint256 index, int24 tickLower, int24 tickUpper, uint128 amount) returns(uint256 amount0, uint256 amount1)
func (_PharaohV3 *PharaohV3TransactorSession) Burn(index *big.Int, tickLower *big.Int, tickUpper *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _PharaohV3.Contract.Burn(&_PharaohV3.TransactOpts, index, tickLower, tickUpper, amount)
}

// Collect is a paid mutator transaction binding the contract method 0xa02f1069.
//
// Solidity: function collect(address recipient, uint256 index, int24 tickLower, int24 tickUpper, uint128 amount0Requested, uint128 amount1Requested) returns(uint128 amount0, uint128 amount1)
func (_PharaohV3 *PharaohV3Transactor) Collect(opts *bind.TransactOpts, recipient common.Address, index *big.Int, tickLower *big.Int, tickUpper *big.Int, amount0Requested *big.Int, amount1Requested *big.Int) (*types.Transaction, error) {
	return _PharaohV3.contract.Transact(opts, "collect", recipient, index, tickLower, tickUpper, amount0Requested, amount1Requested)
}

// Collect is a paid mutator transaction binding the contract method 0xa02f1069.
//
// Solidity: function collect(address recipient, uint256 index, int24 tickLower, int24 tickUpper, uint128 amount0Requested, uint128 amount1Requested) returns(uint128 amount0, uint128 amount1)
func (_PharaohV3 *PharaohV3Session) Collect(recipient common.Address, index *big.Int, tickLower *big.Int, tickUpper *big.Int, amount0Requested *big.Int, amount1Requested *big.Int) (*types.Transaction, error) {
	return _PharaohV3.Contract.Collect(&_PharaohV3.TransactOpts, recipient, index, tickLower, tickUpper, amount0Requested, amount1Requested)
}

// Collect is a paid mutator transaction binding the contract method 0xa02f1069.
//
// Solidity: function collect(address recipient, uint256 index, int24 tickLower, int24 tickUpper, uint128 amount0Requested, uint128 amount1Requested) returns(uint128 amount0, uint128 amount1)
func (_PharaohV3 *PharaohV3TransactorSession) Collect(recipient common.Address, index *big.Int, tickLower *big.Int, tickUpper *big.Int, amount0Requested *big.Int, amount1Requested *big.Int) (*types.Transaction, error) {
	return _PharaohV3.Contract.Collect(&_PharaohV3.TransactOpts, recipient, index, tickLower, tickUpper, amount0Requested, amount1Requested)
}

// CollectProtocol is a paid mutator transaction binding the contract method 0x85b66729.
//
// Solidity: function collectProtocol(address recipient, uint128 amount0Requested, uint128 amount1Requested) returns(uint128 amount0, uint128 amount1)
func (_PharaohV3 *PharaohV3Transactor) CollectProtocol(opts *bind.TransactOpts, recipient common.Address, amount0Requested *big.Int, amount1Requested *big.Int) (*types.Transaction, error) {
	return _PharaohV3.contract.Transact(opts, "collectProtocol", recipient, amount0Requested, amount1Requested)
}

// CollectProtocol is a paid mutator transaction binding the contract method 0x85b66729.
//
// Solidity: function collectProtocol(address recipient, uint128 amount0Requested, uint128 amount1Requested) returns(uint128 amount0, uint128 amount1)
func (_PharaohV3 *PharaohV3Session) CollectProtocol(recipient common.Address, amount0Requested *big.Int, amount1Requested *big.Int) (*types.Transaction, error) {
	return _PharaohV3.Contract.CollectProtocol(&_PharaohV3.TransactOpts, recipient, amount0Requested, amount1Requested)
}

// CollectProtocol is a paid mutator transaction binding the contract method 0x85b66729.
//
// Solidity: function collectProtocol(address recipient, uint128 amount0Requested, uint128 amount1Requested) returns(uint128 amount0, uint128 amount1)
func (_PharaohV3 *PharaohV3TransactorSession) CollectProtocol(recipient common.Address, amount0Requested *big.Int, amount1Requested *big.Int) (*types.Transaction, error) {
	return _PharaohV3.Contract.CollectProtocol(&_PharaohV3.TransactOpts, recipient, amount0Requested, amount1Requested)
}

// Flash is a paid mutator transaction binding the contract method 0x490e6cbc.
//
// Solidity: function flash(address recipient, uint256 amount0, uint256 amount1, bytes data) returns()
func (_PharaohV3 *PharaohV3Transactor) Flash(opts *bind.TransactOpts, recipient common.Address, amount0 *big.Int, amount1 *big.Int, data []byte) (*types.Transaction, error) {
	return _PharaohV3.contract.Transact(opts, "flash", recipient, amount0, amount1, data)
}

// Flash is a paid mutator transaction binding the contract method 0x490e6cbc.
//
// Solidity: function flash(address recipient, uint256 amount0, uint256 amount1, bytes data) returns()
func (_PharaohV3 *PharaohV3Session) Flash(recipient common.Address, amount0 *big.Int, amount1 *big.Int, data []byte) (*types.Transaction, error) {
	return _PharaohV3.Contract.Flash(&_PharaohV3.TransactOpts, recipient, amount0, amount1, data)
}

// Flash is a paid mutator transaction binding the contract method 0x490e6cbc.
//
// Solidity: function flash(address recipient, uint256 amount0, uint256 amount1, bytes data) returns()
func (_PharaohV3 *PharaohV3TransactorSession) Flash(recipient common.Address, amount0 *big.Int, amount1 *big.Int, data []byte) (*types.Transaction, error) {
	return _PharaohV3.Contract.Flash(&_PharaohV3.TransactOpts, recipient, amount0, amount1, data)
}

// IncreaseObservationCardinalityNext is a paid mutator transaction binding the contract method 0x32148f67.
//
// Solidity: function increaseObservationCardinalityNext(uint16 observationCardinalityNext) returns()
func (_PharaohV3 *PharaohV3Transactor) IncreaseObservationCardinalityNext(opts *bind.TransactOpts, observationCardinalityNext uint16) (*types.Transaction, error) {
	return _PharaohV3.contract.Transact(opts, "increaseObservationCardinalityNext", observationCardinalityNext)
}

// IncreaseObservationCardinalityNext is a paid mutator transaction binding the contract method 0x32148f67.
//
// Solidity: function increaseObservationCardinalityNext(uint16 observationCardinalityNext) returns()
func (_PharaohV3 *PharaohV3Session) IncreaseObservationCardinalityNext(observationCardinalityNext uint16) (*types.Transaction, error) {
	return _PharaohV3.Contract.IncreaseObservationCardinalityNext(&_PharaohV3.TransactOpts, observationCardinalityNext)
}

// IncreaseObservationCardinalityNext is a paid mutator transaction binding the contract method 0x32148f67.
//
// Solidity: function increaseObservationCardinalityNext(uint16 observationCardinalityNext) returns()
func (_PharaohV3 *PharaohV3TransactorSession) IncreaseObservationCardinalityNext(observationCardinalityNext uint16) (*types.Transaction, error) {
	return _PharaohV3.Contract.IncreaseObservationCardinalityNext(&_PharaohV3.TransactOpts, observationCardinalityNext)
}

// Initialize is a paid mutator transaction binding the contract method 0xf637731d.
//
// Solidity: function initialize(uint160 sqrtPriceX96) returns()
func (_PharaohV3 *PharaohV3Transactor) Initialize(opts *bind.TransactOpts, sqrtPriceX96 *big.Int) (*types.Transaction, error) {
	return _PharaohV3.contract.Transact(opts, "initialize", sqrtPriceX96)
}

// Initialize is a paid mutator transaction binding the contract method 0xf637731d.
//
// Solidity: function initialize(uint160 sqrtPriceX96) returns()
func (_PharaohV3 *PharaohV3Session) Initialize(sqrtPriceX96 *big.Int) (*types.Transaction, error) {
	return _PharaohV3.Contract.Initialize(&_PharaohV3.TransactOpts, sqrtPriceX96)
}

// Initialize is a paid mutator transaction binding the contract method 0xf637731d.
//
// Solidity: function initialize(uint160 sqrtPriceX96) returns()
func (_PharaohV3 *PharaohV3TransactorSession) Initialize(sqrtPriceX96 *big.Int) (*types.Transaction, error) {
	return _PharaohV3.Contract.Initialize(&_PharaohV3.TransactOpts, sqrtPriceX96)
}

// Mint is a paid mutator transaction binding the contract method 0x8221b8c1.
//
// Solidity: function mint(address recipient, uint256 index, int24 tickLower, int24 tickUpper, uint128 amount, bytes data) returns(uint256 amount0, uint256 amount1)
func (_PharaohV3 *PharaohV3Transactor) Mint(opts *bind.TransactOpts, recipient common.Address, index *big.Int, tickLower *big.Int, tickUpper *big.Int, amount *big.Int, data []byte) (*types.Transaction, error) {
	return _PharaohV3.contract.Transact(opts, "mint", recipient, index, tickLower, tickUpper, amount, data)
}

// Mint is a paid mutator transaction binding the contract method 0x8221b8c1.
//
// Solidity: function mint(address recipient, uint256 index, int24 tickLower, int24 tickUpper, uint128 amount, bytes data) returns(uint256 amount0, uint256 amount1)
func (_PharaohV3 *PharaohV3Session) Mint(recipient common.Address, index *big.Int, tickLower *big.Int, tickUpper *big.Int, amount *big.Int, data []byte) (*types.Transaction, error) {
	return _PharaohV3.Contract.Mint(&_PharaohV3.TransactOpts, recipient, index, tickLower, tickUpper, amount, data)
}

// Mint is a paid mutator transaction binding the contract method 0x8221b8c1.
//
// Solidity: function mint(address recipient, uint256 index, int24 tickLower, int24 tickUpper, uint128 amount, bytes data) returns(uint256 amount0, uint256 amount1)
func (_PharaohV3 *PharaohV3TransactorSession) Mint(recipient common.Address, index *big.Int, tickLower *big.Int, tickUpper *big.Int, amount *big.Int, data []byte) (*types.Transaction, error) {
	return _PharaohV3.Contract.Mint(&_PharaohV3.TransactOpts, recipient, index, tickLower, tickUpper, amount, data)
}

// SetFee is a paid mutator transaction binding the contract method 0xeabb5622.
//
// Solidity: function setFee(uint24 _fee) returns()
func (_PharaohV3 *PharaohV3Transactor) SetFee(opts *bind.TransactOpts, _fee *big.Int) (*types.Transaction, error) {
	return _PharaohV3.contract.Transact(opts, "setFee", _fee)
}

// SetFee is a paid mutator transaction binding the contract method 0xeabb5622.
//
// Solidity: function setFee(uint24 _fee) returns()
func (_PharaohV3 *PharaohV3Session) SetFee(_fee *big.Int) (*types.Transaction, error) {
	return _PharaohV3.Contract.SetFee(&_PharaohV3.TransactOpts, _fee)
}

// SetFee is a paid mutator transaction binding the contract method 0xeabb5622.
//
// Solidity: function setFee(uint24 _fee) returns()
func (_PharaohV3 *PharaohV3TransactorSession) SetFee(_fee *big.Int) (*types.Transaction, error) {
	return _PharaohV3.Contract.SetFee(&_PharaohV3.TransactOpts, _fee)
}

// SetFeeProtocol is a paid mutator transaction binding the contract method 0x7b7d549d.
//
// Solidity: function setFeeProtocol() returns()
func (_PharaohV3 *PharaohV3Transactor) SetFeeProtocol(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PharaohV3.contract.Transact(opts, "setFeeProtocol")
}

// SetFeeProtocol is a paid mutator transaction binding the contract method 0x7b7d549d.
//
// Solidity: function setFeeProtocol() returns()
func (_PharaohV3 *PharaohV3Session) SetFeeProtocol() (*types.Transaction, error) {
	return _PharaohV3.Contract.SetFeeProtocol(&_PharaohV3.TransactOpts)
}

// SetFeeProtocol is a paid mutator transaction binding the contract method 0x7b7d549d.
//
// Solidity: function setFeeProtocol() returns()
func (_PharaohV3 *PharaohV3TransactorSession) SetFeeProtocol() (*types.Transaction, error) {
	return _PharaohV3.Contract.SetFeeProtocol(&_PharaohV3.TransactOpts)
}

// Swap is a paid mutator transaction binding the contract method 0x128acb08.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, bytes data) returns(int256 amount0, int256 amount1)
func (_PharaohV3 *PharaohV3Transactor) Swap(opts *bind.TransactOpts, recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, data []byte) (*types.Transaction, error) {
	return _PharaohV3.contract.Transact(opts, "swap", recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, data)
}

// Swap is a paid mutator transaction binding the contract method 0x128acb08.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, bytes data) returns(int256 amount0, int256 amount1)
func (_PharaohV3 *PharaohV3Session) Swap(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, data []byte) (*types.Transaction, error) {
	return _PharaohV3.Contract.Swap(&_PharaohV3.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, data)
}

// Swap is a paid mutator transaction binding the contract method 0x128acb08.
//
// Solidity: function swap(address recipient, bool zeroForOne, int256 amountSpecified, uint160 sqrtPriceLimitX96, bytes data) returns(int256 amount0, int256 amount1)
func (_PharaohV3 *PharaohV3TransactorSession) Swap(recipient common.Address, zeroForOne bool, amountSpecified *big.Int, sqrtPriceLimitX96 *big.Int, data []byte) (*types.Transaction, error) {
	return _PharaohV3.Contract.Swap(&_PharaohV3.TransactOpts, recipient, zeroForOne, amountSpecified, sqrtPriceLimitX96, data)
}

// PharaohV3BurnIterator is returned from FilterBurn and is used to iterate over the raw logs and unpacked data for Burn events raised by the PharaohV3 contract.
type PharaohV3BurnIterator struct {
	Event *PharaohV3Burn // Event containing the contract specifics and raw log

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
func (it *PharaohV3BurnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PharaohV3Burn)
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
		it.Event = new(PharaohV3Burn)
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
func (it *PharaohV3BurnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PharaohV3BurnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PharaohV3Burn represents a Burn event raised by the PharaohV3 contract.
type PharaohV3Burn struct {
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
func (_PharaohV3 *PharaohV3Filterer) FilterBurn(opts *bind.FilterOpts, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (*PharaohV3BurnIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var tickLowerRule []interface{}
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []interface{}
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _PharaohV3.contract.FilterLogs(opts, "Burn", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return &PharaohV3BurnIterator{contract: _PharaohV3.contract, event: "Burn", logs: logs, sub: sub}, nil
}

// WatchBurn is a free log subscription operation binding the contract event 0x0c396cd989a39f4459b5fa1aed6a9a8dcdbc45908acfd67e028cd568da98982c.
//
// Solidity: event Burn(address indexed owner, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
func (_PharaohV3 *PharaohV3Filterer) WatchBurn(opts *bind.WatchOpts, sink chan<- *PharaohV3Burn, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var tickLowerRule []interface{}
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []interface{}
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _PharaohV3.contract.WatchLogs(opts, "Burn", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PharaohV3Burn)
				if err := _PharaohV3.contract.UnpackLog(event, "Burn", log); err != nil {
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
func (_PharaohV3 *PharaohV3Filterer) ParseBurn(log types.Log) (*PharaohV3Burn, error) {
	event := new(PharaohV3Burn)
	if err := _PharaohV3.contract.UnpackLog(event, "Burn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PharaohV3CollectIterator is returned from FilterCollect and is used to iterate over the raw logs and unpacked data for Collect events raised by the PharaohV3 contract.
type PharaohV3CollectIterator struct {
	Event *PharaohV3Collect // Event containing the contract specifics and raw log

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
func (it *PharaohV3CollectIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PharaohV3Collect)
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
		it.Event = new(PharaohV3Collect)
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
func (it *PharaohV3CollectIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PharaohV3CollectIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PharaohV3Collect represents a Collect event raised by the PharaohV3 contract.
type PharaohV3Collect struct {
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
func (_PharaohV3 *PharaohV3Filterer) FilterCollect(opts *bind.FilterOpts, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (*PharaohV3CollectIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	var tickLowerRule []interface{}
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []interface{}
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _PharaohV3.contract.FilterLogs(opts, "Collect", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return &PharaohV3CollectIterator{contract: _PharaohV3.contract, event: "Collect", logs: logs, sub: sub}, nil
}

// WatchCollect is a free log subscription operation binding the contract event 0x70935338e69775456a85ddef226c395fb668b63fa0115f5f20610b388e6ca9c0.
//
// Solidity: event Collect(address indexed owner, address recipient, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount0, uint128 amount1)
func (_PharaohV3 *PharaohV3Filterer) WatchCollect(opts *bind.WatchOpts, sink chan<- *PharaohV3Collect, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	var tickLowerRule []interface{}
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []interface{}
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _PharaohV3.contract.WatchLogs(opts, "Collect", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PharaohV3Collect)
				if err := _PharaohV3.contract.UnpackLog(event, "Collect", log); err != nil {
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
func (_PharaohV3 *PharaohV3Filterer) ParseCollect(log types.Log) (*PharaohV3Collect, error) {
	event := new(PharaohV3Collect)
	if err := _PharaohV3.contract.UnpackLog(event, "Collect", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PharaohV3CollectProtocolIterator is returned from FilterCollectProtocol and is used to iterate over the raw logs and unpacked data for CollectProtocol events raised by the PharaohV3 contract.
type PharaohV3CollectProtocolIterator struct {
	Event *PharaohV3CollectProtocol // Event containing the contract specifics and raw log

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
func (it *PharaohV3CollectProtocolIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PharaohV3CollectProtocol)
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
		it.Event = new(PharaohV3CollectProtocol)
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
func (it *PharaohV3CollectProtocolIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PharaohV3CollectProtocolIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PharaohV3CollectProtocol represents a CollectProtocol event raised by the PharaohV3 contract.
type PharaohV3CollectProtocol struct {
	Sender    common.Address
	Recipient common.Address
	Amount0   *big.Int
	Amount1   *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterCollectProtocol is a free log retrieval operation binding the contract event 0x596b573906218d3411850b26a6b437d6c4522fdb43d2d2386263f86d50b8b151.
//
// Solidity: event CollectProtocol(address indexed sender, address indexed recipient, uint128 amount0, uint128 amount1)
func (_PharaohV3 *PharaohV3Filterer) FilterCollectProtocol(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address) (*PharaohV3CollectProtocolIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _PharaohV3.contract.FilterLogs(opts, "CollectProtocol", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &PharaohV3CollectProtocolIterator{contract: _PharaohV3.contract, event: "CollectProtocol", logs: logs, sub: sub}, nil
}

// WatchCollectProtocol is a free log subscription operation binding the contract event 0x596b573906218d3411850b26a6b437d6c4522fdb43d2d2386263f86d50b8b151.
//
// Solidity: event CollectProtocol(address indexed sender, address indexed recipient, uint128 amount0, uint128 amount1)
func (_PharaohV3 *PharaohV3Filterer) WatchCollectProtocol(opts *bind.WatchOpts, sink chan<- *PharaohV3CollectProtocol, sender []common.Address, recipient []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _PharaohV3.contract.WatchLogs(opts, "CollectProtocol", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PharaohV3CollectProtocol)
				if err := _PharaohV3.contract.UnpackLog(event, "CollectProtocol", log); err != nil {
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
func (_PharaohV3 *PharaohV3Filterer) ParseCollectProtocol(log types.Log) (*PharaohV3CollectProtocol, error) {
	event := new(PharaohV3CollectProtocol)
	if err := _PharaohV3.contract.UnpackLog(event, "CollectProtocol", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PharaohV3FlashIterator is returned from FilterFlash and is used to iterate over the raw logs and unpacked data for Flash events raised by the PharaohV3 contract.
type PharaohV3FlashIterator struct {
	Event *PharaohV3Flash // Event containing the contract specifics and raw log

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
func (it *PharaohV3FlashIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PharaohV3Flash)
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
		it.Event = new(PharaohV3Flash)
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
func (it *PharaohV3FlashIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PharaohV3FlashIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PharaohV3Flash represents a Flash event raised by the PharaohV3 contract.
type PharaohV3Flash struct {
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
func (_PharaohV3 *PharaohV3Filterer) FilterFlash(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address) (*PharaohV3FlashIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _PharaohV3.contract.FilterLogs(opts, "Flash", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &PharaohV3FlashIterator{contract: _PharaohV3.contract, event: "Flash", logs: logs, sub: sub}, nil
}

// WatchFlash is a free log subscription operation binding the contract event 0xbdbdb71d7860376ba52b25a5028beea23581364a40522f6bcfb86bb1f2dca633.
//
// Solidity: event Flash(address indexed sender, address indexed recipient, uint256 amount0, uint256 amount1, uint256 paid0, uint256 paid1)
func (_PharaohV3 *PharaohV3Filterer) WatchFlash(opts *bind.WatchOpts, sink chan<- *PharaohV3Flash, sender []common.Address, recipient []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _PharaohV3.contract.WatchLogs(opts, "Flash", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PharaohV3Flash)
				if err := _PharaohV3.contract.UnpackLog(event, "Flash", log); err != nil {
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
func (_PharaohV3 *PharaohV3Filterer) ParseFlash(log types.Log) (*PharaohV3Flash, error) {
	event := new(PharaohV3Flash)
	if err := _PharaohV3.contract.UnpackLog(event, "Flash", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PharaohV3IncreaseObservationCardinalityNextIterator is returned from FilterIncreaseObservationCardinalityNext and is used to iterate over the raw logs and unpacked data for IncreaseObservationCardinalityNext events raised by the PharaohV3 contract.
type PharaohV3IncreaseObservationCardinalityNextIterator struct {
	Event *PharaohV3IncreaseObservationCardinalityNext // Event containing the contract specifics and raw log

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
func (it *PharaohV3IncreaseObservationCardinalityNextIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PharaohV3IncreaseObservationCardinalityNext)
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
		it.Event = new(PharaohV3IncreaseObservationCardinalityNext)
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
func (it *PharaohV3IncreaseObservationCardinalityNextIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PharaohV3IncreaseObservationCardinalityNextIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PharaohV3IncreaseObservationCardinalityNext represents a IncreaseObservationCardinalityNext event raised by the PharaohV3 contract.
type PharaohV3IncreaseObservationCardinalityNext struct {
	ObservationCardinalityNextOld uint16
	ObservationCardinalityNextNew uint16
	Raw                           types.Log // Blockchain specific contextual infos
}

// FilterIncreaseObservationCardinalityNext is a free log retrieval operation binding the contract event 0xac49e518f90a358f652e4400164f05a5d8f7e35e7747279bc3a93dbf584e125a.
//
// Solidity: event IncreaseObservationCardinalityNext(uint16 observationCardinalityNextOld, uint16 observationCardinalityNextNew)
func (_PharaohV3 *PharaohV3Filterer) FilterIncreaseObservationCardinalityNext(opts *bind.FilterOpts) (*PharaohV3IncreaseObservationCardinalityNextIterator, error) {

	logs, sub, err := _PharaohV3.contract.FilterLogs(opts, "IncreaseObservationCardinalityNext")
	if err != nil {
		return nil, err
	}
	return &PharaohV3IncreaseObservationCardinalityNextIterator{contract: _PharaohV3.contract, event: "IncreaseObservationCardinalityNext", logs: logs, sub: sub}, nil
}

// WatchIncreaseObservationCardinalityNext is a free log subscription operation binding the contract event 0xac49e518f90a358f652e4400164f05a5d8f7e35e7747279bc3a93dbf584e125a.
//
// Solidity: event IncreaseObservationCardinalityNext(uint16 observationCardinalityNextOld, uint16 observationCardinalityNextNew)
func (_PharaohV3 *PharaohV3Filterer) WatchIncreaseObservationCardinalityNext(opts *bind.WatchOpts, sink chan<- *PharaohV3IncreaseObservationCardinalityNext) (event.Subscription, error) {

	logs, sub, err := _PharaohV3.contract.WatchLogs(opts, "IncreaseObservationCardinalityNext")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PharaohV3IncreaseObservationCardinalityNext)
				if err := _PharaohV3.contract.UnpackLog(event, "IncreaseObservationCardinalityNext", log); err != nil {
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
func (_PharaohV3 *PharaohV3Filterer) ParseIncreaseObservationCardinalityNext(log types.Log) (*PharaohV3IncreaseObservationCardinalityNext, error) {
	event := new(PharaohV3IncreaseObservationCardinalityNext)
	if err := _PharaohV3.contract.UnpackLog(event, "IncreaseObservationCardinalityNext", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PharaohV3InitializeIterator is returned from FilterInitialize and is used to iterate over the raw logs and unpacked data for Initialize events raised by the PharaohV3 contract.
type PharaohV3InitializeIterator struct {
	Event *PharaohV3Initialize // Event containing the contract specifics and raw log

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
func (it *PharaohV3InitializeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PharaohV3Initialize)
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
		it.Event = new(PharaohV3Initialize)
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
func (it *PharaohV3InitializeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PharaohV3InitializeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PharaohV3Initialize represents a Initialize event raised by the PharaohV3 contract.
type PharaohV3Initialize struct {
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterInitialize is a free log retrieval operation binding the contract event 0x98636036cb66a9c19a37435efc1e90142190214e8abeb821bdba3f2990dd4c95.
//
// Solidity: event Initialize(uint160 sqrtPriceX96, int24 tick)
func (_PharaohV3 *PharaohV3Filterer) FilterInitialize(opts *bind.FilterOpts) (*PharaohV3InitializeIterator, error) {

	logs, sub, err := _PharaohV3.contract.FilterLogs(opts, "Initialize")
	if err != nil {
		return nil, err
	}
	return &PharaohV3InitializeIterator{contract: _PharaohV3.contract, event: "Initialize", logs: logs, sub: sub}, nil
}

// WatchInitialize is a free log subscription operation binding the contract event 0x98636036cb66a9c19a37435efc1e90142190214e8abeb821bdba3f2990dd4c95.
//
// Solidity: event Initialize(uint160 sqrtPriceX96, int24 tick)
func (_PharaohV3 *PharaohV3Filterer) WatchInitialize(opts *bind.WatchOpts, sink chan<- *PharaohV3Initialize) (event.Subscription, error) {

	logs, sub, err := _PharaohV3.contract.WatchLogs(opts, "Initialize")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PharaohV3Initialize)
				if err := _PharaohV3.contract.UnpackLog(event, "Initialize", log); err != nil {
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
func (_PharaohV3 *PharaohV3Filterer) ParseInitialize(log types.Log) (*PharaohV3Initialize, error) {
	event := new(PharaohV3Initialize)
	if err := _PharaohV3.contract.UnpackLog(event, "Initialize", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PharaohV3MintIterator is returned from FilterMint and is used to iterate over the raw logs and unpacked data for Mint events raised by the PharaohV3 contract.
type PharaohV3MintIterator struct {
	Event *PharaohV3Mint // Event containing the contract specifics and raw log

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
func (it *PharaohV3MintIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PharaohV3Mint)
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
		it.Event = new(PharaohV3Mint)
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
func (it *PharaohV3MintIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PharaohV3MintIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PharaohV3Mint represents a Mint event raised by the PharaohV3 contract.
type PharaohV3Mint struct {
	Sender    common.Address
	Owner     common.Address
	Index     *big.Int
	TickLower *big.Int
	TickUpper *big.Int
	Amount    *big.Int
	Amount0   *big.Int
	Amount1   *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterMint is a free log retrieval operation binding the contract event 0xd78218c0d304e8893cb3200abe394bbc8d5b7804d9c51f236df9fdcf481d02d3.
//
// Solidity: event Mint(address sender, address indexed owner, uint256 index, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
func (_PharaohV3 *PharaohV3Filterer) FilterMint(opts *bind.FilterOpts, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (*PharaohV3MintIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	var tickLowerRule []interface{}
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []interface{}
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _PharaohV3.contract.FilterLogs(opts, "Mint", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return &PharaohV3MintIterator{contract: _PharaohV3.contract, event: "Mint", logs: logs, sub: sub}, nil
}

// WatchMint is a free log subscription operation binding the contract event 0xd78218c0d304e8893cb3200abe394bbc8d5b7804d9c51f236df9fdcf481d02d3.
//
// Solidity: event Mint(address sender, address indexed owner, uint256 index, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
func (_PharaohV3 *PharaohV3Filterer) WatchMint(opts *bind.WatchOpts, sink chan<- *PharaohV3Mint, owner []common.Address, tickLower []*big.Int, tickUpper []*big.Int) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	var tickLowerRule []interface{}
	for _, tickLowerItem := range tickLower {
		tickLowerRule = append(tickLowerRule, tickLowerItem)
	}
	var tickUpperRule []interface{}
	for _, tickUpperItem := range tickUpper {
		tickUpperRule = append(tickUpperRule, tickUpperItem)
	}

	logs, sub, err := _PharaohV3.contract.WatchLogs(opts, "Mint", ownerRule, tickLowerRule, tickUpperRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PharaohV3Mint)
				if err := _PharaohV3.contract.UnpackLog(event, "Mint", log); err != nil {
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

// ParseMint is a log parse operation binding the contract event 0xd78218c0d304e8893cb3200abe394bbc8d5b7804d9c51f236df9fdcf481d02d3.
//
// Solidity: event Mint(address sender, address indexed owner, uint256 index, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
func (_PharaohV3 *PharaohV3Filterer) ParseMint(log types.Log) (*PharaohV3Mint, error) {
	event := new(PharaohV3Mint)
	if err := _PharaohV3.contract.UnpackLog(event, "Mint", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PharaohV3SetFeeProtocolIterator is returned from FilterSetFeeProtocol and is used to iterate over the raw logs and unpacked data for SetFeeProtocol events raised by the PharaohV3 contract.
type PharaohV3SetFeeProtocolIterator struct {
	Event *PharaohV3SetFeeProtocol // Event containing the contract specifics and raw log

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
func (it *PharaohV3SetFeeProtocolIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PharaohV3SetFeeProtocol)
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
		it.Event = new(PharaohV3SetFeeProtocol)
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
func (it *PharaohV3SetFeeProtocolIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PharaohV3SetFeeProtocolIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PharaohV3SetFeeProtocol represents a SetFeeProtocol event raised by the PharaohV3 contract.
type PharaohV3SetFeeProtocol struct {
	FeeProtocol0Old uint8
	FeeProtocol1Old uint8
	FeeProtocol0New uint8
	FeeProtocol1New uint8
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterSetFeeProtocol is a free log retrieval operation binding the contract event 0x973d8d92bb299f4af6ce49b52a8adb85ae46b9f214c4c4fc06ac77401237b133.
//
// Solidity: event SetFeeProtocol(uint8 feeProtocol0Old, uint8 feeProtocol1Old, uint8 feeProtocol0New, uint8 feeProtocol1New)
func (_PharaohV3 *PharaohV3Filterer) FilterSetFeeProtocol(opts *bind.FilterOpts) (*PharaohV3SetFeeProtocolIterator, error) {

	logs, sub, err := _PharaohV3.contract.FilterLogs(opts, "SetFeeProtocol")
	if err != nil {
		return nil, err
	}
	return &PharaohV3SetFeeProtocolIterator{contract: _PharaohV3.contract, event: "SetFeeProtocol", logs: logs, sub: sub}, nil
}

// WatchSetFeeProtocol is a free log subscription operation binding the contract event 0x973d8d92bb299f4af6ce49b52a8adb85ae46b9f214c4c4fc06ac77401237b133.
//
// Solidity: event SetFeeProtocol(uint8 feeProtocol0Old, uint8 feeProtocol1Old, uint8 feeProtocol0New, uint8 feeProtocol1New)
func (_PharaohV3 *PharaohV3Filterer) WatchSetFeeProtocol(opts *bind.WatchOpts, sink chan<- *PharaohV3SetFeeProtocol) (event.Subscription, error) {

	logs, sub, err := _PharaohV3.contract.WatchLogs(opts, "SetFeeProtocol")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PharaohV3SetFeeProtocol)
				if err := _PharaohV3.contract.UnpackLog(event, "SetFeeProtocol", log); err != nil {
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
func (_PharaohV3 *PharaohV3Filterer) ParseSetFeeProtocol(log types.Log) (*PharaohV3SetFeeProtocol, error) {
	event := new(PharaohV3SetFeeProtocol)
	if err := _PharaohV3.contract.UnpackLog(event, "SetFeeProtocol", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PharaohV3SwapIterator is returned from FilterSwap and is used to iterate over the raw logs and unpacked data for Swap events raised by the PharaohV3 contract.
type PharaohV3SwapIterator struct {
	Event *PharaohV3Swap // Event containing the contract specifics and raw log

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
func (it *PharaohV3SwapIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PharaohV3Swap)
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
		it.Event = new(PharaohV3Swap)
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
func (it *PharaohV3SwapIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PharaohV3SwapIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PharaohV3Swap represents a Swap event raised by the PharaohV3 contract.
type PharaohV3Swap struct {
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
func (_PharaohV3 *PharaohV3Filterer) FilterSwap(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address) (*PharaohV3SwapIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _PharaohV3.contract.FilterLogs(opts, "Swap", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &PharaohV3SwapIterator{contract: _PharaohV3.contract, event: "Swap", logs: logs, sub: sub}, nil
}

// WatchSwap is a free log subscription operation binding the contract event 0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67.
//
// Solidity: event Swap(address indexed sender, address indexed recipient, int256 amount0, int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick)
func (_PharaohV3 *PharaohV3Filterer) WatchSwap(opts *bind.WatchOpts, sink chan<- *PharaohV3Swap, sender []common.Address, recipient []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _PharaohV3.contract.WatchLogs(opts, "Swap", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PharaohV3Swap)
				if err := _PharaohV3.contract.UnpackLog(event, "Swap", log); err != nil {
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
func (_PharaohV3 *PharaohV3Filterer) ParseSwap(log types.Log) (*PharaohV3Swap, error) {
	event := new(PharaohV3Swap)
	if err := _PharaohV3.contract.UnpackLog(event, "Swap", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
