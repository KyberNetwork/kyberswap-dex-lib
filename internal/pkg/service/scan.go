package service

import (
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
)

type ScanService struct {
	rpcRepo repository.IRPCRepository
}

func NewScanService(
	rpcRepo repository.IRPCRepository,
) *ScanService {
	var ret = &ScanService{
		rpcRepo: rpcRepo,
	}

	return ret
}

func GetPairAddressKey(a, b string) string {
	if a > b {
		return a + "-" + b
	}
	return b + "-" + a
}
