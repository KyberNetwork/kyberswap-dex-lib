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
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func PackKyberLimitOrderDS(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	kyberLimitOrder, err := buildKyberLimitOrderDS(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packKyberLimitOrderDS(kyberLimitOrder)
}

func UnpackKyberLimitOrderDS(encodedSwap []byte) (KyberLimitOrderDS, error) {
	encodedSwapStr := hex.EncodeToString(encodedSwap)
	packedEncodedSwapDataStr := strings.Replace(encodedSwapStr, OffsetToTheStartOfData, "", 1)
	packedEncodedSwapBytes := common.Hex2Bytes(packedEncodedSwapDataStr)
	unpacked, err := KyberLimitOrderDSABIArguments.Unpack(packedEncodedSwapBytes)
	if err != nil {
		return KyberLimitOrderDS{}, err
	}

	var swap KyberLimitOrderDS
	if err = KyberLimitOrderDSABIArguments.Copy(&swap, unpacked); err != nil {
		return KyberLimitOrderDS{}, err
	}

	return swap, nil
}

func buildKyberLimitOrderDS(swap types.EncodingSwap) (KyberLimitOrderDS, error) {
	byteData, err := json.Marshal(swap.Extra)
	if err != nil {
		return KyberLimitOrderDS{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildKyberLimitOrder] err :[%v]",
			err,
		)
	}

	var swapInfo limitorder.OpSignatureExtra
	if err = json.Unmarshal(byteData, &swapInfo); err != nil {
		return KyberLimitOrderDS{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[BuildKyberLimitOrder] err :[%v]",
			err,
		)
	}
	if len(swapInfo.FilledOrders) == 0 {
		return KyberLimitOrderDS{}, fmt.Errorf("[BuildKyberLimitOrder] cause by filledOrder is empty")
	}
	Params, err := toFillBatchOrdersParamsDS(&swapInfo)
	if err != nil {
		return KyberLimitOrderDS{}, fmt.Errorf("[BuildKyberLimitOrder] error at toFillBatchOrdersParams func error cause by %v", err)
	}
	return KyberLimitOrderDS{
		KyberLOAddress: common.HexToAddress(swap.Pool),
		MakerAsset:     common.HexToAddress(swapInfo.FilledOrders[0].MakerAsset),
		TakerAsset:     common.HexToAddress(swapInfo.FilledOrders[0].TakerAsset),
		Params:         Params,
	}, nil
}

func toFillBatchOrdersParamsDS(swapInfo *limitorder.OpSignatureExtra) (FillBatchOrdersParamsDS, error) {
	signatures := make([]Signature, len(swapInfo.FilledOrders))
	orders := make([]OrderDS, len(swapInfo.FilledOrders))
	opExpireTimes := make([]uint32, len(swapInfo.FilledOrders))

	for i, filledOrder := range swapInfo.FilledOrders {
		bytesSignature, err := hex.DecodeString(filledOrder.Signature)
		if err != nil {
			return FillBatchOrdersParamsDS{}, err
		}
		opSignature, ok := swapInfo.OperatorSignaturesById[filledOrder.OrderID]
		if !ok {
			return FillBatchOrdersParamsDS{}, fmt.Errorf("Operator signature not found for order %v", filledOrder.OrderID)
		}
		logger.Debugf("Operator signature %v %v %v", filledOrder.OrderID, opSignature.OperatorSignature, opSignature.OperatorSignatureExpiredAt)
		bytesOpSignature, err := hex.DecodeString(opSignature.OperatorSignature)
		if err != nil {
			return FillBatchOrdersParamsDS{}, err
		}
		signatures[i] = Signature{
			OrderSignature: bytesSignature,
			OpSignature:    bytesOpSignature,
		}
		opExpireTimes[i] = uint32(opSignature.OperatorSignatureExpiredAt)

		feeConfig, ok := new(big.Int).SetString(filledOrder.FeeConfig, 10)
		if !ok {
			return FillBatchOrdersParamsDS{}, fmt.Errorf("invalid feeConfig %v", filledOrder.FeeConfig)
		}
		bytesTakerAssetData, err := hex.DecodeString(filledOrder.TakerAssetData)
		if err != nil {
			return FillBatchOrdersParamsDS{}, err
		}
		bytesGetMakerAmount, err := hex.DecodeString(filledOrder.GetMakerAmount)
		if err != nil {
			return FillBatchOrdersParamsDS{}, err
		}
		bytesGetTakerAmount, err := hex.DecodeString(filledOrder.GetTakerAmount)
		if err != nil {
			return FillBatchOrdersParamsDS{}, err
		}
		bytesPredicate, err := hex.DecodeString(filledOrder.Predicate)
		if err != nil {
			return FillBatchOrdersParamsDS{}, err
		}
		bytesMakerPermit, err := hex.DecodeString(filledOrder.Permit)
		if err != nil {
			return FillBatchOrdersParamsDS{}, err
		}
		bytesInteraction, err := hex.DecodeString(filledOrder.Interaction)
		if err != nil {
			return FillBatchOrdersParamsDS{}, err
		}
		makingAmount, ok := new(big.Int).SetString(filledOrder.MakingAmount, 10)
		if !ok {
			return FillBatchOrdersParamsDS{}, fmt.Errorf("invalid makingAmount %v", filledOrder.MakingAmount)
		}
		takingAmount, ok := new(big.Int).SetString(filledOrder.TakingAmount, 10)
		if !ok {
			return FillBatchOrdersParamsDS{}, fmt.Errorf("invalid takingAmount %v", filledOrder.TakingAmount)
		}
		orders[i] = OrderDS{
			MakerAsset:     common.HexToAddress(filledOrder.MakerAsset),
			TakerAsset:     common.HexToAddress(filledOrder.TakerAsset),
			Maker:          common.HexToAddress(filledOrder.Maker),
			Receiver:       common.HexToAddress(filledOrder.Receiver),
			AllowedSender:  common.HexToAddress(filledOrder.AllowedSenders),
			MakingAmount:   makingAmount,
			TakingAmount:   takingAmount,
			FeeConfig:      feeConfig,
			MakerAssetData: bytesTakerAssetData,
			TakerAssetData: bytesTakerAssetData,
			GetMakerAmount: bytesGetMakerAmount,
			GetTakerAmount: bytesGetTakerAmount,
			Predicate:      bytesPredicate,
			Permit:         bytesMakerPermit,
			Interaction:    bytesInteraction,
		}
		if len(filledOrder.Salt) == 0 {
			return FillBatchOrdersParamsDS{}, fmt.Errorf("salt is empty")
		}
		salt, ok := new(big.Int).SetString(filledOrder.Salt, 10)
		if !ok {
			return FillBatchOrdersParamsDS{}, fmt.Errorf("invalid salt")
		}
		orders[i].Salt = salt
	}
	amountIn, ok := new(big.Int).SetString(swapInfo.AmountIn, 10)
	if !ok {
		return FillBatchOrdersParamsDS{}, fmt.Errorf("toFillBatchOrdersParams error cause by parsing amountIn")
	}
	return FillBatchOrdersParamsDS{
		Orders:          orders,
		Signatures:      signatures,
		OpExpireTimes:   opExpireTimes,
		TakingAmount:    amountIn,
		ThresholdAmount: &big.Int{},
		Target:          [20]byte{},
	}, nil
}

func packKyberLimitOrderDS(kyberLimitOrder KyberLimitOrderDS) ([]byte, error) {
	packedData, err := KyberLimitOrderDSABIArguments.Pack(
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
