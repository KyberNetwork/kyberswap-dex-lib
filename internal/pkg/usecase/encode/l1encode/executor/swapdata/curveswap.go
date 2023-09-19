package swapdata

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackCurveSwap(chainID valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildCurveSwap(chainID, encodingSwap)
	if err != nil {
		return nil, err
	}

	return packCurveSwap(swap)
}

func UnpackCurveSwap(encodedSwap []byte) (CurveSwap, error) {
	unpacked, err := CurveSwapABIArguments.Unpack(encodedSwap)
	if err != nil {
		return CurveSwap{}, err
	}

	var swap CurveSwap
	if err = CurveSwapABIArguments.Copy(&swap, unpacked); err != nil {
		return CurveSwap{}, err
	}

	return swap, nil
}

func buildCurveSwap(chainID valueobject.ChainID, swap types.EncodingSwap) (CurveSwap, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return CurveSwap{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildCurveSwap] err :[%v]",
			err,
		)
	}

	var extra struct {
		TokenInIndex  int64 `json:"tokenInIndex"`
		TokenOutIndex int64 `json:"tokenOutIndex"`
		Underlying    bool  `json:"underlying"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return CurveSwap{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[BuildCurveSwap] err :[%v]",
			err,
		)
	}

	useTriCrypto := swap.PoolType == constant.PoolTypes.CurveTricrypto || swap.PoolType == constant.PoolTypes.CurveTwo

	tokenFrom := common.HexToAddress(swap.TokenIn)
	if !useTriCrypto && eth.IsWETH(swap.TokenIn, chainID) {
		tokenFrom = common.HexToAddress(valueobject.EtherAddress)
	}

	tokenTo := common.HexToAddress(swap.TokenOut)
	if !useTriCrypto && eth.IsWETH(swap.TokenOut, chainID) {
		tokenTo = common.HexToAddress(valueobject.EtherAddress)
	}

	return CurveSwap{
		Pool:              common.HexToAddress(swap.Pool),
		TokenFrom:         tokenFrom,
		TokenTo:           tokenTo,
		TokenIndexFrom:    big.NewInt(extra.TokenInIndex),
		TokenIndexTo:      big.NewInt(extra.TokenOutIndex),
		Dx:                swap.SwapAmount,
		UsePoolUnderlying: extra.Underlying,
		UseTriCrypto:      useTriCrypto,
	}, nil
}

func packCurveSwap(swap CurveSwap) ([]byte, error) {
	return CurveSwapABIArguments.Pack(
		swap.Pool,
		swap.TokenFrom,
		swap.TokenTo,
		swap.TokenIndexFrom,
		swap.TokenIndexTo,
		swap.Dx,
		swap.UsePoolUnderlying,
		swap.UseTriCrypto,
	)
}
