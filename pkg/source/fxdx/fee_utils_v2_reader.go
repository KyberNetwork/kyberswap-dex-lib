package fxdx

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type FeeUtilsV2Reader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
}

func NewFeeUtilsV2Reader(ethrpcClient *ethrpc.Client) *FeeUtilsV2Reader {
	return &FeeUtilsV2Reader{
		abi:          feeUtilsABI,
		ethrpcClient: ethrpcClient,
	}
}

func (r *FeeUtilsV2Reader) Read(ctx context.Context, vault *Vault) (*FeeUtilsV2, error) {
	feeUtils := NewFeeUtilsV2()

	var (
		address = vault.FeeUtils.Hex()

		boolValues    = make([]bool, 2)
		addressValues = make([]string, 2)
		intValues     = make([]*big.Int, 3+len(vault.WhitelistedTokens))

		tokens = make([]common.Address, len(vault.WhitelistedTokens))

		isInitialized bool
	)

	for i, token := range vault.WhitelistedTokens {
		tokens[i] = common.HexToAddress(token)
	}

	request := r.ethrpcClient.R()

	request.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: feeUtilsV2MethodGetStates,
		Params: []interface{}{tokens},
	}, []interface{}{&addressValues, &intValues, &boolValues})

	request.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: feeUtilsV2IsInitialized,
	}, []interface{}{&isInitialized})

	if _, err := request.Call(); err != nil {
		return nil, err
	}

	feeUtils.HasDynamicFees = boolValues[0]
	feeUtils.IsActive = boolValues[1]

	return feeUtils, nil
}
