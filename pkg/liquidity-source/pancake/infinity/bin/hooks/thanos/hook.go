// Package thanos handles SquadSwap's Bin hook (bitmap 0x442: afterInitialize, beforeSwap,
// beforeSwapReturnsDelta). beforeSwap takes floor(|amountSpecified| * swapFees(poolId) / 1e6)
// of the specified currency for the hook. The LP fee is the pool's dynamic fee in slot0.
package thanos

import (
	"context"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// All share runtime codehash 0x920fed38...; storage (fees) is per hook.
var BinHookAddresses = []common.Address{
	common.HexToAddress("0xe7C583425bB682ac321701aa954e99A352c5aa53"),
	common.HexToAddress("0x15da6326080b3b592a4da0402d2278fec86075f7"),
	common.HexToAddress("0x3fee70b37a3e44111c541d2b5bc7e82e8339b895"),
	common.HexToAddress("0x60ea610fe21051b8c50975dc5c741dedf8465ada"),
	common.HexToAddress("0x7fcbf855f6890f7b6dcf0c7e9ec7e88f165c7f19"),
	common.HexToAddress("0xa6cefd4d922d27e1b3a3fa775161f599a790de4d"),
	common.HexToAddress("0xa82c229ef2fffe711dc1ad0d49c0baf1ece8ab4e"),
	common.HexToAddress("0xa90787c1af856bb56a767b58345f7bc1363923f1"),
	common.HexToAddress("0xd45a254058a30d1af557b267c6d66cd17627d43a"),
	common.HexToAddress("0xfba8e8a5fad24c43473020cf22af0da4c072c714"),
}

// Output declared uint32 (on-chain uint24) so it unpacks to uint32; selector depends on inputs only.
var swapFeesABI = lo.Must(abi.JSON(strings.NewReader(
	`[{"inputs":[{"type":"bytes32"}],"name":"swapFees","outputs":[{"type":"uint32"}],"stateMutability":"view","type":"function"}]`)))

type Hook struct {
	Exchange valueobject.Exchange
}

func NewHook(exchange valueobject.Exchange) *Hook {
	return &Hook{Exchange: exchange}
}

func (h *Hook) GetExchange() string {
	return string(h.Exchange)
}

// GetDynamicFee returns the LP fee unchanged: beforeSwap returns no LP fee override, so the
// manager charges the dynamic fee stored in slot0 (set by the hook owner, at most 10%).
func (h *Hook) GetDynamicFee(_ context.Context, _ *ethrpc.Client, _ string, _ common.Address, lpFee uint32) uint32 {
	return lpFee
}

// SwapFeeCall reads the hook fee in pips (1e6 = 100%) for the pool.
func (h *Hook) SwapFeeCall(hookAddress common.Address, poolID string) *ethrpc.Call {
	return &ethrpc.Call{
		ABI:    swapFeesABI,
		Target: hexutil.Encode(hookAddress[:]),
		Method: "swapFees",
		Params: []any{common.HexToHash(poolID)},
	}
}
