//go:build livegate

// Live old/new comparison harness (not committed). Env: LG_RPC (upstream URL, never printed), LG_POOLS
// (chain pool file), LG_OUT (output json), LG_BLOCK (pinned block), LG_TS (pinned block timestamp).
package twocryptong_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	twocryptong "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/twocrypto-ng"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/twocrypto-ng/lazy"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const lgMcABI = `[{"name":"tryBlockAndAggregate","type":"function","stateMutability":"payable",
"inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`

type lgCall struct {
	Target, Method string
	OK             bool
}

type lgProxy struct {
	mu      sync.Mutex
	batches [][]lgCall
	srv     *httptest.Server
}

var lgHTTP = &http.Client{Timeout: 60 * time.Second}

func methodName(data []byte) string {
	if len(data) < 4 {
		return "?"
	}
	for _, a := range []abi.ABI{twocryptong.CurveTwocryptoNGABI, twocryptong.YBTwocryptoPolicyABI,
		twocryptong.YBBankPolicyABI, twocryptong.YBPolicyABI} {
		if m, err := a.MethodById(data[:4]); err == nil {
			return m.Name
		}
	}
	return hexutil.Encode(data[:4])
}

func forward(upstream string, body []byte) ([]byte, error) {
	var last error
	for a := 0; a < 8; a++ {
		resp, err := lgHTTP.Post(upstream, "application/json", bytes.NewReader(body))
		if err == nil {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 200 && !bytes.Contains(b, []byte(`"error"`)) {
				return b, nil
			}
			last = fmt.Errorf("status %d: %.120s", resp.StatusCode, b)
		} else {
			last = err
		}
		time.Sleep(time.Duration(a+1) * time.Second)
	}
	return nil, last
}

func newProxy(t *testing.T, upstream string, block uint64) *lgProxy {
	mc, _ := abi.JSON(strings.NewReader(lgMcABI))
	p := &lgProxy{}
	p.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		if json.Unmarshal(body, &req) != nil {
			http.Error(w, "batch unsupported", 400)
			return
		}
		var calls []lgCall
		if req["method"] == "eth_call" {
			params := req["params"].([]any)
			if len(params) < 2 {
				params = append(params, nil)
			}
			params[1] = hexutil.EncodeUint64(block) // pin every read to the same block
			req["params"] = params
			msg := params[0].(map[string]any)
			in, _ := msg["data"].(string)
			if in == "" {
				in, _ = msg["input"].(string)
			}
			data, _ := hexutil.Decode(in)
			if len(data) > 4 {
				if args, err := mc.Methods["tryBlockAndAggregate"].Inputs.Unpack(data[4:]); err == nil {
					v := reflect.ValueOf(args[1])
					for i := 0; i < v.Len(); i++ {
						calls = append(calls, lgCall{
							Target: strings.ToLower(v.Index(i).FieldByName("Target").Interface().(common.Address).Hex()),
							Method: methodName(v.Index(i).FieldByName("CallData").Interface().([]byte))})
					}
				}
			}
			body, _ = json.Marshal(req)
		}
		out, err := forward(upstream, body)
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		if calls != nil {
			var resp struct{ Result string }
			_ = json.Unmarshal(out, &resp)
			raw, _ := hexutil.Decode(resp.Result)
			if vals, err := mc.Methods["tryBlockAndAggregate"].Outputs.Unpack(raw); err == nil {
				v := reflect.ValueOf(vals[2])
				for i := 0; i < v.Len() && i < len(calls); i++ {
					calls[i].OK = v.Index(i).FieldByName("Success").Bool()
				}
			}
			p.mu.Lock()
			p.batches = append(p.batches, calls)
			p.mu.Unlock()
		}
		_, _ = w.Write(out)
	}))
	t.Cleanup(p.srv.Close)
	return p
}

type lgPoolIn struct {
	Address, Version string
	Tokens           []string
	Native           []bool
	Decimals         []uint8
}

type lgRefresh struct {
	Err         string `json:",omitempty"`
	Extra       string
	Reserves    []string
	BlockNumber uint64
	Batches     [][]lgCall
	PoolCalls   int
	Reverts     []string
	Quotes      map[string]string
}

type lgResult struct {
	Address, Tracker, Version string
	R1, R2                    lgRefresh
}

var lgFracs = []int64{1_000_000, 10_000, 100, 10} // reserve / f

func quotes(p entity.Pool, ts int64) map[string]string {
	out := map[string]string{}
	var e twocryptong.Extra
	if json.Unmarshal([]byte(p.Extra), &e) != nil {
		out["sim"] = "bad extra"
		return out
	}
	if e.PriceOracleTimestamp != 0 {
		e.PriceOracleTimestamp = ts
	}
	b, _ := json.Marshal(e)
	p.Extra = string(b)
	for i := 0; i < 2; i++ {
		for _, f := range lgFracs {
			key := fmt.Sprintf("%d->%d/%d", i, 1-i, f)
			res, ok := new(big.Int).SetString(p.Reserves[i], 10)
			if !ok {
				out[key] = "err: reserve"
				continue
			}
			amt := new(big.Int).Div(res, big.NewInt(f))
			sim, err := twocryptong.NewPoolSimulator(p)
			if err != nil {
				out[key] = "err: " + err.Error()
				continue
			}
			r, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: p.Tokens[i].Address, Amount: amt},
				TokenOut:      p.Tokens[1-i].Address})
			if err != nil {
				out[key] = "err: " + err.Error()
			} else {
				out[key] = r.TokenAmountOut.Amount.String() + " fee " + r.Fee.Amount.String()
			}
		}
	}
	return out
}

type refresher func(ctx context.Context, p entity.Pool) (entity.Pool, error)

func refreshOnce(t *testing.T, upstream string, block uint64, kind string, p entity.Pool, ts int64) (entity.Pool, lgRefresh) {
	var last lgRefresh
	for a := 0; a < 3; a++ {
		px := newProxy(t, upstream, block)
		cl := ethrpc.New(px.srv.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
		cfg := &shared.Config{DexID: "curve-twocrypto-ng"}
		var out entity.Pool
		var err error
		if kind == "tracker" {
			tr, _ := twocryptong.NewPoolTracker(cfg, cl)
			out, err = tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		} else {
			tr, _ := lazy.NewPoolTracker(cfg, cl)
			out, err = tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		}
		r := lgRefresh{Batches: px.batches}
		for _, b := range px.batches {
			for _, c := range b {
				if c.Target == strings.ToLower(p.Address) {
					r.PoolCalls++
				}
				if !c.OK {
					r.Reverts = append(r.Reverts, c.Target[:10]+":"+c.Method)
				}
			}
		}
		if err != nil {
			r.Err = err.Error()
			last = r
			time.Sleep(2 * time.Second)
			continue
		}
		r.Extra, r.Reserves, r.BlockNumber = out.Extra, out.Reserves, out.BlockNumber
		r.Quotes = quotes(out, ts)
		return out, r
	}
	return p, last
}

func TestLiveGate(t *testing.T) {
	upstream, file, outFile := os.Getenv("LG_RPC"), os.Getenv("LG_POOLS"), os.Getenv("LG_OUT")
	if upstream == "" || file == "" {
		t.Skip("LG_RPC/LG_POOLS unset")
	}
	block, _ := strconv.ParseUint(os.Getenv("LG_BLOCK"), 10, 64)
	ts, _ := strconv.ParseInt(os.Getenv("LG_TS"), 10, 64)
	twocryptong.SetNowUnix(ts + 1)
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var pools []lgPoolIn
	if err := json.Unmarshal(raw, &pools); err != nil {
		t.Fatal(err)
	}
	var (
		mu      sync.Mutex
		results []lgResult
		wg      sync.WaitGroup
		sem     = make(chan struct{}, 3)
	)
	for _, pi := range pools {
		for _, kind := range []string{"tracker", "lazy"} {
			wg.Add(1)
			go func(pi lgPoolIn, kind string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				p := entity.Pool{Address: pi.Address, Exchange: "curve-twocrypto-ng", Type: twocryptong.DexType,
					Reserves: entity.PoolReserves{"0", "0"}}
				for i, tk := range pi.Tokens {
					p.Tokens = append(p.Tokens, &entity.PoolToken{Address: tk, Swappable: true, Decimals: pi.Decimals[i]})
				}
				se, _ := json.Marshal(twocryptong.StaticExtra{IsNativeCoins: pi.Native})
				p.StaticExtra = string(se)
				p1, r1 := refreshOnce(t, upstream, block, kind, p, ts)
				var r2 lgRefresh
				if r1.Err == "" {
					_, r2 = refreshOnce(t, upstream, block, kind, p1, ts)
				}
				mu.Lock()
				results = append(results, lgResult{pi.Address, kind, pi.Version, r1, r2})
				mu.Unlock()
			}(pi, kind)
		}
	}
	wg.Wait()
	sort.Slice(results, func(i, j int) bool {
		if results[i].Address != results[j].Address {
			return results[i].Address < results[j].Address
		}
		return results[i].Tracker < results[j].Tracker
	})
	b, _ := json.MarshalIndent(results, "", " ")
	if err := os.WriteFile(outFile, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
