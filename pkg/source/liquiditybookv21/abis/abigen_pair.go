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

// ILBPairPriceData is an auto generated low-level Go binding around an user-defined struct.
type ILBPairPriceData struct {
	NativePriceUSD    *big.Int
	TokenXPriceNative *big.Int
	TokenYPriceNative *big.Int
}

// LBPairMetaData contains all meta data concerning the LBPair contract.
var LBPairMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"ApprovalForAll\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"feeRecipient\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"protocolFees\",\"type\":\"bytes32\"}],\"name\":\"CollectedProtocolFees\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"id\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"totalFees\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"protocolFees\",\"type\":\"bytes32\"}],\"name\":\"CompositionFees\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"internalType\":\"bytes32[]\",\"name\":\"amounts\",\"type\":\"bytes32[]\"}],\"name\":\"DepositedToBins\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"internalType\":\"bytes32[]\",\"name\":\"amounts\",\"type\":\"bytes32[]\"},{\"internalType\":\"structILBPair.PriceData\",\"name\":\"priceData\",\"type\":\"tuple\",\"components\":[{\"internalType\":\"uint256\",\"name\":\"nativePriceUSD\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"tokenXPriceUSD\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"tokenYPriceUSD\",\"type\":\"uint256\"}]}],\"name\":\"DepositedToBins\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"contractILBFlashLoanCallback\",\"name\":\"receiver\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"activeId\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"amounts\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"totalFees\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"protocolFees\",\"type\":\"bytes32\"}],\"name\":\"FlashLoan\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"contractILBFlashLoanCallback\",\"name\":\"receiver\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"activeId\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"amounts\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"totalFees\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"protocolFees\",\"type\":\"bytes32\"},{\"internalType\":\"structILBPair.PriceData\",\"name\":\"priceData\",\"type\":\"tuple\",\"components\":[{\"internalType\":\"uint256\",\"name\":\"nativePriceUSD\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"tokenXPriceNative\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"tokenYPriceNative\",\"type\":\"uint256\"}]}],\"name\":\"FlashLoan\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"idReference\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"volatilityReference\",\"type\":\"uint24\"}],\"name\":\"ForcedDecay\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint16\",\"name\":\"oracleLength\",\"type\":\"uint16\"}],\"name\":\"OracleLengthIncreased\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint16\",\"name\":\"baseFactor\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"filterPeriod\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"decayPeriod\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"reductionFactor\",\"type\":\"uint16\"},{\"internalType\":\"uint24\",\"name\":\"variableFeeControl\",\"type\":\"uint24\"},{\"internalType\":\"uint16\",\"name\":\"protocolShare\",\"type\":\"uint16\"},{\"internalType\":\"uint24\",\"name\":\"maxVolatilityAccumulator\",\"type\":\"uint24\"}],\"name\":\"StaticFeeParametersSet\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"id\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"amountsIn\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"amountsOut\",\"type\":\"bytes32\"},{\"internalType\":\"uint24\",\"name\":\"volatilityAccumulator\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"totalFees\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"protocolFees\",\"type\":\"bytes32\"}],\"name\":\"Swap\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"id\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"amountsIn\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"amountsOut\",\"type\":\"bytes32\"},{\"internalType\":\"uint24\",\"name\":\"volatilityAccumulator\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"totalFees\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"protocolFees\",\"type\":\"bytes32\"},{\"internalType\":\"structILBPair.PriceData\",\"name\":\"priceData\",\"type\":\"tuple\",\"components\":[{\"internalType\":\"uint256\",\"name\":\"nativePriceUSD\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"tokenXPriceNative\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"tokenYPriceNative\",\"type\":\"uint256\"}]}],\"name\":\"Swap\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"amounts\",\"type\":\"uint256[]\"}],\"name\":\"TransferBatch\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"internalType\":\"bytes32[]\",\"name\":\"amounts\",\"type\":\"bytes32[]\"}],\"name\":\"WithdrawnFromBins\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"internalType\":\"bytes32[]\",\"name\":\"amounts\",\"type\":\"bytes32[]\"},{\"internalType\":\"structILBPair.PriceData\",\"name\":\"priceData\",\"type\":\"tuple\",\"components\":[{\"internalType\":\"uint256\",\"name\":\"nativePriceUSD\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"tokenXPriceNative\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"tokenYPriceNative\",\"type\":\"uint256\"}]}],\"name\":\"WithdrawnFromBins\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"accounts\",\"type\":\"address[]\"},{\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"}],\"name\":\"balanceOfBatch\",\"outputs\":[{\"internalType\":\"uint256[]\",\"name\":\"batchBalances\",\"type\":\"uint256[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"name\":\"getActiveId\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"activeId\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint24\",\"name\":\"id\",\"type\":\"uint24\"}],\"name\":\"getBin\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"binReserveX\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"binReserveY\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"name\":\"getBinStep\",\"outputs\":[{\"internalType\":\"uint16\",\"type\":\"uint16\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"name\":\"getFactory\",\"outputs\":[{\"internalType\":\"contractILBFactory\",\"name\":\"factory\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"}],\"name\":\"getIdFromPrice\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"id\",\"type\":\"uint24\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"swapForY\",\"type\":\"bool\"},{\"internalType\":\"uint24\",\"name\":\"id\",\"type\":\"uint24\"}],\"name\":\"getNextNonEmptyBin\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"nextId\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"name\":\"getOracleParameters\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"sampleLifetime\",\"type\":\"uint8\"},{\"internalType\":\"uint16\",\"name\":\"size\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"activeSize\",\"type\":\"uint16\"},{\"internalType\":\"uint40\",\"name\":\"lastUpdated\",\"type\":\"uint40\"},{\"internalType\":\"uint40\",\"name\":\"firstTimestamp\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint40\",\"name\":\"lookupTimestamp\",\"type\":\"uint40\"}],\"name\":\"getOracleSampleAt\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"cumulativeId\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"cumulativeVolatility\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"cumulativeBinCrossed\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint24\",\"name\":\"id\",\"type\":\"uint24\"}],\"name\":\"getPriceFromId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"name\":\"getProtocolFees\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"protocolFeeX\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"protocolFeeY\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"name\":\"getReserves\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"reserveX\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"reserveY\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"name\":\"getStaticFeeParameters\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"baseFactor\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"filterPeriod\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"decayPeriod\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"reductionFactor\",\"type\":\"uint16\"},{\"internalType\":\"uint24\",\"name\":\"variableFeeControl\",\"type\":\"uint24\"},{\"internalType\":\"uint16\",\"name\":\"protocolShare\",\"type\":\"uint16\"},{\"internalType\":\"uint24\",\"name\":\"maxVolatilityAccumulator\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint128\",\"name\":\"amountOut\",\"type\":\"uint128\"},{\"internalType\":\"bool\",\"name\":\"swapForY\",\"type\":\"bool\"}],\"name\":\"getSwapIn\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"amountIn\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amountOutLeft\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"fee\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint128\",\"name\":\"amountIn\",\"type\":\"uint128\"},{\"internalType\":\"bool\",\"name\":\"swapForY\",\"type\":\"bool\"}],\"name\":\"getSwapOut\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"amountInLeft\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amountOut\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"fee\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"name\":\"getTokenX\",\"outputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"tokenX\",\"type\":\"address\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"name\":\"getTokenY\",\"outputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"tokenY\",\"type\":\"address\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"name\":\"getVariableFeeParameters\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"volatilityAccumulator\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"volatilityReference\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"idReference\",\"type\":\"uint24\"},{\"internalType\":\"uint40\",\"name\":\"timeOfLastUpdate\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"isApprovedForAll\",\"outputs\":[{\"internalType\":\"bool\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// LBPairABI is the input ABI used to generate the binding from.
// Deprecated: Use LBPairMetaData.ABI instead.
var LBPairABI = LBPairMetaData.ABI

// LBPair is an auto generated Go binding around an Ethereum contract.
type LBPair struct {
	LBPairCaller     // Read-only binding to the contract
	LBPairTransactor // Write-only binding to the contract
	LBPairFilterer   // Log filterer for contract events
}

// LBPairCaller is an auto generated read-only Go binding around an Ethereum contract.
type LBPairCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LBPairTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LBPairTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LBPairFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LBPairFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LBPairSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LBPairSession struct {
	Contract     *LBPair           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// LBPairCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LBPairCallerSession struct {
	Contract *LBPairCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// LBPairTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LBPairTransactorSession struct {
	Contract     *LBPairTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// LBPairRaw is an auto generated low-level Go binding around an Ethereum contract.
type LBPairRaw struct {
	Contract *LBPair // Generic contract binding to access the raw methods on
}

// LBPairCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LBPairCallerRaw struct {
	Contract *LBPairCaller // Generic read-only contract binding to access the raw methods on
}

// LBPairTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LBPairTransactorRaw struct {
	Contract *LBPairTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLBPair creates a new instance of LBPair, bound to a specific deployed contract.
func NewLBPair(address common.Address, backend bind.ContractBackend) (*LBPair, error) {
	contract, err := bindLBPair(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LBPair{LBPairCaller: LBPairCaller{contract: contract}, LBPairTransactor: LBPairTransactor{contract: contract}, LBPairFilterer: LBPairFilterer{contract: contract}}, nil
}

// NewLBPairCaller creates a new read-only instance of LBPair, bound to a specific deployed contract.
func NewLBPairCaller(address common.Address, caller bind.ContractCaller) (*LBPairCaller, error) {
	contract, err := bindLBPair(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LBPairCaller{contract: contract}, nil
}

// NewLBPairTransactor creates a new write-only instance of LBPair, bound to a specific deployed contract.
func NewLBPairTransactor(address common.Address, transactor bind.ContractTransactor) (*LBPairTransactor, error) {
	contract, err := bindLBPair(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LBPairTransactor{contract: contract}, nil
}

// NewLBPairFilterer creates a new log filterer instance of LBPair, bound to a specific deployed contract.
func NewLBPairFilterer(address common.Address, filterer bind.ContractFilterer) (*LBPairFilterer, error) {
	contract, err := bindLBPair(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LBPairFilterer{contract: contract}, nil
}

// bindLBPair binds a generic wrapper to an already deployed contract.
func bindLBPair(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := LBPairMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LBPair *LBPairRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LBPair.Contract.LBPairCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LBPair *LBPairRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LBPair.Contract.LBPairTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LBPair *LBPairRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LBPair.Contract.LBPairTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LBPair *LBPairCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LBPair.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LBPair *LBPairTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LBPair.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LBPair *LBPairTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LBPair.Contract.contract.Transact(opts, method, params...)
}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address account, uint256 id) view returns(uint256)
func (_LBPair *LBPairCaller) BalanceOf(opts *bind.CallOpts, account common.Address, id *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "balanceOf", account, id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address account, uint256 id) view returns(uint256)
func (_LBPair *LBPairSession) BalanceOf(account common.Address, id *big.Int) (*big.Int, error) {
	return _LBPair.Contract.BalanceOf(&_LBPair.CallOpts, account, id)
}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address account, uint256 id) view returns(uint256)
func (_LBPair *LBPairCallerSession) BalanceOf(account common.Address, id *big.Int) (*big.Int, error) {
	return _LBPair.Contract.BalanceOf(&_LBPair.CallOpts, account, id)
}

// BalanceOfBatch is a free data retrieval call binding the contract method 0x4e1273f4.
//
// Solidity: function balanceOfBatch(address[] accounts, uint256[] ids) view returns(uint256[] batchBalances)
func (_LBPair *LBPairCaller) BalanceOfBatch(opts *bind.CallOpts, accounts []common.Address, ids []*big.Int) ([]*big.Int, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "balanceOfBatch", accounts, ids)

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// BalanceOfBatch is a free data retrieval call binding the contract method 0x4e1273f4.
//
// Solidity: function balanceOfBatch(address[] accounts, uint256[] ids) view returns(uint256[] batchBalances)
func (_LBPair *LBPairSession) BalanceOfBatch(accounts []common.Address, ids []*big.Int) ([]*big.Int, error) {
	return _LBPair.Contract.BalanceOfBatch(&_LBPair.CallOpts, accounts, ids)
}

// BalanceOfBatch is a free data retrieval call binding the contract method 0x4e1273f4.
//
// Solidity: function balanceOfBatch(address[] accounts, uint256[] ids) view returns(uint256[] batchBalances)
func (_LBPair *LBPairCallerSession) BalanceOfBatch(accounts []common.Address, ids []*big.Int) ([]*big.Int, error) {
	return _LBPair.Contract.BalanceOfBatch(&_LBPair.CallOpts, accounts, ids)
}

// GetActiveId is a free data retrieval call binding the contract method 0xdbe65edc.
//
// Solidity: function getActiveId() view returns(uint24 activeId)
func (_LBPair *LBPairCaller) GetActiveId(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getActiveId")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetActiveId is a free data retrieval call binding the contract method 0xdbe65edc.
//
// Solidity: function getActiveId() view returns(uint24 activeId)
func (_LBPair *LBPairSession) GetActiveId() (*big.Int, error) {
	return _LBPair.Contract.GetActiveId(&_LBPair.CallOpts)
}

// GetActiveId is a free data retrieval call binding the contract method 0xdbe65edc.
//
// Solidity: function getActiveId() view returns(uint24 activeId)
func (_LBPair *LBPairCallerSession) GetActiveId() (*big.Int, error) {
	return _LBPair.Contract.GetActiveId(&_LBPair.CallOpts)
}

// GetBin is a free data retrieval call binding the contract method 0x0abe9688.
//
// Solidity: function getBin(uint24 id) view returns(uint128 binReserveX, uint128 binReserveY)
func (_LBPair *LBPairCaller) GetBin(opts *bind.CallOpts, id *big.Int) (struct {
	BinReserveX *big.Int
	BinReserveY *big.Int
}, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getBin", id)

	outstruct := new(struct {
		BinReserveX *big.Int
		BinReserveY *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.BinReserveX = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.BinReserveY = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetBin is a free data retrieval call binding the contract method 0x0abe9688.
//
// Solidity: function getBin(uint24 id) view returns(uint128 binReserveX, uint128 binReserveY)
func (_LBPair *LBPairSession) GetBin(id *big.Int) (struct {
	BinReserveX *big.Int
	BinReserveY *big.Int
}, error) {
	return _LBPair.Contract.GetBin(&_LBPair.CallOpts, id)
}

// GetBin is a free data retrieval call binding the contract method 0x0abe9688.
//
// Solidity: function getBin(uint24 id) view returns(uint128 binReserveX, uint128 binReserveY)
func (_LBPair *LBPairCallerSession) GetBin(id *big.Int) (struct {
	BinReserveX *big.Int
	BinReserveY *big.Int
}, error) {
	return _LBPair.Contract.GetBin(&_LBPair.CallOpts, id)
}

// GetBinStep is a free data retrieval call binding the contract method 0x17f11ecc.
//
// Solidity: function getBinStep() pure returns(uint16)
func (_LBPair *LBPairCaller) GetBinStep(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getBinStep")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// GetBinStep is a free data retrieval call binding the contract method 0x17f11ecc.
//
// Solidity: function getBinStep() pure returns(uint16)
func (_LBPair *LBPairSession) GetBinStep() (uint16, error) {
	return _LBPair.Contract.GetBinStep(&_LBPair.CallOpts)
}

// GetBinStep is a free data retrieval call binding the contract method 0x17f11ecc.
//
// Solidity: function getBinStep() pure returns(uint16)
func (_LBPair *LBPairCallerSession) GetBinStep() (uint16, error) {
	return _LBPair.Contract.GetBinStep(&_LBPair.CallOpts)
}

// GetFactory is a free data retrieval call binding the contract method 0x88cc58e4.
//
// Solidity: function getFactory() view returns(address factory)
func (_LBPair *LBPairCaller) GetFactory(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getFactory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetFactory is a free data retrieval call binding the contract method 0x88cc58e4.
//
// Solidity: function getFactory() view returns(address factory)
func (_LBPair *LBPairSession) GetFactory() (common.Address, error) {
	return _LBPair.Contract.GetFactory(&_LBPair.CallOpts)
}

// GetFactory is a free data retrieval call binding the contract method 0x88cc58e4.
//
// Solidity: function getFactory() view returns(address factory)
func (_LBPair *LBPairCallerSession) GetFactory() (common.Address, error) {
	return _LBPair.Contract.GetFactory(&_LBPair.CallOpts)
}

// GetIdFromPrice is a free data retrieval call binding the contract method 0xf5e29329.
//
// Solidity: function getIdFromPrice(uint256 price) pure returns(uint24 id)
func (_LBPair *LBPairCaller) GetIdFromPrice(opts *bind.CallOpts, price *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getIdFromPrice", price)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetIdFromPrice is a free data retrieval call binding the contract method 0xf5e29329.
//
// Solidity: function getIdFromPrice(uint256 price) pure returns(uint24 id)
func (_LBPair *LBPairSession) GetIdFromPrice(price *big.Int) (*big.Int, error) {
	return _LBPair.Contract.GetIdFromPrice(&_LBPair.CallOpts, price)
}

// GetIdFromPrice is a free data retrieval call binding the contract method 0xf5e29329.
//
// Solidity: function getIdFromPrice(uint256 price) pure returns(uint24 id)
func (_LBPair *LBPairCallerSession) GetIdFromPrice(price *big.Int) (*big.Int, error) {
	return _LBPair.Contract.GetIdFromPrice(&_LBPair.CallOpts, price)
}

// GetNextNonEmptyBin is a free data retrieval call binding the contract method 0xa41a01fb.
//
// Solidity: function getNextNonEmptyBin(bool swapForY, uint24 id) view returns(uint24 nextId)
func (_LBPair *LBPairCaller) GetNextNonEmptyBin(opts *bind.CallOpts, swapForY bool, id *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getNextNonEmptyBin", swapForY, id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNextNonEmptyBin is a free data retrieval call binding the contract method 0xa41a01fb.
//
// Solidity: function getNextNonEmptyBin(bool swapForY, uint24 id) view returns(uint24 nextId)
func (_LBPair *LBPairSession) GetNextNonEmptyBin(swapForY bool, id *big.Int) (*big.Int, error) {
	return _LBPair.Contract.GetNextNonEmptyBin(&_LBPair.CallOpts, swapForY, id)
}

// GetNextNonEmptyBin is a free data retrieval call binding the contract method 0xa41a01fb.
//
// Solidity: function getNextNonEmptyBin(bool swapForY, uint24 id) view returns(uint24 nextId)
func (_LBPair *LBPairCallerSession) GetNextNonEmptyBin(swapForY bool, id *big.Int) (*big.Int, error) {
	return _LBPair.Contract.GetNextNonEmptyBin(&_LBPair.CallOpts, swapForY, id)
}

// GetOracleParameters is a free data retrieval call binding the contract method 0x55182894.
//
// Solidity: function getOracleParameters() view returns(uint8 sampleLifetime, uint16 size, uint16 activeSize, uint40 lastUpdated, uint40 firstTimestamp)
func (_LBPair *LBPairCaller) GetOracleParameters(opts *bind.CallOpts) (struct {
	SampleLifetime uint8
	Size           uint16
	ActiveSize     uint16
	LastUpdated    *big.Int
	FirstTimestamp *big.Int
}, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getOracleParameters")

	outstruct := new(struct {
		SampleLifetime uint8
		Size           uint16
		ActiveSize     uint16
		LastUpdated    *big.Int
		FirstTimestamp *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.SampleLifetime = *abi.ConvertType(out[0], new(uint8)).(*uint8)
	outstruct.Size = *abi.ConvertType(out[1], new(uint16)).(*uint16)
	outstruct.ActiveSize = *abi.ConvertType(out[2], new(uint16)).(*uint16)
	outstruct.LastUpdated = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.FirstTimestamp = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetOracleParameters is a free data retrieval call binding the contract method 0x55182894.
//
// Solidity: function getOracleParameters() view returns(uint8 sampleLifetime, uint16 size, uint16 activeSize, uint40 lastUpdated, uint40 firstTimestamp)
func (_LBPair *LBPairSession) GetOracleParameters() (struct {
	SampleLifetime uint8
	Size           uint16
	ActiveSize     uint16
	LastUpdated    *big.Int
	FirstTimestamp *big.Int
}, error) {
	return _LBPair.Contract.GetOracleParameters(&_LBPair.CallOpts)
}

// GetOracleParameters is a free data retrieval call binding the contract method 0x55182894.
//
// Solidity: function getOracleParameters() view returns(uint8 sampleLifetime, uint16 size, uint16 activeSize, uint40 lastUpdated, uint40 firstTimestamp)
func (_LBPair *LBPairCallerSession) GetOracleParameters() (struct {
	SampleLifetime uint8
	Size           uint16
	ActiveSize     uint16
	LastUpdated    *big.Int
	FirstTimestamp *big.Int
}, error) {
	return _LBPair.Contract.GetOracleParameters(&_LBPair.CallOpts)
}

// GetOracleSampleAt is a free data retrieval call binding the contract method 0x8940a16a.
//
// Solidity: function getOracleSampleAt(uint40 lookupTimestamp) view returns(uint64 cumulativeId, uint64 cumulativeVolatility, uint64 cumulativeBinCrossed)
func (_LBPair *LBPairCaller) GetOracleSampleAt(opts *bind.CallOpts, lookupTimestamp *big.Int) (struct {
	CumulativeId         uint64
	CumulativeVolatility uint64
	CumulativeBinCrossed uint64
}, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getOracleSampleAt", lookupTimestamp)

	outstruct := new(struct {
		CumulativeId         uint64
		CumulativeVolatility uint64
		CumulativeBinCrossed uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.CumulativeId = *abi.ConvertType(out[0], new(uint64)).(*uint64)
	outstruct.CumulativeVolatility = *abi.ConvertType(out[1], new(uint64)).(*uint64)
	outstruct.CumulativeBinCrossed = *abi.ConvertType(out[2], new(uint64)).(*uint64)

	return *outstruct, err

}

// GetOracleSampleAt is a free data retrieval call binding the contract method 0x8940a16a.
//
// Solidity: function getOracleSampleAt(uint40 lookupTimestamp) view returns(uint64 cumulativeId, uint64 cumulativeVolatility, uint64 cumulativeBinCrossed)
func (_LBPair *LBPairSession) GetOracleSampleAt(lookupTimestamp *big.Int) (struct {
	CumulativeId         uint64
	CumulativeVolatility uint64
	CumulativeBinCrossed uint64
}, error) {
	return _LBPair.Contract.GetOracleSampleAt(&_LBPair.CallOpts, lookupTimestamp)
}

// GetOracleSampleAt is a free data retrieval call binding the contract method 0x8940a16a.
//
// Solidity: function getOracleSampleAt(uint40 lookupTimestamp) view returns(uint64 cumulativeId, uint64 cumulativeVolatility, uint64 cumulativeBinCrossed)
func (_LBPair *LBPairCallerSession) GetOracleSampleAt(lookupTimestamp *big.Int) (struct {
	CumulativeId         uint64
	CumulativeVolatility uint64
	CumulativeBinCrossed uint64
}, error) {
	return _LBPair.Contract.GetOracleSampleAt(&_LBPair.CallOpts, lookupTimestamp)
}

// GetPriceFromId is a free data retrieval call binding the contract method 0x4c7cffbd.
//
// Solidity: function getPriceFromId(uint24 id) pure returns(uint256 price)
func (_LBPair *LBPairCaller) GetPriceFromId(opts *bind.CallOpts, id *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getPriceFromId", id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetPriceFromId is a free data retrieval call binding the contract method 0x4c7cffbd.
//
// Solidity: function getPriceFromId(uint24 id) pure returns(uint256 price)
func (_LBPair *LBPairSession) GetPriceFromId(id *big.Int) (*big.Int, error) {
	return _LBPair.Contract.GetPriceFromId(&_LBPair.CallOpts, id)
}

// GetPriceFromId is a free data retrieval call binding the contract method 0x4c7cffbd.
//
// Solidity: function getPriceFromId(uint24 id) pure returns(uint256 price)
func (_LBPair *LBPairCallerSession) GetPriceFromId(id *big.Int) (*big.Int, error) {
	return _LBPair.Contract.GetPriceFromId(&_LBPair.CallOpts, id)
}

// GetProtocolFees is a free data retrieval call binding the contract method 0xd8dfcea0.
//
// Solidity: function getProtocolFees() view returns(uint128 protocolFeeX, uint128 protocolFeeY)
func (_LBPair *LBPairCaller) GetProtocolFees(opts *bind.CallOpts) (struct {
	ProtocolFeeX *big.Int
	ProtocolFeeY *big.Int
}, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getProtocolFees")

	outstruct := new(struct {
		ProtocolFeeX *big.Int
		ProtocolFeeY *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.ProtocolFeeX = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.ProtocolFeeY = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetProtocolFees is a free data retrieval call binding the contract method 0xd8dfcea0.
//
// Solidity: function getProtocolFees() view returns(uint128 protocolFeeX, uint128 protocolFeeY)
func (_LBPair *LBPairSession) GetProtocolFees() (struct {
	ProtocolFeeX *big.Int
	ProtocolFeeY *big.Int
}, error) {
	return _LBPair.Contract.GetProtocolFees(&_LBPair.CallOpts)
}

// GetProtocolFees is a free data retrieval call binding the contract method 0xd8dfcea0.
//
// Solidity: function getProtocolFees() view returns(uint128 protocolFeeX, uint128 protocolFeeY)
func (_LBPair *LBPairCallerSession) GetProtocolFees() (struct {
	ProtocolFeeX *big.Int
	ProtocolFeeY *big.Int
}, error) {
	return _LBPair.Contract.GetProtocolFees(&_LBPair.CallOpts)
}

// GetReserves is a free data retrieval call binding the contract method 0x0902f1ac.
//
// Solidity: function getReserves() view returns(uint128 reserveX, uint128 reserveY)
func (_LBPair *LBPairCaller) GetReserves(opts *bind.CallOpts) (struct {
	ReserveX *big.Int
	ReserveY *big.Int
}, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getReserves")

	outstruct := new(struct {
		ReserveX *big.Int
		ReserveY *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.ReserveX = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.ReserveY = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetReserves is a free data retrieval call binding the contract method 0x0902f1ac.
//
// Solidity: function getReserves() view returns(uint128 reserveX, uint128 reserveY)
func (_LBPair *LBPairSession) GetReserves() (struct {
	ReserveX *big.Int
	ReserveY *big.Int
}, error) {
	return _LBPair.Contract.GetReserves(&_LBPair.CallOpts)
}

// GetReserves is a free data retrieval call binding the contract method 0x0902f1ac.
//
// Solidity: function getReserves() view returns(uint128 reserveX, uint128 reserveY)
func (_LBPair *LBPairCallerSession) GetReserves() (struct {
	ReserveX *big.Int
	ReserveY *big.Int
}, error) {
	return _LBPair.Contract.GetReserves(&_LBPair.CallOpts)
}

// GetStaticFeeParameters is a free data retrieval call binding the contract method 0x7ca0de30.
//
// Solidity: function getStaticFeeParameters() view returns(uint16 baseFactor, uint16 filterPeriod, uint16 decayPeriod, uint16 reductionFactor, uint24 variableFeeControl, uint16 protocolShare, uint24 maxVolatilityAccumulator)
func (_LBPair *LBPairCaller) GetStaticFeeParameters(opts *bind.CallOpts) (struct {
	BaseFactor               uint16
	FilterPeriod             uint16
	DecayPeriod              uint16
	ReductionFactor          uint16
	VariableFeeControl       *big.Int
	ProtocolShare            uint16
	MaxVolatilityAccumulator *big.Int
}, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getStaticFeeParameters")

	outstruct := new(struct {
		BaseFactor               uint16
		FilterPeriod             uint16
		DecayPeriod              uint16
		ReductionFactor          uint16
		VariableFeeControl       *big.Int
		ProtocolShare            uint16
		MaxVolatilityAccumulator *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.BaseFactor = *abi.ConvertType(out[0], new(uint16)).(*uint16)
	outstruct.FilterPeriod = *abi.ConvertType(out[1], new(uint16)).(*uint16)
	outstruct.DecayPeriod = *abi.ConvertType(out[2], new(uint16)).(*uint16)
	outstruct.ReductionFactor = *abi.ConvertType(out[3], new(uint16)).(*uint16)
	outstruct.VariableFeeControl = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.ProtocolShare = *abi.ConvertType(out[5], new(uint16)).(*uint16)
	outstruct.MaxVolatilityAccumulator = *abi.ConvertType(out[6], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetStaticFeeParameters is a free data retrieval call binding the contract method 0x7ca0de30.
//
// Solidity: function getStaticFeeParameters() view returns(uint16 baseFactor, uint16 filterPeriod, uint16 decayPeriod, uint16 reductionFactor, uint24 variableFeeControl, uint16 protocolShare, uint24 maxVolatilityAccumulator)
func (_LBPair *LBPairSession) GetStaticFeeParameters() (struct {
	BaseFactor               uint16
	FilterPeriod             uint16
	DecayPeriod              uint16
	ReductionFactor          uint16
	VariableFeeControl       *big.Int
	ProtocolShare            uint16
	MaxVolatilityAccumulator *big.Int
}, error) {
	return _LBPair.Contract.GetStaticFeeParameters(&_LBPair.CallOpts)
}

// GetStaticFeeParameters is a free data retrieval call binding the contract method 0x7ca0de30.
//
// Solidity: function getStaticFeeParameters() view returns(uint16 baseFactor, uint16 filterPeriod, uint16 decayPeriod, uint16 reductionFactor, uint24 variableFeeControl, uint16 protocolShare, uint24 maxVolatilityAccumulator)
func (_LBPair *LBPairCallerSession) GetStaticFeeParameters() (struct {
	BaseFactor               uint16
	FilterPeriod             uint16
	DecayPeriod              uint16
	ReductionFactor          uint16
	VariableFeeControl       *big.Int
	ProtocolShare            uint16
	MaxVolatilityAccumulator *big.Int
}, error) {
	return _LBPair.Contract.GetStaticFeeParameters(&_LBPair.CallOpts)
}

// GetSwapIn is a free data retrieval call binding the contract method 0xabcd7830.
//
// Solidity: function getSwapIn(uint128 amountOut, bool swapForY) view returns(uint128 amountIn, uint128 amountOutLeft, uint128 fee)
func (_LBPair *LBPairCaller) GetSwapIn(opts *bind.CallOpts, amountOut *big.Int, swapForY bool) (struct {
	AmountIn      *big.Int
	AmountOutLeft *big.Int
	Fee           *big.Int
}, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getSwapIn", amountOut, swapForY)

	outstruct := new(struct {
		AmountIn      *big.Int
		AmountOutLeft *big.Int
		Fee           *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.AmountIn = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.AmountOutLeft = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Fee = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetSwapIn is a free data retrieval call binding the contract method 0xabcd7830.
//
// Solidity: function getSwapIn(uint128 amountOut, bool swapForY) view returns(uint128 amountIn, uint128 amountOutLeft, uint128 fee)
func (_LBPair *LBPairSession) GetSwapIn(amountOut *big.Int, swapForY bool) (struct {
	AmountIn      *big.Int
	AmountOutLeft *big.Int
	Fee           *big.Int
}, error) {
	return _LBPair.Contract.GetSwapIn(&_LBPair.CallOpts, amountOut, swapForY)
}

// GetSwapIn is a free data retrieval call binding the contract method 0xabcd7830.
//
// Solidity: function getSwapIn(uint128 amountOut, bool swapForY) view returns(uint128 amountIn, uint128 amountOutLeft, uint128 fee)
func (_LBPair *LBPairCallerSession) GetSwapIn(amountOut *big.Int, swapForY bool) (struct {
	AmountIn      *big.Int
	AmountOutLeft *big.Int
	Fee           *big.Int
}, error) {
	return _LBPair.Contract.GetSwapIn(&_LBPair.CallOpts, amountOut, swapForY)
}

// GetSwapOut is a free data retrieval call binding the contract method 0xe77366f8.
//
// Solidity: function getSwapOut(uint128 amountIn, bool swapForY) view returns(uint128 amountInLeft, uint128 amountOut, uint128 fee)
func (_LBPair *LBPairCaller) GetSwapOut(opts *bind.CallOpts, amountIn *big.Int, swapForY bool) (struct {
	AmountInLeft *big.Int
	AmountOut    *big.Int
	Fee          *big.Int
}, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getSwapOut", amountIn, swapForY)

	outstruct := new(struct {
		AmountInLeft *big.Int
		AmountOut    *big.Int
		Fee          *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.AmountInLeft = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.AmountOut = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Fee = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetSwapOut is a free data retrieval call binding the contract method 0xe77366f8.
//
// Solidity: function getSwapOut(uint128 amountIn, bool swapForY) view returns(uint128 amountInLeft, uint128 amountOut, uint128 fee)
func (_LBPair *LBPairSession) GetSwapOut(amountIn *big.Int, swapForY bool) (struct {
	AmountInLeft *big.Int
	AmountOut    *big.Int
	Fee          *big.Int
}, error) {
	return _LBPair.Contract.GetSwapOut(&_LBPair.CallOpts, amountIn, swapForY)
}

// GetSwapOut is a free data retrieval call binding the contract method 0xe77366f8.
//
// Solidity: function getSwapOut(uint128 amountIn, bool swapForY) view returns(uint128 amountInLeft, uint128 amountOut, uint128 fee)
func (_LBPair *LBPairCallerSession) GetSwapOut(amountIn *big.Int, swapForY bool) (struct {
	AmountInLeft *big.Int
	AmountOut    *big.Int
	Fee          *big.Int
}, error) {
	return _LBPair.Contract.GetSwapOut(&_LBPair.CallOpts, amountIn, swapForY)
}

// GetTokenX is a free data retrieval call binding the contract method 0x05e8746d.
//
// Solidity: function getTokenX() pure returns(address tokenX)
func (_LBPair *LBPairCaller) GetTokenX(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getTokenX")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetTokenX is a free data retrieval call binding the contract method 0x05e8746d.
//
// Solidity: function getTokenX() pure returns(address tokenX)
func (_LBPair *LBPairSession) GetTokenX() (common.Address, error) {
	return _LBPair.Contract.GetTokenX(&_LBPair.CallOpts)
}

// GetTokenX is a free data retrieval call binding the contract method 0x05e8746d.
//
// Solidity: function getTokenX() pure returns(address tokenX)
func (_LBPair *LBPairCallerSession) GetTokenX() (common.Address, error) {
	return _LBPair.Contract.GetTokenX(&_LBPair.CallOpts)
}

// GetTokenY is a free data retrieval call binding the contract method 0xda10610c.
//
// Solidity: function getTokenY() pure returns(address tokenY)
func (_LBPair *LBPairCaller) GetTokenY(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getTokenY")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetTokenY is a free data retrieval call binding the contract method 0xda10610c.
//
// Solidity: function getTokenY() pure returns(address tokenY)
func (_LBPair *LBPairSession) GetTokenY() (common.Address, error) {
	return _LBPair.Contract.GetTokenY(&_LBPair.CallOpts)
}

// GetTokenY is a free data retrieval call binding the contract method 0xda10610c.
//
// Solidity: function getTokenY() pure returns(address tokenY)
func (_LBPair *LBPairCallerSession) GetTokenY() (common.Address, error) {
	return _LBPair.Contract.GetTokenY(&_LBPair.CallOpts)
}

// GetVariableFeeParameters is a free data retrieval call binding the contract method 0x8d7024e5.
//
// Solidity: function getVariableFeeParameters() view returns(uint24 volatilityAccumulator, uint24 volatilityReference, uint24 idReference, uint40 timeOfLastUpdate)
func (_LBPair *LBPairCaller) GetVariableFeeParameters(opts *bind.CallOpts) (struct {
	VolatilityAccumulator *big.Int
	VolatilityReference   *big.Int
	IdReference           *big.Int
	TimeOfLastUpdate      *big.Int
}, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "getVariableFeeParameters")

	outstruct := new(struct {
		VolatilityAccumulator *big.Int
		VolatilityReference   *big.Int
		IdReference           *big.Int
		TimeOfLastUpdate      *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.VolatilityAccumulator = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.VolatilityReference = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.IdReference = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.TimeOfLastUpdate = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetVariableFeeParameters is a free data retrieval call binding the contract method 0x8d7024e5.
//
// Solidity: function getVariableFeeParameters() view returns(uint24 volatilityAccumulator, uint24 volatilityReference, uint24 idReference, uint40 timeOfLastUpdate)
func (_LBPair *LBPairSession) GetVariableFeeParameters() (struct {
	VolatilityAccumulator *big.Int
	VolatilityReference   *big.Int
	IdReference           *big.Int
	TimeOfLastUpdate      *big.Int
}, error) {
	return _LBPair.Contract.GetVariableFeeParameters(&_LBPair.CallOpts)
}

// GetVariableFeeParameters is a free data retrieval call binding the contract method 0x8d7024e5.
//
// Solidity: function getVariableFeeParameters() view returns(uint24 volatilityAccumulator, uint24 volatilityReference, uint24 idReference, uint40 timeOfLastUpdate)
func (_LBPair *LBPairCallerSession) GetVariableFeeParameters() (struct {
	VolatilityAccumulator *big.Int
	VolatilityReference   *big.Int
	IdReference           *big.Int
	TimeOfLastUpdate      *big.Int
}, error) {
	return _LBPair.Contract.GetVariableFeeParameters(&_LBPair.CallOpts)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address spender) view returns(bool)
func (_LBPair *LBPairCaller) IsApprovedForAll(opts *bind.CallOpts, owner common.Address, spender common.Address) (bool, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "isApprovedForAll", owner, spender)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address spender) view returns(bool)
func (_LBPair *LBPairSession) IsApprovedForAll(owner common.Address, spender common.Address) (bool, error) {
	return _LBPair.Contract.IsApprovedForAll(&_LBPair.CallOpts, owner, spender)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address spender) view returns(bool)
func (_LBPair *LBPairCallerSession) IsApprovedForAll(owner common.Address, spender common.Address) (bool, error) {
	return _LBPair.Contract.IsApprovedForAll(&_LBPair.CallOpts, owner, spender)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_LBPair *LBPairCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_LBPair *LBPairSession) Name() (string, error) {
	return _LBPair.Contract.Name(&_LBPair.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_LBPair *LBPairCallerSession) Name() (string, error) {
	return _LBPair.Contract.Name(&_LBPair.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_LBPair *LBPairCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_LBPair *LBPairSession) Symbol() (string, error) {
	return _LBPair.Contract.Symbol(&_LBPair.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_LBPair *LBPairCallerSession) Symbol() (string, error) {
	return _LBPair.Contract.Symbol(&_LBPair.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0xbd85b039.
//
// Solidity: function totalSupply(uint256 id) view returns(uint256)
func (_LBPair *LBPairCaller) TotalSupply(opts *bind.CallOpts, id *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _LBPair.contract.Call(opts, &out, "totalSupply", id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0xbd85b039.
//
// Solidity: function totalSupply(uint256 id) view returns(uint256)
func (_LBPair *LBPairSession) TotalSupply(id *big.Int) (*big.Int, error) {
	return _LBPair.Contract.TotalSupply(&_LBPair.CallOpts, id)
}

// TotalSupply is a free data retrieval call binding the contract method 0xbd85b039.
//
// Solidity: function totalSupply(uint256 id) view returns(uint256)
func (_LBPair *LBPairCallerSession) TotalSupply(id *big.Int) (*big.Int, error) {
	return _LBPair.Contract.TotalSupply(&_LBPair.CallOpts, id)
}

// LBPairApprovalForAllIterator is returned from FilterApprovalForAll and is used to iterate over the raw logs and unpacked data for ApprovalForAll events raised by the LBPair contract.
type LBPairApprovalForAllIterator struct {
	Event *LBPairApprovalForAll // Event containing the contract specifics and raw log

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
func (it *LBPairApprovalForAllIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairApprovalForAll)
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
		it.Event = new(LBPairApprovalForAll)
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
func (it *LBPairApprovalForAllIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairApprovalForAllIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairApprovalForAll represents a ApprovalForAll event raised by the LBPair contract.
type LBPairApprovalForAll struct {
	Account  common.Address
	Sender   common.Address
	Approved bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApprovalForAll is a free log retrieval operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed account, address indexed sender, bool approved)
func (_LBPair *LBPairFilterer) FilterApprovalForAll(opts *bind.FilterOpts, account []common.Address, sender []common.Address) (*LBPairApprovalForAllIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "ApprovalForAll", accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &LBPairApprovalForAllIterator{contract: _LBPair.contract, event: "ApprovalForAll", logs: logs, sub: sub}, nil
}

// WatchApprovalForAll is a free log subscription operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed account, address indexed sender, bool approved)
func (_LBPair *LBPairFilterer) WatchApprovalForAll(opts *bind.WatchOpts, sink chan<- *LBPairApprovalForAll, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "ApprovalForAll", accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairApprovalForAll)
				if err := _LBPair.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
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

// ParseApprovalForAll is a log parse operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed account, address indexed sender, bool approved)
func (_LBPair *LBPairFilterer) ParseApprovalForAll(log types.Log) (*LBPairApprovalForAll, error) {
	event := new(LBPairApprovalForAll)
	if err := _LBPair.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairCollectedProtocolFeesIterator is returned from FilterCollectedProtocolFees and is used to iterate over the raw logs and unpacked data for CollectedProtocolFees events raised by the LBPair contract.
type LBPairCollectedProtocolFeesIterator struct {
	Event *LBPairCollectedProtocolFees // Event containing the contract specifics and raw log

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
func (it *LBPairCollectedProtocolFeesIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairCollectedProtocolFees)
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
		it.Event = new(LBPairCollectedProtocolFees)
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
func (it *LBPairCollectedProtocolFeesIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairCollectedProtocolFeesIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairCollectedProtocolFees represents a CollectedProtocolFees event raised by the LBPair contract.
type LBPairCollectedProtocolFees struct {
	FeeRecipient common.Address
	ProtocolFees [32]byte
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterCollectedProtocolFees is a free log retrieval operation binding the contract event 0x3f41a5ddc53701cc7db577ade4f1fca9838a8ec0b5ea50b9f0f5d17bc4554e32.
//
// Solidity: event CollectedProtocolFees(address indexed feeRecipient, bytes32 protocolFees)
func (_LBPair *LBPairFilterer) FilterCollectedProtocolFees(opts *bind.FilterOpts, feeRecipient []common.Address) (*LBPairCollectedProtocolFeesIterator, error) {

	var feeRecipientRule []interface{}
	for _, feeRecipientItem := range feeRecipient {
		feeRecipientRule = append(feeRecipientRule, feeRecipientItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "CollectedProtocolFees", feeRecipientRule)
	if err != nil {
		return nil, err
	}
	return &LBPairCollectedProtocolFeesIterator{contract: _LBPair.contract, event: "CollectedProtocolFees", logs: logs, sub: sub}, nil
}

// WatchCollectedProtocolFees is a free log subscription operation binding the contract event 0x3f41a5ddc53701cc7db577ade4f1fca9838a8ec0b5ea50b9f0f5d17bc4554e32.
//
// Solidity: event CollectedProtocolFees(address indexed feeRecipient, bytes32 protocolFees)
func (_LBPair *LBPairFilterer) WatchCollectedProtocolFees(opts *bind.WatchOpts, sink chan<- *LBPairCollectedProtocolFees, feeRecipient []common.Address) (event.Subscription, error) {

	var feeRecipientRule []interface{}
	for _, feeRecipientItem := range feeRecipient {
		feeRecipientRule = append(feeRecipientRule, feeRecipientItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "CollectedProtocolFees", feeRecipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairCollectedProtocolFees)
				if err := _LBPair.contract.UnpackLog(event, "CollectedProtocolFees", log); err != nil {
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

// ParseCollectedProtocolFees is a log parse operation binding the contract event 0x3f41a5ddc53701cc7db577ade4f1fca9838a8ec0b5ea50b9f0f5d17bc4554e32.
//
// Solidity: event CollectedProtocolFees(address indexed feeRecipient, bytes32 protocolFees)
func (_LBPair *LBPairFilterer) ParseCollectedProtocolFees(log types.Log) (*LBPairCollectedProtocolFees, error) {
	event := new(LBPairCollectedProtocolFees)
	if err := _LBPair.contract.UnpackLog(event, "CollectedProtocolFees", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairCompositionFeesIterator is returned from FilterCompositionFees and is used to iterate over the raw logs and unpacked data for CompositionFees events raised by the LBPair contract.
type LBPairCompositionFeesIterator struct {
	Event *LBPairCompositionFees // Event containing the contract specifics and raw log

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
func (it *LBPairCompositionFeesIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairCompositionFees)
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
		it.Event = new(LBPairCompositionFees)
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
func (it *LBPairCompositionFeesIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairCompositionFeesIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairCompositionFees represents a CompositionFees event raised by the LBPair contract.
type LBPairCompositionFees struct {
	Sender       common.Address
	Id           *big.Int
	TotalFees    [32]byte
	ProtocolFees [32]byte
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterCompositionFees is a free log retrieval operation binding the contract event 0x3f0b46725027bb418b2005f4683538eccdbcdf1de2b8649a29dbd9c507d16ff4.
//
// Solidity: event CompositionFees(address indexed sender, uint24 id, bytes32 totalFees, bytes32 protocolFees)
func (_LBPair *LBPairFilterer) FilterCompositionFees(opts *bind.FilterOpts, sender []common.Address) (*LBPairCompositionFeesIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "CompositionFees", senderRule)
	if err != nil {
		return nil, err
	}
	return &LBPairCompositionFeesIterator{contract: _LBPair.contract, event: "CompositionFees", logs: logs, sub: sub}, nil
}

// WatchCompositionFees is a free log subscription operation binding the contract event 0x3f0b46725027bb418b2005f4683538eccdbcdf1de2b8649a29dbd9c507d16ff4.
//
// Solidity: event CompositionFees(address indexed sender, uint24 id, bytes32 totalFees, bytes32 protocolFees)
func (_LBPair *LBPairFilterer) WatchCompositionFees(opts *bind.WatchOpts, sink chan<- *LBPairCompositionFees, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "CompositionFees", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairCompositionFees)
				if err := _LBPair.contract.UnpackLog(event, "CompositionFees", log); err != nil {
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

// ParseCompositionFees is a log parse operation binding the contract event 0x3f0b46725027bb418b2005f4683538eccdbcdf1de2b8649a29dbd9c507d16ff4.
//
// Solidity: event CompositionFees(address indexed sender, uint24 id, bytes32 totalFees, bytes32 protocolFees)
func (_LBPair *LBPairFilterer) ParseCompositionFees(log types.Log) (*LBPairCompositionFees, error) {
	event := new(LBPairCompositionFees)
	if err := _LBPair.contract.UnpackLog(event, "CompositionFees", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairDepositedToBinsIterator is returned from FilterDepositedToBins and is used to iterate over the raw logs and unpacked data for DepositedToBins events raised by the LBPair contract.
type LBPairDepositedToBinsIterator struct {
	Event *LBPairDepositedToBins // Event containing the contract specifics and raw log

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
func (it *LBPairDepositedToBinsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairDepositedToBins)
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
		it.Event = new(LBPairDepositedToBins)
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
func (it *LBPairDepositedToBinsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairDepositedToBinsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairDepositedToBins represents a DepositedToBins event raised by the LBPair contract.
type LBPairDepositedToBins struct {
	Sender  common.Address
	To      common.Address
	Ids     []*big.Int
	Amounts [][32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterDepositedToBins is a free log retrieval operation binding the contract event 0x87f1f9dcf5e8089a3e00811b6a008d8f30293a3da878cb1fe8c90ca376402f8a.
//
// Solidity: event DepositedToBins(address indexed sender, address indexed to, uint256[] ids, bytes32[] amounts)
func (_LBPair *LBPairFilterer) FilterDepositedToBins(opts *bind.FilterOpts, sender []common.Address, to []common.Address) (*LBPairDepositedToBinsIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "DepositedToBins", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return &LBPairDepositedToBinsIterator{contract: _LBPair.contract, event: "DepositedToBins", logs: logs, sub: sub}, nil
}

// WatchDepositedToBins is a free log subscription operation binding the contract event 0x87f1f9dcf5e8089a3e00811b6a008d8f30293a3da878cb1fe8c90ca376402f8a.
//
// Solidity: event DepositedToBins(address indexed sender, address indexed to, uint256[] ids, bytes32[] amounts)
func (_LBPair *LBPairFilterer) WatchDepositedToBins(opts *bind.WatchOpts, sink chan<- *LBPairDepositedToBins, sender []common.Address, to []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "DepositedToBins", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairDepositedToBins)
				if err := _LBPair.contract.UnpackLog(event, "DepositedToBins", log); err != nil {
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

// ParseDepositedToBins is a log parse operation binding the contract event 0x87f1f9dcf5e8089a3e00811b6a008d8f30293a3da878cb1fe8c90ca376402f8a.
//
// Solidity: event DepositedToBins(address indexed sender, address indexed to, uint256[] ids, bytes32[] amounts)
func (_LBPair *LBPairFilterer) ParseDepositedToBins(log types.Log) (*LBPairDepositedToBins, error) {
	event := new(LBPairDepositedToBins)
	if err := _LBPair.contract.UnpackLog(event, "DepositedToBins", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairDepositedToBins0Iterator is returned from FilterDepositedToBins0 and is used to iterate over the raw logs and unpacked data for DepositedToBins0 events raised by the LBPair contract.
type LBPairDepositedToBins0Iterator struct {
	Event *LBPairDepositedToBins0 // Event containing the contract specifics and raw log

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
func (it *LBPairDepositedToBins0Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairDepositedToBins0)
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
		it.Event = new(LBPairDepositedToBins0)
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
func (it *LBPairDepositedToBins0Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairDepositedToBins0Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairDepositedToBins0 represents a DepositedToBins0 event raised by the LBPair contract.
type LBPairDepositedToBins0 struct {
	Sender    common.Address
	To        common.Address
	Ids       []*big.Int
	Amounts   [][32]byte
	PriceData ILBPairPriceData
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterDepositedToBins0 is a free log retrieval operation binding the contract event 0xd1940621deed4d2534f11049d437a78045133333c214f098e79d0fd6693125e2.
//
// Solidity: event DepositedToBins(address indexed sender, address indexed to, uint256[] ids, bytes32[] amounts, (uint256,uint256,uint256) priceData)
func (_LBPair *LBPairFilterer) FilterDepositedToBins0(opts *bind.FilterOpts, sender []common.Address, to []common.Address) (*LBPairDepositedToBins0Iterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "DepositedToBins0", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return &LBPairDepositedToBins0Iterator{contract: _LBPair.contract, event: "DepositedToBins0", logs: logs, sub: sub}, nil
}

// WatchDepositedToBins0 is a free log subscription operation binding the contract event 0xd1940621deed4d2534f11049d437a78045133333c214f098e79d0fd6693125e2.
//
// Solidity: event DepositedToBins(address indexed sender, address indexed to, uint256[] ids, bytes32[] amounts, (uint256,uint256,uint256) priceData)
func (_LBPair *LBPairFilterer) WatchDepositedToBins0(opts *bind.WatchOpts, sink chan<- *LBPairDepositedToBins0, sender []common.Address, to []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "DepositedToBins0", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairDepositedToBins0)
				if err := _LBPair.contract.UnpackLog(event, "DepositedToBins0", log); err != nil {
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

// ParseDepositedToBins0 is a log parse operation binding the contract event 0xd1940621deed4d2534f11049d437a78045133333c214f098e79d0fd6693125e2.
//
// Solidity: event DepositedToBins(address indexed sender, address indexed to, uint256[] ids, bytes32[] amounts, (uint256,uint256,uint256) priceData)
func (_LBPair *LBPairFilterer) ParseDepositedToBins0(log types.Log) (*LBPairDepositedToBins0, error) {
	event := new(LBPairDepositedToBins0)
	if err := _LBPair.contract.UnpackLog(event, "DepositedToBins0", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairFlashLoanIterator is returned from FilterFlashLoan and is used to iterate over the raw logs and unpacked data for FlashLoan events raised by the LBPair contract.
type LBPairFlashLoanIterator struct {
	Event *LBPairFlashLoan // Event containing the contract specifics and raw log

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
func (it *LBPairFlashLoanIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairFlashLoan)
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
		it.Event = new(LBPairFlashLoan)
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
func (it *LBPairFlashLoanIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairFlashLoanIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairFlashLoan represents a FlashLoan event raised by the LBPair contract.
type LBPairFlashLoan struct {
	Sender       common.Address
	Receiver     common.Address
	ActiveId     *big.Int
	Amounts      [32]byte
	TotalFees    [32]byte
	ProtocolFees [32]byte
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterFlashLoan is a free log retrieval operation binding the contract event 0xd126bd9d94daca8e55ffd8283fac05394aec8326c6b1639e1e8a445fbe8bbc7d.
//
// Solidity: event FlashLoan(address indexed sender, address indexed receiver, uint24 activeId, bytes32 amounts, bytes32 totalFees, bytes32 protocolFees)
func (_LBPair *LBPairFilterer) FilterFlashLoan(opts *bind.FilterOpts, sender []common.Address, receiver []common.Address) (*LBPairFlashLoanIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var receiverRule []interface{}
	for _, receiverItem := range receiver {
		receiverRule = append(receiverRule, receiverItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "FlashLoan", senderRule, receiverRule)
	if err != nil {
		return nil, err
	}
	return &LBPairFlashLoanIterator{contract: _LBPair.contract, event: "FlashLoan", logs: logs, sub: sub}, nil
}

// WatchFlashLoan is a free log subscription operation binding the contract event 0xd126bd9d94daca8e55ffd8283fac05394aec8326c6b1639e1e8a445fbe8bbc7d.
//
// Solidity: event FlashLoan(address indexed sender, address indexed receiver, uint24 activeId, bytes32 amounts, bytes32 totalFees, bytes32 protocolFees)
func (_LBPair *LBPairFilterer) WatchFlashLoan(opts *bind.WatchOpts, sink chan<- *LBPairFlashLoan, sender []common.Address, receiver []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var receiverRule []interface{}
	for _, receiverItem := range receiver {
		receiverRule = append(receiverRule, receiverItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "FlashLoan", senderRule, receiverRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairFlashLoan)
				if err := _LBPair.contract.UnpackLog(event, "FlashLoan", log); err != nil {
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

// ParseFlashLoan is a log parse operation binding the contract event 0xd126bd9d94daca8e55ffd8283fac05394aec8326c6b1639e1e8a445fbe8bbc7d.
//
// Solidity: event FlashLoan(address indexed sender, address indexed receiver, uint24 activeId, bytes32 amounts, bytes32 totalFees, bytes32 protocolFees)
func (_LBPair *LBPairFilterer) ParseFlashLoan(log types.Log) (*LBPairFlashLoan, error) {
	event := new(LBPairFlashLoan)
	if err := _LBPair.contract.UnpackLog(event, "FlashLoan", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairFlashLoan0Iterator is returned from FilterFlashLoan0 and is used to iterate over the raw logs and unpacked data for FlashLoan0 events raised by the LBPair contract.
type LBPairFlashLoan0Iterator struct {
	Event *LBPairFlashLoan0 // Event containing the contract specifics and raw log

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
func (it *LBPairFlashLoan0Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairFlashLoan0)
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
		it.Event = new(LBPairFlashLoan0)
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
func (it *LBPairFlashLoan0Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairFlashLoan0Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairFlashLoan0 represents a FlashLoan0 event raised by the LBPair contract.
type LBPairFlashLoan0 struct {
	Sender       common.Address
	Receiver     common.Address
	ActiveId     *big.Int
	Amounts      [32]byte
	TotalFees    [32]byte
	ProtocolFees [32]byte
	PriceData    ILBPairPriceData
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterFlashLoan0 is a free log retrieval operation binding the contract event 0x4982622959c7520f0d1d70ab83b6da3fcd99f7a5f0baee8a79dee3590fa10ddc.
//
// Solidity: event FlashLoan(address indexed sender, address indexed receiver, uint24 activeId, bytes32 amounts, bytes32 totalFees, bytes32 protocolFees, (uint256,uint256,uint256) priceData)
func (_LBPair *LBPairFilterer) FilterFlashLoan0(opts *bind.FilterOpts, sender []common.Address, receiver []common.Address) (*LBPairFlashLoan0Iterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var receiverRule []interface{}
	for _, receiverItem := range receiver {
		receiverRule = append(receiverRule, receiverItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "FlashLoan0", senderRule, receiverRule)
	if err != nil {
		return nil, err
	}
	return &LBPairFlashLoan0Iterator{contract: _LBPair.contract, event: "FlashLoan0", logs: logs, sub: sub}, nil
}

// WatchFlashLoan0 is a free log subscription operation binding the contract event 0x4982622959c7520f0d1d70ab83b6da3fcd99f7a5f0baee8a79dee3590fa10ddc.
//
// Solidity: event FlashLoan(address indexed sender, address indexed receiver, uint24 activeId, bytes32 amounts, bytes32 totalFees, bytes32 protocolFees, (uint256,uint256,uint256) priceData)
func (_LBPair *LBPairFilterer) WatchFlashLoan0(opts *bind.WatchOpts, sink chan<- *LBPairFlashLoan0, sender []common.Address, receiver []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var receiverRule []interface{}
	for _, receiverItem := range receiver {
		receiverRule = append(receiverRule, receiverItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "FlashLoan0", senderRule, receiverRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairFlashLoan0)
				if err := _LBPair.contract.UnpackLog(event, "FlashLoan0", log); err != nil {
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

// ParseFlashLoan0 is a log parse operation binding the contract event 0x4982622959c7520f0d1d70ab83b6da3fcd99f7a5f0baee8a79dee3590fa10ddc.
//
// Solidity: event FlashLoan(address indexed sender, address indexed receiver, uint24 activeId, bytes32 amounts, bytes32 totalFees, bytes32 protocolFees, (uint256,uint256,uint256) priceData)
func (_LBPair *LBPairFilterer) ParseFlashLoan0(log types.Log) (*LBPairFlashLoan0, error) {
	event := new(LBPairFlashLoan0)
	if err := _LBPair.contract.UnpackLog(event, "FlashLoan0", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairForcedDecayIterator is returned from FilterForcedDecay and is used to iterate over the raw logs and unpacked data for ForcedDecay events raised by the LBPair contract.
type LBPairForcedDecayIterator struct {
	Event *LBPairForcedDecay // Event containing the contract specifics and raw log

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
func (it *LBPairForcedDecayIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairForcedDecay)
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
		it.Event = new(LBPairForcedDecay)
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
func (it *LBPairForcedDecayIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairForcedDecayIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairForcedDecay represents a ForcedDecay event raised by the LBPair contract.
type LBPairForcedDecay struct {
	Sender              common.Address
	IdReference         *big.Int
	VolatilityReference *big.Int
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterForcedDecay is a free log retrieval operation binding the contract event 0x282afaeeae84c1d85ad1424a3aa2ddbedaeefca3b1e53d889d15265fe44db7fc.
//
// Solidity: event ForcedDecay(address indexed sender, uint24 idReference, uint24 volatilityReference)
func (_LBPair *LBPairFilterer) FilterForcedDecay(opts *bind.FilterOpts, sender []common.Address) (*LBPairForcedDecayIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "ForcedDecay", senderRule)
	if err != nil {
		return nil, err
	}
	return &LBPairForcedDecayIterator{contract: _LBPair.contract, event: "ForcedDecay", logs: logs, sub: sub}, nil
}

// WatchForcedDecay is a free log subscription operation binding the contract event 0x282afaeeae84c1d85ad1424a3aa2ddbedaeefca3b1e53d889d15265fe44db7fc.
//
// Solidity: event ForcedDecay(address indexed sender, uint24 idReference, uint24 volatilityReference)
func (_LBPair *LBPairFilterer) WatchForcedDecay(opts *bind.WatchOpts, sink chan<- *LBPairForcedDecay, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "ForcedDecay", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairForcedDecay)
				if err := _LBPair.contract.UnpackLog(event, "ForcedDecay", log); err != nil {
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

// ParseForcedDecay is a log parse operation binding the contract event 0x282afaeeae84c1d85ad1424a3aa2ddbedaeefca3b1e53d889d15265fe44db7fc.
//
// Solidity: event ForcedDecay(address indexed sender, uint24 idReference, uint24 volatilityReference)
func (_LBPair *LBPairFilterer) ParseForcedDecay(log types.Log) (*LBPairForcedDecay, error) {
	event := new(LBPairForcedDecay)
	if err := _LBPair.contract.UnpackLog(event, "ForcedDecay", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairOracleLengthIncreasedIterator is returned from FilterOracleLengthIncreased and is used to iterate over the raw logs and unpacked data for OracleLengthIncreased events raised by the LBPair contract.
type LBPairOracleLengthIncreasedIterator struct {
	Event *LBPairOracleLengthIncreased // Event containing the contract specifics and raw log

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
func (it *LBPairOracleLengthIncreasedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairOracleLengthIncreased)
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
		it.Event = new(LBPairOracleLengthIncreased)
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
func (it *LBPairOracleLengthIncreasedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairOracleLengthIncreasedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairOracleLengthIncreased represents a OracleLengthIncreased event raised by the LBPair contract.
type LBPairOracleLengthIncreased struct {
	Sender       common.Address
	OracleLength uint16
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterOracleLengthIncreased is a free log retrieval operation binding the contract event 0xc975541e72d695746a43ba65745d79963a23082637c8f4609354d9bcf70194d6.
//
// Solidity: event OracleLengthIncreased(address indexed sender, uint16 oracleLength)
func (_LBPair *LBPairFilterer) FilterOracleLengthIncreased(opts *bind.FilterOpts, sender []common.Address) (*LBPairOracleLengthIncreasedIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "OracleLengthIncreased", senderRule)
	if err != nil {
		return nil, err
	}
	return &LBPairOracleLengthIncreasedIterator{contract: _LBPair.contract, event: "OracleLengthIncreased", logs: logs, sub: sub}, nil
}

// WatchOracleLengthIncreased is a free log subscription operation binding the contract event 0xc975541e72d695746a43ba65745d79963a23082637c8f4609354d9bcf70194d6.
//
// Solidity: event OracleLengthIncreased(address indexed sender, uint16 oracleLength)
func (_LBPair *LBPairFilterer) WatchOracleLengthIncreased(opts *bind.WatchOpts, sink chan<- *LBPairOracleLengthIncreased, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "OracleLengthIncreased", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairOracleLengthIncreased)
				if err := _LBPair.contract.UnpackLog(event, "OracleLengthIncreased", log); err != nil {
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

// ParseOracleLengthIncreased is a log parse operation binding the contract event 0xc975541e72d695746a43ba65745d79963a23082637c8f4609354d9bcf70194d6.
//
// Solidity: event OracleLengthIncreased(address indexed sender, uint16 oracleLength)
func (_LBPair *LBPairFilterer) ParseOracleLengthIncreased(log types.Log) (*LBPairOracleLengthIncreased, error) {
	event := new(LBPairOracleLengthIncreased)
	if err := _LBPair.contract.UnpackLog(event, "OracleLengthIncreased", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairStaticFeeParametersSetIterator is returned from FilterStaticFeeParametersSet and is used to iterate over the raw logs and unpacked data for StaticFeeParametersSet events raised by the LBPair contract.
type LBPairStaticFeeParametersSetIterator struct {
	Event *LBPairStaticFeeParametersSet // Event containing the contract specifics and raw log

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
func (it *LBPairStaticFeeParametersSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairStaticFeeParametersSet)
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
		it.Event = new(LBPairStaticFeeParametersSet)
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
func (it *LBPairStaticFeeParametersSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairStaticFeeParametersSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairStaticFeeParametersSet represents a StaticFeeParametersSet event raised by the LBPair contract.
type LBPairStaticFeeParametersSet struct {
	Sender                   common.Address
	BaseFactor               uint16
	FilterPeriod             uint16
	DecayPeriod              uint16
	ReductionFactor          uint16
	VariableFeeControl       *big.Int
	ProtocolShare            uint16
	MaxVolatilityAccumulator *big.Int
	Raw                      types.Log // Blockchain specific contextual infos
}

// FilterStaticFeeParametersSet is a free log retrieval operation binding the contract event 0xd09e5ddc721ff14c5c1e66a305cbba1fd70b82c5232bc391aad6f55e62e4b046.
//
// Solidity: event StaticFeeParametersSet(address indexed sender, uint16 baseFactor, uint16 filterPeriod, uint16 decayPeriod, uint16 reductionFactor, uint24 variableFeeControl, uint16 protocolShare, uint24 maxVolatilityAccumulator)
func (_LBPair *LBPairFilterer) FilterStaticFeeParametersSet(opts *bind.FilterOpts, sender []common.Address) (*LBPairStaticFeeParametersSetIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "StaticFeeParametersSet", senderRule)
	if err != nil {
		return nil, err
	}
	return &LBPairStaticFeeParametersSetIterator{contract: _LBPair.contract, event: "StaticFeeParametersSet", logs: logs, sub: sub}, nil
}

// WatchStaticFeeParametersSet is a free log subscription operation binding the contract event 0xd09e5ddc721ff14c5c1e66a305cbba1fd70b82c5232bc391aad6f55e62e4b046.
//
// Solidity: event StaticFeeParametersSet(address indexed sender, uint16 baseFactor, uint16 filterPeriod, uint16 decayPeriod, uint16 reductionFactor, uint24 variableFeeControl, uint16 protocolShare, uint24 maxVolatilityAccumulator)
func (_LBPair *LBPairFilterer) WatchStaticFeeParametersSet(opts *bind.WatchOpts, sink chan<- *LBPairStaticFeeParametersSet, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "StaticFeeParametersSet", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairStaticFeeParametersSet)
				if err := _LBPair.contract.UnpackLog(event, "StaticFeeParametersSet", log); err != nil {
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

// ParseStaticFeeParametersSet is a log parse operation binding the contract event 0xd09e5ddc721ff14c5c1e66a305cbba1fd70b82c5232bc391aad6f55e62e4b046.
//
// Solidity: event StaticFeeParametersSet(address indexed sender, uint16 baseFactor, uint16 filterPeriod, uint16 decayPeriod, uint16 reductionFactor, uint24 variableFeeControl, uint16 protocolShare, uint24 maxVolatilityAccumulator)
func (_LBPair *LBPairFilterer) ParseStaticFeeParametersSet(log types.Log) (*LBPairStaticFeeParametersSet, error) {
	event := new(LBPairStaticFeeParametersSet)
	if err := _LBPair.contract.UnpackLog(event, "StaticFeeParametersSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairSwapIterator is returned from FilterSwap and is used to iterate over the raw logs and unpacked data for Swap events raised by the LBPair contract.
type LBPairSwapIterator struct {
	Event *LBPairSwap // Event containing the contract specifics and raw log

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
func (it *LBPairSwapIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairSwap)
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
		it.Event = new(LBPairSwap)
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
func (it *LBPairSwapIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairSwapIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairSwap represents a Swap event raised by the LBPair contract.
type LBPairSwap struct {
	Sender                common.Address
	To                    common.Address
	Id                    *big.Int
	AmountsIn             [32]byte
	AmountsOut            [32]byte
	VolatilityAccumulator *big.Int
	TotalFees             [32]byte
	ProtocolFees          [32]byte
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterSwap is a free log retrieval operation binding the contract event 0xad7d6f97abf51ce18e17a38f4d70e975be9c0708474987bb3e26ad21bd93ca70.
//
// Solidity: event Swap(address indexed sender, address indexed to, uint24 id, bytes32 amountsIn, bytes32 amountsOut, uint24 volatilityAccumulator, bytes32 totalFees, bytes32 protocolFees)
func (_LBPair *LBPairFilterer) FilterSwap(opts *bind.FilterOpts, sender []common.Address, to []common.Address) (*LBPairSwapIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "Swap", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return &LBPairSwapIterator{contract: _LBPair.contract, event: "Swap", logs: logs, sub: sub}, nil
}

// WatchSwap is a free log subscription operation binding the contract event 0xad7d6f97abf51ce18e17a38f4d70e975be9c0708474987bb3e26ad21bd93ca70.
//
// Solidity: event Swap(address indexed sender, address indexed to, uint24 id, bytes32 amountsIn, bytes32 amountsOut, uint24 volatilityAccumulator, bytes32 totalFees, bytes32 protocolFees)
func (_LBPair *LBPairFilterer) WatchSwap(opts *bind.WatchOpts, sink chan<- *LBPairSwap, sender []common.Address, to []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "Swap", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairSwap)
				if err := _LBPair.contract.UnpackLog(event, "Swap", log); err != nil {
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

// ParseSwap is a log parse operation binding the contract event 0xad7d6f97abf51ce18e17a38f4d70e975be9c0708474987bb3e26ad21bd93ca70.
//
// Solidity: event Swap(address indexed sender, address indexed to, uint24 id, bytes32 amountsIn, bytes32 amountsOut, uint24 volatilityAccumulator, bytes32 totalFees, bytes32 protocolFees)
func (_LBPair *LBPairFilterer) ParseSwap(log types.Log) (*LBPairSwap, error) {
	event := new(LBPairSwap)
	if err := _LBPair.contract.UnpackLog(event, "Swap", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairSwap0Iterator is returned from FilterSwap0 and is used to iterate over the raw logs and unpacked data for Swap0 events raised by the LBPair contract.
type LBPairSwap0Iterator struct {
	Event *LBPairSwap0 // Event containing the contract specifics and raw log

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
func (it *LBPairSwap0Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairSwap0)
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
		it.Event = new(LBPairSwap0)
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
func (it *LBPairSwap0Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairSwap0Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairSwap0 represents a Swap0 event raised by the LBPair contract.
type LBPairSwap0 struct {
	Sender                common.Address
	To                    common.Address
	Id                    *big.Int
	AmountsIn             [32]byte
	AmountsOut            [32]byte
	VolatilityAccumulator *big.Int
	TotalFees             [32]byte
	ProtocolFees          [32]byte
	PriceData             ILBPairPriceData
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterSwap0 is a free log retrieval operation binding the contract event 0x8b7ff4e98ef6b513fa9b897cabe874443711a7c76b08c2f2346b3316df396351.
//
// Solidity: event Swap(address indexed sender, address indexed to, uint24 id, bytes32 amountsIn, bytes32 amountsOut, uint24 volatilityAccumulator, bytes32 totalFees, bytes32 protocolFees, (uint256,uint256,uint256) priceData)
func (_LBPair *LBPairFilterer) FilterSwap0(opts *bind.FilterOpts, sender []common.Address, to []common.Address) (*LBPairSwap0Iterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "Swap0", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return &LBPairSwap0Iterator{contract: _LBPair.contract, event: "Swap0", logs: logs, sub: sub}, nil
}

// WatchSwap0 is a free log subscription operation binding the contract event 0x8b7ff4e98ef6b513fa9b897cabe874443711a7c76b08c2f2346b3316df396351.
//
// Solidity: event Swap(address indexed sender, address indexed to, uint24 id, bytes32 amountsIn, bytes32 amountsOut, uint24 volatilityAccumulator, bytes32 totalFees, bytes32 protocolFees, (uint256,uint256,uint256) priceData)
func (_LBPair *LBPairFilterer) WatchSwap0(opts *bind.WatchOpts, sink chan<- *LBPairSwap0, sender []common.Address, to []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "Swap0", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairSwap0)
				if err := _LBPair.contract.UnpackLog(event, "Swap0", log); err != nil {
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

// ParseSwap0 is a log parse operation binding the contract event 0x8b7ff4e98ef6b513fa9b897cabe874443711a7c76b08c2f2346b3316df396351.
//
// Solidity: event Swap(address indexed sender, address indexed to, uint24 id, bytes32 amountsIn, bytes32 amountsOut, uint24 volatilityAccumulator, bytes32 totalFees, bytes32 protocolFees, (uint256,uint256,uint256) priceData)
func (_LBPair *LBPairFilterer) ParseSwap0(log types.Log) (*LBPairSwap0, error) {
	event := new(LBPairSwap0)
	if err := _LBPair.contract.UnpackLog(event, "Swap0", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairTransferBatchIterator is returned from FilterTransferBatch and is used to iterate over the raw logs and unpacked data for TransferBatch events raised by the LBPair contract.
type LBPairTransferBatchIterator struct {
	Event *LBPairTransferBatch // Event containing the contract specifics and raw log

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
func (it *LBPairTransferBatchIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairTransferBatch)
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
		it.Event = new(LBPairTransferBatch)
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
func (it *LBPairTransferBatchIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairTransferBatchIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairTransferBatch represents a TransferBatch event raised by the LBPair contract.
type LBPairTransferBatch struct {
	Sender  common.Address
	From    common.Address
	To      common.Address
	Ids     []*big.Int
	Amounts []*big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterTransferBatch is a free log retrieval operation binding the contract event 0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb.
//
// Solidity: event TransferBatch(address indexed sender, address indexed from, address indexed to, uint256[] ids, uint256[] amounts)
func (_LBPair *LBPairFilterer) FilterTransferBatch(opts *bind.FilterOpts, sender []common.Address, from []common.Address, to []common.Address) (*LBPairTransferBatchIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "TransferBatch", senderRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &LBPairTransferBatchIterator{contract: _LBPair.contract, event: "TransferBatch", logs: logs, sub: sub}, nil
}

// WatchTransferBatch is a free log subscription operation binding the contract event 0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb.
//
// Solidity: event TransferBatch(address indexed sender, address indexed from, address indexed to, uint256[] ids, uint256[] amounts)
func (_LBPair *LBPairFilterer) WatchTransferBatch(opts *bind.WatchOpts, sink chan<- *LBPairTransferBatch, sender []common.Address, from []common.Address, to []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "TransferBatch", senderRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairTransferBatch)
				if err := _LBPair.contract.UnpackLog(event, "TransferBatch", log); err != nil {
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

// ParseTransferBatch is a log parse operation binding the contract event 0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb.
//
// Solidity: event TransferBatch(address indexed sender, address indexed from, address indexed to, uint256[] ids, uint256[] amounts)
func (_LBPair *LBPairFilterer) ParseTransferBatch(log types.Log) (*LBPairTransferBatch, error) {
	event := new(LBPairTransferBatch)
	if err := _LBPair.contract.UnpackLog(event, "TransferBatch", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairWithdrawnFromBinsIterator is returned from FilterWithdrawnFromBins and is used to iterate over the raw logs and unpacked data for WithdrawnFromBins events raised by the LBPair contract.
type LBPairWithdrawnFromBinsIterator struct {
	Event *LBPairWithdrawnFromBins // Event containing the contract specifics and raw log

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
func (it *LBPairWithdrawnFromBinsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairWithdrawnFromBins)
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
		it.Event = new(LBPairWithdrawnFromBins)
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
func (it *LBPairWithdrawnFromBinsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairWithdrawnFromBinsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairWithdrawnFromBins represents a WithdrawnFromBins event raised by the LBPair contract.
type LBPairWithdrawnFromBins struct {
	Sender  common.Address
	To      common.Address
	Ids     []*big.Int
	Amounts [][32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterWithdrawnFromBins is a free log retrieval operation binding the contract event 0xa32e146844d6144a22e94c586715a1317d58a8aa3581ec33d040113ddcb24350.
//
// Solidity: event WithdrawnFromBins(address indexed sender, address indexed to, uint256[] ids, bytes32[] amounts)
func (_LBPair *LBPairFilterer) FilterWithdrawnFromBins(opts *bind.FilterOpts, sender []common.Address, to []common.Address) (*LBPairWithdrawnFromBinsIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "WithdrawnFromBins", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return &LBPairWithdrawnFromBinsIterator{contract: _LBPair.contract, event: "WithdrawnFromBins", logs: logs, sub: sub}, nil
}

// WatchWithdrawnFromBins is a free log subscription operation binding the contract event 0xa32e146844d6144a22e94c586715a1317d58a8aa3581ec33d040113ddcb24350.
//
// Solidity: event WithdrawnFromBins(address indexed sender, address indexed to, uint256[] ids, bytes32[] amounts)
func (_LBPair *LBPairFilterer) WatchWithdrawnFromBins(opts *bind.WatchOpts, sink chan<- *LBPairWithdrawnFromBins, sender []common.Address, to []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "WithdrawnFromBins", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairWithdrawnFromBins)
				if err := _LBPair.contract.UnpackLog(event, "WithdrawnFromBins", log); err != nil {
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

// ParseWithdrawnFromBins is a log parse operation binding the contract event 0xa32e146844d6144a22e94c586715a1317d58a8aa3581ec33d040113ddcb24350.
//
// Solidity: event WithdrawnFromBins(address indexed sender, address indexed to, uint256[] ids, bytes32[] amounts)
func (_LBPair *LBPairFilterer) ParseWithdrawnFromBins(log types.Log) (*LBPairWithdrawnFromBins, error) {
	event := new(LBPairWithdrawnFromBins)
	if err := _LBPair.contract.UnpackLog(event, "WithdrawnFromBins", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairWithdrawnFromBins0Iterator is returned from FilterWithdrawnFromBins0 and is used to iterate over the raw logs and unpacked data for WithdrawnFromBins0 events raised by the LBPair contract.
type LBPairWithdrawnFromBins0Iterator struct {
	Event *LBPairWithdrawnFromBins0 // Event containing the contract specifics and raw log

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
func (it *LBPairWithdrawnFromBins0Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairWithdrawnFromBins0)
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
		it.Event = new(LBPairWithdrawnFromBins0)
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
func (it *LBPairWithdrawnFromBins0Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairWithdrawnFromBins0Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairWithdrawnFromBins0 represents a WithdrawnFromBins0 event raised by the LBPair contract.
type LBPairWithdrawnFromBins0 struct {
	Sender    common.Address
	To        common.Address
	Ids       []*big.Int
	Amounts   [][32]byte
	PriceData ILBPairPriceData
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterWithdrawnFromBins0 is a free log retrieval operation binding the contract event 0x97aca70579f5049b7872b1d30aef1eab31c1c91ebf2e73d812cd626af0dcbba3.
//
// Solidity: event WithdrawnFromBins(address indexed sender, address indexed to, uint256[] ids, bytes32[] amounts, (uint256,uint256,uint256) priceData)
func (_LBPair *LBPairFilterer) FilterWithdrawnFromBins0(opts *bind.FilterOpts, sender []common.Address, to []common.Address) (*LBPairWithdrawnFromBins0Iterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "WithdrawnFromBins0", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return &LBPairWithdrawnFromBins0Iterator{contract: _LBPair.contract, event: "WithdrawnFromBins0", logs: logs, sub: sub}, nil
}

// WatchWithdrawnFromBins0 is a free log subscription operation binding the contract event 0x97aca70579f5049b7872b1d30aef1eab31c1c91ebf2e73d812cd626af0dcbba3.
//
// Solidity: event WithdrawnFromBins(address indexed sender, address indexed to, uint256[] ids, bytes32[] amounts, (uint256,uint256,uint256) priceData)
func (_LBPair *LBPairFilterer) WatchWithdrawnFromBins0(opts *bind.WatchOpts, sink chan<- *LBPairWithdrawnFromBins0, sender []common.Address, to []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "WithdrawnFromBins0", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairWithdrawnFromBins0)
				if err := _LBPair.contract.UnpackLog(event, "WithdrawnFromBins0", log); err != nil {
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

// ParseWithdrawnFromBins0 is a log parse operation binding the contract event 0x97aca70579f5049b7872b1d30aef1eab31c1c91ebf2e73d812cd626af0dcbba3.
//
// Solidity: event WithdrawnFromBins(address indexed sender, address indexed to, uint256[] ids, bytes32[] amounts, (uint256,uint256,uint256) priceData)
func (_LBPair *LBPairFilterer) ParseWithdrawnFromBins0(log types.Log) (*LBPairWithdrawnFromBins0, error) {
	event := new(LBPairWithdrawnFromBins0)
	if err := _LBPair.contract.UnpackLog(event, "WithdrawnFromBins0", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
