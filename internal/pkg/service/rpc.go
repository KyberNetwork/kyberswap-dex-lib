package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/model"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

const jobIntervalSec = 30

type RPCService struct {
	config     *config.Common
	activeRpcs []*model.RPC
	rpcs       []string
	//needUpdate bool
}
type IRPCService interface {
	ActiveRPC() []string
}

func NewRPC(config *config.Common) *RPCService {
	var ret = &RPCService{
		config: config,
	}
	ret.InitRpc()
	ret.updateStatus(context.Background())
	return ret
}
func (t *RPCService) InitRpc() {
	for _, rpcConfig := range t.config.RPCs {
		t.activeRpcs = append(t.activeRpcs, &model.RPC{
			URL:    rpcConfig,
			Active: true,
		})

	}

}
func (t *RPCService) GetEthClient(ctx context.Context) (*ethclient.Client, error) {
	rpcs := t.rpcs
	rpcsLen := len(rpcs)
	if rpcsLen == 0 {
		return nil, fmt.Errorf("no RPCs to connect")
	}

	indexRand, err := rand.Int(rand.Reader, big.NewInt(int64(rpcsLen)))
	if err != nil {
		return nil, err
	}
	index := int(indexRand.Int64())

	for i := 0; i < rpcsLen; i++ {
		rpc := rpcs[(index+i)%rpcsLen]
		client, err := ethclient.Dial(rpc)
		if err == nil {
			return client, nil
		}
		logger.Errorf("failed to connect %s, err: %v", rpc, err)
	}

	return nil, fmt.Errorf("failed to connect any RPCs")
}

func (t *RPCService) UpdateData(ctx context.Context) {
	for {
		err := t.updateStatus(ctx)
		if err != nil {
			logger.Errorf("failed to update err=%v", err)
		}
		time.Sleep(jobIntervalSec * time.Second)
	}
}

func (t *RPCService) updateStatus(ctx context.Context) error {

	timestamp := time.Now().Unix()
	defaultRPC, err := ethclient.Dial(t.config.PublicRPC)

	if err != nil {
		logger.Errorf("failed to load default rpc")
		return err
	}
	maxBlock, err := defaultRPC.BlockNumber(ctx)
	if err != nil {
		logger.Errorf("failed to get current block")
	}
	activeRpcs := t.activeRpcs

	var wg sync.WaitGroup
	wg.Add(len(activeRpcs))

	for _, rpc := range activeRpcs {
		go func(rpc *model.RPC) {
			defer wg.Done()
			logger.Debugf("checking %v", rpc.URL)
			client, err := ethclient.Dial(rpc.URL)
			ok := false
			if err == nil {
				block, err := client.BlockNumber(ctx)
				if err == nil {
					rpc.Block = block
					if block >= maxBlock || maxBlock-block <= 1 {
						ok = true
					}
				}
			}
			rpc.Status = ok
			rpc.UpdatedAt = timestamp
		}(rpc)
	}
	wg.Wait()
	var ret = make([]string, 0)
	for _, rpc := range t.activeRpcs {
		if rpc.Active {
			ret = append(ret, rpc.URL)
		}
	}
	logger.Infof("rpc active %d", len(ret))
	t.rpcs = ret
	return nil
}

func (t *RPCService) ActiveRPC() []string {
	return t.rpcs
}
