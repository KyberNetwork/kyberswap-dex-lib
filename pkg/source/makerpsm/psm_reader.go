package makerpsm

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

type PSMReader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
}

func NewPSMReader(ethrpcClient *ethrpc.Client) *PSMReader {
	return &PSMReader{
		abi:          makerPSMPSM,
		ethrpcClient: ethrpcClient,
	}
}

func (r *PSMReader) Read(ctx context.Context, address string, overrides map[common.Address]gethclient.OverrideAccount) (*PSM, error) {
	var psm PSM

	req := r.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: psmMethodTIn,
			Params: nil,
		}, []interface{}{&psm.TIn}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: psmMethodTOut,
			Params: nil,
		}, []interface{}{&psm.TOut}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: psmMethodVat,
			Params: nil,
		}, []interface{}{&psm.VatAddress}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: psmMethodIlk,
			Params: nil,
		}, []interface{}{&psm.ILK})

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

	return &psm, nil
}
