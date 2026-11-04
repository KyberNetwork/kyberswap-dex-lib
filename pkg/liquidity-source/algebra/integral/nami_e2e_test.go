package integral

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestNamiHistoricalTrackerQuote is an opt-in differential test against the Nami hop in
// Base transaction 0x4207ca5dbb2fc85e6eff3b41ed2eb3eb7727a503a4213712b729de4e974c7bee.
// The previous-block state is read through the real tracker. The two surrounding ticks
// are synthetic: neither the original swap nor the quote crosses an initialized tick.
func TestNamiHistoricalTrackerQuote(t *testing.T) {
	rpcURL := os.Getenv("NAMI_E2E_BASE_RPC")
	if rpcURL == "" {
		t.Skip("set NAMI_E2E_BASE_RPC to a Base archive RPC")
	}
	const timestamp = uint32(1793598911)
	oldTime := blockTimestamp
	blockTimestamp = func() uint32 { return timestamp }
	t.Cleanup(func() { blockTimestamp = oldTime })
	rpc, err := ethclient.DialContext(context.Background(), rpcURL)
	require.NoError(t, err)
	t.Cleanup(rpc.Close)
	tracker := &PoolTracker{
		EthrpcClient: ethrpc.NewWithClient(rpc).SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11")),
		config:       &Config{DexID: "nami-v4"},
	}
	p := entity.Pool{Address: "0x6dbf1772aee6f453c6dafe705bb388685d5df1ea", Tokens: []*entity.PoolToken{{Address: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"}, {Address: "0xb2000000000000000000008bc8786b856e61707c"}}}
	state, err := tracker.FetchRPCData(context.Background(), &p, 53404781, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(53404781), state.BlockNumber.Uint64())
	require.NotNil(t, state.NamiFee)
	sim := ps.CloneState().(*PoolSimulator)
	sim.globalState = state.State
	sim.liquidity = uint256.MustFromBig(state.Liquidity)
	sim.dynamicFee = state.DynamicFee
	sim.volatilityOracle = state.VolatilityOracle
	sim.timepoints = NewTimepointStorage(state.Timepoints)
	sim.namiFee = state.NamiFee
	sim.ticks, err = v3Entities.NewTickListDataProvider([]v3Entities.Tick{
		{Index: -18540, LiquidityGross: uint256.NewInt(1), LiquidityNet: int256.NewInt(1)},
		{Index: -17800, LiquidityGross: uint256.NewInt(1), LiquidityNet: int256.NewInt(-1)},
	}, 1)
	require.NoError(t, err)
	sim.tickMin, sim.tickMax = -18540, -17800
	tokens := sim.GetTokens()
	quote, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[0], Amount: big.NewInt(4505817)}, TokenOut: tokens[1]})
	require.NoError(t, err)
	require.Equal(t, big.NewInt(728168), quote.TokenAmountOut.Amount)
	t.Logf("block=%d quote=%s", state.BlockNumber.Uint64(), quote.TokenAmountOut.Amount)
	if os.Getenv("TENDERLY_API_KEY") == "" || os.Getenv("TENDERLY_ACCOUNT") == "" || os.Getenv("TENDERLY_PROJECT") == "" {
		t.Skip("tracker quote passed; set Tenderly credentials to compare against a replay")
	}
	replayNamiTenderly(t, quote.TokenAmountOut.Amount)
}

func replayNamiTenderly(t *testing.T, want *big.Int) {
	t.Helper()
	const input = "0x00bbeac203001bfb00833589fcd6edb6e08f4c7c32d4f71b54bda02913b2000000000000000000008bc8786b856e61707c42000000000000000000000000000000000000060a6dbf1772aee6f453c6dafe705bb388685d5df1ea01b305829bbb29f21f760c8ee23c021e603a2db4160f404c7b0000a000000a02c17501e53fbb7eb0e243c74afc6a9e01c265cf000000000023"
	payload := map[string]any{"network_id": "8453", "block_number": 53404782, "transaction_index": 80,
		"from": "0x477fd98c600c36f46e135376d1e276afc3472ed3", "to": "0x563203867b12cf2405f415042db11f6fa9fb937b",
		"input": input, "gas": 1833171, "gas_price": "5003752", "value": "0", "save": true, "save_if_fails": true, "simulation_type": "full"}
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	endpoint := fmt.Sprintf("https://api.tenderly.co/api/v1/account/%s/project/%s/simulate", os.Getenv("TENDERLY_ACCOUNT"), os.Getenv("TENDERLY_PROJECT"))
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(data))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Access-Key", os.Getenv("TENDERLY_API_KEY"))
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, resp.StatusCode, "Tenderly: %s", body)
	var result struct {
		Simulation struct {
			ID string `json:"id"`
		} `json:"simulation"`
		Transaction struct {
			Status bool `json:"status"`
			Info   struct {
				Logs []struct {
					Raw struct {
						Address string   `json:"address"`
						Topics  []string `json:"topics"`
						Data    string   `json:"data"`
					} `json:"raw"`
				} `json:"logs"`
			} `json:"transaction_info"`
		} `json:"transaction"`
	}
	require.NoError(t, json.Unmarshal(body, &result))
	require.True(t, result.Transaction.Status, "Tenderly transaction reverted")
	var found bool
	for _, log := range result.Transaction.Info.Logs {
		if !strings.EqualFold(log.Raw.Address, "0x6dbf1772aee6f453c6dafe705bb388685d5df1ea") || len(log.Raw.Topics) == 0 || !strings.EqualFold(log.Raw.Topics[0], "0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67") {
			continue
		}
		raw, err := hex.DecodeString(strings.TrimPrefix(log.Raw.Data, "0x"))
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(raw), 64)
		require.Equal(t, big.NewInt(4505817), new(big.Int).SetBytes(raw[:32]))
		out := new(big.Int).SetBytes(raw[32:64])
		out.Sub(new(big.Int).Lsh(big.NewInt(1), 256), out)
		require.Equal(t, want, out)
		found = true
	}
	require.True(t, found, "Nami swap log not found")
	t.Logf("Tenderly replay: https://dashboard.tenderly.co/%s/%s/simulator/%s", os.Getenv("TENDERLY_ACCOUNT"), os.Getenv("TENDERLY_PROJECT"), result.Simulation.ID)
}
