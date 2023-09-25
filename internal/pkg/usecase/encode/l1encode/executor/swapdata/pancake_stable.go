package swapdata

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackPancakeStableSwap(chainID valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildPancakeStableSwap(chainID, encodingSwap)
	if err != nil {
		return nil, err
	}

	return packPancakeStableSwap(swap)
}

func UnpackPancakeStableSwap(encodedSwap []byte) (CurveSwap, error) {
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

// buildPancakeStableSwap: same with buildCurveSwap but not unwrap WETH
func buildPancakeStableSwap(_ valueobject.ChainID, swap types.EncodingSwap) (CurveSwap, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return CurveSwap{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildPancakeStableSwap] err :[%v]",
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
			"[BuildPancakeStableSwap] err :[%v]",
			err,
		)
	}

	return CurveSwap{
		Pool:              common.HexToAddress(swap.Pool),
		TokenFrom:         common.HexToAddress(swap.TokenIn),
		TokenTo:           common.HexToAddress(swap.TokenOut),
		TokenIndexFrom:    big.NewInt(extra.TokenInIndex),
		TokenIndexTo:      big.NewInt(extra.TokenOutIndex),
		Dx:                swap.SwapAmount,
		UsePoolUnderlying: extra.Underlying,
		UseTriCrypto:      false,
	}, nil
}

func packPancakeStableSwap(swap CurveSwap) ([]byte, error) {
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
