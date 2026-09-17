package arcade

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// arcadeHookABIJson covers the public getters ArcadeHook.sol exposes that decide the
// swap-visible amounts: lifecycle and mode (curveStates), the PUMP fee oracle
// (feeObs), and the CLANKER / RWA per-transaction buy cap inputs.
const arcadeHookABIJson = `[
	{"type":"function","name":"curveStates","stateMutability":"view",
	 "inputs":[{"name":"","type":"bytes32"}],
	 "outputs":[
		{"name":"virtualUsdcReserve","type":"uint128"},
		{"name":"realUsdcReserve","type":"uint128"},
		{"name":"tokensSold","type":"uint128"},
		{"name":"mode","type":"uint8"},
		{"name":"status","type":"uint8"},
		{"name":"creator","type":"address"},
		{"name":"creator2","type":"address"},
		{"name":"creator2Bps","type":"uint16"}]},
	{"type":"function","name":"feeObs","stateMutability":"view",
	 "inputs":[{"name":"","type":"bytes32"}],
	 "outputs":[
		{"name":"emaTickE3","type":"int64"},
		{"name":"gradMcapTick","type":"int24"},
		{"name":"lastTs","type":"uint32"},
		{"name":"init","type":"bool"}]},
	{"type":"function","name":"USDC","stateMutability":"view","inputs":[],
	 "outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"registeredLaunches","stateMutability":"view",
	 "inputs":[{"name":"","type":"address"}],"outputs":[{"name":"","type":"bool"}]},
	{"type":"function","name":"quoteAssetOf","stateMutability":"view",
	 "inputs":[{"name":"","type":"address"}],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"clankerPos","stateMutability":"view",
	 "inputs":[{"name":"","type":"address"}],
	 "outputs":[
		{"name":"tickLower","type":"int24"},
		{"name":"tickUpper","type":"int24"},
		{"name":"seeded","type":"bool"},
		{"name":"launchedAt","type":"uint64"}]},
	{"type":"function","name":"clankerMaxBuyBps","stateMutability":"view","inputs":[],
	 "outputs":[{"name":"","type":"uint16"}]}
]`

var ArcadeHookABI abi.ABI

func init() {
	var err error
	ArcadeHookABI, err = abi.JSON(bytes.NewReader([]byte(arcadeHookABIJson)))
	if err != nil {
		panic(err)
	}
}

// Decode targets, field order matching the ABI outputs (go-ethereum fills
// positionally, so every output is kept even when unused).
type curveStateRaw struct {
	VirtualUsdcReserve *big.Int
	RealUsdcReserve    *big.Int
	TokensSold         *big.Int
	Mode               uint8
	Status             uint8
	Creator            common.Address
	Creator2           common.Address
	Creator2Bps        uint16
}

type feeObsRaw struct {
	EmaTickE3    int64
	GradMcapTick *big.Int
	LastTs       uint32
	Init         bool
}

type clankerPosRaw struct {
	TickLower  *big.Int
	TickUpper  *big.Int
	Seeded     bool
	LaunchedAt uint64
}
