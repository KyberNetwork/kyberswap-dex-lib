package swapdata

import (
	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/router-service/internal/pkg/constant/abitypes"
)

var (
	UniswapABIArguments           abi.Arguments
	StableSwapABIArguments        abi.Arguments
	CurveSwapABIArguments         abi.Arguments
	KokonutCryptoABIArguments     abi.Arguments
	UniswapV3KSElasticABIArgument abi.Arguments
	BalancerV2ABIArguments        abi.Arguments
	DODOABIArguments              abi.Arguments
	SynthetixABIArguments         abi.Arguments
	PSMABIArguments               abi.Arguments
	WSTETHABIArguments            abi.Arguments
	StETHABIArguments             abi.Arguments
	PlatypusABIArguments          abi.Arguments
	KyberLimitOrderABIArguments   abi.Arguments
	KyberRFQABIType               abi.Type
	KyberRFQABIArguments          abi.Arguments

	GMXABIArguments abi.Arguments
	// GmxGlpABIArguments https://github.com/KyberNetwork/ks-dex-aggregator-sc/pull/228/files#diff-aef4b18ab626112c08de702796dd44471d23fa6e19d45afef4a4ab126ebbb3e0
	GmxGlpABIArguments abi.Arguments

	// Mantis: https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/develop/contracts/executor-helpers/ExecutorHelper4.sol#L17
	WombatABIArguments abi.Arguments

	// Syncswap
	SyncSwapABIArguments     abi.Arguments
	SyncSwapDataABIArguments abi.Arguments

	// MaverickV1
	MaverickABIArguments abi.Arguments

	AlgebraV1ABIArguments abi.Arguments

	TraderJoeV2Arguments abi.Arguments

	FillBatchOrdersParamsABIType abi.Type

	FillBatchOrdersParamsDSABIType abi.Type
	KyberLimitOrderDSABIArguments  abi.Arguments

	IZiSwapArguments      abi.Arguments
	VooiArguments         abi.Arguments
	MaticMigrateArguments abi.Arguments
	SmardexArguments      abi.Arguments
	BalancerV1Arguments   abi.Arguments

	VelocoreV2Arguments abi.Arguments
)

func init() {
	UniswapABIArguments = abi.Arguments{
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "recipient", Type: abitypes.Address},
		{Name: "collectAmount", Type: abitypes.Uint256},
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
		{Name: "usePoolUnderlying", Type: abitypes.Bool},
		{Name: "useTriCrypto", Type: abitypes.Bool},
	}

	KokonutCryptoABIArguments = abi.Arguments{
		{Name: "pool", Type: abitypes.Address},
		{Name: "dx", Type: abitypes.Uint256},
		{Name: "tokenIndexFrom", Type: abitypes.Int128},
		{Name: "fromToken", Type: abitypes.Address},
		{Name: "toToken", Type: abitypes.Address},
	}

	UniswapV3KSElasticABIArgument = abi.Arguments{
		{Name: "recipient", Type: abitypes.Address},
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "swapAmount", Type: abitypes.Uint256},
		{Name: "sqrtPriceLimitX96", Type: abitypes.Uint160},
		{Name: "isUniV3", Type: abitypes.Bool},
	}

	AlgebraV1ABIArguments = abi.Arguments{
		{Name: "recipient", Type: abitypes.Address},
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "swapAmount", Type: abitypes.Uint256},
		{Name: "sqrtPriceLimitX96", Type: abitypes.Uint160},
		{Name: "senderFeeOnTransfer", Type: abitypes.Uint256},
	}

	BalancerV2ABIArguments = abi.Arguments{
		{Name: "vault", Type: abitypes.Address},
		{Name: "poolId", Type: abitypes.Bytes32},
		{Name: "assetIn", Type: abitypes.Address},
		{Name: "assetOut", Type: abitypes.Address},
		{Name: "amount", Type: abitypes.Uint256},
	}

	DODOABIArguments = abi.Arguments{
		{Name: "recipient", Type: abitypes.Address},
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenFrom", Type: abitypes.Address},
		{Name: "tokenTo", Type: abitypes.Address},
		{Name: "amount", Type: abitypes.Uint256},
		{Name: "sellHelper", Type: abitypes.Address},
		{Name: "isSellBase", Type: abitypes.Bool},
		{Name: "isVersion2", Type: abitypes.Bool},
	}

	GMXABIArguments = abi.Arguments{
		{Name: "vault", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "amount", Type: abitypes.Uint256},
		{Name: "receiver", Type: abitypes.Address},
	}

	SynthetixABIArguments = abi.Arguments{
		{Name: "synthetixProxy", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "sourceCurrencyKey", Type: abitypes.Bytes32},
		{Name: "sourceAmount", Type: abitypes.Uint256},
		{Name: "destinationCurrencyKey", Type: abitypes.Bytes32},
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

	StETHABIArguments = abi.Arguments{
		{Name: "amount", Type: abitypes.Uint256},
	}

	PlatypusABIArguments = abi.Arguments{
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "recipient", Type: abitypes.Address},
		{Name: "collectAmount", Type: abitypes.Uint256},
		{Name: "limitReturnAmount", Type: abitypes.Uint256},
	}

	IZiSwapArguments = abi.Arguments{
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "recipient", Type: abitypes.Address},
		{Name: "swapAmount", Type: abitypes.Uint256},
		{Name: "limitPoint", Type: abitypes.Int24},
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

	FillBatchOrdersParamsDSABIType, _ = abi.NewType("tuple", "",
		[]abi.ArgumentMarshaling{
			{
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
					{Name: "feeConfig", Type: "uint256"},
					{Name: "makerAssetData", Type: "bytes"},
					{Name: "takerAssetData", Type: "bytes"},
					{Name: "getMakerAmount", Type: "bytes"},
					{Name: "getTakerAmount", Type: "bytes"},
					{Name: "predicate", Type: "bytes"},
					{Name: "interaction", Type: "bytes"},
				},
			},
			{
				Name: "signatures", Type: "tuple[]",
				Components: []abi.ArgumentMarshaling{
					{Name: "orderSignature", Type: "bytes"},
					{Name: "opSignature", Type: "bytes"},
				},
			},
			{Name: "opExpireTimes", Type: "uint32[]"},
			{Name: "takingAmount", Type: "uint256"},
			{Name: "thresholdAmount", Type: "uint256"},
			{Name: "target", Type: "address"},
		})

	KyberLimitOrderDSABIArguments = abi.Arguments{
		{Name: "kyberLOAddress", Type: abitypes.Address},
		{Name: "makerAsset", Type: abitypes.Address},
		{Name: "takerAsset", Type: abitypes.Address},
		{Name: "params", Type: FillBatchOrdersParamsDSABIType},
	}

	// Reference from SC
	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/develop_zk/contracts/executor-helpers/ZkSyncExecutorHelper.sol#L72-L77
	SyncSwapABIArguments = abi.Arguments{
		{Name: "_data", Type: abitypes.Bytes},
		{Name: "vault", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "pool", Type: abitypes.Address},
		{Name: "collectAmount", Type: abitypes.Uint256},
	}
	// _data encode of (address, address, uint8) : (tokenIn, recipient, withdrawMode)
	// withdrawMode: always using 0 (DEFAULT)
	SyncSwapDataABIArguments = abi.Arguments{
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "recipient", Type: abitypes.Address},
		{Name: "withdrawMode", Type: abitypes.Uint8},
	}

	MaverickABIArguments = abi.Arguments{
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "recipient", Type: abitypes.Address},
		{Name: "swapAmount", Type: abitypes.Uint256},
		{Name: "sqrtPriceLimitD18", Type: abitypes.Uint256},
	}

	TraderJoeV2Arguments = abi.Arguments{
		{Name: "recipient", Type: abitypes.Address},
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "collectAmount", Type: abitypes.Uint256},
	}

	KyberRFQABIType, _ = abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "rfq", Type: "address"},
		// Order
		// Reference: https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/fda542505b49252f6c59273d9ee542377be6c3a9/contracts/interfaces/pool-types/IRFQ.sol#L7-L18
		{Name: "order", Type: "tuple", Components: []abi.ArgumentMarshaling{
			{Name: "info", Type: "uint256"},
			{Name: "makerAsset", Type: "address"},
			{Name: "takerAsset", Type: "address"},
			{Name: "maker", Type: "address"},
			{Name: "allowedSender", Type: "address"},
			{Name: "makingAmount", Type: "uint256"},
			{Name: "takingAmount", Type: "uint256"},
		}},
		{Name: "signature", Type: "bytes"},
		{Name: "amount", Type: "uint256"},
		{Name: "target", Type: "address"},
	})

	// KyberRFQABIArguments
	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/fda542505b49252f6c59273d9ee542377be6c3a9/contracts/executor-helpers/ExecutorHelper2.sol#L91-L97
	KyberRFQABIArguments = abi.Arguments{
		{Type: KyberRFQABIType},
	}

	// WombatABIArguments
	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/develop/contracts/executor-helpers/ExecutorHelper3.sol#L335
	WombatABIArguments = abi.Arguments{
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "amount", Type: abitypes.Uint256},
		{Name: "recipient", Type: abitypes.Address},
	}

	VooiArguments = abi.Arguments{
		{Name: "pool", Type: abitypes.Address},
		{Name: "fromToken", Type: abitypes.Address},
		{Name: "toToken", Type: abitypes.Address},
		{Name: "fromID", Type: abitypes.Uint256},
		{Name: "toID", Type: abitypes.Uint256},
		{Name: "fromAmount", Type: abitypes.Uint256},
		{Name: "to", Type: abitypes.Address},
	}

	MaticMigrateArguments = abi.Arguments{
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenAddress", Type: abitypes.Address},
		{Name: "amount", Type: abitypes.Uint256},
		{Name: "recipient", Type: abitypes.Address},
	}

	SmardexArguments = abi.Arguments{
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "amount", Type: abitypes.Uint256},
		{Name: "recipient", Type: abitypes.Address},
	}

	BalancerV1Arguments = abi.Arguments{
		{Name: "pool", Type: abitypes.Address},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "amount", Type: abitypes.Uint256},
	}

	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/3f9af867bd74dfb9d23321f500ed266c15d9d59e/src/contracts-zksync/ExecutorHelper4.sol#L674
	VelocoreV2Arguments = abi.Arguments{
		{Name: "vault", Type: abitypes.Address},
		{Name: "amount", Type: abitypes.Uint256},
		{Name: "tokenIn", Type: abitypes.Address},
		{Name: "tokenOut", Type: abitypes.Address},
		{Name: "stablePool", Type: abitypes.Address},
		{Name: "wrapToken", Type: abitypes.Address},
		{Name: "isConvertFirst", Type: abitypes.Bool},
	}
}
