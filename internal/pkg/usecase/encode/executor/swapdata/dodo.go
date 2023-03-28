package swapdata

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

const DodoV1ExtraType = "CLASSICAL"

func PackDODO(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildDODO(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packDODO(swap)
}

func UnpackDODO(encodedSwap []byte) (DODO, error) {
	unpacked, err := DODOABIArguments.Unpack(encodedSwap)
	if err != nil {
		return DODO{}, err
	}

	var swap DODO
	if err = DODOABIArguments.Copy(&swap, unpacked); err != nil {
		return DODO{}, err
	}

	return swap, nil
}

func buildDODO(swap types.EncodingSwap) (DODO, error) {
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

	isSellBase := true
	if swap.TokenIn == extra.QuoteToken {
		isSellBase = false
	}

	isVersion2 := false
	if extra.Type != DodoV1ExtraType {
		isVersion2 = true
	}

	return DODO{
		Recipient:       common.HexToAddress(swap.Recipient),
		Pool:            common.HexToAddress(swap.Pool),
		TokenFrom:       common.HexToAddress(swap.TokenIn),
		TokenTo:         common.HexToAddress(swap.TokenOut),
		Amount:          swap.SwapAmount,
		MinReceiveQuote: constant.Zero,
		SellHelper:      common.HexToAddress(extra.DodoV1SellHelper),
		IsSellBase:      isSellBase,
		IsVersion2:      isVersion2,
	}, nil
}

func packDODO(swap DODO) ([]byte, error) {
	return DODOABIArguments.Pack(
		swap.Recipient,
		swap.Pool,
		swap.TokenFrom,
		swap.TokenTo,
		swap.Amount,
		swap.MinReceiveQuote,
		swap.SellHelper,
		swap.IsSellBase,
		swap.IsVersion2,
	)
}
