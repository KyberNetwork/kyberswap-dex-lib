// // Code generated - DO NOT EDIT.
// // This file is a generated binding and any manual changes will be lost.

package bunniv2

// import (
// 	"errors"
// 	"math/big"
// 	"strings"

// 	ethereum "github.com/ethereum/go-ethereum"
// 	"github.com/ethereum/go-ethereum/accounts/abi"
// 	"github.com/ethereum/go-ethereum/accounts/abi/bind"
// 	"github.com/ethereum/go-ethereum/common"
// 	"github.com/ethereum/go-ethereum/core/types"
// 	"github.com/ethereum/go-ethereum/event"
// )

// // Reference imports to suppress errors if they are not otherwise used.
// var (
// 	_ = errors.New
// 	_ = big.NewInt
// 	_ = strings.NewReader
// 	_ = ethereum.NotFound
// 	_ = bind.Bind
// 	_ = common.Big1
// 	_ = types.BloomLookup
// 	_ = event.NewSubscription
// 	_ = abi.ConvertType
// )

// // IBunniHubDeployBunniTokenParams is an auto generated low-level Go binding around an user-defined struct.
// type IBunniHubDeployBunniTokenParams struct {
// 	Currency0                common.Address
// 	Currency1                common.Address
// 	TickSpacing              *big.Int
// 	TwapSecondsAgo           *big.Int
// 	LiquidityDensityFunction common.Address
// 	Hooklet                  common.Address
// 	LdfType                  uint8
// 	LdfParams                [32]byte
// 	Hooks                    common.Address
// 	HookParams               []byte
// 	Vault0                   common.Address
// 	Vault1                   common.Address
// 	MinRawTokenRatio0        *big.Int
// 	TargetRawTokenRatio0     *big.Int
// 	MaxRawTokenRatio0        *big.Int
// 	MinRawTokenRatio1        *big.Int
// 	TargetRawTokenRatio1     *big.Int
// 	MaxRawTokenRatio1        *big.Int
// 	SqrtPriceX96             *big.Int
// 	Name                     [32]byte
// 	Symbol                   [32]byte
// 	Owner                    common.Address
// 	MetadataURI              string
// 	Salt                     [32]byte
// }

// // IBunniHubDepositParams is an auto generated low-level Go binding around an user-defined struct.
// type IBunniHubDepositParams struct {
// 	PoolKey         PoolKey
// 	Recipient       common.Address
// 	RefundRecipient common.Address
// 	Amount0Desired  *big.Int
// 	Amount1Desired  *big.Int
// 	Amount0Min      *big.Int
// 	Amount1Min      *big.Int
// 	VaultFee0       *big.Int
// 	VaultFee1       *big.Int
// 	Deadline        *big.Int
// 	Referrer        common.Address
// }

// // IBunniHubQueueWithdrawParams is an auto generated low-level Go binding around an user-defined struct.
// type IBunniHubQueueWithdrawParams struct {
// 	PoolKey PoolKey
// 	Shares  *big.Int
// }

// // IBunniHubWithdrawParams is an auto generated low-level Go binding around an user-defined struct.
// type IBunniHubWithdrawParams struct {
// 	PoolKey             PoolKey
// 	Recipient           common.Address
// 	Shares              *big.Int
// 	Amount0Min          *big.Int
// 	Amount1Min          *big.Int
// 	Deadline            *big.Int
// 	UseQueuedWithdrawal bool
// }

// // PoolKey is an auto generated low-level Go binding around an user-defined struct.
// type PoolKey struct {
// 	Currency0   common.Address
// 	Currency1   common.Address
// 	Fee         *big.Int
// 	TickSpacing *big.Int
// 	Hooks       common.Address
// }

// // PoolState is an auto generated low-level Go binding around an user-defined struct.
// type PoolState struct {
// 	LiquidityDensityFunction common.Address
// 	BunniToken               common.Address
// 	Hooklet                  common.Address
// 	TwapSecondsAgo           *big.Int
// 	LdfParams                [32]byte
// 	HookParams               []byte
// 	Vault0                   common.Address
// 	Vault1                   common.Address
// 	LdfType                  uint8
// 	MinRawTokenRatio0        *big.Int
// 	TargetRawTokenRatio0     *big.Int
// 	MaxRawTokenRatio0        *big.Int
// 	MinRawTokenRatio1        *big.Int
// 	TargetRawTokenRatio1     *big.Int
// 	MaxRawTokenRatio1        *big.Int
// 	RawBalance0              *big.Int
// 	RawBalance1              *big.Int
// 	Reserve0                 *big.Int
// 	Reserve1                 *big.Int
// 	IdleBalance              [32]byte
// }

// // BunniHubMetaData contains all meta data concerning the BunniHub contract.
// var BunniHubMetaData = &bind.MetaData{
// 	ABI: "[{\"inputs\":[{\"internalType\":\"contractIPoolManager\",\"name\":\"poolManager_\",\"type\":\"address\"},{\"internalType\":\"contractWETH\",\"name\":\"weth_\",\"type\":\"address\"},{\"internalType\":\"contractIPermit2\",\"name\":\"permit2_\",\"type\":\"address\"},{\"internalType\":\"contractIBunniToken\",\"name\":\"bunniTokenImplementation_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"initialOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"initialReferralRewardRecipient\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AlreadyInitialized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHub__BunniTokenNotInitialized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHub__MsgValueInsufficient\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHub__PastDeadline\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHub__Paused\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BunniHub__Unauthorized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NewOwnerIsZeroAddress\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NoHandoverRequest\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ReentrancyGuard__ReentrantCall\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Unauthorized\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"BurnPauseFuse\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"shares\",\"type\":\"uint256\"}],\"name\":\"Deposit\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"contractIBunniToken\",\"name\":\"bunniToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"NewBunni\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"pendingOwner\",\"type\":\"address\"}],\"name\":\"OwnershipHandoverCanceled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"pendingOwner\",\"type\":\"address\"}],\"name\":\"OwnershipHandoverRequested\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"shares\",\"type\":\"uint256\"}],\"name\":\"QueueWithdraw\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint8\",\"name\":\"pauseFlags\",\"type\":\"uint8\"}],\"name\":\"SetPauseFlags\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"guy\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bool\",\"name\":\"isPauser\",\"type\":\"bool\"}],\"name\":\"SetPauser\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newReferralRewardRecipient\",\"type\":\"address\"}],\"name\":\"SetReferralRewardRecipient\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"shares\",\"type\":\"uint256\"}],\"name\":\"Withdraw\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"bunniTokenOfPool\",\"outputs\":[{\"internalType\":\"contractIBunniToken\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"burnPauseFuse\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"cancelOwnershipHandover\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"pendingOwner\",\"type\":\"address\"}],\"name\":\"completeOwnershipHandover\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"uint24\",\"name\":\"twapSecondsAgo\",\"type\":\"uint24\"},{\"internalType\":\"contractILiquidityDensityFunction\",\"name\":\"liquidityDensityFunction\",\"type\":\"address\"},{\"internalType\":\"contractIHooklet\",\"name\":\"hooklet\",\"type\":\"address\"},{\"internalType\":\"enumLDFType\",\"name\":\"ldfType\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"ldfParams\",\"type\":\"bytes32\"},{\"internalType\":\"contractIBunniHook\",\"name\":\"hooks\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"hookParams\",\"type\":\"bytes\"},{\"internalType\":\"contractERC4626\",\"name\":\"vault0\",\"type\":\"address\"},{\"internalType\":\"contractERC4626\",\"name\":\"vault1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"minRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"targetRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"maxRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"minRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"targetRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"maxRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceX96\",\"type\":\"uint160\"},{\"internalType\":\"bytes32\",\"name\":\"name\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"symbol\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"metadataURI\",\"type\":\"string\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"}],\"internalType\":\"structIBunniHub.DeployBunniTokenParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"deployBunniToken\",\"outputs\":[{\"internalType\":\"contractIBunniToken\",\"name\":\"token\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"poolKey\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"refundRecipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount0Desired\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1Desired\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount0Min\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1Min\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"vaultFee0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"vaultFee1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"referrer\",\"type\":\"address\"}],\"internalType\":\"structIBunniHub.DepositParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"deposit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"shares\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getPauseStatus\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"pauseFlags\",\"type\":\"uint8\"},{\"internalType\":\"bool\",\"name\":\"unpauseFuse\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getReferralRewardRecipient\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"bool\",\"name\":\"zeroForOne\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"inputAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"outputAmount\",\"type\":\"uint256\"}],\"name\":\"hookHandleSwap\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"hookParams\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"IdleBalance\",\"name\":\"newIdleBalance\",\"type\":\"bytes32\"}],\"name\":\"hookSetIdleBalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"idleBalance\",\"outputs\":[{\"internalType\":\"IdleBalance\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"guy\",\"type\":\"address\"}],\"name\":\"isPauser\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"}],\"name\":\"lockForRebalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"bunniSubspace\",\"type\":\"bytes32\"}],\"name\":\"nonce\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"result\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"pendingOwner\",\"type\":\"address\"}],\"name\":\"ownershipHandoverExpiresAt\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"result\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"poolBalances\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"balance0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"balance1\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBunniToken\",\"name\":\"bunniToken\",\"type\":\"address\"}],\"name\":\"poolIdOfBunniToken\",\"outputs\":[{\"internalType\":\"PoolId\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"poolInitData\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"poolParams\",\"outputs\":[{\"components\":[{\"internalType\":\"contractILiquidityDensityFunction\",\"name\":\"liquidityDensityFunction\",\"type\":\"address\"},{\"internalType\":\"contractIBunniToken\",\"name\":\"bunniToken\",\"type\":\"address\"},{\"internalType\":\"contractIHooklet\",\"name\":\"hooklet\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"twapSecondsAgo\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"ldfParams\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"hookParams\",\"type\":\"bytes\"},{\"internalType\":\"contractERC4626\",\"name\":\"vault0\",\"type\":\"address\"},{\"internalType\":\"contractERC4626\",\"name\":\"vault1\",\"type\":\"address\"},{\"internalType\":\"enumLDFType\",\"name\":\"ldfType\",\"type\":\"uint8\"},{\"internalType\":\"uint24\",\"name\":\"minRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"targetRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"maxRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"minRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"targetRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"maxRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint256\",\"name\":\"rawBalance0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"rawBalance1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reserve0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reserve1\",\"type\":\"uint256\"},{\"internalType\":\"IdleBalance\",\"name\":\"idleBalance\",\"type\":\"bytes32\"}],\"internalType\":\"structPoolState\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"PoolId\",\"name\":\"poolId\",\"type\":\"bytes32\"}],\"name\":\"poolState\",\"outputs\":[{\"components\":[{\"internalType\":\"contractILiquidityDensityFunction\",\"name\":\"liquidityDensityFunction\",\"type\":\"address\"},{\"internalType\":\"contractIBunniToken\",\"name\":\"bunniToken\",\"type\":\"address\"},{\"internalType\":\"contractIHooklet\",\"name\":\"hooklet\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"twapSecondsAgo\",\"type\":\"uint24\"},{\"internalType\":\"bytes32\",\"name\":\"ldfParams\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"hookParams\",\"type\":\"bytes\"},{\"internalType\":\"contractERC4626\",\"name\":\"vault0\",\"type\":\"address\"},{\"internalType\":\"contractERC4626\",\"name\":\"vault1\",\"type\":\"address\"},{\"internalType\":\"enumLDFType\",\"name\":\"ldfType\",\"type\":\"uint8\"},{\"internalType\":\"uint24\",\"name\":\"minRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"targetRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"maxRawTokenRatio0\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"minRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"targetRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"maxRawTokenRatio1\",\"type\":\"uint24\"},{\"internalType\":\"uint256\",\"name\":\"rawBalance0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"rawBalance1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reserve0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reserve1\",\"type\":\"uint256\"},{\"internalType\":\"IdleBalance\",\"name\":\"idleBalance\",\"type\":\"bytes32\"}],\"internalType\":\"structPoolState\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"poolKey\",\"type\":\"tuple\"},{\"internalType\":\"uint200\",\"name\":\"shares\",\"type\":\"uint200\"}],\"internalType\":\"structIBunniHub.QueueWithdrawParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"queueWithdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"requestOwnershipHandover\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"pauseFlags\",\"type\":\"uint8\"}],\"name\":\"setPauseFlags\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"guy\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"status\",\"type\":\"bool\"}],\"name\":\"setPauser\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newReferralRewardRecipient\",\"type\":\"address\"}],\"name\":\"setReferralRewardRecipient\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"unlockCallback\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"key\",\"type\":\"tuple\"}],\"name\":\"unlockForRebalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"Currency\",\"name\":\"currency0\",\"type\":\"address\"},{\"internalType\":\"Currency\",\"name\":\"currency1\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"int24\",\"name\":\"tickSpacing\",\"type\":\"int24\"},{\"internalType\":\"contractIHooks\",\"name\":\"hooks\",\"type\":\"address\"}],\"internalType\":\"structPoolKey\",\"name\":\"poolKey\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"shares\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount0Min\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1Min\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"useQueuedWithdrawal\",\"type\":\"bool\"}],\"internalType\":\"structIBunniHub.WithdrawParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"withdraw\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount1\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
// }

// // BunniHubABI is the input ABI used to generate the binding from.
// // Deprecated: Use BunniHubMetaData.ABI instead.
// var BunniHubABI = BunniHubMetaData.ABI

// // BunniHub is an auto generated Go binding around an Ethereum contract.
// type BunniHub struct {
// 	BunniHubCaller     // Read-only binding to the contract
// 	BunniHubTransactor // Write-only binding to the contract
// 	BunniHubFilterer   // Log filterer for contract events
// }

// // BunniHubCaller is an auto generated read-only Go binding around an Ethereum contract.
// type BunniHubCaller struct {
// 	contract *bind.BoundContract // Generic contract wrapper for the low level calls
// }

// // BunniHubTransactor is an auto generated write-only Go binding around an Ethereum contract.
// type BunniHubTransactor struct {
// 	contract *bind.BoundContract // Generic contract wrapper for the low level calls
// }

// // BunniHubFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
// type BunniHubFilterer struct {
// 	contract *bind.BoundContract // Generic contract wrapper for the low level calls
// }

// // BunniHubSession is an auto generated Go binding around an Ethereum contract,
// // with pre-set call and transact options.
// type BunniHubSession struct {
// 	Contract     *BunniHub         // Generic contract binding to set the session for
// 	CallOpts     bind.CallOpts     // Call options to use throughout this session
// 	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
// }

// // BunniHubCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// // with pre-set call options.
// type BunniHubCallerSession struct {
// 	Contract *BunniHubCaller // Generic contract caller binding to set the session for
// 	CallOpts bind.CallOpts   // Call options to use throughout this session
// }

// // BunniHubTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// // with pre-set transact options.
// type BunniHubTransactorSession struct {
// 	Contract     *BunniHubTransactor // Generic contract transactor binding to set the session for
// 	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
// }

// // BunniHubRaw is an auto generated low-level Go binding around an Ethereum contract.
// type BunniHubRaw struct {
// 	Contract *BunniHub // Generic contract binding to access the raw methods on
// }

// // BunniHubCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
// type BunniHubCallerRaw struct {
// 	Contract *BunniHubCaller // Generic read-only contract binding to access the raw methods on
// }

// // BunniHubTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
// type BunniHubTransactorRaw struct {
// 	Contract *BunniHubTransactor // Generic write-only contract binding to access the raw methods on
// }

// // NewBunniHub creates a new instance of BunniHub, bound to a specific deployed contract.
// func NewBunniHub(address common.Address, backend bind.ContractBackend) (*BunniHub, error) {
// 	contract, err := bindBunniHub(address, backend, backend, backend)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &BunniHub{BunniHubCaller: BunniHubCaller{contract: contract}, BunniHubTransactor: BunniHubTransactor{contract: contract}, BunniHubFilterer: BunniHubFilterer{contract: contract}}, nil
// }

// // NewBunniHubCaller creates a new read-only instance of BunniHub, bound to a specific deployed contract.
// func NewBunniHubCaller(address common.Address, caller bind.ContractCaller) (*BunniHubCaller, error) {
// 	contract, err := bindBunniHub(address, caller, nil, nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &BunniHubCaller{contract: contract}, nil
// }

// // NewBunniHubTransactor creates a new write-only instance of BunniHub, bound to a specific deployed contract.
// func NewBunniHubTransactor(address common.Address, transactor bind.ContractTransactor) (*BunniHubTransactor, error) {
// 	contract, err := bindBunniHub(address, nil, transactor, nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &BunniHubTransactor{contract: contract}, nil
// }

// // NewBunniHubFilterer creates a new log filterer instance of BunniHub, bound to a specific deployed contract.
// func NewBunniHubFilterer(address common.Address, filterer bind.ContractFilterer) (*BunniHubFilterer, error) {
// 	contract, err := bindBunniHub(address, nil, nil, filterer)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &BunniHubFilterer{contract: contract}, nil
// }

// // bindBunniHub binds a generic wrapper to an already deployed contract.
// func bindBunniHub(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
// 	parsed, err := BunniHubMetaData.GetAbi()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
// }

// // Call invokes the (constant) contract method with params as input values and
// // sets the output to result. The result type might be a single field for simple
// // returns, a slice of interfaces for anonymous returns and a struct for named
// // returns.
// func (_BunniHub *BunniHubRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
// 	return _BunniHub.Contract.BunniHubCaller.contract.Call(opts, result, method, params...)
// }

// // Transfer initiates a plain transaction to move funds to the contract, calling
// // its default method if one is available.
// func (_BunniHub *BunniHubRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
// 	return _BunniHub.Contract.BunniHubTransactor.contract.Transfer(opts)
// }

// // Transact invokes the (paid) contract method with params as input values.
// func (_BunniHub *BunniHubRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
// 	return _BunniHub.Contract.BunniHubTransactor.contract.Transact(opts, method, params...)
// }

// // Call invokes the (constant) contract method with params as input values and
// // sets the output to result. The result type might be a single field for simple
// // returns, a slice of interfaces for anonymous returns and a struct for named
// // returns.
// func (_BunniHub *BunniHubCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
// 	return _BunniHub.Contract.contract.Call(opts, result, method, params...)
// }

// // Transfer initiates a plain transaction to move funds to the contract, calling
// // its default method if one is available.
// func (_BunniHub *BunniHubTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
// 	return _BunniHub.Contract.contract.Transfer(opts)
// }

// // Transact invokes the (paid) contract method with params as input values.
// func (_BunniHub *BunniHubTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
// 	return _BunniHub.Contract.contract.Transact(opts, method, params...)
// }

// // BunniTokenOfPool is a free data retrieval call binding the contract method 0xa2a56697.
// //
// // Solidity: function bunniTokenOfPool(bytes32 poolId) view returns(address)
// func (_BunniHub *BunniHubCaller) BunniTokenOfPool(opts *bind.CallOpts, poolId [32]byte) (common.Address, error) {
// 	var out []interface{}
// 	err := _BunniHub.contract.Call(opts, &out, "bunniTokenOfPool", poolId)

// 	if err != nil {
// 		return *new(common.Address), err
// 	}

// 	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

// 	return out0, err

// }

// // BunniTokenOfPool is a free data retrieval call binding the contract method 0xa2a56697.
// //
// // Solidity: function bunniTokenOfPool(bytes32 poolId) view returns(address)
// func (_BunniHub *BunniHubSession) BunniTokenOfPool(poolId [32]byte) (common.Address, error) {
// 	return _BunniHub.Contract.BunniTokenOfPool(&_BunniHub.CallOpts, poolId)
// }

// // BunniTokenOfPool is a free data retrieval call binding the contract method 0xa2a56697.
// //
// // Solidity: function bunniTokenOfPool(bytes32 poolId) view returns(address)
// func (_BunniHub *BunniHubCallerSession) BunniTokenOfPool(poolId [32]byte) (common.Address, error) {
// 	return _BunniHub.Contract.BunniTokenOfPool(&_BunniHub.CallOpts, poolId)
// }

// // GetPauseStatus is a free data retrieval call binding the contract method 0x1d9023cb.
// //
// // Solidity: function getPauseStatus() view returns(uint8 pauseFlags, bool unpauseFuse)
// func (_BunniHub *BunniHubCaller) GetPauseStatus(opts *bind.CallOpts) (struct {
// 	PauseFlags  uint8
// 	UnpauseFuse bool
// }, error) {
// 	var out []interface{}
// 	err := _BunniHub.contract.Call(opts, &out, "getPauseStatus")

// 	outstruct := new(struct {
// 		PauseFlags  uint8
// 		UnpauseFuse bool
// 	})
// 	if err != nil {
// 		return *outstruct, err
// 	}

// 	outstruct.PauseFlags = *abi.ConvertType(out[0], new(uint8)).(*uint8)
// 	outstruct.UnpauseFuse = *abi.ConvertType(out[1], new(bool)).(*bool)

// 	return *outstruct, err

// }

// // GetPauseStatus is a free data retrieval call binding the contract method 0x1d9023cb.
// //
// // Solidity: function getPauseStatus() view returns(uint8 pauseFlags, bool unpauseFuse)
// func (_BunniHub *BunniHubSession) GetPauseStatus() (struct {
// 	PauseFlags  uint8
// 	UnpauseFuse bool
// }, error) {
// 	return _BunniHub.Contract.GetPauseStatus(&_BunniHub.CallOpts)
// }

// // GetPauseStatus is a free data retrieval call binding the contract method 0x1d9023cb.
// //
// // Solidity: function getPauseStatus() view returns(uint8 pauseFlags, bool unpauseFuse)
// func (_BunniHub *BunniHubCallerSession) GetPauseStatus() (struct {
// 	PauseFlags  uint8
// 	UnpauseFuse bool
// }, error) {
// 	return _BunniHub.Contract.GetPauseStatus(&_BunniHub.CallOpts)
// }

// // GetReferralRewardRecipient is a free data retrieval call binding the contract method 0x565f4b21.
// //
// // Solidity: function getReferralRewardRecipient() view returns(address)
// func (_BunniHub *BunniHubCaller) GetReferralRewardRecipient(opts *bind.CallOpts) (common.Address, error) {
// 	var out []interface{}
// 	err := _BunniHub.contract.Call(opts, &out, "getReferralRewardRecipient")

// 	if err != nil {
// 		return *new(common.Address), err
// 	}

// 	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

// 	return out0, err

// }

// // GetReferralRewardRecipient is a free data retrieval call binding the contract method 0x565f4b21.
// //
// // Solidity: function getReferralRewardRecipient() view returns(address)
// func (_BunniHub *BunniHubSession) GetReferralRewardRecipient() (common.Address, error) {
// 	return _BunniHub.Contract.GetReferralRewardRecipient(&_BunniHub.CallOpts)
// }

// // GetReferralRewardRecipient is a free data retrieval call binding the contract method 0x565f4b21.
// //
// // Solidity: function getReferralRewardRecipient() view returns(address)
// func (_BunniHub *BunniHubCallerSession) GetReferralRewardRecipient() (common.Address, error) {
// 	return _BunniHub.Contract.GetReferralRewardRecipient(&_BunniHub.CallOpts)
// }

// // HookParams is a free data retrieval call binding the contract method 0x129f38ea.
// //
// // Solidity: function hookParams(bytes32 poolId) view returns(bytes)
// func (_BunniHub *BunniHubCaller) HookParams(opts *bind.CallOpts, poolId [32]byte) ([]byte, error) {
// 	var out []interface{}
// 	err := _BunniHub.contract.Call(opts, &out, "hookParams", poolId)

// 	if err != nil {
// 		return *new([]byte), err
// 	}

// 	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

// 	return out0, err

// }

// // HookParams is a free data retrieval call binding the contract method 0x129f38ea.
// //
// // Solidity: function hookParams(bytes32 poolId) view returns(bytes)
// func (_BunniHub *BunniHubSession) HookParams(poolId [32]byte) ([]byte, error) {
// 	return _BunniHub.Contract.HookParams(&_BunniHub.CallOpts, poolId)
// }

// // HookParams is a free data retrieval call binding the contract method 0x129f38ea.
// //
// // Solidity: function hookParams(bytes32 poolId) view returns(bytes)
// func (_BunniHub *BunniHubCallerSession) HookParams(poolId [32]byte) ([]byte, error) {
// 	return _BunniHub.Contract.HookParams(&_BunniHub.CallOpts, poolId)
// }

// // IdleBalance is a free data retrieval call binding the contract method 0x88dd6e53.
// //
// // Solidity: function idleBalance(bytes32 poolId) view returns(bytes32)
// func (_BunniHub *BunniHubCaller) IdleBalance(opts *bind.CallOpts, poolId [32]byte) ([32]byte, error) {
// 	var out []interface{}
// 	err := _BunniHub.contract.Call(opts, &out, "idleBalance", poolId)

// 	if err != nil {
// 		return *new([32]byte), err
// 	}

// 	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

// 	return out0, err

// }

// // IdleBalance is a free data retrieval call binding the contract method 0x88dd6e53.
// //
// // Solidity: function idleBalance(bytes32 poolId) view returns(bytes32)
// func (_BunniHub *BunniHubSession) IdleBalance(poolId [32]byte) ([32]byte, error) {
// 	return _BunniHub.Contract.IdleBalance(&_BunniHub.CallOpts, poolId)
// }

// // IdleBalance is a free data retrieval call binding the contract method 0x88dd6e53.
// //
// // Solidity: function idleBalance(bytes32 poolId) view returns(bytes32)
// func (_BunniHub *BunniHubCallerSession) IdleBalance(poolId [32]byte) ([32]byte, error) {
// 	return _BunniHub.Contract.IdleBalance(&_BunniHub.CallOpts, poolId)
// }

// // IsPauser is a free data retrieval call binding the contract method 0x46fbf68e.
// //
// // Solidity: function isPauser(address guy) view returns(bool)
// func (_BunniHub *BunniHubCaller) IsPauser(opts *bind.CallOpts, guy common.Address) (bool, error) {
// 	var out []interface{}
// 	err := _BunniHub.contract.Call(opts, &out, "isPauser", guy)

// 	if err != nil {
// 		return *new(bool), err
// 	}

// 	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

// 	return out0, err

// }

// // IsPauser is a free data retrieval call binding the contract method 0x46fbf68e.
// //
// // Solidity: function isPauser(address guy) view returns(bool)
// func (_BunniHub *BunniHubSession) IsPauser(guy common.Address) (bool, error) {
// 	return _BunniHub.Contract.IsPauser(&_BunniHub.CallOpts, guy)
// }

// // IsPauser is a free data retrieval call binding the contract method 0x46fbf68e.
// //
// // Solidity: function isPauser(address guy) view returns(bool)
// func (_BunniHub *BunniHubCallerSession) IsPauser(guy common.Address) (bool, error) {
// 	return _BunniHub.Contract.IsPauser(&_BunniHub.CallOpts, guy)
// }

// // Nonce is a free data retrieval call binding the contract method 0x905da30f.
// //
// // Solidity: function nonce(bytes32 bunniSubspace) view returns(uint24)
// func (_BunniHub *BunniHubCaller) Nonce(opts *bind.CallOpts, bunniSubspace [32]byte) (*big.Int, error) {
// 	var out []interface{}
// 	err := _BunniHub.contract.Call(opts, &out, "nonce", bunniSubspace)

// 	if err != nil {
// 		return *new(*big.Int), err
// 	}

// 	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

// 	return out0, err

// }

// // Nonce is a free data retrieval call binding the contract method 0x905da30f.
// //
// // Solidity: function nonce(bytes32 bunniSubspace) view returns(uint24)
// func (_BunniHub *BunniHubSession) Nonce(bunniSubspace [32]byte) (*big.Int, error) {
// 	return _BunniHub.Contract.Nonce(&_BunniHub.CallOpts, bunniSubspace)
// }

// // Nonce is a free data retrieval call binding the contract method 0x905da30f.
// //
// // Solidity: function nonce(bytes32 bunniSubspace) view returns(uint24)
// func (_BunniHub *BunniHubCallerSession) Nonce(bunniSubspace [32]byte) (*big.Int, error) {
// 	return _BunniHub.Contract.Nonce(&_BunniHub.CallOpts, bunniSubspace)
// }

// // Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
// //
// // Solidity: function owner() view returns(address result)
// func (_BunniHub *BunniHubCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
// 	var out []interface{}
// 	err := _BunniHub.contract.Call(opts, &out, "owner")

// 	if err != nil {
// 		return *new(common.Address), err
// 	}

// 	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

// 	return out0, err

// }

// // Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
// //
// // Solidity: function owner() view returns(address result)
// func (_BunniHub *BunniHubSession) Owner() (common.Address, error) {
// 	return _BunniHub.Contract.Owner(&_BunniHub.CallOpts)
// }

// // Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
// //
// // Solidity: function owner() view returns(address result)
// func (_BunniHub *BunniHubCallerSession) Owner() (common.Address, error) {
// 	return _BunniHub.Contract.Owner(&_BunniHub.CallOpts)
// }

// // OwnershipHandoverExpiresAt is a free data retrieval call binding the contract method 0xfee81cf4.
// //
// // Solidity: function ownershipHandoverExpiresAt(address pendingOwner) view returns(uint256 result)
// func (_BunniHub *BunniHubCaller) OwnershipHandoverExpiresAt(opts *bind.CallOpts, pendingOwner common.Address) (*big.Int, error) {
// 	var out []interface{}
// 	err := _BunniHub.contract.Call(opts, &out, "ownershipHandoverExpiresAt", pendingOwner)

// 	if err != nil {
// 		return *new(*big.Int), err
// 	}

// 	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

// 	return out0, err

// }

// // OwnershipHandoverExpiresAt is a free data retrieval call binding the contract method 0xfee81cf4.
// //
// // Solidity: function ownershipHandoverExpiresAt(address pendingOwner) view returns(uint256 result)
// func (_BunniHub *BunniHubSession) OwnershipHandoverExpiresAt(pendingOwner common.Address) (*big.Int, error) {
// 	return _BunniHub.Contract.OwnershipHandoverExpiresAt(&_BunniHub.CallOpts, pendingOwner)
// }

// // OwnershipHandoverExpiresAt is a free data retrieval call binding the contract method 0xfee81cf4.
// //
// // Solidity: function ownershipHandoverExpiresAt(address pendingOwner) view returns(uint256 result)
// func (_BunniHub *BunniHubCallerSession) OwnershipHandoverExpiresAt(pendingOwner common.Address) (*big.Int, error) {
// 	return _BunniHub.Contract.OwnershipHandoverExpiresAt(&_BunniHub.CallOpts, pendingOwner)
// }

// // PoolBalances is a free data retrieval call binding the contract method 0x809b1f38.
// //
// // Solidity: function poolBalances(bytes32 poolId) view returns(uint256 balance0, uint256 balance1)
// func (_BunniHub *BunniHubCaller) PoolBalances(opts *bind.CallOpts, poolId [32]byte) (struct {
// 	Balance0 *big.Int
// 	Balance1 *big.Int
// }, error) {
// 	var out []interface{}
// 	err := _BunniHub.contract.Call(opts, &out, "poolBalances", poolId)

// 	outstruct := new(struct {
// 		Balance0 *big.Int
// 		Balance1 *big.Int
// 	})
// 	if err != nil {
// 		return *outstruct, err
// 	}

// 	outstruct.Balance0 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
// 	outstruct.Balance1 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

// 	return *outstruct, err

// }

// // PoolBalances is a free data retrieval call binding the contract method 0x809b1f38.
// //
// // Solidity: function poolBalances(bytes32 poolId) view returns(uint256 balance0, uint256 balance1)
// func (_BunniHub *BunniHubSession) PoolBalances(poolId [32]byte) (struct {
// 	Balance0 *big.Int
// 	Balance1 *big.Int
// }, error) {
// 	return _BunniHub.Contract.PoolBalances(&_BunniHub.CallOpts, poolId)
// }

// // PoolBalances is a free data retrieval call binding the contract method 0x809b1f38.
// //
// // Solidity: function poolBalances(bytes32 poolId) view returns(uint256 balance0, uint256 balance1)
// func (_BunniHub *BunniHubCallerSession) PoolBalances(poolId [32]byte) (struct {
// 	Balance0 *big.Int
// 	Balance1 *big.Int
// }, error) {
// 	return _BunniHub.Contract.PoolBalances(&_BunniHub.CallOpts, poolId)
// }

// // PoolIdOfBunniToken is a free data retrieval call binding the contract method 0x7676cce0.
// //
// // Solidity: function poolIdOfBunniToken(address bunniToken) view returns(bytes32)
// func (_BunniHub *BunniHubCaller) PoolIdOfBunniToken(opts *bind.CallOpts, bunniToken common.Address) ([32]byte, error) {
// 	var out []interface{}
// 	err := _BunniHub.contract.Call(opts, &out, "poolIdOfBunniToken", bunniToken)

// 	if err != nil {
// 		return *new([32]byte), err
// 	}

// 	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

// 	return out0, err

// }

// // PoolIdOfBunniToken is a free data retrieval call binding the contract method 0x7676cce0.
// //
// // Solidity: function poolIdOfBunniToken(address bunniToken) view returns(bytes32)
// func (_BunniHub *BunniHubSession) PoolIdOfBunniToken(bunniToken common.Address) ([32]byte, error) {
// 	return _BunniHub.Contract.PoolIdOfBunniToken(&_BunniHub.CallOpts, bunniToken)
// }

// // PoolIdOfBunniToken is a free data retrieval call binding the contract method 0x7676cce0.
// //
// // Solidity: function poolIdOfBunniToken(address bunniToken) view returns(bytes32)
// func (_BunniHub *BunniHubCallerSession) PoolIdOfBunniToken(bunniToken common.Address) ([32]byte, error) {
// 	return _BunniHub.Contract.PoolIdOfBunniToken(&_BunniHub.CallOpts, bunniToken)
// }

// // PoolInitData is a free data retrieval call binding the contract method 0xf0960848.
// //
// // Solidity: function poolInitData() view returns(bytes)
// func (_BunniHub *BunniHubCaller) PoolInitData(opts *bind.CallOpts) ([]byte, error) {
// 	var out []interface{}
// 	err := _BunniHub.contract.Call(opts, &out, "poolInitData")

// 	if err != nil {
// 		return *new([]byte), err
// 	}

// 	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

// 	return out0, err

// }

// // PoolInitData is a free data retrieval call binding the contract method 0xf0960848.
// //
// // Solidity: function poolInitData() view returns(bytes)
// func (_BunniHub *BunniHubSession) PoolInitData() ([]byte, error) {
// 	return _BunniHub.Contract.PoolInitData(&_BunniHub.CallOpts)
// }

// // PoolInitData is a free data retrieval call binding the contract method 0xf0960848.
// //
// // Solidity: function poolInitData() view returns(bytes)
// func (_BunniHub *BunniHubCallerSession) PoolInitData() ([]byte, error) {
// 	return _BunniHub.Contract.PoolInitData(&_BunniHub.CallOpts)
// }

// // PoolParams is a free data retrieval call binding the contract method 0xa0fd3f7e.
// //
// // Solidity: function poolParams(bytes32 poolId) view returns((address,address,address,uint24,bytes32,bytes,address,address,uint8,uint24,uint24,uint24,uint24,uint24,uint24,uint256,uint256,uint256,uint256,bytes32))
// func (_BunniHub *BunniHubCaller) PoolParams(opts *bind.CallOpts, poolId [32]byte) (PoolState, error) {
// 	var out []interface{}
// 	err := _BunniHub.contract.Call(opts, &out, "poolParams", poolId)

// 	if err != nil {
// 		return *new(PoolState), err
// 	}

// 	out0 := *abi.ConvertType(out[0], new(PoolState)).(*PoolState)

// 	return out0, err

// }

// // PoolParams is a free data retrieval call binding the contract method 0xa0fd3f7e.
// //
// // Solidity: function poolParams(bytes32 poolId) view returns((address,address,address,uint24,bytes32,bytes,address,address,uint8,uint24,uint24,uint24,uint24,uint24,uint24,uint256,uint256,uint256,uint256,bytes32))
// func (_BunniHub *BunniHubSession) PoolParams(poolId [32]byte) (PoolState, error) {
// 	return _BunniHub.Contract.PoolParams(&_BunniHub.CallOpts, poolId)
// }

// // PoolParams is a free data retrieval call binding the contract method 0xa0fd3f7e.
// //
// // Solidity: function poolParams(bytes32 poolId) view returns((address,address,address,uint24,bytes32,bytes,address,address,uint8,uint24,uint24,uint24,uint24,uint24,uint24,uint256,uint256,uint256,uint256,bytes32))
// func (_BunniHub *BunniHubCallerSession) PoolParams(poolId [32]byte) (PoolState, error) {
// 	return _BunniHub.Contract.PoolParams(&_BunniHub.CallOpts, poolId)
// }

// // PoolState is a free data retrieval call binding the contract method 0xe0b01bac.
// //
// // Solidity: function poolState(bytes32 poolId) view returns((address,address,address,uint24,bytes32,bytes,address,address,uint8,uint24,uint24,uint24,uint24,uint24,uint24,uint256,uint256,uint256,uint256,bytes32))
// func (_BunniHub *BunniHubCaller) PoolState(opts *bind.CallOpts, poolId [32]byte) (PoolState, error) {
// 	var out []interface{}
// 	err := _BunniHub.contract.Call(opts, &out, "poolState", poolId)

// 	if err != nil {
// 		return *new(PoolState), err
// 	}

// 	out0 := *abi.ConvertType(out[0], new(PoolState)).(*PoolState)

// 	return out0, err

// }

// // PoolState is a free data retrieval call binding the contract method 0xe0b01bac.
// //
// // Solidity: function poolState(bytes32 poolId) view returns((address,address,address,uint24,bytes32,bytes,address,address,uint8,uint24,uint24,uint24,uint24,uint24,uint24,uint256,uint256,uint256,uint256,bytes32))
// func (_BunniHub *BunniHubSession) PoolState(poolId [32]byte) (PoolState, error) {
// 	return _BunniHub.Contract.PoolState(&_BunniHub.CallOpts, poolId)
// }

// // PoolState is a free data retrieval call binding the contract method 0xe0b01bac.
// //
// // Solidity: function poolState(bytes32 poolId) view returns((address,address,address,uint24,bytes32,bytes,address,address,uint8,uint24,uint24,uint24,uint24,uint24,uint24,uint256,uint256,uint256,uint256,bytes32))
// func (_BunniHub *BunniHubCallerSession) PoolState(poolId [32]byte) (PoolState, error) {
// 	return _BunniHub.Contract.PoolState(&_BunniHub.CallOpts, poolId)
// }

// // BurnPauseFuse is a paid mutator transaction binding the contract method 0x1ed08cb9.
// //
// // Solidity: function burnPauseFuse() returns()
// func (_BunniHub *BunniHubTransactor) BurnPauseFuse(opts *bind.TransactOpts) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "burnPauseFuse")
// }

// // BurnPauseFuse is a paid mutator transaction binding the contract method 0x1ed08cb9.
// //
// // Solidity: function burnPauseFuse() returns()
// func (_BunniHub *BunniHubSession) BurnPauseFuse() (*types.Transaction, error) {
// 	return _BunniHub.Contract.BurnPauseFuse(&_BunniHub.TransactOpts)
// }

// // BurnPauseFuse is a paid mutator transaction binding the contract method 0x1ed08cb9.
// //
// // Solidity: function burnPauseFuse() returns()
// func (_BunniHub *BunniHubTransactorSession) BurnPauseFuse() (*types.Transaction, error) {
// 	return _BunniHub.Contract.BurnPauseFuse(&_BunniHub.TransactOpts)
// }

// // CancelOwnershipHandover is a paid mutator transaction binding the contract method 0x54d1f13d.
// //
// // Solidity: function cancelOwnershipHandover() payable returns()
// func (_BunniHub *BunniHubTransactor) CancelOwnershipHandover(opts *bind.TransactOpts) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "cancelOwnershipHandover")
// }

// // CancelOwnershipHandover is a paid mutator transaction binding the contract method 0x54d1f13d.
// //
// // Solidity: function cancelOwnershipHandover() payable returns()
// func (_BunniHub *BunniHubSession) CancelOwnershipHandover() (*types.Transaction, error) {
// 	return _BunniHub.Contract.CancelOwnershipHandover(&_BunniHub.TransactOpts)
// }

// // CancelOwnershipHandover is a paid mutator transaction binding the contract method 0x54d1f13d.
// //
// // Solidity: function cancelOwnershipHandover() payable returns()
// func (_BunniHub *BunniHubTransactorSession) CancelOwnershipHandover() (*types.Transaction, error) {
// 	return _BunniHub.Contract.CancelOwnershipHandover(&_BunniHub.TransactOpts)
// }

// // CompleteOwnershipHandover is a paid mutator transaction binding the contract method 0xf04e283e.
// //
// // Solidity: function completeOwnershipHandover(address pendingOwner) payable returns()
// func (_BunniHub *BunniHubTransactor) CompleteOwnershipHandover(opts *bind.TransactOpts, pendingOwner common.Address) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "completeOwnershipHandover", pendingOwner)
// }

// // CompleteOwnershipHandover is a paid mutator transaction binding the contract method 0xf04e283e.
// //
// // Solidity: function completeOwnershipHandover(address pendingOwner) payable returns()
// func (_BunniHub *BunniHubSession) CompleteOwnershipHandover(pendingOwner common.Address) (*types.Transaction, error) {
// 	return _BunniHub.Contract.CompleteOwnershipHandover(&_BunniHub.TransactOpts, pendingOwner)
// }

// // CompleteOwnershipHandover is a paid mutator transaction binding the contract method 0xf04e283e.
// //
// // Solidity: function completeOwnershipHandover(address pendingOwner) payable returns()
// func (_BunniHub *BunniHubTransactorSession) CompleteOwnershipHandover(pendingOwner common.Address) (*types.Transaction, error) {
// 	return _BunniHub.Contract.CompleteOwnershipHandover(&_BunniHub.TransactOpts, pendingOwner)
// }

// // DeployBunniToken is a paid mutator transaction binding the contract method 0xe56ba808.
// //
// // Solidity: function deployBunniToken((address,address,int24,uint24,address,address,uint8,bytes32,address,bytes,address,address,uint24,uint24,uint24,uint24,uint24,uint24,uint160,bytes32,bytes32,address,string,bytes32) params) returns(address token, (address,address,uint24,int24,address) key)
// func (_BunniHub *BunniHubTransactor) DeployBunniToken(opts *bind.TransactOpts, params IBunniHubDeployBunniTokenParams) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "deployBunniToken", params)
// }

// // DeployBunniToken is a paid mutator transaction binding the contract method 0xe56ba808.
// //
// // Solidity: function deployBunniToken((address,address,int24,uint24,address,address,uint8,bytes32,address,bytes,address,address,uint24,uint24,uint24,uint24,uint24,uint24,uint160,bytes32,bytes32,address,string,bytes32) params) returns(address token, (address,address,uint24,int24,address) key)
// func (_BunniHub *BunniHubSession) DeployBunniToken(params IBunniHubDeployBunniTokenParams) (*types.Transaction, error) {
// 	return _BunniHub.Contract.DeployBunniToken(&_BunniHub.TransactOpts, params)
// }

// // DeployBunniToken is a paid mutator transaction binding the contract method 0xe56ba808.
// //
// // Solidity: function deployBunniToken((address,address,int24,uint24,address,address,uint8,bytes32,address,bytes,address,address,uint24,uint24,uint24,uint24,uint24,uint24,uint160,bytes32,bytes32,address,string,bytes32) params) returns(address token, (address,address,uint24,int24,address) key)
// func (_BunniHub *BunniHubTransactorSession) DeployBunniToken(params IBunniHubDeployBunniTokenParams) (*types.Transaction, error) {
// 	return _BunniHub.Contract.DeployBunniToken(&_BunniHub.TransactOpts, params)
// }

// // Deposit is a paid mutator transaction binding the contract method 0xf69da336.
// //
// // Solidity: function deposit(((address,address,uint24,int24,address),address,address,uint256,uint256,uint256,uint256,uint256,uint256,uint256,address) params) payable returns(uint256 shares, uint256 amount0, uint256 amount1)
// func (_BunniHub *BunniHubTransactor) Deposit(opts *bind.TransactOpts, params IBunniHubDepositParams) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "deposit", params)
// }

// // Deposit is a paid mutator transaction binding the contract method 0xf69da336.
// //
// // Solidity: function deposit(((address,address,uint24,int24,address),address,address,uint256,uint256,uint256,uint256,uint256,uint256,uint256,address) params) payable returns(uint256 shares, uint256 amount0, uint256 amount1)
// func (_BunniHub *BunniHubSession) Deposit(params IBunniHubDepositParams) (*types.Transaction, error) {
// 	return _BunniHub.Contract.Deposit(&_BunniHub.TransactOpts, params)
// }

// // Deposit is a paid mutator transaction binding the contract method 0xf69da336.
// //
// // Solidity: function deposit(((address,address,uint24,int24,address),address,address,uint256,uint256,uint256,uint256,uint256,uint256,uint256,address) params) payable returns(uint256 shares, uint256 amount0, uint256 amount1)
// func (_BunniHub *BunniHubTransactorSession) Deposit(params IBunniHubDepositParams) (*types.Transaction, error) {
// 	return _BunniHub.Contract.Deposit(&_BunniHub.TransactOpts, params)
// }

// // HookHandleSwap is a paid mutator transaction binding the contract method 0xf89ee44e.
// //
// // Solidity: function hookHandleSwap((address,address,uint24,int24,address) key, bool zeroForOne, uint256 inputAmount, uint256 outputAmount) returns()
// func (_BunniHub *BunniHubTransactor) HookHandleSwap(opts *bind.TransactOpts, key PoolKey, zeroForOne bool, inputAmount *big.Int, outputAmount *big.Int) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "hookHandleSwap", key, zeroForOne, inputAmount, outputAmount)
// }

// // HookHandleSwap is a paid mutator transaction binding the contract method 0xf89ee44e.
// //
// // Solidity: function hookHandleSwap((address,address,uint24,int24,address) key, bool zeroForOne, uint256 inputAmount, uint256 outputAmount) returns()
// func (_BunniHub *BunniHubSession) HookHandleSwap(key PoolKey, zeroForOne bool, inputAmount *big.Int, outputAmount *big.Int) (*types.Transaction, error) {
// 	return _BunniHub.Contract.HookHandleSwap(&_BunniHub.TransactOpts, key, zeroForOne, inputAmount, outputAmount)
// }

// // HookHandleSwap is a paid mutator transaction binding the contract method 0xf89ee44e.
// //
// // Solidity: function hookHandleSwap((address,address,uint24,int24,address) key, bool zeroForOne, uint256 inputAmount, uint256 outputAmount) returns()
// func (_BunniHub *BunniHubTransactorSession) HookHandleSwap(key PoolKey, zeroForOne bool, inputAmount *big.Int, outputAmount *big.Int) (*types.Transaction, error) {
// 	return _BunniHub.Contract.HookHandleSwap(&_BunniHub.TransactOpts, key, zeroForOne, inputAmount, outputAmount)
// }

// // HookSetIdleBalance is a paid mutator transaction binding the contract method 0xef760335.
// //
// // Solidity: function hookSetIdleBalance((address,address,uint24,int24,address) key, bytes32 newIdleBalance) returns()
// func (_BunniHub *BunniHubTransactor) HookSetIdleBalance(opts *bind.TransactOpts, key PoolKey, newIdleBalance [32]byte) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "hookSetIdleBalance", key, newIdleBalance)
// }

// // HookSetIdleBalance is a paid mutator transaction binding the contract method 0xef760335.
// //
// // Solidity: function hookSetIdleBalance((address,address,uint24,int24,address) key, bytes32 newIdleBalance) returns()
// func (_BunniHub *BunniHubSession) HookSetIdleBalance(key PoolKey, newIdleBalance [32]byte) (*types.Transaction, error) {
// 	return _BunniHub.Contract.HookSetIdleBalance(&_BunniHub.TransactOpts, key, newIdleBalance)
// }

// // HookSetIdleBalance is a paid mutator transaction binding the contract method 0xef760335.
// //
// // Solidity: function hookSetIdleBalance((address,address,uint24,int24,address) key, bytes32 newIdleBalance) returns()
// func (_BunniHub *BunniHubTransactorSession) HookSetIdleBalance(key PoolKey, newIdleBalance [32]byte) (*types.Transaction, error) {
// 	return _BunniHub.Contract.HookSetIdleBalance(&_BunniHub.TransactOpts, key, newIdleBalance)
// }

// // LockForRebalance is a paid mutator transaction binding the contract method 0x3fac6506.
// //
// // Solidity: function lockForRebalance((address,address,uint24,int24,address) key) returns()
// func (_BunniHub *BunniHubTransactor) LockForRebalance(opts *bind.TransactOpts, key PoolKey) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "lockForRebalance", key)
// }

// // LockForRebalance is a paid mutator transaction binding the contract method 0x3fac6506.
// //
// // Solidity: function lockForRebalance((address,address,uint24,int24,address) key) returns()
// func (_BunniHub *BunniHubSession) LockForRebalance(key PoolKey) (*types.Transaction, error) {
// 	return _BunniHub.Contract.LockForRebalance(&_BunniHub.TransactOpts, key)
// }

// // LockForRebalance is a paid mutator transaction binding the contract method 0x3fac6506.
// //
// // Solidity: function lockForRebalance((address,address,uint24,int24,address) key) returns()
// func (_BunniHub *BunniHubTransactorSession) LockForRebalance(key PoolKey) (*types.Transaction, error) {
// 	return _BunniHub.Contract.LockForRebalance(&_BunniHub.TransactOpts, key)
// }

// // QueueWithdraw is a paid mutator transaction binding the contract method 0x5658d0b4.
// //
// // Solidity: function queueWithdraw(((address,address,uint24,int24,address),uint200) params) returns()
// func (_BunniHub *BunniHubTransactor) QueueWithdraw(opts *bind.TransactOpts, params IBunniHubQueueWithdrawParams) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "queueWithdraw", params)
// }

// // QueueWithdraw is a paid mutator transaction binding the contract method 0x5658d0b4.
// //
// // Solidity: function queueWithdraw(((address,address,uint24,int24,address),uint200) params) returns()
// func (_BunniHub *BunniHubSession) QueueWithdraw(params IBunniHubQueueWithdrawParams) (*types.Transaction, error) {
// 	return _BunniHub.Contract.QueueWithdraw(&_BunniHub.TransactOpts, params)
// }

// // QueueWithdraw is a paid mutator transaction binding the contract method 0x5658d0b4.
// //
// // Solidity: function queueWithdraw(((address,address,uint24,int24,address),uint200) params) returns()
// func (_BunniHub *BunniHubTransactorSession) QueueWithdraw(params IBunniHubQueueWithdrawParams) (*types.Transaction, error) {
// 	return _BunniHub.Contract.QueueWithdraw(&_BunniHub.TransactOpts, params)
// }

// // RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
// //
// // Solidity: function renounceOwnership() payable returns()
// func (_BunniHub *BunniHubTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "renounceOwnership")
// }

// // RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
// //
// // Solidity: function renounceOwnership() payable returns()
// func (_BunniHub *BunniHubSession) RenounceOwnership() (*types.Transaction, error) {
// 	return _BunniHub.Contract.RenounceOwnership(&_BunniHub.TransactOpts)
// }

// // RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
// //
// // Solidity: function renounceOwnership() payable returns()
// func (_BunniHub *BunniHubTransactorSession) RenounceOwnership() (*types.Transaction, error) {
// 	return _BunniHub.Contract.RenounceOwnership(&_BunniHub.TransactOpts)
// }

// // RequestOwnershipHandover is a paid mutator transaction binding the contract method 0x25692962.
// //
// // Solidity: function requestOwnershipHandover() payable returns()
// func (_BunniHub *BunniHubTransactor) RequestOwnershipHandover(opts *bind.TransactOpts) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "requestOwnershipHandover")
// }

// // RequestOwnershipHandover is a paid mutator transaction binding the contract method 0x25692962.
// //
// // Solidity: function requestOwnershipHandover() payable returns()
// func (_BunniHub *BunniHubSession) RequestOwnershipHandover() (*types.Transaction, error) {
// 	return _BunniHub.Contract.RequestOwnershipHandover(&_BunniHub.TransactOpts)
// }

// // RequestOwnershipHandover is a paid mutator transaction binding the contract method 0x25692962.
// //
// // Solidity: function requestOwnershipHandover() payable returns()
// func (_BunniHub *BunniHubTransactorSession) RequestOwnershipHandover() (*types.Transaction, error) {
// 	return _BunniHub.Contract.RequestOwnershipHandover(&_BunniHub.TransactOpts)
// }

// // SetPauseFlags is a paid mutator transaction binding the contract method 0xa56dd053.
// //
// // Solidity: function setPauseFlags(uint8 pauseFlags) returns()
// func (_BunniHub *BunniHubTransactor) SetPauseFlags(opts *bind.TransactOpts, pauseFlags uint8) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "setPauseFlags", pauseFlags)
// }

// // SetPauseFlags is a paid mutator transaction binding the contract method 0xa56dd053.
// //
// // Solidity: function setPauseFlags(uint8 pauseFlags) returns()
// func (_BunniHub *BunniHubSession) SetPauseFlags(pauseFlags uint8) (*types.Transaction, error) {
// 	return _BunniHub.Contract.SetPauseFlags(&_BunniHub.TransactOpts, pauseFlags)
// }

// // SetPauseFlags is a paid mutator transaction binding the contract method 0xa56dd053.
// //
// // Solidity: function setPauseFlags(uint8 pauseFlags) returns()
// func (_BunniHub *BunniHubTransactorSession) SetPauseFlags(pauseFlags uint8) (*types.Transaction, error) {
// 	return _BunniHub.Contract.SetPauseFlags(&_BunniHub.TransactOpts, pauseFlags)
// }

// // SetPauser is a paid mutator transaction binding the contract method 0x7180c8ca.
// //
// // Solidity: function setPauser(address guy, bool status) returns()
// func (_BunniHub *BunniHubTransactor) SetPauser(opts *bind.TransactOpts, guy common.Address, status bool) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "setPauser", guy, status)
// }

// // SetPauser is a paid mutator transaction binding the contract method 0x7180c8ca.
// //
// // Solidity: function setPauser(address guy, bool status) returns()
// func (_BunniHub *BunniHubSession) SetPauser(guy common.Address, status bool) (*types.Transaction, error) {
// 	return _BunniHub.Contract.SetPauser(&_BunniHub.TransactOpts, guy, status)
// }

// // SetPauser is a paid mutator transaction binding the contract method 0x7180c8ca.
// //
// // Solidity: function setPauser(address guy, bool status) returns()
// func (_BunniHub *BunniHubTransactorSession) SetPauser(guy common.Address, status bool) (*types.Transaction, error) {
// 	return _BunniHub.Contract.SetPauser(&_BunniHub.TransactOpts, guy, status)
// }

// // SetReferralRewardRecipient is a paid mutator transaction binding the contract method 0xcd639491.
// //
// // Solidity: function setReferralRewardRecipient(address newReferralRewardRecipient) returns()
// func (_BunniHub *BunniHubTransactor) SetReferralRewardRecipient(opts *bind.TransactOpts, newReferralRewardRecipient common.Address) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "setReferralRewardRecipient", newReferralRewardRecipient)
// }

// // SetReferralRewardRecipient is a paid mutator transaction binding the contract method 0xcd639491.
// //
// // Solidity: function setReferralRewardRecipient(address newReferralRewardRecipient) returns()
// func (_BunniHub *BunniHubSession) SetReferralRewardRecipient(newReferralRewardRecipient common.Address) (*types.Transaction, error) {
// 	return _BunniHub.Contract.SetReferralRewardRecipient(&_BunniHub.TransactOpts, newReferralRewardRecipient)
// }

// // SetReferralRewardRecipient is a paid mutator transaction binding the contract method 0xcd639491.
// //
// // Solidity: function setReferralRewardRecipient(address newReferralRewardRecipient) returns()
// func (_BunniHub *BunniHubTransactorSession) SetReferralRewardRecipient(newReferralRewardRecipient common.Address) (*types.Transaction, error) {
// 	return _BunniHub.Contract.SetReferralRewardRecipient(&_BunniHub.TransactOpts, newReferralRewardRecipient)
// }

// // TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
// //
// // Solidity: function transferOwnership(address newOwner) payable returns()
// func (_BunniHub *BunniHubTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "transferOwnership", newOwner)
// }

// // TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
// //
// // Solidity: function transferOwnership(address newOwner) payable returns()
// func (_BunniHub *BunniHubSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
// 	return _BunniHub.Contract.TransferOwnership(&_BunniHub.TransactOpts, newOwner)
// }

// // TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
// //
// // Solidity: function transferOwnership(address newOwner) payable returns()
// func (_BunniHub *BunniHubTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
// 	return _BunniHub.Contract.TransferOwnership(&_BunniHub.TransactOpts, newOwner)
// }

// // UnlockCallback is a paid mutator transaction binding the contract method 0x91dd7346.
// //
// // Solidity: function unlockCallback(bytes data) returns(bytes)
// func (_BunniHub *BunniHubTransactor) UnlockCallback(opts *bind.TransactOpts, data []byte) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "unlockCallback", data)
// }

// // UnlockCallback is a paid mutator transaction binding the contract method 0x91dd7346.
// //
// // Solidity: function unlockCallback(bytes data) returns(bytes)
// func (_BunniHub *BunniHubSession) UnlockCallback(data []byte) (*types.Transaction, error) {
// 	return _BunniHub.Contract.UnlockCallback(&_BunniHub.TransactOpts, data)
// }

// // UnlockCallback is a paid mutator transaction binding the contract method 0x91dd7346.
// //
// // Solidity: function unlockCallback(bytes data) returns(bytes)
// func (_BunniHub *BunniHubTransactorSession) UnlockCallback(data []byte) (*types.Transaction, error) {
// 	return _BunniHub.Contract.UnlockCallback(&_BunniHub.TransactOpts, data)
// }

// // UnlockForRebalance is a paid mutator transaction binding the contract method 0x9445c4a8.
// //
// // Solidity: function unlockForRebalance((address,address,uint24,int24,address) key) returns()
// func (_BunniHub *BunniHubTransactor) UnlockForRebalance(opts *bind.TransactOpts, key PoolKey) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "unlockForRebalance", key)
// }

// // UnlockForRebalance is a paid mutator transaction binding the contract method 0x9445c4a8.
// //
// // Solidity: function unlockForRebalance((address,address,uint24,int24,address) key) returns()
// func (_BunniHub *BunniHubSession) UnlockForRebalance(key PoolKey) (*types.Transaction, error) {
// 	return _BunniHub.Contract.UnlockForRebalance(&_BunniHub.TransactOpts, key)
// }

// // UnlockForRebalance is a paid mutator transaction binding the contract method 0x9445c4a8.
// //
// // Solidity: function unlockForRebalance((address,address,uint24,int24,address) key) returns()
// func (_BunniHub *BunniHubTransactorSession) UnlockForRebalance(key PoolKey) (*types.Transaction, error) {
// 	return _BunniHub.Contract.UnlockForRebalance(&_BunniHub.TransactOpts, key)
// }

// // Withdraw is a paid mutator transaction binding the contract method 0x5d4a505e.
// //
// // Solidity: function withdraw(((address,address,uint24,int24,address),address,uint256,uint256,uint256,uint256,bool) params) returns(uint256 amount0, uint256 amount1)
// func (_BunniHub *BunniHubTransactor) Withdraw(opts *bind.TransactOpts, params IBunniHubWithdrawParams) (*types.Transaction, error) {
// 	return _BunniHub.contract.Transact(opts, "withdraw", params)
// }

// // Withdraw is a paid mutator transaction binding the contract method 0x5d4a505e.
// //
// // Solidity: function withdraw(((address,address,uint24,int24,address),address,uint256,uint256,uint256,uint256,bool) params) returns(uint256 amount0, uint256 amount1)
// func (_BunniHub *BunniHubSession) Withdraw(params IBunniHubWithdrawParams) (*types.Transaction, error) {
// 	return _BunniHub.Contract.Withdraw(&_BunniHub.TransactOpts, params)
// }

// // Withdraw is a paid mutator transaction binding the contract method 0x5d4a505e.
// //
// // Solidity: function withdraw(((address,address,uint24,int24,address),address,uint256,uint256,uint256,uint256,bool) params) returns(uint256 amount0, uint256 amount1)
// func (_BunniHub *BunniHubTransactorSession) Withdraw(params IBunniHubWithdrawParams) (*types.Transaction, error) {
// 	return _BunniHub.Contract.Withdraw(&_BunniHub.TransactOpts, params)
// }

// // Receive is a paid mutator transaction binding the contract receive function.
// //
// // Solidity: receive() payable returns()
// func (_BunniHub *BunniHubTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
// 	return _BunniHub.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
// }

// // Receive is a paid mutator transaction binding the contract receive function.
// //
// // Solidity: receive() payable returns()
// func (_BunniHub *BunniHubSession) Receive() (*types.Transaction, error) {
// 	return _BunniHub.Contract.Receive(&_BunniHub.TransactOpts)
// }

// // Receive is a paid mutator transaction binding the contract receive function.
// //
// // Solidity: receive() payable returns()
// func (_BunniHub *BunniHubTransactorSession) Receive() (*types.Transaction, error) {
// 	return _BunniHub.Contract.Receive(&_BunniHub.TransactOpts)
// }

// // BunniHubBurnPauseFuseIterator is returned from FilterBurnPauseFuse and is used to iterate over the raw logs and unpacked data for BurnPauseFuse events raised by the BunniHub contract.
// type BunniHubBurnPauseFuseIterator struct {
// 	Event *BunniHubBurnPauseFuse // Event containing the contract specifics and raw log

// 	contract *bind.BoundContract // Generic contract to use for unpacking event data
// 	event    string              // Event name to use for unpacking event data

// 	logs chan types.Log        // Log channel receiving the found contract events
// 	sub  ethereum.Subscription // Subscription for errors, completion and termination
// 	done bool                  // Whether the subscription completed delivering logs
// 	fail error                 // Occurred error to stop iteration
// }

// // Next advances the iterator to the subsequent event, returning whether there
// // are any more events found. In case of a retrieval or parsing error, false is
// // returned and Error() can be queried for the exact failure.
// func (it *BunniHubBurnPauseFuseIterator) Next() bool {
// 	// If the iterator failed, stop iterating
// 	if it.fail != nil {
// 		return false
// 	}
// 	// If the iterator completed, deliver directly whatever's available
// 	if it.done {
// 		select {
// 		case log := <-it.logs:
// 			it.Event = new(BunniHubBurnPauseFuse)
// 			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 				it.fail = err
// 				return false
// 			}
// 			it.Event.Raw = log
// 			return true

// 		default:
// 			return false
// 		}
// 	}
// 	// Iterator still in progress, wait for either a data or an error event
// 	select {
// 	case log := <-it.logs:
// 		it.Event = new(BunniHubBurnPauseFuse)
// 		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 			it.fail = err
// 			return false
// 		}
// 		it.Event.Raw = log
// 		return true

// 	case err := <-it.sub.Err():
// 		it.done = true
// 		it.fail = err
// 		return it.Next()
// 	}
// }

// // Error returns any retrieval or parsing error occurred during filtering.
// func (it *BunniHubBurnPauseFuseIterator) Error() error {
// 	return it.fail
// }

// // Close terminates the iteration process, releasing any pending underlying
// // resources.
// func (it *BunniHubBurnPauseFuseIterator) Close() error {
// 	it.sub.Unsubscribe()
// 	return nil
// }

// // BunniHubBurnPauseFuse represents a BurnPauseFuse event raised by the BunniHub contract.
// type BunniHubBurnPauseFuse struct {
// 	Raw types.Log // Blockchain specific contextual infos
// }

// // FilterBurnPauseFuse is a free log retrieval operation binding the contract event 0xa4058ba547bb832da5ae671cc4d748c09c98c85226ec320325d641a1a3d64adf.
// //
// // Solidity: event BurnPauseFuse()
// func (_BunniHub *BunniHubFilterer) FilterBurnPauseFuse(opts *bind.FilterOpts) (*BunniHubBurnPauseFuseIterator, error) {

// 	logs, sub, err := _BunniHub.contract.FilterLogs(opts, "BurnPauseFuse")
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &BunniHubBurnPauseFuseIterator{contract: _BunniHub.contract, event: "BurnPauseFuse", logs: logs, sub: sub}, nil
// }

// // WatchBurnPauseFuse is a free log subscription operation binding the contract event 0xa4058ba547bb832da5ae671cc4d748c09c98c85226ec320325d641a1a3d64adf.
// //
// // Solidity: event BurnPauseFuse()
// func (_BunniHub *BunniHubFilterer) WatchBurnPauseFuse(opts *bind.WatchOpts, sink chan<- *BunniHubBurnPauseFuse) (event.Subscription, error) {

// 	logs, sub, err := _BunniHub.contract.WatchLogs(opts, "BurnPauseFuse")
// 	if err != nil {
// 		return nil, err
// 	}
// 	return event.NewSubscription(func(quit <-chan struct{}) error {
// 		defer sub.Unsubscribe()
// 		for {
// 			select {
// 			case log := <-logs:
// 				// New log arrived, parse the event and forward to the user
// 				event := new(BunniHubBurnPauseFuse)
// 				if err := _BunniHub.contract.UnpackLog(event, "BurnPauseFuse", log); err != nil {
// 					return err
// 				}
// 				event.Raw = log

// 				select {
// 				case sink <- event:
// 				case err := <-sub.Err():
// 					return err
// 				case <-quit:
// 					return nil
// 				}
// 			case err := <-sub.Err():
// 				return err
// 			case <-quit:
// 				return nil
// 			}
// 		}
// 	}), nil
// }

// // ParseBurnPauseFuse is a log parse operation binding the contract event 0xa4058ba547bb832da5ae671cc4d748c09c98c85226ec320325d641a1a3d64adf.
// //
// // Solidity: event BurnPauseFuse()
// func (_BunniHub *BunniHubFilterer) ParseBurnPauseFuse(log types.Log) (*BunniHubBurnPauseFuse, error) {
// 	event := new(BunniHubBurnPauseFuse)
// 	if err := _BunniHub.contract.UnpackLog(event, "BurnPauseFuse", log); err != nil {
// 		return nil, err
// 	}
// 	event.Raw = log
// 	return event, nil
// }

// // BunniHubDepositIterator is returned from FilterDeposit and is used to iterate over the raw logs and unpacked data for Deposit events raised by the BunniHub contract.
// type BunniHubDepositIterator struct {
// 	Event *BunniHubDeposit // Event containing the contract specifics and raw log

// 	contract *bind.BoundContract // Generic contract to use for unpacking event data
// 	event    string              // Event name to use for unpacking event data

// 	logs chan types.Log        // Log channel receiving the found contract events
// 	sub  ethereum.Subscription // Subscription for errors, completion and termination
// 	done bool                  // Whether the subscription completed delivering logs
// 	fail error                 // Occurred error to stop iteration
// }

// // Next advances the iterator to the subsequent event, returning whether there
// // are any more events found. In case of a retrieval or parsing error, false is
// // returned and Error() can be queried for the exact failure.
// func (it *BunniHubDepositIterator) Next() bool {
// 	// If the iterator failed, stop iterating
// 	if it.fail != nil {
// 		return false
// 	}
// 	// If the iterator completed, deliver directly whatever's available
// 	if it.done {
// 		select {
// 		case log := <-it.logs:
// 			it.Event = new(BunniHubDeposit)
// 			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 				it.fail = err
// 				return false
// 			}
// 			it.Event.Raw = log
// 			return true

// 		default:
// 			return false
// 		}
// 	}
// 	// Iterator still in progress, wait for either a data or an error event
// 	select {
// 	case log := <-it.logs:
// 		it.Event = new(BunniHubDeposit)
// 		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 			it.fail = err
// 			return false
// 		}
// 		it.Event.Raw = log
// 		return true

// 	case err := <-it.sub.Err():
// 		it.done = true
// 		it.fail = err
// 		return it.Next()
// 	}
// }

// // Error returns any retrieval or parsing error occurred during filtering.
// func (it *BunniHubDepositIterator) Error() error {
// 	return it.fail
// }

// // Close terminates the iteration process, releasing any pending underlying
// // resources.
// func (it *BunniHubDepositIterator) Close() error {
// 	it.sub.Unsubscribe()
// 	return nil
// }

// // BunniHubDeposit represents a Deposit event raised by the BunniHub contract.
// type BunniHubDeposit struct {
// 	Sender    common.Address
// 	Recipient common.Address
// 	PoolId    [32]byte
// 	Amount0   *big.Int
// 	Amount1   *big.Int
// 	Shares    *big.Int
// 	Raw       types.Log // Blockchain specific contextual infos
// }

// // FilterDeposit is a free log retrieval operation binding the contract event 0xb18066d48ef2004e3dcc96ec09f8e738f9e8692565ae7108c2b593f8199af466.
// //
// // Solidity: event Deposit(address indexed sender, address indexed recipient, bytes32 indexed poolId, uint256 amount0, uint256 amount1, uint256 shares)
// func (_BunniHub *BunniHubFilterer) FilterDeposit(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address, poolId [][32]byte) (*BunniHubDepositIterator, error) {

// 	var senderRule []interface{}
// 	for _, senderItem := range sender {
// 		senderRule = append(senderRule, senderItem)
// 	}
// 	var recipientRule []interface{}
// 	for _, recipientItem := range recipient {
// 		recipientRule = append(recipientRule, recipientItem)
// 	}
// 	var poolIdRule []interface{}
// 	for _, poolIdItem := range poolId {
// 		poolIdRule = append(poolIdRule, poolIdItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.FilterLogs(opts, "Deposit", senderRule, recipientRule, poolIdRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &BunniHubDepositIterator{contract: _BunniHub.contract, event: "Deposit", logs: logs, sub: sub}, nil
// }

// // WatchDeposit is a free log subscription operation binding the contract event 0xb18066d48ef2004e3dcc96ec09f8e738f9e8692565ae7108c2b593f8199af466.
// //
// // Solidity: event Deposit(address indexed sender, address indexed recipient, bytes32 indexed poolId, uint256 amount0, uint256 amount1, uint256 shares)
// func (_BunniHub *BunniHubFilterer) WatchDeposit(opts *bind.WatchOpts, sink chan<- *BunniHubDeposit, sender []common.Address, recipient []common.Address, poolId [][32]byte) (event.Subscription, error) {

// 	var senderRule []interface{}
// 	for _, senderItem := range sender {
// 		senderRule = append(senderRule, senderItem)
// 	}
// 	var recipientRule []interface{}
// 	for _, recipientItem := range recipient {
// 		recipientRule = append(recipientRule, recipientItem)
// 	}
// 	var poolIdRule []interface{}
// 	for _, poolIdItem := range poolId {
// 		poolIdRule = append(poolIdRule, poolIdItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.WatchLogs(opts, "Deposit", senderRule, recipientRule, poolIdRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return event.NewSubscription(func(quit <-chan struct{}) error {
// 		defer sub.Unsubscribe()
// 		for {
// 			select {
// 			case log := <-logs:
// 				// New log arrived, parse the event and forward to the user
// 				event := new(BunniHubDeposit)
// 				if err := _BunniHub.contract.UnpackLog(event, "Deposit", log); err != nil {
// 					return err
// 				}
// 				event.Raw = log

// 				select {
// 				case sink <- event:
// 				case err := <-sub.Err():
// 					return err
// 				case <-quit:
// 					return nil
// 				}
// 			case err := <-sub.Err():
// 				return err
// 			case <-quit:
// 				return nil
// 			}
// 		}
// 	}), nil
// }

// // ParseDeposit is a log parse operation binding the contract event 0xb18066d48ef2004e3dcc96ec09f8e738f9e8692565ae7108c2b593f8199af466.
// //
// // Solidity: event Deposit(address indexed sender, address indexed recipient, bytes32 indexed poolId, uint256 amount0, uint256 amount1, uint256 shares)
// func (_BunniHub *BunniHubFilterer) ParseDeposit(log types.Log) (*BunniHubDeposit, error) {
// 	event := new(BunniHubDeposit)
// 	if err := _BunniHub.contract.UnpackLog(event, "Deposit", log); err != nil {
// 		return nil, err
// 	}
// 	event.Raw = log
// 	return event, nil
// }

// // BunniHubNewBunniIterator is returned from FilterNewBunni and is used to iterate over the raw logs and unpacked data for NewBunni events raised by the BunniHub contract.
// type BunniHubNewBunniIterator struct {
// 	Event *BunniHubNewBunni // Event containing the contract specifics and raw log

// 	contract *bind.BoundContract // Generic contract to use for unpacking event data
// 	event    string              // Event name to use for unpacking event data

// 	logs chan types.Log        // Log channel receiving the found contract events
// 	sub  ethereum.Subscription // Subscription for errors, completion and termination
// 	done bool                  // Whether the subscription completed delivering logs
// 	fail error                 // Occurred error to stop iteration
// }

// // Next advances the iterator to the subsequent event, returning whether there
// // are any more events found. In case of a retrieval or parsing error, false is
// // returned and Error() can be queried for the exact failure.
// func (it *BunniHubNewBunniIterator) Next() bool {
// 	// If the iterator failed, stop iterating
// 	if it.fail != nil {
// 		return false
// 	}
// 	// If the iterator completed, deliver directly whatever's available
// 	if it.done {
// 		select {
// 		case log := <-it.logs:
// 			it.Event = new(BunniHubNewBunni)
// 			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 				it.fail = err
// 				return false
// 			}
// 			it.Event.Raw = log
// 			return true

// 		default:
// 			return false
// 		}
// 	}
// 	// Iterator still in progress, wait for either a data or an error event
// 	select {
// 	case log := <-it.logs:
// 		it.Event = new(BunniHubNewBunni)
// 		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 			it.fail = err
// 			return false
// 		}
// 		it.Event.Raw = log
// 		return true

// 	case err := <-it.sub.Err():
// 		it.done = true
// 		it.fail = err
// 		return it.Next()
// 	}
// }

// // Error returns any retrieval or parsing error occurred during filtering.
// func (it *BunniHubNewBunniIterator) Error() error {
// 	return it.fail
// }

// // Close terminates the iteration process, releasing any pending underlying
// // resources.
// func (it *BunniHubNewBunniIterator) Close() error {
// 	it.sub.Unsubscribe()
// 	return nil
// }

// // BunniHubNewBunni represents a NewBunni event raised by the BunniHub contract.
// type BunniHubNewBunni struct {
// 	BunniToken common.Address
// 	PoolId     [32]byte
// 	Raw        types.Log // Blockchain specific contextual infos
// }

// // FilterNewBunni is a free log retrieval operation binding the contract event 0x3ba5df143a8e4c83b7cb37037a87ae0cfb0f8c3a784d4a10e0b329d5706dce1a.
// //
// // Solidity: event NewBunni(address indexed bunniToken, bytes32 indexed poolId)
// func (_BunniHub *BunniHubFilterer) FilterNewBunni(opts *bind.FilterOpts, bunniToken []common.Address, poolId [][32]byte) (*BunniHubNewBunniIterator, error) {

// 	var bunniTokenRule []interface{}
// 	for _, bunniTokenItem := range bunniToken {
// 		bunniTokenRule = append(bunniTokenRule, bunniTokenItem)
// 	}
// 	var poolIdRule []interface{}
// 	for _, poolIdItem := range poolId {
// 		poolIdRule = append(poolIdRule, poolIdItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.FilterLogs(opts, "NewBunni", bunniTokenRule, poolIdRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &BunniHubNewBunniIterator{contract: _BunniHub.contract, event: "NewBunni", logs: logs, sub: sub}, nil
// }

// // WatchNewBunni is a free log subscription operation binding the contract event 0x3ba5df143a8e4c83b7cb37037a87ae0cfb0f8c3a784d4a10e0b329d5706dce1a.
// //
// // Solidity: event NewBunni(address indexed bunniToken, bytes32 indexed poolId)
// func (_BunniHub *BunniHubFilterer) WatchNewBunni(opts *bind.WatchOpts, sink chan<- *BunniHubNewBunni, bunniToken []common.Address, poolId [][32]byte) (event.Subscription, error) {

// 	var bunniTokenRule []interface{}
// 	for _, bunniTokenItem := range bunniToken {
// 		bunniTokenRule = append(bunniTokenRule, bunniTokenItem)
// 	}
// 	var poolIdRule []interface{}
// 	for _, poolIdItem := range poolId {
// 		poolIdRule = append(poolIdRule, poolIdItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.WatchLogs(opts, "NewBunni", bunniTokenRule, poolIdRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return event.NewSubscription(func(quit <-chan struct{}) error {
// 		defer sub.Unsubscribe()
// 		for {
// 			select {
// 			case log := <-logs:
// 				// New log arrived, parse the event and forward to the user
// 				event := new(BunniHubNewBunni)
// 				if err := _BunniHub.contract.UnpackLog(event, "NewBunni", log); err != nil {
// 					return err
// 				}
// 				event.Raw = log

// 				select {
// 				case sink <- event:
// 				case err := <-sub.Err():
// 					return err
// 				case <-quit:
// 					return nil
// 				}
// 			case err := <-sub.Err():
// 				return err
// 			case <-quit:
// 				return nil
// 			}
// 		}
// 	}), nil
// }

// // ParseNewBunni is a log parse operation binding the contract event 0x3ba5df143a8e4c83b7cb37037a87ae0cfb0f8c3a784d4a10e0b329d5706dce1a.
// //
// // Solidity: event NewBunni(address indexed bunniToken, bytes32 indexed poolId)
// func (_BunniHub *BunniHubFilterer) ParseNewBunni(log types.Log) (*BunniHubNewBunni, error) {
// 	event := new(BunniHubNewBunni)
// 	if err := _BunniHub.contract.UnpackLog(event, "NewBunni", log); err != nil {
// 		return nil, err
// 	}
// 	event.Raw = log
// 	return event, nil
// }

// // BunniHubOwnershipHandoverCanceledIterator is returned from FilterOwnershipHandoverCanceled and is used to iterate over the raw logs and unpacked data for OwnershipHandoverCanceled events raised by the BunniHub contract.
// type BunniHubOwnershipHandoverCanceledIterator struct {
// 	Event *BunniHubOwnershipHandoverCanceled // Event containing the contract specifics and raw log

// 	contract *bind.BoundContract // Generic contract to use for unpacking event data
// 	event    string              // Event name to use for unpacking event data

// 	logs chan types.Log        // Log channel receiving the found contract events
// 	sub  ethereum.Subscription // Subscription for errors, completion and termination
// 	done bool                  // Whether the subscription completed delivering logs
// 	fail error                 // Occurred error to stop iteration
// }

// // Next advances the iterator to the subsequent event, returning whether there
// // are any more events found. In case of a retrieval or parsing error, false is
// // returned and Error() can be queried for the exact failure.
// func (it *BunniHubOwnershipHandoverCanceledIterator) Next() bool {
// 	// If the iterator failed, stop iterating
// 	if it.fail != nil {
// 		return false
// 	}
// 	// If the iterator completed, deliver directly whatever's available
// 	if it.done {
// 		select {
// 		case log := <-it.logs:
// 			it.Event = new(BunniHubOwnershipHandoverCanceled)
// 			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 				it.fail = err
// 				return false
// 			}
// 			it.Event.Raw = log
// 			return true

// 		default:
// 			return false
// 		}
// 	}
// 	// Iterator still in progress, wait for either a data or an error event
// 	select {
// 	case log := <-it.logs:
// 		it.Event = new(BunniHubOwnershipHandoverCanceled)
// 		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 			it.fail = err
// 			return false
// 		}
// 		it.Event.Raw = log
// 		return true

// 	case err := <-it.sub.Err():
// 		it.done = true
// 		it.fail = err
// 		return it.Next()
// 	}
// }

// // Error returns any retrieval or parsing error occurred during filtering.
// func (it *BunniHubOwnershipHandoverCanceledIterator) Error() error {
// 	return it.fail
// }

// // Close terminates the iteration process, releasing any pending underlying
// // resources.
// func (it *BunniHubOwnershipHandoverCanceledIterator) Close() error {
// 	it.sub.Unsubscribe()
// 	return nil
// }

// // BunniHubOwnershipHandoverCanceled represents a OwnershipHandoverCanceled event raised by the BunniHub contract.
// type BunniHubOwnershipHandoverCanceled struct {
// 	PendingOwner common.Address
// 	Raw          types.Log // Blockchain specific contextual infos
// }

// // FilterOwnershipHandoverCanceled is a free log retrieval operation binding the contract event 0xfa7b8eab7da67f412cc9575ed43464468f9bfbae89d1675917346ca6d8fe3c92.
// //
// // Solidity: event OwnershipHandoverCanceled(address indexed pendingOwner)
// func (_BunniHub *BunniHubFilterer) FilterOwnershipHandoverCanceled(opts *bind.FilterOpts, pendingOwner []common.Address) (*BunniHubOwnershipHandoverCanceledIterator, error) {

// 	var pendingOwnerRule []interface{}
// 	for _, pendingOwnerItem := range pendingOwner {
// 		pendingOwnerRule = append(pendingOwnerRule, pendingOwnerItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.FilterLogs(opts, "OwnershipHandoverCanceled", pendingOwnerRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &BunniHubOwnershipHandoverCanceledIterator{contract: _BunniHub.contract, event: "OwnershipHandoverCanceled", logs: logs, sub: sub}, nil
// }

// // WatchOwnershipHandoverCanceled is a free log subscription operation binding the contract event 0xfa7b8eab7da67f412cc9575ed43464468f9bfbae89d1675917346ca6d8fe3c92.
// //
// // Solidity: event OwnershipHandoverCanceled(address indexed pendingOwner)
// func (_BunniHub *BunniHubFilterer) WatchOwnershipHandoverCanceled(opts *bind.WatchOpts, sink chan<- *BunniHubOwnershipHandoverCanceled, pendingOwner []common.Address) (event.Subscription, error) {

// 	var pendingOwnerRule []interface{}
// 	for _, pendingOwnerItem := range pendingOwner {
// 		pendingOwnerRule = append(pendingOwnerRule, pendingOwnerItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.WatchLogs(opts, "OwnershipHandoverCanceled", pendingOwnerRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return event.NewSubscription(func(quit <-chan struct{}) error {
// 		defer sub.Unsubscribe()
// 		for {
// 			select {
// 			case log := <-logs:
// 				// New log arrived, parse the event and forward to the user
// 				event := new(BunniHubOwnershipHandoverCanceled)
// 				if err := _BunniHub.contract.UnpackLog(event, "OwnershipHandoverCanceled", log); err != nil {
// 					return err
// 				}
// 				event.Raw = log

// 				select {
// 				case sink <- event:
// 				case err := <-sub.Err():
// 					return err
// 				case <-quit:
// 					return nil
// 				}
// 			case err := <-sub.Err():
// 				return err
// 			case <-quit:
// 				return nil
// 			}
// 		}
// 	}), nil
// }

// // ParseOwnershipHandoverCanceled is a log parse operation binding the contract event 0xfa7b8eab7da67f412cc9575ed43464468f9bfbae89d1675917346ca6d8fe3c92.
// //
// // Solidity: event OwnershipHandoverCanceled(address indexed pendingOwner)
// func (_BunniHub *BunniHubFilterer) ParseOwnershipHandoverCanceled(log types.Log) (*BunniHubOwnershipHandoverCanceled, error) {
// 	event := new(BunniHubOwnershipHandoverCanceled)
// 	if err := _BunniHub.contract.UnpackLog(event, "OwnershipHandoverCanceled", log); err != nil {
// 		return nil, err
// 	}
// 	event.Raw = log
// 	return event, nil
// }

// // BunniHubOwnershipHandoverRequestedIterator is returned from FilterOwnershipHandoverRequested and is used to iterate over the raw logs and unpacked data for OwnershipHandoverRequested events raised by the BunniHub contract.
// type BunniHubOwnershipHandoverRequestedIterator struct {
// 	Event *BunniHubOwnershipHandoverRequested // Event containing the contract specifics and raw log

// 	contract *bind.BoundContract // Generic contract to use for unpacking event data
// 	event    string              // Event name to use for unpacking event data

// 	logs chan types.Log        // Log channel receiving the found contract events
// 	sub  ethereum.Subscription // Subscription for errors, completion and termination
// 	done bool                  // Whether the subscription completed delivering logs
// 	fail error                 // Occurred error to stop iteration
// }

// // Next advances the iterator to the subsequent event, returning whether there
// // are any more events found. In case of a retrieval or parsing error, false is
// // returned and Error() can be queried for the exact failure.
// func (it *BunniHubOwnershipHandoverRequestedIterator) Next() bool {
// 	// If the iterator failed, stop iterating
// 	if it.fail != nil {
// 		return false
// 	}
// 	// If the iterator completed, deliver directly whatever's available
// 	if it.done {
// 		select {
// 		case log := <-it.logs:
// 			it.Event = new(BunniHubOwnershipHandoverRequested)
// 			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 				it.fail = err
// 				return false
// 			}
// 			it.Event.Raw = log
// 			return true

// 		default:
// 			return false
// 		}
// 	}
// 	// Iterator still in progress, wait for either a data or an error event
// 	select {
// 	case log := <-it.logs:
// 		it.Event = new(BunniHubOwnershipHandoverRequested)
// 		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 			it.fail = err
// 			return false
// 		}
// 		it.Event.Raw = log
// 		return true

// 	case err := <-it.sub.Err():
// 		it.done = true
// 		it.fail = err
// 		return it.Next()
// 	}
// }

// // Error returns any retrieval or parsing error occurred during filtering.
// func (it *BunniHubOwnershipHandoverRequestedIterator) Error() error {
// 	return it.fail
// }

// // Close terminates the iteration process, releasing any pending underlying
// // resources.
// func (it *BunniHubOwnershipHandoverRequestedIterator) Close() error {
// 	it.sub.Unsubscribe()
// 	return nil
// }

// // BunniHubOwnershipHandoverRequested represents a OwnershipHandoverRequested event raised by the BunniHub contract.
// type BunniHubOwnershipHandoverRequested struct {
// 	PendingOwner common.Address
// 	Raw          types.Log // Blockchain specific contextual infos
// }

// // FilterOwnershipHandoverRequested is a free log retrieval operation binding the contract event 0xdbf36a107da19e49527a7176a1babf963b4b0ff8cde35ee35d6cd8f1f9ac7e1d.
// //
// // Solidity: event OwnershipHandoverRequested(address indexed pendingOwner)
// func (_BunniHub *BunniHubFilterer) FilterOwnershipHandoverRequested(opts *bind.FilterOpts, pendingOwner []common.Address) (*BunniHubOwnershipHandoverRequestedIterator, error) {

// 	var pendingOwnerRule []interface{}
// 	for _, pendingOwnerItem := range pendingOwner {
// 		pendingOwnerRule = append(pendingOwnerRule, pendingOwnerItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.FilterLogs(opts, "OwnershipHandoverRequested", pendingOwnerRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &BunniHubOwnershipHandoverRequestedIterator{contract: _BunniHub.contract, event: "OwnershipHandoverRequested", logs: logs, sub: sub}, nil
// }

// // WatchOwnershipHandoverRequested is a free log subscription operation binding the contract event 0xdbf36a107da19e49527a7176a1babf963b4b0ff8cde35ee35d6cd8f1f9ac7e1d.
// //
// // Solidity: event OwnershipHandoverRequested(address indexed pendingOwner)
// func (_BunniHub *BunniHubFilterer) WatchOwnershipHandoverRequested(opts *bind.WatchOpts, sink chan<- *BunniHubOwnershipHandoverRequested, pendingOwner []common.Address) (event.Subscription, error) {

// 	var pendingOwnerRule []interface{}
// 	for _, pendingOwnerItem := range pendingOwner {
// 		pendingOwnerRule = append(pendingOwnerRule, pendingOwnerItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.WatchLogs(opts, "OwnershipHandoverRequested", pendingOwnerRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return event.NewSubscription(func(quit <-chan struct{}) error {
// 		defer sub.Unsubscribe()
// 		for {
// 			select {
// 			case log := <-logs:
// 				// New log arrived, parse the event and forward to the user
// 				event := new(BunniHubOwnershipHandoverRequested)
// 				if err := _BunniHub.contract.UnpackLog(event, "OwnershipHandoverRequested", log); err != nil {
// 					return err
// 				}
// 				event.Raw = log

// 				select {
// 				case sink <- event:
// 				case err := <-sub.Err():
// 					return err
// 				case <-quit:
// 					return nil
// 				}
// 			case err := <-sub.Err():
// 				return err
// 			case <-quit:
// 				return nil
// 			}
// 		}
// 	}), nil
// }

// // ParseOwnershipHandoverRequested is a log parse operation binding the contract event 0xdbf36a107da19e49527a7176a1babf963b4b0ff8cde35ee35d6cd8f1f9ac7e1d.
// //
// // Solidity: event OwnershipHandoverRequested(address indexed pendingOwner)
// func (_BunniHub *BunniHubFilterer) ParseOwnershipHandoverRequested(log types.Log) (*BunniHubOwnershipHandoverRequested, error) {
// 	event := new(BunniHubOwnershipHandoverRequested)
// 	if err := _BunniHub.contract.UnpackLog(event, "OwnershipHandoverRequested", log); err != nil {
// 		return nil, err
// 	}
// 	event.Raw = log
// 	return event, nil
// }

// // BunniHubOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the BunniHub contract.
// type BunniHubOwnershipTransferredIterator struct {
// 	Event *BunniHubOwnershipTransferred // Event containing the contract specifics and raw log

// 	contract *bind.BoundContract // Generic contract to use for unpacking event data
// 	event    string              // Event name to use for unpacking event data

// 	logs chan types.Log        // Log channel receiving the found contract events
// 	sub  ethereum.Subscription // Subscription for errors, completion and termination
// 	done bool                  // Whether the subscription completed delivering logs
// 	fail error                 // Occurred error to stop iteration
// }

// // Next advances the iterator to the subsequent event, returning whether there
// // are any more events found. In case of a retrieval or parsing error, false is
// // returned and Error() can be queried for the exact failure.
// func (it *BunniHubOwnershipTransferredIterator) Next() bool {
// 	// If the iterator failed, stop iterating
// 	if it.fail != nil {
// 		return false
// 	}
// 	// If the iterator completed, deliver directly whatever's available
// 	if it.done {
// 		select {
// 		case log := <-it.logs:
// 			it.Event = new(BunniHubOwnershipTransferred)
// 			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 				it.fail = err
// 				return false
// 			}
// 			it.Event.Raw = log
// 			return true

// 		default:
// 			return false
// 		}
// 	}
// 	// Iterator still in progress, wait for either a data or an error event
// 	select {
// 	case log := <-it.logs:
// 		it.Event = new(BunniHubOwnershipTransferred)
// 		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 			it.fail = err
// 			return false
// 		}
// 		it.Event.Raw = log
// 		return true

// 	case err := <-it.sub.Err():
// 		it.done = true
// 		it.fail = err
// 		return it.Next()
// 	}
// }

// // Error returns any retrieval or parsing error occurred during filtering.
// func (it *BunniHubOwnershipTransferredIterator) Error() error {
// 	return it.fail
// }

// // Close terminates the iteration process, releasing any pending underlying
// // resources.
// func (it *BunniHubOwnershipTransferredIterator) Close() error {
// 	it.sub.Unsubscribe()
// 	return nil
// }

// // BunniHubOwnershipTransferred represents a OwnershipTransferred event raised by the BunniHub contract.
// type BunniHubOwnershipTransferred struct {
// 	OldOwner common.Address
// 	NewOwner common.Address
// 	Raw      types.Log // Blockchain specific contextual infos
// }

// // FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
// //
// // Solidity: event OwnershipTransferred(address indexed oldOwner, address indexed newOwner)
// func (_BunniHub *BunniHubFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, oldOwner []common.Address, newOwner []common.Address) (*BunniHubOwnershipTransferredIterator, error) {

// 	var oldOwnerRule []interface{}
// 	for _, oldOwnerItem := range oldOwner {
// 		oldOwnerRule = append(oldOwnerRule, oldOwnerItem)
// 	}
// 	var newOwnerRule []interface{}
// 	for _, newOwnerItem := range newOwner {
// 		newOwnerRule = append(newOwnerRule, newOwnerItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.FilterLogs(opts, "OwnershipTransferred", oldOwnerRule, newOwnerRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &BunniHubOwnershipTransferredIterator{contract: _BunniHub.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
// }

// // WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
// //
// // Solidity: event OwnershipTransferred(address indexed oldOwner, address indexed newOwner)
// func (_BunniHub *BunniHubFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *BunniHubOwnershipTransferred, oldOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

// 	var oldOwnerRule []interface{}
// 	for _, oldOwnerItem := range oldOwner {
// 		oldOwnerRule = append(oldOwnerRule, oldOwnerItem)
// 	}
// 	var newOwnerRule []interface{}
// 	for _, newOwnerItem := range newOwner {
// 		newOwnerRule = append(newOwnerRule, newOwnerItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.WatchLogs(opts, "OwnershipTransferred", oldOwnerRule, newOwnerRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return event.NewSubscription(func(quit <-chan struct{}) error {
// 		defer sub.Unsubscribe()
// 		for {
// 			select {
// 			case log := <-logs:
// 				// New log arrived, parse the event and forward to the user
// 				event := new(BunniHubOwnershipTransferred)
// 				if err := _BunniHub.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
// 					return err
// 				}
// 				event.Raw = log

// 				select {
// 				case sink <- event:
// 				case err := <-sub.Err():
// 					return err
// 				case <-quit:
// 					return nil
// 				}
// 			case err := <-sub.Err():
// 				return err
// 			case <-quit:
// 				return nil
// 			}
// 		}
// 	}), nil
// }

// // ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
// //
// // Solidity: event OwnershipTransferred(address indexed oldOwner, address indexed newOwner)
// func (_BunniHub *BunniHubFilterer) ParseOwnershipTransferred(log types.Log) (*BunniHubOwnershipTransferred, error) {
// 	event := new(BunniHubOwnershipTransferred)
// 	if err := _BunniHub.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
// 		return nil, err
// 	}
// 	event.Raw = log
// 	return event, nil
// }

// // BunniHubQueueWithdrawIterator is returned from FilterQueueWithdraw and is used to iterate over the raw logs and unpacked data for QueueWithdraw events raised by the BunniHub contract.
// type BunniHubQueueWithdrawIterator struct {
// 	Event *BunniHubQueueWithdraw // Event containing the contract specifics and raw log

// 	contract *bind.BoundContract // Generic contract to use for unpacking event data
// 	event    string              // Event name to use for unpacking event data

// 	logs chan types.Log        // Log channel receiving the found contract events
// 	sub  ethereum.Subscription // Subscription for errors, completion and termination
// 	done bool                  // Whether the subscription completed delivering logs
// 	fail error                 // Occurred error to stop iteration
// }

// // Next advances the iterator to the subsequent event, returning whether there
// // are any more events found. In case of a retrieval or parsing error, false is
// // returned and Error() can be queried for the exact failure.
// func (it *BunniHubQueueWithdrawIterator) Next() bool {
// 	// If the iterator failed, stop iterating
// 	if it.fail != nil {
// 		return false
// 	}
// 	// If the iterator completed, deliver directly whatever's available
// 	if it.done {
// 		select {
// 		case log := <-it.logs:
// 			it.Event = new(BunniHubQueueWithdraw)
// 			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 				it.fail = err
// 				return false
// 			}
// 			it.Event.Raw = log
// 			return true

// 		default:
// 			return false
// 		}
// 	}
// 	// Iterator still in progress, wait for either a data or an error event
// 	select {
// 	case log := <-it.logs:
// 		it.Event = new(BunniHubQueueWithdraw)
// 		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 			it.fail = err
// 			return false
// 		}
// 		it.Event.Raw = log
// 		return true

// 	case err := <-it.sub.Err():
// 		it.done = true
// 		it.fail = err
// 		return it.Next()
// 	}
// }

// // Error returns any retrieval or parsing error occurred during filtering.
// func (it *BunniHubQueueWithdrawIterator) Error() error {
// 	return it.fail
// }

// // Close terminates the iteration process, releasing any pending underlying
// // resources.
// func (it *BunniHubQueueWithdrawIterator) Close() error {
// 	it.sub.Unsubscribe()
// 	return nil
// }

// // BunniHubQueueWithdraw represents a QueueWithdraw event raised by the BunniHub contract.
// type BunniHubQueueWithdraw struct {
// 	Sender common.Address
// 	PoolId [32]byte
// 	Shares *big.Int
// 	Raw    types.Log // Blockchain specific contextual infos
// }

// // FilterQueueWithdraw is a free log retrieval operation binding the contract event 0x0ee885e060478e5bf89befb890ae82fdcc47aa2a9c8e4d668fcce310318d28a1.
// //
// // Solidity: event QueueWithdraw(address indexed sender, bytes32 indexed poolId, uint256 shares)
// func (_BunniHub *BunniHubFilterer) FilterQueueWithdraw(opts *bind.FilterOpts, sender []common.Address, poolId [][32]byte) (*BunniHubQueueWithdrawIterator, error) {

// 	var senderRule []interface{}
// 	for _, senderItem := range sender {
// 		senderRule = append(senderRule, senderItem)
// 	}
// 	var poolIdRule []interface{}
// 	for _, poolIdItem := range poolId {
// 		poolIdRule = append(poolIdRule, poolIdItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.FilterLogs(opts, "QueueWithdraw", senderRule, poolIdRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &BunniHubQueueWithdrawIterator{contract: _BunniHub.contract, event: "QueueWithdraw", logs: logs, sub: sub}, nil
// }

// // WatchQueueWithdraw is a free log subscription operation binding the contract event 0x0ee885e060478e5bf89befb890ae82fdcc47aa2a9c8e4d668fcce310318d28a1.
// //
// // Solidity: event QueueWithdraw(address indexed sender, bytes32 indexed poolId, uint256 shares)
// func (_BunniHub *BunniHubFilterer) WatchQueueWithdraw(opts *bind.WatchOpts, sink chan<- *BunniHubQueueWithdraw, sender []common.Address, poolId [][32]byte) (event.Subscription, error) {

// 	var senderRule []interface{}
// 	for _, senderItem := range sender {
// 		senderRule = append(senderRule, senderItem)
// 	}
// 	var poolIdRule []interface{}
// 	for _, poolIdItem := range poolId {
// 		poolIdRule = append(poolIdRule, poolIdItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.WatchLogs(opts, "QueueWithdraw", senderRule, poolIdRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return event.NewSubscription(func(quit <-chan struct{}) error {
// 		defer sub.Unsubscribe()
// 		for {
// 			select {
// 			case log := <-logs:
// 				// New log arrived, parse the event and forward to the user
// 				event := new(BunniHubQueueWithdraw)
// 				if err := _BunniHub.contract.UnpackLog(event, "QueueWithdraw", log); err != nil {
// 					return err
// 				}
// 				event.Raw = log

// 				select {
// 				case sink <- event:
// 				case err := <-sub.Err():
// 					return err
// 				case <-quit:
// 					return nil
// 				}
// 			case err := <-sub.Err():
// 				return err
// 			case <-quit:
// 				return nil
// 			}
// 		}
// 	}), nil
// }

// // ParseQueueWithdraw is a log parse operation binding the contract event 0x0ee885e060478e5bf89befb890ae82fdcc47aa2a9c8e4d668fcce310318d28a1.
// //
// // Solidity: event QueueWithdraw(address indexed sender, bytes32 indexed poolId, uint256 shares)
// func (_BunniHub *BunniHubFilterer) ParseQueueWithdraw(log types.Log) (*BunniHubQueueWithdraw, error) {
// 	event := new(BunniHubQueueWithdraw)
// 	if err := _BunniHub.contract.UnpackLog(event, "QueueWithdraw", log); err != nil {
// 		return nil, err
// 	}
// 	event.Raw = log
// 	return event, nil
// }

// // BunniHubSetPauseFlagsIterator is returned from FilterSetPauseFlags and is used to iterate over the raw logs and unpacked data for SetPauseFlags events raised by the BunniHub contract.
// type BunniHubSetPauseFlagsIterator struct {
// 	Event *BunniHubSetPauseFlags // Event containing the contract specifics and raw log

// 	contract *bind.BoundContract // Generic contract to use for unpacking event data
// 	event    string              // Event name to use for unpacking event data

// 	logs chan types.Log        // Log channel receiving the found contract events
// 	sub  ethereum.Subscription // Subscription for errors, completion and termination
// 	done bool                  // Whether the subscription completed delivering logs
// 	fail error                 // Occurred error to stop iteration
// }

// // Next advances the iterator to the subsequent event, returning whether there
// // are any more events found. In case of a retrieval or parsing error, false is
// // returned and Error() can be queried for the exact failure.
// func (it *BunniHubSetPauseFlagsIterator) Next() bool {
// 	// If the iterator failed, stop iterating
// 	if it.fail != nil {
// 		return false
// 	}
// 	// If the iterator completed, deliver directly whatever's available
// 	if it.done {
// 		select {
// 		case log := <-it.logs:
// 			it.Event = new(BunniHubSetPauseFlags)
// 			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 				it.fail = err
// 				return false
// 			}
// 			it.Event.Raw = log
// 			return true

// 		default:
// 			return false
// 		}
// 	}
// 	// Iterator still in progress, wait for either a data or an error event
// 	select {
// 	case log := <-it.logs:
// 		it.Event = new(BunniHubSetPauseFlags)
// 		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 			it.fail = err
// 			return false
// 		}
// 		it.Event.Raw = log
// 		return true

// 	case err := <-it.sub.Err():
// 		it.done = true
// 		it.fail = err
// 		return it.Next()
// 	}
// }

// // Error returns any retrieval or parsing error occurred during filtering.
// func (it *BunniHubSetPauseFlagsIterator) Error() error {
// 	return it.fail
// }

// // Close terminates the iteration process, releasing any pending underlying
// // resources.
// func (it *BunniHubSetPauseFlagsIterator) Close() error {
// 	it.sub.Unsubscribe()
// 	return nil
// }

// // BunniHubSetPauseFlags represents a SetPauseFlags event raised by the BunniHub contract.
// type BunniHubSetPauseFlags struct {
// 	PauseFlags uint8
// 	Raw        types.Log // Blockchain specific contextual infos
// }

// // FilterSetPauseFlags is a free log retrieval operation binding the contract event 0x3021cc5514f1ea312648df4d3e6c9cf9c5bd12c429f0849d4c903af7010c6afa.
// //
// // Solidity: event SetPauseFlags(uint8 indexed pauseFlags)
// func (_BunniHub *BunniHubFilterer) FilterSetPauseFlags(opts *bind.FilterOpts, pauseFlags []uint8) (*BunniHubSetPauseFlagsIterator, error) {

// 	var pauseFlagsRule []interface{}
// 	for _, pauseFlagsItem := range pauseFlags {
// 		pauseFlagsRule = append(pauseFlagsRule, pauseFlagsItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.FilterLogs(opts, "SetPauseFlags", pauseFlagsRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &BunniHubSetPauseFlagsIterator{contract: _BunniHub.contract, event: "SetPauseFlags", logs: logs, sub: sub}, nil
// }

// // WatchSetPauseFlags is a free log subscription operation binding the contract event 0x3021cc5514f1ea312648df4d3e6c9cf9c5bd12c429f0849d4c903af7010c6afa.
// //
// // Solidity: event SetPauseFlags(uint8 indexed pauseFlags)
// func (_BunniHub *BunniHubFilterer) WatchSetPauseFlags(opts *bind.WatchOpts, sink chan<- *BunniHubSetPauseFlags, pauseFlags []uint8) (event.Subscription, error) {

// 	var pauseFlagsRule []interface{}
// 	for _, pauseFlagsItem := range pauseFlags {
// 		pauseFlagsRule = append(pauseFlagsRule, pauseFlagsItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.WatchLogs(opts, "SetPauseFlags", pauseFlagsRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return event.NewSubscription(func(quit <-chan struct{}) error {
// 		defer sub.Unsubscribe()
// 		for {
// 			select {
// 			case log := <-logs:
// 				// New log arrived, parse the event and forward to the user
// 				event := new(BunniHubSetPauseFlags)
// 				if err := _BunniHub.contract.UnpackLog(event, "SetPauseFlags", log); err != nil {
// 					return err
// 				}
// 				event.Raw = log

// 				select {
// 				case sink <- event:
// 				case err := <-sub.Err():
// 					return err
// 				case <-quit:
// 					return nil
// 				}
// 			case err := <-sub.Err():
// 				return err
// 			case <-quit:
// 				return nil
// 			}
// 		}
// 	}), nil
// }

// // ParseSetPauseFlags is a log parse operation binding the contract event 0x3021cc5514f1ea312648df4d3e6c9cf9c5bd12c429f0849d4c903af7010c6afa.
// //
// // Solidity: event SetPauseFlags(uint8 indexed pauseFlags)
// func (_BunniHub *BunniHubFilterer) ParseSetPauseFlags(log types.Log) (*BunniHubSetPauseFlags, error) {
// 	event := new(BunniHubSetPauseFlags)
// 	if err := _BunniHub.contract.UnpackLog(event, "SetPauseFlags", log); err != nil {
// 		return nil, err
// 	}
// 	event.Raw = log
// 	return event, nil
// }

// // BunniHubSetPauserIterator is returned from FilterSetPauser and is used to iterate over the raw logs and unpacked data for SetPauser events raised by the BunniHub contract.
// type BunniHubSetPauserIterator struct {
// 	Event *BunniHubSetPauser // Event containing the contract specifics and raw log

// 	contract *bind.BoundContract // Generic contract to use for unpacking event data
// 	event    string              // Event name to use for unpacking event data

// 	logs chan types.Log        // Log channel receiving the found contract events
// 	sub  ethereum.Subscription // Subscription for errors, completion and termination
// 	done bool                  // Whether the subscription completed delivering logs
// 	fail error                 // Occurred error to stop iteration
// }

// // Next advances the iterator to the subsequent event, returning whether there
// // are any more events found. In case of a retrieval or parsing error, false is
// // returned and Error() can be queried for the exact failure.
// func (it *BunniHubSetPauserIterator) Next() bool {
// 	// If the iterator failed, stop iterating
// 	if it.fail != nil {
// 		return false
// 	}
// 	// If the iterator completed, deliver directly whatever's available
// 	if it.done {
// 		select {
// 		case log := <-it.logs:
// 			it.Event = new(BunniHubSetPauser)
// 			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 				it.fail = err
// 				return false
// 			}
// 			it.Event.Raw = log
// 			return true

// 		default:
// 			return false
// 		}
// 	}
// 	// Iterator still in progress, wait for either a data or an error event
// 	select {
// 	case log := <-it.logs:
// 		it.Event = new(BunniHubSetPauser)
// 		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 			it.fail = err
// 			return false
// 		}
// 		it.Event.Raw = log
// 		return true

// 	case err := <-it.sub.Err():
// 		it.done = true
// 		it.fail = err
// 		return it.Next()
// 	}
// }

// // Error returns any retrieval or parsing error occurred during filtering.
// func (it *BunniHubSetPauserIterator) Error() error {
// 	return it.fail
// }

// // Close terminates the iteration process, releasing any pending underlying
// // resources.
// func (it *BunniHubSetPauserIterator) Close() error {
// 	it.sub.Unsubscribe()
// 	return nil
// }

// // BunniHubSetPauser represents a SetPauser event raised by the BunniHub contract.
// type BunniHubSetPauser struct {
// 	Guy      common.Address
// 	IsPauser bool
// 	Raw      types.Log // Blockchain specific contextual infos
// }

// // FilterSetPauser is a free log retrieval operation binding the contract event 0xd34f4aa5f94a385f2fa0ca25e5f01c6f331018f35c3d43a7b8057a86704de3df.
// //
// // Solidity: event SetPauser(address indexed guy, bool indexed isPauser)
// func (_BunniHub *BunniHubFilterer) FilterSetPauser(opts *bind.FilterOpts, guy []common.Address, isPauser []bool) (*BunniHubSetPauserIterator, error) {

// 	var guyRule []interface{}
// 	for _, guyItem := range guy {
// 		guyRule = append(guyRule, guyItem)
// 	}
// 	var isPauserRule []interface{}
// 	for _, isPauserItem := range isPauser {
// 		isPauserRule = append(isPauserRule, isPauserItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.FilterLogs(opts, "SetPauser", guyRule, isPauserRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &BunniHubSetPauserIterator{contract: _BunniHub.contract, event: "SetPauser", logs: logs, sub: sub}, nil
// }

// // WatchSetPauser is a free log subscription operation binding the contract event 0xd34f4aa5f94a385f2fa0ca25e5f01c6f331018f35c3d43a7b8057a86704de3df.
// //
// // Solidity: event SetPauser(address indexed guy, bool indexed isPauser)
// func (_BunniHub *BunniHubFilterer) WatchSetPauser(opts *bind.WatchOpts, sink chan<- *BunniHubSetPauser, guy []common.Address, isPauser []bool) (event.Subscription, error) {

// 	var guyRule []interface{}
// 	for _, guyItem := range guy {
// 		guyRule = append(guyRule, guyItem)
// 	}
// 	var isPauserRule []interface{}
// 	for _, isPauserItem := range isPauser {
// 		isPauserRule = append(isPauserRule, isPauserItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.WatchLogs(opts, "SetPauser", guyRule, isPauserRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return event.NewSubscription(func(quit <-chan struct{}) error {
// 		defer sub.Unsubscribe()
// 		for {
// 			select {
// 			case log := <-logs:
// 				// New log arrived, parse the event and forward to the user
// 				event := new(BunniHubSetPauser)
// 				if err := _BunniHub.contract.UnpackLog(event, "SetPauser", log); err != nil {
// 					return err
// 				}
// 				event.Raw = log

// 				select {
// 				case sink <- event:
// 				case err := <-sub.Err():
// 					return err
// 				case <-quit:
// 					return nil
// 				}
// 			case err := <-sub.Err():
// 				return err
// 			case <-quit:
// 				return nil
// 			}
// 		}
// 	}), nil
// }

// // ParseSetPauser is a log parse operation binding the contract event 0xd34f4aa5f94a385f2fa0ca25e5f01c6f331018f35c3d43a7b8057a86704de3df.
// //
// // Solidity: event SetPauser(address indexed guy, bool indexed isPauser)
// func (_BunniHub *BunniHubFilterer) ParseSetPauser(log types.Log) (*BunniHubSetPauser, error) {
// 	event := new(BunniHubSetPauser)
// 	if err := _BunniHub.contract.UnpackLog(event, "SetPauser", log); err != nil {
// 		return nil, err
// 	}
// 	event.Raw = log
// 	return event, nil
// }

// // BunniHubSetReferralRewardRecipientIterator is returned from FilterSetReferralRewardRecipient and is used to iterate over the raw logs and unpacked data for SetReferralRewardRecipient events raised by the BunniHub contract.
// type BunniHubSetReferralRewardRecipientIterator struct {
// 	Event *BunniHubSetReferralRewardRecipient // Event containing the contract specifics and raw log

// 	contract *bind.BoundContract // Generic contract to use for unpacking event data
// 	event    string              // Event name to use for unpacking event data

// 	logs chan types.Log        // Log channel receiving the found contract events
// 	sub  ethereum.Subscription // Subscription for errors, completion and termination
// 	done bool                  // Whether the subscription completed delivering logs
// 	fail error                 // Occurred error to stop iteration
// }

// // Next advances the iterator to the subsequent event, returning whether there
// // are any more events found. In case of a retrieval or parsing error, false is
// // returned and Error() can be queried for the exact failure.
// func (it *BunniHubSetReferralRewardRecipientIterator) Next() bool {
// 	// If the iterator failed, stop iterating
// 	if it.fail != nil {
// 		return false
// 	}
// 	// If the iterator completed, deliver directly whatever's available
// 	if it.done {
// 		select {
// 		case log := <-it.logs:
// 			it.Event = new(BunniHubSetReferralRewardRecipient)
// 			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 				it.fail = err
// 				return false
// 			}
// 			it.Event.Raw = log
// 			return true

// 		default:
// 			return false
// 		}
// 	}
// 	// Iterator still in progress, wait for either a data or an error event
// 	select {
// 	case log := <-it.logs:
// 		it.Event = new(BunniHubSetReferralRewardRecipient)
// 		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 			it.fail = err
// 			return false
// 		}
// 		it.Event.Raw = log
// 		return true

// 	case err := <-it.sub.Err():
// 		it.done = true
// 		it.fail = err
// 		return it.Next()
// 	}
// }

// // Error returns any retrieval or parsing error occurred during filtering.
// func (it *BunniHubSetReferralRewardRecipientIterator) Error() error {
// 	return it.fail
// }

// // Close terminates the iteration process, releasing any pending underlying
// // resources.
// func (it *BunniHubSetReferralRewardRecipientIterator) Close() error {
// 	it.sub.Unsubscribe()
// 	return nil
// }

// // BunniHubSetReferralRewardRecipient represents a SetReferralRewardRecipient event raised by the BunniHub contract.
// type BunniHubSetReferralRewardRecipient struct {
// 	NewReferralRewardRecipient common.Address
// 	Raw                        types.Log // Blockchain specific contextual infos
// }

// // FilterSetReferralRewardRecipient is a free log retrieval operation binding the contract event 0x905a74aecdb46e2b4d535f80ff231418f1c5684536fe968901427f2c826bd030.
// //
// // Solidity: event SetReferralRewardRecipient(address indexed newReferralRewardRecipient)
// func (_BunniHub *BunniHubFilterer) FilterSetReferralRewardRecipient(opts *bind.FilterOpts, newReferralRewardRecipient []common.Address) (*BunniHubSetReferralRewardRecipientIterator, error) {

// 	var newReferralRewardRecipientRule []interface{}
// 	for _, newReferralRewardRecipientItem := range newReferralRewardRecipient {
// 		newReferralRewardRecipientRule = append(newReferralRewardRecipientRule, newReferralRewardRecipientItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.FilterLogs(opts, "SetReferralRewardRecipient", newReferralRewardRecipientRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &BunniHubSetReferralRewardRecipientIterator{contract: _BunniHub.contract, event: "SetReferralRewardRecipient", logs: logs, sub: sub}, nil
// }

// // WatchSetReferralRewardRecipient is a free log subscription operation binding the contract event 0x905a74aecdb46e2b4d535f80ff231418f1c5684536fe968901427f2c826bd030.
// //
// // Solidity: event SetReferralRewardRecipient(address indexed newReferralRewardRecipient)
// func (_BunniHub *BunniHubFilterer) WatchSetReferralRewardRecipient(opts *bind.WatchOpts, sink chan<- *BunniHubSetReferralRewardRecipient, newReferralRewardRecipient []common.Address) (event.Subscription, error) {

// 	var newReferralRewardRecipientRule []interface{}
// 	for _, newReferralRewardRecipientItem := range newReferralRewardRecipient {
// 		newReferralRewardRecipientRule = append(newReferralRewardRecipientRule, newReferralRewardRecipientItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.WatchLogs(opts, "SetReferralRewardRecipient", newReferralRewardRecipientRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return event.NewSubscription(func(quit <-chan struct{}) error {
// 		defer sub.Unsubscribe()
// 		for {
// 			select {
// 			case log := <-logs:
// 				// New log arrived, parse the event and forward to the user
// 				event := new(BunniHubSetReferralRewardRecipient)
// 				if err := _BunniHub.contract.UnpackLog(event, "SetReferralRewardRecipient", log); err != nil {
// 					return err
// 				}
// 				event.Raw = log

// 				select {
// 				case sink <- event:
// 				case err := <-sub.Err():
// 					return err
// 				case <-quit:
// 					return nil
// 				}
// 			case err := <-sub.Err():
// 				return err
// 			case <-quit:
// 				return nil
// 			}
// 		}
// 	}), nil
// }

// // ParseSetReferralRewardRecipient is a log parse operation binding the contract event 0x905a74aecdb46e2b4d535f80ff231418f1c5684536fe968901427f2c826bd030.
// //
// // Solidity: event SetReferralRewardRecipient(address indexed newReferralRewardRecipient)
// func (_BunniHub *BunniHubFilterer) ParseSetReferralRewardRecipient(log types.Log) (*BunniHubSetReferralRewardRecipient, error) {
// 	event := new(BunniHubSetReferralRewardRecipient)
// 	if err := _BunniHub.contract.UnpackLog(event, "SetReferralRewardRecipient", log); err != nil {
// 		return nil, err
// 	}
// 	event.Raw = log
// 	return event, nil
// }

// // BunniHubWithdrawIterator is returned from FilterWithdraw and is used to iterate over the raw logs and unpacked data for Withdraw events raised by the BunniHub contract.
// type BunniHubWithdrawIterator struct {
// 	Event *BunniHubWithdraw // Event containing the contract specifics and raw log

// 	contract *bind.BoundContract // Generic contract to use for unpacking event data
// 	event    string              // Event name to use for unpacking event data

// 	logs chan types.Log        // Log channel receiving the found contract events
// 	sub  ethereum.Subscription // Subscription for errors, completion and termination
// 	done bool                  // Whether the subscription completed delivering logs
// 	fail error                 // Occurred error to stop iteration
// }

// // Next advances the iterator to the subsequent event, returning whether there
// // are any more events found. In case of a retrieval or parsing error, false is
// // returned and Error() can be queried for the exact failure.
// func (it *BunniHubWithdrawIterator) Next() bool {
// 	// If the iterator failed, stop iterating
// 	if it.fail != nil {
// 		return false
// 	}
// 	// If the iterator completed, deliver directly whatever's available
// 	if it.done {
// 		select {
// 		case log := <-it.logs:
// 			it.Event = new(BunniHubWithdraw)
// 			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 				it.fail = err
// 				return false
// 			}
// 			it.Event.Raw = log
// 			return true

// 		default:
// 			return false
// 		}
// 	}
// 	// Iterator still in progress, wait for either a data or an error event
// 	select {
// 	case log := <-it.logs:
// 		it.Event = new(BunniHubWithdraw)
// 		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
// 			it.fail = err
// 			return false
// 		}
// 		it.Event.Raw = log
// 		return true

// 	case err := <-it.sub.Err():
// 		it.done = true
// 		it.fail = err
// 		return it.Next()
// 	}
// }

// // Error returns any retrieval or parsing error occurred during filtering.
// func (it *BunniHubWithdrawIterator) Error() error {
// 	return it.fail
// }

// // Close terminates the iteration process, releasing any pending underlying
// // resources.
// func (it *BunniHubWithdrawIterator) Close() error {
// 	it.sub.Unsubscribe()
// 	return nil
// }

// // BunniHubWithdraw represents a Withdraw event raised by the BunniHub contract.
// type BunniHubWithdraw struct {
// 	Sender    common.Address
// 	Recipient common.Address
// 	PoolId    [32]byte
// 	Amount0   *big.Int
// 	Amount1   *big.Int
// 	Shares    *big.Int
// 	Raw       types.Log // Blockchain specific contextual infos
// }

// // FilterWithdraw is a free log retrieval operation binding the contract event 0xbc70c2ef3795ca1df41695488a7ff6060de75f86dd892696bbfd76bdd123270f.
// //
// // Solidity: event Withdraw(address indexed sender, address indexed recipient, bytes32 indexed poolId, uint256 amount0, uint256 amount1, uint256 shares)
// func (_BunniHub *BunniHubFilterer) FilterWithdraw(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address, poolId [][32]byte) (*BunniHubWithdrawIterator, error) {

// 	var senderRule []interface{}
// 	for _, senderItem := range sender {
// 		senderRule = append(senderRule, senderItem)
// 	}
// 	var recipientRule []interface{}
// 	for _, recipientItem := range recipient {
// 		recipientRule = append(recipientRule, recipientItem)
// 	}
// 	var poolIdRule []interface{}
// 	for _, poolIdItem := range poolId {
// 		poolIdRule = append(poolIdRule, poolIdItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.FilterLogs(opts, "Withdraw", senderRule, recipientRule, poolIdRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &BunniHubWithdrawIterator{contract: _BunniHub.contract, event: "Withdraw", logs: logs, sub: sub}, nil
// }

// // WatchWithdraw is a free log subscription operation binding the contract event 0xbc70c2ef3795ca1df41695488a7ff6060de75f86dd892696bbfd76bdd123270f.
// //
// // Solidity: event Withdraw(address indexed sender, address indexed recipient, bytes32 indexed poolId, uint256 amount0, uint256 amount1, uint256 shares)
// func (_BunniHub *BunniHubFilterer) WatchWithdraw(opts *bind.WatchOpts, sink chan<- *BunniHubWithdraw, sender []common.Address, recipient []common.Address, poolId [][32]byte) (event.Subscription, error) {

// 	var senderRule []interface{}
// 	for _, senderItem := range sender {
// 		senderRule = append(senderRule, senderItem)
// 	}
// 	var recipientRule []interface{}
// 	for _, recipientItem := range recipient {
// 		recipientRule = append(recipientRule, recipientItem)
// 	}
// 	var poolIdRule []interface{}
// 	for _, poolIdItem := range poolId {
// 		poolIdRule = append(poolIdRule, poolIdItem)
// 	}

// 	logs, sub, err := _BunniHub.contract.WatchLogs(opts, "Withdraw", senderRule, recipientRule, poolIdRule)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return event.NewSubscription(func(quit <-chan struct{}) error {
// 		defer sub.Unsubscribe()
// 		for {
// 			select {
// 			case log := <-logs:
// 				// New log arrived, parse the event and forward to the user
// 				event := new(BunniHubWithdraw)
// 				if err := _BunniHub.contract.UnpackLog(event, "Withdraw", log); err != nil {
// 					return err
// 				}
// 				event.Raw = log

// 				select {
// 				case sink <- event:
// 				case err := <-sub.Err():
// 					return err
// 				case <-quit:
// 					return nil
// 				}
// 			case err := <-sub.Err():
// 				return err
// 			case <-quit:
// 				return nil
// 			}
// 		}
// 	}), nil
// }

// // ParseWithdraw is a log parse operation binding the contract event 0xbc70c2ef3795ca1df41695488a7ff6060de75f86dd892696bbfd76bdd123270f.
// //
// // Solidity: event Withdraw(address indexed sender, address indexed recipient, bytes32 indexed poolId, uint256 amount0, uint256 amount1, uint256 shares)
// func (_BunniHub *BunniHubFilterer) ParseWithdraw(log types.Log) (*BunniHubWithdraw, error) {
// 	event := new(BunniHubWithdraw)
// 	if err := _BunniHub.contract.UnpackLog(event, "Withdraw", log); err != nil {
// 		return nil, err
// 	}
// 	event.Raw = log
// 	return event, nil
// }
