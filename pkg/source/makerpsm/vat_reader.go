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
		}, []interface{}{&vat.Debt}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: vatMethodLine,
			Params: nil,
		}, []interface{}{&vat.Line}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: vatMethodIlks,
			Params: []interface{}{ilk},
		}, []interface{}{&vat.ILK})

	if overrides != nil {
		req.SetOverrides(overrides)
	}
	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": DexTypeMakerPSM,
			"error": err,
		}).Error("eth rpc call error")
		return nil, err
	}

	return &vat, nil
}
