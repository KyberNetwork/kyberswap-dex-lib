package chain

import (
	"bytes"
	"math/big"

	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/multicall"
	"github.com/KyberNetwork/router-service/pkg/logger"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

type TokenInfo struct {
	Address  string
	Name     string
	Symbol   string
	Decimals uint8
}

func GetTokenInfo(ctx context.Context, address, multicallAddress string, rpcs []string) (TokenInfo, error) {
	var ret = TokenInfo{
		Address:  address,
		Name:     "",
		Symbol:   "",
		Decimals: 0,
	}
	var successName, successDecimals, successSymbol bool
	var mDecimals *big.Int
	var mSymbol, mName [32]byte
	calls := []*multicall.TryCallUnPack{
		{
			ABI:       abis.ERC20,
			UnpackABI: []abi.ABI{abis.ERC20, abis.ERC20DS},
			Target:    address,
			Method:    "decimals",
			Params:    nil,
			Output:    []interface{}{&ret.Decimals, &mDecimals},
			Success:   &successDecimals,
		},
		{
			ABI:       abis.ERC20,
			UnpackABI: []abi.ABI{abis.ERC20, abis.ERC20DS},
			Target:    address,
			Method:    "symbol",
			Params:    nil,
			Output:    []interface{}{&ret.Symbol, &mSymbol},
			Success:   &successSymbol,
		},
		{
			ABI:       abis.ERC20,
			UnpackABI: []abi.ABI{abis.ERC20, abis.ERC20DS},
			Target:    address,
			Method:    "name",
			Params:    nil,
			Output:    []interface{}{&ret.Name, &mName},
			Success:   &successName,
		},
	}
	if err := multicall.TryAggregateUnpack(ctx, multicallAddress, rpcs, false, calls); err != nil {
		return ret, err
	}

	if ret.Decimals == 0 && mDecimals != nil {
		ret.Decimals = uint8(mDecimals.Int64())
	}
	if len(ret.Symbol) == 0 {
		ret.Symbol = getString(mSymbol)
	}
	if len(ret.Name) == 0 {
		ret.Name = getString(mName)
	}
	if !successDecimals || !successSymbol {
		logger.Warnf("can not get decimals or symbol of address %s", address)
		return ret, nil
	}
	if !successName {
		ret.Name = ret.Symbol
	}
	return ret, nil
}

func getString(r interface{}) string {
	switch r := r.(type) {
	case string:
		return r
	case [32]byte:
		t := bytes.TrimRightFunc(r[:], func(r rune) bool {
			return r == 0
		})
		return string(t)
	}
	return ""
}
