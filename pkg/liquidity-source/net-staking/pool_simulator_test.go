package netstaking

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// indexLive is the observed on-chain sNET.index() at chain 4663, block-time of writing.
// sNET.index() = 2718290645 (raw, 9-decimal fixed-point, same convention as gOHM).
var indexLive = uint256.NewInt(2718290645)

const (
	testStakingAddress = "0xb078cc304a0b264c5f3680dc0488954accd02e87"
	testWrapAddress    = "0x63c12667638f2ae6fc6ae09b43d98ec84a8586ea"
	testNETAddress     = "0xca9c78dd337a67f6e0077f65f5e9218719d30edf"
	testSNETAddress    = "0xb773ec2c326b7f98a5a83fc098825492f020a4c7"
	testWSNETAddress   = testWrapAddress
)

// testNETReserve and testSNetStakingReserve/testSNetWrapReserve mirror the on-chain
// balances observed at the same block as indexLive.
var (
	testNETReserve         = uint256.NewInt(62271915033974)
	testSNetStakingReserve = uint256.NewInt(13591391263407162106)
	testSNetWrapReserve    = uint256.NewInt(3240061835399)
)

func newSimulator(t *testing.T, index *uint256.Int) *PoolSimulator {
	t.Helper()
	return newSimulatorWithReserves(t, index, testNETReserve, testSNetStakingReserve, testSNetWrapReserve)
}

func newSimulatorWithReserves(
	t *testing.T,
	index *uint256.Int,
	netReserve, sNetStakingReserve, sNetWrapReserve *uint256.Int,
) *PoolSimulator {
	t.Helper()
	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  testStakingAddress,
			Exchange: DexType,
			Type:     DexType,
			Tokens:   []string{testNETAddress, testSNETAddress, testWSNETAddress},
			Reserves: []*big.Int{
				netReserve.ToBig(),
				sNetStakingReserve.ToBig(),
				big.NewInt(0),
			},
		}},
		hasWrap:            true,
		index:              index,
		netReserve:         new(uint256.Int).Set(netReserve),
		sNetStakingReserve: new(uint256.Int).Set(sNetStakingReserve),
		sNetWrapReserve:    new(uint256.Int).Set(sNetWrapReserve),
	}
}

// ---- 1:1 leg ----

func TestCalcAmountOut_NETtoSNET(t *testing.T) {
	s := newSimulator(t, indexLive)
	amtIn := big.NewInt(1_000_000_000) // 1 NET
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testNETAddress, Amount: amtIn},
		TokenOut:      testSNETAddress,
	})
	require.NoError(t, err)
	assert.Equal(t, amtIn, res.TokenAmountOut.Amount, "NET->sNET must be 1:1")
	assert.Equal(t, ActionStake, res.SwapInfo.(SwapInfo).Action)
}

func TestCalcAmountOut_SNETtoNET(t *testing.T) {
	s := newSimulator(t, indexLive)
	amtIn := big.NewInt(500_000_000) // 0.5 sNET
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testSNETAddress, Amount: amtIn},
		TokenOut:      testNETAddress,
	})
	require.NoError(t, err)
	assert.Equal(t, amtIn, res.TokenAmountOut.Amount, "sNET->NET must be 1:1")
	assert.Equal(t, ActionUnstake, res.SwapInfo.(SwapInfo).Action)
}

// ---- index-ratio leg ----

func TestCalcAmountOut_SNETtoWSNET(t *testing.T) {
	s := newSimulator(t, indexLive)
	amtIn := big.NewInt(1_000_000_000) // 1 sNET (9 dec)
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testSNETAddress, Amount: amtIn},
		TokenOut:      testWSNETAddress,
	})
	require.NoError(t, err)
	// expected = 1e9 * 1e18 / 2718290645 = 367878247986245047
	assert.Equal(t, "367878247986245047", res.TokenAmountOut.Amount.String())
	assert.Equal(t, ActionWrap, res.SwapInfo.(SwapInfo).Action)
}

func TestCalcAmountOut_WSNETtoSNET(t *testing.T) {
	s := newSimulator(t, indexLive)
	amtIn := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // 1 wsNET (18 dec)
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testWSNETAddress, Amount: amtIn},
		TokenOut:      testSNETAddress,
	})
	require.NoError(t, err)
	// expected = 1e18 * 2718290645 / 1e18 = 2718290645
	assert.Equal(t, big.NewInt(2718290645), res.TokenAmountOut.Amount)
	assert.Equal(t, ActionUnwrap, res.SwapInfo.(SwapInfo).Action)
}

// ---- composite legs ----

func TestCalcAmountOut_NETtoWSNET_Composite(t *testing.T) {
	s := newSimulator(t, indexLive)
	amtIn := big.NewInt(1_000_000_000) // 1 NET
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testNETAddress, Amount: amtIn},
		TokenOut:      testWSNETAddress,
	})
	require.NoError(t, err)
	// Same math as staking 1 NET -> 1 sNET (1:1) then wrapping 1 sNET -> wsNET.
	assert.Equal(t, "367878247986245047", res.TokenAmountOut.Amount.String())
	assert.Equal(t, ActionStakeAndWrap, res.SwapInfo.(SwapInfo).Action)
}

// wsNET->NET has no reverse composite action in INetStaking.NetAction (the executor
// helper only implements StakeToSNet/UnstakeSNet/Wrap/Unwrap/StakeThenWrap), and
// pathfinder-lib forbids reusing the same pool twice, so no 2-hop workaround exists.
// This direction must stay rejected.
func TestCalcAmountOut_WSNETtoNET_Unsupported(t *testing.T) {
	s := newSimulator(t, indexLive)
	amtIn := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // 1 wsNET
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testWSNETAddress, Amount: amtIn},
		TokenOut:      testNETAddress,
	})
	assert.ErrorIs(t, err, ErrInvalidTokenOut)
}

// ---- cap tests ----

func TestCalcAmountOut_CapExceeded_NETtoSNET(t *testing.T) {
	small := uint256.NewInt(1000)
	s := newSimulatorWithReserves(t, indexLive, testNETReserve, small, testSNetWrapReserve)
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testNETAddress, Amount: big.NewInt(1001)},
		TokenOut:      testSNETAddress,
	})
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}

func TestCalcAmountOut_CapExceeded_WSNETtoSNET(t *testing.T) {
	small := uint256.NewInt(1000)
	s := newSimulatorWithReserves(t, indexLive, testNETReserve, testSNetStakingReserve, small)
	oneWSNET := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testWSNETAddress, Amount: oneWSNET},
		TokenOut:      testSNETAddress,
	})
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}

func TestCalcAmountOut_NoCap_SNETtoWSNET(t *testing.T) {
	zero := uint256.NewInt(0)
	s := newSimulatorWithReserves(t, indexLive, zero, zero, zero)
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testSNETAddress, Amount: big.NewInt(1_000_000_000)},
		TokenOut:      testWSNETAddress,
	})
	assert.NoError(t, err, "sNET->wsNET must not be blocked by reserve cap")
}

func TestCalcAmountOut_ZeroAmount(t *testing.T) {
	s := newSimulator(t, indexLive)
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testNETAddress, Amount: big.NewInt(0)},
		TokenOut:      testSNETAddress,
	})
	assert.ErrorIs(t, err, ErrZeroAmount)
}

func TestCalcAmountOut_InvalidToken(t *testing.T) {
	s := newSimulator(t, indexLive)
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", Amount: big.NewInt(1)},
		TokenOut:      testNETAddress,
	})
	assert.ErrorIs(t, err, ErrInvalidTokenIn)
}

// ---- composite UpdateBalance ----

// TestUpdateBalance_StakeAndWrap verifies the three-reserve shift for the NET->wsNET
// composite: staking's NET balance grows by amountIn, staking's sNET balance shrinks by
// amountIn (paid out to the intermediate leg), and wrap's sNET balance grows by that same
// amountIn (deposited for the wrap leg). The intermediate == amountIn because the stake
// leg is 1:1.
func TestUpdateBalance_StakeAndWrap(t *testing.T) {
	s := newSimulator(t, indexLive)
	amountIn := big.NewInt(1_000_000_000)       // 1 NET
	amountOut := big.NewInt(367878247986245047) // wsNET out, unused by the reserve shift

	wantNET := new(big.Int).Add(testNETReserve.ToBig(), amountIn)
	wantSNetStaking := new(big.Int).Sub(testSNetStakingReserve.ToBig(), amountIn)
	wantSNetWrap := new(big.Int).Add(testSNetWrapReserve.ToBig(), amountIn)

	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testNETAddress, Amount: amountIn},
		TokenAmountOut: pool.TokenAmount{Token: testWSNETAddress, Amount: amountOut},
		SwapInfo:       SwapInfo{Action: ActionStakeAndWrap},
	})

	assert.Equal(t, wantNET, s.netReserve.ToBig())
	assert.Equal(t, wantSNetStaking, s.sNetStakingReserve.ToBig())
	assert.Equal(t, wantSNetWrap, s.sNetWrapReserve.ToBig())
}

// ---- GetApprovalAddress ----

// TestGetApprovalAddress asserts the approval target matches
// ExecutorV3Helper9.executeNetStaking's spender selection: the wrap contract only for
// the Wrap action (sNET->wsNET); staking for every other direction, including Unwrap
// (wsNET->sNET), which needs no allowance at all but still names staking as the target.
func TestGetApprovalAddress(t *testing.T) {
	s := newSimulator(t, indexLive)

	assert.Equal(t, testStakingAddress, s.GetApprovalAddress(testNETAddress, testSNETAddress), "stake: approve staking")
	assert.Equal(t, testStakingAddress, s.GetApprovalAddress(testSNETAddress, testNETAddress), "unstake: approve staking")
	assert.Equal(t, testWrapAddress, s.GetApprovalAddress(testSNETAddress, testWSNETAddress), "wrap: approve wrap contract")
	assert.Equal(t, testStakingAddress, s.GetApprovalAddress(testWSNETAddress, testSNETAddress), "unwrap: approve staking, not wrap")
	assert.Equal(t, testStakingAddress, s.GetApprovalAddress(testNETAddress, testWSNETAddress), "composite stake+wrap: approve staking (first hop)")
	assert.Equal(t, testStakingAddress, s.GetApprovalAddress(testWSNETAddress, testNETAddress), "unsupported direction falls back to staking")
}

// TestCanSwap_WSNETtoNETExcluded asserts the pool never advertises wsNET->NET as a
// candidate hop: CanSwapFrom(wsNET) must not contain NET, and CanSwapTo(NET) must not
// contain wsNET. Without this, pathfinder-lib would still try the direction (via the
// default all-tokens CanSwapTo/CanSwapFrom) and fail on every CalcAmountOut call.
func TestCanSwap_WSNETtoNETExcluded(t *testing.T) {
	s := newSimulator(t, indexLive)

	assert.NotContains(t, s.CanSwapFrom(testWSNETAddress), testNETAddress)
	assert.ElementsMatch(t, []string{testSNETAddress}, s.CanSwapFrom(testWSNETAddress))

	assert.NotContains(t, s.CanSwapTo(testNETAddress), testWSNETAddress)
	assert.ElementsMatch(t, []string{testSNETAddress}, s.CanSwapTo(testNETAddress))
}

func TestCloneState_DeepCopy(t *testing.T) {
	s := newSimulator(t, indexLive)
	clone := s.CloneState().(*PoolSimulator)

	clone.index.SetUint64(1)
	assert.Equal(t, uint64(2718290645), s.index.Uint64(), "original index must be unmodified after clone mutation")

	origNETReserve := new(uint256.Int).Set(s.netReserve)
	clone.netReserve.SetUint64(1)
	assert.Equal(t, origNETReserve, s.netReserve, "original netReserve must be unmodified after clone mutation")
}

// ---- no-wrap pool (NUKE-shaped: 2 tokens, base<->staked only) ----

const (
	testNukeStakingAddress = "0x9c648d57e929f59b483b2903390725449f990cb8"
	testNukeAddress        = "0xca9c78dd337a67f6e0077f65f5e9218719d30ed1"
	testSNukeAddress       = "0xb773ec2c326b7f98a5a83fc098825492f020a4c1"
)

var (
	testNukeReserve  = uint256.NewInt(500_000_000_000) // 500 NUKE (9 dec)
	testSNukeReserve = uint256.NewInt(300_000_000_000) // 300 sNUKE (9 dec)
)

// newNoWrapSimulator builds a 2-token pool (no wrap contract), matching the shape
// NewPoolSimulator produces when entity.Pool has exactly 2 tokens.
func newNoWrapSimulator(t *testing.T) *PoolSimulator {
	t.Helper()
	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  testNukeStakingAddress,
			Exchange: DexType,
			Type:     DexType,
			Tokens:   []string{testNukeAddress, testSNukeAddress},
			Reserves: []*big.Int{testNukeReserve.ToBig(), testSNukeReserve.ToBig()},
		}},
		hasWrap:            false,
		netReserve:         new(uint256.Int).Set(testNukeReserve),
		sNetStakingReserve: new(uint256.Int).Set(testSNukeReserve),
		sNetWrapReserve:    new(uint256.Int),
	}
}

func TestNoWrap_CalcAmountOut_BaseToStaked(t *testing.T) {
	s := newNoWrapSimulator(t)
	amtIn := big.NewInt(1_000_000_000) // 1 NUKE
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testNukeAddress, Amount: amtIn},
		TokenOut:      testSNukeAddress,
	})
	require.NoError(t, err)
	assert.Equal(t, amtIn, res.TokenAmountOut.Amount, "base->staked must be 1:1")
	assert.Equal(t, ActionStake, res.SwapInfo.(SwapInfo).Action)
}

func TestNoWrap_CalcAmountOut_StakedToBase(t *testing.T) {
	s := newNoWrapSimulator(t)
	amtIn := big.NewInt(500_000_000) // 0.5 sNUKE
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testSNukeAddress, Amount: amtIn},
		TokenOut:      testNukeAddress,
	})
	require.NoError(t, err)
	assert.Equal(t, amtIn, res.TokenAmountOut.Amount, "staked->base must be 1:1")
	assert.Equal(t, ActionUnstake, res.SwapInfo.(SwapInfo).Action)
}

// TestNoWrap_CalcAmountOut_WrapDirection_NoPanic guards the most important regression:
// a naive port of the old 3-token code would index s.Info.Tokens[idxWSNET] out of bounds
// on a 2-token pool. Any wrap-related direction must error cleanly, never panic.
func TestNoWrap_CalcAmountOut_WrapDirection_NoPanic(t *testing.T) {
	s := newNoWrapSimulator(t)
	fakeWsToken := "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

	assert.NotPanics(t, func() {
		_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: testSNukeAddress, Amount: big.NewInt(1)},
			TokenOut:      fakeWsToken,
		})
		assert.ErrorIs(t, err, ErrInvalidTokenOut)
	})

	assert.NotPanics(t, func() {
		_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: fakeWsToken, Amount: big.NewInt(1)},
			TokenOut:      testSNukeAddress,
		})
		assert.ErrorIs(t, err, ErrInvalidTokenIn)
	})
}

func TestNoWrap_CanSwap_NeverLeaksWrapOrEmptyToken(t *testing.T) {
	s := newNoWrapSimulator(t)

	for _, tok := range []string{testNukeAddress, testSNukeAddress} {
		to := s.CanSwapTo(tok)
		from := s.CanSwapFrom(tok)
		assert.NotContains(t, to, "", "CanSwapTo must never contain an empty-string token")
		assert.NotContains(t, from, "", "CanSwapFrom must never contain an empty-string token")
		assert.Len(t, to, 1, "no-wrap pool has exactly one counterpart token")
		assert.Len(t, from, 1, "no-wrap pool has exactly one counterpart token")
	}
}

func TestNoWrap_GetApprovalAddress_AlwaysStaking(t *testing.T) {
	s := newNoWrapSimulator(t)
	assert.Equal(t, testNukeStakingAddress, s.GetApprovalAddress(testNukeAddress, testSNukeAddress))
	assert.Equal(t, testNukeStakingAddress, s.GetApprovalAddress(testSNukeAddress, testNukeAddress))
}

func TestNoWrap_GetMetaInfo_WSNETEmpty(t *testing.T) {
	s := newNoWrapSimulator(t)
	meta := s.GetMetaInfo(testNukeAddress, testSNukeAddress).(PoolMeta)
	assert.Empty(t, meta.WSNET)
}

// TestNewPoolSimulator_NoWrap confirms the constructor sets hasWrap from the token count
// (2 tokens => no wrap) and never dereferences a nil wrap-side extra field.
func TestNewPoolSimulator_NoWrap(t *testing.T) {
	extra := PoolExtra{
		NETReserve:         testNukeReserve,
		SNETStakingReserve: testSNukeReserve,
	}
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)

	ep := entity.Pool{
		Address:  testNukeStakingAddress,
		Exchange: DexType,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: testNukeAddress, Swappable: true},
			{Address: testSNukeAddress, Swappable: true},
		},
		Reserves: entity.PoolReserves{testNukeReserve.ToBig().String(), testSNukeReserve.ToBig().String()},
		Extra:    string(extraBytes),
	}

	s, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	assert.False(t, s.hasWrap)
	assert.Nil(t, s.index)

	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testNukeAddress, Amount: big.NewInt(1_000_000_000)},
		TokenOut:      testSNukeAddress,
	})
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(1_000_000_000), res.TokenAmountOut.Amount)
}
