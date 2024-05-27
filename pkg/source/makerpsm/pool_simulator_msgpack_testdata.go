package makerpsm

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var usdxWAD = bignumber.TenPowInt(6)
var tollOnePct = bignumber.TenPowInt(16)

func newPool(t require.TestingT, eth *big.Int, tIn *big.Int, tOut *big.Int) *PoolSimulator {
	eth = new(big.Int).Mul(eth, bignumber.BONE)
	p, err := NewPoolSimulator(entity.Pool{
		Tokens: []*entity.PoolToken{{Address: "USDX", Decimals: 6}, {Address: DAIAddress}},
		Extra:  fmt.Sprintf("{\"psm\":{\"tIn\":%v,\"tOut\":%v,\"vat\":{\"ilk\":{\"art\":0,\"rate\":1,\"line\":%v},\"debt\":0,\"line\":%v}}}", tIn, tOut, eth, eth),
	})
	require.Nil(t, err)
	assert.Equal(t, []string{DAIAddress}, p.CanSwapTo("USDX"))
	assert.Equal(t, []string{"USDX"}, p.CanSwapTo(DAIAddress))
	return p
}

type panicTestingT struct{}

func (*panicTestingT) Errorf(format string, args ...interface{}) { panic(fmt.Sprintf(format, args...)) }
func (*panicTestingT) FailNow()                                  { panic("fail") }

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	return []*PoolSimulator{
		newPool((*panicTestingT)(nil), big.NewInt(100), big.NewInt(0), big.NewInt(0)),
		newPool((*panicTestingT)(nil), big.NewInt(100), tollOnePct, big.NewInt(0)),
		newPool((*panicTestingT)(nil), big.NewInt(100), new(big.Int).Mul(big.NewInt(5), tollOnePct), new(big.Int).Mul(big.NewInt(10), tollOnePct)),
	}
}
