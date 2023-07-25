package swapdata

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

var (
	// OffsetToTheStartOfData 32 bytes string
	// https://docs.soliditylang.org/en/develop/abi-spec.html#use-of-dynamic-types
	OffsetToTheStartOfData = "0000000000000000000000000000000000000000000000000000000000000020"
)

func PackKyberLimitOrder(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	kyberLimitOrder, err := buildKyberLimitOrder(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packKyberLimitOrder(kyberLimitOrder)
}

func UnpackKyberLimitOrder(encodedSwap []byte) (KyberLimitOrder, error) {
	encodedSwapStr := hex.EncodeToString(encodedSwap)
	packedEncodedSwapDataStr := strings.Replace(encodedSwapStr, OffsetToTheStartOfData, "", 1)
	packedEncodedSwapBytes := common.Hex2Bytes(packedEncodedSwapDataStr)
	unpacked, err := KyberLimitOrderABIArguments.Unpack(packedEncodedSwapBytes)
	if err != nil {
		return KyberLimitOrder{}, err
	}

	var swap KyberLimitOrder
	if err = KyberLimitOrderABIArguments.Copy(&swap, unpacked); err != nil {
		return KyberLimitOrder{}, err
	}

	return swap, nil
}

func buildKyberLimitOrder(swap types.EncodingSwap) (KyberLimitOrder, error) {
	byteData, err := json.Marshal(swap.Extra)
	if err != nil {
		return KyberLimitOrder{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildKyberLimitOrder] err :[%v]",
			err,
		)
	}

	var swapInfo limitorder.SwapInfo
	if err = json.Unmarshal(byteData, &swapInfo); err != nil {
		return KyberLimitOrder{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[BuildKyberLimitOrder] err :[%v]",
			err,
		)
	}
	if len(swapInfo.FilledOrders) == 0 {
		return KyberLimitOrder{}, fmt.Errorf("[BuildKyberLimitOrder] cause by filledOrder is empty")
	}
	Params, err := toFillBatchOrdersParams(&swapInfo)
	if err != nil {
		return KyberLimitOrder{}, fmt.Errorf("[BuildKyberLimitOrder] error at toFillBatchOrdersParams func error cause by %v", err)
	}
	return KyberLimitOrder{
		KyberLOAddress: common.HexToAddress(swap.Pool),
		MakerAsset:     common.HexToAddress(swapInfo.FilledOrders[0].MakerAsset),
		TakerAsset:     common.HexToAddress(swapInfo.FilledOrders[0].TakerAsset),
		Params:         Params,
	}, nil
}

func toFillBatchOrdersParams(swapInfo *limitorder.SwapInfo) (FillBatchOrdersParams, error) {
	signatures := make([][]byte, len(swapInfo.FilledOrders))
	orders := make([]Order, len(swapInfo.FilledOrders))

	for i, filledOrder := range swapInfo.FilledOrders {
		bytesSignature, err := hex.DecodeString(filledOrder.Signature)
		if err != nil {
			return FillBatchOrdersParams{}, err
		}
		signatures[i] = bytesSignature

		bytesTakerAssetData, err := hex.DecodeString(filledOrder.TakerAssetData)
		if err != nil {
			return FillBatchOrdersParams{}, err
		}
		bytesGetMakerAmount, err := hex.DecodeString(filledOrder.GetMakerAmount)
		if err != nil {
			return FillBatchOrdersParams{}, err
		}
		bytesGetTakerAmount, err := hex.DecodeString(filledOrder.GetTakerAmount)
		if err != nil {
			return FillBatchOrdersParams{}, err
		}
		bytesPredicate, err := hex.DecodeString(filledOrder.Predicate)
		if err != nil {
			return FillBatchOrdersParams{}, err
		}
		bytesMakerPermit, err := hex.DecodeString(filledOrder.Permit)
		if err != nil {
			return FillBatchOrdersParams{}, err
		}
		bytesInteraction, err := hex.DecodeString(filledOrder.Interaction)
		if err != nil {
			return FillBatchOrdersParams{}, err
		}
		makingAmount, ok := new(big.Int).SetString(filledOrder.MakingAmount, 10)
		if !ok {
			return FillBatchOrdersParams{}, fmt.Errorf("toFillBatchOrdersParams error cause by parsing makingAmount")
		}
		takingAmount, ok := new(big.Int).SetString(filledOrder.TakingAmount, 10)
		if !ok {
			return FillBatchOrdersParams{}, fmt.Errorf("toFillBatchOrdersParams error cause by parsing takingAmount")
		}
		orders[i] = Order{
			MakerAsset:           common.HexToAddress(filledOrder.MakerAsset),
			TakerAsset:           common.HexToAddress(filledOrder.TakerAsset),
			Maker:                common.HexToAddress(filledOrder.Maker),
			Receiver:             common.HexToAddress(filledOrder.Receiver),
			AllowedSender:        common.HexToAddress(filledOrder.AllowedSenders),
			MakingAmount:         makingAmount,
			TakingAmount:         takingAmount,
			FeeRecipient:         common.HexToAddress(filledOrder.FeeRecipient),
			MakerTokenFeePercent: filledOrder.MakerTokenFeePercent,
			MakerAssetData:       bytesTakerAssetData,
			TakerAssetData:       bytesTakerAssetData,
			GetMakerAmount:       bytesGetMakerAmount,
			GetTakerAmount:       bytesGetTakerAmount,
			Predicate:            bytesPredicate,
			Permit:               bytesMakerPermit,
			Interaction:          bytesInteraction,
		}
		if len(filledOrder.Salt) == 0 {
			return FillBatchOrdersParams{}, fmt.Errorf("salt is empty")
		}
		salt, ok := new(big.Int).SetString(filledOrder.Salt, 10)
		if !ok {
			return FillBatchOrdersParams{}, fmt.Errorf("invalid salt")
		}
		orders[i].Salt = salt
	}
	amountIn, ok := new(big.Int).SetString(swapInfo.AmountIn, 10)
	if !ok {
		return FillBatchOrdersParams{}, fmt.Errorf("toFillBatchOrdersParams error cause by parsing amountIn")
	}
	return FillBatchOrdersParams{
		Orders:          orders,
		Signatures:      signatures,
		TakingAmount:    amountIn,
		ThresholdAmount: &big.Int{},
		Target:          [20]byte{},
	}, nil
}

func packKyberLimitOrder(kyberLimitOrder KyberLimitOrder) ([]byte, error) {
	packedData, err := KyberLimitOrderABIArguments.Pack(
		kyberLimitOrder.KyberLOAddress,
		kyberLimitOrder.MakerAsset,
		kyberLimitOrder.TakerAsset,
		kyberLimitOrder.Params,
	)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(OffsetToTheStartOfData + common.Bytes2Hex(packedData))
}
