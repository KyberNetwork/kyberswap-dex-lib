package ezeth

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	EzETHTokenABI        abi.ABI
	RestakeManagerABI    abi.ABI
	RenzoOracleABI       abi.ABI
	PriceFeedABI         abi.ABI
	StrategyManagerABI   abi.ABI
	OperatorDelegatorABI abi.ABI
	TokenOracleABI       abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&EzETHTokenABI, ezETHTokenABIJson,
		},
		{
			&RestakeManagerABI, restakeManagerABIJson,
		},
		{
			&RenzoOracleABI, renzoOracleABIJson,
		},
		{
			&PriceFeedABI, priceFeedABIJson,
		},
		{
			&StrategyManagerABI, strategyManagerABIJson,
		},
		{
			&OperatorDelegatorABI, operatorDelegatorABIJson,
		},
		{
			&TokenOracleABI, tokenOracleABIJson,
		},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
