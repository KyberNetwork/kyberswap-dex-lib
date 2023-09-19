package swapdata

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type DODO struct {
	Recipient     common.Address
	PoolMappingID pack.UInt24
	Pool          common.Address
	Amount        *big.Int
	SellHelper    common.Address
	IsSellBase    bool
	IsVersion2    bool

	recipientFlag uint8
	isFirstSwap   bool
}

const DodoV1ExtraType = "CLASSICAL"

func PackDODO(_ valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap, err := buildDODO(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packDODO(swap)
}

func UnpackDODO(data []byte, isFirstSwap bool) (DODO, error) {
	var swap DODO
	var startByte int

	swap.recipientFlag, startByte = pack.ReadUInt8(data, startByte)
	if swap.recipientFlag == 0 {
		swap.Recipient, startByte = pack.ReadAddress(data, startByte)
	} else {
		swap.Recipient = common.BytesToAddress([]byte{swap.recipientFlag})
	}

	swap.PoolMappingID, startByte = pack.ReadUInt24(data, startByte)
	if swap.PoolMappingID == 0 {
		swap.Pool, startByte = pack.ReadAddress(data, startByte)
	}

	if isFirstSwap {
		swap.Amount, startByte = pack.ReadBigInt(data, startByte)
		swap.isFirstSwap = true
	} else {
		var collectAmountFlag bool
		collectAmountFlag, startByte = pack.ReadBoolean(data, startByte)
		if collectAmountFlag {
			swap.Amount = abi.MaxUint256
		}
	}

	swap.SellHelper, startByte = pack.ReadAddress(data, startByte)
	swap.IsSellBase, startByte = pack.ReadBoolean(data, startByte)
	swap.IsVersion2, _ = pack.ReadBoolean(data, startByte)

	return swap, nil
}

func buildDODO(swap types.L2EncodingSwap) (DODO, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return DODO{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildDODO] err :[%v]",
			err,
		)
	}

	var extra struct {
		Type             string `json:"type"`
		DodoV1SellHelper string `json:"dodoV1SellHelper"`
		BaseToken        string `json:"baseToken"`
		QuoteToken       string `json:"quoteToken"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return DODO{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[BuildDODO] err :[%v]",
			err,
		)
	}

	isSellBase := swap.TokenIn != extra.QuoteToken
	isVersion2 := extra.Type != DodoV1ExtraType

	return DODO{
		Recipient:     common.HexToAddress(swap.Recipient),
		PoolMappingID: swap.PoolMappingID,
		Pool:          common.HexToAddress(swap.Pool),
		Amount:        swap.SwapAmount,
		SellHelper:    common.HexToAddress(extra.DodoV1SellHelper),
		IsSellBase:    isSellBase,
		IsVersion2:    isVersion2,

		recipientFlag: swap.RecipientFlag,
		isFirstSwap:   swap.IsFirstSwap,
	}, nil
}

func packDODO(swap DODO) ([]byte, error) {
	var args []interface{}

	args = append(args, swap.recipientFlag)
	if swap.recipientFlag == 0 {
		args = append(args, swap.Recipient)
	}

	args = append(args, swap.PoolMappingID)
	if swap.PoolMappingID == 0 {
		args = append(args, swap.Pool)
	}

	if swap.isFirstSwap {
		args = append(args, swap.Amount)
	} else {
		var collectAmountFlag bool
		if swap.Amount.Cmp(constant.Zero) > 0 {
			collectAmountFlag = true
		}
		args = append(args, collectAmountFlag)
	}

	args = append(args, swap.SellHelper, swap.IsSellBase, swap.IsVersion2)

	return pack.Pack(args...)
}
