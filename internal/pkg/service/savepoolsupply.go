package service

import (
	"context"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type savePoolSupply struct {
	scan      *ScanService
	scanEvent *ScanEventService
}

func NewSavePoolSupply(scan *ScanService, scanEvent *ScanEventService) *savePoolSupply {
	return &savePoolSupply{
		scan:      scan,
		scanEvent: scanEvent,
	}
}

func (t *savePoolSupply) UpdateData(ctx context.Context) {

	run := func() error {
		startTime := time.Now()
		pools := t.scanEvent.mintOrRemoveMap.Keys()
		bulk := 200
		sum := int32(0)

		for i := 0; i < len(pools); i += bulk {
			end := i + bulk
			if end > len(pools) {
				end = len(pools)
			}

			pools, err := t.scan.GetPoolsByAddresses(ctx, pools[i:end])
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			var calls = make([]*repository.CallParams, 0)
			var totalSupplies = make([]*big.Int, len(pools))
			for i, pool := range pools {
				lpToken := pool.GetLpToken()
				_, err := t.scan.FetchOrGetTokenType(ctx, lpToken, pool.Exchange, pool.Address)
				if err != nil {
					return err
				}
				calls = append(calls, &repository.CallParams{
					ABI:    abis.ERC20,
					Target: lpToken,
					Method: "totalSupply",
					Params: nil,
					Output: &totalSupplies[i],
				})
			}
			if err := t.scan.MultiCall(ctx, calls); err != nil {
				logger.Errorf("failed to process multicall, err: %v", err)
				return err
			}
			var count = 0
			for i, pool := range pools {
				err := t.scan.UpdatePoolSupply(ctx, pool.Address, totalSupplies[i].String())
				if err != nil {
					logger.Errorf("failed to save pool: %v err %v", pool.Address, err)
				} else {
					t.scanEvent.mintOrRemoveMap.Remove(pool.Address)
					count++
				}
			}
			atomic.AddInt32(&sum, int32(count))
		}
		logger.Infof("update total supply %v pairs in %v", sum, time.Since(startTime))
		return nil
	}
	for {
		err := run()
		if err != nil {
			logger.Errorf("can not update total supply err=%v", err)
		}
		time.Sleep(30 * time.Second)
	}
}
