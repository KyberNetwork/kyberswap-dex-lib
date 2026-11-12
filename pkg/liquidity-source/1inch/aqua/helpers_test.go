package oneinchaqua

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	abiutil "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

// Addresses of the deployments the fixtures come from. They are test inputs: the package itself reads
// every address from its Config.
const (
	testAquaHex   = "0x1111113ccf1426a8e30e2bff5e005d929bf6a90a"
	testRouterHex = "0x111111338c5091e8440b67b168bae16a668ac0de"
	testLaunchHex = "0x1111113db0e0ef9d0e3a50d5f094a3a57a26c0de"
)

// routerABI is the router's quote, for the parity and oracle tests.
var routerABI = abiutil.MustParseABI(mustReadFile("testdata/Router.json"))

func mustReadFile(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}

// testConfig is a complete config for a chain; the options adjust it.
func testConfig(opts ...func(*Config)) *Config {
	zero := uint64(0)
	cfg := &Config{DexId: DexType, ChainId: 1, AquaAddress: testAquaHex, RouterAddress: testRouterHex,
		StartBlock: 1, LogWindow: 50_000, MaxWindowsPerRun: 20, Confirmations: &zero}
	for _, o := range opts {
		o(cfg)
	}
	return cfg
}

var (
	testMaker     = common.HexToAddress("0x34b53a760cc91ed37085bfc6493621d10a6a25b8")
	testRecipient = common.HexToAddress("0x6e6b8063d4faf54bf8c898dc6ddc689c76ab12b4")
	testAqua      = common.HexToAddress(testAquaHex)
	testRouter    = common.HexToAddress(testRouterHex)
	testLaunch    = common.HexToAddress(testLaunchHex)
	// token0 < token1 as addresses.
	testToken0 = "0x111111111117dc0aa78b770fa6a738034120c302"
	testToken1 = "0x6b175474e89094c44da98b954eedeac495271d0f"
)

func ins(op byte, args ...byte) []byte { return append([]byte{op, byte(len(args))}, args...) }

func prog(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func iSwap() []byte { return ins(opXYCSwap) }
func iSalt(b ...byte) []byte {
	return ins(opSalt, b...)
}
func iFlat(bps uint32) []byte {
	var a [4]byte
	binary.BigEndian.PutUint32(a[:], bps)
	return ins(opFlatFeeIn, a[:]...)
}
func iProtocol(bps uint32, to common.Address) []byte {
	var a [4]byte
	binary.BigEndian.PutUint32(a[:], bps)
	return ins(opAquaProtocolFee, append(a[:], to[:]...)...)
}
func iConcentrate(pmin, pmax *uint256.Int) []byte {
	lo, hi := pmin.Bytes32(), pmax.Bytes32()
	return ins(opXYCConcentrate, append(lo[:], hi[:]...)...)
}

// aquaTraits is the maker traits of a plain Aqua order: the Aqua flag and nothing else.
func aquaTraits() uint256.Int {
	var t uint256.Int
	setBit(&t, traitsUseAqua)
	return t
}

func setBit(z *uint256.Int, bit uint) { z[bit/64] |= 1 << (bit % 64) }

func orderOf(maker common.Address, program []byte) *Order {
	return &Order{Maker: maker, Traits: aquaTraits(), Data: program}
}

// addrHex is the lowercase 0x form of an address, how the pool state stores it.
func addrHex(a common.Address) string { return hexutil.Encode(a[:]) }

func mustParse(t testing.TB, program []byte, launch bool) []Instruction {
	t.Helper()
	p, err := ParseOrder(orderOf(testMaker, program), launch)
	require.NoError(t, err)
	return p
}

// max248 is the largest Aqua balance, 2^248 - 1 (independent of the production bound).
var max248 = new(uint256.Int).SubUint64(new(uint256.Int).Lsh(uint256.NewInt(1), 248), 1)

func u(s string) *uint256.Int { return uint256.MustFromDecimal(s) }

// e18 returns n * 1e18.
func e18(n uint64) *uint256.Int {
	return new(uint256.Int).Mul(uint256.NewInt(n), uint256.NewInt(1e18))
}

// sqrtPrice returns floor(sqrt(price * 1e18)) for a price in 1e18 fixed point, the form the SDK builds.
func sqrtPrice(price *uint256.Int) *uint256.Int {
	var p uint256.Int
	p.Mul(price, uOne)
	return p.Sqrt(&p)
}

// runQuote runs a program and returns its result by value, for tests.
func runQuote(prog []Instruction, balanceIn, balanceOut, amountIn *uint256.Int, tokenInLt bool) (quoteResult, error) {
	var res quoteResult
	err := runProgram(&res, prog, balanceIn, balanceOut, amountIn, tokenInLt)
	return res, err
}

// liquidityOf returns L for the balances and bounds, for tests.
func liquidityOf(bLt, bGt, sqrtPMin, sqrtPMax *uint256.Int) (*uint256.Int, error) {
	var l uint256.Int
	err := computeL(&l, bLt, bGt, sqrtPMin, sqrtPMax)
	return &l, err
}

func mustUnmarshal[T any](t testing.TB, data string) T {
	t.Helper()
	var v T
	require.NoError(t, json.Unmarshal([]byte(data), &v))
	return v
}

// goldenVector is one recorded router quote with the strategy balances it was taken at.
type goldenVector struct {
	ChainId   int      `json:"chainId"`
	Block     uint64   `json:"block"`
	App       string   `json:"app"`
	Strategy  string   `json:"strategy"`
	Tokens    []string `json:"tokens"`
	Raw       []string `json:"raw"`
	In        int      `json:"in"`
	AmountIn  string   `json:"amountIn"`
	AmountOut string   `json:"amountOut"`
}

const parityTakerTraits = "0x" + "0000000000000000000000000000000000000000" + "0001" // exact in, no threshold
