package odysfun

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
)

// Guard reproduces the per-tx cap OdysToken._move enforces during the first
// restrictionSeconds after launch: "For the first 15 seconds after a Classic launch each
// wallet is capped at 1% of supply per transaction" (docs.odys.fun). ODYS Classic pools are
// plain, canonical Uniswap v3 pools -- discovered and priced by the generic uniswapv3
// package with no ODYS-specific state -- except for this short-lived guard living in the
// launched token's own contract, not the pool. Without it, quotes in that window can exceed
// what the token will actually let through and the swap reverts on-chain.
//
// Only the per-tx cap (maxTx) is ported. OdysToken._move also caps the recipient's resulting
// balance (maxWallet), but that depends on the taker's existing balance, which is outside
// what a pool simulator can see; that check is intentionally left unenforced (see PR notes).
//
// Neither Classic token in a pool is known to be the ODYS launch token ahead of time -- it
// can be paired against WETH or ARB (see OdysFactoryQ) -- so the guard probes both sides and
// keeps whichever one actually exposes launchTime().
type Guard struct {
	launchToken string

	launchTime0         *big.Int
	restrictionSeconds0 uint32
	maxTx0              *big.Int

	launchTime1         *big.Int
	restrictionSeconds1 uint32
	maxTx1              *big.Int

	token0 string
	token1 string
}

func NewGuard(token0, token1 string) *Guard {
	return &Guard{token0: token0, token1: token1}
}

// AddCalls must be run through a multicall that tolerates individual call failures (e.g.
// ethrpc's TryAggregate), since only one side, at most, is actually an ODYS token.
func (g *Guard) AddCalls(req *ethrpc.Request) {
	if g == nil {
		return
	}

	req.AddCall(&ethrpc.Call{
		ABI: odysTokenABI, Target: g.token0, Method: "launchTime",
	}, []any{&g.launchTime0}).AddCall(&ethrpc.Call{
		ABI: odysTokenABI, Target: g.token0, Method: "restrictionSeconds",
	}, []any{&g.restrictionSeconds0}).AddCall(&ethrpc.Call{
		ABI: odysTokenABI, Target: g.token0, Method: "maxTx",
	}, []any{&g.maxTx0}).AddCall(&ethrpc.Call{
		ABI: odysTokenABI, Target: g.token1, Method: "launchTime",
	}, []any{&g.launchTime1}).AddCall(&ethrpc.Call{
		ABI: odysTokenABI, Target: g.token1, Method: "restrictionSeconds",
	}, []any{&g.restrictionSeconds1}).AddCall(&ethrpc.Call{
		ABI: odysTokenABI, Target: g.token1, Method: "maxTx",
	}, []any{&g.maxTx1})
}

// MaxTxAmount returns the launched token's current per-tx cap, and the launched token's
// address, if the guard window (launchTime..launchTime+restrictionSeconds) is still active.
// Returns ("", nil) once the window has elapsed or the pool isn't an ODYS Classic launch.
func (g *Guard) MaxTxAmount() (token string, maxTx *big.Int) {
	if g == nil {
		return "", nil
	}

	if active, maxTx := guardedAmount(g.launchTime0, g.restrictionSeconds0, g.maxTx0); active {
		return g.token0, maxTx
	}
	if active, maxTx := guardedAmount(g.launchTime1, g.restrictionSeconds1, g.maxTx1); active {
		return g.token1, maxTx
	}
	return "", nil
}

func guardedAmount(launchTime *big.Int, restrictionSeconds uint32, maxTx *big.Int) (bool, *big.Int) {
	if launchTime == nil || launchTime.Sign() == 0 || restrictionSeconds == 0 || maxTx == nil {
		return false, nil
	}

	restrictionEnd := new(big.Int).Add(launchTime, big.NewInt(int64(restrictionSeconds)))
	if big.NewInt(time.Now().Unix()).Cmp(restrictionEnd) >= 0 {
		return false, nil
	}

	return true, maxTx
}
