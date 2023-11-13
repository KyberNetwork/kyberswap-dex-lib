package fxdx

import (
	"context"
	"math/big"
	"strings"

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
		tokens  = make([]common.Address, len(vault.WhitelistedTokens))

		isInitialized bool

		boolValues    = make([]bool, 2)
		addressValues = make([]common.Address, 2)
		intValues     = make([]*big.Int, 3+len(vault.WhitelistedTokens))

		getStateResponse = []interface{}{&addressValues, &intValues, &boolValues}
	)

	for i, token := range vault.WhitelistedTokens {
		tokens[i] = common.HexToAddress(token)
	}

	request := r.ethrpcClient.R()

	request.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: feeUtilsV2IsInitialized,
	}, []interface{}{&isInitialized})

	request.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: feeUtilsV2MethodGetStates,
		Params: []interface{}{tokens},
	}, []interface{}{&getStateResponse})

	if _, err := request.Aggregate(); err != nil {
		return nil, err
	}

	feeUtils.Address = address

	feeUtils.IsInitialized = isInitialized
	feeUtils.IsActive = boolValues[1]
	feeUtils.FeeMultiplierIfInactive = intValues[2]
	feeUtils.HasDynamicFees = boolValues[0]

	index := 3
	for _, token := range tokens {
		tokenAddr := strings.ToLower(token.Hex())
		feeUtils.TaxBasisPoints[tokenAddr] = intValues[index]
		feeUtils.SwapFeeBasisPoints[tokenAddr] = intValues[index+2]
	}

	return feeUtils, nil
}
