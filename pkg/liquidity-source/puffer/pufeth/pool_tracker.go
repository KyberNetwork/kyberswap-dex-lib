package pufeth

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	extra, blockNumber, err := t.getExtra(ctx)
	if err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) getExtra(ctx context.Context) (PoolExtra, uint64, error) {
	var (
		totalSupply      *big.Int
		totalAssets      *big.Int
		totalShares      *big.Int
		totalPooledEther *big.Int
	)

	getPoolStateRequest := t.ethrpcClient.NewRequest().SetContext(ctx)

	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    pufferVaultABI,
		Target: PUFETH,
		Method: PufferVaultMethodTotalSupply,
		Params: []interface{}{},
	}, []interface{}{&totalSupply})

	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    pufferVaultABI,
		Target: PUFETH,
		Method: PufferVaultMethodTotalAssets,
		Params: []interface{}{},
	}, []interface{}{&totalAssets})

	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    lidoABI,
		Target: STETH,
		Method: LidoMethodGetTotalShares,
		Params: []interface{}{},
	}, []interface{}{&totalShares})

	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    lidoABI,
		Target: STETH,
		Method: LidoMethodGetTotalPooledEther,
		Params: []interface{}{},
	}, []interface{}{&totalPooledEther})

	resp, err := getPoolStateRequest.TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}

	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	return PoolExtra{
		TotalSupply:      uint256.MustFromBig(totalSupply),
		TotalAssets:      uint256.MustFromBig(totalAssets),
		TotalPooledEther: uint256.MustFromBig(totalPooledEther),
		TotalShares:      uint256.MustFromBig(totalShares),
	}, resp.BlockNumber.Uint64(), nil
}
