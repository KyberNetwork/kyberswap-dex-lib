// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package pandafun

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

// IPandaStructsPandaPoolParams is an auto generated low-level Go binding around an user-defined struct.
type IPandaStructsPandaPoolParams struct {
	BaseToken     common.Address
	SqrtPa        *big.Int
	SqrtPb        *big.Int
	VestingPeriod *big.Int
}

// PoolContractMetaData contains all meta data concerning the PoolContract contract.
var PoolContractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"InvalidShortString\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"str\",\"type\":\"string\"}],\"name\":\"StringTooLong\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"EIP712DomainChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"excessPandaTokens\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"excessBaseTokens\",\"type\":\"uint256\"}],\"name\":\"ExcessCollected\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountPanda\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountBase\",\"type\":\"uint256\"}],\"name\":\"LiquidityMoved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"pandaToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"sqrtPa\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"sqrtPb\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"vestingPeriod\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"deployer\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"PoolInitialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0In\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1In\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount0Out\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount1Out\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"Swap\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"pandaReserve\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"baseReserve\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"sqrtPrice\",\"type\":\"uint256\"}],\"name\":\"Sync\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"TokensClaimed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DOMAIN_SEPARATOR\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"GRADUATION_THRESHOLD\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"VERSION\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"baseReserve\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"baseToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"buyAllTokens\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountOut\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amountIn\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"minAmountOut\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"buyTokens\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountOut\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amountIn\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"minAmountOut\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"buyTokens\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountOut\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"minAmountOut\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"buyTokensWithBera\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountOut\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"canClaimIncentive\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"}],\"name\":\"claimTokens\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"}],\"name\":\"claimableTokens\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"collectExcessTokens\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"deployer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"dexFactory\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"dexPair\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"eip712Domain\",\"outputs\":[{\"internalType\":\"bytes1\",\"name\":\"fields\",\"type\":\"bytes1\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"version\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"verifyingContract\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"},{\"internalType\":\"uint256[]\",\"name\":\"extensions\",\"type\":\"uint256[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amountOut\",\"type\":\"uint256\"}],\"name\":\"getAmountInBuy\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountIn\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"sqrtP_new\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getAmountInBuyRemainingTokens\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountIn\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amountOut\",\"type\":\"uint256\"}],\"name\":\"getAmountInSell\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountIn\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"sqrtP_new\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amountIn\",\"type\":\"uint256\"}],\"name\":\"getAmountOutBuy\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountOut\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"sqrtP_new\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amountIn\",\"type\":\"uint256\"}],\"name\":\"getAmountOutSell\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountOut\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"sqrtP_new\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCurrentPrice\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_sqrtPa\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_sqrtPb\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_totalTokens\",\"type\":\"uint256\"},{\"internalType\":\"uint16\",\"name\":\"_graduationFee\",\"type\":\"uint16\"}],\"name\":\"getTokensInPool\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_sqrtPa\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_sqrtPb\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_tokensInPool\",\"type\":\"uint256\"}],\"name\":\"getTotalRaise\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalRaise\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"graduated\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"graduationTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_pandaToken\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"baseToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"sqrtPa\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"sqrtPb\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"vestingPeriod\",\"type\":\"uint256\"}],\"internalType\":\"structIPandaStructs.PandaPoolParams\",\"name\":\"pp\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"_totalTokens\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_deployer\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"initializePool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isPandaToken\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"liquidity\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minTradeSize\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"moveLiquidity\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"nonces\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pandaFactory\",\"outputs\":[{\"internalType\":\"contractIPandaFactory\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pandaReserve\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pandaToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"name\":\"permit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"poolFees\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"buyFee\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"sellFee\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"graduationFee\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"deployerFeeShare\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"remainingTokensInPool\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amountIn\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"minAmountOut\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"sellTokens\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountOut\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amountIn\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"minAmountOut\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"sellTokens\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountOut\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amountIn\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"minAmountOut\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"sellTokensForBera\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountOut\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sqrtP\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sqrtPa\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sqrtPb\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"tokensBoughtInPool\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"tokensClaimed\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"tokensForLp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"tokensInPool\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"}],\"name\":\"totalBalanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalRaise\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalTokens\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"treasury\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"}],\"name\":\"vestedBalanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"vestingPeriod\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"viewExcessTokens\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"excessPandaTokens\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"excessBaseTokens\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wbera\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
}

// PoolContractABI is the input ABI used to generate the binding from.
// Deprecated: Use PoolContractMetaData.ABI instead.
var PoolContractABI = PoolContractMetaData.ABI

// PoolContract is an auto generated Go binding around an Ethereum contract.
type PoolContract struct {
	PoolContractCaller     // Read-only binding to the contract
	PoolContractTransactor // Write-only binding to the contract
	PoolContractFilterer   // Log filterer for contract events
}

// PoolContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type PoolContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PoolContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PoolContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PoolContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PoolContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PoolContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PoolContractSession struct {
	Contract     *PoolContract     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PoolContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PoolContractCallerSession struct {
	Contract *PoolContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// PoolContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PoolContractTransactorSession struct {
	Contract     *PoolContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// PoolContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type PoolContractRaw struct {
	Contract *PoolContract // Generic contract binding to access the raw methods on
}

// PoolContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PoolContractCallerRaw struct {
	Contract *PoolContractCaller // Generic read-only contract binding to access the raw methods on
}

// PoolContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PoolContractTransactorRaw struct {
	Contract *PoolContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPoolContract creates a new instance of PoolContract, bound to a specific deployed contract.
func NewPoolContract(address common.Address, backend bind.ContractBackend) (*PoolContract, error) {
	contract, err := bindPoolContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PoolContract{PoolContractCaller: PoolContractCaller{contract: contract}, PoolContractTransactor: PoolContractTransactor{contract: contract}, PoolContractFilterer: PoolContractFilterer{contract: contract}}, nil
}

// NewPoolContractCaller creates a new read-only instance of PoolContract, bound to a specific deployed contract.
func NewPoolContractCaller(address common.Address, caller bind.ContractCaller) (*PoolContractCaller, error) {
	contract, err := bindPoolContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PoolContractCaller{contract: contract}, nil
}

// NewPoolContractTransactor creates a new write-only instance of PoolContract, bound to a specific deployed contract.
func NewPoolContractTransactor(address common.Address, transactor bind.ContractTransactor) (*PoolContractTransactor, error) {
	contract, err := bindPoolContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PoolContractTransactor{contract: contract}, nil
}

// NewPoolContractFilterer creates a new log filterer instance of PoolContract, bound to a specific deployed contract.
func NewPoolContractFilterer(address common.Address, filterer bind.ContractFilterer) (*PoolContractFilterer, error) {
	contract, err := bindPoolContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PoolContractFilterer{contract: contract}, nil
}

// bindPoolContract binds a generic wrapper to an already deployed contract.
func bindPoolContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := PoolContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PoolContract *PoolContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PoolContract.Contract.PoolContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PoolContract *PoolContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PoolContract.Contract.PoolContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PoolContract *PoolContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PoolContract.Contract.PoolContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PoolContract *PoolContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PoolContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PoolContract *PoolContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PoolContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PoolContract *PoolContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PoolContract.Contract.contract.Transact(opts, method, params...)
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_PoolContract *PoolContractCaller) DOMAINSEPARATOR(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "DOMAIN_SEPARATOR")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_PoolContract *PoolContractSession) DOMAINSEPARATOR() ([32]byte, error) {
	return _PoolContract.Contract.DOMAINSEPARATOR(&_PoolContract.CallOpts)
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_PoolContract *PoolContractCallerSession) DOMAINSEPARATOR() ([32]byte, error) {
	return _PoolContract.Contract.DOMAINSEPARATOR(&_PoolContract.CallOpts)
}

// GRADUATIONTHRESHOLD is a free data retrieval call binding the contract method 0xfcfc0c09.
//
// Solidity: function GRADUATION_THRESHOLD() view returns(uint256)
func (_PoolContract *PoolContractCaller) GRADUATIONTHRESHOLD(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "GRADUATION_THRESHOLD")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GRADUATIONTHRESHOLD is a free data retrieval call binding the contract method 0xfcfc0c09.
//
// Solidity: function GRADUATION_THRESHOLD() view returns(uint256)
func (_PoolContract *PoolContractSession) GRADUATIONTHRESHOLD() (*big.Int, error) {
	return _PoolContract.Contract.GRADUATIONTHRESHOLD(&_PoolContract.CallOpts)
}

// GRADUATIONTHRESHOLD is a free data retrieval call binding the contract method 0xfcfc0c09.
//
// Solidity: function GRADUATION_THRESHOLD() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) GRADUATIONTHRESHOLD() (*big.Int, error) {
	return _PoolContract.Contract.GRADUATIONTHRESHOLD(&_PoolContract.CallOpts)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() pure returns(string)
func (_PoolContract *PoolContractCaller) VERSION(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "VERSION")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() pure returns(string)
func (_PoolContract *PoolContractSession) VERSION() (string, error) {
	return _PoolContract.Contract.VERSION(&_PoolContract.CallOpts)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() pure returns(string)
func (_PoolContract *PoolContractCallerSession) VERSION() (string, error) {
	return _PoolContract.Contract.VERSION(&_PoolContract.CallOpts)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_PoolContract *PoolContractCaller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_PoolContract *PoolContractSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _PoolContract.Contract.Allowance(&_PoolContract.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_PoolContract *PoolContractCallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _PoolContract.Contract.Allowance(&_PoolContract.CallOpts, owner, spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_PoolContract *PoolContractCaller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_PoolContract *PoolContractSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _PoolContract.Contract.BalanceOf(&_PoolContract.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_PoolContract *PoolContractCallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _PoolContract.Contract.BalanceOf(&_PoolContract.CallOpts, account)
}

// BaseReserve is a free data retrieval call binding the contract method 0xdfdf2a72.
//
// Solidity: function baseReserve() view returns(uint256)
func (_PoolContract *PoolContractCaller) BaseReserve(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "baseReserve")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BaseReserve is a free data retrieval call binding the contract method 0xdfdf2a72.
//
// Solidity: function baseReserve() view returns(uint256)
func (_PoolContract *PoolContractSession) BaseReserve() (*big.Int, error) {
	return _PoolContract.Contract.BaseReserve(&_PoolContract.CallOpts)
}

// BaseReserve is a free data retrieval call binding the contract method 0xdfdf2a72.
//
// Solidity: function baseReserve() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) BaseReserve() (*big.Int, error) {
	return _PoolContract.Contract.BaseReserve(&_PoolContract.CallOpts)
}

// BaseToken is a free data retrieval call binding the contract method 0xc55dae63.
//
// Solidity: function baseToken() view returns(address)
func (_PoolContract *PoolContractCaller) BaseToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "baseToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BaseToken is a free data retrieval call binding the contract method 0xc55dae63.
//
// Solidity: function baseToken() view returns(address)
func (_PoolContract *PoolContractSession) BaseToken() (common.Address, error) {
	return _PoolContract.Contract.BaseToken(&_PoolContract.CallOpts)
}

// BaseToken is a free data retrieval call binding the contract method 0xc55dae63.
//
// Solidity: function baseToken() view returns(address)
func (_PoolContract *PoolContractCallerSession) BaseToken() (common.Address, error) {
	return _PoolContract.Contract.BaseToken(&_PoolContract.CallOpts)
}

// CanClaimIncentive is a free data retrieval call binding the contract method 0x295d9127.
//
// Solidity: function canClaimIncentive() view returns(bool)
func (_PoolContract *PoolContractCaller) CanClaimIncentive(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "canClaimIncentive")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// CanClaimIncentive is a free data retrieval call binding the contract method 0x295d9127.
//
// Solidity: function canClaimIncentive() view returns(bool)
func (_PoolContract *PoolContractSession) CanClaimIncentive() (bool, error) {
	return _PoolContract.Contract.CanClaimIncentive(&_PoolContract.CallOpts)
}

// CanClaimIncentive is a free data retrieval call binding the contract method 0x295d9127.
//
// Solidity: function canClaimIncentive() view returns(bool)
func (_PoolContract *PoolContractCallerSession) CanClaimIncentive() (bool, error) {
	return _PoolContract.Contract.CanClaimIncentive(&_PoolContract.CallOpts)
}

// ClaimableTokens is a free data retrieval call binding the contract method 0x84d24226.
//
// Solidity: function claimableTokens(address user) view returns(uint256)
func (_PoolContract *PoolContractCaller) ClaimableTokens(opts *bind.CallOpts, user common.Address) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "claimableTokens", user)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ClaimableTokens is a free data retrieval call binding the contract method 0x84d24226.
//
// Solidity: function claimableTokens(address user) view returns(uint256)
func (_PoolContract *PoolContractSession) ClaimableTokens(user common.Address) (*big.Int, error) {
	return _PoolContract.Contract.ClaimableTokens(&_PoolContract.CallOpts, user)
}

// ClaimableTokens is a free data retrieval call binding the contract method 0x84d24226.
//
// Solidity: function claimableTokens(address user) view returns(uint256)
func (_PoolContract *PoolContractCallerSession) ClaimableTokens(user common.Address) (*big.Int, error) {
	return _PoolContract.Contract.ClaimableTokens(&_PoolContract.CallOpts, user)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_PoolContract *PoolContractCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_PoolContract *PoolContractSession) Decimals() (uint8, error) {
	return _PoolContract.Contract.Decimals(&_PoolContract.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_PoolContract *PoolContractCallerSession) Decimals() (uint8, error) {
	return _PoolContract.Contract.Decimals(&_PoolContract.CallOpts)
}

// Deployer is a free data retrieval call binding the contract method 0xd5f39488.
//
// Solidity: function deployer() view returns(address)
func (_PoolContract *PoolContractCaller) Deployer(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "deployer")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Deployer is a free data retrieval call binding the contract method 0xd5f39488.
//
// Solidity: function deployer() view returns(address)
func (_PoolContract *PoolContractSession) Deployer() (common.Address, error) {
	return _PoolContract.Contract.Deployer(&_PoolContract.CallOpts)
}

// Deployer is a free data retrieval call binding the contract method 0xd5f39488.
//
// Solidity: function deployer() view returns(address)
func (_PoolContract *PoolContractCallerSession) Deployer() (common.Address, error) {
	return _PoolContract.Contract.Deployer(&_PoolContract.CallOpts)
}

// DexFactory is a free data retrieval call binding the contract method 0xb8d8fbb4.
//
// Solidity: function dexFactory() view returns(address)
func (_PoolContract *PoolContractCaller) DexFactory(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "dexFactory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DexFactory is a free data retrieval call binding the contract method 0xb8d8fbb4.
//
// Solidity: function dexFactory() view returns(address)
func (_PoolContract *PoolContractSession) DexFactory() (common.Address, error) {
	return _PoolContract.Contract.DexFactory(&_PoolContract.CallOpts)
}

// DexFactory is a free data retrieval call binding the contract method 0xb8d8fbb4.
//
// Solidity: function dexFactory() view returns(address)
func (_PoolContract *PoolContractCallerSession) DexFactory() (common.Address, error) {
	return _PoolContract.Contract.DexFactory(&_PoolContract.CallOpts)
}

// DexPair is a free data retrieval call binding the contract method 0xf242ab41.
//
// Solidity: function dexPair() view returns(address)
func (_PoolContract *PoolContractCaller) DexPair(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "dexPair")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DexPair is a free data retrieval call binding the contract method 0xf242ab41.
//
// Solidity: function dexPair() view returns(address)
func (_PoolContract *PoolContractSession) DexPair() (common.Address, error) {
	return _PoolContract.Contract.DexPair(&_PoolContract.CallOpts)
}

// DexPair is a free data retrieval call binding the contract method 0xf242ab41.
//
// Solidity: function dexPair() view returns(address)
func (_PoolContract *PoolContractCallerSession) DexPair() (common.Address, error) {
	return _PoolContract.Contract.DexPair(&_PoolContract.CallOpts)
}

// Eip712Domain is a free data retrieval call binding the contract method 0x84b0196e.
//
// Solidity: function eip712Domain() view returns(bytes1 fields, string name, string version, uint256 chainId, address verifyingContract, bytes32 salt, uint256[] extensions)
func (_PoolContract *PoolContractCaller) Eip712Domain(opts *bind.CallOpts) (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "eip712Domain")

	outstruct := new(struct {
		Fields            [1]byte
		Name              string
		Version           string
		ChainId           *big.Int
		VerifyingContract common.Address
		Salt              [32]byte
		Extensions        []*big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Fields = *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)
	outstruct.Name = *abi.ConvertType(out[1], new(string)).(*string)
	outstruct.Version = *abi.ConvertType(out[2], new(string)).(*string)
	outstruct.ChainId = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.VerifyingContract = *abi.ConvertType(out[4], new(common.Address)).(*common.Address)
	outstruct.Salt = *abi.ConvertType(out[5], new([32]byte)).(*[32]byte)
	outstruct.Extensions = *abi.ConvertType(out[6], new([]*big.Int)).(*[]*big.Int)

	return *outstruct, err

}

// Eip712Domain is a free data retrieval call binding the contract method 0x84b0196e.
//
// Solidity: function eip712Domain() view returns(bytes1 fields, string name, string version, uint256 chainId, address verifyingContract, bytes32 salt, uint256[] extensions)
func (_PoolContract *PoolContractSession) Eip712Domain() (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	return _PoolContract.Contract.Eip712Domain(&_PoolContract.CallOpts)
}

// Eip712Domain is a free data retrieval call binding the contract method 0x84b0196e.
//
// Solidity: function eip712Domain() view returns(bytes1 fields, string name, string version, uint256 chainId, address verifyingContract, bytes32 salt, uint256[] extensions)
func (_PoolContract *PoolContractCallerSession) Eip712Domain() (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	return _PoolContract.Contract.Eip712Domain(&_PoolContract.CallOpts)
}

// GetAmountInBuy is a free data retrieval call binding the contract method 0x091be0ca.
//
// Solidity: function getAmountInBuy(uint256 amountOut) view returns(uint256 amountIn, uint256 fee, uint256 sqrtP_new)
func (_PoolContract *PoolContractCaller) GetAmountInBuy(opts *bind.CallOpts, amountOut *big.Int) (struct {
	AmountIn *big.Int
	Fee      *big.Int
	SqrtPNew *big.Int
}, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "getAmountInBuy", amountOut)

	outstruct := new(struct {
		AmountIn *big.Int
		Fee      *big.Int
		SqrtPNew *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.AmountIn = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Fee = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.SqrtPNew = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetAmountInBuy is a free data retrieval call binding the contract method 0x091be0ca.
//
// Solidity: function getAmountInBuy(uint256 amountOut) view returns(uint256 amountIn, uint256 fee, uint256 sqrtP_new)
func (_PoolContract *PoolContractSession) GetAmountInBuy(amountOut *big.Int) (struct {
	AmountIn *big.Int
	Fee      *big.Int
	SqrtPNew *big.Int
}, error) {
	return _PoolContract.Contract.GetAmountInBuy(&_PoolContract.CallOpts, amountOut)
}

// GetAmountInBuy is a free data retrieval call binding the contract method 0x091be0ca.
//
// Solidity: function getAmountInBuy(uint256 amountOut) view returns(uint256 amountIn, uint256 fee, uint256 sqrtP_new)
func (_PoolContract *PoolContractCallerSession) GetAmountInBuy(amountOut *big.Int) (struct {
	AmountIn *big.Int
	Fee      *big.Int
	SqrtPNew *big.Int
}, error) {
	return _PoolContract.Contract.GetAmountInBuy(&_PoolContract.CallOpts, amountOut)
}

// GetAmountInBuyRemainingTokens is a free data retrieval call binding the contract method 0xf49bb024.
//
// Solidity: function getAmountInBuyRemainingTokens() view returns(uint256 amountIn)
func (_PoolContract *PoolContractCaller) GetAmountInBuyRemainingTokens(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "getAmountInBuyRemainingTokens")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetAmountInBuyRemainingTokens is a free data retrieval call binding the contract method 0xf49bb024.
//
// Solidity: function getAmountInBuyRemainingTokens() view returns(uint256 amountIn)
func (_PoolContract *PoolContractSession) GetAmountInBuyRemainingTokens() (*big.Int, error) {
	return _PoolContract.Contract.GetAmountInBuyRemainingTokens(&_PoolContract.CallOpts)
}

// GetAmountInBuyRemainingTokens is a free data retrieval call binding the contract method 0xf49bb024.
//
// Solidity: function getAmountInBuyRemainingTokens() view returns(uint256 amountIn)
func (_PoolContract *PoolContractCallerSession) GetAmountInBuyRemainingTokens() (*big.Int, error) {
	return _PoolContract.Contract.GetAmountInBuyRemainingTokens(&_PoolContract.CallOpts)
}

// GetAmountInSell is a free data retrieval call binding the contract method 0x6021c6ba.
//
// Solidity: function getAmountInSell(uint256 amountOut) view returns(uint256 amountIn, uint256 fee, uint256 sqrtP_new)
func (_PoolContract *PoolContractCaller) GetAmountInSell(opts *bind.CallOpts, amountOut *big.Int) (struct {
	AmountIn *big.Int
	Fee      *big.Int
	SqrtPNew *big.Int
}, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "getAmountInSell", amountOut)

	outstruct := new(struct {
		AmountIn *big.Int
		Fee      *big.Int
		SqrtPNew *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.AmountIn = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Fee = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.SqrtPNew = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetAmountInSell is a free data retrieval call binding the contract method 0x6021c6ba.
//
// Solidity: function getAmountInSell(uint256 amountOut) view returns(uint256 amountIn, uint256 fee, uint256 sqrtP_new)
func (_PoolContract *PoolContractSession) GetAmountInSell(amountOut *big.Int) (struct {
	AmountIn *big.Int
	Fee      *big.Int
	SqrtPNew *big.Int
}, error) {
	return _PoolContract.Contract.GetAmountInSell(&_PoolContract.CallOpts, amountOut)
}

// GetAmountInSell is a free data retrieval call binding the contract method 0x6021c6ba.
//
// Solidity: function getAmountInSell(uint256 amountOut) view returns(uint256 amountIn, uint256 fee, uint256 sqrtP_new)
func (_PoolContract *PoolContractCallerSession) GetAmountInSell(amountOut *big.Int) (struct {
	AmountIn *big.Int
	Fee      *big.Int
	SqrtPNew *big.Int
}, error) {
	return _PoolContract.Contract.GetAmountInSell(&_PoolContract.CallOpts, amountOut)
}

// GetAmountOutBuy is a free data retrieval call binding the contract method 0x49717a72.
//
// Solidity: function getAmountOutBuy(uint256 amountIn) view returns(uint256 amountOut, uint256 fee, uint256 sqrtP_new)
func (_PoolContract *PoolContractCaller) GetAmountOutBuy(opts *bind.CallOpts, amountIn *big.Int) (struct {
	AmountOut *big.Int
	Fee       *big.Int
	SqrtPNew  *big.Int
}, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "getAmountOutBuy", amountIn)

	outstruct := new(struct {
		AmountOut *big.Int
		Fee       *big.Int
		SqrtPNew  *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.AmountOut = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Fee = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.SqrtPNew = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetAmountOutBuy is a free data retrieval call binding the contract method 0x49717a72.
//
// Solidity: function getAmountOutBuy(uint256 amountIn) view returns(uint256 amountOut, uint256 fee, uint256 sqrtP_new)
func (_PoolContract *PoolContractSession) GetAmountOutBuy(amountIn *big.Int) (struct {
	AmountOut *big.Int
	Fee       *big.Int
	SqrtPNew  *big.Int
}, error) {
	return _PoolContract.Contract.GetAmountOutBuy(&_PoolContract.CallOpts, amountIn)
}

// GetAmountOutBuy is a free data retrieval call binding the contract method 0x49717a72.
//
// Solidity: function getAmountOutBuy(uint256 amountIn) view returns(uint256 amountOut, uint256 fee, uint256 sqrtP_new)
func (_PoolContract *PoolContractCallerSession) GetAmountOutBuy(amountIn *big.Int) (struct {
	AmountOut *big.Int
	Fee       *big.Int
	SqrtPNew  *big.Int
}, error) {
	return _PoolContract.Contract.GetAmountOutBuy(&_PoolContract.CallOpts, amountIn)
}

// GetAmountOutSell is a free data retrieval call binding the contract method 0x17e79db0.
//
// Solidity: function getAmountOutSell(uint256 amountIn) view returns(uint256 amountOut, uint256 fee, uint256 sqrtP_new)
func (_PoolContract *PoolContractCaller) GetAmountOutSell(opts *bind.CallOpts, amountIn *big.Int) (struct {
	AmountOut *big.Int
	Fee       *big.Int
	SqrtPNew  *big.Int
}, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "getAmountOutSell", amountIn)

	outstruct := new(struct {
		AmountOut *big.Int
		Fee       *big.Int
		SqrtPNew  *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.AmountOut = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Fee = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.SqrtPNew = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetAmountOutSell is a free data retrieval call binding the contract method 0x17e79db0.
//
// Solidity: function getAmountOutSell(uint256 amountIn) view returns(uint256 amountOut, uint256 fee, uint256 sqrtP_new)
func (_PoolContract *PoolContractSession) GetAmountOutSell(amountIn *big.Int) (struct {
	AmountOut *big.Int
	Fee       *big.Int
	SqrtPNew  *big.Int
}, error) {
	return _PoolContract.Contract.GetAmountOutSell(&_PoolContract.CallOpts, amountIn)
}

// GetAmountOutSell is a free data retrieval call binding the contract method 0x17e79db0.
//
// Solidity: function getAmountOutSell(uint256 amountIn) view returns(uint256 amountOut, uint256 fee, uint256 sqrtP_new)
func (_PoolContract *PoolContractCallerSession) GetAmountOutSell(amountIn *big.Int) (struct {
	AmountOut *big.Int
	Fee       *big.Int
	SqrtPNew  *big.Int
}, error) {
	return _PoolContract.Contract.GetAmountOutSell(&_PoolContract.CallOpts, amountIn)
}

// GetCurrentPrice is a free data retrieval call binding the contract method 0xeb91d37e.
//
// Solidity: function getCurrentPrice() view returns(uint256)
func (_PoolContract *PoolContractCaller) GetCurrentPrice(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "getCurrentPrice")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrentPrice is a free data retrieval call binding the contract method 0xeb91d37e.
//
// Solidity: function getCurrentPrice() view returns(uint256)
func (_PoolContract *PoolContractSession) GetCurrentPrice() (*big.Int, error) {
	return _PoolContract.Contract.GetCurrentPrice(&_PoolContract.CallOpts)
}

// GetCurrentPrice is a free data retrieval call binding the contract method 0xeb91d37e.
//
// Solidity: function getCurrentPrice() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) GetCurrentPrice() (*big.Int, error) {
	return _PoolContract.Contract.GetCurrentPrice(&_PoolContract.CallOpts)
}

// GetTokensInPool is a free data retrieval call binding the contract method 0x8aab0a09.
//
// Solidity: function getTokensInPool(uint256 _sqrtPa, uint256 _sqrtPb, uint256 _totalTokens, uint16 _graduationFee) view returns(uint256)
func (_PoolContract *PoolContractCaller) GetTokensInPool(opts *bind.CallOpts, _sqrtPa *big.Int, _sqrtPb *big.Int, _totalTokens *big.Int, _graduationFee uint16) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "getTokensInPool", _sqrtPa, _sqrtPb, _totalTokens, _graduationFee)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTokensInPool is a free data retrieval call binding the contract method 0x8aab0a09.
//
// Solidity: function getTokensInPool(uint256 _sqrtPa, uint256 _sqrtPb, uint256 _totalTokens, uint16 _graduationFee) view returns(uint256)
func (_PoolContract *PoolContractSession) GetTokensInPool(_sqrtPa *big.Int, _sqrtPb *big.Int, _totalTokens *big.Int, _graduationFee uint16) (*big.Int, error) {
	return _PoolContract.Contract.GetTokensInPool(&_PoolContract.CallOpts, _sqrtPa, _sqrtPb, _totalTokens, _graduationFee)
}

// GetTokensInPool is a free data retrieval call binding the contract method 0x8aab0a09.
//
// Solidity: function getTokensInPool(uint256 _sqrtPa, uint256 _sqrtPb, uint256 _totalTokens, uint16 _graduationFee) view returns(uint256)
func (_PoolContract *PoolContractCallerSession) GetTokensInPool(_sqrtPa *big.Int, _sqrtPb *big.Int, _totalTokens *big.Int, _graduationFee uint16) (*big.Int, error) {
	return _PoolContract.Contract.GetTokensInPool(&_PoolContract.CallOpts, _sqrtPa, _sqrtPb, _totalTokens, _graduationFee)
}

// GetTotalRaise is a free data retrieval call binding the contract method 0xa913f1c1.
//
// Solidity: function getTotalRaise(uint256 _sqrtPa, uint256 _sqrtPb, uint256 _tokensInPool) view returns(uint256)
func (_PoolContract *PoolContractCaller) GetTotalRaise(opts *bind.CallOpts, _sqrtPa *big.Int, _sqrtPb *big.Int, _tokensInPool *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "getTotalRaise", _sqrtPa, _sqrtPb, _tokensInPool)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTotalRaise is a free data retrieval call binding the contract method 0xa913f1c1.
//
// Solidity: function getTotalRaise(uint256 _sqrtPa, uint256 _sqrtPb, uint256 _tokensInPool) view returns(uint256)
func (_PoolContract *PoolContractSession) GetTotalRaise(_sqrtPa *big.Int, _sqrtPb *big.Int, _tokensInPool *big.Int) (*big.Int, error) {
	return _PoolContract.Contract.GetTotalRaise(&_PoolContract.CallOpts, _sqrtPa, _sqrtPb, _tokensInPool)
}

// GetTotalRaise is a free data retrieval call binding the contract method 0xa913f1c1.
//
// Solidity: function getTotalRaise(uint256 _sqrtPa, uint256 _sqrtPb, uint256 _tokensInPool) view returns(uint256)
func (_PoolContract *PoolContractCallerSession) GetTotalRaise(_sqrtPa *big.Int, _sqrtPb *big.Int, _tokensInPool *big.Int) (*big.Int, error) {
	return _PoolContract.Contract.GetTotalRaise(&_PoolContract.CallOpts, _sqrtPa, _sqrtPb, _tokensInPool)
}

// GetTotalRaise0 is a free data retrieval call binding the contract method 0xadfee4c1.
//
// Solidity: function getTotalRaise() view returns(uint256)
func (_PoolContract *PoolContractCaller) GetTotalRaise0(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "getTotalRaise0")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTotalRaise0 is a free data retrieval call binding the contract method 0xadfee4c1.
//
// Solidity: function getTotalRaise() view returns(uint256)
func (_PoolContract *PoolContractSession) GetTotalRaise0() (*big.Int, error) {
	return _PoolContract.Contract.GetTotalRaise0(&_PoolContract.CallOpts)
}

// GetTotalRaise0 is a free data retrieval call binding the contract method 0xadfee4c1.
//
// Solidity: function getTotalRaise() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) GetTotalRaise0() (*big.Int, error) {
	return _PoolContract.Contract.GetTotalRaise0(&_PoolContract.CallOpts)
}

// Graduated is a free data retrieval call binding the contract method 0xe7c2b772.
//
// Solidity: function graduated() view returns(bool)
func (_PoolContract *PoolContractCaller) Graduated(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "graduated")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Graduated is a free data retrieval call binding the contract method 0xe7c2b772.
//
// Solidity: function graduated() view returns(bool)
func (_PoolContract *PoolContractSession) Graduated() (bool, error) {
	return _PoolContract.Contract.Graduated(&_PoolContract.CallOpts)
}

// Graduated is a free data retrieval call binding the contract method 0xe7c2b772.
//
// Solidity: function graduated() view returns(bool)
func (_PoolContract *PoolContractCallerSession) Graduated() (bool, error) {
	return _PoolContract.Contract.Graduated(&_PoolContract.CallOpts)
}

// GraduationTime is a free data retrieval call binding the contract method 0x57f3b5f4.
//
// Solidity: function graduationTime() view returns(uint256)
func (_PoolContract *PoolContractCaller) GraduationTime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "graduationTime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GraduationTime is a free data retrieval call binding the contract method 0x57f3b5f4.
//
// Solidity: function graduationTime() view returns(uint256)
func (_PoolContract *PoolContractSession) GraduationTime() (*big.Int, error) {
	return _PoolContract.Contract.GraduationTime(&_PoolContract.CallOpts)
}

// GraduationTime is a free data retrieval call binding the contract method 0x57f3b5f4.
//
// Solidity: function graduationTime() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) GraduationTime() (*big.Int, error) {
	return _PoolContract.Contract.GraduationTime(&_PoolContract.CallOpts)
}

// IsPandaToken is a free data retrieval call binding the contract method 0x251965c4.
//
// Solidity: function isPandaToken() view returns(bool)
func (_PoolContract *PoolContractCaller) IsPandaToken(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "isPandaToken")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsPandaToken is a free data retrieval call binding the contract method 0x251965c4.
//
// Solidity: function isPandaToken() view returns(bool)
func (_PoolContract *PoolContractSession) IsPandaToken() (bool, error) {
	return _PoolContract.Contract.IsPandaToken(&_PoolContract.CallOpts)
}

// IsPandaToken is a free data retrieval call binding the contract method 0x251965c4.
//
// Solidity: function isPandaToken() view returns(bool)
func (_PoolContract *PoolContractCallerSession) IsPandaToken() (bool, error) {
	return _PoolContract.Contract.IsPandaToken(&_PoolContract.CallOpts)
}

// Liquidity is a free data retrieval call binding the contract method 0x1a686502.
//
// Solidity: function liquidity() view returns(uint256)
func (_PoolContract *PoolContractCaller) Liquidity(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "liquidity")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Liquidity is a free data retrieval call binding the contract method 0x1a686502.
//
// Solidity: function liquidity() view returns(uint256)
func (_PoolContract *PoolContractSession) Liquidity() (*big.Int, error) {
	return _PoolContract.Contract.Liquidity(&_PoolContract.CallOpts)
}

// Liquidity is a free data retrieval call binding the contract method 0x1a686502.
//
// Solidity: function liquidity() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) Liquidity() (*big.Int, error) {
	return _PoolContract.Contract.Liquidity(&_PoolContract.CallOpts)
}

// MinTradeSize is a free data retrieval call binding the contract method 0x6155be59.
//
// Solidity: function minTradeSize() view returns(uint256)
func (_PoolContract *PoolContractCaller) MinTradeSize(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "minTradeSize")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinTradeSize is a free data retrieval call binding the contract method 0x6155be59.
//
// Solidity: function minTradeSize() view returns(uint256)
func (_PoolContract *PoolContractSession) MinTradeSize() (*big.Int, error) {
	return _PoolContract.Contract.MinTradeSize(&_PoolContract.CallOpts)
}

// MinTradeSize is a free data retrieval call binding the contract method 0x6155be59.
//
// Solidity: function minTradeSize() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) MinTradeSize() (*big.Int, error) {
	return _PoolContract.Contract.MinTradeSize(&_PoolContract.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_PoolContract *PoolContractCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_PoolContract *PoolContractSession) Name() (string, error) {
	return _PoolContract.Contract.Name(&_PoolContract.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_PoolContract *PoolContractCallerSession) Name() (string, error) {
	return _PoolContract.Contract.Name(&_PoolContract.CallOpts)
}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces(address owner) view returns(uint256)
func (_PoolContract *PoolContractCaller) Nonces(opts *bind.CallOpts, owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "nonces", owner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces(address owner) view returns(uint256)
func (_PoolContract *PoolContractSession) Nonces(owner common.Address) (*big.Int, error) {
	return _PoolContract.Contract.Nonces(&_PoolContract.CallOpts, owner)
}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces(address owner) view returns(uint256)
func (_PoolContract *PoolContractCallerSession) Nonces(owner common.Address) (*big.Int, error) {
	return _PoolContract.Contract.Nonces(&_PoolContract.CallOpts, owner)
}

// PandaFactory is a free data retrieval call binding the contract method 0xb46c69e2.
//
// Solidity: function pandaFactory() view returns(address)
func (_PoolContract *PoolContractCaller) PandaFactory(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "pandaFactory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PandaFactory is a free data retrieval call binding the contract method 0xb46c69e2.
//
// Solidity: function pandaFactory() view returns(address)
func (_PoolContract *PoolContractSession) PandaFactory() (common.Address, error) {
	return _PoolContract.Contract.PandaFactory(&_PoolContract.CallOpts)
}

// PandaFactory is a free data retrieval call binding the contract method 0xb46c69e2.
//
// Solidity: function pandaFactory() view returns(address)
func (_PoolContract *PoolContractCallerSession) PandaFactory() (common.Address, error) {
	return _PoolContract.Contract.PandaFactory(&_PoolContract.CallOpts)
}

// PandaReserve is a free data retrieval call binding the contract method 0xf232fe73.
//
// Solidity: function pandaReserve() view returns(uint256)
func (_PoolContract *PoolContractCaller) PandaReserve(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "pandaReserve")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PandaReserve is a free data retrieval call binding the contract method 0xf232fe73.
//
// Solidity: function pandaReserve() view returns(uint256)
func (_PoolContract *PoolContractSession) PandaReserve() (*big.Int, error) {
	return _PoolContract.Contract.PandaReserve(&_PoolContract.CallOpts)
}

// PandaReserve is a free data retrieval call binding the contract method 0xf232fe73.
//
// Solidity: function pandaReserve() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) PandaReserve() (*big.Int, error) {
	return _PoolContract.Contract.PandaReserve(&_PoolContract.CallOpts)
}

// PandaToken is a free data retrieval call binding the contract method 0xe7caf3ae.
//
// Solidity: function pandaToken() view returns(address)
func (_PoolContract *PoolContractCaller) PandaToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "pandaToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PandaToken is a free data retrieval call binding the contract method 0xe7caf3ae.
//
// Solidity: function pandaToken() view returns(address)
func (_PoolContract *PoolContractSession) PandaToken() (common.Address, error) {
	return _PoolContract.Contract.PandaToken(&_PoolContract.CallOpts)
}

// PandaToken is a free data retrieval call binding the contract method 0xe7caf3ae.
//
// Solidity: function pandaToken() view returns(address)
func (_PoolContract *PoolContractCallerSession) PandaToken() (common.Address, error) {
	return _PoolContract.Contract.PandaToken(&_PoolContract.CallOpts)
}

// PoolFees is a free data retrieval call binding the contract method 0x33580959.
//
// Solidity: function poolFees() view returns(uint16 buyFee, uint16 sellFee, uint16 graduationFee, uint16 deployerFeeShare)
func (_PoolContract *PoolContractCaller) PoolFees(opts *bind.CallOpts) (struct {
	BuyFee           uint16
	SellFee          uint16
	GraduationFee    uint16
	DeployerFeeShare uint16
}, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "poolFees")

	outstruct := new(struct {
		BuyFee           uint16
		SellFee          uint16
		GraduationFee    uint16
		DeployerFeeShare uint16
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.BuyFee = *abi.ConvertType(out[0], new(uint16)).(*uint16)
	outstruct.SellFee = *abi.ConvertType(out[1], new(uint16)).(*uint16)
	outstruct.GraduationFee = *abi.ConvertType(out[2], new(uint16)).(*uint16)
	outstruct.DeployerFeeShare = *abi.ConvertType(out[3], new(uint16)).(*uint16)

	return *outstruct, err

}

// PoolFees is a free data retrieval call binding the contract method 0x33580959.
//
// Solidity: function poolFees() view returns(uint16 buyFee, uint16 sellFee, uint16 graduationFee, uint16 deployerFeeShare)
func (_PoolContract *PoolContractSession) PoolFees() (struct {
	BuyFee           uint16
	SellFee          uint16
	GraduationFee    uint16
	DeployerFeeShare uint16
}, error) {
	return _PoolContract.Contract.PoolFees(&_PoolContract.CallOpts)
}

// PoolFees is a free data retrieval call binding the contract method 0x33580959.
//
// Solidity: function poolFees() view returns(uint16 buyFee, uint16 sellFee, uint16 graduationFee, uint16 deployerFeeShare)
func (_PoolContract *PoolContractCallerSession) PoolFees() (struct {
	BuyFee           uint16
	SellFee          uint16
	GraduationFee    uint16
	DeployerFeeShare uint16
}, error) {
	return _PoolContract.Contract.PoolFees(&_PoolContract.CallOpts)
}

// RemainingTokensInPool is a free data retrieval call binding the contract method 0x44badeca.
//
// Solidity: function remainingTokensInPool() view returns(uint256)
func (_PoolContract *PoolContractCaller) RemainingTokensInPool(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "remainingTokensInPool")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// RemainingTokensInPool is a free data retrieval call binding the contract method 0x44badeca.
//
// Solidity: function remainingTokensInPool() view returns(uint256)
func (_PoolContract *PoolContractSession) RemainingTokensInPool() (*big.Int, error) {
	return _PoolContract.Contract.RemainingTokensInPool(&_PoolContract.CallOpts)
}

// RemainingTokensInPool is a free data retrieval call binding the contract method 0x44badeca.
//
// Solidity: function remainingTokensInPool() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) RemainingTokensInPool() (*big.Int, error) {
	return _PoolContract.Contract.RemainingTokensInPool(&_PoolContract.CallOpts)
}

// SqrtP is a free data retrieval call binding the contract method 0x33d5e549.
//
// Solidity: function sqrtP() view returns(uint256)
func (_PoolContract *PoolContractCaller) SqrtP(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "sqrtP")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SqrtP is a free data retrieval call binding the contract method 0x33d5e549.
//
// Solidity: function sqrtP() view returns(uint256)
func (_PoolContract *PoolContractSession) SqrtP() (*big.Int, error) {
	return _PoolContract.Contract.SqrtP(&_PoolContract.CallOpts)
}

// SqrtP is a free data retrieval call binding the contract method 0x33d5e549.
//
// Solidity: function sqrtP() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) SqrtP() (*big.Int, error) {
	return _PoolContract.Contract.SqrtP(&_PoolContract.CallOpts)
}

// SqrtPa is a free data retrieval call binding the contract method 0xf2505c52.
//
// Solidity: function sqrtPa() view returns(uint256)
func (_PoolContract *PoolContractCaller) SqrtPa(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "sqrtPa")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SqrtPa is a free data retrieval call binding the contract method 0xf2505c52.
//
// Solidity: function sqrtPa() view returns(uint256)
func (_PoolContract *PoolContractSession) SqrtPa() (*big.Int, error) {
	return _PoolContract.Contract.SqrtPa(&_PoolContract.CallOpts)
}

// SqrtPa is a free data retrieval call binding the contract method 0xf2505c52.
//
// Solidity: function sqrtPa() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) SqrtPa() (*big.Int, error) {
	return _PoolContract.Contract.SqrtPa(&_PoolContract.CallOpts)
}

// SqrtPb is a free data retrieval call binding the contract method 0xc8b8c2a6.
//
// Solidity: function sqrtPb() view returns(uint256)
func (_PoolContract *PoolContractCaller) SqrtPb(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "sqrtPb")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SqrtPb is a free data retrieval call binding the contract method 0xc8b8c2a6.
//
// Solidity: function sqrtPb() view returns(uint256)
func (_PoolContract *PoolContractSession) SqrtPb() (*big.Int, error) {
	return _PoolContract.Contract.SqrtPb(&_PoolContract.CallOpts)
}

// SqrtPb is a free data retrieval call binding the contract method 0xc8b8c2a6.
//
// Solidity: function sqrtPb() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) SqrtPb() (*big.Int, error) {
	return _PoolContract.Contract.SqrtPb(&_PoolContract.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_PoolContract *PoolContractCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_PoolContract *PoolContractSession) Symbol() (string, error) {
	return _PoolContract.Contract.Symbol(&_PoolContract.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_PoolContract *PoolContractCallerSession) Symbol() (string, error) {
	return _PoolContract.Contract.Symbol(&_PoolContract.CallOpts)
}

// TokensBoughtInPool is a free data retrieval call binding the contract method 0x9b6f705b.
//
// Solidity: function tokensBoughtInPool(address ) view returns(uint256)
func (_PoolContract *PoolContractCaller) TokensBoughtInPool(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "tokensBoughtInPool", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TokensBoughtInPool is a free data retrieval call binding the contract method 0x9b6f705b.
//
// Solidity: function tokensBoughtInPool(address ) view returns(uint256)
func (_PoolContract *PoolContractSession) TokensBoughtInPool(arg0 common.Address) (*big.Int, error) {
	return _PoolContract.Contract.TokensBoughtInPool(&_PoolContract.CallOpts, arg0)
}

// TokensBoughtInPool is a free data retrieval call binding the contract method 0x9b6f705b.
//
// Solidity: function tokensBoughtInPool(address ) view returns(uint256)
func (_PoolContract *PoolContractCallerSession) TokensBoughtInPool(arg0 common.Address) (*big.Int, error) {
	return _PoolContract.Contract.TokensBoughtInPool(&_PoolContract.CallOpts, arg0)
}

// TokensClaimed is a free data retrieval call binding the contract method 0x624601b6.
//
// Solidity: function tokensClaimed(address ) view returns(uint256)
func (_PoolContract *PoolContractCaller) TokensClaimed(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "tokensClaimed", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TokensClaimed is a free data retrieval call binding the contract method 0x624601b6.
//
// Solidity: function tokensClaimed(address ) view returns(uint256)
func (_PoolContract *PoolContractSession) TokensClaimed(arg0 common.Address) (*big.Int, error) {
	return _PoolContract.Contract.TokensClaimed(&_PoolContract.CallOpts, arg0)
}

// TokensClaimed is a free data retrieval call binding the contract method 0x624601b6.
//
// Solidity: function tokensClaimed(address ) view returns(uint256)
func (_PoolContract *PoolContractCallerSession) TokensClaimed(arg0 common.Address) (*big.Int, error) {
	return _PoolContract.Contract.TokensClaimed(&_PoolContract.CallOpts, arg0)
}

// TokensForLp is a free data retrieval call binding the contract method 0x8a0c84e2.
//
// Solidity: function tokensForLp() view returns(uint256)
func (_PoolContract *PoolContractCaller) TokensForLp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "tokensForLp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TokensForLp is a free data retrieval call binding the contract method 0x8a0c84e2.
//
// Solidity: function tokensForLp() view returns(uint256)
func (_PoolContract *PoolContractSession) TokensForLp() (*big.Int, error) {
	return _PoolContract.Contract.TokensForLp(&_PoolContract.CallOpts)
}

// TokensForLp is a free data retrieval call binding the contract method 0x8a0c84e2.
//
// Solidity: function tokensForLp() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) TokensForLp() (*big.Int, error) {
	return _PoolContract.Contract.TokensForLp(&_PoolContract.CallOpts)
}

// TokensInPool is a free data retrieval call binding the contract method 0x40e1e4c7.
//
// Solidity: function tokensInPool() view returns(uint256)
func (_PoolContract *PoolContractCaller) TokensInPool(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "tokensInPool")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TokensInPool is a free data retrieval call binding the contract method 0x40e1e4c7.
//
// Solidity: function tokensInPool() view returns(uint256)
func (_PoolContract *PoolContractSession) TokensInPool() (*big.Int, error) {
	return _PoolContract.Contract.TokensInPool(&_PoolContract.CallOpts)
}

// TokensInPool is a free data retrieval call binding the contract method 0x40e1e4c7.
//
// Solidity: function tokensInPool() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) TokensInPool() (*big.Int, error) {
	return _PoolContract.Contract.TokensInPool(&_PoolContract.CallOpts)
}

// TotalBalanceOf is a free data retrieval call binding the contract method 0x4b0ee02a.
//
// Solidity: function totalBalanceOf(address user) view returns(uint256)
func (_PoolContract *PoolContractCaller) TotalBalanceOf(opts *bind.CallOpts, user common.Address) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "totalBalanceOf", user)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalBalanceOf is a free data retrieval call binding the contract method 0x4b0ee02a.
//
// Solidity: function totalBalanceOf(address user) view returns(uint256)
func (_PoolContract *PoolContractSession) TotalBalanceOf(user common.Address) (*big.Int, error) {
	return _PoolContract.Contract.TotalBalanceOf(&_PoolContract.CallOpts, user)
}

// TotalBalanceOf is a free data retrieval call binding the contract method 0x4b0ee02a.
//
// Solidity: function totalBalanceOf(address user) view returns(uint256)
func (_PoolContract *PoolContractCallerSession) TotalBalanceOf(user common.Address) (*big.Int, error) {
	return _PoolContract.Contract.TotalBalanceOf(&_PoolContract.CallOpts, user)
}

// TotalRaise is a free data retrieval call binding the contract method 0x3996dc8f.
//
// Solidity: function totalRaise() view returns(uint256)
func (_PoolContract *PoolContractCaller) TotalRaise(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "totalRaise")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalRaise is a free data retrieval call binding the contract method 0x3996dc8f.
//
// Solidity: function totalRaise() view returns(uint256)
func (_PoolContract *PoolContractSession) TotalRaise() (*big.Int, error) {
	return _PoolContract.Contract.TotalRaise(&_PoolContract.CallOpts)
}

// TotalRaise is a free data retrieval call binding the contract method 0x3996dc8f.
//
// Solidity: function totalRaise() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) TotalRaise() (*big.Int, error) {
	return _PoolContract.Contract.TotalRaise(&_PoolContract.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_PoolContract *PoolContractCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_PoolContract *PoolContractSession) TotalSupply() (*big.Int, error) {
	return _PoolContract.Contract.TotalSupply(&_PoolContract.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) TotalSupply() (*big.Int, error) {
	return _PoolContract.Contract.TotalSupply(&_PoolContract.CallOpts)
}

// TotalTokens is a free data retrieval call binding the contract method 0x7e1c0c09.
//
// Solidity: function totalTokens() view returns(uint256)
func (_PoolContract *PoolContractCaller) TotalTokens(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "totalTokens")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalTokens is a free data retrieval call binding the contract method 0x7e1c0c09.
//
// Solidity: function totalTokens() view returns(uint256)
func (_PoolContract *PoolContractSession) TotalTokens() (*big.Int, error) {
	return _PoolContract.Contract.TotalTokens(&_PoolContract.CallOpts)
}

// TotalTokens is a free data retrieval call binding the contract method 0x7e1c0c09.
//
// Solidity: function totalTokens() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) TotalTokens() (*big.Int, error) {
	return _PoolContract.Contract.TotalTokens(&_PoolContract.CallOpts)
}

// Treasury is a free data retrieval call binding the contract method 0x61d027b3.
//
// Solidity: function treasury() view returns(address)
func (_PoolContract *PoolContractCaller) Treasury(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "treasury")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Treasury is a free data retrieval call binding the contract method 0x61d027b3.
//
// Solidity: function treasury() view returns(address)
func (_PoolContract *PoolContractSession) Treasury() (common.Address, error) {
	return _PoolContract.Contract.Treasury(&_PoolContract.CallOpts)
}

// Treasury is a free data retrieval call binding the contract method 0x61d027b3.
//
// Solidity: function treasury() view returns(address)
func (_PoolContract *PoolContractCallerSession) Treasury() (common.Address, error) {
	return _PoolContract.Contract.Treasury(&_PoolContract.CallOpts)
}

// VestedBalanceOf is a free data retrieval call binding the contract method 0x0e2d1a2a.
//
// Solidity: function vestedBalanceOf(address user) view returns(uint256)
func (_PoolContract *PoolContractCaller) VestedBalanceOf(opts *bind.CallOpts, user common.Address) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "vestedBalanceOf", user)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// VestedBalanceOf is a free data retrieval call binding the contract method 0x0e2d1a2a.
//
// Solidity: function vestedBalanceOf(address user) view returns(uint256)
func (_PoolContract *PoolContractSession) VestedBalanceOf(user common.Address) (*big.Int, error) {
	return _PoolContract.Contract.VestedBalanceOf(&_PoolContract.CallOpts, user)
}

// VestedBalanceOf is a free data retrieval call binding the contract method 0x0e2d1a2a.
//
// Solidity: function vestedBalanceOf(address user) view returns(uint256)
func (_PoolContract *PoolContractCallerSession) VestedBalanceOf(user common.Address) (*big.Int, error) {
	return _PoolContract.Contract.VestedBalanceOf(&_PoolContract.CallOpts, user)
}

// VestingPeriod is a free data retrieval call binding the contract method 0x7313ee5a.
//
// Solidity: function vestingPeriod() view returns(uint256)
func (_PoolContract *PoolContractCaller) VestingPeriod(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "vestingPeriod")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// VestingPeriod is a free data retrieval call binding the contract method 0x7313ee5a.
//
// Solidity: function vestingPeriod() view returns(uint256)
func (_PoolContract *PoolContractSession) VestingPeriod() (*big.Int, error) {
	return _PoolContract.Contract.VestingPeriod(&_PoolContract.CallOpts)
}

// VestingPeriod is a free data retrieval call binding the contract method 0x7313ee5a.
//
// Solidity: function vestingPeriod() view returns(uint256)
func (_PoolContract *PoolContractCallerSession) VestingPeriod() (*big.Int, error) {
	return _PoolContract.Contract.VestingPeriod(&_PoolContract.CallOpts)
}

// ViewExcessTokens is a free data retrieval call binding the contract method 0xd47ba4fd.
//
// Solidity: function viewExcessTokens() view returns(uint256 excessPandaTokens, uint256 excessBaseTokens)
func (_PoolContract *PoolContractCaller) ViewExcessTokens(opts *bind.CallOpts) (struct {
	ExcessPandaTokens *big.Int
	ExcessBaseTokens  *big.Int
}, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "viewExcessTokens")

	outstruct := new(struct {
		ExcessPandaTokens *big.Int
		ExcessBaseTokens  *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.ExcessPandaTokens = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.ExcessBaseTokens = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// ViewExcessTokens is a free data retrieval call binding the contract method 0xd47ba4fd.
//
// Solidity: function viewExcessTokens() view returns(uint256 excessPandaTokens, uint256 excessBaseTokens)
func (_PoolContract *PoolContractSession) ViewExcessTokens() (struct {
	ExcessPandaTokens *big.Int
	ExcessBaseTokens  *big.Int
}, error) {
	return _PoolContract.Contract.ViewExcessTokens(&_PoolContract.CallOpts)
}

// ViewExcessTokens is a free data retrieval call binding the contract method 0xd47ba4fd.
//
// Solidity: function viewExcessTokens() view returns(uint256 excessPandaTokens, uint256 excessBaseTokens)
func (_PoolContract *PoolContractCallerSession) ViewExcessTokens() (struct {
	ExcessPandaTokens *big.Int
	ExcessBaseTokens  *big.Int
}, error) {
	return _PoolContract.Contract.ViewExcessTokens(&_PoolContract.CallOpts)
}

// Wbera is a free data retrieval call binding the contract method 0x31f41a33.
//
// Solidity: function wbera() view returns(address)
func (_PoolContract *PoolContractCaller) Wbera(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PoolContract.contract.Call(opts, &out, "wbera")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Wbera is a free data retrieval call binding the contract method 0x31f41a33.
//
// Solidity: function wbera() view returns(address)
func (_PoolContract *PoolContractSession) Wbera() (common.Address, error) {
	return _PoolContract.Contract.Wbera(&_PoolContract.CallOpts)
}

// Wbera is a free data retrieval call binding the contract method 0x31f41a33.
//
// Solidity: function wbera() view returns(address)
func (_PoolContract *PoolContractCallerSession) Wbera() (common.Address, error) {
	return _PoolContract.Contract.Wbera(&_PoolContract.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_PoolContract *PoolContractTransactor) Approve(opts *bind.TransactOpts, spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "approve", spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_PoolContract *PoolContractSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _PoolContract.Contract.Approve(&_PoolContract.TransactOpts, spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_PoolContract *PoolContractTransactorSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _PoolContract.Contract.Approve(&_PoolContract.TransactOpts, spender, amount)
}

// BuyAllTokens is a paid mutator transaction binding the contract method 0x27773f6d.
//
// Solidity: function buyAllTokens(address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractTransactor) BuyAllTokens(opts *bind.TransactOpts, to common.Address) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "buyAllTokens", to)
}

// BuyAllTokens is a paid mutator transaction binding the contract method 0x27773f6d.
//
// Solidity: function buyAllTokens(address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractSession) BuyAllTokens(to common.Address) (*types.Transaction, error) {
	return _PoolContract.Contract.BuyAllTokens(&_PoolContract.TransactOpts, to)
}

// BuyAllTokens is a paid mutator transaction binding the contract method 0x27773f6d.
//
// Solidity: function buyAllTokens(address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractTransactorSession) BuyAllTokens(to common.Address) (*types.Transaction, error) {
	return _PoolContract.Contract.BuyAllTokens(&_PoolContract.TransactOpts, to)
}

// BuyTokens is a paid mutator transaction binding the contract method 0xbf8f8ce5.
//
// Solidity: function buyTokens(uint256 amountIn, uint256 minAmountOut, address from, address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractTransactor) BuyTokens(opts *bind.TransactOpts, amountIn *big.Int, minAmountOut *big.Int, from common.Address, to common.Address) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "buyTokens", amountIn, minAmountOut, from, to)
}

// BuyTokens is a paid mutator transaction binding the contract method 0xbf8f8ce5.
//
// Solidity: function buyTokens(uint256 amountIn, uint256 minAmountOut, address from, address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractSession) BuyTokens(amountIn *big.Int, minAmountOut *big.Int, from common.Address, to common.Address) (*types.Transaction, error) {
	return _PoolContract.Contract.BuyTokens(&_PoolContract.TransactOpts, amountIn, minAmountOut, from, to)
}

// BuyTokens is a paid mutator transaction binding the contract method 0xbf8f8ce5.
//
// Solidity: function buyTokens(uint256 amountIn, uint256 minAmountOut, address from, address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractTransactorSession) BuyTokens(amountIn *big.Int, minAmountOut *big.Int, from common.Address, to common.Address) (*types.Transaction, error) {
	return _PoolContract.Contract.BuyTokens(&_PoolContract.TransactOpts, amountIn, minAmountOut, from, to)
}

// BuyTokens0 is a paid mutator transaction binding the contract method 0xc1687877.
//
// Solidity: function buyTokens(uint256 amountIn, uint256 minAmountOut, address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractTransactor) BuyTokens0(opts *bind.TransactOpts, amountIn *big.Int, minAmountOut *big.Int, to common.Address) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "buyTokens0", amountIn, minAmountOut, to)
}

// BuyTokens0 is a paid mutator transaction binding the contract method 0xc1687877.
//
// Solidity: function buyTokens(uint256 amountIn, uint256 minAmountOut, address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractSession) BuyTokens0(amountIn *big.Int, minAmountOut *big.Int, to common.Address) (*types.Transaction, error) {
	return _PoolContract.Contract.BuyTokens0(&_PoolContract.TransactOpts, amountIn, minAmountOut, to)
}

// BuyTokens0 is a paid mutator transaction binding the contract method 0xc1687877.
//
// Solidity: function buyTokens(uint256 amountIn, uint256 minAmountOut, address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractTransactorSession) BuyTokens0(amountIn *big.Int, minAmountOut *big.Int, to common.Address) (*types.Transaction, error) {
	return _PoolContract.Contract.BuyTokens0(&_PoolContract.TransactOpts, amountIn, minAmountOut, to)
}

// BuyTokensWithBera is a paid mutator transaction binding the contract method 0x064cad2a.
//
// Solidity: function buyTokensWithBera(uint256 minAmountOut, address to) payable returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractTransactor) BuyTokensWithBera(opts *bind.TransactOpts, minAmountOut *big.Int, to common.Address) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "buyTokensWithBera", minAmountOut, to)
}

// BuyTokensWithBera is a paid mutator transaction binding the contract method 0x064cad2a.
//
// Solidity: function buyTokensWithBera(uint256 minAmountOut, address to) payable returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractSession) BuyTokensWithBera(minAmountOut *big.Int, to common.Address) (*types.Transaction, error) {
	return _PoolContract.Contract.BuyTokensWithBera(&_PoolContract.TransactOpts, minAmountOut, to)
}

// BuyTokensWithBera is a paid mutator transaction binding the contract method 0x064cad2a.
//
// Solidity: function buyTokensWithBera(uint256 minAmountOut, address to) payable returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractTransactorSession) BuyTokensWithBera(minAmountOut *big.Int, to common.Address) (*types.Transaction, error) {
	return _PoolContract.Contract.BuyTokensWithBera(&_PoolContract.TransactOpts, minAmountOut, to)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address user) returns(uint256)
func (_PoolContract *PoolContractTransactor) ClaimTokens(opts *bind.TransactOpts, user common.Address) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "claimTokens", user)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address user) returns(uint256)
func (_PoolContract *PoolContractSession) ClaimTokens(user common.Address) (*types.Transaction, error) {
	return _PoolContract.Contract.ClaimTokens(&_PoolContract.TransactOpts, user)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address user) returns(uint256)
func (_PoolContract *PoolContractTransactorSession) ClaimTokens(user common.Address) (*types.Transaction, error) {
	return _PoolContract.Contract.ClaimTokens(&_PoolContract.TransactOpts, user)
}

// CollectExcessTokens is a paid mutator transaction binding the contract method 0x4f1ee2f3.
//
// Solidity: function collectExcessTokens() returns()
func (_PoolContract *PoolContractTransactor) CollectExcessTokens(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "collectExcessTokens")
}

// CollectExcessTokens is a paid mutator transaction binding the contract method 0x4f1ee2f3.
//
// Solidity: function collectExcessTokens() returns()
func (_PoolContract *PoolContractSession) CollectExcessTokens() (*types.Transaction, error) {
	return _PoolContract.Contract.CollectExcessTokens(&_PoolContract.TransactOpts)
}

// CollectExcessTokens is a paid mutator transaction binding the contract method 0x4f1ee2f3.
//
// Solidity: function collectExcessTokens() returns()
func (_PoolContract *PoolContractTransactorSession) CollectExcessTokens() (*types.Transaction, error) {
	return _PoolContract.Contract.CollectExcessTokens(&_PoolContract.TransactOpts)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_PoolContract *PoolContractTransactor) DecreaseAllowance(opts *bind.TransactOpts, spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "decreaseAllowance", spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_PoolContract *PoolContractSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _PoolContract.Contract.DecreaseAllowance(&_PoolContract.TransactOpts, spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_PoolContract *PoolContractTransactorSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _PoolContract.Contract.DecreaseAllowance(&_PoolContract.TransactOpts, spender, subtractedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_PoolContract *PoolContractTransactor) IncreaseAllowance(opts *bind.TransactOpts, spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "increaseAllowance", spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_PoolContract *PoolContractSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _PoolContract.Contract.IncreaseAllowance(&_PoolContract.TransactOpts, spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_PoolContract *PoolContractTransactorSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _PoolContract.Contract.IncreaseAllowance(&_PoolContract.TransactOpts, spender, addedValue)
}

// InitializePool is a paid mutator transaction binding the contract method 0x0f373b53.
//
// Solidity: function initializePool(address _pandaToken, (address,uint256,uint256,uint256) pp, uint256 _totalTokens, address _deployer, bytes data) returns()
func (_PoolContract *PoolContractTransactor) InitializePool(opts *bind.TransactOpts, _pandaToken common.Address, pp IPandaStructsPandaPoolParams, _totalTokens *big.Int, _deployer common.Address, data []byte) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "initializePool", _pandaToken, pp, _totalTokens, _deployer, data)
}

// InitializePool is a paid mutator transaction binding the contract method 0x0f373b53.
//
// Solidity: function initializePool(address _pandaToken, (address,uint256,uint256,uint256) pp, uint256 _totalTokens, address _deployer, bytes data) returns()
func (_PoolContract *PoolContractSession) InitializePool(_pandaToken common.Address, pp IPandaStructsPandaPoolParams, _totalTokens *big.Int, _deployer common.Address, data []byte) (*types.Transaction, error) {
	return _PoolContract.Contract.InitializePool(&_PoolContract.TransactOpts, _pandaToken, pp, _totalTokens, _deployer, data)
}

// InitializePool is a paid mutator transaction binding the contract method 0x0f373b53.
//
// Solidity: function initializePool(address _pandaToken, (address,uint256,uint256,uint256) pp, uint256 _totalTokens, address _deployer, bytes data) returns()
func (_PoolContract *PoolContractTransactorSession) InitializePool(_pandaToken common.Address, pp IPandaStructsPandaPoolParams, _totalTokens *big.Int, _deployer common.Address, data []byte) (*types.Transaction, error) {
	return _PoolContract.Contract.InitializePool(&_PoolContract.TransactOpts, _pandaToken, pp, _totalTokens, _deployer, data)
}

// MoveLiquidity is a paid mutator transaction binding the contract method 0xb90a26ab.
//
// Solidity: function moveLiquidity() returns()
func (_PoolContract *PoolContractTransactor) MoveLiquidity(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "moveLiquidity")
}

// MoveLiquidity is a paid mutator transaction binding the contract method 0xb90a26ab.
//
// Solidity: function moveLiquidity() returns()
func (_PoolContract *PoolContractSession) MoveLiquidity() (*types.Transaction, error) {
	return _PoolContract.Contract.MoveLiquidity(&_PoolContract.TransactOpts)
}

// MoveLiquidity is a paid mutator transaction binding the contract method 0xb90a26ab.
//
// Solidity: function moveLiquidity() returns()
func (_PoolContract *PoolContractTransactorSession) MoveLiquidity() (*types.Transaction, error) {
	return _PoolContract.Contract.MoveLiquidity(&_PoolContract.TransactOpts)
}

// Permit is a paid mutator transaction binding the contract method 0xd505accf.
//
// Solidity: function permit(address owner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_PoolContract *PoolContractTransactor) Permit(opts *bind.TransactOpts, owner common.Address, spender common.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "permit", owner, spender, value, deadline, v, r, s)
}

// Permit is a paid mutator transaction binding the contract method 0xd505accf.
//
// Solidity: function permit(address owner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_PoolContract *PoolContractSession) Permit(owner common.Address, spender common.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _PoolContract.Contract.Permit(&_PoolContract.TransactOpts, owner, spender, value, deadline, v, r, s)
}

// Permit is a paid mutator transaction binding the contract method 0xd505accf.
//
// Solidity: function permit(address owner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_PoolContract *PoolContractTransactorSession) Permit(owner common.Address, spender common.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _PoolContract.Contract.Permit(&_PoolContract.TransactOpts, owner, spender, value, deadline, v, r, s)
}

// SellTokens is a paid mutator transaction binding the contract method 0xb6785df3.
//
// Solidity: function sellTokens(uint256 amountIn, uint256 minAmountOut, address from, address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractTransactor) SellTokens(opts *bind.TransactOpts, amountIn *big.Int, minAmountOut *big.Int, from common.Address, to common.Address) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "sellTokens", amountIn, minAmountOut, from, to)
}

// SellTokens is a paid mutator transaction binding the contract method 0xb6785df3.
//
// Solidity: function sellTokens(uint256 amountIn, uint256 minAmountOut, address from, address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractSession) SellTokens(amountIn *big.Int, minAmountOut *big.Int, from common.Address, to common.Address) (*types.Transaction, error) {
	return _PoolContract.Contract.SellTokens(&_PoolContract.TransactOpts, amountIn, minAmountOut, from, to)
}

// SellTokens is a paid mutator transaction binding the contract method 0xb6785df3.
//
// Solidity: function sellTokens(uint256 amountIn, uint256 minAmountOut, address from, address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractTransactorSession) SellTokens(amountIn *big.Int, minAmountOut *big.Int, from common.Address, to common.Address) (*types.Transaction, error) {
	return _PoolContract.Contract.SellTokens(&_PoolContract.TransactOpts, amountIn, minAmountOut, from, to)
}

// SellTokens0 is a paid mutator transaction binding the contract method 0xef569f9a.
//
// Solidity: function sellTokens(uint256 amountIn, uint256 minAmountOut, address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractTransactor) SellTokens0(opts *bind.TransactOpts, amountIn *big.Int, minAmountOut *big.Int, to common.Address) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "sellTokens0", amountIn, minAmountOut, to)
}

// SellTokens0 is a paid mutator transaction binding the contract method 0xef569f9a.
//
// Solidity: function sellTokens(uint256 amountIn, uint256 minAmountOut, address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractSession) SellTokens0(amountIn *big.Int, minAmountOut *big.Int, to common.Address) (*types.Transaction, error) {
	return _PoolContract.Contract.SellTokens0(&_PoolContract.TransactOpts, amountIn, minAmountOut, to)
}

// SellTokens0 is a paid mutator transaction binding the contract method 0xef569f9a.
//
// Solidity: function sellTokens(uint256 amountIn, uint256 minAmountOut, address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractTransactorSession) SellTokens0(amountIn *big.Int, minAmountOut *big.Int, to common.Address) (*types.Transaction, error) {
	return _PoolContract.Contract.SellTokens0(&_PoolContract.TransactOpts, amountIn, minAmountOut, to)
}

// SellTokensForBera is a paid mutator transaction binding the contract method 0x133d66c0.
//
// Solidity: function sellTokensForBera(uint256 amountIn, uint256 minAmountOut, address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractTransactor) SellTokensForBera(opts *bind.TransactOpts, amountIn *big.Int, minAmountOut *big.Int, to common.Address) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "sellTokensForBera", amountIn, minAmountOut, to)
}

// SellTokensForBera is a paid mutator transaction binding the contract method 0x133d66c0.
//
// Solidity: function sellTokensForBera(uint256 amountIn, uint256 minAmountOut, address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractSession) SellTokensForBera(amountIn *big.Int, minAmountOut *big.Int, to common.Address) (*types.Transaction, error) {
	return _PoolContract.Contract.SellTokensForBera(&_PoolContract.TransactOpts, amountIn, minAmountOut, to)
}

// SellTokensForBera is a paid mutator transaction binding the contract method 0x133d66c0.
//
// Solidity: function sellTokensForBera(uint256 amountIn, uint256 minAmountOut, address to) returns(uint256 amountOut, uint256 fee)
func (_PoolContract *PoolContractTransactorSession) SellTokensForBera(amountIn *big.Int, minAmountOut *big.Int, to common.Address) (*types.Transaction, error) {
	return _PoolContract.Contract.SellTokensForBera(&_PoolContract.TransactOpts, amountIn, minAmountOut, to)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_PoolContract *PoolContractTransactor) Transfer(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "transfer", to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_PoolContract *PoolContractSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _PoolContract.Contract.Transfer(&_PoolContract.TransactOpts, to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_PoolContract *PoolContractTransactorSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _PoolContract.Contract.Transfer(&_PoolContract.TransactOpts, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_PoolContract *PoolContractTransactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _PoolContract.contract.Transact(opts, "transferFrom", from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_PoolContract *PoolContractSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _PoolContract.Contract.TransferFrom(&_PoolContract.TransactOpts, from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_PoolContract *PoolContractTransactorSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _PoolContract.Contract.TransferFrom(&_PoolContract.TransactOpts, from, to, amount)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_PoolContract *PoolContractTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PoolContract.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_PoolContract *PoolContractSession) Receive() (*types.Transaction, error) {
	return _PoolContract.Contract.Receive(&_PoolContract.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_PoolContract *PoolContractTransactorSession) Receive() (*types.Transaction, error) {
	return _PoolContract.Contract.Receive(&_PoolContract.TransactOpts)
}

// PoolContractApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the PoolContract contract.
type PoolContractApprovalIterator struct {
	Event *PoolContractApproval // Event containing the contract specifics and raw log

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
func (it *PoolContractApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolContractApproval)
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
		it.Event = new(PoolContractApproval)
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
func (it *PoolContractApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolContractApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolContractApproval represents a Approval event raised by the PoolContract contract.
type PoolContractApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_PoolContract *PoolContractFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*PoolContractApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _PoolContract.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &PoolContractApprovalIterator{contract: _PoolContract.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_PoolContract *PoolContractFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *PoolContractApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _PoolContract.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolContractApproval)
				if err := _PoolContract.contract.UnpackLog(event, "Approval", log); err != nil {
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
func (_PoolContract *PoolContractFilterer) ParseApproval(log types.Log) (*PoolContractApproval, error) {
	event := new(PoolContractApproval)
	if err := _PoolContract.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolContractEIP712DomainChangedIterator is returned from FilterEIP712DomainChanged and is used to iterate over the raw logs and unpacked data for EIP712DomainChanged events raised by the PoolContract contract.
type PoolContractEIP712DomainChangedIterator struct {
	Event *PoolContractEIP712DomainChanged // Event containing the contract specifics and raw log

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
func (it *PoolContractEIP712DomainChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolContractEIP712DomainChanged)
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
		it.Event = new(PoolContractEIP712DomainChanged)
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
func (it *PoolContractEIP712DomainChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolContractEIP712DomainChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolContractEIP712DomainChanged represents a EIP712DomainChanged event raised by the PoolContract contract.
type PoolContractEIP712DomainChanged struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterEIP712DomainChanged is a free log retrieval operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_PoolContract *PoolContractFilterer) FilterEIP712DomainChanged(opts *bind.FilterOpts) (*PoolContractEIP712DomainChangedIterator, error) {

	logs, sub, err := _PoolContract.contract.FilterLogs(opts, "EIP712DomainChanged")
	if err != nil {
		return nil, err
	}
	return &PoolContractEIP712DomainChangedIterator{contract: _PoolContract.contract, event: "EIP712DomainChanged", logs: logs, sub: sub}, nil
}

// WatchEIP712DomainChanged is a free log subscription operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_PoolContract *PoolContractFilterer) WatchEIP712DomainChanged(opts *bind.WatchOpts, sink chan<- *PoolContractEIP712DomainChanged) (event.Subscription, error) {

	logs, sub, err := _PoolContract.contract.WatchLogs(opts, "EIP712DomainChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolContractEIP712DomainChanged)
				if err := _PoolContract.contract.UnpackLog(event, "EIP712DomainChanged", log); err != nil {
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

// ParseEIP712DomainChanged is a log parse operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_PoolContract *PoolContractFilterer) ParseEIP712DomainChanged(log types.Log) (*PoolContractEIP712DomainChanged, error) {
	event := new(PoolContractEIP712DomainChanged)
	if err := _PoolContract.contract.UnpackLog(event, "EIP712DomainChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolContractExcessCollectedIterator is returned from FilterExcessCollected and is used to iterate over the raw logs and unpacked data for ExcessCollected events raised by the PoolContract contract.
type PoolContractExcessCollectedIterator struct {
	Event *PoolContractExcessCollected // Event containing the contract specifics and raw log

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
func (it *PoolContractExcessCollectedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolContractExcessCollected)
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
		it.Event = new(PoolContractExcessCollected)
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
func (it *PoolContractExcessCollectedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolContractExcessCollectedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolContractExcessCollected represents a ExcessCollected event raised by the PoolContract contract.
type PoolContractExcessCollected struct {
	ExcessPandaTokens *big.Int
	ExcessBaseTokens  *big.Int
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterExcessCollected is a free log retrieval operation binding the contract event 0x0bc9de8618db7520d2390f3540611ed430618e118e80dcf5b5268e5c3e41d3f8.
//
// Solidity: event ExcessCollected(uint256 excessPandaTokens, uint256 excessBaseTokens)
func (_PoolContract *PoolContractFilterer) FilterExcessCollected(opts *bind.FilterOpts) (*PoolContractExcessCollectedIterator, error) {

	logs, sub, err := _PoolContract.contract.FilterLogs(opts, "ExcessCollected")
	if err != nil {
		return nil, err
	}
	return &PoolContractExcessCollectedIterator{contract: _PoolContract.contract, event: "ExcessCollected", logs: logs, sub: sub}, nil
}

// WatchExcessCollected is a free log subscription operation binding the contract event 0x0bc9de8618db7520d2390f3540611ed430618e118e80dcf5b5268e5c3e41d3f8.
//
// Solidity: event ExcessCollected(uint256 excessPandaTokens, uint256 excessBaseTokens)
func (_PoolContract *PoolContractFilterer) WatchExcessCollected(opts *bind.WatchOpts, sink chan<- *PoolContractExcessCollected) (event.Subscription, error) {

	logs, sub, err := _PoolContract.contract.WatchLogs(opts, "ExcessCollected")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolContractExcessCollected)
				if err := _PoolContract.contract.UnpackLog(event, "ExcessCollected", log); err != nil {
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

// ParseExcessCollected is a log parse operation binding the contract event 0x0bc9de8618db7520d2390f3540611ed430618e118e80dcf5b5268e5c3e41d3f8.
//
// Solidity: event ExcessCollected(uint256 excessPandaTokens, uint256 excessBaseTokens)
func (_PoolContract *PoolContractFilterer) ParseExcessCollected(log types.Log) (*PoolContractExcessCollected, error) {
	event := new(PoolContractExcessCollected)
	if err := _PoolContract.contract.UnpackLog(event, "ExcessCollected", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolContractLiquidityMovedIterator is returned from FilterLiquidityMoved and is used to iterate over the raw logs and unpacked data for LiquidityMoved events raised by the PoolContract contract.
type PoolContractLiquidityMovedIterator struct {
	Event *PoolContractLiquidityMoved // Event containing the contract specifics and raw log

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
func (it *PoolContractLiquidityMovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolContractLiquidityMoved)
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
		it.Event = new(PoolContractLiquidityMoved)
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
func (it *PoolContractLiquidityMovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolContractLiquidityMovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolContractLiquidityMoved represents a LiquidityMoved event raised by the PoolContract contract.
type PoolContractLiquidityMoved struct {
	AmountPanda *big.Int
	AmountBase  *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterLiquidityMoved is a free log retrieval operation binding the contract event 0x5280a0db66ccaf06018aded553c3eb913c2a3db70d0dfeeb0fb2042d68e4cef1.
//
// Solidity: event LiquidityMoved(uint256 amountPanda, uint256 amountBase)
func (_PoolContract *PoolContractFilterer) FilterLiquidityMoved(opts *bind.FilterOpts) (*PoolContractLiquidityMovedIterator, error) {

	logs, sub, err := _PoolContract.contract.FilterLogs(opts, "LiquidityMoved")
	if err != nil {
		return nil, err
	}
	return &PoolContractLiquidityMovedIterator{contract: _PoolContract.contract, event: "LiquidityMoved", logs: logs, sub: sub}, nil
}

// WatchLiquidityMoved is a free log subscription operation binding the contract event 0x5280a0db66ccaf06018aded553c3eb913c2a3db70d0dfeeb0fb2042d68e4cef1.
//
// Solidity: event LiquidityMoved(uint256 amountPanda, uint256 amountBase)
func (_PoolContract *PoolContractFilterer) WatchLiquidityMoved(opts *bind.WatchOpts, sink chan<- *PoolContractLiquidityMoved) (event.Subscription, error) {

	logs, sub, err := _PoolContract.contract.WatchLogs(opts, "LiquidityMoved")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolContractLiquidityMoved)
				if err := _PoolContract.contract.UnpackLog(event, "LiquidityMoved", log); err != nil {
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

// ParseLiquidityMoved is a log parse operation binding the contract event 0x5280a0db66ccaf06018aded553c3eb913c2a3db70d0dfeeb0fb2042d68e4cef1.
//
// Solidity: event LiquidityMoved(uint256 amountPanda, uint256 amountBase)
func (_PoolContract *PoolContractFilterer) ParseLiquidityMoved(log types.Log) (*PoolContractLiquidityMoved, error) {
	event := new(PoolContractLiquidityMoved)
	if err := _PoolContract.contract.UnpackLog(event, "LiquidityMoved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolContractPoolInitializedIterator is returned from FilterPoolInitialized and is used to iterate over the raw logs and unpacked data for PoolInitialized events raised by the PoolContract contract.
type PoolContractPoolInitializedIterator struct {
	Event *PoolContractPoolInitialized // Event containing the contract specifics and raw log

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
func (it *PoolContractPoolInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolContractPoolInitialized)
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
		it.Event = new(PoolContractPoolInitialized)
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
func (it *PoolContractPoolInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolContractPoolInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolContractPoolInitialized represents a PoolInitialized event raised by the PoolContract contract.
type PoolContractPoolInitialized struct {
	PandaToken    common.Address
	BaseToken     common.Address
	SqrtPa        *big.Int
	SqrtPb        *big.Int
	VestingPeriod *big.Int
	Deployer      common.Address
	Data          []byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterPoolInitialized is a free log retrieval operation binding the contract event 0xafb9d9a8f3a9fe695e6e5b7b1213fbfc5f019706ebf598c49c10ccd23c9988d7.
//
// Solidity: event PoolInitialized(address pandaToken, address baseToken, uint256 sqrtPa, uint256 sqrtPb, uint256 vestingPeriod, address deployer, bytes data)
func (_PoolContract *PoolContractFilterer) FilterPoolInitialized(opts *bind.FilterOpts) (*PoolContractPoolInitializedIterator, error) {

	logs, sub, err := _PoolContract.contract.FilterLogs(opts, "PoolInitialized")
	if err != nil {
		return nil, err
	}
	return &PoolContractPoolInitializedIterator{contract: _PoolContract.contract, event: "PoolInitialized", logs: logs, sub: sub}, nil
}

// WatchPoolInitialized is a free log subscription operation binding the contract event 0xafb9d9a8f3a9fe695e6e5b7b1213fbfc5f019706ebf598c49c10ccd23c9988d7.
//
// Solidity: event PoolInitialized(address pandaToken, address baseToken, uint256 sqrtPa, uint256 sqrtPb, uint256 vestingPeriod, address deployer, bytes data)
func (_PoolContract *PoolContractFilterer) WatchPoolInitialized(opts *bind.WatchOpts, sink chan<- *PoolContractPoolInitialized) (event.Subscription, error) {

	logs, sub, err := _PoolContract.contract.WatchLogs(opts, "PoolInitialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolContractPoolInitialized)
				if err := _PoolContract.contract.UnpackLog(event, "PoolInitialized", log); err != nil {
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

// ParsePoolInitialized is a log parse operation binding the contract event 0xafb9d9a8f3a9fe695e6e5b7b1213fbfc5f019706ebf598c49c10ccd23c9988d7.
//
// Solidity: event PoolInitialized(address pandaToken, address baseToken, uint256 sqrtPa, uint256 sqrtPb, uint256 vestingPeriod, address deployer, bytes data)
func (_PoolContract *PoolContractFilterer) ParsePoolInitialized(log types.Log) (*PoolContractPoolInitialized, error) {
	event := new(PoolContractPoolInitialized)
	if err := _PoolContract.contract.UnpackLog(event, "PoolInitialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolContractSwapIterator is returned from FilterSwap and is used to iterate over the raw logs and unpacked data for Swap events raised by the PoolContract contract.
type PoolContractSwapIterator struct {
	Event *PoolContractSwap // Event containing the contract specifics and raw log

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
func (it *PoolContractSwapIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolContractSwap)
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
		it.Event = new(PoolContractSwap)
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
func (it *PoolContractSwapIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolContractSwapIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolContractSwap represents a Swap event raised by the PoolContract contract.
type PoolContractSwap struct {
	Sender     common.Address
	Amount0In  *big.Int
	Amount1In  *big.Int
	Amount0Out *big.Int
	Amount1Out *big.Int
	To         common.Address
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterSwap is a free log retrieval operation binding the contract event 0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822.
//
// Solidity: event Swap(address indexed sender, uint256 amount0In, uint256 amount1In, uint256 amount0Out, uint256 amount1Out, address indexed to)
func (_PoolContract *PoolContractFilterer) FilterSwap(opts *bind.FilterOpts, sender []common.Address, to []common.Address) (*PoolContractSwapIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _PoolContract.contract.FilterLogs(opts, "Swap", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return &PoolContractSwapIterator{contract: _PoolContract.contract, event: "Swap", logs: logs, sub: sub}, nil
}

// WatchSwap is a free log subscription operation binding the contract event 0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822.
//
// Solidity: event Swap(address indexed sender, uint256 amount0In, uint256 amount1In, uint256 amount0Out, uint256 amount1Out, address indexed to)
func (_PoolContract *PoolContractFilterer) WatchSwap(opts *bind.WatchOpts, sink chan<- *PoolContractSwap, sender []common.Address, to []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _PoolContract.contract.WatchLogs(opts, "Swap", senderRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolContractSwap)
				if err := _PoolContract.contract.UnpackLog(event, "Swap", log); err != nil {
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

// ParseSwap is a log parse operation binding the contract event 0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822.
//
// Solidity: event Swap(address indexed sender, uint256 amount0In, uint256 amount1In, uint256 amount0Out, uint256 amount1Out, address indexed to)
func (_PoolContract *PoolContractFilterer) ParseSwap(log types.Log) (*PoolContractSwap, error) {
	event := new(PoolContractSwap)
	if err := _PoolContract.contract.UnpackLog(event, "Swap", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolContractSyncIterator is returned from FilterSync and is used to iterate over the raw logs and unpacked data for Sync events raised by the PoolContract contract.
type PoolContractSyncIterator struct {
	Event *PoolContractSync // Event containing the contract specifics and raw log

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
func (it *PoolContractSyncIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolContractSync)
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
		it.Event = new(PoolContractSync)
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
func (it *PoolContractSyncIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolContractSyncIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolContractSync represents a Sync event raised by the PoolContract contract.
type PoolContractSync struct {
	PandaReserve *big.Int
	BaseReserve  *big.Int
	SqrtPrice    *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterSync is a free log retrieval operation binding the contract event 0x9ea8a9dd7d3733c6dd274b7139f05a2bfce1a4bb22f0f7bdc1ccd49c267b858d.
//
// Solidity: event Sync(uint256 pandaReserve, uint256 baseReserve, uint256 sqrtPrice)
func (_PoolContract *PoolContractFilterer) FilterSync(opts *bind.FilterOpts) (*PoolContractSyncIterator, error) {

	logs, sub, err := _PoolContract.contract.FilterLogs(opts, "Sync")
	if err != nil {
		return nil, err
	}
	return &PoolContractSyncIterator{contract: _PoolContract.contract, event: "Sync", logs: logs, sub: sub}, nil
}

// WatchSync is a free log subscription operation binding the contract event 0x9ea8a9dd7d3733c6dd274b7139f05a2bfce1a4bb22f0f7bdc1ccd49c267b858d.
//
// Solidity: event Sync(uint256 pandaReserve, uint256 baseReserve, uint256 sqrtPrice)
func (_PoolContract *PoolContractFilterer) WatchSync(opts *bind.WatchOpts, sink chan<- *PoolContractSync) (event.Subscription, error) {

	logs, sub, err := _PoolContract.contract.WatchLogs(opts, "Sync")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolContractSync)
				if err := _PoolContract.contract.UnpackLog(event, "Sync", log); err != nil {
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

// ParseSync is a log parse operation binding the contract event 0x9ea8a9dd7d3733c6dd274b7139f05a2bfce1a4bb22f0f7bdc1ccd49c267b858d.
//
// Solidity: event Sync(uint256 pandaReserve, uint256 baseReserve, uint256 sqrtPrice)
func (_PoolContract *PoolContractFilterer) ParseSync(log types.Log) (*PoolContractSync, error) {
	event := new(PoolContractSync)
	if err := _PoolContract.contract.UnpackLog(event, "Sync", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolContractTokensClaimedIterator is returned from FilterTokensClaimed and is used to iterate over the raw logs and unpacked data for TokensClaimed events raised by the PoolContract contract.
type PoolContractTokensClaimedIterator struct {
	Event *PoolContractTokensClaimed // Event containing the contract specifics and raw log

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
func (it *PoolContractTokensClaimedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolContractTokensClaimed)
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
		it.Event = new(PoolContractTokensClaimed)
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
func (it *PoolContractTokensClaimedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolContractTokensClaimedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolContractTokensClaimed represents a TokensClaimed event raised by the PoolContract contract.
type PoolContractTokensClaimed struct {
	User   common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterTokensClaimed is a free log retrieval operation binding the contract event 0x896e034966eaaf1adc54acc0f257056febbd300c9e47182cf761982cf1f5e430.
//
// Solidity: event TokensClaimed(address indexed user, uint256 amount)
func (_PoolContract *PoolContractFilterer) FilterTokensClaimed(opts *bind.FilterOpts, user []common.Address) (*PoolContractTokensClaimedIterator, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _PoolContract.contract.FilterLogs(opts, "TokensClaimed", userRule)
	if err != nil {
		return nil, err
	}
	return &PoolContractTokensClaimedIterator{contract: _PoolContract.contract, event: "TokensClaimed", logs: logs, sub: sub}, nil
}

// WatchTokensClaimed is a free log subscription operation binding the contract event 0x896e034966eaaf1adc54acc0f257056febbd300c9e47182cf761982cf1f5e430.
//
// Solidity: event TokensClaimed(address indexed user, uint256 amount)
func (_PoolContract *PoolContractFilterer) WatchTokensClaimed(opts *bind.WatchOpts, sink chan<- *PoolContractTokensClaimed, user []common.Address) (event.Subscription, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _PoolContract.contract.WatchLogs(opts, "TokensClaimed", userRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolContractTokensClaimed)
				if err := _PoolContract.contract.UnpackLog(event, "TokensClaimed", log); err != nil {
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

// ParseTokensClaimed is a log parse operation binding the contract event 0x896e034966eaaf1adc54acc0f257056febbd300c9e47182cf761982cf1f5e430.
//
// Solidity: event TokensClaimed(address indexed user, uint256 amount)
func (_PoolContract *PoolContractFilterer) ParseTokensClaimed(log types.Log) (*PoolContractTokensClaimed, error) {
	event := new(PoolContractTokensClaimed)
	if err := _PoolContract.contract.UnpackLog(event, "TokensClaimed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PoolContractTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the PoolContract contract.
type PoolContractTransferIterator struct {
	Event *PoolContractTransfer // Event containing the contract specifics and raw log

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
func (it *PoolContractTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PoolContractTransfer)
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
		it.Event = new(PoolContractTransfer)
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
func (it *PoolContractTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PoolContractTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PoolContractTransfer represents a Transfer event raised by the PoolContract contract.
type PoolContractTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_PoolContract *PoolContractFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*PoolContractTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _PoolContract.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &PoolContractTransferIterator{contract: _PoolContract.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_PoolContract *PoolContractFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *PoolContractTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _PoolContract.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PoolContractTransfer)
				if err := _PoolContract.contract.UnpackLog(event, "Transfer", log); err != nil {
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
func (_PoolContract *PoolContractFilterer) ParseTransfer(log types.Log) (*PoolContractTransfer, error) {
	event := new(PoolContractTransfer)
	if err := _PoolContract.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
