package erc20balanceslot

import "github.com/ethereum/go-ethereum/common"

type IProbe interface {
	ProbeBalanceSlot(token common.Address) (common.Hash, error)
	GetWallet() common.Address
}
