package ephemeral

import (
	"bytes"
	"context"
	"math/big"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// refusingNode serves V3Lens one word a call at block 100, and refuses, as a proxy does for its
// size, a call sent more gas than refuseAbove. It keeps the gas each lens call was sent.
type refusingNode struct {
	t           *testing.T
	lens        *Contract
	refuseAbove uint64
	mu          sync.Mutex
	sent        []uint64
}

func (n *refusingNode) ChainID(context.Context) (*big.Int, error) { return big.NewInt(1), nil }

func (n *refusingNode) CallContract(_ context.Context, msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	if bytes.Equal(msg.Data, probeCode) {
		return nil, failing{msg: "no probe"}
	}
	n.mu.Lock()
	n.sent = append(n.sent, msg.Gas)
	n.mu.Unlock()
	if msg.Gas > n.refuseAbove {
		return nil, rpc.HTTPError{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway"}
	}
	args, err := n.lens.UnpackArgs(msg.Data)
	require.NoError(n.t, err)
	from, to := args[1].(int16), args[2].(int16)
	out, err := n.lens.PackResult(big.NewInt(60), big.NewInt(1<<40), big.NewInt(0), big.NewInt(0),
		big.NewInt(100), from+1, from >= to, []struct{ Tick, LiquidityNet, LiquidityGross *big.Int }{})
	require.NoError(n.t, err)
	return nil, dataError{hexutil.Encode(out)}
}

// TestReadTicksKeepsTheGasARefusalSettles: a node that refuses a large response refuses the first
// page of a read once, and the pages after it are sent the gas that page settled on rather than
// refused again each.
func TestReadTicksKeepsTheGasARefusalSettles(t *testing.T) {
	raw, err := os.ReadFile("../../liquidity-source/uniswap/v3/abis/lens.json")
	require.NoError(t, err)
	node := &refusingNode{t: t, lens: MustLoad("V3Lens", raw), refuseAbove: 25_000_000}

	p, err := ReadTicks(t.Context(), node, node.lens, 50_000_000, 0, 7, nil, common.Address{1})
	require.NoError(t, err)
	assert.EqualValues(t, 100, p.Block)

	refused := 0
	for _, gas := range node.sent {
		if gas > node.refuseAbove {
			refused++
		}
	}
	assert.Equal(t, 1, refused, "gas sent: %v", node.sent)
	assert.Len(t, node.sent, 9, "one refused call and a call for each of the 8 words")
}
