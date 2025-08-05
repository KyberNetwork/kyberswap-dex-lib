// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bunniv2

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

// IAmAmmBid is an auto generated low-level Go binding around an user-defined struct.
type IAmAmmBid struct {
	Manager  common.Address
	BlockIdx *big.Int
	Payload  [6]byte
	Rent     *big.Int
	Deposit  *big.Int
}

// IBunniHookRebalanceOrderHookArgs is an auto generated low-level Go binding around an user-defined struct.
type IBunniHookRebalanceOrderHookArgs struct {
	Key          PoolKey
	PreHookArgs  IBunniHookRebalanceOrderPreHookArgs
	PostHookArgs IBunniHookRebalanceOrderPostHookArgs
}

// IBunniHookRebalanceOrderPostHookArgs is an auto generated low-level Go binding around an user-defined struct.
type IBunniHookRebalanceOrderPostHookArgs struct {
	Currency common.Address
}

// IBunniHookRebalanceOrderPreHookArgs is an auto generated low-level Go binding around an user-defined struct.
type IBunniHookRebalanceOrderPreHookArgs struct {
	Currency common.Address
	Amount   *big.Int
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

// Bunniv2MetaData contains all meta data concerning the Bunniv2 contract.
var Bunniv2MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIPoolManager\",\"name\":\"poolManager_\",\"type\":\"address\"},{\"internalType\":\"contractIBunniHub\",\"name\":\"hub_\",\"type\":\"address\"},{\"internalType\":\"contractIFloodPlain\",\"name\":\"floodPlain_\",\"type\":\"address\"},{\"internalType\":\"contractWETH\",\"name\":\"weth_\",\"type\":\"address\"},{\"internalType\":\"contractIZone\",\"name\":\"floodZone_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"hookFeeRecipientController_\",\"type\":\"address\"},{\"internalType\":\"uint48\",\"name\":\"k_\",\"type\":\"uint48\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AlreadyInitialized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AmAmm__BidLocked\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AmAmm__InvalidBid\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AmAmm__InvalidDepositAmount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AmAmm__NotEnabled\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AmAmm__Unauthorized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHook__HookFeeRecipientAlreadySet\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHook__HookFeeRecipientNotSet\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHook__InvalidActiveBlock\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHook__InvalidCuratorFee\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHook__InvalidK\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHook__InvalidModifier\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHook__InvalidRebalanceOrderHash\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHook__PrehookPostConditionFailed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHook__RebalanceInProgress\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHook__Unauthorized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NewOwnerIsZeroAddress\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NoHandoverRequest\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotPoolManager\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OracleCardinalityCannotBeZero\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ReentrancyGuard__ReentrantCall\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Unauthorized\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"Currency\",\"name\":\"currency\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"fees\",\"type\":\"uint256\"}],\"name\":\"ClaimFees\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"Currency[]\",\"name\":\"currencyList\",\"type\":\"address[]\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"}],\"name\":\"ClaimProtocolFees\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"refund\",\"type\":\"uint256\"}],\"name\":\"ClaimRefund\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"feeAmount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"feeAmount1\",\"type\":\"uint256\"}],\"name\":\"CuratorClaimFees\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint16\",\"name\":\"newFeeRate\",\"type\":\"uint16\"}],\"name\":\"CuratorSetFeeRate\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"}],\"name\":\"DepositIntoNextBid\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"}],\"name\":\"DepositIntoTopBid\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"additionalRent\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"updatedDeposit\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"topBid\",\"type\":\"bool\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"withdrawRecipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amountDeposited\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amountWithdrawn\",\"type\":\"uint128\"}],\"name\":\"IncreaseBidRent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"pendingOwner\",\"type\":\"address\"}],\"name\":\"OwnershipHandoverCanceled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"pendingOwner\",\"type\":\"address\"}],\"name\":\"OwnershipHandoverRequested\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint48\",\"name\":\"currentK\",\"type\":\"uint48\"},{\"indexed\":true,\"internalType\":\"uint48\",\"name\":\"newK\",\"type\":\"uint48\"},{\"indexed\":true,\"internalType\":\"uint160\",\"name\":\"activeBlock\",\"type\":\"uint160\"}],\"name\":\"ScheduleKChange\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes6\",\"name\":\"payload\",\"type\":\"bytes6\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"topBid\",\"type\":\"bool\"}],\"name\":\"SetBidPayload\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint32\",\"name\":\"hookFeeModifier\",\"type\":\"uint32\"}],\"name\":\"SetHookFeeModifier\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"hookFeeRecipient\",\"type\":\"address\"}],\"name\":\"SetHookFeeRecipient\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"unblocked\",\"type\":\"bool\"}],\"name\":\"SetWithdrawalUnblocked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"contractIZone\",\"name\":\"zone\",\"type\":\"address\"}],\"name\":\"SetZone\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint48\",\"name\":\"blockIdx\",\"type\":\"uint48\"},{\"indexed\":false,\"internalType\":\"bytes6\",\"name\":\"payload\",\"type\":\"bytes6\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"rent\",\"type\":\"uint128\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"deposit\",\"type\":\"uint128\"}],\"name\":\"SubmitBid\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"exactIn\",\"type\":\"bool\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"inputAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"outputAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"indexed\":false,\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"},{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"totalLiquidity\",\"type\":\"uint256\"}],\"name\":\"Swap\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"}],\"name\":\"WithdrawFromNextBid\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"}],\"name\":\"WithdrawFromTopBid\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"caller\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"}],\"name\":\"afterInitialize\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"int256\",\"name\":\"amountSpecified\",\"type\":\"int256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceLimitX96\",\"type\":\"uint160\"}],\"internalType\":\"structIPoolManager.SwapParams\",\"name\":\"params\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"beforeSwap\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"},{\"internalType\":\"BeforeSwapDelta\",\"name\":\"\",\"type\":\"int256\"},{\"internalType\":\"uint24\",\"name\":\"\",\"type\":\"uint24\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"},{\"internalType\":\"bytes6\",\"name\":\"payload\",\"type\":\"bytes6\"},{\"internalType\":\"uint128\",\"name\":\"rent\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"deposit\",\"type\":\"uint128\"}],\"name\":\"bid\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"canWithdraw\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"cancelOwnershipHandover\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Currency\",\"name\":\"currency\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"}],\"name\":\"claimFees\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"fees\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"Currency[]\",\"name\":\"currencyList\",\"type\":\"address[]\"}],\"name\":\"claimProtocolFees\",\"outputs\":[{\"internalType\":\"uint256[]\",\"name\":\"claimedAmounts\",\"type\":\"uint256[]\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"}],\"name\":\"claimRefund\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"refund\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"pendingOwner\",\"type\":\"address\"}],\"name\":\"completeOwnershipHandover\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"}],\"name\":\"curatorClaimFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"uint16\",\"name\":\"newFeeRate\",\"type\":\"uint16\"}],\"name\":\"curatorSetFeeRate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"},{\"internalType\":\"bool\",\"name\":\"isTopBid\",\"type\":\"bool\"}],\"name\":\"depositIntoBid\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"slots\",\"type\":\"bytes32[]\"}],\"name\":\"extsload\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"getAmAmmEnabled\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"bool\",\"name\":\"isTopBid\",\"type\":\"bool\"}],\"name\":\"getBid\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"},{\"internalType\":\"uint48\",\"name\":\"blockIdx\",\"type\":\"uint48\"},{\"internalType\":\"bytes6\",\"name\":\"payload\",\"type\":\"bytes6\"},{\"internalType\":\"uint128\",\"name\":\"rent\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"deposit\",\"type\":\"uint128\"}],\"internalType\":\"structIAmAmm.Bid\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"bool\",\"name\":\"isTopBid\",\"type\":\"bool\"}],\"name\":\"getBidWrite\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"},{\"internalType\":\"uint48\",\"name\":\"blockIdx\",\"type\":\"uint48\"},{\"internalType\":\"bytes6\",\"name\":\"payload\",\"type\":\"bytes6\"},{\"internalType\":\"uint128\",\"name\":\"rent\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"deposit\",\"type\":\"uint128\"}],\"internalType\":\"structIAmAmm.Bid\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency\",\"type\":\"address\"}],\"name\":\"getFees\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"},{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"getRefund\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"},{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"getRefundWrite\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"uint128\",\"name\":\"additionalRent\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"updatedDeposit\",\"type\":\"uint128\"},{\"internalType\":\"bool\",\"name\":\"isTopBid\",\"type\":\"bool\"},{\"internalType\":\"address\",\"name\":\"withdrawRecipient\",\"type\":\"address\"}],\"name\":\"increaseBidRent\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"amountDeposited\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amountWithdrawn\",\"type\":\"uint128\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint32\",\"name\":\"cardinalityNext\",\"type\":\"uint32\"}],\"name\":\"increaseCardinalityNext\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"cardinalityNextOld\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"cardinalityNextNew\",\"type\":\"uint32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"hookParams\",\"type\":\"bytes\"}],\"name\":\"isValidParams\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"isValidSignature\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"magicValue\",\"type\":\"bytes4\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"ldfStates\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint32[]\",\"name\":\"secondsAgos\",\"type\":\"uint32[]\"}],\"name\":\"observe\",\"outputs\":[{\"internalType\":\"int56[]\",\"name\":\"tickCumulatives\",\"type\":\"int56[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"result\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"pendingOwner\",\"type\":\"address\"}],\"name\":\"ownershipHandoverExpiresAt\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"result\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"isPreHook\",\"type\":\"bool\"},{\"components\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"internalType\":\"structIBunniHook.RebalanceOrderPreHookArgs\",\"name\":\"preHookArgs\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency\",\"type\":\"address\"}],\"internalType\":\"structIBunniHook.RebalanceOrderPostHookArgs\",\"name\":\"postHookArgs\",\"type\":\"tuple\"}],\"internalType\":\"structIBunniHook.RebalanceOrderHookArgs\",\"name\":\"hookArgs\",\"type\":\"tuple\"}],\"name\":\"rebalanceOrderHook\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"requestOwnershipHandover\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint48\",\"name\":\"newK\",\"type\":\"uint48\"},{\"internalType\":\"uint160\",\"name\":\"activeBlock\",\"type\":\"uint160\"}],\"name\":\"scheduleKChange\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"bytes6\",\"name\":\"payload\",\"type\":\"bytes6\"},{\"internalType\":\"bool\",\"name\":\"isTopBid\",\"type\":\"bool\"}],\"name\":\"setBidPayload\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"newHookFeeModifier\",\"type\":\"uint32\"}],\"name\":\"setHookFeeModifier\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newHookFeeRecipient\",\"type\":\"address\"}],\"name\":\"setHookFeeRecipient\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"bool\",\"name\":\"unblocked\",\"type\":\"bool\"}],\"name\":\"setWithdrawalUnblocked\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIZone\",\"name\":\"zone\",\"type\":\"address\"}],\"name\":\"setZone\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"slot0s\",\"outputs\":[{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"internalType\":\"int24\",\"name\":\"tick\",\"type\":\"int24\"},{\"internalType\":\"uint32\",\"name\":\"lastSwapTimestamp\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"lastSurgeTimestamp\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"unlockCallback\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"newState\",\"type\":\"bytes32\"}],\"name\":\"updateLdfState\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"updateStateMachine\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"id\",\"type\":\"bytes32\"},{\"internalType\":\"uint128\",\"name\":\"amount\",\"type\":\"uint128\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isTopBid\",\"type\":\"bool\"}],\"name\":\"withdrawFromBid\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
}

// Bunniv2ABI is the input ABI used to generate the binding from.
// Deprecated: Use Bunniv2MetaData.ABI instead.
var Bunniv2ABI = Bunniv2MetaData.ABI

// Bunniv2 is an auto generated Go binding around an Ethereum contract.
type Bunniv2 struct {
	Bunniv2Caller     // Read-only binding to the contract
	Bunniv2Transactor // Write-only binding to the contract
	Bunniv2Filterer   // Log filterer for contract events
}

// Bunniv2Caller is an auto generated read-only Go binding around an Ethereum contract.
type Bunniv2Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Bunniv2Transactor is an auto generated write-only Go binding around an Ethereum contract.
type Bunniv2Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Bunniv2Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Bunniv2Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Bunniv2Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Bunniv2Session struct {
	Contract     *Bunniv2          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Bunniv2CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Bunniv2CallerSession struct {
	Contract *Bunniv2Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// Bunniv2TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Bunniv2TransactorSession struct {
	Contract     *Bunniv2Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// Bunniv2Raw is an auto generated low-level Go binding around an Ethereum contract.
type Bunniv2Raw struct {
	Contract *Bunniv2 // Generic contract binding to access the raw methods on
}

// Bunniv2CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Bunniv2CallerRaw struct {
	Contract *Bunniv2Caller // Generic read-only contract binding to access the raw methods on
}

// Bunniv2TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Bunniv2TransactorRaw struct {
	Contract *Bunniv2Transactor // Generic write-only contract binding to access the raw methods on
}

// NewBunniv2 creates a new instance of Bunniv2, bound to a specific deployed contract.
func NewBunniv2(address common.Address, backend bind.ContractBackend) (*Bunniv2, error) {
	contract, err := bindBunniv2(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Bunniv2{Bunniv2Caller: Bunniv2Caller{contract: contract}, Bunniv2Transactor: Bunniv2Transactor{contract: contract}, Bunniv2Filterer: Bunniv2Filterer{contract: contract}}, nil
}

// NewBunniv2Caller creates a new read-only instance of Bunniv2, bound to a specific deployed contract.
func NewBunniv2Caller(address common.Address, caller bind.ContractCaller) (*Bunniv2Caller, error) {
	contract, err := bindBunniv2(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Bunniv2Caller{contract: contract}, nil
}

// NewBunniv2Transactor creates a new write-only instance of Bunniv2, bound to a specific deployed contract.
func NewBunniv2Transactor(address common.Address, transactor bind.ContractTransactor) (*Bunniv2Transactor, error) {
	contract, err := bindBunniv2(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Bunniv2Transactor{contract: contract}, nil
}

// NewBunniv2Filterer creates a new log filterer instance of Bunniv2, bound to a specific deployed contract.
func NewBunniv2Filterer(address common.Address, filterer bind.ContractFilterer) (*Bunniv2Filterer, error) {
	contract, err := bindBunniv2(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Bunniv2Filterer{contract: contract}, nil
}

// bindBunniv2 binds a generic wrapper to an already deployed contract.
func bindBunniv2(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := Bunniv2MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Bunniv2 *Bunniv2Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Bunniv2.Contract.Bunniv2Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Bunniv2 *Bunniv2Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Bunniv2.Contract.Bunniv2Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Bunniv2 *Bunniv2Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Bunniv2.Contract.Bunniv2Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Bunniv2 *Bunniv2CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Bunniv2.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Bunniv2 *Bunniv2TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Bunniv2.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Bunniv2 *Bunniv2TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Bunniv2.Contract.contract.Transact(opts, method, params...)
}

// CanWithdraw is a free data retrieval call binding the contract method 0x6fb70c76.
//
// Solidity: function canWithdraw(bytes32 id) view returns(bool)
func (_Bunniv2 *Bunniv2Caller) CanWithdraw(opts *bind.CallOpts, id [32]byte) (bool, error) {
	var out []interface{}
	err := _Bunniv2.contract.Call(opts, &out, "canWithdraw", id)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// CanWithdraw is a free data retrieval call binding the contract method 0x6fb70c76.
//
// Solidity: function canWithdraw(bytes32 id) view returns(bool)
func (_Bunniv2 *Bunniv2Session) CanWithdraw(id [32]byte) (bool, error) {
	return _Bunniv2.Contract.CanWithdraw(&_Bunniv2.CallOpts, id)
}

// CanWithdraw is a free data retrieval call binding the contract method 0x6fb70c76.
//
// Solidity: function canWithdraw(bytes32 id) view returns(bool)
func (_Bunniv2 *Bunniv2CallerSession) CanWithdraw(id [32]byte) (bool, error) {
	return _Bunniv2.Contract.CanWithdraw(&_Bunniv2.CallOpts, id)
}

// Extsload is a free data retrieval call binding the contract method 0xdbd035ff.
//
// Solidity: function extsload(bytes32[] slots) view returns(bytes32[])
func (_Bunniv2 *Bunniv2Caller) Extsload(opts *bind.CallOpts, slots [][32]byte) ([][32]byte, error) {
	var out []interface{}
	err := _Bunniv2.contract.Call(opts, &out, "extsload", slots)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// Extsload is a free data retrieval call binding the contract method 0xdbd035ff.
//
// Solidity: function extsload(bytes32[] slots) view returns(bytes32[])
func (_Bunniv2 *Bunniv2Session) Extsload(slots [][32]byte) ([][32]byte, error) {
	return _Bunniv2.Contract.Extsload(&_Bunniv2.CallOpts, slots)
}

// Extsload is a free data retrieval call binding the contract method 0xdbd035ff.
//
// Solidity: function extsload(bytes32[] slots) view returns(bytes32[])
func (_Bunniv2 *Bunniv2CallerSession) Extsload(slots [][32]byte) ([][32]byte, error) {
	return _Bunniv2.Contract.Extsload(&_Bunniv2.CallOpts, slots)
}

// GetAmAmmEnabled is a free data retrieval call binding the contract method 0xcceb6765.
//
// Solidity: function getAmAmmEnabled(bytes32 id) view returns(bool)
func (_Bunniv2 *Bunniv2Caller) GetAmAmmEnabled(opts *bind.CallOpts, id [32]byte) (bool, error) {
	var out []interface{}
	err := _Bunniv2.contract.Call(opts, &out, "getAmAmmEnabled", id)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// GetAmAmmEnabled is a free data retrieval call binding the contract method 0xcceb6765.
//
// Solidity: function getAmAmmEnabled(bytes32 id) view returns(bool)
func (_Bunniv2 *Bunniv2Session) GetAmAmmEnabled(id [32]byte) (bool, error) {
	return _Bunniv2.Contract.GetAmAmmEnabled(&_Bunniv2.CallOpts, id)
}

// GetAmAmmEnabled is a free data retrieval call binding the contract method 0xcceb6765.
//
// Solidity: function getAmAmmEnabled(bytes32 id) view returns(bool)
func (_Bunniv2 *Bunniv2CallerSession) GetAmAmmEnabled(id [32]byte) (bool, error) {
	return _Bunniv2.Contract.GetAmAmmEnabled(&_Bunniv2.CallOpts, id)
}

// GetBid is a free data retrieval call binding the contract method 0xbbb546b5.
//
// Solidity: function getBid(bytes32 id, bool isTopBid) view returns((address,uint48,bytes6,uint128,uint128))
func (_Bunniv2 *Bunniv2Caller) GetBid(opts *bind.CallOpts, id [32]byte, isTopBid bool) (IAmAmmBid, error) {
	var out []interface{}
	err := _Bunniv2.contract.Call(opts, &out, "getBid", id, isTopBid)

	if err != nil {
		return *new(IAmAmmBid), err
	}

	out0 := *abi.ConvertType(out[0], new(IAmAmmBid)).(*IAmAmmBid)

	return out0, err

}

// GetBid is a free data retrieval call binding the contract method 0xbbb546b5.
//
// Solidity: function getBid(bytes32 id, bool isTopBid) view returns((address,uint48,bytes6,uint128,uint128))
func (_Bunniv2 *Bunniv2Session) GetBid(id [32]byte, isTopBid bool) (IAmAmmBid, error) {
	return _Bunniv2.Contract.GetBid(&_Bunniv2.CallOpts, id, isTopBid)
}

// GetBid is a free data retrieval call binding the contract method 0xbbb546b5.
//
// Solidity: function getBid(bytes32 id, bool isTopBid) view returns((address,uint48,bytes6,uint128,uint128))
func (_Bunniv2 *Bunniv2CallerSession) GetBid(id [32]byte, isTopBid bool) (IAmAmmBid, error) {
	return _Bunniv2.Contract.GetBid(&_Bunniv2.CallOpts, id, isTopBid)
}

// GetFees is a free data retrieval call binding the contract method 0xc982bcca.
//
// Solidity: function getFees(address manager, address currency) view returns(uint256)
func (_Bunniv2 *Bunniv2Caller) GetFees(opts *bind.CallOpts, manager common.Address, currency common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Bunniv2.contract.Call(opts, &out, "getFees", manager, currency)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetFees is a free data retrieval call binding the contract method 0xc982bcca.
//
// Solidity: function getFees(address manager, address currency) view returns(uint256)
func (_Bunniv2 *Bunniv2Session) GetFees(manager common.Address, currency common.Address) (*big.Int, error) {
	return _Bunniv2.Contract.GetFees(&_Bunniv2.CallOpts, manager, currency)
}

// GetFees is a free data retrieval call binding the contract method 0xc982bcca.
//
// Solidity: function getFees(address manager, address currency) view returns(uint256)
func (_Bunniv2 *Bunniv2CallerSession) GetFees(manager common.Address, currency common.Address) (*big.Int, error) {
	return _Bunniv2.Contract.GetFees(&_Bunniv2.CallOpts, manager, currency)
}

// GetRefund is a free data retrieval call binding the contract method 0x2e6ee7c3.
//
// Solidity: function getRefund(address manager, bytes32 id) view returns(uint256)
func (_Bunniv2 *Bunniv2Caller) GetRefund(opts *bind.CallOpts, manager common.Address, id [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Bunniv2.contract.Call(opts, &out, "getRefund", manager, id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetRefund is a free data retrieval call binding the contract method 0x2e6ee7c3.
//
// Solidity: function getRefund(address manager, bytes32 id) view returns(uint256)
func (_Bunniv2 *Bunniv2Session) GetRefund(manager common.Address, id [32]byte) (*big.Int, error) {
	return _Bunniv2.Contract.GetRefund(&_Bunniv2.CallOpts, manager, id)
}

// GetRefund is a free data retrieval call binding the contract method 0x2e6ee7c3.
//
// Solidity: function getRefund(address manager, bytes32 id) view returns(uint256)
func (_Bunniv2 *Bunniv2CallerSession) GetRefund(manager common.Address, id [32]byte) (*big.Int, error) {
	return _Bunniv2.Contract.GetRefund(&_Bunniv2.CallOpts, manager, id)
}

// IsValidParams is a free data retrieval call binding the contract method 0x7ba36684.
//
// Solidity: function isValidParams(bytes hookParams) pure returns(bool)
func (_Bunniv2 *Bunniv2Caller) IsValidParams(opts *bind.CallOpts, hookParams []byte) (bool, error) {
	var out []interface{}
	err := _Bunniv2.contract.Call(opts, &out, "isValidParams", hookParams)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsValidParams is a free data retrieval call binding the contract method 0x7ba36684.
//
// Solidity: function isValidParams(bytes hookParams) pure returns(bool)
func (_Bunniv2 *Bunniv2Session) IsValidParams(hookParams []byte) (bool, error) {
	return _Bunniv2.Contract.IsValidParams(&_Bunniv2.CallOpts, hookParams)
}

// IsValidParams is a free data retrieval call binding the contract method 0x7ba36684.
//
// Solidity: function isValidParams(bytes hookParams) pure returns(bool)
func (_Bunniv2 *Bunniv2CallerSession) IsValidParams(hookParams []byte) (bool, error) {
	return _Bunniv2.Contract.IsValidParams(&_Bunniv2.CallOpts, hookParams)
}

// IsValidSignature is a free data retrieval call binding the contract method 0x1626ba7e.
//
// Solidity: function isValidSignature(bytes32 hash, bytes signature) view returns(bytes4 magicValue)
func (_Bunniv2 *Bunniv2Caller) IsValidSignature(opts *bind.CallOpts, hash [32]byte, signature []byte) ([4]byte, error) {
	var out []interface{}
	err := _Bunniv2.contract.Call(opts, &out, "isValidSignature", hash, signature)

	if err != nil {
		return *new([4]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([4]byte)).(*[4]byte)

	return out0, err

}

// IsValidSignature is a free data retrieval call binding the contract method 0x1626ba7e.
//
// Solidity: function isValidSignature(bytes32 hash, bytes signature) view returns(bytes4 magicValue)
func (_Bunniv2 *Bunniv2Session) IsValidSignature(hash [32]byte, signature []byte) ([4]byte, error) {
	return _Bunniv2.Contract.IsValidSignature(&_Bunniv2.CallOpts, hash, signature)
}

// IsValidSignature is a free data retrieval call binding the contract method 0x1626ba7e.
//
// Solidity: function isValidSignature(bytes32 hash, bytes signature) view returns(bytes4 magicValue)
func (_Bunniv2 *Bunniv2CallerSession) IsValidSignature(hash [32]byte, signature []byte) ([4]byte, error) {
	return _Bunniv2.Contract.IsValidSignature(&_Bunniv2.CallOpts, hash, signature)
}

// LdfStates is a free data retrieval call binding the contract method 0x3465dc0c.
//
// Solidity: function ldfStates(bytes32 id) view returns(bytes32)
func (_Bunniv2 *Bunniv2Caller) LdfStates(opts *bind.CallOpts, id [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Bunniv2.contract.Call(opts, &out, "ldfStates", id)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// LdfStates is a free data retrieval call binding the contract method 0x3465dc0c.
//
// Solidity: function ldfStates(bytes32 id) view returns(bytes32)
func (_Bunniv2 *Bunniv2Session) LdfStates(id [32]byte) ([32]byte, error) {
	return _Bunniv2.Contract.LdfStates(&_Bunniv2.CallOpts, id)
}

// LdfStates is a free data retrieval call binding the contract method 0x3465dc0c.
//
// Solidity: function ldfStates(bytes32 id) view returns(bytes32)
func (_Bunniv2 *Bunniv2CallerSession) LdfStates(id [32]byte) ([32]byte, error) {
	return _Bunniv2.Contract.LdfStates(&_Bunniv2.CallOpts, id)
}

// Observe is a free data retrieval call binding the contract method 0xf96f97f2.
//
// Solidity: function observe((address,address,uint24,int24,address) key, uint32[] secondsAgos) view returns(int56[] tickCumulatives)
func (_Bunniv2 *Bunniv2Caller) Observe(opts *bind.CallOpts, key PoolKey, secondsAgos []uint32) ([]*big.Int, error) {
	var out []interface{}
	err := _Bunniv2.contract.Call(opts, &out, "observe", key, secondsAgos)

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// Observe is a free data retrieval call binding the contract method 0xf96f97f2.
//
// Solidity: function observe((address,address,uint24,int24,address) key, uint32[] secondsAgos) view returns(int56[] tickCumulatives)
func (_Bunniv2 *Bunniv2Session) Observe(key PoolKey, secondsAgos []uint32) ([]*big.Int, error) {
	return _Bunniv2.Contract.Observe(&_Bunniv2.CallOpts, key, secondsAgos)
}

// Observe is a free data retrieval call binding the contract method 0xf96f97f2.
//
// Solidity: function observe((address,address,uint24,int24,address) key, uint32[] secondsAgos) view returns(int56[] tickCumulatives)
func (_Bunniv2 *Bunniv2CallerSession) Observe(key PoolKey, secondsAgos []uint32) ([]*big.Int, error) {
	return _Bunniv2.Contract.Observe(&_Bunniv2.CallOpts, key, secondsAgos)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address result)
func (_Bunniv2 *Bunniv2Caller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Bunniv2.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address result)
func (_Bunniv2 *Bunniv2Session) Owner() (common.Address, error) {
	return _Bunniv2.Contract.Owner(&_Bunniv2.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address result)
func (_Bunniv2 *Bunniv2CallerSession) Owner() (common.Address, error) {
	return _Bunniv2.Contract.Owner(&_Bunniv2.CallOpts)
}

// OwnershipHandoverExpiresAt is a free data retrieval call binding the contract method 0xfee81cf4.
//
// Solidity: function ownershipHandoverExpiresAt(address pendingOwner) view returns(uint256 result)
func (_Bunniv2 *Bunniv2Caller) OwnershipHandoverExpiresAt(opts *bind.CallOpts, pendingOwner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Bunniv2.contract.Call(opts, &out, "ownershipHandoverExpiresAt", pendingOwner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// OwnershipHandoverExpiresAt is a free data retrieval call binding the contract method 0xfee81cf4.
//
// Solidity: function ownershipHandoverExpiresAt(address pendingOwner) view returns(uint256 result)
func (_Bunniv2 *Bunniv2Session) OwnershipHandoverExpiresAt(pendingOwner common.Address) (*big.Int, error) {
	return _Bunniv2.Contract.OwnershipHandoverExpiresAt(&_Bunniv2.CallOpts, pendingOwner)
}

// OwnershipHandoverExpiresAt is a free data retrieval call binding the contract method 0xfee81cf4.
//
// Solidity: function ownershipHandoverExpiresAt(address pendingOwner) view returns(uint256 result)
func (_Bunniv2 *Bunniv2CallerSession) OwnershipHandoverExpiresAt(pendingOwner common.Address) (*big.Int, error) {
	return _Bunniv2.Contract.OwnershipHandoverExpiresAt(&_Bunniv2.CallOpts, pendingOwner)
}

// Slot0s is a free data retrieval call binding the contract method 0xfb20e17e.
//
// Solidity: function slot0s(bytes32 id) view returns(uint160 sqrtPriceX96, int24 tick, uint32 lastSwapTimestamp, uint32 lastSurgeTimestamp)
func (_Bunniv2 *Bunniv2Caller) Slot0s(opts *bind.CallOpts, id [32]byte) (struct {
	SqrtPriceX96       *big.Int
	Tick               *big.Int
	LastSwapTimestamp  uint32
	LastSurgeTimestamp uint32
}, error) {
	var out []interface{}
	err := _Bunniv2.contract.Call(opts, &out, "slot0s", id)

	outstruct := new(struct {
		SqrtPriceX96       *big.Int
		Tick               *big.Int
		LastSwapTimestamp  uint32
		LastSurgeTimestamp uint32
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.SqrtPriceX96 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Tick = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.LastSwapTimestamp = *abi.ConvertType(out[2], new(uint32)).(*uint32)
	outstruct.LastSurgeTimestamp = *abi.ConvertType(out[3], new(uint32)).(*uint32)

	return *outstruct, err

}

// Slot0s is a free data retrieval call binding the contract method 0xfb20e17e.
//
// Solidity: function slot0s(bytes32 id) view returns(uint160 sqrtPriceX96, int24 tick, uint32 lastSwapTimestamp, uint32 lastSurgeTimestamp)
func (_Bunniv2 *Bunniv2Session) Slot0s(id [32]byte) (struct {
	SqrtPriceX96       *big.Int
	Tick               *big.Int
	LastSwapTimestamp  uint32
	LastSurgeTimestamp uint32
}, error) {
	return _Bunniv2.Contract.Slot0s(&_Bunniv2.CallOpts, id)
}

// Slot0s is a free data retrieval call binding the contract method 0xfb20e17e.
//
// Solidity: function slot0s(bytes32 id) view returns(uint160 sqrtPriceX96, int24 tick, uint32 lastSwapTimestamp, uint32 lastSurgeTimestamp)
func (_Bunniv2 *Bunniv2CallerSession) Slot0s(id [32]byte) (struct {
	SqrtPriceX96       *big.Int
	Tick               *big.Int
	LastSwapTimestamp  uint32
	LastSurgeTimestamp uint32
}, error) {
	return _Bunniv2.Contract.Slot0s(&_Bunniv2.CallOpts, id)
}

// AfterInitialize is a paid mutator transaction binding the contract method 0x6fe7e6eb.
//
// Solidity: function afterInitialize(address caller, (address,address,uint24,int24,address) key, uint160 sqrtPriceX96, int24 tick) returns(bytes4)
func (_Bunniv2 *Bunniv2Transactor) AfterInitialize(opts *bind.TransactOpts, caller common.Address, key PoolKey, sqrtPriceX96 *big.Int, tick *big.Int) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "afterInitialize", caller, key, sqrtPriceX96, tick)
}

// AfterInitialize is a paid mutator transaction binding the contract method 0x6fe7e6eb.
//
// Solidity: function afterInitialize(address caller, (address,address,uint24,int24,address) key, uint160 sqrtPriceX96, int24 tick) returns(bytes4)
func (_Bunniv2 *Bunniv2Session) AfterInitialize(caller common.Address, key PoolKey, sqrtPriceX96 *big.Int, tick *big.Int) (*types.Transaction, error) {
	return _Bunniv2.Contract.AfterInitialize(&_Bunniv2.TransactOpts, caller, key, sqrtPriceX96, tick)
}

// AfterInitialize is a paid mutator transaction binding the contract method 0x6fe7e6eb.
//
// Solidity: function afterInitialize(address caller, (address,address,uint24,int24,address) key, uint160 sqrtPriceX96, int24 tick) returns(bytes4)
func (_Bunniv2 *Bunniv2TransactorSession) AfterInitialize(caller common.Address, key PoolKey, sqrtPriceX96 *big.Int, tick *big.Int) (*types.Transaction, error) {
	return _Bunniv2.Contract.AfterInitialize(&_Bunniv2.TransactOpts, caller, key, sqrtPriceX96, tick)
}

// BeforeSwap is a paid mutator transaction binding the contract method 0x575e24b4.
//
// Solidity: function beforeSwap(address sender, (address,address,uint24,int24,address) key, (bool,int256,uint160) params, bytes ) returns(bytes4, int256, uint24)
func (_Bunniv2 *Bunniv2Transactor) BeforeSwap(opts *bind.TransactOpts, sender common.Address, key PoolKey, params IPoolManagerSwapParams, arg3 []byte) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "beforeSwap", sender, key, params, arg3)
}

// BeforeSwap is a paid mutator transaction binding the contract method 0x575e24b4.
//
// Solidity: function beforeSwap(address sender, (address,address,uint24,int24,address) key, (bool,int256,uint160) params, bytes ) returns(bytes4, int256, uint24)
func (_Bunniv2 *Bunniv2Session) BeforeSwap(sender common.Address, key PoolKey, params IPoolManagerSwapParams, arg3 []byte) (*types.Transaction, error) {
	return _Bunniv2.Contract.BeforeSwap(&_Bunniv2.TransactOpts, sender, key, params, arg3)
}

// BeforeSwap is a paid mutator transaction binding the contract method 0x575e24b4.
//
// Solidity: function beforeSwap(address sender, (address,address,uint24,int24,address) key, (bool,int256,uint160) params, bytes ) returns(bytes4, int256, uint24)
func (_Bunniv2 *Bunniv2TransactorSession) BeforeSwap(sender common.Address, key PoolKey, params IPoolManagerSwapParams, arg3 []byte) (*types.Transaction, error) {
	return _Bunniv2.Contract.BeforeSwap(&_Bunniv2.TransactOpts, sender, key, params, arg3)
}

// Bid is a paid mutator transaction binding the contract method 0xa15dd0b5.
//
// Solidity: function bid(bytes32 id, address manager, bytes6 payload, uint128 rent, uint128 deposit) returns()
func (_Bunniv2 *Bunniv2Transactor) Bid(opts *bind.TransactOpts, id [32]byte, manager common.Address, payload [6]byte, rent *big.Int, deposit *big.Int) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "bid", id, manager, payload, rent, deposit)
}

// Bid is a paid mutator transaction binding the contract method 0xa15dd0b5.
//
// Solidity: function bid(bytes32 id, address manager, bytes6 payload, uint128 rent, uint128 deposit) returns()
func (_Bunniv2 *Bunniv2Session) Bid(id [32]byte, manager common.Address, payload [6]byte, rent *big.Int, deposit *big.Int) (*types.Transaction, error) {
	return _Bunniv2.Contract.Bid(&_Bunniv2.TransactOpts, id, manager, payload, rent, deposit)
}

// Bid is a paid mutator transaction binding the contract method 0xa15dd0b5.
//
// Solidity: function bid(bytes32 id, address manager, bytes6 payload, uint128 rent, uint128 deposit) returns()
func (_Bunniv2 *Bunniv2TransactorSession) Bid(id [32]byte, manager common.Address, payload [6]byte, rent *big.Int, deposit *big.Int) (*types.Transaction, error) {
	return _Bunniv2.Contract.Bid(&_Bunniv2.TransactOpts, id, manager, payload, rent, deposit)
}

// CancelOwnershipHandover is a paid mutator transaction binding the contract method 0x54d1f13d.
//
// Solidity: function cancelOwnershipHandover() payable returns()
func (_Bunniv2 *Bunniv2Transactor) CancelOwnershipHandover(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "cancelOwnershipHandover")
}

// CancelOwnershipHandover is a paid mutator transaction binding the contract method 0x54d1f13d.
//
// Solidity: function cancelOwnershipHandover() payable returns()
func (_Bunniv2 *Bunniv2Session) CancelOwnershipHandover() (*types.Transaction, error) {
	return _Bunniv2.Contract.CancelOwnershipHandover(&_Bunniv2.TransactOpts)
}

// CancelOwnershipHandover is a paid mutator transaction binding the contract method 0x54d1f13d.
//
// Solidity: function cancelOwnershipHandover() payable returns()
func (_Bunniv2 *Bunniv2TransactorSession) CancelOwnershipHandover() (*types.Transaction, error) {
	return _Bunniv2.Contract.CancelOwnershipHandover(&_Bunniv2.TransactOpts)
}

// ClaimFees is a paid mutator transaction binding the contract method 0x2dbfa735.
//
// Solidity: function claimFees(address currency, address recipient) returns(uint256 fees)
func (_Bunniv2 *Bunniv2Transactor) ClaimFees(opts *bind.TransactOpts, currency common.Address, recipient common.Address) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "claimFees", currency, recipient)
}

// ClaimFees is a paid mutator transaction binding the contract method 0x2dbfa735.
//
// Solidity: function claimFees(address currency, address recipient) returns(uint256 fees)
func (_Bunniv2 *Bunniv2Session) ClaimFees(currency common.Address, recipient common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.ClaimFees(&_Bunniv2.TransactOpts, currency, recipient)
}

// ClaimFees is a paid mutator transaction binding the contract method 0x2dbfa735.
//
// Solidity: function claimFees(address currency, address recipient) returns(uint256 fees)
func (_Bunniv2 *Bunniv2TransactorSession) ClaimFees(currency common.Address, recipient common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.ClaimFees(&_Bunniv2.TransactOpts, currency, recipient)
}

// ClaimProtocolFees is a paid mutator transaction binding the contract method 0x859049d3.
//
// Solidity: function claimProtocolFees(address[] currencyList) returns(uint256[] claimedAmounts)
func (_Bunniv2 *Bunniv2Transactor) ClaimProtocolFees(opts *bind.TransactOpts, currencyList []common.Address) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "claimProtocolFees", currencyList)
}

// ClaimProtocolFees is a paid mutator transaction binding the contract method 0x859049d3.
//
// Solidity: function claimProtocolFees(address[] currencyList) returns(uint256[] claimedAmounts)
func (_Bunniv2 *Bunniv2Session) ClaimProtocolFees(currencyList []common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.ClaimProtocolFees(&_Bunniv2.TransactOpts, currencyList)
}

// ClaimProtocolFees is a paid mutator transaction binding the contract method 0x859049d3.
//
// Solidity: function claimProtocolFees(address[] currencyList) returns(uint256[] claimedAmounts)
func (_Bunniv2 *Bunniv2TransactorSession) ClaimProtocolFees(currencyList []common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.ClaimProtocolFees(&_Bunniv2.TransactOpts, currencyList)
}

// ClaimRefund is a paid mutator transaction binding the contract method 0xc6b49688.
//
// Solidity: function claimRefund(bytes32 id, address recipient) returns(uint256 refund)
func (_Bunniv2 *Bunniv2Transactor) ClaimRefund(opts *bind.TransactOpts, id [32]byte, recipient common.Address) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "claimRefund", id, recipient)
}

// ClaimRefund is a paid mutator transaction binding the contract method 0xc6b49688.
//
// Solidity: function claimRefund(bytes32 id, address recipient) returns(uint256 refund)
func (_Bunniv2 *Bunniv2Session) ClaimRefund(id [32]byte, recipient common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.ClaimRefund(&_Bunniv2.TransactOpts, id, recipient)
}

// ClaimRefund is a paid mutator transaction binding the contract method 0xc6b49688.
//
// Solidity: function claimRefund(bytes32 id, address recipient) returns(uint256 refund)
func (_Bunniv2 *Bunniv2TransactorSession) ClaimRefund(id [32]byte, recipient common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.ClaimRefund(&_Bunniv2.TransactOpts, id, recipient)
}

// CompleteOwnershipHandover is a paid mutator transaction binding the contract method 0xf04e283e.
//
// Solidity: function completeOwnershipHandover(address pendingOwner) payable returns()
func (_Bunniv2 *Bunniv2Transactor) CompleteOwnershipHandover(opts *bind.TransactOpts, pendingOwner common.Address) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "completeOwnershipHandover", pendingOwner)
}

// CompleteOwnershipHandover is a paid mutator transaction binding the contract method 0xf04e283e.
//
// Solidity: function completeOwnershipHandover(address pendingOwner) payable returns()
func (_Bunniv2 *Bunniv2Session) CompleteOwnershipHandover(pendingOwner common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.CompleteOwnershipHandover(&_Bunniv2.TransactOpts, pendingOwner)
}

// CompleteOwnershipHandover is a paid mutator transaction binding the contract method 0xf04e283e.
//
// Solidity: function completeOwnershipHandover(address pendingOwner) payable returns()
func (_Bunniv2 *Bunniv2TransactorSession) CompleteOwnershipHandover(pendingOwner common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.CompleteOwnershipHandover(&_Bunniv2.TransactOpts, pendingOwner)
}

// CuratorClaimFees is a paid mutator transaction binding the contract method 0x729b45c1.
//
// Solidity: function curatorClaimFees((address,address,uint24,int24,address) key, address recipient) returns()
func (_Bunniv2 *Bunniv2Transactor) CuratorClaimFees(opts *bind.TransactOpts, key PoolKey, recipient common.Address) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "curatorClaimFees", key, recipient)
}

// CuratorClaimFees is a paid mutator transaction binding the contract method 0x729b45c1.
//
// Solidity: function curatorClaimFees((address,address,uint24,int24,address) key, address recipient) returns()
func (_Bunniv2 *Bunniv2Session) CuratorClaimFees(key PoolKey, recipient common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.CuratorClaimFees(&_Bunniv2.TransactOpts, key, recipient)
}

// CuratorClaimFees is a paid mutator transaction binding the contract method 0x729b45c1.
//
// Solidity: function curatorClaimFees((address,address,uint24,int24,address) key, address recipient) returns()
func (_Bunniv2 *Bunniv2TransactorSession) CuratorClaimFees(key PoolKey, recipient common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.CuratorClaimFees(&_Bunniv2.TransactOpts, key, recipient)
}

// CuratorSetFeeRate is a paid mutator transaction binding the contract method 0xfee2ba44.
//
// Solidity: function curatorSetFeeRate(bytes32 id, uint16 newFeeRate) returns()
func (_Bunniv2 *Bunniv2Transactor) CuratorSetFeeRate(opts *bind.TransactOpts, id [32]byte, newFeeRate uint16) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "curatorSetFeeRate", id, newFeeRate)
}

// CuratorSetFeeRate is a paid mutator transaction binding the contract method 0xfee2ba44.
//
// Solidity: function curatorSetFeeRate(bytes32 id, uint16 newFeeRate) returns()
func (_Bunniv2 *Bunniv2Session) CuratorSetFeeRate(id [32]byte, newFeeRate uint16) (*types.Transaction, error) {
	return _Bunniv2.Contract.CuratorSetFeeRate(&_Bunniv2.TransactOpts, id, newFeeRate)
}

// CuratorSetFeeRate is a paid mutator transaction binding the contract method 0xfee2ba44.
//
// Solidity: function curatorSetFeeRate(bytes32 id, uint16 newFeeRate) returns()
func (_Bunniv2 *Bunniv2TransactorSession) CuratorSetFeeRate(id [32]byte, newFeeRate uint16) (*types.Transaction, error) {
	return _Bunniv2.Contract.CuratorSetFeeRate(&_Bunniv2.TransactOpts, id, newFeeRate)
}

// DepositIntoBid is a paid mutator transaction binding the contract method 0x54d8d9a5.
//
// Solidity: function depositIntoBid(bytes32 id, uint128 amount, bool isTopBid) returns()
func (_Bunniv2 *Bunniv2Transactor) DepositIntoBid(opts *bind.TransactOpts, id [32]byte, amount *big.Int, isTopBid bool) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "depositIntoBid", id, amount, isTopBid)
}

// DepositIntoBid is a paid mutator transaction binding the contract method 0x54d8d9a5.
//
// Solidity: function depositIntoBid(bytes32 id, uint128 amount, bool isTopBid) returns()
func (_Bunniv2 *Bunniv2Session) DepositIntoBid(id [32]byte, amount *big.Int, isTopBid bool) (*types.Transaction, error) {
	return _Bunniv2.Contract.DepositIntoBid(&_Bunniv2.TransactOpts, id, amount, isTopBid)
}

// DepositIntoBid is a paid mutator transaction binding the contract method 0x54d8d9a5.
//
// Solidity: function depositIntoBid(bytes32 id, uint128 amount, bool isTopBid) returns()
func (_Bunniv2 *Bunniv2TransactorSession) DepositIntoBid(id [32]byte, amount *big.Int, isTopBid bool) (*types.Transaction, error) {
	return _Bunniv2.Contract.DepositIntoBid(&_Bunniv2.TransactOpts, id, amount, isTopBid)
}

// GetBidWrite is a paid mutator transaction binding the contract method 0xe3fe3452.
//
// Solidity: function getBidWrite(bytes32 id, bool isTopBid) returns((address,uint48,bytes6,uint128,uint128))
func (_Bunniv2 *Bunniv2Transactor) GetBidWrite(opts *bind.TransactOpts, id [32]byte, isTopBid bool) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "getBidWrite", id, isTopBid)
}

// GetBidWrite is a paid mutator transaction binding the contract method 0xe3fe3452.
//
// Solidity: function getBidWrite(bytes32 id, bool isTopBid) returns((address,uint48,bytes6,uint128,uint128))
func (_Bunniv2 *Bunniv2Session) GetBidWrite(id [32]byte, isTopBid bool) (*types.Transaction, error) {
	return _Bunniv2.Contract.GetBidWrite(&_Bunniv2.TransactOpts, id, isTopBid)
}

// GetBidWrite is a paid mutator transaction binding the contract method 0xe3fe3452.
//
// Solidity: function getBidWrite(bytes32 id, bool isTopBid) returns((address,uint48,bytes6,uint128,uint128))
func (_Bunniv2 *Bunniv2TransactorSession) GetBidWrite(id [32]byte, isTopBid bool) (*types.Transaction, error) {
	return _Bunniv2.Contract.GetBidWrite(&_Bunniv2.TransactOpts, id, isTopBid)
}

// GetRefundWrite is a paid mutator transaction binding the contract method 0xababb83c.
//
// Solidity: function getRefundWrite(address manager, bytes32 id) returns(uint256)
func (_Bunniv2 *Bunniv2Transactor) GetRefundWrite(opts *bind.TransactOpts, manager common.Address, id [32]byte) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "getRefundWrite", manager, id)
}

// GetRefundWrite is a paid mutator transaction binding the contract method 0xababb83c.
//
// Solidity: function getRefundWrite(address manager, bytes32 id) returns(uint256)
func (_Bunniv2 *Bunniv2Session) GetRefundWrite(manager common.Address, id [32]byte) (*types.Transaction, error) {
	return _Bunniv2.Contract.GetRefundWrite(&_Bunniv2.TransactOpts, manager, id)
}

// GetRefundWrite is a paid mutator transaction binding the contract method 0xababb83c.
//
// Solidity: function getRefundWrite(address manager, bytes32 id) returns(uint256)
func (_Bunniv2 *Bunniv2TransactorSession) GetRefundWrite(manager common.Address, id [32]byte) (*types.Transaction, error) {
	return _Bunniv2.Contract.GetRefundWrite(&_Bunniv2.TransactOpts, manager, id)
}

// IncreaseBidRent is a paid mutator transaction binding the contract method 0x1a362e40.
//
// Solidity: function increaseBidRent(bytes32 id, uint128 additionalRent, uint128 updatedDeposit, bool isTopBid, address withdrawRecipient) returns(uint128 amountDeposited, uint128 amountWithdrawn)
func (_Bunniv2 *Bunniv2Transactor) IncreaseBidRent(opts *bind.TransactOpts, id [32]byte, additionalRent *big.Int, updatedDeposit *big.Int, isTopBid bool, withdrawRecipient common.Address) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "increaseBidRent", id, additionalRent, updatedDeposit, isTopBid, withdrawRecipient)
}

// IncreaseBidRent is a paid mutator transaction binding the contract method 0x1a362e40.
//
// Solidity: function increaseBidRent(bytes32 id, uint128 additionalRent, uint128 updatedDeposit, bool isTopBid, address withdrawRecipient) returns(uint128 amountDeposited, uint128 amountWithdrawn)
func (_Bunniv2 *Bunniv2Session) IncreaseBidRent(id [32]byte, additionalRent *big.Int, updatedDeposit *big.Int, isTopBid bool, withdrawRecipient common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.IncreaseBidRent(&_Bunniv2.TransactOpts, id, additionalRent, updatedDeposit, isTopBid, withdrawRecipient)
}

// IncreaseBidRent is a paid mutator transaction binding the contract method 0x1a362e40.
//
// Solidity: function increaseBidRent(bytes32 id, uint128 additionalRent, uint128 updatedDeposit, bool isTopBid, address withdrawRecipient) returns(uint128 amountDeposited, uint128 amountWithdrawn)
func (_Bunniv2 *Bunniv2TransactorSession) IncreaseBidRent(id [32]byte, additionalRent *big.Int, updatedDeposit *big.Int, isTopBid bool, withdrawRecipient common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.IncreaseBidRent(&_Bunniv2.TransactOpts, id, additionalRent, updatedDeposit, isTopBid, withdrawRecipient)
}

// IncreaseCardinalityNext is a paid mutator transaction binding the contract method 0x2b02feaf.
//
// Solidity: function increaseCardinalityNext((address,address,uint24,int24,address) key, uint32 cardinalityNext) returns(uint32 cardinalityNextOld, uint32 cardinalityNextNew)
func (_Bunniv2 *Bunniv2Transactor) IncreaseCardinalityNext(opts *bind.TransactOpts, key PoolKey, cardinalityNext uint32) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "increaseCardinalityNext", key, cardinalityNext)
}

// IncreaseCardinalityNext is a paid mutator transaction binding the contract method 0x2b02feaf.
//
// Solidity: function increaseCardinalityNext((address,address,uint24,int24,address) key, uint32 cardinalityNext) returns(uint32 cardinalityNextOld, uint32 cardinalityNextNew)
func (_Bunniv2 *Bunniv2Session) IncreaseCardinalityNext(key PoolKey, cardinalityNext uint32) (*types.Transaction, error) {
	return _Bunniv2.Contract.IncreaseCardinalityNext(&_Bunniv2.TransactOpts, key, cardinalityNext)
}

// IncreaseCardinalityNext is a paid mutator transaction binding the contract method 0x2b02feaf.
//
// Solidity: function increaseCardinalityNext((address,address,uint24,int24,address) key, uint32 cardinalityNext) returns(uint32 cardinalityNextOld, uint32 cardinalityNextNew)
func (_Bunniv2 *Bunniv2TransactorSession) IncreaseCardinalityNext(key PoolKey, cardinalityNext uint32) (*types.Transaction, error) {
	return _Bunniv2.Contract.IncreaseCardinalityNext(&_Bunniv2.TransactOpts, key, cardinalityNext)
}

// RebalanceOrderHook is a paid mutator transaction binding the contract method 0x142a3c8c.
//
// Solidity: function rebalanceOrderHook(bool isPreHook, ((address,address,uint24,int24,address),(address,uint256),(address)) hookArgs) returns()
func (_Bunniv2 *Bunniv2Transactor) RebalanceOrderHook(opts *bind.TransactOpts, isPreHook bool, hookArgs IBunniHookRebalanceOrderHookArgs) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "rebalanceOrderHook", isPreHook, hookArgs)
}

// RebalanceOrderHook is a paid mutator transaction binding the contract method 0x142a3c8c.
//
// Solidity: function rebalanceOrderHook(bool isPreHook, ((address,address,uint24,int24,address),(address,uint256),(address)) hookArgs) returns()
func (_Bunniv2 *Bunniv2Session) RebalanceOrderHook(isPreHook bool, hookArgs IBunniHookRebalanceOrderHookArgs) (*types.Transaction, error) {
	return _Bunniv2.Contract.RebalanceOrderHook(&_Bunniv2.TransactOpts, isPreHook, hookArgs)
}

// RebalanceOrderHook is a paid mutator transaction binding the contract method 0x142a3c8c.
//
// Solidity: function rebalanceOrderHook(bool isPreHook, ((address,address,uint24,int24,address),(address,uint256),(address)) hookArgs) returns()
func (_Bunniv2 *Bunniv2TransactorSession) RebalanceOrderHook(isPreHook bool, hookArgs IBunniHookRebalanceOrderHookArgs) (*types.Transaction, error) {
	return _Bunniv2.Contract.RebalanceOrderHook(&_Bunniv2.TransactOpts, isPreHook, hookArgs)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() payable returns()
func (_Bunniv2 *Bunniv2Transactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() payable returns()
func (_Bunniv2 *Bunniv2Session) RenounceOwnership() (*types.Transaction, error) {
	return _Bunniv2.Contract.RenounceOwnership(&_Bunniv2.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() payable returns()
func (_Bunniv2 *Bunniv2TransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Bunniv2.Contract.RenounceOwnership(&_Bunniv2.TransactOpts)
}

// RequestOwnershipHandover is a paid mutator transaction binding the contract method 0x25692962.
//
// Solidity: function requestOwnershipHandover() payable returns()
func (_Bunniv2 *Bunniv2Transactor) RequestOwnershipHandover(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "requestOwnershipHandover")
}

// RequestOwnershipHandover is a paid mutator transaction binding the contract method 0x25692962.
//
// Solidity: function requestOwnershipHandover() payable returns()
func (_Bunniv2 *Bunniv2Session) RequestOwnershipHandover() (*types.Transaction, error) {
	return _Bunniv2.Contract.RequestOwnershipHandover(&_Bunniv2.TransactOpts)
}

// RequestOwnershipHandover is a paid mutator transaction binding the contract method 0x25692962.
//
// Solidity: function requestOwnershipHandover() payable returns()
func (_Bunniv2 *Bunniv2TransactorSession) RequestOwnershipHandover() (*types.Transaction, error) {
	return _Bunniv2.Contract.RequestOwnershipHandover(&_Bunniv2.TransactOpts)
}

// ScheduleKChange is a paid mutator transaction binding the contract method 0x214ada3e.
//
// Solidity: function scheduleKChange(uint48 newK, uint160 activeBlock) returns()
func (_Bunniv2 *Bunniv2Transactor) ScheduleKChange(opts *bind.TransactOpts, newK *big.Int, activeBlock *big.Int) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "scheduleKChange", newK, activeBlock)
}

// ScheduleKChange is a paid mutator transaction binding the contract method 0x214ada3e.
//
// Solidity: function scheduleKChange(uint48 newK, uint160 activeBlock) returns()
func (_Bunniv2 *Bunniv2Session) ScheduleKChange(newK *big.Int, activeBlock *big.Int) (*types.Transaction, error) {
	return _Bunniv2.Contract.ScheduleKChange(&_Bunniv2.TransactOpts, newK, activeBlock)
}

// ScheduleKChange is a paid mutator transaction binding the contract method 0x214ada3e.
//
// Solidity: function scheduleKChange(uint48 newK, uint160 activeBlock) returns()
func (_Bunniv2 *Bunniv2TransactorSession) ScheduleKChange(newK *big.Int, activeBlock *big.Int) (*types.Transaction, error) {
	return _Bunniv2.Contract.ScheduleKChange(&_Bunniv2.TransactOpts, newK, activeBlock)
}

// SetBidPayload is a paid mutator transaction binding the contract method 0x0be21ab5.
//
// Solidity: function setBidPayload(bytes32 id, bytes6 payload, bool isTopBid) returns()
func (_Bunniv2 *Bunniv2Transactor) SetBidPayload(opts *bind.TransactOpts, id [32]byte, payload [6]byte, isTopBid bool) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "setBidPayload", id, payload, isTopBid)
}

// SetBidPayload is a paid mutator transaction binding the contract method 0x0be21ab5.
//
// Solidity: function setBidPayload(bytes32 id, bytes6 payload, bool isTopBid) returns()
func (_Bunniv2 *Bunniv2Session) SetBidPayload(id [32]byte, payload [6]byte, isTopBid bool) (*types.Transaction, error) {
	return _Bunniv2.Contract.SetBidPayload(&_Bunniv2.TransactOpts, id, payload, isTopBid)
}

// SetBidPayload is a paid mutator transaction binding the contract method 0x0be21ab5.
//
// Solidity: function setBidPayload(bytes32 id, bytes6 payload, bool isTopBid) returns()
func (_Bunniv2 *Bunniv2TransactorSession) SetBidPayload(id [32]byte, payload [6]byte, isTopBid bool) (*types.Transaction, error) {
	return _Bunniv2.Contract.SetBidPayload(&_Bunniv2.TransactOpts, id, payload, isTopBid)
}

// SetHookFeeModifier is a paid mutator transaction binding the contract method 0xe6333979.
//
// Solidity: function setHookFeeModifier(uint32 newHookFeeModifier) returns()
func (_Bunniv2 *Bunniv2Transactor) SetHookFeeModifier(opts *bind.TransactOpts, newHookFeeModifier uint32) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "setHookFeeModifier", newHookFeeModifier)
}

// SetHookFeeModifier is a paid mutator transaction binding the contract method 0xe6333979.
//
// Solidity: function setHookFeeModifier(uint32 newHookFeeModifier) returns()
func (_Bunniv2 *Bunniv2Session) SetHookFeeModifier(newHookFeeModifier uint32) (*types.Transaction, error) {
	return _Bunniv2.Contract.SetHookFeeModifier(&_Bunniv2.TransactOpts, newHookFeeModifier)
}

// SetHookFeeModifier is a paid mutator transaction binding the contract method 0xe6333979.
//
// Solidity: function setHookFeeModifier(uint32 newHookFeeModifier) returns()
func (_Bunniv2 *Bunniv2TransactorSession) SetHookFeeModifier(newHookFeeModifier uint32) (*types.Transaction, error) {
	return _Bunniv2.Contract.SetHookFeeModifier(&_Bunniv2.TransactOpts, newHookFeeModifier)
}

// SetHookFeeRecipient is a paid mutator transaction binding the contract method 0x23c0571b.
//
// Solidity: function setHookFeeRecipient(address newHookFeeRecipient) returns()
func (_Bunniv2 *Bunniv2Transactor) SetHookFeeRecipient(opts *bind.TransactOpts, newHookFeeRecipient common.Address) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "setHookFeeRecipient", newHookFeeRecipient)
}

// SetHookFeeRecipient is a paid mutator transaction binding the contract method 0x23c0571b.
//
// Solidity: function setHookFeeRecipient(address newHookFeeRecipient) returns()
func (_Bunniv2 *Bunniv2Session) SetHookFeeRecipient(newHookFeeRecipient common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.SetHookFeeRecipient(&_Bunniv2.TransactOpts, newHookFeeRecipient)
}

// SetHookFeeRecipient is a paid mutator transaction binding the contract method 0x23c0571b.
//
// Solidity: function setHookFeeRecipient(address newHookFeeRecipient) returns()
func (_Bunniv2 *Bunniv2TransactorSession) SetHookFeeRecipient(newHookFeeRecipient common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.SetHookFeeRecipient(&_Bunniv2.TransactOpts, newHookFeeRecipient)
}

// SetWithdrawalUnblocked is a paid mutator transaction binding the contract method 0x89a6dccf.
//
// Solidity: function setWithdrawalUnblocked(bytes32 id, bool unblocked) returns()
func (_Bunniv2 *Bunniv2Transactor) SetWithdrawalUnblocked(opts *bind.TransactOpts, id [32]byte, unblocked bool) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "setWithdrawalUnblocked", id, unblocked)
}

// SetWithdrawalUnblocked is a paid mutator transaction binding the contract method 0x89a6dccf.
//
// Solidity: function setWithdrawalUnblocked(bytes32 id, bool unblocked) returns()
func (_Bunniv2 *Bunniv2Session) SetWithdrawalUnblocked(id [32]byte, unblocked bool) (*types.Transaction, error) {
	return _Bunniv2.Contract.SetWithdrawalUnblocked(&_Bunniv2.TransactOpts, id, unblocked)
}

// SetWithdrawalUnblocked is a paid mutator transaction binding the contract method 0x89a6dccf.
//
// Solidity: function setWithdrawalUnblocked(bytes32 id, bool unblocked) returns()
func (_Bunniv2 *Bunniv2TransactorSession) SetWithdrawalUnblocked(id [32]byte, unblocked bool) (*types.Transaction, error) {
	return _Bunniv2.Contract.SetWithdrawalUnblocked(&_Bunniv2.TransactOpts, id, unblocked)
}

// SetZone is a paid mutator transaction binding the contract method 0x531f4dd2.
//
// Solidity: function setZone(address zone) returns()
func (_Bunniv2 *Bunniv2Transactor) SetZone(opts *bind.TransactOpts, zone common.Address) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "setZone", zone)
}

// SetZone is a paid mutator transaction binding the contract method 0x531f4dd2.
//
// Solidity: function setZone(address zone) returns()
func (_Bunniv2 *Bunniv2Session) SetZone(zone common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.SetZone(&_Bunniv2.TransactOpts, zone)
}

// SetZone is a paid mutator transaction binding the contract method 0x531f4dd2.
//
// Solidity: function setZone(address zone) returns()
func (_Bunniv2 *Bunniv2TransactorSession) SetZone(zone common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.SetZone(&_Bunniv2.TransactOpts, zone)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) payable returns()
func (_Bunniv2 *Bunniv2Transactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) payable returns()
func (_Bunniv2 *Bunniv2Session) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.TransferOwnership(&_Bunniv2.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) payable returns()
func (_Bunniv2 *Bunniv2TransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Bunniv2.Contract.TransferOwnership(&_Bunniv2.TransactOpts, newOwner)
}

// UnlockCallback is a paid mutator transaction binding the contract method 0x91dd7346.
//
// Solidity: function unlockCallback(bytes data) returns(bytes)
func (_Bunniv2 *Bunniv2Transactor) UnlockCallback(opts *bind.TransactOpts, data []byte) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "unlockCallback", data)
}

// UnlockCallback is a paid mutator transaction binding the contract method 0x91dd7346.
//
// Solidity: function unlockCallback(bytes data) returns(bytes)
func (_Bunniv2 *Bunniv2Session) UnlockCallback(data []byte) (*types.Transaction, error) {
	return _Bunniv2.Contract.UnlockCallback(&_Bunniv2.TransactOpts, data)
}

// UnlockCallback is a paid mutator transaction binding the contract method 0x91dd7346.
//
// Solidity: function unlockCallback(bytes data) returns(bytes)
func (_Bunniv2 *Bunniv2TransactorSession) UnlockCallback(data []byte) (*types.Transaction, error) {
	return _Bunniv2.Contract.UnlockCallback(&_Bunniv2.TransactOpts, data)
}

// UpdateLdfState is a paid mutator transaction binding the contract method 0x13679355.
//
// Solidity: function updateLdfState(bytes32 id, bytes32 newState) returns()
func (_Bunniv2 *Bunniv2Transactor) UpdateLdfState(opts *bind.TransactOpts, id [32]byte, newState [32]byte) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "updateLdfState", id, newState)
}

// UpdateLdfState is a paid mutator transaction binding the contract method 0x13679355.
//
// Solidity: function updateLdfState(bytes32 id, bytes32 newState) returns()
func (_Bunniv2 *Bunniv2Session) UpdateLdfState(id [32]byte, newState [32]byte) (*types.Transaction, error) {
	return _Bunniv2.Contract.UpdateLdfState(&_Bunniv2.TransactOpts, id, newState)
}

// UpdateLdfState is a paid mutator transaction binding the contract method 0x13679355.
//
// Solidity: function updateLdfState(bytes32 id, bytes32 newState) returns()
func (_Bunniv2 *Bunniv2TransactorSession) UpdateLdfState(id [32]byte, newState [32]byte) (*types.Transaction, error) {
	return _Bunniv2.Contract.UpdateLdfState(&_Bunniv2.TransactOpts, id, newState)
}

// UpdateStateMachine is a paid mutator transaction binding the contract method 0x15dcc2e0.
//
// Solidity: function updateStateMachine(bytes32 id) returns()
func (_Bunniv2 *Bunniv2Transactor) UpdateStateMachine(opts *bind.TransactOpts, id [32]byte) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "updateStateMachine", id)
}

// UpdateStateMachine is a paid mutator transaction binding the contract method 0x15dcc2e0.
//
// Solidity: function updateStateMachine(bytes32 id) returns()
func (_Bunniv2 *Bunniv2Session) UpdateStateMachine(id [32]byte) (*types.Transaction, error) {
	return _Bunniv2.Contract.UpdateStateMachine(&_Bunniv2.TransactOpts, id)
}

// UpdateStateMachine is a paid mutator transaction binding the contract method 0x15dcc2e0.
//
// Solidity: function updateStateMachine(bytes32 id) returns()
func (_Bunniv2 *Bunniv2TransactorSession) UpdateStateMachine(id [32]byte) (*types.Transaction, error) {
	return _Bunniv2.Contract.UpdateStateMachine(&_Bunniv2.TransactOpts, id)
}

// WithdrawFromBid is a paid mutator transaction binding the contract method 0x1254f2fe.
//
// Solidity: function withdrawFromBid(bytes32 id, uint128 amount, address recipient, bool isTopBid) returns()
func (_Bunniv2 *Bunniv2Transactor) WithdrawFromBid(opts *bind.TransactOpts, id [32]byte, amount *big.Int, recipient common.Address, isTopBid bool) (*types.Transaction, error) {
	return _Bunniv2.contract.Transact(opts, "withdrawFromBid", id, amount, recipient, isTopBid)
}

// WithdrawFromBid is a paid mutator transaction binding the contract method 0x1254f2fe.
//
// Solidity: function withdrawFromBid(bytes32 id, uint128 amount, address recipient, bool isTopBid) returns()
func (_Bunniv2 *Bunniv2Session) WithdrawFromBid(id [32]byte, amount *big.Int, recipient common.Address, isTopBid bool) (*types.Transaction, error) {
	return _Bunniv2.Contract.WithdrawFromBid(&_Bunniv2.TransactOpts, id, amount, recipient, isTopBid)
}

// WithdrawFromBid is a paid mutator transaction binding the contract method 0x1254f2fe.
//
// Solidity: function withdrawFromBid(bytes32 id, uint128 amount, address recipient, bool isTopBid) returns()
func (_Bunniv2 *Bunniv2TransactorSession) WithdrawFromBid(id [32]byte, amount *big.Int, recipient common.Address, isTopBid bool) (*types.Transaction, error) {
	return _Bunniv2.Contract.WithdrawFromBid(&_Bunniv2.TransactOpts, id, amount, recipient, isTopBid)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Bunniv2 *Bunniv2Transactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Bunniv2.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Bunniv2 *Bunniv2Session) Receive() (*types.Transaction, error) {
	return _Bunniv2.Contract.Receive(&_Bunniv2.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Bunniv2 *Bunniv2TransactorSession) Receive() (*types.Transaction, error) {
	return _Bunniv2.Contract.Receive(&_Bunniv2.TransactOpts)
}

// Bunniv2ClaimFeesIterator is returned from FilterClaimFees and is used to iterate over the raw logs and unpacked data for ClaimFees events raised by the Bunniv2 contract.
type Bunniv2ClaimFeesIterator struct {
	Event *Bunniv2ClaimFees // Event containing the contract specifics and raw log

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
func (it *Bunniv2ClaimFeesIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2ClaimFees)
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
		it.Event = new(Bunniv2ClaimFees)
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
func (it *Bunniv2ClaimFeesIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2ClaimFeesIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2ClaimFees represents a ClaimFees event raised by the Bunniv2 contract.
type Bunniv2ClaimFees struct {
	Currency  common.Address
	Manager   common.Address
	Recipient common.Address
	Fees      *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterClaimFees is a free log retrieval operation binding the contract event 0xee447878094bcc87bd81b55e4831b5e9e3d095866fce10f4f7e4caa64ea7a558.
//
// Solidity: event ClaimFees(address indexed currency, address indexed manager, address indexed recipient, uint256 fees)
func (_Bunniv2 *Bunniv2Filterer) FilterClaimFees(opts *bind.FilterOpts, currency []common.Address, manager []common.Address, recipient []common.Address) (*Bunniv2ClaimFeesIterator, error) {

	var currencyRule []interface{}
	for _, currencyItem := range currency {
		currencyRule = append(currencyRule, currencyItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "ClaimFees", currencyRule, managerRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2ClaimFeesIterator{contract: _Bunniv2.contract, event: "ClaimFees", logs: logs, sub: sub}, nil
}

// WatchClaimFees is a free log subscription operation binding the contract event 0xee447878094bcc87bd81b55e4831b5e9e3d095866fce10f4f7e4caa64ea7a558.
//
// Solidity: event ClaimFees(address indexed currency, address indexed manager, address indexed recipient, uint256 fees)
func (_Bunniv2 *Bunniv2Filterer) WatchClaimFees(opts *bind.WatchOpts, sink chan<- *Bunniv2ClaimFees, currency []common.Address, manager []common.Address, recipient []common.Address) (event.Subscription, error) {

	var currencyRule []interface{}
	for _, currencyItem := range currency {
		currencyRule = append(currencyRule, currencyItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "ClaimFees", currencyRule, managerRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2ClaimFees)
				if err := _Bunniv2.contract.UnpackLog(event, "ClaimFees", log); err != nil {
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

// ParseClaimFees is a log parse operation binding the contract event 0xee447878094bcc87bd81b55e4831b5e9e3d095866fce10f4f7e4caa64ea7a558.
//
// Solidity: event ClaimFees(address indexed currency, address indexed manager, address indexed recipient, uint256 fees)
func (_Bunniv2 *Bunniv2Filterer) ParseClaimFees(log types.Log) (*Bunniv2ClaimFees, error) {
	event := new(Bunniv2ClaimFees)
	if err := _Bunniv2.contract.UnpackLog(event, "ClaimFees", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2ClaimProtocolFeesIterator is returned from FilterClaimProtocolFees and is used to iterate over the raw logs and unpacked data for ClaimProtocolFees events raised by the Bunniv2 contract.
type Bunniv2ClaimProtocolFeesIterator struct {
	Event *Bunniv2ClaimProtocolFees // Event containing the contract specifics and raw log

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
func (it *Bunniv2ClaimProtocolFeesIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2ClaimProtocolFees)
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
		it.Event = new(Bunniv2ClaimProtocolFees)
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
func (it *Bunniv2ClaimProtocolFeesIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2ClaimProtocolFeesIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2ClaimProtocolFees represents a ClaimProtocolFees event raised by the Bunniv2 contract.
type Bunniv2ClaimProtocolFees struct {
	CurrencyList []common.Address
	Recipient    common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterClaimProtocolFees is a free log retrieval operation binding the contract event 0x343ec735634cde6c74e128c9997c907fee57a10d2d60ddd6c6edd5b3deae4363.
//
// Solidity: event ClaimProtocolFees(address[] currencyList, address indexed recipient)
func (_Bunniv2 *Bunniv2Filterer) FilterClaimProtocolFees(opts *bind.FilterOpts, recipient []common.Address) (*Bunniv2ClaimProtocolFeesIterator, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "ClaimProtocolFees", recipientRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2ClaimProtocolFeesIterator{contract: _Bunniv2.contract, event: "ClaimProtocolFees", logs: logs, sub: sub}, nil
}

// WatchClaimProtocolFees is a free log subscription operation binding the contract event 0x343ec735634cde6c74e128c9997c907fee57a10d2d60ddd6c6edd5b3deae4363.
//
// Solidity: event ClaimProtocolFees(address[] currencyList, address indexed recipient)
func (_Bunniv2 *Bunniv2Filterer) WatchClaimProtocolFees(opts *bind.WatchOpts, sink chan<- *Bunniv2ClaimProtocolFees, recipient []common.Address) (event.Subscription, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "ClaimProtocolFees", recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2ClaimProtocolFees)
				if err := _Bunniv2.contract.UnpackLog(event, "ClaimProtocolFees", log); err != nil {
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

// ParseClaimProtocolFees is a log parse operation binding the contract event 0x343ec735634cde6c74e128c9997c907fee57a10d2d60ddd6c6edd5b3deae4363.
//
// Solidity: event ClaimProtocolFees(address[] currencyList, address indexed recipient)
func (_Bunniv2 *Bunniv2Filterer) ParseClaimProtocolFees(log types.Log) (*Bunniv2ClaimProtocolFees, error) {
	event := new(Bunniv2ClaimProtocolFees)
	if err := _Bunniv2.contract.UnpackLog(event, "ClaimProtocolFees", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2ClaimRefundIterator is returned from FilterClaimRefund and is used to iterate over the raw logs and unpacked data for ClaimRefund events raised by the Bunniv2 contract.
type Bunniv2ClaimRefundIterator struct {
	Event *Bunniv2ClaimRefund // Event containing the contract specifics and raw log

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
func (it *Bunniv2ClaimRefundIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2ClaimRefund)
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
		it.Event = new(Bunniv2ClaimRefund)
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
func (it *Bunniv2ClaimRefundIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2ClaimRefundIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2ClaimRefund represents a ClaimRefund event raised by the Bunniv2 contract.
type Bunniv2ClaimRefund struct {
	Id        [32]byte
	Manager   common.Address
	Recipient common.Address
	Refund    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterClaimRefund is a free log retrieval operation binding the contract event 0xb50cab7f3cadbfcd988880bdb754510d79ca993ead3b8c09b58559769f6d9206.
//
// Solidity: event ClaimRefund(bytes32 indexed id, address indexed manager, address indexed recipient, uint256 refund)
func (_Bunniv2 *Bunniv2Filterer) FilterClaimRefund(opts *bind.FilterOpts, id [][32]byte, manager []common.Address, recipient []common.Address) (*Bunniv2ClaimRefundIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "ClaimRefund", idRule, managerRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2ClaimRefundIterator{contract: _Bunniv2.contract, event: "ClaimRefund", logs: logs, sub: sub}, nil
}

// WatchClaimRefund is a free log subscription operation binding the contract event 0xb50cab7f3cadbfcd988880bdb754510d79ca993ead3b8c09b58559769f6d9206.
//
// Solidity: event ClaimRefund(bytes32 indexed id, address indexed manager, address indexed recipient, uint256 refund)
func (_Bunniv2 *Bunniv2Filterer) WatchClaimRefund(opts *bind.WatchOpts, sink chan<- *Bunniv2ClaimRefund, id [][32]byte, manager []common.Address, recipient []common.Address) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "ClaimRefund", idRule, managerRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2ClaimRefund)
				if err := _Bunniv2.contract.UnpackLog(event, "ClaimRefund", log); err != nil {
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

// ParseClaimRefund is a log parse operation binding the contract event 0xb50cab7f3cadbfcd988880bdb754510d79ca993ead3b8c09b58559769f6d9206.
//
// Solidity: event ClaimRefund(bytes32 indexed id, address indexed manager, address indexed recipient, uint256 refund)
func (_Bunniv2 *Bunniv2Filterer) ParseClaimRefund(log types.Log) (*Bunniv2ClaimRefund, error) {
	event := new(Bunniv2ClaimRefund)
	if err := _Bunniv2.contract.UnpackLog(event, "ClaimRefund", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2CuratorClaimFeesIterator is returned from FilterCuratorClaimFees and is used to iterate over the raw logs and unpacked data for CuratorClaimFees events raised by the Bunniv2 contract.
type Bunniv2CuratorClaimFeesIterator struct {
	Event *Bunniv2CuratorClaimFees // Event containing the contract specifics and raw log

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
func (it *Bunniv2CuratorClaimFeesIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2CuratorClaimFees)
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
		it.Event = new(Bunniv2CuratorClaimFees)
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
func (it *Bunniv2CuratorClaimFeesIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2CuratorClaimFeesIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2CuratorClaimFees represents a CuratorClaimFees event raised by the Bunniv2 contract.
type Bunniv2CuratorClaimFees struct {
	Id         [32]byte
	Recipient  common.Address
	FeeAmount0 *big.Int
	FeeAmount1 *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterCuratorClaimFees is a free log retrieval operation binding the contract event 0x6af496ab2167a3cd8281d42f68c2b5789b2d3e1c627eda4ad62f7d63d3d35029.
//
// Solidity: event CuratorClaimFees(bytes32 indexed id, address indexed recipient, uint256 feeAmount0, uint256 feeAmount1)
func (_Bunniv2 *Bunniv2Filterer) FilterCuratorClaimFees(opts *bind.FilterOpts, id [][32]byte, recipient []common.Address) (*Bunniv2CuratorClaimFeesIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "CuratorClaimFees", idRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2CuratorClaimFeesIterator{contract: _Bunniv2.contract, event: "CuratorClaimFees", logs: logs, sub: sub}, nil
}

// WatchCuratorClaimFees is a free log subscription operation binding the contract event 0x6af496ab2167a3cd8281d42f68c2b5789b2d3e1c627eda4ad62f7d63d3d35029.
//
// Solidity: event CuratorClaimFees(bytes32 indexed id, address indexed recipient, uint256 feeAmount0, uint256 feeAmount1)
func (_Bunniv2 *Bunniv2Filterer) WatchCuratorClaimFees(opts *bind.WatchOpts, sink chan<- *Bunniv2CuratorClaimFees, id [][32]byte, recipient []common.Address) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "CuratorClaimFees", idRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2CuratorClaimFees)
				if err := _Bunniv2.contract.UnpackLog(event, "CuratorClaimFees", log); err != nil {
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

// ParseCuratorClaimFees is a log parse operation binding the contract event 0x6af496ab2167a3cd8281d42f68c2b5789b2d3e1c627eda4ad62f7d63d3d35029.
//
// Solidity: event CuratorClaimFees(bytes32 indexed id, address indexed recipient, uint256 feeAmount0, uint256 feeAmount1)
func (_Bunniv2 *Bunniv2Filterer) ParseCuratorClaimFees(log types.Log) (*Bunniv2CuratorClaimFees, error) {
	event := new(Bunniv2CuratorClaimFees)
	if err := _Bunniv2.contract.UnpackLog(event, "CuratorClaimFees", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2CuratorSetFeeRateIterator is returned from FilterCuratorSetFeeRate and is used to iterate over the raw logs and unpacked data for CuratorSetFeeRate events raised by the Bunniv2 contract.
type Bunniv2CuratorSetFeeRateIterator struct {
	Event *Bunniv2CuratorSetFeeRate // Event containing the contract specifics and raw log

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
func (it *Bunniv2CuratorSetFeeRateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2CuratorSetFeeRate)
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
		it.Event = new(Bunniv2CuratorSetFeeRate)
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
func (it *Bunniv2CuratorSetFeeRateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2CuratorSetFeeRateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2CuratorSetFeeRate represents a CuratorSetFeeRate event raised by the Bunniv2 contract.
type Bunniv2CuratorSetFeeRate struct {
	Id         [32]byte
	NewFeeRate uint16
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterCuratorSetFeeRate is a free log retrieval operation binding the contract event 0x790a677be3faa0bace16b44a99f888bdeb9169c827ca24d85f2843108a95863f.
//
// Solidity: event CuratorSetFeeRate(bytes32 indexed id, uint16 indexed newFeeRate)
func (_Bunniv2 *Bunniv2Filterer) FilterCuratorSetFeeRate(opts *bind.FilterOpts, id [][32]byte, newFeeRate []uint16) (*Bunniv2CuratorSetFeeRateIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var newFeeRateRule []interface{}
	for _, newFeeRateItem := range newFeeRate {
		newFeeRateRule = append(newFeeRateRule, newFeeRateItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "CuratorSetFeeRate", idRule, newFeeRateRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2CuratorSetFeeRateIterator{contract: _Bunniv2.contract, event: "CuratorSetFeeRate", logs: logs, sub: sub}, nil
}

// WatchCuratorSetFeeRate is a free log subscription operation binding the contract event 0x790a677be3faa0bace16b44a99f888bdeb9169c827ca24d85f2843108a95863f.
//
// Solidity: event CuratorSetFeeRate(bytes32 indexed id, uint16 indexed newFeeRate)
func (_Bunniv2 *Bunniv2Filterer) WatchCuratorSetFeeRate(opts *bind.WatchOpts, sink chan<- *Bunniv2CuratorSetFeeRate, id [][32]byte, newFeeRate []uint16) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var newFeeRateRule []interface{}
	for _, newFeeRateItem := range newFeeRate {
		newFeeRateRule = append(newFeeRateRule, newFeeRateItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "CuratorSetFeeRate", idRule, newFeeRateRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2CuratorSetFeeRate)
				if err := _Bunniv2.contract.UnpackLog(event, "CuratorSetFeeRate", log); err != nil {
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

// ParseCuratorSetFeeRate is a log parse operation binding the contract event 0x790a677be3faa0bace16b44a99f888bdeb9169c827ca24d85f2843108a95863f.
//
// Solidity: event CuratorSetFeeRate(bytes32 indexed id, uint16 indexed newFeeRate)
func (_Bunniv2 *Bunniv2Filterer) ParseCuratorSetFeeRate(log types.Log) (*Bunniv2CuratorSetFeeRate, error) {
	event := new(Bunniv2CuratorSetFeeRate)
	if err := _Bunniv2.contract.UnpackLog(event, "CuratorSetFeeRate", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2DepositIntoNextBidIterator is returned from FilterDepositIntoNextBid and is used to iterate over the raw logs and unpacked data for DepositIntoNextBid events raised by the Bunniv2 contract.
type Bunniv2DepositIntoNextBidIterator struct {
	Event *Bunniv2DepositIntoNextBid // Event containing the contract specifics and raw log

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
func (it *Bunniv2DepositIntoNextBidIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2DepositIntoNextBid)
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
		it.Event = new(Bunniv2DepositIntoNextBid)
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
func (it *Bunniv2DepositIntoNextBidIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2DepositIntoNextBidIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2DepositIntoNextBid represents a DepositIntoNextBid event raised by the Bunniv2 contract.
type Bunniv2DepositIntoNextBid struct {
	Id      [32]byte
	Manager common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterDepositIntoNextBid is a free log retrieval operation binding the contract event 0x018ca21103a82dec7bc9c112d22fe77a99b58608eac0ed93262243e30fe1da15.
//
// Solidity: event DepositIntoNextBid(bytes32 indexed id, address indexed manager, uint128 amount)
func (_Bunniv2 *Bunniv2Filterer) FilterDepositIntoNextBid(opts *bind.FilterOpts, id [][32]byte, manager []common.Address) (*Bunniv2DepositIntoNextBidIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "DepositIntoNextBid", idRule, managerRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2DepositIntoNextBidIterator{contract: _Bunniv2.contract, event: "DepositIntoNextBid", logs: logs, sub: sub}, nil
}

// WatchDepositIntoNextBid is a free log subscription operation binding the contract event 0x018ca21103a82dec7bc9c112d22fe77a99b58608eac0ed93262243e30fe1da15.
//
// Solidity: event DepositIntoNextBid(bytes32 indexed id, address indexed manager, uint128 amount)
func (_Bunniv2 *Bunniv2Filterer) WatchDepositIntoNextBid(opts *bind.WatchOpts, sink chan<- *Bunniv2DepositIntoNextBid, id [][32]byte, manager []common.Address) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "DepositIntoNextBid", idRule, managerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2DepositIntoNextBid)
				if err := _Bunniv2.contract.UnpackLog(event, "DepositIntoNextBid", log); err != nil {
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

// ParseDepositIntoNextBid is a log parse operation binding the contract event 0x018ca21103a82dec7bc9c112d22fe77a99b58608eac0ed93262243e30fe1da15.
//
// Solidity: event DepositIntoNextBid(bytes32 indexed id, address indexed manager, uint128 amount)
func (_Bunniv2 *Bunniv2Filterer) ParseDepositIntoNextBid(log types.Log) (*Bunniv2DepositIntoNextBid, error) {
	event := new(Bunniv2DepositIntoNextBid)
	if err := _Bunniv2.contract.UnpackLog(event, "DepositIntoNextBid", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2DepositIntoTopBidIterator is returned from FilterDepositIntoTopBid and is used to iterate over the raw logs and unpacked data for DepositIntoTopBid events raised by the Bunniv2 contract.
type Bunniv2DepositIntoTopBidIterator struct {
	Event *Bunniv2DepositIntoTopBid // Event containing the contract specifics and raw log

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
func (it *Bunniv2DepositIntoTopBidIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2DepositIntoTopBid)
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
		it.Event = new(Bunniv2DepositIntoTopBid)
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
func (it *Bunniv2DepositIntoTopBidIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2DepositIntoTopBidIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2DepositIntoTopBid represents a DepositIntoTopBid event raised by the Bunniv2 contract.
type Bunniv2DepositIntoTopBid struct {
	Id      [32]byte
	Manager common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterDepositIntoTopBid is a free log retrieval operation binding the contract event 0x5fedbb7fbc8b2dd98c779ccc8c3d8e18dc48cc2eb1cb0215af9c1f77741d869a.
//
// Solidity: event DepositIntoTopBid(bytes32 indexed id, address indexed manager, uint128 amount)
func (_Bunniv2 *Bunniv2Filterer) FilterDepositIntoTopBid(opts *bind.FilterOpts, id [][32]byte, manager []common.Address) (*Bunniv2DepositIntoTopBidIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "DepositIntoTopBid", idRule, managerRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2DepositIntoTopBidIterator{contract: _Bunniv2.contract, event: "DepositIntoTopBid", logs: logs, sub: sub}, nil
}

// WatchDepositIntoTopBid is a free log subscription operation binding the contract event 0x5fedbb7fbc8b2dd98c779ccc8c3d8e18dc48cc2eb1cb0215af9c1f77741d869a.
//
// Solidity: event DepositIntoTopBid(bytes32 indexed id, address indexed manager, uint128 amount)
func (_Bunniv2 *Bunniv2Filterer) WatchDepositIntoTopBid(opts *bind.WatchOpts, sink chan<- *Bunniv2DepositIntoTopBid, id [][32]byte, manager []common.Address) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "DepositIntoTopBid", idRule, managerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2DepositIntoTopBid)
				if err := _Bunniv2.contract.UnpackLog(event, "DepositIntoTopBid", log); err != nil {
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

// ParseDepositIntoTopBid is a log parse operation binding the contract event 0x5fedbb7fbc8b2dd98c779ccc8c3d8e18dc48cc2eb1cb0215af9c1f77741d869a.
//
// Solidity: event DepositIntoTopBid(bytes32 indexed id, address indexed manager, uint128 amount)
func (_Bunniv2 *Bunniv2Filterer) ParseDepositIntoTopBid(log types.Log) (*Bunniv2DepositIntoTopBid, error) {
	event := new(Bunniv2DepositIntoTopBid)
	if err := _Bunniv2.contract.UnpackLog(event, "DepositIntoTopBid", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2IncreaseBidRentIterator is returned from FilterIncreaseBidRent and is used to iterate over the raw logs and unpacked data for IncreaseBidRent events raised by the Bunniv2 contract.
type Bunniv2IncreaseBidRentIterator struct {
	Event *Bunniv2IncreaseBidRent // Event containing the contract specifics and raw log

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
func (it *Bunniv2IncreaseBidRentIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2IncreaseBidRent)
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
		it.Event = new(Bunniv2IncreaseBidRent)
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
func (it *Bunniv2IncreaseBidRentIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2IncreaseBidRentIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2IncreaseBidRent represents a IncreaseBidRent event raised by the Bunniv2 contract.
type Bunniv2IncreaseBidRent struct {
	Id                [32]byte
	Manager           common.Address
	AdditionalRent    *big.Int
	UpdatedDeposit    *big.Int
	TopBid            bool
	WithdrawRecipient common.Address
	AmountDeposited   *big.Int
	AmountWithdrawn   *big.Int
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterIncreaseBidRent is a free log retrieval operation binding the contract event 0xf9f1ab7a2f5f90eeba49c837a758919739dc7e5ea102551b4251fb11552d57b2.
//
// Solidity: event IncreaseBidRent(bytes32 indexed id, address indexed manager, uint128 additionalRent, uint128 updatedDeposit, bool topBid, address indexed withdrawRecipient, uint128 amountDeposited, uint128 amountWithdrawn)
func (_Bunniv2 *Bunniv2Filterer) FilterIncreaseBidRent(opts *bind.FilterOpts, id [][32]byte, manager []common.Address, withdrawRecipient []common.Address) (*Bunniv2IncreaseBidRentIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}

	var withdrawRecipientRule []interface{}
	for _, withdrawRecipientItem := range withdrawRecipient {
		withdrawRecipientRule = append(withdrawRecipientRule, withdrawRecipientItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "IncreaseBidRent", idRule, managerRule, withdrawRecipientRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2IncreaseBidRentIterator{contract: _Bunniv2.contract, event: "IncreaseBidRent", logs: logs, sub: sub}, nil
}

// WatchIncreaseBidRent is a free log subscription operation binding the contract event 0xf9f1ab7a2f5f90eeba49c837a758919739dc7e5ea102551b4251fb11552d57b2.
//
// Solidity: event IncreaseBidRent(bytes32 indexed id, address indexed manager, uint128 additionalRent, uint128 updatedDeposit, bool topBid, address indexed withdrawRecipient, uint128 amountDeposited, uint128 amountWithdrawn)
func (_Bunniv2 *Bunniv2Filterer) WatchIncreaseBidRent(opts *bind.WatchOpts, sink chan<- *Bunniv2IncreaseBidRent, id [][32]byte, manager []common.Address, withdrawRecipient []common.Address) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}

	var withdrawRecipientRule []interface{}
	for _, withdrawRecipientItem := range withdrawRecipient {
		withdrawRecipientRule = append(withdrawRecipientRule, withdrawRecipientItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "IncreaseBidRent", idRule, managerRule, withdrawRecipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2IncreaseBidRent)
				if err := _Bunniv2.contract.UnpackLog(event, "IncreaseBidRent", log); err != nil {
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

// ParseIncreaseBidRent is a log parse operation binding the contract event 0xf9f1ab7a2f5f90eeba49c837a758919739dc7e5ea102551b4251fb11552d57b2.
//
// Solidity: event IncreaseBidRent(bytes32 indexed id, address indexed manager, uint128 additionalRent, uint128 updatedDeposit, bool topBid, address indexed withdrawRecipient, uint128 amountDeposited, uint128 amountWithdrawn)
func (_Bunniv2 *Bunniv2Filterer) ParseIncreaseBidRent(log types.Log) (*Bunniv2IncreaseBidRent, error) {
	event := new(Bunniv2IncreaseBidRent)
	if err := _Bunniv2.contract.UnpackLog(event, "IncreaseBidRent", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2OwnershipHandoverCanceledIterator is returned from FilterOwnershipHandoverCanceled and is used to iterate over the raw logs and unpacked data for OwnershipHandoverCanceled events raised by the Bunniv2 contract.
type Bunniv2OwnershipHandoverCanceledIterator struct {
	Event *Bunniv2OwnershipHandoverCanceled // Event containing the contract specifics and raw log

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
func (it *Bunniv2OwnershipHandoverCanceledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2OwnershipHandoverCanceled)
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
		it.Event = new(Bunniv2OwnershipHandoverCanceled)
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
func (it *Bunniv2OwnershipHandoverCanceledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2OwnershipHandoverCanceledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2OwnershipHandoverCanceled represents a OwnershipHandoverCanceled event raised by the Bunniv2 contract.
type Bunniv2OwnershipHandoverCanceled struct {
	PendingOwner common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterOwnershipHandoverCanceled is a free log retrieval operation binding the contract event 0xfa7b8eab7da67f412cc9575ed43464468f9bfbae89d1675917346ca6d8fe3c92.
//
// Solidity: event OwnershipHandoverCanceled(address indexed pendingOwner)
func (_Bunniv2 *Bunniv2Filterer) FilterOwnershipHandoverCanceled(opts *bind.FilterOpts, pendingOwner []common.Address) (*Bunniv2OwnershipHandoverCanceledIterator, error) {

	var pendingOwnerRule []interface{}
	for _, pendingOwnerItem := range pendingOwner {
		pendingOwnerRule = append(pendingOwnerRule, pendingOwnerItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "OwnershipHandoverCanceled", pendingOwnerRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2OwnershipHandoverCanceledIterator{contract: _Bunniv2.contract, event: "OwnershipHandoverCanceled", logs: logs, sub: sub}, nil
}

// WatchOwnershipHandoverCanceled is a free log subscription operation binding the contract event 0xfa7b8eab7da67f412cc9575ed43464468f9bfbae89d1675917346ca6d8fe3c92.
//
// Solidity: event OwnershipHandoverCanceled(address indexed pendingOwner)
func (_Bunniv2 *Bunniv2Filterer) WatchOwnershipHandoverCanceled(opts *bind.WatchOpts, sink chan<- *Bunniv2OwnershipHandoverCanceled, pendingOwner []common.Address) (event.Subscription, error) {

	var pendingOwnerRule []interface{}
	for _, pendingOwnerItem := range pendingOwner {
		pendingOwnerRule = append(pendingOwnerRule, pendingOwnerItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "OwnershipHandoverCanceled", pendingOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2OwnershipHandoverCanceled)
				if err := _Bunniv2.contract.UnpackLog(event, "OwnershipHandoverCanceled", log); err != nil {
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

// ParseOwnershipHandoverCanceled is a log parse operation binding the contract event 0xfa7b8eab7da67f412cc9575ed43464468f9bfbae89d1675917346ca6d8fe3c92.
//
// Solidity: event OwnershipHandoverCanceled(address indexed pendingOwner)
func (_Bunniv2 *Bunniv2Filterer) ParseOwnershipHandoverCanceled(log types.Log) (*Bunniv2OwnershipHandoverCanceled, error) {
	event := new(Bunniv2OwnershipHandoverCanceled)
	if err := _Bunniv2.contract.UnpackLog(event, "OwnershipHandoverCanceled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2OwnershipHandoverRequestedIterator is returned from FilterOwnershipHandoverRequested and is used to iterate over the raw logs and unpacked data for OwnershipHandoverRequested events raised by the Bunniv2 contract.
type Bunniv2OwnershipHandoverRequestedIterator struct {
	Event *Bunniv2OwnershipHandoverRequested // Event containing the contract specifics and raw log

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
func (it *Bunniv2OwnershipHandoverRequestedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2OwnershipHandoverRequested)
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
		it.Event = new(Bunniv2OwnershipHandoverRequested)
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
func (it *Bunniv2OwnershipHandoverRequestedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2OwnershipHandoverRequestedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2OwnershipHandoverRequested represents a OwnershipHandoverRequested event raised by the Bunniv2 contract.
type Bunniv2OwnershipHandoverRequested struct {
	PendingOwner common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterOwnershipHandoverRequested is a free log retrieval operation binding the contract event 0xdbf36a107da19e49527a7176a1babf963b4b0ff8cde35ee35d6cd8f1f9ac7e1d.
//
// Solidity: event OwnershipHandoverRequested(address indexed pendingOwner)
func (_Bunniv2 *Bunniv2Filterer) FilterOwnershipHandoverRequested(opts *bind.FilterOpts, pendingOwner []common.Address) (*Bunniv2OwnershipHandoverRequestedIterator, error) {

	var pendingOwnerRule []interface{}
	for _, pendingOwnerItem := range pendingOwner {
		pendingOwnerRule = append(pendingOwnerRule, pendingOwnerItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "OwnershipHandoverRequested", pendingOwnerRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2OwnershipHandoverRequestedIterator{contract: _Bunniv2.contract, event: "OwnershipHandoverRequested", logs: logs, sub: sub}, nil
}

// WatchOwnershipHandoverRequested is a free log subscription operation binding the contract event 0xdbf36a107da19e49527a7176a1babf963b4b0ff8cde35ee35d6cd8f1f9ac7e1d.
//
// Solidity: event OwnershipHandoverRequested(address indexed pendingOwner)
func (_Bunniv2 *Bunniv2Filterer) WatchOwnershipHandoverRequested(opts *bind.WatchOpts, sink chan<- *Bunniv2OwnershipHandoverRequested, pendingOwner []common.Address) (event.Subscription, error) {

	var pendingOwnerRule []interface{}
	for _, pendingOwnerItem := range pendingOwner {
		pendingOwnerRule = append(pendingOwnerRule, pendingOwnerItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "OwnershipHandoverRequested", pendingOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2OwnershipHandoverRequested)
				if err := _Bunniv2.contract.UnpackLog(event, "OwnershipHandoverRequested", log); err != nil {
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

// ParseOwnershipHandoverRequested is a log parse operation binding the contract event 0xdbf36a107da19e49527a7176a1babf963b4b0ff8cde35ee35d6cd8f1f9ac7e1d.
//
// Solidity: event OwnershipHandoverRequested(address indexed pendingOwner)
func (_Bunniv2 *Bunniv2Filterer) ParseOwnershipHandoverRequested(log types.Log) (*Bunniv2OwnershipHandoverRequested, error) {
	event := new(Bunniv2OwnershipHandoverRequested)
	if err := _Bunniv2.contract.UnpackLog(event, "OwnershipHandoverRequested", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2OwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Bunniv2 contract.
type Bunniv2OwnershipTransferredIterator struct {
	Event *Bunniv2OwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *Bunniv2OwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2OwnershipTransferred)
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
		it.Event = new(Bunniv2OwnershipTransferred)
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
func (it *Bunniv2OwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2OwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2OwnershipTransferred represents a OwnershipTransferred event raised by the Bunniv2 contract.
type Bunniv2OwnershipTransferred struct {
	OldOwner common.Address
	NewOwner common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed oldOwner, address indexed newOwner)
func (_Bunniv2 *Bunniv2Filterer) FilterOwnershipTransferred(opts *bind.FilterOpts, oldOwner []common.Address, newOwner []common.Address) (*Bunniv2OwnershipTransferredIterator, error) {

	var oldOwnerRule []interface{}
	for _, oldOwnerItem := range oldOwner {
		oldOwnerRule = append(oldOwnerRule, oldOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "OwnershipTransferred", oldOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2OwnershipTransferredIterator{contract: _Bunniv2.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed oldOwner, address indexed newOwner)
func (_Bunniv2 *Bunniv2Filterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *Bunniv2OwnershipTransferred, oldOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var oldOwnerRule []interface{}
	for _, oldOwnerItem := range oldOwner {
		oldOwnerRule = append(oldOwnerRule, oldOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "OwnershipTransferred", oldOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2OwnershipTransferred)
				if err := _Bunniv2.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
// Solidity: event OwnershipTransferred(address indexed oldOwner, address indexed newOwner)
func (_Bunniv2 *Bunniv2Filterer) ParseOwnershipTransferred(log types.Log) (*Bunniv2OwnershipTransferred, error) {
	event := new(Bunniv2OwnershipTransferred)
	if err := _Bunniv2.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2ScheduleKChangeIterator is returned from FilterScheduleKChange and is used to iterate over the raw logs and unpacked data for ScheduleKChange events raised by the Bunniv2 contract.
type Bunniv2ScheduleKChangeIterator struct {
	Event *Bunniv2ScheduleKChange // Event containing the contract specifics and raw log

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
func (it *Bunniv2ScheduleKChangeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2ScheduleKChange)
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
		it.Event = new(Bunniv2ScheduleKChange)
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
func (it *Bunniv2ScheduleKChangeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2ScheduleKChangeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2ScheduleKChange represents a ScheduleKChange event raised by the Bunniv2 contract.
type Bunniv2ScheduleKChange struct {
	CurrentK    *big.Int
	NewK        *big.Int
	ActiveBlock *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterScheduleKChange is a free log retrieval operation binding the contract event 0xe327dd1af0911b252ef7e49d7a72cd9fbca0275a5be0590eb13e7482272a4674.
//
// Solidity: event ScheduleKChange(uint48 currentK, uint48 indexed newK, uint160 indexed activeBlock)
func (_Bunniv2 *Bunniv2Filterer) FilterScheduleKChange(opts *bind.FilterOpts, newK []*big.Int, activeBlock []*big.Int) (*Bunniv2ScheduleKChangeIterator, error) {

	var newKRule []interface{}
	for _, newKItem := range newK {
		newKRule = append(newKRule, newKItem)
	}
	var activeBlockRule []interface{}
	for _, activeBlockItem := range activeBlock {
		activeBlockRule = append(activeBlockRule, activeBlockItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "ScheduleKChange", newKRule, activeBlockRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2ScheduleKChangeIterator{contract: _Bunniv2.contract, event: "ScheduleKChange", logs: logs, sub: sub}, nil
}

// WatchScheduleKChange is a free log subscription operation binding the contract event 0xe327dd1af0911b252ef7e49d7a72cd9fbca0275a5be0590eb13e7482272a4674.
//
// Solidity: event ScheduleKChange(uint48 currentK, uint48 indexed newK, uint160 indexed activeBlock)
func (_Bunniv2 *Bunniv2Filterer) WatchScheduleKChange(opts *bind.WatchOpts, sink chan<- *Bunniv2ScheduleKChange, newK []*big.Int, activeBlock []*big.Int) (event.Subscription, error) {

	var newKRule []interface{}
	for _, newKItem := range newK {
		newKRule = append(newKRule, newKItem)
	}
	var activeBlockRule []interface{}
	for _, activeBlockItem := range activeBlock {
		activeBlockRule = append(activeBlockRule, activeBlockItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "ScheduleKChange", newKRule, activeBlockRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2ScheduleKChange)
				if err := _Bunniv2.contract.UnpackLog(event, "ScheduleKChange", log); err != nil {
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

// ParseScheduleKChange is a log parse operation binding the contract event 0xe327dd1af0911b252ef7e49d7a72cd9fbca0275a5be0590eb13e7482272a4674.
//
// Solidity: event ScheduleKChange(uint48 currentK, uint48 indexed newK, uint160 indexed activeBlock)
func (_Bunniv2 *Bunniv2Filterer) ParseScheduleKChange(log types.Log) (*Bunniv2ScheduleKChange, error) {
	event := new(Bunniv2ScheduleKChange)
	if err := _Bunniv2.contract.UnpackLog(event, "ScheduleKChange", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2SetBidPayloadIterator is returned from FilterSetBidPayload and is used to iterate over the raw logs and unpacked data for SetBidPayload events raised by the Bunniv2 contract.
type Bunniv2SetBidPayloadIterator struct {
	Event *Bunniv2SetBidPayload // Event containing the contract specifics and raw log

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
func (it *Bunniv2SetBidPayloadIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2SetBidPayload)
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
		it.Event = new(Bunniv2SetBidPayload)
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
func (it *Bunniv2SetBidPayloadIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2SetBidPayloadIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2SetBidPayload represents a SetBidPayload event raised by the Bunniv2 contract.
type Bunniv2SetBidPayload struct {
	Id      [32]byte
	Manager common.Address
	Payload [6]byte
	TopBid  bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterSetBidPayload is a free log retrieval operation binding the contract event 0xd89758def3a89808b83911284af1db38111ea1b042a45f0d39d526f67fab7a37.
//
// Solidity: event SetBidPayload(bytes32 indexed id, address indexed manager, bytes6 payload, bool topBid)
func (_Bunniv2 *Bunniv2Filterer) FilterSetBidPayload(opts *bind.FilterOpts, id [][32]byte, manager []common.Address) (*Bunniv2SetBidPayloadIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "SetBidPayload", idRule, managerRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2SetBidPayloadIterator{contract: _Bunniv2.contract, event: "SetBidPayload", logs: logs, sub: sub}, nil
}

// WatchSetBidPayload is a free log subscription operation binding the contract event 0xd89758def3a89808b83911284af1db38111ea1b042a45f0d39d526f67fab7a37.
//
// Solidity: event SetBidPayload(bytes32 indexed id, address indexed manager, bytes6 payload, bool topBid)
func (_Bunniv2 *Bunniv2Filterer) WatchSetBidPayload(opts *bind.WatchOpts, sink chan<- *Bunniv2SetBidPayload, id [][32]byte, manager []common.Address) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "SetBidPayload", idRule, managerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2SetBidPayload)
				if err := _Bunniv2.contract.UnpackLog(event, "SetBidPayload", log); err != nil {
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

// ParseSetBidPayload is a log parse operation binding the contract event 0xd89758def3a89808b83911284af1db38111ea1b042a45f0d39d526f67fab7a37.
//
// Solidity: event SetBidPayload(bytes32 indexed id, address indexed manager, bytes6 payload, bool topBid)
func (_Bunniv2 *Bunniv2Filterer) ParseSetBidPayload(log types.Log) (*Bunniv2SetBidPayload, error) {
	event := new(Bunniv2SetBidPayload)
	if err := _Bunniv2.contract.UnpackLog(event, "SetBidPayload", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2SetHookFeeModifierIterator is returned from FilterSetHookFeeModifier and is used to iterate over the raw logs and unpacked data for SetHookFeeModifier events raised by the Bunniv2 contract.
type Bunniv2SetHookFeeModifierIterator struct {
	Event *Bunniv2SetHookFeeModifier // Event containing the contract specifics and raw log

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
func (it *Bunniv2SetHookFeeModifierIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2SetHookFeeModifier)
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
		it.Event = new(Bunniv2SetHookFeeModifier)
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
func (it *Bunniv2SetHookFeeModifierIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2SetHookFeeModifierIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2SetHookFeeModifier represents a SetHookFeeModifier event raised by the Bunniv2 contract.
type Bunniv2SetHookFeeModifier struct {
	HookFeeModifier uint32
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterSetHookFeeModifier is a free log retrieval operation binding the contract event 0x1830fd731bc638f82d569312f1d49094df6daec186ff017fe0b73c79df44f29e.
//
// Solidity: event SetHookFeeModifier(uint32 indexed hookFeeModifier)
func (_Bunniv2 *Bunniv2Filterer) FilterSetHookFeeModifier(opts *bind.FilterOpts, hookFeeModifier []uint32) (*Bunniv2SetHookFeeModifierIterator, error) {

	var hookFeeModifierRule []interface{}
	for _, hookFeeModifierItem := range hookFeeModifier {
		hookFeeModifierRule = append(hookFeeModifierRule, hookFeeModifierItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "SetHookFeeModifier", hookFeeModifierRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2SetHookFeeModifierIterator{contract: _Bunniv2.contract, event: "SetHookFeeModifier", logs: logs, sub: sub}, nil
}

// WatchSetHookFeeModifier is a free log subscription operation binding the contract event 0x1830fd731bc638f82d569312f1d49094df6daec186ff017fe0b73c79df44f29e.
//
// Solidity: event SetHookFeeModifier(uint32 indexed hookFeeModifier)
func (_Bunniv2 *Bunniv2Filterer) WatchSetHookFeeModifier(opts *bind.WatchOpts, sink chan<- *Bunniv2SetHookFeeModifier, hookFeeModifier []uint32) (event.Subscription, error) {

	var hookFeeModifierRule []interface{}
	for _, hookFeeModifierItem := range hookFeeModifier {
		hookFeeModifierRule = append(hookFeeModifierRule, hookFeeModifierItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "SetHookFeeModifier", hookFeeModifierRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2SetHookFeeModifier)
				if err := _Bunniv2.contract.UnpackLog(event, "SetHookFeeModifier", log); err != nil {
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

// ParseSetHookFeeModifier is a log parse operation binding the contract event 0x1830fd731bc638f82d569312f1d49094df6daec186ff017fe0b73c79df44f29e.
//
// Solidity: event SetHookFeeModifier(uint32 indexed hookFeeModifier)
func (_Bunniv2 *Bunniv2Filterer) ParseSetHookFeeModifier(log types.Log) (*Bunniv2SetHookFeeModifier, error) {
	event := new(Bunniv2SetHookFeeModifier)
	if err := _Bunniv2.contract.UnpackLog(event, "SetHookFeeModifier", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2SetHookFeeRecipientIterator is returned from FilterSetHookFeeRecipient and is used to iterate over the raw logs and unpacked data for SetHookFeeRecipient events raised by the Bunniv2 contract.
type Bunniv2SetHookFeeRecipientIterator struct {
	Event *Bunniv2SetHookFeeRecipient // Event containing the contract specifics and raw log

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
func (it *Bunniv2SetHookFeeRecipientIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2SetHookFeeRecipient)
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
		it.Event = new(Bunniv2SetHookFeeRecipient)
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
func (it *Bunniv2SetHookFeeRecipientIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2SetHookFeeRecipientIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2SetHookFeeRecipient represents a SetHookFeeRecipient event raised by the Bunniv2 contract.
type Bunniv2SetHookFeeRecipient struct {
	HookFeeRecipient common.Address
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterSetHookFeeRecipient is a free log retrieval operation binding the contract event 0x1ad2c8d0e069471d98a667bcf612463b372082f8870dc6d167a3793a133931ab.
//
// Solidity: event SetHookFeeRecipient(address hookFeeRecipient)
func (_Bunniv2 *Bunniv2Filterer) FilterSetHookFeeRecipient(opts *bind.FilterOpts) (*Bunniv2SetHookFeeRecipientIterator, error) {

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "SetHookFeeRecipient")
	if err != nil {
		return nil, err
	}
	return &Bunniv2SetHookFeeRecipientIterator{contract: _Bunniv2.contract, event: "SetHookFeeRecipient", logs: logs, sub: sub}, nil
}

// WatchSetHookFeeRecipient is a free log subscription operation binding the contract event 0x1ad2c8d0e069471d98a667bcf612463b372082f8870dc6d167a3793a133931ab.
//
// Solidity: event SetHookFeeRecipient(address hookFeeRecipient)
func (_Bunniv2 *Bunniv2Filterer) WatchSetHookFeeRecipient(opts *bind.WatchOpts, sink chan<- *Bunniv2SetHookFeeRecipient) (event.Subscription, error) {

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "SetHookFeeRecipient")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2SetHookFeeRecipient)
				if err := _Bunniv2.contract.UnpackLog(event, "SetHookFeeRecipient", log); err != nil {
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

// ParseSetHookFeeRecipient is a log parse operation binding the contract event 0x1ad2c8d0e069471d98a667bcf612463b372082f8870dc6d167a3793a133931ab.
//
// Solidity: event SetHookFeeRecipient(address hookFeeRecipient)
func (_Bunniv2 *Bunniv2Filterer) ParseSetHookFeeRecipient(log types.Log) (*Bunniv2SetHookFeeRecipient, error) {
	event := new(Bunniv2SetHookFeeRecipient)
	if err := _Bunniv2.contract.UnpackLog(event, "SetHookFeeRecipient", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2SetWithdrawalUnblockedIterator is returned from FilterSetWithdrawalUnblocked and is used to iterate over the raw logs and unpacked data for SetWithdrawalUnblocked events raised by the Bunniv2 contract.
type Bunniv2SetWithdrawalUnblockedIterator struct {
	Event *Bunniv2SetWithdrawalUnblocked // Event containing the contract specifics and raw log

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
func (it *Bunniv2SetWithdrawalUnblockedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2SetWithdrawalUnblocked)
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
		it.Event = new(Bunniv2SetWithdrawalUnblocked)
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
func (it *Bunniv2SetWithdrawalUnblockedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2SetWithdrawalUnblockedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2SetWithdrawalUnblocked represents a SetWithdrawalUnblocked event raised by the Bunniv2 contract.
type Bunniv2SetWithdrawalUnblocked struct {
	Id        [32]byte
	Unblocked bool
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterSetWithdrawalUnblocked is a free log retrieval operation binding the contract event 0xa74dd19ca00a9e1befbb93daa40f943456e44ffc5b645c00ac4d68467cc4df8d.
//
// Solidity: event SetWithdrawalUnblocked(bytes32 indexed id, bool unblocked)
func (_Bunniv2 *Bunniv2Filterer) FilterSetWithdrawalUnblocked(opts *bind.FilterOpts, id [][32]byte) (*Bunniv2SetWithdrawalUnblockedIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "SetWithdrawalUnblocked", idRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2SetWithdrawalUnblockedIterator{contract: _Bunniv2.contract, event: "SetWithdrawalUnblocked", logs: logs, sub: sub}, nil
}

// WatchSetWithdrawalUnblocked is a free log subscription operation binding the contract event 0xa74dd19ca00a9e1befbb93daa40f943456e44ffc5b645c00ac4d68467cc4df8d.
//
// Solidity: event SetWithdrawalUnblocked(bytes32 indexed id, bool unblocked)
func (_Bunniv2 *Bunniv2Filterer) WatchSetWithdrawalUnblocked(opts *bind.WatchOpts, sink chan<- *Bunniv2SetWithdrawalUnblocked, id [][32]byte) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "SetWithdrawalUnblocked", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2SetWithdrawalUnblocked)
				if err := _Bunniv2.contract.UnpackLog(event, "SetWithdrawalUnblocked", log); err != nil {
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

// ParseSetWithdrawalUnblocked is a log parse operation binding the contract event 0xa74dd19ca00a9e1befbb93daa40f943456e44ffc5b645c00ac4d68467cc4df8d.
//
// Solidity: event SetWithdrawalUnblocked(bytes32 indexed id, bool unblocked)
func (_Bunniv2 *Bunniv2Filterer) ParseSetWithdrawalUnblocked(log types.Log) (*Bunniv2SetWithdrawalUnblocked, error) {
	event := new(Bunniv2SetWithdrawalUnblocked)
	if err := _Bunniv2.contract.UnpackLog(event, "SetWithdrawalUnblocked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2SetZoneIterator is returned from FilterSetZone and is used to iterate over the raw logs and unpacked data for SetZone events raised by the Bunniv2 contract.
type Bunniv2SetZoneIterator struct {
	Event *Bunniv2SetZone // Event containing the contract specifics and raw log

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
func (it *Bunniv2SetZoneIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2SetZone)
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
		it.Event = new(Bunniv2SetZone)
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
func (it *Bunniv2SetZoneIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2SetZoneIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2SetZone represents a SetZone event raised by the Bunniv2 contract.
type Bunniv2SetZone struct {
	Zone common.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterSetZone is a free log retrieval operation binding the contract event 0x21014fa276f518db71a4b74f0526c61ed7661bf69a707c64a30d5b0040b7113c.
//
// Solidity: event SetZone(address zone)
func (_Bunniv2 *Bunniv2Filterer) FilterSetZone(opts *bind.FilterOpts) (*Bunniv2SetZoneIterator, error) {

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "SetZone")
	if err != nil {
		return nil, err
	}
	return &Bunniv2SetZoneIterator{contract: _Bunniv2.contract, event: "SetZone", logs: logs, sub: sub}, nil
}

// WatchSetZone is a free log subscription operation binding the contract event 0x21014fa276f518db71a4b74f0526c61ed7661bf69a707c64a30d5b0040b7113c.
//
// Solidity: event SetZone(address zone)
func (_Bunniv2 *Bunniv2Filterer) WatchSetZone(opts *bind.WatchOpts, sink chan<- *Bunniv2SetZone) (event.Subscription, error) {

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "SetZone")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2SetZone)
				if err := _Bunniv2.contract.UnpackLog(event, "SetZone", log); err != nil {
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

// ParseSetZone is a log parse operation binding the contract event 0x21014fa276f518db71a4b74f0526c61ed7661bf69a707c64a30d5b0040b7113c.
//
// Solidity: event SetZone(address zone)
func (_Bunniv2 *Bunniv2Filterer) ParseSetZone(log types.Log) (*Bunniv2SetZone, error) {
	event := new(Bunniv2SetZone)
	if err := _Bunniv2.contract.UnpackLog(event, "SetZone", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2SubmitBidIterator is returned from FilterSubmitBid and is used to iterate over the raw logs and unpacked data for SubmitBid events raised by the Bunniv2 contract.
type Bunniv2SubmitBidIterator struct {
	Event *Bunniv2SubmitBid // Event containing the contract specifics and raw log

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
func (it *Bunniv2SubmitBidIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2SubmitBid)
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
		it.Event = new(Bunniv2SubmitBid)
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
func (it *Bunniv2SubmitBidIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2SubmitBidIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2SubmitBid represents a SubmitBid event raised by the Bunniv2 contract.
type Bunniv2SubmitBid struct {
	Id       [32]byte
	Manager  common.Address
	BlockIdx *big.Int
	Payload  [6]byte
	Rent     *big.Int
	Deposit  *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterSubmitBid is a free log retrieval operation binding the contract event 0x892cf75ad2a5896151df4a595c62378620b908182714c75fc026987990dfa7c9.
//
// Solidity: event SubmitBid(bytes32 indexed id, address indexed manager, uint48 indexed blockIdx, bytes6 payload, uint128 rent, uint128 deposit)
func (_Bunniv2 *Bunniv2Filterer) FilterSubmitBid(opts *bind.FilterOpts, id [][32]byte, manager []common.Address, blockIdx []*big.Int) (*Bunniv2SubmitBidIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}
	var blockIdxRule []interface{}
	for _, blockIdxItem := range blockIdx {
		blockIdxRule = append(blockIdxRule, blockIdxItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "SubmitBid", idRule, managerRule, blockIdxRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2SubmitBidIterator{contract: _Bunniv2.contract, event: "SubmitBid", logs: logs, sub: sub}, nil
}

// WatchSubmitBid is a free log subscription operation binding the contract event 0x892cf75ad2a5896151df4a595c62378620b908182714c75fc026987990dfa7c9.
//
// Solidity: event SubmitBid(bytes32 indexed id, address indexed manager, uint48 indexed blockIdx, bytes6 payload, uint128 rent, uint128 deposit)
func (_Bunniv2 *Bunniv2Filterer) WatchSubmitBid(opts *bind.WatchOpts, sink chan<- *Bunniv2SubmitBid, id [][32]byte, manager []common.Address, blockIdx []*big.Int) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}
	var blockIdxRule []interface{}
	for _, blockIdxItem := range blockIdx {
		blockIdxRule = append(blockIdxRule, blockIdxItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "SubmitBid", idRule, managerRule, blockIdxRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2SubmitBid)
				if err := _Bunniv2.contract.UnpackLog(event, "SubmitBid", log); err != nil {
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

// ParseSubmitBid is a log parse operation binding the contract event 0x892cf75ad2a5896151df4a595c62378620b908182714c75fc026987990dfa7c9.
//
// Solidity: event SubmitBid(bytes32 indexed id, address indexed manager, uint48 indexed blockIdx, bytes6 payload, uint128 rent, uint128 deposit)
func (_Bunniv2 *Bunniv2Filterer) ParseSubmitBid(log types.Log) (*Bunniv2SubmitBid, error) {
	event := new(Bunniv2SubmitBid)
	if err := _Bunniv2.contract.UnpackLog(event, "SubmitBid", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2SwapIterator is returned from FilterSwap and is used to iterate over the raw logs and unpacked data for Swap events raised by the Bunniv2 contract.
type Bunniv2SwapIterator struct {
	Event *Bunniv2Swap // Event containing the contract specifics and raw log

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
func (it *Bunniv2SwapIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2Swap)
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
		it.Event = new(Bunniv2Swap)
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
func (it *Bunniv2SwapIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2SwapIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2Swap represents a Swap event raised by the Bunniv2 contract.
type Bunniv2Swap struct {
	Id             [32]byte
	Sender         common.Address
	ExactIn        bool
	ZeroForOne     bool
	InputAmount    *big.Int
	OutputAmount   *big.Int
	SqrtPriceX96   *big.Int
	Tick           *big.Int
	Fee            *big.Int
	TotalLiquidity *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterSwap is a free log retrieval operation binding the contract event 0xcd42809a29fc60d050d9e34a2f48fe30855a6451eca6c8a61ca7f21e1881644d.
//
// Solidity: event Swap(bytes32 indexed id, address indexed sender, bool exactIn, bool zeroForOne, uint256 inputAmount, uint256 outputAmount, uint160 sqrtPriceX96, int24 tick, uint24 fee, uint256 totalLiquidity)
func (_Bunniv2 *Bunniv2Filterer) FilterSwap(opts *bind.FilterOpts, id [][32]byte, sender []common.Address) (*Bunniv2SwapIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "Swap", idRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2SwapIterator{contract: _Bunniv2.contract, event: "Swap", logs: logs, sub: sub}, nil
}

// WatchSwap is a free log subscription operation binding the contract event 0xcd42809a29fc60d050d9e34a2f48fe30855a6451eca6c8a61ca7f21e1881644d.
//
// Solidity: event Swap(bytes32 indexed id, address indexed sender, bool exactIn, bool zeroForOne, uint256 inputAmount, uint256 outputAmount, uint160 sqrtPriceX96, int24 tick, uint24 fee, uint256 totalLiquidity)
func (_Bunniv2 *Bunniv2Filterer) WatchSwap(opts *bind.WatchOpts, sink chan<- *Bunniv2Swap, id [][32]byte, sender []common.Address) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "Swap", idRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2Swap)
				if err := _Bunniv2.contract.UnpackLog(event, "Swap", log); err != nil {
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

// ParseSwap is a log parse operation binding the contract event 0xcd42809a29fc60d050d9e34a2f48fe30855a6451eca6c8a61ca7f21e1881644d.
//
// Solidity: event Swap(bytes32 indexed id, address indexed sender, bool exactIn, bool zeroForOne, uint256 inputAmount, uint256 outputAmount, uint160 sqrtPriceX96, int24 tick, uint24 fee, uint256 totalLiquidity)
func (_Bunniv2 *Bunniv2Filterer) ParseSwap(log types.Log) (*Bunniv2Swap, error) {
	event := new(Bunniv2Swap)
	if err := _Bunniv2.contract.UnpackLog(event, "Swap", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2WithdrawFromNextBidIterator is returned from FilterWithdrawFromNextBid and is used to iterate over the raw logs and unpacked data for WithdrawFromNextBid events raised by the Bunniv2 contract.
type Bunniv2WithdrawFromNextBidIterator struct {
	Event *Bunniv2WithdrawFromNextBid // Event containing the contract specifics and raw log

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
func (it *Bunniv2WithdrawFromNextBidIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2WithdrawFromNextBid)
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
		it.Event = new(Bunniv2WithdrawFromNextBid)
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
func (it *Bunniv2WithdrawFromNextBidIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2WithdrawFromNextBidIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2WithdrawFromNextBid represents a WithdrawFromNextBid event raised by the Bunniv2 contract.
type Bunniv2WithdrawFromNextBid struct {
	Id        [32]byte
	Manager   common.Address
	Recipient common.Address
	Amount    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterWithdrawFromNextBid is a free log retrieval operation binding the contract event 0xd6cbb6e489a2ca95eea500cfdffb93e083287ee5a015abb7ed0fe2fdd770fe80.
//
// Solidity: event WithdrawFromNextBid(bytes32 indexed id, address indexed manager, address indexed recipient, uint128 amount)
func (_Bunniv2 *Bunniv2Filterer) FilterWithdrawFromNextBid(opts *bind.FilterOpts, id [][32]byte, manager []common.Address, recipient []common.Address) (*Bunniv2WithdrawFromNextBidIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "WithdrawFromNextBid", idRule, managerRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2WithdrawFromNextBidIterator{contract: _Bunniv2.contract, event: "WithdrawFromNextBid", logs: logs, sub: sub}, nil
}

// WatchWithdrawFromNextBid is a free log subscription operation binding the contract event 0xd6cbb6e489a2ca95eea500cfdffb93e083287ee5a015abb7ed0fe2fdd770fe80.
//
// Solidity: event WithdrawFromNextBid(bytes32 indexed id, address indexed manager, address indexed recipient, uint128 amount)
func (_Bunniv2 *Bunniv2Filterer) WatchWithdrawFromNextBid(opts *bind.WatchOpts, sink chan<- *Bunniv2WithdrawFromNextBid, id [][32]byte, manager []common.Address, recipient []common.Address) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "WithdrawFromNextBid", idRule, managerRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2WithdrawFromNextBid)
				if err := _Bunniv2.contract.UnpackLog(event, "WithdrawFromNextBid", log); err != nil {
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

// ParseWithdrawFromNextBid is a log parse operation binding the contract event 0xd6cbb6e489a2ca95eea500cfdffb93e083287ee5a015abb7ed0fe2fdd770fe80.
//
// Solidity: event WithdrawFromNextBid(bytes32 indexed id, address indexed manager, address indexed recipient, uint128 amount)
func (_Bunniv2 *Bunniv2Filterer) ParseWithdrawFromNextBid(log types.Log) (*Bunniv2WithdrawFromNextBid, error) {
	event := new(Bunniv2WithdrawFromNextBid)
	if err := _Bunniv2.contract.UnpackLog(event, "WithdrawFromNextBid", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Bunniv2WithdrawFromTopBidIterator is returned from FilterWithdrawFromTopBid and is used to iterate over the raw logs and unpacked data for WithdrawFromTopBid events raised by the Bunniv2 contract.
type Bunniv2WithdrawFromTopBidIterator struct {
	Event *Bunniv2WithdrawFromTopBid // Event containing the contract specifics and raw log

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
func (it *Bunniv2WithdrawFromTopBidIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Bunniv2WithdrawFromTopBid)
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
		it.Event = new(Bunniv2WithdrawFromTopBid)
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
func (it *Bunniv2WithdrawFromTopBidIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Bunniv2WithdrawFromTopBidIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Bunniv2WithdrawFromTopBid represents a WithdrawFromTopBid event raised by the Bunniv2 contract.
type Bunniv2WithdrawFromTopBid struct {
	Id        [32]byte
	Manager   common.Address
	Recipient common.Address
	Amount    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterWithdrawFromTopBid is a free log retrieval operation binding the contract event 0x804678e3e29c23a5bf87029444349f6b833eab92ed696f51ed1d9412e738d027.
//
// Solidity: event WithdrawFromTopBid(bytes32 indexed id, address indexed manager, address indexed recipient, uint128 amount)
func (_Bunniv2 *Bunniv2Filterer) FilterWithdrawFromTopBid(opts *bind.FilterOpts, id [][32]byte, manager []common.Address, recipient []common.Address) (*Bunniv2WithdrawFromTopBidIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Bunniv2.contract.FilterLogs(opts, "WithdrawFromTopBid", idRule, managerRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &Bunniv2WithdrawFromTopBidIterator{contract: _Bunniv2.contract, event: "WithdrawFromTopBid", logs: logs, sub: sub}, nil
}

// WatchWithdrawFromTopBid is a free log subscription operation binding the contract event 0x804678e3e29c23a5bf87029444349f6b833eab92ed696f51ed1d9412e738d027.
//
// Solidity: event WithdrawFromTopBid(bytes32 indexed id, address indexed manager, address indexed recipient, uint128 amount)
func (_Bunniv2 *Bunniv2Filterer) WatchWithdrawFromTopBid(opts *bind.WatchOpts, sink chan<- *Bunniv2WithdrawFromTopBid, id [][32]byte, manager []common.Address, recipient []common.Address) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}
	var managerRule []interface{}
	for _, managerItem := range manager {
		managerRule = append(managerRule, managerItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Bunniv2.contract.WatchLogs(opts, "WithdrawFromTopBid", idRule, managerRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Bunniv2WithdrawFromTopBid)
				if err := _Bunniv2.contract.UnpackLog(event, "WithdrawFromTopBid", log); err != nil {
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

// ParseWithdrawFromTopBid is a log parse operation binding the contract event 0x804678e3e29c23a5bf87029444349f6b833eab92ed696f51ed1d9412e738d027.
//
// Solidity: event WithdrawFromTopBid(bytes32 indexed id, address indexed manager, address indexed recipient, uint128 amount)
func (_Bunniv2 *Bunniv2Filterer) ParseWithdrawFromTopBid(log types.Log) (*Bunniv2WithdrawFromTopBid, error) {
	event := new(Bunniv2WithdrawFromTopBid)
	if err := _Bunniv2.contract.UnpackLog(event, "WithdrawFromTopBid", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
