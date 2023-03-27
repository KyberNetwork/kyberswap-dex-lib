package metavault

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
)

type ChainlinkFlagsReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewChainlinkFlagsReader(scanService *service.ScanService) *ChainlinkFlagsReader {
	return &ChainlinkFlagsReader{
		abi:         abis.MetavaultChainlinkFlags,
		scanService: scanService,
	}
}

func (r *ChainlinkFlagsReader) Read(ctx context.Context, address string) (*ChainlinkFlags, error) {
	var value bool

	err := r.scanService.Call(ctx, &repository.CallParams{
		ABI:    r.abi,
		Target: address,
		Method: ChainlinkFlagsMethodGetFlag,
		Params: []interface{}{common.HexToAddress(FlagArbitrumSeqOffline)},
		Output: &value,
	})
	if err != nil {
		return nil, err
	}

	return &ChainlinkFlags{
		Flags: map[string]bool{
			FlagArbitrumSeqOffline: value,
		},
	}, nil
}
