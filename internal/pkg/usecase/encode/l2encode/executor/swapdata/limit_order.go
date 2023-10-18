package swapdata

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/validator"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type KyberLimitOrder struct {
	PoolMappingID  pack.UInt24
	KyberLOAddress common.Address
	MakerAsset     common.Address
	Params         FillBatchOrdersParams

	isFirstSwap bool
}

type FillBatchOrdersParams struct {
	Orders          []Order
	Signatures      [][]byte
	TakingAmount    *big.Int
	ThresholdAmount *big.Int
	Target          common.Address
}

type Order struct {
	Salt                 *big.Int
	MakerAsset           common.Address
	TakerAsset           common.Address
	Maker                common.Address
	Receiver             common.Address
	AllowedSender        common.Address
	MakingAmount         *big.Int
	TakingAmount         *big.Int
	FeeRecipient         common.Address
	MakerTokenFeePercent uint32
	MakerAssetData       []byte
	TakerAssetData       []byte
	GetMakerAmount       []byte
	GetTakerAmount       []byte
	Predicate            []byte
	Permit               []byte
	Interaction          []byte
}

func PackKyberLimitOrder(_ valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	// get contract address for LO.
	if encodingSwap.PoolExtra == nil {
		return nil, fmt.Errorf("[PackKyberLimitOrder] PoolExtra is nil")
	}

	contractAddress, ok := encodingSwap.PoolExtra.(string)
	if !ok || !validator.IsEthereumAddress(contractAddress) {
		errMsg := fmt.Sprintf("Invalid LO contract address: %v, pool: %v", encodingSwap.PoolExtra, encodingSwap.Pool)
		return nil, fmt.Errorf("[PackKyberLimitOrder] %s", errMsg)
	}
	encodingSwap.Pool = contractAddress

	kyberLimitOrder, err := buildKyberLimitOrder(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packKyberLimitOrder(kyberLimitOrder)
}

func UnpackKyberLimitOrder(data []byte, isFirstSwap bool) (KyberLimitOrder, error) {
	var swap KyberLimitOrder
	var startByte int

	swap.PoolMappingID, startByte = pack.ReadUInt24(data, startByte)
	if swap.PoolMappingID == 0 {
		swap.KyberLOAddress, startByte = pack.ReadAddress(data, startByte)
	}

	swap.MakerAsset, startByte = pack.ReadAddress(data, startByte)

	ordersLength, startByte := pack.ReadUInt8(data, startByte)
	swap.Params.Orders = make([]Order, ordersLength)
	for i := uint8(0); i < ordersLength; i++ {
		swap.Params.Orders[i].Salt, startByte = pack.ReadBigInt(data, startByte)
		swap.Params.Orders[i].MakerAsset, startByte = pack.ReadAddress(data, startByte)
		swap.Params.Orders[i].TakerAsset, startByte = pack.ReadAddress(data, startByte)
		swap.Params.Orders[i].Maker, startByte = pack.ReadAddress(data, startByte)
		swap.Params.Orders[i].Receiver, startByte = pack.ReadAddress(data, startByte)
		swap.Params.Orders[i].AllowedSender, startByte = pack.ReadAddress(data, startByte)
		swap.Params.Orders[i].MakingAmount, startByte = pack.ReadBigInt(data, startByte)
		swap.Params.Orders[i].TakingAmount, startByte = pack.ReadBigInt(data, startByte)
		swap.Params.Orders[i].FeeRecipient, startByte = pack.ReadAddress(data, startByte)
		swap.Params.Orders[i].MakerTokenFeePercent, startByte = pack.ReadUInt32(data, startByte)
		swap.Params.Orders[i].MakerAssetData, startByte = pack.ReadBytes(data, startByte)
		swap.Params.Orders[i].TakerAssetData, startByte = pack.ReadBytes(data, startByte)
		swap.Params.Orders[i].GetMakerAmount, startByte = pack.ReadBytes(data, startByte)
		swap.Params.Orders[i].GetTakerAmount, startByte = pack.ReadBytes(data, startByte)
		swap.Params.Orders[i].Predicate, startByte = pack.ReadBytes(data, startByte)
		swap.Params.Orders[i].Permit, startByte = pack.ReadBytes(data, startByte)
		swap.Params.Orders[i].Interaction, startByte = pack.ReadBytes(data, startByte)
	}

	signaturesLength, startByte := pack.ReadUInt8(data, startByte)
	swap.Params.Signatures = make([][]byte, signaturesLength)
	for i := uint8(0); i < ordersLength; i++ {
		swap.Params.Signatures[i], startByte = pack.ReadBytes(data, startByte)
	}

	if isFirstSwap {
		swap.Params.TakingAmount, startByte = pack.ReadBigInt(data, startByte)
		swap.isFirstSwap = true
	} else {
		var collectAmountFlag bool
		collectAmountFlag, startByte = pack.ReadBoolean(data, startByte)
		if collectAmountFlag {
			swap.Params.TakingAmount = abi.MaxUint256
		}
	}
	swap.Params.ThresholdAmount, startByte = pack.ReadBigInt(data, startByte)
	swap.Params.Target, _ = pack.ReadAddress(data, startByte)

	return swap, nil
}

func buildKyberLimitOrder(swap types.L2EncodingSwap) (KyberLimitOrder, error) {
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
	params, err := toFillBatchOrdersParams(&swapInfo)
	if err != nil {
		return KyberLimitOrder{}, fmt.Errorf("[BuildKyberLimitOrder] error at toFillBatchOrdersParams func error cause by %v", err)
	}
	return KyberLimitOrder{
		PoolMappingID:  swap.PoolMappingID,
		KyberLOAddress: common.HexToAddress(swap.Pool),
		MakerAsset:     common.HexToAddress(swapInfo.FilledOrders[0].MakerAsset),
		Params:         params,

		isFirstSwap: swap.IsFirstSwap,
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
			return FillBatchOrdersParams{}, fmt.Errorf("[toFillBatchOrdersParams] error cause by parsing makingAmount")
		}
		takingAmount, ok := new(big.Int).SetString(filledOrder.TakingAmount, 10)
		if !ok {
			return FillBatchOrdersParams{}, fmt.Errorf("[toFillBatchOrdersParams] error cause by parsing takingAmount")
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
			return FillBatchOrdersParams{}, fmt.Errorf("[toFillBatchOrdersParams] salt is empty")
		}
		salt, ok := new(big.Int).SetString(filledOrder.Salt, 10)
		if !ok {
			return FillBatchOrdersParams{}, fmt.Errorf("[toFillBatchOrdersParams] invalid salt")
		}
		orders[i].Salt = salt
	}
	amountIn, ok := new(big.Int).SetString(swapInfo.AmountIn, 10)
	if !ok {
		return FillBatchOrdersParams{}, fmt.Errorf("[toFillBatchOrdersParams] error cause by parsing amountIn")
	}
	return FillBatchOrdersParams{
		Orders:          orders,
		Signatures:      signatures,
		TakingAmount:    amountIn,
		ThresholdAmount: &big.Int{},
		Target:          common.HexToAddress(valueobject.ZeroAddress),
	}, nil
}

func packKyberLimitOrder(order KyberLimitOrder) ([]byte, error) {
	// Pack []Order
	var ordersPacked []pack.RawBytes
	for _, order := range order.Params.Orders {
		orderPacked, err := pack.Pack(
			order.Salt,
			order.MakerAsset,
			order.TakerAsset,
			order.Maker,
			order.Receiver,
			order.AllowedSender,
			order.MakingAmount,
			order.TakingAmount,
			order.FeeRecipient,
			order.MakerTokenFeePercent,
			order.MakerAssetData,
			order.TakerAssetData,
			order.GetMakerAmount,
			order.GetTakerAmount,
			order.Predicate,
			order.Permit,
			order.Interaction,
		)
		if err != nil {
			return nil, err
		}
		ordersPacked = append(ordersPacked, pack.RawBytes(orderPacked))
	}

	// Pack FillBatchOrdersParams
	var params []interface{}

	params = append(params, ordersPacked, order.Params.Signatures)
	if order.isFirstSwap {
		params = append(params, order.Params.TakingAmount)
	} else {
		var collectAmountFlag bool
		if order.Params.TakingAmount.Cmp(constant.Zero) > 0 {
			collectAmountFlag = true
		}
		params = append(params, collectAmountFlag)
	}
	params = append(params, order.Params.ThresholdAmount, order.Params.Target)

	paramsPacked, err := pack.Pack(params...)
	if err != nil {
		return nil, err
	}

	// Pack KyberLimitOrder
	var args []interface{}

	args = append(args, order.PoolMappingID)
	if order.PoolMappingID == 0 {
		args = append(args, order.KyberLOAddress)
	}
	args = append(args, order.MakerAsset, pack.RawBytes(paramsPacked))

	return pack.Pack(args...)
}
