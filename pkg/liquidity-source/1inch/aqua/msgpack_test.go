package oneinchaqua_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	oneinchaqua "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/1inch/aqua"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestMsgpackRoundTrip(t *testing.T) {
	t.Parallel()
	e18 := func(n uint64) *uint256.Int { return new(uint256.Int).Mul(uint256.NewInt(n), uint256.NewInt(1e18)) }
	maker := common.HexToAddress("0x34b53a760cc91ed37085bfc6493621d10a6a25b8")
	var traits uint256.Int
	traits[3] = 1 << 62 // useAquaInsteadOfSignature, bit 254
	// protocolFee(1%, recipient), flatFee(1%), xycSwap, salt
	bps := []byte{0x00, 0x98, 0x96, 0x80}
	recipient := common.HexToAddress("0x6e6b8063d4faf54bf8c898dc6ddc689c76ab12b4")
	program := append([]byte{0x1c, 24}, append(append([]byte{}, bps...), recipient[:]...)...)
	program = append(program, 0x15, 4)
	program = append(program, bps...)
	program = append(program, 0x11, 0, 0x14, 1, 7)
	order := &oneinchaqua.Order{Maker: maker, Traits: traits, Data: program}
	static, err := json.Marshal(oneinchaqua.StaticExtra{Maker: "0x34b53a760cc91ed37085bfc6493621d10a6a25b8",
		App: "0x111111338c5091e8440b67b168bae16a668ac0de", StrategyHash: order.Hash().Hex(), Traits: traits.Hex(), Data: hexutil.Encode(program)})
	require.NoError(t, err)
	unlimited := new(uint256.Int).Not(new(uint256.Int))
	extra, err := json.Marshal(&oneinchaqua.Extra{RawBalance0: e18(1000), RawBalance1: e18(1000), RawBlock: 5,
		WalletBalance0: e18(500), WalletBalance1: e18(900), Allowance0: unlimited, Allowance1: unlimited, WalletBlock: 5})
	require.NoError(t, err)
	e := entity.Pool{Address: "0xabc", Exchange: oneinchaqua.DexType, Type: oneinchaqua.DexType,
		Reserves: entity.PoolReserves{"500000000000000000000", "900000000000000000000"},
		Tokens:   []*entity.PoolToken{{Address: "0x111111111117dc0aa78b770fa6a738034120c302"}, {Address: "0x6b175474e89094c44da98b954eedeac495271d0f"}},
		Extra:    string(extra), StaticExtra: string(static), BlockNumber: 5}
	sim, err := oneinchaqua.NewPoolSimulator(e)
	require.NoError(t, err)

	quote := func(s pool.IPoolSimulator) *pool.CalcAmountOutResult {
		res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: e.Tokens[0].Address, Amount: e18(10).ToBig()}, TokenOut: e.Tokens[1].Address})
		require.NoError(t, err)
		return res
	}
	first := quote(sim)
	// A swap changes the state; the encoded and decoded simulator carries the changed state.
	sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: e.Tokens[0].Address, Amount: e18(10).ToBig()},
		TokenAmountOut: *first.TokenAmountOut, Fee: *first.Fee, SwapInfo: first.SwapInfo})

	encoded, err := msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{"p": sim})
	require.NoError(t, err)
	decoded, err := msgpack.DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	back := decoded["p"]
	a, b := quote(sim), quote(back)
	require.Equal(t, a.TokenAmountOut.Amount, b.TokenAmountOut.Amount)
	require.Equal(t, a.Fee.Amount, b.Fee.Amount)
	require.Equal(t, a.Gas, b.Gas)
	require.NotEqual(t, first.TokenAmountOut.Amount, b.TokenAmountOut.Amount, "the swap is in the decoded state")
	require.Equal(t, sim.CalculateLimit(), back.CalculateLimit())
	require.Equal(t, sim.GetMetaInfo("", ""), back.GetMetaInfo("", ""))
	require.Equal(t, sim.GetReserves(), back.GetReserves())
	var _ *big.Int
}
