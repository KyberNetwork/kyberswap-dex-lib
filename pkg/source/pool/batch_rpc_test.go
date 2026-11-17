package pool

import (
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/stretchr/testify/assert"
)

// A batch consumer finds the overrides through ILazyRequestWithOverrides; losing them would silently run
// override-dependent reads (e.g. the erc4626 cap probe) against real state.
func TestLazyRequestGetOverrides(t *testing.T) {
	t.Parallel()
	addr := common.HexToAddress("0x01")
	ov := map[common.Address]gethclient.OverrideAccount{addr: {Nonce: 7}}

	for name, in := range map[string]map[common.Address]gethclient.OverrideAccount{"set": ov, "none": nil} {
		req := &LazyRequest{Request: ethrpc.New("http://127.0.0.1:1").NewRequest().SetOverrides(in)}
		var r ILazyRequest = req
		o, ok := r.(ILazyRequestWithOverrides)
		assert.True(t, ok, name)
		assert.Equal(t, in, o.GetOverrides(), name)
	}
}
