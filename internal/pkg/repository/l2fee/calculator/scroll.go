package calculator

import (
	"bytes"
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	PRECISION, _ = new(big.Int).SetString("1000000000", 10)
)

// based on https://scrollscan.com/address/0x5300000000000000000000000000000000000002#code
func CalculateScrollL1Fee(params entity.ScrollL1FeeParams, unsignedTx *types.Transaction) (*big.Int, error) {
	var b bytes.Buffer
	err := unsignedTx.EncodeRLP(&b)
	if err != nil {
		return nil, err
	}
	txBytes := b.Bytes()
	l1GasUsed := getScrollL1GasUsed(txBytes, params.Overhead)
	l1Fee := new(big.Int).Div(
		new(big.Int).Mul(
			new(big.Int).Mul(l1GasUsed, params.L1BaseFee),
			params.Scalar,
		),
		PRECISION,
	)
	return l1Fee, nil
}

// The `_data` is the RLP-encoded transaction *without* signature
// we'll reserve 68 non-zero bytes for the signature, and 4 non-zero bytes to store the number of bytes in the RLP-encoded transaction
func getScrollL1GasUsed(data []byte, overhead *big.Int) *big.Int {
	total := big.NewInt(0)
	costZeroByte := big.NewInt(4)
	costNonZeroByte := big.NewInt(16)

	for i := 0; i < len(data); i++ {
		if data[i] == 0 {
			total.Add(total, costZeroByte)
		} else {
			total.Add(total, costNonZeroByte)
		}
	}

	total.Add(total, overhead)
	total.Add(total, big.NewInt((68+4)*16))
	return total
}
