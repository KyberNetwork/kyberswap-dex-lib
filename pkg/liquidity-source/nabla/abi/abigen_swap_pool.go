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

// NablaSwapPoolMetaData contains all meta data concerning the NablaSwapPool contract.
var NablaSwapPoolMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"allowance\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"asset\",\"inputs\":[],\"outputs\":[{\"name\":\"token_\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"assetDecimals\",\"inputs\":[],\"outputs\":[{\"name\":\"decimals_\",\"type\":\"uint8\",\"internalType\":\"uint8\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"backstop\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIBackstopPool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"backstopBurn\",\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_sharesToBurn\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"amount_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"backstopDrain\",\"inputs\":[{\"name\":\"_amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_recipient\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"swapAmount_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"balanceOf\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"coverage\",\"inputs\":[],\"outputs\":[{\"name\":\"reserves_\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"liabilities_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"decimals\",\"inputs\":[],\"outputs\":[{\"name\":\"decimals_\",\"type\":\"uint8\",\"internalType\":\"uint8\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"decreaseAllowance\",\"inputs\":[{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"subtractedValue\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"deposit\",\"inputs\":[{\"name\":\"_depositAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_minLPAmountOut\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_deadline\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"sharesToMint_\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"fee_\",\"type\":\"int256\",\"internalType\":\"int256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"depositingUnfreezesAt\",\"inputs\":[],\"outputs\":[{\"name\":\"depositingUnfreezesAt_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"disableGatedAccess\",\"inputs\":[],\"outputs\":[{\"name\":\"success_\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"enableGatedAccess\",\"inputs\":[],\"outputs\":[{\"name\":\"success_\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"freezeDepositing\",\"inputs\":[{\"name\":\"_forHowLong\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"getExcessLiquidity\",\"inputs\":[],\"outputs\":[{\"name\":\"excessLiquidity_\",\"type\":\"int256\",\"internalType\":\"int256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getGate\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getMaxRedeemSwapPoolShares\",\"inputs\":[],\"outputs\":[{\"name\":\"maxShares_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"increaseAllowance\",\"inputs\":[{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"addedValue\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"isAllowed\",\"inputs\":[{\"name\":\"_user\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isGated\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"lastAvailableAmountForSafeRedeem\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"lastBackstopBurnTimestamp\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"maxCoverageRatioForSwapIn\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"name\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pause\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"paused\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"poolCap\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"protocolTreasury\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"quoteBackstopDrain\",\"inputs\":[{\"name\":\"_amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"swapAmount_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"quoteDeposit\",\"inputs\":[{\"name\":\"_depositAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"sharesToMint_\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"slippageReward_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"quoteSwapInto\",\"inputs\":[{\"name\":\"_amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"effectiveAmount_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"quoteSwapOut\",\"inputs\":[{\"name\":\"_amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"effectiveAmount_\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"protocolFeeWithSlippage_\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"effectiveLpFee_\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"backstopFee_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"quoteWithdraw\",\"inputs\":[{\"name\":\"_sharesToBurn\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"payoutAmount_\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"slippagePenalty_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"reserve\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"reserveWithSlippage\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"router\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIRouter\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"safeRedeemInterval\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"safeRedeemPercentage\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setGate\",\"inputs\":[{\"name\":\"_newGate\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"success_\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setMaxCoverageRatioForSwapIn\",\"inputs\":[{\"name\":\"_maxCoverageRatio\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setPoolCap\",\"inputs\":[{\"name\":\"_maxTokens\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setProtocolTreasury\",\"inputs\":[{\"name\":\"_newProtocolTreasury\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"success_\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setSafeRedeemPercentageAndInterval\",\"inputs\":[{\"name\":\"_safeRedeemPercentage\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_safeRedeemInterval\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"success_\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setSwapFees\",\"inputs\":[{\"name\":\"_lpFee\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_backstopFee\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_protocolFee\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"sharesTargetWorth\",\"inputs\":[{\"name\":\"_sharesToBurn\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"amount_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"slippageCurve\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractISlippageCurve\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"swapFees\",\"inputs\":[],\"outputs\":[{\"name\":\"lpFee_\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"backstopFee_\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"protocolFee_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"swapIntoFromRouter\",\"inputs\":[{\"name\":\"_amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"effectiveAmount_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"swapOutFromRouter\",\"inputs\":[{\"name\":\"_amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"effectiveAmount_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"symbol\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalLiabilities\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalSupply\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transfer\",\"inputs\":[{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferFrom\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"unfreezeDepositing\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"unpause\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"withdraw\",\"inputs\":[{\"name\":\"_sharesToBurn\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_minimumAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_deadline\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"payoutAmount_\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"fee_\",\"type\":\"int256\",\"internalType\":\"int256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"Approval\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"spender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"BackstopBurn\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"sharesToBurn\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"BackstopDrain\",\"inputs\":[{\"name\":\"recipient\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"amountSwapTokens\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ChargedSwapFees\",\"inputs\":[{\"name\":\"lpFees\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"backstopFees\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"protocolFees\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DepositingFrozen\",\"inputs\":[{\"name\":\"frozenUntil\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DepositingFrozenByOwner\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"freezeDuration\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DepositingUnfrozenByOwner\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"GateUpdated\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"oldGate\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newGate\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"GatedAccessDisabled\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"GatedAccessEnabled\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"MaxCoverageRatioForSwapInSet\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"maxCoverageRatio\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Mint\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"poolSharesMinted\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"fee\",\"type\":\"int256\",\"indexed\":false,\"internalType\":\"int256\"},{\"name\":\"amountDeposited\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Paused\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"PoolCapSet\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"oldPoolCap\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"newPoolCap\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ProtocolTreasuryChanged\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"oldProtocolTreasury\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"newProtocolTreasury\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SafeRedeemPercentageAndIntervalSet\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"safeRedeemPercentage\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"safeRedeemInterval\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SwapFeesSet\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"lpFee\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"backstopFee\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"protocolFee\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Transfer\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Unpaused\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Withdrawal\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"poolSharesBurned\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"fee\",\"type\":\"int256\",\"indexed\":false,\"internalType\":\"int256\"},{\"name\":\"amountWithdrawn\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ReserveUpdated\",\"inputs\":[{\"name\":\"newReserve\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"newReserveWithSlippage\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"newTotalLiabilities\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false}]",
}

// NablaSwapPoolABI is the input ABI used to generate the binding from.
// Deprecated: Use NablaSwapPoolMetaData.ABI instead.
var NablaSwapPoolABI = NablaSwapPoolMetaData.ABI

// NablaSwapPool is an auto generated Go binding around an Ethereum contract.
type NablaSwapPool struct {
	NablaSwapPoolCaller     // Read-only binding to the contract
	NablaSwapPoolTransactor // Write-only binding to the contract
	NablaSwapPoolFilterer   // Log filterer for contract events
}

// NablaSwapPoolCaller is an auto generated read-only Go binding around an Ethereum contract.
type NablaSwapPoolCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NablaSwapPoolTransactor is an auto generated write-only Go binding around an Ethereum contract.
type NablaSwapPoolTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NablaSwapPoolFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type NablaSwapPoolFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NablaSwapPoolSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type NablaSwapPoolSession struct {
	Contract     *NablaSwapPool    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// NablaSwapPoolCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type NablaSwapPoolCallerSession struct {
	Contract *NablaSwapPoolCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// NablaSwapPoolTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type NablaSwapPoolTransactorSession struct {
	Contract     *NablaSwapPoolTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// NablaSwapPoolRaw is an auto generated low-level Go binding around an Ethereum contract.
type NablaSwapPoolRaw struct {
	Contract *NablaSwapPool // Generic contract binding to access the raw methods on
}

// NablaSwapPoolCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type NablaSwapPoolCallerRaw struct {
	Contract *NablaSwapPoolCaller // Generic read-only contract binding to access the raw methods on
}

// NablaSwapPoolTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type NablaSwapPoolTransactorRaw struct {
	Contract *NablaSwapPoolTransactor // Generic write-only contract binding to access the raw methods on
}

// NewNablaSwapPool creates a new instance of NablaSwapPool, bound to a specific deployed contract.
func NewNablaSwapPool(address common.Address, backend bind.ContractBackend) (*NablaSwapPool, error) {
	contract, err := bindNablaSwapPool(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPool{NablaSwapPoolCaller: NablaSwapPoolCaller{contract: contract}, NablaSwapPoolTransactor: NablaSwapPoolTransactor{contract: contract}, NablaSwapPoolFilterer: NablaSwapPoolFilterer{contract: contract}}, nil
}

// NewNablaSwapPoolCaller creates a new read-only instance of NablaSwapPool, bound to a specific deployed contract.
func NewNablaSwapPoolCaller(address common.Address, caller bind.ContractCaller) (*NablaSwapPoolCaller, error) {
	contract, err := bindNablaSwapPool(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolCaller{contract: contract}, nil
}

// NewNablaSwapPoolTransactor creates a new write-only instance of NablaSwapPool, bound to a specific deployed contract.
func NewNablaSwapPoolTransactor(address common.Address, transactor bind.ContractTransactor) (*NablaSwapPoolTransactor, error) {
	contract, err := bindNablaSwapPool(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolTransactor{contract: contract}, nil
}

// NewNablaSwapPoolFilterer creates a new log filterer instance of NablaSwapPool, bound to a specific deployed contract.
func NewNablaSwapPoolFilterer(address common.Address, filterer bind.ContractFilterer) (*NablaSwapPoolFilterer, error) {
	contract, err := bindNablaSwapPool(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolFilterer{contract: contract}, nil
}

// bindNablaSwapPool binds a generic wrapper to an already deployed contract.
func bindNablaSwapPool(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := NablaSwapPoolMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NablaSwapPool *NablaSwapPoolRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NablaSwapPool.Contract.NablaSwapPoolCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NablaSwapPool *NablaSwapPoolRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.NablaSwapPoolTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NablaSwapPool *NablaSwapPoolRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.NablaSwapPoolTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NablaSwapPool *NablaSwapPoolCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NablaSwapPool.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NablaSwapPool *NablaSwapPoolTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NablaSwapPool *NablaSwapPoolTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCaller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _NablaSwapPool.Contract.Allowance(&_NablaSwapPool.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _NablaSwapPool.Contract.Allowance(&_NablaSwapPool.CallOpts, owner, spender)
}

// Asset is a free data retrieval call binding the contract method 0x38d52e0f.
//
// Solidity: function asset() view returns(address token_)
func (_NablaSwapPool *NablaSwapPoolCaller) Asset(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "asset")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Asset is a free data retrieval call binding the contract method 0x38d52e0f.
//
// Solidity: function asset() view returns(address token_)
func (_NablaSwapPool *NablaSwapPoolSession) Asset() (common.Address, error) {
	return _NablaSwapPool.Contract.Asset(&_NablaSwapPool.CallOpts)
}

// Asset is a free data retrieval call binding the contract method 0x38d52e0f.
//
// Solidity: function asset() view returns(address token_)
func (_NablaSwapPool *NablaSwapPoolCallerSession) Asset() (common.Address, error) {
	return _NablaSwapPool.Contract.Asset(&_NablaSwapPool.CallOpts)
}

// AssetDecimals is a free data retrieval call binding the contract method 0xc2d41601.
//
// Solidity: function assetDecimals() view returns(uint8 decimals_)
func (_NablaSwapPool *NablaSwapPoolCaller) AssetDecimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "assetDecimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// AssetDecimals is a free data retrieval call binding the contract method 0xc2d41601.
//
// Solidity: function assetDecimals() view returns(uint8 decimals_)
func (_NablaSwapPool *NablaSwapPoolSession) AssetDecimals() (uint8, error) {
	return _NablaSwapPool.Contract.AssetDecimals(&_NablaSwapPool.CallOpts)
}

// AssetDecimals is a free data retrieval call binding the contract method 0xc2d41601.
//
// Solidity: function assetDecimals() view returns(uint8 decimals_)
func (_NablaSwapPool *NablaSwapPoolCallerSession) AssetDecimals() (uint8, error) {
	return _NablaSwapPool.Contract.AssetDecimals(&_NablaSwapPool.CallOpts)
}

// Backstop is a free data retrieval call binding the contract method 0x7dea1817.
//
// Solidity: function backstop() view returns(address)
func (_NablaSwapPool *NablaSwapPoolCaller) Backstop(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "backstop")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Backstop is a free data retrieval call binding the contract method 0x7dea1817.
//
// Solidity: function backstop() view returns(address)
func (_NablaSwapPool *NablaSwapPoolSession) Backstop() (common.Address, error) {
	return _NablaSwapPool.Contract.Backstop(&_NablaSwapPool.CallOpts)
}

// Backstop is a free data retrieval call binding the contract method 0x7dea1817.
//
// Solidity: function backstop() view returns(address)
func (_NablaSwapPool *NablaSwapPoolCallerSession) Backstop() (common.Address, error) {
	return _NablaSwapPool.Contract.Backstop(&_NablaSwapPool.CallOpts)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCaller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _NablaSwapPool.Contract.BalanceOf(&_NablaSwapPool.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _NablaSwapPool.Contract.BalanceOf(&_NablaSwapPool.CallOpts, account)
}

// Coverage is a free data retrieval call binding the contract method 0xee8f6a0e.
//
// Solidity: function coverage() view returns(uint256 reserves_, uint256 liabilities_)
func (_NablaSwapPool *NablaSwapPoolCaller) Coverage(opts *bind.CallOpts) (struct {
	Reserves    *big.Int
	Liabilities *big.Int
}, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "coverage")

	outstruct := new(struct {
		Reserves    *big.Int
		Liabilities *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Reserves = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Liabilities = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Coverage is a free data retrieval call binding the contract method 0xee8f6a0e.
//
// Solidity: function coverage() view returns(uint256 reserves_, uint256 liabilities_)
func (_NablaSwapPool *NablaSwapPoolSession) Coverage() (struct {
	Reserves    *big.Int
	Liabilities *big.Int
}, error) {
	return _NablaSwapPool.Contract.Coverage(&_NablaSwapPool.CallOpts)
}

// Coverage is a free data retrieval call binding the contract method 0xee8f6a0e.
//
// Solidity: function coverage() view returns(uint256 reserves_, uint256 liabilities_)
func (_NablaSwapPool *NablaSwapPoolCallerSession) Coverage() (struct {
	Reserves    *big.Int
	Liabilities *big.Int
}, error) {
	return _NablaSwapPool.Contract.Coverage(&_NablaSwapPool.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8 decimals_)
func (_NablaSwapPool *NablaSwapPoolCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8 decimals_)
func (_NablaSwapPool *NablaSwapPoolSession) Decimals() (uint8, error) {
	return _NablaSwapPool.Contract.Decimals(&_NablaSwapPool.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8 decimals_)
func (_NablaSwapPool *NablaSwapPoolCallerSession) Decimals() (uint8, error) {
	return _NablaSwapPool.Contract.Decimals(&_NablaSwapPool.CallOpts)
}

// DepositingUnfreezesAt is a free data retrieval call binding the contract method 0xb9b3de2d.
//
// Solidity: function depositingUnfreezesAt() view returns(uint256 depositingUnfreezesAt_)
func (_NablaSwapPool *NablaSwapPoolCaller) DepositingUnfreezesAt(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "depositingUnfreezesAt")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DepositingUnfreezesAt is a free data retrieval call binding the contract method 0xb9b3de2d.
//
// Solidity: function depositingUnfreezesAt() view returns(uint256 depositingUnfreezesAt_)
func (_NablaSwapPool *NablaSwapPoolSession) DepositingUnfreezesAt() (*big.Int, error) {
	return _NablaSwapPool.Contract.DepositingUnfreezesAt(&_NablaSwapPool.CallOpts)
}

// DepositingUnfreezesAt is a free data retrieval call binding the contract method 0xb9b3de2d.
//
// Solidity: function depositingUnfreezesAt() view returns(uint256 depositingUnfreezesAt_)
func (_NablaSwapPool *NablaSwapPoolCallerSession) DepositingUnfreezesAt() (*big.Int, error) {
	return _NablaSwapPool.Contract.DepositingUnfreezesAt(&_NablaSwapPool.CallOpts)
}

// GetExcessLiquidity is a free data retrieval call binding the contract method 0xace0f0d5.
//
// Solidity: function getExcessLiquidity() view returns(int256 excessLiquidity_)
func (_NablaSwapPool *NablaSwapPoolCaller) GetExcessLiquidity(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "getExcessLiquidity")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetExcessLiquidity is a free data retrieval call binding the contract method 0xace0f0d5.
//
// Solidity: function getExcessLiquidity() view returns(int256 excessLiquidity_)
func (_NablaSwapPool *NablaSwapPoolSession) GetExcessLiquidity() (*big.Int, error) {
	return _NablaSwapPool.Contract.GetExcessLiquidity(&_NablaSwapPool.CallOpts)
}

// GetExcessLiquidity is a free data retrieval call binding the contract method 0xace0f0d5.
//
// Solidity: function getExcessLiquidity() view returns(int256 excessLiquidity_)
func (_NablaSwapPool *NablaSwapPoolCallerSession) GetExcessLiquidity() (*big.Int, error) {
	return _NablaSwapPool.Contract.GetExcessLiquidity(&_NablaSwapPool.CallOpts)
}

// GetGate is a free data retrieval call binding the contract method 0xe301665d.
//
// Solidity: function getGate() view returns(address)
func (_NablaSwapPool *NablaSwapPoolCaller) GetGate(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "getGate")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetGate is a free data retrieval call binding the contract method 0xe301665d.
//
// Solidity: function getGate() view returns(address)
func (_NablaSwapPool *NablaSwapPoolSession) GetGate() (common.Address, error) {
	return _NablaSwapPool.Contract.GetGate(&_NablaSwapPool.CallOpts)
}

// GetGate is a free data retrieval call binding the contract method 0xe301665d.
//
// Solidity: function getGate() view returns(address)
func (_NablaSwapPool *NablaSwapPoolCallerSession) GetGate() (common.Address, error) {
	return _NablaSwapPool.Contract.GetGate(&_NablaSwapPool.CallOpts)
}

// GetMaxRedeemSwapPoolShares is a free data retrieval call binding the contract method 0xeb9b611f.
//
// Solidity: function getMaxRedeemSwapPoolShares() view returns(uint256 maxShares_)
func (_NablaSwapPool *NablaSwapPoolCaller) GetMaxRedeemSwapPoolShares(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "getMaxRedeemSwapPoolShares")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetMaxRedeemSwapPoolShares is a free data retrieval call binding the contract method 0xeb9b611f.
//
// Solidity: function getMaxRedeemSwapPoolShares() view returns(uint256 maxShares_)
func (_NablaSwapPool *NablaSwapPoolSession) GetMaxRedeemSwapPoolShares() (*big.Int, error) {
	return _NablaSwapPool.Contract.GetMaxRedeemSwapPoolShares(&_NablaSwapPool.CallOpts)
}

// GetMaxRedeemSwapPoolShares is a free data retrieval call binding the contract method 0xeb9b611f.
//
// Solidity: function getMaxRedeemSwapPoolShares() view returns(uint256 maxShares_)
func (_NablaSwapPool *NablaSwapPoolCallerSession) GetMaxRedeemSwapPoolShares() (*big.Int, error) {
	return _NablaSwapPool.Contract.GetMaxRedeemSwapPoolShares(&_NablaSwapPool.CallOpts)
}

// IsAllowed is a free data retrieval call binding the contract method 0xf8350ed0.
//
// Solidity: function isAllowed(address _user, uint256 _amount) view returns(bool)
func (_NablaSwapPool *NablaSwapPoolCaller) IsAllowed(opts *bind.CallOpts, _user common.Address, _amount *big.Int) (bool, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "isAllowed", _user, _amount)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAllowed is a free data retrieval call binding the contract method 0xf8350ed0.
//
// Solidity: function isAllowed(address _user, uint256 _amount) view returns(bool)
func (_NablaSwapPool *NablaSwapPoolSession) IsAllowed(_user common.Address, _amount *big.Int) (bool, error) {
	return _NablaSwapPool.Contract.IsAllowed(&_NablaSwapPool.CallOpts, _user, _amount)
}

// IsAllowed is a free data retrieval call binding the contract method 0xf8350ed0.
//
// Solidity: function isAllowed(address _user, uint256 _amount) view returns(bool)
func (_NablaSwapPool *NablaSwapPoolCallerSession) IsAllowed(_user common.Address, _amount *big.Int) (bool, error) {
	return _NablaSwapPool.Contract.IsAllowed(&_NablaSwapPool.CallOpts, _user, _amount)
}

// IsGated is a free data retrieval call binding the contract method 0xe4741da4.
//
// Solidity: function isGated() view returns(bool)
func (_NablaSwapPool *NablaSwapPoolCaller) IsGated(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "isGated")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsGated is a free data retrieval call binding the contract method 0xe4741da4.
//
// Solidity: function isGated() view returns(bool)
func (_NablaSwapPool *NablaSwapPoolSession) IsGated() (bool, error) {
	return _NablaSwapPool.Contract.IsGated(&_NablaSwapPool.CallOpts)
}

// IsGated is a free data retrieval call binding the contract method 0xe4741da4.
//
// Solidity: function isGated() view returns(bool)
func (_NablaSwapPool *NablaSwapPoolCallerSession) IsGated() (bool, error) {
	return _NablaSwapPool.Contract.IsGated(&_NablaSwapPool.CallOpts)
}

// LastAvailableAmountForSafeRedeem is a free data retrieval call binding the contract method 0x29e97cdd.
//
// Solidity: function lastAvailableAmountForSafeRedeem() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCaller) LastAvailableAmountForSafeRedeem(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "lastAvailableAmountForSafeRedeem")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LastAvailableAmountForSafeRedeem is a free data retrieval call binding the contract method 0x29e97cdd.
//
// Solidity: function lastAvailableAmountForSafeRedeem() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolSession) LastAvailableAmountForSafeRedeem() (*big.Int, error) {
	return _NablaSwapPool.Contract.LastAvailableAmountForSafeRedeem(&_NablaSwapPool.CallOpts)
}

// LastAvailableAmountForSafeRedeem is a free data retrieval call binding the contract method 0x29e97cdd.
//
// Solidity: function lastAvailableAmountForSafeRedeem() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCallerSession) LastAvailableAmountForSafeRedeem() (*big.Int, error) {
	return _NablaSwapPool.Contract.LastAvailableAmountForSafeRedeem(&_NablaSwapPool.CallOpts)
}

// LastBackstopBurnTimestamp is a free data retrieval call binding the contract method 0xd3a37e2a.
//
// Solidity: function lastBackstopBurnTimestamp() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCaller) LastBackstopBurnTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "lastBackstopBurnTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LastBackstopBurnTimestamp is a free data retrieval call binding the contract method 0xd3a37e2a.
//
// Solidity: function lastBackstopBurnTimestamp() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolSession) LastBackstopBurnTimestamp() (*big.Int, error) {
	return _NablaSwapPool.Contract.LastBackstopBurnTimestamp(&_NablaSwapPool.CallOpts)
}

// LastBackstopBurnTimestamp is a free data retrieval call binding the contract method 0xd3a37e2a.
//
// Solidity: function lastBackstopBurnTimestamp() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCallerSession) LastBackstopBurnTimestamp() (*big.Int, error) {
	return _NablaSwapPool.Contract.LastBackstopBurnTimestamp(&_NablaSwapPool.CallOpts)
}

// MaxCoverageRatioForSwapIn is a free data retrieval call binding the contract method 0xb2f3447a.
//
// Solidity: function maxCoverageRatioForSwapIn() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCaller) MaxCoverageRatioForSwapIn(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "maxCoverageRatioForSwapIn")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxCoverageRatioForSwapIn is a free data retrieval call binding the contract method 0xb2f3447a.
//
// Solidity: function maxCoverageRatioForSwapIn() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolSession) MaxCoverageRatioForSwapIn() (*big.Int, error) {
	return _NablaSwapPool.Contract.MaxCoverageRatioForSwapIn(&_NablaSwapPool.CallOpts)
}

// MaxCoverageRatioForSwapIn is a free data retrieval call binding the contract method 0xb2f3447a.
//
// Solidity: function maxCoverageRatioForSwapIn() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCallerSession) MaxCoverageRatioForSwapIn() (*big.Int, error) {
	return _NablaSwapPool.Contract.MaxCoverageRatioForSwapIn(&_NablaSwapPool.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_NablaSwapPool *NablaSwapPoolCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_NablaSwapPool *NablaSwapPoolSession) Name() (string, error) {
	return _NablaSwapPool.Contract.Name(&_NablaSwapPool.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_NablaSwapPool *NablaSwapPoolCallerSession) Name() (string, error) {
	return _NablaSwapPool.Contract.Name(&_NablaSwapPool.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_NablaSwapPool *NablaSwapPoolCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_NablaSwapPool *NablaSwapPoolSession) Owner() (common.Address, error) {
	return _NablaSwapPool.Contract.Owner(&_NablaSwapPool.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_NablaSwapPool *NablaSwapPoolCallerSession) Owner() (common.Address, error) {
	return _NablaSwapPool.Contract.Owner(&_NablaSwapPool.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_NablaSwapPool *NablaSwapPoolCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_NablaSwapPool *NablaSwapPoolSession) Paused() (bool, error) {
	return _NablaSwapPool.Contract.Paused(&_NablaSwapPool.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_NablaSwapPool *NablaSwapPoolCallerSession) Paused() (bool, error) {
	return _NablaSwapPool.Contract.Paused(&_NablaSwapPool.CallOpts)
}

// PoolCap is a free data retrieval call binding the contract method 0xb954dc57.
//
// Solidity: function poolCap() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCaller) PoolCap(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "poolCap")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PoolCap is a free data retrieval call binding the contract method 0xb954dc57.
//
// Solidity: function poolCap() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolSession) PoolCap() (*big.Int, error) {
	return _NablaSwapPool.Contract.PoolCap(&_NablaSwapPool.CallOpts)
}

// PoolCap is a free data retrieval call binding the contract method 0xb954dc57.
//
// Solidity: function poolCap() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCallerSession) PoolCap() (*big.Int, error) {
	return _NablaSwapPool.Contract.PoolCap(&_NablaSwapPool.CallOpts)
}

// ProtocolTreasury is a free data retrieval call binding the contract method 0x803db96d.
//
// Solidity: function protocolTreasury() view returns(address)
func (_NablaSwapPool *NablaSwapPoolCaller) ProtocolTreasury(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "protocolTreasury")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ProtocolTreasury is a free data retrieval call binding the contract method 0x803db96d.
//
// Solidity: function protocolTreasury() view returns(address)
func (_NablaSwapPool *NablaSwapPoolSession) ProtocolTreasury() (common.Address, error) {
	return _NablaSwapPool.Contract.ProtocolTreasury(&_NablaSwapPool.CallOpts)
}

// ProtocolTreasury is a free data retrieval call binding the contract method 0x803db96d.
//
// Solidity: function protocolTreasury() view returns(address)
func (_NablaSwapPool *NablaSwapPoolCallerSession) ProtocolTreasury() (common.Address, error) {
	return _NablaSwapPool.Contract.ProtocolTreasury(&_NablaSwapPool.CallOpts)
}

// QuoteBackstopDrain is a free data retrieval call binding the contract method 0xe237fb3d.
//
// Solidity: function quoteBackstopDrain(uint256 _amount) view returns(uint256 swapAmount_)
func (_NablaSwapPool *NablaSwapPoolCaller) QuoteBackstopDrain(opts *bind.CallOpts, _amount *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "quoteBackstopDrain", _amount)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// QuoteBackstopDrain is a free data retrieval call binding the contract method 0xe237fb3d.
//
// Solidity: function quoteBackstopDrain(uint256 _amount) view returns(uint256 swapAmount_)
func (_NablaSwapPool *NablaSwapPoolSession) QuoteBackstopDrain(_amount *big.Int) (*big.Int, error) {
	return _NablaSwapPool.Contract.QuoteBackstopDrain(&_NablaSwapPool.CallOpts, _amount)
}

// QuoteBackstopDrain is a free data retrieval call binding the contract method 0xe237fb3d.
//
// Solidity: function quoteBackstopDrain(uint256 _amount) view returns(uint256 swapAmount_)
func (_NablaSwapPool *NablaSwapPoolCallerSession) QuoteBackstopDrain(_amount *big.Int) (*big.Int, error) {
	return _NablaSwapPool.Contract.QuoteBackstopDrain(&_NablaSwapPool.CallOpts, _amount)
}

// QuoteDeposit is a free data retrieval call binding the contract method 0xdb431f06.
//
// Solidity: function quoteDeposit(uint256 _depositAmount) view returns(uint256 sharesToMint_, uint256 slippageReward_)
func (_NablaSwapPool *NablaSwapPoolCaller) QuoteDeposit(opts *bind.CallOpts, _depositAmount *big.Int) (struct {
	SharesToMint   *big.Int
	SlippageReward *big.Int
}, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "quoteDeposit", _depositAmount)

	outstruct := new(struct {
		SharesToMint   *big.Int
		SlippageReward *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.SharesToMint = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.SlippageReward = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// QuoteDeposit is a free data retrieval call binding the contract method 0xdb431f06.
//
// Solidity: function quoteDeposit(uint256 _depositAmount) view returns(uint256 sharesToMint_, uint256 slippageReward_)
func (_NablaSwapPool *NablaSwapPoolSession) QuoteDeposit(_depositAmount *big.Int) (struct {
	SharesToMint   *big.Int
	SlippageReward *big.Int
}, error) {
	return _NablaSwapPool.Contract.QuoteDeposit(&_NablaSwapPool.CallOpts, _depositAmount)
}

// QuoteDeposit is a free data retrieval call binding the contract method 0xdb431f06.
//
// Solidity: function quoteDeposit(uint256 _depositAmount) view returns(uint256 sharesToMint_, uint256 slippageReward_)
func (_NablaSwapPool *NablaSwapPoolCallerSession) QuoteDeposit(_depositAmount *big.Int) (struct {
	SharesToMint   *big.Int
	SlippageReward *big.Int
}, error) {
	return _NablaSwapPool.Contract.QuoteDeposit(&_NablaSwapPool.CallOpts, _depositAmount)
}

// QuoteSwapInto is a free data retrieval call binding the contract method 0x3c945248.
//
// Solidity: function quoteSwapInto(uint256 _amount) view returns(uint256 effectiveAmount_)
func (_NablaSwapPool *NablaSwapPoolCaller) QuoteSwapInto(opts *bind.CallOpts, _amount *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "quoteSwapInto", _amount)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// QuoteSwapInto is a free data retrieval call binding the contract method 0x3c945248.
//
// Solidity: function quoteSwapInto(uint256 _amount) view returns(uint256 effectiveAmount_)
func (_NablaSwapPool *NablaSwapPoolSession) QuoteSwapInto(_amount *big.Int) (*big.Int, error) {
	return _NablaSwapPool.Contract.QuoteSwapInto(&_NablaSwapPool.CallOpts, _amount)
}

// QuoteSwapInto is a free data retrieval call binding the contract method 0x3c945248.
//
// Solidity: function quoteSwapInto(uint256 _amount) view returns(uint256 effectiveAmount_)
func (_NablaSwapPool *NablaSwapPoolCallerSession) QuoteSwapInto(_amount *big.Int) (*big.Int, error) {
	return _NablaSwapPool.Contract.QuoteSwapInto(&_NablaSwapPool.CallOpts, _amount)
}

// QuoteSwapOut is a free data retrieval call binding the contract method 0x8735c246.
//
// Solidity: function quoteSwapOut(uint256 _amount) view returns(uint256 effectiveAmount_, uint256 protocolFeeWithSlippage_, uint256 effectiveLpFee_, uint256 backstopFee_)
func (_NablaSwapPool *NablaSwapPoolCaller) QuoteSwapOut(opts *bind.CallOpts, _amount *big.Int) (struct {
	EffectiveAmount         *big.Int
	ProtocolFeeWithSlippage *big.Int
	EffectiveLpFee          *big.Int
	BackstopFee             *big.Int
}, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "quoteSwapOut", _amount)

	outstruct := new(struct {
		EffectiveAmount         *big.Int
		ProtocolFeeWithSlippage *big.Int
		EffectiveLpFee          *big.Int
		BackstopFee             *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.EffectiveAmount = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.ProtocolFeeWithSlippage = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.EffectiveLpFee = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.BackstopFee = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// QuoteSwapOut is a free data retrieval call binding the contract method 0x8735c246.
//
// Solidity: function quoteSwapOut(uint256 _amount) view returns(uint256 effectiveAmount_, uint256 protocolFeeWithSlippage_, uint256 effectiveLpFee_, uint256 backstopFee_)
func (_NablaSwapPool *NablaSwapPoolSession) QuoteSwapOut(_amount *big.Int) (struct {
	EffectiveAmount         *big.Int
	ProtocolFeeWithSlippage *big.Int
	EffectiveLpFee          *big.Int
	BackstopFee             *big.Int
}, error) {
	return _NablaSwapPool.Contract.QuoteSwapOut(&_NablaSwapPool.CallOpts, _amount)
}

// QuoteSwapOut is a free data retrieval call binding the contract method 0x8735c246.
//
// Solidity: function quoteSwapOut(uint256 _amount) view returns(uint256 effectiveAmount_, uint256 protocolFeeWithSlippage_, uint256 effectiveLpFee_, uint256 backstopFee_)
func (_NablaSwapPool *NablaSwapPoolCallerSession) QuoteSwapOut(_amount *big.Int) (struct {
	EffectiveAmount         *big.Int
	ProtocolFeeWithSlippage *big.Int
	EffectiveLpFee          *big.Int
	BackstopFee             *big.Int
}, error) {
	return _NablaSwapPool.Contract.QuoteSwapOut(&_NablaSwapPool.CallOpts, _amount)
}

// QuoteWithdraw is a free data retrieval call binding the contract method 0xec211840.
//
// Solidity: function quoteWithdraw(uint256 _sharesToBurn) view returns(uint256 payoutAmount_, uint256 slippagePenalty_)
func (_NablaSwapPool *NablaSwapPoolCaller) QuoteWithdraw(opts *bind.CallOpts, _sharesToBurn *big.Int) (struct {
	PayoutAmount    *big.Int
	SlippagePenalty *big.Int
}, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "quoteWithdraw", _sharesToBurn)

	outstruct := new(struct {
		PayoutAmount    *big.Int
		SlippagePenalty *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.PayoutAmount = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.SlippagePenalty = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// QuoteWithdraw is a free data retrieval call binding the contract method 0xec211840.
//
// Solidity: function quoteWithdraw(uint256 _sharesToBurn) view returns(uint256 payoutAmount_, uint256 slippagePenalty_)
func (_NablaSwapPool *NablaSwapPoolSession) QuoteWithdraw(_sharesToBurn *big.Int) (struct {
	PayoutAmount    *big.Int
	SlippagePenalty *big.Int
}, error) {
	return _NablaSwapPool.Contract.QuoteWithdraw(&_NablaSwapPool.CallOpts, _sharesToBurn)
}

// QuoteWithdraw is a free data retrieval call binding the contract method 0xec211840.
//
// Solidity: function quoteWithdraw(uint256 _sharesToBurn) view returns(uint256 payoutAmount_, uint256 slippagePenalty_)
func (_NablaSwapPool *NablaSwapPoolCallerSession) QuoteWithdraw(_sharesToBurn *big.Int) (struct {
	PayoutAmount    *big.Int
	SlippagePenalty *big.Int
}, error) {
	return _NablaSwapPool.Contract.QuoteWithdraw(&_NablaSwapPool.CallOpts, _sharesToBurn)
}

// Reserve is a free data retrieval call binding the contract method 0xcd3293de.
//
// Solidity: function reserve() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCaller) Reserve(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "reserve")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Reserve is a free data retrieval call binding the contract method 0xcd3293de.
//
// Solidity: function reserve() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolSession) Reserve() (*big.Int, error) {
	return _NablaSwapPool.Contract.Reserve(&_NablaSwapPool.CallOpts)
}

// Reserve is a free data retrieval call binding the contract method 0xcd3293de.
//
// Solidity: function reserve() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCallerSession) Reserve() (*big.Int, error) {
	return _NablaSwapPool.Contract.Reserve(&_NablaSwapPool.CallOpts)
}

// ReserveWithSlippage is a free data retrieval call binding the contract method 0x0b09d91e.
//
// Solidity: function reserveWithSlippage() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCaller) ReserveWithSlippage(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "reserveWithSlippage")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ReserveWithSlippage is a free data retrieval call binding the contract method 0x0b09d91e.
//
// Solidity: function reserveWithSlippage() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolSession) ReserveWithSlippage() (*big.Int, error) {
	return _NablaSwapPool.Contract.ReserveWithSlippage(&_NablaSwapPool.CallOpts)
}

// ReserveWithSlippage is a free data retrieval call binding the contract method 0x0b09d91e.
//
// Solidity: function reserveWithSlippage() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCallerSession) ReserveWithSlippage() (*big.Int, error) {
	return _NablaSwapPool.Contract.ReserveWithSlippage(&_NablaSwapPool.CallOpts)
}

// Router is a free data retrieval call binding the contract method 0xf887ea40.
//
// Solidity: function router() view returns(address)
func (_NablaSwapPool *NablaSwapPoolCaller) Router(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "router")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Router is a free data retrieval call binding the contract method 0xf887ea40.
//
// Solidity: function router() view returns(address)
func (_NablaSwapPool *NablaSwapPoolSession) Router() (common.Address, error) {
	return _NablaSwapPool.Contract.Router(&_NablaSwapPool.CallOpts)
}

// Router is a free data retrieval call binding the contract method 0xf887ea40.
//
// Solidity: function router() view returns(address)
func (_NablaSwapPool *NablaSwapPoolCallerSession) Router() (common.Address, error) {
	return _NablaSwapPool.Contract.Router(&_NablaSwapPool.CallOpts)
}

// SafeRedeemInterval is a free data retrieval call binding the contract method 0xd0eddb59.
//
// Solidity: function safeRedeemInterval() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCaller) SafeRedeemInterval(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "safeRedeemInterval")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SafeRedeemInterval is a free data retrieval call binding the contract method 0xd0eddb59.
//
// Solidity: function safeRedeemInterval() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolSession) SafeRedeemInterval() (*big.Int, error) {
	return _NablaSwapPool.Contract.SafeRedeemInterval(&_NablaSwapPool.CallOpts)
}

// SafeRedeemInterval is a free data retrieval call binding the contract method 0xd0eddb59.
//
// Solidity: function safeRedeemInterval() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCallerSession) SafeRedeemInterval() (*big.Int, error) {
	return _NablaSwapPool.Contract.SafeRedeemInterval(&_NablaSwapPool.CallOpts)
}

// SafeRedeemPercentage is a free data retrieval call binding the contract method 0x696d8171.
//
// Solidity: function safeRedeemPercentage() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCaller) SafeRedeemPercentage(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "safeRedeemPercentage")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SafeRedeemPercentage is a free data retrieval call binding the contract method 0x696d8171.
//
// Solidity: function safeRedeemPercentage() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolSession) SafeRedeemPercentage() (*big.Int, error) {
	return _NablaSwapPool.Contract.SafeRedeemPercentage(&_NablaSwapPool.CallOpts)
}

// SafeRedeemPercentage is a free data retrieval call binding the contract method 0x696d8171.
//
// Solidity: function safeRedeemPercentage() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCallerSession) SafeRedeemPercentage() (*big.Int, error) {
	return _NablaSwapPool.Contract.SafeRedeemPercentage(&_NablaSwapPool.CallOpts)
}

// SharesTargetWorth is a free data retrieval call binding the contract method 0xcc045745.
//
// Solidity: function sharesTargetWorth(uint256 _sharesToBurn) view returns(uint256 amount_)
func (_NablaSwapPool *NablaSwapPoolCaller) SharesTargetWorth(opts *bind.CallOpts, _sharesToBurn *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "sharesTargetWorth", _sharesToBurn)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SharesTargetWorth is a free data retrieval call binding the contract method 0xcc045745.
//
// Solidity: function sharesTargetWorth(uint256 _sharesToBurn) view returns(uint256 amount_)
func (_NablaSwapPool *NablaSwapPoolSession) SharesTargetWorth(_sharesToBurn *big.Int) (*big.Int, error) {
	return _NablaSwapPool.Contract.SharesTargetWorth(&_NablaSwapPool.CallOpts, _sharesToBurn)
}

// SharesTargetWorth is a free data retrieval call binding the contract method 0xcc045745.
//
// Solidity: function sharesTargetWorth(uint256 _sharesToBurn) view returns(uint256 amount_)
func (_NablaSwapPool *NablaSwapPoolCallerSession) SharesTargetWorth(_sharesToBurn *big.Int) (*big.Int, error) {
	return _NablaSwapPool.Contract.SharesTargetWorth(&_NablaSwapPool.CallOpts, _sharesToBurn)
}

// SlippageCurve is a free data retrieval call binding the contract method 0xebe26b9e.
//
// Solidity: function slippageCurve() view returns(address)
func (_NablaSwapPool *NablaSwapPoolCaller) SlippageCurve(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "slippageCurve")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SlippageCurve is a free data retrieval call binding the contract method 0xebe26b9e.
//
// Solidity: function slippageCurve() view returns(address)
func (_NablaSwapPool *NablaSwapPoolSession) SlippageCurve() (common.Address, error) {
	return _NablaSwapPool.Contract.SlippageCurve(&_NablaSwapPool.CallOpts)
}

// SlippageCurve is a free data retrieval call binding the contract method 0xebe26b9e.
//
// Solidity: function slippageCurve() view returns(address)
func (_NablaSwapPool *NablaSwapPoolCallerSession) SlippageCurve() (common.Address, error) {
	return _NablaSwapPool.Contract.SlippageCurve(&_NablaSwapPool.CallOpts)
}

// SwapFees is a free data retrieval call binding the contract method 0xb9ccf21d.
//
// Solidity: function swapFees() view returns(uint256 lpFee_, uint256 backstopFee_, uint256 protocolFee_)
func (_NablaSwapPool *NablaSwapPoolCaller) SwapFees(opts *bind.CallOpts) (struct {
	LpFee       *big.Int
	BackstopFee *big.Int
	ProtocolFee *big.Int
}, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "swapFees")

	outstruct := new(struct {
		LpFee       *big.Int
		BackstopFee *big.Int
		ProtocolFee *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.LpFee = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.BackstopFee = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.ProtocolFee = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// SwapFees is a free data retrieval call binding the contract method 0xb9ccf21d.
//
// Solidity: function swapFees() view returns(uint256 lpFee_, uint256 backstopFee_, uint256 protocolFee_)
func (_NablaSwapPool *NablaSwapPoolSession) SwapFees() (struct {
	LpFee       *big.Int
	BackstopFee *big.Int
	ProtocolFee *big.Int
}, error) {
	return _NablaSwapPool.Contract.SwapFees(&_NablaSwapPool.CallOpts)
}

// SwapFees is a free data retrieval call binding the contract method 0xb9ccf21d.
//
// Solidity: function swapFees() view returns(uint256 lpFee_, uint256 backstopFee_, uint256 protocolFee_)
func (_NablaSwapPool *NablaSwapPoolCallerSession) SwapFees() (struct {
	LpFee       *big.Int
	BackstopFee *big.Int
	ProtocolFee *big.Int
}, error) {
	return _NablaSwapPool.Contract.SwapFees(&_NablaSwapPool.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_NablaSwapPool *NablaSwapPoolCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_NablaSwapPool *NablaSwapPoolSession) Symbol() (string, error) {
	return _NablaSwapPool.Contract.Symbol(&_NablaSwapPool.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_NablaSwapPool *NablaSwapPoolCallerSession) Symbol() (string, error) {
	return _NablaSwapPool.Contract.Symbol(&_NablaSwapPool.CallOpts)
}

// TotalLiabilities is a free data retrieval call binding the contract method 0xf73579a9.
//
// Solidity: function totalLiabilities() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCaller) TotalLiabilities(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "totalLiabilities")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalLiabilities is a free data retrieval call binding the contract method 0xf73579a9.
//
// Solidity: function totalLiabilities() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolSession) TotalLiabilities() (*big.Int, error) {
	return _NablaSwapPool.Contract.TotalLiabilities(&_NablaSwapPool.CallOpts)
}

// TotalLiabilities is a free data retrieval call binding the contract method 0xf73579a9.
//
// Solidity: function totalLiabilities() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCallerSession) TotalLiabilities() (*big.Int, error) {
	return _NablaSwapPool.Contract.TotalLiabilities(&_NablaSwapPool.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NablaSwapPool.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolSession) TotalSupply() (*big.Int, error) {
	return _NablaSwapPool.Contract.TotalSupply(&_NablaSwapPool.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_NablaSwapPool *NablaSwapPoolCallerSession) TotalSupply() (*big.Int, error) {
	return _NablaSwapPool.Contract.TotalSupply(&_NablaSwapPool.CallOpts)
}

// BackstopBurn is a paid mutator transaction binding the contract method 0xe45f37bd.
//
// Solidity: function backstopBurn(address _owner, uint256 _sharesToBurn) returns(uint256 amount_)
func (_NablaSwapPool *NablaSwapPoolTransactor) BackstopBurn(opts *bind.TransactOpts, _owner common.Address, _sharesToBurn *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "backstopBurn", _owner, _sharesToBurn)
}

// BackstopBurn is a paid mutator transaction binding the contract method 0xe45f37bd.
//
// Solidity: function backstopBurn(address _owner, uint256 _sharesToBurn) returns(uint256 amount_)
func (_NablaSwapPool *NablaSwapPoolSession) BackstopBurn(_owner common.Address, _sharesToBurn *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.BackstopBurn(&_NablaSwapPool.TransactOpts, _owner, _sharesToBurn)
}

// BackstopBurn is a paid mutator transaction binding the contract method 0xe45f37bd.
//
// Solidity: function backstopBurn(address _owner, uint256 _sharesToBurn) returns(uint256 amount_)
func (_NablaSwapPool *NablaSwapPoolTransactorSession) BackstopBurn(_owner common.Address, _sharesToBurn *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.BackstopBurn(&_NablaSwapPool.TransactOpts, _owner, _sharesToBurn)
}

// BackstopDrain is a paid mutator transaction binding the contract method 0xc2cb15de.
//
// Solidity: function backstopDrain(uint256 _amount, address _recipient) returns(uint256 swapAmount_)
func (_NablaSwapPool *NablaSwapPoolTransactor) BackstopDrain(opts *bind.TransactOpts, _amount *big.Int, _recipient common.Address) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "backstopDrain", _amount, _recipient)
}

// BackstopDrain is a paid mutator transaction binding the contract method 0xc2cb15de.
//
// Solidity: function backstopDrain(uint256 _amount, address _recipient) returns(uint256 swapAmount_)
func (_NablaSwapPool *NablaSwapPoolSession) BackstopDrain(_amount *big.Int, _recipient common.Address) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.BackstopDrain(&_NablaSwapPool.TransactOpts, _amount, _recipient)
}

// BackstopDrain is a paid mutator transaction binding the contract method 0xc2cb15de.
//
// Solidity: function backstopDrain(uint256 _amount, address _recipient) returns(uint256 swapAmount_)
func (_NablaSwapPool *NablaSwapPoolTransactorSession) BackstopDrain(_amount *big.Int, _recipient common.Address) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.BackstopDrain(&_NablaSwapPool.TransactOpts, _amount, _recipient)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_NablaSwapPool *NablaSwapPoolTransactor) DecreaseAllowance(opts *bind.TransactOpts, spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "decreaseAllowance", spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_NablaSwapPool *NablaSwapPoolSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.DecreaseAllowance(&_NablaSwapPool.TransactOpts, spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_NablaSwapPool *NablaSwapPoolTransactorSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.DecreaseAllowance(&_NablaSwapPool.TransactOpts, spender, subtractedValue)
}

// Deposit is a paid mutator transaction binding the contract method 0x00aeef8a.
//
// Solidity: function deposit(uint256 _depositAmount, uint256 _minLPAmountOut, uint256 _deadline) returns(uint256 sharesToMint_, int256 fee_)
func (_NablaSwapPool *NablaSwapPoolTransactor) Deposit(opts *bind.TransactOpts, _depositAmount *big.Int, _minLPAmountOut *big.Int, _deadline *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "deposit", _depositAmount, _minLPAmountOut, _deadline)
}

// Deposit is a paid mutator transaction binding the contract method 0x00aeef8a.
//
// Solidity: function deposit(uint256 _depositAmount, uint256 _minLPAmountOut, uint256 _deadline) returns(uint256 sharesToMint_, int256 fee_)
func (_NablaSwapPool *NablaSwapPoolSession) Deposit(_depositAmount *big.Int, _minLPAmountOut *big.Int, _deadline *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.Deposit(&_NablaSwapPool.TransactOpts, _depositAmount, _minLPAmountOut, _deadline)
}

// Deposit is a paid mutator transaction binding the contract method 0x00aeef8a.
//
// Solidity: function deposit(uint256 _depositAmount, uint256 _minLPAmountOut, uint256 _deadline) returns(uint256 sharesToMint_, int256 fee_)
func (_NablaSwapPool *NablaSwapPoolTransactorSession) Deposit(_depositAmount *big.Int, _minLPAmountOut *big.Int, _deadline *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.Deposit(&_NablaSwapPool.TransactOpts, _depositAmount, _minLPAmountOut, _deadline)
}

// DisableGatedAccess is a paid mutator transaction binding the contract method 0xcedca4ff.
//
// Solidity: function disableGatedAccess() returns(bool success_)
func (_NablaSwapPool *NablaSwapPoolTransactor) DisableGatedAccess(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "disableGatedAccess")
}

// DisableGatedAccess is a paid mutator transaction binding the contract method 0xcedca4ff.
//
// Solidity: function disableGatedAccess() returns(bool success_)
func (_NablaSwapPool *NablaSwapPoolSession) DisableGatedAccess() (*types.Transaction, error) {
	return _NablaSwapPool.Contract.DisableGatedAccess(&_NablaSwapPool.TransactOpts)
}

// DisableGatedAccess is a paid mutator transaction binding the contract method 0xcedca4ff.
//
// Solidity: function disableGatedAccess() returns(bool success_)
func (_NablaSwapPool *NablaSwapPoolTransactorSession) DisableGatedAccess() (*types.Transaction, error) {
	return _NablaSwapPool.Contract.DisableGatedAccess(&_NablaSwapPool.TransactOpts)
}

// EnableGatedAccess is a paid mutator transaction binding the contract method 0x6bfdbf3f.
//
// Solidity: function enableGatedAccess() returns(bool success_)
func (_NablaSwapPool *NablaSwapPoolTransactor) EnableGatedAccess(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "enableGatedAccess")
}

// EnableGatedAccess is a paid mutator transaction binding the contract method 0x6bfdbf3f.
//
// Solidity: function enableGatedAccess() returns(bool success_)
func (_NablaSwapPool *NablaSwapPoolSession) EnableGatedAccess() (*types.Transaction, error) {
	return _NablaSwapPool.Contract.EnableGatedAccess(&_NablaSwapPool.TransactOpts)
}

// EnableGatedAccess is a paid mutator transaction binding the contract method 0x6bfdbf3f.
//
// Solidity: function enableGatedAccess() returns(bool success_)
func (_NablaSwapPool *NablaSwapPoolTransactorSession) EnableGatedAccess() (*types.Transaction, error) {
	return _NablaSwapPool.Contract.EnableGatedAccess(&_NablaSwapPool.TransactOpts)
}

// FreezeDepositing is a paid mutator transaction binding the contract method 0xeaccbb75.
//
// Solidity: function freezeDepositing(uint256 _forHowLong) returns()
func (_NablaSwapPool *NablaSwapPoolTransactor) FreezeDepositing(opts *bind.TransactOpts, _forHowLong *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "freezeDepositing", _forHowLong)
}

// FreezeDepositing is a paid mutator transaction binding the contract method 0xeaccbb75.
//
// Solidity: function freezeDepositing(uint256 _forHowLong) returns()
func (_NablaSwapPool *NablaSwapPoolSession) FreezeDepositing(_forHowLong *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.FreezeDepositing(&_NablaSwapPool.TransactOpts, _forHowLong)
}

// FreezeDepositing is a paid mutator transaction binding the contract method 0xeaccbb75.
//
// Solidity: function freezeDepositing(uint256 _forHowLong) returns()
func (_NablaSwapPool *NablaSwapPoolTransactorSession) FreezeDepositing(_forHowLong *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.FreezeDepositing(&_NablaSwapPool.TransactOpts, _forHowLong)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_NablaSwapPool *NablaSwapPoolTransactor) IncreaseAllowance(opts *bind.TransactOpts, spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "increaseAllowance", spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_NablaSwapPool *NablaSwapPoolSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.IncreaseAllowance(&_NablaSwapPool.TransactOpts, spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_NablaSwapPool *NablaSwapPoolTransactorSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.IncreaseAllowance(&_NablaSwapPool.TransactOpts, spender, addedValue)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_NablaSwapPool *NablaSwapPoolTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_NablaSwapPool *NablaSwapPoolSession) Pause() (*types.Transaction, error) {
	return _NablaSwapPool.Contract.Pause(&_NablaSwapPool.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_NablaSwapPool *NablaSwapPoolTransactorSession) Pause() (*types.Transaction, error) {
	return _NablaSwapPool.Contract.Pause(&_NablaSwapPool.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_NablaSwapPool *NablaSwapPoolTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_NablaSwapPool *NablaSwapPoolSession) RenounceOwnership() (*types.Transaction, error) {
	return _NablaSwapPool.Contract.RenounceOwnership(&_NablaSwapPool.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_NablaSwapPool *NablaSwapPoolTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _NablaSwapPool.Contract.RenounceOwnership(&_NablaSwapPool.TransactOpts)
}

// SetGate is a paid mutator transaction binding the contract method 0x88315a40.
//
// Solidity: function setGate(address _newGate) returns(bool success_)
func (_NablaSwapPool *NablaSwapPoolTransactor) SetGate(opts *bind.TransactOpts, _newGate common.Address) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "setGate", _newGate)
}

// SetGate is a paid mutator transaction binding the contract method 0x88315a40.
//
// Solidity: function setGate(address _newGate) returns(bool success_)
func (_NablaSwapPool *NablaSwapPoolSession) SetGate(_newGate common.Address) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.SetGate(&_NablaSwapPool.TransactOpts, _newGate)
}

// SetGate is a paid mutator transaction binding the contract method 0x88315a40.
//
// Solidity: function setGate(address _newGate) returns(bool success_)
func (_NablaSwapPool *NablaSwapPoolTransactorSession) SetGate(_newGate common.Address) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.SetGate(&_NablaSwapPool.TransactOpts, _newGate)
}

// SetMaxCoverageRatioForSwapIn is a paid mutator transaction binding the contract method 0x0668d07c.
//
// Solidity: function setMaxCoverageRatioForSwapIn(uint256 _maxCoverageRatio) returns()
func (_NablaSwapPool *NablaSwapPoolTransactor) SetMaxCoverageRatioForSwapIn(opts *bind.TransactOpts, _maxCoverageRatio *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "setMaxCoverageRatioForSwapIn", _maxCoverageRatio)
}

// SetMaxCoverageRatioForSwapIn is a paid mutator transaction binding the contract method 0x0668d07c.
//
// Solidity: function setMaxCoverageRatioForSwapIn(uint256 _maxCoverageRatio) returns()
func (_NablaSwapPool *NablaSwapPoolSession) SetMaxCoverageRatioForSwapIn(_maxCoverageRatio *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.SetMaxCoverageRatioForSwapIn(&_NablaSwapPool.TransactOpts, _maxCoverageRatio)
}

// SetMaxCoverageRatioForSwapIn is a paid mutator transaction binding the contract method 0x0668d07c.
//
// Solidity: function setMaxCoverageRatioForSwapIn(uint256 _maxCoverageRatio) returns()
func (_NablaSwapPool *NablaSwapPoolTransactorSession) SetMaxCoverageRatioForSwapIn(_maxCoverageRatio *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.SetMaxCoverageRatioForSwapIn(&_NablaSwapPool.TransactOpts, _maxCoverageRatio)
}

// SetPoolCap is a paid mutator transaction binding the contract method 0xd835f535.
//
// Solidity: function setPoolCap(uint256 _maxTokens) returns()
func (_NablaSwapPool *NablaSwapPoolTransactor) SetPoolCap(opts *bind.TransactOpts, _maxTokens *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "setPoolCap", _maxTokens)
}

// SetPoolCap is a paid mutator transaction binding the contract method 0xd835f535.
//
// Solidity: function setPoolCap(uint256 _maxTokens) returns()
func (_NablaSwapPool *NablaSwapPoolSession) SetPoolCap(_maxTokens *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.SetPoolCap(&_NablaSwapPool.TransactOpts, _maxTokens)
}

// SetPoolCap is a paid mutator transaction binding the contract method 0xd835f535.
//
// Solidity: function setPoolCap(uint256 _maxTokens) returns()
func (_NablaSwapPool *NablaSwapPoolTransactorSession) SetPoolCap(_maxTokens *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.SetPoolCap(&_NablaSwapPool.TransactOpts, _maxTokens)
}

// SetProtocolTreasury is a paid mutator transaction binding the contract method 0x0c5a61f8.
//
// Solidity: function setProtocolTreasury(address _newProtocolTreasury) returns(bool success_)
func (_NablaSwapPool *NablaSwapPoolTransactor) SetProtocolTreasury(opts *bind.TransactOpts, _newProtocolTreasury common.Address) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "setProtocolTreasury", _newProtocolTreasury)
}

// SetProtocolTreasury is a paid mutator transaction binding the contract method 0x0c5a61f8.
//
// Solidity: function setProtocolTreasury(address _newProtocolTreasury) returns(bool success_)
func (_NablaSwapPool *NablaSwapPoolSession) SetProtocolTreasury(_newProtocolTreasury common.Address) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.SetProtocolTreasury(&_NablaSwapPool.TransactOpts, _newProtocolTreasury)
}

// SetProtocolTreasury is a paid mutator transaction binding the contract method 0x0c5a61f8.
//
// Solidity: function setProtocolTreasury(address _newProtocolTreasury) returns(bool success_)
func (_NablaSwapPool *NablaSwapPoolTransactorSession) SetProtocolTreasury(_newProtocolTreasury common.Address) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.SetProtocolTreasury(&_NablaSwapPool.TransactOpts, _newProtocolTreasury)
}

// SetSafeRedeemPercentageAndInterval is a paid mutator transaction binding the contract method 0xa7242dae.
//
// Solidity: function setSafeRedeemPercentageAndInterval(uint256 _safeRedeemPercentage, uint256 _safeRedeemInterval) returns(bool success_)
func (_NablaSwapPool *NablaSwapPoolTransactor) SetSafeRedeemPercentageAndInterval(opts *bind.TransactOpts, _safeRedeemPercentage *big.Int, _safeRedeemInterval *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "setSafeRedeemPercentageAndInterval", _safeRedeemPercentage, _safeRedeemInterval)
}

// SetSafeRedeemPercentageAndInterval is a paid mutator transaction binding the contract method 0xa7242dae.
//
// Solidity: function setSafeRedeemPercentageAndInterval(uint256 _safeRedeemPercentage, uint256 _safeRedeemInterval) returns(bool success_)
func (_NablaSwapPool *NablaSwapPoolSession) SetSafeRedeemPercentageAndInterval(_safeRedeemPercentage *big.Int, _safeRedeemInterval *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.SetSafeRedeemPercentageAndInterval(&_NablaSwapPool.TransactOpts, _safeRedeemPercentage, _safeRedeemInterval)
}

// SetSafeRedeemPercentageAndInterval is a paid mutator transaction binding the contract method 0xa7242dae.
//
// Solidity: function setSafeRedeemPercentageAndInterval(uint256 _safeRedeemPercentage, uint256 _safeRedeemInterval) returns(bool success_)
func (_NablaSwapPool *NablaSwapPoolTransactorSession) SetSafeRedeemPercentageAndInterval(_safeRedeemPercentage *big.Int, _safeRedeemInterval *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.SetSafeRedeemPercentageAndInterval(&_NablaSwapPool.TransactOpts, _safeRedeemPercentage, _safeRedeemInterval)
}

// SetSwapFees is a paid mutator transaction binding the contract method 0xeb43434e.
//
// Solidity: function setSwapFees(uint256 _lpFee, uint256 _backstopFee, uint256 _protocolFee) returns()
func (_NablaSwapPool *NablaSwapPoolTransactor) SetSwapFees(opts *bind.TransactOpts, _lpFee *big.Int, _backstopFee *big.Int, _protocolFee *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "setSwapFees", _lpFee, _backstopFee, _protocolFee)
}

// SetSwapFees is a paid mutator transaction binding the contract method 0xeb43434e.
//
// Solidity: function setSwapFees(uint256 _lpFee, uint256 _backstopFee, uint256 _protocolFee) returns()
func (_NablaSwapPool *NablaSwapPoolSession) SetSwapFees(_lpFee *big.Int, _backstopFee *big.Int, _protocolFee *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.SetSwapFees(&_NablaSwapPool.TransactOpts, _lpFee, _backstopFee, _protocolFee)
}

// SetSwapFees is a paid mutator transaction binding the contract method 0xeb43434e.
//
// Solidity: function setSwapFees(uint256 _lpFee, uint256 _backstopFee, uint256 _protocolFee) returns()
func (_NablaSwapPool *NablaSwapPoolTransactorSession) SetSwapFees(_lpFee *big.Int, _backstopFee *big.Int, _protocolFee *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.SetSwapFees(&_NablaSwapPool.TransactOpts, _lpFee, _backstopFee, _protocolFee)
}

// SwapIntoFromRouter is a paid mutator transaction binding the contract method 0x4d8ea83f.
//
// Solidity: function swapIntoFromRouter(uint256 _amount) returns(uint256 effectiveAmount_)
func (_NablaSwapPool *NablaSwapPoolTransactor) SwapIntoFromRouter(opts *bind.TransactOpts, _amount *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "swapIntoFromRouter", _amount)
}

// SwapIntoFromRouter is a paid mutator transaction binding the contract method 0x4d8ea83f.
//
// Solidity: function swapIntoFromRouter(uint256 _amount) returns(uint256 effectiveAmount_)
func (_NablaSwapPool *NablaSwapPoolSession) SwapIntoFromRouter(_amount *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.SwapIntoFromRouter(&_NablaSwapPool.TransactOpts, _amount)
}

// SwapIntoFromRouter is a paid mutator transaction binding the contract method 0x4d8ea83f.
//
// Solidity: function swapIntoFromRouter(uint256 _amount) returns(uint256 effectiveAmount_)
func (_NablaSwapPool *NablaSwapPoolTransactorSession) SwapIntoFromRouter(_amount *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.SwapIntoFromRouter(&_NablaSwapPool.TransactOpts, _amount)
}

// SwapOutFromRouter is a paid mutator transaction binding the contract method 0x5f79d44f.
//
// Solidity: function swapOutFromRouter(uint256 _amount) returns(uint256 effectiveAmount_)
func (_NablaSwapPool *NablaSwapPoolTransactor) SwapOutFromRouter(opts *bind.TransactOpts, _amount *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "swapOutFromRouter", _amount)
}

// SwapOutFromRouter is a paid mutator transaction binding the contract method 0x5f79d44f.
//
// Solidity: function swapOutFromRouter(uint256 _amount) returns(uint256 effectiveAmount_)
func (_NablaSwapPool *NablaSwapPoolSession) SwapOutFromRouter(_amount *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.SwapOutFromRouter(&_NablaSwapPool.TransactOpts, _amount)
}

// SwapOutFromRouter is a paid mutator transaction binding the contract method 0x5f79d44f.
//
// Solidity: function swapOutFromRouter(uint256 _amount) returns(uint256 effectiveAmount_)
func (_NablaSwapPool *NablaSwapPoolTransactorSession) SwapOutFromRouter(_amount *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.SwapOutFromRouter(&_NablaSwapPool.TransactOpts, _amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_NablaSwapPool *NablaSwapPoolTransactor) Transfer(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "transfer", to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_NablaSwapPool *NablaSwapPoolSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.Transfer(&_NablaSwapPool.TransactOpts, to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_NablaSwapPool *NablaSwapPoolTransactorSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.Transfer(&_NablaSwapPool.TransactOpts, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_NablaSwapPool *NablaSwapPoolTransactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "transferFrom", from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_NablaSwapPool *NablaSwapPoolSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.TransferFrom(&_NablaSwapPool.TransactOpts, from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_NablaSwapPool *NablaSwapPoolTransactorSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.TransferFrom(&_NablaSwapPool.TransactOpts, from, to, amount)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_NablaSwapPool *NablaSwapPoolTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_NablaSwapPool *NablaSwapPoolSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.TransferOwnership(&_NablaSwapPool.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_NablaSwapPool *NablaSwapPoolTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.TransferOwnership(&_NablaSwapPool.TransactOpts, newOwner)
}

// UnfreezeDepositing is a paid mutator transaction binding the contract method 0x7db54075.
//
// Solidity: function unfreezeDepositing() returns()
func (_NablaSwapPool *NablaSwapPoolTransactor) UnfreezeDepositing(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "unfreezeDepositing")
}

// UnfreezeDepositing is a paid mutator transaction binding the contract method 0x7db54075.
//
// Solidity: function unfreezeDepositing() returns()
func (_NablaSwapPool *NablaSwapPoolSession) UnfreezeDepositing() (*types.Transaction, error) {
	return _NablaSwapPool.Contract.UnfreezeDepositing(&_NablaSwapPool.TransactOpts)
}

// UnfreezeDepositing is a paid mutator transaction binding the contract method 0x7db54075.
//
// Solidity: function unfreezeDepositing() returns()
func (_NablaSwapPool *NablaSwapPoolTransactorSession) UnfreezeDepositing() (*types.Transaction, error) {
	return _NablaSwapPool.Contract.UnfreezeDepositing(&_NablaSwapPool.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_NablaSwapPool *NablaSwapPoolTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_NablaSwapPool *NablaSwapPoolSession) Unpause() (*types.Transaction, error) {
	return _NablaSwapPool.Contract.Unpause(&_NablaSwapPool.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_NablaSwapPool *NablaSwapPoolTransactorSession) Unpause() (*types.Transaction, error) {
	return _NablaSwapPool.Contract.Unpause(&_NablaSwapPool.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0xa41fe49f.
//
// Solidity: function withdraw(uint256 _sharesToBurn, uint256 _minimumAmount, uint256 _deadline) returns(uint256 payoutAmount_, int256 fee_)
func (_NablaSwapPool *NablaSwapPoolTransactor) Withdraw(opts *bind.TransactOpts, _sharesToBurn *big.Int, _minimumAmount *big.Int, _deadline *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.contract.Transact(opts, "withdraw", _sharesToBurn, _minimumAmount, _deadline)
}

// Withdraw is a paid mutator transaction binding the contract method 0xa41fe49f.
//
// Solidity: function withdraw(uint256 _sharesToBurn, uint256 _minimumAmount, uint256 _deadline) returns(uint256 payoutAmount_, int256 fee_)
func (_NablaSwapPool *NablaSwapPoolSession) Withdraw(_sharesToBurn *big.Int, _minimumAmount *big.Int, _deadline *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.Withdraw(&_NablaSwapPool.TransactOpts, _sharesToBurn, _minimumAmount, _deadline)
}

// Withdraw is a paid mutator transaction binding the contract method 0xa41fe49f.
//
// Solidity: function withdraw(uint256 _sharesToBurn, uint256 _minimumAmount, uint256 _deadline) returns(uint256 payoutAmount_, int256 fee_)
func (_NablaSwapPool *NablaSwapPoolTransactorSession) Withdraw(_sharesToBurn *big.Int, _minimumAmount *big.Int, _deadline *big.Int) (*types.Transaction, error) {
	return _NablaSwapPool.Contract.Withdraw(&_NablaSwapPool.TransactOpts, _sharesToBurn, _minimumAmount, _deadline)
}

// NablaSwapPoolApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the NablaSwapPool contract.
type NablaSwapPoolApprovalIterator struct {
	Event *NablaSwapPoolApproval // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolApproval)
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
		it.Event = new(NablaSwapPoolApproval)
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
func (it *NablaSwapPoolApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolApproval represents a Approval event raised by the NablaSwapPool contract.
type NablaSwapPoolApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*NablaSwapPoolApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolApprovalIterator{contract: _NablaSwapPool.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolApproval)
				if err := _NablaSwapPool.contract.UnpackLog(event, "Approval", log); err != nil {
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
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseApproval(log types.Log) (*NablaSwapPoolApproval, error) {
	event := new(NablaSwapPoolApproval)
	if err := _NablaSwapPool.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolBackstopBurnIterator is returned from FilterBackstopBurn and is used to iterate over the raw logs and unpacked data for BackstopBurn events raised by the NablaSwapPool contract.
type NablaSwapPoolBackstopBurnIterator struct {
	Event *NablaSwapPoolBackstopBurn // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolBackstopBurnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolBackstopBurn)
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
		it.Event = new(NablaSwapPoolBackstopBurn)
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
func (it *NablaSwapPoolBackstopBurnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolBackstopBurnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolBackstopBurn represents a BackstopBurn event raised by the NablaSwapPool contract.
type NablaSwapPoolBackstopBurn struct {
	Owner        common.Address
	SharesToBurn *big.Int
	Amount       *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterBackstopBurn is a free log retrieval operation binding the contract event 0x97f21f75943bcc0594d00eb5038083d7f90822ca6b0c632bf75188950a3e2131.
//
// Solidity: event BackstopBurn(address indexed owner, uint256 sharesToBurn, uint256 amount)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterBackstopBurn(opts *bind.FilterOpts, owner []common.Address) (*NablaSwapPoolBackstopBurnIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "BackstopBurn", ownerRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolBackstopBurnIterator{contract: _NablaSwapPool.contract, event: "BackstopBurn", logs: logs, sub: sub}, nil
}

// WatchBackstopBurn is a free log subscription operation binding the contract event 0x97f21f75943bcc0594d00eb5038083d7f90822ca6b0c632bf75188950a3e2131.
//
// Solidity: event BackstopBurn(address indexed owner, uint256 sharesToBurn, uint256 amount)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchBackstopBurn(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolBackstopBurn, owner []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "BackstopBurn", ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolBackstopBurn)
				if err := _NablaSwapPool.contract.UnpackLog(event, "BackstopBurn", log); err != nil {
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

// ParseBackstopBurn is a log parse operation binding the contract event 0x97f21f75943bcc0594d00eb5038083d7f90822ca6b0c632bf75188950a3e2131.
//
// Solidity: event BackstopBurn(address indexed owner, uint256 sharesToBurn, uint256 amount)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseBackstopBurn(log types.Log) (*NablaSwapPoolBackstopBurn, error) {
	event := new(NablaSwapPoolBackstopBurn)
	if err := _NablaSwapPool.contract.UnpackLog(event, "BackstopBurn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolBackstopDrainIterator is returned from FilterBackstopDrain and is used to iterate over the raw logs and unpacked data for BackstopDrain events raised by the NablaSwapPool contract.
type NablaSwapPoolBackstopDrainIterator struct {
	Event *NablaSwapPoolBackstopDrain // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolBackstopDrainIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolBackstopDrain)
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
		it.Event = new(NablaSwapPoolBackstopDrain)
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
func (it *NablaSwapPoolBackstopDrainIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolBackstopDrainIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolBackstopDrain represents a BackstopDrain event raised by the NablaSwapPool contract.
type NablaSwapPoolBackstopDrain struct {
	Recipient        common.Address
	AmountSwapTokens *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterBackstopDrain is a free log retrieval operation binding the contract event 0x439c15c5ffd384d65af60124e574f5642b7e5d6750b762ddeef70abac573ab27.
//
// Solidity: event BackstopDrain(address indexed recipient, uint256 amountSwapTokens)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterBackstopDrain(opts *bind.FilterOpts, recipient []common.Address) (*NablaSwapPoolBackstopDrainIterator, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "BackstopDrain", recipientRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolBackstopDrainIterator{contract: _NablaSwapPool.contract, event: "BackstopDrain", logs: logs, sub: sub}, nil
}

// WatchBackstopDrain is a free log subscription operation binding the contract event 0x439c15c5ffd384d65af60124e574f5642b7e5d6750b762ddeef70abac573ab27.
//
// Solidity: event BackstopDrain(address indexed recipient, uint256 amountSwapTokens)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchBackstopDrain(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolBackstopDrain, recipient []common.Address) (event.Subscription, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "BackstopDrain", recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolBackstopDrain)
				if err := _NablaSwapPool.contract.UnpackLog(event, "BackstopDrain", log); err != nil {
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

// ParseBackstopDrain is a log parse operation binding the contract event 0x439c15c5ffd384d65af60124e574f5642b7e5d6750b762ddeef70abac573ab27.
//
// Solidity: event BackstopDrain(address indexed recipient, uint256 amountSwapTokens)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseBackstopDrain(log types.Log) (*NablaSwapPoolBackstopDrain, error) {
	event := new(NablaSwapPoolBackstopDrain)
	if err := _NablaSwapPool.contract.UnpackLog(event, "BackstopDrain", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolChargedSwapFeesIterator is returned from FilterChargedSwapFees and is used to iterate over the raw logs and unpacked data for ChargedSwapFees events raised by the NablaSwapPool contract.
type NablaSwapPoolChargedSwapFeesIterator struct {
	Event *NablaSwapPoolChargedSwapFees // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolChargedSwapFeesIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolChargedSwapFees)
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
		it.Event = new(NablaSwapPoolChargedSwapFees)
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
func (it *NablaSwapPoolChargedSwapFeesIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolChargedSwapFeesIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolChargedSwapFees represents a ChargedSwapFees event raised by the NablaSwapPool contract.
type NablaSwapPoolChargedSwapFees struct {
	LpFees       *big.Int
	BackstopFees *big.Int
	ProtocolFees *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterChargedSwapFees is a free log retrieval operation binding the contract event 0x3eb07265dc949e6776beb7b2e85d9e292a8a411eabd500cbe06b6bec16d87721.
//
// Solidity: event ChargedSwapFees(uint256 lpFees, uint256 backstopFees, uint256 protocolFees)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterChargedSwapFees(opts *bind.FilterOpts) (*NablaSwapPoolChargedSwapFeesIterator, error) {

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "ChargedSwapFees")
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolChargedSwapFeesIterator{contract: _NablaSwapPool.contract, event: "ChargedSwapFees", logs: logs, sub: sub}, nil
}

// WatchChargedSwapFees is a free log subscription operation binding the contract event 0x3eb07265dc949e6776beb7b2e85d9e292a8a411eabd500cbe06b6bec16d87721.
//
// Solidity: event ChargedSwapFees(uint256 lpFees, uint256 backstopFees, uint256 protocolFees)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchChargedSwapFees(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolChargedSwapFees) (event.Subscription, error) {

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "ChargedSwapFees")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolChargedSwapFees)
				if err := _NablaSwapPool.contract.UnpackLog(event, "ChargedSwapFees", log); err != nil {
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

// ParseChargedSwapFees is a log parse operation binding the contract event 0x3eb07265dc949e6776beb7b2e85d9e292a8a411eabd500cbe06b6bec16d87721.
//
// Solidity: event ChargedSwapFees(uint256 lpFees, uint256 backstopFees, uint256 protocolFees)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseChargedSwapFees(log types.Log) (*NablaSwapPoolChargedSwapFees, error) {
	event := new(NablaSwapPoolChargedSwapFees)
	if err := _NablaSwapPool.contract.UnpackLog(event, "ChargedSwapFees", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolDepositingFrozenIterator is returned from FilterDepositingFrozen and is used to iterate over the raw logs and unpacked data for DepositingFrozen events raised by the NablaSwapPool contract.
type NablaSwapPoolDepositingFrozenIterator struct {
	Event *NablaSwapPoolDepositingFrozen // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolDepositingFrozenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolDepositingFrozen)
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
		it.Event = new(NablaSwapPoolDepositingFrozen)
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
func (it *NablaSwapPoolDepositingFrozenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolDepositingFrozenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolDepositingFrozen represents a DepositingFrozen event raised by the NablaSwapPool contract.
type NablaSwapPoolDepositingFrozen struct {
	FrozenUntil *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterDepositingFrozen is a free log retrieval operation binding the contract event 0xfec0706bd021b215f399d77a94fc7b5ee4de46581179c34076729c3f279710a1.
//
// Solidity: event DepositingFrozen(uint256 frozenUntil)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterDepositingFrozen(opts *bind.FilterOpts) (*NablaSwapPoolDepositingFrozenIterator, error) {

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "DepositingFrozen")
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolDepositingFrozenIterator{contract: _NablaSwapPool.contract, event: "DepositingFrozen", logs: logs, sub: sub}, nil
}

// WatchDepositingFrozen is a free log subscription operation binding the contract event 0xfec0706bd021b215f399d77a94fc7b5ee4de46581179c34076729c3f279710a1.
//
// Solidity: event DepositingFrozen(uint256 frozenUntil)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchDepositingFrozen(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolDepositingFrozen) (event.Subscription, error) {

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "DepositingFrozen")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolDepositingFrozen)
				if err := _NablaSwapPool.contract.UnpackLog(event, "DepositingFrozen", log); err != nil {
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

// ParseDepositingFrozen is a log parse operation binding the contract event 0xfec0706bd021b215f399d77a94fc7b5ee4de46581179c34076729c3f279710a1.
//
// Solidity: event DepositingFrozen(uint256 frozenUntil)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseDepositingFrozen(log types.Log) (*NablaSwapPoolDepositingFrozen, error) {
	event := new(NablaSwapPoolDepositingFrozen)
	if err := _NablaSwapPool.contract.UnpackLog(event, "DepositingFrozen", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolDepositingFrozenByOwnerIterator is returned from FilterDepositingFrozenByOwner and is used to iterate over the raw logs and unpacked data for DepositingFrozenByOwner events raised by the NablaSwapPool contract.
type NablaSwapPoolDepositingFrozenByOwnerIterator struct {
	Event *NablaSwapPoolDepositingFrozenByOwner // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolDepositingFrozenByOwnerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolDepositingFrozenByOwner)
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
		it.Event = new(NablaSwapPoolDepositingFrozenByOwner)
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
func (it *NablaSwapPoolDepositingFrozenByOwnerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolDepositingFrozenByOwnerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolDepositingFrozenByOwner represents a DepositingFrozenByOwner event raised by the NablaSwapPool contract.
type NablaSwapPoolDepositingFrozenByOwner struct {
	Sender         common.Address
	FreezeDuration *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterDepositingFrozenByOwner is a free log retrieval operation binding the contract event 0x23dbd77920ff24f21dde956338a6187be877b6f210124e8a00417afa51a5d01a.
//
// Solidity: event DepositingFrozenByOwner(address indexed sender, uint256 freezeDuration)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterDepositingFrozenByOwner(opts *bind.FilterOpts, sender []common.Address) (*NablaSwapPoolDepositingFrozenByOwnerIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "DepositingFrozenByOwner", senderRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolDepositingFrozenByOwnerIterator{contract: _NablaSwapPool.contract, event: "DepositingFrozenByOwner", logs: logs, sub: sub}, nil
}

// WatchDepositingFrozenByOwner is a free log subscription operation binding the contract event 0x23dbd77920ff24f21dde956338a6187be877b6f210124e8a00417afa51a5d01a.
//
// Solidity: event DepositingFrozenByOwner(address indexed sender, uint256 freezeDuration)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchDepositingFrozenByOwner(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolDepositingFrozenByOwner, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "DepositingFrozenByOwner", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolDepositingFrozenByOwner)
				if err := _NablaSwapPool.contract.UnpackLog(event, "DepositingFrozenByOwner", log); err != nil {
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

// ParseDepositingFrozenByOwner is a log parse operation binding the contract event 0x23dbd77920ff24f21dde956338a6187be877b6f210124e8a00417afa51a5d01a.
//
// Solidity: event DepositingFrozenByOwner(address indexed sender, uint256 freezeDuration)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseDepositingFrozenByOwner(log types.Log) (*NablaSwapPoolDepositingFrozenByOwner, error) {
	event := new(NablaSwapPoolDepositingFrozenByOwner)
	if err := _NablaSwapPool.contract.UnpackLog(event, "DepositingFrozenByOwner", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolDepositingUnfrozenByOwnerIterator is returned from FilterDepositingUnfrozenByOwner and is used to iterate over the raw logs and unpacked data for DepositingUnfrozenByOwner events raised by the NablaSwapPool contract.
type NablaSwapPoolDepositingUnfrozenByOwnerIterator struct {
	Event *NablaSwapPoolDepositingUnfrozenByOwner // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolDepositingUnfrozenByOwnerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolDepositingUnfrozenByOwner)
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
		it.Event = new(NablaSwapPoolDepositingUnfrozenByOwner)
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
func (it *NablaSwapPoolDepositingUnfrozenByOwnerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolDepositingUnfrozenByOwnerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolDepositingUnfrozenByOwner represents a DepositingUnfrozenByOwner event raised by the NablaSwapPool contract.
type NablaSwapPoolDepositingUnfrozenByOwner struct {
	Sender common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterDepositingUnfrozenByOwner is a free log retrieval operation binding the contract event 0x6e3b2f8f185483a81012599582ce7e01e309c47e9d7fccbbaecd8fea3620ef7e.
//
// Solidity: event DepositingUnfrozenByOwner(address indexed sender)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterDepositingUnfrozenByOwner(opts *bind.FilterOpts, sender []common.Address) (*NablaSwapPoolDepositingUnfrozenByOwnerIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "DepositingUnfrozenByOwner", senderRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolDepositingUnfrozenByOwnerIterator{contract: _NablaSwapPool.contract, event: "DepositingUnfrozenByOwner", logs: logs, sub: sub}, nil
}

// WatchDepositingUnfrozenByOwner is a free log subscription operation binding the contract event 0x6e3b2f8f185483a81012599582ce7e01e309c47e9d7fccbbaecd8fea3620ef7e.
//
// Solidity: event DepositingUnfrozenByOwner(address indexed sender)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchDepositingUnfrozenByOwner(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolDepositingUnfrozenByOwner, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "DepositingUnfrozenByOwner", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolDepositingUnfrozenByOwner)
				if err := _NablaSwapPool.contract.UnpackLog(event, "DepositingUnfrozenByOwner", log); err != nil {
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

// ParseDepositingUnfrozenByOwner is a log parse operation binding the contract event 0x6e3b2f8f185483a81012599582ce7e01e309c47e9d7fccbbaecd8fea3620ef7e.
//
// Solidity: event DepositingUnfrozenByOwner(address indexed sender)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseDepositingUnfrozenByOwner(log types.Log) (*NablaSwapPoolDepositingUnfrozenByOwner, error) {
	event := new(NablaSwapPoolDepositingUnfrozenByOwner)
	if err := _NablaSwapPool.contract.UnpackLog(event, "DepositingUnfrozenByOwner", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolGateUpdatedIterator is returned from FilterGateUpdated and is used to iterate over the raw logs and unpacked data for GateUpdated events raised by the NablaSwapPool contract.
type NablaSwapPoolGateUpdatedIterator struct {
	Event *NablaSwapPoolGateUpdated // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolGateUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolGateUpdated)
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
		it.Event = new(NablaSwapPoolGateUpdated)
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
func (it *NablaSwapPoolGateUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolGateUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolGateUpdated represents a GateUpdated event raised by the NablaSwapPool contract.
type NablaSwapPoolGateUpdated struct {
	Owner   common.Address
	OldGate common.Address
	NewGate common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterGateUpdated is a free log retrieval operation binding the contract event 0x813354ca4cb97163408e1ba5321da6835530227f79ec18ff6959761817379571.
//
// Solidity: event GateUpdated(address indexed owner, address indexed oldGate, address indexed newGate)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterGateUpdated(opts *bind.FilterOpts, owner []common.Address, oldGate []common.Address, newGate []common.Address) (*NablaSwapPoolGateUpdatedIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var oldGateRule []interface{}
	for _, oldGateItem := range oldGate {
		oldGateRule = append(oldGateRule, oldGateItem)
	}
	var newGateRule []interface{}
	for _, newGateItem := range newGate {
		newGateRule = append(newGateRule, newGateItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "GateUpdated", ownerRule, oldGateRule, newGateRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolGateUpdatedIterator{contract: _NablaSwapPool.contract, event: "GateUpdated", logs: logs, sub: sub}, nil
}

// WatchGateUpdated is a free log subscription operation binding the contract event 0x813354ca4cb97163408e1ba5321da6835530227f79ec18ff6959761817379571.
//
// Solidity: event GateUpdated(address indexed owner, address indexed oldGate, address indexed newGate)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchGateUpdated(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolGateUpdated, owner []common.Address, oldGate []common.Address, newGate []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var oldGateRule []interface{}
	for _, oldGateItem := range oldGate {
		oldGateRule = append(oldGateRule, oldGateItem)
	}
	var newGateRule []interface{}
	for _, newGateItem := range newGate {
		newGateRule = append(newGateRule, newGateItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "GateUpdated", ownerRule, oldGateRule, newGateRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolGateUpdated)
				if err := _NablaSwapPool.contract.UnpackLog(event, "GateUpdated", log); err != nil {
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

// ParseGateUpdated is a log parse operation binding the contract event 0x813354ca4cb97163408e1ba5321da6835530227f79ec18ff6959761817379571.
//
// Solidity: event GateUpdated(address indexed owner, address indexed oldGate, address indexed newGate)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseGateUpdated(log types.Log) (*NablaSwapPoolGateUpdated, error) {
	event := new(NablaSwapPoolGateUpdated)
	if err := _NablaSwapPool.contract.UnpackLog(event, "GateUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolGatedAccessDisabledIterator is returned from FilterGatedAccessDisabled and is used to iterate over the raw logs and unpacked data for GatedAccessDisabled events raised by the NablaSwapPool contract.
type NablaSwapPoolGatedAccessDisabledIterator struct {
	Event *NablaSwapPoolGatedAccessDisabled // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolGatedAccessDisabledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolGatedAccessDisabled)
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
		it.Event = new(NablaSwapPoolGatedAccessDisabled)
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
func (it *NablaSwapPoolGatedAccessDisabledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolGatedAccessDisabledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolGatedAccessDisabled represents a GatedAccessDisabled event raised by the NablaSwapPool contract.
type NablaSwapPoolGatedAccessDisabled struct {
	Owner common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterGatedAccessDisabled is a free log retrieval operation binding the contract event 0xfd6cf388b28ff97ffec560bff7ff5da92131bff2432800a51593e25d6975d11a.
//
// Solidity: event GatedAccessDisabled(address indexed owner)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterGatedAccessDisabled(opts *bind.FilterOpts, owner []common.Address) (*NablaSwapPoolGatedAccessDisabledIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "GatedAccessDisabled", ownerRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolGatedAccessDisabledIterator{contract: _NablaSwapPool.contract, event: "GatedAccessDisabled", logs: logs, sub: sub}, nil
}

// WatchGatedAccessDisabled is a free log subscription operation binding the contract event 0xfd6cf388b28ff97ffec560bff7ff5da92131bff2432800a51593e25d6975d11a.
//
// Solidity: event GatedAccessDisabled(address indexed owner)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchGatedAccessDisabled(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolGatedAccessDisabled, owner []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "GatedAccessDisabled", ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolGatedAccessDisabled)
				if err := _NablaSwapPool.contract.UnpackLog(event, "GatedAccessDisabled", log); err != nil {
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

// ParseGatedAccessDisabled is a log parse operation binding the contract event 0xfd6cf388b28ff97ffec560bff7ff5da92131bff2432800a51593e25d6975d11a.
//
// Solidity: event GatedAccessDisabled(address indexed owner)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseGatedAccessDisabled(log types.Log) (*NablaSwapPoolGatedAccessDisabled, error) {
	event := new(NablaSwapPoolGatedAccessDisabled)
	if err := _NablaSwapPool.contract.UnpackLog(event, "GatedAccessDisabled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolGatedAccessEnabledIterator is returned from FilterGatedAccessEnabled and is used to iterate over the raw logs and unpacked data for GatedAccessEnabled events raised by the NablaSwapPool contract.
type NablaSwapPoolGatedAccessEnabledIterator struct {
	Event *NablaSwapPoolGatedAccessEnabled // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolGatedAccessEnabledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolGatedAccessEnabled)
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
		it.Event = new(NablaSwapPoolGatedAccessEnabled)
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
func (it *NablaSwapPoolGatedAccessEnabledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolGatedAccessEnabledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolGatedAccessEnabled represents a GatedAccessEnabled event raised by the NablaSwapPool contract.
type NablaSwapPoolGatedAccessEnabled struct {
	Owner common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterGatedAccessEnabled is a free log retrieval operation binding the contract event 0x31a73b480536599fe974b9122c8a0f5c6a4236d8dc8355dadbc84ec70e336563.
//
// Solidity: event GatedAccessEnabled(address indexed owner)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterGatedAccessEnabled(opts *bind.FilterOpts, owner []common.Address) (*NablaSwapPoolGatedAccessEnabledIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "GatedAccessEnabled", ownerRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolGatedAccessEnabledIterator{contract: _NablaSwapPool.contract, event: "GatedAccessEnabled", logs: logs, sub: sub}, nil
}

// WatchGatedAccessEnabled is a free log subscription operation binding the contract event 0x31a73b480536599fe974b9122c8a0f5c6a4236d8dc8355dadbc84ec70e336563.
//
// Solidity: event GatedAccessEnabled(address indexed owner)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchGatedAccessEnabled(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolGatedAccessEnabled, owner []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "GatedAccessEnabled", ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolGatedAccessEnabled)
				if err := _NablaSwapPool.contract.UnpackLog(event, "GatedAccessEnabled", log); err != nil {
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

// ParseGatedAccessEnabled is a log parse operation binding the contract event 0x31a73b480536599fe974b9122c8a0f5c6a4236d8dc8355dadbc84ec70e336563.
//
// Solidity: event GatedAccessEnabled(address indexed owner)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseGatedAccessEnabled(log types.Log) (*NablaSwapPoolGatedAccessEnabled, error) {
	event := new(NablaSwapPoolGatedAccessEnabled)
	if err := _NablaSwapPool.contract.UnpackLog(event, "GatedAccessEnabled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolMaxCoverageRatioForSwapInSetIterator is returned from FilterMaxCoverageRatioForSwapInSet and is used to iterate over the raw logs and unpacked data for MaxCoverageRatioForSwapInSet events raised by the NablaSwapPool contract.
type NablaSwapPoolMaxCoverageRatioForSwapInSetIterator struct {
	Event *NablaSwapPoolMaxCoverageRatioForSwapInSet // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolMaxCoverageRatioForSwapInSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolMaxCoverageRatioForSwapInSet)
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
		it.Event = new(NablaSwapPoolMaxCoverageRatioForSwapInSet)
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
func (it *NablaSwapPoolMaxCoverageRatioForSwapInSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolMaxCoverageRatioForSwapInSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolMaxCoverageRatioForSwapInSet represents a MaxCoverageRatioForSwapInSet event raised by the NablaSwapPool contract.
type NablaSwapPoolMaxCoverageRatioForSwapInSet struct {
	Sender           common.Address
	MaxCoverageRatio *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterMaxCoverageRatioForSwapInSet is a free log retrieval operation binding the contract event 0x2ce654ac95050edca59bac0cbd3a69fef35903527a00d735d45b82d96f24d6af.
//
// Solidity: event MaxCoverageRatioForSwapInSet(address indexed sender, uint256 maxCoverageRatio)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterMaxCoverageRatioForSwapInSet(opts *bind.FilterOpts, sender []common.Address) (*NablaSwapPoolMaxCoverageRatioForSwapInSetIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "MaxCoverageRatioForSwapInSet", senderRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolMaxCoverageRatioForSwapInSetIterator{contract: _NablaSwapPool.contract, event: "MaxCoverageRatioForSwapInSet", logs: logs, sub: sub}, nil
}

// WatchMaxCoverageRatioForSwapInSet is a free log subscription operation binding the contract event 0x2ce654ac95050edca59bac0cbd3a69fef35903527a00d735d45b82d96f24d6af.
//
// Solidity: event MaxCoverageRatioForSwapInSet(address indexed sender, uint256 maxCoverageRatio)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchMaxCoverageRatioForSwapInSet(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolMaxCoverageRatioForSwapInSet, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "MaxCoverageRatioForSwapInSet", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolMaxCoverageRatioForSwapInSet)
				if err := _NablaSwapPool.contract.UnpackLog(event, "MaxCoverageRatioForSwapInSet", log); err != nil {
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

// ParseMaxCoverageRatioForSwapInSet is a log parse operation binding the contract event 0x2ce654ac95050edca59bac0cbd3a69fef35903527a00d735d45b82d96f24d6af.
//
// Solidity: event MaxCoverageRatioForSwapInSet(address indexed sender, uint256 maxCoverageRatio)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseMaxCoverageRatioForSwapInSet(log types.Log) (*NablaSwapPoolMaxCoverageRatioForSwapInSet, error) {
	event := new(NablaSwapPoolMaxCoverageRatioForSwapInSet)
	if err := _NablaSwapPool.contract.UnpackLog(event, "MaxCoverageRatioForSwapInSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolMintIterator is returned from FilterMint and is used to iterate over the raw logs and unpacked data for Mint events raised by the NablaSwapPool contract.
type NablaSwapPoolMintIterator struct {
	Event *NablaSwapPoolMint // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolMintIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolMint)
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
		it.Event = new(NablaSwapPoolMint)
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
func (it *NablaSwapPoolMintIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolMintIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolMint represents a Mint event raised by the NablaSwapPool contract.
type NablaSwapPoolMint struct {
	Sender           common.Address
	PoolSharesMinted *big.Int
	Fee              *big.Int
	AmountDeposited  *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterMint is a free log retrieval operation binding the contract event 0x5b59f4d0107f10a3b93fbfc9dceb27d728c1e61fc71534ea39038e7685813ee2.
//
// Solidity: event Mint(address indexed sender, uint256 poolSharesMinted, int256 fee, uint256 amountDeposited)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterMint(opts *bind.FilterOpts, sender []common.Address) (*NablaSwapPoolMintIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "Mint", senderRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolMintIterator{contract: _NablaSwapPool.contract, event: "Mint", logs: logs, sub: sub}, nil
}

// WatchMint is a free log subscription operation binding the contract event 0x5b59f4d0107f10a3b93fbfc9dceb27d728c1e61fc71534ea39038e7685813ee2.
//
// Solidity: event Mint(address indexed sender, uint256 poolSharesMinted, int256 fee, uint256 amountDeposited)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchMint(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolMint, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "Mint", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolMint)
				if err := _NablaSwapPool.contract.UnpackLog(event, "Mint", log); err != nil {
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

// ParseMint is a log parse operation binding the contract event 0x5b59f4d0107f10a3b93fbfc9dceb27d728c1e61fc71534ea39038e7685813ee2.
//
// Solidity: event Mint(address indexed sender, uint256 poolSharesMinted, int256 fee, uint256 amountDeposited)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseMint(log types.Log) (*NablaSwapPoolMint, error) {
	event := new(NablaSwapPoolMint)
	if err := _NablaSwapPool.contract.UnpackLog(event, "Mint", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the NablaSwapPool contract.
type NablaSwapPoolOwnershipTransferredIterator struct {
	Event *NablaSwapPoolOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolOwnershipTransferred)
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
		it.Event = new(NablaSwapPoolOwnershipTransferred)
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
func (it *NablaSwapPoolOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolOwnershipTransferred represents a OwnershipTransferred event raised by the NablaSwapPool contract.
type NablaSwapPoolOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*NablaSwapPoolOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolOwnershipTransferredIterator{contract: _NablaSwapPool.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolOwnershipTransferred)
				if err := _NablaSwapPool.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseOwnershipTransferred(log types.Log) (*NablaSwapPoolOwnershipTransferred, error) {
	event := new(NablaSwapPoolOwnershipTransferred)
	if err := _NablaSwapPool.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the NablaSwapPool contract.
type NablaSwapPoolPausedIterator struct {
	Event *NablaSwapPoolPaused // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolPaused)
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
		it.Event = new(NablaSwapPoolPaused)
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
func (it *NablaSwapPoolPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolPaused represents a Paused event raised by the NablaSwapPool contract.
type NablaSwapPoolPaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterPaused(opts *bind.FilterOpts) (*NablaSwapPoolPausedIterator, error) {

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolPausedIterator{contract: _NablaSwapPool.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolPaused) (event.Subscription, error) {

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolPaused)
				if err := _NablaSwapPool.contract.UnpackLog(event, "Paused", log); err != nil {
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

// ParsePaused is a log parse operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParsePaused(log types.Log) (*NablaSwapPoolPaused, error) {
	event := new(NablaSwapPoolPaused)
	if err := _NablaSwapPool.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolPoolCapSetIterator is returned from FilterPoolCapSet and is used to iterate over the raw logs and unpacked data for PoolCapSet events raised by the NablaSwapPool contract.
type NablaSwapPoolPoolCapSetIterator struct {
	Event *NablaSwapPoolPoolCapSet // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolPoolCapSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolPoolCapSet)
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
		it.Event = new(NablaSwapPoolPoolCapSet)
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
func (it *NablaSwapPoolPoolCapSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolPoolCapSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolPoolCapSet represents a PoolCapSet event raised by the NablaSwapPool contract.
type NablaSwapPoolPoolCapSet struct {
	Sender     common.Address
	OldPoolCap *big.Int
	NewPoolCap *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterPoolCapSet is a free log retrieval operation binding the contract event 0x73c0350fb1b9cc92d0694a2b3bff8060a5431bfa8f95545c862d2b5c2b4c667c.
//
// Solidity: event PoolCapSet(address indexed sender, uint256 oldPoolCap, uint256 newPoolCap)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterPoolCapSet(opts *bind.FilterOpts, sender []common.Address) (*NablaSwapPoolPoolCapSetIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "PoolCapSet", senderRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolPoolCapSetIterator{contract: _NablaSwapPool.contract, event: "PoolCapSet", logs: logs, sub: sub}, nil
}

// WatchPoolCapSet is a free log subscription operation binding the contract event 0x73c0350fb1b9cc92d0694a2b3bff8060a5431bfa8f95545c862d2b5c2b4c667c.
//
// Solidity: event PoolCapSet(address indexed sender, uint256 oldPoolCap, uint256 newPoolCap)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchPoolCapSet(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolPoolCapSet, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "PoolCapSet", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolPoolCapSet)
				if err := _NablaSwapPool.contract.UnpackLog(event, "PoolCapSet", log); err != nil {
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

// ParsePoolCapSet is a log parse operation binding the contract event 0x73c0350fb1b9cc92d0694a2b3bff8060a5431bfa8f95545c862d2b5c2b4c667c.
//
// Solidity: event PoolCapSet(address indexed sender, uint256 oldPoolCap, uint256 newPoolCap)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParsePoolCapSet(log types.Log) (*NablaSwapPoolPoolCapSet, error) {
	event := new(NablaSwapPoolPoolCapSet)
	if err := _NablaSwapPool.contract.UnpackLog(event, "PoolCapSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolProtocolTreasuryChangedIterator is returned from FilterProtocolTreasuryChanged and is used to iterate over the raw logs and unpacked data for ProtocolTreasuryChanged events raised by the NablaSwapPool contract.
type NablaSwapPoolProtocolTreasuryChangedIterator struct {
	Event *NablaSwapPoolProtocolTreasuryChanged // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolProtocolTreasuryChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolProtocolTreasuryChanged)
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
		it.Event = new(NablaSwapPoolProtocolTreasuryChanged)
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
func (it *NablaSwapPoolProtocolTreasuryChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolProtocolTreasuryChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolProtocolTreasuryChanged represents a ProtocolTreasuryChanged event raised by the NablaSwapPool contract.
type NablaSwapPoolProtocolTreasuryChanged struct {
	Sender              common.Address
	OldProtocolTreasury common.Address
	NewProtocolTreasury common.Address
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterProtocolTreasuryChanged is a free log retrieval operation binding the contract event 0x9873a3eebbe9d89a604258c3566d39e4b79ace9be320679ec52c8f79e103fe1a.
//
// Solidity: event ProtocolTreasuryChanged(address indexed sender, address oldProtocolTreasury, address newProtocolTreasury)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterProtocolTreasuryChanged(opts *bind.FilterOpts, sender []common.Address) (*NablaSwapPoolProtocolTreasuryChangedIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "ProtocolTreasuryChanged", senderRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolProtocolTreasuryChangedIterator{contract: _NablaSwapPool.contract, event: "ProtocolTreasuryChanged", logs: logs, sub: sub}, nil
}

// WatchProtocolTreasuryChanged is a free log subscription operation binding the contract event 0x9873a3eebbe9d89a604258c3566d39e4b79ace9be320679ec52c8f79e103fe1a.
//
// Solidity: event ProtocolTreasuryChanged(address indexed sender, address oldProtocolTreasury, address newProtocolTreasury)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchProtocolTreasuryChanged(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolProtocolTreasuryChanged, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "ProtocolTreasuryChanged", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolProtocolTreasuryChanged)
				if err := _NablaSwapPool.contract.UnpackLog(event, "ProtocolTreasuryChanged", log); err != nil {
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

// ParseProtocolTreasuryChanged is a log parse operation binding the contract event 0x9873a3eebbe9d89a604258c3566d39e4b79ace9be320679ec52c8f79e103fe1a.
//
// Solidity: event ProtocolTreasuryChanged(address indexed sender, address oldProtocolTreasury, address newProtocolTreasury)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseProtocolTreasuryChanged(log types.Log) (*NablaSwapPoolProtocolTreasuryChanged, error) {
	event := new(NablaSwapPoolProtocolTreasuryChanged)
	if err := _NablaSwapPool.contract.UnpackLog(event, "ProtocolTreasuryChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolReserveUpdatedIterator is returned from FilterReserveUpdated and is used to iterate over the raw logs and unpacked data for ReserveUpdated events raised by the NablaSwapPool contract.
type NablaSwapPoolReserveUpdatedIterator struct {
	Event *NablaSwapPoolReserveUpdated // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolReserveUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolReserveUpdated)
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
		it.Event = new(NablaSwapPoolReserveUpdated)
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
func (it *NablaSwapPoolReserveUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolReserveUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolReserveUpdated represents a ReserveUpdated event raised by the NablaSwapPool contract.
type NablaSwapPoolReserveUpdated struct {
	NewReserve             *big.Int
	NewReserveWithSlippage *big.Int
	NewTotalLiabilities    *big.Int
	Raw                    types.Log // Blockchain specific contextual infos
}

// FilterReserveUpdated is a free log retrieval operation binding the contract event 0x736a4a5812ced57865d349f18ffc358079c6b479326c0dfd1dae30c465b1daf2.
//
// Solidity: event ReserveUpdated(uint256 newReserve, uint256 newReserveWithSlippage, uint256 newTotalLiabilities)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterReserveUpdated(opts *bind.FilterOpts) (*NablaSwapPoolReserveUpdatedIterator, error) {

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "ReserveUpdated")
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolReserveUpdatedIterator{contract: _NablaSwapPool.contract, event: "ReserveUpdated", logs: logs, sub: sub}, nil
}

// WatchReserveUpdated is a free log subscription operation binding the contract event 0x736a4a5812ced57865d349f18ffc358079c6b479326c0dfd1dae30c465b1daf2.
//
// Solidity: event ReserveUpdated(uint256 newReserve, uint256 newReserveWithSlippage, uint256 newTotalLiabilities)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchReserveUpdated(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolReserveUpdated) (event.Subscription, error) {

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "ReserveUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolReserveUpdated)
				if err := _NablaSwapPool.contract.UnpackLog(event, "ReserveUpdated", log); err != nil {
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

// ParseReserveUpdated is a log parse operation binding the contract event 0x736a4a5812ced57865d349f18ffc358079c6b479326c0dfd1dae30c465b1daf2.
//
// Solidity: event ReserveUpdated(uint256 newReserve, uint256 newReserveWithSlippage, uint256 newTotalLiabilities)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseReserveUpdated(log types.Log) (*NablaSwapPoolReserveUpdated, error) {
	event := new(NablaSwapPoolReserveUpdated)
	if err := _NablaSwapPool.contract.UnpackLog(event, "ReserveUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolSafeRedeemPercentageAndIntervalSetIterator is returned from FilterSafeRedeemPercentageAndIntervalSet and is used to iterate over the raw logs and unpacked data for SafeRedeemPercentageAndIntervalSet events raised by the NablaSwapPool contract.
type NablaSwapPoolSafeRedeemPercentageAndIntervalSetIterator struct {
	Event *NablaSwapPoolSafeRedeemPercentageAndIntervalSet // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolSafeRedeemPercentageAndIntervalSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolSafeRedeemPercentageAndIntervalSet)
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
		it.Event = new(NablaSwapPoolSafeRedeemPercentageAndIntervalSet)
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
func (it *NablaSwapPoolSafeRedeemPercentageAndIntervalSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolSafeRedeemPercentageAndIntervalSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolSafeRedeemPercentageAndIntervalSet represents a SafeRedeemPercentageAndIntervalSet event raised by the NablaSwapPool contract.
type NablaSwapPoolSafeRedeemPercentageAndIntervalSet struct {
	Owner                common.Address
	SafeRedeemPercentage *big.Int
	SafeRedeemInterval   *big.Int
	Raw                  types.Log // Blockchain specific contextual infos
}

// FilterSafeRedeemPercentageAndIntervalSet is a free log retrieval operation binding the contract event 0x250134bd3375cec695a3a64cd4dd192f58e1107f6a8187b158c50d13c8b9b06b.
//
// Solidity: event SafeRedeemPercentageAndIntervalSet(address indexed owner, uint256 safeRedeemPercentage, uint256 safeRedeemInterval)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterSafeRedeemPercentageAndIntervalSet(opts *bind.FilterOpts, owner []common.Address) (*NablaSwapPoolSafeRedeemPercentageAndIntervalSetIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "SafeRedeemPercentageAndIntervalSet", ownerRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolSafeRedeemPercentageAndIntervalSetIterator{contract: _NablaSwapPool.contract, event: "SafeRedeemPercentageAndIntervalSet", logs: logs, sub: sub}, nil
}

// WatchSafeRedeemPercentageAndIntervalSet is a free log subscription operation binding the contract event 0x250134bd3375cec695a3a64cd4dd192f58e1107f6a8187b158c50d13c8b9b06b.
//
// Solidity: event SafeRedeemPercentageAndIntervalSet(address indexed owner, uint256 safeRedeemPercentage, uint256 safeRedeemInterval)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchSafeRedeemPercentageAndIntervalSet(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolSafeRedeemPercentageAndIntervalSet, owner []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "SafeRedeemPercentageAndIntervalSet", ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolSafeRedeemPercentageAndIntervalSet)
				if err := _NablaSwapPool.contract.UnpackLog(event, "SafeRedeemPercentageAndIntervalSet", log); err != nil {
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

// ParseSafeRedeemPercentageAndIntervalSet is a log parse operation binding the contract event 0x250134bd3375cec695a3a64cd4dd192f58e1107f6a8187b158c50d13c8b9b06b.
//
// Solidity: event SafeRedeemPercentageAndIntervalSet(address indexed owner, uint256 safeRedeemPercentage, uint256 safeRedeemInterval)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseSafeRedeemPercentageAndIntervalSet(log types.Log) (*NablaSwapPoolSafeRedeemPercentageAndIntervalSet, error) {
	event := new(NablaSwapPoolSafeRedeemPercentageAndIntervalSet)
	if err := _NablaSwapPool.contract.UnpackLog(event, "SafeRedeemPercentageAndIntervalSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolSwapFeesSetIterator is returned from FilterSwapFeesSet and is used to iterate over the raw logs and unpacked data for SwapFeesSet events raised by the NablaSwapPool contract.
type NablaSwapPoolSwapFeesSetIterator struct {
	Event *NablaSwapPoolSwapFeesSet // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolSwapFeesSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolSwapFeesSet)
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
		it.Event = new(NablaSwapPoolSwapFeesSet)
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
func (it *NablaSwapPoolSwapFeesSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolSwapFeesSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolSwapFeesSet represents a SwapFeesSet event raised by the NablaSwapPool contract.
type NablaSwapPoolSwapFeesSet struct {
	Sender      common.Address
	LpFee       *big.Int
	BackstopFee *big.Int
	ProtocolFee *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterSwapFeesSet is a free log retrieval operation binding the contract event 0xd51891e6ac27da6065760e4843c63beb01795531a5c017b29f959a4c1055c498.
//
// Solidity: event SwapFeesSet(address indexed sender, uint256 lpFee, uint256 backstopFee, uint256 protocolFee)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterSwapFeesSet(opts *bind.FilterOpts, sender []common.Address) (*NablaSwapPoolSwapFeesSetIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "SwapFeesSet", senderRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolSwapFeesSetIterator{contract: _NablaSwapPool.contract, event: "SwapFeesSet", logs: logs, sub: sub}, nil
}

// WatchSwapFeesSet is a free log subscription operation binding the contract event 0xd51891e6ac27da6065760e4843c63beb01795531a5c017b29f959a4c1055c498.
//
// Solidity: event SwapFeesSet(address indexed sender, uint256 lpFee, uint256 backstopFee, uint256 protocolFee)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchSwapFeesSet(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolSwapFeesSet, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "SwapFeesSet", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolSwapFeesSet)
				if err := _NablaSwapPool.contract.UnpackLog(event, "SwapFeesSet", log); err != nil {
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

// ParseSwapFeesSet is a log parse operation binding the contract event 0xd51891e6ac27da6065760e4843c63beb01795531a5c017b29f959a4c1055c498.
//
// Solidity: event SwapFeesSet(address indexed sender, uint256 lpFee, uint256 backstopFee, uint256 protocolFee)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseSwapFeesSet(log types.Log) (*NablaSwapPoolSwapFeesSet, error) {
	event := new(NablaSwapPoolSwapFeesSet)
	if err := _NablaSwapPool.contract.UnpackLog(event, "SwapFeesSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the NablaSwapPool contract.
type NablaSwapPoolTransferIterator struct {
	Event *NablaSwapPoolTransfer // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolTransfer)
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
		it.Event = new(NablaSwapPoolTransfer)
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
func (it *NablaSwapPoolTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolTransfer represents a Transfer event raised by the NablaSwapPool contract.
type NablaSwapPoolTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*NablaSwapPoolTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolTransferIterator{contract: _NablaSwapPool.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolTransfer)
				if err := _NablaSwapPool.contract.UnpackLog(event, "Transfer", log); err != nil {
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
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseTransfer(log types.Log) (*NablaSwapPoolTransfer, error) {
	event := new(NablaSwapPoolTransfer)
	if err := _NablaSwapPool.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the NablaSwapPool contract.
type NablaSwapPoolUnpausedIterator struct {
	Event *NablaSwapPoolUnpaused // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolUnpaused)
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
		it.Event = new(NablaSwapPoolUnpaused)
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
func (it *NablaSwapPoolUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolUnpaused represents a Unpaused event raised by the NablaSwapPool contract.
type NablaSwapPoolUnpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterUnpaused(opts *bind.FilterOpts) (*NablaSwapPoolUnpausedIterator, error) {

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolUnpausedIterator{contract: _NablaSwapPool.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolUnpaused) (event.Subscription, error) {

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolUnpaused)
				if err := _NablaSwapPool.contract.UnpackLog(event, "Unpaused", log); err != nil {
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

// ParseUnpaused is a log parse operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseUnpaused(log types.Log) (*NablaSwapPoolUnpaused, error) {
	event := new(NablaSwapPoolUnpaused)
	if err := _NablaSwapPool.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NablaSwapPoolWithdrawalIterator is returned from FilterWithdrawal and is used to iterate over the raw logs and unpacked data for Withdrawal events raised by the NablaSwapPool contract.
type NablaSwapPoolWithdrawalIterator struct {
	Event *NablaSwapPoolWithdrawal // Event containing the contract specifics and raw log

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
func (it *NablaSwapPoolWithdrawalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NablaSwapPoolWithdrawal)
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
		it.Event = new(NablaSwapPoolWithdrawal)
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
func (it *NablaSwapPoolWithdrawalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NablaSwapPoolWithdrawalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NablaSwapPoolWithdrawal represents a Withdrawal event raised by the NablaSwapPool contract.
type NablaSwapPoolWithdrawal struct {
	Sender           common.Address
	PoolSharesBurned *big.Int
	Fee              *big.Int
	AmountWithdrawn  *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterWithdrawal is a free log retrieval operation binding the contract event 0xdd79c4beb640a28168d87b12ccb05db3bc7a269273eb8274e16073027cc2aa58.
//
// Solidity: event Withdrawal(address indexed sender, uint256 poolSharesBurned, int256 fee, uint256 amountWithdrawn)
func (_NablaSwapPool *NablaSwapPoolFilterer) FilterWithdrawal(opts *bind.FilterOpts, sender []common.Address) (*NablaSwapPoolWithdrawalIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.FilterLogs(opts, "Withdrawal", senderRule)
	if err != nil {
		return nil, err
	}
	return &NablaSwapPoolWithdrawalIterator{contract: _NablaSwapPool.contract, event: "Withdrawal", logs: logs, sub: sub}, nil
}

// WatchWithdrawal is a free log subscription operation binding the contract event 0xdd79c4beb640a28168d87b12ccb05db3bc7a269273eb8274e16073027cc2aa58.
//
// Solidity: event Withdrawal(address indexed sender, uint256 poolSharesBurned, int256 fee, uint256 amountWithdrawn)
func (_NablaSwapPool *NablaSwapPoolFilterer) WatchWithdrawal(opts *bind.WatchOpts, sink chan<- *NablaSwapPoolWithdrawal, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NablaSwapPool.contract.WatchLogs(opts, "Withdrawal", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NablaSwapPoolWithdrawal)
				if err := _NablaSwapPool.contract.UnpackLog(event, "Withdrawal", log); err != nil {
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

// ParseWithdrawal is a log parse operation binding the contract event 0xdd79c4beb640a28168d87b12ccb05db3bc7a269273eb8274e16073027cc2aa58.
//
// Solidity: event Withdrawal(address indexed sender, uint256 poolSharesBurned, int256 fee, uint256 amountWithdrawn)
func (_NablaSwapPool *NablaSwapPoolFilterer) ParseWithdrawal(log types.Log) (*NablaSwapPoolWithdrawal, error) {
	event := new(NablaSwapPoolWithdrawal)
	if err := _NablaSwapPool.contract.UnpackLog(event, "Withdrawal", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
