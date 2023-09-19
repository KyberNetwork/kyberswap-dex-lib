package swapdata

import (
	"encoding/json"
	"math/big"

	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackKyberRFQ(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	kyberRFQ, err := buildKyberRFQ(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packKyberRFQ(kyberRFQ)
}

func UnpackKyberRFQ(encodedSwap []byte) (KyberRFQ, error) {
	unpacked, err := KyberRFQABIArguments.Unpack(encodedSwap)
	if err != nil {
		return KyberRFQ{}, err
	}

	var swap struct {
		KyberRFQ
	}
	if err := KyberRFQABIArguments.Copy(&swap, unpacked); err != nil {
		return KyberRFQ{}, err
	}

	return swap.KyberRFQ, nil
}

func buildKyberRFQ(swap types.EncodingSwap) (KyberRFQ, error) {
	byteData, err := json.Marshal(swap.Extra)
	if err != nil {
		return KyberRFQ{}, errors.Wrapf(
			ErrMarshalFailed,
			"[buildKyberRFQ] err :[%v]",
			err,
		)
	}

	var rfqExtra kyberpmm.RFQExtra
	if err = json.Unmarshal(byteData, &rfqExtra); err != nil {
		return KyberRFQ{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[buildKyberRFQ] err :[%v]",
			err,
		)
	}

	// The contract requires these values to be uint256, so we have to convert them to *big.Int
	info, _ := new(big.Int).SetString(rfqExtra.Info, 10)
	makingAmount, _ := new(big.Int).SetString(rfqExtra.MakerAmount, 10)
	takingAmount, _ := new(big.Int).SetString(rfqExtra.TakerAmount, 10)

	orderRFQ := OrderRFQ{
		Info:          info,
		MakerAsset:    common.HexToAddress(rfqExtra.MakerAsset),
		TakerAsset:    common.HexToAddress(rfqExtra.TakerAsset),
		Maker:         common.HexToAddress(rfqExtra.Maker),
		AllowedSender: common.Address{}, // null address on public orders
		MakingAmount:  makingAmount,
		TakingAmount:  takingAmount,
	}

	return KyberRFQ{
		RFQ:       common.HexToAddress(rfqExtra.RFQContractAddress),
		Order:     orderRFQ,
		Signature: common.FromHex(rfqExtra.Signature),
		Amount:    takingAmount,
		Target:    common.HexToAddress(swap.Recipient),
	}, nil
}

func packKyberRFQ(kyberRFQ KyberRFQ) ([]byte, error) {
	return KyberRFQABIArguments.Pack(
		kyberRFQ,
	)
}
