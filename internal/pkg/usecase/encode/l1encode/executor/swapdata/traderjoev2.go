package swapdata

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackTraderJoeV2(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildTraderJoeV2(encodingSwap)
	if err != nil {
		return nil, err
	}
	return packTraderJoeV2(swap)
}

func UnpackTraderJoeV2(data []byte) (TraderJoeV2, error) {
	unpacked, err := TraderJoeV2Arguments.Unpack(data)
	if err != nil {
		return TraderJoeV2{}, err
	}

	var swap TraderJoeV2
	if err := TraderJoeV2Arguments.Copy(&swap, unpacked); err != nil {
		return TraderJoeV2{}, err
	}

	return swap, nil
}

func buildTraderJoeV2(swap types.EncodingSwap) (TraderJoeV2, error) {
	if new(big.Int).Rsh(swap.CollectAmount, 255).Cmp(constant.Zero) != 0 {
		return TraderJoeV2{}, fmt.Errorf("the most significant bit is reserved to discriminate between V2.0 and V2.1")
	}

	var versionMask *big.Int
	switch swap.Exchange {
	case valueobject.ExchangeTraderJoeV20:
		// 1 << 255
		versionMask = new(big.Int).Lsh(constant.One, 255)
	case valueobject.ExchangeTraderJoeV21:
		versionMask = constant.Zero
	default:
		return TraderJoeV2{}, fmt.Errorf("unsupported exchange %s", swap.Exchange)
	}

	packedCollectAmount := new(big.Int).Or(swap.CollectAmount, versionMask)

	return TraderJoeV2{
		Recipient:           common.HexToAddress(swap.Recipient),
		Pool:                common.HexToAddress(swap.Pool),
		TokenIn:             common.HexToAddress(swap.TokenIn),
		TokenOut:            common.HexToAddress(swap.TokenOut),
		PackedCollectAmount: packedCollectAmount,
	}, nil
}

func packTraderJoeV2(swap TraderJoeV2) ([]byte, error) {
	return TraderJoeV2Arguments.Pack(
		swap.Recipient,
		swap.Pool,
		swap.TokenIn,
		swap.TokenOut,
		swap.PackedCollectAmount,
	)
}
