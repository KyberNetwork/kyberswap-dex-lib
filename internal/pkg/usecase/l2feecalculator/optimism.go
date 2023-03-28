package l2feecalculator

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type OptimismFeeCalculator struct {
	decimals  *big.Int
	l1BaseFee *big.Int
	overhead  *big.Int
	scalar    *big.Int
}

func NewOptimismFeeCalculator() *OptimismFeeCalculator {
	return &OptimismFeeCalculator{}
}

func (lf *OptimismFeeCalculator) SetParams(l2Fee *entity.L2Fee) {
	lf.decimals = l2Fee.Decimals
	lf.l1BaseFee = l2Fee.L1BaseFee
	lf.overhead = l2Fee.Overhead
	lf.scalar = l2Fee.Scalar
}

func (lf *OptimismFeeCalculator) CreateRawTxFromInputData(encodedSwapData string) ([]byte, error) {
	nonce := uint64(0)

	value := big.NewInt(DummyValue)   // in wei (1 eth)
	gasLimit := uint64(DummyGasLimit) // in units

	toAddress := common.HexToAddress(DummyToAddress)
	encodedSwapDataBytes := common.FromHex(encodedSwapData)
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, lf.l1BaseFee, encodedSwapDataBytes)

	chainID := big.NewInt(int64(valueobject.ChainIDOptimism))

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		logger.Errorf("failed to sign transaction, err: %v", err)
		return nil, err
	}

	ts := types.Transactions{signedTx}
	b := new(bytes.Buffer)
	ts.EncodeIndex(0, b)
	rawTxBytes := b.Bytes()

	return rawTxBytes, nil
}

// GetL1Fee Computes the L1 portion of the fee
// based on the size of the RLP encoded tx
// and the current l1BaseFee
// @param _data Unsigned RLP encoded tx, 6 elements
// @return L1 fee that should be paid for the tx
func (lf *OptimismFeeCalculator) GetL1Fee(_data []byte) *big.Int {
	l1GasUsed := lf.getL1GasUsed(_data)
	l1Fee := new(big.Int).Mul(l1GasUsed, lf.l1BaseFee)
	divisor := new(big.Int).Exp(big.NewInt(10), lf.decimals, nil)
	unscaled := new(big.Int).Mul(l1Fee, lf.scalar)
	scaled := new(big.Int).Div(unscaled, divisor)

	return scaled
}

// getL1GasUsed Computes the amount of L1 gas used for a transaction
// The overhead represents the per batch gas overhead of
// posting both transaction and state roots to L1 given larger
// batch sizes.
// 4 gas for 0 byte
// https://github.com/ethereum/go-ethereum/blob/9ada4a2e2c415e6b0b51c50e901336872e028872/params/protocol_params.go#L33
// 16 gas for non zero byte
// https://github.com/ethereum/go-ethereum/blob/9ada4a2e2c415e6b0b51c50e901336872e028872/params/protocol_params.go#L87
// This will need to be updated if calldata gas prices change
// Account for the transaction being unsigned
// Padding is added to account for lack of signature on transaction
// 1 byte for RLP V prefix
// 1 byte for V
// 1 byte for RLP R prefix
// 32 bytes for R
// 1 byte for RLP S prefix
// 32 bytes for S
// Total: 68 bytes of padding
// @param _data Unsigned RLP encoded tx, 6 elements
// @return Amount of L1 gas used for a transaction
// Contract: https://optimistic.etherscan.io/address/0x420000000000000000000000000000000000000F#code
func (lf *OptimismFeeCalculator) getL1GasUsed(_data []byte) *big.Int {
	total := constant.Zero

	for i := 0; i < len(_data); i++ {
		if _data[i] == 0 {
			total = new(big.Int).Add(total, big.NewInt(int64(params.TxDataZeroGas)))
		} else {
			total = new(big.Int).Add(total, big.NewInt(int64(params.TxDataNonZeroGasEIP2028)))
		}
	}

	unsigned := new(big.Int).Add(total, lf.overhead)

	return new(big.Int).Add(unsigned, big.NewInt(68*16))
}
