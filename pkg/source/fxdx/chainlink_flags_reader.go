package fxdx

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type ChainlinkFlagsReader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
	log          logger.Logger
}

func NewChainlinkFlagsReader(ethrpcClient *ethrpc.Client) *ChainlinkFlagsReader {
	return &ChainlinkFlagsReader{
		abi:          chainlinkABI,
		ethrpcClient: ethrpcClient,
		log: logger.WithFields(logger.Fields{
			"dexType": DexTypeFxdx,
			"reader":  "ChainlinkFlagsReader",
		}),
	}
}

func (r *ChainlinkFlagsReader) Read(ctx context.Context, address string) (*ChainlinkFlags, error) {
	var value bool

	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: chainlinkFlagsMethodGetFlag,
		Params: []interface{}{common.HexToAddress(flagArbitrumSeqOffline)},
	}, []interface{}{&value})

	if _, err := rpcRequest.Call(); err != nil {
		r.log.Errorf("error when call rpc: %s", err)
		return nil, err
	}

	return &ChainlinkFlags{
		Flags: map[string]bool{
			flagArbitrumSeqOffline: value,
		},
	}, nil
}
