package pufeth

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdater struct {
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

var _ = poollist.RegisterFactoryE(DexType, NewPoolListUpdater)

func NewPoolListUpdater(
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	return &PoolListUpdater{
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}

	u.hasInitialized = true

	extra, blockNumber, err := u.getExtra(ctx)
	if err != nil {
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, nil, err
	}

	return []entity.Pool{
		{
			Address:   strings.ToLower(PufferDepositor),
			Exchange:  string(valueobject.ExchangePufferPufETH),
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{reserves, reserves, reserves},
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(PUFETH),
					Symbol:    "pufETH",
					Decimals:  18,
					Name:      "pufETH",
					Swappable: true,
				},
				{
					Address:   strings.ToLower(STETH),
					Symbol:    "stETH",
					Decimals:  18,
					Name:      "Liquid staked Ether 2.0",
					Swappable: true,
				},
				{
					Address:   strings.ToLower(WSTETH),
					Symbol:    "wstETH",
					Decimals:  18,
					Name:      "Wrapped liquid staked Ether 2.0 ",
					Swappable: true,
				},
			},
			BlockNumber: blockNumber,
			Extra:       string(extraBytes),
		},
	}, nil, nil
}

func (u *PoolListUpdater) getExtra(ctx context.Context) (PoolExtra, uint64, error) {
	var (
		totalSupply      *big.Int
		totalAssets      *big.Int
		totalShares      *big.Int
		totalPooledEther *big.Int
	)

	getPoolStateRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

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
