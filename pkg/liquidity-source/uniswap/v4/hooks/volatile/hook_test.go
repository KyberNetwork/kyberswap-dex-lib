package volatile

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

func TestBeforeSwap_RequiresTracked(t *testing.T) {
	h := &Hook{}
	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{ZeroForOne: true, AmountSpecified: big.NewInt(1), CalcOut: true})
	require.ErrorIs(t, err, ErrPoolIsNotTracked)
}

func TestBeforeSwap_UsesDirectionalFee(t *testing.T) {
	h := &Hook{Extra: Extra{Tracked: true, FeeZeroForOne: 900, FeeOneForZero: 1200}}

	res0, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{ZeroForOne: true, AmountSpecified: big.NewInt(1), CalcOut: true})
	require.NoError(t, err)
	require.Equal(t, uniswapv4.FeeAmount(900), res0.SwapFee)
	require.Equal(t, HookGas, res0.Gas)
	require.Zero(t, res0.DeltaSpecified.Sign())
	require.Zero(t, res0.DeltaUnspecified.Sign())

	res1, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{ZeroForOne: false, AmountSpecified: big.NewInt(1), CalcOut: true})
	require.NoError(t, err)
	require.Equal(t, uniswapv4.FeeAmount(1200), res1.SwapFee)
	require.Zero(t, res1.DeltaSpecified.Sign())
	require.Zero(t, res1.DeltaUnspecified.Sign())
}

// The hook's getFee reverts PoolNotInitialized unless the key hashes to the pool id, so the key built
// from StaticExtra must reproduce the real ETH/USDC pool id (native currency0 = address(0)).
func TestNewPoolKey_MatchesOnchainPoolID(t *testing.T) {
	pool := &entity.Pool{Tokens: []*entity.PoolToken{
		{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"},
		{Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"},
	}}
	se := &uniswapv4.StaticExtra{IsNative: [2]bool{true, false}, Fee: 0x800000, TickSpacing: 4}
	key := newPoolKey(pool, se, HookAddress)

	var enc []byte
	for _, w := range [][]byte{key.Currency0[:], key.Currency1[:], key.Fee.Bytes(), key.TickSpacing.Bytes(), key.Hooks[:]} {
		enc = append(enc, common.LeftPadBytes(w, 32)...)
	}
	require.Equal(t, "0x98316e8405502774a5d0a0761276b5d26505a8195c1905169864008d5e7a3eea", crypto.Keccak256Hash(enc).Hex())
}

// ethrpc unpacks a multi-output call only into one abi-tagged struct; two pointers fail with
// "cannot unmarshal tuple in to uint32" (found against the live hook).
func TestGetFeeABI_UnpacksIntoTaggedStruct(t *testing.T) {
	data, err := volatilePairHookABI.Methods["getFee"].Outputs.Pack(big.NewInt(800), big.NewInt(0))
	require.NoError(t, err)

	var fees struct {
		FeeZeroForOne *big.Int `abi:"feeE6ZeroForOne"`
		FeeOneForZero *big.Int `abi:"feeE6OneForZero"`
	}
	require.NoError(t, volatilePairHookABI.UnpackIntoInterface(&fees, "getFee", data))
	require.Equal(t, int64(800), fees.FeeZeroForOne.Int64())
	require.Zero(t, fees.FeeOneForZero.Sign())
}
