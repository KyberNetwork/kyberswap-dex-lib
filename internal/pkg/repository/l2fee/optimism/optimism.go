package optimism

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

func EstimateFjordL1Fees(params *entity.OptimismL1FeeParams) (*big.Int, *big.Int) {
	return calcFjordL1Fee(params, fastLZDataLenOverhead), calcFjordL1Fee(params, fastLZDataLenPerPool)
}

func CalcFjordL1Fee(params *entity.OptimismL1FeeParams, unsignedTx *types.Transaction) (*big.Int, error) {
	var b bytes.Buffer
	if err := unsignedTx.EncodeRLP(&b); err != nil {
		return nil, err
	}
	fastLZSize := big.NewInt(int64(FlzCompressLen(b.Bytes())) + 68)
	l1Fee := calcFjordL1Fee(params, fastLZSize)
	return l1Fee, nil
}

func calcFjordL1Fee(params *entity.OptimismL1FeeParams, fastLZSize *big.Int) (l1Fee *big.Int) {
	// feeScaled = baseFeeScalar * 16 * l1BaseFee + blobBaseFeeScalar * blobBaseFee;
	// l1Fee = estimatedSize * feeScaled / (10 ** (decimals * 2))

	feeScaled := new(big.Int).Mul(params.L1BaseFeeScalar, params.L1BaseFee)
	feeScaled.Mul(feeScaled, sixteen)
	feeScaled.Add(feeScaled, new(big.Int).Mul(params.L1BlobBaseFeeScalar, params.L1BlobBaseFee))

	estimatedSize := estimatedDASizeScaled(fastLZSize)

	l1Fee = new(big.Int).Mul(estimatedSize, feeScaled)
	l1Fee.Div(l1Fee, fjordDivisor)

	return l1Fee
}

func estimatedDASizeScaled(fastLZSize *big.Int) *big.Int {
	estimatedDASizeScaled := new(big.Int).Add(l1CostIntercept, new(big.Int).Mul(l1CostFastlzCoef, fastLZSize))

	if estimatedDASizeScaled.Cmp(minTransactionSizeScaled) < 0 {
		estimatedDASizeScaled.Set(minTransactionSizeScaled)
	}
	return estimatedDASizeScaled
}

func getL1GasUsed(estimatedDASizeScaled *big.Int) *big.Int {
	// l1GasUsed = estimatedSize * params.TxDataNonZeroGasEIP2028 / 1e6

	l1GasUsed := new(big.Int).Mul(estimatedDASizeScaled, sixteen)
	l1GasUsed.Div(l1GasUsed, oneMillion)

	return l1GasUsed
}

func EstimateEcotoneL1Fees(params *entity.OptimismL1FeeParams) (*big.Int, *big.Int) {
	return calcEcotoneL1Fee(params, ecotonL1GasOverhead), calcEcotoneL1Fee(params, ecotonL1GasPerPool)
}

func CalcEcotoneL1Fee(params *entity.OptimismL1FeeParams, unsignedTx *types.Transaction) (*big.Int, error) {
	var b bytes.Buffer
	if err := unsignedTx.EncodeRLP(&b); err != nil {
		return nil, err
	}
	l1GasUsed := getCalldataGas(b.Bytes())
	l1Fee := calcEcotoneL1Fee(params, big.NewInt(int64(l1GasUsed)))
	return l1Fee, nil
}

func calcEcotoneL1Fee(params *entity.OptimismL1FeeParams, l1GasUsed *big.Int) (l1Fee *big.Int) {
	// l1Fee = ((l1BaseFeeScalar * l1BaseFee * 16) + (l1BlobBaseFeeScalar * l1BlobBaseFee)) * l1GasUsed / (1e6 * 16)

	l1Fee = new(big.Int).Mul(params.L1BaseFeeScalar, params.L1BaseFee)
	l1Fee.Mul(l1Fee, sixteen)
	l1Fee.Add(l1Fee, new(big.Int).Mul(params.L1BlobBaseFeeScalar, params.L1BlobBaseFee))

	l1Fee.Mul(l1Fee, l1GasUsed)
	l1Fee.Div(l1Fee, ecotoneDivisor)

	return l1Fee
}

func getCalldataGas(data []byte) uint64 {
	var total uint64 = 0
	for _, b := range data {
		if b == 0 {
			total += 4
		} else {
			total += 16
		}
	}
	return total + (68 * 16)
}
