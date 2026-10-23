package makerpsm

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	utilEth "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
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

func (r *VatReader) Read(ctx context.Context, address string, ilk [32]byte, overrides map[common.Address]gethclient.OverrideAccount, blockNumber ...*big.Int) (*Vat, error) {
	var vat Vat

	req := r.ethrpcClient.
		NewRequest().
		SetContext(ctx)
	if len(blockNumber) > 0 {
		req.SetBlockNumber(utilEth.BlockNumberOrLatest(blockNumber[0]))
	}
	req.
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
	var resp *ethrpc.Response
	var err error
	if len(blockNumber) > 0 && blockNumber[0] != nil {
		resp, err = utilEth.CallWithBlockRetry(ctx, req.Aggregate, blockNumber[0])
	} else {
		resp, err = req.Aggregate()
	}
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
