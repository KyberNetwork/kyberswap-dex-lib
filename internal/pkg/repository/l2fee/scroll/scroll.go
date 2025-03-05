package scroll

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

func EstimateL1Fees(params *entity.ScrollL1FeeParams) (*big.Int, *big.Int) {
	return calcCurieL1Fee(params, rlpDataLenOverhead), calcCurieL1FeeWithoutCommitFee(params, rlpDataLenPerPool)
}

func CalcCurieL1Fee(params *entity.ScrollL1FeeParams, unsignedTx *types.Transaction) (*big.Int, error) {
	var b bytes.Buffer
	if err := unsignedTx.EncodeRLP(&b); err != nil {
		return nil, err
	}
	l1Fee := calcCurieL1Fee(params, len(b.Bytes()))
	return l1Fee, nil
}

// CalcCurieL1Fee based on L1GasPriceOracle, after Curie fork
// https://scrollscan.com/address/0x5300000000000000000000000000000000000002#code
func calcCurieL1Fee(params *entity.ScrollL1FeeParams, rlpDataLen int) (l1Fee *big.Int) {
	// l1Fee = (commitScalar * l1BaseFee + blobScalar * rlpDataLen * l1BlobBaseFee) / precision

	l1Fee = new(big.Int).SetInt64(int64(rlpDataLen))
	l1Fee.Mul(params.L1BlobScalar, l1Fee)
	l1Fee.Mul(l1Fee, params.L1BlobBaseFee)

	l1Fee.Add(l1Fee, new(big.Int).Mul(params.L1CommitScalar, params.L1BaseFee))

	l1Fee.Div(l1Fee, precision)

	return l1Fee
}

func calcCurieL1FeeWithoutCommitFee(params *entity.ScrollL1FeeParams, rlpDataLen int) *big.Int {
	// l1FeeWithoutCommitFee = (blobScalar * rlpDataLength * l1BlobBaseFee) / precision

	fee := new(big.Int).SetInt64(int64(rlpDataLen))
	fee.Mul(params.L1BlobScalar, fee)
	fee.Mul(fee, params.L1BlobBaseFee)

	fee.Div(fee, precision)

	return fee
}
