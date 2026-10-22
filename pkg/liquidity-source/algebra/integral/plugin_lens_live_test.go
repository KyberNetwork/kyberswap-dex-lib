package integral

import (
	"bytes"
	"context"
	"io"
	"math/big"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

// pluginLiveDex is a deployment whose plugin reads TestPluginLensLive checks, at pools known to be
// live, with the multicall and plugin settings production reads it with.
type pluginLiveDex struct {
	dexID           string
	rpcEnv          string
	multicall       string
	useBasePluginV2 bool
	pools           []string
}

const multicall3 = "0xcA11bde05977b3631167028862bE2a173976CA11"

var pluginLiveDexes = []pluginLiveDex{
	{"supernova-v4", "ETHEREUM_RPC_URL", multicall3, false, []string{"0x2beb35e78c9427899353c41c96bcc96c5647ec63",
		"0xde758db54c1b4a87b06b34b30ef0a710dc35388f", "0xe2cd94c812aed174b2e09750ff146f45d845d668",
		"0xed1050e2fe96f7327dc659613c79e41458e9cf05"}},
	{"cypher-v4", "ETHEREUM_RPC_URL", multicall3, true, []string{"0x177f07c0843776b2a6342ed6c488af64e6f2fd65",
		"0x76a278bd71f566ee6ba2fe438f6099c8d8f98f43", "0x915fd34cadd63907b51eb64dddc2eadd114a0bed",
		"0xdb90e2723e998cf3072d1efbbcdaab4e24d8c640"}},
	{"quickswap-v4", "BASE_RPC_URL", multicall3, false, []string{"0x5a9ad2bb92b0b3e5c571fdd5125114e04e02be1a",
		"0x603f3fd0247e5a444a561d7da081c3b00fcf7de9", "0x846e89cffc54160a7e4acbeecc4f99c2528d1ca8",
		"0xbe50cc9cf0905cac4f8f8453bb6c9798898658bd"}},
	{"hydrex-v4", "BASE_RPC_URL", multicall3, false, []string{"0x0ba69825c4c033e72309f6ac0bde0023b15cc97c",
		"0x3f9b863ef4b295d6ba370215bcca3785fcc44f44", "0x82dbe18346a8656dbb5e76f74bf3ae279cc16b29",
		"0xc3f617c8a3da6d286c8f97af946a8f293632c852"}},
	{"thena-fusion-v3", "BSC_RPC_URL", multicall3, true, []string{"0x6ee903d7ee90270338ea833f6f3d0bb08abf93be",
		"0x7491c04dc4575e086a8ee31f7ce1c6d56fb7dcc1", "0x7d1d3649232f28bf7467e9cdb27e6d902c16a165",
		"0x8829abfa1a7b017078195c10a966d7411a0c9515", "0xc268ee337543a62115d46109d6771f1cf068063b",
		"0xd1d3b61504ac6f3fe2aac12641cddaf1ec8bf91a", "0xfc371b6c64837a96bd35185a98a69afa2525f3e7"}},
	{"camelot-v4", "ARBITRUM_RPC_URL", "0x7eCfBaa8742fDf5756DAC92fbc8b90a19b8815bF", true, []string{
		"0x20cdea535dcf2b93bce5e4db8786ac5163e28d81", "0x7529fe052324e377c83e49af304258cf524460bb",
		"0xb1e3451b8ecbf1dede4d7748876f3d7589a0d9fc", "0xee9c3bc28c325fb82e78fa68e3c6e521240433e5"}},
	{"swap-x-cl", "SONIC_RPC_URL", multicall3, false, []string{"0x0d13400cc7c46d77a43957fe614ba58c827dfde6",
		"0x5c4b7d607aaf7b5cde9f09b5f03cf3b5c923aeea", "0x5ddbef774488cc68266d5f15bfb08eaa7cd513f9",
		"0xd760791b29e7894fb827a94ca433254bb5afb653"}},
	{"kittenswap-v4", "ROBINHOOD_RPC_URL", "0x2cAC2D899eCC914d704FeaAE33ac1bF36277DaD1", false, []string{
		"0x3146b7c7fc7da5b3268fc7ff533054c0e3a7a88e", "0x3b337fcb8ad70e95c2948849900f5849456d1ff9",
		"0x99caa6ade20d0f8f64f405e80c0f1ea1b8275bab", "0xdaeacb593e66f87a04582879f51fd6ebf1a4dac0"}},
	{"nami-v4", "ROBINHOOD_RPC_URL", "0x2cAC2D899eCC914d704FeaAE33ac1bF36277DaD1", false, []string{
		"0xa730c5d09dd9926f798530d52e572d7594b952f3"}},
}

// TestPluginLensLive: for live pools of each deployment, the plugin read through the lens is the
// multicalls' at the same block, both for a pool read for the first time and for one refreshed
// from what a read ten minutes earlier kept, and it takes one call. Each deployment runs when its
// RPC URL is set, such as ETHEREUM_RPC_URL; with PLUGIN_LENS_GAS set, the gas of every call is
// measured too.
func TestPluginLensLive(t *testing.T) {
	for _, dex := range pluginLiveDexes {
		t.Run(dex.dexID, func(t *testing.T) {
			url := os.Getenv(dex.rpcEnv)
			if url == "" {
				t.Skipf("%s is not set", dex.rpcEnv)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Minute)
			defer cancel()

			calls := &recordCalls{next: retryOn429{}}
			raw, err := gethrpc.DialOptions(ctx, url, gethrpc.WithHTTPClient(&http.Client{Transport: calls}))
			require.NoError(t, err)
			client := ethclient.NewClient(raw)
			rpc := ethrpc.NewWithClient(client).SetMulticallContract(common.HexToAddress(dex.multicall))
			byRPC := NewPoolTracker(&Config{DexID: dex.dexID, UseBasePluginV2: dex.useBasePluginV2}, rpc, nil)
			byLens := NewPoolTracker(&Config{DexID: dex.dexID, UseBasePluginV2: dex.useBasePluginV2, UseLens: true},
				rpc, nil)
			// The node's gas cap is probed once per client, which no read would count.
			_, err = ephemeral.GasCap(ctx, client)
			require.NoError(t, err)

			head, err := client.HeaderByNumber(ctx, nil)
			require.NoError(t, err)
			block := new(big.Int).Sub(head.Number, big.NewInt(3))
			header, err := client.HeaderByNumber(ctx, block)
			require.NoError(t, err)
			earlier := blockAgo(ctx, t, client, header, 10*time.Minute)

			for _, address := range dex.pools {
				checkPluginPool(ctx, t, client, raw, calls, byRPC, byLens, address, header, earlier)
			}
		})
	}
}

// checkPluginPool reads a pool's plugin at header both ways, with no timepoints and with those a read
// at earlier kept, and requires the same result.
func checkPluginPool(ctx context.Context, t *testing.T, client *ethclient.Client, raw *gethrpc.Client,
	calls *recordCalls,
	byRPC, byLens *PoolTracker, address string, header, earlier *ethtypes.Header) {
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: new(common.HexToAddress(address)),
		Data: poolV12ABI.Methods[poolPluginMethod].ID}, header.Number)
	require.NoError(t, err, address)
	plugin := common.BytesToAddress(out)
	if plugin == (common.Address{}) {
		t.Logf("%s: no plugin", address)
		return
	}

	defer func(old func() uint32) { blockTimestamp = old }(blockTimestamp)
	blockTimestamp = func() uint32 { return uint32(earlier.Time) }
	p := &entity.Pool{Address: address, Exchange: byRPC.config.DexID, Type: DexType}
	kept, _, _, _, err := byRPC.getPluginData(ctx, p, plugin, earlier.Number, nil)
	require.NoError(t, err, address)
	blockTimestamp = func() uint32 { return uint32(header.Time) }

	for _, cache := range []map[uint16]Timepoint{nil, kept} {
		p := &entity.Pool{Address: address, Exchange: byRPC.config.DexID, Type: DexType,
			Extra: string(lo.Must(json.Marshal(Extra{ExtraTimepoint: ExtraTimepoint{Timepoints: cache}})))}
		read := func(tracker *PoolTracker) (pluginData, []recordedCall) {
			calls.start()
			var d pluginData
			d.timepoints, d.vo, d.dynamicFee, d.slidingFee, err = tracker.getPluginData(ctx, p, plugin,
				header.Number, nil)
			require.NoError(t, err, address)
			return d, calls.stop()
		}
		want, rpcCalls := read(byRPC)
		got, lensCalls := read(byLens)
		require.Equal(t, want, got, "%s: the lens read other plugin data", address)
		for _, c := range lensCalls {
			require.Nil(t, c.To, "%s: the lens path made a multicall", address)
		}

		state := "first read"
		if cache != nil {
			state = "refresh"
		}
		var rpcGas, lensGas uint64
		if os.Getenv("PLUGIN_LENS_GAS") != "" {
			for _, c := range rpcCalls {
				rpcGas += gasUsed(ctx, t, raw, c, header.Number)
			}
			for _, c := range lensCalls {
				lensGas += gasUsed(ctx, t, raw, c, header.Number)
			}
		}
		t.Logf("%s %s: %d timepoints; multicalls %d (gas %d), lens %d (gas %d)", address, state,
			len(want.timepoints), len(rpcCalls), rpcGas, len(lensCalls), lensGas)
	}
}

// blockAgo is the block about d before header, by the chain's recent block time.
func blockAgo(ctx context.Context, t *testing.T, client *ethclient.Client, header *ethtypes.Header,
	d time.Duration) *ethtypes.Header {
	back := new(big.Int).Sub(header.Number, big.NewInt(1000))
	h, err := client.HeaderByNumber(ctx, back)
	require.NoError(t, err)
	perBlock := float64(header.Time-h.Time) / 1000
	n := int64(d.Seconds() / max(perBlock, 0.05))
	h, err = client.HeaderByNumber(ctx, new(big.Int).Sub(header.Number, big.NewInt(n)))
	require.NoError(t, err)
	return h
}

// gasUsed is the gas call uses at block, as debug_traceCall reports it, reverting or not; 0 when the
// node does not trace.
func gasUsed(ctx context.Context, t *testing.T, raw *gethrpc.Client, call recordedCall, block *big.Int) uint64 {
	var trace struct {
		GasUsed hexutil.Uint64 `json:"gasUsed"`
	}
	err := raw.CallContext(ctx, &trace, "debug_traceCall", call, hexutil.EncodeBig(block),
		map[string]string{"tracer": "callTracer"})
	if err != nil {
		t.Logf("no gas: %v", err)
		return 0
	}
	return uint64(trace.GasUsed)
}

// recordedCall is one eth_call a read made.
type recordedCall struct {
	To   *common.Address `json:"to,omitempty"`
	Data hexutil.Bytes   `json:"data"`
}

// recordCalls records the eth_calls made between start and stop.
type recordCalls struct {
	next      http.RoundTripper
	mu        sync.Mutex
	recording bool
	calls     []recordedCall
}

func (r *recordCalls) start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recording, r.calls = true, nil
}

func (r *recordCalls) stop() []recordedCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recording = false
	return r.calls
}

func (r *recordCalls) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	var msg struct {
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	if json.Unmarshal(body, &msg) == nil && msg.Method == "eth_call" && len(msg.Params) > 0 {
		var c struct {
			recordedCall
			Input hexutil.Bytes `json:"input"`
		}
		if json.Unmarshal(msg.Params[0], &c) == nil {
			if len(c.Data) == 0 {
				c.Data = c.Input
			}
			r.mu.Lock()
			if r.recording {
				r.calls = append(r.calls, c.recordedCall)
			}
			r.mu.Unlock()
		}
	}
	return r.next.RoundTrip(req)
}
