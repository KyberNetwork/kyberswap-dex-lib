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

// FeeHelperFeeParameters is an auto generated low-level Go binding around an user-defined struct.
type FeeHelperFeeParameters struct {
	BinStep                  uint16
	BaseFactor               uint16
	FilterPeriod             uint16
	DecayPeriod              uint16
	ReductionFactor          uint16
	VariableFeeControl       *big.Int
	ProtocolShare            uint16
	MaxVolatilityAccumulated *big.Int
	VolatilityAccumulated    *big.Int
	VolatilityReference      *big.Int
	IndexRef                 *big.Int
	Time                     *big.Int
}

// LBPairMetaData contains all meta data concerning the LBPair contract.
var LBPairMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractILBFactory\",\"name\":\"_factory\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"bp\",\"type\":\"uint256\"}],\"name\":\"BinHelper__BinStepOverflows\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BinHelper__IdOverflows\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBPair__AddressZero\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBPair__AddressZeroOrThis\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBPair__AlreadyInitialized\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"LBPair__CompositionFactorFlawed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBPair__DistributionsOverflow\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBPair__FlashLoanCallbackFailed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBPair__FlashLoanInvalidBalance\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBPair__FlashLoanInvalidToken\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBPair__InsufficientAmounts\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"LBPair__InsufficientLiquidityBurned\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"LBPair__InsufficientLiquidityMinted\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBPair__OnlyFactory\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"feeRecipient\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"LBPair__OnlyFeeRecipient\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBPair__OnlyStrictlyIncreasingId\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newSize\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"oracleSize\",\"type\":\"uint256\"}],\"name\":\"LBPair__OracleNewSizeTooSmall\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBPair__WrongLengths\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"LBToken__BurnExceedsBalance\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBToken__BurnFromAddress0\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"accountsLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"idsLength\",\"type\":\"uint256\"}],\"name\":\"LBToken__LengthMismatch\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBToken__MintToAddress0\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"LBToken__SelfApproval\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"LBToken__SpenderNotApproved\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"LBToken__TransferExceedsBalance\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBToken__TransferFromOrToAddress0\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LBToken__TransferToSelf\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"},{\"internalType\":\"int256\",\"name\":\"y\",\"type\":\"int256\"}],\"name\":\"Math128x128__PowerUnderflow\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"prod1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"denominator\",\"type\":\"uint256\"}],\"name\":\"Math512Bits__MulDivOverflow\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"prod1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"offset\",\"type\":\"uint256\"}],\"name\":\"Math512Bits__MulShiftOverflow\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"offset\",\"type\":\"uint256\"}],\"name\":\"Math512Bits__OffsetOverflows\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_minTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_lookUpTimestamp\",\"type\":\"uint256\"}],\"name\":\"Oracle__LookUpTimestampTooOld\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Oracle__NotInitialized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ReentrancyGuardUpgradeable__AlreadyInitialized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ReentrancyGuardUpgradeable__ReentrantCall\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"}],\"name\":\"SafeCast__Exceeds112Bits\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"}],\"name\":\"SafeCast__Exceeds128Bits\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"}],\"name\":\"SafeCast__Exceeds24Bits\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"}],\"name\":\"SafeCast__Exceeds40Bits\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TokenHelper__CallFailed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TokenHelper__NonContract\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TokenHelper__TransferFailed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TreeMath__ErrorDepthSearch\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"ApprovalForAll\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"feesX\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"feesY\",\"type\":\"uint256\"}],\"name\":\"CompositionFee\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountX\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountY\",\"type\":\"uint256\"}],\"name\":\"DepositedToBin\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountX\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountY\",\"type\":\"uint256\"}],\"name\":\"FeesCollected\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"contractILBFlashLoanCallback\",\"name\":\"receiver\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"contractIERC20\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"FlashLoan\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"previousSize\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newSize\",\"type\":\"uint256\"}],\"name\":\"OracleSizeIncreased\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountX\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountY\",\"type\":\"uint256\"}],\"name\":\"ProtocolFeesCollected\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"swapForY\",\"type\":\"bool\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountIn\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountOut\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"volatilityAccumulated\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"fees\",\"type\":\"uint256\"}],\"name\":\"Swap\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"indexed\":false,\"internalType\":\"uint256[]\",\"name\":\"amounts\",\"type\":\"uint256[]\"}],\"name\":\"TransferBatch\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"TransferSingle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountX\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountY\",\"type\":\"uint256\"}],\"name\":\"WithdrawnFromBin\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_id\",\"type\":\"uint256\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"_accounts\",\"type\":\"address[]\"},{\"internalType\":\"uint256[]\",\"name\":\"_ids\",\"type\":\"uint256[]\"}],\"name\":\"balanceOfBatch\",\"outputs\":[{\"internalType\":\"uint256[]\",\"name\":\"batchBalances\",\"type\":\"uint256[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"_ids\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"_amounts\",\"type\":\"uint256[]\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"}],\"name\":\"burn\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountX\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountY\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_account\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"_ids\",\"type\":\"uint256[]\"}],\"name\":\"collectFees\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountX\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountY\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"collectProtocolFees\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"amountX\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"amountY\",\"type\":\"uint128\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"factory\",\"outputs\":[{\"internalType\":\"contractILBFactory\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"feeParameters\",\"outputs\":[{\"components\":[{\"internalType\":\"uint16\",\"name\":\"binStep\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"baseFactor\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"filterPeriod\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"decayPeriod\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"reductionFactor\",\"type\":\"uint16\"},{\"internalType\":\"uint24\",\"name\":\"variableFeeControl\",\"type\":\"uint24\"},{\"internalType\":\"uint16\",\"name\":\"protocolShare\",\"type\":\"uint16\"},{\"internalType\":\"uint24\",\"name\":\"maxVolatilityAccumulated\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"volatilityAccumulated\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"volatilityReference\",\"type\":\"uint24\"},{\"internalType\":\"uint24\",\"name\":\"indexRef\",\"type\":\"uint24\"},{\"internalType\":\"uint40\",\"name\":\"time\",\"type\":\"uint40\"}],\"internalType\":\"structFeeHelper.FeeParameters\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint24\",\"name\":\"_id\",\"type\":\"uint24\"},{\"internalType\":\"bool\",\"name\":\"_swapForY\",\"type\":\"bool\"}],\"name\":\"findFirstNonEmptyBinId\",\"outputs\":[{\"internalType\":\"uint24\",\"name\":\"\",\"type\":\"uint24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractILBFlashLoanCallback\",\"name\":\"_receiver\",\"type\":\"address\"},{\"internalType\":\"contractIERC20\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"flashLoan\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"forceDecay\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint24\",\"name\":\"_id\",\"type\":\"uint24\"}],\"name\":\"getBin\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"reserveX\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reserveY\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGlobalFees\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"feesXTotal\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"feesYTotal\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"feesXProtocol\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"feesYProtocol\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getOracleParameters\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"oracleSampleLifetime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"oracleSize\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"oracleActiveSize\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"oracleLastTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"oracleId\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"min\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"max\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_timeDelta\",\"type\":\"uint256\"}],\"name\":\"getOracleSampleFrom\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"cumulativeId\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"cumulativeVolatilityAccumulated\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"cumulativeBinCrossed\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getReservesAndId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"reserveX\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reserveY\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"activeId\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"_newLength\",\"type\":\"uint16\"}],\"name\":\"increaseOracleLength\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"_tokenX\",\"type\":\"address\"},{\"internalType\":\"contractIERC20\",\"name\":\"_tokenY\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"_activeId\",\"type\":\"uint24\"},{\"internalType\":\"uint16\",\"name\":\"_sampleLifetime\",\"type\":\"uint16\"},{\"internalType\":\"bytes32\",\"name\":\"_packedFeeParameters\",\"type\":\"bytes32\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_spender\",\"type\":\"address\"}],\"name\":\"isApprovedForAll\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"_ids\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"_distributionX\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"_distributionY\",\"type\":\"uint256[]\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"}],\"name\":\"mint\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256[]\",\"name\":\"liquidityMinted\",\"type\":\"uint256[]\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_account\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"_ids\",\"type\":\"uint256[]\"}],\"name\":\"pendingFees\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountX\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountY\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"_ids\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"_amounts\",\"type\":\"uint256[]\"}],\"name\":\"safeBatchTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"safeTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_spender\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"_approved\",\"type\":\"bool\"}],\"name\":\"setApprovalForAll\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_packedFeeParameters\",\"type\":\"bytes32\"}],\"name\":\"setFeesParameters\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"_interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"_swapForY\",\"type\":\"bool\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"}],\"name\":\"swap\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountXOut\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountYOut\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"tokenX\",\"outputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"tokenY\",\"outputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_id\",\"type\":\"uint256\"}],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
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
func (_LBPair *LBPairRaw) Call(opts *bind.CallOpts, result *[]any, method string, params ...any) error {
	return _LBPair.Contract.LBPairCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LBPair *LBPairRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LBPair.Contract.LBPairTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LBPair *LBPairRaw) Transact(opts *bind.TransactOpts, method string, params ...any) (*types.Transaction, error) {
	return _LBPair.Contract.LBPairTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LBPair *LBPairCallerRaw) Call(opts *bind.CallOpts, result *[]any, method string, params ...any) error {
	return _LBPair.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LBPair *LBPairTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LBPair.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LBPair *LBPairTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...any) (*types.Transaction, error) {
	return _LBPair.Contract.contract.Transact(opts, method, params...)
}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address _account, uint256 _id) view returns(uint256)
func (_LBPair *LBPairCaller) BalanceOf(opts *bind.CallOpts, _account common.Address, _id *big.Int) (*big.Int, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "balanceOf", _account, _id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address _account, uint256 _id) view returns(uint256)
func (_LBPair *LBPairSession) BalanceOf(_account common.Address, _id *big.Int) (*big.Int, error) {
	return _LBPair.Contract.BalanceOf(&_LBPair.CallOpts, _account, _id)
}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address _account, uint256 _id) view returns(uint256)
func (_LBPair *LBPairCallerSession) BalanceOf(_account common.Address, _id *big.Int) (*big.Int, error) {
	return _LBPair.Contract.BalanceOf(&_LBPair.CallOpts, _account, _id)
}

// BalanceOfBatch is a free data retrieval call binding the contract method 0x4e1273f4.
//
// Solidity: function balanceOfBatch(address[] _accounts, uint256[] _ids) view returns(uint256[] batchBalances)
func (_LBPair *LBPairCaller) BalanceOfBatch(opts *bind.CallOpts, _accounts []common.Address, _ids []*big.Int) ([]*big.Int, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "balanceOfBatch", _accounts, _ids)

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// BalanceOfBatch is a free data retrieval call binding the contract method 0x4e1273f4.
//
// Solidity: function balanceOfBatch(address[] _accounts, uint256[] _ids) view returns(uint256[] batchBalances)
func (_LBPair *LBPairSession) BalanceOfBatch(_accounts []common.Address, _ids []*big.Int) ([]*big.Int, error) {
	return _LBPair.Contract.BalanceOfBatch(&_LBPair.CallOpts, _accounts, _ids)
}

// BalanceOfBatch is a free data retrieval call binding the contract method 0x4e1273f4.
//
// Solidity: function balanceOfBatch(address[] _accounts, uint256[] _ids) view returns(uint256[] batchBalances)
func (_LBPair *LBPairCallerSession) BalanceOfBatch(_accounts []common.Address, _ids []*big.Int) ([]*big.Int, error) {
	return _LBPair.Contract.BalanceOfBatch(&_LBPair.CallOpts, _accounts, _ids)
}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_LBPair *LBPairCaller) Factory(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "factory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_LBPair *LBPairSession) Factory() (common.Address, error) {
	return _LBPair.Contract.Factory(&_LBPair.CallOpts)
}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_LBPair *LBPairCallerSession) Factory() (common.Address, error) {
	return _LBPair.Contract.Factory(&_LBPair.CallOpts)
}

// FeeParameters is a free data retrieval call binding the contract method 0x98c7adf3.
//
// Solidity: function feeParameters() view returns((uint16,uint16,uint16,uint16,uint16,uint24,uint16,uint24,uint24,uint24,uint24,uint40))
func (_LBPair *LBPairCaller) FeeParameters(opts *bind.CallOpts) (FeeHelperFeeParameters, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "feeParameters")

	if err != nil {
		return *new(FeeHelperFeeParameters), err
	}

	out0 := *abi.ConvertType(out[0], new(FeeHelperFeeParameters)).(*FeeHelperFeeParameters)

	return out0, err

}

// FeeParameters is a free data retrieval call binding the contract method 0x98c7adf3.
//
// Solidity: function feeParameters() view returns((uint16,uint16,uint16,uint16,uint16,uint24,uint16,uint24,uint24,uint24,uint24,uint40))
func (_LBPair *LBPairSession) FeeParameters() (FeeHelperFeeParameters, error) {
	return _LBPair.Contract.FeeParameters(&_LBPair.CallOpts)
}

// FeeParameters is a free data retrieval call binding the contract method 0x98c7adf3.
//
// Solidity: function feeParameters() view returns((uint16,uint16,uint16,uint16,uint16,uint24,uint16,uint24,uint24,uint24,uint24,uint40))
func (_LBPair *LBPairCallerSession) FeeParameters() (FeeHelperFeeParameters, error) {
	return _LBPair.Contract.FeeParameters(&_LBPair.CallOpts)
}

// FindFirstNonEmptyBinId is a free data retrieval call binding the contract method 0x8f919a83.
//
// Solidity: function findFirstNonEmptyBinId(uint24 _id, bool _swapForY) view returns(uint24)
func (_LBPair *LBPairCaller) FindFirstNonEmptyBinId(opts *bind.CallOpts, _id *big.Int, _swapForY bool) (*big.Int, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "findFirstNonEmptyBinId", _id, _swapForY)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FindFirstNonEmptyBinId is a free data retrieval call binding the contract method 0x8f919a83.
//
// Solidity: function findFirstNonEmptyBinId(uint24 _id, bool _swapForY) view returns(uint24)
func (_LBPair *LBPairSession) FindFirstNonEmptyBinId(_id *big.Int, _swapForY bool) (*big.Int, error) {
	return _LBPair.Contract.FindFirstNonEmptyBinId(&_LBPair.CallOpts, _id, _swapForY)
}

// FindFirstNonEmptyBinId is a free data retrieval call binding the contract method 0x8f919a83.
//
// Solidity: function findFirstNonEmptyBinId(uint24 _id, bool _swapForY) view returns(uint24)
func (_LBPair *LBPairCallerSession) FindFirstNonEmptyBinId(_id *big.Int, _swapForY bool) (*big.Int, error) {
	return _LBPair.Contract.FindFirstNonEmptyBinId(&_LBPair.CallOpts, _id, _swapForY)
}

// GetBin is a free data retrieval call binding the contract method 0x0abe9688.
//
// Solidity: function getBin(uint24 _id) view returns(uint256 reserveX, uint256 reserveY)
func (_LBPair *LBPairCaller) GetBin(opts *bind.CallOpts, _id *big.Int) (struct {
	ReserveX *big.Int
	ReserveY *big.Int
}, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "getBin", _id)

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

// GetBin is a free data retrieval call binding the contract method 0x0abe9688.
//
// Solidity: function getBin(uint24 _id) view returns(uint256 reserveX, uint256 reserveY)
func (_LBPair *LBPairSession) GetBin(_id *big.Int) (struct {
	ReserveX *big.Int
	ReserveY *big.Int
}, error) {
	return _LBPair.Contract.GetBin(&_LBPair.CallOpts, _id)
}

// GetBin is a free data retrieval call binding the contract method 0x0abe9688.
//
// Solidity: function getBin(uint24 _id) view returns(uint256 reserveX, uint256 reserveY)
func (_LBPair *LBPairCallerSession) GetBin(_id *big.Int) (struct {
	ReserveX *big.Int
	ReserveY *big.Int
}, error) {
	return _LBPair.Contract.GetBin(&_LBPair.CallOpts, _id)
}

// GetGlobalFees is a free data retrieval call binding the contract method 0xa582cdaa.
//
// Solidity: function getGlobalFees() view returns(uint128 feesXTotal, uint128 feesYTotal, uint128 feesXProtocol, uint128 feesYProtocol)
func (_LBPair *LBPairCaller) GetGlobalFees(opts *bind.CallOpts) (struct {
	FeesXTotal    *big.Int
	FeesYTotal    *big.Int
	FeesXProtocol *big.Int
	FeesYProtocol *big.Int
}, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "getGlobalFees")

	outstruct := new(struct {
		FeesXTotal    *big.Int
		FeesYTotal    *big.Int
		FeesXProtocol *big.Int
		FeesYProtocol *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.FeesXTotal = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.FeesYTotal = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.FeesXProtocol = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.FeesYProtocol = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetGlobalFees is a free data retrieval call binding the contract method 0xa582cdaa.
//
// Solidity: function getGlobalFees() view returns(uint128 feesXTotal, uint128 feesYTotal, uint128 feesXProtocol, uint128 feesYProtocol)
func (_LBPair *LBPairSession) GetGlobalFees() (struct {
	FeesXTotal    *big.Int
	FeesYTotal    *big.Int
	FeesXProtocol *big.Int
	FeesYProtocol *big.Int
}, error) {
	return _LBPair.Contract.GetGlobalFees(&_LBPair.CallOpts)
}

// GetGlobalFees is a free data retrieval call binding the contract method 0xa582cdaa.
//
// Solidity: function getGlobalFees() view returns(uint128 feesXTotal, uint128 feesYTotal, uint128 feesXProtocol, uint128 feesYProtocol)
func (_LBPair *LBPairCallerSession) GetGlobalFees() (struct {
	FeesXTotal    *big.Int
	FeesYTotal    *big.Int
	FeesXProtocol *big.Int
	FeesYProtocol *big.Int
}, error) {
	return _LBPair.Contract.GetGlobalFees(&_LBPair.CallOpts)
}

// GetOracleParameters is a free data retrieval call binding the contract method 0x55182894.
//
// Solidity: function getOracleParameters() view returns(uint256 oracleSampleLifetime, uint256 oracleSize, uint256 oracleActiveSize, uint256 oracleLastTimestamp, uint256 oracleId, uint256 min, uint256 max)
func (_LBPair *LBPairCaller) GetOracleParameters(opts *bind.CallOpts) (struct {
	OracleSampleLifetime *big.Int
	OracleSize           *big.Int
	OracleActiveSize     *big.Int
	OracleLastTimestamp  *big.Int
	OracleId             *big.Int
	Min                  *big.Int
	Max                  *big.Int
}, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "getOracleParameters")

	outstruct := new(struct {
		OracleSampleLifetime *big.Int
		OracleSize           *big.Int
		OracleActiveSize     *big.Int
		OracleLastTimestamp  *big.Int
		OracleId             *big.Int
		Min                  *big.Int
		Max                  *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.OracleSampleLifetime = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.OracleSize = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.OracleActiveSize = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.OracleLastTimestamp = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.OracleId = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.Min = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)
	outstruct.Max = *abi.ConvertType(out[6], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetOracleParameters is a free data retrieval call binding the contract method 0x55182894.
//
// Solidity: function getOracleParameters() view returns(uint256 oracleSampleLifetime, uint256 oracleSize, uint256 oracleActiveSize, uint256 oracleLastTimestamp, uint256 oracleId, uint256 min, uint256 max)
func (_LBPair *LBPairSession) GetOracleParameters() (struct {
	OracleSampleLifetime *big.Int
	OracleSize           *big.Int
	OracleActiveSize     *big.Int
	OracleLastTimestamp  *big.Int
	OracleId             *big.Int
	Min                  *big.Int
	Max                  *big.Int
}, error) {
	return _LBPair.Contract.GetOracleParameters(&_LBPair.CallOpts)
}

// GetOracleParameters is a free data retrieval call binding the contract method 0x55182894.
//
// Solidity: function getOracleParameters() view returns(uint256 oracleSampleLifetime, uint256 oracleSize, uint256 oracleActiveSize, uint256 oracleLastTimestamp, uint256 oracleId, uint256 min, uint256 max)
func (_LBPair *LBPairCallerSession) GetOracleParameters() (struct {
	OracleSampleLifetime *big.Int
	OracleSize           *big.Int
	OracleActiveSize     *big.Int
	OracleLastTimestamp  *big.Int
	OracleId             *big.Int
	Min                  *big.Int
	Max                  *big.Int
}, error) {
	return _LBPair.Contract.GetOracleParameters(&_LBPair.CallOpts)
}

// GetOracleSampleFrom is a free data retrieval call binding the contract method 0xa21635a7.
//
// Solidity: function getOracleSampleFrom(uint256 _timeDelta) view returns(uint256 cumulativeId, uint256 cumulativeVolatilityAccumulated, uint256 cumulativeBinCrossed)
func (_LBPair *LBPairCaller) GetOracleSampleFrom(opts *bind.CallOpts, _timeDelta *big.Int) (struct {
	CumulativeId                    *big.Int
	CumulativeVolatilityAccumulated *big.Int
	CumulativeBinCrossed            *big.Int
}, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "getOracleSampleFrom", _timeDelta)

	outstruct := new(struct {
		CumulativeId                    *big.Int
		CumulativeVolatilityAccumulated *big.Int
		CumulativeBinCrossed            *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.CumulativeId = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.CumulativeVolatilityAccumulated = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.CumulativeBinCrossed = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetOracleSampleFrom is a free data retrieval call binding the contract method 0xa21635a7.
//
// Solidity: function getOracleSampleFrom(uint256 _timeDelta) view returns(uint256 cumulativeId, uint256 cumulativeVolatilityAccumulated, uint256 cumulativeBinCrossed)
func (_LBPair *LBPairSession) GetOracleSampleFrom(_timeDelta *big.Int) (struct {
	CumulativeId                    *big.Int
	CumulativeVolatilityAccumulated *big.Int
	CumulativeBinCrossed            *big.Int
}, error) {
	return _LBPair.Contract.GetOracleSampleFrom(&_LBPair.CallOpts, _timeDelta)
}

// GetOracleSampleFrom is a free data retrieval call binding the contract method 0xa21635a7.
//
// Solidity: function getOracleSampleFrom(uint256 _timeDelta) view returns(uint256 cumulativeId, uint256 cumulativeVolatilityAccumulated, uint256 cumulativeBinCrossed)
func (_LBPair *LBPairCallerSession) GetOracleSampleFrom(_timeDelta *big.Int) (struct {
	CumulativeId                    *big.Int
	CumulativeVolatilityAccumulated *big.Int
	CumulativeBinCrossed            *big.Int
}, error) {
	return _LBPair.Contract.GetOracleSampleFrom(&_LBPair.CallOpts, _timeDelta)
}

// GetReservesAndId is a free data retrieval call binding the contract method 0x1b05b83e.
//
// Solidity: function getReservesAndId() view returns(uint256 reserveX, uint256 reserveY, uint256 activeId)
func (_LBPair *LBPairCaller) GetReservesAndId(opts *bind.CallOpts) (struct {
	ReserveX *big.Int
	ReserveY *big.Int
	ActiveId *big.Int
}, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "getReservesAndId")

	outstruct := new(struct {
		ReserveX *big.Int
		ReserveY *big.Int
		ActiveId *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.ReserveX = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.ReserveY = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.ActiveId = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetReservesAndId is a free data retrieval call binding the contract method 0x1b05b83e.
//
// Solidity: function getReservesAndId() view returns(uint256 reserveX, uint256 reserveY, uint256 activeId)
func (_LBPair *LBPairSession) GetReservesAndId() (struct {
	ReserveX *big.Int
	ReserveY *big.Int
	ActiveId *big.Int
}, error) {
	return _LBPair.Contract.GetReservesAndId(&_LBPair.CallOpts)
}

// GetReservesAndId is a free data retrieval call binding the contract method 0x1b05b83e.
//
// Solidity: function getReservesAndId() view returns(uint256 reserveX, uint256 reserveY, uint256 activeId)
func (_LBPair *LBPairCallerSession) GetReservesAndId() (struct {
	ReserveX *big.Int
	ReserveY *big.Int
	ActiveId *big.Int
}, error) {
	return _LBPair.Contract.GetReservesAndId(&_LBPair.CallOpts)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address _owner, address _spender) view returns(bool)
func (_LBPair *LBPairCaller) IsApprovedForAll(opts *bind.CallOpts, _owner common.Address, _spender common.Address) (bool, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "isApprovedForAll", _owner, _spender)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address _owner, address _spender) view returns(bool)
func (_LBPair *LBPairSession) IsApprovedForAll(_owner common.Address, _spender common.Address) (bool, error) {
	return _LBPair.Contract.IsApprovedForAll(&_LBPair.CallOpts, _owner, _spender)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address _owner, address _spender) view returns(bool)
func (_LBPair *LBPairCallerSession) IsApprovedForAll(_owner common.Address, _spender common.Address) (bool, error) {
	return _LBPair.Contract.IsApprovedForAll(&_LBPair.CallOpts, _owner, _spender)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() pure returns(string)
func (_LBPair *LBPairCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() pure returns(string)
func (_LBPair *LBPairSession) Name() (string, error) {
	return _LBPair.Contract.Name(&_LBPair.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() pure returns(string)
func (_LBPair *LBPairCallerSession) Name() (string, error) {
	return _LBPair.Contract.Name(&_LBPair.CallOpts)
}

// PendingFees is a free data retrieval call binding the contract method 0xf7cff1f8.
//
// Solidity: function pendingFees(address _account, uint256[] _ids) view returns(uint256 amountX, uint256 amountY)
func (_LBPair *LBPairCaller) PendingFees(opts *bind.CallOpts, _account common.Address, _ids []*big.Int) (struct {
	AmountX *big.Int
	AmountY *big.Int
}, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "pendingFees", _account, _ids)

	outstruct := new(struct {
		AmountX *big.Int
		AmountY *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.AmountX = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.AmountY = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// PendingFees is a free data retrieval call binding the contract method 0xf7cff1f8.
//
// Solidity: function pendingFees(address _account, uint256[] _ids) view returns(uint256 amountX, uint256 amountY)
func (_LBPair *LBPairSession) PendingFees(_account common.Address, _ids []*big.Int) (struct {
	AmountX *big.Int
	AmountY *big.Int
}, error) {
	return _LBPair.Contract.PendingFees(&_LBPair.CallOpts, _account, _ids)
}

// PendingFees is a free data retrieval call binding the contract method 0xf7cff1f8.
//
// Solidity: function pendingFees(address _account, uint256[] _ids) view returns(uint256 amountX, uint256 amountY)
func (_LBPair *LBPairCallerSession) PendingFees(_account common.Address, _ids []*big.Int) (struct {
	AmountX *big.Int
	AmountY *big.Int
}, error) {
	return _LBPair.Contract.PendingFees(&_LBPair.CallOpts, _account, _ids)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 _interfaceId) view returns(bool)
func (_LBPair *LBPairCaller) SupportsInterface(opts *bind.CallOpts, _interfaceId [4]byte) (bool, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "supportsInterface", _interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 _interfaceId) view returns(bool)
func (_LBPair *LBPairSession) SupportsInterface(_interfaceId [4]byte) (bool, error) {
	return _LBPair.Contract.SupportsInterface(&_LBPair.CallOpts, _interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 _interfaceId) view returns(bool)
func (_LBPair *LBPairCallerSession) SupportsInterface(_interfaceId [4]byte) (bool, error) {
	return _LBPair.Contract.SupportsInterface(&_LBPair.CallOpts, _interfaceId)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() pure returns(string)
func (_LBPair *LBPairCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() pure returns(string)
func (_LBPair *LBPairSession) Symbol() (string, error) {
	return _LBPair.Contract.Symbol(&_LBPair.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() pure returns(string)
func (_LBPair *LBPairCallerSession) Symbol() (string, error) {
	return _LBPair.Contract.Symbol(&_LBPair.CallOpts)
}

// TokenX is a free data retrieval call binding the contract method 0x16dc165b.
//
// Solidity: function tokenX() view returns(address)
func (_LBPair *LBPairCaller) TokenX(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "tokenX")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// TokenX is a free data retrieval call binding the contract method 0x16dc165b.
//
// Solidity: function tokenX() view returns(address)
func (_LBPair *LBPairSession) TokenX() (common.Address, error) {
	return _LBPair.Contract.TokenX(&_LBPair.CallOpts)
}

// TokenX is a free data retrieval call binding the contract method 0x16dc165b.
//
// Solidity: function tokenX() view returns(address)
func (_LBPair *LBPairCallerSession) TokenX() (common.Address, error) {
	return _LBPair.Contract.TokenX(&_LBPair.CallOpts)
}

// TokenY is a free data retrieval call binding the contract method 0xb7d19fc4.
//
// Solidity: function tokenY() view returns(address)
func (_LBPair *LBPairCaller) TokenY(opts *bind.CallOpts) (common.Address, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "tokenY")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// TokenY is a free data retrieval call binding the contract method 0xb7d19fc4.
//
// Solidity: function tokenY() view returns(address)
func (_LBPair *LBPairSession) TokenY() (common.Address, error) {
	return _LBPair.Contract.TokenY(&_LBPair.CallOpts)
}

// TokenY is a free data retrieval call binding the contract method 0xb7d19fc4.
//
// Solidity: function tokenY() view returns(address)
func (_LBPair *LBPairCallerSession) TokenY() (common.Address, error) {
	return _LBPair.Contract.TokenY(&_LBPair.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0xbd85b039.
//
// Solidity: function totalSupply(uint256 _id) view returns(uint256)
func (_LBPair *LBPairCaller) TotalSupply(opts *bind.CallOpts, _id *big.Int) (*big.Int, error) {
	var out []any
	err := _LBPair.contract.Call(opts, &out, "totalSupply", _id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0xbd85b039.
//
// Solidity: function totalSupply(uint256 _id) view returns(uint256)
func (_LBPair *LBPairSession) TotalSupply(_id *big.Int) (*big.Int, error) {
	return _LBPair.Contract.TotalSupply(&_LBPair.CallOpts, _id)
}

// TotalSupply is a free data retrieval call binding the contract method 0xbd85b039.
//
// Solidity: function totalSupply(uint256 _id) view returns(uint256)
func (_LBPair *LBPairCallerSession) TotalSupply(_id *big.Int) (*big.Int, error) {
	return _LBPair.Contract.TotalSupply(&_LBPair.CallOpts, _id)
}

// Burn is a paid mutator transaction binding the contract method 0x0acd451d.
//
// Solidity: function burn(uint256[] _ids, uint256[] _amounts, address _to) returns(uint256 amountX, uint256 amountY)
func (_LBPair *LBPairTransactor) Burn(opts *bind.TransactOpts, _ids []*big.Int, _amounts []*big.Int, _to common.Address) (*types.Transaction, error) {
	return _LBPair.contract.Transact(opts, "burn", _ids, _amounts, _to)
}

// Burn is a paid mutator transaction binding the contract method 0x0acd451d.
//
// Solidity: function burn(uint256[] _ids, uint256[] _amounts, address _to) returns(uint256 amountX, uint256 amountY)
func (_LBPair *LBPairSession) Burn(_ids []*big.Int, _amounts []*big.Int, _to common.Address) (*types.Transaction, error) {
	return _LBPair.Contract.Burn(&_LBPair.TransactOpts, _ids, _amounts, _to)
}

// Burn is a paid mutator transaction binding the contract method 0x0acd451d.
//
// Solidity: function burn(uint256[] _ids, uint256[] _amounts, address _to) returns(uint256 amountX, uint256 amountY)
func (_LBPair *LBPairTransactorSession) Burn(_ids []*big.Int, _amounts []*big.Int, _to common.Address) (*types.Transaction, error) {
	return _LBPair.Contract.Burn(&_LBPair.TransactOpts, _ids, _amounts, _to)
}

// CollectFees is a paid mutator transaction binding the contract method 0x225b20b9.
//
// Solidity: function collectFees(address _account, uint256[] _ids) returns(uint256 amountX, uint256 amountY)
func (_LBPair *LBPairTransactor) CollectFees(opts *bind.TransactOpts, _account common.Address, _ids []*big.Int) (*types.Transaction, error) {
	return _LBPair.contract.Transact(opts, "collectFees", _account, _ids)
}

// CollectFees is a paid mutator transaction binding the contract method 0x225b20b9.
//
// Solidity: function collectFees(address _account, uint256[] _ids) returns(uint256 amountX, uint256 amountY)
func (_LBPair *LBPairSession) CollectFees(_account common.Address, _ids []*big.Int) (*types.Transaction, error) {
	return _LBPair.Contract.CollectFees(&_LBPair.TransactOpts, _account, _ids)
}

// CollectFees is a paid mutator transaction binding the contract method 0x225b20b9.
//
// Solidity: function collectFees(address _account, uint256[] _ids) returns(uint256 amountX, uint256 amountY)
func (_LBPair *LBPairTransactorSession) CollectFees(_account common.Address, _ids []*big.Int) (*types.Transaction, error) {
	return _LBPair.Contract.CollectFees(&_LBPair.TransactOpts, _account, _ids)
}

// CollectProtocolFees is a paid mutator transaction binding the contract method 0xa1af5b9a.
//
// Solidity: function collectProtocolFees() returns(uint128 amountX, uint128 amountY)
func (_LBPair *LBPairTransactor) CollectProtocolFees(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LBPair.contract.Transact(opts, "collectProtocolFees")
}

// CollectProtocolFees is a paid mutator transaction binding the contract method 0xa1af5b9a.
//
// Solidity: function collectProtocolFees() returns(uint128 amountX, uint128 amountY)
func (_LBPair *LBPairSession) CollectProtocolFees() (*types.Transaction, error) {
	return _LBPair.Contract.CollectProtocolFees(&_LBPair.TransactOpts)
}

// CollectProtocolFees is a paid mutator transaction binding the contract method 0xa1af5b9a.
//
// Solidity: function collectProtocolFees() returns(uint128 amountX, uint128 amountY)
func (_LBPair *LBPairTransactorSession) CollectProtocolFees() (*types.Transaction, error) {
	return _LBPair.Contract.CollectProtocolFees(&_LBPair.TransactOpts)
}

// FlashLoan is a paid mutator transaction binding the contract method 0x5cffe9de.
//
// Solidity: function flashLoan(address _receiver, address _token, uint256 _amount, bytes _data) returns()
func (_LBPair *LBPairTransactor) FlashLoan(opts *bind.TransactOpts, _receiver common.Address, _token common.Address, _amount *big.Int, _data []byte) (*types.Transaction, error) {
	return _LBPair.contract.Transact(opts, "flashLoan", _receiver, _token, _amount, _data)
}

// FlashLoan is a paid mutator transaction binding the contract method 0x5cffe9de.
//
// Solidity: function flashLoan(address _receiver, address _token, uint256 _amount, bytes _data) returns()
func (_LBPair *LBPairSession) FlashLoan(_receiver common.Address, _token common.Address, _amount *big.Int, _data []byte) (*types.Transaction, error) {
	return _LBPair.Contract.FlashLoan(&_LBPair.TransactOpts, _receiver, _token, _amount, _data)
}

// FlashLoan is a paid mutator transaction binding the contract method 0x5cffe9de.
//
// Solidity: function flashLoan(address _receiver, address _token, uint256 _amount, bytes _data) returns()
func (_LBPair *LBPairTransactorSession) FlashLoan(_receiver common.Address, _token common.Address, _amount *big.Int, _data []byte) (*types.Transaction, error) {
	return _LBPair.Contract.FlashLoan(&_LBPair.TransactOpts, _receiver, _token, _amount, _data)
}

// ForceDecay is a paid mutator transaction binding the contract method 0xd3b9fbe4.
//
// Solidity: function forceDecay() returns()
func (_LBPair *LBPairTransactor) ForceDecay(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LBPair.contract.Transact(opts, "forceDecay")
}

// ForceDecay is a paid mutator transaction binding the contract method 0xd3b9fbe4.
//
// Solidity: function forceDecay() returns()
func (_LBPair *LBPairSession) ForceDecay() (*types.Transaction, error) {
	return _LBPair.Contract.ForceDecay(&_LBPair.TransactOpts)
}

// ForceDecay is a paid mutator transaction binding the contract method 0xd3b9fbe4.
//
// Solidity: function forceDecay() returns()
func (_LBPair *LBPairTransactorSession) ForceDecay() (*types.Transaction, error) {
	return _LBPair.Contract.ForceDecay(&_LBPair.TransactOpts)
}

// IncreaseOracleLength is a paid mutator transaction binding the contract method 0xc7bd6586.
//
// Solidity: function increaseOracleLength(uint16 _newLength) returns()
func (_LBPair *LBPairTransactor) IncreaseOracleLength(opts *bind.TransactOpts, _newLength uint16) (*types.Transaction, error) {
	return _LBPair.contract.Transact(opts, "increaseOracleLength", _newLength)
}

// IncreaseOracleLength is a paid mutator transaction binding the contract method 0xc7bd6586.
//
// Solidity: function increaseOracleLength(uint16 _newLength) returns()
func (_LBPair *LBPairSession) IncreaseOracleLength(_newLength uint16) (*types.Transaction, error) {
	return _LBPair.Contract.IncreaseOracleLength(&_LBPair.TransactOpts, _newLength)
}

// IncreaseOracleLength is a paid mutator transaction binding the contract method 0xc7bd6586.
//
// Solidity: function increaseOracleLength(uint16 _newLength) returns()
func (_LBPair *LBPairTransactorSession) IncreaseOracleLength(_newLength uint16) (*types.Transaction, error) {
	return _LBPair.Contract.IncreaseOracleLength(&_LBPair.TransactOpts, _newLength)
}

// Initialize is a paid mutator transaction binding the contract method 0xd32db437.
//
// Solidity: function initialize(address _tokenX, address _tokenY, uint24 _activeId, uint16 _sampleLifetime, bytes32 _packedFeeParameters) returns()
func (_LBPair *LBPairTransactor) Initialize(opts *bind.TransactOpts, _tokenX common.Address, _tokenY common.Address, _activeId *big.Int, _sampleLifetime uint16, _packedFeeParameters [32]byte) (*types.Transaction, error) {
	return _LBPair.contract.Transact(opts, "initialize", _tokenX, _tokenY, _activeId, _sampleLifetime, _packedFeeParameters)
}

// Initialize is a paid mutator transaction binding the contract method 0xd32db437.
//
// Solidity: function initialize(address _tokenX, address _tokenY, uint24 _activeId, uint16 _sampleLifetime, bytes32 _packedFeeParameters) returns()
func (_LBPair *LBPairSession) Initialize(_tokenX common.Address, _tokenY common.Address, _activeId *big.Int, _sampleLifetime uint16, _packedFeeParameters [32]byte) (*types.Transaction, error) {
	return _LBPair.Contract.Initialize(&_LBPair.TransactOpts, _tokenX, _tokenY, _activeId, _sampleLifetime, _packedFeeParameters)
}

// Initialize is a paid mutator transaction binding the contract method 0xd32db437.
//
// Solidity: function initialize(address _tokenX, address _tokenY, uint24 _activeId, uint16 _sampleLifetime, bytes32 _packedFeeParameters) returns()
func (_LBPair *LBPairTransactorSession) Initialize(_tokenX common.Address, _tokenY common.Address, _activeId *big.Int, _sampleLifetime uint16, _packedFeeParameters [32]byte) (*types.Transaction, error) {
	return _LBPair.Contract.Initialize(&_LBPair.TransactOpts, _tokenX, _tokenY, _activeId, _sampleLifetime, _packedFeeParameters)
}

// Mint is a paid mutator transaction binding the contract method 0x714c8592.
//
// Solidity: function mint(uint256[] _ids, uint256[] _distributionX, uint256[] _distributionY, address _to) returns(uint256, uint256, uint256[] liquidityMinted)
func (_LBPair *LBPairTransactor) Mint(opts *bind.TransactOpts, _ids []*big.Int, _distributionX []*big.Int, _distributionY []*big.Int, _to common.Address) (*types.Transaction, error) {
	return _LBPair.contract.Transact(opts, "mint", _ids, _distributionX, _distributionY, _to)
}

// Mint is a paid mutator transaction binding the contract method 0x714c8592.
//
// Solidity: function mint(uint256[] _ids, uint256[] _distributionX, uint256[] _distributionY, address _to) returns(uint256, uint256, uint256[] liquidityMinted)
func (_LBPair *LBPairSession) Mint(_ids []*big.Int, _distributionX []*big.Int, _distributionY []*big.Int, _to common.Address) (*types.Transaction, error) {
	return _LBPair.Contract.Mint(&_LBPair.TransactOpts, _ids, _distributionX, _distributionY, _to)
}

// Mint is a paid mutator transaction binding the contract method 0x714c8592.
//
// Solidity: function mint(uint256[] _ids, uint256[] _distributionX, uint256[] _distributionY, address _to) returns(uint256, uint256, uint256[] liquidityMinted)
func (_LBPair *LBPairTransactorSession) Mint(_ids []*big.Int, _distributionX []*big.Int, _distributionY []*big.Int, _to common.Address) (*types.Transaction, error) {
	return _LBPair.Contract.Mint(&_LBPair.TransactOpts, _ids, _distributionX, _distributionY, _to)
}

// SafeBatchTransferFrom is a paid mutator transaction binding the contract method 0xfba0ee64.
//
// Solidity: function safeBatchTransferFrom(address _from, address _to, uint256[] _ids, uint256[] _amounts) returns()
func (_LBPair *LBPairTransactor) SafeBatchTransferFrom(opts *bind.TransactOpts, _from common.Address, _to common.Address, _ids []*big.Int, _amounts []*big.Int) (*types.Transaction, error) {
	return _LBPair.contract.Transact(opts, "safeBatchTransferFrom", _from, _to, _ids, _amounts)
}

// SafeBatchTransferFrom is a paid mutator transaction binding the contract method 0xfba0ee64.
//
// Solidity: function safeBatchTransferFrom(address _from, address _to, uint256[] _ids, uint256[] _amounts) returns()
func (_LBPair *LBPairSession) SafeBatchTransferFrom(_from common.Address, _to common.Address, _ids []*big.Int, _amounts []*big.Int) (*types.Transaction, error) {
	return _LBPair.Contract.SafeBatchTransferFrom(&_LBPair.TransactOpts, _from, _to, _ids, _amounts)
}

// SafeBatchTransferFrom is a paid mutator transaction binding the contract method 0xfba0ee64.
//
// Solidity: function safeBatchTransferFrom(address _from, address _to, uint256[] _ids, uint256[] _amounts) returns()
func (_LBPair *LBPairTransactorSession) SafeBatchTransferFrom(_from common.Address, _to common.Address, _ids []*big.Int, _amounts []*big.Int) (*types.Transaction, error) {
	return _LBPair.Contract.SafeBatchTransferFrom(&_LBPair.TransactOpts, _from, _to, _ids, _amounts)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x0febdd49.
//
// Solidity: function safeTransferFrom(address _from, address _to, uint256 _id, uint256 _amount) returns()
func (_LBPair *LBPairTransactor) SafeTransferFrom(opts *bind.TransactOpts, _from common.Address, _to common.Address, _id *big.Int, _amount *big.Int) (*types.Transaction, error) {
	return _LBPair.contract.Transact(opts, "safeTransferFrom", _from, _to, _id, _amount)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x0febdd49.
//
// Solidity: function safeTransferFrom(address _from, address _to, uint256 _id, uint256 _amount) returns()
func (_LBPair *LBPairSession) SafeTransferFrom(_from common.Address, _to common.Address, _id *big.Int, _amount *big.Int) (*types.Transaction, error) {
	return _LBPair.Contract.SafeTransferFrom(&_LBPair.TransactOpts, _from, _to, _id, _amount)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x0febdd49.
//
// Solidity: function safeTransferFrom(address _from, address _to, uint256 _id, uint256 _amount) returns()
func (_LBPair *LBPairTransactorSession) SafeTransferFrom(_from common.Address, _to common.Address, _id *big.Int, _amount *big.Int) (*types.Transaction, error) {
	return _LBPair.Contract.SafeTransferFrom(&_LBPair.TransactOpts, _from, _to, _id, _amount)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address _spender, bool _approved) returns()
func (_LBPair *LBPairTransactor) SetApprovalForAll(opts *bind.TransactOpts, _spender common.Address, _approved bool) (*types.Transaction, error) {
	return _LBPair.contract.Transact(opts, "setApprovalForAll", _spender, _approved)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address _spender, bool _approved) returns()
func (_LBPair *LBPairSession) SetApprovalForAll(_spender common.Address, _approved bool) (*types.Transaction, error) {
	return _LBPair.Contract.SetApprovalForAll(&_LBPair.TransactOpts, _spender, _approved)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address _spender, bool _approved) returns()
func (_LBPair *LBPairTransactorSession) SetApprovalForAll(_spender common.Address, _approved bool) (*types.Transaction, error) {
	return _LBPair.Contract.SetApprovalForAll(&_LBPair.TransactOpts, _spender, _approved)
}

// SetFeesParameters is a paid mutator transaction binding the contract method 0x54b5fc87.
//
// Solidity: function setFeesParameters(bytes32 _packedFeeParameters) returns()
func (_LBPair *LBPairTransactor) SetFeesParameters(opts *bind.TransactOpts, _packedFeeParameters [32]byte) (*types.Transaction, error) {
	return _LBPair.contract.Transact(opts, "setFeesParameters", _packedFeeParameters)
}

// SetFeesParameters is a paid mutator transaction binding the contract method 0x54b5fc87.
//
// Solidity: function setFeesParameters(bytes32 _packedFeeParameters) returns()
func (_LBPair *LBPairSession) SetFeesParameters(_packedFeeParameters [32]byte) (*types.Transaction, error) {
	return _LBPair.Contract.SetFeesParameters(&_LBPair.TransactOpts, _packedFeeParameters)
}

// SetFeesParameters is a paid mutator transaction binding the contract method 0x54b5fc87.
//
// Solidity: function setFeesParameters(bytes32 _packedFeeParameters) returns()
func (_LBPair *LBPairTransactorSession) SetFeesParameters(_packedFeeParameters [32]byte) (*types.Transaction, error) {
	return _LBPair.Contract.SetFeesParameters(&_LBPair.TransactOpts, _packedFeeParameters)
}

// Swap is a paid mutator transaction binding the contract method 0x53c059a0.
//
// Solidity: function swap(bool _swapForY, address _to) returns(uint256 amountXOut, uint256 amountYOut)
func (_LBPair *LBPairTransactor) Swap(opts *bind.TransactOpts, _swapForY bool, _to common.Address) (*types.Transaction, error) {
	return _LBPair.contract.Transact(opts, "swap", _swapForY, _to)
}

// Swap is a paid mutator transaction binding the contract method 0x53c059a0.
//
// Solidity: function swap(bool _swapForY, address _to) returns(uint256 amountXOut, uint256 amountYOut)
func (_LBPair *LBPairSession) Swap(_swapForY bool, _to common.Address) (*types.Transaction, error) {
	return _LBPair.Contract.Swap(&_LBPair.TransactOpts, _swapForY, _to)
}

// Swap is a paid mutator transaction binding the contract method 0x53c059a0.
//
// Solidity: function swap(bool _swapForY, address _to) returns(uint256 amountXOut, uint256 amountYOut)
func (_LBPair *LBPairTransactorSession) Swap(_swapForY bool, _to common.Address) (*types.Transaction, error) {
	return _LBPair.Contract.Swap(&_LBPair.TransactOpts, _swapForY, _to)
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

	var accountRule []any
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []any
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

	var accountRule []any
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []any
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

// LBPairCompositionFeeIterator is returned from FilterCompositionFee and is used to iterate over the raw logs and unpacked data for CompositionFee events raised by the LBPair contract.
type LBPairCompositionFeeIterator struct {
	Event *LBPairCompositionFee // Event containing the contract specifics and raw log

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
func (it *LBPairCompositionFeeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairCompositionFee)
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
		it.Event = new(LBPairCompositionFee)
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
func (it *LBPairCompositionFeeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairCompositionFeeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairCompositionFee represents a CompositionFee event raised by the LBPair contract.
type LBPairCompositionFee struct {
	Sender    common.Address
	Recipient common.Address
	Id        *big.Int
	FeesX     *big.Int
	FeesY     *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterCompositionFee is a free log retrieval operation binding the contract event 0x56f8e764728c77dd99ffbc1b64e6d02e227e6ec8214f165d4ef31351de136a0d.
//
// Solidity: event CompositionFee(address indexed sender, address indexed recipient, uint256 indexed id, uint256 feesX, uint256 feesY)
func (_LBPair *LBPairFilterer) FilterCompositionFee(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address, id []*big.Int) (*LBPairCompositionFeeIterator, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "CompositionFee", senderRule, recipientRule, idRule)
	if err != nil {
		return nil, err
	}
	return &LBPairCompositionFeeIterator{contract: _LBPair.contract, event: "CompositionFee", logs: logs, sub: sub}, nil
}

// WatchCompositionFee is a free log subscription operation binding the contract event 0x56f8e764728c77dd99ffbc1b64e6d02e227e6ec8214f165d4ef31351de136a0d.
//
// Solidity: event CompositionFee(address indexed sender, address indexed recipient, uint256 indexed id, uint256 feesX, uint256 feesY)
func (_LBPair *LBPairFilterer) WatchCompositionFee(opts *bind.WatchOpts, sink chan<- *LBPairCompositionFee, sender []common.Address, recipient []common.Address, id []*big.Int) (event.Subscription, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "CompositionFee", senderRule, recipientRule, idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairCompositionFee)
				if err := _LBPair.contract.UnpackLog(event, "CompositionFee", log); err != nil {
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

// ParseCompositionFee is a log parse operation binding the contract event 0x56f8e764728c77dd99ffbc1b64e6d02e227e6ec8214f165d4ef31351de136a0d.
//
// Solidity: event CompositionFee(address indexed sender, address indexed recipient, uint256 indexed id, uint256 feesX, uint256 feesY)
func (_LBPair *LBPairFilterer) ParseCompositionFee(log types.Log) (*LBPairCompositionFee, error) {
	event := new(LBPairCompositionFee)
	if err := _LBPair.contract.UnpackLog(event, "CompositionFee", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairDepositedToBinIterator is returned from FilterDepositedToBin and is used to iterate over the raw logs and unpacked data for DepositedToBin events raised by the LBPair contract.
type LBPairDepositedToBinIterator struct {
	Event *LBPairDepositedToBin // Event containing the contract specifics and raw log

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
func (it *LBPairDepositedToBinIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairDepositedToBin)
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
		it.Event = new(LBPairDepositedToBin)
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
func (it *LBPairDepositedToBinIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairDepositedToBinIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairDepositedToBin represents a DepositedToBin event raised by the LBPair contract.
type LBPairDepositedToBin struct {
	Sender    common.Address
	Recipient common.Address
	Id        *big.Int
	AmountX   *big.Int
	AmountY   *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterDepositedToBin is a free log retrieval operation binding the contract event 0x4216cc3bd0c40a90259d92f800c06ede5c47765f41a488072b7e7104a1f95841.
//
// Solidity: event DepositedToBin(address indexed sender, address indexed recipient, uint256 indexed id, uint256 amountX, uint256 amountY)
func (_LBPair *LBPairFilterer) FilterDepositedToBin(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address, id []*big.Int) (*LBPairDepositedToBinIterator, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "DepositedToBin", senderRule, recipientRule, idRule)
	if err != nil {
		return nil, err
	}
	return &LBPairDepositedToBinIterator{contract: _LBPair.contract, event: "DepositedToBin", logs: logs, sub: sub}, nil
}

// WatchDepositedToBin is a free log subscription operation binding the contract event 0x4216cc3bd0c40a90259d92f800c06ede5c47765f41a488072b7e7104a1f95841.
//
// Solidity: event DepositedToBin(address indexed sender, address indexed recipient, uint256 indexed id, uint256 amountX, uint256 amountY)
func (_LBPair *LBPairFilterer) WatchDepositedToBin(opts *bind.WatchOpts, sink chan<- *LBPairDepositedToBin, sender []common.Address, recipient []common.Address, id []*big.Int) (event.Subscription, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "DepositedToBin", senderRule, recipientRule, idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairDepositedToBin)
				if err := _LBPair.contract.UnpackLog(event, "DepositedToBin", log); err != nil {
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

// ParseDepositedToBin is a log parse operation binding the contract event 0x4216cc3bd0c40a90259d92f800c06ede5c47765f41a488072b7e7104a1f95841.
//
// Solidity: event DepositedToBin(address indexed sender, address indexed recipient, uint256 indexed id, uint256 amountX, uint256 amountY)
func (_LBPair *LBPairFilterer) ParseDepositedToBin(log types.Log) (*LBPairDepositedToBin, error) {
	event := new(LBPairDepositedToBin)
	if err := _LBPair.contract.UnpackLog(event, "DepositedToBin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairFeesCollectedIterator is returned from FilterFeesCollected and is used to iterate over the raw logs and unpacked data for FeesCollected events raised by the LBPair contract.
type LBPairFeesCollectedIterator struct {
	Event *LBPairFeesCollected // Event containing the contract specifics and raw log

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
func (it *LBPairFeesCollectedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairFeesCollected)
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
		it.Event = new(LBPairFeesCollected)
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
func (it *LBPairFeesCollectedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairFeesCollectedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairFeesCollected represents a FeesCollected event raised by the LBPair contract.
type LBPairFeesCollected struct {
	Sender    common.Address
	Recipient common.Address
	AmountX   *big.Int
	AmountY   *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterFeesCollected is a free log retrieval operation binding the contract event 0x28a87b6059180e46de5fb9ab35eb043e8fe00ab45afcc7789e3934ecbbcde3ea.
//
// Solidity: event FeesCollected(address indexed sender, address indexed recipient, uint256 amountX, uint256 amountY)
func (_LBPair *LBPairFilterer) FilterFeesCollected(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address) (*LBPairFeesCollectedIterator, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "FeesCollected", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &LBPairFeesCollectedIterator{contract: _LBPair.contract, event: "FeesCollected", logs: logs, sub: sub}, nil
}

// WatchFeesCollected is a free log subscription operation binding the contract event 0x28a87b6059180e46de5fb9ab35eb043e8fe00ab45afcc7789e3934ecbbcde3ea.
//
// Solidity: event FeesCollected(address indexed sender, address indexed recipient, uint256 amountX, uint256 amountY)
func (_LBPair *LBPairFilterer) WatchFeesCollected(opts *bind.WatchOpts, sink chan<- *LBPairFeesCollected, sender []common.Address, recipient []common.Address) (event.Subscription, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "FeesCollected", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairFeesCollected)
				if err := _LBPair.contract.UnpackLog(event, "FeesCollected", log); err != nil {
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

// ParseFeesCollected is a log parse operation binding the contract event 0x28a87b6059180e46de5fb9ab35eb043e8fe00ab45afcc7789e3934ecbbcde3ea.
//
// Solidity: event FeesCollected(address indexed sender, address indexed recipient, uint256 amountX, uint256 amountY)
func (_LBPair *LBPairFilterer) ParseFeesCollected(log types.Log) (*LBPairFeesCollected, error) {
	event := new(LBPairFeesCollected)
	if err := _LBPair.contract.UnpackLog(event, "FeesCollected", log); err != nil {
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
	Sender   common.Address
	Receiver common.Address
	Token    common.Address
	Amount   *big.Int
	Fee      *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterFlashLoan is a free log retrieval operation binding the contract event 0x3659d15bd4bb92ab352a8d35bc3119ec6e7e0ab48e4d46201c8a28e02b6a8a86.
//
// Solidity: event FlashLoan(address indexed sender, address indexed receiver, address token, uint256 amount, uint256 fee)
func (_LBPair *LBPairFilterer) FilterFlashLoan(opts *bind.FilterOpts, sender []common.Address, receiver []common.Address) (*LBPairFlashLoanIterator, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var receiverRule []any
	for _, receiverItem := range receiver {
		receiverRule = append(receiverRule, receiverItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "FlashLoan", senderRule, receiverRule)
	if err != nil {
		return nil, err
	}
	return &LBPairFlashLoanIterator{contract: _LBPair.contract, event: "FlashLoan", logs: logs, sub: sub}, nil
}

// WatchFlashLoan is a free log subscription operation binding the contract event 0x3659d15bd4bb92ab352a8d35bc3119ec6e7e0ab48e4d46201c8a28e02b6a8a86.
//
// Solidity: event FlashLoan(address indexed sender, address indexed receiver, address token, uint256 amount, uint256 fee)
func (_LBPair *LBPairFilterer) WatchFlashLoan(opts *bind.WatchOpts, sink chan<- *LBPairFlashLoan, sender []common.Address, receiver []common.Address) (event.Subscription, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var receiverRule []any
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

// ParseFlashLoan is a log parse operation binding the contract event 0x3659d15bd4bb92ab352a8d35bc3119ec6e7e0ab48e4d46201c8a28e02b6a8a86.
//
// Solidity: event FlashLoan(address indexed sender, address indexed receiver, address token, uint256 amount, uint256 fee)
func (_LBPair *LBPairFilterer) ParseFlashLoan(log types.Log) (*LBPairFlashLoan, error) {
	event := new(LBPairFlashLoan)
	if err := _LBPair.contract.UnpackLog(event, "FlashLoan", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairOracleSizeIncreasedIterator is returned from FilterOracleSizeIncreased and is used to iterate over the raw logs and unpacked data for OracleSizeIncreased events raised by the LBPair contract.
type LBPairOracleSizeIncreasedIterator struct {
	Event *LBPairOracleSizeIncreased // Event containing the contract specifics and raw log

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
func (it *LBPairOracleSizeIncreasedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairOracleSizeIncreased)
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
		it.Event = new(LBPairOracleSizeIncreased)
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
func (it *LBPairOracleSizeIncreasedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairOracleSizeIncreasedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairOracleSizeIncreased represents a OracleSizeIncreased event raised by the LBPair contract.
type LBPairOracleSizeIncreased struct {
	PreviousSize *big.Int
	NewSize      *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterOracleSizeIncreased is a free log retrieval operation binding the contract event 0x525a4241308ea122822834c841f67b00d5efc977ad9118724750f974f7f6531c.
//
// Solidity: event OracleSizeIncreased(uint256 previousSize, uint256 newSize)
func (_LBPair *LBPairFilterer) FilterOracleSizeIncreased(opts *bind.FilterOpts) (*LBPairOracleSizeIncreasedIterator, error) {

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "OracleSizeIncreased")
	if err != nil {
		return nil, err
	}
	return &LBPairOracleSizeIncreasedIterator{contract: _LBPair.contract, event: "OracleSizeIncreased", logs: logs, sub: sub}, nil
}

// WatchOracleSizeIncreased is a free log subscription operation binding the contract event 0x525a4241308ea122822834c841f67b00d5efc977ad9118724750f974f7f6531c.
//
// Solidity: event OracleSizeIncreased(uint256 previousSize, uint256 newSize)
func (_LBPair *LBPairFilterer) WatchOracleSizeIncreased(opts *bind.WatchOpts, sink chan<- *LBPairOracleSizeIncreased) (event.Subscription, error) {

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "OracleSizeIncreased")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairOracleSizeIncreased)
				if err := _LBPair.contract.UnpackLog(event, "OracleSizeIncreased", log); err != nil {
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

// ParseOracleSizeIncreased is a log parse operation binding the contract event 0x525a4241308ea122822834c841f67b00d5efc977ad9118724750f974f7f6531c.
//
// Solidity: event OracleSizeIncreased(uint256 previousSize, uint256 newSize)
func (_LBPair *LBPairFilterer) ParseOracleSizeIncreased(log types.Log) (*LBPairOracleSizeIncreased, error) {
	event := new(LBPairOracleSizeIncreased)
	if err := _LBPair.contract.UnpackLog(event, "OracleSizeIncreased", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairProtocolFeesCollectedIterator is returned from FilterProtocolFeesCollected and is used to iterate over the raw logs and unpacked data for ProtocolFeesCollected events raised by the LBPair contract.
type LBPairProtocolFeesCollectedIterator struct {
	Event *LBPairProtocolFeesCollected // Event containing the contract specifics and raw log

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
func (it *LBPairProtocolFeesCollectedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairProtocolFeesCollected)
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
		it.Event = new(LBPairProtocolFeesCollected)
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
func (it *LBPairProtocolFeesCollectedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairProtocolFeesCollectedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairProtocolFeesCollected represents a ProtocolFeesCollected event raised by the LBPair contract.
type LBPairProtocolFeesCollected struct {
	Sender    common.Address
	Recipient common.Address
	AmountX   *big.Int
	AmountY   *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterProtocolFeesCollected is a free log retrieval operation binding the contract event 0x26b782206d6b531bf95d487110cfefdc443291f176f1977e94abcb7e67bd1b79.
//
// Solidity: event ProtocolFeesCollected(address indexed sender, address indexed recipient, uint256 amountX, uint256 amountY)
func (_LBPair *LBPairFilterer) FilterProtocolFeesCollected(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address) (*LBPairProtocolFeesCollectedIterator, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "ProtocolFeesCollected", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &LBPairProtocolFeesCollectedIterator{contract: _LBPair.contract, event: "ProtocolFeesCollected", logs: logs, sub: sub}, nil
}

// WatchProtocolFeesCollected is a free log subscription operation binding the contract event 0x26b782206d6b531bf95d487110cfefdc443291f176f1977e94abcb7e67bd1b79.
//
// Solidity: event ProtocolFeesCollected(address indexed sender, address indexed recipient, uint256 amountX, uint256 amountY)
func (_LBPair *LBPairFilterer) WatchProtocolFeesCollected(opts *bind.WatchOpts, sink chan<- *LBPairProtocolFeesCollected, sender []common.Address, recipient []common.Address) (event.Subscription, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "ProtocolFeesCollected", senderRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairProtocolFeesCollected)
				if err := _LBPair.contract.UnpackLog(event, "ProtocolFeesCollected", log); err != nil {
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

// ParseProtocolFeesCollected is a log parse operation binding the contract event 0x26b782206d6b531bf95d487110cfefdc443291f176f1977e94abcb7e67bd1b79.
//
// Solidity: event ProtocolFeesCollected(address indexed sender, address indexed recipient, uint256 amountX, uint256 amountY)
func (_LBPair *LBPairFilterer) ParseProtocolFeesCollected(log types.Log) (*LBPairProtocolFeesCollected, error) {
	event := new(LBPairProtocolFeesCollected)
	if err := _LBPair.contract.UnpackLog(event, "ProtocolFeesCollected", log); err != nil {
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
	Recipient             common.Address
	Id                    *big.Int
	SwapForY              bool
	AmountIn              *big.Int
	AmountOut             *big.Int
	VolatilityAccumulated *big.Int
	Fees                  *big.Int
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterSwap is a free log retrieval operation binding the contract event 0xc528cda9e500228b16ce84fadae290d9a49aecb17483110004c5af0a07f6fd73.
//
// Solidity: event Swap(address indexed sender, address indexed recipient, uint256 indexed id, bool swapForY, uint256 amountIn, uint256 amountOut, uint256 volatilityAccumulated, uint256 fees)
func (_LBPair *LBPairFilterer) FilterSwap(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address, id []*big.Int) (*LBPairSwapIterator, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "Swap", senderRule, recipientRule, idRule)
	if err != nil {
		return nil, err
	}
	return &LBPairSwapIterator{contract: _LBPair.contract, event: "Swap", logs: logs, sub: sub}, nil
}

// WatchSwap is a free log subscription operation binding the contract event 0xc528cda9e500228b16ce84fadae290d9a49aecb17483110004c5af0a07f6fd73.
//
// Solidity: event Swap(address indexed sender, address indexed recipient, uint256 indexed id, bool swapForY, uint256 amountIn, uint256 amountOut, uint256 volatilityAccumulated, uint256 fees)
func (_LBPair *LBPairFilterer) WatchSwap(opts *bind.WatchOpts, sink chan<- *LBPairSwap, sender []common.Address, recipient []common.Address, id []*big.Int) (event.Subscription, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "Swap", senderRule, recipientRule, idRule)
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

// ParseSwap is a log parse operation binding the contract event 0xc528cda9e500228b16ce84fadae290d9a49aecb17483110004c5af0a07f6fd73.
//
// Solidity: event Swap(address indexed sender, address indexed recipient, uint256 indexed id, bool swapForY, uint256 amountIn, uint256 amountOut, uint256 volatilityAccumulated, uint256 fees)
func (_LBPair *LBPairFilterer) ParseSwap(log types.Log) (*LBPairSwap, error) {
	event := new(LBPairSwap)
	if err := _LBPair.contract.UnpackLog(event, "Swap", log); err != nil {
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

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var fromRule []any
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []any
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

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var fromRule []any
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []any
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

// LBPairTransferSingleIterator is returned from FilterTransferSingle and is used to iterate over the raw logs and unpacked data for TransferSingle events raised by the LBPair contract.
type LBPairTransferSingleIterator struct {
	Event *LBPairTransferSingle // Event containing the contract specifics and raw log

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
func (it *LBPairTransferSingleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairTransferSingle)
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
		it.Event = new(LBPairTransferSingle)
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
func (it *LBPairTransferSingleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairTransferSingleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairTransferSingle represents a TransferSingle event raised by the LBPair contract.
type LBPairTransferSingle struct {
	Sender common.Address
	From   common.Address
	To     common.Address
	Id     *big.Int
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterTransferSingle is a free log retrieval operation binding the contract event 0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62.
//
// Solidity: event TransferSingle(address indexed sender, address indexed from, address indexed to, uint256 id, uint256 amount)
func (_LBPair *LBPairFilterer) FilterTransferSingle(opts *bind.FilterOpts, sender []common.Address, from []common.Address, to []common.Address) (*LBPairTransferSingleIterator, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var fromRule []any
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []any
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "TransferSingle", senderRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &LBPairTransferSingleIterator{contract: _LBPair.contract, event: "TransferSingle", logs: logs, sub: sub}, nil
}

// WatchTransferSingle is a free log subscription operation binding the contract event 0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62.
//
// Solidity: event TransferSingle(address indexed sender, address indexed from, address indexed to, uint256 id, uint256 amount)
func (_LBPair *LBPairFilterer) WatchTransferSingle(opts *bind.WatchOpts, sink chan<- *LBPairTransferSingle, sender []common.Address, from []common.Address, to []common.Address) (event.Subscription, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var fromRule []any
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []any
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "TransferSingle", senderRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairTransferSingle)
				if err := _LBPair.contract.UnpackLog(event, "TransferSingle", log); err != nil {
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

// ParseTransferSingle is a log parse operation binding the contract event 0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62.
//
// Solidity: event TransferSingle(address indexed sender, address indexed from, address indexed to, uint256 id, uint256 amount)
func (_LBPair *LBPairFilterer) ParseTransferSingle(log types.Log) (*LBPairTransferSingle, error) {
	event := new(LBPairTransferSingle)
	if err := _LBPair.contract.UnpackLog(event, "TransferSingle", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LBPairWithdrawnFromBinIterator is returned from FilterWithdrawnFromBin and is used to iterate over the raw logs and unpacked data for WithdrawnFromBin events raised by the LBPair contract.
type LBPairWithdrawnFromBinIterator struct {
	Event *LBPairWithdrawnFromBin // Event containing the contract specifics and raw log

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
func (it *LBPairWithdrawnFromBinIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LBPairWithdrawnFromBin)
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
		it.Event = new(LBPairWithdrawnFromBin)
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
func (it *LBPairWithdrawnFromBinIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LBPairWithdrawnFromBinIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LBPairWithdrawnFromBin represents a WithdrawnFromBin event raised by the LBPair contract.
type LBPairWithdrawnFromBin struct {
	Sender    common.Address
	Recipient common.Address
	Id        *big.Int
	AmountX   *big.Int
	AmountY   *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterWithdrawnFromBin is a free log retrieval operation binding the contract event 0xda5e7177dface55f5e0eff7dfc67420a1db4243ddfcf0ecc84ed93e034dd8cc2.
//
// Solidity: event WithdrawnFromBin(address indexed sender, address indexed recipient, uint256 indexed id, uint256 amountX, uint256 amountY)
func (_LBPair *LBPairFilterer) FilterWithdrawnFromBin(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address, id []*big.Int) (*LBPairWithdrawnFromBinIterator, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _LBPair.contract.FilterLogs(opts, "WithdrawnFromBin", senderRule, recipientRule, idRule)
	if err != nil {
		return nil, err
	}
	return &LBPairWithdrawnFromBinIterator{contract: _LBPair.contract, event: "WithdrawnFromBin", logs: logs, sub: sub}, nil
}

// WatchWithdrawnFromBin is a free log subscription operation binding the contract event 0xda5e7177dface55f5e0eff7dfc67420a1db4243ddfcf0ecc84ed93e034dd8cc2.
//
// Solidity: event WithdrawnFromBin(address indexed sender, address indexed recipient, uint256 indexed id, uint256 amountX, uint256 amountY)
func (_LBPair *LBPairFilterer) WatchWithdrawnFromBin(opts *bind.WatchOpts, sink chan<- *LBPairWithdrawnFromBin, sender []common.Address, recipient []common.Address, id []*big.Int) (event.Subscription, error) {

	var senderRule []any
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []any
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var idRule []any
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _LBPair.contract.WatchLogs(opts, "WithdrawnFromBin", senderRule, recipientRule, idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LBPairWithdrawnFromBin)
				if err := _LBPair.contract.UnpackLog(event, "WithdrawnFromBin", log); err != nil {
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

// ParseWithdrawnFromBin is a log parse operation binding the contract event 0xda5e7177dface55f5e0eff7dfc67420a1db4243ddfcf0ecc84ed93e034dd8cc2.
//
// Solidity: event WithdrawnFromBin(address indexed sender, address indexed recipient, uint256 indexed id, uint256 amountX, uint256 amountY)
func (_LBPair *LBPairFilterer) ParseWithdrawnFromBin(log types.Log) (*LBPairWithdrawnFromBin, error) {
	event := new(LBPairWithdrawnFromBin)
	if err := _LBPair.contract.UnpackLog(event, "WithdrawnFromBin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
