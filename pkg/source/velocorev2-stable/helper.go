package velocorev2stable

import "github.com/ethereum/go-ethereum/common"

func decodeAddress(b bytes32) string {
	return common.BytesToAddress(b[:]).String() // TODO: check address
}
