package swapdata

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackKokonutCrypto(chainID valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildKokonutCrypto(chainID, encodingSwap)
	if err != nil {
		return nil, err
	}

	return packKokonutCrypto(swap)
}

func UnpackKokonutCrypto(encodedSwap []byte) (KokonutCrypto, error) {
	unpacked, err := KokonutCryptoABIArguments.Unpack(encodedSwap)
	if err != nil {
		return KokonutCrypto{}, err
	}

	var swap KokonutCrypto
	if err = KokonutCryptoABIArguments.Copy(&swap, unpacked); err != nil {
		return KokonutCrypto{}, err
	}

	return swap, nil
}

func buildKokonutCrypto(_ valueobject.ChainID, swap types.EncodingSwap) (KokonutCrypto, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return KokonutCrypto{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildKokonutCrypto] err :[%v]",
			err,
		)
	}

	var extra struct {
		TokenInIndex int64 `json:"tokenInIndex"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return KokonutCrypto{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[BuildKokonutCrypto] err :[%v]",
			err,
		)
	}

	fromToken := common.HexToAddress(swap.TokenIn)
	toToken := common.HexToAddress(swap.TokenOut)

	return KokonutCrypto{
		Pool:           common.HexToAddress(swap.Pool),
		Dx:             swap.SwapAmount,
		TokenIndexFrom: big.NewInt(extra.TokenInIndex),
		FromToken:      fromToken,
		ToToken:        toToken,
	}, nil
}

func packKokonutCrypto(swap KokonutCrypto) ([]byte, error) {
	return KokonutCryptoABIArguments.Pack(
		swap.Pool,
		swap.Dx,
		swap.TokenIndexFrom,
		swap.FromToken,
		swap.ToToken,
	)
}
