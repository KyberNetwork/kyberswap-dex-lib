package midas

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	dataFeedABI        abi.ABI
	DepositVaultABI    abi.ABI
	RedemptionVaultABI abi.ABI

	depositInstantSelector [4]byte
	redeemInstantSelector  [4]byte
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&dataFeedABI, dataFeedBytes},
		{&DepositVaultABI, depositVaultBytes},
		{&RedemptionVaultABI, redemptionVaultBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	depositInstantSelector = [4]byte(DepositVaultABI.Methods["depositInstant"].ID)
	redeemInstantSelector = [4]byte(RedemptionVaultABI.Methods["redeemInstant"].ID)
}
