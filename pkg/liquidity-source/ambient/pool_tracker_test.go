package ambient

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	rpcURL           = "http://localhost:8545"
	multicallAddress = "0x5ba1e12693dc8f9c48aad8770482f4739beed696" // UniswapV3: Multicall 2
)

func TestPoolTracker(t *testing.T) {
	t.Skip()

	cfg := Config{
		DexID:                    DexTypeAmbient,
		QueryContractAddress:     "0xCA00926b6190c2C59336E73F02569c356d7B6b56",
		SwapDexContractAddress:   "0xAaAaAAAaA24eEeb8d57D431224f73832bC34f688",
		MulticallContractAddress: multicallAddress,
	}

	client := ethrpc.New(rpcURL)
	client.SetMulticallContract(common.HexToAddress(multicallAddress))
	tracker := NewPoolTracker(cfg, client)

	staticExtra := StaticExtra{
		Base:        "0x0000000000000000000000000000000000000000",
		Quote:       "0xdac17f958d2ee523a2206206994597c13d831ec7",
		PoolIdx:     "420",
		SwapAddress: cfg.SwapDexContractAddress,
	}
	encodedStaticExtra, _ := json.Marshal(staticExtra)
	poolEntity := entity.Pool{
		Address:     "0x471e4d62cdce74782d2e37637bc454bb698cbec66c8f97c118ea96ee38e857da",
		StaticExtra: string(encodedStaticExtra),
	}

	pool, err := tracker.GetNewPoolState(context.Background(), poolEntity, pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	spew.Dump(pool)
}
