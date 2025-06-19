package brevis

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

func GetFee(ctx context.Context, hookAddress common.Address, ethrpcClient *ethrpc.Client) (*big.Int, error) {
	hookCaller, err := NewBrevisCaller(hookAddress, ethrpcClient.GetETHClient())
	if err != nil {
		return nil, err
	}

	origFee, err := hookCaller.OrigFee(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, err
	}

	print("GetFee: hookAddress: ", hookAddress.Hex(), " origFee: ", origFee.String())

	return origFee, nil
}
