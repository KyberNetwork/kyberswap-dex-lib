package l2feecalculator

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type ArbitrumFeeCalculator struct {
	l1BaseFee *big.Int
}

func NewArbitrumFeeCalculator() *ArbitrumFeeCalculator {
	return &ArbitrumFeeCalculator{}
}

func (lf *ArbitrumFeeCalculator) SetParams(l2Fee *entity.L2Fee) {
	lf.l1BaseFee = l2Fee.L1BaseFee
}

func (lf *ArbitrumFeeCalculator) CreateRawTxFromInputData(encodedSwapData string) ([]byte, error) {
	nonce := uint64(0)

	value := big.NewInt(DummyValue)   // in wei (1 eth)
	gasLimit := uint64(DummyGasLimit) // in units

	toAddress := common.HexToAddress(DummyToAddress)
	encodedSwapDataBytes := common.FromHex(encodedSwapData)
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, lf.l1BaseFee, encodedSwapDataBytes)

	chainID := big.NewInt(int64(valueobject.ChainIDArbitrumOne))

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
// Check this article: https://medium.com/offchainlabs/understanding-arbitrum-2-dimensional-fees-fd1d582596c9
// @param _data Unsigned RLP encoded tx, 6 elements
// @return L1 fee that should be paid for the tx
func (lf *ArbitrumFeeCalculator) GetL1Fee(_data []byte) *big.Int {
	l1GasUsed := lf.getL1GasUsed(_data)
	l1Fee := new(big.Int).Mul(l1GasUsed, lf.l1BaseFee)

	return l1Fee
}

// getL1GasUsed Computes the amount of L1 gas used for a transaction
func (lf *ArbitrumFeeCalculator) getL1GasUsed(_data []byte) *big.Int {
	// In Arbitrum, all bytes cost 16 gas
	// https://github.com/OffchainLabs/nitro/blob/9c6648e250db9c2d064136ef6a0aeb5512130b23/precompiles/ArbGasInfo.go#L75
	return new(big.Int).Mul(
		big.NewInt(int64(len(_data))),
		big.NewInt(int64(params.TxDataNonZeroGasEIP2028)))
}
