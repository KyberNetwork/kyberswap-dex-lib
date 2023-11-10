package levelfinance

import (
	"bytes"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	LiquidityPoolAbi       abi.ABI
	LevelOracleABI         abi.ABI
	LiquidityCalculatorABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&LiquidityPoolAbi, LiquidityPoolABIBytes},
		{&LevelOracleABI, LevelOracleABIBytes},
		{&LiquidityCalculatorABI, LiquidityCalculatorABIBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
