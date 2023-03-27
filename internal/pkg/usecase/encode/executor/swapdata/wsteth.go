package swapdata

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/types"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
)

func PackWSTETH(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap := buildWSTETH(encodingSwap)

	return packWSTETH(swap)
}

func UnpackWSTETH(encodedSwap []byte) (WSTETH, error) {
	unpacked, err := WSTETHABIArguments.Unpack(encodedSwap)
	if err != nil {
		return WSTETH{}, err
	}

	var swap WSTETH
	if err = WSTETHABIArguments.Copy(&swap, unpacked); err != nil {
		return WSTETH{}, err
	}

	return swap, nil
}

func buildWSTETH(swap types.EncodingSwap) WSTETH {
	var isWrapping bool
	// If the tokenOut is wstETH, it is wrapping
	if strings.EqualFold(swap.TokenOut, swap.Pool) {
		isWrapping = true
	}

	return WSTETH{
		Pool:       common.HexToAddress(swap.Pool),
		Amount:     swap.SwapAmount,
		IsWrapping: isWrapping,
	}
}

func packWSTETH(swap WSTETH) ([]byte, error) {
	return WSTETHABIArguments.Pack(
		swap.Pool,
		swap.Amount,
		swap.IsWrapping,
	)
}
