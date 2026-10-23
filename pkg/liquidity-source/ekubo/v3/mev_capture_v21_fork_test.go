package ekubov3

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The fork test executes MEVCapture v2.1 quotes on a local mainnet fork (anvil) against the deployed extension, Core
// and KyberSwap router and executor. It needs `anvil` on the PATH and an Ethereum RPC URL in EKUBO_V3_FORK_RPC_URL (EKUBO_V3_FORK_ANVIL_PORT optionally fixes the
// anvil port);
// nothing is sent to mainnet.
//
// The pool (ETH/USDT, with the fee and tick spacing of the MEVCapture v1 pool the route templates trade through) is
// created on the fork. The KyberSwap executor only accepts route data signed by KyberSwap, so the test takes route
// calldata the KyberSwap API built for a v1 MEVCapture hop (testdata), substitutes the v2.1 pool key, forward address,
// price limit, skipAhead and amounts from this package's SwapInfo, and grants the recovered signer of the modified
// data the executor's signer role in the fork's storage. Everything else runs unmodified on the deployed contracts.
const (
	v21ForkBlock          = 26_250_392
	v21ForkRPCEnv         = "EKUBO_V3_FORK_RPC_URL"
	v21ForkPortEnv        = "EKUBO_V3_FORK_ANVIL_PORT"
	v21ForkKyberRoutes    = "testdata/mevcapture-v21-kyber-routes.json"
	v21ForkTickSpacing    = 100
	v21ForkFee            = uint64(0x000d1b71758e2196)
	v21ForkExecutorSlot   = 0 // OpenZeppelin AccessControl `_roles` mapping of the executor
	v21ForkUsdtBalanceMap = 2
)

var (
	v21ForkExtension   = common.HexToAddress("0x51f831c23fa4051862881Ef8EfC0c3C37905Ce49")
	v21ForkV1Extension = common.HexToAddress("0x5555fF9Ff2757500BF4EE020DcfD0210CFfa41Be")
	v21ForkUSDT        = common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	v21ForkPositions   = common.HexToAddress("0xa2971e0c37cfdb13ae8440a0c94ef1a1af39e326")
	v21ForkKyberRouter = common.HexToAddress("0x6131B5fae19EA4f9D964eAc0408E4408b66337b5")
	v21ForkExecutor    = common.HexToAddress("0x8F10B468b06c6FD214B65F87778827F7D113f996")
	v21ForkSignerRole  = common.HexToHash("0xe2f4eaae4a9751e85a3e4a7b9587827a877f29914755229b07a7b2da98285f70")
	v21ForkYulRouter   = common.HexToAddress("0x7B2aA7Ecc0B5936b7C52E6259A19C3BA557d0748")
	v21ForkMulticall   = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")
	v21ForkTrader      = common.HexToAddress("0x2222222222222222222222222222222222222222")
	v21ForkEcrecover   = common.HexToAddress("0x0000000000000000000000000000000000000001")

	v21ForkABI = mustABI(`[
		{"type":"function","name":"mintAndDeposit","stateMutability":"payable","inputs":[
			{"name":"poolKey","type":"tuple","components":[{"name":"token0","type":"address"},{"name":"token1","type":"address"},{"name":"config","type":"bytes32"}]},
			{"name":"tickLower","type":"int32"},{"name":"tickUpper","type":"int32"},
			{"name":"maxAmount0","type":"uint128"},{"name":"maxAmount1","type":"uint128"},{"name":"minLiquidity","type":"uint128"}],
		 "outputs":[{"type":"uint256"},{"type":"uint128"},{"type":"uint128"},{"type":"uint128"}]},
		{"type":"function","name":"approve","stateMutability":"nonpayable","inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[]},
		{"type":"function","name":"balanceOf","stateMutability":"view","inputs":[{"name":"owner","type":"address"}],"outputs":[{"type":"uint256"}]},
		{"type":"function","name":"swap","stateMutability":"payable","inputs":[{"name":"execution","type":"tuple","components":[
			{"name":"callTarget","type":"address"},{"name":"approveTarget","type":"address"},{"name":"targetData","type":"bytes"},
			{"name":"desc","type":"tuple","components":[
				{"name":"srcToken","type":"address"},{"name":"dstToken","type":"address"},{"name":"srcReceivers","type":"address[]"},
				{"name":"srcAmounts","type":"uint256[]"},{"name":"feeReceivers","type":"address[]"},{"name":"feeAmounts","type":"uint256[]"},
				{"name":"dstReceiver","type":"address"},{"name":"amount","type":"uint256"},{"name":"minReturnAmount","type":"uint256"},
				{"name":"flags","type":"uint256"},{"name":"permit","type":"bytes"}]},
			{"name":"clientData","type":"bytes"}]}],"outputs":[]}
	]`)
)

func mustABI(s string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(s))
	if err != nil {
		panic(err)
	}
	return parsed
}

type (
	v21Fork struct {
		t   *testing.T
		url string
		rpc *gethrpc.Client
		eth *ethclient.Client
	}

	v21KyberRoute struct {
		TokenIn        string `json:"tokenIn"`
		TokenOut       string `json:"tokenOut"`
		AmountIn       string `json:"amountIn"`
		RouteAmountOut string `json:"routeAmountOut"`
		Lim            string `json:"lim"`
		SkipAhead      uint32 `json:"skipAhead"`
		Pool           string `json:"pool"`
		PoolKey        struct {
			Token0 common.Address `json:"token0"`
			Token1 common.Address `json:"token1"`
			Config common.Hash    `json:"config"`
		} `json:"poolKey"`
		Value string `json:"value"`
		Data  string `json:"data"`
	}

	// SwapExecutionParams of the KyberSwap MetaAggregationRouterV2
	v21KyberSwap struct {
		CallTarget    common.Address `abi:"callTarget"`
		ApproveTarget common.Address `abi:"approveTarget"`
		TargetData    []byte         `abi:"targetData"`
		Desc          struct {
			SrcToken        common.Address   `abi:"srcToken"`
			DstToken        common.Address   `abi:"dstToken"`
			SrcReceivers    []common.Address `abi:"srcReceivers"`
			SrcAmounts      []*big.Int       `abi:"srcAmounts"`
			FeeReceivers    []common.Address `abi:"feeReceivers"`
			FeeAmounts      []*big.Int       `abi:"feeAmounts"`
			DstReceiver     common.Address   `abi:"dstReceiver"`
			Amount          *big.Int         `abi:"amount"`
			MinReturnAmount *big.Int         `abi:"minReturnAmount"`
			Flags           *big.Int         `abi:"flags"`
			Permit          []byte           `abi:"permit"`
		} `abi:"desc"`
		ClientData []byte `abi:"clientData"`
	}
)

func startV21Fork(t *testing.T) *v21Fork {
	upstream := os.Getenv(v21ForkRPCEnv)
	if upstream == "" {
		t.Skipf("%s is not set", v21ForkRPCEnv)
	}
	anvil, err := exec.LookPath("anvil")
	if err != nil {
		t.Skip("anvil is not installed")
	}

	port, err := strconv.Atoi(os.Getenv(v21ForkPortEnv))
	if err != nil {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		port = listener.Addr().(*net.TCPAddr).Port
		require.NoError(t, listener.Close())
	}

	cmd := exec.Command(anvil, "--fork-url", upstream, "--fork-block-number", strconv.Itoa(v21ForkBlock),
		"--port", strconv.Itoa(port), "--auto-impersonate", "--silent")
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	client, err := gethrpc.Dial(url)
	require.NoError(t, err)
	f := &v21Fork{t: t, url: url, rpc: client, eth: ethclient.NewClient(client)}
	require.Eventually(t, func() bool {
		n, err := f.eth.BlockNumber(context.Background())
		return err == nil && n == v21ForkBlock
	}, 2*time.Minute, 200*time.Millisecond)
	return f
}

func (f *v21Fork) call(result any, method string, args ...any) {
	f.t.Helper()
	require.NoError(f.t, f.rpc.Call(result, method, args...), method)
}

func (f *v21Fork) send(from, to common.Address, value *big.Int, data []byte) common.Hash {
	f.t.Helper()
	var hash common.Hash
	f.call(&hash, "eth_sendTransaction", map[string]any{
		"from":  from,
		"to":    to,
		"value": (*hexutil.Big)(lo.Ternary(value == nil, new(big.Int), value)),
		"data":  hexutil.Bytes(data),
		"gas":   hexutil.Uint64(30_000_000),
	})
	return hash
}

func (f *v21Fork) receipt(hash common.Hash) *types.Receipt {
	f.t.Helper()
	var receipt *types.Receipt
	require.Eventually(f.t, func() bool {
		var err error
		receipt, err = f.eth.TransactionReceipt(context.Background(), hash)
		return err == nil
	}, 30*time.Second, 20*time.Millisecond, "receipt of %s", hash)
	require.Equal(f.t, types.ReceiptStatusSuccessful, receipt.Status, "transaction %s reverted", hash)
	return receipt
}

func (f *v21Fork) mustSend(from, to common.Address, value *big.Int, data []byte) *types.Receipt {
	f.t.Helper()
	return f.receipt(f.send(from, to, value, data))
}

func (f *v21Fork) setStorage(address common.Address, slot common.Hash, value common.Hash) {
	f.call(nil, "anvil_setStorageAt", address, slot, value)
}

func (f *v21Fork) erc20Balance(token, owner common.Address) *big.Int {
	data, err := v21ForkABI.Pack("balanceOf", owner)
	require.NoError(f.t, err)
	out, err := f.eth.CallContract(context.Background(), ethereum.CallMsg{To: &token, Data: data}, nil)
	require.NoError(f.t, err)
	return new(big.Int).SetBytes(out)
}

func (f *v21Fork) ethBalance(owner common.Address, block *big.Int) *big.Int {
	balance, err := f.eth.BalanceAt(context.Background(), owner, block)
	require.NoError(f.t, err)
	return balance
}

func mappingSlot(key common.Hash, slot common.Hash) common.Hash {
	return crypto.Keccak256Hash(key[:], slot[:])
}

func v21ForkPoolKey() *pools.ConcentratedPoolKey {
	return pools.NewPoolKey(common.Address{}, v21ForkUSDT,
		pools.NewPoolConfig(v21ForkExtension, v21ForkFee, pools.NewConcentratedPoolTypeConfig(v21ForkTickSpacing)))
}

// setUpPool initializes the v2.1 pool at the price of the v1 pool and adds a wide position and narrow positions every
// 4 tick spacings above and below the price, so that swaps cross initialized ticks in several bitmap positions.
func (f *v21Fork) setUpPool() *pools.ConcentratedPoolKey {
	t := f.t
	f.call(nil, "anvil_setBalance", v21ForkTrader, (*hexutil.Big)(new(big.Int).Lsh(big.NewInt(1), 80)))
	f.setStorage(v21ForkUSDT, mappingSlot(common.BytesToHash(v21ForkTrader[:]), common.BigToHash(big.NewInt(v21ForkUsdtBalanceMap))),
		common.BigToHash(big.NewInt(10_000_000_000_000)))
	for _, spender := range []common.Address{v21ForkPositions, v21ForkKyberRouter, v21ForkYulRouter} {
		data, err := v21ForkABI.Pack("approve", spender, new(big.Int).Lsh(big.NewInt(1), 255))
		require.NoError(t, err)
		f.mustSend(v21ForkTrader, v21ForkUSDT, nil, data)
	}

	v1Key := pools.NewPoolKey(common.Address{}, v21ForkUSDT,
		pools.NewPoolConfig(v21ForkV1Extension, v21ForkFee, pools.NewConcentratedPoolTypeConfig(v21ForkTickSpacing)))
	v1PoolID, err := v1Key.NumId()
	require.NoError(t, err)
	var v1State hexutil.Bytes
	f.call(&v1State, "eth_getStorageAt", MainnetConfig.Core, common.BytesToHash(v1PoolID), "latest")
	tick := int32(binary.BigEndian.Uint32(v1State[12:16]))
	tick -= tick % v21ForkTickSpacing
	if tick < 0 {
		tick -= v21ForkTickSpacing
	}

	key := v21ForkPoolKey()
	data, err := abis.CoreABI.Pack("initializePool", key.ToAbi(), tick)
	require.NoError(t, err)
	f.mustSend(v21ForkTrader, MainnetConfig.Core, nil, data)

	eth := func(milli int64) *big.Int { return new(big.Int).Mul(big.NewInt(milli), big.NewInt(1e15)) }
	usdt := func(units int64) *big.Int { return big.NewInt(units * 1_000_000) }
	mint := func(lower, upper int32, amount0, amount1 *big.Int) {
		data, err := v21ForkABI.Pack("mintAndDeposit", key.ToAbi(), lower, upper, amount0, amount1, new(big.Int))
		require.NoError(t, err)
		f.mustSend(v21ForkTrader, v21ForkPositions, amount0, data)
	}
	const s = v21ForkTickSpacing
	mint(tick-400*s, tick+400*s, eth(10_000), usdt(24_000))
	for i := int32(1); i <= 12; i++ {
		mint(tick+4*i*s, tick+4*i*s+s, eth(300), new(big.Int))
		mint(tick-4*i*s-s, tick-4*i*s, new(big.Int), usdt(700))
	}
	return key
}

func v21ForkConfig() *Config {
	config := *MainnetConfig
	config.supportedExtensions = nil
	config.MevCaptureV21 = v21ForkExtension
	return &config
}

// newV21ForkSimulator lists the pool like the pool list updater does and builds its simulator.
func (f *v21Fork) newV21ForkSimulator(config *Config, key *pools.ConcentratedPoolKey) (*PoolSimulator, entity.Pool) {
	t := f.t
	client := ethrpc.New(f.url).SetMulticallContract(v21ForkMulticall)
	anyKey := pools.AnyPoolKey{PoolKey: pools.NewPoolKey(key.Token0, key.Token1,
		pools.NewPoolConfig[pools.PoolTypeConfig](key.Config.Extension, key.Config.Fee, key.Config.TypeConfig))}
	fetched, err := NewDataFetchers(client, config).fetchPools(context.Background(), []pools.AnyPoolKey{anyKey}, nil)
	require.NoError(t, err)
	require.Len(t, fetched, 1)

	staticExtra, err := json.Marshal(StaticExtra{
		Core:          config.Core,
		ExtensionType: config.ExtensionType(key.Config.Extension),
		PoolKey:       anyKey,
	})
	require.NoError(t, err)
	extra, err := json.Marshal(fetched[0].GetState())
	require.NoError(t, err)
	address, err := anyKey.ToPoolAddress()
	require.NoError(t, err)
	entityPool := entity.Pool{
		Address:  address,
		Exchange: string(config.DexId),
		Type:     DexType,
		Reserves: []string{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: valueobject.ZeroToWrappedLower(hexutil.Encode(key.Token0[:]), config.ChainId), Swappable: true},
			{Address: valueobject.ZeroToWrappedLower(hexutil.Encode(key.Token1[:]), config.ChainId), Swappable: true},
		},
		StaticExtra: string(staticExtra),
		Extra:       string(extra),
		BlockNumber: fetched[0].blockNumber,
	}
	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)
	return sim, entityPool
}

// refresh updates the pool from the logs of executed swaps through the pool tracker.
func (f *v21Fork) refresh(config *Config, entityPool entity.Pool, receipts ...*types.Receipt) (*PoolSimulator, entity.Pool) {
	t := f.t
	tracker := NewPoolTracker(config, ethrpc.New(f.url).SetMulticallContract(v21ForkMulticall))
	var logs []types.Log
	for _, receipt := range receipts {
		for _, log := range receipt.Logs {
			logs = append(logs, *log)
		}
	}
	updated, err := tracker.GetNewPoolState(context.Background(), entityPool, pool.GetNewPoolStateParams{Logs: logs})
	require.NoError(t, err)
	require.Greater(t, updated.BlockNumber, entityPool.BlockNumber)
	sim, err := NewPoolSimulator(updated)
	require.NoError(t, err)
	return sim, updated
}

func v21SwapInfo(t *testing.T, swapInfo any) (quoting.SwapInfo, *pools.MevCaptureV21PoolSwapState) {
	info, ok := swapInfo.(quoting.SwapInfo)
	require.True(t, ok)
	state, ok := info.SwapStateAfter.(*pools.MevCaptureV21PoolSwapState)
	require.True(t, ok)
	return info, state
}

// simulate records the Core calls of the quoted swap, for comparison with the Core events of the execution.
func simulateV21(t *testing.T, sim *PoolSimulator, info quoting.SwapInfo, amount *int256.Int, now uint32) *pools.MevCaptureV21Swap {
	v21Pool, ok := sim.EkuboPool.(*pools.MevCaptureV21Pool)
	require.True(t, ok)
	swap, err := v21Pool.SimulateSwap(v21Pool.MevCaptureV21PoolSwapState, &pools.MevCaptureV21SwapParams{
		IsToken1:       info.IsToken1,
		Amount:         amount,
		SqrtRatioLimit: math.FloatSqrtRatioToFixed(info.PriceLimit),
		SkipAhead:      info.SkipAhead,
	}, now, true)
	require.NoError(t, err)
	return swap
}

func loadV21KyberRoutes(t *testing.T) ([]v21KyberRoute, v21KyberRoute) {
	data, err := os.ReadFile(v21ForkKyberRoutes)
	require.NoError(t, err)
	var routes struct {
		Routes   []v21KyberRoute `json:"routes"`
		Baseline v21KyberRoute   `json:"baseline"`
	}
	require.NoError(t, json.Unmarshal(data, &routes))
	require.Len(t, routes.Routes, 2)
	return routes.Routes, routes.Baseline
}

func replaceAll(t *testing.T, data, old, new []byte, minCount int) []byte {
	require.GreaterOrEqual(t, bytes.Count(data, old), minCount, "%x", old)
	return bytes.ReplaceAll(data, old, new)
}

func hopControlWord(skipAhead uint32, lim *big.Int) []byte {
	word := make([]byte, 32)
	binary.BigEndian.PutUint32(word[12:16], skipAhead)
	lim.FillBytes(word[20:32])
	return word
}

// kyberCalldata turns the route template of a v1 MEVCapture hop into the same route through the v2.1 pool, with the
// price limit, skipAhead and expected and minimum amounts of the quote.
func (f *v21Fork) kyberCalldata(route v21KyberRoute, info quoting.SwapInfo, amountOut *big.Int) []byte {
	t := f.t
	calldata := hexutil.MustDecode(route.Data)
	method := v21ForkABI.Methods["swap"]
	require.Equal(t, method.ID, calldata[:4])
	values, err := method.Inputs.Unpack(calldata[4:])
	require.NoError(t, err)
	execution := *abi.ConvertType(values[0], new(v21KyberSwap)).(*v21KyberSwap)
	require.Equal(t, v21ForkExecutor, execution.CallTarget)

	require.Equal(t, v21ForkExtension, *info.Forward)
	data := replaceAll(t, execution.TargetData, v21ForkV1Extension[:], info.Forward[:], 2)
	templateLim, ok := new(big.Int).SetString(route.Lim, 10)
	require.True(t, ok)
	data = replaceAll(t, data, hopControlWord(route.SkipAhead, templateLim),
		hopControlWord(info.SkipAhead, info.PriceLimit.ToBig()), 1)
	routeOut, ok := new(big.Int).SetString(route.RouteAmountOut, 10)
	require.True(t, ok)
	data = replaceAll(t, data, routeOut.FillBytes(make([]byte, 16)), amountOut.FillBytes(make([]byte, 16)), 1)

	execution.TargetData = data
	execution.Desc.MinReturnAmount = amountOut
	packed, err := method.Inputs.Pack(execution)
	require.NoError(t, err)
	return append(method.ID, packed...)
}

type v21CallFrame struct {
	To      common.Address `json:"to"`
	Input   hexutil.Bytes  `json:"input"`
	Output  hexutil.Bytes  `json:"output"`
	GasUsed hexutil.Uint64 `json:"gasUsed"`
	Calls   []v21CallFrame `json:"calls"`
}

// findPoolCall finds the outermost `Core.forward` or `Core.swap` call.
func findPoolCall(frame *v21CallFrame) *v21CallFrame {
	if frame.To == MainnetConfig.Core && len(frame.Input) >= 4 &&
		(bytes.Equal(frame.Input[:4], abis.CoreABI.Methods["forward"].ID) ||
			bytes.Equal(frame.Input[:4], abis.CoreABI.Methods["swap_6269342730"].ID)) {
		return frame
	}
	for i := range frame.Calls {
		if found := findPoolCall(&frame.Calls[i]); found != nil {
			return found
		}
	}
	return nil
}

// poolCallGas is the gas used by the pool call of a single-hop swap: `Core.forward` (the extension and all its Core
// calls) or `Core.swap`.
func (f *v21Fork) poolCallGas(hash common.Hash) uint64 {
	var frame v21CallFrame
	f.call(&frame, "debug_traceTransaction", hash, map[string]any{"tracer": "callTracer"})
	poolCall := findPoolCall(&frame)
	require.NotNil(f.t, poolCall)
	return uint64(poolCall.GasUsed)
}

func findSigner(frame *v21CallFrame) *common.Address {
	if frame.To == v21ForkEcrecover && len(frame.Output) == 32 {
		signer := common.BytesToAddress(frame.Output)
		return &signer
	}
	for i := range frame.Calls {
		if signer := findSigner(&frame.Calls[i]); signer != nil {
			return signer
		}
	}
	return nil
}

// authorizeRoute grants the signer recovered from the modified route data the executor's signer role.
func (f *v21Fork) authorizeRoute(value *big.Int, calldata []byte) {
	var frame v21CallFrame
	f.call(&frame, "debug_traceCall", map[string]any{
		"from":  v21ForkTrader,
		"to":    v21ForkKyberRouter,
		"value": (*hexutil.Big)(value),
		"data":  hexutil.Bytes(calldata),
	}, "latest", map[string]any{"tracer": "callTracer"})
	signer := findSigner(&frame)
	require.NotNil(f.t, signer)
	roleSlot := mappingSlot(v21ForkSignerRole, common.BigToHash(big.NewInt(v21ForkExecutorSlot)))
	f.setStorage(v21ForkExecutor, mappingSlot(common.BytesToHash(signer[:]), roleSlot), common.BigToHash(big.NewInt(1)))
}

// coreSwaps returns the balance updates and states of the Core swaps of the pool in the receipt (Core's anonymous
// Swapped events).
func coreSwaps(t *testing.T, receipt *types.Receipt, poolID []byte) [][3]*big.Int {
	var swaps [][3]*big.Int
	for _, log := range receipt.Logs {
		if log.Address != MainnetConfig.Core || len(log.Topics) != 0 || len(log.Data) < 116 ||
			!bytes.Equal(log.Data[20:52], poolID) {
			continue
		}
		delta0 := new(big.Int).SetBytes(log.Data[52:68])
		delta1 := new(big.Int).SetBytes(log.Data[68:84])
		for _, d := range []*big.Int{delta0, delta1} {
			if d.Bit(127) == 1 {
				d.Sub(d, new(big.Int).Lsh(big.NewInt(1), 128))
			}
		}
		swaps = append(swaps, [3]*big.Int{delta0, delta1, new(big.Int).SetBytes(log.Data[84:96])})
	}
	return swaps
}

func requireCoreCallsMatch(t *testing.T, name string, receipt *types.Receipt, poolID []byte, sim *pools.MevCaptureV21Swap) {
	executed := coreSwaps(t, receipt, poolID)
	// Core does not emit an event for a call that moves neither the price nor the amounts
	var expected [][3]*big.Int
	for _, call := range sim.Calls {
		if call.Delta0.IsZero() && call.Delta1.IsZero() && call.Before.SqrtRatio().Eq(call.After.SqrtRatio()) {
			continue
		}
		expected = append(expected, [3]*big.Int{call.Delta0.ToBig(), call.Delta1.ToBig(),
			pools.SqrtRatioToRaw(call.After.SqrtRatio()).ToBig()})
	}
	require.Len(t, executed, len(expected), "%s: Core swaps", name)
	for i := range expected {
		for j := range 3 {
			require.Zero(t, expected[i][j].Cmp(executed[i][j]), "%s: Core swap %d field %d: %v != %v", name, i, j,
				expected[i][j], executed[i][j])
		}
	}
}

func intrinsicGas(data []byte) uint64 {
	gas := uint64(21_000)
	for _, b := range data {
		gas += lo.Ternary[uint64](b == 0, 4, 16)
	}
	return gas
}

func transferredTo(receipt *types.Receipt, token, to common.Address) *big.Int {
	return transferred(receipt, token, nil, &to)
}

func transferred(receipt *types.Receipt, token common.Address, from, to *common.Address) *big.Int {
	transferTopic := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	total := new(big.Int)
	for _, log := range receipt.Logs {
		if log.Address == token && len(log.Topics) == 3 && log.Topics[0] == transferTopic &&
			(from == nil || common.BytesToAddress(log.Topics[1][:]) == *from) &&
			(to == nil || common.BytesToAddress(log.Topics[2][:]) == *to) {
			total.Add(total, new(big.Int).SetBytes(log.Data))
		}
	}
	return total
}

type v21ForkResult struct {
	name       string
	specified  string
	quoted     *big.Int
	executed   *big.Int
	sim        *pools.MevCaptureV21Swap
	estimate   int64
	receipt    *types.Receipt
	calldata   []byte
	forwardGas uint64
}

// ekuboModelForwardGas is Ekubo's own calibrated estimate of the forwarded call (repeated swap base 7,900, extra base
// 8,250, cold extension access 2,500, and the per-resource constants), for comparison.
func ekuboModelForwardGas(r *pools.MevCaptureV21Resources) int64 {
	return 7_900 + 8_250 + 2_500 + 12_350*int64(r.CoreCalls) + 2_850*int64(r.CoreSteps-min(r.CoreSteps, r.CoreCalls)) +
		13_150*int64(r.InitializedTicksCrossed) + 7_950*int64(r.AccumulateAsFees) + 4_650*int64(r.AnchorUpdates)
}

func (f *v21Fork) report(r *v21ForkResult) {
	t := f.t
	r.forwardGas = f.poolCallGas(r.receipt.TxHash)
	poolEstimate := r.estimate - quoting.BaseGasCost
	kinds := lo.CountValues(lo.Map(r.sim.Calls, func(c pools.MevCaptureV21CoreCall, _ int) string { return c.Kind.String() }))
	t.Logf("%s: specified %s, quoted %s, executed %s; block %d; %d Core calls %v, %d steps, %d ticks crossed, "+
		"%d accumulateAsFees, %d anchor updates; gas: estimate %d (pool %d, Ekubo model %d), forward call %d "+
		"(pool estimate %+.1f%%), tx %d, tx execution %d (estimate %+.1f%%)",
		r.name, r.specified, r.quoted, r.executed, r.receipt.BlockNumber, r.sim.Resources.CoreCalls, kinds,
		r.sim.Resources.CoreSteps, r.sim.Resources.InitializedTicksCrossed, r.sim.Resources.AccumulateAsFees,
		r.sim.Resources.AnchorUpdates, r.estimate, poolEstimate, ekuboModelForwardGas(&r.sim.Resources),
		r.forwardGas, 100*(float64(poolEstimate)/float64(r.forwardGas)-1), r.receipt.GasUsed,
		r.receipt.GasUsed-intrinsicGas(r.calldata),
		100*(float64(r.estimate)/float64(r.receipt.GasUsed-intrinsicGas(r.calldata))-1))
	require.Equal(t, r.quoted.String(), r.executed.String(), r.name)
	require.InEpsilon(t, r.forwardGas, poolEstimate, 0.2, "%s: pool gas estimate", r.name)
}

func TestMevCaptureV21ForkExecutesQuotesExactly(t *testing.T) {
	test.SkipCI(t)
	f := startV21Fork(t)
	key := f.setUpPool()
	poolID, err := key.NumId()
	require.NoError(t, err)
	config := v21ForkConfig()
	routes, baseline := loadV21KyberRoutes(t)
	ethToUsdt, usdtToEth := routes[0], routes[1]
	weth := strings.ToLower(valueobject.WrappedNativeMap[valueobject.ChainIDEthereum])
	usdt := hexutil.Encode(v21ForkUSDT[:])
	ctx := context.Background()
	t.Logf("fork of Ethereum block %d", v21ForkBlock)

	// Gas baseline: the unmodified route through a plain concentrated pool and this package's estimate for it.
	baselineExtension, baselineFee, baselineTypeConfig := decodePoolConfig(baseline.PoolKey.Config)
	baselineKey := pools.NewPoolKey(baseline.PoolKey.Token0, baseline.PoolKey.Token1, pools.NewPoolConfig(
		baselineExtension, baselineFee, baselineTypeConfig.(pools.ConcentratedPoolTypeConfig)))
	baselineSim, _ := f.newV21ForkSimulator(config, baselineKey)
	amountIn, _ := new(big.Int).SetString(ethToUsdt.AmountIn, 10)
	baselineQuote, err := baselineSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: amountIn},
		TokenOut:      hexutil.Encode(baseline.PoolKey.Token1[:]),
	})
	require.NoError(t, err)
	baselineCalldata := hexutil.MustDecode(baseline.Data)
	baselineReceipt := f.mustSend(v21ForkTrader, v21ForkKyberRouter, amountIn, baselineCalldata)
	baselineExecution := baselineReceipt.GasUsed - intrinsicGas(baselineCalldata)
	baselinePoolCall := f.poolCallGas(baselineReceipt.TxHash)
	baselinePoolEstimate := baselineQuote.Gas - quoting.BaseGasCost
	// the executor forwards at most the expected amount of the route; the pool output is what Core sent
	t.Logf("baseline plain concentrated pool %s through the same router and executor: quoted %s, pool output %s; gas: "+
		"estimate %d (pool %d), Core.swap call %d (pool estimate %+.1f%%), tx %d, tx execution %d (estimate %+.1f%%)",
		baseline.Pool, baselineQuote.TokenAmountOut.Amount,
		transferred(baselineReceipt, baseline.PoolKey.Token1, &MainnetConfig.Core, nil), baselineQuote.Gas,
		baselinePoolEstimate, baselinePoolCall, 100*(float64(baselinePoolEstimate)/float64(baselinePoolCall)-1),
		baselineReceipt.GasUsed, baselineExecution, 100*(float64(baselineQuote.Gas)/float64(baselineExecution)-1))

	sim, entityPool := f.newV21ForkSimulator(config, key)
	head, err := f.eth.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, head.Number.Uint64(), entityPool.BlockNumber)
	t.Logf("pool %s listed at block %d", entityPool.Address, entityPool.BlockNumber)

	// Two exact-input ETH -> USDT swaps through the KyberSwap executor in one block: the first runs the anchor
	// update of the timestamp, the second is quoted on the state UpdateBalance left and must not run it again.
	quote1, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: amountIn},
		TokenOut:      usdt,
	})
	require.NoError(t, err)
	require.Zero(t, quote1.RemainingTokenAmountIn.Amount.Sign())
	info1, after1 := v21SwapInfo(t, quote1.SwapInfo)
	now := after1.Anchor.LastUpdateTime
	sim1 := simulateV21(t, sim, info1, int256.MustFromBig(amountIn), now)
	require.NotNil(t, sim1.Gate)
	require.Equal(t, quote1.TokenAmountOut.Amount, new(big.Int).Neg(sim1.Delta1.ToBig()))

	cloned := sim.CloneState()
	cloned.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: quote1.SwapInfo})
	quote2, err := cloned.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: amountIn},
		TokenOut:      usdt,
	})
	require.NoError(t, err)
	info2, after2 := v21SwapInfo(t, quote2.SwapInfo)
	require.Equal(t, now, after2.Anchor.LastUpdateTime)
	sim2 := simulateV21(t, cloned.(*PoolSimulator), info2, int256.MustFromBig(amountIn), now)
	require.Nil(t, sim2.Gate)
	require.Equal(t, quote2.TokenAmountOut.Amount, new(big.Int).Neg(sim2.Delta1.ToBig()))

	// the original simulator is unaffected by UpdateBalance on its clone
	again, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: amountIn},
		TokenOut:      usdt,
	})
	require.NoError(t, err)
	require.Equal(t, quote1.TokenAmountOut.Amount, again.TokenAmountOut.Amount)

	calldata1 := f.kyberCalldata(ethToUsdt, info1, quote1.TokenAmountOut.Amount)
	calldata2 := f.kyberCalldata(ethToUsdt, info2, quote2.TokenAmountOut.Amount)
	f.authorizeRoute(amountIn, calldata1)
	f.authorizeRoute(amountIn, calldata2)

	f.call(nil, "evm_setAutomine", false)
	tx1 := f.send(v21ForkTrader, v21ForkKyberRouter, amountIn, calldata1)
	tx2 := f.send(v21ForkTrader, v21ForkKyberRouter, amountIn, calldata2)
	f.call(nil, "evm_mine", hexutil.Uint64(now))
	f.call(nil, "evm_setAutomine", true)
	receipt1, receipt2 := f.receipt(tx1), f.receipt(tx2)
	require.Equal(t, receipt1.BlockNumber, receipt2.BlockNumber)
	block, err := f.eth.HeaderByNumber(ctx, receipt1.BlockNumber)
	require.NoError(t, err)
	require.Equal(t, uint64(now), block.Time)

	results := []*v21ForkResult{
		{name: "ETH->USDT exact in, first swap of the timestamp", specified: amountIn.String() + " wei",
			quoted: quote1.TokenAmountOut.Amount, executed: transferredTo(receipt1, v21ForkUSDT, v21ForkTrader),
			sim: sim1, estimate: quote1.Gas, receipt: receipt1, calldata: calldata1},
		{name: "ETH->USDT exact in, same block", specified: amountIn.String() + " wei",
			quoted: quote2.TokenAmountOut.Amount, executed: transferredTo(receipt2, v21ForkUSDT, v21ForkTrader),
			sim: sim2, estimate: quote2.Gas, receipt: receipt2, calldata: calldata2},
	}
	require.Greater(t, sim1.Resources.AccumulateAsFees, uint32(1), "multi-segment away swap")
	require.Greater(t, sim2.Resources.AccumulateAsFees, uint32(1), "multi-segment away swap")

	// The opposite direction in a later block, quoted on the state the pool tracker refetched on the swaps' events:
	// toward the anchor, then away from it through several segments.
	time.Sleep(1100 * time.Millisecond)
	sim, entityPool = f.refresh(config, entityPool, receipt1, receipt2)
	usdtIn, _ := new(big.Int).SetString(usdtToEth.AmountIn, 10)
	quote3, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: usdt, Amount: usdtIn},
		TokenOut:      weth,
	})
	require.NoError(t, err)
	require.Zero(t, quote3.RemainingTokenAmountIn.Amount.Sign())
	info3, after3 := v21SwapInfo(t, quote3.SwapInfo)
	now3 := after3.Anchor.LastUpdateTime
	require.Greater(t, now3, now)
	sim3 := simulateV21(t, sim, info3, int256.MustFromBig(usdtIn), now3)
	require.Equal(t, pools.MevCaptureV21CallToward, sim3.Calls[0].Kind)
	require.Greater(t, sim3.Resources.AccumulateAsFees, uint32(1), "multi-segment away swap")
	calldata3 := f.kyberCalldata(usdtToEth, info3, quote3.TokenAmountOut.Amount)
	f.authorizeRoute(new(big.Int), calldata3)
	f.call(nil, "evm_setNextBlockTimestamp", hexutil.Uint64(now3))
	ethBefore := f.ethBalance(v21ForkTrader, nil)
	receipt3 := f.receipt(f.send(v21ForkTrader, v21ForkKyberRouter, nil, calldata3))
	gasCost := new(big.Int).Mul(new(big.Int).SetUint64(receipt3.GasUsed), receipt3.EffectiveGasPrice)
	results = append(results, &v21ForkResult{name: "USDT->ETH exact in", specified: usdtIn.String() + " USDT units",
		quoted: quote3.TokenAmountOut.Amount,
		executed: new(big.Int).Add(new(big.Int).Sub(f.ethBalance(v21ForkTrader, receipt3.BlockNumber), ethBefore),
			gasCost),
		sim: sim3, estimate: quote3.Gas, receipt: receipt3, calldata: calldata3})

	// Exact output (the KyberSwap API only builds exact-input routes): buy ETH with USDT through the Ekubo router's
	// forwarded hop, with the SwapInfo of the quote.
	time.Sleep(1100 * time.Millisecond)
	sim, _ = f.refresh(config, entityPool, receipt3)
	ethOut := big.NewInt(1e18)
	quote4, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: weth, Amount: ethOut},
		TokenIn:        usdt,
	})
	require.NoError(t, err)
	require.Zero(t, quote4.RemainingTokenAmountOut.Amount.Sign())
	info4, after4 := v21SwapInfo(t, quote4.SwapInfo)
	now4 := after4.Anchor.LastUpdateTime
	sim4 := simulateV21(t, sim, info4, int256.MustFromBig(new(big.Int).Neg(ethOut)), now4)
	require.Greater(t, sim4.Resources.AccumulateAsFees, uint32(1), "multi-segment away swap")
	calldata4 := yulRouterForwardedRoute(key, info4, new(big.Int).Neg(ethOut),
		new(big.Int).Neg(quote4.TokenAmountIn.Amount))
	f.call(nil, "evm_setNextBlockTimestamp", hexutil.Uint64(now4))
	usdtBefore := f.erc20Balance(v21ForkUSDT, v21ForkTrader)
	ethBefore = f.ethBalance(v21ForkTrader, nil)
	receipt4 := f.receipt(f.send(v21ForkTrader, v21ForkYulRouter, nil, calldata4))
	gasCost = new(big.Int).Mul(new(big.Int).SetUint64(receipt4.GasUsed), receipt4.EffectiveGasPrice)
	received := new(big.Int).Add(new(big.Int).Sub(f.ethBalance(v21ForkTrader, receipt4.BlockNumber), ethBefore), gasCost)
	require.Equal(t, ethOut.String(), received.String())
	results = append(results, &v21ForkResult{name: "USDT->ETH exact out (Ekubo router)",
		specified: ethOut.String() + " wei out", quoted: quote4.TokenAmountIn.Amount,
		executed: new(big.Int).Sub(usdtBefore, f.erc20Balance(v21ForkUSDT, v21ForkTrader)),
		sim:      sim4, estimate: quote4.Gas, receipt: receipt4, calldata: calldata4})

	for _, r := range results {
		f.report(r)
		requireCoreCallsMatch(t, r.name, r.receipt, poolID, r.sim)
	}
}

// yulRouterForwardedRoute encodes a single forwarded hop for the Ekubo Yul router (see its SDK `encodeRoutes`).
func yulRouterForwardedRoute(key *pools.ConcentratedPoolKey, info quoting.SwapInfo, specifiedAmount,
	threshold *big.Int) []byte {
	specified, calculated := key.Token0, key.Token1
	if info.IsToken1 {
		specified, calculated = calculated, specified
	}
	int128 := func(x *big.Int) []byte {
		u := new(big.Int).Set(x)
		if u.Sign() < 0 {
			u.Add(u, new(big.Int).Lsh(big.NewInt(1), 128))
		}
		return u.FillBytes(make([]byte, 16))
	}
	abiKey := key.ToAbi()
	var route []byte
	route = append(route, 1, 0)
	route = append(route, specified[:]...)
	route = append(route, calculated[:]...)
	route = append(route, int128(threshold)...)
	route = append(route, v21ForkTrader[:]...)
	route = append(route, int128(specifiedAmount)...)
	route = append(route, 0, 1)
	route = append(route, info.Forward[:]...)
	route = append(route, abiKey.Token0[:]...)
	route = append(route, abiKey.Token1[:]...)
	route = append(route, abiKey.Config[:]...)
	route = append(route, info.PriceLimit.ToBig().FillBytes(make([]byte, 12))...)
	return binary.BigEndian.AppendUint32(route, info.SkipAhead)
}
