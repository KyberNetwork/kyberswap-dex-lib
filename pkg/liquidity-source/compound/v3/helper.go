package v3

import "github.com/KyberNetwork/kutils"

func parseUint64(s string) uint64 {
	res, _ := kutils.Atou[uint64](s)
	return res
}
