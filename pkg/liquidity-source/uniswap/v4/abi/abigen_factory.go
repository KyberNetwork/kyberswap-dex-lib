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

// IPoolManagerModifyLiquidityParams is an auto generated low-level Go binding around an user-defined struct.
type IPoolManagerModifyLiquidityParams struct {
	TickLower      *big.Int
	TickUpper      *big.Int
	LiquidityDelta *big.Int
	Salt           [32]byte
}

// IPoolManagerSwapParams is an auto generated low-level Go binding around an user-defined struct.
type IPoolManagerSwapParams struct {
	ZeroForOne        bool
	AmountSpecified   *big.Int
	SqrtPriceLimitX96 *big.Int
}

// PoolKey is an auto generated low-level Go binding around an user-defined struct.
type PoolKey struct {
	Currency0   common.Address
	Currency1   common.Address
	Fee         *big.Int
	TickSpacing *big.Int
	Hooks       common.Address
}

// UniswapV4PoolManagerMetaData contains all meta data concerning the UniswapV4PoolManager contract.
var UniswapV4PoolManagerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"initialOwner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AlreadyUnlocked\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"currency1\",\"type\":\"address\"}],\"name\":\"CurrenciesOutOfOrderOrEqual\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"CurrencyNotSettled\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DelegateCallNotAllowed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidCaller\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ManagerLocked\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MustClearExactPositiveDelta\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NonzeroNativeValue\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PoolNotInitialized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ProtocolFeeCurrencySynced\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"}],\"name\":\"ProtocolFeeTooLarge\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"SwapAmountCannotBeZero\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"}],\"name\":\"TickSpacingTooLarge\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"}],\"name\":\"TickSpacingTooSmall\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"UnauthorizedDynamicLPFeeUpdate\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"name\":\"Donate\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"Initialize\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"liquidityDelta\",\"type\":\"int256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"}],\"name\":\"ModifyLiquidity\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"OperatorSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"protocolFeeController\",\"type\":\"address\"}],\"name\":\"ProtocolFeeControllerUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"protocolFee\",\"type\":\"uint24\"}],\"name\":\"ProtocolFeeUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"int128\",\"name\":\"amount0\",\"type\":\"int128\"},{\"indexed\":false,\"internalType\":\"int128\",\"name\":\"amount1\",\"type\":\"int128\"},{\"indexed\":false,\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"liquidity\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"}],\"name\":\"Swap\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"caller\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"balance\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Currency\",\"name\":\"currency\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"clear\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"collectProtocolFees\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountCollected\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"donate\",\"outputs\":[{\"internalType\":\"BalanceDelta\",\"name\":\"delta\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"slot\",\"type\":\"bytes32\"}],\"name\":\"extsload\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"startSlot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"nSlots\",\"type\":\"uint256\"}],\"name\":\"extsload\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"slots\",\"type\":\"bytes32[]\"}],\"name\":\"extsload\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"slots\",\"type\":\"bytes32[]\"}],\"name\":\"exttload\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"slot\",\"type\":\"bytes32\"}],\"name\":\"exttload\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"}],\"name\":\"initialize\",\"outputs\":[{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"}],\"name\":\"isOperator\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"isOperator\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"mint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"},{\"internalType\":\"int256\",\"name\":\"liquidityDelta\",\"type\":\"int256\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"}],\"internalType\":\"structIPoolManager.ModifyLiquidityParams\",\"name\":\"params\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"modifyLiquidity\",\"outputs\":[{\"internalType\":\"BalanceDelta\",\"name\":\"callerDelta\",\"type\":\"int256\"},{\"internalType\":\"BalanceDelta\",\"name\":\"feesAccrued\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"protocolFeeController\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Currency\",\"name\":\"currency\",\"type\":\"address\"}],\"name\":\"protocolFeesAccrued\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"setOperator\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint24\",\"name\":\"newProtocolFee\",\"type\":\"uint24\"}],\"name\":\"setProtocolFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"controller\",\"type\":\"address\"}],\"name\":\"setProtocolFeeController\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"settle\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"}],\"name\":\"settleFor\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountSpecified\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceLimitX96\",\"type\":\"uint160\"}],\"internalType\":\"structIPoolManager.SwapParams\",\"name\":\"params\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"swap\",\"outputs\":[{\"internalType\":\"BalanceDelta\",\"name\":\"swapDelta\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Currency\",\"name\":\"currency\",\"type\":\"address\"}],\"name\":\"sync\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Currency\",\"name\":\"currency\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"take\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"receiver\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"receiver\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"unlock\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"result\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint24\",\"name\":\"newDynamicLPFee\",\"type\":\"uint24\"}],\"name\":\"updateDynamicLPFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// UniswapV4PoolManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use UniswapV4PoolManagerMetaData.ABI instead.
var UniswapV4PoolManagerABI = UniswapV4PoolManagerMetaData.ABI

// UniswapV4PoolManager is an auto generated Go binding around an Ethereum contract.
type UniswapV4PoolManager struct {
	UniswapV4PoolManagerCaller     // Read-only binding to the contract
	UniswapV4PoolManagerTransactor // Write-only binding to the contract
	UniswapV4PoolManagerFilterer   // Log filterer for contract events
}

// UniswapV4PoolManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type UniswapV4PoolManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// UniswapV4PoolManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type UniswapV4PoolManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// UniswapV4PoolManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type UniswapV4PoolManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// UniswapV4PoolManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type UniswapV4PoolManagerSession struct {
	Contract     *UniswapV4PoolManager // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// UniswapV4PoolManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type UniswapV4PoolManagerCallerSession struct {
	Contract *UniswapV4PoolManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// UniswapV4PoolManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type UniswapV4PoolManagerTransactorSession struct {
	Contract     *UniswapV4PoolManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// UniswapV4PoolManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type UniswapV4PoolManagerRaw struct {
	Contract *UniswapV4PoolManager // Generic contract binding to access the raw methods on
}

// UniswapV4PoolManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type UniswapV4PoolManagerCallerRaw struct {
	Contract *UniswapV4PoolManagerCaller // Generic read-only contract binding to access the raw methods on
}

// UniswapV4PoolManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type UniswapV4PoolManagerTransactorRaw struct {
	Contract *UniswapV4PoolManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewUniswapV4PoolManager creates a new instance of UniswapV4PoolManager, bound to a specific deployed contract.
func NewUniswapV4PoolManager(address common.Address, backend bind.ContractBackend) (*UniswapV4PoolManager, error) {
	contract, err := bindUniswapV4PoolManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &UniswapV4PoolManager{UniswapV4PoolManagerCaller: UniswapV4PoolManagerCaller{contract: contract}, UniswapV4PoolManagerTransactor: UniswapV4PoolManagerTransactor{contract: contract}, UniswapV4PoolManagerFilterer: UniswapV4PoolManagerFilterer{contract: contract}}, nil
}

// NewUniswapV4PoolManagerCaller creates a new read-only instance of UniswapV4PoolManager, bound to a specific deployed contract.
func NewUniswapV4PoolManagerCaller(address common.Address, caller bind.ContractCaller) (*UniswapV4PoolManagerCaller, error) {
	contract, err := bindUniswapV4PoolManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &UniswapV4PoolManagerCaller{contract: contract}, nil
}

// NewUniswapV4PoolManagerTransactor creates a new write-only instance of UniswapV4PoolManager, bound to a specific deployed contract.
func NewUniswapV4PoolManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*UniswapV4PoolManagerTransactor, error) {
	contract, err := bindUniswapV4PoolManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &UniswapV4PoolManagerTransactor{contract: contract}, nil
}

// NewUniswapV4PoolManagerFilterer creates a new log filterer instance of UniswapV4PoolManager, bound to a specific deployed contract.
func NewUniswapV4PoolManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*UniswapV4PoolManagerFilterer, error) {
	contract, err := bindUniswapV4PoolManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &UniswapV4PoolManagerFilterer{contract: contract}, nil
}

// bindUniswapV4PoolManager binds a generic wrapper to an already deployed contract.
func bindUniswapV4PoolManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := UniswapV4PoolManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_UniswapV4PoolManager *UniswapV4PoolManagerRaw) Call(opts *bind.CallOpts, result *[]any, method string, params ...any) error {
	return _UniswapV4PoolManager.Contract.UniswapV4PoolManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_UniswapV4PoolManager *UniswapV4PoolManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.UniswapV4PoolManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_UniswapV4PoolManager *UniswapV4PoolManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...any) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.UniswapV4PoolManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_UniswapV4PoolManager *UniswapV4PoolManagerCallerRaw) Call(opts *bind.CallOpts, result *[]any, method string, params ...any) error {
	return _UniswapV4PoolManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...any) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0x598af9e7.
//
// Solidity: function allowance(address owner, address spender, uint256 id) view returns(uint256 amount)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCaller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address, id *big.Int) (*big.Int, error) {
	var out []any
	err := _UniswapV4PoolManager.contract.Call(opts, &out, "allowance", owner, spender, id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0x598af9e7.
//
// Solidity: function allowance(address owner, address spender, uint256 id) view returns(uint256 amount)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Allowance(owner common.Address, spender common.Address, id *big.Int) (*big.Int, error) {
	return _UniswapV4PoolManager.Contract.Allowance(&_UniswapV4PoolManager.CallOpts, owner, spender, id)
}

// Allowance is a free data retrieval call binding the contract method 0x598af9e7.
//
// Solidity: function allowance(address owner, address spender, uint256 id) view returns(uint256 amount)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCallerSession) Allowance(owner common.Address, spender common.Address, id *big.Int) (*big.Int, error) {
	return _UniswapV4PoolManager.Contract.Allowance(&_UniswapV4PoolManager.CallOpts, owner, spender, id)
}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address owner, uint256 id) view returns(uint256 balance)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCaller) BalanceOf(opts *bind.CallOpts, owner common.Address, id *big.Int) (*big.Int, error) {
	var out []any
	err := _UniswapV4PoolManager.contract.Call(opts, &out, "balanceOf", owner, id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address owner, uint256 id) view returns(uint256 balance)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) BalanceOf(owner common.Address, id *big.Int) (*big.Int, error) {
	return _UniswapV4PoolManager.Contract.BalanceOf(&_UniswapV4PoolManager.CallOpts, owner, id)
}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address owner, uint256 id) view returns(uint256 balance)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCallerSession) BalanceOf(owner common.Address, id *big.Int) (*big.Int, error) {
	return _UniswapV4PoolManager.Contract.BalanceOf(&_UniswapV4PoolManager.CallOpts, owner, id)
}

// Extsload is a free data retrieval call binding the contract method 0x1e2eaeaf.
//
// Solidity: function extsload(bytes32 slot) view returns(bytes32)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCaller) Extsload(opts *bind.CallOpts, slot [32]byte) ([32]byte, error) {
	var out []any
	err := _UniswapV4PoolManager.contract.Call(opts, &out, "extsload", slot)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Extsload is a free data retrieval call binding the contract method 0x1e2eaeaf.
//
// Solidity: function extsload(bytes32 slot) view returns(bytes32)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Extsload(slot [32]byte) ([32]byte, error) {
	return _UniswapV4PoolManager.Contract.Extsload(&_UniswapV4PoolManager.CallOpts, slot)
}

// Extsload is a free data retrieval call binding the contract method 0x1e2eaeaf.
//
// Solidity: function extsload(bytes32 slot) view returns(bytes32)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCallerSession) Extsload(slot [32]byte) ([32]byte, error) {
	return _UniswapV4PoolManager.Contract.Extsload(&_UniswapV4PoolManager.CallOpts, slot)
}

// Extsload0 is a free data retrieval call binding the contract method 0x35fd631a.
//
// Solidity: function extsload(bytes32 startSlot, uint256 nSlots) view returns(bytes32[])
func (_UniswapV4PoolManager *UniswapV4PoolManagerCaller) Extsload0(opts *bind.CallOpts, startSlot [32]byte, nSlots *big.Int) ([][32]byte, error) {
	var out []any
	err := _UniswapV4PoolManager.contract.Call(opts, &out, "extsload0", startSlot, nSlots)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// Extsload0 is a free data retrieval call binding the contract method 0x35fd631a.
//
// Solidity: function extsload(bytes32 startSlot, uint256 nSlots) view returns(bytes32[])
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Extsload0(startSlot [32]byte, nSlots *big.Int) ([][32]byte, error) {
	return _UniswapV4PoolManager.Contract.Extsload0(&_UniswapV4PoolManager.CallOpts, startSlot, nSlots)
}

// Extsload0 is a free data retrieval call binding the contract method 0x35fd631a.
//
// Solidity: function extsload(bytes32 startSlot, uint256 nSlots) view returns(bytes32[])
func (_UniswapV4PoolManager *UniswapV4PoolManagerCallerSession) Extsload0(startSlot [32]byte, nSlots *big.Int) ([][32]byte, error) {
	return _UniswapV4PoolManager.Contract.Extsload0(&_UniswapV4PoolManager.CallOpts, startSlot, nSlots)
}

// Extsload1 is a free data retrieval call binding the contract method 0xdbd035ff.
//
// Solidity: function extsload(bytes32[] slots) view returns(bytes32[])
func (_UniswapV4PoolManager *UniswapV4PoolManagerCaller) Extsload1(opts *bind.CallOpts, slots [][32]byte) ([][32]byte, error) {
	var out []any
	err := _UniswapV4PoolManager.contract.Call(opts, &out, "extsload1", slots)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// Extsload1 is a free data retrieval call binding the contract method 0xdbd035ff.
//
// Solidity: function extsload(bytes32[] slots) view returns(bytes32[])
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Extsload1(slots [][32]byte) ([][32]byte, error) {
	return _UniswapV4PoolManager.Contract.Extsload1(&_UniswapV4PoolManager.CallOpts, slots)
}

// Extsload1 is a free data retrieval call binding the contract method 0xdbd035ff.
//
// Solidity: function extsload(bytes32[] slots) view returns(bytes32[])
func (_UniswapV4PoolManager *UniswapV4PoolManagerCallerSession) Extsload1(slots [][32]byte) ([][32]byte, error) {
	return _UniswapV4PoolManager.Contract.Extsload1(&_UniswapV4PoolManager.CallOpts, slots)
}

// Exttload is a free data retrieval call binding the contract method 0x9bf6645f.
//
// Solidity: function exttload(bytes32[] slots) view returns(bytes32[])
func (_UniswapV4PoolManager *UniswapV4PoolManagerCaller) Exttload(opts *bind.CallOpts, slots [][32]byte) ([][32]byte, error) {
	var out []any
	err := _UniswapV4PoolManager.contract.Call(opts, &out, "exttload", slots)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// Exttload is a free data retrieval call binding the contract method 0x9bf6645f.
//
// Solidity: function exttload(bytes32[] slots) view returns(bytes32[])
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Exttload(slots [][32]byte) ([][32]byte, error) {
	return _UniswapV4PoolManager.Contract.Exttload(&_UniswapV4PoolManager.CallOpts, slots)
}

// Exttload is a free data retrieval call binding the contract method 0x9bf6645f.
//
// Solidity: function exttload(bytes32[] slots) view returns(bytes32[])
func (_UniswapV4PoolManager *UniswapV4PoolManagerCallerSession) Exttload(slots [][32]byte) ([][32]byte, error) {
	return _UniswapV4PoolManager.Contract.Exttload(&_UniswapV4PoolManager.CallOpts, slots)
}

// Exttload0 is a free data retrieval call binding the contract method 0xf135baaa.
//
// Solidity: function exttload(bytes32 slot) view returns(bytes32)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCaller) Exttload0(opts *bind.CallOpts, slot [32]byte) ([32]byte, error) {
	var out []any
	err := _UniswapV4PoolManager.contract.Call(opts, &out, "exttload0", slot)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Exttload0 is a free data retrieval call binding the contract method 0xf135baaa.
//
// Solidity: function exttload(bytes32 slot) view returns(bytes32)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Exttload0(slot [32]byte) ([32]byte, error) {
	return _UniswapV4PoolManager.Contract.Exttload0(&_UniswapV4PoolManager.CallOpts, slot)
}

// Exttload0 is a free data retrieval call binding the contract method 0xf135baaa.
//
// Solidity: function exttload(bytes32 slot) view returns(bytes32)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCallerSession) Exttload0(slot [32]byte) ([32]byte, error) {
	return _UniswapV4PoolManager.Contract.Exttload0(&_UniswapV4PoolManager.CallOpts, slot)
}

// IsOperator is a free data retrieval call binding the contract method 0xb6363cf2.
//
// Solidity: function isOperator(address owner, address operator) view returns(bool isOperator)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCaller) IsOperator(opts *bind.CallOpts, owner common.Address, operator common.Address) (bool, error) {
	var out []any
	err := _UniswapV4PoolManager.contract.Call(opts, &out, "isOperator", owner, operator)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOperator is a free data retrieval call binding the contract method 0xb6363cf2.
//
// Solidity: function isOperator(address owner, address operator) view returns(bool isOperator)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) IsOperator(owner common.Address, operator common.Address) (bool, error) {
	return _UniswapV4PoolManager.Contract.IsOperator(&_UniswapV4PoolManager.CallOpts, owner, operator)
}

// IsOperator is a free data retrieval call binding the contract method 0xb6363cf2.
//
// Solidity: function isOperator(address owner, address operator) view returns(bool isOperator)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCallerSession) IsOperator(owner common.Address, operator common.Address) (bool, error) {
	return _UniswapV4PoolManager.Contract.IsOperator(&_UniswapV4PoolManager.CallOpts, owner, operator)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _UniswapV4PoolManager.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Owner() (common.Address, error) {
	return _UniswapV4PoolManager.Contract.Owner(&_UniswapV4PoolManager.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCallerSession) Owner() (common.Address, error) {
	return _UniswapV4PoolManager.Contract.Owner(&_UniswapV4PoolManager.CallOpts)
}

// ProtocolFeeController is a free data retrieval call binding the contract method 0xf02de3b2.
//
// Solidity: function protocolFeeController() view returns(address)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCaller) ProtocolFeeController(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _UniswapV4PoolManager.contract.Call(opts, &out, "protocolFeeController")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ProtocolFeeController is a free data retrieval call binding the contract method 0xf02de3b2.
//
// Solidity: function protocolFeeController() view returns(address)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) ProtocolFeeController() (common.Address, error) {
	return _UniswapV4PoolManager.Contract.ProtocolFeeController(&_UniswapV4PoolManager.CallOpts)
}

// ProtocolFeeController is a free data retrieval call binding the contract method 0xf02de3b2.
//
// Solidity: function protocolFeeController() view returns(address)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCallerSession) ProtocolFeeController() (common.Address, error) {
	return _UniswapV4PoolManager.Contract.ProtocolFeeController(&_UniswapV4PoolManager.CallOpts)
}

// ProtocolFeesAccrued is a free data retrieval call binding the contract method 0x97e8cd4e.
//
// Solidity: function protocolFeesAccrued(address currency) view returns(uint256 amount)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCaller) ProtocolFeesAccrued(opts *bind.CallOpts, currency common.Address) (*big.Int, error) {
	var out []any
	err := _UniswapV4PoolManager.contract.Call(opts, &out, "protocolFeesAccrued", currency)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ProtocolFeesAccrued is a free data retrieval call binding the contract method 0x97e8cd4e.
//
// Solidity: function protocolFeesAccrued(address currency) view returns(uint256 amount)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) ProtocolFeesAccrued(currency common.Address) (*big.Int, error) {
	return _UniswapV4PoolManager.Contract.ProtocolFeesAccrued(&_UniswapV4PoolManager.CallOpts, currency)
}

// ProtocolFeesAccrued is a free data retrieval call binding the contract method 0x97e8cd4e.
//
// Solidity: function protocolFeesAccrued(address currency) view returns(uint256 amount)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCallerSession) ProtocolFeesAccrued(currency common.Address) (*big.Int, error) {
	return _UniswapV4PoolManager.Contract.ProtocolFeesAccrued(&_UniswapV4PoolManager.CallOpts, currency)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []any
	err := _UniswapV4PoolManager.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _UniswapV4PoolManager.Contract.SupportsInterface(&_UniswapV4PoolManager.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_UniswapV4PoolManager *UniswapV4PoolManagerCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _UniswapV4PoolManager.Contract.SupportsInterface(&_UniswapV4PoolManager.CallOpts, interfaceId)
}

// Approve is a paid mutator transaction binding the contract method 0x426a8493.
//
// Solidity: function approve(address spender, uint256 id, uint256 amount) returns(bool)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) Approve(opts *bind.TransactOpts, spender common.Address, id *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "approve", spender, id, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x426a8493.
//
// Solidity: function approve(address spender, uint256 id, uint256 amount) returns(bool)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Approve(spender common.Address, id *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Approve(&_UniswapV4PoolManager.TransactOpts, spender, id, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x426a8493.
//
// Solidity: function approve(address spender, uint256 id, uint256 amount) returns(bool)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) Approve(spender common.Address, id *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Approve(&_UniswapV4PoolManager.TransactOpts, spender, id, amount)
}

// Burn is a paid mutator transaction binding the contract method 0xf5298aca.
//
// Solidity: function burn(address from, uint256 id, uint256 amount) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) Burn(opts *bind.TransactOpts, from common.Address, id *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "burn", from, id, amount)
}

// Burn is a paid mutator transaction binding the contract method 0xf5298aca.
//
// Solidity: function burn(address from, uint256 id, uint256 amount) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Burn(from common.Address, id *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Burn(&_UniswapV4PoolManager.TransactOpts, from, id, amount)
}

// Burn is a paid mutator transaction binding the contract method 0xf5298aca.
//
// Solidity: function burn(address from, uint256 id, uint256 amount) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) Burn(from common.Address, id *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Burn(&_UniswapV4PoolManager.TransactOpts, from, id, amount)
}

// Clear is a paid mutator transaction binding the contract method 0x80f0b44c.
//
// Solidity: function clear(address currency, uint256 amount) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) Clear(opts *bind.TransactOpts, currency common.Address, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "clear", currency, amount)
}

// Clear is a paid mutator transaction binding the contract method 0x80f0b44c.
//
// Solidity: function clear(address currency, uint256 amount) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Clear(currency common.Address, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Clear(&_UniswapV4PoolManager.TransactOpts, currency, amount)
}

// Clear is a paid mutator transaction binding the contract method 0x80f0b44c.
//
// Solidity: function clear(address currency, uint256 amount) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) Clear(currency common.Address, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Clear(&_UniswapV4PoolManager.TransactOpts, currency, amount)
}

// CollectProtocolFees is a paid mutator transaction binding the contract method 0x8161b874.
//
// Solidity: function collectProtocolFees(address recipient, address currency, uint256 amount) returns(uint256 amountCollected)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) CollectProtocolFees(opts *bind.TransactOpts, recipient common.Address, currency common.Address, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "collectProtocolFees", recipient, currency, amount)
}

// CollectProtocolFees is a paid mutator transaction binding the contract method 0x8161b874.
//
// Solidity: function collectProtocolFees(address recipient, address currency, uint256 amount) returns(uint256 amountCollected)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) CollectProtocolFees(recipient common.Address, currency common.Address, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.CollectProtocolFees(&_UniswapV4PoolManager.TransactOpts, recipient, currency, amount)
}

// CollectProtocolFees is a paid mutator transaction binding the contract method 0x8161b874.
//
// Solidity: function collectProtocolFees(address recipient, address currency, uint256 amount) returns(uint256 amountCollected)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) CollectProtocolFees(recipient common.Address, currency common.Address, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.CollectProtocolFees(&_UniswapV4PoolManager.TransactOpts, recipient, currency, amount)
}

// Donate is a paid mutator transaction binding the contract method 0x234266d7.
//
// Solidity: function donate((address,address,uint24,int24,address) key, uint256 amount0, uint256 amount1, bytes hookData) returns(int256 delta)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) Donate(opts *bind.TransactOpts, key PoolKey, amount0 *big.Int, amount1 *big.Int, hookData []byte) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "donate", key, amount0, amount1, hookData)
}

// Donate is a paid mutator transaction binding the contract method 0x234266d7.
//
// Solidity: function donate((address,address,uint24,int24,address) key, uint256 amount0, uint256 amount1, bytes hookData) returns(int256 delta)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Donate(key PoolKey, amount0 *big.Int, amount1 *big.Int, hookData []byte) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Donate(&_UniswapV4PoolManager.TransactOpts, key, amount0, amount1, hookData)
}

// Donate is a paid mutator transaction binding the contract method 0x234266d7.
//
// Solidity: function donate((address,address,uint24,int24,address) key, uint256 amount0, uint256 amount1, bytes hookData) returns(int256 delta)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) Donate(key PoolKey, amount0 *big.Int, amount1 *big.Int, hookData []byte) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Donate(&_UniswapV4PoolManager.TransactOpts, key, amount0, amount1, hookData)
}

// Initialize is a paid mutator transaction binding the contract method 0x6276cbbe.
//
// Solidity: function initialize((address,address,uint24,int24,address) key, uint160 sqrtPriceX96) returns(int24 tick)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) Initialize(opts *bind.TransactOpts, key PoolKey, sqrtPriceX96 *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "initialize", key, sqrtPriceX96)
}

// Initialize is a paid mutator transaction binding the contract method 0x6276cbbe.
//
// Solidity: function initialize((address,address,uint24,int24,address) key, uint160 sqrtPriceX96) returns(int24 tick)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Initialize(key PoolKey, sqrtPriceX96 *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Initialize(&_UniswapV4PoolManager.TransactOpts, key, sqrtPriceX96)
}

// Initialize is a paid mutator transaction binding the contract method 0x6276cbbe.
//
// Solidity: function initialize((address,address,uint24,int24,address) key, uint160 sqrtPriceX96) returns(int24 tick)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) Initialize(key PoolKey, sqrtPriceX96 *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Initialize(&_UniswapV4PoolManager.TransactOpts, key, sqrtPriceX96)
}

// Mint is a paid mutator transaction binding the contract method 0x156e29f6.
//
// Solidity: function mint(address to, uint256 id, uint256 amount) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) Mint(opts *bind.TransactOpts, to common.Address, id *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "mint", to, id, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x156e29f6.
//
// Solidity: function mint(address to, uint256 id, uint256 amount) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Mint(to common.Address, id *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Mint(&_UniswapV4PoolManager.TransactOpts, to, id, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x156e29f6.
//
// Solidity: function mint(address to, uint256 id, uint256 amount) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) Mint(to common.Address, id *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Mint(&_UniswapV4PoolManager.TransactOpts, to, id, amount)
}

// ModifyLiquidity is a paid mutator transaction binding the contract method 0x5a6bcfda.
//
// Solidity: function modifyLiquidity((address,address,uint24,int24,address) key, (int24,int24,int256,bytes32) params, bytes hookData) returns(int256 callerDelta, int256 feesAccrued)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) ModifyLiquidity(opts *bind.TransactOpts, key PoolKey, params IPoolManagerModifyLiquidityParams, hookData []byte) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "modifyLiquidity", key, params, hookData)
}

// ModifyLiquidity is a paid mutator transaction binding the contract method 0x5a6bcfda.
//
// Solidity: function modifyLiquidity((address,address,uint24,int24,address) key, (int24,int24,int256,bytes32) params, bytes hookData) returns(int256 callerDelta, int256 feesAccrued)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) ModifyLiquidity(key PoolKey, params IPoolManagerModifyLiquidityParams, hookData []byte) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.ModifyLiquidity(&_UniswapV4PoolManager.TransactOpts, key, params, hookData)
}

// ModifyLiquidity is a paid mutator transaction binding the contract method 0x5a6bcfda.
//
// Solidity: function modifyLiquidity((address,address,uint24,int24,address) key, (int24,int24,int256,bytes32) params, bytes hookData) returns(int256 callerDelta, int256 feesAccrued)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) ModifyLiquidity(key PoolKey, params IPoolManagerModifyLiquidityParams, hookData []byte) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.ModifyLiquidity(&_UniswapV4PoolManager.TransactOpts, key, params, hookData)
}

// SetOperator is a paid mutator transaction binding the contract method 0x558a7297.
//
// Solidity: function setOperator(address operator, bool approved) returns(bool)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) SetOperator(opts *bind.TransactOpts, operator common.Address, approved bool) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "setOperator", operator, approved)
}

// SetOperator is a paid mutator transaction binding the contract method 0x558a7297.
//
// Solidity: function setOperator(address operator, bool approved) returns(bool)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) SetOperator(operator common.Address, approved bool) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.SetOperator(&_UniswapV4PoolManager.TransactOpts, operator, approved)
}

// SetOperator is a paid mutator transaction binding the contract method 0x558a7297.
//
// Solidity: function setOperator(address operator, bool approved) returns(bool)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) SetOperator(operator common.Address, approved bool) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.SetOperator(&_UniswapV4PoolManager.TransactOpts, operator, approved)
}

// SetProtocolFee is a paid mutator transaction binding the contract method 0x7e87ce7d.
//
// Solidity: function setProtocolFee((address,address,uint24,int24,address) key, uint24 newProtocolFee) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) SetProtocolFee(opts *bind.TransactOpts, key PoolKey, newProtocolFee *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "setProtocolFee", key, newProtocolFee)
}

// SetProtocolFee is a paid mutator transaction binding the contract method 0x7e87ce7d.
//
// Solidity: function setProtocolFee((address,address,uint24,int24,address) key, uint24 newProtocolFee) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) SetProtocolFee(key PoolKey, newProtocolFee *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.SetProtocolFee(&_UniswapV4PoolManager.TransactOpts, key, newProtocolFee)
}

// SetProtocolFee is a paid mutator transaction binding the contract method 0x7e87ce7d.
//
// Solidity: function setProtocolFee((address,address,uint24,int24,address) key, uint24 newProtocolFee) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) SetProtocolFee(key PoolKey, newProtocolFee *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.SetProtocolFee(&_UniswapV4PoolManager.TransactOpts, key, newProtocolFee)
}

// SetProtocolFeeController is a paid mutator transaction binding the contract method 0x2d771389.
//
// Solidity: function setProtocolFeeController(address controller) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) SetProtocolFeeController(opts *bind.TransactOpts, controller common.Address) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "setProtocolFeeController", controller)
}

// SetProtocolFeeController is a paid mutator transaction binding the contract method 0x2d771389.
//
// Solidity: function setProtocolFeeController(address controller) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) SetProtocolFeeController(controller common.Address) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.SetProtocolFeeController(&_UniswapV4PoolManager.TransactOpts, controller)
}

// SetProtocolFeeController is a paid mutator transaction binding the contract method 0x2d771389.
//
// Solidity: function setProtocolFeeController(address controller) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) SetProtocolFeeController(controller common.Address) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.SetProtocolFeeController(&_UniswapV4PoolManager.TransactOpts, controller)
}

// Settle is a paid mutator transaction binding the contract method 0x11da60b4.
//
// Solidity: function settle() payable returns(uint256)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) Settle(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "settle")
}

// Settle is a paid mutator transaction binding the contract method 0x11da60b4.
//
// Solidity: function settle() payable returns(uint256)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Settle() (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Settle(&_UniswapV4PoolManager.TransactOpts)
}

// Settle is a paid mutator transaction binding the contract method 0x11da60b4.
//
// Solidity: function settle() payable returns(uint256)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) Settle() (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Settle(&_UniswapV4PoolManager.TransactOpts)
}

// SettleFor is a paid mutator transaction binding the contract method 0x3dd45adb.
//
// Solidity: function settleFor(address recipient) payable returns(uint256)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) SettleFor(opts *bind.TransactOpts, recipient common.Address) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "settleFor", recipient)
}

// SettleFor is a paid mutator transaction binding the contract method 0x3dd45adb.
//
// Solidity: function settleFor(address recipient) payable returns(uint256)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) SettleFor(recipient common.Address) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.SettleFor(&_UniswapV4PoolManager.TransactOpts, recipient)
}

// SettleFor is a paid mutator transaction binding the contract method 0x3dd45adb.
//
// Solidity: function settleFor(address recipient) payable returns(uint256)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) SettleFor(recipient common.Address) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.SettleFor(&_UniswapV4PoolManager.TransactOpts, recipient)
}

// Swap is a paid mutator transaction binding the contract method 0xf3cd914c.
//
// Solidity: function swap((address,address,uint24,int24,address) key, (bool,int256,uint160) params, bytes hookData) returns(int256 swapDelta)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) Swap(opts *bind.TransactOpts, key PoolKey, params IPoolManagerSwapParams, hookData []byte) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "swap", key, params, hookData)
}

// Swap is a paid mutator transaction binding the contract method 0xf3cd914c.
//
// Solidity: function swap((address,address,uint24,int24,address) key, (bool,int256,uint160) params, bytes hookData) returns(int256 swapDelta)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Swap(key PoolKey, params IPoolManagerSwapParams, hookData []byte) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Swap(&_UniswapV4PoolManager.TransactOpts, key, params, hookData)
}

// Swap is a paid mutator transaction binding the contract method 0xf3cd914c.
//
// Solidity: function swap((address,address,uint24,int24,address) key, (bool,int256,uint160) params, bytes hookData) returns(int256 swapDelta)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) Swap(key PoolKey, params IPoolManagerSwapParams, hookData []byte) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Swap(&_UniswapV4PoolManager.TransactOpts, key, params, hookData)
}

// Sync is a paid mutator transaction binding the contract method 0xa5841194.
//
// Solidity: function sync(address currency) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) Sync(opts *bind.TransactOpts, currency common.Address) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "sync", currency)
}

// Sync is a paid mutator transaction binding the contract method 0xa5841194.
//
// Solidity: function sync(address currency) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Sync(currency common.Address) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Sync(&_UniswapV4PoolManager.TransactOpts, currency)
}

// Sync is a paid mutator transaction binding the contract method 0xa5841194.
//
// Solidity: function sync(address currency) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) Sync(currency common.Address) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Sync(&_UniswapV4PoolManager.TransactOpts, currency)
}

// Take is a paid mutator transaction binding the contract method 0x0b0d9c09.
//
// Solidity: function take(address currency, address to, uint256 amount) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) Take(opts *bind.TransactOpts, currency common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "take", currency, to, amount)
}

// Take is a paid mutator transaction binding the contract method 0x0b0d9c09.
//
// Solidity: function take(address currency, address to, uint256 amount) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Take(currency common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Take(&_UniswapV4PoolManager.TransactOpts, currency, to, amount)
}

// Take is a paid mutator transaction binding the contract method 0x0b0d9c09.
//
// Solidity: function take(address currency, address to, uint256 amount) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) Take(currency common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Take(&_UniswapV4PoolManager.TransactOpts, currency, to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0x095bcdb6.
//
// Solidity: function transfer(address receiver, uint256 id, uint256 amount) returns(bool)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) Transfer(opts *bind.TransactOpts, receiver common.Address, id *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "transfer", receiver, id, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0x095bcdb6.
//
// Solidity: function transfer(address receiver, uint256 id, uint256 amount) returns(bool)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Transfer(receiver common.Address, id *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Transfer(&_UniswapV4PoolManager.TransactOpts, receiver, id, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0x095bcdb6.
//
// Solidity: function transfer(address receiver, uint256 id, uint256 amount) returns(bool)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) Transfer(receiver common.Address, id *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Transfer(&_UniswapV4PoolManager.TransactOpts, receiver, id, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0xfe99049a.
//
// Solidity: function transferFrom(address sender, address receiver, uint256 id, uint256 amount) returns(bool)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) TransferFrom(opts *bind.TransactOpts, sender common.Address, receiver common.Address, id *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "transferFrom", sender, receiver, id, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0xfe99049a.
//
// Solidity: function transferFrom(address sender, address receiver, uint256 id, uint256 amount) returns(bool)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) TransferFrom(sender common.Address, receiver common.Address, id *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.TransferFrom(&_UniswapV4PoolManager.TransactOpts, sender, receiver, id, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0xfe99049a.
//
// Solidity: function transferFrom(address sender, address receiver, uint256 id, uint256 amount) returns(bool)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) TransferFrom(sender common.Address, receiver common.Address, id *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.TransferFrom(&_UniswapV4PoolManager.TransactOpts, sender, receiver, id, amount)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.TransferOwnership(&_UniswapV4PoolManager.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.TransferOwnership(&_UniswapV4PoolManager.TransactOpts, newOwner)
}

// Unlock is a paid mutator transaction binding the contract method 0x48c89491.
//
// Solidity: function unlock(bytes data) returns(bytes result)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) Unlock(opts *bind.TransactOpts, data []byte) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "unlock", data)
}

// Unlock is a paid mutator transaction binding the contract method 0x48c89491.
//
// Solidity: function unlock(bytes data) returns(bytes result)
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) Unlock(data []byte) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Unlock(&_UniswapV4PoolManager.TransactOpts, data)
}

// Unlock is a paid mutator transaction binding the contract method 0x48c89491.
//
// Solidity: function unlock(bytes data) returns(bytes result)
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) Unlock(data []byte) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.Unlock(&_UniswapV4PoolManager.TransactOpts, data)
}

// UpdateDynamicLPFee is a paid mutator transaction binding the contract method 0x52759651.
//
// Solidity: function updateDynamicLPFee((address,address,uint24,int24,address) key, uint24 newDynamicLPFee) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactor) UpdateDynamicLPFee(opts *bind.TransactOpts, key PoolKey, newDynamicLPFee *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.contract.Transact(opts, "updateDynamicLPFee", key, newDynamicLPFee)
}

// UpdateDynamicLPFee is a paid mutator transaction binding the contract method 0x52759651.
//
// Solidity: function updateDynamicLPFee((address,address,uint24,int24,address) key, uint24 newDynamicLPFee) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerSession) UpdateDynamicLPFee(key PoolKey, newDynamicLPFee *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.UpdateDynamicLPFee(&_UniswapV4PoolManager.TransactOpts, key, newDynamicLPFee)
}

// UpdateDynamicLPFee is a paid mutator transaction binding the contract method 0x52759651.
//
// Solidity: function updateDynamicLPFee((address,address,uint24,int24,address) key, uint24 newDynamicLPFee) returns()
func (_UniswapV4PoolManager *UniswapV4PoolManagerTransactorSession) UpdateDynamicLPFee(key PoolKey, newDynamicLPFee *big.Int) (*types.Transaction, error) {
	return _UniswapV4PoolManager.Contract.UpdateDynamicLPFee(&_UniswapV4PoolManager.TransactOpts, key, newDynamicLPFee)
}

// UniswapV4PoolManagerApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerApprovalIterator struct {
	Event *UniswapV4PoolManagerApproval // Event containing the contract specifics and raw log

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
func (it *UniswapV4PoolManagerApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UniswapV4PoolManagerApproval)
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
		it.Event = new(UniswapV4PoolManagerApproval)
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
func (it *UniswapV4PoolManagerApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UniswapV4PoolManagerApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UniswapV4PoolManagerApproval represents a Approval event raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerApproval struct {
	Owner   common.Address
	Spender common.Address
	Id      *big.Int
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0xb3fd5071835887567a0671151121894ddccc2842f1d10bedad13e0d17cace9a7.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 indexed id, uint256 amount)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address, id []*big.Int) (*UniswapV4PoolManagerApprovalIterator, error) {

	var ownerRule []any
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []any
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}
	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule, idRule)
	if err != nil {
		return nil, err
	}
	return &UniswapV4PoolManagerApprovalIterator{contract: _UniswapV4PoolManager.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0xb3fd5071835887567a0671151121894ddccc2842f1d10bedad13e0d17cace9a7.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 indexed id, uint256 amount)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *UniswapV4PoolManagerApproval, owner []common.Address, spender []common.Address, id []*big.Int) (event.Subscription, error) {

	var ownerRule []any
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []any
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}
	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule, idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UniswapV4PoolManagerApproval)
				if err := _UniswapV4PoolManager.contract.UnpackLog(event, "Approval", log); err != nil {
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

// ParseApproval is a log parse operation binding the contract event 0xb3fd5071835887567a0671151121894ddccc2842f1d10bedad13e0d17cace9a7.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 indexed id, uint256 amount)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) ParseApproval(log types.Log) (*UniswapV4PoolManagerApproval, error) {
	event := new(UniswapV4PoolManagerApproval)
	if err := _UniswapV4PoolManager.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UniswapV4PoolManagerDonateIterator is returned from FilterDonate and is used to iterate over the raw logs and unpacked data for Donate events raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerDonateIterator struct {
	Event *UniswapV4PoolManagerDonate // Event containing the contract specifics and raw log

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
func (it *UniswapV4PoolManagerDonateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UniswapV4PoolManagerDonate)
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
		it.Event = new(UniswapV4PoolManagerDonate)
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
func (it *UniswapV4PoolManagerDonateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UniswapV4PoolManagerDonateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UniswapV4PoolManagerDonate represents a Donate event raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerDonate struct {
	Id      [32]byte
	Sender  common.Address
	Amount0 *big.Int
	Amount1 *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterDonate is a free log retrieval operation binding the contract event 0x29ef05caaff9404b7cb6d1c0e9bbae9eaa7ab2541feba1a9c4248594c08156cb.
//
// Solidity: event Donate(bytes32 indexed id, address indexed sender, uint256 amount0, uint256 amount1)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) FilterDonate(opts *bind.FilterOpts, id [][32]byte, sender []common.Address) (*UniswapV4PoolManagerDonateIterator, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.FilterLogs(opts, "Donate", idRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &UniswapV4PoolManagerDonateIterator{contract: _UniswapV4PoolManager.contract, event: "Donate", logs: logs, sub: sub}, nil
}

// WatchDonate is a free log subscription operation binding the contract event 0x29ef05caaff9404b7cb6d1c0e9bbae9eaa7ab2541feba1a9c4248594c08156cb.
//
// Solidity: event Donate(bytes32 indexed id, address indexed sender, uint256 amount0, uint256 amount1)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) WatchDonate(opts *bind.WatchOpts, sink chan<- *UniswapV4PoolManagerDonate, id [][32]byte, sender []common.Address) (event.Subscription, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.WatchLogs(opts, "Donate", idRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UniswapV4PoolManagerDonate)
				if err := _UniswapV4PoolManager.contract.UnpackLog(event, "Donate", log); err != nil {
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

// ParseDonate is a log parse operation binding the contract event 0x29ef05caaff9404b7cb6d1c0e9bbae9eaa7ab2541feba1a9c4248594c08156cb.
//
// Solidity: event Donate(bytes32 indexed id, address indexed sender, uint256 amount0, uint256 amount1)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) ParseDonate(log types.Log) (*UniswapV4PoolManagerDonate, error) {
	event := new(UniswapV4PoolManagerDonate)
	if err := _UniswapV4PoolManager.contract.UnpackLog(event, "Donate", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UniswapV4PoolManagerInitializeIterator is returned from FilterInitialize and is used to iterate over the raw logs and unpacked data for Initialize events raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerInitializeIterator struct {
	Event *UniswapV4PoolManagerInitialize // Event containing the contract specifics and raw log

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
func (it *UniswapV4PoolManagerInitializeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UniswapV4PoolManagerInitialize)
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
		it.Event = new(UniswapV4PoolManagerInitialize)
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
func (it *UniswapV4PoolManagerInitializeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UniswapV4PoolManagerInitializeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UniswapV4PoolManagerInitialize represents a Initialize event raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerInitialize struct {
	Id           [32]byte
	Currency0    common.Address
	Currency1    common.Address
	Fee          *big.Int
	TickSpacing  *big.Int
	Hooks        common.Address
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterInitialize is a free log retrieval operation binding the contract event 0xdd466e674ea557f56295e2d0218a125ea4b4f0f6f3307b95f85e6110838d6438.
//
// Solidity: event Initialize(bytes32 indexed id, address indexed currency0, address indexed currency1, uint24 fee, int24 tickSpacing, address hooks, uint160 sqrtPriceX96, int24 tick)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) FilterInitialize(opts *bind.FilterOpts, id [][32]byte, currency0 []common.Address, currency1 []common.Address) (*UniswapV4PoolManagerInitializeIterator, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var currency0Rule []any
	for _, currency0Item := range currency0 {
		currency0Rule = append(currency0Rule, currency0Item)
	}
	var currency1Rule []any
	for _, currency1Item := range currency1 {
		currency1Rule = append(currency1Rule, currency1Item)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.FilterLogs(opts, "Initialize", idRule, currency0Rule, currency1Rule)
	if err != nil {
		return nil, err
	}
	return &UniswapV4PoolManagerInitializeIterator{contract: _UniswapV4PoolManager.contract, event: "Initialize", logs: logs, sub: sub}, nil
}

// WatchInitialize is a free log subscription operation binding the contract event 0xdd466e674ea557f56295e2d0218a125ea4b4f0f6f3307b95f85e6110838d6438.
//
// Solidity: event Initialize(bytes32 indexed id, address indexed currency0, address indexed currency1, uint24 fee, int24 tickSpacing, address hooks, uint160 sqrtPriceX96, int24 tick)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) WatchInitialize(opts *bind.WatchOpts, sink chan<- *UniswapV4PoolManagerInitialize, id [][32]byte, currency0 []common.Address, currency1 []common.Address) (event.Subscription, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var currency0Rule []any
	for _, currency0Item := range currency0 {
		currency0Rule = append(currency0Rule, currency0Item)
	}
	var currency1Rule []any
	for _, currency1Item := range currency1 {
		currency1Rule = append(currency1Rule, currency1Item)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.WatchLogs(opts, "Initialize", idRule, currency0Rule, currency1Rule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UniswapV4PoolManagerInitialize)
				if err := _UniswapV4PoolManager.contract.UnpackLog(event, "Initialize", log); err != nil {
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

// ParseInitialize is a log parse operation binding the contract event 0xdd466e674ea557f56295e2d0218a125ea4b4f0f6f3307b95f85e6110838d6438.
//
// Solidity: event Initialize(bytes32 indexed id, address indexed currency0, address indexed currency1, uint24 fee, int24 tickSpacing, address hooks, uint160 sqrtPriceX96, int24 tick)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) ParseInitialize(log types.Log) (*UniswapV4PoolManagerInitialize, error) {
	event := new(UniswapV4PoolManagerInitialize)
	if err := _UniswapV4PoolManager.contract.UnpackLog(event, "Initialize", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UniswapV4PoolManagerModifyLiquidityIterator is returned from FilterModifyLiquidity and is used to iterate over the raw logs and unpacked data for ModifyLiquidity events raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerModifyLiquidityIterator struct {
	Event *UniswapV4PoolManagerModifyLiquidity // Event containing the contract specifics and raw log

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
func (it *UniswapV4PoolManagerModifyLiquidityIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UniswapV4PoolManagerModifyLiquidity)
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
		it.Event = new(UniswapV4PoolManagerModifyLiquidity)
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
func (it *UniswapV4PoolManagerModifyLiquidityIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UniswapV4PoolManagerModifyLiquidityIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UniswapV4PoolManagerModifyLiquidity represents a ModifyLiquidity event raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerModifyLiquidity struct {
	Id             [32]byte
	Sender         common.Address
	TickLower      *big.Int
	TickUpper      *big.Int
	LiquidityDelta *big.Int
	Salt           [32]byte
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterModifyLiquidity is a free log retrieval operation binding the contract event 0xf208f4912782fd25c7f114ca3723a2d5dd6f3bcc3ac8db5af63baa85f711d5ec.
//
// Solidity: event ModifyLiquidity(bytes32 indexed id, address indexed sender, int24 tickLower, int24 tickUpper, int256 liquidityDelta, bytes32 salt)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) FilterModifyLiquidity(opts *bind.FilterOpts, id [][32]byte, sender []common.Address) (*UniswapV4PoolManagerModifyLiquidityIterator, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.FilterLogs(opts, "ModifyLiquidity", idRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &UniswapV4PoolManagerModifyLiquidityIterator{contract: _UniswapV4PoolManager.contract, event: "ModifyLiquidity", logs: logs, sub: sub}, nil
}

// WatchModifyLiquidity is a free log subscription operation binding the contract event 0xf208f4912782fd25c7f114ca3723a2d5dd6f3bcc3ac8db5af63baa85f711d5ec.
//
// Solidity: event ModifyLiquidity(bytes32 indexed id, address indexed sender, int24 tickLower, int24 tickUpper, int256 liquidityDelta, bytes32 salt)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) WatchModifyLiquidity(opts *bind.WatchOpts, sink chan<- *UniswapV4PoolManagerModifyLiquidity, id [][32]byte, sender []common.Address) (event.Subscription, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.WatchLogs(opts, "ModifyLiquidity", idRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UniswapV4PoolManagerModifyLiquidity)
				if err := _UniswapV4PoolManager.contract.UnpackLog(event, "ModifyLiquidity", log); err != nil {
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

// ParseModifyLiquidity is a log parse operation binding the contract event 0xf208f4912782fd25c7f114ca3723a2d5dd6f3bcc3ac8db5af63baa85f711d5ec.
//
// Solidity: event ModifyLiquidity(bytes32 indexed id, address indexed sender, int24 tickLower, int24 tickUpper, int256 liquidityDelta, bytes32 salt)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) ParseModifyLiquidity(log types.Log) (*UniswapV4PoolManagerModifyLiquidity, error) {
	event := new(UniswapV4PoolManagerModifyLiquidity)
	if err := _UniswapV4PoolManager.contract.UnpackLog(event, "ModifyLiquidity", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UniswapV4PoolManagerOperatorSetIterator is returned from FilterOperatorSet and is used to iterate over the raw logs and unpacked data for OperatorSet events raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerOperatorSetIterator struct {
	Event *UniswapV4PoolManagerOperatorSet // Event containing the contract specifics and raw log

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
func (it *UniswapV4PoolManagerOperatorSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UniswapV4PoolManagerOperatorSet)
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
		it.Event = new(UniswapV4PoolManagerOperatorSet)
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
func (it *UniswapV4PoolManagerOperatorSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UniswapV4PoolManagerOperatorSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UniswapV4PoolManagerOperatorSet represents a OperatorSet event raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerOperatorSet struct {
	Owner    common.Address
	Operator common.Address
	Approved bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterOperatorSet is a free log retrieval operation binding the contract event 0xceb576d9f15e4e200fdb5096d64d5dfd667e16def20c1eefd14256d8e3faa267.
//
// Solidity: event OperatorSet(address indexed owner, address indexed operator, bool approved)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) FilterOperatorSet(opts *bind.FilterOpts, owner []common.Address, operator []common.Address) (*UniswapV4PoolManagerOperatorSetIterator, error) {

	var ownerRule []any
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []any
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.FilterLogs(opts, "OperatorSet", ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return &UniswapV4PoolManagerOperatorSetIterator{contract: _UniswapV4PoolManager.contract, event: "OperatorSet", logs: logs, sub: sub}, nil
}

// WatchOperatorSet is a free log subscription operation binding the contract event 0xceb576d9f15e4e200fdb5096d64d5dfd667e16def20c1eefd14256d8e3faa267.
//
// Solidity: event OperatorSet(address indexed owner, address indexed operator, bool approved)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) WatchOperatorSet(opts *bind.WatchOpts, sink chan<- *UniswapV4PoolManagerOperatorSet, owner []common.Address, operator []common.Address) (event.Subscription, error) {

	var ownerRule []any
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []any
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.WatchLogs(opts, "OperatorSet", ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UniswapV4PoolManagerOperatorSet)
				if err := _UniswapV4PoolManager.contract.UnpackLog(event, "OperatorSet", log); err != nil {
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

// ParseOperatorSet is a log parse operation binding the contract event 0xceb576d9f15e4e200fdb5096d64d5dfd667e16def20c1eefd14256d8e3faa267.
//
// Solidity: event OperatorSet(address indexed owner, address indexed operator, bool approved)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) ParseOperatorSet(log types.Log) (*UniswapV4PoolManagerOperatorSet, error) {
	event := new(UniswapV4PoolManagerOperatorSet)
	if err := _UniswapV4PoolManager.contract.UnpackLog(event, "OperatorSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UniswapV4PoolManagerOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerOwnershipTransferredIterator struct {
	Event *UniswapV4PoolManagerOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *UniswapV4PoolManagerOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UniswapV4PoolManagerOwnershipTransferred)
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
		it.Event = new(UniswapV4PoolManagerOwnershipTransferred)
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
func (it *UniswapV4PoolManagerOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UniswapV4PoolManagerOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UniswapV4PoolManagerOwnershipTransferred represents a OwnershipTransferred event raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerOwnershipTransferred struct {
	User     common.Address
	NewOwner common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed user, address indexed newOwner)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, user []common.Address, newOwner []common.Address) (*UniswapV4PoolManagerOwnershipTransferredIterator, error) {

	var userRule []any
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}
	var newOwnerRule []any
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.FilterLogs(opts, "OwnershipTransferred", userRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &UniswapV4PoolManagerOwnershipTransferredIterator{contract: _UniswapV4PoolManager.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed user, address indexed newOwner)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *UniswapV4PoolManagerOwnershipTransferred, user []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var userRule []any
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}
	var newOwnerRule []any
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.WatchLogs(opts, "OwnershipTransferred", userRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UniswapV4PoolManagerOwnershipTransferred)
				if err := _UniswapV4PoolManager.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
// Solidity: event OwnershipTransferred(address indexed user, address indexed newOwner)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) ParseOwnershipTransferred(log types.Log) (*UniswapV4PoolManagerOwnershipTransferred, error) {
	event := new(UniswapV4PoolManagerOwnershipTransferred)
	if err := _UniswapV4PoolManager.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UniswapV4PoolManagerProtocolFeeControllerUpdatedIterator is returned from FilterProtocolFeeControllerUpdated and is used to iterate over the raw logs and unpacked data for ProtocolFeeControllerUpdated events raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerProtocolFeeControllerUpdatedIterator struct {
	Event *UniswapV4PoolManagerProtocolFeeControllerUpdated // Event containing the contract specifics and raw log

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
func (it *UniswapV4PoolManagerProtocolFeeControllerUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UniswapV4PoolManagerProtocolFeeControllerUpdated)
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
		it.Event = new(UniswapV4PoolManagerProtocolFeeControllerUpdated)
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
func (it *UniswapV4PoolManagerProtocolFeeControllerUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UniswapV4PoolManagerProtocolFeeControllerUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UniswapV4PoolManagerProtocolFeeControllerUpdated represents a ProtocolFeeControllerUpdated event raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerProtocolFeeControllerUpdated struct {
	ProtocolFeeController common.Address
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterProtocolFeeControllerUpdated is a free log retrieval operation binding the contract event 0xb4bd8ef53df690b9943d3318996006dbb82a25f54719d8c8035b516a2a5b8acc.
//
// Solidity: event ProtocolFeeControllerUpdated(address indexed protocolFeeController)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) FilterProtocolFeeControllerUpdated(opts *bind.FilterOpts, protocolFeeController []common.Address) (*UniswapV4PoolManagerProtocolFeeControllerUpdatedIterator, error) {

	var protocolFeeControllerRule []any
	for _, protocolFeeControllerItem := range protocolFeeController {
		protocolFeeControllerRule = append(protocolFeeControllerRule, protocolFeeControllerItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.FilterLogs(opts, "ProtocolFeeControllerUpdated", protocolFeeControllerRule)
	if err != nil {
		return nil, err
	}
	return &UniswapV4PoolManagerProtocolFeeControllerUpdatedIterator{contract: _UniswapV4PoolManager.contract, event: "ProtocolFeeControllerUpdated", logs: logs, sub: sub}, nil
}

// WatchProtocolFeeControllerUpdated is a free log subscription operation binding the contract event 0xb4bd8ef53df690b9943d3318996006dbb82a25f54719d8c8035b516a2a5b8acc.
//
// Solidity: event ProtocolFeeControllerUpdated(address indexed protocolFeeController)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) WatchProtocolFeeControllerUpdated(opts *bind.WatchOpts, sink chan<- *UniswapV4PoolManagerProtocolFeeControllerUpdated, protocolFeeController []common.Address) (event.Subscription, error) {

	var protocolFeeControllerRule []any
	for _, protocolFeeControllerItem := range protocolFeeController {
		protocolFeeControllerRule = append(protocolFeeControllerRule, protocolFeeControllerItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.WatchLogs(opts, "ProtocolFeeControllerUpdated", protocolFeeControllerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UniswapV4PoolManagerProtocolFeeControllerUpdated)
				if err := _UniswapV4PoolManager.contract.UnpackLog(event, "ProtocolFeeControllerUpdated", log); err != nil {
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

// ParseProtocolFeeControllerUpdated is a log parse operation binding the contract event 0xb4bd8ef53df690b9943d3318996006dbb82a25f54719d8c8035b516a2a5b8acc.
//
// Solidity: event ProtocolFeeControllerUpdated(address indexed protocolFeeController)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) ParseProtocolFeeControllerUpdated(log types.Log) (*UniswapV4PoolManagerProtocolFeeControllerUpdated, error) {
	event := new(UniswapV4PoolManagerProtocolFeeControllerUpdated)
	if err := _UniswapV4PoolManager.contract.UnpackLog(event, "ProtocolFeeControllerUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UniswapV4PoolManagerProtocolFeeUpdatedIterator is returned from FilterProtocolFeeUpdated and is used to iterate over the raw logs and unpacked data for ProtocolFeeUpdated events raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerProtocolFeeUpdatedIterator struct {
	Event *UniswapV4PoolManagerProtocolFeeUpdated // Event containing the contract specifics and raw log

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
func (it *UniswapV4PoolManagerProtocolFeeUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UniswapV4PoolManagerProtocolFeeUpdated)
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
		it.Event = new(UniswapV4PoolManagerProtocolFeeUpdated)
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
func (it *UniswapV4PoolManagerProtocolFeeUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UniswapV4PoolManagerProtocolFeeUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UniswapV4PoolManagerProtocolFeeUpdated represents a ProtocolFeeUpdated event raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerProtocolFeeUpdated struct {
	Id          [32]byte
	ProtocolFee *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterProtocolFeeUpdated is a free log retrieval operation binding the contract event 0xe9c42593e71f84403b84352cd168d693e2c9fcd1fdbcc3feb21d92b43e6696f9.
//
// Solidity: event ProtocolFeeUpdated(bytes32 indexed id, uint24 protocolFee)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) FilterProtocolFeeUpdated(opts *bind.FilterOpts, id [][32]byte) (*UniswapV4PoolManagerProtocolFeeUpdatedIterator, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.FilterLogs(opts, "ProtocolFeeUpdated", idRule)
	if err != nil {
		return nil, err
	}
	return &UniswapV4PoolManagerProtocolFeeUpdatedIterator{contract: _UniswapV4PoolManager.contract, event: "ProtocolFeeUpdated", logs: logs, sub: sub}, nil
}

// WatchProtocolFeeUpdated is a free log subscription operation binding the contract event 0xe9c42593e71f84403b84352cd168d693e2c9fcd1fdbcc3feb21d92b43e6696f9.
//
// Solidity: event ProtocolFeeUpdated(bytes32 indexed id, uint24 protocolFee)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) WatchProtocolFeeUpdated(opts *bind.WatchOpts, sink chan<- *UniswapV4PoolManagerProtocolFeeUpdated, id [][32]byte) (event.Subscription, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.WatchLogs(opts, "ProtocolFeeUpdated", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UniswapV4PoolManagerProtocolFeeUpdated)
				if err := _UniswapV4PoolManager.contract.UnpackLog(event, "ProtocolFeeUpdated", log); err != nil {
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

// ParseProtocolFeeUpdated is a log parse operation binding the contract event 0xe9c42593e71f84403b84352cd168d693e2c9fcd1fdbcc3feb21d92b43e6696f9.
//
// Solidity: event ProtocolFeeUpdated(bytes32 indexed id, uint24 protocolFee)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) ParseProtocolFeeUpdated(log types.Log) (*UniswapV4PoolManagerProtocolFeeUpdated, error) {
	event := new(UniswapV4PoolManagerProtocolFeeUpdated)
	if err := _UniswapV4PoolManager.contract.UnpackLog(event, "ProtocolFeeUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UniswapV4PoolManagerSwapIterator is returned from FilterSwap and is used to iterate over the raw logs and unpacked data for Swap events raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerSwapIterator struct {
	Event *UniswapV4PoolManagerSwap // Event containing the contract specifics and raw log

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
func (it *UniswapV4PoolManagerSwapIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UniswapV4PoolManagerSwap)
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
		it.Event = new(UniswapV4PoolManagerSwap)
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
func (it *UniswapV4PoolManagerSwapIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UniswapV4PoolManagerSwapIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UniswapV4PoolManagerSwap represents a Swap event raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerSwap struct {
	Id           [32]byte
	Sender       common.Address
	Amount0      *big.Int
	Amount1      *big.Int
	SqrtPriceX96 *big.Int
	Liquidity    *big.Int
	Tick         *big.Int
	Fee          *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterSwap is a free log retrieval operation binding the contract event 0x40e9cecb9f5f1f1c5b9c97dec2917b7ee92e57ba5563708daca94dd84ad7112f.
//
// Solidity: event Swap(bytes32 indexed id, address indexed sender, int128 amount0, int128 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick, uint24 fee)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) FilterSwap(opts *bind.FilterOpts, id [][32]byte, sender []common.Address) (*UniswapV4PoolManagerSwapIterator, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.FilterLogs(opts, "Swap", idRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &UniswapV4PoolManagerSwapIterator{contract: _UniswapV4PoolManager.contract, event: "Swap", logs: logs, sub: sub}, nil
}

// WatchSwap is a free log subscription operation binding the contract event 0x40e9cecb9f5f1f1c5b9c97dec2917b7ee92e57ba5563708daca94dd84ad7112f.
//
// Solidity: event Swap(bytes32 indexed id, address indexed sender, int128 amount0, int128 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick, uint24 fee)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) WatchSwap(opts *bind.WatchOpts, sink chan<- *UniswapV4PoolManagerSwap, id [][32]byte, sender []common.Address) (event.Subscription, error) {

	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.WatchLogs(opts, "Swap", idRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UniswapV4PoolManagerSwap)
				if err := _UniswapV4PoolManager.contract.UnpackLog(event, "Swap", log); err != nil {
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

// ParseSwap is a log parse operation binding the contract event 0x40e9cecb9f5f1f1c5b9c97dec2917b7ee92e57ba5563708daca94dd84ad7112f.
//
// Solidity: event Swap(bytes32 indexed id, address indexed sender, int128 amount0, int128 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick, uint24 fee)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) ParseSwap(log types.Log) (*UniswapV4PoolManagerSwap, error) {
	event := new(UniswapV4PoolManagerSwap)
	if err := _UniswapV4PoolManager.contract.UnpackLog(event, "Swap", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UniswapV4PoolManagerTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerTransferIterator struct {
	Event *UniswapV4PoolManagerTransfer // Event containing the contract specifics and raw log

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
func (it *UniswapV4PoolManagerTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UniswapV4PoolManagerTransfer)
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
		it.Event = new(UniswapV4PoolManagerTransfer)
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
func (it *UniswapV4PoolManagerTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UniswapV4PoolManagerTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UniswapV4PoolManagerTransfer represents a Transfer event raised by the UniswapV4PoolManager contract.
type UniswapV4PoolManagerTransfer struct {
	Caller common.Address
	From   common.Address
	To     common.Address
	Id     *big.Int
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0x1b3d7edb2e9c0b0e7c525b20aaaef0f5940d2ed71663c7d39266ecafac728859.
//
// Solidity: event Transfer(address caller, address indexed from, address indexed to, uint256 indexed id, uint256 amount)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address, id []*big.Int) (*UniswapV4PoolManagerTransferIterator, error) {

	var fromRule []any
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []any
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.FilterLogs(opts, "Transfer", fromRule, toRule, idRule)
	if err != nil {
		return nil, err
	}
	return &UniswapV4PoolManagerTransferIterator{contract: _UniswapV4PoolManager.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0x1b3d7edb2e9c0b0e7c525b20aaaef0f5940d2ed71663c7d39266ecafac728859.
//
// Solidity: event Transfer(address caller, address indexed from, address indexed to, uint256 indexed id, uint256 amount)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *UniswapV4PoolManagerTransfer, from []common.Address, to []common.Address, id []*big.Int) (event.Subscription, error) {

	var fromRule []any
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []any
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _UniswapV4PoolManager.contract.WatchLogs(opts, "Transfer", fromRule, toRule, idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UniswapV4PoolManagerTransfer)
				if err := _UniswapV4PoolManager.contract.UnpackLog(event, "Transfer", log); err != nil {
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

// ParseTransfer is a log parse operation binding the contract event 0x1b3d7edb2e9c0b0e7c525b20aaaef0f5940d2ed71663c7d39266ecafac728859.
//
// Solidity: event Transfer(address caller, address indexed from, address indexed to, uint256 indexed id, uint256 amount)
func (_UniswapV4PoolManager *UniswapV4PoolManagerFilterer) ParseTransfer(log types.Log) (*UniswapV4PoolManagerTransfer, error) {
	event := new(UniswapV4PoolManagerTransfer)
	if err := _UniswapV4PoolManager.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
