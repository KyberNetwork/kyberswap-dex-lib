package service

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/pkg/logger"
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

func (s *ScanService) TryAggregate(ctx context.Context, requireSuccess bool, calls []*repository.TryCallParams) (err error) {
	defer func() {
		if err != nil {
			targets := make([]string, 0, len(calls))
			methods := make([]string, 0, len(calls))

			for _, call := range calls {
				targets = append(targets, call.Target)
				methods = append(methods, call.Method)
			}

			handleRPCError(err, strings.Join(targets, "-"), strings.Join(methods, "-"), "TryAggregate")
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrapf(ErrRPCPanic, "[%v]", r)
		}
	}()

	err = s.rpcRepo.TryAggregate(ctx, requireSuccess, calls)

	return
}

func handleRPCError(err error, callTargets string, callMethods string, method string) {
	logger.WithFields(map[string]interface{}{
		"method":      method,
		"error":       err,
		"callTargets": callTargets,
		"callMethods": callMethods,
	}).Error("RPC call failed")
}
