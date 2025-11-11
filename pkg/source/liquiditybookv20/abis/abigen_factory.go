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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_feeRecipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_flashLoanFee\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"bp\",\"type\":\"uint256\"}],\"name\":\"BinHelper__BinStepOverflows\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BinHelper__IdOverflows\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBFactory__AddressZero\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"binStep\",\"type\":\"uint256\"}],\"name\":\"LBFactory__BinStepHasNoPreset\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"lowerBound\",\"type\":\"uint256\"},{\"internalType\":\"uint16\",\"name\":\"binStep\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"higherBound\",\"type\":\"uint256\"}],\"name\":\"LBFactory__BinStepRequirementsBreached\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"filterPeriod\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"decayPeriod\",\"type\":\"uint16\"}],\"name\":\"LBFactory__DecreasingPeriods\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBFactory__FactoryLockIsAlreadyInTheSameState\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"fees\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFees\",\"type\":\"uint256\"}],\"name\":\"LBFactory__FeesAboveMax\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"fees\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFees\",\"type\":\"uint256\"}],\"name\":\"LBFactory__FlashLoanFeeAboveMax\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"}],\"name\":\"LBFactory__FunctionIsLockedForUsers\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"token\",\"type\":\"address\"}],\"name\":\"LBFactory__IdenticalAddresses\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBFactory__ImplementationNotSet\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"tokenX\",\"type\":\"address\"},{\"internalType\":\"contractIERC20\",\"name\":\"tokenY\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_binStep\",\"type\":\"uint256\"}],\"name\":\"LBFactory__LBPairAlreadyExists\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBFactory__LBPairIgnoredIsAlreadyInTheSameState\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"tokenX\",\"type\":\"address\"},{\"internalType\":\"contractIERC20\",\"name\":\"tokenY\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"binStep\",\"type\":\"uint256\"}],\"name\":\"LBFactory__LBPairNotCreated\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"LBPairImplementation\",\"type\":\"address\"}],\"name\":\"LBFactory__LBPairSafetyCheckFailed\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"protocolShare\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"max\",\"type\":\"uint256\"}],\"name\":\"LBFactory__ProtocolShareOverflows\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"quoteAsset\",\"type\":\"address\"}],\"name\":\"LBFactory__QuoteAssetAlreadyWhitelisted\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"quoteAsset\",\"type\":\"address\"}],\"name\":\"LBFactory__QuoteAssetNotWhitelisted\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"reductionFactor\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"max\",\"type\":\"uint256\"}],\"name\":\"LBFactory__ReductionFactorOverflows\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"feeRecipient\",\"type\":\"address\"}],\"name\":\"LBFactory__SameFeeRecipient\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"flashLoanFee\",\"type\":\"uint256\"}],\"name\":\"LBFactory__SameFlashLoanFee\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"LBPairImplementation\",\"type\":\"address\"}],\"name\":\"LBFactory__SameImplementation\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"},{\"internalType\":\"int256\",\"name\":\"y\",\"type\":\"int256\"}],\"name\":\"Math128x128__PowerUnderflow\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PendingOwnable__AddressZero\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PendingOwnable__NoPendingOwner\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PendingOwnable__NotOwner\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PendingOwnable__NotPendingOwner\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PendingOwnable__PendingOwnerAlreadySet\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"}],\"name\":\"SafeCast__Exceeds16Bits\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"unlocked\",\"type\":\"bool\"}],\"name\":\"FactoryLockedStatusUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"contractILBPair\",\"name\":\"LBPair\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"binStep\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"baseFactor\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"filterPeriod\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"decayPeriod\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"reductionFactor\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"variableFeeControl\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"protocolShare\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"maxVolatilityAccumulated\",\"type\":\"uint256\"}],\"name\":\"FeeParametersSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"oldRecipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newRecipient\",\"type\":\"address\"}],\"name\":\"FeeRecipientSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"oldFlashLoanFee\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newFlashLoanFee\",\"type\":\"uint256\"}],\"name\":\"FlashLoanFeeSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"contractIERC20\",\"name\":\"tokenX\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"contractIERC20\",\"name\":\"tokenY\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"binStep\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"contractILBPair\",\"name\":\"LBPair\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"pid\",\"type\":\"uint256\"}],\"name\":\"LBPairCreated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"contractILBPair\",\"name\":\"LBPair\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"ignored\",\"type\":\"bool\"}],\"name\":\"LBPairIgnoredStateChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"oldLBPairImplementation\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"LBPairImplementation\",\"type\":\"address\"}],\"name\":\"LBPairImplementationSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"pendingOwner\",\"type\":\"address\"}],\"name\":\"PendingOwnerSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"binStep\",\"type\":\"uint256\"}],\"name\":\"PresetRemoved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"binStep\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"baseFactor\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"filterPeriod\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"decayPeriod\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"reductionFactor\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"variableFeeControl\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"protocolShare\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"maxVolatilityAccumulated\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"sampleLifetime\",\"type\":\"uint256\"}],\"name\":\"PresetSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"contractIERC20\",\"name\":\"quoteAsset\",\"type\":\"address\"}],\"name\":\"QuoteAssetAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"contractIERC20\",\"name\":\"quoteAsset\",\"type\":\"address\"}],\"name\":\"QuoteAssetRemoved\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"LBPairImplementation\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MAX_BIN_STEP\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MAX_FEE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MAX_PROTOCOL_SHARE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_BIN_STEP\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"_quoteAsset\",\"type\":\"address\"}],\"name\":\"addQuoteAsset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allLBPairs\",\"outputs\":[{\"internalType\":\"contractILBPair\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"becomeOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"_tokenX\",\"type\":\"address\"},{\"internalType\":\"contractIERC20\",\"name\":\"_tokenY\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"_activeId\",\"type\":\"uint24\"},{\"internalType\":\"uint16\",\"name\":\"_binStep\",\"type\":\"uint16\"}],\"name\":\"createLBPair\",\"outputs\":[{\"internalType\":\"contractILBPair\",\"name\":\"_LBPair\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"creationUnlocked\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"feeRecipient\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"flashLoanFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractILBPair\",\"name\":\"_LBPair\",\"type\":\"address\"}],\"name\":\"forceDecay\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getAllBinSteps\",\"outputs\":[{\"internalType\":\"uint256[]\",\"name\":\"presetsBinStep\",\"type\":\"uint256[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"_tokenX\",\"type\":\"address\"},{\"internalType\":\"contractIERC20\",\"name\":\"_tokenY\",\"type\":\"address\"}],\"name\":\"getAllLBPairs\",\"outputs\":[{\"components\":[{\"internalType\":\"uint16\",\"name\":\"binStep\",\"type\":\"uint16\"},{\"internalType\":\"contractILBPair\",\"name\":\"LBPair\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"createdByOwner\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"ignoredForRouting\",\"type\":\"bool\"}],\"internalType\":\"structILBFactory.LBPairInformation[]\",\"name\":\"LBPairsAvailable\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"_tokenA\",\"type\":\"address\"},{\"internalType\":\"contractIERC20\",\"name\":\"_tokenB\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_binStep\",\"type\":\"uint256\"}],\"name\":\"getLBPairInformation\",\"outputs\":[{\"components\":[{\"internalType\":\"uint16\",\"name\":\"binStep\",\"type\":\"uint16\"},{\"internalType\":\"contractILBPair\",\"name\":\"LBPair\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"createdByOwner\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"ignoredForRouting\",\"type\":\"bool\"}],\"internalType\":\"structILBFactory.LBPairInformation\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNumberOfLBPairs\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNumberOfQuoteAssets\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"_binStep\",\"type\":\"uint16\"}],\"name\":\"getPreset\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"baseFactor\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"filterPeriod\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"decayPeriod\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reductionFactor\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"variableFeeControl\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"protocolShare\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxVolatilityAccumulated\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"sampleLifetime\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_index\",\"type\":\"uint256\"}],\"name\":\"getQuoteAsset\",\"outputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"_token\",\"type\":\"address\"}],\"name\":\"isQuoteAsset\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pendingOwner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"_binStep\",\"type\":\"uint16\"}],\"name\":\"removePreset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"_quoteAsset\",\"type\":\"address\"}],\"name\":\"removeQuoteAsset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"revokePendingOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"_locked\",\"type\":\"bool\"}],\"name\":\"setFactoryLockedState\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_feeRecipient\",\"type\":\"address\"}],\"name\":\"setFeeRecipient\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"_tokenX\",\"type\":\"address\"},{\"internalType\":\"contractIERC20\",\"name\":\"_tokenY\",\"type\":\"address\"},{\"internalType\":\"uint16\",\"name\":\"_binStep\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"_baseFactor\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"_filterPeriod\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"_decayPeriod\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"_reductionFactor\",\"type\":\"uint16\"},{\"internalType\":\"uint24\",\"name\":\"_variableFeeControl\",\"type\":\"uint24\"},{\"internalType\":\"uint16\",\"name\":\"_protocolShare\",\"type\":\"uint16\"},{\"internalType\":\"uint24\",\"name\":\"_maxVolatilityAccumulated\",\"type\":\"uint24\"}],\"name\":\"setFeesParametersOnPair\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_flashLoanFee\",\"type\":\"uint256\"}],\"name\":\"setFlashLoanFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"_tokenX\",\"type\":\"address\"},{\"internalType\":\"contractIERC20\",\"name\":\"_tokenY\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_binStep\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"_ignored\",\"type\":\"bool\"}],\"name\":\"setLBPairIgnored\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_LBPairImplementation\",\"type\":\"address\"}],\"name\":\"setLBPairImplementation\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"pendingOwner_\",\"type\":\"address\"}],\"name\":\"setPendingOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"_binStep\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"_baseFactor\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"_filterPeriod\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"_decayPeriod\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"_reductionFactor\",\"type\":\"uint16\"},{\"internalType\":\"uint24\",\"name\":\"_variableFeeControl\",\"type\":\"uint24\"},{\"internalType\":\"uint16\",\"name\":\"_protocolShare\",\"type\":\"uint16\"},{\"internalType\":\"uint24\",\"name\":\"_maxVolatilityAccumulated\",\"type\":\"uint24\"},{\"internalType\":\"uint16\",\"name\":\"_sampleLifetime\",\"type\":\"uint16\"}],\"name\":\"setPreset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
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
func (_LBFactory *LBFactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LBFactory.Contract.LBFactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LBFactory *LBFactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LBFactory.Contract.LBFactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LBFactory *LBFactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LBFactory.Contract.LBFactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LBFactory *LBFactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LBFactory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LBFactory *LBFactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LBFactory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LBFactory *LBFactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LBFactory.Contract.contract.Transact(opts, method, params...)
}

// LBPairImplementation is a free data retrieval call binding the contract method 0x509ceb90.
//
// Solidity: function LBPairImplementation() view returns(address)
func (_LBFactory *LBFactoryCaller) LBPairImplementation(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "LBPairImplementation")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// LBPairImplementation is a free data retrieval call binding the contract method 0x509ceb90.
//
// Solidity: function LBPairImplementation() view returns(address)
func (_LBFactory *LBFactorySession) LBPairImplementation() (common.Address, error) {
	return _LBFactory.Contract.LBPairImplementation(&_LBFactory.CallOpts)
}

// LBPairImplementation is a free data retrieval call binding the contract method 0x509ceb90.
//
// Solidity: function LBPairImplementation() view returns(address)
func (_LBFactory *LBFactoryCallerSession) LBPairImplementation() (common.Address, error) {
	return _LBFactory.Contract.LBPairImplementation(&_LBFactory.CallOpts)
}

// MAXBINSTEP is a free data retrieval call binding the contract method 0x10e9ec4a.
//
// Solidity: function MAX_BIN_STEP() view returns(uint256)
func (_LBFactory *LBFactoryCaller) MAXBINSTEP(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "MAX_BIN_STEP")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXBINSTEP is a free data retrieval call binding the contract method 0x10e9ec4a.
//
// Solidity: function MAX_BIN_STEP() view returns(uint256)
func (_LBFactory *LBFactorySession) MAXBINSTEP() (*big.Int, error) {
	return _LBFactory.Contract.MAXBINSTEP(&_LBFactory.CallOpts)
}

// MAXBINSTEP is a free data retrieval call binding the contract method 0x10e9ec4a.
//
// Solidity: function MAX_BIN_STEP() view returns(uint256)
func (_LBFactory *LBFactoryCallerSession) MAXBINSTEP() (*big.Int, error) {
	return _LBFactory.Contract.MAXBINSTEP(&_LBFactory.CallOpts)
}

// MAXFEE is a free data retrieval call binding the contract method 0xbc063e1a.
//
// Solidity: function MAX_FEE() view returns(uint256)
func (_LBFactory *LBFactoryCaller) MAXFEE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "MAX_FEE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXFEE is a free data retrieval call binding the contract method 0xbc063e1a.
//
// Solidity: function MAX_FEE() view returns(uint256)
func (_LBFactory *LBFactorySession) MAXFEE() (*big.Int, error) {
	return _LBFactory.Contract.MAXFEE(&_LBFactory.CallOpts)
}

// MAXFEE is a free data retrieval call binding the contract method 0xbc063e1a.
//
// Solidity: function MAX_FEE() view returns(uint256)
func (_LBFactory *LBFactoryCallerSession) MAXFEE() (*big.Int, error) {
	return _LBFactory.Contract.MAXFEE(&_LBFactory.CallOpts)
}

// MAXPROTOCOLSHARE is a free data retrieval call binding the contract method 0xa931208f.
//
// Solidity: function MAX_PROTOCOL_SHARE() view returns(uint256)
func (_LBFactory *LBFactoryCaller) MAXPROTOCOLSHARE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "MAX_PROTOCOL_SHARE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXPROTOCOLSHARE is a free data retrieval call binding the contract method 0xa931208f.
//
// Solidity: function MAX_PROTOCOL_SHARE() view returns(uint256)
func (_LBFactory *LBFactorySession) MAXPROTOCOLSHARE() (*big.Int, error) {
	return _LBFactory.Contract.MAXPROTOCOLSHARE(&_LBFactory.CallOpts)
}

// MAXPROTOCOLSHARE is a free data retrieval call binding the contract method 0xa931208f.
//
// Solidity: function MAX_PROTOCOL_SHARE() view returns(uint256)
func (_LBFactory *LBFactoryCallerSession) MAXPROTOCOLSHARE() (*big.Int, error) {
	return _LBFactory.Contract.MAXPROTOCOLSHARE(&_LBFactory.CallOpts)
}

// MINBINSTEP is a free data retrieval call binding the contract method 0x7df880e3.
//
// Solidity: function MIN_BIN_STEP() view returns(uint256)
func (_LBFactory *LBFactoryCaller) MINBINSTEP(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "MIN_BIN_STEP")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINBINSTEP is a free data retrieval call binding the contract method 0x7df880e3.
//
// Solidity: function MIN_BIN_STEP() view returns(uint256)
func (_LBFactory *LBFactorySession) MINBINSTEP() (*big.Int, error) {
	return _LBFactory.Contract.MINBINSTEP(&_LBFactory.CallOpts)
}

// MINBINSTEP is a free data retrieval call binding the contract method 0x7df880e3.
//
// Solidity: function MIN_BIN_STEP() view returns(uint256)
func (_LBFactory *LBFactoryCallerSession) MINBINSTEP() (*big.Int, error) {
	return _LBFactory.Contract.MINBINSTEP(&_LBFactory.CallOpts)
}

// AllLBPairs is a free data retrieval call binding the contract method 0x72e47b8c.
//
// Solidity: function allLBPairs(uint256 ) view returns(address)
func (_LBFactory *LBFactoryCaller) AllLBPairs(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "allLBPairs", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AllLBPairs is a free data retrieval call binding the contract method 0x72e47b8c.
//
// Solidity: function allLBPairs(uint256 ) view returns(address)
func (_LBFactory *LBFactorySession) AllLBPairs(arg0 *big.Int) (common.Address, error) {
	return _LBFactory.Contract.AllLBPairs(&_LBFactory.CallOpts, arg0)
}

// AllLBPairs is a free data retrieval call binding the contract method 0x72e47b8c.
//
// Solidity: function allLBPairs(uint256 ) view returns(address)
func (_LBFactory *LBFactoryCallerSession) AllLBPairs(arg0 *big.Int) (common.Address, error) {
	return _LBFactory.Contract.AllLBPairs(&_LBFactory.CallOpts, arg0)
}

// CreationUnlocked is a free data retrieval call binding the contract method 0x5c779d6d.
//
// Solidity: function creationUnlocked() view returns(bool)
func (_LBFactory *LBFactoryCaller) CreationUnlocked(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "creationUnlocked")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// CreationUnlocked is a free data retrieval call binding the contract method 0x5c779d6d.
//
// Solidity: function creationUnlocked() view returns(bool)
func (_LBFactory *LBFactorySession) CreationUnlocked() (bool, error) {
	return _LBFactory.Contract.CreationUnlocked(&_LBFactory.CallOpts)
}

// CreationUnlocked is a free data retrieval call binding the contract method 0x5c779d6d.
//
// Solidity: function creationUnlocked() view returns(bool)
func (_LBFactory *LBFactoryCallerSession) CreationUnlocked() (bool, error) {
	return _LBFactory.Contract.CreationUnlocked(&_LBFactory.CallOpts)
}

// FeeRecipient is a free data retrieval call binding the contract method 0x46904840.
//
// Solidity: function feeRecipient() view returns(address)
func (_LBFactory *LBFactoryCaller) FeeRecipient(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "feeRecipient")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FeeRecipient is a free data retrieval call binding the contract method 0x46904840.
//
// Solidity: function feeRecipient() view returns(address)
func (_LBFactory *LBFactorySession) FeeRecipient() (common.Address, error) {
	return _LBFactory.Contract.FeeRecipient(&_LBFactory.CallOpts)
}

// FeeRecipient is a free data retrieval call binding the contract method 0x46904840.
//
// Solidity: function feeRecipient() view returns(address)
func (_LBFactory *LBFactoryCallerSession) FeeRecipient() (common.Address, error) {
	return _LBFactory.Contract.FeeRecipient(&_LBFactory.CallOpts)
}

// FlashLoanFee is a free data retrieval call binding the contract method 0x4847cdc8.
//
// Solidity: function flashLoanFee() view returns(uint256)
func (_LBFactory *LBFactoryCaller) FlashLoanFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "flashLoanFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FlashLoanFee is a free data retrieval call binding the contract method 0x4847cdc8.
//
// Solidity: function flashLoanFee() view returns(uint256)
func (_LBFactory *LBFactorySession) FlashLoanFee() (*big.Int, error) {
	return _LBFactory.Contract.FlashLoanFee(&_LBFactory.CallOpts)
}

// FlashLoanFee is a free data retrieval call binding the contract method 0x4847cdc8.
//
// Solidity: function flashLoanFee() view returns(uint256)
func (_LBFactory *LBFactoryCallerSession) FlashLoanFee() (*big.Int, error) {
	return _LBFactory.Contract.FlashLoanFee(&_LBFactory.CallOpts)
}

// GetAllBinSteps is a free data retrieval call binding the contract method 0x5b35875c.
//
// Solidity: function getAllBinSteps() view returns(uint256[] presetsBinStep)
func (_LBFactory *LBFactoryCaller) GetAllBinSteps(opts *bind.CallOpts) ([]*big.Int, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "getAllBinSteps")

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// GetAllBinSteps is a free data retrieval call binding the contract method 0x5b35875c.
//
// Solidity: function getAllBinSteps() view returns(uint256[] presetsBinStep)
func (_LBFactory *LBFactorySession) GetAllBinSteps() ([]*big.Int, error) {
	return _LBFactory.Contract.GetAllBinSteps(&_LBFactory.CallOpts)
}

// GetAllBinSteps is a free data retrieval call binding the contract method 0x5b35875c.
//
// Solidity: function getAllBinSteps() view returns(uint256[] presetsBinStep)
func (_LBFactory *LBFactoryCallerSession) GetAllBinSteps() ([]*big.Int, error) {
	return _LBFactory.Contract.GetAllBinSteps(&_LBFactory.CallOpts)
}

// GetAllLBPairs is a free data retrieval call binding the contract method 0x6622e0d7.
//
// Solidity: function getAllLBPairs(address _tokenX, address _tokenY) view returns((uint16,address,bool,bool)[] LBPairsAvailable)
func (_LBFactory *LBFactoryCaller) GetAllLBPairs(opts *bind.CallOpts, _tokenX common.Address, _tokenY common.Address) ([]ILBFactoryLBPairInformation, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "getAllLBPairs", _tokenX, _tokenY)

	if err != nil {
		return *new([]ILBFactoryLBPairInformation), err
	}

	out0 := *abi.ConvertType(out[0], new([]ILBFactoryLBPairInformation)).(*[]ILBFactoryLBPairInformation)

	return out0, err

}

// GetAllLBPairs is a free data retrieval call binding the contract method 0x6622e0d7.
//
// Solidity: function getAllLBPairs(address _tokenX, address _tokenY) view returns((uint16,address,bool,bool)[] LBPairsAvailable)
func (_LBFactory *LBFactorySession) GetAllLBPairs(_tokenX common.Address, _tokenY common.Address) ([]ILBFactoryLBPairInformation, error) {
	return _LBFactory.Contract.GetAllLBPairs(&_LBFactory.CallOpts, _tokenX, _tokenY)
}

// GetAllLBPairs is a free data retrieval call binding the contract method 0x6622e0d7.
//
// Solidity: function getAllLBPairs(address _tokenX, address _tokenY) view returns((uint16,address,bool,bool)[] LBPairsAvailable)
func (_LBFactory *LBFactoryCallerSession) GetAllLBPairs(_tokenX common.Address, _tokenY common.Address) ([]ILBFactoryLBPairInformation, error) {
	return _LBFactory.Contract.GetAllLBPairs(&_LBFactory.CallOpts, _tokenX, _tokenY)
}

// GetLBPairInformation is a free data retrieval call binding the contract method 0x704037bd.
//
// Solidity: function getLBPairInformation(address _tokenA, address _tokenB, uint256 _binStep) view returns((uint16,address,bool,bool))
func (_LBFactory *LBFactoryCaller) GetLBPairInformation(opts *bind.CallOpts, _tokenA common.Address, _tokenB common.Address, _binStep *big.Int) (ILBFactoryLBPairInformation, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "getLBPairInformation", _tokenA, _tokenB, _binStep)

	if err != nil {
		return *new(ILBFactoryLBPairInformation), err
	}

	out0 := *abi.ConvertType(out[0], new(ILBFactoryLBPairInformation)).(*ILBFactoryLBPairInformation)

	return out0, err

}

// GetLBPairInformation is a free data retrieval call binding the contract method 0x704037bd.
//
// Solidity: function getLBPairInformation(address _tokenA, address _tokenB, uint256 _binStep) view returns((uint16,address,bool,bool))
func (_LBFactory *LBFactorySession) GetLBPairInformation(_tokenA common.Address, _tokenB common.Address, _binStep *big.Int) (ILBFactoryLBPairInformation, error) {
	return _LBFactory.Contract.GetLBPairInformation(&_LBFactory.CallOpts, _tokenA, _tokenB, _binStep)
}

// GetLBPairInformation is a free data retrieval call binding the contract method 0x704037bd.
//
// Solidity: function getLBPairInformation(address _tokenA, address _tokenB, uint256 _binStep) view returns((uint16,address,bool,bool))
func (_LBFactory *LBFactoryCallerSession) GetLBPairInformation(_tokenA common.Address, _tokenB common.Address, _binStep *big.Int) (ILBFactoryLBPairInformation, error) {
	return _LBFactory.Contract.GetLBPairInformation(&_LBFactory.CallOpts, _tokenA, _tokenB, _binStep)
}

// GetNumberOfLBPairs is a free data retrieval call binding the contract method 0x4e937c3a.
//
// Solidity: function getNumberOfLBPairs() view returns(uint256)
func (_LBFactory *LBFactoryCaller) GetNumberOfLBPairs(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "getNumberOfLBPairs")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNumberOfLBPairs is a free data retrieval call binding the contract method 0x4e937c3a.
//
// Solidity: function getNumberOfLBPairs() view returns(uint256)
func (_LBFactory *LBFactorySession) GetNumberOfLBPairs() (*big.Int, error) {
	return _LBFactory.Contract.GetNumberOfLBPairs(&_LBFactory.CallOpts)
}

// GetNumberOfLBPairs is a free data retrieval call binding the contract method 0x4e937c3a.
//
// Solidity: function getNumberOfLBPairs() view returns(uint256)
func (_LBFactory *LBFactoryCallerSession) GetNumberOfLBPairs() (*big.Int, error) {
	return _LBFactory.Contract.GetNumberOfLBPairs(&_LBFactory.CallOpts)
}

// GetNumberOfQuoteAssets is a free data retrieval call binding the contract method 0x80c5061e.
//
// Solidity: function getNumberOfQuoteAssets() view returns(uint256)
func (_LBFactory *LBFactoryCaller) GetNumberOfQuoteAssets(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "getNumberOfQuoteAssets")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNumberOfQuoteAssets is a free data retrieval call binding the contract method 0x80c5061e.
//
// Solidity: function getNumberOfQuoteAssets() view returns(uint256)
func (_LBFactory *LBFactorySession) GetNumberOfQuoteAssets() (*big.Int, error) {
	return _LBFactory.Contract.GetNumberOfQuoteAssets(&_LBFactory.CallOpts)
}

// GetNumberOfQuoteAssets is a free data retrieval call binding the contract method 0x80c5061e.
//
// Solidity: function getNumberOfQuoteAssets() view returns(uint256)
func (_LBFactory *LBFactoryCallerSession) GetNumberOfQuoteAssets() (*big.Int, error) {
	return _LBFactory.Contract.GetNumberOfQuoteAssets(&_LBFactory.CallOpts)
}

// GetPreset is a free data retrieval call binding the contract method 0x935ea51b.
//
// Solidity: function getPreset(uint16 _binStep) view returns(uint256 baseFactor, uint256 filterPeriod, uint256 decayPeriod, uint256 reductionFactor, uint256 variableFeeControl, uint256 protocolShare, uint256 maxVolatilityAccumulated, uint256 sampleLifetime)
func (_LBFactory *LBFactoryCaller) GetPreset(opts *bind.CallOpts, _binStep uint16) (struct {
	BaseFactor               *big.Int
	FilterPeriod             *big.Int
	DecayPeriod              *big.Int
	ReductionFactor          *big.Int
	VariableFeeControl       *big.Int
	ProtocolShare            *big.Int
	MaxVolatilityAccumulated *big.Int
	SampleLifetime           *big.Int
}, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "getPreset", _binStep)

	outstruct := new(struct {
		BaseFactor               *big.Int
		FilterPeriod             *big.Int
		DecayPeriod              *big.Int
		ReductionFactor          *big.Int
		VariableFeeControl       *big.Int
		ProtocolShare            *big.Int
		MaxVolatilityAccumulated *big.Int
		SampleLifetime           *big.Int
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
	outstruct.MaxVolatilityAccumulated = *abi.ConvertType(out[6], new(*big.Int)).(**big.Int)
	outstruct.SampleLifetime = *abi.ConvertType(out[7], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetPreset is a free data retrieval call binding the contract method 0x935ea51b.
//
// Solidity: function getPreset(uint16 _binStep) view returns(uint256 baseFactor, uint256 filterPeriod, uint256 decayPeriod, uint256 reductionFactor, uint256 variableFeeControl, uint256 protocolShare, uint256 maxVolatilityAccumulated, uint256 sampleLifetime)
func (_LBFactory *LBFactorySession) GetPreset(_binStep uint16) (struct {
	BaseFactor               *big.Int
	FilterPeriod             *big.Int
	DecayPeriod              *big.Int
	ReductionFactor          *big.Int
	VariableFeeControl       *big.Int
	ProtocolShare            *big.Int
	MaxVolatilityAccumulated *big.Int
	SampleLifetime           *big.Int
}, error) {
	return _LBFactory.Contract.GetPreset(&_LBFactory.CallOpts, _binStep)
}

// GetPreset is a free data retrieval call binding the contract method 0x935ea51b.
//
// Solidity: function getPreset(uint16 _binStep) view returns(uint256 baseFactor, uint256 filterPeriod, uint256 decayPeriod, uint256 reductionFactor, uint256 variableFeeControl, uint256 protocolShare, uint256 maxVolatilityAccumulated, uint256 sampleLifetime)
func (_LBFactory *LBFactoryCallerSession) GetPreset(_binStep uint16) (struct {
	BaseFactor               *big.Int
	FilterPeriod             *big.Int
	DecayPeriod              *big.Int
	ReductionFactor          *big.Int
	VariableFeeControl       *big.Int
	ProtocolShare            *big.Int
	MaxVolatilityAccumulated *big.Int
	SampleLifetime           *big.Int
}, error) {
	return _LBFactory.Contract.GetPreset(&_LBFactory.CallOpts, _binStep)
}

// GetQuoteAsset is a free data retrieval call binding the contract method 0xf89a4cd5.
//
// Solidity: function getQuoteAsset(uint256 _index) view returns(address)
func (_LBFactory *LBFactoryCaller) GetQuoteAsset(opts *bind.CallOpts, _index *big.Int) (common.Address, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "getQuoteAsset", _index)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetQuoteAsset is a free data retrieval call binding the contract method 0xf89a4cd5.
//
// Solidity: function getQuoteAsset(uint256 _index) view returns(address)
func (_LBFactory *LBFactorySession) GetQuoteAsset(_index *big.Int) (common.Address, error) {
	return _LBFactory.Contract.GetQuoteAsset(&_LBFactory.CallOpts, _index)
}

// GetQuoteAsset is a free data retrieval call binding the contract method 0xf89a4cd5.
//
// Solidity: function getQuoteAsset(uint256 _index) view returns(address)
func (_LBFactory *LBFactoryCallerSession) GetQuoteAsset(_index *big.Int) (common.Address, error) {
	return _LBFactory.Contract.GetQuoteAsset(&_LBFactory.CallOpts, _index)
}

// IsQuoteAsset is a free data retrieval call binding the contract method 0x27721842.
//
// Solidity: function isQuoteAsset(address _token) view returns(bool)
func (_LBFactory *LBFactoryCaller) IsQuoteAsset(opts *bind.CallOpts, _token common.Address) (bool, error) {
	var out []interface{}
	err := _LBFactory.contract.Call(opts, &out, "isQuoteAsset", _token)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsQuoteAsset is a free data retrieval call binding the contract method 0x27721842.
//
// Solidity: function isQuoteAsset(address _token) view returns(bool)
func (_LBFactory *LBFactorySession) IsQuoteAsset(_token common.Address) (bool, error) {
	return _LBFactory.Contract.IsQuoteAsset(&_LBFactory.CallOpts, _token)
}

// IsQuoteAsset is a free data retrieval call binding the contract method 0x27721842.
//
// Solidity: function isQuoteAsset(address _token) view returns(bool)
func (_LBFactory *LBFactoryCallerSession) IsQuoteAsset(_token common.Address) (bool, error) {
	return _LBFactory.Contract.IsQuoteAsset(&_LBFactory.CallOpts, _token)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_LBFactory *LBFactoryCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
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
	var out []interface{}
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

// AddQuoteAsset is a paid mutator transaction binding the contract method 0x5a440923.
//
// Solidity: function addQuoteAsset(address _quoteAsset) returns()
func (_LBFactory *LBFactoryTransactor) AddQuoteAsset(opts *bind.TransactOpts, _quoteAsset common.Address) (*types.Transaction, error) {
	return _LBFactory.contract.Transact(opts, "addQuoteAsset", _quoteAsset)
}

// AddQuoteAsset is a paid mutator transaction binding the contract method 0x5a440923.
//
// Solidity: function addQuoteAsset(address _quoteAsset) returns()
func (_LBFactory *LBFactorySession) AddQuoteAsset(_quoteAsset common.Address) (*types.Transaction, error) {
	return _LBFactory.Contract.AddQuoteAsset(&_LBFactory.TransactOpts, _quoteAsset)
}

// AddQuoteAsset is a paid mutator transaction binding the contract method 0x5a440923.
//
// Solidity: function addQuoteAsset(address _quoteAsset) returns()
func (_LBFactory *LBFactoryTransactorSession) AddQuoteAsset(_quoteAsset common.Address) (*types.Transaction, error) {
	return _LBFactory.Contract.AddQuoteAsset(&_LBFactory.TransactOpts, _quoteAsset)
}

// BecomeOwner is a paid mutator transaction binding the contract method 0xf9dca989.
//
// Solidity: function becomeOwner() returns()
func (_LBFactory *LBFactoryTransactor) BecomeOwner(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LBFactory.contract.Transact(opts, "becomeOwner")
}

// BecomeOwner is a paid mutator transaction binding the contract method 0xf9dca989.
//
// Solidity: function becomeOwner() returns()
func (_LBFactory *LBFactorySession) BecomeOwner() (*types.Transaction, error) {
	return _LBFactory.Contract.BecomeOwner(&_LBFactory.TransactOpts)
}

// BecomeOwner is a paid mutator transaction binding the contract method 0xf9dca989.
//
// Solidity: function becomeOwner() returns()
func (_LBFactory *LBFactoryTransactorSession) BecomeOwner() (*types.Transaction, error) {
	return _LBFactory.Contract.BecomeOwner(&_LBFactory.TransactOpts)
}

// CreateLBPair is a paid mutator transaction binding the contract method 0x659ac74b.
//
// Solidity: function createLBPair(address _tokenX, address _tokenY, uint24 _activeId, uint16 _binStep) returns(address _LBPair)
func (_LBFactory *LBFactoryTransactor) CreateLBPair(opts *bind.TransactOpts, _tokenX common.Address, _tokenY common.Address, _activeId *big.Int, _binStep uint16) (*types.Transaction, error) {
	return _LBFactory.contract.Transact(opts, "createLBPair", _tokenX, _tokenY, _activeId, _binStep)
}

// CreateLBPair is a paid mutator transaction binding the contract method 0x659ac74b.
//
// Solidity: function createLBPair(address _tokenX, address _tokenY, uint24 _activeId, uint16 _binStep) returns(address _LBPair)
func (_LBFactory *LBFactorySession) CreateLBPair(_tokenX common.Address, _tokenY common.Address, _activeId *big.Int, _binStep uint16) (*types.Transaction, error) {
	return _LBFactory.Contract.CreateLBPair(&_LBFactory.TransactOpts, _tokenX, _tokenY, _activeId, _binStep)
}

// CreateLBPair is a paid mutator transaction binding the contract method 0x659ac74b.
//
// Solidity: function createLBPair(address _tokenX, address _tokenY, uint24 _activeId, uint16 _binStep) returns(address _LBPair)
func (_LBFactory *LBFactoryTransactorSession) CreateLBPair(_tokenX common.Address, _tokenY common.Address, _activeId *big.Int, _binStep uint16) (*types.Transaction, error) {
	return _LBFactory.Contract.CreateLBPair(&_LBFactory.TransactOpts, _tokenX, _tokenY, _activeId, _binStep)
}

// ForceDecay is a paid mutator transaction binding the contract method 0x3c78a941.
//
// Solidity: function forceDecay(address _LBPair) returns()
func (_LBFactory *LBFactoryTransactor) ForceDecay(opts *bind.TransactOpts, _LBPair common.Address) (*types.Transaction, error) {
	return _LBFactory.contract.Transact(opts, "forceDecay", _LBPair)
}

// ForceDecay is a paid mutator transaction binding the contract method 0x3c78a941.
//
// Solidity: function forceDecay(address _LBPair) returns()
func (_LBFactory *LBFactorySession) ForceDecay(_LBPair common.Address) (*types.Transaction, error) {
	return _LBFactory.Contract.ForceDecay(&_LBFactory.TransactOpts, _LBPair)
}

// ForceDecay is a paid mutator transaction binding the contract method 0x3c78a941.
//
// Solidity: function forceDecay(address _LBPair) returns()
func (_LBFactory *LBFactoryTransactorSession) ForceDecay(_LBPair common.Address) (*types.Transaction, error) {
	return _LBFactory.Contract.ForceDecay(&_LBFactory.TransactOpts, _LBPair)
}

// RemovePreset is a paid mutator transaction binding the contract method 0xe203a31f.
//
// Solidity: function removePreset(uint16 _binStep) returns()
func (_LBFactory *LBFactoryTransactor) RemovePreset(opts *bind.TransactOpts, _binStep uint16) (*types.Transaction, error) {
	return _LBFactory.contract.Transact(opts, "removePreset", _binStep)
}

// RemovePreset is a paid mutator transaction binding the contract method 0xe203a31f.
//
// Solidity: function removePreset(uint16 _binStep) returns()
func (_LBFactory *LBFactorySession) RemovePreset(_binStep uint16) (*types.Transaction, error) {
	return _LBFactory.Contract.RemovePreset(&_LBFactory.TransactOpts, _binStep)
}

// RemovePreset is a paid mutator transaction binding the contract method 0xe203a31f.
//
// Solidity: function removePreset(uint16 _binStep) returns()
func (_LBFactory *LBFactoryTransactorSession) RemovePreset(_binStep uint16) (*types.Transaction, error) {
	return _LBFactory.Contract.RemovePreset(&_LBFactory.TransactOpts, _binStep)
}

// RemoveQuoteAsset is a paid mutator transaction binding the contract method 0xddbfd941.
//
// Solidity: function removeQuoteAsset(address _quoteAsset) returns()
func (_LBFactory *LBFactoryTransactor) RemoveQuoteAsset(opts *bind.TransactOpts, _quoteAsset common.Address) (*types.Transaction, error) {
	return _LBFactory.contract.Transact(opts, "removeQuoteAsset", _quoteAsset)
}

// RemoveQuoteAsset is a paid mutator transaction binding the contract method 0xddbfd941.
//
// Solidity: function removeQuoteAsset(address _quoteAsset) returns()
func (_LBFactory *LBFactorySession) RemoveQuoteAsset(_quoteAsset common.Address) (*types.Transaction, error) {
	return _LBFactory.Contract.RemoveQuoteAsset(&_LBFactory.TransactOpts, _quoteAsset)
}

// RemoveQuoteAsset is a paid mutator transaction binding the contract method 0xddbfd941.
//
// Solidity: function removeQuoteAsset(address _quoteAsset) returns()
func (_LBFactory *LBFactoryTransactorSession) RemoveQuoteAsset(_quoteAsset common.Address) (*types.Transaction, error) {
	return _LBFactory.Contract.RemoveQuoteAsset(&_LBFactory.TransactOpts, _quoteAsset)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_LBFactory *LBFactoryTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LBFactory.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_LBFactory *LBFactorySession) RenounceOwnership() (*types.Transaction, error) {
	return _LBFactory.Contract.RenounceOwnership(&_LBFactory.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_LBFactory *LBFactoryTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _LBFactory.Contract.RenounceOwnership(&_LBFactory.TransactOpts)
}

// RevokePendingOwner is a paid mutator transaction binding the contract method 0x67ab8a4e.
//
// Solidity: function revokePendingOwner() returns()
func (_LBFactory *LBFactoryTransactor) RevokePendingOwner(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LBFactory.contract.Transact(opts, "revokePendingOwner")
}

// RevokePendingOwner is a paid mutator transaction binding the contract method 0x67ab8a4e.
//
// Solidity: function revokePendingOwner() returns()
func (_LBFactory *LBFactorySession) RevokePendingOwner() (*types.Transaction, error) {
	return _LBFactory.Contract.RevokePendingOwner(&_LBFactory.TransactOpts)
}

// RevokePendingOwner is a paid mutator transaction binding the contract method 0x67ab8a4e.
//
// Solidity: function revokePendingOwner() returns()
func (_LBFactory *LBFactoryTransactorSession) RevokePendingOwner() (*types.Transaction, error) {
	return _LBFactory.Contract.RevokePendingOwner(&_LBFactory.TransactOpts)
}

// SetFactoryLockedState is a paid mutator transaction binding the contract method 0x22f3fe14.
//
// Solidity: function setFactoryLockedState(bool _locked) returns()
func (_LBFactory *LBFactoryTransactor) SetFactoryLockedState(opts *bind.TransactOpts, _locked bool) (*types.Transaction, error) {
	return _LBFactory.contract.Transact(opts, "setFactoryLockedState", _locked)
}

// SetFactoryLockedState is a paid mutator transaction binding the contract method 0x22f3fe14.
//
// Solidity: function setFactoryLockedState(bool _locked) returns()
func (_LBFactory *LBFactorySession) SetFactoryLockedState(_locked bool) (*types.Transaction, error) {
	return _LBFactory.Contract.SetFactoryLockedState(&_LBFactory.TransactOpts, _locked)
}

// SetFactoryLockedState is a paid mutator transaction binding the contract method 0x22f3fe14.
//
// Solidity: function setFactoryLockedState(bool _locked) returns()
func (_LBFactory *LBFactoryTransactorSession) SetFactoryLockedState(_locked bool) (*types.Transaction, error) {
	return _LBFactory.Contract.SetFactoryLockedState(&_LBFactory.TransactOpts, _locked)
}

// SetFeeRecipient is a paid mutator transaction binding the contract method 0xe74b981b.
//
// Solidity: function setFeeRecipient(address _feeRecipient) returns()
func (_LBFactory *LBFactoryTransactor) SetFeeRecipient(opts *bind.TransactOpts, _feeRecipient common.Address) (*types.Transaction, error) {
	return _LBFactory.contract.Transact(opts, "setFeeRecipient", _feeRecipient)
}

// SetFeeRecipient is a paid mutator transaction binding the contract method 0xe74b981b.
//
// Solidity: function setFeeRecipient(address _feeRecipient) returns()
func (_LBFactory *LBFactorySession) SetFeeRecipient(_feeRecipient common.Address) (*types.Transaction, error) {
	return _LBFactory.Contract.SetFeeRecipient(&_LBFactory.TransactOpts, _feeRecipient)
}

// SetFeeRecipient is a paid mutator transaction binding the contract method 0xe74b981b.
//
// Solidity: function setFeeRecipient(address _feeRecipient) returns()
func (_LBFactory *LBFactoryTransactorSession) SetFeeRecipient(_feeRecipient common.Address) (*types.Transaction, error) {
	return _LBFactory.Contract.SetFeeRecipient(&_LBFactory.TransactOpts, _feeRecipient)
}

// SetFeesParametersOnPair is a paid mutator transaction binding the contract method 0x093ff769.
//
// Solidity: function setFeesParametersOnPair(address _tokenX, address _tokenY, uint16 _binStep, uint16 _baseFactor, uint16 _filterPeriod, uint16 _decayPeriod, uint16 _reductionFactor, uint24 _variableFeeControl, uint16 _protocolShare, uint24 _maxVolatilityAccumulated) returns()
func (_LBFactory *LBFactoryTransactor) SetFeesParametersOnPair(opts *bind.TransactOpts, _tokenX common.Address, _tokenY common.Address, _binStep uint16, _baseFactor uint16, _filterPeriod uint16, _decayPeriod uint16, _reductionFactor uint16, _variableFeeControl *big.Int, _protocolShare uint16, _maxVolatilityAccumulated *big.Int) (*types.Transaction, error) {
	return _LBFactory.contract.Transact(opts, "setFeesParametersOnPair", _tokenX, _tokenY, _binStep, _baseFactor, _filterPeriod, _decayPeriod, _reductionFactor, _variableFeeControl, _protocolShare, _maxVolatilityAccumulated)
}

// SetFeesParametersOnPair is a paid mutator transaction binding the contract method 0x093ff769.
//
// Solidity: function setFeesParametersOnPair(address _tokenX, address _tokenY, uint16 _binStep, uint16 _baseFactor, uint16 _filterPeriod, uint16 _decayPeriod, uint16 _reductionFactor, uint24 _variableFeeControl, uint16 _protocolShare, uint24 _maxVolatilityAccumulated) returns()
func (_LBFactory *LBFactorySession) SetFeesParametersOnPair(_tokenX common.Address, _tokenY common.Address, _binStep uint16, _baseFactor uint16, _filterPeriod uint16, _decayPeriod uint16, _reductionFactor uint16, _variableFeeControl *big.Int, _protocolShare uint16, _maxVolatilityAccumulated *big.Int) (*types.Transaction, error) {
	return _LBFactory.Contract.SetFeesParametersOnPair(&_LBFactory.TransactOpts, _tokenX, _tokenY, _binStep, _baseFactor, _filterPeriod, _decayPeriod, _reductionFactor, _variableFeeControl, _protocolShare, _maxVolatilityAccumulated)
}

// SetFeesParametersOnPair is a paid mutator transaction binding the contract method 0x093ff769.
//
// Solidity: function setFeesParametersOnPair(address _tokenX, address _tokenY, uint16 _binStep, uint16 _baseFactor, uint16 _filterPeriod, uint16 _decayPeriod, uint16 _reductionFactor, uint24 _variableFeeControl, uint16 _protocolShare, uint24 _maxVolatilityAccumulated) returns()
func (_LBFactory *LBFactoryTransactorSession) SetFeesParametersOnPair(_tokenX common.Address, _tokenY common.Address, _binStep uint16, _baseFactor uint16, _filterPeriod uint16, _decayPeriod uint16, _reductionFactor uint16, _variableFeeControl *big.Int, _protocolShare uint16, _maxVolatilityAccumulated *big.Int) (*types.Transaction, error) {
	return _LBFactory.Contract.SetFeesParametersOnPair(&_LBFactory.TransactOpts, _tokenX, _tokenY, _binStep, _baseFactor, _filterPeriod, _decayPeriod, _reductionFactor, _variableFeeControl, _protocolShare, _maxVolatilityAccumulated)
}

// SetFlashLoanFee is a paid mutator transaction binding the contract method 0xe92d0d5d.
//
// Solidity: function setFlashLoanFee(uint256 _flashLoanFee) returns()
func (_LBFactory *LBFactoryTransactor) SetFlashLoanFee(opts *bind.TransactOpts, _flashLoanFee *big.Int) (*types.Transaction, error) {
	return _LBFactory.contract.Transact(opts, "setFlashLoanFee", _flashLoanFee)
}

// SetFlashLoanFee is a paid mutator transaction binding the contract method 0xe92d0d5d.
//
// Solidity: function setFlashLoanFee(uint256 _flashLoanFee) returns()
func (_LBFactory *LBFactorySession) SetFlashLoanFee(_flashLoanFee *big.Int) (*types.Transaction, error) {
	return _LBFactory.Contract.SetFlashLoanFee(&_LBFactory.TransactOpts, _flashLoanFee)
}

// SetFlashLoanFee is a paid mutator transaction binding the contract method 0xe92d0d5d.
//
// Solidity: function setFlashLoanFee(uint256 _flashLoanFee) returns()
func (_LBFactory *LBFactoryTransactorSession) SetFlashLoanFee(_flashLoanFee *big.Int) (*types.Transaction, error) {
	return _LBFactory.Contract.SetFlashLoanFee(&_LBFactory.TransactOpts, _flashLoanFee)
}

// SetLBPairIgnored is a paid mutator transaction binding the contract method 0x200aa7e3.
//
// Solidity: function setLBPairIgnored(address _tokenX, address _tokenY, uint256 _binStep, bool _ignored) returns()
func (_LBFactory *LBFactoryTransactor) SetLBPairIgnored(opts *bind.TransactOpts, _tokenX common.Address, _tokenY common.Address, _binStep *big.Int, _ignored bool) (*types.Transaction, error) {
	return _LBFactory.contract.Transact(opts, "setLBPairIgnored", _tokenX, _tokenY, _binStep, _ignored)
}

// SetLBPairIgnored is a paid mutator transaction binding the contract method 0x200aa7e3.
//
// Solidity: function setLBPairIgnored(address _tokenX, address _tokenY, uint256 _binStep, bool _ignored) returns()
func (_LBFactory *LBFactorySession) SetLBPairIgnored(_tokenX common.Address, _tokenY common.Address, _binStep *big.Int, _ignored bool) (*types.Transaction, error) {
	return _LBFactory.Contract.SetLBPairIgnored(&_LBFactory.TransactOpts, _tokenX, _tokenY, _binStep, _ignored)
}

// SetLBPairIgnored is a paid mutator transaction binding the contract method 0x200aa7e3.
//
// Solidity: function setLBPairIgnored(address _tokenX, address _tokenY, uint256 _binStep, bool _ignored) returns()
func (_LBFactory *LBFactoryTransactorSession) SetLBPairIgnored(_tokenX common.Address, _tokenY common.Address, _binStep *big.Int, _ignored bool) (*types.Transaction, error) {
	return _LBFactory.Contract.SetLBPairIgnored(&_LBFactory.TransactOpts, _tokenX, _tokenY, _binStep, _ignored)
}

// SetLBPairImplementation is a paid mutator transaction binding the contract method 0xb0384781.
//
// Solidity: function setLBPairImplementation(address _LBPairImplementation) returns()
func (_LBFactory *LBFactoryTransactor) SetLBPairImplementation(opts *bind.TransactOpts, _LBPairImplementation common.Address) (*types.Transaction, error) {
	return _LBFactory.contract.Transact(opts, "setLBPairImplementation", _LBPairImplementation)
}

// SetLBPairImplementation is a paid mutator transaction binding the contract method 0xb0384781.
//
// Solidity: function setLBPairImplementation(address _LBPairImplementation) returns()
func (_LBFactory *LBFactorySession) SetLBPairImplementation(_LBPairImplementation common.Address) (*types.Transaction, error) {
	return _LBFactory.Contract.SetLBPairImplementation(&_LBFactory.TransactOpts, _LBPairImplementation)
}

// SetLBPairImplementation is a paid mutator transaction binding the contract method 0xb0384781.
//
// Solidity: function setLBPairImplementation(address _LBPairImplementation) returns()
func (_LBFactory *LBFactoryTransactorSession) SetLBPairImplementation(_LBPairImplementation common.Address) (*types.Transaction, error) {
	return _LBFactory.Contract.SetLBPairImplementation(&_LBFactory.TransactOpts, _LBPairImplementation)
}

// SetPendingOwner is a paid mutator transaction binding the contract method 0xc42069ec.
//
// Solidity: function setPendingOwner(address pendingOwner_) returns()
func (_LBFactory *LBFactoryTransactor) SetPendingOwner(opts *bind.TransactOpts, pendingOwner_ common.Address) (*types.Transaction, error) {
	return _LBFactory.contract.Transact(opts, "setPendingOwner", pendingOwner_)
}

// SetPendingOwner is a paid mutator transaction binding the contract method 0xc42069ec.
//
// Solidity: function setPendingOwner(address pendingOwner_) returns()
func (_LBFactory *LBFactorySession) SetPendingOwner(pendingOwner_ common.Address) (*types.Transaction, error) {
	return _LBFactory.Contract.SetPendingOwner(&_LBFactory.TransactOpts, pendingOwner_)
}

// SetPendingOwner is a paid mutator transaction binding the contract method 0xc42069ec.
//
// Solidity: function setPendingOwner(address pendingOwner_) returns()
func (_LBFactory *LBFactoryTransactorSession) SetPendingOwner(pendingOwner_ common.Address) (*types.Transaction, error) {
	return _LBFactory.Contract.SetPendingOwner(&_LBFactory.TransactOpts, pendingOwner_)
}

// SetPreset is a paid mutator transaction binding the contract method 0x0af97c9a.
//
// Solidity: function setPreset(uint16 _binStep, uint16 _baseFactor, uint16 _filterPeriod, uint16 _decayPeriod, uint16 _reductionFactor, uint24 _variableFeeControl, uint16 _protocolShare, uint24 _maxVolatilityAccumulated, uint16 _sampleLifetime) returns()
func (_LBFactory *LBFactoryTransactor) SetPreset(opts *bind.TransactOpts, _binStep uint16, _baseFactor uint16, _filterPeriod uint16, _decayPeriod uint16, _reductionFactor uint16, _variableFeeControl *big.Int, _protocolShare uint16, _maxVolatilityAccumulated *big.Int, _sampleLifetime uint16) (*types.Transaction, error) {
	return _LBFactory.contract.Transact(opts, "setPreset", _binStep, _baseFactor, _filterPeriod, _decayPeriod, _reductionFactor, _variableFeeControl, _protocolShare, _maxVolatilityAccumulated, _sampleLifetime)
}

// SetPreset is a paid mutator transaction binding the contract method 0x0af97c9a.
//
// Solidity: function setPreset(uint16 _binStep, uint16 _baseFactor, uint16 _filterPeriod, uint16 _decayPeriod, uint16 _reductionFactor, uint24 _variableFeeControl, uint16 _protocolShare, uint24 _maxVolatilityAccumulated, uint16 _sampleLifetime) returns()
func (_LBFactory *LBFactorySession) SetPreset(_binStep uint16, _baseFactor uint16, _filterPeriod uint16, _decayPeriod uint16, _reductionFactor uint16, _variableFeeControl *big.Int, _protocolShare uint16, _maxVolatilityAccumulated *big.Int, _sampleLifetime uint16) (*types.Transaction, error) {
	return _LBFactory.Contract.SetPreset(&_LBFactory.TransactOpts, _binStep, _baseFactor, _filterPeriod, _decayPeriod, _reductionFactor, _variableFeeControl, _protocolShare, _maxVolatilityAccumulated, _sampleLifetime)
}

// SetPreset is a paid mutator transaction binding the contract method 0x0af97c9a.
//
// Solidity: function setPreset(uint16 _binStep, uint16 _baseFactor, uint16 _filterPeriod, uint16 _decayPeriod, uint16 _reductionFactor, uint24 _variableFeeControl, uint16 _protocolShare, uint24 _maxVolatilityAccumulated, uint16 _sampleLifetime) returns()
func (_LBFactory *LBFactoryTransactorSession) SetPreset(_binStep uint16, _baseFactor uint16, _filterPeriod uint16, _decayPeriod uint16, _reductionFactor uint16, _variableFeeControl *big.Int, _protocolShare uint16, _maxVolatilityAccumulated *big.Int, _sampleLifetime uint16) (*types.Transaction, error) {
	return _LBFactory.Contract.SetPreset(&_LBFactory.TransactOpts, _binStep, _baseFactor, _filterPeriod, _decayPeriod, _reductionFactor, _variableFeeControl, _protocolShare, _maxVolatilityAccumulated, _sampleLifetime)
}

// LBFactoryFactoryLockedStatusUpdatedIterator is returned from FilterFactoryLockedStatusUpdated and is used to iterate over the raw logs and unpacked data for FactoryLockedStatusUpdated events raised by the LBFactory contract.
type LBFactoryFactoryLockedStatusUpdatedIterator struct {
	Event *LBFactoryFactoryLockedStatusUpdated // Event containing the contract specifics and raw log

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
func (it *LBFactoryFactoryLockedStatusUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBFactoryFactoryLockedStatusUpdated)
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
		it.Event = new(LBFactoryFactoryLockedStatusUpdated)
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
func (it *LBFactoryFactoryLockedStatusUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBFactoryFactoryLockedStatusUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBFactoryFactoryLockedStatusUpdated represents a FactoryLockedStatusUpdated event raised by the LBFactory contract.
type LBFactoryFactoryLockedStatusUpdated struct {
	Unlocked bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterFactoryLockedStatusUpdated is a free log retrieval operation binding the contract event 0xcdee7bf87b7a743b4cbe1d2d534c5248621b76f58460337e7fda92d5d23f4124.
//
// Solidity: event FactoryLockedStatusUpdated(bool unlocked)
func (_LBFactory *LBFactoryFilterer) FilterFactoryLockedStatusUpdated(opts *bind.FilterOpts) (*LBFactoryFactoryLockedStatusUpdatedIterator, error) {

	logs, sub, err := _LBFactory.contract.FilterLogs(opts, "FactoryLockedStatusUpdated")
	if err != nil {
		return nil, err
	}
	return &LBFactoryFactoryLockedStatusUpdatedIterator{contract: _LBFactory.contract, event: "FactoryLockedStatusUpdated", logs: logs, sub: sub}, nil
}

// WatchFactoryLockedStatusUpdated is a free log subscription operation binding the contract event 0xcdee7bf87b7a743b4cbe1d2d534c5248621b76f58460337e7fda92d5d23f4124.
//
// Solidity: event FactoryLockedStatusUpdated(bool unlocked)
func (_LBFactory *LBFactoryFilterer) WatchFactoryLockedStatusUpdated(opts *bind.WatchOpts, sink chan<- *LBFactoryFactoryLockedStatusUpdated) (event.Subscription, error) {

	logs, sub, err := _LBFactory.contract.WatchLogs(opts, "FactoryLockedStatusUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBFactoryFactoryLockedStatusUpdated)
				if err := _LBFactory.contract.UnpackLog(event, "FactoryLockedStatusUpdated", log); err != nil {
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

// ParseFactoryLockedStatusUpdated is a log parse operation binding the contract event 0xcdee7bf87b7a743b4cbe1d2d534c5248621b76f58460337e7fda92d5d23f4124.
//
// Solidity: event FactoryLockedStatusUpdated(bool unlocked)
func (_LBFactory *LBFactoryFilterer) ParseFactoryLockedStatusUpdated(log types.Log) (*LBFactoryFactoryLockedStatusUpdated, error) {
	event := new(LBFactoryFactoryLockedStatusUpdated)
	if err := _LBFactory.contract.UnpackLog(event, "FactoryLockedStatusUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBFactoryFeeParametersSetIterator is returned from FilterFeeParametersSet and is used to iterate over the raw logs and unpacked data for FeeParametersSet events raised by the LBFactory contract.
type LBFactoryFeeParametersSetIterator struct {
	Event *LBFactoryFeeParametersSet // Event containing the contract specifics and raw log

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
func (it *LBFactoryFeeParametersSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBFactoryFeeParametersSet)
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
		it.Event = new(LBFactoryFeeParametersSet)
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
func (it *LBFactoryFeeParametersSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBFactoryFeeParametersSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBFactoryFeeParametersSet represents a FeeParametersSet event raised by the LBFactory contract.
type LBFactoryFeeParametersSet struct {
	Sender                   common.Address
	LBPair                   common.Address
	BinStep                  *big.Int
	BaseFactor               *big.Int
	FilterPeriod             *big.Int
	DecayPeriod              *big.Int
	ReductionFactor          *big.Int
	VariableFeeControl       *big.Int
	ProtocolShare            *big.Int
	MaxVolatilityAccumulated *big.Int
	Raw                      types.Log // Blockchain specific contextual infos
}

// FilterFeeParametersSet is a free log retrieval operation binding the contract event 0x63a7af39b7b68b9c3f2dfe93e5f32d9faecb4c6c98733bb608f757e62f816c0d.
//
// Solidity: event FeeParametersSet(address indexed sender, address indexed LBPair, uint256 binStep, uint256 baseFactor, uint256 filterPeriod, uint256 decayPeriod, uint256 reductionFactor, uint256 variableFeeControl, uint256 protocolShare, uint256 maxVolatilityAccumulated)
func (_LBFactory *LBFactoryFilterer) FilterFeeParametersSet(opts *bind.FilterOpts, sender []common.Address, LBPair []common.Address) (*LBFactoryFeeParametersSetIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var LBPairRule []interface{}
	for _, LBPairItem := range LBPair {
		LBPairRule = append(LBPairRule, LBPairItem)
	}

	logs, sub, err := _LBFactory.contract.FilterLogs(opts, "FeeParametersSet", senderRule, LBPairRule)
	if err != nil {
		return nil, err
	}
	return &LBFactoryFeeParametersSetIterator{contract: _LBFactory.contract, event: "FeeParametersSet", logs: logs, sub: sub}, nil
}

// WatchFeeParametersSet is a free log subscription operation binding the contract event 0x63a7af39b7b68b9c3f2dfe93e5f32d9faecb4c6c98733bb608f757e62f816c0d.
//
// Solidity: event FeeParametersSet(address indexed sender, address indexed LBPair, uint256 binStep, uint256 baseFactor, uint256 filterPeriod, uint256 decayPeriod, uint256 reductionFactor, uint256 variableFeeControl, uint256 protocolShare, uint256 maxVolatilityAccumulated)
func (_LBFactory *LBFactoryFilterer) WatchFeeParametersSet(opts *bind.WatchOpts, sink chan<- *LBFactoryFeeParametersSet, sender []common.Address, LBPair []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var LBPairRule []interface{}
	for _, LBPairItem := range LBPair {
		LBPairRule = append(LBPairRule, LBPairItem)
	}

	logs, sub, err := _LBFactory.contract.WatchLogs(opts, "FeeParametersSet", senderRule, LBPairRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBFactoryFeeParametersSet)
				if err := _LBFactory.contract.UnpackLog(event, "FeeParametersSet", log); err != nil {
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

// ParseFeeParametersSet is a log parse operation binding the contract event 0x63a7af39b7b68b9c3f2dfe93e5f32d9faecb4c6c98733bb608f757e62f816c0d.
//
// Solidity: event FeeParametersSet(address indexed sender, address indexed LBPair, uint256 binStep, uint256 baseFactor, uint256 filterPeriod, uint256 decayPeriod, uint256 reductionFactor, uint256 variableFeeControl, uint256 protocolShare, uint256 maxVolatilityAccumulated)
func (_LBFactory *LBFactoryFilterer) ParseFeeParametersSet(log types.Log) (*LBFactoryFeeParametersSet, error) {
	event := new(LBFactoryFeeParametersSet)
	if err := _LBFactory.contract.UnpackLog(event, "FeeParametersSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
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

	var tokenXRule []interface{}
	for _, tokenXItem := range tokenX {
		tokenXRule = append(tokenXRule, tokenXItem)
	}
	var tokenYRule []interface{}
	for _, tokenYItem := range tokenY {
		tokenYRule = append(tokenYRule, tokenYItem)
	}
	var binStepRule []interface{}
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

	var tokenXRule []interface{}
	for _, tokenXItem := range tokenX {
		tokenXRule = append(tokenXRule, tokenXItem)
	}
	var tokenYRule []interface{}
	for _, tokenYItem := range tokenY {
		tokenYRule = append(tokenYRule, tokenYItem)
	}
	var binStepRule []interface{}
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

	var LBPairRule []interface{}
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

	var LBPairRule []interface{}
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

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
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

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
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

	var pendingOwnerRule []interface{}
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

	var pendingOwnerRule []interface{}
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

	var binStepRule []interface{}
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

	var binStepRule []interface{}
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
	MaxVolatilityAccumulated *big.Int
	SampleLifetime           *big.Int
	Raw                      types.Log // Blockchain specific contextual infos
}

// FilterPresetSet is a free log retrieval operation binding the contract event 0x2f6cfdcc0e02e7355350f527dd3b5a957787b96f231165e48a3fdf90332a40cb.
//
// Solidity: event PresetSet(uint256 indexed binStep, uint256 baseFactor, uint256 filterPeriod, uint256 decayPeriod, uint256 reductionFactor, uint256 variableFeeControl, uint256 protocolShare, uint256 maxVolatilityAccumulated, uint256 sampleLifetime)
func (_LBFactory *LBFactoryFilterer) FilterPresetSet(opts *bind.FilterOpts, binStep []*big.Int) (*LBFactoryPresetSetIterator, error) {

	var binStepRule []interface{}
	for _, binStepItem := range binStep {
		binStepRule = append(binStepRule, binStepItem)
	}

	logs, sub, err := _LBFactory.contract.FilterLogs(opts, "PresetSet", binStepRule)
	if err != nil {
		return nil, err
	}
	return &LBFactoryPresetSetIterator{contract: _LBFactory.contract, event: "PresetSet", logs: logs, sub: sub}, nil
}

// WatchPresetSet is a free log subscription operation binding the contract event 0x2f6cfdcc0e02e7355350f527dd3b5a957787b96f231165e48a3fdf90332a40cb.
//
// Solidity: event PresetSet(uint256 indexed binStep, uint256 baseFactor, uint256 filterPeriod, uint256 decayPeriod, uint256 reductionFactor, uint256 variableFeeControl, uint256 protocolShare, uint256 maxVolatilityAccumulated, uint256 sampleLifetime)
func (_LBFactory *LBFactoryFilterer) WatchPresetSet(opts *bind.WatchOpts, sink chan<- *LBFactoryPresetSet, binStep []*big.Int) (event.Subscription, error) {

	var binStepRule []interface{}
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

// ParsePresetSet is a log parse operation binding the contract event 0x2f6cfdcc0e02e7355350f527dd3b5a957787b96f231165e48a3fdf90332a40cb.
//
// Solidity: event PresetSet(uint256 indexed binStep, uint256 baseFactor, uint256 filterPeriod, uint256 decayPeriod, uint256 reductionFactor, uint256 variableFeeControl, uint256 protocolShare, uint256 maxVolatilityAccumulated, uint256 sampleLifetime)
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

	var quoteAssetRule []interface{}
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

	var quoteAssetRule []interface{}
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

	var quoteAssetRule []interface{}
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

	var quoteAssetRule []interface{}
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
