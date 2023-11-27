package buildroute

import (
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func NewUnsignedTransaction(sender string, recipient string, data string,
	value *big.Int, gasPrice *big.Int) UnsignedTransaction {
	return UnsignedTransaction{
		sender,
		recipient,
		data,
		value,
		nil,
	}
}

func ConvertTransactionToMsg(tx UnsignedTransaction) ethereum.CallMsg {
	var (
		from           = common.HexToAddress(tx.sender)
		to             = common.HexToAddress(tx.recipient)
		encodedData, _ = hexutil.Decode(tx.data)
	)
	return ethereum.CallMsg{
		From:       from,
		To:         &to,
		Gas:        0,
		GasPrice:   tx.gasPrice,
		GasFeeCap:  nil,
		GasTipCap:  nil,
		Value:      tx.value,
		Data:       encodedData,
		AccessList: nil,
	}
}
