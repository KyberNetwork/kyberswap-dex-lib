package swapdata

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackAlgebraV1(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap := buildAlgebraV1(encodingSwap)

	return packAlgebraV1(swap)
}

func UnpackAlgebraV1(encodedSwap []byte) (AlgebraV1, error) {
	unpacked, err := AlgebraV1ABIArguments.Unpack(encodedSwap)
	if err != nil {
		return AlgebraV1{}, err
	}

	var swap AlgebraV1
	if err = AlgebraV1ABIArguments.Copy(&swap, unpacked); err != nil {
		return AlgebraV1{}, err
	}

	return swap, nil
}

func buildAlgebraV1(swap types.EncodingSwap) AlgebraV1 {
	return AlgebraV1{
		Recipient:           common.HexToAddress(swap.Recipient),
		Pool:                common.HexToAddress(swap.Pool),
		TokenIn:             common.HexToAddress(swap.TokenIn),
		TokenOut:            common.HexToAddress(swap.TokenOut),
		SwapAmount:          swap.SwapAmount,
		SqrtPriceLimitX96:   constant.Zero,
		SenderFeeOnTransfer: constant.Zero, // TODO: support FoT token
	}
}

func packAlgebraV1(swap AlgebraV1) ([]byte, error) {
	return AlgebraV1ABIArguments.Pack(
		swap.Recipient,
		swap.Pool,
		swap.TokenIn,
		swap.TokenOut,
		swap.SwapAmount,
		swap.SqrtPriceLimitX96,
		swap.SenderFeeOnTransfer,
	)
}
