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

// ILBFactoryLBPairInformation is an auto generated low-level Go binding around an user-defined struct.
type ILBFactoryLBPairInformation struct {
	BinStep           uint16
	LBPair            common.Address
	CreatedByOwner    bool
	IgnoredForRouting bool
}

// LBFactoryMetaData contains all meta data concerning the LBFactory contract.
var LBFactoryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"oldRecipient\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"newRecipient\",\"type\":\"address\"}],\"name\":\"FeeRecipientSet\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"oldFlashLoanFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newFlashLoanFee\",\"type\":\"uint256\"}],\"name\":\"FlashLoanFeeSet\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"contractIERC20\",\"name\":\"tokenX\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"contractIERC20\",\"name\":\"tokenY\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"binStep\",\"type\":\"uint256\"},{\"internalType\":\"contractILBPair\",\"name\":\"LBPair\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"pid\",\"type\":\"uint256\"}],\"name\":\"LBPairCreated\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"contractIERC20\",\"name\":\"tokenX\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"contractIERC20\",\"name\":\"tokenY\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"binStep\",\"type\":\"uint256\"},{\"internalType\":\"contractILBPair\",\"name\":\"LBPair\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"pid\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativePriceUSD\",\"type\":\"uint256\"}],\"name\":\"LBPairCreated\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"contractILBPair\",\"name\":\"LBPair\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"ignored\",\"type\":\"bool\"}],\"name\":\"LBPairIgnoredStateChanged\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"oldLBPairImplementation\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"LBPairImplementation\",\"type\":\"address\"}],\"name\":\"LBPairImplementationSet\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"pendingOwner\",\"type\":\"address\"}],\"name\":\"PendingOwnerSet\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"binStep\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bool\",\"name\":\"isOpen\",\"type\":\"bool\"}],\"name\":\"PresetOpenStateChanged\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"binStep\",\"type\":\"uint256\"}],\"name\":\"PresetRemoved\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"binStep\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseFactor\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"filterPeriod\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"decayPeriod\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reductionFactor\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"variableFeeControl\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"protocolShare\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxVolatilityAccumulator\",\"type\":\"uint256\"}],\"name\":\"PresetSet\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"contractIERC20\",\"name\":\"quoteAsset\",\"type\":\"address\"}],\"name\":\"QuoteAssetAdded\",\"type\":\"event\"},{\"inputs\":[{\"indexed\":true,\"internalType\":\"contractIERC20\",\"name\":\"quoteAsset\",\"type\":\"address\"}],\"name\":\"QuoteAssetRemoved\",\"type\":\"event\"},{\"name\":\"getAllBinSteps\",\"outputs\":[{\"internalType\":\"uint256[]\",\"name\":\"binStepWithPreset\",\"type\":\"uint256[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"tokenX\",\"type\":\"address\"},{\"internalType\":\"contractIERC20\",\"name\":\"tokenY\",\"type\":\"address\"}],\"name\":\"getAllLBPairs\",\"outputs\":[{\"components\":[{\"internalType\":\"uint16\",\"name\":\"binStep\",\"type\":\"uint16\"},{\"internalType\":\"contractILBPair\",\"name\":\"LBPair\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"createdByOwner\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"ignoredForRouting\",\"type\":\"bool\"}],\"internalType\":\"structILBFactory.LBPairInformation[]\",\"name\":\"lbPairsAvailable\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"name\":\"getFeeRecipient\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"feeRecipient\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"name\":\"getFlashLoanFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"flashLoanFee\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"getLBPairAtIndex\",\"outputs\":[{\"internalType\":\"contractILBPair\",\"name\":\"lbPair\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"name\":\"getLBPairImplementation\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"lbPairImplementation\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"contractIERC20\",\"name\":\"tokenB\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"binStep\",\"type\":\"uint256\"}],\"name\":\"getLBPairInformation\",\"outputs\":[{\"components\":[{\"internalType\":\"uint16\",\"name\":\"binStep\",\"type\":\"uint16\"},{\"internalType\":\"contractILBPair\",\"name\":\"LBPair\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"createdByOwner\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"ignoredForRouting\",\"type\":\"bool\"}],\"internalType\":\"structILBFactory.LBPairInformation\",\"name\":\"lbPairInformation\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"name\":\"getMaxFlashLoanFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"maxFee\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"name\":\"getMinBinStep\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"minBinStep\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"name\":\"getNumberOfLBPairs\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"lbPairNumber\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"name\":\"getNumberOfQuoteAssets\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"numberOfQuoteAssets\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"name\":\"getOpenBinSteps\",\"outputs\":[{\"internalType\":\"uint256[]\",\"name\":\"openBinStep\",\"type\":\"uint256[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"binStep\",\"type\":\"uint256\"}],\"name\":\"getPreset\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"baseFactor\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"filterPeriod\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"decayPeriod\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reductionFactor\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"variableFeeControl\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"protocolShare\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxVolatilityAccumulator\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isOpen\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"getQuoteAssetAtIndex\",\"outputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"asset\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"token\",\"type\":\"address\"}],\"name\":\"isQuoteAsset\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"isQuote\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"name\":\"pendingOwner\",\"outputs\":[{\"internalType\":\"address\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// LBFactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use LBFactoryMetaData.ABI instead.
var LBFactoryABI = LBFactoryMetaData.ABI

// LBFactory is an auto generated Go binding around an Ethereum contract.
type LBFactory struct {
	LBFactoryCaller     // Read-only binding to the contract
	LBFactoryTransactor // Write-only binding to the contract
	LBFactoryFilterer   // Log filterer for contract events
}

// LBFactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type LBFactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LBFactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LBFactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LBFactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LBFactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LBFactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LBFactorySession struct {
	Contract     *LBFactory        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// LBFactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LBFactoryCallerSession struct {
	Contract *LBFactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// LBFactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LBFactoryTransactorSession struct {
	Contract     *LBFactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// LBFactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type LBFactoryRaw struct {
	Contract *LBFactory // Generic contract binding to access the raw methods on
}

// LBFactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LBFactoryCallerRaw struct {
	Contract *LBFactoryCaller // Generic read-only contract binding to access the raw methods on
}

// LBFactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LBFactoryTransactorRaw struct {
	Contract *LBFactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLBFactory creates a new instance of LBFactory, bound to a specific deployed contract.
func NewLBFactory(address common.Address, backend bind.ContractBackend) (*LBFactory, error) {
	contract, err := bindLBFactory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LBFactory{LBFactoryCaller: LBFactoryCaller{contract: contract}, LBFactoryTransactor: LBFactoryTransactor{contract: contract}, LBFactoryFilterer: LBFactoryFilterer{contract: contract}}, nil
}

// NewLBFactoryCaller creates a new read-only instance of LBFactory, bound to a specific deployed contract.
func NewLBFactoryCaller(address common.Address, caller bind.ContractCaller) (*LBFactoryCaller, error) {
	contract, err := bindLBFactory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LBFactoryCaller{contract: contract}, nil
}

// NewLBFactoryTransactor creates a new write-only instance of LBFactory, bound to a specific deployed contract.
func NewLBFactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*LBFactoryTransactor, error) {
	contract, err := bindLBFactory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LBFactoryTransactor{contract: contract}, nil
}

// NewLBFactoryFilterer creates a new log filterer instance of LBFactory, bound to a specific deployed contract.
func NewLBFactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*LBFactoryFilterer, error) {
	contract, err := bindLBFactory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LBFactoryFilterer{contract: contract}, nil
}

// bindLBFactory binds a generic wrapper to an already deployed contract.
func bindLBFactory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := LBFactoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LBFactory *LBFactoryRaw) Call(opts *bind.CallOpts, result *[]any, method string, params ...any) error {
	return _LBFactory.Contract.LBFactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LBFactory *LBFactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LBFactory.Contract.LBFactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LBFactory *LBFactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...any) (*types.Transaction, error) {
	return _LBFactory.Contract.LBFactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LBFactory *LBFactoryCallerRaw) Call(opts *bind.CallOpts, result *[]any, method string, params ...any) error {
	return _LBFactory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LBFactory *LBFactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LBFactory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LBFactory *LBFactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...any) (*types.Transaction, error) {
	return _LBFactory.Contract.contract.Transact(opts, method, params...)
}

// GetAllBinSteps is a free data retrieval call binding the contract method 0x5b35875c.
//
// Solidity: function getAllBinSteps() view returns(uint256[] binStepWithPreset)
func (_LBFactory *LBFactoryCaller) GetAllBinSteps(opts *bind.CallOpts) ([]*big.Int, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "getAllBinSteps")

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// GetAllBinSteps is a free data retrieval call binding the contract method 0x5b35875c.
//
// Solidity: function getAllBinSteps() view returns(uint256[] binStepWithPreset)
func (_LBFactory *LBFactorySession) GetAllBinSteps() ([]*big.Int, error) {
	return _LBFactory.Contract.GetAllBinSteps(&_LBFactory.CallOpts)
}

// GetAllBinSteps is a free data retrieval call binding the contract method 0x5b35875c.
//
// Solidity: function getAllBinSteps() view returns(uint256[] binStepWithPreset)
func (_LBFactory *LBFactoryCallerSession) GetAllBinSteps() ([]*big.Int, error) {
	return _LBFactory.Contract.GetAllBinSteps(&_LBFactory.CallOpts)
}

// GetAllLBPairs is a free data retrieval call binding the contract method 0x6622e0d7.
//
// Solidity: function getAllLBPairs(address tokenX, address tokenY) view returns((uint16,address,bool,bool)[] lbPairsAvailable)
func (_LBFactory *LBFactoryCaller) GetAllLBPairs(opts *bind.CallOpts, tokenX common.Address, tokenY common.Address) ([]ILBFactoryLBPairInformation, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "getAllLBPairs", tokenX, tokenY)

	if err != nil {
		return *new([]ILBFactoryLBPairInformation), err
	}

	out0 := *abi.ConvertType(out[0], new([]ILBFactoryLBPairInformation)).(*[]ILBFactoryLBPairInformation)

	return out0, err

}

// GetAllLBPairs is a free data retrieval call binding the contract method 0x6622e0d7.
//
// Solidity: function getAllLBPairs(address tokenX, address tokenY) view returns((uint16,address,bool,bool)[] lbPairsAvailable)
func (_LBFactory *LBFactorySession) GetAllLBPairs(tokenX common.Address, tokenY common.Address) ([]ILBFactoryLBPairInformation, error) {
	return _LBFactory.Contract.GetAllLBPairs(&_LBFactory.CallOpts, tokenX, tokenY)
}

// GetAllLBPairs is a free data retrieval call binding the contract method 0x6622e0d7.
//
// Solidity: function getAllLBPairs(address tokenX, address tokenY) view returns((uint16,address,bool,bool)[] lbPairsAvailable)
func (_LBFactory *LBFactoryCallerSession) GetAllLBPairs(tokenX common.Address, tokenY common.Address) ([]ILBFactoryLBPairInformation, error) {
	return _LBFactory.Contract.GetAllLBPairs(&_LBFactory.CallOpts, tokenX, tokenY)
}

// GetFeeRecipient is a free data retrieval call binding the contract method 0x4ccb20c0.
//
// Solidity: function getFeeRecipient() view returns(address feeRecipient)
func (_LBFactory *LBFactoryCaller) GetFeeRecipient(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "getFeeRecipient")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetFeeRecipient is a free data retrieval call binding the contract method 0x4ccb20c0.
//
// Solidity: function getFeeRecipient() view returns(address feeRecipient)
func (_LBFactory *LBFactorySession) GetFeeRecipient() (common.Address, error) {
	return _LBFactory.Contract.GetFeeRecipient(&_LBFactory.CallOpts)
}

// GetFeeRecipient is a free data retrieval call binding the contract method 0x4ccb20c0.
//
// Solidity: function getFeeRecipient() view returns(address feeRecipient)
func (_LBFactory *LBFactoryCallerSession) GetFeeRecipient() (common.Address, error) {
	return _LBFactory.Contract.GetFeeRecipient(&_LBFactory.CallOpts)
}

// GetFlashLoanFee is a free data retrieval call binding the contract method 0xfd90c2be.
//
// Solidity: function getFlashLoanFee() view returns(uint256 flashLoanFee)
func (_LBFactory *LBFactoryCaller) GetFlashLoanFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "getFlashLoanFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetFlashLoanFee is a free data retrieval call binding the contract method 0xfd90c2be.
//
// Solidity: function getFlashLoanFee() view returns(uint256 flashLoanFee)
func (_LBFactory *LBFactorySession) GetFlashLoanFee() (*big.Int, error) {
	return _LBFactory.Contract.GetFlashLoanFee(&_LBFactory.CallOpts)
}

// GetFlashLoanFee is a free data retrieval call binding the contract method 0xfd90c2be.
//
// Solidity: function getFlashLoanFee() view returns(uint256 flashLoanFee)
func (_LBFactory *LBFactoryCallerSession) GetFlashLoanFee() (*big.Int, error) {
	return _LBFactory.Contract.GetFlashLoanFee(&_LBFactory.CallOpts)
}

// GetLBPairAtIndex is a free data retrieval call binding the contract method 0x7daf5d66.
//
// Solidity: function getLBPairAtIndex(uint256 index) view returns(address lbPair)
func (_LBFactory *LBFactoryCaller) GetLBPairAtIndex(opts *bind.CallOpts, index *big.Int) (common.Address, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "getLBPairAtIndex", index)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetLBPairAtIndex is a free data retrieval call binding the contract method 0x7daf5d66.
//
// Solidity: function getLBPairAtIndex(uint256 index) view returns(address lbPair)
func (_LBFactory *LBFactorySession) GetLBPairAtIndex(index *big.Int) (common.Address, error) {
	return _LBFactory.Contract.GetLBPairAtIndex(&_LBFactory.CallOpts, index)
}

// GetLBPairAtIndex is a free data retrieval call binding the contract method 0x7daf5d66.
//
// Solidity: function getLBPairAtIndex(uint256 index) view returns(address lbPair)
func (_LBFactory *LBFactoryCallerSession) GetLBPairAtIndex(index *big.Int) (common.Address, error) {
	return _LBFactory.Contract.GetLBPairAtIndex(&_LBFactory.CallOpts, index)
}

// GetLBPairImplementation is a free data retrieval call binding the contract method 0xaf371065.
//
// Solidity: function getLBPairImplementation() view returns(address lbPairImplementation)
func (_LBFactory *LBFactoryCaller) GetLBPairImplementation(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "getLBPairImplementation")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetLBPairImplementation is a free data retrieval call binding the contract method 0xaf371065.
//
// Solidity: function getLBPairImplementation() view returns(address lbPairImplementation)
func (_LBFactory *LBFactorySession) GetLBPairImplementation() (common.Address, error) {
	return _LBFactory.Contract.GetLBPairImplementation(&_LBFactory.CallOpts)
}

// GetLBPairImplementation is a free data retrieval call binding the contract method 0xaf371065.
//
// Solidity: function getLBPairImplementation() view returns(address lbPairImplementation)
func (_LBFactory *LBFactoryCallerSession) GetLBPairImplementation() (common.Address, error) {
	return _LBFactory.Contract.GetLBPairImplementation(&_LBFactory.CallOpts)
}

// GetLBPairInformation is a free data retrieval call binding the contract method 0x704037bd.
//
// Solidity: function getLBPairInformation(address tokenA, address tokenB, uint256 binStep) view returns((uint16,address,bool,bool) lbPairInformation)
func (_LBFactory *LBFactoryCaller) GetLBPairInformation(opts *bind.CallOpts, tokenA common.Address, tokenB common.Address, binStep *big.Int) (ILBFactoryLBPairInformation, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "getLBPairInformation", tokenA, tokenB, binStep)

	if err != nil {
		return *new(ILBFactoryLBPairInformation), err
	}

	out0 := *abi.ConvertType(out[0], new(ILBFactoryLBPairInformation)).(*ILBFactoryLBPairInformation)

	return out0, err

}

// GetLBPairInformation is a free data retrieval call binding the contract method 0x704037bd.
//
// Solidity: function getLBPairInformation(address tokenA, address tokenB, uint256 binStep) view returns((uint16,address,bool,bool) lbPairInformation)
func (_LBFactory *LBFactorySession) GetLBPairInformation(tokenA common.Address, tokenB common.Address, binStep *big.Int) (ILBFactoryLBPairInformation, error) {
	return _LBFactory.Contract.GetLBPairInformation(&_LBFactory.CallOpts, tokenA, tokenB, binStep)
}

// GetLBPairInformation is a free data retrieval call binding the contract method 0x704037bd.
//
// Solidity: function getLBPairInformation(address tokenA, address tokenB, uint256 binStep) view returns((uint16,address,bool,bool) lbPairInformation)
func (_LBFactory *LBFactoryCallerSession) GetLBPairInformation(tokenA common.Address, tokenB common.Address, binStep *big.Int) (ILBFactoryLBPairInformation, error) {
	return _LBFactory.Contract.GetLBPairInformation(&_LBFactory.CallOpts, tokenA, tokenB, binStep)
}

// GetMaxFlashLoanFee is a free data retrieval call binding the contract method 0x8ce9aa1c.
//
// Solidity: function getMaxFlashLoanFee() pure returns(uint256 maxFee)
func (_LBFactory *LBFactoryCaller) GetMaxFlashLoanFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "getMaxFlashLoanFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetMaxFlashLoanFee is a free data retrieval call binding the contract method 0x8ce9aa1c.
//
// Solidity: function getMaxFlashLoanFee() pure returns(uint256 maxFee)
func (_LBFactory *LBFactorySession) GetMaxFlashLoanFee() (*big.Int, error) {
	return _LBFactory.Contract.GetMaxFlashLoanFee(&_LBFactory.CallOpts)
}

// GetMaxFlashLoanFee is a free data retrieval call binding the contract method 0x8ce9aa1c.
//
// Solidity: function getMaxFlashLoanFee() pure returns(uint256 maxFee)
func (_LBFactory *LBFactoryCallerSession) GetMaxFlashLoanFee() (*big.Int, error) {
	return _LBFactory.Contract.GetMaxFlashLoanFee(&_LBFactory.CallOpts)
}

// GetMinBinStep is a free data retrieval call binding the contract method 0x701ab8c1.
//
// Solidity: function getMinBinStep() pure returns(uint256 minBinStep)
func (_LBFactory *LBFactoryCaller) GetMinBinStep(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "getMinBinStep")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetMinBinStep is a free data retrieval call binding the contract method 0x701ab8c1.
//
// Solidity: function getMinBinStep() pure returns(uint256 minBinStep)
func (_LBFactory *LBFactorySession) GetMinBinStep() (*big.Int, error) {
	return _LBFactory.Contract.GetMinBinStep(&_LBFactory.CallOpts)
}

// GetMinBinStep is a free data retrieval call binding the contract method 0x701ab8c1.
//
// Solidity: function getMinBinStep() pure returns(uint256 minBinStep)
func (_LBFactory *LBFactoryCallerSession) GetMinBinStep() (*big.Int, error) {
	return _LBFactory.Contract.GetMinBinStep(&_LBFactory.CallOpts)
}

// GetNumberOfLBPairs is a free data retrieval call binding the contract method 0x4e937c3a.
//
// Solidity: function getNumberOfLBPairs() view returns(uint256 lbPairNumber)
func (_LBFactory *LBFactoryCaller) GetNumberOfLBPairs(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "getNumberOfLBPairs")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNumberOfLBPairs is a free data retrieval call binding the contract method 0x4e937c3a.
//
// Solidity: function getNumberOfLBPairs() view returns(uint256 lbPairNumber)
func (_LBFactory *LBFactorySession) GetNumberOfLBPairs() (*big.Int, error) {
	return _LBFactory.Contract.GetNumberOfLBPairs(&_LBFactory.CallOpts)
}

// GetNumberOfLBPairs is a free data retrieval call binding the contract method 0x4e937c3a.
//
// Solidity: function getNumberOfLBPairs() view returns(uint256 lbPairNumber)
func (_LBFactory *LBFactoryCallerSession) GetNumberOfLBPairs() (*big.Int, error) {
	return _LBFactory.Contract.GetNumberOfLBPairs(&_LBFactory.CallOpts)
}

// GetNumberOfQuoteAssets is a free data retrieval call binding the contract method 0x80c5061e.
//
// Solidity: function getNumberOfQuoteAssets() view returns(uint256 numberOfQuoteAssets)
func (_LBFactory *LBFactoryCaller) GetNumberOfQuoteAssets(opts *bind.CallOpts) (*big.Int, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "getNumberOfQuoteAssets")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNumberOfQuoteAssets is a free data retrieval call binding the contract method 0x80c5061e.
//
// Solidity: function getNumberOfQuoteAssets() view returns(uint256 numberOfQuoteAssets)
func (_LBFactory *LBFactorySession) GetNumberOfQuoteAssets() (*big.Int, error) {
	return _LBFactory.Contract.GetNumberOfQuoteAssets(&_LBFactory.CallOpts)
}

// GetNumberOfQuoteAssets is a free data retrieval call binding the contract method 0x80c5061e.
//
// Solidity: function getNumberOfQuoteAssets() view returns(uint256 numberOfQuoteAssets)
func (_LBFactory *LBFactoryCallerSession) GetNumberOfQuoteAssets() (*big.Int, error) {
	return _LBFactory.Contract.GetNumberOfQuoteAssets(&_LBFactory.CallOpts)
}

// GetOpenBinSteps is a free data retrieval call binding the contract method 0x0282c9c1.
//
// Solidity: function getOpenBinSteps() view returns(uint256[] openBinStep)
func (_LBFactory *LBFactoryCaller) GetOpenBinSteps(opts *bind.CallOpts) ([]*big.Int, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "getOpenBinSteps")

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// GetOpenBinSteps is a free data retrieval call binding the contract method 0x0282c9c1.
//
// Solidity: function getOpenBinSteps() view returns(uint256[] openBinStep)
func (_LBFactory *LBFactorySession) GetOpenBinSteps() ([]*big.Int, error) {
	return _LBFactory.Contract.GetOpenBinSteps(&_LBFactory.CallOpts)
}

// GetOpenBinSteps is a free data retrieval call binding the contract method 0x0282c9c1.
//
// Solidity: function getOpenBinSteps() view returns(uint256[] openBinStep)
func (_LBFactory *LBFactoryCallerSession) GetOpenBinSteps() ([]*big.Int, error) {
	return _LBFactory.Contract.GetOpenBinSteps(&_LBFactory.CallOpts)
}

// GetPreset is a free data retrieval call binding the contract method 0xaabc4b3c.
//
// Solidity: function getPreset(uint256 binStep) view returns(uint256 baseFactor, uint256 filterPeriod, uint256 decayPeriod, uint256 reductionFactor, uint256 variableFeeControl, uint256 protocolShare, uint256 maxVolatilityAccumulator, bool isOpen)
func (_LBFactory *LBFactoryCaller) GetPreset(opts *bind.CallOpts, binStep *big.Int) (struct {
	BaseFactor               *big.Int
	FilterPeriod             *big.Int
	DecayPeriod              *big.Int
	ReductionFactor          *big.Int
	VariableFeeControl       *big.Int
	ProtocolShare            *big.Int
	MaxVolatilityAccumulator *big.Int
	IsOpen                   bool
}, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "getPreset", binStep)

	outstruct := new(struct {
		BaseFactor               *big.Int
		FilterPeriod             *big.Int
		DecayPeriod              *big.Int
		ReductionFactor          *big.Int
		VariableFeeControl       *big.Int
		ProtocolShare            *big.Int
		MaxVolatilityAccumulator *big.Int
		IsOpen                   bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.BaseFactor = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.FilterPeriod = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.DecayPeriod = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.ReductionFactor = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.VariableFeeControl = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.ProtocolShare = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)
	outstruct.MaxVolatilityAccumulator = *abi.ConvertType(out[6], new(*big.Int)).(**big.Int)
	outstruct.IsOpen = *abi.ConvertType(out[7], new(bool)).(*bool)

	return *outstruct, err

}

// GetPreset is a free data retrieval call binding the contract method 0xaabc4b3c.
//
// Solidity: function getPreset(uint256 binStep) view returns(uint256 baseFactor, uint256 filterPeriod, uint256 decayPeriod, uint256 reductionFactor, uint256 variableFeeControl, uint256 protocolShare, uint256 maxVolatilityAccumulator, bool isOpen)
func (_LBFactory *LBFactorySession) GetPreset(binStep *big.Int) (struct {
	BaseFactor               *big.Int
	FilterPeriod             *big.Int
	DecayPeriod              *big.Int
	ReductionFactor          *big.Int
	VariableFeeControl       *big.Int
	ProtocolShare            *big.Int
	MaxVolatilityAccumulator *big.Int
	IsOpen                   bool
}, error) {
	return _LBFactory.Contract.GetPreset(&_LBFactory.CallOpts, binStep)
}

// GetPreset is a free data retrieval call binding the contract method 0xaabc4b3c.
//
// Solidity: function getPreset(uint256 binStep) view returns(uint256 baseFactor, uint256 filterPeriod, uint256 decayPeriod, uint256 reductionFactor, uint256 variableFeeControl, uint256 protocolShare, uint256 maxVolatilityAccumulator, bool isOpen)
func (_LBFactory *LBFactoryCallerSession) GetPreset(binStep *big.Int) (struct {
	BaseFactor               *big.Int
	FilterPeriod             *big.Int
	DecayPeriod              *big.Int
	ReductionFactor          *big.Int
	VariableFeeControl       *big.Int
	ProtocolShare            *big.Int
	MaxVolatilityAccumulator *big.Int
	IsOpen                   bool
}, error) {
	return _LBFactory.Contract.GetPreset(&_LBFactory.CallOpts, binStep)
}

// GetQuoteAssetAtIndex is a free data retrieval call binding the contract method 0x0752092b.
//
// Solidity: function getQuoteAssetAtIndex(uint256 index) view returns(address asset)
func (_LBFactory *LBFactoryCaller) GetQuoteAssetAtIndex(opts *bind.CallOpts, index *big.Int) (common.Address, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "getQuoteAssetAtIndex", index)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetQuoteAssetAtIndex is a free data retrieval call binding the contract method 0x0752092b.
//
// Solidity: function getQuoteAssetAtIndex(uint256 index) view returns(address asset)
func (_LBFactory *LBFactorySession) GetQuoteAssetAtIndex(index *big.Int) (common.Address, error) {
	return _LBFactory.Contract.GetQuoteAssetAtIndex(&_LBFactory.CallOpts, index)
}

// GetQuoteAssetAtIndex is a free data retrieval call binding the contract method 0x0752092b.
//
// Solidity: function getQuoteAssetAtIndex(uint256 index) view returns(address asset)
func (_LBFactory *LBFactoryCallerSession) GetQuoteAssetAtIndex(index *big.Int) (common.Address, error) {
	return _LBFactory.Contract.GetQuoteAssetAtIndex(&_LBFactory.CallOpts, index)
}

// IsQuoteAsset is a free data retrieval call binding the contract method 0x27721842.
//
// Solidity: function isQuoteAsset(address token) view returns(bool isQuote)
func (_LBFactory *LBFactoryCaller) IsQuoteAsset(opts *bind.CallOpts, token common.Address) (bool, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "isQuoteAsset", token)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsQuoteAsset is a free data retrieval call binding the contract method 0x27721842.
//
// Solidity: function isQuoteAsset(address token) view returns(bool isQuote)
func (_LBFactory *LBFactorySession) IsQuoteAsset(token common.Address) (bool, error) {
	return _LBFactory.Contract.IsQuoteAsset(&_LBFactory.CallOpts, token)
}

// IsQuoteAsset is a free data retrieval call binding the contract method 0x27721842.
//
// Solidity: function isQuoteAsset(address token) view returns(bool isQuote)
func (_LBFactory *LBFactoryCallerSession) IsQuoteAsset(token common.Address) (bool, error) {
	return _LBFactory.Contract.IsQuoteAsset(&_LBFactory.CallOpts, token)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_LBFactory *LBFactoryCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_LBFactory *LBFactorySession) Owner() (common.Address, error) {
	return _LBFactory.Contract.Owner(&_LBFactory.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_LBFactory *LBFactoryCallerSession) Owner() (common.Address, error) {
	return _LBFactory.Contract.Owner(&_LBFactory.CallOpts)
}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_LBFactory *LBFactoryCaller) PendingOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _LBFactory.contract.Call(opts, &out, "pendingOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_LBFactory *LBFactorySession) PendingOwner() (common.Address, error) {
	return _LBFactory.Contract.PendingOwner(&_LBFactory.CallOpts)
}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_LBFactory *LBFactoryCallerSession) PendingOwner() (common.Address, error) {
	return _LBFactory.Contract.PendingOwner(&_LBFactory.CallOpts)
}

// LBFactoryFeeRecipientSetIterator is returned from FilterFeeRecipientSet and is used to iterate over the raw logs and unpacked data for FeeRecipientSet events raised by the LBFactory contract.
type LBFactoryFeeRecipientSetIterator struct {
	Event *LBFactoryFeeRecipientSet // Event containing the contract specifics and raw log

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
func (it *LBFactoryFeeRecipientSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBFactoryFeeRecipientSet)
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
		it.Event = new(LBFactoryFeeRecipientSet)
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
func (it *LBFactoryFeeRecipientSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBFactoryFeeRecipientSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBFactoryFeeRecipientSet represents a FeeRecipientSet event raised by the LBFactory contract.
type LBFactoryFeeRecipientSet struct {
	OldRecipient common.Address
	NewRecipient common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterFeeRecipientSet is a free log retrieval operation binding the contract event 0x15d80a013f22151bc7246e3bc132e12828cde19de98870475e3fa70840152721.
//
// Solidity: event FeeRecipientSet(address oldRecipient, address newRecipient)
func (_LBFactory *LBFactoryFilterer) FilterFeeRecipientSet(opts *bind.FilterOpts) (*LBFactoryFeeRecipientSetIterator, error) {

	logs, sub, err := _LBFactory.contract.FilterLogs(opts, "FeeRecipientSet")
	if err != nil {
		return nil, err
	}
	return &LBFactoryFeeRecipientSetIterator{contract: _LBFactory.contract, event: "FeeRecipientSet", logs: logs, sub: sub}, nil
}

// WatchFeeRecipientSet is a free log subscription operation binding the contract event 0x15d80a013f22151bc7246e3bc132e12828cde19de98870475e3fa70840152721.
//
// Solidity: event FeeRecipientSet(address oldRecipient, address newRecipient)
func (_LBFactory *LBFactoryFilterer) WatchFeeRecipientSet(opts *bind.WatchOpts, sink chan<- *LBFactoryFeeRecipientSet) (event.Subscription, error) {

	logs, sub, err := _LBFactory.contract.WatchLogs(opts, "FeeRecipientSet")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBFactoryFeeRecipientSet)
				if err := _LBFactory.contract.UnpackLog(event, "FeeRecipientSet", log); err != nil {
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

// ParseFeeRecipientSet is a log parse operation binding the contract event 0x15d80a013f22151bc7246e3bc132e12828cde19de98870475e3fa70840152721.
//
// Solidity: event FeeRecipientSet(address oldRecipient, address newRecipient)
func (_LBFactory *LBFactoryFilterer) ParseFeeRecipientSet(log types.Log) (*LBFactoryFeeRecipientSet, error) {
	event := new(LBFactoryFeeRecipientSet)
	if err := _LBFactory.contract.UnpackLog(event, "FeeRecipientSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBFactoryFlashLoanFeeSetIterator is returned from FilterFlashLoanFeeSet and is used to iterate over the raw logs and unpacked data for FlashLoanFeeSet events raised by the LBFactory contract.
type LBFactoryFlashLoanFeeSetIterator struct {
	Event *LBFactoryFlashLoanFeeSet // Event containing the contract specifics and raw log

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
func (it *LBFactoryFlashLoanFeeSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBFactoryFlashLoanFeeSet)
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
		it.Event = new(LBFactoryFlashLoanFeeSet)
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
func (it *LBFactoryFlashLoanFeeSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBFactoryFlashLoanFeeSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBFactoryFlashLoanFeeSet represents a FlashLoanFeeSet event raised by the LBFactory contract.
type LBFactoryFlashLoanFeeSet struct {
	OldFlashLoanFee *big.Int
	NewFlashLoanFee *big.Int
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterFlashLoanFeeSet is a free log retrieval operation binding the contract event 0x5c34e91c94c78b662a45d0bd4a25a4e32c584c54a45a76e4a4d43be27ba40e50.
//
// Solidity: event FlashLoanFeeSet(uint256 oldFlashLoanFee, uint256 newFlashLoanFee)
func (_LBFactory *LBFactoryFilterer) FilterFlashLoanFeeSet(opts *bind.FilterOpts) (*LBFactoryFlashLoanFeeSetIterator, error) {

	logs, sub, err := _LBFactory.contract.FilterLogs(opts, "FlashLoanFeeSet")
	if err != nil {
		return nil, err
	}
	return &LBFactoryFlashLoanFeeSetIterator{contract: _LBFactory.contract, event: "FlashLoanFeeSet", logs: logs, sub: sub}, nil
}

// WatchFlashLoanFeeSet is a free log subscription operation binding the contract event 0x5c34e91c94c78b662a45d0bd4a25a4e32c584c54a45a76e4a4d43be27ba40e50.
//
// Solidity: event FlashLoanFeeSet(uint256 oldFlashLoanFee, uint256 newFlashLoanFee)
func (_LBFactory *LBFactoryFilterer) WatchFlashLoanFeeSet(opts *bind.WatchOpts, sink chan<- *LBFactoryFlashLoanFeeSet) (event.Subscription, error) {

	logs, sub, err := _LBFactory.contract.WatchLogs(opts, "FlashLoanFeeSet")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBFactoryFlashLoanFeeSet)
				if err := _LBFactory.contract.UnpackLog(event, "FlashLoanFeeSet", log); err != nil {
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

// ParseFlashLoanFeeSet is a log parse operation binding the contract event 0x5c34e91c94c78b662a45d0bd4a25a4e32c584c54a45a76e4a4d43be27ba40e50.
//
// Solidity: event FlashLoanFeeSet(uint256 oldFlashLoanFee, uint256 newFlashLoanFee)
func (_LBFactory *LBFactoryFilterer) ParseFlashLoanFeeSet(log types.Log) (*LBFactoryFlashLoanFeeSet, error) {
	event := new(LBFactoryFlashLoanFeeSet)
	if err := _LBFactory.contract.UnpackLog(event, "FlashLoanFeeSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBFactoryLBPairCreatedIterator is returned from FilterLBPairCreated and is used to iterate over the raw logs and unpacked data for LBPairCreated events raised by the LBFactory contract.
type LBFactoryLBPairCreatedIterator struct {
	Event *LBFactoryLBPairCreated // Event containing the contract specifics and raw log

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
func (it *LBFactoryLBPairCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBFactoryLBPairCreated)
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
		it.Event = new(LBFactoryLBPairCreated)
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
func (it *LBFactoryLBPairCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBFactoryLBPairCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBFactoryLBPairCreated represents a LBPairCreated event raised by the LBFactory contract.
type LBFactoryLBPairCreated struct {
	TokenX  common.Address
	TokenY  common.Address
	BinStep *big.Int
	LBPair  common.Address
	Pid     *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterLBPairCreated is a free log retrieval operation binding the contract event 0x2c8d104b27c6b7f4492017a6f5cf3803043688934ebcaa6a03540beeaf976aff.
//
// Solidity: event LBPairCreated(address indexed tokenX, address indexed tokenY, uint256 indexed binStep, address LBPair, uint256 pid)
func (_LBFactory *LBFactoryFilterer) FilterLBPairCreated(opts *bind.FilterOpts, tokenX []common.Address, tokenY []common.Address, binStep []*big.Int) (*LBFactoryLBPairCreatedIterator, error) {

	var tokenXRule []any
	for _, tokenXItem := range tokenX {
		tokenXRule = append(tokenXRule, tokenXItem)
	}
	var tokenYRule []any
	for _, tokenYItem := range tokenY {
		tokenYRule = append(tokenYRule, tokenYItem)
	}
	var binStepRule []any
	for _, binStepItem := range binStep {
		binStepRule = append(binStepRule, binStepItem)
	}

	logs, sub, err := _LBFactory.contract.FilterLogs(opts, "LBPairCreated", tokenXRule, tokenYRule, binStepRule)
	if err != nil {
		return nil, err
	}
	return &LBFactoryLBPairCreatedIterator{contract: _LBFactory.contract, event: "LBPairCreated", logs: logs, sub: sub}, nil
}

// WatchLBPairCreated is a free log subscription operation binding the contract event 0x2c8d104b27c6b7f4492017a6f5cf3803043688934ebcaa6a03540beeaf976aff.
//
// Solidity: event LBPairCreated(address indexed tokenX, address indexed tokenY, uint256 indexed binStep, address LBPair, uint256 pid)
func (_LBFactory *LBFactoryFilterer) WatchLBPairCreated(opts *bind.WatchOpts, sink chan<- *LBFactoryLBPairCreated, tokenX []common.Address, tokenY []common.Address, binStep []*big.Int) (event.Subscription, error) {

	var tokenXRule []any
	for _, tokenXItem := range tokenX {
		tokenXRule = append(tokenXRule, tokenXItem)
	}
	var tokenYRule []any
	for _, tokenYItem := range tokenY {
		tokenYRule = append(tokenYRule, tokenYItem)
	}
	var binStepRule []any
	for _, binStepItem := range binStep {
		binStepRule = append(binStepRule, binStepItem)
	}

	logs, sub, err := _LBFactory.contract.WatchLogs(opts, "LBPairCreated", tokenXRule, tokenYRule, binStepRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBFactoryLBPairCreated)
				if err := _LBFactory.contract.UnpackLog(event, "LBPairCreated", log); err != nil {
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

// ParseLBPairCreated is a log parse operation binding the contract event 0x2c8d104b27c6b7f4492017a6f5cf3803043688934ebcaa6a03540beeaf976aff.
//
// Solidity: event LBPairCreated(address indexed tokenX, address indexed tokenY, uint256 indexed binStep, address LBPair, uint256 pid)
func (_LBFactory *LBFactoryFilterer) ParseLBPairCreated(log types.Log) (*LBFactoryLBPairCreated, error) {
	event := new(LBFactoryLBPairCreated)
	if err := _LBFactory.contract.UnpackLog(event, "LBPairCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBFactoryLBPairCreated0Iterator is returned from FilterLBPairCreated0 and is used to iterate over the raw logs and unpacked data for LBPairCreated0 events raised by the LBFactory contract.
type LBFactoryLBPairCreated0Iterator struct {
	Event *LBFactoryLBPairCreated0 // Event containing the contract specifics and raw log

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
func (it *LBFactoryLBPairCreated0Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBFactoryLBPairCreated0)
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
		it.Event = new(LBFactoryLBPairCreated0)
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
func (it *LBFactoryLBPairCreated0Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBFactoryLBPairCreated0Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBFactoryLBPairCreated0 represents a LBPairCreated0 event raised by the LBFactory contract.
type LBFactoryLBPairCreated0 struct {
	TokenX         common.Address
	TokenY         common.Address
	BinStep        *big.Int
	LBPair         common.Address
	Pid            *big.Int
	NativePriceUSD *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterLBPairCreated0 is a free log retrieval operation binding the contract event 0x58ec17066e02d81c6cbaa59d0167362f9fcbf8b93254183e0817644693658dff.
//
// Solidity: event LBPairCreated(address indexed tokenX, address indexed tokenY, uint256 indexed binStep, address LBPair, uint256 pid, uint256 nativePriceUSD)
func (_LBFactory *LBFactoryFilterer) FilterLBPairCreated0(opts *bind.FilterOpts, tokenX []common.Address, tokenY []common.Address, binStep []*big.Int) (*LBFactoryLBPairCreated0Iterator, error) {

	var tokenXRule []any
	for _, tokenXItem := range tokenX {
		tokenXRule = append(tokenXRule, tokenXItem)
	}
	var tokenYRule []any
	for _, tokenYItem := range tokenY {
		tokenYRule = append(tokenYRule, tokenYItem)
	}
	var binStepRule []any
	for _, binStepItem := range binStep {
		binStepRule = append(binStepRule, binStepItem)
	}

	logs, sub, err := _LBFactory.contract.FilterLogs(opts, "LBPairCreated0", tokenXRule, tokenYRule, binStepRule)
	if err != nil {
		return nil, err
	}
	return &LBFactoryLBPairCreated0Iterator{contract: _LBFactory.contract, event: "LBPairCreated0", logs: logs, sub: sub}, nil
}

// WatchLBPairCreated0 is a free log subscription operation binding the contract event 0x58ec17066e02d81c6cbaa59d0167362f9fcbf8b93254183e0817644693658dff.
//
// Solidity: event LBPairCreated(address indexed tokenX, address indexed tokenY, uint256 indexed binStep, address LBPair, uint256 pid, uint256 nativePriceUSD)
func (_LBFactory *LBFactoryFilterer) WatchLBPairCreated0(opts *bind.WatchOpts, sink chan<- *LBFactoryLBPairCreated0, tokenX []common.Address, tokenY []common.Address, binStep []*big.Int) (event.Subscription, error) {

	var tokenXRule []any
	for _, tokenXItem := range tokenX {
		tokenXRule = append(tokenXRule, tokenXItem)
	}
	var tokenYRule []any
	for _, tokenYItem := range tokenY {
		tokenYRule = append(tokenYRule, tokenYItem)
	}
	var binStepRule []any
	for _, binStepItem := range binStep {
		binStepRule = append(binStepRule, binStepItem)
	}

	logs, sub, err := _LBFactory.contract.WatchLogs(opts, "LBPairCreated0", tokenXRule, tokenYRule, binStepRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBFactoryLBPairCreated0)
				if err := _LBFactory.contract.UnpackLog(event, "LBPairCreated0", log); err != nil {
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

// ParseLBPairCreated0 is a log parse operation binding the contract event 0x58ec17066e02d81c6cbaa59d0167362f9fcbf8b93254183e0817644693658dff.
//
// Solidity: event LBPairCreated(address indexed tokenX, address indexed tokenY, uint256 indexed binStep, address LBPair, uint256 pid, uint256 nativePriceUSD)
func (_LBFactory *LBFactoryFilterer) ParseLBPairCreated0(log types.Log) (*LBFactoryLBPairCreated0, error) {
	event := new(LBFactoryLBPairCreated0)
	if err := _LBFactory.contract.UnpackLog(event, "LBPairCreated0", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBFactoryLBPairIgnoredStateChangedIterator is returned from FilterLBPairIgnoredStateChanged and is used to iterate over the raw logs and unpacked data for LBPairIgnoredStateChanged events raised by the LBFactory contract.
type LBFactoryLBPairIgnoredStateChangedIterator struct {
	Event *LBFactoryLBPairIgnoredStateChanged // Event containing the contract specifics and raw log

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
func (it *LBFactoryLBPairIgnoredStateChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBFactoryLBPairIgnoredStateChanged)
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
		it.Event = new(LBFactoryLBPairIgnoredStateChanged)
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
func (it *LBFactoryLBPairIgnoredStateChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBFactoryLBPairIgnoredStateChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBFactoryLBPairIgnoredStateChanged represents a LBPairIgnoredStateChanged event raised by the LBFactory contract.
type LBFactoryLBPairIgnoredStateChanged struct {
	LBPair  common.Address
	Ignored bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterLBPairIgnoredStateChanged is a free log retrieval operation binding the contract event 0x44cf35361c9ff3c8c1397ec6410d5495cc481feaef35c9af11da1a637107de4f.
//
// Solidity: event LBPairIgnoredStateChanged(address indexed LBPair, bool ignored)
func (_LBFactory *LBFactoryFilterer) FilterLBPairIgnoredStateChanged(opts *bind.FilterOpts, LBPair []common.Address) (*LBFactoryLBPairIgnoredStateChangedIterator, error) {

	var LBPairRule []any
	for _, LBPairItem := range LBPair {
		LBPairRule = append(LBPairRule, LBPairItem)
	}

	logs, sub, err := _LBFactory.contract.FilterLogs(opts, "LBPairIgnoredStateChanged", LBPairRule)
	if err != nil {
		return nil, err
	}
	return &LBFactoryLBPairIgnoredStateChangedIterator{contract: _LBFactory.contract, event: "LBPairIgnoredStateChanged", logs: logs, sub: sub}, nil
}

// WatchLBPairIgnoredStateChanged is a free log subscription operation binding the contract event 0x44cf35361c9ff3c8c1397ec6410d5495cc481feaef35c9af11da1a637107de4f.
//
// Solidity: event LBPairIgnoredStateChanged(address indexed LBPair, bool ignored)
func (_LBFactory *LBFactoryFilterer) WatchLBPairIgnoredStateChanged(opts *bind.WatchOpts, sink chan<- *LBFactoryLBPairIgnoredStateChanged, LBPair []common.Address) (event.Subscription, error) {

	var LBPairRule []any
	for _, LBPairItem := range LBPair {
		LBPairRule = append(LBPairRule, LBPairItem)
	}

	logs, sub, err := _LBFactory.contract.WatchLogs(opts, "LBPairIgnoredStateChanged", LBPairRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBFactoryLBPairIgnoredStateChanged)
				if err := _LBFactory.contract.UnpackLog(event, "LBPairIgnoredStateChanged", log); err != nil {
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

// ParseLBPairIgnoredStateChanged is a log parse operation binding the contract event 0x44cf35361c9ff3c8c1397ec6410d5495cc481feaef35c9af11da1a637107de4f.
//
// Solidity: event LBPairIgnoredStateChanged(address indexed LBPair, bool ignored)
func (_LBFactory *LBFactoryFilterer) ParseLBPairIgnoredStateChanged(log types.Log) (*LBFactoryLBPairIgnoredStateChanged, error) {
	event := new(LBFactoryLBPairIgnoredStateChanged)
	if err := _LBFactory.contract.UnpackLog(event, "LBPairIgnoredStateChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBFactoryLBPairImplementationSetIterator is returned from FilterLBPairImplementationSet and is used to iterate over the raw logs and unpacked data for LBPairImplementationSet events raised by the LBFactory contract.
type LBFactoryLBPairImplementationSetIterator struct {
	Event *LBFactoryLBPairImplementationSet // Event containing the contract specifics and raw log

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
func (it *LBFactoryLBPairImplementationSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBFactoryLBPairImplementationSet)
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
		it.Event = new(LBFactoryLBPairImplementationSet)
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
func (it *LBFactoryLBPairImplementationSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBFactoryLBPairImplementationSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBFactoryLBPairImplementationSet represents a LBPairImplementationSet event raised by the LBFactory contract.
type LBFactoryLBPairImplementationSet struct {
	OldLBPairImplementation common.Address
	LBPairImplementation    common.Address
	Raw                     types.Log // Blockchain specific contextual infos
}

// FilterLBPairImplementationSet is a free log retrieval operation binding the contract event 0x900d0e3d359f50e4f923ecdc06b401e07dbb9f485e17b07bcfc91a13000b277e.
//
// Solidity: event LBPairImplementationSet(address oldLBPairImplementation, address LBPairImplementation)
func (_LBFactory *LBFactoryFilterer) FilterLBPairImplementationSet(opts *bind.FilterOpts) (*LBFactoryLBPairImplementationSetIterator, error) {

	logs, sub, err := _LBFactory.contract.FilterLogs(opts, "LBPairImplementationSet")
	if err != nil {
		return nil, err
	}
	return &LBFactoryLBPairImplementationSetIterator{contract: _LBFactory.contract, event: "LBPairImplementationSet", logs: logs, sub: sub}, nil
}

// WatchLBPairImplementationSet is a free log subscription operation binding the contract event 0x900d0e3d359f50e4f923ecdc06b401e07dbb9f485e17b07bcfc91a13000b277e.
//
// Solidity: event LBPairImplementationSet(address oldLBPairImplementation, address LBPairImplementation)
func (_LBFactory *LBFactoryFilterer) WatchLBPairImplementationSet(opts *bind.WatchOpts, sink chan<- *LBFactoryLBPairImplementationSet) (event.Subscription, error) {

	logs, sub, err := _LBFactory.contract.WatchLogs(opts, "LBPairImplementationSet")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBFactoryLBPairImplementationSet)
				if err := _LBFactory.contract.UnpackLog(event, "LBPairImplementationSet", log); err != nil {
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

// ParseLBPairImplementationSet is a log parse operation binding the contract event 0x900d0e3d359f50e4f923ecdc06b401e07dbb9f485e17b07bcfc91a13000b277e.
//
// Solidity: event LBPairImplementationSet(address oldLBPairImplementation, address LBPairImplementation)
func (_LBFactory *LBFactoryFilterer) ParseLBPairImplementationSet(log types.Log) (*LBFactoryLBPairImplementationSet, error) {
	event := new(LBFactoryLBPairImplementationSet)
	if err := _LBFactory.contract.UnpackLog(event, "LBPairImplementationSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBFactoryOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the LBFactory contract.
type LBFactoryOwnershipTransferredIterator struct {
	Event *LBFactoryOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *LBFactoryOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBFactoryOwnershipTransferred)
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
		it.Event = new(LBFactoryOwnershipTransferred)
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
func (it *LBFactoryOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBFactoryOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBFactoryOwnershipTransferred represents a OwnershipTransferred event raised by the LBFactory contract.
type LBFactoryOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_LBFactory *LBFactoryFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*LBFactoryOwnershipTransferredIterator, error) {

	var previousOwnerRule []any
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []any
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _LBFactory.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &LBFactoryOwnershipTransferredIterator{contract: _LBFactory.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_LBFactory *LBFactoryFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *LBFactoryOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []any
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []any
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _LBFactory.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBFactoryOwnershipTransferred)
				if err := _LBFactory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_LBFactory *LBFactoryFilterer) ParseOwnershipTransferred(log types.Log) (*LBFactoryOwnershipTransferred, error) {
	event := new(LBFactoryOwnershipTransferred)
	if err := _LBFactory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBFactoryPendingOwnerSetIterator is returned from FilterPendingOwnerSet and is used to iterate over the raw logs and unpacked data for PendingOwnerSet events raised by the LBFactory contract.
type LBFactoryPendingOwnerSetIterator struct {
	Event *LBFactoryPendingOwnerSet // Event containing the contract specifics and raw log

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
func (it *LBFactoryPendingOwnerSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBFactoryPendingOwnerSet)
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
		it.Event = new(LBFactoryPendingOwnerSet)
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
func (it *LBFactoryPendingOwnerSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBFactoryPendingOwnerSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBFactoryPendingOwnerSet represents a PendingOwnerSet event raised by the LBFactory contract.
type LBFactoryPendingOwnerSet struct {
	PendingOwner common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterPendingOwnerSet is a free log retrieval operation binding the contract event 0x68f49b346b94582a8b5f9d10e3fe3365318fe8f191ff8dce7c59c6cad06b02f5.
//
// Solidity: event PendingOwnerSet(address indexed pendingOwner)
func (_LBFactory *LBFactoryFilterer) FilterPendingOwnerSet(opts *bind.FilterOpts, pendingOwner []common.Address) (*LBFactoryPendingOwnerSetIterator, error) {

	var pendingOwnerRule []any
	for _, pendingOwnerItem := range pendingOwner {
		pendingOwnerRule = append(pendingOwnerRule, pendingOwnerItem)
	}

	logs, sub, err := _LBFactory.contract.FilterLogs(opts, "PendingOwnerSet", pendingOwnerRule)
	if err != nil {
		return nil, err
	}
	return &LBFactoryPendingOwnerSetIterator{contract: _LBFactory.contract, event: "PendingOwnerSet", logs: logs, sub: sub}, nil
}

// WatchPendingOwnerSet is a free log subscription operation binding the contract event 0x68f49b346b94582a8b5f9d10e3fe3365318fe8f191ff8dce7c59c6cad06b02f5.
//
// Solidity: event PendingOwnerSet(address indexed pendingOwner)
func (_LBFactory *LBFactoryFilterer) WatchPendingOwnerSet(opts *bind.WatchOpts, sink chan<- *LBFactoryPendingOwnerSet, pendingOwner []common.Address) (event.Subscription, error) {

	var pendingOwnerRule []any
	for _, pendingOwnerItem := range pendingOwner {
		pendingOwnerRule = append(pendingOwnerRule, pendingOwnerItem)
	}

	logs, sub, err := _LBFactory.contract.WatchLogs(opts, "PendingOwnerSet", pendingOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBFactoryPendingOwnerSet)
				if err := _LBFactory.contract.UnpackLog(event, "PendingOwnerSet", log); err != nil {
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

// ParsePendingOwnerSet is a log parse operation binding the contract event 0x68f49b346b94582a8b5f9d10e3fe3365318fe8f191ff8dce7c59c6cad06b02f5.
//
// Solidity: event PendingOwnerSet(address indexed pendingOwner)
func (_LBFactory *LBFactoryFilterer) ParsePendingOwnerSet(log types.Log) (*LBFactoryPendingOwnerSet, error) {
	event := new(LBFactoryPendingOwnerSet)
	if err := _LBFactory.contract.UnpackLog(event, "PendingOwnerSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBFactoryPresetOpenStateChangedIterator is returned from FilterPresetOpenStateChanged and is used to iterate over the raw logs and unpacked data for PresetOpenStateChanged events raised by the LBFactory contract.
type LBFactoryPresetOpenStateChangedIterator struct {
	Event *LBFactoryPresetOpenStateChanged // Event containing the contract specifics and raw log

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
func (it *LBFactoryPresetOpenStateChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBFactoryPresetOpenStateChanged)
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
		it.Event = new(LBFactoryPresetOpenStateChanged)
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
func (it *LBFactoryPresetOpenStateChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBFactoryPresetOpenStateChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBFactoryPresetOpenStateChanged represents a PresetOpenStateChanged event raised by the LBFactory contract.
type LBFactoryPresetOpenStateChanged struct {
	BinStep *big.Int
	IsOpen  bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPresetOpenStateChanged is a free log retrieval operation binding the contract event 0x58a8b6a02b964cca2712e5a71d7b0d564a56b4a0f573b4c47f389341ade14cfd.
//
// Solidity: event PresetOpenStateChanged(uint256 indexed binStep, bool indexed isOpen)
func (_LBFactory *LBFactoryFilterer) FilterPresetOpenStateChanged(opts *bind.FilterOpts, binStep []*big.Int, isOpen []bool) (*LBFactoryPresetOpenStateChangedIterator, error) {

	var binStepRule []any
	for _, binStepItem := range binStep {
		binStepRule = append(binStepRule, binStepItem)
	}
	var isOpenRule []any
	for _, isOpenItem := range isOpen {
		isOpenRule = append(isOpenRule, isOpenItem)
	}

	logs, sub, err := _LBFactory.contract.FilterLogs(opts, "PresetOpenStateChanged", binStepRule, isOpenRule)
	if err != nil {
		return nil, err
	}
	return &LBFactoryPresetOpenStateChangedIterator{contract: _LBFactory.contract, event: "PresetOpenStateChanged", logs: logs, sub: sub}, nil
}

// WatchPresetOpenStateChanged is a free log subscription operation binding the contract event 0x58a8b6a02b964cca2712e5a71d7b0d564a56b4a0f573b4c47f389341ade14cfd.
//
// Solidity: event PresetOpenStateChanged(uint256 indexed binStep, bool indexed isOpen)
func (_LBFactory *LBFactoryFilterer) WatchPresetOpenStateChanged(opts *bind.WatchOpts, sink chan<- *LBFactoryPresetOpenStateChanged, binStep []*big.Int, isOpen []bool) (event.Subscription, error) {

	var binStepRule []any
	for _, binStepItem := range binStep {
		binStepRule = append(binStepRule, binStepItem)
	}
	var isOpenRule []any
	for _, isOpenItem := range isOpen {
		isOpenRule = append(isOpenRule, isOpenItem)
	}

	logs, sub, err := _LBFactory.contract.WatchLogs(opts, "PresetOpenStateChanged", binStepRule, isOpenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBFactoryPresetOpenStateChanged)
				if err := _LBFactory.contract.UnpackLog(event, "PresetOpenStateChanged", log); err != nil {
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

// ParsePresetOpenStateChanged is a log parse operation binding the contract event 0x58a8b6a02b964cca2712e5a71d7b0d564a56b4a0f573b4c47f389341ade14cfd.
//
// Solidity: event PresetOpenStateChanged(uint256 indexed binStep, bool indexed isOpen)
func (_LBFactory *LBFactoryFilterer) ParsePresetOpenStateChanged(log types.Log) (*LBFactoryPresetOpenStateChanged, error) {
	event := new(LBFactoryPresetOpenStateChanged)
	if err := _LBFactory.contract.UnpackLog(event, "PresetOpenStateChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBFactoryPresetRemovedIterator is returned from FilterPresetRemoved and is used to iterate over the raw logs and unpacked data for PresetRemoved events raised by the LBFactory contract.
type LBFactoryPresetRemovedIterator struct {
	Event *LBFactoryPresetRemoved // Event containing the contract specifics and raw log

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
func (it *LBFactoryPresetRemovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBFactoryPresetRemoved)
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
		it.Event = new(LBFactoryPresetRemoved)
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
func (it *LBFactoryPresetRemovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBFactoryPresetRemovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBFactoryPresetRemoved represents a PresetRemoved event raised by the LBFactory contract.
type LBFactoryPresetRemoved struct {
	BinStep *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPresetRemoved is a free log retrieval operation binding the contract event 0xdd86b848bb56ff540caa68683fa467d0e7eb5f8b2d44e4ee435742eeeae9be13.
//
// Solidity: event PresetRemoved(uint256 indexed binStep)
func (_LBFactory *LBFactoryFilterer) FilterPresetRemoved(opts *bind.FilterOpts, binStep []*big.Int) (*LBFactoryPresetRemovedIterator, error) {

	var binStepRule []any
	for _, binStepItem := range binStep {
		binStepRule = append(binStepRule, binStepItem)
	}

	logs, sub, err := _LBFactory.contract.FilterLogs(opts, "PresetRemoved", binStepRule)
	if err != nil {
		return nil, err
	}
	return &LBFactoryPresetRemovedIterator{contract: _LBFactory.contract, event: "PresetRemoved", logs: logs, sub: sub}, nil
}

// WatchPresetRemoved is a free log subscription operation binding the contract event 0xdd86b848bb56ff540caa68683fa467d0e7eb5f8b2d44e4ee435742eeeae9be13.
//
// Solidity: event PresetRemoved(uint256 indexed binStep)
func (_LBFactory *LBFactoryFilterer) WatchPresetRemoved(opts *bind.WatchOpts, sink chan<- *LBFactoryPresetRemoved, binStep []*big.Int) (event.Subscription, error) {

	var binStepRule []any
	for _, binStepItem := range binStep {
		binStepRule = append(binStepRule, binStepItem)
	}

	logs, sub, err := _LBFactory.contract.WatchLogs(opts, "PresetRemoved", binStepRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBFactoryPresetRemoved)
				if err := _LBFactory.contract.UnpackLog(event, "PresetRemoved", log); err != nil {
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

// ParsePresetRemoved is a log parse operation binding the contract event 0xdd86b848bb56ff540caa68683fa467d0e7eb5f8b2d44e4ee435742eeeae9be13.
//
// Solidity: event PresetRemoved(uint256 indexed binStep)
func (_LBFactory *LBFactoryFilterer) ParsePresetRemoved(log types.Log) (*LBFactoryPresetRemoved, error) {
	event := new(LBFactoryPresetRemoved)
	if err := _LBFactory.contract.UnpackLog(event, "PresetRemoved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBFactoryPresetSetIterator is returned from FilterPresetSet and is used to iterate over the raw logs and unpacked data for PresetSet events raised by the LBFactory contract.
type LBFactoryPresetSetIterator struct {
	Event *LBFactoryPresetSet // Event containing the contract specifics and raw log

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
func (it *LBFactoryPresetSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBFactoryPresetSet)
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
		it.Event = new(LBFactoryPresetSet)
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
func (it *LBFactoryPresetSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBFactoryPresetSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBFactoryPresetSet represents a PresetSet event raised by the LBFactory contract.
type LBFactoryPresetSet struct {
	BinStep                  *big.Int
	BaseFactor               *big.Int
	FilterPeriod             *big.Int
	DecayPeriod              *big.Int
	ReductionFactor          *big.Int
	VariableFeeControl       *big.Int
	ProtocolShare            *big.Int
	MaxVolatilityAccumulator *big.Int
	Raw                      types.Log // Blockchain specific contextual infos
}

// FilterPresetSet is a free log retrieval operation binding the contract event 0x839844a256a87f87c9c835117d9a1c40be013954064c937072acb32d36db6a28.
//
// Solidity: event PresetSet(uint256 indexed binStep, uint256 baseFactor, uint256 filterPeriod, uint256 decayPeriod, uint256 reductionFactor, uint256 variableFeeControl, uint256 protocolShare, uint256 maxVolatilityAccumulator)
func (_LBFactory *LBFactoryFilterer) FilterPresetSet(opts *bind.FilterOpts, binStep []*big.Int) (*LBFactoryPresetSetIterator, error) {

	var binStepRule []any
	for _, binStepItem := range binStep {
		binStepRule = append(binStepRule, binStepItem)
	}

	logs, sub, err := _LBFactory.contract.FilterLogs(opts, "PresetSet", binStepRule)
	if err != nil {
		return nil, err
	}
	return &LBFactoryPresetSetIterator{contract: _LBFactory.contract, event: "PresetSet", logs: logs, sub: sub}, nil
}

// WatchPresetSet is a free log subscription operation binding the contract event 0x839844a256a87f87c9c835117d9a1c40be013954064c937072acb32d36db6a28.
//
// Solidity: event PresetSet(uint256 indexed binStep, uint256 baseFactor, uint256 filterPeriod, uint256 decayPeriod, uint256 reductionFactor, uint256 variableFeeControl, uint256 protocolShare, uint256 maxVolatilityAccumulator)
func (_LBFactory *LBFactoryFilterer) WatchPresetSet(opts *bind.WatchOpts, sink chan<- *LBFactoryPresetSet, binStep []*big.Int) (event.Subscription, error) {

	var binStepRule []any
	for _, binStepItem := range binStep {
		binStepRule = append(binStepRule, binStepItem)
	}

	logs, sub, err := _LBFactory.contract.WatchLogs(opts, "PresetSet", binStepRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBFactoryPresetSet)
				if err := _LBFactory.contract.UnpackLog(event, "PresetSet", log); err != nil {
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

// ParsePresetSet is a log parse operation binding the contract event 0x839844a256a87f87c9c835117d9a1c40be013954064c937072acb32d36db6a28.
//
// Solidity: event PresetSet(uint256 indexed binStep, uint256 baseFactor, uint256 filterPeriod, uint256 decayPeriod, uint256 reductionFactor, uint256 variableFeeControl, uint256 protocolShare, uint256 maxVolatilityAccumulator)
func (_LBFactory *LBFactoryFilterer) ParsePresetSet(log types.Log) (*LBFactoryPresetSet, error) {
	event := new(LBFactoryPresetSet)
	if err := _LBFactory.contract.UnpackLog(event, "PresetSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBFactoryQuoteAssetAddedIterator is returned from FilterQuoteAssetAdded and is used to iterate over the raw logs and unpacked data for QuoteAssetAdded events raised by the LBFactory contract.
type LBFactoryQuoteAssetAddedIterator struct {
	Event *LBFactoryQuoteAssetAdded // Event containing the contract specifics and raw log

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
func (it *LBFactoryQuoteAssetAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBFactoryQuoteAssetAdded)
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
		it.Event = new(LBFactoryQuoteAssetAdded)
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
func (it *LBFactoryQuoteAssetAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBFactoryQuoteAssetAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBFactoryQuoteAssetAdded represents a QuoteAssetAdded event raised by the LBFactory contract.
type LBFactoryQuoteAssetAdded struct {
	QuoteAsset common.Address
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterQuoteAssetAdded is a free log retrieval operation binding the contract event 0x84cc2115995684dcb0cd3d3a9565e3d32f075de81db70c8dc3a719b2a47af67e.
//
// Solidity: event QuoteAssetAdded(address indexed quoteAsset)
func (_LBFactory *LBFactoryFilterer) FilterQuoteAssetAdded(opts *bind.FilterOpts, quoteAsset []common.Address) (*LBFactoryQuoteAssetAddedIterator, error) {

	var quoteAssetRule []any
	for _, quoteAssetItem := range quoteAsset {
		quoteAssetRule = append(quoteAssetRule, quoteAssetItem)
	}

	logs, sub, err := _LBFactory.contract.FilterLogs(opts, "QuoteAssetAdded", quoteAssetRule)
	if err != nil {
		return nil, err
	}
	return &LBFactoryQuoteAssetAddedIterator{contract: _LBFactory.contract, event: "QuoteAssetAdded", logs: logs, sub: sub}, nil
}

// WatchQuoteAssetAdded is a free log subscription operation binding the contract event 0x84cc2115995684dcb0cd3d3a9565e3d32f075de81db70c8dc3a719b2a47af67e.
//
// Solidity: event QuoteAssetAdded(address indexed quoteAsset)
func (_LBFactory *LBFactoryFilterer) WatchQuoteAssetAdded(opts *bind.WatchOpts, sink chan<- *LBFactoryQuoteAssetAdded, quoteAsset []common.Address) (event.Subscription, error) {

	var quoteAssetRule []any
	for _, quoteAssetItem := range quoteAsset {
		quoteAssetRule = append(quoteAssetRule, quoteAssetItem)
	}

	logs, sub, err := _LBFactory.contract.WatchLogs(opts, "QuoteAssetAdded", quoteAssetRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBFactoryQuoteAssetAdded)
				if err := _LBFactory.contract.UnpackLog(event, "QuoteAssetAdded", log); err != nil {
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

// ParseQuoteAssetAdded is a log parse operation binding the contract event 0x84cc2115995684dcb0cd3d3a9565e3d32f075de81db70c8dc3a719b2a47af67e.
//
// Solidity: event QuoteAssetAdded(address indexed quoteAsset)
func (_LBFactory *LBFactoryFilterer) ParseQuoteAssetAdded(log types.Log) (*LBFactoryQuoteAssetAdded, error) {
	event := new(LBFactoryQuoteAssetAdded)
	if err := _LBFactory.contract.UnpackLog(event, "QuoteAssetAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBFactoryQuoteAssetRemovedIterator is returned from FilterQuoteAssetRemoved and is used to iterate over the raw logs and unpacked data for QuoteAssetRemoved events raised by the LBFactory contract.
type LBFactoryQuoteAssetRemovedIterator struct {
	Event *LBFactoryQuoteAssetRemoved // Event containing the contract specifics and raw log

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
func (it *LBFactoryQuoteAssetRemovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBFactoryQuoteAssetRemoved)
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
		it.Event = new(LBFactoryQuoteAssetRemoved)
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
func (it *LBFactoryQuoteAssetRemovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBFactoryQuoteAssetRemovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBFactoryQuoteAssetRemoved represents a QuoteAssetRemoved event raised by the LBFactory contract.
type LBFactoryQuoteAssetRemoved struct {
	QuoteAsset common.Address
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterQuoteAssetRemoved is a free log retrieval operation binding the contract event 0x0b767739217755d8af5a2ba75b181a19fa1750f8bb701f09311cb19a90140cb3.
//
// Solidity: event QuoteAssetRemoved(address indexed quoteAsset)
func (_LBFactory *LBFactoryFilterer) FilterQuoteAssetRemoved(opts *bind.FilterOpts, quoteAsset []common.Address) (*LBFactoryQuoteAssetRemovedIterator, error) {

	var quoteAssetRule []any
	for _, quoteAssetItem := range quoteAsset {
		quoteAssetRule = append(quoteAssetRule, quoteAssetItem)
	}

	logs, sub, err := _LBFactory.contract.FilterLogs(opts, "QuoteAssetRemoved", quoteAssetRule)
	if err != nil {
		return nil, err
	}
	return &LBFactoryQuoteAssetRemovedIterator{contract: _LBFactory.contract, event: "QuoteAssetRemoved", logs: logs, sub: sub}, nil
}

// WatchQuoteAssetRemoved is a free log subscription operation binding the contract event 0x0b767739217755d8af5a2ba75b181a19fa1750f8bb701f09311cb19a90140cb3.
//
// Solidity: event QuoteAssetRemoved(address indexed quoteAsset)
func (_LBFactory *LBFactoryFilterer) WatchQuoteAssetRemoved(opts *bind.WatchOpts, sink chan<- *LBFactoryQuoteAssetRemoved, quoteAsset []common.Address) (event.Subscription, error) {

	var quoteAssetRule []any
	for _, quoteAssetItem := range quoteAsset {
		quoteAssetRule = append(quoteAssetRule, quoteAssetItem)
	}

	logs, sub, err := _LBFactory.contract.WatchLogs(opts, "QuoteAssetRemoved", quoteAssetRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBFactoryQuoteAssetRemoved)
				if err := _LBFactory.contract.UnpackLog(event, "QuoteAssetRemoved", log); err != nil {
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

// ParseQuoteAssetRemoved is a log parse operation binding the contract event 0x0b767739217755d8af5a2ba75b181a19fa1750f8bb701f09311cb19a90140cb3.
//
// Solidity: event QuoteAssetRemoved(address indexed quoteAsset)
func (_LBFactory *LBFactoryFilterer) ParseQuoteAssetRemoved(log types.Log) (*LBFactoryQuoteAssetRemoved, error) {
	event := new(LBFactoryQuoteAssetRemoved)
	if err := _LBFactory.contract.UnpackLog(event, "QuoteAssetRemoved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
