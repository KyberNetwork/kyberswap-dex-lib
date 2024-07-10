package ambient

import (
	"context"
	"encoding/json"
	"os"
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
	if os.Getenv("CI") != "" {
		t.Skip()
	}

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
		Base:    "0x0000000000000000000000000000000000000000",
		Quote:   "0xa0b73e1ff0b80914ab6fe0444e65848c4c34450b",
		PoolIdx: "420",
	}
	encodedStaticExtra, _ := json.Marshal(staticExtra)

	pool, err := tracker.GetNewPoolState(context.Background(), entity.Pool{
		Address:     "0x018b1f1a6fa4d7cebf8a2ea31bf76d0bc52c112bf75a4969f0e1596db32a826c",
		StaticExtra: string(encodedStaticExtra),
	}, pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	spew.Dump(pool)
}
