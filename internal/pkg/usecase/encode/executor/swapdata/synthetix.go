package swapdata

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackSynthetix(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildSynthetix(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packSynthetix(swap)
}

func UnpackSynthetix(encodedSwap []byte) (Synthetix, error) {
	unpacked, err := SynthetixABIArguments.Unpack(encodedSwap)
	if err != nil {
		return Synthetix{}, err
	}

	var swap Synthetix
	if err = SynthetixABIArguments.Copy(&swap, unpacked); err != nil {
		return Synthetix{}, err
	}

	return swap, nil
}

func buildSynthetix(swap types.EncodingSwap) (Synthetix, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return Synthetix{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildSynthetix] err :[%v]",
			err,
		)
	}

	var meta struct {
		SourceCurrencyKey      string `json:"sourceCurrencyKey"`
		DestinationCurrencyKey string `json:"destinationCurrencyKey"`
		UseAtomicExchange      bool   `json:"useAtomicExchange"`
	}

	if err = json.Unmarshal(byteData, &meta); err != nil {
		return Synthetix{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[BuildSynthetix] err :[%v]",
			err,
		)
	}

	return Synthetix{
		SynthetixProxy:         common.HexToAddress(swap.Pool),
		TokenIn:                common.HexToAddress(swap.TokenIn),
		TokenOut:               common.HexToAddress(swap.TokenOut),
		SourceCurrencyKey:      eth.StringToBytes32(meta.SourceCurrencyKey),
		SourceAmount:           swap.SwapAmount,
		DestinationCurrencyKey: eth.StringToBytes32(meta.DestinationCurrencyKey),
		MinAmount:              swap.LimitReturnAmount,
		UseAtomicExchange:      meta.UseAtomicExchange,
	}, nil
}

func packSynthetix(swap Synthetix) ([]byte, error) {
	return SynthetixABIArguments.Pack(
		swap.SynthetixProxy,
		swap.TokenIn,
		swap.TokenOut,
		swap.SourceCurrencyKey,
		swap.SourceAmount,
		swap.DestinationCurrencyKey,
		swap.MinAmount,
		swap.UseAtomicExchange,
	)
}
