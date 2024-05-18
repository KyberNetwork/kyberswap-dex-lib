package univ3_test

import (
	"bytes"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/router-service/internal/pkg/core/univ3"
)

type exactInputSingleParams struct {
	TokenIn           common.Address
	TokenOut          common.Address
	Fee               *big.Int
	Recipient         common.Address
	Deadline          *big.Int
	AmountIn          *big.Int
	AmountOutMinimum  *big.Int
	SqrtPriceLimitX96 *big.Int
}

var uniswapV3RouterABI abi.ABI

func init() {
	var err error
	uniswapV3RouterABI, err = abi.JSON(bytes.NewReader([]byte(`[
		{
			"inputs": [
				{
					"components": [
						{
							"internalType": "address",
							"name": "tokenIn",
							"type": "address"
						},
						{
							"internalType": "address",
							"name": "tokenOut",
							"type": "address"
						},
						{
							"internalType": "uint24",
							"name": "fee",
							"type": "uint24"
						},
						{
							"internalType": "address",
							"name": "recipient",
							"type": "address"
						},
						{
							"internalType": "uint256",
							"name": "deadline",
							"type": "uint256"
						},
						{
							"internalType": "uint256",
							"name": "amountIn",
							"type": "uint256"
						},
						{
							"internalType": "uint256",
							"name": "amountOutMinimum",
							"type": "uint256"
						},
						{
							"internalType": "uint160",
							"name": "sqrtPriceLimitX96",
							"type": "uint160"
						}
					],
					"internalType": "struct ISwapRouter.ExactInputSingleParams",
					"name": "params",
					"type": "tuple"
				}
			],
			"name": "exactInputSingle",
			"outputs": [
				{
					"internalType": "uint256",
					"name": "amountOut",
					"type": "uint256"
				}
			],
			"stateMutability": "payable",
			"type": "function"
		}
	]`)))
	if err != nil {
		panic(err)
	}
}

func TestRouterExactInputSingle(t *testing.T) {
	for _, param := range []struct {
		poolFee  *big.Int
		amountIn *big.Int
		tokenIn  common.Address
		tokenOut common.Address
		wallet   common.Address
	}{
		{
			big.NewInt(3000),
			big.NewInt(500_000_000_000),
			common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
			common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"),
			common.HexToAddress("0x95222290dd7278aa3ddd389cc1e1d165cc4bafe5"),
		},
	} {
		expected, err := uniswapV3RouterABI.Pack("exactInputSingle", &exactInputSingleParams{
			TokenIn:           param.tokenIn,
			TokenOut:          param.tokenOut,
			Fee:               param.poolFee,
			Recipient:         param.wallet,
			Deadline:          new(big.Int).SetUint64(math.MaxUint64),
			AmountIn:          param.amountIn,
			AmountOutMinimum:  big.NewInt(0),
			SqrtPriceLimitX96: big.NewInt(0),
		})
		require.NoError(t, err)

		actual, err := univ3.PackRouterExactInputSingleCalldata(param.amountIn, param.poolFee, param.tokenIn, param.tokenOut, param.wallet)
		require.NoError(t, err)

		require.Equal(t, expected, actual)
	}
}
