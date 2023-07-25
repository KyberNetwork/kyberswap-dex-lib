package swapdata

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
)

func PackMaverickV1(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildMaverickV1(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packMaverickV1(swap)
}

func UnpackMaverickV1(data []byte) (MaverickV1Swap, error) {
	unpacked, err := MaverickABIArguments.Unpack(data)
	if err != nil {
		return MaverickV1Swap{}, err
	}

	var swap MaverickV1Swap
	if err = MaverickABIArguments.Copy(&swap, unpacked); err != nil {
		return MaverickV1Swap{}, err
	}

	return swap, nil
}

func buildMaverickV1(swap types.EncodingSwap) (MaverickV1Swap, error) {
	// Always use all tokenIn amount
	// Following this discussion https://team-kyber.slack.com/archives/C03RRK1FDGT/p1688706963191379?thread_ts=1688532975.118269&cid=C03RRK1FDGT
	sqrtPriceLimitD18 := big.NewInt(0)

	return MaverickV1Swap{
		Pool:              common.HexToAddress(swap.Pool),
		TokenIn:           common.HexToAddress(swap.TokenIn),
		TokenOut:          common.HexToAddress(swap.TokenOut),
		Recipient:         common.HexToAddress(swap.Recipient),
		SwapAmount:        swap.SwapAmount,
		SqrtPriceLimitD18: sqrtPriceLimitD18,
	}, nil
}

func packMaverickV1(swap MaverickV1Swap) ([]byte, error) {
	return MaverickABIArguments.Pack(
		swap.Pool,
		swap.TokenIn,
		swap.TokenOut,
		swap.Recipient,
		swap.SwapAmount,
		big.NewInt(0),
	)
}
