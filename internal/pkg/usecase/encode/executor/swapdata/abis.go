package swapdata

import (
	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant/abitypes"
)

var (
	UniSwapABIArguments         abi.Arguments
	StableSwapABIArguments      abi.Arguments
	CurveSwapABIArguments       abi.Arguments
	UniSwapV3ProMMABIArguments  abi.Arguments
	BalancerV2ABIArguments      abi.Arguments
	DODOABIArguments            abi.Arguments
	GMXABIArguments             abi.Arguments
	SynthetixABIArguments       abi.Arguments
	PSMABIArguments             abi.Arguments
	WSTETHABIArguments          abi.Arguments
	PlatypusABIArguments        abi.Arguments
	KyberLimitOrderABIArguments abi.Arguments

	FillBatchOrdersParamsABIType abi.Type
)

func init() {
	UniSwapABIArguments = abi.Arguments{
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "recipient", Type: abitypes.Address},
		{Name: "collectAmount", Type: abitypes.Uint256},
		{Name: "limitReturnAmount", Type: abitypes.Uint256},
		{Name: "swapFee", Type: abitypes.Uint32},
		{Name: "feePrecision", Type: abitypes.Uint32},
		{Name: "tokenWeightInput", Type: abitypes.Uint32},
	}

	StableSwapABIArguments = abi.Arguments{
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenFrom", Type: abitypes.Address},
		{Name: "tokenTo", Type: abitypes.Address},
		{Name: "tokenIndexFrom", Type: abitypes.Uint8},
		{Name: "tokenIndexTo", Type: abitypes.Uint8},
		{Name: "dx", Type: abitypes.Uint256},
		{Name: "minDy", Type: abitypes.Uint256},
		{Name: "poolLength", Type: abitypes.Uint256},
		{Name: "poolLp", Type: abitypes.Address},
		{Name: "isSaddle", Type: abitypes.Bool},
	}

	CurveSwapABIArguments = abi.Arguments{
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenFrom", Type: abitypes.Address},
		{Name: "tokenTo", Type: abitypes.Address},
		{Name: "tokenIndexFrom", Type: abitypes.Int128},
		{Name: "tokenIndexTo", Type: abitypes.Int128},
		{Name: "dx", Type: abitypes.Uint256},
		{Name: "minDy", Type: abitypes.Uint256},
		{Name: "usePoolUnderlying", Type: abitypes.Bool},
		{Name: "useTriCrypto", Type: abitypes.Bool},
	}

	UniSwapV3ProMMABIArguments = abi.Arguments{
		{Name: "recipient", Type: abitypes.Address},
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "swapAmount", Type: abitypes.Uint256},
		{Name: "limitReturnAmount", Type: abitypes.Uint256},
		{Name: "sqrtPriceLimitX96", Type: abitypes.Uint160},
		{Name: "isUniV3", Type: abitypes.Bool},
	}

	BalancerV2ABIArguments = abi.Arguments{
		{Name: "vault", Type: abitypes.Address},
		{Name: "poolId", Type: abitypes.Bytes32},
		{Name: "assetIn", Type: abitypes.Address},
		{Name: "assetOut", Type: abitypes.Address},
		{Name: "amount", Type: abitypes.Uint256},
		{Name: "limit", Type: abitypes.Uint256},
	}

	DODOABIArguments = abi.Arguments{
		{Name: "recipient", Type: abitypes.Address},
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenFrom", Type: abitypes.Address},
		{Name: "tokenTo", Type: abitypes.Address},
		{Name: "amount", Type: abitypes.Uint256},
		{Name: "minReceiveQuote", Type: abitypes.Uint256},
		{Name: "sellHelper", Type: abitypes.Address},
		{Name: "isSellBase", Type: abitypes.Bool},
		{Name: "isVersion2", Type: abitypes.Bool},
	}

	GMXABIArguments = abi.Arguments{
		{Name: "vault", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "amount", Type: abitypes.Uint256},
		{Name: "minOut", Type: abitypes.Uint256},
		{Name: "receiver", Type: abitypes.Address},
	}

	SynthetixABIArguments = abi.Arguments{
		{Name: "synthetixProxy", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "sourceCurrencyKey", Type: abitypes.Bytes32},
		{Name: "sourceAmount", Type: abitypes.Uint256},
		{Name: "destinationCurrencyKey", Type: abitypes.Bytes32},
		{Name: "minAmount", Type: abitypes.Uint256},
		{Name: "useAtomicExchange", Type: abitypes.Bool},
	}

	PSMABIArguments = abi.Arguments{
		{Name: "router", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "amountIn", Type: abitypes.Uint256},
		{Name: "recipient", Type: abitypes.Address},
	}

	WSTETHABIArguments = abi.Arguments{
		{Name: "pool", Type: abitypes.Address},
		{Name: "amount", Type: abitypes.Uint256},
		{Name: "isWrapping", Type: abitypes.Bool},
	}

	PlatypusABIArguments = abi.Arguments{
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "recipient", Type: abitypes.Address},
		{Name: "collectAmount", Type: abitypes.Uint256},
		{Name: "limitReturnAmount", Type: abitypes.Uint256},
	}

	// Reference from SC
	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/022e3cf6952004824194d5df8184489df573721c/contracts/interfaces/IKyberLimitOrder.sol#L26
	FillBatchOrdersParamsABIType, _ = abi.NewType("tuple", "",
		[]abi.ArgumentMarshaling{
			{
				// Reference from SC
				// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/022e3cf6952004824194d5df8184489df573721c/contracts/interfaces/IKyberLimitOrder.sol#L6
				Name: "orders", Type: "tuple[]",
				Components: []abi.ArgumentMarshaling{
					{Name: "salt", Type: "uint256"},
					{Name: "makerAsset", Type: "address"},
					{Name: "takerAsset", Type: "address"},
					{Name: "maker", Type: "address"},
					{Name: "receiver", Type: "address"},
					{Name: "allowedSender", Type: "address"},
					{Name: "makingAmount", Type: "uint256"},
					{Name: "takingAmount", Type: "uint256"},
					{Name: "feeRecipient", Type: "address"},
					{Name: "makerTokenFeePercent", Type: "uint32"},
					{Name: "makerAssetData", Type: "bytes"},
					{Name: "takerAssetData", Type: "bytes"},
					{Name: "getMakerAmount", Type: "bytes"},
					{Name: "getTakerAmount", Type: "bytes"},
					{Name: "predicate", Type: "bytes"},
					{Name: "permit", Type: "bytes"},
					{Name: "interaction", Type: "bytes"},
				},
			},
			{Name: "signatures", Type: "bytes[]"},
			{Name: "takingAmount", Type: "uint256"},
			{Name: "thresholdAmount", Type: "uint256"},
			{Name: "target", Type: "address"},
		})

	// Reference from SC
	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/022e3cf6952004824194d5df8184489df573721c/contracts/executor-helpers/ExecutorHelper2.sol#L18
	KyberLimitOrderABIArguments = abi.Arguments{
		{Name: "kyberLOAddress", Type: abitypes.Address},
		{Name: "makerAsset", Type: abitypes.Address},
		{Name: "takerAsset", Type: abitypes.Address},
		{Name: "params", Type: FillBatchOrdersParamsABIType},
	}
}
