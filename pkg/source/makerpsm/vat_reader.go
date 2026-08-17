package makerpsm

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

type VatReader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
}

func NewVatReader(ethrpcClient *ethrpc.Client) *VatReader {
	return &VatReader{
		abi:          makerPSMVat,
		ethrpcClient: ethrpcClient,
	}
}

func (r *VatReader) Read(ctx context.Context, address string, ilk [32]byte, overrides map[common.Address]gethclient.OverrideAccount) (*Vat, error) {
	var vat Vat

	req := r.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: vatMethodDebt,
			Params: nil,
		}, []any{&vat.Debt}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: vatMethodLine,
			Params: nil,
		}, []any{&vat.Line}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: vatMethodIlks,
			Params: []any{ilk},
		}, []any{&vat.ILK}).SetOverrides(overrides)
	resp, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": DexTypeMakerPSM,
			"error": err,
		}).Error("eth rpc call error")
		return nil, err
	}
	if resp != nil && resp.BlockNumber != nil {
		vat.BlockNumber = resp.BlockNumber.Uint64()
		vat.HasBlockNumber = true
	}

	return &vat, nil
}
