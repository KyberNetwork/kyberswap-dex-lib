package uniswapv4

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Vectors come from a live Elysium V4 pool (id 0x7fe1...9339) read through both StateView and
// PoolManager.extsload at the same block, so a layout drift (slot offsets, packing) fails here.
func TestExtsloadMatchesStateView(t *testing.T) {
	t.Parallel()

	id := common.HexToHash("0x7fe1c643dcf953d4d85840598b44239e617f45bd35e36eb591f881908ca19339")
	base := stateSlot(id)
	assert.Equal(t, "0x336833bad0b92a71563372cd4e95c39c4212324e19ec4da6bdc82698a8355feb", base.Hex())
	assert.Equal(t, "0x336833bad0b92a71563372cd4e95c39c4212324e19ec4da6bdc82698a8355fee", addOffset(base, liquidityOffset).Hex())
	assert.Equal(t, "0x946995b7d093a0960de88be7e0db8a86334b83e399e1c2c2f08f5636a19f26fd", tickSlot(base, -887200).Hex())
	assert.Equal(t, "0xda9c7afd3d1c9fa16a97a9906b23ebb8d5b4d9a3dc0835e061ae974467ef98bf", tickSlot(base, 166200).Hex())

	s0 := decodeSlot0(common.HexToHash("0x000000000000000000020b3c000000000000032a08d26598f58e25a4791893af"))
	assert.Equal(t, "64177541870835417273859426522031", s0.SqrtPriceX96.String()) // StateView getSlot0
	assert.Equal(t, int64(133948), s0.Tick.Int64())
	assert.Zero(t, s0.ProtocolFee)
	assert.Zero(t, s0.LpFee)

	assert.Equal(t, "173459393557747126473114",
		decodeLiquidity(common.HexToHash("0x0000000000000000000000000000000000000000000024bb40a83584267f919a")).String())

	gross, net := decodeTickInfo(common.HexToHash("0x00000000000024bb40a83584267f919a00000000000024bb40a83584267f919a"))
	assert.Equal(t, "173459393557747126473114", gross.String())
	assert.Equal(t, "173459393557747126473114", net.String())
}

func TestExtsloadSignedFields(t *testing.T) {
	t.Parallel()

	// lpFee 3000 | protocolFee 0x123 | tick -5 | sqrt 1
	s0 := decodeSlot0(common.HexToHash("0x000000000bb8000123fffffb0000000000000000000000000000000000000001"))
	assert.Equal(t, int64(-5), s0.Tick.Int64())
	assert.Equal(t, uint32(0x123), s0.ProtocolFee)
	assert.Equal(t, uint32(3000), s0.LpFee)
	assert.Equal(t, int64(1), s0.SqrtPriceX96.Int64())

	// lpFee 1_000_000 (0x0f4240) spans bits 208-231, above 16 bits
	s0 = decodeSlot0(common.HexToHash("0x000f4240000000fffffb0000000000000000000000000000000000000001"))
	assert.Equal(t, uint32(1_000_000), s0.LpFee)

	// liquidityNet = -7 (two's complement int128 in the high half), gross = 7
	var neg [32]byte
	new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(7)).FillBytes(neg[:16])
	neg[31] = 7
	gross, net := decodeTickInfo(neg)
	require.Equal(t, int64(7), gross.Int64())
	assert.Equal(t, int64(-7), net.Int64())

	// negative ticks hash as sign-extended int256
	base := stateSlot(common.HexToHash("0x01"))
	assert.NotEqual(t, tickSlot(base, -1), tickSlot(base, 1))
}
