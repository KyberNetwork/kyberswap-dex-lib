package litepsm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	liteDai  = common.HexToAddress("0xf6e72Db5454dd049d0788e411b06CfAF16853042") // joins itself
	wrapper  = common.HexToAddress("0xA188EEC8F81263234dA3622A406892F3D630f98c") // wraps liteDai
	usddPsm  = common.HexToAddress("0xcE355440c00014A229bbEc030A2B8f8EB45a2897")
	usddJoin = common.HexToAddress("0x217e42CEB2eAE9ECB788fDF0e31c806c531760A3") // separate gemJoin
)

// gemJoin is read from the inner psm, so a wrapper over a self-joining lite psm must not
// get GemJoin (it would be charged gasJoin for a join it never calls).
func TestSetJoin(t *testing.T) {
	t.Parallel()
	var s StaticExtra
	s.setJoin(liteDai, liteDai, liteDai)
	require.Equal(t, StaticExtra{}, s)

	s = StaticExtra{}
	s.setJoin(wrapper, liteDai, liteDai)
	require.Nil(t, s.GemJoin)
	require.Equal(t, &liteDai, s.InnerPsm) // tracker reads the dai balance here

	s = StaticExtra{}
	s.setJoin(usddPsm, usddPsm, usddJoin)
	require.Equal(t, &usddJoin, s.GemJoin)
	require.Nil(t, s.InnerPsm)
}

func TestWrapperGas(t *testing.T) {
	t.Parallel()
	staticExtra, _ := json.Marshal(StaticExtra{Dai: &liteDai, InnerPsm: &liteDai})
	p, err := NewPoolSimulator(entity.Pool{
		Address:  "0xa188eec8f81263234da3622a406892f3d630f98c",
		Exchange: "lite-psm",
		Reserves: entity.PoolReserves{"1000000000000000000000000", "1000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0xdc035d45d973e3ec169d2276ddab16f1e407384f", Decimals: 18},
			{Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Decimals: 6},
		},
		Extra:       "{}",
		StaticExtra: string(staticExtra),
	})
	require.NoError(t, err)

	res, err := p.CalcAmountOut(poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{Token: p.Info.Tokens[0], Amount: big.NewInt(1e18)},
		TokenOut:      p.Info.Tokens[1],
	})
	require.NoError(t, err)
	require.Equal(t, int64(gasBuy+gasWrap), res.Gas)
}
