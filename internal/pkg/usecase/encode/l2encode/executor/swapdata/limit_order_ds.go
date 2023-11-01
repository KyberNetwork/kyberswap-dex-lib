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
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type KyberLimitOrderDS struct {
	PoolMappingID  pack.UInt24
	KyberLOAddress common.Address
	MakerAsset     common.Address
	Params         FillBatchOrdersParamsDS

	isFirstSwap bool
}

type FillBatchOrdersParamsDS struct {
	Orders          []OrderDS
	Signatures      []Signature
	OpExpireTimes   []uint32
	TakingAmount    *big.Int
	ThresholdAmount *big.Int
	Target          common.Address
}

type OrderDS struct {
	Salt           *big.Int
	MakerAsset     common.Address
	TakerAsset     common.Address
	Maker          common.Address
	Receiver       common.Address
	AllowedSender  common.Address
	MakingAmount   *big.Int
	TakingAmount   *big.Int
	FeeConfig      pack.UInt200
	MakerAssetData []byte
	TakerAssetData []byte
	GetMakerAmount []byte
	GetTakerAmount []byte
	Predicate      []byte
	Interaction    []byte
}

type Signature struct {
	OrderSignature []byte // Signature to confirm quote ownership
	OpSignature    []byte // OP Signature to confirm quote ownership
}

func PackKyberLimitOrderDS(_ valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	// get contract address for LO.
	if encodingSwap.PoolExtra == nil {
		return nil, fmt.Errorf("[PackKyberLimitOrderDS] PoolExtra is nil")
	}

	contractAddress, ok := encodingSwap.PoolExtra.(string)
	if !ok || !validator.IsEthereumAddress(contractAddress) {
		errMsg := fmt.Sprintf("Invalid LO contract address: %v, pool: %v", encodingSwap.PoolExtra, encodingSwap.Pool)
		return nil, fmt.Errorf("[PackKyberLimitOrderDS] %s", errMsg)
	}
	encodingSwap.Pool = contractAddress

	kyberLimitOrder, err := buildKyberLimitOrderDS(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packKyberLimitOrderDS(kyberLimitOrder)
}

func UnpackKyberLimitOrderDS(data []byte, isFirstSwap bool) (KyberLimitOrderDS, error) {
	var swap KyberLimitOrderDS
	var startByte int

	swap.PoolMappingID, startByte = pack.ReadUInt24(data, startByte)
	if swap.PoolMappingID == 0 {
		swap.KyberLOAddress, startByte = pack.ReadAddress(data, startByte)
	}

	swap.MakerAsset, startByte = pack.ReadAddress(data, startByte)

	// unpack orders
	ordersLength, startByte := pack.ReadUInt8(data, startByte)
	swap.Params.Orders = make([]OrderDS, ordersLength)
	for i := uint8(0); i < ordersLength; i++ {
		swap.Params.Orders[i].Salt, startByte = pack.ReadBigInt(data, startByte)
		swap.Params.Orders[i].MakerAsset, startByte = pack.ReadAddress(data, startByte)
		swap.Params.Orders[i].TakerAsset, startByte = pack.ReadAddress(data, startByte)
		swap.Params.Orders[i].Maker, startByte = pack.ReadAddress(data, startByte)
		swap.Params.Orders[i].Receiver, startByte = pack.ReadAddress(data, startByte)
		swap.Params.Orders[i].AllowedSender, startByte = pack.ReadAddress(data, startByte)
		swap.Params.Orders[i].MakingAmount, startByte = pack.ReadBigInt(data, startByte)
		swap.Params.Orders[i].TakingAmount, startByte = pack.ReadBigInt(data, startByte)
		swap.Params.Orders[i].FeeConfig, startByte = pack.ReadUInt200(data, startByte)
		swap.Params.Orders[i].MakerAssetData, startByte = pack.ReadBytes(data, startByte)
		swap.Params.Orders[i].TakerAssetData, startByte = pack.ReadBytes(data, startByte)
		swap.Params.Orders[i].GetMakerAmount, startByte = pack.ReadBytes(data, startByte)
		swap.Params.Orders[i].GetTakerAmount, startByte = pack.ReadBytes(data, startByte)
		swap.Params.Orders[i].Predicate, startByte = pack.ReadBytes(data, startByte)
		swap.Params.Orders[i].Interaction, startByte = pack.ReadBytes(data, startByte)
	}

	// unpack signature
	signaturesLength, startByte := pack.ReadUInt8(data, startByte)
	swap.Params.Signatures = make([]Signature, signaturesLength)
	for i := uint8(0); i < signaturesLength; i++ {
		swap.Params.Signatures[i].OrderSignature, startByte = pack.ReadBytes(data, startByte)
		swap.Params.Signatures[i].OpSignature, startByte = pack.ReadBytes(data, startByte)
	}

	opExpireTimesLength, startByte := pack.ReadUInt8(data, startByte)
	swap.Params.OpExpireTimes = make([]uint32, opExpireTimesLength)
	for i := uint8(0); i < opExpireTimesLength; i++ {
		swap.Params.OpExpireTimes[i], startByte = pack.ReadUInt32(data, startByte)
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

func buildKyberLimitOrderDS(swap types.L2EncodingSwap) (KyberLimitOrderDS, error) {
	byteData, err := json.Marshal(swap.Extra)
	if err != nil {
		return KyberLimitOrderDS{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildKyberLimitOrderDS] err :[%v]",
			err,
		)
	}

	var swapInfo limitorder.OpSignatureExtra
	if err = json.Unmarshal(byteData, &swapInfo); err != nil {
		return KyberLimitOrderDS{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[BuildKyberLimitOrderDS] err :[%v]",
			err,
		)
	}
	if len(swapInfo.FilledOrders) == 0 {
		return KyberLimitOrderDS{}, fmt.Errorf("[BuildKyberLimitOrderDS] cause by filledOrder is empty")
	}
	params, err := toFillBatchOrdersParamsDS(&swapInfo)
	if err != nil {
		return KyberLimitOrderDS{}, fmt.Errorf("[BuildKyberLimitOrderDS] error at toFillBatchOrdersParamsDS func error cause by %v", err)
	}
	return KyberLimitOrderDS{
		PoolMappingID:  swap.PoolMappingID,
		KyberLOAddress: common.HexToAddress(swap.Pool),
		MakerAsset:     common.HexToAddress(swapInfo.FilledOrders[0].MakerAsset),
		Params:         params,

		isFirstSwap: swap.IsFirstSwap,
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
			return FillBatchOrdersParamsDS{}, fmt.Errorf("operator signature not found for order %v", filledOrder.OrderID)
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

func packKyberLimitOrderDS(order KyberLimitOrderDS) ([]byte, error) {
	// Pack []OrderDS
	var orderDSsPacked []pack.RawBytes
	for _, order := range order.Params.Orders {
		orderDSPacked, err := pack.Pack(
			order.Salt,
			order.MakerAsset,
			order.TakerAsset,
			order.Maker,
			order.Receiver,
			order.AllowedSender,
			order.MakingAmount,
			order.TakingAmount,
			order.FeeConfig,
			order.MakerAssetData,
			order.TakerAssetData,
			order.GetMakerAmount,
			order.GetTakerAmount,
			order.Predicate,
			order.Interaction,
		)
		if err != nil {
			return nil, err
		}
		orderDSsPacked = append(orderDSsPacked, pack.RawBytes(orderDSPacked))
	}

	// pack []Signature
	var signaturesPacked []pack.RawBytes
	for _, signature := range order.Params.Signatures {
		signaturePacked, err := pack.Pack(
			signature.OrderSignature,
			signature.OpSignature,
		)

		if err != nil {
			return nil, err
		}
		signaturesPacked = append(signaturesPacked, pack.RawBytes(signaturePacked))
	}

	// Pack FillBatchOrdersParamsDS
	var params []interface{}

	params = append(params, orderDSsPacked, signaturesPacked, order.Params.OpExpireTimes)
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
