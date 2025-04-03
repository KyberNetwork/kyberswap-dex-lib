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

// IBunniHubDeployBunniTokenParams is an auto generated low-level Go binding around an user-defined struct.
type IBunniHubDeployBunniTokenParams struct {
	Currency0                common.Address
	Currency1                common.Address
	TickSpacing              *big.Int
	TwapSecondsAgo           *big.Int
	LiquidityDensityFunction common.Address
	Hooklet                  common.Address
	LdfType                  uint8
	LdfParams                [32]byte
	Hooks                    common.Address
	HookParams               []byte
	Vault0                   common.Address
	Vault1                   common.Address
	MinRawTokenRatio0        *big.Int
	TargetRawTokenRatio0     *big.Int
	MaxRawTokenRatio0        *big.Int
	MinRawTokenRatio1        *big.Int
	TargetRawTokenRatio1     *big.Int
	MaxRawTokenRatio1        *big.Int
	SqrtPriceX96             *big.Int
	Name                     [32]byte
	Symbol                   [32]byte
	Owner                    common.Address
	MetadataURI              string
	Salt                     [32]byte
}

// IBunniHubDepositParams is an auto generated low-level Go binding around an user-defined struct.
type IBunniHubDepositParams struct {
	PoolKey         PoolKey
	Recipient       common.Address
	RefundRecipient common.Address
	Amount0Desired  *big.Int
	Amount1Desired  *big.Int
	Amount0Min      *big.Int
	Amount1Min      *big.Int
	VaultFee0       *big.Int
	VaultFee1       *big.Int
	Deadline        *big.Int
	Referrer        common.Address
}

// IBunniHubQueueWithdrawParams is an auto generated low-level Go binding around an user-defined struct.
type IBunniHubQueueWithdrawParams struct {
	PoolKey PoolKey
	Shares  *big.Int
}

// IBunniHubWithdrawParams is an auto generated low-level Go binding around an user-defined struct.
type IBunniHubWithdrawParams struct {
	PoolKey             PoolKey
	Recipient           common.Address
	Shares              *big.Int
	Amount0Min          *big.Int
	Amount1Min          *big.Int
	Deadline            *big.Int
	UseQueuedWithdrawal bool
}

// PoolKey is an auto generated low-level Go binding around an user-defined struct.
type PoolKey struct {
	Currency0   common.Address
	Currency1   common.Address
	Fee         *big.Int
	TickSpacing *big.Int
	Hooks       common.Address
}

// PoolState is an auto generated low-level Go binding around an user-defined struct.
type PoolState struct {
	LiquidityDensityFunction common.Address
	BunniToken               common.Address
	Hooklet                  common.Address
	TwapSecondsAgo           *big.Int
	LdfParams                [32]byte
	HookParams               []byte
	Vault0                   common.Address
	Vault1                   common.Address
	LdfType                  uint8
	MinRawTokenRatio0        *big.Int
	TargetRawTokenRatio0     *big.Int
	MaxRawTokenRatio0        *big.Int
	MinRawTokenRatio1        *big.Int
	TargetRawTokenRatio1     *big.Int
	MaxRawTokenRatio1        *big.Int
	RawBalance0              *big.Int
	RawBalance1              *big.Int
	Reserve0                 *big.Int
	Reserve1                 *big.Int
	IdleBalance              [32]byte
}

// BunniV2HubContractMetaData contains all meta data concerning the BunniV2HubContract contract.
var BunniV2HubContractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIPoolManager\",\"name\":\"poolManager_\",\"type\":\"address\"},{\"internalType\":\"contractWETH\",\"name\":\"weth_\",\"type\":\"address\"},{\"internalType\":\"contractIPermit2\",\"name\":\"permit2_\",\"type\":\"address\"},{\"internalType\":\"contractIBunniToken\",\"name\":\"bunniTokenImplementation_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"initialOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"initialReferralRewardRecipient\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AlreadyInitialized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHub__BunniTokenNotInitialized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHub__MsgValueInsufficient\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHub__PastDeadline\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHub__Paused\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHub__Unauthorized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NewOwnerIsZeroAddress\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NoHandoverRequest\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ReentrancyGuard__ReentrantCall\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Unauthorized\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"BurnPauseFuse\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"shares\",\"type\":\"uint256\"}],\"name\":\"Deposit\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"contractIBunniToken\",\"name\":\"bunniToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"NewBunni\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"pendingOwner\",\"type\":\"address\"}],\"name\":\"OwnershipHandoverCanceled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"pendingOwner\",\"type\":\"address\"}],\"name\":\"OwnershipHandoverRequested\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"shares\",\"type\":\"uint256\"}],\"name\":\"QueueWithdraw\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint8\",\"name\":\"pauseFlags\",\"type\":\"uint8\"}],\"name\":\"SetPauseFlags\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"guy\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bool\",\"name\":\"isPauser\",\"type\":\"bool\"}],\"name\":\"SetPauser\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newReferralRewardRecipient\",\"type\":\"address\"}],\"name\":\"SetReferralRewardRecipient\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"shares\",\"type\":\"uint256\"}],\"name\":\"Withdraw\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"bunniTokenOfPool\",\"outputs\":[{\"internalType\":\"contractIBunniToken\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"burnPauseFuse\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"cancelOwnershipHandover\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"pendingOwner\",\"type\":\"address\"}],\"name\":\"completeOwnershipHandover\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"uint24\",\"name\":\"twapSecondsAgo\",\"type\":\"uint24\"},{\"internalType\":\"contractILiquidityDensityFunction\",\"name\":\"liquidityDensityFunction\",\"type\":\"address\"},{\"internalType\":\"contractIHooklet\",\"name\":\"hooklet\",\"type\":\"address\"},{\"internalType\":\"enumLDFType\",\"name\":\"ldfType\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"ldfParams\",\"type\":\"bytes32\"},{\"internalType\":\"contractIBunniHook\",\"name\":\"hooks\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"hookParams\",\"type\":\"bytes\"},{\"internalType\":\"contractERC4626\",\"name\":\"vault0\",\"type\":\"address\"},{\"internalType\":\"contractERC4626\",\"name\":\"vault1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"minRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"targetRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"maxRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"minRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"targetRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"maxRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"internalType\":\"bytes32\",\"name\":\"name\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"symbol\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"metadataURI\",\"type\":\"string\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"}],\"internalType\":\"structIBunniHub.DeployBunniTokenParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"deployBunniToken\",\"outputs\":[{\"internalType\":\"contractIBunniToken\",\"name\":\"token\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"poolKey\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"refundRecipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount0Desired\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1Desired\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount0Min\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1Min\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"vaultFee0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"vaultFee1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"referrer\",\"type\":\"address\"}],\"internalType\":\"structIBunniHub.DepositParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"deposit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"shares\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getPauseStatus\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"pauseFlags\",\"type\":\"uint8\"},{\"internalType\":\"bool\",\"name\":\"unpauseFuse\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getReferralRewardRecipient\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"inputAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"outputAmount\",\"type\":\"uint256\"}],\"name\":\"hookHandleSwap\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"hookParams\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"IdleBalance\",\"name\":\"newIdleBalance\",\"type\":\"bytes32\"}],\"name\":\"hookSetIdleBalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"idleBalance\",\"outputs\":[{\"internalType\":\"IdleBalance\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"guy\",\"type\":\"address\"}],\"name\":\"isPauser\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"}],\"name\":\"lockForRebalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"bunniSubspace\",\"type\":\"bytes32\"}],\"name\":\"nonce\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"result\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"pendingOwner\",\"type\":\"address\"}],\"name\":\"ownershipHandoverExpiresAt\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"result\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"poolBalances\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"balance0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"balance1\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBunniToken\",\"name\":\"bunniToken\",\"type\":\"address\"}],\"name\":\"poolIdOfBunniToken\",\"outputs\":[{\"internalType\":\"PoolId\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"poolInitData\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"poolParams\",\"outputs\":[{\"components\":[{\"internalType\":\"contractILiquidityDensityFunction\",\"name\":\"liquidityDensityFunction\",\"type\":\"address\"},{\"internalType\":\"contractIBunniToken\",\"name\":\"bunniToken\",\"type\":\"address\"},{\"internalType\":\"contractIHooklet\",\"name\":\"hooklet\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"twapSecondsAgo\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"ldfParams\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"hookParams\",\"type\":\"bytes\"},{\"internalType\":\"contractERC4626\",\"name\":\"vault0\",\"type\":\"address\"},{\"internalType\":\"contractERC4626\",\"name\":\"vault1\",\"type\":\"address\"},{\"internalType\":\"enumLDFType\",\"name\":\"ldfType\",\"type\":\"uint8\"},{\"internalType\":\"uint24\",\"name\":\"minRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"targetRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"maxRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"minRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"targetRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"maxRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint256\",\"name\":\"rawBalance0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"rawBalance1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reserve0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reserve1\",\"type\":\"uint256\"},{\"internalType\":\"IdleBalance\",\"name\":\"idleBalance\",\"type\":\"bytes32\"}],\"internalType\":\"structPoolState\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"poolState\",\"outputs\":[{\"components\":[{\"internalType\":\"contractILiquidityDensityFunction\",\"name\":\"liquidityDensityFunction\",\"type\":\"address\"},{\"internalType\":\"contractIBunniToken\",\"name\":\"bunniToken\",\"type\":\"address\"},{\"internalType\":\"contractIHooklet\",\"name\":\"hooklet\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"twapSecondsAgo\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"ldfParams\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"hookParams\",\"type\":\"bytes\"},{\"internalType\":\"contractERC4626\",\"name\":\"vault0\",\"type\":\"address\"},{\"internalType\":\"contractERC4626\",\"name\":\"vault1\",\"type\":\"address\"},{\"internalType\":\"enumLDFType\",\"name\":\"ldfType\",\"type\":\"uint8\"},{\"internalType\":\"uint24\",\"name\":\"minRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"targetRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"maxRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"minRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"targetRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"maxRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint256\",\"name\":\"rawBalance0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"rawBalance1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reserve0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reserve1\",\"type\":\"uint256\"},{\"internalType\":\"IdleBalance\",\"name\":\"idleBalance\",\"type\":\"bytes32\"}],\"internalType\":\"structPoolState\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"poolKey\",\"type\":\"tuple\"},{\"internalType\":\"uint200\",\"name\":\"shares\",\"type\":\"uint200\"}],\"internalType\":\"structIBunniHub.QueueWithdrawParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"queueWithdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"requestOwnershipHandover\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"pauseFlags\",\"type\":\"uint8\"}],\"name\":\"setPauseFlags\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"guy\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"status\",\"type\":\"bool\"}],\"name\":\"setPauser\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newReferralRewardRecipient\",\"type\":\"address\"}],\"name\":\"setReferralRewardRecipient\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"unlockCallback\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"}],\"name\":\"unlockForRebalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"poolKey\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"shares\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount0Min\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1Min\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"useQueuedWithdrawal\",\"type\":\"bool\"}],\"internalType\":\"structIBunniHub.WithdrawParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"withdraw\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
}

// BunniV2HubContractABI is the input ABI used to generate the binding from.
// Deprecated: Use BunniV2HubContractMetaData.ABI instead.
var BunniV2HubContractABI = BunniV2HubContractMetaData.ABI

// BunniV2HubContract is an auto generated Go binding around an Ethereum contract.
type BunniV2HubContract struct {
	BunniV2HubContractCaller     // Read-only binding to the contract
	BunniV2HubContractTransactor // Write-only binding to the contract
	BunniV2HubContractFilterer   // Log filterer for contract events
}

// BunniV2HubContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type BunniV2HubContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BunniV2HubContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BunniV2HubContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BunniV2HubContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BunniV2HubContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BunniV2HubContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BunniV2HubContractSession struct {
	Contract     *BunniV2HubContract // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// BunniV2HubContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BunniV2HubContractCallerSession struct {
	Contract *BunniV2HubContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// BunniV2HubContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BunniV2HubContractTransactorSession struct {
	Contract     *BunniV2HubContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// BunniV2HubContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type BunniV2HubContractRaw struct {
	Contract *BunniV2HubContract // Generic contract binding to access the raw methods on
}

// BunniV2HubContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BunniV2HubContractCallerRaw struct {
	Contract *BunniV2HubContractCaller // Generic read-only contract binding to access the raw methods on
}

// BunniV2HubContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BunniV2HubContractTransactorRaw struct {
	Contract *BunniV2HubContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBunniV2HubContract creates a new instance of BunniV2HubContract, bound to a specific deployed contract.
func NewBunniV2HubContract(address common.Address, backend bind.ContractBackend) (*BunniV2HubContract, error) {
	contract, err := bindBunniV2HubContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BunniV2HubContract{BunniV2HubContractCaller: BunniV2HubContractCaller{contract: contract}, BunniV2HubContractTransactor: BunniV2HubContractTransactor{contract: contract}, BunniV2HubContractFilterer: BunniV2HubContractFilterer{contract: contract}}, nil
}

// NewBunniV2HubContractCaller creates a new read-only instance of BunniV2HubContract, bound to a specific deployed contract.
func NewBunniV2HubContractCaller(address common.Address, caller bind.ContractCaller) (*BunniV2HubContractCaller, error) {
	contract, err := bindBunniV2HubContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BunniV2HubContractCaller{contract: contract}, nil
}

// NewBunniV2HubContractTransactor creates a new write-only instance of BunniV2HubContract, bound to a specific deployed contract.
func NewBunniV2HubContractTransactor(address common.Address, transactor bind.ContractTransactor) (*BunniV2HubContractTransactor, error) {
	contract, err := bindBunniV2HubContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BunniV2HubContractTransactor{contract: contract}, nil
}

// NewBunniV2HubContractFilterer creates a new log filterer instance of BunniV2HubContract, bound to a specific deployed contract.
func NewBunniV2HubContractFilterer(address common.Address, filterer bind.ContractFilterer) (*BunniV2HubContractFilterer, error) {
	contract, err := bindBunniV2HubContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BunniV2HubContractFilterer{contract: contract}, nil
}

// bindBunniV2HubContract binds a generic wrapper to an already deployed contract.
func bindBunniV2HubContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BunniV2HubContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BunniV2HubContract *BunniV2HubContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BunniV2HubContract.Contract.BunniV2HubContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BunniV2HubContract *BunniV2HubContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.BunniV2HubContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BunniV2HubContract *BunniV2HubContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.BunniV2HubContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BunniV2HubContract *BunniV2HubContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BunniV2HubContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BunniV2HubContract *BunniV2HubContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BunniV2HubContract *BunniV2HubContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.contract.Transact(opts, method, params...)
}

// BunniTokenOfPool is a free data retrieval call binding the contract method 0xa2a56697.
//
// Solidity: function bunniTokenOfPool(bytes32 poolId) view returns(address)
func (_BunniV2HubContract *BunniV2HubContractCaller) BunniTokenOfPool(opts *bind.CallOpts, poolId [32]byte) (common.Address, error) {
	var out []interface{}
	err := _BunniV2HubContract.contract.Call(opts, &out, "bunniTokenOfPool", poolId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BunniTokenOfPool is a free data retrieval call binding the contract method 0xa2a56697.
//
// Solidity: function bunniTokenOfPool(bytes32 poolId) view returns(address)
func (_BunniV2HubContract *BunniV2HubContractSession) BunniTokenOfPool(poolId [32]byte) (common.Address, error) {
	return _BunniV2HubContract.Contract.BunniTokenOfPool(&_BunniV2HubContract.CallOpts, poolId)
}

// BunniTokenOfPool is a free data retrieval call binding the contract method 0xa2a56697.
//
// Solidity: function bunniTokenOfPool(bytes32 poolId) view returns(address)
func (_BunniV2HubContract *BunniV2HubContractCallerSession) BunniTokenOfPool(poolId [32]byte) (common.Address, error) {
	return _BunniV2HubContract.Contract.BunniTokenOfPool(&_BunniV2HubContract.CallOpts, poolId)
}

// GetPauseStatus is a free data retrieval call binding the contract method 0x1d9023cb.
//
// Solidity: function getPauseStatus() view returns(uint8 pauseFlags, bool unpauseFuse)
func (_BunniV2HubContract *BunniV2HubContractCaller) GetPauseStatus(opts *bind.CallOpts) (struct {
	PauseFlags  uint8
	UnpauseFuse bool
}, error) {
	var out []interface{}
	err := _BunniV2HubContract.contract.Call(opts, &out, "getPauseStatus")

	outstruct := new(struct {
		PauseFlags  uint8
		UnpauseFuse bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.PauseFlags = *abi.ConvertType(out[0], new(uint8)).(*uint8)
	outstruct.UnpauseFuse = *abi.ConvertType(out[1], new(bool)).(*bool)

	return *outstruct, err

}

// GetPauseStatus is a free data retrieval call binding the contract method 0x1d9023cb.
//
// Solidity: function getPauseStatus() view returns(uint8 pauseFlags, bool unpauseFuse)
func (_BunniV2HubContract *BunniV2HubContractSession) GetPauseStatus() (struct {
	PauseFlags  uint8
	UnpauseFuse bool
}, error) {
	return _BunniV2HubContract.Contract.GetPauseStatus(&_BunniV2HubContract.CallOpts)
}

// GetPauseStatus is a free data retrieval call binding the contract method 0x1d9023cb.
//
// Solidity: function getPauseStatus() view returns(uint8 pauseFlags, bool unpauseFuse)
func (_BunniV2HubContract *BunniV2HubContractCallerSession) GetPauseStatus() (struct {
	PauseFlags  uint8
	UnpauseFuse bool
}, error) {
	return _BunniV2HubContract.Contract.GetPauseStatus(&_BunniV2HubContract.CallOpts)
}

// GetReferralRewardRecipient is a free data retrieval call binding the contract method 0x565f4b21.
//
// Solidity: function getReferralRewardRecipient() view returns(address)
func (_BunniV2HubContract *BunniV2HubContractCaller) GetReferralRewardRecipient(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BunniV2HubContract.contract.Call(opts, &out, "getReferralRewardRecipient")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetReferralRewardRecipient is a free data retrieval call binding the contract method 0x565f4b21.
//
// Solidity: function getReferralRewardRecipient() view returns(address)
func (_BunniV2HubContract *BunniV2HubContractSession) GetReferralRewardRecipient() (common.Address, error) {
	return _BunniV2HubContract.Contract.GetReferralRewardRecipient(&_BunniV2HubContract.CallOpts)
}

// GetReferralRewardRecipient is a free data retrieval call binding the contract method 0x565f4b21.
//
// Solidity: function getReferralRewardRecipient() view returns(address)
func (_BunniV2HubContract *BunniV2HubContractCallerSession) GetReferralRewardRecipient() (common.Address, error) {
	return _BunniV2HubContract.Contract.GetReferralRewardRecipient(&_BunniV2HubContract.CallOpts)
}

// HookParams is a free data retrieval call binding the contract method 0x129f38ea.
//
// Solidity: function hookParams(bytes32 poolId) view returns(bytes)
func (_BunniV2HubContract *BunniV2HubContractCaller) HookParams(opts *bind.CallOpts, poolId [32]byte) ([]byte, error) {
	var out []interface{}
	err := _BunniV2HubContract.contract.Call(opts, &out, "hookParams", poolId)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// HookParams is a free data retrieval call binding the contract method 0x129f38ea.
//
// Solidity: function hookParams(bytes32 poolId) view returns(bytes)
func (_BunniV2HubContract *BunniV2HubContractSession) HookParams(poolId [32]byte) ([]byte, error) {
	return _BunniV2HubContract.Contract.HookParams(&_BunniV2HubContract.CallOpts, poolId)
}

// HookParams is a free data retrieval call binding the contract method 0x129f38ea.
//
// Solidity: function hookParams(bytes32 poolId) view returns(bytes)
func (_BunniV2HubContract *BunniV2HubContractCallerSession) HookParams(poolId [32]byte) ([]byte, error) {
	return _BunniV2HubContract.Contract.HookParams(&_BunniV2HubContract.CallOpts, poolId)
}

// IdleBalance is a free data retrieval call binding the contract method 0x88dd6e53.
//
// Solidity: function idleBalance(bytes32 poolId) view returns(bytes32)
func (_BunniV2HubContract *BunniV2HubContractCaller) IdleBalance(opts *bind.CallOpts, poolId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _BunniV2HubContract.contract.Call(opts, &out, "idleBalance", poolId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// IdleBalance is a free data retrieval call binding the contract method 0x88dd6e53.
//
// Solidity: function idleBalance(bytes32 poolId) view returns(bytes32)
func (_BunniV2HubContract *BunniV2HubContractSession) IdleBalance(poolId [32]byte) ([32]byte, error) {
	return _BunniV2HubContract.Contract.IdleBalance(&_BunniV2HubContract.CallOpts, poolId)
}

// IdleBalance is a free data retrieval call binding the contract method 0x88dd6e53.
//
// Solidity: function idleBalance(bytes32 poolId) view returns(bytes32)
func (_BunniV2HubContract *BunniV2HubContractCallerSession) IdleBalance(poolId [32]byte) ([32]byte, error) {
	return _BunniV2HubContract.Contract.IdleBalance(&_BunniV2HubContract.CallOpts, poolId)
}

// IsPauser is a free data retrieval call binding the contract method 0x46fbf68e.
//
// Solidity: function isPauser(address guy) view returns(bool)
func (_BunniV2HubContract *BunniV2HubContractCaller) IsPauser(opts *bind.CallOpts, guy common.Address) (bool, error) {
	var out []interface{}
	err := _BunniV2HubContract.contract.Call(opts, &out, "isPauser", guy)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsPauser is a free data retrieval call binding the contract method 0x46fbf68e.
//
// Solidity: function isPauser(address guy) view returns(bool)
func (_BunniV2HubContract *BunniV2HubContractSession) IsPauser(guy common.Address) (bool, error) {
	return _BunniV2HubContract.Contract.IsPauser(&_BunniV2HubContract.CallOpts, guy)
}

// IsPauser is a free data retrieval call binding the contract method 0x46fbf68e.
//
// Solidity: function isPauser(address guy) view returns(bool)
func (_BunniV2HubContract *BunniV2HubContractCallerSession) IsPauser(guy common.Address) (bool, error) {
	return _BunniV2HubContract.Contract.IsPauser(&_BunniV2HubContract.CallOpts, guy)
}

// Nonce is a free data retrieval call binding the contract method 0x905da30f.
//
// Solidity: function nonce(bytes32 bunniSubspace) view returns(uint24)
func (_BunniV2HubContract *BunniV2HubContractCaller) Nonce(opts *bind.CallOpts, bunniSubspace [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _BunniV2HubContract.contract.Call(opts, &out, "nonce", bunniSubspace)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Nonce is a free data retrieval call binding the contract method 0x905da30f.
//
// Solidity: function nonce(bytes32 bunniSubspace) view returns(uint24)
func (_BunniV2HubContract *BunniV2HubContractSession) Nonce(bunniSubspace [32]byte) (*big.Int, error) {
	return _BunniV2HubContract.Contract.Nonce(&_BunniV2HubContract.CallOpts, bunniSubspace)
}

// Nonce is a free data retrieval call binding the contract method 0x905da30f.
//
// Solidity: function nonce(bytes32 bunniSubspace) view returns(uint24)
func (_BunniV2HubContract *BunniV2HubContractCallerSession) Nonce(bunniSubspace [32]byte) (*big.Int, error) {
	return _BunniV2HubContract.Contract.Nonce(&_BunniV2HubContract.CallOpts, bunniSubspace)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address result)
func (_BunniV2HubContract *BunniV2HubContractCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BunniV2HubContract.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address result)
func (_BunniV2HubContract *BunniV2HubContractSession) Owner() (common.Address, error) {
	return _BunniV2HubContract.Contract.Owner(&_BunniV2HubContract.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address result)
func (_BunniV2HubContract *BunniV2HubContractCallerSession) Owner() (common.Address, error) {
	return _BunniV2HubContract.Contract.Owner(&_BunniV2HubContract.CallOpts)
}

// OwnershipHandoverExpiresAt is a free data retrieval call binding the contract method 0xfee81cf4.
//
// Solidity: function ownershipHandoverExpiresAt(address pendingOwner) view returns(uint256 result)
func (_BunniV2HubContract *BunniV2HubContractCaller) OwnershipHandoverExpiresAt(opts *bind.CallOpts, pendingOwner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _BunniV2HubContract.contract.Call(opts, &out, "ownershipHandoverExpiresAt", pendingOwner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// OwnershipHandoverExpiresAt is a free data retrieval call binding the contract method 0xfee81cf4.
//
// Solidity: function ownershipHandoverExpiresAt(address pendingOwner) view returns(uint256 result)
func (_BunniV2HubContract *BunniV2HubContractSession) OwnershipHandoverExpiresAt(pendingOwner common.Address) (*big.Int, error) {
	return _BunniV2HubContract.Contract.OwnershipHandoverExpiresAt(&_BunniV2HubContract.CallOpts, pendingOwner)
}

// OwnershipHandoverExpiresAt is a free data retrieval call binding the contract method 0xfee81cf4.
//
// Solidity: function ownershipHandoverExpiresAt(address pendingOwner) view returns(uint256 result)
func (_BunniV2HubContract *BunniV2HubContractCallerSession) OwnershipHandoverExpiresAt(pendingOwner common.Address) (*big.Int, error) {
	return _BunniV2HubContract.Contract.OwnershipHandoverExpiresAt(&_BunniV2HubContract.CallOpts, pendingOwner)
}

// PoolBalances is a free data retrieval call binding the contract method 0x809b1f38.
//
// Solidity: function poolBalances(bytes32 poolId) view returns(uint256 balance0, uint256 balance1)
func (_BunniV2HubContract *BunniV2HubContractCaller) PoolBalances(opts *bind.CallOpts, poolId [32]byte) (struct {
	Balance0 *big.Int
	Balance1 *big.Int
}, error) {
	var out []interface{}
	err := _BunniV2HubContract.contract.Call(opts, &out, "poolBalances", poolId)

	outstruct := new(struct {
		Balance0 *big.Int
		Balance1 *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Balance0 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Balance1 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// PoolBalances is a free data retrieval call binding the contract method 0x809b1f38.
//
// Solidity: function poolBalances(bytes32 poolId) view returns(uint256 balance0, uint256 balance1)
func (_BunniV2HubContract *BunniV2HubContractSession) PoolBalances(poolId [32]byte) (struct {
	Balance0 *big.Int
	Balance1 *big.Int
}, error) {
	return _BunniV2HubContract.Contract.PoolBalances(&_BunniV2HubContract.CallOpts, poolId)
}

// PoolBalances is a free data retrieval call binding the contract method 0x809b1f38.
//
// Solidity: function poolBalances(bytes32 poolId) view returns(uint256 balance0, uint256 balance1)
func (_BunniV2HubContract *BunniV2HubContractCallerSession) PoolBalances(poolId [32]byte) (struct {
	Balance0 *big.Int
	Balance1 *big.Int
}, error) {
	return _BunniV2HubContract.Contract.PoolBalances(&_BunniV2HubContract.CallOpts, poolId)
}

// PoolIdOfBunniToken is a free data retrieval call binding the contract method 0x7676cce0.
//
// Solidity: function poolIdOfBunniToken(address bunniToken) view returns(bytes32)
func (_BunniV2HubContract *BunniV2HubContractCaller) PoolIdOfBunniToken(opts *bind.CallOpts, bunniToken common.Address) ([32]byte, error) {
	var out []interface{}
	err := _BunniV2HubContract.contract.Call(opts, &out, "poolIdOfBunniToken", bunniToken)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// PoolIdOfBunniToken is a free data retrieval call binding the contract method 0x7676cce0.
//
// Solidity: function poolIdOfBunniToken(address bunniToken) view returns(bytes32)
func (_BunniV2HubContract *BunniV2HubContractSession) PoolIdOfBunniToken(bunniToken common.Address) ([32]byte, error) {
	return _BunniV2HubContract.Contract.PoolIdOfBunniToken(&_BunniV2HubContract.CallOpts, bunniToken)
}

// PoolIdOfBunniToken is a free data retrieval call binding the contract method 0x7676cce0.
//
// Solidity: function poolIdOfBunniToken(address bunniToken) view returns(bytes32)
func (_BunniV2HubContract *BunniV2HubContractCallerSession) PoolIdOfBunniToken(bunniToken common.Address) ([32]byte, error) {
	return _BunniV2HubContract.Contract.PoolIdOfBunniToken(&_BunniV2HubContract.CallOpts, bunniToken)
}

// PoolInitData is a free data retrieval call binding the contract method 0xf0960848.
//
// Solidity: function poolInitData() view returns(bytes)
func (_BunniV2HubContract *BunniV2HubContractCaller) PoolInitData(opts *bind.CallOpts) ([]byte, error) {
	var out []interface{}
	err := _BunniV2HubContract.contract.Call(opts, &out, "poolInitData")

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// PoolInitData is a free data retrieval call binding the contract method 0xf0960848.
//
// Solidity: function poolInitData() view returns(bytes)
func (_BunniV2HubContract *BunniV2HubContractSession) PoolInitData() ([]byte, error) {
	return _BunniV2HubContract.Contract.PoolInitData(&_BunniV2HubContract.CallOpts)
}

// PoolInitData is a free data retrieval call binding the contract method 0xf0960848.
//
// Solidity: function poolInitData() view returns(bytes)
func (_BunniV2HubContract *BunniV2HubContractCallerSession) PoolInitData() ([]byte, error) {
	return _BunniV2HubContract.Contract.PoolInitData(&_BunniV2HubContract.CallOpts)
}

// PoolParams is a free data retrieval call binding the contract method 0xa0fd3f7e.
//
// Solidity: function poolParams(bytes32 poolId) view returns((address,address,address,uint24,bytes32,bytes,address,address,uint8,uint24,uint24,uint24,uint24,uint24,uint24,uint256,uint256,uint256,uint256,bytes32))
func (_BunniV2HubContract *BunniV2HubContractCaller) PoolParams(opts *bind.CallOpts, poolId [32]byte) (PoolState, error) {
	var out []interface{}
	err := _BunniV2HubContract.contract.Call(opts, &out, "poolParams", poolId)

	if err != nil {
		return *new(PoolState), err
	}

	out0 := *abi.ConvertType(out[0], new(PoolState)).(*PoolState)

	return out0, err

}

// PoolParams is a free data retrieval call binding the contract method 0xa0fd3f7e.
//
// Solidity: function poolParams(bytes32 poolId) view returns((address,address,address,uint24,bytes32,bytes,address,address,uint8,uint24,uint24,uint24,uint24,uint24,uint24,uint256,uint256,uint256,uint256,bytes32))
func (_BunniV2HubContract *BunniV2HubContractSession) PoolParams(poolId [32]byte) (PoolState, error) {
	return _BunniV2HubContract.Contract.PoolParams(&_BunniV2HubContract.CallOpts, poolId)
}

// PoolParams is a free data retrieval call binding the contract method 0xa0fd3f7e.
//
// Solidity: function poolParams(bytes32 poolId) view returns((address,address,address,uint24,bytes32,bytes,address,address,uint8,uint24,uint24,uint24,uint24,uint24,uint24,uint256,uint256,uint256,uint256,bytes32))
func (_BunniV2HubContract *BunniV2HubContractCallerSession) PoolParams(poolId [32]byte) (PoolState, error) {
	return _BunniV2HubContract.Contract.PoolParams(&_BunniV2HubContract.CallOpts, poolId)
}

// PoolState is a free data retrieval call binding the contract method 0xe0b01bac.
//
// Solidity: function poolState(bytes32 poolId) view returns((address,address,address,uint24,bytes32,bytes,address,address,uint8,uint24,uint24,uint24,uint24,uint24,uint24,uint256,uint256,uint256,uint256,bytes32))
func (_BunniV2HubContract *BunniV2HubContractCaller) PoolState(opts *bind.CallOpts, poolId [32]byte) (PoolState, error) {
	var out []interface{}
	err := _BunniV2HubContract.contract.Call(opts, &out, "poolState", poolId)

	if err != nil {
		return *new(PoolState), err
	}

	out0 := *abi.ConvertType(out[0], new(PoolState)).(*PoolState)

	return out0, err

}

// PoolState is a free data retrieval call binding the contract method 0xe0b01bac.
//
// Solidity: function poolState(bytes32 poolId) view returns((address,address,address,uint24,bytes32,bytes,address,address,uint8,uint24,uint24,uint24,uint24,uint24,uint24,uint256,uint256,uint256,uint256,bytes32))
func (_BunniV2HubContract *BunniV2HubContractSession) PoolState(poolId [32]byte) (PoolState, error) {
	return _BunniV2HubContract.Contract.PoolState(&_BunniV2HubContract.CallOpts, poolId)
}

// PoolState is a free data retrieval call binding the contract method 0xe0b01bac.
//
// Solidity: function poolState(bytes32 poolId) view returns((address,address,address,uint24,bytes32,bytes,address,address,uint8,uint24,uint24,uint24,uint24,uint24,uint24,uint256,uint256,uint256,uint256,bytes32))
func (_BunniV2HubContract *BunniV2HubContractCallerSession) PoolState(poolId [32]byte) (PoolState, error) {
	return _BunniV2HubContract.Contract.PoolState(&_BunniV2HubContract.CallOpts, poolId)
}

// BurnPauseFuse is a paid mutator transaction binding the contract method 0x1ed08cb9.
//
// Solidity: function burnPauseFuse() returns()
func (_BunniV2HubContract *BunniV2HubContractTransactor) BurnPauseFuse(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "burnPauseFuse")
}

// BurnPauseFuse is a paid mutator transaction binding the contract method 0x1ed08cb9.
//
// Solidity: function burnPauseFuse() returns()
func (_BunniV2HubContract *BunniV2HubContractSession) BurnPauseFuse() (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.BurnPauseFuse(&_BunniV2HubContract.TransactOpts)
}

// BurnPauseFuse is a paid mutator transaction binding the contract method 0x1ed08cb9.
//
// Solidity: function burnPauseFuse() returns()
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) BurnPauseFuse() (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.BurnPauseFuse(&_BunniV2HubContract.TransactOpts)
}

// CancelOwnershipHandover is a paid mutator transaction binding the contract method 0x54d1f13d.
//
// Solidity: function cancelOwnershipHandover() payable returns()
func (_BunniV2HubContract *BunniV2HubContractTransactor) CancelOwnershipHandover(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "cancelOwnershipHandover")
}

// CancelOwnershipHandover is a paid mutator transaction binding the contract method 0x54d1f13d.
//
// Solidity: function cancelOwnershipHandover() payable returns()
func (_BunniV2HubContract *BunniV2HubContractSession) CancelOwnershipHandover() (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.CancelOwnershipHandover(&_BunniV2HubContract.TransactOpts)
}

// CancelOwnershipHandover is a paid mutator transaction binding the contract method 0x54d1f13d.
//
// Solidity: function cancelOwnershipHandover() payable returns()
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) CancelOwnershipHandover() (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.CancelOwnershipHandover(&_BunniV2HubContract.TransactOpts)
}

// CompleteOwnershipHandover is a paid mutator transaction binding the contract method 0xf04e283e.
//
// Solidity: function completeOwnershipHandover(address pendingOwner) payable returns()
func (_BunniV2HubContract *BunniV2HubContractTransactor) CompleteOwnershipHandover(opts *bind.TransactOpts, pendingOwner common.Address) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "completeOwnershipHandover", pendingOwner)
}

// CompleteOwnershipHandover is a paid mutator transaction binding the contract method 0xf04e283e.
//
// Solidity: function completeOwnershipHandover(address pendingOwner) payable returns()
func (_BunniV2HubContract *BunniV2HubContractSession) CompleteOwnershipHandover(pendingOwner common.Address) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.CompleteOwnershipHandover(&_BunniV2HubContract.TransactOpts, pendingOwner)
}

// CompleteOwnershipHandover is a paid mutator transaction binding the contract method 0xf04e283e.
//
// Solidity: function completeOwnershipHandover(address pendingOwner) payable returns()
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) CompleteOwnershipHandover(pendingOwner common.Address) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.CompleteOwnershipHandover(&_BunniV2HubContract.TransactOpts, pendingOwner)
}

// DeployBunniToken is a paid mutator transaction binding the contract method 0xe56ba808.
//
// Solidity: function deployBunniToken((address,address,int24,uint24,address,address,uint8,bytes32,address,bytes,address,address,uint24,uint24,uint24,uint24,uint24,uint24,uint160,bytes32,bytes32,address,string,bytes32) params) returns(address token, (address,address,uint24,int24,address) key)
func (_BunniV2HubContract *BunniV2HubContractTransactor) DeployBunniToken(opts *bind.TransactOpts, params IBunniHubDeployBunniTokenParams) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "deployBunniToken", params)
}

// DeployBunniToken is a paid mutator transaction binding the contract method 0xe56ba808.
//
// Solidity: function deployBunniToken((address,address,int24,uint24,address,address,uint8,bytes32,address,bytes,address,address,uint24,uint24,uint24,uint24,uint24,uint24,uint160,bytes32,bytes32,address,string,bytes32) params) returns(address token, (address,address,uint24,int24,address) key)
func (_BunniV2HubContract *BunniV2HubContractSession) DeployBunniToken(params IBunniHubDeployBunniTokenParams) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.DeployBunniToken(&_BunniV2HubContract.TransactOpts, params)
}

// DeployBunniToken is a paid mutator transaction binding the contract method 0xe56ba808.
//
// Solidity: function deployBunniToken((address,address,int24,uint24,address,address,uint8,bytes32,address,bytes,address,address,uint24,uint24,uint24,uint24,uint24,uint24,uint160,bytes32,bytes32,address,string,bytes32) params) returns(address token, (address,address,uint24,int24,address) key)
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) DeployBunniToken(params IBunniHubDeployBunniTokenParams) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.DeployBunniToken(&_BunniV2HubContract.TransactOpts, params)
}

// Deposit is a paid mutator transaction binding the contract method 0xf69da336.
//
// Solidity: function deposit(((address,address,uint24,int24,address),address,address,uint256,uint256,uint256,uint256,uint256,uint256,uint256,address) params) payable returns(uint256 shares, uint256 amount0, uint256 amount1)
func (_BunniV2HubContract *BunniV2HubContractTransactor) Deposit(opts *bind.TransactOpts, params IBunniHubDepositParams) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "deposit", params)
}

// Deposit is a paid mutator transaction binding the contract method 0xf69da336.
//
// Solidity: function deposit(((address,address,uint24,int24,address),address,address,uint256,uint256,uint256,uint256,uint256,uint256,uint256,address) params) payable returns(uint256 shares, uint256 amount0, uint256 amount1)
func (_BunniV2HubContract *BunniV2HubContractSession) Deposit(params IBunniHubDepositParams) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.Deposit(&_BunniV2HubContract.TransactOpts, params)
}

// Deposit is a paid mutator transaction binding the contract method 0xf69da336.
//
// Solidity: function deposit(((address,address,uint24,int24,address),address,address,uint256,uint256,uint256,uint256,uint256,uint256,uint256,address) params) payable returns(uint256 shares, uint256 amount0, uint256 amount1)
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) Deposit(params IBunniHubDepositParams) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.Deposit(&_BunniV2HubContract.TransactOpts, params)
}

// HookHandleSwap is a paid mutator transaction binding the contract method 0xf89ee44e.
//
// Solidity: function hookHandleSwap((address,address,uint24,int24,address) key, bool zeroForOne, uint256 inputAmount, uint256 outputAmount) returns()
func (_BunniV2HubContract *BunniV2HubContractTransactor) HookHandleSwap(opts *bind.TransactOpts, key PoolKey, zeroForOne bool, inputAmount *big.Int, outputAmount *big.Int) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "hookHandleSwap", key, zeroForOne, inputAmount, outputAmount)
}

// HookHandleSwap is a paid mutator transaction binding the contract method 0xf89ee44e.
//
// Solidity: function hookHandleSwap((address,address,uint24,int24,address) key, bool zeroForOne, uint256 inputAmount, uint256 outputAmount) returns()
func (_BunniV2HubContract *BunniV2HubContractSession) HookHandleSwap(key PoolKey, zeroForOne bool, inputAmount *big.Int, outputAmount *big.Int) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.HookHandleSwap(&_BunniV2HubContract.TransactOpts, key, zeroForOne, inputAmount, outputAmount)
}

// HookHandleSwap is a paid mutator transaction binding the contract method 0xf89ee44e.
//
// Solidity: function hookHandleSwap((address,address,uint24,int24,address) key, bool zeroForOne, uint256 inputAmount, uint256 outputAmount) returns()
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) HookHandleSwap(key PoolKey, zeroForOne bool, inputAmount *big.Int, outputAmount *big.Int) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.HookHandleSwap(&_BunniV2HubContract.TransactOpts, key, zeroForOne, inputAmount, outputAmount)
}

// HookSetIdleBalance is a paid mutator transaction binding the contract method 0xef760335.
//
// Solidity: function hookSetIdleBalance((address,address,uint24,int24,address) key, bytes32 newIdleBalance) returns()
func (_BunniV2HubContract *BunniV2HubContractTransactor) HookSetIdleBalance(opts *bind.TransactOpts, key PoolKey, newIdleBalance [32]byte) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "hookSetIdleBalance", key, newIdleBalance)
}

// HookSetIdleBalance is a paid mutator transaction binding the contract method 0xef760335.
//
// Solidity: function hookSetIdleBalance((address,address,uint24,int24,address) key, bytes32 newIdleBalance) returns()
func (_BunniV2HubContract *BunniV2HubContractSession) HookSetIdleBalance(key PoolKey, newIdleBalance [32]byte) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.HookSetIdleBalance(&_BunniV2HubContract.TransactOpts, key, newIdleBalance)
}

// HookSetIdleBalance is a paid mutator transaction binding the contract method 0xef760335.
//
// Solidity: function hookSetIdleBalance((address,address,uint24,int24,address) key, bytes32 newIdleBalance) returns()
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) HookSetIdleBalance(key PoolKey, newIdleBalance [32]byte) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.HookSetIdleBalance(&_BunniV2HubContract.TransactOpts, key, newIdleBalance)
}

// LockForRebalance is a paid mutator transaction binding the contract method 0x3fac6506.
//
// Solidity: function lockForRebalance((address,address,uint24,int24,address) key) returns()
func (_BunniV2HubContract *BunniV2HubContractTransactor) LockForRebalance(opts *bind.TransactOpts, key PoolKey) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "lockForRebalance", key)
}

// LockForRebalance is a paid mutator transaction binding the contract method 0x3fac6506.
//
// Solidity: function lockForRebalance((address,address,uint24,int24,address) key) returns()
func (_BunniV2HubContract *BunniV2HubContractSession) LockForRebalance(key PoolKey) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.LockForRebalance(&_BunniV2HubContract.TransactOpts, key)
}

// LockForRebalance is a paid mutator transaction binding the contract method 0x3fac6506.
//
// Solidity: function lockForRebalance((address,address,uint24,int24,address) key) returns()
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) LockForRebalance(key PoolKey) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.LockForRebalance(&_BunniV2HubContract.TransactOpts, key)
}

// QueueWithdraw is a paid mutator transaction binding the contract method 0x5658d0b4.
//
// Solidity: function queueWithdraw(((address,address,uint24,int24,address),uint200) params) returns()
func (_BunniV2HubContract *BunniV2HubContractTransactor) QueueWithdraw(opts *bind.TransactOpts, params IBunniHubQueueWithdrawParams) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "queueWithdraw", params)
}

// QueueWithdraw is a paid mutator transaction binding the contract method 0x5658d0b4.
//
// Solidity: function queueWithdraw(((address,address,uint24,int24,address),uint200) params) returns()
func (_BunniV2HubContract *BunniV2HubContractSession) QueueWithdraw(params IBunniHubQueueWithdrawParams) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.QueueWithdraw(&_BunniV2HubContract.TransactOpts, params)
}

// QueueWithdraw is a paid mutator transaction binding the contract method 0x5658d0b4.
//
// Solidity: function queueWithdraw(((address,address,uint24,int24,address),uint200) params) returns()
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) QueueWithdraw(params IBunniHubQueueWithdrawParams) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.QueueWithdraw(&_BunniV2HubContract.TransactOpts, params)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() payable returns()
func (_BunniV2HubContract *BunniV2HubContractTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() payable returns()
func (_BunniV2HubContract *BunniV2HubContractSession) RenounceOwnership() (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.RenounceOwnership(&_BunniV2HubContract.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() payable returns()
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.RenounceOwnership(&_BunniV2HubContract.TransactOpts)
}

// RequestOwnershipHandover is a paid mutator transaction binding the contract method 0x25692962.
//
// Solidity: function requestOwnershipHandover() payable returns()
func (_BunniV2HubContract *BunniV2HubContractTransactor) RequestOwnershipHandover(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "requestOwnershipHandover")
}

// RequestOwnershipHandover is a paid mutator transaction binding the contract method 0x25692962.
//
// Solidity: function requestOwnershipHandover() payable returns()
func (_BunniV2HubContract *BunniV2HubContractSession) RequestOwnershipHandover() (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.RequestOwnershipHandover(&_BunniV2HubContract.TransactOpts)
}

// RequestOwnershipHandover is a paid mutator transaction binding the contract method 0x25692962.
//
// Solidity: function requestOwnershipHandover() payable returns()
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) RequestOwnershipHandover() (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.RequestOwnershipHandover(&_BunniV2HubContract.TransactOpts)
}

// SetPauseFlags is a paid mutator transaction binding the contract method 0xa56dd053.
//
// Solidity: function setPauseFlags(uint8 pauseFlags) returns()
func (_BunniV2HubContract *BunniV2HubContractTransactor) SetPauseFlags(opts *bind.TransactOpts, pauseFlags uint8) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "setPauseFlags", pauseFlags)
}

// SetPauseFlags is a paid mutator transaction binding the contract method 0xa56dd053.
//
// Solidity: function setPauseFlags(uint8 pauseFlags) returns()
func (_BunniV2HubContract *BunniV2HubContractSession) SetPauseFlags(pauseFlags uint8) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.SetPauseFlags(&_BunniV2HubContract.TransactOpts, pauseFlags)
}

// SetPauseFlags is a paid mutator transaction binding the contract method 0xa56dd053.
//
// Solidity: function setPauseFlags(uint8 pauseFlags) returns()
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) SetPauseFlags(pauseFlags uint8) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.SetPauseFlags(&_BunniV2HubContract.TransactOpts, pauseFlags)
}

// SetPauser is a paid mutator transaction binding the contract method 0x7180c8ca.
//
// Solidity: function setPauser(address guy, bool status) returns()
func (_BunniV2HubContract *BunniV2HubContractTransactor) SetPauser(opts *bind.TransactOpts, guy common.Address, status bool) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "setPauser", guy, status)
}

// SetPauser is a paid mutator transaction binding the contract method 0x7180c8ca.
//
// Solidity: function setPauser(address guy, bool status) returns()
func (_BunniV2HubContract *BunniV2HubContractSession) SetPauser(guy common.Address, status bool) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.SetPauser(&_BunniV2HubContract.TransactOpts, guy, status)
}

// SetPauser is a paid mutator transaction binding the contract method 0x7180c8ca.
//
// Solidity: function setPauser(address guy, bool status) returns()
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) SetPauser(guy common.Address, status bool) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.SetPauser(&_BunniV2HubContract.TransactOpts, guy, status)
}

// SetReferralRewardRecipient is a paid mutator transaction binding the contract method 0xcd639491.
//
// Solidity: function setReferralRewardRecipient(address newReferralRewardRecipient) returns()
func (_BunniV2HubContract *BunniV2HubContractTransactor) SetReferralRewardRecipient(opts *bind.TransactOpts, newReferralRewardRecipient common.Address) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "setReferralRewardRecipient", newReferralRewardRecipient)
}

// SetReferralRewardRecipient is a paid mutator transaction binding the contract method 0xcd639491.
//
// Solidity: function setReferralRewardRecipient(address newReferralRewardRecipient) returns()
func (_BunniV2HubContract *BunniV2HubContractSession) SetReferralRewardRecipient(newReferralRewardRecipient common.Address) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.SetReferralRewardRecipient(&_BunniV2HubContract.TransactOpts, newReferralRewardRecipient)
}

// SetReferralRewardRecipient is a paid mutator transaction binding the contract method 0xcd639491.
//
// Solidity: function setReferralRewardRecipient(address newReferralRewardRecipient) returns()
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) SetReferralRewardRecipient(newReferralRewardRecipient common.Address) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.SetReferralRewardRecipient(&_BunniV2HubContract.TransactOpts, newReferralRewardRecipient)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) payable returns()
func (_BunniV2HubContract *BunniV2HubContractTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) payable returns()
func (_BunniV2HubContract *BunniV2HubContractSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.TransferOwnership(&_BunniV2HubContract.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) payable returns()
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.TransferOwnership(&_BunniV2HubContract.TransactOpts, newOwner)
}

// UnlockCallback is a paid mutator transaction binding the contract method 0x91dd7346.
//
// Solidity: function unlockCallback(bytes data) returns(bytes)
func (_BunniV2HubContract *BunniV2HubContractTransactor) UnlockCallback(opts *bind.TransactOpts, data []byte) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "unlockCallback", data)
}

// UnlockCallback is a paid mutator transaction binding the contract method 0x91dd7346.
//
// Solidity: function unlockCallback(bytes data) returns(bytes)
func (_BunniV2HubContract *BunniV2HubContractSession) UnlockCallback(data []byte) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.UnlockCallback(&_BunniV2HubContract.TransactOpts, data)
}

// UnlockCallback is a paid mutator transaction binding the contract method 0x91dd7346.
//
// Solidity: function unlockCallback(bytes data) returns(bytes)
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) UnlockCallback(data []byte) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.UnlockCallback(&_BunniV2HubContract.TransactOpts, data)
}

// UnlockForRebalance is a paid mutator transaction binding the contract method 0x9445c4a8.
//
// Solidity: function unlockForRebalance((address,address,uint24,int24,address) key) returns()
func (_BunniV2HubContract *BunniV2HubContractTransactor) UnlockForRebalance(opts *bind.TransactOpts, key PoolKey) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "unlockForRebalance", key)
}

// UnlockForRebalance is a paid mutator transaction binding the contract method 0x9445c4a8.
//
// Solidity: function unlockForRebalance((address,address,uint24,int24,address) key) returns()
func (_BunniV2HubContract *BunniV2HubContractSession) UnlockForRebalance(key PoolKey) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.UnlockForRebalance(&_BunniV2HubContract.TransactOpts, key)
}

// UnlockForRebalance is a paid mutator transaction binding the contract method 0x9445c4a8.
//
// Solidity: function unlockForRebalance((address,address,uint24,int24,address) key) returns()
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) UnlockForRebalance(key PoolKey) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.UnlockForRebalance(&_BunniV2HubContract.TransactOpts, key)
}

// Withdraw is a paid mutator transaction binding the contract method 0x5d4a505e.
//
// Solidity: function withdraw(((address,address,uint24,int24,address),address,uint256,uint256,uint256,uint256,bool) params) returns(uint256 amount0, uint256 amount1)
func (_BunniV2HubContract *BunniV2HubContractTransactor) Withdraw(opts *bind.TransactOpts, params IBunniHubWithdrawParams) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.Transact(opts, "withdraw", params)
}

// Withdraw is a paid mutator transaction binding the contract method 0x5d4a505e.
//
// Solidity: function withdraw(((address,address,uint24,int24,address),address,uint256,uint256,uint256,uint256,bool) params) returns(uint256 amount0, uint256 amount1)
func (_BunniV2HubContract *BunniV2HubContractSession) Withdraw(params IBunniHubWithdrawParams) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.Withdraw(&_BunniV2HubContract.TransactOpts, params)
}

// Withdraw is a paid mutator transaction binding the contract method 0x5d4a505e.
//
// Solidity: function withdraw(((address,address,uint24,int24,address),address,uint256,uint256,uint256,uint256,bool) params) returns(uint256 amount0, uint256 amount1)
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) Withdraw(params IBunniHubWithdrawParams) (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.Withdraw(&_BunniV2HubContract.TransactOpts, params)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BunniV2HubContract *BunniV2HubContractTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BunniV2HubContract.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BunniV2HubContract *BunniV2HubContractSession) Receive() (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.Receive(&_BunniV2HubContract.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BunniV2HubContract *BunniV2HubContractTransactorSession) Receive() (*types.Transaction, error) {
	return _BunniV2HubContract.Contract.Receive(&_BunniV2HubContract.TransactOpts)
}

// BunniV2HubContractBurnPauseFuseIterator is returned from FilterBurnPauseFuse and is used to iterate over the raw logs and unpacked data for BurnPauseFuse events raised by the BunniV2HubContract contract.
type BunniV2HubContractBurnPauseFuseIterator struct {
	Event *BunniV2HubContractBurnPauseFuse // Event containing the contract specifics and raw log

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
func (it *BunniV2HubContractBurnPauseFuseIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BunniV2HubContractBurnPauseFuse)
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
		it.Event = new(BunniV2HubContractBurnPauseFuse)
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
func (it *BunniV2HubContractBurnPauseFuseIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BunniV2HubContractBurnPauseFuseIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BunniV2HubContractBurnPauseFuse represents a BurnPauseFuse event raised by the BunniV2HubContract contract.
type BunniV2HubContractBurnPauseFuse struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterBurnPauseFuse is a free log retrieval operation binding the contract event 0xa4058ba547bb832da5ae671cc4d748c09c98c85226ec320325d641a1a3d64adf.
//
// Solidity: event BurnPauseFuse()
func (_BunniV2HubContract *BunniV2HubContractFilterer) FilterBurnPauseFuse(opts *bind.FilterOpts) (*BunniV2HubContractBurnPauseFuseIterator, error) {

	logs, sub, err := _BunniV2HubContract.contract.FilterLogs(opts, "BurnPauseFuse")
	if err != nil {
		return nil, err
	}
	return &BunniV2HubContractBurnPauseFuseIterator{contract: _BunniV2HubContract.contract, event: "BurnPauseFuse", logs: logs, sub: sub}, nil
}

// WatchBurnPauseFuse is a free log subscription operation binding the contract event 0xa4058ba547bb832da5ae671cc4d748c09c98c85226ec320325d641a1a3d64adf.
//
// Solidity: event BurnPauseFuse()
func (_BunniV2HubContract *BunniV2HubContractFilterer) WatchBurnPauseFuse(opts *bind.WatchOpts, sink chan<- *BunniV2HubContractBurnPauseFuse) (event.Subscription, error) {

	logs, sub, err := _BunniV2HubContract.contract.WatchLogs(opts, "BurnPauseFuse")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BunniV2HubContractBurnPauseFuse)
				if err := _BunniV2HubContract.contract.UnpackLog(event, "BurnPauseFuse", log); err != nil {
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

// ParseBurnPauseFuse is a log parse operation binding the contract event 0xa4058ba547bb832da5ae671cc4d748c09c98c85226ec320325d641a1a3d64adf.
//
// Solidity: event BurnPauseFuse()
func (_BunniV2HubContract *BunniV2HubContractFilterer) ParseBurnPauseFuse(log types.Log) (*BunniV2HubContractBurnPauseFuse, error) {
	event := new(BunniV2HubContractBurnPauseFuse)
	if err := _BunniV2HubContract.contract.UnpackLog(event, "BurnPauseFuse", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BunniV2HubContractDepositIterator is returned from FilterDeposit and is used to iterate over the raw logs and unpacked data for Deposit events raised by the BunniV2HubContract contract.
type BunniV2HubContractDepositIterator struct {
	Event *BunniV2HubContractDeposit // Event containing the contract specifics and raw log

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
func (it *BunniV2HubContractDepositIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BunniV2HubContractDeposit)
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
		it.Event = new(BunniV2HubContractDeposit)
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
func (it *BunniV2HubContractDepositIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BunniV2HubContractDepositIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BunniV2HubContractDeposit represents a Deposit event raised by the BunniV2HubContract contract.
type BunniV2HubContractDeposit struct {
	Sender    common.Address
	Recipient common.Address
	PoolId    [32]byte
	Amount0   *big.Int
	Amount1   *big.Int
	Shares    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterDeposit is a free log retrieval operation binding the contract event 0xb18066d48ef2004e3dcc96ec09f8e738f9e8692565ae7108c2b593f8199af466.
//
// Solidity: event Deposit(address indexed sender, address indexed recipient, bytes32 indexed poolId, uint256 amount0, uint256 amount1, uint256 shares)
func (_BunniV2HubContract *BunniV2HubContractFilterer) FilterDeposit(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address, poolId [][32]byte) (*BunniV2HubContractDepositIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var poolIdRule []interface{}
	for _, poolIdItem := range poolId {
		poolIdRule = append(poolIdRule, poolIdItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.FilterLogs(opts, "Deposit", senderRule, recipientRule, poolIdRule)
	if err != nil {
		return nil, err
	}
	return &BunniV2HubContractDepositIterator{contract: _BunniV2HubContract.contract, event: "Deposit", logs: logs, sub: sub}, nil
}

// WatchDeposit is a free log subscription operation binding the contract event 0xb18066d48ef2004e3dcc96ec09f8e738f9e8692565ae7108c2b593f8199af466.
//
// Solidity: event Deposit(address indexed sender, address indexed recipient, bytes32 indexed poolId, uint256 amount0, uint256 amount1, uint256 shares)
func (_BunniV2HubContract *BunniV2HubContractFilterer) WatchDeposit(opts *bind.WatchOpts, sink chan<- *BunniV2HubContractDeposit, sender []common.Address, recipient []common.Address, poolId [][32]byte) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var poolIdRule []interface{}
	for _, poolIdItem := range poolId {
		poolIdRule = append(poolIdRule, poolIdItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.WatchLogs(opts, "Deposit", senderRule, recipientRule, poolIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BunniV2HubContractDeposit)
				if err := _BunniV2HubContract.contract.UnpackLog(event, "Deposit", log); err != nil {
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

// ParseDeposit is a log parse operation binding the contract event 0xb18066d48ef2004e3dcc96ec09f8e738f9e8692565ae7108c2b593f8199af466.
//
// Solidity: event Deposit(address indexed sender, address indexed recipient, bytes32 indexed poolId, uint256 amount0, uint256 amount1, uint256 shares)
func (_BunniV2HubContract *BunniV2HubContractFilterer) ParseDeposit(log types.Log) (*BunniV2HubContractDeposit, error) {
	event := new(BunniV2HubContractDeposit)
	if err := _BunniV2HubContract.contract.UnpackLog(event, "Deposit", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BunniV2HubContractNewBunniIterator is returned from FilterNewBunni and is used to iterate over the raw logs and unpacked data for NewBunni events raised by the BunniV2HubContract contract.
type BunniV2HubContractNewBunniIterator struct {
	Event *BunniV2HubContractNewBunni // Event containing the contract specifics and raw log

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
func (it *BunniV2HubContractNewBunniIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BunniV2HubContractNewBunni)
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
		it.Event = new(BunniV2HubContractNewBunni)
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
func (it *BunniV2HubContractNewBunniIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BunniV2HubContractNewBunniIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BunniV2HubContractNewBunni represents a NewBunni event raised by the BunniV2HubContract contract.
type BunniV2HubContractNewBunni struct {
	BunniToken common.Address
	PoolId     [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterNewBunni is a free log retrieval operation binding the contract event 0x3ba5df143a8e4c83b7cb37037a87ae0cfb0f8c3a784d4a10e0b329d5706dce1a.
//
// Solidity: event NewBunni(address indexed bunniToken, bytes32 indexed poolId)
func (_BunniV2HubContract *BunniV2HubContractFilterer) FilterNewBunni(opts *bind.FilterOpts, bunniToken []common.Address, poolId [][32]byte) (*BunniV2HubContractNewBunniIterator, error) {

	var bunniTokenRule []interface{}
	for _, bunniTokenItem := range bunniToken {
		bunniTokenRule = append(bunniTokenRule, bunniTokenItem)
	}
	var poolIdRule []interface{}
	for _, poolIdItem := range poolId {
		poolIdRule = append(poolIdRule, poolIdItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.FilterLogs(opts, "NewBunni", bunniTokenRule, poolIdRule)
	if err != nil {
		return nil, err
	}
	return &BunniV2HubContractNewBunniIterator{contract: _BunniV2HubContract.contract, event: "NewBunni", logs: logs, sub: sub}, nil
}

// WatchNewBunni is a free log subscription operation binding the contract event 0x3ba5df143a8e4c83b7cb37037a87ae0cfb0f8c3a784d4a10e0b329d5706dce1a.
//
// Solidity: event NewBunni(address indexed bunniToken, bytes32 indexed poolId)
func (_BunniV2HubContract *BunniV2HubContractFilterer) WatchNewBunni(opts *bind.WatchOpts, sink chan<- *BunniV2HubContractNewBunni, bunniToken []common.Address, poolId [][32]byte) (event.Subscription, error) {

	var bunniTokenRule []interface{}
	for _, bunniTokenItem := range bunniToken {
		bunniTokenRule = append(bunniTokenRule, bunniTokenItem)
	}
	var poolIdRule []interface{}
	for _, poolIdItem := range poolId {
		poolIdRule = append(poolIdRule, poolIdItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.WatchLogs(opts, "NewBunni", bunniTokenRule, poolIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BunniV2HubContractNewBunni)
				if err := _BunniV2HubContract.contract.UnpackLog(event, "NewBunni", log); err != nil {
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

// ParseNewBunni is a log parse operation binding the contract event 0x3ba5df143a8e4c83b7cb37037a87ae0cfb0f8c3a784d4a10e0b329d5706dce1a.
//
// Solidity: event NewBunni(address indexed bunniToken, bytes32 indexed poolId)
func (_BunniV2HubContract *BunniV2HubContractFilterer) ParseNewBunni(log types.Log) (*BunniV2HubContractNewBunni, error) {
	event := new(BunniV2HubContractNewBunni)
	if err := _BunniV2HubContract.contract.UnpackLog(event, "NewBunni", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BunniV2HubContractOwnershipHandoverCanceledIterator is returned from FilterOwnershipHandoverCanceled and is used to iterate over the raw logs and unpacked data for OwnershipHandoverCanceled events raised by the BunniV2HubContract contract.
type BunniV2HubContractOwnershipHandoverCanceledIterator struct {
	Event *BunniV2HubContractOwnershipHandoverCanceled // Event containing the contract specifics and raw log

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
func (it *BunniV2HubContractOwnershipHandoverCanceledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BunniV2HubContractOwnershipHandoverCanceled)
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
		it.Event = new(BunniV2HubContractOwnershipHandoverCanceled)
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
func (it *BunniV2HubContractOwnershipHandoverCanceledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BunniV2HubContractOwnershipHandoverCanceledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BunniV2HubContractOwnershipHandoverCanceled represents a OwnershipHandoverCanceled event raised by the BunniV2HubContract contract.
type BunniV2HubContractOwnershipHandoverCanceled struct {
	PendingOwner common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterOwnershipHandoverCanceled is a free log retrieval operation binding the contract event 0xfa7b8eab7da67f412cc9575ed43464468f9bfbae89d1675917346ca6d8fe3c92.
//
// Solidity: event OwnershipHandoverCanceled(address indexed pendingOwner)
func (_BunniV2HubContract *BunniV2HubContractFilterer) FilterOwnershipHandoverCanceled(opts *bind.FilterOpts, pendingOwner []common.Address) (*BunniV2HubContractOwnershipHandoverCanceledIterator, error) {

	var pendingOwnerRule []interface{}
	for _, pendingOwnerItem := range pendingOwner {
		pendingOwnerRule = append(pendingOwnerRule, pendingOwnerItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.FilterLogs(opts, "OwnershipHandoverCanceled", pendingOwnerRule)
	if err != nil {
		return nil, err
	}
	return &BunniV2HubContractOwnershipHandoverCanceledIterator{contract: _BunniV2HubContract.contract, event: "OwnershipHandoverCanceled", logs: logs, sub: sub}, nil
}

// WatchOwnershipHandoverCanceled is a free log subscription operation binding the contract event 0xfa7b8eab7da67f412cc9575ed43464468f9bfbae89d1675917346ca6d8fe3c92.
//
// Solidity: event OwnershipHandoverCanceled(address indexed pendingOwner)
func (_BunniV2HubContract *BunniV2HubContractFilterer) WatchOwnershipHandoverCanceled(opts *bind.WatchOpts, sink chan<- *BunniV2HubContractOwnershipHandoverCanceled, pendingOwner []common.Address) (event.Subscription, error) {

	var pendingOwnerRule []interface{}
	for _, pendingOwnerItem := range pendingOwner {
		pendingOwnerRule = append(pendingOwnerRule, pendingOwnerItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.WatchLogs(opts, "OwnershipHandoverCanceled", pendingOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BunniV2HubContractOwnershipHandoverCanceled)
				if err := _BunniV2HubContract.contract.UnpackLog(event, "OwnershipHandoverCanceled", log); err != nil {
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
func (_BunniV2HubContract *BunniV2HubContractFilterer) ParseOwnershipHandoverCanceled(log types.Log) (*BunniV2HubContractOwnershipHandoverCanceled, error) {
	event := new(BunniV2HubContractOwnershipHandoverCanceled)
	if err := _BunniV2HubContract.contract.UnpackLog(event, "OwnershipHandoverCanceled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BunniV2HubContractOwnershipHandoverRequestedIterator is returned from FilterOwnershipHandoverRequested and is used to iterate over the raw logs and unpacked data for OwnershipHandoverRequested events raised by the BunniV2HubContract contract.
type BunniV2HubContractOwnershipHandoverRequestedIterator struct {
	Event *BunniV2HubContractOwnershipHandoverRequested // Event containing the contract specifics and raw log

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
func (it *BunniV2HubContractOwnershipHandoverRequestedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BunniV2HubContractOwnershipHandoverRequested)
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
		it.Event = new(BunniV2HubContractOwnershipHandoverRequested)
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
func (it *BunniV2HubContractOwnershipHandoverRequestedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BunniV2HubContractOwnershipHandoverRequestedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BunniV2HubContractOwnershipHandoverRequested represents a OwnershipHandoverRequested event raised by the BunniV2HubContract contract.
type BunniV2HubContractOwnershipHandoverRequested struct {
	PendingOwner common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterOwnershipHandoverRequested is a free log retrieval operation binding the contract event 0xdbf36a107da19e49527a7176a1babf963b4b0ff8cde35ee35d6cd8f1f9ac7e1d.
//
// Solidity: event OwnershipHandoverRequested(address indexed pendingOwner)
func (_BunniV2HubContract *BunniV2HubContractFilterer) FilterOwnershipHandoverRequested(opts *bind.FilterOpts, pendingOwner []common.Address) (*BunniV2HubContractOwnershipHandoverRequestedIterator, error) {

	var pendingOwnerRule []interface{}
	for _, pendingOwnerItem := range pendingOwner {
		pendingOwnerRule = append(pendingOwnerRule, pendingOwnerItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.FilterLogs(opts, "OwnershipHandoverRequested", pendingOwnerRule)
	if err != nil {
		return nil, err
	}
	return &BunniV2HubContractOwnershipHandoverRequestedIterator{contract: _BunniV2HubContract.contract, event: "OwnershipHandoverRequested", logs: logs, sub: sub}, nil
}

// WatchOwnershipHandoverRequested is a free log subscription operation binding the contract event 0xdbf36a107da19e49527a7176a1babf963b4b0ff8cde35ee35d6cd8f1f9ac7e1d.
//
// Solidity: event OwnershipHandoverRequested(address indexed pendingOwner)
func (_BunniV2HubContract *BunniV2HubContractFilterer) WatchOwnershipHandoverRequested(opts *bind.WatchOpts, sink chan<- *BunniV2HubContractOwnershipHandoverRequested, pendingOwner []common.Address) (event.Subscription, error) {

	var pendingOwnerRule []interface{}
	for _, pendingOwnerItem := range pendingOwner {
		pendingOwnerRule = append(pendingOwnerRule, pendingOwnerItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.WatchLogs(opts, "OwnershipHandoverRequested", pendingOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BunniV2HubContractOwnershipHandoverRequested)
				if err := _BunniV2HubContract.contract.UnpackLog(event, "OwnershipHandoverRequested", log); err != nil {
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
func (_BunniV2HubContract *BunniV2HubContractFilterer) ParseOwnershipHandoverRequested(log types.Log) (*BunniV2HubContractOwnershipHandoverRequested, error) {
	event := new(BunniV2HubContractOwnershipHandoverRequested)
	if err := _BunniV2HubContract.contract.UnpackLog(event, "OwnershipHandoverRequested", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BunniV2HubContractOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the BunniV2HubContract contract.
type BunniV2HubContractOwnershipTransferredIterator struct {
	Event *BunniV2HubContractOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *BunniV2HubContractOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BunniV2HubContractOwnershipTransferred)
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
		it.Event = new(BunniV2HubContractOwnershipTransferred)
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
func (it *BunniV2HubContractOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BunniV2HubContractOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BunniV2HubContractOwnershipTransferred represents a OwnershipTransferred event raised by the BunniV2HubContract contract.
type BunniV2HubContractOwnershipTransferred struct {
	OldOwner common.Address
	NewOwner common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed oldOwner, address indexed newOwner)
func (_BunniV2HubContract *BunniV2HubContractFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, oldOwner []common.Address, newOwner []common.Address) (*BunniV2HubContractOwnershipTransferredIterator, error) {

	var oldOwnerRule []interface{}
	for _, oldOwnerItem := range oldOwner {
		oldOwnerRule = append(oldOwnerRule, oldOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.FilterLogs(opts, "OwnershipTransferred", oldOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &BunniV2HubContractOwnershipTransferredIterator{contract: _BunniV2HubContract.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed oldOwner, address indexed newOwner)
func (_BunniV2HubContract *BunniV2HubContractFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *BunniV2HubContractOwnershipTransferred, oldOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var oldOwnerRule []interface{}
	for _, oldOwnerItem := range oldOwner {
		oldOwnerRule = append(oldOwnerRule, oldOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.WatchLogs(opts, "OwnershipTransferred", oldOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BunniV2HubContractOwnershipTransferred)
				if err := _BunniV2HubContract.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_BunniV2HubContract *BunniV2HubContractFilterer) ParseOwnershipTransferred(log types.Log) (*BunniV2HubContractOwnershipTransferred, error) {
	event := new(BunniV2HubContractOwnershipTransferred)
	if err := _BunniV2HubContract.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BunniV2HubContractQueueWithdrawIterator is returned from FilterQueueWithdraw and is used to iterate over the raw logs and unpacked data for QueueWithdraw events raised by the BunniV2HubContract contract.
type BunniV2HubContractQueueWithdrawIterator struct {
	Event *BunniV2HubContractQueueWithdraw // Event containing the contract specifics and raw log

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
func (it *BunniV2HubContractQueueWithdrawIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BunniV2HubContractQueueWithdraw)
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
		it.Event = new(BunniV2HubContractQueueWithdraw)
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
func (it *BunniV2HubContractQueueWithdrawIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BunniV2HubContractQueueWithdrawIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BunniV2HubContractQueueWithdraw represents a QueueWithdraw event raised by the BunniV2HubContract contract.
type BunniV2HubContractQueueWithdraw struct {
	Sender common.Address
	PoolId [32]byte
	Shares *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterQueueWithdraw is a free log retrieval operation binding the contract event 0x0ee885e060478e5bf89befb890ae82fdcc47aa2a9c8e4d668fcce310318d28a1.
//
// Solidity: event QueueWithdraw(address indexed sender, bytes32 indexed poolId, uint256 shares)
func (_BunniV2HubContract *BunniV2HubContractFilterer) FilterQueueWithdraw(opts *bind.FilterOpts, sender []common.Address, poolId [][32]byte) (*BunniV2HubContractQueueWithdrawIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var poolIdRule []interface{}
	for _, poolIdItem := range poolId {
		poolIdRule = append(poolIdRule, poolIdItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.FilterLogs(opts, "QueueWithdraw", senderRule, poolIdRule)
	if err != nil {
		return nil, err
	}
	return &BunniV2HubContractQueueWithdrawIterator{contract: _BunniV2HubContract.contract, event: "QueueWithdraw", logs: logs, sub: sub}, nil
}

// WatchQueueWithdraw is a free log subscription operation binding the contract event 0x0ee885e060478e5bf89befb890ae82fdcc47aa2a9c8e4d668fcce310318d28a1.
//
// Solidity: event QueueWithdraw(address indexed sender, bytes32 indexed poolId, uint256 shares)
func (_BunniV2HubContract *BunniV2HubContractFilterer) WatchQueueWithdraw(opts *bind.WatchOpts, sink chan<- *BunniV2HubContractQueueWithdraw, sender []common.Address, poolId [][32]byte) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var poolIdRule []interface{}
	for _, poolIdItem := range poolId {
		poolIdRule = append(poolIdRule, poolIdItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.WatchLogs(opts, "QueueWithdraw", senderRule, poolIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BunniV2HubContractQueueWithdraw)
				if err := _BunniV2HubContract.contract.UnpackLog(event, "QueueWithdraw", log); err != nil {
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

// ParseQueueWithdraw is a log parse operation binding the contract event 0x0ee885e060478e5bf89befb890ae82fdcc47aa2a9c8e4d668fcce310318d28a1.
//
// Solidity: event QueueWithdraw(address indexed sender, bytes32 indexed poolId, uint256 shares)
func (_BunniV2HubContract *BunniV2HubContractFilterer) ParseQueueWithdraw(log types.Log) (*BunniV2HubContractQueueWithdraw, error) {
	event := new(BunniV2HubContractQueueWithdraw)
	if err := _BunniV2HubContract.contract.UnpackLog(event, "QueueWithdraw", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BunniV2HubContractSetPauseFlagsIterator is returned from FilterSetPauseFlags and is used to iterate over the raw logs and unpacked data for SetPauseFlags events raised by the BunniV2HubContract contract.
type BunniV2HubContractSetPauseFlagsIterator struct {
	Event *BunniV2HubContractSetPauseFlags // Event containing the contract specifics and raw log

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
func (it *BunniV2HubContractSetPauseFlagsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BunniV2HubContractSetPauseFlags)
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
		it.Event = new(BunniV2HubContractSetPauseFlags)
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
func (it *BunniV2HubContractSetPauseFlagsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BunniV2HubContractSetPauseFlagsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BunniV2HubContractSetPauseFlags represents a SetPauseFlags event raised by the BunniV2HubContract contract.
type BunniV2HubContractSetPauseFlags struct {
	PauseFlags uint8
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterSetPauseFlags is a free log retrieval operation binding the contract event 0x3021cc5514f1ea312648df4d3e6c9cf9c5bd12c429f0849d4c903af7010c6afa.
//
// Solidity: event SetPauseFlags(uint8 indexed pauseFlags)
func (_BunniV2HubContract *BunniV2HubContractFilterer) FilterSetPauseFlags(opts *bind.FilterOpts, pauseFlags []uint8) (*BunniV2HubContractSetPauseFlagsIterator, error) {

	var pauseFlagsRule []interface{}
	for _, pauseFlagsItem := range pauseFlags {
		pauseFlagsRule = append(pauseFlagsRule, pauseFlagsItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.FilterLogs(opts, "SetPauseFlags", pauseFlagsRule)
	if err != nil {
		return nil, err
	}
	return &BunniV2HubContractSetPauseFlagsIterator{contract: _BunniV2HubContract.contract, event: "SetPauseFlags", logs: logs, sub: sub}, nil
}

// WatchSetPauseFlags is a free log subscription operation binding the contract event 0x3021cc5514f1ea312648df4d3e6c9cf9c5bd12c429f0849d4c903af7010c6afa.
//
// Solidity: event SetPauseFlags(uint8 indexed pauseFlags)
func (_BunniV2HubContract *BunniV2HubContractFilterer) WatchSetPauseFlags(opts *bind.WatchOpts, sink chan<- *BunniV2HubContractSetPauseFlags, pauseFlags []uint8) (event.Subscription, error) {

	var pauseFlagsRule []interface{}
	for _, pauseFlagsItem := range pauseFlags {
		pauseFlagsRule = append(pauseFlagsRule, pauseFlagsItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.WatchLogs(opts, "SetPauseFlags", pauseFlagsRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BunniV2HubContractSetPauseFlags)
				if err := _BunniV2HubContract.contract.UnpackLog(event, "SetPauseFlags", log); err != nil {
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

// ParseSetPauseFlags is a log parse operation binding the contract event 0x3021cc5514f1ea312648df4d3e6c9cf9c5bd12c429f0849d4c903af7010c6afa.
//
// Solidity: event SetPauseFlags(uint8 indexed pauseFlags)
func (_BunniV2HubContract *BunniV2HubContractFilterer) ParseSetPauseFlags(log types.Log) (*BunniV2HubContractSetPauseFlags, error) {
	event := new(BunniV2HubContractSetPauseFlags)
	if err := _BunniV2HubContract.contract.UnpackLog(event, "SetPauseFlags", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BunniV2HubContractSetPauserIterator is returned from FilterSetPauser and is used to iterate over the raw logs and unpacked data for SetPauser events raised by the BunniV2HubContract contract.
type BunniV2HubContractSetPauserIterator struct {
	Event *BunniV2HubContractSetPauser // Event containing the contract specifics and raw log

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
func (it *BunniV2HubContractSetPauserIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BunniV2HubContractSetPauser)
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
		it.Event = new(BunniV2HubContractSetPauser)
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
func (it *BunniV2HubContractSetPauserIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BunniV2HubContractSetPauserIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BunniV2HubContractSetPauser represents a SetPauser event raised by the BunniV2HubContract contract.
type BunniV2HubContractSetPauser struct {
	Guy      common.Address
	IsPauser bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterSetPauser is a free log retrieval operation binding the contract event 0xd34f4aa5f94a385f2fa0ca25e5f01c6f331018f35c3d43a7b8057a86704de3df.
//
// Solidity: event SetPauser(address indexed guy, bool indexed isPauser)
func (_BunniV2HubContract *BunniV2HubContractFilterer) FilterSetPauser(opts *bind.FilterOpts, guy []common.Address, isPauser []bool) (*BunniV2HubContractSetPauserIterator, error) {

	var guyRule []interface{}
	for _, guyItem := range guy {
		guyRule = append(guyRule, guyItem)
	}
	var isPauserRule []interface{}
	for _, isPauserItem := range isPauser {
		isPauserRule = append(isPauserRule, isPauserItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.FilterLogs(opts, "SetPauser", guyRule, isPauserRule)
	if err != nil {
		return nil, err
	}
	return &BunniV2HubContractSetPauserIterator{contract: _BunniV2HubContract.contract, event: "SetPauser", logs: logs, sub: sub}, nil
}

// WatchSetPauser is a free log subscription operation binding the contract event 0xd34f4aa5f94a385f2fa0ca25e5f01c6f331018f35c3d43a7b8057a86704de3df.
//
// Solidity: event SetPauser(address indexed guy, bool indexed isPauser)
func (_BunniV2HubContract *BunniV2HubContractFilterer) WatchSetPauser(opts *bind.WatchOpts, sink chan<- *BunniV2HubContractSetPauser, guy []common.Address, isPauser []bool) (event.Subscription, error) {

	var guyRule []interface{}
	for _, guyItem := range guy {
		guyRule = append(guyRule, guyItem)
	}
	var isPauserRule []interface{}
	for _, isPauserItem := range isPauser {
		isPauserRule = append(isPauserRule, isPauserItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.WatchLogs(opts, "SetPauser", guyRule, isPauserRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BunniV2HubContractSetPauser)
				if err := _BunniV2HubContract.contract.UnpackLog(event, "SetPauser", log); err != nil {
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

// ParseSetPauser is a log parse operation binding the contract event 0xd34f4aa5f94a385f2fa0ca25e5f01c6f331018f35c3d43a7b8057a86704de3df.
//
// Solidity: event SetPauser(address indexed guy, bool indexed isPauser)
func (_BunniV2HubContract *BunniV2HubContractFilterer) ParseSetPauser(log types.Log) (*BunniV2HubContractSetPauser, error) {
	event := new(BunniV2HubContractSetPauser)
	if err := _BunniV2HubContract.contract.UnpackLog(event, "SetPauser", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BunniV2HubContractSetReferralRewardRecipientIterator is returned from FilterSetReferralRewardRecipient and is used to iterate over the raw logs and unpacked data for SetReferralRewardRecipient events raised by the BunniV2HubContract contract.
type BunniV2HubContractSetReferralRewardRecipientIterator struct {
	Event *BunniV2HubContractSetReferralRewardRecipient // Event containing the contract specifics and raw log

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
func (it *BunniV2HubContractSetReferralRewardRecipientIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BunniV2HubContractSetReferralRewardRecipient)
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
		it.Event = new(BunniV2HubContractSetReferralRewardRecipient)
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
func (it *BunniV2HubContractSetReferralRewardRecipientIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BunniV2HubContractSetReferralRewardRecipientIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BunniV2HubContractSetReferralRewardRecipient represents a SetReferralRewardRecipient event raised by the BunniV2HubContract contract.
type BunniV2HubContractSetReferralRewardRecipient struct {
	NewReferralRewardRecipient common.Address
	Raw                        types.Log // Blockchain specific contextual infos
}

// FilterSetReferralRewardRecipient is a free log retrieval operation binding the contract event 0x905a74aecdb46e2b4d535f80ff231418f1c5684536fe968901427f2c826bd030.
//
// Solidity: event SetReferralRewardRecipient(address indexed newReferralRewardRecipient)
func (_BunniV2HubContract *BunniV2HubContractFilterer) FilterSetReferralRewardRecipient(opts *bind.FilterOpts, newReferralRewardRecipient []common.Address) (*BunniV2HubContractSetReferralRewardRecipientIterator, error) {

	var newReferralRewardRecipientRule []interface{}
	for _, newReferralRewardRecipientItem := range newReferralRewardRecipient {
		newReferralRewardRecipientRule = append(newReferralRewardRecipientRule, newReferralRewardRecipientItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.FilterLogs(opts, "SetReferralRewardRecipient", newReferralRewardRecipientRule)
	if err != nil {
		return nil, err
	}
	return &BunniV2HubContractSetReferralRewardRecipientIterator{contract: _BunniV2HubContract.contract, event: "SetReferralRewardRecipient", logs: logs, sub: sub}, nil
}

// WatchSetReferralRewardRecipient is a free log subscription operation binding the contract event 0x905a74aecdb46e2b4d535f80ff231418f1c5684536fe968901427f2c826bd030.
//
// Solidity: event SetReferralRewardRecipient(address indexed newReferralRewardRecipient)
func (_BunniV2HubContract *BunniV2HubContractFilterer) WatchSetReferralRewardRecipient(opts *bind.WatchOpts, sink chan<- *BunniV2HubContractSetReferralRewardRecipient, newReferralRewardRecipient []common.Address) (event.Subscription, error) {

	var newReferralRewardRecipientRule []interface{}
	for _, newReferralRewardRecipientItem := range newReferralRewardRecipient {
		newReferralRewardRecipientRule = append(newReferralRewardRecipientRule, newReferralRewardRecipientItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.WatchLogs(opts, "SetReferralRewardRecipient", newReferralRewardRecipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BunniV2HubContractSetReferralRewardRecipient)
				if err := _BunniV2HubContract.contract.UnpackLog(event, "SetReferralRewardRecipient", log); err != nil {
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

// ParseSetReferralRewardRecipient is a log parse operation binding the contract event 0x905a74aecdb46e2b4d535f80ff231418f1c5684536fe968901427f2c826bd030.
//
// Solidity: event SetReferralRewardRecipient(address indexed newReferralRewardRecipient)
func (_BunniV2HubContract *BunniV2HubContractFilterer) ParseSetReferralRewardRecipient(log types.Log) (*BunniV2HubContractSetReferralRewardRecipient, error) {
	event := new(BunniV2HubContractSetReferralRewardRecipient)
	if err := _BunniV2HubContract.contract.UnpackLog(event, "SetReferralRewardRecipient", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BunniV2HubContractWithdrawIterator is returned from FilterWithdraw and is used to iterate over the raw logs and unpacked data for Withdraw events raised by the BunniV2HubContract contract.
type BunniV2HubContractWithdrawIterator struct {
	Event *BunniV2HubContractWithdraw // Event containing the contract specifics and raw log

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
func (it *BunniV2HubContractWithdrawIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BunniV2HubContractWithdraw)
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
		it.Event = new(BunniV2HubContractWithdraw)
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
func (it *BunniV2HubContractWithdrawIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BunniV2HubContractWithdrawIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BunniV2HubContractWithdraw represents a Withdraw event raised by the BunniV2HubContract contract.
type BunniV2HubContractWithdraw struct {
	Sender    common.Address
	Recipient common.Address
	PoolId    [32]byte
	Amount0   *big.Int
	Amount1   *big.Int
	Shares    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterWithdraw is a free log retrieval operation binding the contract event 0xbc70c2ef3795ca1df41695488a7ff6060de75f86dd892696bbfd76bdd123270f.
//
// Solidity: event Withdraw(address indexed sender, address indexed recipient, bytes32 indexed poolId, uint256 amount0, uint256 amount1, uint256 shares)
func (_BunniV2HubContract *BunniV2HubContractFilterer) FilterWithdraw(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address, poolId [][32]byte) (*BunniV2HubContractWithdrawIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var poolIdRule []interface{}
	for _, poolIdItem := range poolId {
		poolIdRule = append(poolIdRule, poolIdItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.FilterLogs(opts, "Withdraw", senderRule, recipientRule, poolIdRule)
	if err != nil {
		return nil, err
	}
	return &BunniV2HubContractWithdrawIterator{contract: _BunniV2HubContract.contract, event: "Withdraw", logs: logs, sub: sub}, nil
}

// WatchWithdraw is a free log subscription operation binding the contract event 0xbc70c2ef3795ca1df41695488a7ff6060de75f86dd892696bbfd76bdd123270f.
//
// Solidity: event Withdraw(address indexed sender, address indexed recipient, bytes32 indexed poolId, uint256 amount0, uint256 amount1, uint256 shares)
func (_BunniV2HubContract *BunniV2HubContractFilterer) WatchWithdraw(opts *bind.WatchOpts, sink chan<- *BunniV2HubContractWithdraw, sender []common.Address, recipient []common.Address, poolId [][32]byte) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var poolIdRule []interface{}
	for _, poolIdItem := range poolId {
		poolIdRule = append(poolIdRule, poolIdItem)
	}

	logs, sub, err := _BunniV2HubContract.contract.WatchLogs(opts, "Withdraw", senderRule, recipientRule, poolIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BunniV2HubContractWithdraw)
				if err := _BunniV2HubContract.contract.UnpackLog(event, "Withdraw", log); err != nil {
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

// ParseWithdraw is a log parse operation binding the contract event 0xbc70c2ef3795ca1df41695488a7ff6060de75f86dd892696bbfd76bdd123270f.
//
// Solidity: event Withdraw(address indexed sender, address indexed recipient, bytes32 indexed poolId, uint256 amount0, uint256 amount1, uint256 shares)
func (_BunniV2HubContract *BunniV2HubContractFilterer) ParseWithdraw(log types.Log) (*BunniV2HubContractWithdraw, error) {
	event := new(BunniV2HubContractWithdraw)
	if err := _BunniV2HubContract.contract.UnpackLog(event, "Withdraw", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
